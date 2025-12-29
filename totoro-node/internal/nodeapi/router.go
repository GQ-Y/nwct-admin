package nodeapi

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"totoro-node/internal/apiresp"
	"totoro-node/internal/bridgeclient"
	"totoro-node/internal/kicker"
	"totoro-node/internal/nodeui"
	"totoro-node/internal/store"
)

type Options struct {
	TicketKey []byte
}

type API struct {
	st   *store.Store
	opts Options
}

func isLoopbackIP(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
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
		// 不要影响 API：NoRoute 只会在未匹配任何路由时触发
		// 静态资源（仅 GET/HEAD）
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			apiresp.Fail(c, http.StatusNotFound, 404, "not found")
			return
		}
		reqPath := strings.TrimPrefix(strings.TrimSpace(c.Request.URL.Path), "/")
		reqPath = path.Clean(reqPath)
		reqPath = strings.TrimPrefix(reqPath, "/")
		if reqPath == "" || reqPath == "." {
			// 兜底到 /
			b, err := fs.ReadFile(distFS, "index.html")
			if err != nil {
				apiresp.Fail(c, http.StatusNotFound, 404, "index.html not found")
				return
			}
			c.Data(http.StatusOK, "text/html; charset=utf-8", b)
			return
		}

		// 先尝试按静态资源返回（例如 /assets/*.js /styles.css）
		if b, err := fs.ReadFile(distFS, reqPath); err == nil {
			ctype := mime.TypeByExtension(path.Ext(reqPath))
			if strings.TrimSpace(ctype) == "" {
				ctype = http.DetectContentType(b)
			}
			if c.Request.Method == http.MethodHead {
				c.Status(http.StatusOK)
				return
			}
			c.Data(http.StatusOK, ctype, b)
			return
		}

		// 找不到静态资源：作为 SPA 路由，返回 index.html
		b, err := fs.ReadFile(distFS, "index.html")
		if err != nil {
			apiresp.Fail(c, http.StatusNotFound, 404, "index.html not found")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", b)
	})

	v1 := r.Group("/api/v1")
	{
		// admin 登录和密码修改（不需要认证）
		v1.POST("/admin/login", api.handleAdminLogin)
		v1.POST("/admin/password/change", api.authAdmin(), api.handleAdminChangePassword)

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

type adminTokenPayload struct {
	Exp int64  `json:"exp"`
	N   string `json:"n"`
}

func (a *API) getAdminKey() string {
	// 从数据库读取 AdminKey
	key, err := a.st.GetAdminKey()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(key)
}

func (a *API) setAdminKey(key string) error {
	// 写入数据库
	return a.st.UpdateAdminKey(key)
}

func (a *API) issueAdminToken(ttl time.Duration) (string, time.Time, error) {
	adminKey := a.getAdminKey()
	if strings.TrimSpace(adminKey) == "" {
		// 未配置 AdminKey：返回一个特殊的 token（表示"无认证"状态）
		// 使用固定的密钥生成 token，这样前端可以验证 token 的有效性
		adminKey = "no_auth_required"
	}
	if ttl <= 0 {
		ttl = 7 * 24 * time.Hour
	}
	if ttl > 30*24*time.Hour {
		ttl = 30 * 24 * time.Hour
	}
	exp := time.Now().Add(ttl).UTC()
	nonce := make([]byte, 16)
	_, _ = rand.Read(nonce)
	p := adminTokenPayload{Exp: exp.Unix(), N: base64.RawURLEncoding.EncodeToString(nonce)}
	b, _ := json.Marshal(p)
	payloadB64 := base64.RawURLEncoding.EncodeToString(b)
	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(adminKey)))
	_, _ = mac.Write([]byte(payloadB64))
	sigB64 := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payloadB64 + "." + sigB64, exp, nil
}

func (a *API) verifyAdminToken(tok string) bool {
	tok = strings.TrimSpace(tok)
	if tok == "" {
		return false
	}
	parts := strings.Split(tok, ".")
	if len(parts) != 2 {
		return false
	}
	payloadB64 := parts[0]
	sigB64 := parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return false
	}
	adminKey := a.getAdminKey()
	// 如果未配置 AdminKey，使用固定密钥验证
	if strings.TrimSpace(adminKey) == "" {
		adminKey = "no_auth_required"
	}
	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(adminKey)))
	_, _ = mac.Write([]byte(payloadB64))
	expect := mac.Sum(nil)
	if !hmac.Equal(sig, expect) {
		return false
	}
	raw, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return false
	}
	var p adminTokenPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return false
	}
	if p.Exp <= 0 {
		return false
	}
	if time.Now().Unix() > p.Exp {
		return false
	}
	return true
}

func (a *API) getBearerToken(c *gin.Context) string {
	// Authorization: Bearer <token>
	h := strings.TrimSpace(c.GetHeader("Authorization"))
	if h != "" {
		parts := strings.SplitN(h, " ", 2)
		if len(parts) == 2 && strings.EqualFold(strings.TrimSpace(parts[0]), "bearer") {
			return strings.TrimSpace(parts[1])
		}
	}
	// fallback: X-Admin-Token
	return strings.TrimSpace(c.GetHeader("X-Admin-Token"))
}

func (a *API) authAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		adminKey := a.getAdminKey()
		// 如果未配置 AdminKey，直接放行
		if strings.TrimSpace(adminKey) == "" {
			c.Next()
			return
		}
		// token 优先
		if a.verifyAdminToken(a.getBearerToken(c)) {
			c.Next()
			return
		}
		// 兼容：直接使用 admin key（仅用于紧急/兼容）
		if strings.TrimSpace(c.GetHeader("X-Admin-Key")) != strings.TrimSpace(adminKey) {
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
	// BridgeURL 不允许通过 API 修改（只读）
	cfg, _, err := a.st.GetNodeConfig()
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	req.BridgeURL = cfg.BridgeURL // 保持原有值，不允许修改

	// 桌面端/自托管场景：若配置了 X-Admin-Key，并且已经通过 authAdmin，
	// 则允许无需 X-Node-Key 也能更新配置（减少"启动两遍/手工拷贝 node_key"的成本）。
	//
	// 若未配置 AdminKey：
	// - 本机回环访问（127.0.0.1/::1）允许不带 X-Node-Key（便于本机部署/桌面端）
	// - 非回环访问仍要求 X-Node-Key（避免公网裸奔）
	adminNodeKey := strings.TrimSpace(c.GetHeader("X-Node-Key"))
	adminKey := a.getAdminKey()
	if strings.TrimSpace(adminKey) == "" {
		if adminNodeKey == "" && !isLoopbackIP(c.ClientIP()) {
			apiresp.Fail(c, http.StatusUnauthorized, 401, "missing X-Node-Key")
			return
		}
		if adminNodeKey != "" {
			if err := a.st.UpdateNodeConfig(adminNodeKey, req); err != nil {
				apiresp.Fail(c, http.StatusForbidden, 403, err.Error())
				return
			}
		} else {
			// loopback：免 node_key
			if err := a.st.UpdateNodeConfigAsAdmin(req); err != nil {
				apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
				return
			}
		}
		apiresp.OK(c, gin.H{"updated": true})
		return
	}

	// AdminKey 已配置：优先走 node_key 校验（如果用户提供了）；否则走"已鉴权的 admin 更新"。
	if adminNodeKey != "" {
		if err := a.st.UpdateNodeConfig(adminNodeKey, req); err != nil {
			apiresp.Fail(c, http.StatusForbidden, 403, err.Error())
			return
		}
		apiresp.OK(c, gin.H{"updated": true})
		return
	}
	if err := a.st.UpdateNodeConfigAsAdmin(req); err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	apiresp.OK(c, gin.H{"updated": true})
}

func (a *API) listInvites(c *gin.Context) {
	limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	// 默认不返回已删除（撤销）的邀请码；需要时可显式 include_revoked=1
	includeRevoked := strings.TrimSpace(c.Query("include_revoked")) == "1"
	items, err := a.st.ListInvites(limit, includeRevoked)
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	if items == nil {
		items = make([]store.Invite, 0)
	}
	apiresp.OK(c, gin.H{"invites": items})
}

type createInviteReq struct {
	TTLDays   int    `json:"ttl_days"`
	MaxUses   int    `json:"max_uses"`
	ScopeJSON string `json:"scope_json"`
}

func (a *API) createInvite(c *gin.Context) {
	var req createInviteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}

	// 统一：只接受 ttl_days（客户端统一按天传递）。
	if req.TTLDays < 0 {
		apiresp.Fail(c, http.StatusBadRequest, 400, "ttl_days invalid")
		return
	}
	ttlSeconds := 0
	if req.TTLDays > 0 {
		ttlSeconds = req.TTLDays * 86400
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
	adminKey := a.getAdminKey()
	// 若配置了 AdminKey 且已通过 authAdmin，可不传 X-Node-Key，直接用节点本地持久化的 node_key。
	if adminNodeKey == "" {
		if strings.TrimSpace(adminKey) == "" && !isLoopbackIP(c.ClientIP()) {
			apiresp.Fail(c, http.StatusUnauthorized, 401, "missing X-Node-Key")
			return
		}
		// loopback 且未配置 AdminKey：允许免 node_key，但需要从 db 取 key 用于签名调用 bridge
		adminNodeKey = strings.TrimSpace(cfg.NodeKey)
		if adminNodeKey == "" {
			apiresp.Fail(c, http.StatusInternalServerError, 500, "node_key missing in node.db")
			return
		}
	} else {
		// 校验 node_key（与更新配置一致）
		if storeHash(adminNodeKey) != keyHash {
			apiresp.Fail(c, http.StatusForbidden, 403, "node_key invalid")
			return
		}
	}
	bc := &bridgeclient.Client{BaseURL: bridge, NodeID: cfg.NodeID, NodeKey: adminNodeKey}
	out, err := bc.CreateInvite(bridgeclient.CreateInviteReq{
		ScopeJSON:  strings.TrimSpace(req.ScopeJSON),
		TTLSeconds: ttlSeconds,
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
	adminKey := a.getAdminKey()
	// 若配置了 AdminKey 且已通过 authAdmin，可不传 X-Node-Key，直接用节点本地持久化的 node_key。
	if adminNodeKey == "" {
		if strings.TrimSpace(adminKey) == "" && !isLoopbackIP(c.ClientIP()) {
			apiresp.Fail(c, http.StatusUnauthorized, 401, "missing X-Node-Key")
			return
		}
		adminNodeKey = strings.TrimSpace(cfg.NodeKey)
		if adminNodeKey == "" {
			apiresp.Fail(c, http.StatusInternalServerError, 500, "node_key missing in node.db")
			return
		}
	} else {
		// 校验 node_key（与更新配置一致）
		if storeHash(adminNodeKey) != keyHash {
			apiresp.Fail(c, http.StatusForbidden, 403, "node_key invalid")
			return
		}
	}
	bc := &bridgeclient.Client{BaseURL: bridge, NodeID: cfg.NodeID, NodeKey: adminNodeKey}
	if err := bc.RevokeInvite(strings.TrimSpace(req.InviteID)); err != nil {
		apiresp.Fail(c, http.StatusBadGateway, 502, err.Error())
		return
	}
	// 本地标记 revoked，便于列表展示
	_ = a.st.RevokeInvite(strings.TrimSpace(req.InviteID))
	kicked := kicker.KickInvite(strings.TrimSpace(req.InviteID))
	// 语义：撤销 == 删除（不再可用，也不再出现在默认列表里）
	apiresp.OK(c, gin.H{"deleted": true, "kicked": kicked})
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

type adminLoginReq struct {
	AdminKey string `json:"admin_key"`
	Password string `json:"password"`
}

func (a *API) handleAdminLogin(c *gin.Context) {
	adminKey := a.getAdminKey()
	// 如果未配置 AdminKey，直接返回 token（允许任何输入）
	if strings.TrimSpace(adminKey) == "" {
		tok, exp, err := a.issueAdminToken(7 * 24 * time.Hour)
		if err != nil {
			apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
			return
		}
		apiresp.OK(c, gin.H{
			"token":      tok,
			"expires_at": exp.UTC().Format(time.RFC3339),
		})
		return
	}
	// 已配置 AdminKey，需要验证
	var req adminLoginReq
	_ = c.ShouldBindJSON(&req)
	k := strings.TrimSpace(req.AdminKey)
	if k == "" {
		k = strings.TrimSpace(req.Password)
	}
	if k == "" {
		k = strings.TrimSpace(c.GetHeader("X-Admin-Key"))
	}
	if k == "" {
		apiresp.Fail(c, http.StatusBadRequest, 400, "admin_key required")
		return
	}
	if k != strings.TrimSpace(adminKey) {
		apiresp.Fail(c, http.StatusUnauthorized, 401, "unauthorized")
		return
	}
	tok, exp, err := a.issueAdminToken(7 * 24 * time.Hour)
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	apiresp.OK(c, gin.H{
		"token":      tok,
		"expires_at": exp.UTC().Format(time.RFC3339),
	})
}

type adminChangePasswordReq struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (a *API) handleAdminChangePassword(c *gin.Context) {
	adminKey := a.getAdminKey()
	// 如果未配置 AdminKey，不允许修改（因为没有旧密码可验证）
	if strings.TrimSpace(adminKey) == "" {
		apiresp.Fail(c, http.StatusForbidden, 403, "admin not configured")
		return
	}
	var req adminChangePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	oldPwd := strings.TrimSpace(req.OldPassword)
	newPwd := strings.TrimSpace(req.NewPassword)
	if oldPwd == "" || newPwd == "" {
		apiresp.Fail(c, http.StatusBadRequest, 400, "old_password/new_password required")
		return
	}
	if oldPwd != strings.TrimSpace(adminKey) {
		apiresp.Fail(c, http.StatusUnauthorized, 401, "unauthorized")
		return
	}
	if len(newPwd) < 8 {
		apiresp.Fail(c, http.StatusBadRequest, 400, "password too short")
		return
	}
	if len(newPwd) > 128 {
		apiresp.Fail(c, http.StatusBadRequest, 400, "password too long")
		return
	}
	// 更新数据库中的 AdminKey
	if err := a.setAdminKey(newPwd); err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	tok, exp, err := a.issueAdminToken(7 * 24 * time.Hour)
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	apiresp.OK(c, gin.H{
		"token":      tok,
		"expires_at": exp.UTC().Format(time.RFC3339),
	})
}
