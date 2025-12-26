package nodeapi

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"totoro-node/internal/apiresp"
	"totoro-node/internal/bridgeclient"
	"totoro-node/internal/kicker"
	"totoro-node/internal/nodeui"
	"totoro-node/internal/store"
)

type Options struct {
	AdminKey string
	TicketKey []byte
}

type API struct {
	st   *store.Store
	opts Options
}

func NewRouter(st *store.Store, opts Options) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	api := &API{st: st, opts: opts}

	// Node Web UI（最小静态页面）
	distFS := nodeui.DistFS()
	r.GET("/", func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		// gin.FileFromFS 在某些场景会触发 301 Location: ./（会导致浏览器/CLI 循环跳转），
		// 这里改为直接读取并返回，确保稳定可访问。
		b, err := fs.ReadFile(distFS, "index.html")
		if err != nil {
			apiresp.Fail(c, http.StatusNotFound, 404, "index.html not found")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", b)
	})
	r.NoRoute(func(c *gin.Context) {
		// 静态资源（仅 GET/HEAD）
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			apiresp.Fail(c, http.StatusNotFound, 404, "not found")
			return
		}
		p := strings.TrimPrefix(c.Request.URL.Path, "/")
		if p == "" {
			apiresp.Fail(c, http.StatusNotFound, 404, "not found")
			return
		}
		// 尝试直接读取静态文件
		if _, err := fs.Stat(distFS, p); err == nil {
			b, rerr := fs.ReadFile(distFS, p)
			if rerr != nil {
				apiresp.Fail(c, http.StatusInternalServerError, 500, rerr.Error())
				return
			}
			ct := mime.TypeByExtension(path.Ext(p))
			if ct == "" {
				ct = http.DetectContentType(b)
			}
			if c.Request.Method == http.MethodHead {
				c.Status(http.StatusOK)
				return
			}
			c.Data(http.StatusOK, ct, b)
			return
		}
		// SPA 回退（虽然当前是单页）
		b, err := fs.ReadFile(distFS, "index.html")
		if err != nil {
			apiresp.Fail(c, http.StatusNotFound, 404, "index.html not found")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", b)
	})

	v1 := r.Group("/api/v1")
	{
		v1.GET("/node/config", api.authAdmin(), api.getNodeConfig)
		v1.POST("/node/config", api.authAdmin(), api.updateNodeConfig)
		v1.GET("/node/invites", api.authAdmin(), api.listInvites)
		v1.POST("/node/invites", api.authAdmin(), api.createInvite)
		v1.POST("/node/invites/revoke", api.authAdmin(), api.revokeInvite)

		// 设备侧：邀请码解析 -> 连接票据（不需要登录）
		v1.POST("/invites/resolve", api.resolveInvite)
	}
	return r
}

type resolveInviteReq struct {
	Code string `json:"code" binding:"required"`
}

func (a *API) resolveInvite(c *gin.Context) {
	// 设计调整：邀请码预览/兑换在桥梁平台完成，节点侧不再解析。
	apiresp.Fail(c, http.StatusGone, 410, "invites.resolve 已迁移到 bridge（/api/v1/invites/preview & /api/v1/invites/redeem）")
}

func (a *API) authAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.opts.AdminKey == "" {
			c.Next()
			return
		}
		if strings.TrimSpace(c.GetHeader("X-Admin-Key")) != a.opts.AdminKey {
			apiresp.Fail(c, http.StatusUnauthorized, 401, "unauthorized")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (a *API) getNodeConfig(c *gin.Context) {
	cfg, _, err := a.st.GetNodeConfig()
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	// 不返回 node_key
	cfg.NodeKey = ""
	apiresp.OK(c, cfg)
}

func (a *API) updateNodeConfig(c *gin.Context) {
	var req store.NodeConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	adminNodeKey := strings.TrimSpace(c.GetHeader("X-Node-Key"))
	if adminNodeKey == "" {
		apiresp.Fail(c, http.StatusUnauthorized, 401, "missing X-Node-Key")
		return
	}
	if err := a.st.UpdateNodeConfig(adminNodeKey, req); err != nil {
		apiresp.Fail(c, http.StatusForbidden, 403, err.Error())
		return
	}
	apiresp.OK(c, gin.H{"updated": true})
}

func (a *API) listInvites(c *gin.Context) {
	limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	includeRevoked := true
	if strings.TrimSpace(c.Query("include_revoked")) == "0" {
		includeRevoked = false
	}
	items, err := a.st.ListInvites(limit, includeRevoked)
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	apiresp.OK(c, gin.H{"invites": items})
}

type createInviteReq struct {
	TTLSeconds int    `json:"ttl_seconds"`
	MaxUses    int    `json:"max_uses"`
	ScopeJSON  string `json:"scope_json"`
}

func (a *API) createInvite(c *gin.Context) {
	var req createInviteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	cfg, keyHash, err := a.st.GetNodeConfig()
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	bridge := strings.TrimRight(strings.TrimSpace(cfg.BridgeURL), "/")
	if bridge == "" {
		bridge = strings.TrimRight(strings.TrimSpace(os.Getenv("TOTOTO_BRIDGE_URL")), "/")
	}
	if bridge == "" {
		apiresp.Fail(c, http.StatusBadRequest, 400, "未配置 bridge_url（节点配置）或 TOTOTO_BRIDGE_URL（环境变量）")
		return
	}
	adminNodeKey := strings.TrimSpace(c.GetHeader("X-Node-Key"))
	if adminNodeKey == "" {
		apiresp.Fail(c, http.StatusUnauthorized, 401, "missing X-Node-Key")
		return
	}
	// 校验 node_key（与更新配置一致）
	if storeHash(adminNodeKey) != keyHash {
		apiresp.Fail(c, http.StatusForbidden, 403, "node_key invalid")
		return
	}
	bc := &bridgeclient.Client{BaseURL: bridge, NodeID: cfg.NodeID, NodeKey: adminNodeKey}
	out, err := bc.CreateInvite(bridgeclient.CreateInviteReq{
		ScopeJSON:  strings.TrimSpace(req.ScopeJSON),
		TTLSeconds: req.TTLSeconds,
		MaxUses:    req.MaxUses,
	})
	if err != nil {
		apiresp.Fail(c, http.StatusBadGateway, 502, err.Error())
		return
	}
	// 落库用于列表管理；失败不阻断 create（邀请码已经在 bridge 创建成功）
	_ = a.st.UpsertInviteFromBridge(
		strings.TrimSpace(out.InviteID),
		strings.TrimSpace(out.Code),
		strings.TrimSpace(out.ExpiresAt),
		req.MaxUses,
		strings.TrimSpace(req.ScopeJSON),
	)
	apiresp.OK(c, gin.H{
		"invite_id":  strings.TrimSpace(out.InviteID),
		"code":       strings.TrimSpace(out.Code),
		"expires_at": strings.TrimSpace(out.ExpiresAt),
	})
}

type revokeInviteReq struct {
	InviteID string `json:"invite_id"`
}

func (a *API) revokeInvite(c *gin.Context) {
	var req revokeInviteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	if strings.TrimSpace(req.InviteID) == "" {
		apiresp.Fail(c, http.StatusBadRequest, 400, "invite_id required")
		return
	}
	cfg, keyHash, err := a.st.GetNodeConfig()
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	bridge := strings.TrimRight(strings.TrimSpace(cfg.BridgeURL), "/")
	if bridge == "" {
		bridge = strings.TrimRight(strings.TrimSpace(os.Getenv("TOTOTO_BRIDGE_URL")), "/")
	}
	if bridge == "" {
		apiresp.Fail(c, http.StatusBadRequest, 400, "未配置 bridge_url（节点配置）或 TOTOTO_BRIDGE_URL（环境变量）")
		return
	}
	adminNodeKey := strings.TrimSpace(c.GetHeader("X-Node-Key"))
	if adminNodeKey == "" {
		apiresp.Fail(c, http.StatusUnauthorized, 401, "missing X-Node-Key")
		return
	}
	// 校验 node_key（与更新配置一致）
	if storeHash(adminNodeKey) != keyHash {
		apiresp.Fail(c, http.StatusForbidden, 403, "node_key invalid")
		return
	}
	bc := &bridgeclient.Client{BaseURL: bridge, NodeID: cfg.NodeID, NodeKey: adminNodeKey}
	if err := bc.RevokeInvite(strings.TrimSpace(req.InviteID)); err != nil {
		apiresp.Fail(c, http.StatusBadGateway, 502, err.Error())
		return
	}
	// 本地标记 revoked，便于列表展示
	_ = a.st.RevokeInvite(strings.TrimSpace(req.InviteID))
	kicked := kicker.KickInvite(strings.TrimSpace(req.InviteID))
	apiresp.OK(c, gin.H{"revoked": true, "kicked": kicked})
}

func generateInviteCode() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	s := strings.ToUpper(hex.EncodeToString(b))
	// ABCD-EFGH-IJKL 这种形态
	return s[0:4] + "-" + s[4:8] + "-" + s[8:12]
}

func storeHash(v string) string {
	// 这里简单用 sha256 hex，复用 store 内部逻辑会导致循环依赖；保持独立实现
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}


