package frp

import (
	"fmt"
	"nwct/client-nps/config"
	"nwct/client-nps/internal/logger"
	"nwct/client-nps/internal/realtime"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

var globalFRPClient Client

// SetGlobalClient 设置全局FRP客户端（用于自动穿透）
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

	// 确保 frpc 可执行文件可用（优先使用嵌入的，否则使用系统 PATH）
	frcPath, err := ensureFRCPath(c.config.FRCPath)
	if err != nil {
		return fmt.Errorf("获取 frpc 可执行文件失败: %v", err)
	}

	// 生成配置文件
	configContent, err := GenerateConfig(c.config, c.tunnels)
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

	// 监控进程退出
	go func() {
		err := cmd.Wait()
		c.mu.Lock()
		defer c.mu.Unlock()
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
	}()

	return nil
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

	// 如果已连接，更新配置文件并重载
	if c.connected {
		if err := c.reloadConfig(); err != nil {
			return fmt.Errorf("重载配置失败: %v", err)
		}
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
	c.tunnels[name] = tunnel

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

// reloadConfig 内部方法：重新生成配置并重载 frpc
func (c *frpClient) reloadConfig() error {
	// 生成新配置
	configContent, err := GenerateConfig(c.config, c.tunnels)
	if err != nil {
		return err
	}

	// 写入配置文件
	if err := os.WriteFile(c.configPath, []byte(configContent), 0644); err != nil {
		return err
	}

	// 尝试发送 SIGHUP 信号重载（如果进程支持）
	if c.cmd != nil && c.cmd.Process != nil {
		if err := c.cmd.Process.Signal(syscall.SIGHUP); err != nil {
			// 如果不支持 SIGHUP，需要重启进程
			logger.Warn("发送 SIGHUP 失败，将重启进程: %v", err)
			return c.restartProcess()
		}
		logger.Info("已发送重载信号给 frpc 进程")
	}

	return nil
}

// restartProcess 重启 frpc 进程
func (c *frpClient) restartProcess() error {
	if c.cmd == nil || c.cmd.Process == nil {
		return fmt.Errorf("进程不存在")
	}

	// 保存当前状态
	wasConnected := c.connected
	oldCmd := c.cmd

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
	c.connected = wasConnected

	// 监控进程退出
	go func() {
		err := cmd.Wait()
		c.mu.Lock()
		defer c.mu.Unlock()
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
		})
	}()

	return nil
}

