package nodeapi

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"totoro-node/internal/store"
)

type Options struct {
	AdminKey string
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

	v1 := r.Group("/api/v1")
	{
		v1.GET("/node/config", api.authAdmin(), api.getNodeConfig)
		v1.POST("/node/config", api.authAdmin(), api.updateNodeConfig)
		v1.POST("/node/invites", api.authAdmin(), api.createInvite)
		v1.POST("/node/invites/revoke", api.authAdmin(), api.revokeInvite)
	}
	return r
}

func (a *API) authAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.opts.AdminKey == "" {
			c.Next()
			return
		}
		if strings.TrimSpace(c.GetHeader("X-Admin-Key")) != a.opts.AdminKey {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func (a *API) getNodeConfig(c *gin.Context) {
	cfg, _, err := a.st.GetNodeConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	// 不返回 node_key
	cfg.NodeKey = ""
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": cfg})
}

func (a *API) updateNodeConfig(c *gin.Context) {
	var req store.NodeConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	adminNodeKey := strings.TrimSpace(c.GetHeader("X-Node-Key"))
	if adminNodeKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "missing X-Node-Key"})
		return
	}
	if err := a.st.UpdateNodeConfig(adminNodeKey, req); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"updated": true}})
}

type createInviteReq struct {
	TTLSeconds int    `json:"ttl_seconds"`
	MaxUses    int    `json:"max_uses"`
	ScopeJSON  string `json:"scope_json"`
}

func (a *API) createInvite(c *gin.Context) {
	var req createInviteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	code := generateInviteCode()
	codeHash := storeHash(code)
	inv, err := a.st.CreateInvite(codeHash, req.TTLSeconds, req.MaxUses, req.ScopeJSON)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	inv.Code = code
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": inv})
}

type revokeInviteReq struct {
	InviteID string `json:"invite_id"`
}

func (a *API) revokeInvite(c *gin.Context) {
	var req revokeInviteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if strings.TrimSpace(req.InviteID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invite_id required"})
		return
	}
	if err := a.st.RevokeInvite(req.InviteID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"revoked": true}})
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


