package network

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (nm *networkManager) scanWiFiDarwinAirport() ([]WiFiNetwork, error) {
	// 新版 macOS 可能没有 airport 可执行文件，这里优先使用 wdutil
	// wdutil 是系统自带（/usr/bin/wdutil），输出可解析到 SSID/安全类型（信号强度可能缺失）
	out, err := nm.runCmd(15*time.Second, "wdutil", "info")
	if err != nil {
		// 尝试旧版 airport（如果存在）
		airport := "/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport"
		out2, err2 := nm.runCmd(15*time.Second, airport, "-s")
		if err2 != nil {
			return nil, fmt.Errorf("WiFi扫描失败（macOS 需要 wdutil 或 airport 可用）: %w", err)
		}
		return parseAirportScan(out2)
	}

	// wdutil info 只能拿到“当前连接信息”，这里返回当前连接的 SSID（当作 in_use=true）
	ssid := ""
	security := ""
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "SSID:") {
			ssid = strings.TrimSpace(strings.TrimPrefix(line, "SSID:"))
		}
		if strings.HasPrefix(line, "Security:") {
			security = strings.TrimSpace(strings.TrimPrefix(line, "Security:"))
		}
	}
	if ssid == "" {
		return []WiFiNetwork{}, nil
	}
	if security == "" {
		security = "UNKNOWN"
	}
	return []WiFiNetwork{
		{SSID: ssid, Signal: 0, Security: security, InUse: true},
	}, nil
}

func parseAirportScan(out string) ([]WiFiNetwork, error) {
	lines := strings.Split(out, "\n")
	if len(lines) <= 1 {
		return []WiFiNetwork{}, nil
	}

	networks := make([]WiFiNetwork, 0, len(lines)-1)
	for i := 1; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], " ")
		if strings.TrimSpace(line) == "" {
			continue
		}

		// 示例：
		// SSID BSSID             RSSI CHANNEL HT CC SECURITY(auth/unicast/group)
		// MyWiFi xx:xx:...       -67  11      Y  -- WPA2(PSK/AES/AES)
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		// SSID 可能包含空格 -> airport 输出中 SSID 列宽对齐，简单 fields 会打散。
		// 这里采用一种折中：从后往前定位 RSSI（负数），并把其之前的 fields 合并成 SSID。
		rssiIdx := -1
		for j := 0; j < len(fields); j++ {
			if strings.HasPrefix(fields[j], "-") {
				if _, e := strconv.Atoi(fields[j]); e == nil {
					rssiIdx = j
					break
				}
			}
		}
		if rssiIdx < 2 {
			continue
		}

		ssid := strings.Join(fields[:rssiIdx-1], " ")
		rssiStr := fields[rssiIdx]
		sec := fields[len(fields)-1]

		rssi, _ := strconv.Atoi(rssiStr)
		// RSSI(-100..0) -> 0..100 粗略映射
		signal := rssi + 100
		if signal < 0 {
			signal = 0
		}
		if signal > 100 {
			signal = 100
		}

		networks = append(networks, WiFiNetwork{
			SSID:     ssid,
			Signal:   signal,
			Security: sec,
			InUse:    false,
		})
	}

	return networks, nil
}

func (nm *networkManager) connectWiFiDarwinNetworksetup(ssid, password string) error {
	device, err := nm.findDarwinWiFiDevice()
	if err != nil {
		return err
	}

	// networksetup -setairportnetwork <device> <network> [password]
	args := []string{"-setairportnetwork", device, ssid}
	if strings.TrimSpace(password) != "" {
		args = append(args, password)
	}

	_, err = nm.runCmd(30*time.Second, "networksetup", args...)
	if err != nil {
		return fmt.Errorf("WiFi连接失败（macOS networksetup）: %w", err)
	}
	return nil
}

func (nm *networkManager) findDarwinWiFiDevice() (string, error) {
	out, err := nm.runCmd(10*time.Second, "networksetup", "-listallhardwareports")
	if err != nil {
		return "", fmt.Errorf("获取WiFi设备失败: %w", err)
	}

	// Hardware Port: Wi-Fi
	// Device: en0
	lines := strings.Split(out, "\n")
	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "Hardware Port:") &&
			strings.Contains(lines[i], "Wi-Fi") {
			for j := i + 1; j < len(lines) && j < i+5; j++ {
				line := strings.TrimSpace(lines[j])
				if strings.HasPrefix(line, "Device:") {
					return strings.TrimSpace(strings.TrimPrefix(line, "Device:")), nil
				}
			}
		}
	}

	return "", fmt.Errorf("未找到WiFi设备（Hardware Port: Wi-Fi）")
}
