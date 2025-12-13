package nps

import (
	"fmt"
	"nwct/client-nps/config"
	"nwct/client-nps/internal/logger"
	"nwct/client-nps/internal/realtime"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// Client NPS客户端接口
type Client interface {
	Connect() error
	Disconnect() error
	IsConnected() bool
	GetStatus() (*NPSStatus, error)
}

// NPSStatus NPS状态
type NPSStatus struct {
	Connected   bool     `json:"connected"`
	Server      string   `json:"server"`
	ClientID    string   `json:"client_id"`
	ConnectedAt string   `json:"connected_at"`
	PID         int      `json:"pid"`
	LastError   string   `json:"last_error,omitempty"`
	Tunnels     []Tunnel `json:"tunnels"`
	NPCPath     string   `json:"npc_path,omitempty"`
	LogPath     string   `json:"log_path,omitempty"`
}

// Tunnel 隧道信息
type Tunnel struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	LocalPort  int    `json:"local_port"`
	RemotePort int    `json:"remote_port"`
	Status     string `json:"status"`
}

// npsClient NPS客户端实现
type npsClient struct {
	config *config.NPSServerConfig

	mu          sync.RWMutex
	connected   bool
	connectedAt time.Time
	lastError   string

	cmd *exec.Cmd

	logPath string
}

// NewClient 创建NPS客户端
func NewClient(cfg *config.NPSServerConfig) Client {
	return &npsClient{
		config: cfg,
	}
}

// Connect 连接到NPS服务端
func (c *npsClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}
	if c.config.Server == "" {
		return fmt.Errorf("NPS服务器地址未配置")
	}
	if c.config.VKey == "" && c.config.NPCConfigPath == "" {
		return fmt.Errorf("NPS vkey 未配置（或提供 npc_config_path）")
	}

	// 采用官方 npc 客户端进程方式（更贴近实际部署）：启动 npc 并常驻
	npcPath := c.config.NPCPath
	if npcPath == "" {
		npcPath = "npc"
	}
	if _, err := exec.LookPath(npcPath); err != nil {
		// 如果是显式路径也给出更清晰错误
		return fmt.Errorf("未找到 npc 可执行文件: %s（可通过 /api/v1/nps/npc/install 一键安装，或设置 nps_server.npc_path）", npcPath)
	}

	args := make([]string, 0, 8)
	if c.config.NPCConfigPath != "" {
		// 如果有专用配置文件，优先使用（不同版本 npc 参数可能有差异，按常见 -config= 形式）
		args = append(args, fmt.Sprintf("-config=%s", c.config.NPCConfigPath))
	} else {
		args = append(args, fmt.Sprintf("-server=%s", c.config.Server))
		args = append(args, fmt.Sprintf("-vkey=%s", c.config.VKey))
	}
	if len(c.config.NPCArgs) > 0 {
		args = append(args, c.config.NPCArgs...)
	}

	logger.Info("启动NPS客户端(npc): %s %v", npcPath, args)
	cmd := exec.Command(npcPath, args...)

	// 将 npc 输出写入日志文件，便于 UI/排障
	logDir := os.Getenv("NWCT_LOG_DIR")
	if logDir == "" {
		logDir = filepath.Join(os.TempDir(), "nwct")
	}
	_ = os.MkdirAll(logDir, 0o755)
	c.logPath = filepath.Join(logDir, "npc.log")
	if f, err := os.OpenFile(c.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666); err == nil {
		cmd.Stdout = f
		cmd.Stderr = f
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 npc 失败: %v", err)
	}
	c.cmd = cmd
	c.connected = true
	c.connectedAt = time.Now()
	c.lastError = ""
	// 推送 NPS 状态变化
	realtime.Default().Broadcast("nps_status_changed", map[string]interface{}{
		"connected":    true,
		"server":       c.config.Server,
		"client_id":    c.config.ClientID,
		"connected_at": c.connectedAt.Format(time.RFC3339),
		"pid":          cmd.Process.Pid,
		"last_error":   "",
		"npc_path":     npcPath,
		"log_path":     c.logPath,
	})

	// 监控退出
	go func() {
		err := cmd.Wait()
		c.mu.Lock()
		defer c.mu.Unlock()
		c.connected = false
		if err != nil {
			c.lastError = err.Error()
			logger.Error("npc 进程退出: %v", err)
		} else {
			c.lastError = ""
			logger.Info("npc 进程已退出")
		}
		c.cmd = nil
		realtime.Default().Broadcast("nps_status_changed", map[string]interface{}{
			"connected":    false,
			"server":       c.config.Server,
			"client_id":    c.config.ClientID,
			"connected_at": "",
			"pid":          0,
			"last_error":   c.lastError,
			"npc_path":     npcPath,
			"log_path":     c.logPath,
		})
	}()
	return nil
}

// Disconnect 断开NPS连接
func (c *npsClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	logger.Info("断开NPS连接")
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
	c.cmd = nil
	c.connected = false
	c.connectedAt = time.Time{}
	c.lastError = ""
	realtime.Default().Broadcast("nps_status_changed", map[string]interface{}{
		"connected":    false,
		"server":       c.config.Server,
		"client_id":    c.config.ClientID,
		"connected_at": "",
		"pid":          0,
		"last_error":   "",
		"npc_path":     c.config.NPCPath,
		"log_path":     c.logPath,
	})
	return nil
}

// IsConnected 检查是否已连接
func (c *npsClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// GetStatus 获取NPS状态
func (c *npsClient) GetStatus() (*NPSStatus, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	connectedAt := ""
	if !c.connectedAt.IsZero() {
		connectedAt = c.connectedAt.Format(time.RFC3339)
	}
	pid := 0
	if c.cmd != nil && c.cmd.Process != nil {
		pid = c.cmd.Process.Pid
	}
	return &NPSStatus{
		Connected:   c.connected,
		Server:      c.config.Server,
		ClientID:    c.config.ClientID,
		ConnectedAt: connectedAt,
		PID:         pid,
		LastError:   c.lastError,
		Tunnels:     []Tunnel{},
		NPCPath:     c.config.NPCPath,
		LogPath:     c.logPath,
	}, nil
}
