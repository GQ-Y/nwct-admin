package scanner

import (
	"fmt"
	"net"
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
	// 使用ping方式发现设备（简化实现）
	devices := []ARPDevice{}

	ip := ipnet.IP
	for ipnet.Contains(ip) {
		if isNetworkOrBroadcast(ip, ipnet) {
			inc(ip)
			continue
		}

		// 尝试连接常见端口来判断设备是否在线
		if isHostAlive(ip.String()) {
			// 尝试获取MAC地址（通过ARP表）
			mac := getMACFromARPTable(ip.String())
			devices = append(devices, ARPDevice{
				IP:  ip.String(),
				MAC: mac,
			})
		}

		inc(ip)
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
	conn, err := net.DialTimeout("tcp", ip+":80", 1*time.Second)
	if err == nil {
		conn.Close()
		return true
	}

	conn, err = net.DialTimeout("tcp", ip+":22", 1*time.Second)
	if err == nil {
		conn.Close()
		return true
	}

	return false
}

// getMACFromARPTable 从ARP表获取MAC地址（简化实现）
func getMACFromARPTable(ip string) string {
	// TODO: 读取系统ARP表
	// Linux: /proc/net/arp
	// macOS: arp -a
	return ""
}

