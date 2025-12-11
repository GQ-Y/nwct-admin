package network

import (
	"fmt"
	"net"
	"nwct/client-nps/config"
	"nwct/client-nps/internal/logger"
	"time"
)

// Manager 网络管理器接口
type Manager interface {
	GetInterfaces() ([]Interface, error)
	ConfigureWiFi(ssid, password string) error
	GetNetworkStatus() (*NetworkStatus, error)
	TestConnection(target string) error
}

// Interface 网络接口
type Interface struct {
	Name    string `json:"name"`
	Type    string `json:"type"` // ethernet, wifi
	Status  string `json:"status"` // up, down
	IP      string `json:"ip"`
	Netmask string `json:"netmask"`
	Gateway string `json:"gateway"`
	MAC     string `json:"mac"`
}

// NetworkStatus 网络状态
type NetworkStatus struct {
	CurrentInterface string  `json:"current_interface"`
	IP               string  `json:"ip"`
	Status           string  `json:"status"` // connected, disconnected
	UploadSpeed      float64 `json:"upload_speed"`
	DownloadSpeed    float64 `json:"download_speed"`
	Latency          int     `json:"latency"`
}

// networkManager 网络管理器实现
type networkManager struct {
	config *config.Config
}

// NewManager 创建网络管理器
func NewManager() Manager {
	return &networkManager{}
}

// GetInterfaces 获取网络接口列表
func (nm *networkManager) GetInterfaces() ([]Interface, error) {
	// 使用标准库获取网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("获取网络接口失败: %v", err)
	}

	result := make([]Interface, 0)
	for _, iface := range interfaces {
		// 跳过loopback接口
		if iface.Name == "lo" || iface.Name == "lo0" {
			continue
		}

		netInterface := Interface{
			Name:   iface.Name,
			Status: "down",
			MAC:    iface.HardwareAddr.String(),
		}

		// 判断接口是否启动
		if iface.Flags&net.FlagUp != 0 {
			netInterface.Status = "up"
		}

		// 判断是有线还是WiFi（简单判断，根据名称）
		if len(iface.Name) >= 4 && (iface.Name[:4] == "wlan" || iface.Name[:2] == "wl") {
			netInterface.Type = "wifi"
		} else if len(iface.Name) >= 3 && iface.Name[:3] == "eth" {
			netInterface.Type = "ethernet"
		} else {
			netInterface.Type = "ethernet" // 默认
		}

		// 获取IP地址
		addrs, err := iface.Addrs()
		if err == nil {
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						netInterface.IP = ipnet.IP.String()
						netInterface.Netmask = ipnet.Mask.String()
						break
					}
				}
			}
		}

		result = append(result, netInterface)
	}

	return result, nil
}

// ConfigureWiFi 配置WiFi连接
func (nm *networkManager) ConfigureWiFi(ssid, password string) error {
	// TODO: 通过NetworkManager D-Bus或wpa_supplicant配置WiFi
	// 这里先返回一个占位实现
	logger.Info("配置WiFi连接: SSID=%s", ssid)
	return fmt.Errorf("WiFi配置功能待实现（需要NetworkManager D-Bus或系统命令）")
}

// GetNetworkStatus 获取网络状态
func (nm *networkManager) GetNetworkStatus() (*NetworkStatus, error) {
	interfaces, err := nm.GetInterfaces()
	if err != nil {
		return nil, err
	}

	status := &NetworkStatus{
		Status: "disconnected",
	}

	// 查找第一个已连接且有IP的接口
	for _, iface := range interfaces {
		if iface.Status == "up" && iface.IP != "" {
			status.CurrentInterface = iface.Name
			status.IP = iface.IP
			status.Status = "connected"
			break
		}
	}

	// 测试网络延迟（ping网关或DNS）
	if status.Status == "connected" {
		// 尝试ping 8.8.8.8测试连通性
		latency, err := nm.testLatency("8.8.8.8")
		if err == nil {
			status.Latency = latency
		}
	}

	return status, nil
}

// TestConnection 测试网络连接
func (nm *networkManager) TestConnection(target string) error {
	conn, err := net.DialTimeout("tcp", target+":80", 5*time.Second)
	if err != nil {
		return fmt.Errorf("连接失败: %v", err)
	}
	conn.Close()
	return nil
}

// testLatency 测试延迟
func (nm *networkManager) testLatency(target string) (int, error) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", target+":53", 3*time.Second)
	if err != nil {
		return 0, err
	}
	conn.Close()
	return int(time.Since(start).Milliseconds()), nil
}
