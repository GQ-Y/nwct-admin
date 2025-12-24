package frp

import (
	"fmt"
	"nwct/client-nps/config"
	"nwct/client-nps/internal/database"
	"nwct/client-nps/internal/logger"
	"nwct/client-nps/internal/realtime"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var globalFRPClient Client

// SetGlobalClient 设置全局FRP客户端
func SetGlobalClient(client Client) {
	globalFRPClient = client
}

// GetGlobalClient 获取全局FRP客户端
func GetGlobalClient() Client {
	return globalFRPClient
}

// Client FRP客户端接口
type Client interface {
	Connect() error
	Disconnect() error
	IsConnected() bool
	GetStatus() (*FRPStatus, error)
	AddTunnel(tunnel *Tunnel) error
	RemoveTunnel(name string) error
	UpdateTunnel(name string, tunnel *Tunnel) error
	GetTunnels() ([]*Tunnel, error)
	Reload() error
}

// FRPStatus FRP状态
type FRPStatus struct {
	Connected   bool      `json:"connected"`
	Server      string    `json:"server"`
	ConnectedAt string    `json:"connected_at"`
	PID         int       `json:"pid"`
	LastError   string    `json:"last_error,omitempty"`
	Tunnels     []*Tunnel `json:"tunnels"`
	FRCPath     string    `json:"frc_path,omitempty"`
	LogPath     string    `json:"log_path,omitempty"`
}

// frpClient FRP客户端实现
type frpClient struct {
	config     *config.FRPServerConfig
	tunnels    map[string]*Tunnel // 内存中维护的隧道配置
	mu         sync.RWMutex
	connected  bool
	connectedAt time.Time
	lastError   string
	cmd         *exec.Cmd
	configPath  string // frpc.ini 路径
	logPath     string
	restartCount int    // 重启计数，避免无限重启
	lastRestartTime time.Time // 上次重启时间
	proxyManager *ProxyManager
}

// NewClient 创建FRP客户端
func NewClient(cfg *config.FRPServerConfig) Client {
	// 确定配置文件路径
	configDir := os.Getenv("NWCT_CONFIG_DIR")
	if configDir == "" {
		configDir = filepath.Join(os.TempDir(), "nwct")
	}
	_ = os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, "frpc.ini")

	return &frpClient{
		config:     cfg,
		tunnels:    make(map[string]*Tunnel),
		configPath: configPath,
		proxyManager: NewProxyManager(),
	}
}

// Connect 连接到FRP服务端
func (c *frpClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	if c.config.Server == "" {
		return fmt.Errorf("FRP服务器地址未配置")
	}

	// 从数据库加载隧道配置
	if err := c.loadTunnelsFromDB(); err != nil {
		logger.Warn("从数据库加载隧道配置失败: %v，使用空配置", err)
		// 继续执行，使用空配置
	}

	// 确保 frpc 可执行文件可用（优先使用嵌入的，否则使用系统 PATH）
	frcPath, err := ensureFRCPath(c.config.FRCPath)
	if err != nil {
		return fmt.Errorf("获取 frpc 可执行文件失败: %v", err)
	}

	// 生成配置文件（对 http/https 启用代理兜底与域名自动生成）
	configContent, err := c.generateConfigWithProxy()
	if err != nil {
		return fmt.Errorf("生成配置文件失败: %v", err)
	}

	// 写入配置文件
	if err := os.WriteFile(c.configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	// 启动 frpc 进程
	args := []string{"-c", c.configPath}
	logger.Info("启动FRP客户端(frpc): %s %v", frcPath, args)
	cmd := exec.Command(frcPath, args...)

	// 将输出写入日志文件
	logDir := os.Getenv("NWCT_LOG_DIR")
	if logDir == "" {
		logDir = filepath.Join(os.TempDir(), "nwct")
	}
	_ = os.MkdirAll(logDir, 0755)
	c.logPath = filepath.Join(logDir, "frpc.log")
	if f, err := os.OpenFile(c.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
		cmd.Stdout = f
		cmd.Stderr = f
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 frpc 失败: %v", err)
	}

	c.cmd = cmd
	c.connected = true
	c.connectedAt = time.Now()
	c.lastError = ""
	c.restartCount = 0 // 连接成功后重置重启计数

	// 推送状态变化
	realtime.Default().Broadcast("frp_status_changed", map[string]interface{}{
		"connected":    true,
		"server":       c.config.Server,
		"connected_at": c.connectedAt.Format(time.RFC3339),
		"pid":          cmd.Process.Pid,
		"last_error":   "",
		"frc_path":     frcPath,
		"log_path":     c.logPath,
	})

	// 监控进程退出并自动重启
	go func() {
		err := cmd.Wait()
		c.mu.Lock()
		defer c.mu.Unlock()
		
		// 只有在当前 cmd 还是这个进程时才处理（避免旧进程的 Wait 干扰）
		if c.cmd != cmd {
			return
		}
		
		c.connected = false
		if err != nil {
			c.lastError = err.Error()
			logger.Error("frpc 进程退出: %v", err)
		} else {
			c.lastError = ""
			logger.Info("frpc 进程已退出")
		}
		c.cmd = nil
		
		realtime.Default().Broadcast("frp_status_changed", map[string]interface{}{
			"connected":    false,
			"server":       c.config.Server,
			"connected_at": "",
			"pid":          0,
			"last_error":   c.lastError,
			"frc_path":     frcPath,
			"log_path":     c.logPath,
		})
		
		// 如果配置了服务器地址，说明应该保持连接，自动重启
		if c.config.Server != "" {
			logger.Info("frpc 进程异常退出，3秒后自动重启...")
			c.mu.Unlock()
			time.Sleep(3 * time.Second)
			c.mu.Lock()
			
			// 再次检查是否应该重启（可能用户已经手动断开或配置已改变）
			if c.config.Server != "" && c.cmd == nil {
				logger.Info("开始自动重启 frpc...")
				// 使用新的 goroutine 避免死锁
				go func() {
					if err := c.Connect(); err != nil {
						logger.Error("frpc 自动重启失败: %v", err)
					} else {
						logger.Info("frpc 自动重启成功")
					}
				}()
			}
		}
	}()

	return nil
}

// generateRandomDomain 生成随机子域名
func (c *frpClient) generateRandomDomain() string {
	const base = "frpc.zyckj.club"
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return fmt.Sprintf("%s.%s", string(b), base)
}

// generateConfigWithProxy 生成配置，自动为 http/https 启用本地代理兜底页
func (c *frpClient) generateConfigWithProxy() (string, error) {
	// 兜底页 HTML（包子节点品牌）
	fallbackHTML := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8" />
<title>包子节点 · 服务暂不可达</title>
<style>
* { box-sizing: border-box; }
body { margin:0; padding:0; font-family: -apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"Helvetica Neue",Arial,sans-serif; background: linear-gradient(135deg, #eef2ff 0%, #f7f9ff 100%); color:#1f2937; }
.wrap { min-height: 100vh; display:flex; align-items:center; justify-content:center; padding:32px 16px; }
.card { width: 100%; max-width: 560px; background:#fff; border-radius:16px; padding:32px 28px; box-shadow:0 12px 30px rgba(15,23,42,0.08); border:1px solid #eef2ff; }
.brand { display:inline-flex; align-items:center; gap:8px; padding:6px 12px; border-radius:999px; background:#eef2ff; color:#4338ca; font-weight:700; font-size:14px; margin-bottom:14px; }
.title { font-size:22px; font-weight:700; margin:0 0 12px; color:#111827; }
.desc { font-size:15px; line-height:1.7; color:#4b5563; margin:0 0 18px; }
.tips { background:#f8fafc; border:1px solid #e5e7eb; border-radius:12px; padding:14px 16px; color:#374151; font-size:14px; text-align:left; }
.tips strong { color:#111827; }
.footer { margin-top:18px; font-size:13px; color:#9ca3af; }
</style>
</head>
<body>
  <div class="wrap">
    <div class="card">
      <div class="brand">包子节点</div>
      <h1 class="title">目标服务暂时不可达</h1>
      <p class="desc">我们无法连接到后端服务，请稍后再试。如果这是您的服务，请检查本地进程是否运行、网络连通性和防火墙设置。</p>
      <div class="tips">
        <strong>自检提示：</strong><br/>
        · 确认本地服务正在运行且监听正确端口。<br/>
        · 检查内网 IP/端口是否填写正确。<br/>
        · 确认设备与 FRP 客户端网络连通，防火墙已放行。<br/>
        · 若使用域名，请确认 DNS 已解析到服务端公网 IP。
      </div>
      <div class="footer">Powered by Baozi Node</div>
    </div>
  </div>
</body>
</html>`

	cfgTunnels := make(map[string]*Tunnel)

	for name, t := range c.tunnels {
		// 复制一份，避免修改原始数据
		tCopy := *t

		// 对 http/https 启用本地代理和兜底页
		if tCopy.Type == "http" || tCopy.Type == "https" {
			if strings.TrimSpace(tCopy.Domain) == "" {
				tCopy.Domain = c.generateRandomDomain()
				logger.Info("隧道缺少域名，已自动生成: %s => %s", name, tCopy.Domain)
			}

			if tCopy.FallbackEnabled {
				ip, port, err := c.proxyManager.EnsureProxy(&tCopy, fallbackHTML)
				if err != nil {
					return "", fmt.Errorf("启动本地代理失败(%s): %v", name, err)
				}
				// 使用代理端口写入 frpc 配置
				tCopy.LocalIP = ip
				tCopy.LocalPort = port
			}
		}

		cfgTunnels[name] = &tCopy
	}

	return GenerateConfig(c.config, cfgTunnels)
}
// monitorProcess 监控 frpc 进程，退出时自动重启
func (c *frpClient) monitorProcess(cmd *exec.Cmd, frcPath string) {
	err := cmd.Wait()
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// 只有在当前 cmd 还是这个进程时才处理（避免旧进程的 Wait 干扰）
	// 如果 cmd 已经被清空（可能是正在重启），则忽略
	if c.cmd != cmd || c.cmd == nil {
		return
	}
	
	c.connected = false
	if err != nil {
		c.lastError = err.Error()
		logger.Error("frpc 进程退出: %v", err)
	} else {
		c.lastError = ""
		logger.Info("frpc 进程已退出")
	}
	c.cmd = nil
	
	realtime.Default().Broadcast("frp_status_changed", map[string]interface{}{
		"connected":    false,
		"server":       c.config.Server,
		"connected_at": "",
		"pid":          0,
		"last_error":   c.lastError,
		"frc_path":     frcPath,
		"log_path":     c.logPath,
	})
	
	// 如果配置了服务器地址，说明应该保持连接，自动重启
	// 但需要避免无限重启循环
	if c.config.Server != "" {
		now := time.Now()
		
		// 如果距离上次重启不到 10 秒，且重启次数超过 3 次，停止自动重启
		if !c.lastRestartTime.IsZero() && now.Sub(c.lastRestartTime) < 10*time.Second {
			c.restartCount++
			if c.restartCount > 3 {
				logger.Error("frpc 在短时间内多次退出，停止自动重启。请检查配置或服务端连接。最后错误: %v", err)
				c.lastError = fmt.Sprintf("多次重启失败，已停止自动重启。请检查配置: %v", err)
				realtime.Default().Broadcast("frp_status_changed", map[string]interface{}{
					"connected":    false,
					"server":       c.config.Server,
					"connected_at": "",
					"pid":          0,
					"last_error":   c.lastError,
					"frc_path":     frcPath,
					"log_path":     c.logPath,
				})
				return
			}
		} else {
			// 重置计数
			c.restartCount = 1
		}
		
		c.lastRestartTime = now
		logger.Info("frpc 进程异常退出，3秒后自动重启... (重试 %d/3)", c.restartCount)
		c.mu.Unlock()
		time.Sleep(3 * time.Second)
		c.mu.Lock()
		
		// 再次检查是否应该重启（可能用户已经手动断开或配置已改变）
		if c.config.Server != "" && c.cmd == nil && c.restartCount <= 3 {
			logger.Info("开始自动重启 frpc...")
			// 使用新的 goroutine 避免死锁
			go func() {
				if err := c.Connect(); err != nil {
					logger.Error("frpc 自动重启失败: %v", err)
				} else {
					logger.Info("frpc 自动重启成功")
					// 重启成功后重置计数
					c.mu.Lock()
					c.restartCount = 0
					c.mu.Unlock()
				}
			}()
		}
	}
}

// Disconnect 断开FRP连接
func (c *frpClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	logger.Info("断开FRP连接")
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
	c.cmd = nil
	c.connected = false
	c.connectedAt = time.Time{}
	c.lastError = ""
	realtime.Default().Broadcast("frp_status_changed", map[string]interface{}{
		"connected":    false,
		"server":       c.config.Server,
		"connected_at": "",
		"pid":          0,
		"last_error":   "",
		"frc_path":     c.config.FRCPath,
		"log_path":     c.logPath,
	})
	return nil
}

// IsConnected 检查是否已连接
func (c *frpClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// GetStatus 获取FRP状态
func (c *frpClient) GetStatus() (*FRPStatus, error) {
	c.mu.RLock()
	connected := c.connected
	server := c.config.Server
	lastErr := c.lastError
	frcPath := c.config.FRCPath
	logPath := c.logPath
	connectedAt := ""
	if !c.connectedAt.IsZero() {
		connectedAt = c.connectedAt.Format(time.RFC3339)
	}
	pid := 0
	if c.cmd != nil && c.cmd.Process != nil {
		pid = c.cmd.Process.Pid
	}

	// 复制隧道列表
	tunnels := make([]*Tunnel, 0, len(c.tunnels))
	for _, t := range c.tunnels {
		tunnels = append(tunnels, t)
	}
	c.mu.RUnlock()

	return &FRPStatus{
		Connected:   connected,
		Server:      server,
		ConnectedAt: connectedAt,
		PID:         pid,
		LastError:   lastErr,
		Tunnels:     tunnels,
		FRCPath:     frcPath,
		LogPath:     logPath,
	}, nil
}

// AddTunnel 添加隧道
func (c *frpClient) AddTunnel(tunnel *Tunnel) error {
	if tunnel == nil {
		return fmt.Errorf("隧道配置不能为空")
	}
	if tunnel.Name == "" {
		return fmt.Errorf("隧道名称不能为空")
	}
	if tunnel.LocalIP == "" {
		return fmt.Errorf("本地IP不能为空")
	}
	if tunnel.LocalPort <= 0 || tunnel.LocalPort > 65535 {
		return fmt.Errorf("本地端口无效: %d", tunnel.LocalPort)
	}
	if tunnel.Type == "" {
		tunnel.Type = "tcp" // 默认类型
	}
	// HTTP/HTTPS 必须填写域名，否则 frpc 会退出；如未提供则自动生成随机域名
	if tunnel.Type == "http" || tunnel.Type == "https" {
		if strings.TrimSpace(tunnel.Domain) == "" {
			tunnel.Domain = c.generateRandomDomain()
			logger.Info("未提供域名，已自动生成: %s", tunnel.Domain)
		}
		if strings.TrimSpace(tunnel.Domain) == "" {
			return fmt.Errorf("HTTP/HTTPS 隧道必须填写域名(custom_domains)")
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查是否已存在
	if _, exists := c.tunnels[tunnel.Name]; exists {
		// 如果配置相同，跳过
		existing := c.tunnels[tunnel.Name]
		if existing.LocalIP == tunnel.LocalIP &&
			existing.LocalPort == tunnel.LocalPort &&
			existing.Type == tunnel.Type {
			logger.Info("隧道已存在，跳过: %s", tunnel.Name)
			return nil
		}
		// 配置不同，更新
		logger.Info("更新隧道配置: %s", tunnel.Name)
	}

	c.tunnels[tunnel.Name] = tunnel
	logger.Info("隧道已添加到内存: %s (总数: %d)", tunnel.Name, len(c.tunnels))

	// 保存到数据库
	if db := database.GetDB(); db != nil {
		// 转换为 database.Tunnel
		dbTunnel := &database.Tunnel{
			Name:       tunnel.Name,
			Type:       tunnel.Type,
			LocalIP:    tunnel.LocalIP,
			LocalPort:  tunnel.LocalPort,
			RemotePort: tunnel.RemotePort,
			Domain:     tunnel.Domain,
			CreatedAt:  tunnel.CreatedAt,
		}
		if err := database.SaveTunnel(db, dbTunnel); err != nil {
			logger.Error("保存隧道到数据库失败: %v", err)
			// 继续执行，不中断流程
		} else {
			logger.Info("隧道已保存到数据库: %s", tunnel.Name)
		}
	}

	// 如果已连接，立即更新配置文件并重启 frpc（因为 frpc 不支持 SIGHUP）
	if c.connected {
		logger.Info("FRP已连接，立即重启 frpc 以应用新隧道配置...")
		if err := c.reloadConfig(); err != nil {
			return fmt.Errorf("重载配置失败: %v", err)
		}
		logger.Info("配置已更新，frpc 已重启")
	} else {
		logger.Warn("FRP未连接，隧道已添加到内存但未应用到配置")
	}

	return nil
}

// RemoveTunnel 删除隧道
func (c *frpClient) RemoveTunnel(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.tunnels[name]; !exists {
		return fmt.Errorf("隧道不存在: %s", name)
	}

	delete(c.tunnels, name)
	logger.Info("隧道已从内存删除: %s (剩余: %d)", name, len(c.tunnels))

	// 从数据库删除
	if db := database.GetDB(); db != nil {
		if err := database.DeleteTunnel(db, name); err != nil {
			logger.Error("从数据库删除隧道失败: %v", err)
			// 继续执行，不中断流程
		} else {
			logger.Info("隧道已从数据库删除: %s", name)
		}
	}

	// 如果已连接，更新配置文件并重载
	if c.connected {
		if err := c.reloadConfig(); err != nil {
			return fmt.Errorf("重载配置失败: %v", err)
		}
	}

	return nil
}

// UpdateTunnel 更新隧道
func (c *frpClient) UpdateTunnel(name string, tunnel *Tunnel) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.tunnels[name]; !exists {
		return fmt.Errorf("隧道不存在: %s", name)
	}

	tunnel.Name = name // 确保名称一致
	// HTTP/HTTPS 必须填写域名，否则 frpc 会退出；如未提供则自动生成随机域名
	if tunnel.Type == "http" || tunnel.Type == "https" {
		if strings.TrimSpace(tunnel.Domain) == "" {
			tunnel.Domain = c.generateRandomDomain()
			logger.Info("未提供域名，已自动生成: %s", tunnel.Domain)
		}
		if strings.TrimSpace(tunnel.Domain) == "" {
			return fmt.Errorf("HTTP/HTTPS 隧道必须填写域名(custom_domains)")
		}
	}
	c.tunnels[name] = tunnel
	logger.Info("隧道已更新到内存: %s", name)

	// 保存到数据库
	if db := database.GetDB(); db != nil {
		// 转换为 database.Tunnel
		dbTunnel := &database.Tunnel{
			Name:       tunnel.Name,
			Type:       tunnel.Type,
			LocalIP:    tunnel.LocalIP,
			LocalPort:  tunnel.LocalPort,
			RemotePort: tunnel.RemotePort,
			Domain:     tunnel.Domain,
			CreatedAt:  tunnel.CreatedAt,
		}
		if err := database.SaveTunnel(db, dbTunnel); err != nil {
			logger.Error("更新隧道到数据库失败: %v", err)
			// 继续执行，不中断流程
		} else {
			logger.Info("隧道已更新到数据库: %s", name)
		}
	}

	// 如果已连接，更新配置文件并重载
	if c.connected {
		if err := c.reloadConfig(); err != nil {
			return fmt.Errorf("重载配置失败: %v", err)
		}
	}

	return nil
}

// GetTunnels 获取所有隧道
func (c *frpClient) GetTunnels() ([]*Tunnel, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tunnels := make([]*Tunnel, 0, len(c.tunnels))
	for _, t := range c.tunnels {
		tunnels = append(tunnels, t)
	}
	return tunnels, nil
}

// Reload 重载配置
func (c *frpClient) Reload() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return fmt.Errorf("FRP未连接，无法重载")
	}

	return c.reloadConfig()
}

// loadTunnelsFromDB 从数据库加载隧道配置
func (c *frpClient) loadTunnelsFromDB() error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}

	dbTunnels, err := database.GetAllTunnels(db)
	if err != nil {
		return fmt.Errorf("从数据库加载隧道失败: %v", err)
	}

	// 清空现有隧道（从数据库重新加载）
	c.tunnels = make(map[string]*Tunnel)
	for _, dbTunnel := range dbTunnels {
		// 转换为 frp.Tunnel
		tunnel := &Tunnel{
			Name:            dbTunnel.Name,
			Type:            dbTunnel.Type,
			LocalIP:         dbTunnel.LocalIP,
			LocalPort:       dbTunnel.LocalPort,
			RemotePort:      dbTunnel.RemotePort,
			Domain:          dbTunnel.Domain,
			CreatedAt:       dbTunnel.CreatedAt,
			FallbackEnabled: dbTunnel.FallbackEnabled,
		}
		// 如果是 http/https 且未提供域名，自动生成，避免启动失败
		if (tunnel.Type == "http" || tunnel.Type == "https") && strings.TrimSpace(tunnel.Domain) == "" {
			tunnel.Domain = c.generateRandomDomain()
			logger.Warn("隧道缺少域名，已自动生成: %s => %s", tunnel.Name, tunnel.Domain)
		}

		c.tunnels[tunnel.Name] = tunnel
	}

	logger.Info("从数据库加载了 %d 个隧道配置", len(c.tunnels))
	return nil
}

// reloadConfig 内部方法：重新生成配置并热重载 frpc
func (c *frpClient) reloadConfig() error {
	logger.Info("重新生成配置，当前隧道数: %d", len(c.tunnels))
	
	// 生成新配置（对 http/https 启用代理兜底与域名自动生成）
	configContent, err := c.generateConfigWithProxy()
	if err != nil {
		return err
	}

	logger.Info("生成的配置内容长度: %d 字节", len(configContent))

	// 写入配置文件
	if err := os.WriteFile(c.configPath, []byte(configContent), 0644); err != nil {
		return err
	}
	
	logger.Info("配置文件已写入: %s", c.configPath)

	// 如果进程正在运行，尝试使用热重载
	if c.cmd != nil && c.cmd.Process != nil {
		// 检查进程是否还在运行
		if err := c.cmd.Process.Signal(os.Signal(nil)); err != nil {
			// 进程已退出，需要重新启动
			logger.Warn("frpc 进程已退出，需要重新启动")
			c.connected = false
			c.cmd = nil
			// 重新连接
			c.mu.Unlock()
			if err := c.Connect(); err != nil {
				c.mu.Lock()
				return fmt.Errorf("重新启动 frpc 失败: %v", err)
			}
			c.mu.Lock()
			return nil
		}

		// 进程正在运行，使用热重载
		logger.Info("使用 frpc reload 热重载配置...")
		frcPath, err := ensureFRCPath(c.config.FRCPath)
		if err != nil {
			return fmt.Errorf("获取 frpc 可执行文件失败: %v", err)
		}

		// 执行 frpc reload 命令
		reloadCmd := exec.Command(frcPath, "reload", "-c", c.configPath)
		reloadCmd.Stdout = os.Stdout
		reloadCmd.Stderr = os.Stderr
		if err := reloadCmd.Run(); err != nil {
			logger.Warn("frpc reload 失败: %v，尝试重启进程", err)
			// 如果热重载失败，回退到重启进程
			return c.restartProcess()
		}
		logger.Info("frpc 配置热重载成功")
		return nil
	}

	// 进程不存在，需要启动
	logger.Info("frpc 进程不存在，需要启动")
	c.connected = false
	c.mu.Unlock()
	if err := c.Connect(); err != nil {
		c.mu.Lock()
		return fmt.Errorf("启动 frpc 失败: %v", err)
	}
	c.mu.Lock()
	return nil
}

// restartProcess 重启 frpc 进程（仅在热重载失败时使用）
func (c *frpClient) restartProcess() error {
	if c.cmd == nil || c.cmd.Process == nil {
		return fmt.Errorf("进程不存在")
	}

	// 标记正在重启，避免 monitorProcess 干扰
	oldCmd := c.cmd
	c.cmd = nil  // 先清空，避免 monitorProcess 处理旧进程退出
	c.connected = false

	// 停止旧进程
	_ = oldCmd.Process.Kill()
	_ = oldCmd.Wait()

	// 重新启动（确保 frpc 路径可用）
	frcPath, err := ensureFRCPath(c.config.FRCPath)
	if err != nil {
		return fmt.Errorf("获取 frpc 可执行文件失败: %v", err)
	}
	args := []string{"-c", c.configPath}
	cmd := exec.Command(frcPath, args...)

	// 设置日志输出
	if c.logPath != "" {
		if f, err := os.OpenFile(c.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
			cmd.Stdout = f
			cmd.Stderr = f
		}
	}

	if err := cmd.Start(); err != nil {
		c.connected = false
		return fmt.Errorf("重启 frpc 失败: %v", err)
	}

	c.cmd = cmd
	c.connected = true
	c.connectedAt = time.Now()
	c.lastError = ""
	c.restartCount = 0 // 重置重启计数

	// 推送状态变化
	realtime.Default().Broadcast("frp_status_changed", map[string]interface{}{
		"connected":    true,
		"server":       c.config.Server,
		"connected_at": c.connectedAt.Format(time.RFC3339),
		"pid":          cmd.Process.Pid,
		"last_error":   "",
		"frc_path":     frcPath,
		"log_path":     c.logPath,
	})

	// 监控进程退出并自动重启
	go c.monitorProcess(cmd, frcPath)

	return nil
}

