package scanner

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"nwct/client-nps/internal/database"
	"nwct/client-nps/internal/fingerprint"
	"nwct/client-nps/internal/logger"
	"nwct/client-nps/internal/realtime"
	"nwct/client-nps/internal/toolkit"
	"strings"
	"sync"
	"time"
)

// Scanner 设备扫描器接口
type Scanner interface {
	StartScan(subnet string) error
	StopScan() error
	GetDevices() ([]Device, error)
	GetDeviceDetail(ip string) (*DeviceDetail, error)
	GetScanStatus() *ScanStatus
}

// Device 设备信息
type Device struct {
	IP        string `json:"ip"`
	MAC       string `json:"mac"`
	Name      string `json:"name"`
	Vendor    string `json:"vendor"`
	Model     string `json:"model"`
	Type      string `json:"type"`
	OS        string `json:"os"`
	Extra     string `json:"extra"`
	Status    string `json:"status"`
	OpenPorts []int  `json:"open_ports"`
	LastSeen  string `json:"last_seen"`
	FirstSeen string `json:"first_seen"`
}

// DeviceDetail 设备详情
type DeviceDetail struct {
	Device
	Ports   []PortInfo `json:"ports"`
	History []History  `json:"history"`
}

// PortInfo 端口信息
type PortInfo struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Service  string `json:"service"`
	Version  string `json:"version"`
	Status   string `json:"status"`
}

// History 历史记录
type History struct {
	Timestamp string `json:"timestamp"`
	Status    string `json:"status"`
}

// ScanStatus 扫描状态
type ScanStatus struct {
	Status       string    `json:"status"` // running, stopped, completed
	Progress     int       `json:"progress"`
	ScannedCount int       `json:"scanned_count"`
	FoundCount   int       `json:"found_count"`
	StartTime    time.Time `json:"start_time"`
}

// deviceScanner 设备扫描器实现
type deviceScanner struct {
	scanning   bool
	scanStatus *ScanStatus
	mu         sync.RWMutex
	db         *sql.DB
}

var (
	globalScanner *deviceScanner
	scannerOnce   sync.Once
)

// NewScanner 创建扫描器
func NewScanner(db *sql.DB) Scanner {
	scannerOnce.Do(func() {
		globalScanner = &deviceScanner{
			scanning: false,
			scanStatus: &ScanStatus{
				Status: "stopped",
			},
			db: db,
		}
	})
	return globalScanner
}

// StartScan 启动扫描
func (ds *deviceScanner) StartScan(subnet string) error {
	ds.mu.Lock()
	if ds.scanning {
		ds.mu.Unlock()
		return fmt.Errorf("扫描已在进行中")
	}
	ds.scanning = true
	ds.scanStatus.Status = "running"
	ds.scanStatus.StartTime = time.Now()
	ds.scanStatus.Progress = 0
	ds.scanStatus.ScannedCount = 0
	ds.scanStatus.FoundCount = 0
	ds.mu.Unlock()

	realtime.Default().Broadcast("scan_started", map[string]interface{}{
		"subnet": subnet,
	})

	// 在goroutine中执行扫描
	go ds.performScan(subnet)

	return nil
}

// performScan 执行扫描
func (ds *deviceScanner) performScan(subnet string) {
	defer func() {
		ds.mu.Lock()
		ds.scanning = false
		ds.scanStatus.Status = "completed"
		ds.scanStatus.Progress = 100
		found := ds.scanStatus.FoundCount
		scanned := ds.scanStatus.ScannedCount
		ds.mu.Unlock()

		realtime.Default().Broadcast("scan_done", map[string]interface{}{
			"subnet":        subnet,
			"status":        "completed",
			"progress":      100,
			"scanned_count": scanned,
			"found_count":   found,
		})
	}()

	logger.Info("开始扫描网段: %s", subnet)

	// 0. SSDP/UPnP 发现（用于补充 friendlyName/model/manufacturer）
	// 这一步不会阻塞扫描太久：超时短，失败不影响主流程
	ssdpMap := map[string]*fingerprint.SSDPDevice{}
	{
		// 发现阶段 2s，但留足时间抓取描述 XML 做信息补全
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()
		if m, err := fingerprint.SSDPDiscover(ctx, 2*time.Second); err == nil && m != nil {
			ssdpMap = m
		}
	}
	logger.Info("SSDP发现设备数: %d", len(ssdpMap))
	if len(ssdpMap) > 0 {
		n := 0
		for ip, d := range ssdpMap {
			// 仅打印少量，避免刷屏
			logger.Info("SSDP设备: ip=%s location=%s server=%s usn=%s st=%s friendly=%s manufacturer=%s model=%s deviceType=%s",
				ip, d.Location, d.Server, d.USN, d.ST, d.FriendlyName, d.Manufacturer, d.ModelName, d.DeviceType)
			n++
			if n >= 5 {
				break
			}
		}
	}

	// 0.5 WS-Discovery（ONVIF/Windows 等），用于发现摄像头并拿到 xaddrs
	wsdMap := map[string]*fingerprint.WSDiscoveryDevice{}
	{
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if m, err := fingerprint.WSDiscoveryProbe(ctx, 2*time.Second); err == nil && m != nil {
			wsdMap = m
		}
	}
	logger.Info("WS-Discovery发现设备数: %d", len(wsdMap))

	// 1. ARP扫描
	arpDevices, err := ARPScan(subnet, 30*time.Second)
	if err != nil {
		logger.Error("ARP扫描失败: %v", err)
		// 继续使用简化方法
	}
	total := len(arpDevices)

	// 2. 处理发现的设备
	lastPush := time.Now()
	for _, arpDevice := range arpDevices {
		ds.mu.Lock()
		ds.scanStatus.FoundCount++
		ds.scanStatus.ScannedCount++
		if total > 0 {
			ds.scanStatus.Progress = int(float64(ds.scanStatus.ScannedCount) / float64(total) * 100.0)
			if ds.scanStatus.Progress > 99 {
				ds.scanStatus.Progress = 99
			}
		}
		st := *ds.scanStatus
		ds.mu.Unlock()

		// 节流推送（最多 2s 一次）
		if time.Since(lastPush) >= 2*time.Second {
			realtime.Default().Broadcast("scan_progress", map[string]interface{}{
				"subnet":        subnet,
				"status":        st.Status,
				"progress":      st.Progress,
				"scanned_count": st.ScannedCount,
				"found_count":   st.FoundCount,
				"start_time":    st.StartTime.Format(time.RFC3339),
			})
			lastPush = time.Now()
		}

		// 识别设备
		device := ds.identifyDevice(arpDevice.IP, arpDevice.MAC)
		evidence := map[string]any{}

		// SSDP/UPnP 补充信息（名称/厂商/类型）
		if adv := ssdpMap[arpDevice.IP]; adv != nil {
			evidence["ssdp"] = map[string]any{
				"location":     adv.Location,
				"server":       adv.Server,
				"usn":          adv.USN,
				"st":           adv.ST,
				"friendlyName": adv.FriendlyName,
				"manufacturer": adv.Manufacturer,
				"modelName":    adv.ModelName,
				"deviceType":   adv.DeviceType,
			}
			if device.Name == "" {
				if adv.FriendlyName != "" {
					device.Name = adv.FriendlyName
				} else if adv.ModelName != "" {
					device.Name = adv.ModelName
				}
			}
			if device.Vendor == "" || strings.EqualFold(device.Vendor, "unknown") {
				if adv.Manufacturer != "" {
					device.Vendor = adv.Manufacturer
				}
			}
			// 仅在当前类型不够明确时用 SSDP 的 deviceType 做辅助判断
			if device.Type == "" || device.Type == "unknown" || device.Type == "network_device" {
				if t := mapDeviceTypeFromUPnP(adv.DeviceType); t != "" {
					device.Type = t
				}
			}
		}

		// WS-Discovery / ONVIF 补充信息（摄像头型号/厂商）
		if w := wsdMap[arpDevice.IP]; w != nil {
			evidence["wsd"] = map[string]any{
				"types":  w.Types,
				"xaddrs": w.XAddrs,
				"scopes": w.Scopes,
			}
			// 从 scopes 简单提取品牌/型号线索（不同厂商 scope 格式差异很大）
			// 主要依赖 ONVIF GetDeviceInformation
			for _, x := range w.XAddrs {
				// 常见 onvif 设备服务路径包含 /onvif/device_service
				if strings.Contains(strings.ToLower(x), "onvif") {
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					info, err := fingerprint.ONVIFGetDeviceInformation(ctx, x)
					cancel()
					if err == nil && info != nil {
						evidence["onvif"] = info
						if device.Vendor == "" || strings.EqualFold(device.Vendor, "unknown") {
							if info.Manufacturer != "" {
								device.Vendor = info.Manufacturer
							}
						}
						if device.Name == "" && info.Model != "" {
							device.Name = info.Model
						}
						if device.OS == "" || strings.EqualFold(device.OS, "unknown") {
							// ONVIF 设备通常归类为 camera
							device.OS = "Embedded"
						}
						if device.Type == "" || device.Type == "unknown" || device.Type == "network_device" {
							device.Type = "camera"
						}
						// 把型号写入 Vendor/Name 以外字段（数据库新增 model）
						// 后续入库时会写入
						if info.Model != "" {
							device.Model = info.Model
						}
						break
					} else if err != nil {
						// 记录最后一次错误，便于排查（常见 401 需要认证）
						evidence["onvif_error"] = err.Error()
					}
				}
			}
		}

		// HTTP 指纹补充（路由器/NAS/摄像头 Web 管理页）
		// 仅当 80/443 端口开放时探测
		if len(device.OpenPorts) > 0 {
			portSet := map[int]bool{}
			for _, p := range device.OpenPorts {
				portSet[p] = true
			}
			if portSet[80] || portSet[443] {
				ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
				defer cancel()
				if portSet[80] {
					if fp, err := fingerprint.ProbeHTTPFingerprint(ctx, arpDevice.IP+":80", false); err == nil && fp != nil {
						evidence["http_80"] = fp
						if device.Model == "" && fp.Title != "" {
							device.Model = fp.Title
						}
						if device.Vendor == "" || strings.EqualFold(device.Vendor, "unknown") {
							if fp.Server != "" {
								device.Vendor = fp.Server
							}
						}
					}
				}
				if device.Model == "" && portSet[443] {
					if fp, err := fingerprint.ProbeHTTPFingerprint(ctx, arpDevice.IP+":443", true); err == nil && fp != nil {
						evidence["https_443"] = fp
						if device.Model == "" && fp.Title != "" {
							device.Model = fp.Title
						}
					}
				}
			}
		}

		extraJSON := ""
		if len(evidence) > 0 {
			if b, err := json.Marshal(evidence); err == nil {
				extraJSON = string(b)
			}
		}

		// 保存到数据库
		dbDevice := &database.Device{
			IP:        device.IP,
			MAC:       device.MAC,
			Name:      device.Name,
			Vendor:    device.Vendor,
			Model:     device.Model,
			Type:      device.Type,
			OS:        device.OS,
			Extra:     extraJSON,
			Status:    "online",
			FirstSeen: time.Now(),
			LastSeen:  time.Now(),
		}

		if err := database.SaveDevice(ds.db, dbDevice); err != nil {
			logger.Error("保存设备失败: %v", err)
		} else {
			// 设备列表变化推送（upsert）
			realtime.Default().Broadcast("device_upsert", map[string]interface{}{
				"ip":        dbDevice.IP,
				"mac":       dbDevice.MAC,
				"name":      dbDevice.Name,
				"vendor":    dbDevice.Vendor,
				"type":      dbDevice.Type,
				"os":        dbDevice.OS,
				"status":    dbDevice.Status,
				"last_seen": dbDevice.LastSeen.Format(time.RFC3339),
			})
		}

		// 端口扫描（异步，避免阻塞）
		if len(device.OpenPorts) == 0 {
			go ds.scanPorts(device.IP)
		}
	}

	logger.Info("扫描完成，发现 %d 个设备", len(arpDevices))
}

func mapDeviceTypeFromUPnP(deviceType string) string {
	s := strings.ToLower(strings.TrimSpace(deviceType))
	if s == "" {
		return ""
	}
	// 常见网关/路由器
	if strings.Contains(s, "internetgatewaydevice") || strings.Contains(s, "wanconnectiondevice") || strings.Contains(s, "wandevice") {
		return "router"
	}
	// 媒体设备
	if strings.Contains(s, "mediaserver") || strings.Contains(s, "mediareceiver") || strings.Contains(s, "renderer") {
		return "media_device"
	}
	// 打印机
	if strings.Contains(s, "printer") {
		return "printer"
	}
	// 摄像头（不少厂商会用自定义 deviceType，这里先做一个保守匹配）
	if strings.Contains(s, "camera") || strings.Contains(s, "ipcamera") {
		return "camera"
	}
	return ""
}

// identifyDevice 识别设备
func (ds *deviceScanner) identifyDevice(ip, mac string) *Device {
	device := &Device{
		IP:     ip,
		MAC:    mac,
		Status: "online",
	}

	// MAC地址OUI识别
	device.Vendor = identifyVendor(mac)

	// 端口扫描识别设备类型
	ports := ds.scanCommonPorts(ip)
	device.OpenPorts = ports

	// 根据端口识别设备类型
	device.Type = identifyDeviceType(ports)
	device.OS = identifyOS(ports)

	// 尝试获取设备名称
	device.Name = getDeviceName(ip)

	return device
}

// scanPorts 扫描设备端口
func (ds *deviceScanner) scanPorts(ip string) {
	// 覆盖常见路由器/NAS/摄像头/NVR Web 端口
	commonPorts := []int{22, 23, 53, 80, 81, 443, 445, 554, 8000, 8008, 8080, 8081, 8088, 8443, 8888, 8899, 5000, 5001, 3306, 5432, 9100}
	updated := 0

	for _, port := range commonPorts {
		result, err := toolkit.PortScan(ip, []int{port}, 2*time.Second, "tcp")
		if err != nil {
			continue
		}

		for _, portInfo := range result.OpenPorts {
			dbPort := &database.DevicePort{
				DeviceIP: ip,
				Port:     portInfo.Port,
				Protocol: portInfo.Protocol,
				Service:  portInfo.Service,
				Version:  portInfo.Version,
				Status:   portInfo.Status,
			}
			if err := database.SaveDevicePort(ds.db, ip, dbPort); err == nil {
				updated++
			}
		}
	}

	if updated > 0 {
		realtime.Default().Broadcast("device_ports_updated", map[string]interface{}{
			"ip":      ip,
			"updated": updated,
		})
	}
}

// scanCommonPorts 扫描常用端口
func (ds *deviceScanner) scanCommonPorts(ip string) []int {
	commonPorts := []int{22, 23, 53, 80, 81, 443, 445, 554, 8000, 8080, 8081, 8443, 8888}
	openPorts := []int{}

	for _, port := range commonPorts {
		result, err := toolkit.PortScan(ip, []int{port}, 1*time.Second, "tcp")
		if err == nil && len(result.OpenPorts) > 0 {
			openPorts = append(openPorts, port)
		}
	}

	return openPorts
}

// StopScan 停止扫描
func (ds *deviceScanner) StopScan() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if !ds.scanning {
		return fmt.Errorf("扫描未在进行中")
	}

	ds.scanning = false
	ds.scanStatus.Status = "stopped"
	return nil
}

// GetDevices 获取设备列表
func (ds *deviceScanner) GetDevices() ([]Device, error) {
	dbDevices, _, err := database.GetDevices(ds.db, "all", "", 1000, 0)
	if err != nil {
		return nil, err
	}

	devices := make([]Device, len(dbDevices))
	for i, d := range dbDevices {
		devices[i] = Device{
			IP:        d.IP,
			MAC:       d.MAC,
			Name:      d.Name,
			Vendor:    d.Vendor,
			Type:      d.Type,
			OS:        d.OS,
			Status:    d.Status,
			LastSeen:  d.LastSeen.Format(time.RFC3339),
			FirstSeen: d.FirstSeen.Format(time.RFC3339),
		}

		// 获取开放端口
		ports, _ := database.GetDevicePorts(ds.db, d.IP)
		openPorts := make([]int, len(ports))
		for j, p := range ports {
			openPorts[j] = p.Port
		}
		devices[i].OpenPorts = openPorts
	}

	return devices, nil
}

// GetDeviceDetail 获取设备详情
func (ds *deviceScanner) GetDeviceDetail(ip string) (*DeviceDetail, error) {
	dbDevice, err := database.GetDevice(ds.db, ip)
	if err != nil {
		return nil, err
	}

	if dbDevice == nil {
		return nil, fmt.Errorf("设备不存在")
	}

	ports, _ := database.GetDevicePorts(ds.db, ip)
	portInfos := make([]PortInfo, len(ports))
	for i, p := range ports {
		portInfos[i] = PortInfo{
			Port:     p.Port,
			Protocol: p.Protocol,
			Service:  p.Service,
			Version:  p.Version,
			Status:   p.Status,
		}
	}

	return &DeviceDetail{
		Device: Device{
			IP:        dbDevice.IP,
			MAC:       dbDevice.MAC,
			Name:      dbDevice.Name,
			Vendor:    dbDevice.Vendor,
			Model:     dbDevice.Model,
			Extra:     dbDevice.Extra,
			Type:      dbDevice.Type,
			OS:        dbDevice.OS,
			Status:    dbDevice.Status,
			LastSeen:  dbDevice.LastSeen.Format(time.RFC3339),
			FirstSeen: dbDevice.FirstSeen.Format(time.RFC3339),
		},
		Ports: portInfos,
	}, nil
}

// GetScanStatus 获取扫描状态
func (ds *deviceScanner) GetScanStatus() *ScanStatus {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	status := *ds.scanStatus
	return &status
}

// identifyVendor 识别厂商（基于MAC地址OUI）
func identifyVendor(mac string) string {
	// 优先使用离线 OUI 库（IEEE oui.txt，支持本地缓存/可选自动下载）
	if v := fingerprint.DefaultOUI().Lookup(mac); v != "" {
		return v
	}

	// 保底：常见虚拟网卡 OUI
	ouiMap := map[string]string{
		"00:50:56": "VMware",
		"00:0C:29": "VMware",
		"08:00:27": "VirtualBox",
		"52:54:00": "QEMU",
	}
	if len(mac) >= 8 {
		oui := strings.ToUpper(mac[:8])
		if vendor, ok := ouiMap[oui]; ok {
			return vendor
		}
	}
	return "Unknown"
}

// identifyDeviceType 根据端口识别设备类型
func identifyDeviceType(ports []int) string {
	portSet := make(map[int]bool)
	for _, p := range ports {
		portSet[p] = true
	}

	// 路由器/网络设备
	if portSet[80] || portSet[443] {
		if portSet[22] {
			return "router"
		}
		return "network_device"
	}

	// 服务器
	if portSet[22] && (portSet[80] || portSet[443] || portSet[3306] || portSet[5432]) {
		return "server"
	}

	// 计算机
	if portSet[3389] || portSet[22] {
		return "computer"
	}

	// IoT设备
	if len(ports) > 0 && len(ports) < 3 {
		return "iot_device"
	}

	return "unknown"
}

// identifyOS 识别操作系统（基于端口）
func identifyOS(ports []int) string {
	portSet := make(map[int]bool)
	for _, p := range ports {
		portSet[p] = true
	}

	if portSet[3389] {
		return "Windows"
	}
	if portSet[22] {
		return "Linux/Unix"
	}

	return "Unknown"
}

// getDeviceName 获取设备名称
func getDeviceName(ip string) string {
	// 尝试反向DNS查询
	names, err := net.LookupAddr(ip)
	if err == nil && len(names) > 0 {
		return strings.TrimSuffix(names[0], ".")
	}

	// NetBIOS (NBNS) 节点状态查询（UDP/137），对 Windows/部分NAS 很有效
	if name, err := nbnsNodeStatusName(ip, 800*time.Millisecond); err == nil && name != "" {
		return name
	}

	// mDNS 反向解析（PTR in-addr.arpa），对 Apple/iOS/IoT 的 `.local` 名称很有效
	if name, err := fingerprint.MDNSReverseLookup(ip, 600*time.Millisecond); err == nil && name != "" {
		return name
	}

	return ""
}

// nbnsNodeStatusName 通过 NBNS Node Status (0x21) 获取设备 NetBIOS 名称
func nbnsNodeStatusName(ip string, timeout time.Duration) (string, error) {
	// 构造 NBNS 请求
	// Header: TransactionID(2) Flags(2) QDCount(2)=1 ANCount(2)=0 NSCount(2)=0 ARCount(2)=0
	// Question: QNAME(ENCODED "*") QTYPE=0x0021 QCLASS=0x0001
	txid := uint16(time.Now().UnixNano() & 0xffff)
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.BigEndian, txid)
	_ = binary.Write(&buf, binary.BigEndian, uint16(0x0000))
	_ = binary.Write(&buf, binary.BigEndian, uint16(0x0001))
	_ = binary.Write(&buf, binary.BigEndian, uint16(0x0000))
	_ = binary.Write(&buf, binary.BigEndian, uint16(0x0000))
	_ = binary.Write(&buf, binary.BigEndian, uint16(0x0000))

	// QNAME: 0x20 + 32 bytes netbios-encoded name + 0x00
	// 对于 Node Status，name 为 "*" (0x2A) + 15 spaces
	encoded := encodeNetBIOSName("*")
	buf.WriteByte(0x20)
	buf.Write(encoded)
	buf.WriteByte(0x00)

	_ = binary.Write(&buf, binary.BigEndian, uint16(0x0021)) // NBSTAT
	_ = binary.Write(&buf, binary.BigEndian, uint16(0x0001)) // IN

	raddr, err := net.ResolveUDPAddr("udp4", net.JoinHostPort(ip, "137"))
	if err != nil {
		return "", err
	}
	conn, err := net.DialUDP("udp4", nil, raddr)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	if _, err := conn.Write(buf.Bytes()); err != nil {
		return "", err
	}

	resp := make([]byte, 1500)
	n, err := conn.Read(resp)
	if err != nil {
		return "", err
	}
	resp = resp[:n]
	if len(resp) < 12 {
		return "", fmt.Errorf("nbns响应过短")
	}

	// 简单校验 txid
	if binary.BigEndian.Uint16(resp[0:2]) != txid {
		// 不严格：有些设备可能不回同 txid，这里不直接失败
	}

	// 跳过 header
	off := 12
	// 跳过 question qname（0x20 + 32 + 0x00）+ qtype/qclass
	if off+1 > len(resp) {
		return "", fmt.Errorf("nbns响应异常")
	}
	// 解析 qname：压缩指针(0xC0) 或 label(0x20)
	if resp[off]&0xC0 == 0xC0 {
		off += 2
	} else {
		// label length
		l := int(resp[off])
		off++
		off += l
		off++ // 0x00
	}
	off += 4 // qtype+qclass
	if off+2 > len(resp) {
		return "", fmt.Errorf("nbns响应异常")
	}

	// Answer section：我们只解析第一个 RR
	// NAME: pointer
	if off+2 > len(resp) {
		return "", fmt.Errorf("nbns响应异常")
	}
	if resp[off]&0xC0 == 0xC0 {
		off += 2
	} else {
		// 极少数情况：非压缩 name，这里粗略跳过
		l := int(resp[off])
		off++
		off += l
		off++
	}
	if off+10 > len(resp) {
		return "", fmt.Errorf("nbns RR 过短")
	}
	rrType := binary.BigEndian.Uint16(resp[off : off+2])
	off += 2
	_ = rrType
	off += 2 // class
	off += 4 // ttl
	rdlen := int(binary.BigEndian.Uint16(resp[off : off+2]))
	off += 2
	if off+rdlen > len(resp) {
		return "", fmt.Errorf("nbns RDATA 过短")
	}
	rdata := resp[off : off+rdlen]

	// Node Status RDATA: NumNames(1) + NameEntry(18)*N + ...
	if len(rdata) < 1 {
		return "", fmt.Errorf("nbns RDATA 过短")
	}
	num := int(rdata[0])
	pos := 1
	best := ""
	for i := 0; i < num; i++ {
		if pos+18 > len(rdata) {
			break
		}
		nameBytes := rdata[pos : pos+15]
		suffix := rdata[pos+15]
		flags := binary.BigEndian.Uint16(rdata[pos+16 : pos+18])
		pos += 18

		name := strings.TrimSpace(string(nameBytes))
		// 选取常见的工作站服务名：suffix 0x00 且非 group
		isGroup := (flags & 0x8000) != 0
		if suffix == 0x00 && !isGroup && name != "" {
			best = name
			break
		}
		// 备用：记录第一个非空
		if best == "" && name != "" {
			best = name
		}
	}
	return best, nil
}

// encodeNetBIOSName 将一个名字编码成 NetBIOS 32 字节表示（RFC1002）
func encodeNetBIOSName(name string) []byte {
	// 取 16 字节：name(<=15) + suffix(1)；这里 suffix 0x00，name 用空格填充
	raw := make([]byte, 16)
	for i := range raw {
		raw[i] = ' '
	}
	n := []byte(name)
	if len(n) > 15 {
		n = n[:15]
	}
	copy(raw, n)
	raw[15] = 0x00

	out := make([]byte, 32)
	for i := 0; i < 16; i++ {
		b := raw[i]
		out[i*2] = 'A' + ((b >> 4) & 0x0F)
		out[i*2+1] = 'A' + (b & 0x0F)
	}
	return out
}
