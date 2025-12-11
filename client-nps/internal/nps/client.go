package nps

import (
	"fmt"
	"nwct/client-nps/config"
	"nwct/client-nps/internal/logger"
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
	Connected  bool      `json:"connected"`
	Server     string    `json:"server"`
	ClientID   string    `json:"client_id"`
	ConnectedAt string   `json:"connected_at"`
	Tunnels    []Tunnel  `json:"tunnels"`
}

// Tunnel 隧道信息
type Tunnel struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	LocalPort   int    `json:"local_port"`
	RemotePort  int    `json:"remote_port"`
	Status      string `json:"status"`
}

// npsClient NPS客户端实现
type npsClient struct {
	config   *config.NPSServerConfig
	connected bool
}

// NewClient 创建NPS客户端
func NewClient(cfg *config.NPSServerConfig) Client {
	return &npsClient{
		config:   cfg,
		connected: false,
	}
}

// Connect 连接到NPS服务端
func (c *npsClient) Connect() error {
	if c.config.Server == "" {
		return fmt.Errorf("NPS服务器地址未配置")
	}
	
	// TODO: 集成NPS客户端库
	logger.Info("连接NPS服务器: %s", c.config.Server)
	c.connected = true
	return nil
}

// Disconnect 断开NPS连接
func (c *npsClient) Disconnect() error {
	logger.Info("断开NPS连接")
	c.connected = false
	return nil
}

// IsConnected 检查是否已连接
func (c *npsClient) IsConnected() bool {
	return c.connected
}

// GetStatus 获取NPS状态
func (c *npsClient) GetStatus() (*NPSStatus, error) {
	return &NPSStatus{
		Connected:  c.connected,
		Server:     c.config.Server,
		ClientID:   c.config.ClientID,
		Tunnels:    []Tunnel{},
	}, nil
}

