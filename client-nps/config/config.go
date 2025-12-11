package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config 应用配置
type Config struct {
	Initialized bool          `json:"initialized"`
	Device      DeviceConfig  `json:"device"`
	Network     NetworkConfig `json:"network"`
	NPSServer   NPSServerConfig `json:"nps_server"`
	MQTT        MQTTConfig    `json:"mqtt"`
	Scanner     ScannerConfig `json:"scanner"`
	Server      ServerConfig  `json:"server"`
	Database    DatabaseConfig `json:"database"`
	Auth        AuthConfig    `json:"auth"`
}

// DeviceConfig 设备配置
type DeviceConfig struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// NetworkConfig 网络配置
type NetworkConfig struct {
	Interface string `json:"interface"` // eth0, wlan0
	IPMode    string `json:"ip_mode"`   // dhcp, static
	IP        string `json:"ip"`
	Netmask   string `json:"netmask"`
	Gateway   string `json:"gateway"`
	DNS       string `json:"dns"`
	WiFi      WiFiConfig `json:"wifi"`
}

// WiFiConfig WiFi配置
type WiFiConfig struct {
	SSID     string `json:"ssid"`
	Password string `json:"password"`
	Security string `json:"security"` // WPA2, WPA, WEP, Open
}

// NPSServerConfig NPS服务端配置
type NPSServerConfig struct {
	Server   string `json:"server"`   // host:port
	VKey     string `json:"vkey"`     // 验证密钥
	ClientID string `json:"client_id"`
}

// MQTTConfig MQTT配置
type MQTTConfig struct {
	Server   string `json:"server"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	ClientID string `json:"client_id"`
	TLS      bool   `json:"tls"`
}

// ScannerConfig 扫描器配置
type ScannerConfig struct {
	AutoScan    bool `json:"auto_scan"`
	ScanInterval int `json:"scan_interval"` // 秒
	Timeout     int  `json:"timeout"`        // 秒
	Concurrency int  `json:"concurrency"`    // 并发数
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port int    `json:"port"`
	Host string `json:"host"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Path string `json:"path"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	PasswordHash string `json:"password_hash"` // bcrypt hash
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Initialized: false,
		Device: DeviceConfig{
			ID:   "device_001",
			Name: "内网穿透盒子",
		},
		Network: NetworkConfig{
			Interface: "eth0",
			IPMode:    "dhcp",
		},
		NPSServer: NPSServerConfig{},
		MQTT: MQTTConfig{
			Port: 1883,
		},
		Scanner: ScannerConfig{
			AutoScan:     true,
			ScanInterval: 300,
			Timeout:      30,
			Concurrency:  10,
		},
		Server: ServerConfig{
			Port: 8080,
			Host: "0.0.0.0",
		},
		Database: DatabaseConfig{
			Path: "/var/nwct/devices.db",
		},
		Auth: AuthConfig{},
	}
}

// GetConfigPath 获取配置文件路径
func GetConfigPath() string {
	configPath := os.Getenv("NWCT_CONFIG_PATH")
	if configPath == "" {
		configPath = "/etc/nwct/config.json"
	}
	return configPath
}

// LoadConfig 加载配置
func LoadConfig() (*Config, error) {
	configPath := GetConfigPath()
	
	// 如果配置文件不存在，返回默认配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := DefaultConfig()
		// 创建配置目录
		dir := filepath.Dir(configPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
		// 保存默认配置
		if err := cfg.Save(); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save 保存配置
func (c *Config) Save() error {
	configPath := GetConfigPath()
	
	// 创建配置目录
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("服务器端口无效: %d", c.Server.Port)
	}
	
	if c.Database.Path == "" {
		return fmt.Errorf("数据库路径不能为空")
	}
	
	return nil
}

