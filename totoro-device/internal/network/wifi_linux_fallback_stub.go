//go:build !linux

package network

import "fmt"

func (nm *networkManager) scanWiFiLinuxFallback(opts ScanWiFiOptions) ([]WiFiNetwork, error) {
	return nil, fmt.Errorf("WiFi扫描仅支持 linux")
}

func (nm *networkManager) connectWiFiLinuxFallback(ssid, password string) error {
	return fmt.Errorf("WiFi连接仅支持 linux")
}
