package display

import (
	"fmt"
	"math"
	"time"

	appcfg "totoro-device/config"
	"totoro-device/internal/frp"
	"totoro-device/internal/network"
	"totoro-device/internal/system"
)

// Manager 显示管理器
type Manager struct {
	display     Display
	graphics    *Graphics
	pageManager *PageManager
	services    *AppServices
	running     bool

	// 屏幕熄屏/唤醒（背光）
	bl           *system.Backlight
	lastInputAt  time.Time
	screenIsOff  bool
	lastBright   int // 0-100（用于恢复）
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
	splashPage := NewSplashPage(pm)
	statusPage := NewStatusPage()
	settingsPage := NewSettingsPage(pm)
	appsPage := NewAppsPage(pm)
	systemSettingsPage := NewSystemSettingsPage(pm)
	soundSettingsPage := NewSoundSettingsPage(pm)
	screenSettingsPage := NewScreenSettingsPage(pm)
	cloudPage := NewCloudPage(pm)
	cloudStatusPage := NewCloudStatusPage(pm)
	cloudPublicNodesPage := NewCloudPublicNodesPage(pm)
	cloudInvitePage := NewCloudInvitePage(pm)
	aboutPage := NewAboutPage(pm)
	networkPage := NewNetworkPage(pm)
	ethernetPage := NewEthernetPage(pm)
	wifiListPage := NewWiFiListPage(pm)
	wifiConnectPage := NewWiFiConnectPage(pm)
	tunnelListPage := NewTunnelListPage(pm)
	tunnelEditPage := NewTunnelEditPage(pm)

	// 注入 services
	statusPage.SetServices(services)
	systemSettingsPage.SetServices(services)
	soundSettingsPage.SetServices(services)
	screenSettingsPage.SetServices(services)
	cloudPage.SetServices(services)
	cloudStatusPage.SetServices(services)
	cloudPublicNodesPage.SetServices(services)
	cloudInvitePage.SetServices(services)
	networkPage.SetServices(services)
	ethernetPage.SetServices(services)
	wifiListPage.SetServices(services)
	wifiConnectPage.SetServices(services)
	tunnelListPage.SetServices(services)
	tunnelEditPage.SetServices(services)

	// 注册页面
	pm.RegisterPage("splash", splashPage)
	pm.RegisterPage("status", statusPage)
	pm.RegisterPage("settings", settingsPage)
	pm.RegisterPage("apps", appsPage)
	pm.RegisterPage("system_settings", systemSettingsPage)
	pm.RegisterPage("sound_settings", soundSettingsPage)
	pm.RegisterPage("screen_settings", screenSettingsPage)
	pm.RegisterPage("cloud", cloudPage)
	pm.RegisterPage("cloud_status", cloudStatusPage)
	pm.RegisterPage("cloud_public_nodes", cloudPublicNodesPage)
	pm.RegisterPage("cloud_invite", cloudInvitePage)
	pm.RegisterPage("about", aboutPage)
	pm.RegisterPage("network", networkPage)
	pm.RegisterPage("ethernet", ethernetPage)
	pm.RegisterPage("wifi_list", wifiListPage)
	pm.RegisterPage("wifi_connect", wifiConnectPage)
	pm.RegisterPage("tunnel_list", tunnelListPage)
	pm.RegisterPage("tunnel_edit", tunnelEditPage)

	// 设置主页跳转逻辑
	statusPage.SetOnEnterSettings(func() {
		pm.NavigateTo("apps")
	})

	// 设置默认页面：先启动页（不入栈），再自动切到 status
	_ = pm.SwitchTo("splash")

	// 背光探测（best-effort）
	var bl *system.Backlight
	if b, err := system.DiscoverBacklight(); err == nil {
		bl = b
	}

	return &Manager{
		display:     disp,
		graphics:    graphics,
		pageManager: pm,
		services:    services,
		running:     false,
		bl:          bl,
		lastInputAt: time.Now(),
		screenIsOff: false,
		lastBright:  100,
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
		if len(events) > 0 {
			m.lastInputAt = time.Now()
			// 触摸唤醒
			if m.screenIsOff {
				m.wakeScreen()
			}
		}
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

		// 熄屏逻辑：空闲到时关闭背光；触摸自动唤醒
		m.maybeScreenOff()

		// 限制帧率
		time.Sleep(16 * time.Millisecond) // ~60 FPS
	}

	return nil
}

func (m *Manager) screenOffSeconds() int {
	if m.services == nil || m.services.Config == nil || m.services.Config.System.ScreenOffSeconds == nil {
		return 0
	}
	sec := *m.services.Config.System.ScreenOffSeconds
	if sec < 0 {
		return 0
	}
	return sec
}

func (m *Manager) desiredBrightness() int {
	if m.services == nil || m.services.Config == nil || m.services.Config.System.Brightness == nil {
		// 若用户未设置，则用 lastBright（一般为 100 或上次读取值）
		return m.lastBright
	}
	v := *m.services.Config.System.Brightness
	if v < 0 {
		v = 0
	}
	if v > 100 {
		v = 100
	}
	return v
}

func (m *Manager) maybeScreenOff() {
	if m.bl == nil {
		return
	}
	sec := m.screenOffSeconds()
	if sec <= 0 {
		// 不熄屏：如果当前处于熄屏态则唤醒
		if m.screenIsOff {
			m.wakeScreen()
		}
		return
	}
	if m.screenIsOff {
		return
	}
	if time.Since(m.lastInputAt) < time.Duration(sec)*time.Second {
		return
	}
	// 记录恢复亮度：优先用配置亮度，否则读取当前亮度
	if m.services != nil && m.services.Config != nil && m.services.Config.System.Brightness != nil {
		m.lastBright = m.desiredBrightness()
	} else if p, err := m.bl.GetPercent(); err == nil {
		m.lastBright = p
	}
	_ = m.bl.Off()
	m.screenIsOff = true
}

func (m *Manager) wakeScreen() {
	if m.bl == nil {
		m.screenIsOff = false
		return
	}
	b := m.desiredBrightness()
	if b <= 0 {
		// 避免“永远黑屏”：最小恢复到 10%
		b = 10
	}
	_ = m.bl.SetPercent(b)
	m.screenIsOff = false
}

// Stop 停止显示循环
func (m *Manager) Stop() {
	m.running = false
}

// GetStatusPage 获取状态页 (暴露给外部更新数据)
func (m *Manager) GetStatusPage() *StatusPage {
	return m.pageManager.GetStatusPage()
}
