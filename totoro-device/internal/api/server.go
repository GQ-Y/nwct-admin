package api

import (
	"database/sql"
	"io/fs"
	"net/http"
	"totoro-device/config"
	"totoro-device/internal/network"
	"totoro-device/internal/frp"
	"totoro-device/internal/scanner"
	"totoro-device/internal/webui"
	"totoro-device/models"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

// 为了简化，将handlers直接放在server.go中
// 实际应该分离到handlers包

// Server API服务器
type Server struct {
	config     *config.Config
	db         *sql.DB
	netManager network.Manager
	frpClient  frp.Client
	scanner    scanner.Scanner
	router     *gin.Engine
}

// NewServer 创建API服务器
func NewServer(cfg *config.Config, db *sql.DB, netManager network.Manager, frpClient frp.Client) *Server {
	// 初始化扫描器
	deviceScanner := scanner.NewScanner(db)

	server := &Server{
		config:     cfg,
		db:         db,
		netManager: netManager,
		frpClient:  frpClient,
		scanner:    deviceScanner,
	}

	// 初始化路由
	server.initRouter()

	return server
}

// Router 获取路由
func (s *Server) Router() *gin.Engine {
	return s.router
}

// initRouter 初始化路由
func (s *Server) initRouter() {
	gin.SetMode(gin.ReleaseMode)
	s.router = gin.New()
	// 移除 Logger 中间件以节省内存（生产环境通常不需要详细日志）
	// s.router.Use(gin.Logger())
	s.router.Use(gin.Recovery())

	// CORS中间件
	s.router.Use(corsMiddleware())

	// Web UI（embed 静态文件 + SPA 回退）
	distFS := webui.DistFS()
	fileServer := http.FileServer(http.FS(distFS))
	serveFromDist := func(c *gin.Context, urlPath string) {
		r := c.Request.Clone(c.Request.Context())
		r.URL.Path = urlPath
		fileServer.ServeHTTP(c.Writer, r)
	}

	// API路由组
	api := s.router.Group("/api/v1")
	{
		// 认证路由（不需要JWT）
		api.POST("/auth/login", s.handleLogin)
		api.POST("/auth/logout", s.handleLogout)
		api.POST("/auth/change-password", s.authMiddleware(), s.handleChangePassword)

		// 系统管理
		api.GET("/system/info", s.authMiddleware(), s.handleSystemInfo)
		api.POST("/system/restart", s.authMiddleware(), s.handleSystemRestart)
		api.GET("/system/logs", s.authMiddleware(), s.handleSystemLogs)
		api.POST("/system/logs/clear", s.authMiddleware(), s.handleSystemLogsClear)

		// 网络管理
		api.GET("/network/interfaces", s.authMiddleware(), s.handleNetworkInterfaces)
		api.GET("/network/wifi/profiles", s.authMiddleware(), s.handleWiFiProfilesList)
		api.POST("/network/wifi/profiles", s.authMiddleware(), s.handleWiFiProfilesUpsert)
		api.DELETE("/network/wifi/profiles", s.authMiddleware(), s.handleWiFiProfilesDelete)
		api.POST("/network/wifi/connect", s.authMiddleware(), s.handleWiFiConnect)
		api.GET("/network/wifi/scan", s.authMiddleware(), s.handleWiFiScan)
		api.GET("/network/status", s.authMiddleware(), s.handleNetworkStatus)
		api.POST("/network/apply", s.authMiddleware(), s.handleNetworkApply)

		// 设备扫描
		api.GET("/devices", s.authMiddleware(), s.handleDevicesList)
		api.GET("/devices/activity", s.authMiddleware(), s.handleDevicesActivity)
		api.GET("/devices/:ip", s.authMiddleware(), s.handleDeviceDetail)
		api.POST("/devices/:ip/ports/scan", s.authMiddleware(), s.handleDevicePortScan)
		api.POST("/devices/scan/start", s.authMiddleware(), s.handleScanStart)
		api.POST("/devices/scan/stop", s.authMiddleware(), s.handleScanStop)
		api.GET("/devices/scan/status", s.authMiddleware(), s.handleScanStatus)

		// 网络工具箱
		api.POST("/tools/ping", s.authMiddleware(), s.handlePing)
		api.POST("/tools/traceroute", s.authMiddleware(), s.handleTraceroute)
		api.POST("/tools/speedtest", s.authMiddleware(), s.handleSpeedTest)
		api.POST("/tools/portscan", s.authMiddleware(), s.handlePortScan)
		api.POST("/tools/dns", s.authMiddleware(), s.handleDNS)

		// FRP管理
		api.GET("/frp/status", s.authMiddleware(), s.handleFRPStatus)
		api.POST("/frp/connect", s.authMiddleware(), s.handleFRPConnect)
		api.POST("/frp/disconnect", s.authMiddleware(), s.handleFRPDisconnect)
		api.GET("/frp/tunnels", s.authMiddleware(), s.handleFRPTunnels)
		api.POST("/frp/tunnels", s.authMiddleware(), s.handleFRPAddTunnel)
		api.DELETE("/frp/tunnels/:name", s.authMiddleware(), s.handleFRPRemoveTunnel)
		api.PUT("/frp/tunnels/:name", s.authMiddleware(), s.handleFRPUpdateTunnel)
		api.POST("/frp/reload", s.authMiddleware(), s.handleFRPReload)

		// 配置管理
		api.GET("/config", s.authMiddleware(), s.handleConfigGet)
		api.POST("/config", s.authMiddleware(), s.handleConfigUpdate)
		api.POST("/config/init", s.handleConfigInit)
		api.GET("/config/init/status", s.handleConfigInitStatus)
		api.GET("/config/export", s.authMiddleware(), s.handleConfigExport)
		api.POST("/config/import", s.authMiddleware(), s.handleConfigImport)
	}

	// WebSocket路由
	s.router.GET("/ws", s.handleWebSocket)

	// 未命中路由：静态资源优先，其次 SPA 回退到 index.html
	s.router.NoRoute(func(c *gin.Context) {
		// API 未命中：返回 JSON 404，避免被前端 index.html “吞掉”
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, models.ErrorResponse(404, "not found"))
			return
		}
		// WS 未命中：也返回 404
		if strings.HasPrefix(c.Request.URL.Path, "/ws") {
			c.JSON(http.StatusNotFound, models.ErrorResponse(404, "not found"))
			return
		}

		// 仅对 GET/HEAD 提供静态页面回退，其它方法保持 404
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.JSON(http.StatusNotFound, models.ErrorResponse(404, "not found"))
			return
		}

		reqPath := c.Request.URL.Path
		clean := path.Clean("/" + strings.TrimPrefix(reqPath, "/"))
		if clean == "/" {
			// 直接用目录方式交给 http.FileServer 处理 index.html，
			// 避免触发它对 /index.html 的“重定向到 ./”规范化逻辑导致循环跳转。
			serveFromDist(c, "/")
			return
		}

		rel := strings.TrimPrefix(clean, "/")
		if st, err := fs.Stat(distFS, rel); err == nil && st != nil && !st.IsDir() {
			serveFromDist(c, clean)
			return
		}
		// 目录路径：尝试 /dir/index.html（兼容部分静态资源布局）
		if st, err := fs.Stat(distFS, path.Join(rel, "index.html")); err == nil && st != nil && !st.IsDir() {
			serveFromDist(c, path.Join(clean, "index.html"))
			return
		}

		// SPA 回退
		serveFromDist(c, "/")
	})
}
