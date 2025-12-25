package display

import (
	"fmt"
	"image"
	"time"
)

// Manager 显示管理器
type Manager struct {
	display      Display
	graphics     *Graphics
	pageManager  *PageManager
	statusPage   *StatusPage
	settingsPage *SettingsPage
	running      bool
}

// NewManager 创建显示管理器
func NewManager(display Display) *Manager {
	// 从 display 获取 backBuffer
	var backBuffer *image.RGBA
	if sdl, ok := display.(*sdlDisplay); ok {
		backBuffer = sdl.backBuffer
	}
	
	graphics := NewGraphics(backBuffer)
	
	pageManager := NewPageManager()

	// 创建页面
	statusPage := NewStatusPage()
	settingsPage := NewSettingsPage()

	// 注册页面
	pageManager.RegisterPage("status", statusPage)
	pageManager.RegisterPage("settings", settingsPage)

	// 设置默认页面
	pageManager.NavigateTo("status")

	return &Manager{
		display:      display,
		graphics:     graphics,
		pageManager:  pageManager,
		statusPage:   statusPage,
		settingsPage: settingsPage,
		running:      false,
	}
}

// Run 运行显示循环
func (m *Manager) Run() error {
	m.running = true
	lastTime := time.Now()
	frameCount := 0
	fpsTime := time.Now()

	fmt.Println("🚀 启动 NWCT 显示预览...")
	fmt.Printf("✅ 显示系统已启动，%dx%d 窗口\n", m.display.GetWidth(), m.display.GetHeight())
	fmt.Println("💡 按 ESC 或关闭窗口退出")

	for m.running {
		// 先处理 SDL 事件（必须在主线程）
		if sdl, ok := m.display.(*sdlDisplay); ok {
			if sdl.PollEvents() {
				m.running = false
				break
			}
		}
		
		// 计算帧时间
		now := time.Now()
		deltaTime := now.Sub(lastTime).Milliseconds()
		lastTime = now

		// 更新当前页面
		m.pageManager.Update(deltaTime)

		// 渲染
		if err := m.pageManager.Render(m.graphics); err != nil {
			return fmt.Errorf("渲染失败: %w", err)
		}

		// 更新显示
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
		for _, event := range events {
			m.pageManager.HandleTouch(event.X, event.Y, event.Type)
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

// GetStatusPage 获取状态页
func (m *Manager) GetStatusPage() *StatusPage {
	return m.statusPage
}
