package display

import (
	"fmt"
	"os/exec"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	appcfg "totoro-device/config"
	"totoro-device/internal/database"
	"totoro-device/internal/frp"
	"totoro-device/internal/network"
)

// AppServices UI 业务服务集合：统一封装 config / network / frp
type AppServices struct {
	Config  *appcfg.Config
	Network network.Manager
	FRP     frp.Client
	Speed   *NetSpeedSampler

	mu sync.Mutex // 保护 Config 写入
}

func NewAppServices(cfg *appcfg.Config, nm network.Manager, fc frp.Client) *AppServices {
	return &AppServices{
		Config:  cfg,
		Network: nm,
		FRP:     fc,
		Speed:   NewNetSpeedSampler(),
	}
}

func (s *AppServices) GetNetworkStatus() (*network.NetworkStatus, error) {
	if s.Network == nil {
		return nil, fmt.Errorf("network manager 未初始化")
	}
	st, err := s.Network.GetNetworkStatus()
	if err != nil {
		return nil, err
	}
	return st, nil
}

func (s *AppServices) ScanWiFi() ([]network.WiFiNetwork, error) {
	if s.Network == nil {
		return nil, fmt.Errorf("network manager 未初始化")
	}
	return s.Network.ScanWiFi(network.ScanWiFiOptions{AllowRedacted: true})
}

func (s *AppServices) ConnectWiFi(ssid, password string) error {
	if s.Network == nil {
		return fmt.Errorf("network manager 未初始化")
	}
	ssid = strings.TrimSpace(ssid)
	if ssid == "" {
		return fmt.Errorf("SSID 不能为空")
	}

	if err := s.Network.ConfigureWiFi(ssid, password); err != nil {
		return err
	}

	// 记忆 WiFi（写入 config）
	if s.Config != nil {
		s.mu.Lock()
		defer s.mu.Unlock()

		// 选择 WiFi 作为主网络接口（重启后依旧优先 WiFi）
		s.Config.Network.Interface = "wlan0"
		s.Config.Network.IPMode = "dhcp"

		// 更新/追加 profile
		found := false
		for i := range s.Config.Network.WiFiProfiles {
			if strings.TrimSpace(s.Config.Network.WiFiProfiles[i].SSID) == ssid {
				s.Config.Network.WiFiProfiles[i].Password = password
				s.Config.Network.WiFiProfiles[i].AutoConnect = true
				found = true
				break
			}
		}
		if !found {
			s.Config.Network.WiFiProfiles = append(s.Config.Network.WiFiProfiles, appcfg.WiFiProfile{
				SSID:        ssid,
				Password:    password,
				Security:    "",
				AutoConnect: true,
				Priority:    10,
			})
		}
		_ = s.Config.Save()
	}

	return nil
}

func (s *AppServices) ForgetWiFi(ssid string) error {
	ssid = strings.TrimSpace(ssid)
	if ssid == "" {
		return fmt.Errorf("SSID 不能为空")
	}
	if s.Config == nil {
		return fmt.Errorf("config 未初始化")
	}

	// 1) 删除保存的 profile（包含密码）
	s.mu.Lock()
	out := make([]appcfg.WiFiProfile, 0, len(s.Config.Network.WiFiProfiles))
	for _, p := range s.Config.Network.WiFiProfiles {
		if strings.TrimSpace(p.SSID) != ssid {
			out = append(out, p)
		}
	}
	s.Config.Network.WiFiProfiles = out
	// 旧字段也同步清空（避免残留）
	if strings.TrimSpace(s.Config.Network.WiFi.SSID) == ssid {
		s.Config.Network.WiFi.SSID = ""
		s.Config.Network.WiFi.Password = ""
		s.Config.Network.WiFi.Security = ""
	}
	_ = s.Config.Save()
	s.mu.Unlock()

	// 2) 立即断开 WiFi（忘记 WiFi 应该断开连接）
	if s.Network != nil {
		_ = s.Network.DisconnectWiFi() // best-effort
	}

	// 3) 有以太网时优先回落到以太网（DHCP）
	// - Linux: 通过 /sys/class/net/eth0/carrier 判断是否插网线
	if runtime.GOOS == "linux" && s.Network != nil {
		if carrierUp("/sys/class/net/eth0/carrier") {
			_ = s.ApplyDHCP("eth0", strings.TrimSpace(s.Config.Network.DNS))
		}
	}

	return nil
}

func carrierUp(path string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(b)) == "1"
}

func (s *AppServices) ApplyDHCP(iface, dns string) error {
	if s.Network == nil {
		return fmt.Errorf("network manager 未初始化")
	}
	iface = strings.TrimSpace(iface)
	if iface == "" {
		iface = "eth0"
	}
	cfg := network.ApplyConfig{
		Interface: iface,
		IPMode:    "dhcp",
		DNS:       strings.TrimSpace(dns),
	}
	if err := s.Network.ApplyNetworkConfig(cfg); err != nil {
		return err
	}
	// 写入 config
	if s.Config != nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.Config.Network.Interface = iface
		s.Config.Network.IPMode = "dhcp"
		if cfg.DNS != "" {
			s.Config.Network.DNS = cfg.DNS
		}
		_ = s.Config.Save()
	}
	return nil
}

func (s *AppServices) ApplyStaticIP(iface, ip, netmask, gateway, dns string) error {
	if s.Network == nil {
		return fmt.Errorf("network manager 未初始化")
	}
	if strings.TrimSpace(ip) == "" {
		return fmt.Errorf("IP 不能为空")
	}

	cfg := network.ApplyConfig{
		Interface: iface,
		IPMode:    "static",
		IP:        strings.TrimSpace(ip),
		Netmask:   strings.TrimSpace(netmask),
		Gateway:   strings.TrimSpace(gateway),
		DNS:       strings.TrimSpace(dns),
	}
	if err := s.Network.ApplyNetworkConfig(cfg); err != nil {
		return err
	}

	// 写入 config
	if s.Config != nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.Config.Network.Interface = iface
		s.Config.Network.IPMode = "static"
		s.Config.Network.IP = cfg.IP
		s.Config.Network.Netmask = cfg.Netmask
		s.Config.Network.Gateway = cfg.Gateway
		s.Config.Network.DNS = cfg.DNS
		_ = s.Config.Save()
	}
	return nil
}

func (s *AppServices) GetTransferRateKBps(iface string) (up, down float64) {
	if s.Speed == nil {
		return 0, 0
	}
	up, down, _ = s.Speed.SampleKBps(iface)
	return up, down
}

func (s *AppServices) GetTunnels() ([]*frp.Tunnel, error) {
	if s.FRP == nil {
		return []*frp.Tunnel{}, nil
	}
	return s.FRP.GetTunnels()
}

func (s *AppServices) DeleteTunnel(name string) error {
	if s.FRP == nil {
		return fmt.Errorf("FRP 未初始化")
	}
	return s.FRP.RemoveTunnel(name)
}

func (s *AppServices) UpdateTunnel(oldName string, t *frp.Tunnel) error {
	if s.FRP == nil {
		return fmt.Errorf("FRP 未初始化")
	}
	return s.FRP.UpdateTunnel(oldName, t)
}

func (s *AppServices) GetFRPStatus() (*frp.FRPStatus, error) {
	if s.FRP == nil {
		return &frp.FRPStatus{Connected: false}, nil
	}
	return s.FRP.GetStatus()
}

func (s *AppServices) SetSystemVolume(vol int) error {
	if s.Config == nil {
		return fmt.Errorf("config 未初始化")
	}
	if vol < 0 {
		vol = 0
	}
	if vol > 30 {
		vol = 30
	}

	// 先落盘
	s.mu.Lock()
	s.Config.System.Volume = &vol
	_ = s.Config.Save()
	s.mu.Unlock()

	// Linux 设备侧：应用到 ALSA（Luckfox 音频文档方式）
	if runtime.GOOS == "linux" {
		if _, err := exec.LookPath("amixer"); err != nil {
			return fmt.Errorf("系统缺少 amixer")
		}
		cmd := exec.Command("amixer", "cset", "name=DAC LINEOUT Volume", strconv.Itoa(vol))
		// 避免卡死
		timer := time.AfterFunc(2*time.Second, func() {
			_ = cmd.Process.Kill()
		})
		defer timer.Stop()
		out, err := cmd.CombinedOutput()
		if err != nil {
			msg := strings.TrimSpace(string(out))
			if msg != "" {
				return fmt.Errorf("设置音量失败: %s", msg)
			}
			return fmt.Errorf("设置音量失败")
		}
	}

	return nil
}

func (s *AppServices) GetBridgeSession() (*database.BridgeSession, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	return database.GetBridgeSession(db)
}
