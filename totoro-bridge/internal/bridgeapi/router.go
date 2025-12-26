package bridgeapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"totoro-bridge/internal/apiresp"
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

	v1 := r.Group("/api/v1")
	{
		v1.GET("/public/nodes", api.handlePublicNodes)
		v1.POST("/nodes/heartbeat", api.handleNodeHeartbeat)

		// admin
		v1.POST("/admin/nodes/upsert", api.authAdmin(), api.handleAdminUpsertNode)
	}

	return r
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

type heartbeatReq struct {
	Ts      string         `json:"ts"`
	NodeID  string         `json:"node_id"`
	Public  bool           `json:"public"`
	Name    string         `json:"name"`
	Region  string         `json:"region"`
	ISP     string         `json:"isp"`
	Tags    []string       `json:"tags"`
	Endpoints    []store.NodeEndpoint `json:"endpoints"`
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


