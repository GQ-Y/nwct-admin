//go:build linux && !preview

package network

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (nm *networkManager) pickWiFiInterface() string {
	ifaces, err := nm.GetInterfaces()
	if err == nil {
		for _, it := range ifaces {
			if it.Type == "wifi" && strings.TrimSpace(it.Name) != "" {
				return strings.TrimSpace(it.Name)
			}
		}
	}
	// 兜底：大多数 Buildroot 是 wlan0
	return "wlan0"
}

func (nm *networkManager) scanWiFiLinuxFallback(opts ScanWiFiOptions) ([]WiFiNetwork, error) {
	iface := nm.pickWiFiInterface()

	// 尽量把接口拉起来（很多固件默认 up）
	if nm.hasCmd("ip") {
		_, _ = nm.runCmd(3*time.Second, "ip", "link", "set", iface, "up")
	}

	type ap struct {
		ssid     string
		signalDB float64
		sec      string
		inUse    bool
	}

	var (
		curSSID   string
		curSignal *float64
		curSec    string
		res       = map[string]*ap{}
	)

	flush := func() {
		s := strings.TrimSpace(curSSID)
		if s == "" {
			curSSID = ""
			curSignal = nil
			curSec = ""
			return
		}
		it := res[s]
		if it == nil {
			it = &ap{ssid: s}
			res[s] = it
		}
		if curSignal != nil {
			// 取更强的信号
			if it.signalDB == 0 || *curSignal > it.signalDB {
				it.signalDB = *curSignal
			}
		}
		if curSec != "" {
			it.sec = curSec
		}
		curSSID = ""
		curSignal = nil
		curSec = ""
	}

	reSignal := regexp.MustCompile(`signal:\s*([\-0-9.]+)\s*dBm`)
	parseIwScan := func(out string) {
		for _, ln := range strings.Split(out, "\n") {
			line := strings.TrimSpace(ln)
			if strings.HasPrefix(line, "BSS ") {
				flush()
				continue
			}
			if strings.HasPrefix(line, "SSID:") {
				curSSID = strings.TrimSpace(strings.TrimPrefix(line, "SSID:"))
				continue
			}
			if m := reSignal.FindStringSubmatch(line); len(m) == 2 {
				if v, e := strconv.ParseFloat(m[1], 64); e == nil {
					curSignal = &v
				}
				continue
			}
			// 简单推断安全类型（够用即可）
			if strings.Contains(line, "RSN:") {
				curSec = "WPA2/WPA3"
			} else if strings.Contains(line, "WPA:") {
				if curSec == "" {
					curSec = "WPA"
				}
			}
		}
		flush()
	}

	// 优先使用 iw（直接扫描）；某些驱动/固件在 wpa_supplicant 运行时会返回 busy
	if nm.hasCmd("iw") {
		if out, err := nm.runCmd(20*time.Second, "iw", "dev", iface, "scan"); err == nil {
			parseIwScan(out)
		} else if nm.hasCmd("wpa_cli") {
			// 降级：通过 wpa_supplicant 扫描
			_, _ = nm.runCmd(4*time.Second, "wpa_cli", "-i", iface, "scan")
			time.Sleep(2 * time.Second)
			rs, e2 := nm.runCmd(6*time.Second, "wpa_cli", "-i", iface, "scan_results")
			if e2 != nil {
				return nil, fmt.Errorf("WiFi扫描失败：iw busy 且 wpa_cli scan_results 失败: %v", e2)
			}
			// scan_results 格式：
			// bssid / frequency / signal level / flags / ssid
			lines := strings.Split(rs, "\n")
			for _, ln := range lines {
				ln = strings.TrimSpace(ln)
				if ln == "" || strings.HasPrefix(ln, "bssid") {
					continue
				}
				parts := strings.Split(ln, "\t")
				if len(parts) < 5 {
					continue
				}
				s := strings.TrimSpace(parts[4])
				if s == "" {
					continue
				}
				sig, _ := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
				flags := strings.TrimSpace(parts[3])
				sec := "OPEN"
				low := strings.ToLower(flags)
				if strings.Contains(low, "wpa2") || strings.Contains(low, "rsn") {
					sec = "WPA2/WPA3"
				} else if strings.Contains(low, "wpa") {
					sec = "WPA"
				}
				it := res[s]
				if it == nil {
					it = &ap{ssid: s}
					res[s] = it
				}
				if it.signalDB == 0 || sig > it.signalDB {
					it.signalDB = sig
				}
				it.sec = sec
			}
		} else {
			return nil, fmt.Errorf("WiFi扫描失败（iw）: %w", err)
		}
	} else if nm.hasCmd("wpa_cli") {
		// 没有 iw：仍可尝试走 wpa_cli
		_, _ = nm.runCmd(4*time.Second, "wpa_cli", "-i", iface, "scan")
		time.Sleep(2 * time.Second)
		rs, e2 := nm.runCmd(6*time.Second, "wpa_cli", "-i", iface, "scan_results")
		if e2 != nil {
			return nil, fmt.Errorf("WiFi扫描失败：系统缺少 iw，且 wpa_cli scan_results 失败: %v", e2)
		}
		lines := strings.Split(rs, "\n")
		for _, ln := range lines {
			ln = strings.TrimSpace(ln)
			if ln == "" || strings.HasPrefix(ln, "bssid") {
				continue
			}
			parts := strings.Split(ln, "\t")
			if len(parts) < 5 {
				continue
			}
			s := strings.TrimSpace(parts[4])
			if s == "" {
				continue
			}
			sig, _ := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
			flags := strings.TrimSpace(parts[3])
			sec := "OPEN"
			low := strings.ToLower(flags)
			if strings.Contains(low, "wpa2") || strings.Contains(low, "rsn") {
				sec = "WPA2/WPA3"
			} else if strings.Contains(low, "wpa") {
				sec = "WPA"
			}
			it := res[s]
			if it == nil {
				it = &ap{ssid: s}
				res[s] = it
			}
			if it.signalDB == 0 || sig > it.signalDB {
				it.signalDB = sig
			}
			it.sec = sec
		}
	} else {
		return nil, fmt.Errorf("WiFi扫描失败：系统缺少 nmcli/iw/wpa_cli")
	}

	// 当前连接 SSID（best-effort）
	curConnSSID := ""
	if nm.hasCmd("wpa_cli") {
		if st, e := nm.runCmd(2*time.Second, "wpa_cli", "-i", iface, "status"); e == nil {
			for _, ln := range strings.Split(st, "\n") {
				ln = strings.TrimSpace(ln)
				if strings.HasPrefix(ln, "ssid=") {
					curConnSSID = strings.TrimSpace(strings.TrimPrefix(ln, "ssid="))
					break
				}
			}
		}
	} else if nm.hasCmd("iw") {
		if st, e := nm.runCmd(2*time.Second, "iw", "dev", iface, "link"); e == nil {
			for _, ln := range strings.Split(st, "\n") {
				ln = strings.TrimSpace(ln)
				if strings.HasPrefix(ln, "SSID:") {
					curConnSSID = strings.TrimSpace(strings.TrimPrefix(ln, "SSID:"))
					break
				}
			}
		}
	}

	list := make([]WiFiNetwork, 0, len(res))
	for _, it := range res {
		sec := it.sec
		if sec == "" {
			sec = "OPEN"
		}
		sig := signalDbmToPercent(it.signalDB)
		list = append(list, WiFiNetwork{
			SSID:     it.ssid,
			Signal:   sig,
			Security: sec,
			InUse:    it.ssid == curConnSSID,
		})
	}

	// 信号强度降序
	sort.SliceStable(list, func(i, j int) bool { return list[i].Signal > list[j].Signal })
	return list, nil
}

func signalDbmToPercent(dbm float64) int {
	// 简单映射：[-100, -50] -> [0, 100]
	if dbm >= -50 {
		return 100
	}
	if dbm <= -100 {
		return 0
	}
	return int((dbm + 100) * 2)
}

func (nm *networkManager) connectWiFiLinuxFallback(ssid, password string) error {
	ssid = strings.TrimSpace(ssid)
	if ssid == "" {
		return fmt.Errorf("SSID 不能为空")
	}
	if !nm.hasCmd("wpa_supplicant") || !nm.hasCmd("wpa_cli") {
		return fmt.Errorf("WiFi连接失败：系统缺少 wpa_supplicant/wpa_cli（也缺少 nmcli）")
	}
	if !nm.hasCmd("udhcpc") {
		return fmt.Errorf("WiFi连接失败：系统缺少 udhcpc（无法获取 DHCP 地址）")
	}

	iface := nm.pickWiFiInterface()
	if nm.hasCmd("ip") {
		_, _ = nm.runCmd(3*time.Second, "ip", "link", "set", iface, "up")
	}

	// 写临时配置（Buildroot 常见做法）
	// 注意：psk 允许使用引号形式的明文 passphrase
	conf := "/tmp/wpa_supplicant_" + iface + ".conf"
	var b strings.Builder
	b.WriteString("ctrl_interface=/var/run/wpa_supplicant\n")
	b.WriteString("update_config=1\n")
	b.WriteString("ap_scan=1\n")
	b.WriteString("network={\n")
	b.WriteString("  ssid=\"")
	b.WriteString(escapeWpaString(ssid))
	b.WriteString("\"\n")
	if strings.TrimSpace(password) == "" {
		b.WriteString("  key_mgmt=NONE\n")
	} else {
		b.WriteString("  psk=\"")
		b.WriteString(escapeWpaString(password))
		b.WriteString("\"\n")
	}
	b.WriteString("}\n")
	if _, err := nm.runCmd(3*time.Second, "sh", "-lc", fmt.Sprintf("cat > %s <<'EOF'\n%sEOF\n", conf, b.String())); err != nil {
		return err
	}

	// 尽量停掉旧的 wpa_supplicant（不强依赖 pkill）
	_, _ = nm.runCmd(2*time.Second, "sh", "-lc", fmt.Sprintf("wpa_cli -i %s terminate >/dev/null 2>&1 || true", iface))

	// 启动 wpa_supplicant
	if _, err := nm.runCmd(6*time.Second, "wpa_supplicant", "-B", "-i", iface, "-c", conf); err != nil {
		return fmt.Errorf("wpa_supplicant 启动失败: %w", err)
	}

	// 等待关联（最多 15 秒）
	assocOK := false
	for i := 0; i < 15; i++ {
		time.Sleep(1 * time.Second)
		if st, err := nm.runCmd(2*time.Second, "wpa_cli", "-i", iface, "status"); err == nil {
			if strings.Contains(st, "wpa_state=COMPLETED") {
				assocOK = true
				break
			}
		}
	}
	if !assocOK {
		return fmt.Errorf("WiFi连接超时：未完成关联（wpa_state!=COMPLETED）")
	}

	// DHCP 获取地址（-n：失败退出）
	if _, err := nm.runCmd(20*time.Second, "udhcpc", "-i", iface, "-n", "-q", "-T", "3", "-t", "3"); err != nil {
		return err
	}
	return nil
}

func escapeWpaString(s string) string {
	// 最小转义：\" 和 \\（避免破坏配置）
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}
