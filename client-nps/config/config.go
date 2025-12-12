package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config 应用配置
type Config struct {
	Initialized bool            `json:"initialized"`
	Device      DeviceConfig    `json:"device"`
	Network     NetworkConfig   `json:"network"`
	NPSServer   NPSServerConfig `json:"nps_server"`
	MQTT        MQTTConfig      `json:"mqtt"`
	Scanner     ScannerConfig   `json:"scanner"`
	Server      ServerConfig    `json:"server"`
	Database    DatabaseConfig  `json:"database"`
	Auth        AuthConfig      `json:"auth"`
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
	// WiFi 旧的单个WiFi配置（保留用于从旧配置迁移）
	WiFi WiFiConfig `json:"wifi"`
	// WiFiProfiles 记忆的多个WiFi配置（用于自动连接）
	WiFiProfiles []WiFiProfile `json:"wifi_profiles"`
}

// WiFiConfig WiFi配置
type WiFiConfig struct {
	SSID     string `json:"ssid"`
	Password string `json:"password"`
	Security string `json:"security"` // WPA2, WPA, WEP, Open
}

// WiFiProfile 记忆的WiFi配置（类似电脑的“已保存网络”）
type WiFiProfile struct {
	SSID        string `json:"ssid"`
	Password    string `json:"password"` // 暂存明文；如需加密可后续引入密钥
	Security    string `json:"security"`
	AutoConnect bool   `json:"auto_connect"`
	Priority    int    `json:"priority"` // 越大越优先

	LastSuccessAt string `json:"last_success_at,omitempty"`
	LastTriedAt   string `json:"last_tried_at,omitempty"`
	LastError     string `json:"last_error,omitempty"`
}

// NPSServerConfig NPS服务端配置
type NPSServerConfig struct {
	Server   string `json:"server"` // host:port
	VKey     string `json:"vkey"`   // 验证密钥
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
	AutoScan     bool `json:"auto_scan"`
	ScanInterval int  `json:"scan_interval"` // 秒
	Timeout      int  `json:"timeout"`       // 秒
	Concurrency  int  `json:"concurrency"`   // 并发数
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

	// 迁移：如果旧的 wifi 配置存在，但 profiles 为空，则自动迁移为一个 profile
	if cfg.Network.WiFiProfiles == nil {
		cfg.Network.WiFiProfiles = []WiFiProfile{}
	}
	if cfg.Network.WiFi.SSID != "" && len(cfg.Network.WiFiProfiles) == 0 {
		cfg.Network.WiFiProfiles = append(cfg.Network.WiFiProfiles, WiFiProfile{
			SSID:        cfg.Network.WiFi.SSID,
			Password:    cfg.Network.WiFi.Password,
			Security:    cfg.Network.WiFi.Security,
			AutoConnect: true,
			Priority:    10,
		})
		// 不强制清空旧字段，避免用户困惑；但保存时会同时存在
		_ = cfg.Save()
	}

	return &cfg, nil
}

// UpsertWiFiProfile 新增或更新一个 WiFiProfile（按 SSID 唯一）
func (c *Config) UpsertWiFiProfile(p WiFiProfile) {
	if c.Network.WiFiProfiles == nil {
		c.Network.WiFiProfiles = []WiFiProfile{}
	}
	for i := range c.Network.WiFiProfiles {
		if c.Network.WiFiProfiles[i].SSID == p.SSID {
			c.Network.WiFiProfiles[i] = p
			return
		}
	}
	c.Network.WiFiProfiles = append(c.Network.WiFiProfiles, p)
}

// DeleteWiFiProfile 删除一个 WiFiProfile（按 SSID）
func (c *Config) DeleteWiFiProfile(ssid string) bool {
	if c.Network.WiFiProfiles == nil {
		return false
	}
	out := c.Network.WiFiProfiles[:0]
	removed := false
	for _, p := range c.Network.WiFiProfiles {
		if p.SSID == ssid {
			removed = true
			continue
		}
		out = append(out, p)
	}
	c.Network.WiFiProfiles = out
	return removed
}

// MaxWiFiPriority 返回当前已保存 WiFiProfiles 中的最大 priority
func (c *Config) MaxWiFiPriority() int {
	max := 0
	for _, p := range c.Network.WiFiProfiles {
		if p.Priority > max {
			max = p.Priority
		}
	}
	return max
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
