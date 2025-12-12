package api

import (
	"database/sql"
	"nwct/client-nps/config"
	"nwct/client-nps/internal/mqtt"
	"nwct/client-nps/internal/network"
	"nwct/client-nps/internal/nps"
	"nwct/client-nps/internal/scanner"

	"github.com/gin-gonic/gin"
)

// 为了简化，将handlers直接放在server.go中
// 实际应该分离到handlers包

// Server API服务器
type Server struct {
	config     *config.Config
	db         *sql.DB
	netManager network.Manager
	npsClient  nps.Client
	mqttClient mqtt.Client
	scanner    scanner.Scanner
	router     *gin.Engine
}

// NewServer 创建API服务器
func NewServer(cfg *config.Config, db *sql.DB, netManager network.Manager, npsClient nps.Client, mqttClient mqtt.Client) *Server {
	// 初始化扫描器
	deviceScanner := scanner.NewScanner(db)

	server := &Server{
		config:     cfg,
		db:         db,
		netManager: netManager,
		npsClient:  npsClient,
		mqttClient: mqttClient,
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
	s.router.Use(gin.Logger())
	s.router.Use(gin.Recovery())

	// CORS中间件
	s.router.Use(corsMiddleware())

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

		// 网络管理
		api.GET("/network/interfaces", s.authMiddleware(), s.handleNetworkInterfaces)
		api.GET("/network/wifi/profiles", s.authMiddleware(), s.handleWiFiProfilesList)
		api.POST("/network/wifi/profiles", s.authMiddleware(), s.handleWiFiProfilesUpsert)
		api.DELETE("/network/wifi/profiles", s.authMiddleware(), s.handleWiFiProfilesDelete)
		api.POST("/network/wifi/connect", s.authMiddleware(), s.handleWiFiConnect)
		api.GET("/network/wifi/scan", s.authMiddleware(), s.handleWiFiScan)
		api.GET("/network/status", s.authMiddleware(), s.handleNetworkStatus)

		// 设备扫描
		api.GET("/devices", s.authMiddleware(), s.handleDevicesList)
		api.GET("/devices/:ip", s.authMiddleware(), s.handleDeviceDetail)
		api.POST("/devices/scan/start", s.authMiddleware(), s.handleScanStart)
		api.POST("/devices/scan/stop", s.authMiddleware(), s.handleScanStop)
		api.GET("/devices/scan/status", s.authMiddleware(), s.handleScanStatus)

		// 网络工具箱
		api.POST("/tools/ping", s.authMiddleware(), s.handlePing)
		api.POST("/tools/traceroute", s.authMiddleware(), s.handleTraceroute)
		api.POST("/tools/speedtest", s.authMiddleware(), s.handleSpeedTest)
		api.POST("/tools/portscan", s.authMiddleware(), s.handlePortScan)
		api.POST("/tools/dns", s.authMiddleware(), s.handleDNS)

		// NPS管理
		api.GET("/nps/status", s.authMiddleware(), s.handleNPSStatus)
		api.POST("/nps/connect", s.authMiddleware(), s.handleNPSConnect)
		api.POST("/nps/disconnect", s.authMiddleware(), s.handleNPSDisconnect)
		api.GET("/nps/tunnels", s.authMiddleware(), s.handleNPSTunnels)

		// MQTT管理
		api.GET("/mqtt/status", s.authMiddleware(), s.handleMQTTStatus)
		api.POST("/mqtt/connect", s.authMiddleware(), s.handleMQTTConnect)
		api.POST("/mqtt/disconnect", s.authMiddleware(), s.handleMQTTDisconnect)
		api.POST("/mqtt/publish", s.authMiddleware(), s.handleMQTTPublish)
		api.GET("/mqtt/logs", s.authMiddleware(), s.handleMQTTLogs)

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
}

