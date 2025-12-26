package bridgeapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
	"strconv"
	"io/fs"

	"github.com/gin-gonic/gin"
	"totoro-bridge/internal/apiresp"
	"totoro-bridge/internal/bridgeui"
	"totoro-bridge/internal/store"
)

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

	// 临时 Admin UI（内嵌静态页面）
	uiFS := bridgeui.DistFS()
	r.GET("/", func(c *gin.Context) {
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
		v1.GET("/official/nodes", api.authDeviceOrAdmin(), api.handleOfficialNodes)
		v1.POST("/nodes/heartbeat", api.handleNodeHeartbeat)

		// admin
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

func (a *Router) authDeviceOrAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// admin 直接放行
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
		if strings.TrimSpace(c.GetHeader("X-Admin-Key")) != strings.TrimSpace(a.opts.AdminKey) {
			apiresp.Fail(c, http.StatusUnauthorized, 401, "unauthorized")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (a *Router) handlePublicNodes(c *gin.Context) {
	nodes, err := a.store.ListPublicNodes()
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	apiresp.OK(c, gin.H{"nodes": nodes})
}

func (a *Router) handleOfficialNodes(c *gin.Context) {
	nodes, err := a.store.ListOfficialNodes()
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	apiresp.OK(c, gin.H{"nodes": nodes})
}

type heartbeatReq struct {
	Ts      string         `json:"ts"`
	NodeID  string         `json:"node_id"`
	Public  bool           `json:"public"`
	Name    string         `json:"name"`
	Region  string         `json:"region"`
	ISP     string         `json:"isp"`
	Tags    []string       `json:"tags"`
	Endpoints    []store.NodeEndpoint `json:"endpoints"`
	NodeAPI      string              `json:"node_api"`
	DomainSuffix string              `json:"domain_suffix"`
	TCPPortPool  *store.PortPool     `json:"tcp_port_pool"`
	UDPPortPool  *store.PortPool     `json:"udp_port_pool"`
	Version any             `json:"version"`
	Metrics any             `json:"metrics"`
	Extra   any             `json:"extra"`
}

func (a *Router) handleNodeHeartbeat(c *gin.Context) {
	// node 级别鉴权：node_id + node_key（bridge 需要预先登记 node_key）
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
		Region:       strings.TrimSpace(req.Region),
		ISP:          strings.TrimSpace(req.ISP),
		Tags:         req.Tags,
		Endpoints:    req.Endpoints,
		NodeAPI:      strings.TrimSpace(req.NodeAPI),
		DomainSuffix: strings.TrimSpace(req.DomainSuffix),
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

type deviceRegisterReq struct {
	DeviceID string `json:"device_id" binding:"required"`
	MAC      string `json:"mac" binding:"required"`
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
	tok, exp, err := a.store.CreateDeviceSession(deviceID, mac, 6*3600)
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	official, _ := a.store.ListOfficialNodes()
	publicNodes, _ := a.store.ListPublicNodes()
	apiresp.OK(c, gin.H{
		"device_token": tok,
		"expires_at":   time.Unix(exp, 0).UTC().Format(time.RFC3339),
		"official_nodes": official,
		"public_nodes":   publicNodes,
	})
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
}

func (a *Router) handleAdminUpsertOfficialNode(c *gin.Context) {
	var req adminUpsertOfficialNodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, 400, err.Error())
		return
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


