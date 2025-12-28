//go:build device_minimal

package api

import "github.com/gin-gonic/gin"

// registerToolRoutes 注册网络工具箱路由（minimal 版本，仅保留核心功能）
func (s *Server) registerToolRoutes(api *gin.RouterGroup) {
	// minimal 版本不包含 Ping、Traceroute、SpeedTest
}
