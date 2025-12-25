package display

import (
	"fmt"
	"strings"
	"sync"

	appcfg "nwct/client-nps/config"
	"nwct/client-nps/internal/frp"
	"nwct/client-nps/internal/network"
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
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]appcfg.WiFiProfile, 0, len(s.Config.Network.WiFiProfiles))
	for _, p := range s.Config.Network.WiFiProfiles {
		if strings.TrimSpace(p.SSID) != ssid {
			out = append(out, p)
		}
	}
	s.Config.Network.WiFiProfiles = out
	return s.Config.Save()
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


