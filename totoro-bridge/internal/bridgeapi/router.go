package bridgeapi

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"totoro-bridge/internal/apiresp"
	"totoro-bridge/internal/bridgeui"
	"totoro-bridge/internal/cryptobox"
	"totoro-bridge/internal/store"
	"totoro-bridge/internal/ticket"

	"github.com/gin-gonic/gin"
	"github.com/shakinm/xlsReader/xls"
	"github.com/xuri/excelize/v2"
)

func ticketTTL() time.Duration {
	// 票据有效期：默认 30 天（用户期望“至少 30 天的换票核验期”）
	// 可通过环境变量 TOTOTO_TICKET_TTL_DAYS 覆盖，例如 7/30/90
	daysStr := strings.TrimSpace(getenvDefault("TOTOTO_TICKET_TTL_DAYS", "30"))
	days, _ := strconv.Atoi(daysStr)
	if days <= 0 {
		days = 30
	}
	if days > 365 {
		days = 365
	}
	return time.Duration(days) * 24 * time.Hour
}

func getenvDefault(k, def string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	return v
}

type Options struct {
	AdminKey string
}

type Router struct {
	store store.Store
	opts  Options
}

func NewRouter(st store.Store, opts Options) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	api := &Router{store: st, opts: opts}

	// Admin UI（内嵌静态资源 + SPA fallback）
	uiFS := bridgeui.DistFS()
	r.GET("/", func(c *gin.Context) {
		b, err := fs.ReadFile(uiFS, "index.html")
		if err != nil {
			apiresp.Fail(c, http.StatusNotFound, 404, "index.html not found")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", b)
	})
	r.NoRoute(func(c *gin.Context) {
		// 不要影响 API：NoRoute 只会在未匹配任何路由时触发
		reqPath := strings.TrimPrefix(strings.TrimSpace(c.Request.URL.Path), "/")
		reqPath = path.Clean(reqPath)
		reqPath = strings.TrimPrefix(reqPath, "/")
		if reqPath == "" || reqPath == "." {
			// 兜底到 /
			b, err := fs.ReadFile(uiFS, "index.html")
			if err != nil {
				apiresp.Fail(c, http.StatusNotFound, 404, "index.html not found")
				return
			}
			c.Data(http.StatusOK, "text/html; charset=utf-8", b)
			return
		}

		// 先尝试按静态资源返回（例如 /assets/*.js /styles.css）
		if b, err := fs.ReadFile(uiFS, reqPath); err == nil {
			ctype := mime.TypeByExtension(path.Ext(reqPath))
			if strings.TrimSpace(ctype) == "" {
				ctype = http.DetectContentType(b)
			}
			c.Data(http.StatusOK, ctype, b)
			return
		}

		// 找不到静态资源：作为 SPA 路由，返回 index.html
		b, err := fs.ReadFile(uiFS, "index.html")
		if err != nil {
			apiresp.Fail(c, http.StatusNotFound, 404, "index.html not found")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", b)
	})

	v1 := r.Group("/api/v1")
	{
		v1.POST("/device/register", api.handleDeviceRegister)
		v1.GET("/public/nodes", api.authDeviceOrAdmin(), api.handlePublicNodes)
		v1.POST("/public/nodes/connect", api.authDeviceOrAdmin(), api.handlePublicNodeConnect)
		v1.GET("/official/nodes", api.authDeviceOrAdmin(), api.handleOfficialNodes)
		v1.POST("/invites/preview", api.authDeviceOrAdmin(), api.handleInvitePreview)
		v1.POST("/invites/redeem", api.authDeviceOrAdmin(), api.handleInviteRedeem)

		v1.POST("/nodes/heartbeat", api.handleNodeHeartbeat)
		v1.POST("/nodes/invites/create", api.handleNodeInviteCreate)
		v1.POST("/nodes/invites/revoke", api.handleNodeInviteRevoke)

		// admin
		v1.POST("/admin/login", api.handleAdminLogin)
		v1.POST("/admin/password/change", api.authAdmin(), api.handleAdminChangePassword)
		v1.POST("/admin/nodes/upsert", api.authAdmin(), api.handleAdminUpsertNode)
		v1.POST("/admin/official_nodes/upsert", api.authAdmin(), api.handleAdminUpsertOfficialNode)
		v1.POST("/admin/official_nodes/delete", api.authAdmin(), api.handleAdminDeleteOfficialNode)
		v1.GET("/admin/official_nodes", api.authAdmin(), api.handleAdminListOfficialNodes)
		v1.POST("/admin/devices/whitelist/upsert", api.authAdmin(), api.handleAdminUpsertWhitelist)
		v1.POST("/admin/devices/whitelist/delete", api.authAdmin(), api.handleAdminDeleteWhitelist)
		v1.GET("/admin/devices/whitelist", api.authAdmin(), api.handleAdminListWhitelist)
		v1.POST("/admin/devices/whitelist/import", api.authAdmin(), api.handleAdminImportWhitelist)
	}

	return r
}

type adminTokenPayload struct {
	Exp int64  `json:"exp"`
	N   string `json:"n"`
}

func (a *Router) issueAdminToken(ttl time.Duration) (string, time.Time, error) {
	if strings.TrimSpace(a.opts.AdminKey) == "" {
		return "", time.Time{}, fmt.Errorf("admin not configured")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
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
	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(a.opts.AdminKey)))
	_, _ = mac.Write([]byte(payloadB64))
	sigB64 := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payloadB64 + "." + sigB64, exp, nil
}

func (a *Router) verifyAdminToken(tok string) bool {
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
	if strings.TrimSpace(a.opts.AdminKey) == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(a.opts.AdminKey)))
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
	if time.Now().UTC().Unix() > p.Exp {
		return false
	}
	return true
}

func (a *Router) getBearerToken(c *gin.Context) string {
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

func (a *Router) authDeviceOrAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// admin 直接放行：token 或 admin key
		if a.verifyAdminToken(a.getBearerToken(c)) {
			c.Set("is_admin", true)
			c.Next()
			return
		}
		if strings.TrimSpace(a.opts.AdminKey) != "" && strings.TrimSpace(c.GetHeader("X-Admin-Key")) == strings.TrimSpace(a.opts.AdminKey) {
			c.Set("is_admin", true)
			c.Next()
			return
		}
		tok := strings.TrimSpace(c.GetHeader("X-Device-Token"))
		ok, deviceID, err := a.store.VerifyDeviceSession(tok)
		if err != nil {
			apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
			c.Abort()
			return
		}
		if !ok {
			apiresp.Fail(c, http.StatusUnauthorized, 401, "unauthorized")
			c.Abort()
			return
		}
		c.Set("device_id", deviceID)
		c.Next()
	}
}

func (a *Router) authAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.TrimSpace(a.opts.AdminKey) == "" {
			apiresp.Fail(c, http.StatusForbidden, 403, "admin not configured")
			c.Abort()
			return
		}
		// token 优先
		if a.verifyAdminToken(a.getBearerToken(c)) {
			c.Next()
			return
		}
		// 兼容：直接使用 admin key（仅用于紧急/兼容）
		if strings.TrimSpace(c.GetHeader("X-Admin-Key")) != strings.TrimSpace(a.opts.AdminKey) {
			apiresp.Fail(c, http.StatusUnauthorized, 401, "unauthorized")
			c.Abort()
			return
		}
		c.Next()
	}
}

type adminLoginReq struct {
	AdminKey string `json:"admin_key"`
	Password string `json:"password"`
}

func (a *Router) handleAdminLogin(c *gin.Context) {
	if strings.TrimSpace(a.opts.AdminKey) == "" {
		apiresp.Fail(c, http.StatusForbidden, 403, "admin not configured")
		return
	}
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
		apiresp.Fail(c, http.StatusBadRequest, 400, "password required")
		return
	}
	if k != strings.TrimSpace(a.opts.AdminKey) {
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

func (a *Router) handleAdminChangePassword(c *gin.Context) {
	if strings.TrimSpace(a.opts.AdminKey) == "" {
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
	if oldPwd != strings.TrimSpace(a.opts.AdminKey) {
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
	// 直接更新内存中的“管理密码”（无需账号系统；重启后若使用环境变量启动，则会回到环境变量的值）
	a.opts.AdminKey = newPwd
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

func (a *Router) handlePublicNodes(c *gin.Context) {
	nodes, err := a.store.ListPublicNodes()
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	if isAdmin, _ := c.Get("is_admin"); isAdmin == true {
		apiresp.OK(c, gin.H{"nodes": nodes})
		return
	}
	deviceID, _ := c.Get("device_id")
	enc, err := a.encryptForDevice(fmt.Sprintf("%v", deviceID), gin.H{"nodes": nodes})
	if err != nil {
		// 设备缺少公钥（例如复用了旧 session）：要求重新注册上报 pub_key
		msg := strings.TrimSpace(err.Error())
		if strings.Contains(msg, "pub_key") || strings.Contains(msg, "device not found") {
			apiresp.Fail(c, http.StatusUnauthorized, 401, "device key missing, please re-register")
			return
		}
		apiresp.Fail(c, http.StatusInternalServerError, 500, msg)
		return
	}
	apiresp.OK(c, gin.H{"encrypted": enc})
}

func (a *Router) handleOfficialNodes(c *gin.Context) {
	nodes, err := a.store.ListOfficialNodes()
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	if isAdmin, _ := c.Get("is_admin"); isAdmin == true {
		apiresp.OK(c, gin.H{"nodes": nodes})
		return
	}
	deviceID, _ := c.Get("device_id")
	enc, err := a.encryptForDevice(fmt.Sprintf("%v", deviceID), gin.H{"nodes": nodes})
	if err != nil {
		msg := strings.TrimSpace(err.Error())
		if strings.Contains(msg, "pub_key") || strings.Contains(msg, "device not found") {
			apiresp.Fail(c, http.StatusUnauthorized, 401, "device key missing, please re-register")
			return
		}
		apiresp.Fail(c, http.StatusInternalServerError, 500, msg)
		return
	}
	apiresp.OK(c, gin.H{"encrypted": enc})
}

type publicNodeConnectReq struct {
	NodeID string `json:"node_id" binding:"required"`
}

// handlePublicNodeConnect 公开节点“直接连接”（无需邀请码）：桥梁签发短期 ticket 并返回节点 endpoints（加密给设备端）。
func (a *Router) handlePublicNodeConnect(c *gin.Context) {
	if isAdmin, _ := c.Get("is_admin"); isAdmin == true {
		apiresp.Fail(c, http.StatusForbidden, 403, "admin not allowed")
		return
	}
	deviceIDAny, _ := c.Get("device_id")
	deviceID := strings.TrimSpace(fmt.Sprintf("%v", deviceIDAny))
	var req publicNodeConnectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	nodeID := strings.TrimSpace(req.NodeID)
	n, err := a.store.GetPublicNodeByID(nodeID)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	// offline 不允许直连（仍可通过邀请码分享私有节点）
	if strings.TrimSpace(n.Status) == "offline" {
		apiresp.Fail(c, http.StatusBadRequest, 400, "node_offline")
		return
	}
	nodeKeyPlain, err := a.store.GetNodeKeyPlain(n.NodeID)
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	// public 直连：不使用邀请码，生成一次性 invite_id 作为计费/限额维度
	inviteID := "pub_" + randomCode(12)
	tok, exp, err := ticket.IssueHMAC(n.NodeID, inviteID, "null", []byte(nodeKeyPlain), ticketTTL())
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	payload := gin.H{
		"node": gin.H{
			"node_id":       n.NodeID,
			"endpoints":     n.Endpoints,
			"domain_suffix": n.DomainSuffix,
			"http_enabled":  n.HTTPEnabled,
			"https_enabled": n.HTTPSEnabled,
		},
		"connection_ticket": tok,
		"expires_at":        exp.UTC().Format(time.RFC3339),
	}
	enc, err := a.encryptForDevice(deviceID, payload)
	if err != nil {
		msg := strings.TrimSpace(err.Error())
		if strings.Contains(msg, "pub_key") || strings.Contains(msg, "device not found") {
			apiresp.Fail(c, http.StatusUnauthorized, 401, "device key missing, please re-register")
			return
		}
		apiresp.Fail(c, http.StatusInternalServerError, 500, msg)
		return
	}
	apiresp.OK(c, gin.H{"encrypted": enc})
}

type heartbeatReq struct {
	Ts           string               `json:"ts"`
	NodeID       string               `json:"node_id"`
	Public       bool                 `json:"public"`
	Name         string               `json:"name"`
	Description  string               `json:"description"`
	Region       string               `json:"region"`
	ISP          string               `json:"isp"`
	Tags         []string             `json:"tags"`
	Endpoints    []store.NodeEndpoint `json:"endpoints"`
	NodeAPI      string               `json:"node_api"`
	DomainSuffix string               `json:"domain_suffix"`
	HTTPEnabled  bool                 `json:"http_enabled"`
	HTTPSEnabled bool                 `json:"https_enabled"`
	TCPPortPool  *store.PortPool      `json:"tcp_port_pool"`
	UDPPortPool  *store.PortPool      `json:"udp_port_pool"`
	Version      any                  `json:"version"`
	Metrics      any                  `json:"metrics"`
	Extra        any                  `json:"extra"`
}

func (a *Router) handleNodeHeartbeat(c *gin.Context) {
	// node 级别鉴权：node_id + node_key
	//
	// 设计调整：允许“首次启动自动注册节点”——当 node_id 从未出现过时，bridge 自动写入 node_auth，
	// 后续再按同一 node_key 校验（避免要求手工预登记 node_key，满足“任何人都可启动节点上报桥梁”的目标）。
	nodeID := strings.TrimSpace(c.GetHeader("X-Node-Id"))
	nodeKey := strings.TrimSpace(c.GetHeader("X-Node-Key"))
	if nodeID == "" || nodeKey == "" {
		apiresp.Fail(c, http.StatusUnauthorized, 401, "missing node auth headers")
		return
	}
	ok, err := a.store.VerifyNodeAuth(nodeID, nodeKey)
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	if !ok {
		exists, eerr := a.store.HasNodeAuth(nodeID)
		if eerr != nil {
			apiresp.Fail(c, http.StatusInternalServerError, 500, eerr.Error())
			return
		}
		if exists {
			apiresp.Fail(c, http.StatusUnauthorized, 401, "invalid node auth")
			return
		}
		// 首次注册：绑定 node_id -> node_key
		if err := a.store.UpsertNodeAuth(nodeID, nodeKey); err != nil {
			apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
			return
		}
	}
	// 重要：bridge 需要 node_key 明文用于签发 ticket。
	// 这里无条件回填明文（避免旧库 node_key_plain 为空导致 500）。
	_ = a.store.UpsertNodeAuth(nodeID, nodeKey)

	var req heartbeatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	if strings.TrimSpace(req.NodeID) != "" && strings.TrimSpace(req.NodeID) != nodeID {
		apiresp.Fail(c, http.StatusBadRequest, 400, "node_id mismatch")
		return
	}
	req.NodeID = nodeID

	// 这里用二次 marshal（简单可靠）
	metricsJSON, _ := jsonMarshal(req.Metrics)
	versionJSON2, _ := jsonMarshal(req.Version)
	extraJSON, _ := jsonMarshal(req.Extra)

	hb := store.NodeHeartbeat{
		NodeID:       nodeID,
		NodeKey:      nodeKey,
		Public:       req.Public,
		Name:         strings.TrimSpace(req.Name),
		Description:  sanitizeText(strings.TrimSpace(req.Description), 200),
		Region:       strings.TrimSpace(req.Region),
		ISP:          strings.TrimSpace(req.ISP),
		Tags:         req.Tags,
		Endpoints:    req.Endpoints,
		NodeAPI:      strings.TrimSpace(req.NodeAPI),
		DomainSuffix: strings.TrimSpace(req.DomainSuffix),
		HTTPEnabled:  req.HTTPEnabled,
		HTTPSEnabled: req.HTTPSEnabled,
		TCPPortPool:  req.TCPPortPool,
		UDPPortPool:  req.UDPPortPool,
		MetricsJSON:  metricsJSON,
		VersionJSON:  versionJSON2,
		ExtraJSON:    extraJSON,
	}
	if err := a.store.UpsertNodeHeartbeat(hb); err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}

	apiresp.OK(c, gin.H{"status": "ok"})
}

func sanitizeText(s string, max int) string {
	if max <= 0 {
		max = 200
	}
	// 纯文本：去掉换行/制表符，避免 UI 被“撑开”
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.TrimSpace(s)
	if len(s) > max {
		s = s[:max]
	}
	return s
}

type deviceRegisterReq struct {
	DeviceID string `json:"device_id" binding:"required"`
	MAC      string `json:"mac" binding:"required"`
	PubKey   string `json:"pub_key" binding:"required"`
}

func (a *Router) handleDeviceRegister(c *gin.Context) {
	var req deviceRegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	deviceID := strings.TrimSpace(req.DeviceID)
	mac := strings.TrimSpace(req.MAC)
	ok, err := a.store.VerifyDeviceWhitelist(deviceID, mac)
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	if !ok {
		apiresp.Fail(c, http.StatusForbidden, 403, "device not allowed")
		return
	}
	// 记录最近一次 mac（不参与校验）
	_ = a.store.UpsertDeviceWhitelist(deviceID, mac, true, "")
	_ = a.store.UpsertDevicePubKey(deviceID, strings.TrimSpace(req.PubKey))
	tok, exp, err := a.store.CreateDeviceSession(deviceID, mac, 6*3600)
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	official, _ := a.store.ListOfficialNodes()
	publicNodes, _ := a.store.ListPublicNodes()
	payload := gin.H{
		"device_token":   tok,
		"expires_at":     time.Unix(exp, 0).UTC().Format(time.RFC3339),
		"official_nodes": official,
		"public_nodes":   publicNodes,
	}
	enc, err := cryptobox.EncryptForDevice(strings.TrimSpace(req.PubKey), deviceID, mustJSON(payload))
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	apiresp.OK(c, gin.H{"encrypted": enc})
}

type nodeInviteCreateReq struct {
	ScopeJSON  string `json:"scope_json"`
	TTLSeconds int    `json:"ttl_s"`
	MaxUses    int    `json:"max_uses"`
}

func (a *Router) handleNodeInviteCreate(c *gin.Context) {
	nodeID := strings.TrimSpace(c.GetHeader("X-Node-Id"))
	nodeKey := strings.TrimSpace(c.GetHeader("X-Node-Key"))
	if nodeID == "" || nodeKey == "" {
		apiresp.Fail(c, http.StatusUnauthorized, 401, "missing node auth headers")
		return
	}
	ok, err := a.store.VerifyNodeAuth(nodeID, nodeKey)
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	if !ok {
		apiresp.Fail(c, http.StatusUnauthorized, 401, "invalid node auth")
		return
	}
	var req nodeInviteCreateReq
	_ = c.ShouldBindJSON(&req)
	code, inviteID, exp, err := a.store.CreateInvite(nodeID, strings.TrimSpace(req.ScopeJSON), req.TTLSeconds, req.MaxUses)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	apiresp.OK(c, gin.H{
		"invite_id":  inviteID,
		"code":       code,
		"expires_at": time.Unix(exp, 0).UTC().Format(time.RFC3339),
	})
}

type nodeInviteRevokeReq struct {
	Code     string `json:"code"`
	InviteID string `json:"invite_id"`
}

func (a *Router) handleNodeInviteRevoke(c *gin.Context) {
	nodeID := strings.TrimSpace(c.GetHeader("X-Node-Id"))
	nodeKey := strings.TrimSpace(c.GetHeader("X-Node-Key"))
	if nodeID == "" || nodeKey == "" {
		apiresp.Fail(c, http.StatusUnauthorized, 401, "missing node auth headers")
		return
	}
	ok, err := a.store.VerifyNodeAuth(nodeID, nodeKey)
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	if !ok {
		apiresp.Fail(c, http.StatusUnauthorized, 401, "invalid node auth")
		return
	}
	var req nodeInviteRevokeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	if strings.TrimSpace(req.InviteID) != "" {
		if err := a.store.RevokeInviteByID(nodeID, strings.TrimSpace(req.InviteID)); err != nil {
			apiresp.Fail(c, http.StatusBadRequest, 400, err.Error())
			return
		}
	} else {
		if strings.TrimSpace(req.Code) == "" {
			apiresp.Fail(c, http.StatusBadRequest, 400, "code or invite_id required")
			return
		}
		if err := a.store.RevokeInvite(nodeID, strings.TrimSpace(req.Code)); err != nil {
			apiresp.Fail(c, http.StatusBadRequest, 400, err.Error())
			return
		}
	}
	apiresp.OK(c, gin.H{"status": "revoked"})
}

type inviteRedeemReq struct {
	Code string `json:"code" binding:"required"`
}

func (a *Router) handleInvitePreview(c *gin.Context) {
	if isAdmin, _ := c.Get("is_admin"); isAdmin == true {
		apiresp.Fail(c, http.StatusForbidden, 403, "admin not allowed")
		return
	}
	deviceIDAny, _ := c.Get("device_id")
	deviceID := strings.TrimSpace(fmt.Sprintf("%v", deviceIDAny))
	var req inviteRedeemReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	node, inviteID, expAt, err := a.store.PreviewInvite(deviceID, strings.TrimSpace(req.Code))
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	payload := gin.H{
		"node": gin.H{
			"node_id":       node.NodeID,
			"endpoints":     node.Endpoints,
			"domain_suffix": node.DomainSuffix,
			"http_enabled":  node.HTTPEnabled,
			"https_enabled": node.HTTPSEnabled,
			"tcp_port_pool": node.TCPPortPool,
			"udp_port_pool": node.UDPPortPool,
		},
		"invite_id":  inviteID,
		"expires_at": time.Unix(expAt, 0).UTC().Format(time.RFC3339),
	}
	enc, err := a.encryptForDevice(deviceID, payload)
	if err != nil {
		msg := strings.TrimSpace(err.Error())
		if strings.Contains(msg, "pub_key") || strings.Contains(msg, "device not found") {
			apiresp.Fail(c, http.StatusUnauthorized, 401, "device key missing, please re-register")
			return
		}
		apiresp.Fail(c, http.StatusInternalServerError, 500, msg)
		return
	}
	apiresp.OK(c, gin.H{"encrypted": enc})
}

func (a *Router) handleInviteRedeem(c *gin.Context) {
	if isAdmin, _ := c.Get("is_admin"); isAdmin == true {
		apiresp.Fail(c, http.StatusForbidden, 403, "admin not allowed")
		return
	}
	deviceIDAny, _ := c.Get("device_id")
	deviceID := strings.TrimSpace(fmt.Sprintf("%v", deviceIDAny))
	var req inviteRedeemReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	node, inviteID, scopeJSON, ttlSeconds, err := a.store.RedeemInvite(deviceID, strings.TrimSpace(req.Code))
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	nodeKeyPlain, err := a.store.GetNodeKeyPlain(node.NodeID)
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	ttl := time.Duration(ttlSeconds) * time.Second
	if ttl < ticketTTL() {
		ttl = ticketTTL()
	}
	tok, exp, err := ticket.IssueHMAC(node.NodeID, inviteID, scopeJSON, []byte(nodeKeyPlain), ttl)
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	payload := gin.H{
		"node": gin.H{
			"node_id":       node.NodeID,
			"endpoints":     node.Endpoints,
			"domain_suffix": node.DomainSuffix,
			"http_enabled":  node.HTTPEnabled,
			"https_enabled": node.HTTPSEnabled,
			"tcp_port_pool": node.TCPPortPool,
			"udp_port_pool": node.UDPPortPool,
		},
		"connection_ticket": tok,
		"expires_at":        exp.UTC().Format(time.RFC3339),
	}
	enc, err := a.encryptForDevice(deviceID, payload)
	if err != nil {
		msg := strings.TrimSpace(err.Error())
		if strings.Contains(msg, "pub_key") || strings.Contains(msg, "device not found") {
			apiresp.Fail(c, http.StatusUnauthorized, 401, "device key missing, please re-register")
			return
		}
		apiresp.Fail(c, http.StatusInternalServerError, 500, msg)
		return
	}
	apiresp.OK(c, gin.H{"encrypted": enc})
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func randomCode(n int) string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	if n <= 0 {
		n = 8
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err == nil {
		for i := 0; i < n; i++ {
			b[i] = alphabet[int(b[i])%len(alphabet)]
		}
		return string(b)
	}
	seed := fmt.Sprintf("%d", time.Now().UnixNano())
	out := make([]byte, 0, n)
	for len(out) < n {
		sum := sha256.Sum256([]byte(seed))
		for _, x := range sum[:] {
			out = append(out, alphabet[int(x)%len(alphabet)])
			if len(out) >= n {
				break
			}
		}
		seed = string(out)
	}
	return string(out[:n])
}

func (a *Router) encryptForDevice(deviceID string, payload any) (*cryptobox.EncryptedPayload, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil, fmt.Errorf("device_id missing")
	}
	pub, err := a.store.GetDevicePubKey(deviceID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(pub) == "" {
		return nil, fmt.Errorf("device pub_key missing")
	}
	return cryptobox.EncryptForDevice(pub, deviceID, mustJSON(payload))
}

type adminUpsertOfficialNodeReq struct {
	NodeID       string `json:"node_id" binding:"required"`
	Name         string `json:"name"`
	Server       string `json:"server" binding:"required"`
	Token        string `json:"token"`
	AdminAddr    string `json:"admin_addr"`
	AdminUser    string `json:"admin_user"`
	AdminPwd     string `json:"admin_pwd"`
	NodeAPI      string `json:"node_api"`
	DomainSuffix string `json:"domain_suffix"`
	HTTPEnabled  *bool  `json:"http_enabled"`
	HTTPSEnabled *bool  `json:"https_enabled"`
}

func (a *Router) handleAdminUpsertOfficialNode(c *gin.Context) {
	var req adminUpsertOfficialNodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	httpEnabled := false
	httpsEnabled := false
	if req.HTTPEnabled != nil {
		httpEnabled = *req.HTTPEnabled
	}
	if req.HTTPSEnabled != nil {
		httpsEnabled = *req.HTTPSEnabled
	}
	n := store.OfficialNode{
		NodeID:       strings.TrimSpace(req.NodeID),
		Name:         strings.TrimSpace(req.Name),
		Server:       strings.TrimSpace(req.Server),
		Token:        strings.TrimSpace(req.Token),
		AdminAddr:    strings.TrimSpace(req.AdminAddr),
		AdminUser:    strings.TrimSpace(req.AdminUser),
		AdminPwd:     strings.TrimSpace(req.AdminPwd),
		NodeAPI:      strings.TrimSpace(req.NodeAPI),
		DomainSuffix: strings.TrimPrefix(strings.TrimSpace(req.DomainSuffix), "."),
		HTTPEnabled:  httpEnabled,
		HTTPSEnabled: httpsEnabled,
	}
	if err := a.store.UpsertOfficialNode(n); err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	apiresp.OK(c, gin.H{"updated": true})
}

type adminDeleteOfficialNodeReq struct {
	NodeID string `json:"node_id" binding:"required"`
}

func (a *Router) handleAdminDeleteOfficialNode(c *gin.Context) {
	var req adminDeleteOfficialNodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	if err := a.store.DeleteOfficialNode(strings.TrimSpace(req.NodeID)); err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	apiresp.OK(c, gin.H{"deleted": true})
}

func (a *Router) handleAdminListOfficialNodes(c *gin.Context) {
	nodes, err := a.store.ListOfficialNodes()
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	apiresp.OK(c, gin.H{"nodes": nodes})
}

type adminUpsertWhitelistReq struct {
	DeviceID string `json:"device_id" binding:"required"`
	MAC      string `json:"mac"`
	Enabled  *bool  `json:"enabled"`
	Note     string `json:"note"`
}

func (a *Router) handleAdminUpsertWhitelist(c *gin.Context) {
	var req adminUpsertWhitelistReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	if err := a.store.UpsertDeviceWhitelist(strings.TrimSpace(req.DeviceID), strings.TrimSpace(req.MAC), enabled, strings.TrimSpace(req.Note)); err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	apiresp.OK(c, gin.H{"updated": true})
}

func (a *Router) handleAdminDeleteWhitelist(c *gin.Context) {
	var req struct {
		DeviceID string `json:"device_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	if err := a.store.DeleteDeviceWhitelist(strings.TrimSpace(req.DeviceID)); err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	apiresp.OK(c, gin.H{"deleted": true})
}

func (a *Router) handleAdminListWhitelist(c *gin.Context) {
	limit := 200
	offset := 0
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	if v := strings.TrimSpace(c.Query("offset")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			offset = n
		}
	}
	rows, total, err := a.store.ListDeviceWhitelist(limit, offset)
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	apiresp.OK(c, gin.H{"devices": rows, "total": total})
}

// handleAdminImportWhitelist 支持表格导入：CSV/纯文本
// 格式（每行）：device_id[,enabled][,note]
// enabled 支持：1/0/true/false/yes/no/on/off
func (a *Router) handleAdminImportWhitelist(c *gin.Context) {
	ct := strings.ToLower(strings.TrimSpace(c.ContentType()))
	// 1) 文件上传导入（Excel）
	if strings.HasPrefix(ct, "multipart/form-data") {
		fh, err := c.FormFile("file")
		if err != nil {
			apiresp.Fail(c, http.StatusBadRequest, 400, "file required")
			return
		}
		name := strings.ToLower(strings.TrimSpace(fh.Filename))
		file, err := fh.Open()
		if err != nil {
			apiresp.Fail(c, http.StatusBadRequest, 400, "open file failed")
			return
		}
		defer file.Close()
		b, err := io.ReadAll(file)
		if err != nil {
			apiresp.Fail(c, http.StatusBadRequest, 400, "read file failed")
			return
		}
		okCount, skipCount, err := a.importWhitelistFromExcel(name, b)
		if err != nil {
			apiresp.Fail(c, http.StatusBadRequest, 400, err.Error())
			return
		}
		apiresp.OK(c, gin.H{"imported": okCount, "skipped": skipCount})
		return
	}

	// 2) 文本导入（CSV/纯文本）
	var req struct {
		CSV string `json:"csv" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	text := strings.TrimSpace(req.CSV)
	if text == "" {
		apiresp.Fail(c, http.StatusBadRequest, 400, "csv empty")
		return
	}
	// 兼容 Windows 换行
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	lines := strings.Split(text, "\n")
	okCount := 0
	skipCount := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// 去 BOM
		if utf8.RuneCountInString(line) > 0 && strings.HasPrefix(line, "\uFEFF") {
			line = strings.TrimPrefix(line, "\uFEFF")
		}
		// 支持逗号/制表符分隔
		parts := splitCSVLoose(line)
		if len(parts) == 0 {
			continue
		}
		deviceID := strings.TrimSpace(parts[0])
		if deviceID == "" {
			skipCount++
			continue
		}
		enabled := true
		note := ""
		if len(parts) >= 2 {
			if v := strings.TrimSpace(parts[1]); v != "" {
				enabled = parseBoolLoose(v, true)
			}
		}
		if len(parts) >= 3 {
			note = strings.TrimSpace(parts[2])
		}
		if err := a.store.UpsertDeviceWhitelist(deviceID, "", enabled, note); err != nil {
			apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
			return
		}
		okCount++
	}
	apiresp.OK(c, gin.H{"imported": okCount, "skipped": skipCount})
}

func (a *Router) importWhitelistFromExcel(filename string, content []byte) (int, int, error) {
	filename = strings.ToLower(strings.TrimSpace(filename))
	if len(content) == 0 {
		return 0, 0, fmt.Errorf("file empty")
	}

	// xlsx
	if strings.HasSuffix(filename, ".xlsx") || strings.HasSuffix(filename, ".xlsm") {
		f, err := excelize.OpenReader(bytes.NewReader(content))
		if err != nil {
			return 0, 0, fmt.Errorf("invalid xlsx")
		}
		defer func() { _ = f.Close() }()
		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			return 0, 0, fmt.Errorf("empty sheet")
		}
		rows, err := f.GetRows(sheets[0])
		if err != nil {
			return 0, 0, fmt.Errorf("read sheet failed")
		}
		ok, skip := a.importWhitelistFromTable(rows)
		return ok, skip, nil
	}

	// xls（老格式）：落地到临时文件再解析
	if strings.HasSuffix(filename, ".xls") {
		wb, err := xls.OpenReader(bytes.NewReader(content))
		if err != nil {
			return 0, 0, fmt.Errorf("invalid xls")
		}
		sheets := wb.GetSheets()
		if len(sheets) == 0 {
			return 0, 0, fmt.Errorf("empty sheet")
		}
		rows := sheets[0].GetRows()
		table := make([][]string, 0, len(rows))
		for _, r := range rows {
			cols := make([]string, 0, 3)
			for c := 0; c < 3; c++ {
				if c < len(r.GetCols()) {
					cols = append(cols, strings.TrimSpace(r.GetCols()[c].GetString()))
				} else {
					cols = append(cols, "")
				}
			}
			table = append(table, cols)
		}
		ok, skip := a.importWhitelistFromTable(table)
		return ok, skip, nil
	}

	return 0, 0, fmt.Errorf("unsupported file type")
}

func (a *Router) importWhitelistFromTable(rows [][]string) (int, int) {
	okCount := 0
	skipCount := 0
	for i, parts := range rows {
		if len(parts) == 0 {
			continue
		}
		// 允许首行表头：device_id / 设备ID
		if i == 0 {
			h := strings.ToLower(strings.TrimSpace(parts[0]))
			if strings.Contains(h, "device") || strings.Contains(h, "设备") {
				continue
			}
		}
		deviceID := ""
		enabled := true
		note := ""

		if len(parts) >= 1 {
			deviceID = strings.TrimSpace(parts[0])
		}
		if deviceID == "" {
			skipCount++
			continue
		}
		if len(parts) >= 2 {
			if v := strings.TrimSpace(parts[1]); v != "" {
				enabled = parseBoolLoose(v, true)
			}
		}
		if len(parts) >= 3 {
			note = strings.TrimSpace(parts[2])
		}
		if err := a.store.UpsertDeviceWhitelist(deviceID, "", enabled, note); err != nil {
			// 不中断整批导入：按跳过计数
			skipCount++
			continue
		}
		okCount++
	}
	return okCount, skipCount
}

func splitCSVLoose(line string) []string {
	// 优先用逗号，否则用制表符；不支持带引号的复杂 CSV（够用即可）
	if strings.Contains(line, ",") {
		return strings.Split(line, ",")
	}
	if strings.Contains(line, "\t") {
		return strings.Split(line, "\t")
	}
	return []string{line}
}

func parseBoolLoose(v string, def bool) bool {
	s := strings.ToLower(strings.TrimSpace(v))
	switch s {
	case "1", "true", "yes", "y", "on", "enable", "enabled":
		return true
	case "0", "false", "no", "n", "off", "disable", "disabled":
		return false
	default:
		return def
	}
}

func jsonMarshal(v any) (json.RawMessage, error) {
	if v == nil {
		return json.RawMessage("null"), nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("null"), err
	}
	return json.RawMessage(b), nil
}

type adminUpsertNodeReq struct {
	NodeID  string `json:"node_id" binding:"required"`
	NodeKey string `json:"node_key" binding:"required"`
}

func (a *Router) handleAdminUpsertNode(c *gin.Context) {
	var req adminUpsertNodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	nodeID := strings.TrimSpace(req.NodeID)
	nodeKey := strings.TrimSpace(req.NodeKey)
	if nodeID == "" || nodeKey == "" {
		apiresp.Fail(c, http.StatusBadRequest, 400, "node_id/node_key required")
		return
	}
	if err := a.store.UpsertNodeAuth(nodeID, nodeKey); err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	apiresp.OK(c, gin.H{"updated": true})
}
