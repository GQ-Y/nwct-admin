//go:build pcap

package scanner

import (
	"fmt"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// ARPScan 执行ARP扫描（pcap 抓包加速版）
// 使用方法：
//   go build -tags pcap
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
		// 如果没有权限/环境不支持（例如容器），回退到纯 Go 兜底
		return arpScanSimple(ipnet, timeout)
	}
	defer handle.Close()

	devices := make(map[string]string) // IP -> MAC
	done := make(chan bool, 1)

	// 启动抓包 goroutine
	go func() {
		packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
		for packet := range packetSource.Packets() {
			arpLayer := packet.Layer(layers.LayerTypeARP)
			if arpLayer == nil {
				continue
			}
			arp := arpLayer.(*layers.ARP)
			if arp.Operation != layers.ARPReply {
				continue
			}
			srcIP := net.IP(arp.SourceProtAddress).String()
			srcMAC := net.HardwareAddr(arp.SourceHwAddress).String()
			if ipnet.Contains(net.ParseIP(srcIP)) {
				devices[srcIP] = srcMAC
			}
		}
		done <- true
	}()

	// 发送 ARP 请求
	srcIP := getInterfaceIP(iface)
	if srcIP == nil {
		return nil, fmt.Errorf("无法获取接口IP地址")
	}
	srcMAC, err := net.ParseMAC(iface.HardwareAddr.String())
	if err != nil {
		return nil, fmt.Errorf("无效的MAC地址: %v", err)
	}

	// 遍历网段内所有 IP
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		// 跳过网络地址和广播地址
		if isNetworkOrBroadcast(ip, ipnet) {
			continue
		}
		sendARPRequest(handle, srcIP, srcMAC, ip)
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
		result = append(result, ARPDevice{IP: ip, MAC: mac})
	}
	return result, nil
}

// sendARPRequest 发送ARP请求（pcap版）
func sendARPRequest(handle *pcap.Handle, srcIP net.IP, srcMAC net.HardwareAddr, dstIP net.IP) {
	if handle == nil {
		return
	}

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
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	if err := gopacket.SerializeLayers(buf, opts, eth, arp); err != nil {
		return
	}
	_ = handle.WritePacketData(buf.Bytes())
}


