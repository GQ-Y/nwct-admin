package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

	// NPS 客户端进程（npc）管理参数：默认使用系统 PATH 中的 `npc`
	NPCPath       string   `json:"npc_path,omitempty"`
	NPCConfigPath string   `json:"npc_config_path,omitempty"`
	NPCArgs       []string `json:"npc_args,omitempty"`
}

// MQTTConfig MQTT配置
type MQTTConfig struct {
	Server   string `json:"server"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	ClientID string `json:"client_id"`
	TLS      bool   `json:"tls"`
	// AutoConnect 是否允许程序启动时自动连接 MQTT（用于“内置默认服务”体验）
	AutoConnect bool `json:"auto_connect"`
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
			// 默认内置 MQTT 服务（可在 UI/API 中覆盖）
			Server:   "mqtt.yingzhu.net",
			Port:     1883,
			Username: "nps",
			Password: "nps",
			// 默认 client_id 与设备 id 保持一致，方便追踪/鉴权
			ClientID:    "device_001",
			AutoConnect: true,
		},
		Scanner: ScannerConfig{
			AutoScan:     true,
			ScanInterval: 300,
			Timeout:      30,
			Concurrency:  5, // 从 10 降到 5，减少并发连接数，节省内存
		},
		Server: ServerConfig{
			Port: 80,
			Host: "0.0.0.0",
		},
		Database: DatabaseConfig{
			Path: defaultDBPath(),
		},
		Auth: AuthConfig{},
	}
}

func findRepoRoot() string {
	wd, err := os.Getwd()
	if err != nil || strings.TrimSpace(wd) == "" {
		return ""
	}
	dir := wd
	for i := 0; i < 10; i++ { // 最多向上找 10 层
		if isRepoRoot(dir) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func isRepoRoot(dir string) bool {
	if dir == "" {
		return false
	}
	if st, err := os.Stat(filepath.Join(dir, "client-nps")); err != nil || !st.IsDir() {
		return false
	}
	if st, err := os.Stat(filepath.Join(dir, "client-web")); err != nil || !st.IsDir() {
		return false
	}
	return true
}

func defaultConfigPath() string {
	// Linux 设备侧保持 /etc；本地开发（macOS/Windows）默认落到仓库根目录，方便调试/可见
	if runtime.GOOS == "linux" {
		return "/etc/nwct/config.json"
	}
	if root := findRepoRoot(); root != "" {
		return filepath.Join(root, "config.json")
	}
	// 兜底：当前工作目录
	if wd, err := os.Getwd(); err == nil && strings.TrimSpace(wd) != "" {
		return filepath.Join(wd, "config.json")
	}
	return filepath.Join(os.TempDir(), "nwct", "config.json")
}

func defaultDBPath() string {
	if runtime.GOOS == "linux" {
		return "/var/nwct/devices.db"
	}
	// 默认跟随 repo root 的 data 目录，避免权限问题，也避免污染根目录
	if root := findRepoRoot(); root != "" {
		return filepath.Join(root, "data", "devices.db")
	}
	if wd, err := os.Getwd(); err == nil && strings.TrimSpace(wd) != "" {
		return filepath.Join(wd, "data", "devices.db")
	}
	return filepath.Join(os.TempDir(), "nwct", "devices.db")
}

// GetConfigPath 获取配置文件路径
func GetConfigPath() string {
	configPath := os.Getenv("NWCT_CONFIG_PATH")
	if configPath == "" {
		configPath = defaultConfigPath()
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

	// 用于判断某些字段是否“在原始 JSON 中存在”（兼容旧版本配置）
	var raw map[string]any
	_ = json.Unmarshal(data, &raw)

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// 兼容补全：当配置文件是旧版本或字段缺失时，仅在“为空/零值”场景下补齐默认值
	// 这样可以实现“内置默认 MQTT 服务”，同时不覆盖用户显式配置。
	changed := false

	// MQTT defaults
	if strings.TrimSpace(cfg.MQTT.Server) == "" {
		cfg.MQTT.Server = "mqtt.yingzhu.net"
		changed = true
	}
	if cfg.MQTT.Port <= 0 {
		cfg.MQTT.Port = 1883
		changed = true
	}
	if strings.TrimSpace(cfg.MQTT.Username) == "" {
		cfg.MQTT.Username = "nps"
		changed = true
	}
	if strings.TrimSpace(cfg.MQTT.Password) == "" {
		cfg.MQTT.Password = "nps"
		changed = true
	}
	if strings.TrimSpace(cfg.MQTT.ClientID) == "" {
		if strings.TrimSpace(cfg.Device.ID) != "" {
			cfg.MQTT.ClientID = strings.TrimSpace(cfg.Device.ID)
		} else {
			cfg.MQTT.ClientID = "device_001"
		}
		changed = true
	}
	// MQTT auto_connect：旧配置里字段不存在时默认开启；如果用户显式保存为 false，则尊重用户值
	{
		mqttRaw, _ := raw["mqtt"].(map[string]any)
		if mqttRaw == nil {
			if !cfg.MQTT.AutoConnect {
				cfg.MQTT.AutoConnect = true
				changed = true
			}
		} else {
			if _, ok := mqttRaw["auto_connect"]; !ok {
				cfg.MQTT.AutoConnect = true
				changed = true
			}
		}
	}

	// NPS defaults（server/client_id 可默认，vkey 由用户填写或由“一键连接”自动创建）
	if strings.TrimSpace(cfg.NPSServer.Server) == "" {
		// 本地开发/测试默认走 docker 映射的 bridge 端口
		cfg.NPSServer.Server = "127.0.0.1:19024"
		changed = true
	}
	if strings.TrimSpace(cfg.NPSServer.ClientID) == "" {
		id := strings.TrimSpace(cfg.Device.ID)
		if id == "" {
			id = "device_001"
		}
		id = strings.ReplaceAll(id, "_", "-")
		cfg.NPSServer.ClientID = "nwct-" + id
		changed = true
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

	// 如果我们补齐了默认值，则回写配置，避免每次启动都重复补齐
	if changed {
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
