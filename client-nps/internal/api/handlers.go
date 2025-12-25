package api

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"nwct/client-nps/config"
	"nwct/client-nps/internal/database"
	"nwct/client-nps/internal/frp"
	"nwct/client-nps/internal/logger"
	"nwct/client-nps/internal/network"
	"nwct/client-nps/internal/scanner"
	"nwct/client-nps/internal/toolkit"
	"nwct/client-nps/internal/version"
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

	// 截断文件（不存在则视为已清空）
	if err := os.Truncate(logPath, 0); err != nil {
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

	c.JSON(http.StatusOK, models.SuccessResponse(status))
}

// handleFRPConnect 处理FRP连接请求
func (s *Server) handleFRPConnect(c *gin.Context) {
	var req struct {
		Server    string `json:"server"`
		Token     string `json:"token"`
		AdminAddr string `json:"admin_addr"`
		AdminUser string `json:"admin_user"`
		AdminPwd  string `json:"admin_pwd"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许空请求体：使用已有配置（实现"一键连接"）
	}

	// 合并配置：请求未提供的字段则沿用已有配置
	if strings.TrimSpace(req.Server) != "" {
		s.config.FRPServer.Server = strings.TrimSpace(req.Server)
	}
	if strings.TrimSpace(req.Token) != "" {
		s.config.FRPServer.Token = strings.TrimSpace(req.Token)
	}
	if strings.TrimSpace(req.AdminAddr) != "" {
		s.config.FRPServer.AdminAddr = strings.TrimSpace(req.AdminAddr)
	}
	if strings.TrimSpace(req.AdminUser) != "" {
		s.config.FRPServer.AdminUser = strings.TrimSpace(req.AdminUser)
	}
	if strings.TrimSpace(req.AdminPwd) != "" {
		s.config.FRPServer.AdminPwd = strings.TrimSpace(req.AdminPwd)
	}

	if err := s.frpClient.Connect(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	_ = s.config.Save()

	c.JSON(http.StatusOK, models.SuccessResponse(nil))
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
			"server":        s.config.FRPServer.Server,
			"token":         "***",
			"admin_addr":    s.config.FRPServer.AdminAddr,
			"admin_user":    s.config.FRPServer.AdminUser,
			"domain_suffix": s.config.FRPServer.DomainSuffix,
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
