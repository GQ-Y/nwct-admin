package scanner

import (
	"database/sql"
	"fmt"
	"net"
	"nwct/client-nps/internal/database"
	"nwct/client-nps/internal/logger"
	"nwct/client-nps/internal/realtime"
	"nwct/client-nps/internal/toolkit"
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
	IP        string   `json:"ip"`
	MAC       string   `json:"mac"`
	Name      string   `json:"name"`
	Vendor    string   `json:"vendor"`
	Type      string   `json:"type"`
	OS        string   `json:"os"`
	Status    string   `json:"status"`
	OpenPorts []int    `json:"open_ports"`
	LastSeen  string   `json:"last_seen"`
	FirstSeen string   `json:"first_seen"`
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
	scanning    bool
	scanStatus  *ScanStatus
	mu          sync.RWMutex
	db          *sql.DB
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

		// 保存到数据库
		dbDevice := &database.Device{
			IP:        device.IP,
			MAC:       device.MAC,
			Name:      device.Name,
			Vendor:    device.Vendor,
			Type:      device.Type,
			OS:        device.OS,
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
	commonPorts := []int{22, 23, 80, 443, 3389, 8080, 3306, 5432}
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
	commonPorts := []int{22, 23, 80, 443, 3389, 8080}
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
	// 简化的OUI识别（实际应该使用完整的OUI数据库）
	ouiMap := map[string]string{
		"00:50:56": "VMware",
		"00:0c:29": "VMware",
		"00:1b:21": "Intel",
		"00:1e:67": "Intel",
		"00:25:90": "Apple",
		"00:23:12": "Apple",
		"00:1a:79": "Apple",
		"00:1e:c2": "Apple",
		"00:26:4a": "Apple",
		"00:26:bb": "Apple",
		"08:00:27": "VirtualBox",
		"52:54:00": "QEMU",
	}

	if len(mac) >= 8 {
		oui := mac[:8]
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
		return names[0]
	}

	// TODO: NetBIOS查询
	// TODO: mDNS查询

	return ""
}
