package api

import (
	"bufio"
	"fmt"
	"net/http"
	"nwct/client-nps/config"
	"nwct/client-nps/internal/database"
	"nwct/client-nps/internal/logger"
	"nwct/client-nps/internal/network"
	"nwct/client-nps/internal/scanner"
	"nwct/client-nps/internal/toolkit"
	"nwct/client-nps/models"
	"nwct/client-nps/utils"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

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

	info := gin.H{
		"device_id":        s.config.Device.ID,
		"firmware_version": "1.0.0",
		"uptime":           uptimeSec,
		"start_time":       startTime,
		"cpu_usage":        cpuUsage,
		"memory_usage":     memoryUsage,
		"disk_usage":       diskUsage,
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

// handleDevicesList 处理获取设备列表请求
func (s *Server) handleDevicesList(c *gin.Context) {
	devices, err := s.scanner.GetDevices()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	// 过滤
	status := c.Query("status")
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
		"type":       detail.Type,
		"os":         detail.OS,
		"status":     detail.Status,
		"open_ports": portList,
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
		Target  string `json:"target" binding:"required"`
		MaxHops int    `json:"max_hops"`
		Timeout int    `json:"timeout"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "参数错误: "+err.Error()))
		return
	}

	if req.MaxHops <= 0 {
		req.MaxHops = 30
	}
	if req.Timeout <= 0 {
		req.Timeout = 5
	}

	// 使用toolkit的Traceroute实现
	result, err := toolkit.Traceroute(req.Target, req.MaxHops, time.Duration(req.Timeout)*time.Second)
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

// handleNPSStatus 处理获取NPS状态请求
func (s *Server) handleNPSStatus(c *gin.Context) {
	status, err := s.npsClient.GetStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(status))
}

// handleNPSConnect 处理NPS连接请求
func (s *Server) handleNPSConnect(c *gin.Context) {
	var req struct {
		Server        string   `json:"server" binding:"required"`
		VKey          string   `json:"vkey" binding:"required"`
		ClientID      string   `json:"client_id" binding:"required"`
		NPCPath       string   `json:"npc_path"`
		NPCConfigPath string   `json:"npc_config_path"`
		NPCArgs       []string `json:"npc_args"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "参数错误: "+err.Error()))
		return
	}

	// 更新配置
	s.config.NPSServer.Server = req.Server
	s.config.NPSServer.VKey = req.VKey
	s.config.NPSServer.ClientID = req.ClientID
	if strings.TrimSpace(req.NPCPath) != "" {
		s.config.NPSServer.NPCPath = strings.TrimSpace(req.NPCPath)
	}
	if strings.TrimSpace(req.NPCConfigPath) != "" {
		s.config.NPSServer.NPCConfigPath = strings.TrimSpace(req.NPCConfigPath)
	}
	if req.NPCArgs != nil {
		s.config.NPSServer.NPCArgs = req.NPCArgs
	}

	if err := s.npsClient.Connect(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil))
}

// handleNPSDisconnect 处理NPS断开请求
func (s *Server) handleNPSDisconnect(c *gin.Context) {
	if err := s.npsClient.Disconnect(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil))
}

// handleNPSTunnels 处理获取NPS隧道列表请求
func (s *Server) handleNPSTunnels(c *gin.Context) {
	status, err := s.npsClient.GetStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"tunnels": status.Tunnels,
	}))
}

// handleMQTTStatus 处理获取MQTT状态请求
func (s *Server) handleMQTTStatus(c *gin.Context) {
	status, err := s.mqttClient.GetStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(status))
}

// handleMQTTConnect 处理MQTT连接请求
func (s *Server) handleMQTTConnect(c *gin.Context) {
	var req struct {
		Server   string `json:"server" binding:"required"`
		Port     int    `json:"port" binding:"required"`
		Username string `json:"username"`
		Password string `json:"password"`
		ClientID string `json:"client_id" binding:"required"`
		TLS      bool   `json:"tls"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "参数错误: "+err.Error()))
		return
	}

	// 更新配置
	s.config.MQTT.Server = req.Server
	s.config.MQTT.Port = req.Port
	s.config.MQTT.Username = req.Username
	s.config.MQTT.Password = req.Password
	s.config.MQTT.ClientID = req.ClientID
	s.config.MQTT.TLS = req.TLS

	if err := s.mqttClient.Connect(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil))
}

// handleMQTTDisconnect 处理MQTT断开请求
func (s *Server) handleMQTTDisconnect(c *gin.Context) {
	if err := s.mqttClient.Disconnect(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil))
}

// handleMQTTLogs 处理获取MQTT日志请求
func (s *Server) handleMQTTLogs(c *gin.Context) {
	topic := c.Query("topic")
	direction := c.Query("direction")
	if direction == "" {
		direction = "all"
	}

	var startTime, endTime time.Time
	if st := c.Query("start_time"); st != "" {
		startTime, _ = time.Parse(time.RFC3339, st)
	}
	if et := c.Query("end_time"); et != "" {
		endTime, _ = time.Parse(time.RFC3339, et)
	}

	page := 1
	pageSize := 50
	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if ps := c.Query("page_size"); ps != "" {
		fmt.Sscanf(ps, "%d", &pageSize)
	}

	offset := (page - 1) * pageSize

	logs, total, err := database.GetMQTTLogs(s.db, topic, direction, startTime, endTime, pageSize, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	// 转换为JSON格式
	logList := make([]gin.H, len(logs))
	for i, l := range logs {
		logList[i] = gin.H{
			"timestamp": l.Timestamp.Format(time.RFC3339),
			"direction": l.Direction,
			"topic":     l.Topic,
			"qos":       l.QoS,
			"payload":   l.Payload,
			"status":    l.Status,
		}
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"logs":      logList,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}))
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
		"nps_server": gin.H{
			"server":    s.config.NPSServer.Server,
			"vkey":      "***",
			"client_id": s.config.NPSServer.ClientID,
		},
		"mqtt": gin.H{
			"server":    s.config.MQTT.Server,
			"port":      s.config.MQTT.Port,
			"username":  s.config.MQTT.Username,
			"client_id": s.config.MQTT.ClientID,
			"tls":       s.config.MQTT.TLS,
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

	// 允许更新的字段：device/network/nps_server/mqtt/scanner/server/database/initialized
	// 安全：不允许通过该接口直接写入 password_hash
	s.config.Device = req.Device
	s.config.Network = req.Network
	s.config.NPSServer = req.NPSServer
	s.config.MQTT = req.MQTT
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
		NPSServer     *config.NPSServerConfig `json:"nps_server"`
		MQTT          *config.MQTTConfig      `json:"mqtt"`
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
	if req.NPSServer != nil {
		s.config.NPSServer = *req.NPSServer
	}
	if req.MQTT != nil {
		s.config.MQTT = *req.MQTT
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
