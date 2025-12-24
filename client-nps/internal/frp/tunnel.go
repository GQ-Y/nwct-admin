package frp

import (
	"fmt"
	"strings"
	"time"
)

// Tunnel 隧道配置
type Tunnel struct {
	Name       string `json:"name"`        // 隧道名称，如 "192.168.1.100_80"
	Type       string `json:"type"`        // tcp, udp, http, https, stcp
	LocalIP    string `json:"local_ip"`    // 本地IP
	LocalPort  int    `json:"local_port"`  // 本地端口
	RemotePort int    `json:"remote_port"` // 远程端口（0表示自动分配）
	Domain     string `json:"domain,omitempty"` // HTTP类型使用
	CreatedAt  string `json:"created_at"`
	FallbackEnabled bool `json:"fallback_enabled,omitempty"` // HTTP/HTTPS 目标不可达时展示默认页
}

// NewTunnel 创建新隧道
func NewTunnel(name, tunnelType, localIP string, localPort, remotePort int) *Tunnel {
	return &Tunnel{
		Name:       name,
		Type:       tunnelType,
		LocalIP:    localIP,
		LocalPort:  localPort,
		RemotePort: remotePort,
		CreatedAt:  time.Now().Format(time.RFC3339),
	}
}

// GenerateTunnelName 生成隧道名称
func GenerateTunnelName(deviceIP string, port int) string {
	return fmt.Sprintf("%s_%d", strings.ReplaceAll(deviceIP, ".", "_"), port)
}

