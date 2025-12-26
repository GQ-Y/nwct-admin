package display

import (
	"fmt"
	"math"
	"time"

	appcfg "totoro-device/config"
	"totoro-device/internal/frp"
	"totoro-device/internal/network"
)

// Manager 显示管理器
type Manager struct {
	display     Display
	graphics    *Graphics
	pageManager *PageManager
	services    *AppServices
	running     bool
}

// NewManager 创建显示管理器
func NewManager(disp Display) *Manager {
	// 初始化业务服务（真实功能接入点）
	cfg, _ := appcfg.LoadConfig()
	nm := network.NewManager()
	fc := frp.GetGlobalClient()
	if fc == nil && cfg != nil {
		// 仅用于 UI 读取/管理隧道；不主动 Connect，避免在预览阶段启动 frpc
		fc = frp.NewClient(&cfg.FRPServer)
		frp.SetGlobalClient(fc)
	}
	services := NewAppServices(cfg, nm, fc)
	return NewManagerWithServices(disp, services)
}

// NewManagerWithServices 使用外部注入的 services（用于与主程序共享 netManager/frpClient/config）
func NewManagerWithServices(disp Display, services *AppServices) *Manager {
	// 获取后缓冲区用于绘图
	backBuffer := disp.GetBackBuffer()
	graphics := NewGraphics(backBuffer)
	pm := NewPageManager()

	// 创建所有页面
	statusPage := NewStatusPage()
	settingsPage := NewSettingsPage(pm)
	aboutPage := NewAboutPage(pm)
	networkPage := NewNetworkPage(pm)
	ethernetPage := NewEthernetPage(pm)
	wifiListPage := NewWiFiListPage(pm)
	wifiConnectPage := NewWiFiConnectPage(pm)
	tunnelListPage := NewTunnelListPage(pm)
	tunnelEditPage := NewTunnelEditPage(pm)

	// 注入 services
	statusPage.SetServices(services)
	networkPage.SetServices(services)
	ethernetPage.SetServices(services)
	wifiListPage.SetServices(services)
	wifiConnectPage.SetServices(services)
	tunnelListPage.SetServices(services)
	tunnelEditPage.SetServices(services)

	// 注册页面
	pm.RegisterPage("status", statusPage)
	pm.RegisterPage("settings", settingsPage)
	pm.RegisterPage("about", aboutPage)
	pm.RegisterPage("network", networkPage)
	pm.RegisterPage("ethernet", ethernetPage)
	pm.RegisterPage("wifi_list", wifiListPage)
	pm.RegisterPage("wifi_connect", wifiConnectPage)
	pm.RegisterPage("tunnel_list", tunnelListPage)
	pm.RegisterPage("tunnel_edit", tunnelEditPage)

	// 设置主页跳转逻辑
	statusPage.SetOnEnterSettings(func() {
		pm.NavigateTo("settings")
	})

	// 设置默认页面
	pm.NavigateTo("status")

	return &Manager{
		display:     disp,
		graphics:    graphics,
		pageManager: pm,
		services:    services,
		running:     false,
	}
}

// Run 运行显示循环
func (m *Manager) Run() error {
	m.running = true
	lastTime := time.Now()
	frameCount := 0
	fpsTime := time.Now()

	for m.running {
		// 轮询事件
		if shouldQuit := m.display.PollEvents(); shouldQuit {
			fmt.Println("🛑 收到退出事件（PollEvents=true）")
			m.running = false
			break
		}
		
		// 计算帧时间
		now := time.Now()
		deltaTime := now.Sub(lastTime).Milliseconds()
		lastTime = now

		// 更新当前页面
		m.pageManager.Update(deltaTime)

		// 渲染
		// 清空背景
		m.graphics.Clear(ColorBackgroundStart)
		
		if err := m.pageManager.Render(m.graphics); err != nil {
			return fmt.Errorf("渲染失败: %w", err)
		}

		// 更新显示硬件/窗口
		if err := m.display.Update(); err != nil {
			return fmt.Errorf("更新显示失败: %w", err)
		}

		// FPS 统计
		frameCount++
		if time.Since(fpsTime) >= time.Second {
			// fmt.Printf("FPS: %d\n", frameCount)
			frameCount = 0
			fpsTime = now
		}

		// 处理触摸事件
		events := m.display.GetTouchEvents()
		// 触摸坐标从“真实像素”映射回 480 逻辑坐标，保证布局/命中区域一致
		sx := float64(m.display.GetWidth()) / float64(designW)
		sy := float64(m.display.GetHeight()) / float64(designH)
		if sx <= 0 {
			sx = 1
		}
		if sy <= 0 {
			sy = 1
		}
		for _, event := range events {
			x := int(math.Round(float64(event.X) / sx))
			y := int(math.Round(float64(event.Y) / sy))
			m.pageManager.HandleTouch(x, y, event.Type)
		}

		// 限制帧率
		time.Sleep(16 * time.Millisecond) // ~60 FPS
	}

	return nil
}

// Stop 停止显示循环
func (m *Manager) Stop() {
	m.running = false
}

// GetStatusPage 获取状态页 (暴露给外部更新数据)
func (m *Manager) GetStatusPage() *StatusPage {
	return m.pageManager.GetStatusPage()
}
