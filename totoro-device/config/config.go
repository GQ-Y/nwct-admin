package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// DefaultBridgeURL 未显式配置（bridge.url / TOTOTO_BRIDGE_URL）时的内置桥梁地址
const DefaultBridgeURL = "http://192.168.2.32:18090"

// DefaultDeviceName 默认设备名称（会写入 config.json 的 device.name；DHCP/hostname 也会用到）
// 可在编译时覆盖：
//
//	go build -ldflags "-X 'totoro-device/config.DefaultDeviceName=Totoro S1 Ultra'"
var DefaultDeviceName = "Totoro S1 Pro"

// Config 应用配置
type Config struct {
	Initialized bool            `json:"initialized"`
	Device      DeviceConfig    `json:"device"`
	Network     NetworkConfig   `json:"network"`
	System      SystemConfig    `json:"system"`
	Bridge      BridgeConfig    `json:"bridge"`
	FRPServer   FRPServerConfig `json:"frp_server"`
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

// SystemConfig 系统设置（设备侧面板设置）
type SystemConfig struct {
	// Volume 音量（0~30，对应 Luckfox 文档中的 DAC LINEOUT Volume）
	// nil 表示未设置（不主动修改系统）
	Volume *int `json:"volume,omitempty"`
	// Brightness 亮度（不同屏幕实现不同；若无背光接口则可能不生效）
	Brightness *int `json:"brightness,omitempty"`
	// ScreenOffSeconds 熄屏时间（秒，0 表示不熄屏）
	ScreenOffSeconds *int `json:"screen_off_seconds,omitempty"`
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

// BridgeConfig 桥梁平台配置（设备侧）
type BridgeConfig struct {
	URL         string `json:"url"`      // 例如 http://127.0.0.1:18090
	DeviceToken string `json:"-"`        // 不落盘：SQLite 仅存密文，用时解密到内存
	ExpiresAt   string `json:"-"`        // 不落盘：同上
	LastMAC     string `json:"last_mac"` // 仅记录
}

// FRPMode 设备端 frpc 连接 frps 的工作模式
// - builtin：默认连接“内置/预设”的 frps（官方/你的默认节点）
// - manual：用户手动填写 frps 连接信息
// - public：通过公开节点（node_api + 邀请码）兑换 ticket 后连接
type FRPMode string

const (
	FRPModeBuiltin FRPMode = "builtin"
	FRPModeManual  FRPMode = "manual"
	FRPModePublic  FRPMode = "public"
)

// FRPProfile 一种模式下的连接配置快照（会持久化）
type FRPProfile struct {
	Server       string `json:"server"`
	Token        string `json:"token"`
	AdminAddr    string `json:"admin_addr"`
	AdminUser    string `json:"admin_user"`
	AdminPwd     string `json:"admin_pwd"`
	TotoroTicket string `json:"-"` // 不落盘：属于短期票据
	DomainSuffix string `json:"domain_suffix"`
	HTTPEnabled  bool   `json:"http_enabled"`
	HTTPSEnabled bool   `json:"https_enabled"`
}

// FRPPublicProfile 公开节点模式的持久化信息
type FRPPublicProfile struct {
	FRPProfile
	NodeAPI          string `json:"-"`                            // 旧实现遗留，不再使用（兑换在 bridge）
	InviteCode       string `json:"-"`                            // 不落盘：SQLite 仅存密文
	TicketExpiresAt  string `json:"-"`                            // 不落盘：短期票据
	LastResolveError string `json:"last_resolve_error,omitempty"` // 最近一次自动换票失败原因（仅用于排障）
}

// FRPServerConfig FRP服务端配置
type FRPServerConfig struct {
	// Mode 当前选择的连接方式（选择后保持）
	Mode FRPMode `json:"mode"`

	// 三种模式各自的“最后一次配置”，用于切换后能恢复
	Builtin FRPProfile       `json:"builtin"`
	Manual  FRPProfile       `json:"manual"`
	Public  FRPPublicProfile `json:"public"`

	// Active（当前生效配置）：为了减少全项目改动，保留这些字段作为“当前模式的展开值”
	Server    string `json:"server"`     // 117.172.29.237:7000
	Token     string `json:"token"`      // token123456
	AdminAddr string `json:"admin_addr"` // 117.172.29.237:7500
	AdminUser string `json:"admin_user"` // admin
	AdminPwd  string `json:"admin_pwd"`  // admin_nAhTnN
	// TotoroTicket 用于 Totoro 节点的连接票据（写入 frpc metas：meta_totoro_ticket）。
	// 注意：这不是 frp 的 token。
	TotoroTicket string `json:"-"` // 不落盘：属于短期票据
	// DomainSuffix HTTP/HTTPS 隧道的默认域名后缀（前端只填写前缀即可）
	DomainSuffix string `json:"domain_suffix"` // frpc.zyckj.club
	// HTTPEnabled/HTTPSEnabled：由桥梁节点配置下发，决定是否允许创建 http/https 隧道
	HTTPEnabled  bool `json:"http_enabled"`
	HTTPSEnabled bool `json:"https_enabled"`
}

func (c *FRPServerConfig) ensureDefaults() {
	// 默认模式：builtin
	if c.Mode == "" {
		c.Mode = FRPModeBuiltin
	}
	// builtin 不再内置任何默认节点信息：必须从桥梁平台 official_nodes 同步

	// 若是旧配置（只写了 Active 字段），把 Active 回填到对应 mode profile，避免丢失
	switch c.Mode {
	case FRPModeManual:
		if strings.TrimSpace(c.Manual.Server) == "" && strings.TrimSpace(c.Server) != "" {
			c.Manual = FRPProfile{
				Server:       c.Server,
				Token:        c.Token,
				AdminAddr:    c.AdminAddr,
				AdminUser:    c.AdminUser,
				AdminPwd:     c.AdminPwd,
				TotoroTicket: c.TotoroTicket,
				DomainSuffix: c.DomainSuffix,
				HTTPEnabled:  c.HTTPEnabled,
				HTTPSEnabled: c.HTTPSEnabled,
			}
		}
	case FRPModePublic:
		if strings.TrimSpace(c.Public.Server) == "" && strings.TrimSpace(c.Server) != "" {
			c.Public.FRPProfile = FRPProfile{
				Server:       c.Server,
				Token:        c.Token,
				AdminAddr:    c.AdminAddr,
				AdminUser:    c.AdminUser,
				AdminPwd:     c.AdminPwd,
				TotoroTicket: c.TotoroTicket,
				DomainSuffix: c.DomainSuffix,
				HTTPEnabled:  c.HTTPEnabled,
				HTTPSEnabled: c.HTTPSEnabled,
			}
		}
	default:
		// builtin：如果旧配置只写了 active，则把 active 当作 builtin 的一次快照
		if strings.TrimSpace(c.Builtin.Server) == "" && strings.TrimSpace(c.Server) != "" {
			c.Builtin = FRPProfile{
				Server:       c.Server,
				Token:        c.Token,
				AdminAddr:    c.AdminAddr,
				AdminUser:    c.AdminUser,
				AdminPwd:     c.AdminPwd,
				TotoroTicket: c.TotoroTicket,
				DomainSuffix: c.DomainSuffix,
				HTTPEnabled:  c.HTTPEnabled,
				HTTPSEnabled: c.HTTPSEnabled,
			}
		}
	}
}

// SyncActiveFromMode 把当前 mode 的 profile 展开到 Active 字段，供其它模块直接读取
func (c *FRPServerConfig) SyncActiveFromMode() {
	c.ensureDefaults()
	var p FRPProfile
	switch c.Mode {
	case FRPModeManual:
		p = c.Manual
	case FRPModePublic:
		p = c.Public.FRPProfile
	default:
		p = c.Builtin
	}
	c.Server = strings.TrimSpace(p.Server)
	c.Token = strings.TrimSpace(p.Token)
	c.AdminAddr = strings.TrimSpace(p.AdminAddr)
	c.AdminUser = strings.TrimSpace(p.AdminUser)
	c.AdminPwd = strings.TrimSpace(p.AdminPwd)
	c.TotoroTicket = strings.TrimSpace(p.TotoroTicket)
	c.DomainSuffix = strings.TrimPrefix(strings.TrimSpace(p.DomainSuffix), ".")
	c.HTTPEnabled = p.HTTPEnabled
	c.HTTPSEnabled = p.HTTPSEnabled
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
	defVol := 15
	defOff := 0
	defPort := 80
	if runtime.GOOS == "linux" {
		// Luckfox/Buildroot 上建议避开系统 80 端口服务
		defPort = 18080
	}
	return &Config{
		Initialized: false,
		Device: DeviceConfig{
			ID:   "DEV001",
			Name: DefaultDeviceName,
		},
		Network: NetworkConfig{
			Interface: "eth0",
			IPMode:    "dhcp",
		},
		System: SystemConfig{
			Volume:           &defVol,
			Brightness:       nil,
			ScreenOffSeconds: &defOff,
		},
		Bridge: BridgeConfig{
			URL: "",
		},
		FRPServer: func() FRPServerConfig {
			// 默认模式：builtin（官方内置节点从桥梁平台同步，不再硬编码）
			builtin := FRPProfile{}
			c := FRPServerConfig{
				Mode:    FRPModeBuiltin,
				Builtin: builtin,
				Manual:  FRPProfile{},
				Public:  FRPPublicProfile{},
			}
			// 展开到 Active 字段（保持旧代码读取不变）
			c.SyncActiveFromMode()
			return c
		}(),
		Scanner: ScannerConfig{
			AutoScan:     true,
			ScanInterval: 300,
			Timeout:      30,
			Concurrency:  5, // 从 10 降到 5，减少并发连接数，节省内存
		},
		Server: ServerConfig{
			Port: defPort,
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

// ResolveBridgeBase 返回设备端应使用的桥梁 BaseURL（带 scheme，且无尾部 /）
// 优先级：
// 1) 配置文件 bridge.url
// 2) 环境变量 TOTOTO_BRIDGE_URL
// 3) DefaultBridgeURL
func ResolveBridgeBase(cfg *Config) string {
	base := ""
	if cfg != nil {
		base = strings.TrimSpace(cfg.Bridge.URL)
	}
	if base == "" {
		base = strings.TrimSpace(os.Getenv("TOTOTO_BRIDGE_URL"))
	}
	if base == "" {
		base = DefaultBridgeURL
	}
	base = strings.TrimSpace(base)
	if base == "" {
		return ""
	}
	// 如果用户只填了 host:port，则默认补 http://
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "http://" + base
	}
	return strings.TrimRight(base, "/")
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

	// FRP defaults：三种模式 + Active 展开
	modeBefore := cfg.FRPServer.Mode
	serverBefore := strings.TrimSpace(cfg.FRPServer.Server)
	domainBefore := strings.TrimSpace(cfg.FRPServer.DomainSuffix)
	cfg.FRPServer.SyncActiveFromMode()
	// 如果之前没有 mode 或 server/域名为空，说明我们补齐过默认值
	if modeBefore == "" || serverBefore == "" || domainBefore == "" {
		changed = true
	}

	// Bridge defaults：未配置时使用内置桥梁地址
	bridgeBefore := strings.TrimSpace(cfg.Bridge.URL)
	cfg.Bridge.URL = ResolveBridgeBase(&cfg)
	if bridgeBefore == "" && cfg.Bridge.URL != "" {
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
