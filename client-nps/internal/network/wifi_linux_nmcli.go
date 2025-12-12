package network

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (nm *networkManager) scanWiFiLinuxNmcli() ([]WiFiNetwork, error) {
	// nmcli -t --separator '\t' -f IN-USE,SSID,SIGNAL,SECURITY dev wifi list --rescan yes
	out, err := nm.runCmd(15*time.Second, "nmcli",
		"-t",
		"--separator", "\t",
		"-f", "IN-USE,SSID,SIGNAL,SECURITY",
		"dev", "wifi", "list",
		"--rescan", "yes",
	)
	if err != nil {
		return nil, fmt.Errorf("WiFi扫描失败（需要 NetworkManager + nmcli）: %w", err)
	}

	lines := strings.Split(out, "\n")
	networks := make([]WiFiNetwork, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// IN-USE \t SSID \t SIGNAL \t SECURITY
		parts := strings.Split(line, "\t")
		if len(parts) < 4 {
			continue
		}

		inUse := strings.TrimSpace(parts[0]) == "*"
		ssid := strings.TrimSpace(parts[1])
		signalStr := strings.TrimSpace(parts[2])
		sec := strings.TrimSpace(parts[3])

		signal := 0
		if signalStr != "" {
			if v, e := strconv.Atoi(signalStr); e == nil {
				signal = v
			}
		}
		if sec == "" {
			sec = "OPEN"
		}

		// 跳过隐藏 SSID
		if ssid == "" {
			continue
		}

		networks = append(networks, WiFiNetwork{
			SSID:     ssid,
			Signal:   signal,
			Security: sec,
			InUse:    inUse,
		})
	}

	return networks, nil
}

func (nm *networkManager) connectWiFiLinuxNmcli(ssid, password string) error {
	// nmcli dev wifi connect "<ssid>" [password "<password>"]
	args := []string{"dev", "wifi", "connect", ssid}
	if strings.TrimSpace(password) != "" {
		args = append(args, "password", password)
	}
	_, err := nm.runCmd(30*time.Second, "nmcli", args...)
	if err != nil {
		return fmt.Errorf("WiFi连接失败（需要 NetworkManager + nmcli）: %w", err)
	}
	return nil
}
