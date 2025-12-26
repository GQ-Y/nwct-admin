package frp

import (
	"fmt"
	"totoro-device/config"
	"strings"
)

// GenerateConfig 根据隧道列表生成 frpc.ini
func GenerateConfig(cfg *config.FRPServerConfig, tunnels map[string]*Tunnel) (string, error) {
	var sb strings.Builder

	// 解析 server 地址和端口
	serverParts := strings.Split(cfg.Server, ":")
	if len(serverParts) != 2 {
		return "", fmt.Errorf("无效的服务器地址格式: %s", cfg.Server)
	}
	serverAddr := serverParts[0]
	serverPort := serverParts[1]

	// 写入 common 段
	sb.WriteString("[common]\n")
	sb.WriteString(fmt.Sprintf("server_addr = %s\n", serverAddr))
	sb.WriteString(fmt.Sprintf("server_port = %s\n", serverPort))
	if cfg.Token != "" {
		sb.WriteString(fmt.Sprintf("token = %s\n", cfg.Token))
	}
	// 启用 webServer 以支持热重载
	sb.WriteString("\n[webServer]\n")
	sb.WriteString("addr = 127.0.0.1\n")
	sb.WriteString("port = 7400\n")
	sb.WriteString("\n")

	// 写入每个隧道配置
	for name, tunnel := range tunnels {
		sb.WriteString(fmt.Sprintf("[%s]\n", name))
		sb.WriteString(fmt.Sprintf("type = %s\n", tunnel.Type))
		sb.WriteString(fmt.Sprintf("local_ip = %s\n", tunnel.LocalIP))
		sb.WriteString(fmt.Sprintf("local_port = %d\n", tunnel.LocalPort))
		if tunnel.RemotePort > 0 {
			sb.WriteString(fmt.Sprintf("remote_port = %d\n", tunnel.RemotePort))
		} else {
			sb.WriteString("remote_port = 0\n")
		}
		if tunnel.Domain != "" {
			sb.WriteString(fmt.Sprintf("custom_domains = %s\n", tunnel.Domain))
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// ParseConfig 解析现有 frpc.ini（用于启动时恢复状态）
func ParseConfig(configPath string) (map[string]*Tunnel, error) {
	// TODO: 实现解析逻辑（如果需要从现有配置文件恢复）
	// 目前返回空 map，因为我们会从内存状态管理
	return make(map[string]*Tunnel), nil
}

