package api

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
	"totoro-device/config"
	"totoro-device/internal/bridgeclient"
	"totoro-device/internal/database"
	"totoro-device/internal/frp"
	"totoro-device/internal/logger"
	"totoro-device/internal/network"
	"totoro-device/internal/scanner"
	"totoro-device/internal/toolkit"
	"totoro-device/internal/version"
	"totoro-device/models"
	"totoro-device/utils"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

func pickDeviceMAC(netManager network.Manager) string {
	if netManager == nil {
		return ""
	}
	interfaces, err := netManager.GetInterfaces()
	if err != nil {
		return ""
	}
	for _, i := range interfaces {
		mac := strings.TrimSpace(i.MAC)
		if mac == "" || strings.EqualFold(mac, "00:00:00:00:00:00") {
			continue
		}
		return mac
	}
	return ""
}

// handlePublicNodes 透传官方桥梁的公开节点列表
// 环境变量：TOTOTO_BRIDGE_URL，例如 http://127.0.0.1:18090
func (s *Server) handlePublicNodes(c *gin.Context) {
	bridge := strings.TrimRight(strings.TrimSpace(os.Getenv("TOTOTO_BRIDGE_URL")), "/")
	if bridge == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "未配置 TOTOTO_BRIDGE_URL"))
		return
	}
	deviceID := strings.TrimSpace(s.config.Device.ID)
	mac := strings.TrimSpace(s.config.Bridge.LastMAC)
	if mac == "" {
		// 兜底：即便 bridge 不可达，也应允许前端继续工作（实时取一次 MAC）
		mac = pickDeviceMAC(s.netManager)
		if mac != "" {
			s.config.Bridge.LastMAC = mac
		}
	}
	if deviceID == "" || mac == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "设备未初始化（缺少 device_id/mac）"))
		return
	}
	db := database.GetDB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, "数据库未初始化"))
		return
	}
	dc, err := database.GetOrCreateDeviceCrypto(db)
	if err != nil || dc == nil || strings.TrimSpace(dc.PrivKeyB64) == "" || strings.TrimSpace(dc.PubKeyB64) == "" {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, "设备密钥不可用"))
		return
	}
	var token string
	if sess, err := database.GetBridgeSession(db); err == nil && sess != nil && !database.BridgeSessionExpired(sess, 30*time.Second) {
		token = strings.TrimSpace(sess.DeviceToken)
	}
	bc := &bridgeclient.Client{
		BaseURL:          bridge,
		DeviceToken:      token,
		DeviceID:         deviceID,
		DevicePrivKeyB64: strings.TrimSpace(dc.PrivKeyB64),
	}
	// token 不可用则注册
	if strings.TrimSpace(bc.DeviceToken) == "" {
		reg, rerr := bc.Register(deviceID, mac, strings.TrimSpace(dc.PubKeyB64))
		if rerr != nil || reg == nil || strings.TrimSpace(reg.DeviceToken) == "" {
			c.JSON(http.StatusBadGateway, models.ErrorResponse(502, "桥梁注册失败"))
			return
		}
		_ = database.UpsertBridgeSession(db, database.BridgeSession{
			BridgeURL:   bridge,
			DeviceID:    deviceID,
			MAC:         mac,
			DeviceToken: strings.TrimSpace(reg.DeviceToken),
			ExpiresAt:   bridgeclient.ParseExpiresAt(reg.ExpiresAt),
		})
		bc.DeviceToken = strings.TrimSpace(reg.DeviceToken)
	}
	nodes, err := bc.GetPublicNodes()
	if err != nil {
		// 兼容：bridge 可能要求重新注册（例如缺少 pub_key 或 token 失效）
		if strings.Contains(err.Error(), "status=401") || strings.Contains(err.Error(), "status=500") {
			reg, rerr := bc.Register(deviceID, mac, strings.TrimSpace(dc.PubKeyB64))
			if rerr == nil && reg != nil && strings.TrimSpace(reg.DeviceToken) != "" {
				_ = database.UpsertBridgeSession(db, database.BridgeSession{
					BridgeURL:   bridge,
					DeviceID:    deviceID,
					MAC:         mac,
					DeviceToken: strings.TrimSpace(reg.DeviceToken),
					ExpiresAt:   bridgeclient.ParseExpiresAt(reg.ExpiresAt),
				})
				bc.DeviceToken = strings.TrimSpace(reg.DeviceToken)
				nodes2, err2 := bc.GetPublicNodes()
				if err2 == nil {
					c.JSON(http.StatusOK, models.SuccessResponse(gin.H{"nodes": nodes2}))
					return
				}
				err = err2
			}
		}
		c.JSON(http.StatusBadGateway, models.ErrorResponse(502, "请求桥梁失败: "+err.Error()))
		return
	}
	// 安全展示：公开节点列表不向前端暴露 IP/端口等敏感信息（endpoints/node_api）
	out := make([]any, 0, len(nodes))
	for _, n := range nodes {
		m, ok := n.(map[string]any)
		if !ok {
			// 兜底：非对象就直接丢弃
			continue
		}
		delete(m, "endpoints")
		delete(m, "node_api")
		out = append(out, m)
	}
	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{"nodes": out}))
}

// handleFRPConfigSave 保存 FRP 配置（manual 模式），不立即连接（连接由 /frp/connect 控制）
func (s *Server) handleFRPConfigSave(c *gin.Context) {
	var req struct {
		Server       string `json:"server"`
		Token        string `json:"token"`
		AdminAddr    string `json:"admin_addr"`
		AdminUser    string `json:"admin_user"`
		AdminPwd     string `json:"admin_pwd"`
		DomainSuffix string `json:"domain_suffix"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "参数错误: "+err.Error()))
		return
	}
	p := s.config.FRPServer.Manual
	if strings.TrimSpace(req.Server) != "" {
		p.Server = strings.TrimSpace(req.Server)
	}
	if strings.TrimSpace(req.Token) != "" || req.Token == "" {
		// 允许显式清空 token（传空字符串）
		p.Token = strings.TrimSpace(req.Token)
	}
	if strings.TrimSpace(req.AdminAddr) != "" || req.AdminAddr == "" {
		p.AdminAddr = strings.TrimSpace(req.AdminAddr)
	}
	if strings.TrimSpace(req.AdminUser) != "" || req.AdminUser == "" {
		p.AdminUser = strings.TrimSpace(req.AdminUser)
	}
	if strings.TrimSpace(req.AdminPwd) != "" || req.AdminPwd == "" {
		p.AdminPwd = strings.TrimSpace(req.AdminPwd)
	}
	if strings.TrimSpace(req.DomainSuffix) != "" || req.DomainSuffix == "" {
		p.DomainSuffix = strings.TrimPrefix(strings.TrimSpace(req.DomainSuffix), ".")
	}

	s.config.FRPServer.Mode = config.FRPModeManual
	s.config.FRPServer.Manual = p
	s.config.FRPServer.SyncActiveFromMode()
	if err := s.config.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, "保存失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{"saved": true, "mode": s.config.FRPServer.Mode}))
}

// handleFRPBuiltinUse 从桥梁同步 official_nodes 并应用为 builtin 模式
func (s *Server) handleFRPBuiltinUse(c *gin.Context) {
	bridge := strings.TrimRight(strings.TrimSpace(os.Getenv("TOTOTO_BRIDGE_URL")), "/")
	if bridge == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "未配置 TOTOTO_BRIDGE_URL"))
		return
	}
	deviceID := strings.TrimSpace(s.config.Device.ID)
	mac := strings.TrimSpace(s.config.Bridge.LastMAC)
	if mac == "" {
		mac = pickDeviceMAC(s.netManager)
	}
	if deviceID == "" || mac == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "设备未初始化（缺少 device_id/mac）"))
		return
	}
	db := database.GetDB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, "数据库未初始化"))
		return
	}
	dc, err := database.GetOrCreateDeviceCrypto(db)
	if err != nil || dc == nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, "设备密钥不可用"))
		return
	}
	// bridge session
	var token string
	if sess, _ := database.GetBridgeSession(db); sess != nil && !database.BridgeSessionExpired(sess, 30*time.Second) {
		token = strings.TrimSpace(sess.DeviceToken)
	}
	bc := &bridgeclient.Client{
		BaseURL:          bridge,
		DeviceToken:      token,
		DeviceID:         deviceID,
		DevicePrivKeyB64: strings.TrimSpace(dc.PrivKeyB64),
	}
	if strings.TrimSpace(bc.DeviceToken) == "" {
		reg, rerr := bc.Register(deviceID, mac, strings.TrimSpace(dc.PubKeyB64))
		if rerr != nil || reg == nil || strings.TrimSpace(reg.DeviceToken) == "" {
			c.JSON(http.StatusBadGateway, models.ErrorResponse(502, "桥梁注册失败"))
			return
		}
		_ = database.UpsertBridgeSession(db, database.BridgeSession{
			BridgeURL:   bridge,
			DeviceID:    deviceID,
			MAC:         mac,
			DeviceToken: strings.TrimSpace(reg.DeviceToken),
			ExpiresAt:   bridgeclient.ParseExpiresAt(reg.ExpiresAt),
		})
		bc.DeviceToken = strings.TrimSpace(reg.DeviceToken)
	}
	official, err := bc.GetOfficialNodes()
	if err != nil {
		c.JSON(http.StatusBadGateway, models.ErrorResponse(502, "请求桥梁失败: "+err.Error()))
		return
	}
	if len(official) == 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "桥梁未配置官方内置节点"))
		return
	}
	off := official[0]
	s.config.FRPServer.Builtin = config.FRPProfile{
		Server:       strings.TrimSpace(off.Server),
		Token:        strings.TrimSpace(off.Token),
		AdminAddr:    strings.TrimSpace(off.AdminAddr),
		AdminUser:    strings.TrimSpace(off.AdminUser),
		AdminPwd:     strings.TrimSpace(off.AdminPwd),
		DomainSuffix: strings.TrimPrefix(strings.TrimSpace(off.DomainSuffix), "."),
		HTTPEnabled:  off.HTTPEnabled,
		HTTPSEnabled: off.HTTPSEnabled,
	}
	s.config.FRPServer.Mode = config.FRPModeBuiltin
	s.config.FRPServer.SyncActiveFromMode()
	_ = s.config.Save()
	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"applied": true,
		"mode":    s.config.FRPServer.Mode,
		"node_id": strings.TrimSpace(off.NodeID),
		"name":    strings.TrimSpace(off.Name),
	}))
}

// handlePublicNodeConnect 公开节点列表“一键连接”（无需邀请码）
func (s *Server) handlePublicNodeConnect(c *gin.Context) {
	var req struct {
		NodeID string `json:"node_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "参数错误: "+err.Error()))
		return
	}
	nodeID := strings.TrimSpace(req.NodeID)
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "node_id 不能为空"))
		return
	}
	bridge := strings.TrimRight(strings.TrimSpace(os.Getenv("TOTOTO_BRIDGE_URL")), "/")
	if bridge == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "未配置 TOTOTO_BRIDGE_URL"))
		return
	}
	deviceID := strings.TrimSpace(s.config.Device.ID)
	mac := strings.TrimSpace(s.config.Bridge.LastMAC)
	if mac == "" {
		mac = pickDeviceMAC(s.netManager)
	}
	if deviceID == "" || mac == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "设备未初始化（缺少 device_id/mac）"))
		return
	}
	db := database.GetDB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, "数据库未初始化"))
		return
	}
	dc, err := database.GetOrCreateDeviceCrypto(db)
	if err != nil || dc == nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, "设备密钥不可用"))
		return
	}
	// bridge session
	var token string
	if sess, _ := database.GetBridgeSession(db); sess != nil && !database.BridgeSessionExpired(sess, 30*time.Second) {
		token = strings.TrimSpace(sess.DeviceToken)
	}
	bc := &bridgeclient.Client{
		BaseURL:          bridge,
		DeviceToken:      token,
		DeviceID:         deviceID,
		DevicePrivKeyB64: strings.TrimSpace(dc.PrivKeyB64),
	}
	if strings.TrimSpace(bc.DeviceToken) == "" {
		reg, rerr := bc.Register(deviceID, mac, strings.TrimSpace(dc.PubKeyB64))
		if rerr != nil || reg == nil || strings.TrimSpace(reg.DeviceToken) == "" {
			c.JSON(http.StatusBadGateway, models.ErrorResponse(502, "桥梁注册失败"))
			return
		}
		_ = database.UpsertBridgeSession(db, database.BridgeSession{
			BridgeURL:   bridge,
			DeviceID:    deviceID,
			MAC:         mac,
			DeviceToken: strings.TrimSpace(reg.DeviceToken),
			ExpiresAt:   bridgeclient.ParseExpiresAt(reg.ExpiresAt),
		})
		bc.DeviceToken = strings.TrimSpace(reg.DeviceToken)
	}
	res, err := bc.ConnectPublicNode(nodeID)
	if err != nil {
		c.JSON(http.StatusBadGateway, models.ErrorResponse(502, "请求桥梁失败: "+err.Error()))
		return
	}
	if len(res.Node.Endpoints) == 0 {
		c.JSON(http.StatusBadGateway, models.ErrorResponse(502, "桥梁未返回 endpoints"))
		return
	}
	ep := res.Node.Endpoints[0]
	s.config.FRPServer.Mode = config.FRPModePublic
	s.config.FRPServer.Public.LastResolveError = ""
	s.config.FRPServer.Public.Server = fmt.Sprintf("%s:%d", strings.TrimSpace(ep.Addr), ep.Port)
	s.config.FRPServer.Public.TotoroTicket = strings.TrimSpace(res.ConnectionTicket)
	s.config.FRPServer.Public.TicketExpiresAt = strings.TrimSpace(res.ExpiresAt)
	s.config.FRPServer.Public.Token = ""
	s.config.FRPServer.Public.DomainSuffix = strings.TrimPrefix(strings.TrimSpace(res.Node.DomainSuffix), ".")
	s.config.FRPServer.SyncActiveFromMode()
	_ = s.config.Save()
	// 持久化选择的 node_id（密文），用于重启后自动续票/重连
	_ = database.SetPublicNodeID(db, nodeID)

	if err := s.frpClient.Connect(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{"connected": true}))
}

func parsePortsInputToList(v any) []int {
	// 复用 toolkit.PortScan 的 ports 解析能力（string / []int / []any），但不执行真实扫描
	// 这里用一个小技巧：调用内部 parsePorts 不可见，因此在本层实现简单解析：
	switch vv := v.(type) {
	case nil:
		return nil
	case []int:
		return vv
	case []any:
		out := make([]int, 0, len(vv))
		for _, x := range vv {
			switch n := x.(type) {
			case int:
				out = append(out, n)
			case float64:
				out = append(out, int(n))
			case string:
				// 允许字符串混入
				out = append(out, parsePortsInputToList(n)...)
			}
		}
		return out
	case string:
		s := strings.TrimSpace(vv)
		if s == "" {
			return nil
		}
		// 支持 "80,443,3000-3010"
		parts := strings.Split(s, ",")
		out := make([]int, 0, 64)
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if strings.Contains(p, "-") {
				bits := strings.Split(p, "-")
				if len(bits) != 2 {
					continue
				}
				a, err1 := strconv.Atoi(strings.TrimSpace(bits[0]))
				b, err2 := strconv.Atoi(strings.TrimSpace(bits[1]))
				if err1 != nil || err2 != nil {
					continue
				}
				if a > b {
					a, b = b, a
				}
				if a < 1 {
					a = 1
				}
				if b > 65535 {
					b = 65535
				}
				// 防止误扫超大范围：最多 2000 个端口
				if b-a+1 > 2000 {
					b = a + 1999
				}
				for i := a; i <= b; i++ {
					out = append(out, i)
				}
				continue
			}
			if n, err := strconv.Atoi(p); err == nil {
				if n >= 1 && n <= 65535 {
					out = append(out, n)
				}
			}
		}
		return out
	default:
		return nil
	}
}

// handleDevicePortScan 扫描指定设备端口（常用端口或用户指定端口列表/范围）
func (s *Server) handleDevicePortScan(c *gin.Context) {
	ip := strings.TrimSpace(c.Param("ip"))
	if ip == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "设备IP不能为空"))
		return
	}

	var req struct {
		Ports any `json:"ports"` // 支持 "80,443,3000-3010" 或 []int
	}
	_ = c.ShouldBindJSON(&req) // 允许空请求体

	ports := parsePortsInputToList(req.Ports)

	if err := s.scanner.ScanDevicePorts(ip, ports); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"status": "started",
	}))
}

func nowRFC3339() string {
	return time.Now().Format(time.RFC3339)
}

func normalizeSSID(v string) string {
	return strings.TrimSpace(v)
}

// handleLogin 处理登录请求
func (s *Server) handleLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "参数错误: "+err.Error()))
		return
	}

	// 验证用户名密码
	if req.Username != "admin" {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse(401, "用户名或密码错误"))
		return
	}

	// 验证密码
	if s.config.Auth.PasswordHash == "" {
		// 默认密码admin
		if req.Password != "admin" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse(401, "用户名或密码错误"))
			return
		}
	} else {
		if !utils.VerifyPassword(req.Password, s.config.Auth.PasswordHash) {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse(401, "用户名或密码错误"))
			return
		}
	}

	// 生成JWT Token
	token, err := utils.GenerateJWT(s.config.Device.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, "生成Token失败"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"token":      token,
		"expires_in": 3600,
	}))
}

// handleLogout 处理登出请求
func (s *Server) handleLogout(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse(nil))
}

// handleChangePassword 处理修改密码请求
func (s *Server) handleChangePassword(c *gin.Context) {
	var req struct {
		OldPassword     string `json:"old_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required"`
		ConfirmPassword string `json:"confirm_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "参数错误: "+err.Error()))
		return
	}

	if req.NewPassword != req.ConfirmPassword {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "新密码和确认密码不匹配"))
		return
	}

	// 验证旧密码
	if s.config.Auth.PasswordHash == "" {
		if req.OldPassword != "admin" {
			c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "旧密码错误"))
			return
		}
	} else {
		if !utils.VerifyPassword(req.OldPassword, s.config.Auth.PasswordHash) {
			c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "旧密码错误"))
			return
		}
	}

	// 加密新密码
	hash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, "密码加密失败"))
		return
	}

	// 更新配置
	s.config.Auth.PasswordHash = hash
	if err := s.config.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, "保存配置失败"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil))
}

// handleSystemInfo 处理获取系统信息请求
func (s *Server) handleSystemInfo(c *gin.Context) {
	// 获取CPU使用率
	cpuPercent, _ := cpu.Percent(time.Second, false)
	cpuUsage := 0.0
	if len(cpuPercent) > 0 {
		cpuUsage = cpuPercent[0]
	}

	// 获取内存使用率
	memInfo, _ := mem.VirtualMemory()
	memoryUsage := 0.0
	if memInfo != nil {
		memoryUsage = memInfo.UsedPercent
	}

	// 获取网络状态
	netStatus, _ := s.netManager.GetNetworkStatus()

	// uptime（秒）
	uptimeSec := uint64(0)
	startTime := ""
	if up, err := host.Uptime(); err == nil {
		uptimeSec = up
	}
	if bt, err := host.BootTime(); err == nil && bt > 0 {
		startTime = time.Unix(int64(bt), 0).Format(time.RFC3339)
	}

	// disk usage（根目录）
	diskUsage := 0.0
	if du, err := disk.Usage("/"); err == nil && du != nil {
		diskUsage = du.UsedPercent
	}

	hostname, _ := os.Hostname()
	sshListening := false
	if conn, err := net.DialTimeout("tcp", "127.0.0.1:22", 200*time.Millisecond); err == nil {
		sshListening = true
		_ = conn.Close()
	}

	info := gin.H{
		"hostname":         hostname,
		"device_id":        s.config.Device.ID,
		"firmware_version": version.Version,
		"build_time":       version.BuildTime,
		"commit":           version.Commit,
		"uptime":           uptimeSec,
		"start_time":       startTime,
		"cpu_usage":        cpuUsage,
		"memory_usage":     memoryUsage,
		"disk_usage":       diskUsage,
		"ssh": gin.H{
			"listening": sshListening,
			"port":      22,
		},
		"network": gin.H{
			"interface": netStatus.CurrentInterface,
			"ip":        netStatus.IP,
			"status":    netStatus.Status,
		},
	}

	c.JSON(http.StatusOK, models.SuccessResponse(info))
}

// handleSystemRestart 处理重启请求
func (s *Server) handleSystemRestart(c *gin.Context) {
	var req struct {
		Type string `json:"type"` // soft, hard
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.Type = "soft"
	}

	logger.Info("收到重启请求，类型: %s", req.Type)

	// 为了让 API 立即返回，重启命令放到 goroutine
	go func(t string) {
		var cmd *exec.Cmd
		if t == "soft" {
			// soft：重启本进程（由外部守护进程拉起），此处先退出
			time.Sleep(500 * time.Millisecond)
			os.Exit(0)
			return
		}

		// hard：调用系统重启命令（可能需要 root 权限）
		switch runtime.GOOS {
		case "linux":
			cmd = exec.Command("shutdown", "-r", "now")
		case "darwin":
			cmd = exec.Command("shutdown", "-r", "now")
		default:
			logger.Error("不支持的系统重启平台: %s", runtime.GOOS)
			return
		}
		_ = cmd.Run()
	}(req.Type)

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"message": "重启命令已发送",
	}))
}

// handleSystemLogs 处理获取系统日志请求
func (s *Server) handleSystemLogs(c *gin.Context) {
	// query: lines=200
	lines := 200
	if v := strings.TrimSpace(c.Query("lines")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 2000 {
			lines = n
		}
	}

	// 推断日志路径：优先 NWCT_LOG_DIR，否则 /var/log/nwct，否则 /tmp/nwct
	logDir := os.Getenv("NWCT_LOG_DIR")
	if logDir == "" {
		logDir = "/var/log/nwct"
	}
	logPath := filepath.Join(logDir, "system.log")
	if _, err := os.Stat(logPath); err != nil {
		alt := filepath.Join(os.TempDir(), "nwct", "system.log")
		if _, err2 := os.Stat(alt); err2 == nil {
			logPath = alt
		}
	}

	f, err := os.Open(logPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, "读取日志失败: "+err.Error()))
		return
	}
	defer f.Close()

	// 简单 tail：全量扫描，保留最后 N 行（日志文件通常不大；后续可优化为 seek）
	buf := make([]string, 0, lines)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		buf = append(buf, sc.Text())
		if len(buf) > lines {
			buf = buf[len(buf)-lines:]
		}
	}
	if err := sc.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, "读取日志失败: "+err.Error()))
		return
	}

	logs := make([]gin.H, 0, len(buf))
	for _, line := range buf {
		logs = append(logs, gin.H{"line": line})
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"logs":      logs,
		"total":     len(logs),
		"page":      1,
		"page_size": 50,
		"source":    logPath,
	}))
}

// handleSystemLogsClear 清空系统日志（截断日志文件）
func (s *Server) handleSystemLogsClear(c *gin.Context) {
	// 使用 logger 当前实际写入的日志路径，避免因 /var/log 无权限导致清空失败
	logPath := logger.CurrentLogPath()
	if strings.TrimSpace(logPath) == "" {
		logPath = filepath.Join(os.TempDir(), "nwct", "system.log")
	}

	// 截断文件（不存在则视为已清空）
	if err := os.Truncate(logPath, 0); err != nil {
		// 如果无权限，尝试回退到 /tmp/nwct/system.log
		alt := filepath.Join(os.TempDir(), "nwct", "system.log")
		if (os.IsPermission(err) || strings.Contains(strings.ToLower(err.Error()), "permission denied")) && alt != logPath {
			if err2 := os.Truncate(alt, 0); err2 == nil || os.IsNotExist(err2) {
				c.JSON(http.StatusOK, models.SuccessResponse(gin.H{"cleared": true, "path": alt}))
				return
			}
		}
		if os.IsNotExist(err) {
			c.JSON(http.StatusOK, models.SuccessResponse(gin.H{"cleared": true}))
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, "清空日志失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{"cleared": true}))
}

// handleNetworkInterfaces 处理获取网络接口列表请求
func (s *Server) handleNetworkInterfaces(c *gin.Context) {
	interfaces, err := s.netManager.GetInterfaces()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"interfaces": interfaces,
	}))
}

// handleWiFiConnect 处理WiFi连接请求
func (s *Server) handleWiFiConnect(c *gin.Context) {
	var req struct {
		SSID        string `json:"ssid" binding:"required"`
		Password    string `json:"password"`
		Security    string `json:"security"`
		Save        bool   `json:"save"`
		AutoConnect *bool  `json:"auto_connect"`
		Priority    *int   `json:"priority"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "参数错误: "+err.Error()))
		return
	}

	ssid := normalizeSSID(req.SSID)
	pass := strings.TrimSpace(req.Password)
	if err := s.netManager.ConfigureWiFi(ssid, pass); err != nil {
		// 保存失败信息到 profile（如果用户要求 save）
		if req.Save {
			p := config.WiFiProfile{
				SSID:        ssid,
				Password:    pass,
				Security:    strings.TrimSpace(req.Security),
				AutoConnect: true,
				Priority:    10,
				LastTriedAt: nowRFC3339(),
				LastError:   err.Error(),
			}
			if req.AutoConnect != nil {
				p.AutoConnect = *req.AutoConnect
			}
			if req.Priority != nil {
				p.Priority = *req.Priority
			}
			s.config.UpsertWiFiProfile(p)
			_ = s.config.Save()
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	// 连接成功：按需保存
	if req.Save {
		priority := 10
		// 如果用户没有显式指定 priority，则采用“连接成功自动置顶”策略
		if req.Priority == nil {
			priority = s.config.MaxWiFiPriority() + 1
		} else {
			priority = *req.Priority
		}

		p := config.WiFiProfile{
			SSID:          ssid,
			Password:      pass,
			Security:      strings.TrimSpace(req.Security),
			AutoConnect:   true,
			Priority:      priority,
			LastTriedAt:   nowRFC3339(),
			LastSuccessAt: nowRFC3339(),
			LastError:     "",
		}
		if req.AutoConnect != nil {
			p.AutoConnect = *req.AutoConnect
		}
		s.config.UpsertWiFiProfile(p)
		_ = s.config.Save()
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"message": "WiFi连接成功",
	}))
}

// handleWiFiProfilesList 获取已保存WiFi列表
func (s *Server) handleWiFiProfilesList(c *gin.Context) {
	list := s.config.Network.WiFiProfiles
	if list == nil {
		list = []config.WiFiProfile{}
	}
	// 出于安全考虑：不返回密码
	out := make([]gin.H, 0, len(list))
	for _, p := range list {
		out = append(out, gin.H{
			"ssid":            p.SSID,
			"security":        p.Security,
			"auto_connect":    p.AutoConnect,
			"priority":        p.Priority,
			"last_success_at": p.LastSuccessAt,
			"last_tried_at":   p.LastTriedAt,
			"last_error":      p.LastError,
		})
	}
	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{"profiles": out}))
}

// handleWiFiProfilesUpsert 新增/更新已保存WiFi（用于“手动输入SSID+密码并记忆”）
func (s *Server) handleWiFiProfilesUpsert(c *gin.Context) {
	var req struct {
		SSID        string `json:"ssid" binding:"required"`
		Password    string `json:"password"`
		Security    string `json:"security"`
		AutoConnect *bool  `json:"auto_connect"`
		Priority    *int   `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "参数错误: "+err.Error()))
		return
	}

	ssid := normalizeSSID(req.SSID)
	if ssid == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "ssid 不能为空"))
		return
	}

	p := config.WiFiProfile{
		SSID:        ssid,
		Password:    strings.TrimSpace(req.Password),
		Security:    strings.TrimSpace(req.Security),
		AutoConnect: true,
		Priority:    10,
	}
	if req.AutoConnect != nil {
		p.AutoConnect = *req.AutoConnect
	}
	if req.Priority != nil {
		p.Priority = *req.Priority
	}

	// 保留旧状态字段（如果存在）
	for _, old := range s.config.Network.WiFiProfiles {
		if old.SSID == ssid {
			p.LastSuccessAt = old.LastSuccessAt
			p.LastTriedAt = old.LastTriedAt
			p.LastError = old.LastError
			// 如果这次没有给 password，则保留旧 password
			if p.Password == "" {
				p.Password = old.Password
			}
			break
		}
	}

	s.config.UpsertWiFiProfile(p)
	if err := s.config.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, "保存失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{"message": "保存成功"}))
}

// handleWiFiProfilesDelete 删除已保存WiFi
func (s *Server) handleWiFiProfilesDelete(c *gin.Context) {
	var req struct {
		SSID string `json:"ssid" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "参数错误: "+err.Error()))
		return
	}
	ssid := normalizeSSID(req.SSID)
	if ssid == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "ssid 不能为空"))
		return
	}
	if !s.config.DeleteWiFiProfile(ssid) {
		c.JSON(http.StatusNotFound, models.ErrorResponse(404, "未找到该ssid"))
		return
	}
	if err := s.config.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, "保存失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{"message": "删除成功"}))
}

// handleWiFiScan 处理WiFi扫描请求
func (s *Server) handleWiFiScan(c *gin.Context) {
	allow := strings.EqualFold(c.Query("allow_redacted"), "true") || c.Query("allow_redacted") == "1"
	networks, err := s.netManager.ScanWiFi(network.ScanWiFiOptions{AllowRedacted: allow})
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"networks": networks,
	}))
}

// handleNetworkStatus 处理获取网络状态请求
func (s *Server) handleNetworkStatus(c *gin.Context) {
	status, err := s.netManager.GetNetworkStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(status))
}

// handleNetworkApply 实际下发 DHCP/静态 IP 配置，并回写到 config
func (s *Server) handleNetworkApply(c *gin.Context) {
	var req struct {
		Interface string `json:"interface"`
		IPMode    string `json:"ip_mode"` // dhcp/static
		IP        string `json:"ip"`
		Netmask   string `json:"netmask"`
		Gateway   string `json:"gateway"`
		DNS       string `json:"dns"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "参数错误: "+err.Error()))
		return
	}

	cfg := network.ApplyConfig{
		Interface: strings.TrimSpace(req.Interface),
		IPMode:    strings.TrimSpace(req.IPMode),
		IP:        strings.TrimSpace(req.IP),
		Netmask:   strings.TrimSpace(req.Netmask),
		Gateway:   strings.TrimSpace(req.Gateway),
		DNS:       strings.TrimSpace(req.DNS),
	}

	if err := s.netManager.ApplyNetworkConfig(cfg); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	// 回写配置（即使未初始化也允许提前保存网络配置）
	if cfg.Interface != "" {
		s.config.Network.Interface = cfg.Interface
	}
	if cfg.IPMode != "" {
		s.config.Network.IPMode = strings.ToLower(strings.TrimSpace(cfg.IPMode))
	}
	if s.config.Network.IPMode == "static" {
		s.config.Network.IP = cfg.IP
		s.config.Network.Netmask = cfg.Netmask
		s.config.Network.Gateway = cfg.Gateway
		s.config.Network.DNS = cfg.DNS
	} else {
		// dhcp：不强制清空静态字段（便于用户来回切换），这里只同步 dns
		if cfg.DNS != "" {
			s.config.Network.DNS = cfg.DNS
		}
	}
	_ = s.config.Save()

	// 返回最新网络状态
	st, _ := s.netManager.GetNetworkStatus()
	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"message": "应用成功",
		"status":  st,
	}))
}

// handleDevicesList 处理获取设备列表请求
func (s *Server) handleDevicesList(c *gin.Context) {
	devices, err := s.scanner.GetDevices()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	// 过滤
	status := c.Query("status")
	// 默认只显示在线设备（符合“默认只显示在线”的产品预期）
	if strings.TrimSpace(status) == "" {
		status = "online"
	}
	deviceType := c.Query("type")
	filtered := []scanner.Device{}
	for _, d := range devices {
		if status != "" && status != "all" && d.Status != status {
			continue
		}
		if deviceType != "" && d.Type != deviceType {
			continue
		}
		filtered = append(filtered, d)
	}

	// 分页
	page := 1
	pageSize := 20
	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if ps := c.Query("page_size"); ps != "" {
		fmt.Sscanf(ps, "%d", &pageSize)
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}

	var pagedDevices []scanner.Device
	if start < len(filtered) {
		pagedDevices = filtered[start:end]
	}

	// 转换为JSON格式
	deviceList := make([]gin.H, len(pagedDevices))
	for i, d := range pagedDevices {
		deviceList[i] = gin.H{
			"ip":         d.IP,
			"mac":        d.MAC,
			"name":       d.Name,
			"vendor":     d.Vendor,
			"model":      d.Model,
			"type":       d.Type,
			"os":         d.OS,
			"status":     d.Status,
			"open_ports": d.OpenPorts,
			"last_seen":  d.LastSeen,
			"first_seen": d.FirstSeen,
		}
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"devices":   deviceList,
		"total":     len(filtered),
		"page":      page,
		"page_size": pageSize,
	}))
}

// handleDevicesActivity 最近活动（设备上线/离线历史）
func (s *Server) handleDevicesActivity(c *gin.Context) {
	limit := 20
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	list, err := database.GetRecentActivity(s.db, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, a := range list {
		out = append(out, gin.H{
			"timestamp": a.Timestamp.Format(time.RFC3339),
			"ip":        a.IP,
			"status":    a.Status,
			"name":      a.Name,
			"vendor":    a.Vendor,
			"model":     a.Model,
		})
	}
	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{"activities": out}))
}

// handleDeviceDetail 处理获取设备详情请求
func (s *Server) handleDeviceDetail(c *gin.Context) {
	ip := c.Param("ip")

	detail, err := s.scanner.GetDeviceDetail(ip)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse(404, "设备不存在"))
		return
	}

	// 转换为JSON格式
	portList := make([]gin.H, len(detail.Ports))
	for i, p := range detail.Ports {
		portList[i] = gin.H{
			"port":     p.Port,
			"protocol": p.Protocol,
			"service":  p.Service,
			"version":  p.Version,
			"status":   p.Status,
		}
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"ip":         detail.IP,
		"mac":        detail.MAC,
		"name":       detail.Name,
		"vendor":     detail.Vendor,
		"model":      detail.Model,
		"type":       detail.Type,
		"os":         detail.OS,
		"status":     detail.Status,
		"open_ports": portList,
		"extra":      detail.Extra,
		"last_seen":  detail.LastSeen,
		"first_seen": detail.FirstSeen,
		"history":    detail.History,
	}))
}

// handleScanStart 处理启动扫描请求
func (s *Server) handleScanStart(c *gin.Context) {
	var req struct {
		Subnet  string `json:"subnet"`
		Timeout int    `json:"timeout"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许空请求体，使用默认值
	}

	// 如果没有指定网段，尝试自动检测
	if req.Subnet == "" {
		netStatus, err := s.netManager.GetNetworkStatus()
		if err == nil && netStatus.IP != "" {
			// 从IP计算网段（简化：假设/24）
			req.Subnet = netStatus.IP + "/24"
		} else {
			req.Subnet = "192.168.1.0/24" // 默认网段
		}
	}

	if err := s.scanner.StartScan(req.Subnet); err != nil {
		// 扫描已在进行中：返回当前状态（避免前端报错/重复点击导致 500）
		if strings.Contains(err.Error(), "扫描已在进行中") || strings.Contains(err.Error(), "进行中") {
			status := s.scanner.GetScanStatus()
			c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
				"scan_id":       "scan_001",
				"status":        status.Status,
				"progress":      status.Progress,
				"scanned_count": status.ScannedCount,
				"found_count":   status.FoundCount,
				"start_time":    status.StartTime.Format(time.RFC3339),
			}))
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"scan_id": "scan_001",
		"status":  "running",
	}))
}

// handleScanStop 处理停止扫描请求
func (s *Server) handleScanStop(c *gin.Context) {
	if err := s.scanner.StopScan(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil))
}

// handleScanStatus 处理获取扫描状态请求
func (s *Server) handleScanStatus(c *gin.Context) {
	status := s.scanner.GetScanStatus()

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"status":        status.Status,
		"progress":      status.Progress,
		"scanned_count": status.ScannedCount,
		"found_count":   status.FoundCount,
		"start_time":    status.StartTime.Format(time.RFC3339),
	}))
}

// handlePing 处理Ping测试请求
func (s *Server) handlePing(c *gin.Context) {
	var req struct {
		Target  string `json:"target" binding:"required"`
		Count   int    `json:"count"`
		Timeout int    `json:"timeout"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "参数错误: "+err.Error()))
		return
	}

	if req.Count <= 0 {
		req.Count = 4
	}
	if req.Timeout <= 0 {
		req.Timeout = 5
	}

	// 使用toolkit的Ping实现
	result, err := toolkit.Ping(req.Target, req.Count, time.Duration(req.Timeout)*time.Second)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(result))
}

// handleTraceroute 处理Traceroute请求
func (s *Server) handleTraceroute(c *gin.Context) {
	var req struct {
		Target  string `json:"target"`
		MaxHops int    `json:"max_hops"`
		Timeout int    `json:"timeout"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许空请求体（将自动使用网关作为目标）
	}

	if req.MaxHops <= 0 {
		req.MaxHops = 30
	}
	if req.Timeout <= 0 {
		req.Timeout = 5
	}

	target := strings.TrimSpace(req.Target)
	if target == "" {
		st, err := s.netManager.GetNetworkStatus()
		if err == nil && st != nil && strings.TrimSpace(st.Gateway) != "" {
			target = strings.TrimSpace(st.Gateway)
		}
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "未指定target，且无法获取默认网关"))
		return
	}

	// 使用toolkit的Traceroute实现
	result, err := toolkit.Traceroute(target, req.MaxHops, time.Duration(req.Timeout)*time.Second)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(result))
}

// handleSpeedTest 处理网速测试请求
func (s *Server) handleSpeedTest(c *gin.Context) {
	var req struct {
		// mode:
		// - web: 访问网站测速（DNS/TCP/TLS/TTFB/Total），默认
		// - download: 下载带宽测速（旧逻辑）
		Mode          string `json:"mode"`
		URL           string `json:"url"`
		Method        string `json:"method"` // GET(默认)/HEAD
		Count         int    `json:"count"`
		Timeout       int    `json:"timeout"` // 秒
		DownloadBytes int64  `json:"download_bytes"`

		// 旧字段兼容（download 模式使用）
		Server   string `json:"server"`
		TestType string `json:"test_type"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许空请求体
		req.Mode = "web"
	}

	mode := strings.TrimSpace(req.Mode)
	if mode == "" {
		mode = "web"
	}

	switch mode {
	case "download":
		// 使用toolkit的下载测速（兼容旧面板/脚本）
		if req.TestType == "" {
			req.TestType = "download"
		}
		if req.Server == "" {
			req.Server = "default"
		}
		result, err := toolkit.SpeedTest(req.Server, req.TestType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
			return
		}
		c.JSON(http.StatusOK, models.SuccessResponse(result))
		return
	case "web":
		fallthrough
	default:
		if req.Count <= 0 {
			req.Count = 3
		}
		if req.Timeout <= 0 {
			req.Timeout = 8
		}
		// 兼容：如果 url 为空但 server 有值，把 server 当作 url
		targetURL := strings.TrimSpace(req.URL)
		if targetURL == "" {
			targetURL = strings.TrimSpace(req.Server)
		}
		method := strings.TrimSpace(req.Method)
		if method == "" {
			method = "GET"
		}
		result, err := toolkit.WebSpeedTestWithOptions(targetURL, method, req.Count, time.Duration(req.Timeout)*time.Second, req.DownloadBytes)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
			return
		}
		c.JSON(http.StatusOK, models.SuccessResponse(result))
		return
	}
}

// handlePortScan 处理端口扫描请求
func (s *Server) handlePortScan(c *gin.Context) {
	var req struct {
		Target   string      `json:"target" binding:"required"`
		Ports    interface{} `json:"ports"`
		Timeout  int         `json:"timeout"`
		ScanType string      `json:"scan_type"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "参数错误: "+err.Error()))
		return
	}

	if req.Timeout <= 0 {
		req.Timeout = 5
	}
	if req.ScanType == "" {
		req.ScanType = "tcp"
	}

	// 使用toolkit的端口扫描
	result, err := toolkit.PortScan(req.Target, req.Ports, time.Duration(req.Timeout)*time.Second, req.ScanType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(result))
}

// handleDNS 处理DNS查询请求
func (s *Server) handleDNS(c *gin.Context) {
	var req struct {
		Query  string `json:"query" binding:"required"`
		Type   string `json:"type"`
		Server string `json:"server"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "参数错误: "+err.Error()))
		return
	}

	if req.Type == "" {
		req.Type = "A"
	}

	// 使用toolkit的DNS查询
	records, err := toolkit.DNSQuery(req.Query, req.Type, req.Server)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"query":   req.Query,
		"type":    req.Type,
		"records": records,
	}))
}

// handleFRPStatus 处理获取FRP状态请求
func (s *Server) handleFRPStatus(c *gin.Context) {
	status, err := s.frpClient.GetStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}
	// 显示策略：只有手动配置显示真实 server；其余模式显示“云节点”标签，避免暴露 IP/端口
	mode := s.config.FRPServer.Mode
	serverRaw := strings.TrimSpace(status.Server)
	display := ""
	source := ""
	switch mode {
	case config.FRPModeManual:
		display = serverRaw
		source = "manual"
	case config.FRPModeBuiltin:
		display = "Totoro云节点"
		source = "builtin"
	case config.FRPModePublic:
		// public 可能是：公开节点直连 或 邀请码（私有分享）
		db := database.GetDB()
		code := ""
		nodeID := ""
		if db != nil {
			code, _ = database.GetPublicInviteCode(db)
			nodeID, _ = database.GetPublicNodeID(db)
		}
		if strings.TrimSpace(code) != "" {
			display = "私有分享云节点"
			source = "invite"
		} else if strings.TrimSpace(nodeID) != "" {
			display = "公开云节点"
			source = "public"
		} else {
			display = "公开云节点"
			source = "public"
		}
	default:
		display = "Totoro云节点"
		source = "unknown"
	}

	serverOut := status.Server
	if mode != config.FRPModeManual {
		// 非手动模式禁止暴露 server
		serverOut = ""
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"connected":      status.Connected,
		"server":         serverOut,
		"connected_at":   status.ConnectedAt,
		"pid":            status.PID,
		"last_error":     status.LastError,
		"tunnels":        status.Tunnels,
		"log_path":       status.LogPath,
		"display_server": display,
		"mode":           string(mode),
		"source":         source,
	}))
}

// handleFRPConnect 处理FRP连接请求
func (s *Server) handleFRPConnect(c *gin.Context) {
	var req struct {
		Server       string `json:"server"`
		Token        string `json:"token"`
		TotoroTicket string `json:"totoro_ticket"`
		AdminAddr    string `json:"admin_addr"`
		AdminUser    string `json:"admin_user"`
		AdminPwd     string `json:"admin_pwd"`
	}

	_ = c.ShouldBindJSON(&req) // 允许空请求体：使用“当前模式”一键连接

	// 如果请求显式传了 server/token 等，认为是“手动模式”
	if strings.TrimSpace(req.Server) != "" || strings.TrimSpace(req.Token) != "" || strings.TrimSpace(req.AdminAddr) != "" || strings.TrimSpace(req.AdminUser) != "" || strings.TrimSpace(req.AdminPwd) != "" || strings.TrimSpace(req.TotoroTicket) != "" {
		s.config.FRPServer.Mode = config.FRPModeManual

		// 合并 manual profile：请求未提供的字段则沿用已有 manual
		p := s.config.FRPServer.Manual
		if strings.TrimSpace(req.Server) != "" {
			p.Server = strings.TrimSpace(req.Server)
		}
		if strings.TrimSpace(req.Token) != "" {
			p.Token = strings.TrimSpace(req.Token)
		}
		if strings.TrimSpace(req.TotoroTicket) != "" {
			p.TotoroTicket = strings.TrimSpace(req.TotoroTicket)
		}
		if strings.TrimSpace(req.AdminAddr) != "" {
			p.AdminAddr = strings.TrimSpace(req.AdminAddr)
		}
		if strings.TrimSpace(req.AdminUser) != "" {
			p.AdminUser = strings.TrimSpace(req.AdminUser)
		}
		if strings.TrimSpace(req.AdminPwd) != "" {
			p.AdminPwd = strings.TrimSpace(req.AdminPwd)
		}
		// domain_suffix 仍通过 /config 更新（这里不改动）
		s.config.FRPServer.Manual = p
		s.config.FRPServer.SyncActiveFromMode()
		_ = s.config.Save()
	} else {
		// 走当前模式
		s.config.FRPServer.SyncActiveFromMode()

		// public 模式：如果 ticket 缺失/过期，尝试自动换票
		if s.config.FRPServer.Mode == config.FRPModePublic {
			db := database.GetDB()
			if db == nil {
				c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, "数据库未初始化"))
				return
			}
			code, _ := database.GetPublicInviteCode(db)
			if strings.TrimSpace(code) == "" {
				c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "public 模式未配置邀请码"))
				return
			}
			// ticket 缺失/过期 -> bridge 兑换
			if strings.TrimSpace(s.config.FRPServer.Public.TotoroTicket) == "" || frp.TicketExpired(s.config.FRPServer.Public.TicketExpiresAt, 30*time.Second) {
				bridgeBase := strings.TrimRight(strings.TrimSpace(os.Getenv("TOTOTO_BRIDGE_URL")), "/")
				if bridgeBase == "" {
					c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "未配置 TOTOTO_BRIDGE_URL"))
					return
				}
				dc, err := database.GetOrCreateDeviceCrypto(db)
				if err != nil || dc == nil {
					c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, "设备密钥不可用"))
					return
				}
				sess, _ := database.GetBridgeSession(db)
				if sess == nil || database.BridgeSessionExpired(sess, 30*time.Second) {
					// 尝试重新注册（需要 mac）
					bc := &bridgeclient.Client{
						BaseURL:          bridgeBase,
						DeviceID:         strings.TrimSpace(s.config.Device.ID),
						DevicePrivKeyB64: strings.TrimSpace(dc.PrivKeyB64),
					}
					reg, rerr := bc.Register(strings.TrimSpace(s.config.Device.ID), strings.TrimSpace(s.config.Bridge.LastMAC), strings.TrimSpace(dc.PubKeyB64))
					if rerr != nil || reg == nil || strings.TrimSpace(reg.DeviceToken) == "" {
						c.JSON(http.StatusBadGateway, models.ErrorResponse(502, "桥梁注册失败"))
						return
					}
					_ = database.UpsertBridgeSession(db, database.BridgeSession{
						BridgeURL:   bridgeBase,
						DeviceID:    strings.TrimSpace(s.config.Device.ID),
						MAC:         strings.TrimSpace(s.config.Bridge.LastMAC),
						DeviceToken: strings.TrimSpace(reg.DeviceToken),
						ExpiresAt:   bridgeclient.ParseExpiresAt(reg.ExpiresAt),
					})
					sess, _ = database.GetBridgeSession(db)
				}
				if sess == nil || strings.TrimSpace(sess.DeviceToken) == "" {
					c.JSON(http.StatusBadGateway, models.ErrorResponse(502, "缺少 device_token"))
					return
				}
				bc := &bridgeclient.Client{
					BaseURL:          bridgeBase,
					DeviceToken:      strings.TrimSpace(sess.DeviceToken),
					DeviceID:         strings.TrimSpace(s.config.Device.ID),
					DevicePrivKeyB64: strings.TrimSpace(dc.PrivKeyB64),
				}
				res, err := bc.RedeemInvite(code)
				if err != nil {
					s.config.FRPServer.Public.LastResolveError = err.Error()
					c.JSON(http.StatusBadGateway, models.ErrorResponse(502, "自动换票失败: "+err.Error()))
					return
				}
				s.config.FRPServer.Public.LastResolveError = ""
				if len(res.Node.Endpoints) == 0 {
					c.JSON(http.StatusBadGateway, models.ErrorResponse(502, "桥梁未返回 endpoints"))
					return
				}
				ep := res.Node.Endpoints[0]
				s.config.FRPServer.Public.Server = fmt.Sprintf("%s:%d", strings.TrimSpace(ep.Addr), ep.Port)
				s.config.FRPServer.Public.TotoroTicket = strings.TrimSpace(res.ConnectionTicket)
				s.config.FRPServer.Public.TicketExpiresAt = strings.TrimSpace(res.ExpiresAt)
				s.config.FRPServer.Public.DomainSuffix = strings.TrimPrefix(strings.TrimSpace(res.Node.DomainSuffix), ".")
				s.config.FRPServer.Public.Token = ""
				s.config.FRPServer.SyncActiveFromMode()
			}
		}
	}

	if err := s.frpClient.Connect(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	_ = s.config.Save()

	c.JSON(http.StatusOK, models.SuccessResponse(nil))
}

// handleInviteConnect 通过桥梁兑换邀请码获取 ticket，并一键连接到该 frps 节点
func (s *Server) handleInviteConnect(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "参数错误: "+err.Error()))
		return
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "code 不能为空"))
		return
	}
	db := database.GetDB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, "数据库未初始化"))
		return
	}
	_ = database.SetPublicInviteCode(db, code) // 仅存密文
	dc, err := database.GetOrCreateDeviceCrypto(db)
	if err != nil || dc == nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, "设备密钥不可用"))
		return
	}
	bridgeBase := strings.TrimRight(strings.TrimSpace(os.Getenv("TOTOTO_BRIDGE_URL")), "/")
	if bridgeBase == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "未配置 TOTOTO_BRIDGE_URL"))
		return
	}
	sess, _ := database.GetBridgeSession(db)
	if sess == nil || database.BridgeSessionExpired(sess, 30*time.Second) {
		bc0 := &bridgeclient.Client{
			BaseURL:          bridgeBase,
			DeviceID:         strings.TrimSpace(s.config.Device.ID),
			DevicePrivKeyB64: strings.TrimSpace(dc.PrivKeyB64),
		}
		reg, rerr := bc0.Register(strings.TrimSpace(s.config.Device.ID), strings.TrimSpace(s.config.Bridge.LastMAC), strings.TrimSpace(dc.PubKeyB64))
		if rerr != nil || reg == nil || strings.TrimSpace(reg.DeviceToken) == "" {
			c.JSON(http.StatusBadGateway, models.ErrorResponse(502, "桥梁注册失败"))
			return
		}
		_ = database.UpsertBridgeSession(db, database.BridgeSession{
			BridgeURL:   bridgeBase,
			DeviceID:    strings.TrimSpace(s.config.Device.ID),
			MAC:         strings.TrimSpace(s.config.Bridge.LastMAC),
			DeviceToken: strings.TrimSpace(reg.DeviceToken),
			ExpiresAt:   bridgeclient.ParseExpiresAt(reg.ExpiresAt),
		})
		sess, _ = database.GetBridgeSession(db)
	}
	if sess == nil || strings.TrimSpace(sess.DeviceToken) == "" {
		c.JSON(http.StatusBadGateway, models.ErrorResponse(502, "缺少 device_token"))
		return
	}
	bc := &bridgeclient.Client{
		BaseURL:          bridgeBase,
		DeviceToken:      strings.TrimSpace(sess.DeviceToken),
		DeviceID:         strings.TrimSpace(s.config.Device.ID),
		DevicePrivKeyB64: strings.TrimSpace(dc.PrivKeyB64),
	}
	res, err := bc.RedeemInvite(code)
	if err != nil {
		c.JSON(http.StatusBadGateway, models.ErrorResponse(502, err.Error()))
		return
	}
	if len(res.Node.Endpoints) == 0 {
		c.JSON(http.StatusBadGateway, models.ErrorResponse(502, "桥梁未返回 endpoints"))
		return
	}
	ep := res.Node.Endpoints[0]
	server := fmt.Sprintf("%s:%d", strings.TrimSpace(ep.Addr), ep.Port)

	// 切换为 public 模式并持久化（下次启动自动换票 + 自动连接）
	s.config.FRPServer.Mode = config.FRPModePublic
	s.config.FRPServer.Public.LastResolveError = ""
	s.config.FRPServer.Public.Server = server
	s.config.FRPServer.Public.TotoroTicket = strings.TrimSpace(res.ConnectionTicket)
	s.config.FRPServer.Public.TicketExpiresAt = strings.TrimSpace(res.ExpiresAt)
	s.config.FRPServer.Public.Token = ""
	s.config.FRPServer.Public.DomainSuffix = strings.TrimPrefix(strings.TrimSpace(res.Node.DomainSuffix), ".")
	s.config.FRPServer.SyncActiveFromMode()
	_ = s.config.Save()

	if err := s.frpClient.Connect(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"expires_at": strings.TrimSpace(res.ExpiresAt),
	}))
}

// handleInviteResolve 仅预览邀请码并返回节点信息（不消耗次数，不兑换 ticket）
func (s *Server) handleInviteResolve(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "参数错误: "+err.Error()))
		return
	}
	code := strings.TrimSpace(req.Code)
	db := database.GetDB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, "数据库未初始化"))
		return
	}
	dc, err := database.GetOrCreateDeviceCrypto(db)
	if err != nil || dc == nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, "设备密钥不可用"))
		return
	}
	bridgeBase := strings.TrimRight(strings.TrimSpace(os.Getenv("TOTOTO_BRIDGE_URL")), "/")
	if bridgeBase == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "未配置 TOTOTO_BRIDGE_URL"))
		return
	}
	sess, _ := database.GetBridgeSession(db)
	if sess == nil || database.BridgeSessionExpired(sess, 30*time.Second) {
		bc0 := &bridgeclient.Client{
			BaseURL:          bridgeBase,
			DeviceID:         strings.TrimSpace(s.config.Device.ID),
			DevicePrivKeyB64: strings.TrimSpace(dc.PrivKeyB64),
		}
		reg, rerr := bc0.Register(strings.TrimSpace(s.config.Device.ID), strings.TrimSpace(s.config.Bridge.LastMAC), strings.TrimSpace(dc.PubKeyB64))
		if rerr != nil || reg == nil || strings.TrimSpace(reg.DeviceToken) == "" {
			c.JSON(http.StatusBadGateway, models.ErrorResponse(502, "桥梁注册失败"))
			return
		}
		_ = database.UpsertBridgeSession(db, database.BridgeSession{
			BridgeURL:   bridgeBase,
			DeviceID:    strings.TrimSpace(s.config.Device.ID),
			MAC:         strings.TrimSpace(s.config.Bridge.LastMAC),
			DeviceToken: strings.TrimSpace(reg.DeviceToken),
			ExpiresAt:   bridgeclient.ParseExpiresAt(reg.ExpiresAt),
		})
		sess, _ = database.GetBridgeSession(db)
	}
	if sess == nil || strings.TrimSpace(sess.DeviceToken) == "" {
		c.JSON(http.StatusBadGateway, models.ErrorResponse(502, "缺少 device_token"))
		return
	}
	bc := &bridgeclient.Client{
		BaseURL:          bridgeBase,
		DeviceToken:      strings.TrimSpace(sess.DeviceToken),
		DeviceID:         strings.TrimSpace(s.config.Device.ID),
		DevicePrivKeyB64: strings.TrimSpace(dc.PrivKeyB64),
	}
	res, err := bc.PreviewInvite(code)
	if err != nil {
		c.JSON(http.StatusBadGateway, models.ErrorResponse(502, err.Error()))
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"node": gin.H{
			"node_id":       strings.TrimSpace(res.Node.NodeID),
			"domain_suffix": strings.TrimSpace(res.Node.DomainSuffix),
		},
		"expires_at": strings.TrimSpace(res.ExpiresAt),
	}))
}

// handleFRPModeSet 切换 frpc 连接模式，并自动连接
func (s *Server) handleFRPModeSet(c *gin.Context) {
	var req struct {
		Mode config.FRPMode `json:"mode" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "参数错误: "+err.Error()))
		return
	}
	if req.Mode != config.FRPModeBuiltin && req.Mode != config.FRPModeManual && req.Mode != config.FRPModePublic {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "mode 无效"))
		return
	}

	s.config.FRPServer.Mode = req.Mode
	s.config.FRPServer.SyncActiveFromMode()
	_ = s.config.Save()

	// public 模式不在此处强制换票（避免必须带邀请码）；需要连接请走 /public/invites/connect 或 /frp/connect(空体)
	if req.Mode == config.FRPModePublic {
		c.JSON(http.StatusOK, models.SuccessResponse(gin.H{"mode": req.Mode}))
		return
	}

	if err := s.frpClient.Connect(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{"mode": req.Mode}))
}

// handleFRPDisconnect 处理FRP断开请求
func (s *Server) handleFRPDisconnect(c *gin.Context) {
	if err := s.frpClient.Disconnect(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil))
}

// handleFRPTunnels 处理获取FRP隧道列表请求
func (s *Server) handleFRPTunnels(c *gin.Context) {
	tunnels, err := s.frpClient.GetTunnels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"tunnels": tunnels,
	}))
}

// handleFRPAddTunnel 处理添加隧道请求
func (s *Server) handleFRPAddTunnel(c *gin.Context) {
	var req frp.Tunnel
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "参数错误: "+err.Error()))
		return
	}
	// HTTP/HTTPS 隧道能力由桥梁节点配置下发决定：未开启或缺少 domain_suffix 时禁止创建
	t := strings.ToLower(strings.TrimSpace(req.Type))
	if t == "http" {
		if !s.config.FRPServer.HTTPEnabled || strings.TrimSpace(s.config.FRPServer.DomainSuffix) == "" {
			c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "当前节点未开启 HTTP 隧道或缺少 domain_suffix"))
			return
		}
	}
	if t == "https" {
		if !s.config.FRPServer.HTTPSEnabled || strings.TrimSpace(s.config.FRPServer.DomainSuffix) == "" {
			c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "当前节点未开启 HTTPS 隧道或缺少 domain_suffix"))
			return
		}
	}

	if err := s.frpClient.AddTunnel(&req); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil))
}

// handleFRPRemoveTunnel 处理删除隧道请求
func (s *Server) handleFRPRemoveTunnel(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "隧道名称不能为空"))
		return
	}

	if err := s.frpClient.RemoveTunnel(name); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil))
}

// handleFRPUpdateTunnel 处理更新隧道请求
func (s *Server) handleFRPUpdateTunnel(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "隧道名称不能为空"))
		return
	}

	var req frp.Tunnel
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "参数错误: "+err.Error()))
		return
	}
	// 同上：禁止绕过前端创建 HTTP/HTTPS 隧道
	t := strings.ToLower(strings.TrimSpace(req.Type))
	if t == "http" {
		if !s.config.FRPServer.HTTPEnabled || strings.TrimSpace(s.config.FRPServer.DomainSuffix) == "" {
			c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "当前节点未开启 HTTP 隧道或缺少 domain_suffix"))
			return
		}
	}
	if t == "https" {
		if !s.config.FRPServer.HTTPSEnabled || strings.TrimSpace(s.config.FRPServer.DomainSuffix) == "" {
			c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "当前节点未开启 HTTPS 隧道或缺少 domain_suffix"))
			return
		}
	}

	if err := s.frpClient.UpdateTunnel(name, &req); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil))
}

// handleFRPReload 处理重载配置请求
func (s *Server) handleFRPReload(c *gin.Context) {
	if err := s.frpClient.Reload(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil))
}

// handleConfigGet 处理获取配置请求
func (s *Server) handleConfigGet(c *gin.Context) {
	// 隐藏敏感信息
	// 注意：network 里包含 wifi_profiles（内含密码），这里必须做脱敏
	net := gin.H{
		"interface": s.config.Network.Interface,
		"ip_mode":   s.config.Network.IPMode,
		"ip":        s.config.Network.IP,
		"netmask":   s.config.Network.Netmask,
		"gateway":   s.config.Network.Gateway,
		"dns":       s.config.Network.DNS,
		"wifi": gin.H{
			"ssid":     s.config.Network.WiFi.SSID,
			"password": "***",
			"security": s.config.Network.WiFi.Security,
		},
	}

	config := gin.H{
		"device": gin.H{
			"device_id": s.config.Device.ID,
			"name":      s.config.Device.Name,
		},
		"network": net,
		"frp_server": gin.H{
			"mode":          s.config.FRPServer.Mode,
			"server":        s.config.FRPServer.Server,
			"token":         "***",
			"totoro_ticket": "***",
			"admin_addr":    s.config.FRPServer.AdminAddr,
			"admin_user":    s.config.FRPServer.AdminUser,
			"domain_suffix": s.config.FRPServer.DomainSuffix,
			"http_enabled":  s.config.FRPServer.HTTPEnabled,
			"https_enabled": s.config.FRPServer.HTTPSEnabled,
			"public": gin.H{
				"node_api":          s.config.FRPServer.Public.NodeAPI,
				"invite_code":       "***",
				"ticket_expires_at": s.config.FRPServer.Public.TicketExpiresAt,
				"last_resolve_error": func() string {
					// 仅用于排障，不算敏感
					return s.config.FRPServer.Public.LastResolveError
				}(),
			},
		},
		"scanner": s.config.Scanner,
	}

	c.JSON(http.StatusOK, models.SuccessResponse(config))
}

// handleConfigUpdate 处理更新配置请求
func (s *Server) handleConfigUpdate(c *gin.Context) {
	var req config.Config
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "参数错误: "+err.Error()))
		return
	}

	// 允许更新的字段：device/network/frp_server/scanner/server/database/initialized
	// 安全：不允许通过该接口直接写入 password_hash
	s.config.Device = req.Device
	s.config.Network = req.Network
	s.config.FRPServer = req.FRPServer
	s.config.Scanner = req.Scanner
	s.config.Server = req.Server
	s.config.Database = req.Database
	s.config.Initialized = req.Initialized

	if err := s.config.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "配置校验失败: "+err.Error()))
		return
	}
	if err := s.config.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, "保存失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{"message": "更新成功"}))
}

// handleConfigInit 处理初始化配置请求
func (s *Server) handleConfigInit(c *gin.Context) {
	var req struct {
		Device        *config.DeviceConfig    `json:"device"`
		Network       *config.NetworkConfig   `json:"network"`
		FRPServer     *config.FRPServerConfig `json:"frp_server"`
		Scanner       *config.ScannerConfig   `json:"scanner"`
		AdminPassword string                  `json:"admin_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "参数错误: "+err.Error()))
		return
	}

	// 更新配置（仅覆盖传入部分）
	if req.Device != nil {
		s.config.Device = *req.Device
	}
	if req.Network != nil {
		s.config.Network = *req.Network
	}
	if req.FRPServer != nil {
		s.config.FRPServer = *req.FRPServer
	}
	if req.Scanner != nil {
		s.config.Scanner = *req.Scanner
	}

	// 设置管理员密码
	hash, err := utils.HashPassword(req.AdminPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, "密码加密失败"))
		return
	}
	s.config.Auth.PasswordHash = hash

	// 标记为已初始化
	s.config.Initialized = true

	if err := s.config.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "配置校验失败: "+err.Error()))
		return
	}
	if err := s.config.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, "保存配置失败"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil))
}

// handleConfigInitStatus 处理获取初始化状态请求
func (s *Server) handleConfigInitStatus(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"initialized": s.config.Initialized,
	}))
}

// handleConfigExport 处理导出配置请求
func (s *Server) handleConfigExport(c *gin.Context) {
	c.JSON(http.StatusOK, models.SuccessResponse(s.config))
}

// handleConfigImport 处理导入配置请求
func (s *Server) handleConfigImport(c *gin.Context) {
	var req config.Config
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "参数错误: "+err.Error()))
		return
	}

	// 安全：不允许导入覆盖 password_hash（必须走改密接口）
	req.Auth.PasswordHash = s.config.Auth.PasswordHash

	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "配置校验失败: "+err.Error()))
		return
	}

	*s.config = req
	if err := s.config.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, "保存失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{"message": "导入成功"}))
}
