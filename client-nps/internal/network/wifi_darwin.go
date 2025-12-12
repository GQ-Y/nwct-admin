package network

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (nm *networkManager) scanWiFiDarwinAirport(allowRedacted bool) ([]WiFiNetwork, error) {
	// 1) 优先使用 system_profiler 的 JSON 输出获取“附近 WiFi 列表 + 当前连接”
	if out, err := nm.runCmd(20*time.Second, "system_profiler", "SPAirPortDataType", "-json"); err == nil && out != "" {
		nets, perr := parseSystemProfilerWiFi(out, allowRedacted)
		if perr != nil {
			// 这里直接返回错误（例如 <redacted> / 定位权限），不要静默降级成空列表
			return nil, perr
		}
		return nets, nil
	}

	// 2) 兜底：旧版 airport（如果存在）
	airport := "/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport"
	out2, err2 := nm.runCmd(15*time.Second, airport, "-s")
	if err2 == nil && out2 != "" {
		return parseAirportScan(out2)
	}

	// 3) 兜底：wdutil info（需要 sudo；否则只会输出 usage 且 exit=0）
	out, err := nm.runCmd(15*time.Second, "wdutil", "info")
	if err == nil && out != "" && !strings.HasPrefix(strings.TrimSpace(out), "usage:") {
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
		return []WiFiNetwork{{SSID: ssid, Signal: 0, Security: security, InUse: true}}, nil
	}

	return nil, fmt.Errorf("WiFi扫描失败：macOS 需要 system_profiler(推荐) 或 airport/wdutil 可用。若 system_profiler 返回 <redacted>，请开启定位权限后重试")
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

func parseSystemProfilerWiFi(out string, allowRedacted bool) ([]WiFiNetwork, error) {
	var root map[string]any
	if err := json.Unmarshal([]byte(out), &root); err != nil {
		return nil, err
	}

	data, ok := root["SPAirPortDataType"].([]any)
	if !ok || len(data) == 0 {
		return []WiFiNetwork{}, nil
	}

	// SPAirPortDataType[0].spairport_airport_interfaces[*]
	first, ok := data[0].(map[string]any)
	if !ok {
		return []WiFiNetwork{}, nil
	}
	ifaces, _ := first["spairport_airport_interfaces"].([]any)

	currentSSID := ""
	networks := make([]WiFiNetwork, 0, 16)
	seen := map[string]WiFiNetwork{}
	redactedCount := 0
	validSSIDCount := 0

	for _, ifaceAny := range ifaces {
		iface, ok := ifaceAny.(map[string]any)
		if !ok {
			continue
		}

		// current network
		if cur, ok := iface["spairport_current_network_information"].(map[string]any); ok {
			if name, ok := cur["_name"].(string); ok && name != "" && name != "<redacted>" {
				currentSSID = name
			}
		}

		// nearby networks
		others, _ := iface["spairport_airport_other_local_wireless_networks"].([]any)
		for _, nAny := range others {
			n, ok := nAny.(map[string]any)
			if !ok {
				continue
			}
			ssid, _ := n["_name"].(string)
			if ssid == "<redacted>" {
				redactedCount++
			}
			if ssid == "" {
				continue
			}
			if ssid != "<redacted>" {
				validSSIDCount++
			}

			sec := ""
			if v, ok := n["spairport_security_mode"].(string); ok && v != "" {
				sec = strings.TrimPrefix(v, "spairport_security_mode_")
				// 个别系统版本可能出现不同前缀/拼写，做一次兜底清理
				sec = strings.TrimPrefix(sec, "pairport_security_mode_")
			}
			if sec == "" {
				sec = "UNKNOWN"
			}

			signal := 0
			if sn, ok := n["spairport_signal_noise"].(string); ok && sn != "" {
				// "-35 dBm / -90 dBm" -> -35
				parts := strings.Split(sn, "/")
				if len(parts) > 0 {
					left := strings.TrimSpace(parts[0])
					left = strings.TrimSuffix(left, "dBm")
					left = strings.TrimSpace(left)
					if rssi, err := strconv.Atoi(strings.Fields(left)[0]); err == nil {
						signal = rssi + 100
						if signal < 0 {
							signal = 0
						}
						if signal > 100 {
							signal = 100
						}
					}
				}
			}

			item := WiFiNetwork{
				SSID:     ssid,
				Signal:   signal,
				Security: sec,
				InUse:    ssid == currentSSID,
			}

			// 去重：
			// - 正常 SSID：同 SSID 取 signal 更强
			// - redacted：用 security+signal 组合做去重，避免 UI 出现大量同名项
			key := ssid
			if ssid == "<redacted>" {
				key = fmt.Sprintf("<redacted>|%s|%d", sec, signal)
			}
			if prev, ok := seen[key]; ok {
				if item.Signal > prev.Signal {
					seen[key] = item
				} else if prev.InUse {
					// 保留 InUse
					seen[key] = prev
				}
			} else {
				seen[key] = item
			}
		}
	}

	for _, v := range seen {
		networks = append(networks, v)
	}

	// 如果扫描到了网络但 SSID 都被 redacted，说明缺少定位权限（macOS 需要 Location Services）
	if !allowRedacted && validSSIDCount == 0 && redactedCount > 0 {
		return nil, fmt.Errorf("macOS 已检测到附近WiFi，但 SSID 被系统隐藏（<redacted>）。请在“系统设置 → 隐私与安全 → 定位服务”中允许运行本程序的终端/应用访问定位后重试")
	}
	return networks, nil
}
