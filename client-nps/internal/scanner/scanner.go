package scanner

// Scanner 设备扫描器接口
type Scanner interface {
	StartScan(subnet string) error
	StopScan() error
	GetDevices() ([]Device, error)
	GetDeviceDetail(ip string) (*DeviceDetail, error)
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

// deviceScanner 设备扫描器实现
type deviceScanner struct {
	scanning bool
}

// NewScanner 创建扫描器
func NewScanner() Scanner {
	return &deviceScanner{
		scanning: false,
	}
}

// StartScan 启动扫描
func (ds *deviceScanner) StartScan(subnet string) error {
	ds.scanning = true
	// TODO: 实现扫描逻辑
	return nil
}

// StopScan 停止扫描
func (ds *deviceScanner) StopScan() error {
	ds.scanning = false
	return nil
}

// GetDevices 获取设备列表
func (ds *deviceScanner) GetDevices() ([]Device, error) {
	// TODO: 从数据库查询设备
	return []Device{}, nil
}

// GetDeviceDetail 获取设备详情
func (ds *deviceScanner) GetDeviceDetail(ip string) (*DeviceDetail, error) {
	// TODO: 从数据库查询设备详情
	return &DeviceDetail{}, nil
}

