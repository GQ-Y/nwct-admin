package display

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	appcfg "totoro-device/config"
	"totoro-device/internal/bridgeclient"
	"totoro-device/internal/database"
	"totoro-device/internal/frp"
	"totoro-device/internal/network"
	"totoro-device/internal/system"
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

		// 双网模式（A 有线优先）：不强制把“主接口”切到 WiFi，只保存 WiFi Profile 供自动连接

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

	// 双网策略路由：有线优先 + WiFi 仍可达
	network.EnsureDualNetworkWiredPreferred(s.Network)

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
	// 忘记当前 WiFi 后，网络偏好回落到以太网（即使未插网线也允许离线）
	if strings.TrimSpace(s.Config.Network.Interface) == "wlan0" {
		s.Config.Network.Interface = "eth0"
		s.Config.Network.IPMode = "dhcp"
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

	// 先落盘（用于重启后保持）
	s.mu.Lock()
	s.Config.System.Volume = &vol
	_ = s.Config.Save()
	s.mu.Unlock()

	// Linux 设备侧：立即应用到 ALSA（Luckfox 音频文档方式）
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

func (s *AppServices) SetSystemBrightness(percent int) error {
	if s.Config == nil {
		return fmt.Errorf("config 未初始化")
	}
	// 限制：亮度最低 10%，避免用户设为 0 后彻底黑屏无法操作
	if percent < 10 {
		percent = 10
	}
	if percent > 100 {
		percent = 100
	}

	// 先落盘（用于重启保持）
	s.mu.Lock()
	s.Config.System.Brightness = &percent
	_ = s.Config.Save()
	s.mu.Unlock()

	// Linux 设备侧：立即应用（写 sysfs 背光）
	if runtime.GOOS == "linux" {
		bl, err := system.DiscoverBacklight()
		if err != nil || bl == nil {
			return fmt.Errorf("未检测到背光设备")
		}
		if err := bl.SetPercent(percent); err != nil {
			return err
		}
	}
	return nil
}

func (s *AppServices) SetScreenOffSeconds(sec int) error {
	if s.Config == nil {
		return fmt.Errorf("config 未初始化")
	}
	if sec < 0 {
		sec = 0
	}
	// 只落盘，运行时逻辑由 display manager 执行
	s.mu.Lock()
	s.Config.System.ScreenOffSeconds = &sec
	_ = s.Config.Save()
	s.mu.Unlock()
	return nil
}

func (s *AppServices) ConnectPublicNode(nodeID string) error {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return fmt.Errorf("node_id 不能为空")
	}
	if s.Config == nil {
		return fmt.Errorf("config 未初始化")
	}
	if s.FRP == nil {
		return fmt.Errorf("FRP 未初始化")
	}
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}

	var res *bridgeclient.PublicNodeConnectResp
	err := s.RegisterBridgeAndRetryOn401(func(bc *bridgeclient.Client) error {
		r, e := bc.ConnectPublicNode(nodeID)
		if e != nil {
			return e
		}
		res = r
		return nil
	})
	if err != nil {
		return err
	}
	if res == nil || len(res.Node.Endpoints) == 0 {
		return fmt.Errorf("桥梁未返回 endpoints")
	}
	ep := res.Node.Endpoints[0]

	s.Config.FRPServer.Mode = appcfg.FRPModePublic
	s.Config.FRPServer.Public.LastResolveError = ""
	s.Config.FRPServer.Public.Server = fmt.Sprintf("%s:%d", strings.TrimSpace(ep.Addr), ep.Port)
	s.Config.FRPServer.Public.TotoroTicket = strings.TrimSpace(res.ConnectionTicket)
	s.Config.FRPServer.Public.TicketExpiresAt = strings.TrimSpace(res.ExpiresAt)
	s.Config.FRPServer.Public.Token = ""
	s.Config.FRPServer.Public.DomainSuffix = strings.TrimPrefix(strings.TrimSpace(res.Node.DomainSuffix), ".")
	s.Config.FRPServer.Public.HTTPEnabled = res.Node.HTTPEnabled
	s.Config.FRPServer.Public.HTTPSEnabled = res.Node.HTTPSEnabled
	s.Config.FRPServer.SyncActiveFromMode()
	_ = s.Config.Save()

	// 记录选择：node_id 优先于 invite_code
	_ = database.SetPublicNodeID(db, nodeID)
	_ = database.ClearPublicInviteCode(db)

	return s.FRP.Connect()
}

func (s *AppServices) ConnectInvite(code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return fmt.Errorf("邀请码不能为空")
	}
	if s.Config == nil {
		return fmt.Errorf("config 未初始化")
	}
	if s.FRP == nil {
		return fmt.Errorf("FRP 未初始化")
	}
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}

	var res *bridgeclient.RedeemResp
	err := s.RegisterBridgeAndRetryOn401(func(bc *bridgeclient.Client) error {
		r, e := bc.RedeemInvite(code)
		if e != nil {
			return e
		}
		res = r
		return nil
	})
	if err != nil {
		return err
	}
	if res == nil || len(res.Node.Endpoints) == 0 {
		return fmt.Errorf("桥梁未返回 endpoints")
	}
	ep := res.Node.Endpoints[0]

	s.Config.FRPServer.Mode = appcfg.FRPModePublic
	s.Config.FRPServer.Public.LastResolveError = ""
	s.Config.FRPServer.Public.Server = fmt.Sprintf("%s:%d", strings.TrimSpace(ep.Addr), ep.Port)
	s.Config.FRPServer.Public.TotoroTicket = strings.TrimSpace(res.ConnectionTicket)
	s.Config.FRPServer.Public.TicketExpiresAt = strings.TrimSpace(res.ExpiresAt)
	s.Config.FRPServer.Public.Token = ""
	s.Config.FRPServer.Public.DomainSuffix = strings.TrimPrefix(strings.TrimSpace(res.Node.DomainSuffix), ".")
	s.Config.FRPServer.Public.HTTPEnabled = res.Node.HTTPEnabled
	s.Config.FRPServer.Public.HTTPSEnabled = res.Node.HTTPSEnabled
	s.Config.FRPServer.SyncActiveFromMode()
	_ = s.Config.Save()

	_ = database.SetPublicInviteCode(db, code)
	_ = database.ClearPublicNodeID(db)

	return s.FRP.Connect()
}

func (s *AppServices) MeasureLatencyToServer(addr string) (int, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return 0, fmt.Errorf("addr empty")
	}
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, 900*time.Millisecond)
	if err != nil {
		return 0, err
	}
	_ = conn.Close()
	return int(time.Since(start).Milliseconds()), nil
}

func (s *AppServices) GetBridgeSession() (*database.BridgeSession, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	return database.GetBridgeSession(db)
}

func (s *AppServices) GetDeviceCrypto() (*database.DeviceCrypto, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	return database.GetOrCreateDeviceCrypto(db)
}

// bridgeClientForUI 构造桥梁客户端（可自动注册 token）
func (s *AppServices) bridgeClientForUI() (*bridgeclient.Client, *database.BridgeSession, *database.DeviceCrypto, error) {
	if s.Config == nil {
		return nil, nil, nil, fmt.Errorf("config 未初始化")
	}
	db := database.GetDB()
	if db == nil {
		return nil, nil, nil, fmt.Errorf("数据库未初始化")
	}
	dc, err := database.GetOrCreateDeviceCrypto(db)
	if err != nil || dc == nil || strings.TrimSpace(dc.PrivKeyB64) == "" || strings.TrimSpace(dc.PubKeyB64) == "" {
		return nil, nil, nil, fmt.Errorf("设备密钥不可用")
	}
	sess, _ := database.GetBridgeSession(db)

	bc := &bridgeclient.Client{
		BaseURL:          appcfg.ResolveBridgeBase(s.Config),
		DeviceID:         strings.TrimSpace(s.Config.Device.ID),
		DevicePrivKeyB64: strings.TrimSpace(dc.PrivKeyB64),
		DeviceToken:      "",
	}
	if sess != nil && !database.BridgeSessionExpired(sess, 30*time.Second) {
		bc.DeviceToken = strings.TrimSpace(sess.DeviceToken)
	}

	return bc, sess, dc, nil
}

func (s *AppServices) ensureBridgeToken(bc *bridgeclient.Client, dc *database.DeviceCrypto) error {
	if bc == nil || s.Config == nil {
		return fmt.Errorf("bridge client 未初始化")
	}
	if strings.TrimSpace(bc.DeviceToken) != "" {
		return nil
	}
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}
	deviceID := strings.TrimSpace(s.Config.Device.ID)
	mac := strings.TrimSpace(s.Config.Bridge.LastMAC)
	if mac == "" && s.Network != nil {
		// best-effort：尝试用当前接口 MAC
		if st, _ := s.Network.GetNetworkStatus(); st != nil {
			ifaces, _ := s.Network.GetInterfaces()
			for _, it := range ifaces {
				if strings.TrimSpace(it.Name) == strings.TrimSpace(st.CurrentInterface) && strings.TrimSpace(it.MAC) != "" {
					mac = strings.TrimSpace(it.MAC)
					break
				}
			}
		}
	}
	if deviceID == "" || mac == "" {
		return fmt.Errorf("设备未初始化（缺少 device_id/mac）")
	}
	reg, err := bc.Register(deviceID, mac, strings.TrimSpace(dc.PubKeyB64))
	if err != nil || reg == nil || strings.TrimSpace(reg.DeviceToken) == "" {
		return fmt.Errorf("桥梁注册失败")
	}
	_ = database.UpsertBridgeSession(db, database.BridgeSession{
		BridgeURL:   bc.BaseURL,
		DeviceID:    deviceID,
		MAC:         mac,
		DeviceToken: strings.TrimSpace(reg.DeviceToken),
		ExpiresAt:   bridgeclient.ParseExpiresAt(reg.ExpiresAt),
	})
	bc.DeviceToken = strings.TrimSpace(reg.DeviceToken)
	return nil
}

func (s *AppServices) RegisterBridgeAndRetryOn401(do func(bc *bridgeclient.Client) error) error {
	bc, _, dc, err := s.bridgeClientForUI()
	if err != nil {
		return err
	}
	// token 为空先注册
	if err := s.ensureBridgeToken(bc, dc); err != nil {
		return err
	}
	err = do(bc)
	if err == nil {
		return nil
	}
	// 401：强制重注册并重试一次
	if strings.Contains(err.Error(), "status=401") {
		bc.DeviceToken = ""
		_ = s.ensureBridgeToken(bc, dc)
		return do(bc)
	}
	return err
}
