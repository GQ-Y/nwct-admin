package database

import "time"

// Device 设备模型
type Device struct {
	IP        string    `json:"ip"`
	MAC       string    `json:"mac"`
	Name      string    `json:"name"`
	Vendor    string    `json:"vendor"`
	Model     string    `json:"model"`
	Type      string    `json:"type"`
	OS        string    `json:"os"`
	Extra     string    `json:"extra"`
	Status    string    `json:"status"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}

// DevicePort 设备端口模型
type DevicePort struct {
	ID        int       `json:"id"`
	DeviceIP  string    `json:"device_ip"`
	Port      int       `json:"port"`
	Protocol  string    `json:"protocol"`
	Service   string    `json:"service"`
	Version   string    `json:"version"`
	Status    string    `json:"status"`
	ScannedAt time.Time `json:"scanned_at"`
}

// DeviceHistory 设备历史模型
type DeviceHistory struct {
	ID        int       `json:"id"`
	DeviceIP  string    `json:"device_ip"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

