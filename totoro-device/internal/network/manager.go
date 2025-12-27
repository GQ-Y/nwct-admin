package network

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
	"totoro-device/internal/logger"
)

// Manager 网络管理器接口
type Manager interface {
	GetInterfaces() ([]Interface, error)
	ConfigureWiFi(ssid, password string) error
	ScanWiFi(opts ScanWiFiOptions) ([]WiFiNetwork, error)
	GetNetworkStatus() (*NetworkStatus, error)
	TestConnection(target string) error

	// ApplyNetworkConfig 实际下发网络配置（dhcp/static）
	ApplyNetworkConfig(cfg ApplyConfig) error
}

type ApplyConfig struct {
	Interface string `json:"interface"` // 如 eth0/en0；可为空（则自动选当前接口）
	IPMode    string `json:"ip_mode"`   // dhcp/static
	IP        string `json:"ip"`
	Netmask   string `json:"netmask"` // 支持 255.255.255.0 或 /24 或 24
	Gateway   string `json:"gateway"`
	DNS       string `json:"dns"` // 支持 "8.8.8.8,1.1.1.1" 或空格分隔
}

// ScanWiFiOptions WiFi扫描选项
type ScanWiFiOptions struct {
	AllowRedacted bool `json:"allow_redacted"`
}

// WiFiNetwork WiFi网络信息
type WiFiNetwork struct {
	SSID     string `json:"ssid"`
	Signal   int    `json:"signal"`   // 0-100
	Security string `json:"security"` // WPA2/WPA3/OPEN...
	InUse    bool   `json:"in_use"`
}

// Interface 网络接口
type Interface struct {
	Name    string `json:"name"`
	Type    string `json:"type"`   // ethernet, wifi
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
	Gateway          string  `json:"gateway"`
	Status           string  `json:"status"` // connected, disconnected
	UploadSpeed      float64 `json:"upload_speed"`
	DownloadSpeed    float64 `json:"download_speed"`
	Latency          int     `json:"latency"`
}

// networkManager 网络管理器实现
type networkManager struct {
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
	logger.Info("配置WiFi连接: SSID=%s", ssid)

	switch runtime.GOOS {
	case "linux":
		if nm.hasCmd("nmcli") {
			return nm.connectWiFiLinuxNmcli(ssid, password)
		}
		return nm.connectWiFiLinuxFallback(ssid, password)
	case "darwin":
		return nm.connectWiFiDarwinNetworksetup(ssid, password)
	default:
		return fmt.Errorf("当前系统不支持WiFi配置: %s", runtime.GOOS)
	}
}

// ScanWiFi 扫描WiFi网络
func (nm *networkManager) ScanWiFi(opts ScanWiFiOptions) ([]WiFiNetwork, error) {
	switch runtime.GOOS {
	case "linux":
		if nm.hasCmd("nmcli") {
			return nm.scanWiFiLinuxNmcli()
		}
		return nm.scanWiFiLinuxFallback(opts)
	case "darwin":
		return nm.scanWiFiDarwinAirport(opts.AllowRedacted)
	default:
		return nil, fmt.Errorf("当前系统不支持WiFi扫描: %s", runtime.GOOS)
	}
}

func (nm *networkManager) runCmd(timeout time.Duration, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	s := strings.TrimSpace(string(out))
	if ctx.Err() == context.DeadlineExceeded {
		return s, fmt.Errorf("命令超时: %s %v", name, args)
	}
	if err != nil {
		if s != "" {
			return s, fmt.Errorf("命令失败: %s %v: %v: %s", name, args, err, s)
		}
		return s, fmt.Errorf("命令失败: %s %v: %v", name, args, err)
	}
	return s, nil
}

func (nm *networkManager) hasCmd(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func (nm *networkManager) ApplyNetworkConfig(cfg ApplyConfig) error {
	mode := strings.ToLower(strings.TrimSpace(cfg.IPMode))
	if mode == "" {
		mode = "dhcp"
	}
	if mode != "dhcp" && mode != "static" {
		return fmt.Errorf("ip_mode 无效: %s", cfg.IPMode)
	}

	iface := strings.TrimSpace(cfg.Interface)
	if iface == "" {
		// 自动选择当前有 IP 的接口
		st, err := nm.GetNetworkStatus()
		if err == nil && st != nil && strings.TrimSpace(st.CurrentInterface) != "" {
			iface = strings.TrimSpace(st.CurrentInterface)
		}
	}
	if iface == "" {
		return fmt.Errorf("未指定网络接口，且无法自动识别当前接口")
	}

	switch runtime.GOOS {
	case "darwin":
		if mode == "dhcp" {
			return nm.applyDHCPDarwin(iface, cfg.DNS)
		}
		return nm.applyStaticDarwin(iface, cfg.IP, cfg.Netmask, cfg.Gateway, cfg.DNS)
	case "linux":
		if mode == "dhcp" {
			// 优先 nmcli（NetworkManager），无 nmcli 则降级到 Buildroot 常见的 udhcpc/ip/ifconfig
			if nm.hasCmd("nmcli") {
				return nm.applyDHCPLinux(iface, cfg.DNS)
			}
			return nm.applyDHCPLinuxFallback(iface, cfg.DNS)
		}
		if nm.hasCmd("nmcli") {
			return nm.applyStaticLinux(iface, cfg.IP, cfg.Netmask, cfg.Gateway, cfg.DNS)
		}
		return nm.applyStaticLinuxFallback(iface, cfg.IP, cfg.Netmask, cfg.Gateway, cfg.DNS)
	default:
		return fmt.Errorf("当前系统不支持 IP 配置下发: %s", runtime.GOOS)
	}
}

func (nm *networkManager) splitDNS(dns string) []string {
	s := strings.TrimSpace(dns)
	if s == "" {
		return nil
	}
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func netmaskToPrefix(netmask string) (int, error) {
	s := strings.TrimSpace(netmask)
	if s == "" {
		return 0, fmt.Errorf("netmask 不能为空")
	}
	// "/24" or "24"
	s = strings.TrimPrefix(s, "/")
	if n, err := strconv.Atoi(s); err == nil && n >= 0 && n <= 32 {
		return n, nil
	}
	// dotted
	ip := net.ParseIP(s)
	if ip == nil {
		return 0, fmt.Errorf("netmask 无效: %s", netmask)
	}
	ip = ip.To4()
	if ip == nil {
		return 0, fmt.Errorf("netmask 不是 IPv4: %s", netmask)
	}
	mask := net.IPMask(ip)
	ones, bits := mask.Size()
	if bits != 32 || ones < 0 {
		return 0, fmt.Errorf("netmask 无效: %s", netmask)
	}
	return ones, nil
}

func (nm *networkManager) darwinServiceForDevice(device string) (string, error) {
	// networksetup -listallhardwareports 输出块：Hardware Port: Wi-Fi / Device: en0
	out, err := nm.runCmd(4*time.Second, "networksetup", "-listallhardwareports")
	if err != nil {
		return "", err
	}
	lines := strings.Split(out, "\n")
	var curPort, curDev string
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if strings.HasPrefix(ln, "Hardware Port:") {
			curPort = strings.TrimSpace(strings.TrimPrefix(ln, "Hardware Port:"))
			curDev = ""
			continue
		}
		if strings.HasPrefix(ln, "Device:") {
			curDev = strings.TrimSpace(strings.TrimPrefix(ln, "Device:"))
			if curDev == device && curPort != "" {
				return curPort, nil
			}
		}
	}
	return "", fmt.Errorf("未找到对应网络服务: device=%s（networksetup -listallhardwareports）", device)
}

func (nm *networkManager) applyDHCPDarwin(device, dns string) error {
	svc, err := nm.darwinServiceForDevice(device)
	if err != nil {
		return err
	}
	logger.Info("设置 DHCP: service=%s device=%s", svc, device)
	if _, err := nm.runCmd(10*time.Second, "networksetup", "-setdhcp", svc); err != nil {
		return err
	}
	servers := nm.splitDNS(dns)
	if len(servers) > 0 {
		args := append([]string{"-setdnsservers", svc}, servers...)
		if _, err := nm.runCmd(6*time.Second, "networksetup", args...); err != nil {
			return err
		}
	}
	return nil
}

func (nm *networkManager) applyStaticDarwin(device, ip, netmask, gateway, dns string) error {
	ip = strings.TrimSpace(ip)
	netmask = strings.TrimSpace(netmask)
	gateway = strings.TrimSpace(gateway)
	if ip == "" || netmask == "" || gateway == "" {
		return fmt.Errorf("静态 IP 配置缺失（ip/netmask/gateway 必填）")
	}
	svc, err := nm.darwinServiceForDevice(device)
	if err != nil {
		return err
	}
	logger.Info("设置静态 IP: service=%s device=%s ip=%s netmask=%s gw=%s", svc, device, ip, netmask, gateway)
	if _, err := nm.runCmd(12*time.Second, "networksetup", "-setmanual", svc, ip, netmask, gateway); err != nil {
		return err
	}
	servers := nm.splitDNS(dns)
	if len(servers) > 0 {
		args := append([]string{"-setdnsservers", svc}, servers...)
		if _, err := nm.runCmd(6*time.Second, "networksetup", args...); err != nil {
			return err
		}
	}
	return nil
}

func (nm *networkManager) applyDHCPLinux(device, dns string) error {
	logger.Info("设置 DHCP: device=%s", device)
	// 找 active connection
	con, err := nm.activeNmcliConnection(device)
	if err != nil {
		return err
	}
	if _, err := nm.runCmd(10*time.Second, "nmcli", "con", "mod", con, "ipv4.method", "auto"); err != nil {
		return err
	}
	servers := nm.splitDNS(dns)
	if len(servers) > 0 {
		if _, err := nm.runCmd(10*time.Second, "nmcli", "con", "mod", con, "ipv4.dns", strings.Join(servers, ",")); err != nil {
			return err
		}
	}
	if _, err := nm.runCmd(15*time.Second, "nmcli", "con", "up", con); err != nil {
		return err
	}
	return nil
}

func (nm *networkManager) applyStaticLinux(device, ip, netmask, gateway, dns string) error {
	ip = strings.TrimSpace(ip)
	netmask = strings.TrimSpace(netmask)
	gateway = strings.TrimSpace(gateway)
	if ip == "" || netmask == "" || gateway == "" {
		return fmt.Errorf("静态 IP 配置缺失（ip/netmask/gateway 必填）")
	}
	prefix, err := netmaskToPrefix(netmask)
	if err != nil {
		return err
	}
	con, err := nm.activeNmcliConnection(device)
	if err != nil {
		return err
	}
	addr := fmt.Sprintf("%s/%d", ip, prefix)
	logger.Info("设置静态 IP: device=%s con=%s addr=%s gw=%s", device, con, addr, gateway)
	if _, err := nm.runCmd(10*time.Second, "nmcli", "con", "mod", con, "ipv4.method", "manual"); err != nil {
		return err
	}
	if _, err := nm.runCmd(10*time.Second, "nmcli", "con", "mod", con, "ipv4.addresses", addr); err != nil {
		return err
	}
	if _, err := nm.runCmd(10*time.Second, "nmcli", "con", "mod", con, "ipv4.gateway", gateway); err != nil {
		return err
	}
	servers := nm.splitDNS(dns)
	if len(servers) > 0 {
		if _, err := nm.runCmd(10*time.Second, "nmcli", "con", "mod", con, "ipv4.dns", strings.Join(servers, ",")); err != nil {
			return err
		}
	}
	if _, err := nm.runCmd(15*time.Second, "nmcli", "con", "up", con); err != nil {
		return err
	}
	return nil
}

func (nm *networkManager) activeNmcliConnection(device string) (string, error) {
	// nmcli -t -f NAME,DEVICE con show --active
	out, err := nm.runCmd(6*time.Second, "nmcli", "-t", "-f", "NAME,DEVICE", "con", "show", "--active")
	if err != nil {
		return "", err
	}
	for _, ln := range strings.Split(out, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		parts := strings.SplitN(ln, ":", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		dev := strings.TrimSpace(parts[1])
		if dev == device && name != "" {
			return name, nil
		}
	}
	return "", fmt.Errorf("未找到活动连接（nmcli）: device=%s", device)
}

func (nm *networkManager) writeResolvConfBestEffort(dns string) {
	servers := nm.splitDNS(dns)
	if len(servers) == 0 {
		return
	}
	var b strings.Builder
	for _, s := range servers {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		b.WriteString("nameserver ")
		b.WriteString(s)
		b.WriteByte('\n')
	}
	content := strings.TrimSpace(b.String())
	if content == "" {
		return
	}
	// Buildroot 常见位置：/etc/resolv.conf
	_ = exec.Command("sh", "-lc", fmt.Sprintf("printf '%s\n' \"%s\" > /etc/resolv.conf", content, strings.ReplaceAll(content, "'", "'\\''"))).Run()
}

func (nm *networkManager) applyDHCPLinuxFallback(device, dns string) error {
	logger.Info("设置 DHCP（fallback）: device=%s", device)

	// 尽量把接口拉起来
	if nm.hasCmd("ip") {
		_, _ = nm.runCmd(3*time.Second, "ip", "link", "set", device, "up")
	} else if nm.hasCmd("ifconfig") {
		_, _ = nm.runCmd(3*time.Second, "ifconfig", device, "up")
	}

	// BusyBox/Buildroot 常见 DHCP 客户端：udhcpc
	if nm.hasCmd("udhcpc") {
		// -n：失败直接退出；-q：安静；-T/-t：超时/次数
		if _, err := nm.runCmd(20*time.Second, "udhcpc", "-i", device, "-n", "-q", "-T", "3", "-t", "3"); err != nil {
			return err
		}
		nm.writeResolvConfBestEffort(dns)
		return nil
	}

	// 兼容 ifup（部分 Buildroot 会带）
	if nm.hasCmd("ifup") {
		if _, err := nm.runCmd(20*time.Second, "ifup", device); err != nil {
			return err
		}
		nm.writeResolvConfBestEffort(dns)
		return nil
	}

	return fmt.Errorf("DHCP 下发失败：系统缺少 nmcli/udhcpc/ifup（Buildroot 通常应提供 udhcpc）")
}

func (nm *networkManager) applyStaticLinuxFallback(device, ip, netmask, gateway, dns string) error {
	ip = strings.TrimSpace(ip)
	netmask = strings.TrimSpace(netmask)
	gateway = strings.TrimSpace(gateway)
	if ip == "" || netmask == "" || gateway == "" {
		return fmt.Errorf("静态 IP 配置缺失（ip/netmask/gateway 必填）")
	}
	prefix, err := netmaskToPrefix(netmask)
	if err != nil {
		return err
	}
	addr := fmt.Sprintf("%s/%d", ip, prefix)
	logger.Info("设置静态 IP（fallback）: device=%s addr=%s gw=%s", device, addr, gateway)

	if nm.hasCmd("ip") {
		_, _ = nm.runCmd(3*time.Second, "ip", "link", "set", device, "up")
		// 清理旧地址（尽量 best-effort，不要因为 flush 失败而中断）
		_, _ = nm.runCmd(4*time.Second, "ip", "addr", "flush", "dev", device)
		if _, err := nm.runCmd(6*time.Second, "ip", "addr", "add", addr, "dev", device); err != nil {
			return err
		}
		if _, err := nm.runCmd(6*time.Second, "ip", "route", "replace", "default", "via", gateway, "dev", device); err != nil {
			return err
		}
		nm.writeResolvConfBestEffort(dns)
		return nil
	}

	// 兜底：ifconfig + route（BusyBox 常见）
	if nm.hasCmd("ifconfig") && nm.hasCmd("route") {
		if _, err := nm.runCmd(6*time.Second, "ifconfig", device, ip, "netmask", netmask, "up"); err != nil {
			return err
		}
		// 先删再加，避免重复 default
		_, _ = nm.runCmd(3*time.Second, "route", "del", "default")
		if _, err := nm.runCmd(6*time.Second, "route", "add", "default", "gw", gateway, device); err != nil {
			// 有些 BusyBox 语法不带 device
			_, err2 := nm.runCmd(6*time.Second, "route", "add", "default", "gw", gateway)
			if err2 != nil {
				return err
			}
		}
		nm.writeResolvConfBestEffort(dns)
		return nil
	}

	return fmt.Errorf("静态 IP 下发失败：系统缺少 ip 或 ifconfig/route")
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

	// 优先使用默认路由的网卡（更符合“当前已连接网络”的直觉，避免 utun/bridge 等虚拟接口干扰）
	if defDev, gw, err := nm.getDefaultRouteDeviceAndGateway(); err == nil && defDev != "" {
		for _, iface := range interfaces {
			if iface.Name == defDev && iface.Status == "up" && iface.IP != "" {
				status.CurrentInterface = iface.Name
				status.IP = iface.IP
				status.Status = "connected"
				status.Gateway = gw
				break
			}
		}
	}

	// 兜底：查找第一个已连接且有IP的“物理接口”
	if status.Status != "connected" {
		for _, iface := range interfaces {
			if iface.Status != "up" || iface.IP == "" {
				continue
			}
			if isVirtualInterfaceName(iface.Name) {
				continue
			}
			status.CurrentInterface = iface.Name
			status.IP = iface.IP
			status.Status = "connected"
			break
		}
	}

	// 测试网络延迟（ping网关或DNS）
	if status.Status == "connected" {
		// 读取默认网关（用于 traceroute 默认目标）
		if status.Gateway == "" {
			if gw, err := nm.getDefaultGateway(); err == nil {
				status.Gateway = gw
			}
		}

		// 尝试ping 8.8.8.8测试连通性
		latency, err := nm.testLatency("8.8.8.8")
		if err == nil {
			status.Latency = latency
		}
	}

	return status, nil
}

func isVirtualInterfaceName(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		return true
	}
	// macOS 常见虚拟/系统接口：utun(vpn), bridge(docker), awdl/llw(Apple), lo
	if strings.HasPrefix(n, "utun") ||
		strings.HasPrefix(n, "bridge") ||
		strings.HasPrefix(n, "awdl") ||
		strings.HasPrefix(n, "llw") ||
		strings.HasPrefix(n, "lo") ||
		strings.HasPrefix(n, "gif") ||
		strings.HasPrefix(n, "stf") ||
		strings.HasPrefix(n, "vmnet") {
		return true
	}
	return false
}

func (nm *networkManager) getDefaultRouteDeviceAndGateway() (string, string, error) {
	switch runtime.GOOS {
	case "darwin":
		out, err := nm.runCmd(3*time.Second, "route", "-n", "get", "default")
		if err != nil {
			return "", "", err
		}
		var gw, dev string
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "gateway:") {
				gw = strings.TrimSpace(strings.TrimPrefix(line, "gateway:"))
			}
			if strings.HasPrefix(line, "interface:") {
				dev = strings.TrimSpace(strings.TrimPrefix(line, "interface:"))
			}
		}
		if dev == "" && gw == "" {
			return "", "", fmt.Errorf("未找到默认路由信息")
		}
		return dev, gw, nil
	case "linux":
		out, err := nm.runCmd(3*time.Second, "ip", "route", "show", "default")
		if err != nil {
			return "", "", err
		}
		// default via 192.168.1.1 dev eth0 ...
		fields := strings.Fields(out)
		gw := ""
		dev := ""
		for i := 0; i < len(fields)-1; i++ {
			if fields[i] == "via" {
				gw = fields[i+1]
			}
			if fields[i] == "dev" {
				dev = fields[i+1]
			}
		}
		if dev == "" && gw == "" {
			return "", "", fmt.Errorf("未找到默认路由信息")
		}
		return dev, gw, nil
	default:
		return "", "", fmt.Errorf("不支持的系统: %s", runtime.GOOS)
	}
}

func (nm *networkManager) getDefaultGateway() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		out, err := nm.runCmd(3*time.Second, "route", "-n", "get", "default")
		if err != nil {
			return "", err
		}
		// gateway: 192.168.1.1
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "gateway:") {
				return strings.TrimSpace(strings.TrimPrefix(line, "gateway:")), nil
			}
		}
		return "", fmt.Errorf("未找到默认网关")
	case "linux":
		out, err := nm.runCmd(3*time.Second, "ip", "route", "show", "default")
		if err != nil {
			return "", err
		}
		// default via 192.168.1.1 dev eth0 ...
		fields := strings.Fields(out)
		for i := 0; i < len(fields)-1; i++ {
			if fields[i] == "via" {
				return fields[i+1], nil
			}
		}
		return "", fmt.Errorf("未找到默认网关")
	default:
		return "", fmt.Errorf("不支持的系统: %s", runtime.GOOS)
	}
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
