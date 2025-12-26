//go:build !darwin

package network

import "fmt"

// 仅用于让非 darwin 平台编译通过（实际不会走到这些分支）

func (nm *networkManager) scanWiFiDarwinAirport(allowRedacted bool) ([]WiFiNetwork, error) {
	return nil, fmt.Errorf("darwin WiFi 扫描不支持: scanWiFiDarwinAirport")
}

func (nm *networkManager) connectWiFiDarwinNetworksetup(ssid, password string) error {
	return fmt.Errorf("darwin WiFi 连接不支持: connectWiFiDarwinNetworksetup")
}

func (nm *networkManager) findDarwinWiFiDevice() (string, error) {
	return "", fmt.Errorf("darwin WiFi 设备不支持: findDarwinWiFiDevice")
}


