package bridgeapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
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
	}

	return r
}

func (a *Router) handlePublicNodes(c *gin.Context) {
	nodes, err := a.store.ListPublicNodes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"nodes": nodes}})
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
	// node 级别鉴权：简化为 node_id + node_key 直传（后续可换成签名/时间戳）
	nodeID := strings.TrimSpace(c.GetHeader("X-Node-Id"))
	nodeKey := strings.TrimSpace(c.GetHeader("X-Node-Key"))
	if nodeID == "" || nodeKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "missing node auth headers"})
		return
	}

	var req heartbeatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if strings.TrimSpace(req.NodeID) != "" && strings.TrimSpace(req.NodeID) != nodeID {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "node_id mismatch"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"status": "ok"}})
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


