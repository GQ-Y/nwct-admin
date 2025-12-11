package config

import "fmt"

// Manager 配置管理器
type Manager struct {
	config *Config
}

// NewManager 创建配置管理器
func NewManager() *Manager {
	return &Manager{}
}

// GetConfig 获取配置
func (m *Manager) GetConfig() *Config {
	return m.config
}

// SetConfig 设置配置
func (m *Manager) SetConfig(cfg *Config) {
	m.config = cfg
}

// Reload 重新加载配置
func (m *Manager) Reload() error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	m.config = cfg
	return nil
}

// Save 保存配置
func (m *Manager) Save() error {
	if m.config == nil {
		return fmt.Errorf("配置未初始化")
	}
	return m.config.Save()
}

