//go:build !device_minimal

package api

import "github.com/gin-gonic/gin"

// registerToolRoutes 注册网络工具箱路由（非 minimal 版本）
func (s *Server) registerToolRoutes(api *gin.RouterGroup) {
	api.POST("/tools/ping", s.authMiddleware(), s.handlePing)
	api.POST("/tools/traceroute", s.authMiddleware(), s.handleTraceroute)
	api.POST("/tools/speedtest", s.authMiddleware(), s.handleSpeedTest)
}
