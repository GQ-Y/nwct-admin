package scanner

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// ARPDevice ARP扫描发现的设备
type ARPDevice struct {
	IP  string
	MAC string
}

// ARPScan 执行ARP扫描
func ARPScan(subnet string, timeout time.Duration) ([]ARPDevice, error) {
	// 解析网段
	ip, ipnet, err := net.ParseCIDR(subnet)
	if err != nil {
		return nil, fmt.Errorf("无效的网段: %v", err)
	}

	// 获取本地网络接口
	iface, err := getInterfaceForSubnet(ipnet)
	if err != nil {
		return nil, fmt.Errorf("获取网络接口失败: %v", err)
	}

	// 打开网络接口进行抓包
	handle, err := pcap.OpenLive(iface.Name, 1024, true, timeout)
	if err != nil {
		// 如果没有权限，使用简化方法
		return arpScanSimple(ipnet, timeout)
	}
	defer handle.Close()

	devices := make(map[string]string) // IP -> MAC
	done := make(chan bool)

	// 启动抓包goroutine
	go func() {
		packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
		for packet := range packetSource.Packets() {
			arpLayer := packet.Layer(layers.LayerTypeARP)
			if arpLayer != nil {
				arp := arpLayer.(*layers.ARP)
				if arp.Operation == layers.ARPReply {
					srcIP := net.IP(arp.SourceProtAddress).String()
					srcMAC := net.HardwareAddr(arp.SourceHwAddress).String()
					if ipnet.Contains(net.ParseIP(srcIP)) {
						devices[srcIP] = srcMAC
					}
				}
			}
		}
		done <- true
	}()

	// 发送ARP请求
	srcIP := getInterfaceIP(iface)
	if srcIP == nil {
		return nil, fmt.Errorf("无法获取接口IP地址")
	}

	srcMAC, err := net.ParseMAC(iface.HardwareAddr.String())
	if err != nil {
		return nil, fmt.Errorf("无效的MAC地址: %v", err)
	}

	// 遍历网段内所有IP
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		// 跳过网络地址和广播地址
		if isNetworkOrBroadcast(ip, ipnet) {
			continue
		}

		// 发送ARP请求
		sendARPRequest(handle, srcIP, srcMAC, ip, iface)
		time.Sleep(10 * time.Millisecond) // 避免发送过快
	}

	// 等待响应
	select {
	case <-done:
	case <-time.After(timeout):
	}

	// 转换为结果
	result := make([]ARPDevice, 0, len(devices))
	for ip, mac := range devices {
		result = append(result, ARPDevice{
			IP:  ip,
			MAC: mac,
		})
	}

	return result, nil
}

// arpScanSimple 简化的ARP扫描（当没有抓包权限时）
func arpScanSimple(ipnet *net.IPNet, timeout time.Duration) ([]ARPDevice, error) {
	// 无 pcap 权限时的兜底：
	// 1) 并发 ICMP ping sweep 填充 ARP 表
	// 2) 读取系统 ARP 表，回收子网内的 IP/MAC

	// 枚举子网内 IP（/24 通常 254 个）
	ips := make([]string, 0, 256) // 从 512 降到 256，减少预分配内存
	for ip := ipnet.IP.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		if isNetworkOrBroadcast(ip, ipnet) {
			continue
		}
		ips = append(ips, ip.String())
	}

	// 并发 ping sweep（不要求全部成功，只为尽可能填充 ARP 表）
	workers := 64
	if len(ips) < workers {
		workers = len(ips)
	}
	if workers <= 0 {
		return []ARPDevice{}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ch := make(chan string)
	for i := 0; i < workers; i++ {
		go func() {
			for ip := range ch {
				_ = pingOnce(ctx, ip)
			}
		}()
	}
	for _, ip := range ips {
		select {
		case <-ctx.Done():
			close(ch)
			goto done
		case ch <- ip:
		}
	}
	close(ch)

done:
	// 读取 ARP 表（批量），返回子网内的 IP/MAC
	entries, _ := getARPTableEntries(ipnet)
	devices := make([]ARPDevice, 0, len(entries))
	for ip, mac := range entries {
		devices = append(devices, ARPDevice{IP: ip, MAC: mac})
	}
	return devices, nil
}

// getInterfaceForSubnet 获取指定网段的网络接口
func getInterfaceForSubnet(ipnet *net.IPNet) (*net.Interface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if addrNet, ok := addr.(*net.IPNet); ok {
				if ipnet.Contains(addrNet.IP) {
					return &iface, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("未找到匹配的网络接口")
}

// getInterfaceIP 获取接口的IP地址
func getInterfaceIP(iface *net.Interface) net.IP {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if ip := ipnet.IP.To4(); ip != nil {
				return ip
			}
		}
	}

	return nil
}

// sendARPRequest 发送ARP请求
func sendARPRequest(handle *pcap.Handle, srcIP net.IP, srcMAC net.HardwareAddr, dstIP net.IP, iface *net.Interface) {
	if handle == nil {
		return
	}

	// 创建ARP请求包
	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, // 广播
		EthernetType: layers.EthernetTypeARP,
	}

	arp := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPRequest,
		SourceHwAddress:   []byte(srcMAC),
		SourceProtAddress: []byte(srcIP.To4()),
		DstHwAddress:      []byte{0, 0, 0, 0, 0, 0},
		DstProtAddress:    []byte(dstIP.To4()),
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	if err := gopacket.SerializeLayers(buf, opts, eth, arp); err != nil {
		return
	}

	handle.WritePacketData(buf.Bytes())
}

// inc 增加IP地址
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// isNetworkOrBroadcast 判断是否为网络地址或广播地址
func isNetworkOrBroadcast(ip net.IP, ipnet *net.IPNet) bool {
	return ip.Equal(ipnet.IP) || ip.Equal(broadcastAddr(ipnet))
}

// broadcastAddr 计算广播地址
func broadcastAddr(ipnet *net.IPNet) net.IP {
	ip := make(net.IP, len(ipnet.IP))
	copy(ip, ipnet.IP)
	mask := ipnet.Mask

	for i := range ip {
		ip[i] |= ^mask[i]
	}
	return ip
}

// isHostAlive 检查主机是否存活
func isHostAlive(ip string) bool {
	// 仅用于扫描兜底：用系统 ping 做 ICMP 探测（更通用）
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	return pingOnce(ctx, ip) == nil
}

// getMACFromARPTable 从ARP表获取MAC地址（简化实现）
func getMACFromARPTable(ip string) string {
	entries, err := getARPTableEntries(nil)
	if err != nil {
		return ""
	}
	return entries[ip]
}

func pingOnce(ctx context.Context, ip string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		// -W: waittime (ms) on macOS
		cmd = exec.CommandContext(ctx, "ping", "-c", "1", "-W", "1000", ip)
	default:
		// Linux: -W timeout (sec)
		cmd = exec.CommandContext(ctx, "ping", "-c", "1", "-W", "1", ip)
	}
	err := cmd.Run()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

func getARPTableEntries(ipnet *net.IPNet) (map[string]string, error) {
	switch runtime.GOOS {
	case "linux":
		return readLinuxARPTable(ipnet)
	case "darwin":
		return readDarwinARPTable(ipnet)
	default:
		// best-effort
		return map[string]string{}, nil
	}
}

func readLinuxARPTable(ipnet *net.IPNet) (map[string]string, error) {
	f, err := os.Open("/proc/net/arp")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	out := map[string]string{}
	first := true
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if first {
			// skip header
			first = false
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		ip := fields[0]
		mac := normalizeMAC(fields[3])
		if mac == "" || mac == "00:00:00:00:00:00" || mac == "FF:FF:FF:FF:FF:FF" {
			continue
		}
		if ipnet != nil {
			nip := net.ParseIP(ip)
			if nip == nil || !ipnet.Contains(nip) || isNetworkOrBroadcast(nip, ipnet) {
				continue
			}
		}
		out[ip] = mac
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func readDarwinARPTable(ipnet *net.IPNet) (map[string]string, error) {
	// arp -a 输出类似：
	// ? (192.168.2.1) at 18:aa:0f:f7:9e:62 on en0 ifscope [ethernet]
	b, err := exec.Command("arp", "-a").Output()
	if err != nil {
		return nil, err
	}
	reIP := regexp.MustCompile(`\((\d+\.\d+\.\d+\.\d+)\)`)
	reMAC := regexp.MustCompile(`(?i)\bat\s+(([0-9a-f]{1,2}:){5}[0-9a-f]{1,2})\b`)

	out := map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	for sc.Scan() {
		line := sc.Text()
		mip := reIP.FindStringSubmatch(line)
		if len(mip) != 2 {
			continue
		}
		ip := mip[1]
		if ipnet != nil {
			nip := net.ParseIP(ip)
			if nip == nil || !ipnet.Contains(nip) || isNetworkOrBroadcast(nip, ipnet) {
				continue
			}
		}
		mmac := reMAC.FindStringSubmatch(line)
		if len(mmac) != 3 {
			continue
		}
		mac := normalizeMAC(mmac[1])
		if mac == "" || mac == "00:00:00:00:00:00" || mac == "FF:FF:FF:FF:FF:FF" {
			continue
		}
		out[ip] = mac
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func normalizeMAC(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, "-", ":")
	parts := strings.Split(s, ":")
	if len(parts) != 6 {
		return ""
	}
	for i := range parts {
		p := strings.TrimSpace(parts[i])
		p = strings.TrimPrefix(p, "0x")
		if p == "" {
			return ""
		}
		if len(p) > 2 {
			p = p[len(p)-2:]
		}
		if len(p) == 1 {
			p = "0" + p
		}
		if !reHex2.MatchString(p) {
			// reuse a local compiled regex to avoid importing fingerprint package
			return ""
		}
		parts[i] = strings.ToUpper(p)
	}
	return strings.Join(parts, ":")
}

var reHex2 = regexp.MustCompile(`^[0-9a-f]{2}$`)
