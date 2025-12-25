package display

import (
	"fmt"
	"time"
)

// PageManager 页面管理器
type PageManager struct {
	pages               map[string]Page
	pageStack           []Page    // 页面导航堆栈
	currentPage         Page      // 当前显示的页面
	lastInteractionTime time.Time // 最后一次交互时间
	homePageName        string    // 首页名称（用于超时归位）
}

// NewPageManager 创建页面管理器
func NewPageManager() *PageManager {
	return &PageManager{
		pages:               make(map[string]Page),
		pageStack:           make([]Page, 0),
		lastInteractionTime: time.Now(),
		homePageName:        "status", // 默认 status 为首页
	}
}

// RegisterPage 注册页面
func (pm *PageManager) RegisterPage(name string, page Page) {
	pm.pages[name] = page
}

// NavigateTo 导航到新页面 (Push)
func (pm *PageManager) NavigateTo(name string) error {
	page, ok := pm.pages[name]
	if !ok {
		return fmt.Errorf("页面不存在: %s", name)
	}

	// 如果当前有页面，压入堆栈
	if pm.currentPage != nil {
		pm.currentPage.OnExit()
		pm.pageStack = append(pm.pageStack, pm.currentPage)
	}

	pm.currentPage = page
	pm.currentPage.OnEnter()
	pm.lastInteractionTime = time.Now() // 重置超时计时

	return nil
}

// SwitchTo 切换页面 (Replace - 清空堆栈)
// 通常用于直接跳转回首页，或者平级切换不保留历史
func (pm *PageManager) SwitchTo(name string) error {
	page, ok := pm.pages[name]
	if !ok {
		return fmt.Errorf("页面不存在: %s", name)
	}

	if pm.currentPage != nil {
		pm.currentPage.OnExit()
	}

	// 清空堆栈
	pm.pageStack = make([]Page, 0)

	pm.currentPage = page
	pm.currentPage.OnEnter()
	pm.lastInteractionTime = time.Now()

	return nil
}

// Back 返回上一页 (Pop)
func (pm *PageManager) Back() {
	if len(pm.pageStack) == 0 {
		return // 已经在根页面，无法返回
	}

	// 退出当前页面
	if pm.currentPage != nil {
		pm.currentPage.OnExit()
	}

	// 弹出栈顶页面
	lastIdx := len(pm.pageStack) - 1
	prevPage := pm.pageStack[lastIdx]
	pm.pageStack = pm.pageStack[:lastIdx]

	pm.currentPage = prevPage
	pm.currentPage.OnEnter()
	pm.lastInteractionTime = time.Now()
}

// GetCurrentPage 获取当前页面
func (pm *PageManager) GetCurrentPage() Page {
	return pm.currentPage
}

// GetStatusPage 特殊方法获取状态页（因为它是首页）
func (pm *PageManager) GetStatusPage() *StatusPage {
	if p, ok := pm.pages["status"]; ok {
		if sp, ok := p.(*StatusPage); ok {
			return sp
		}
	}
	return nil
}

// GetWiFiConnectPage 获取 WiFi 连接页
func (pm *PageManager) GetWiFiConnectPage() *WiFiConnectPage {
	if p, ok := pm.pages["wifi_connect"]; ok {
		if cp, ok := p.(*WiFiConnectPage); ok {
			return cp
		}
	}
	return nil
}

// GetTunnelEditPage 获取隧道编辑页
func (pm *PageManager) GetTunnelEditPage() *TunnelEditPage {
	if p, ok := pm.pages["tunnel_edit"]; ok {
		if tp, ok := p.(*TunnelEditPage); ok {
			return tp
		}
	}
	return nil
}

// Render 渲染当前页面
func (pm *PageManager) Render(g *Graphics) error {
	if pm.currentPage != nil {
		return pm.currentPage.Render(g)
	}
	return nil
}

// HandleTouch 处理触摸事件
func (pm *PageManager) HandleTouch(x, y int, touchType TouchType) bool {
	// 任何触摸都视为交互，更新时间
	if touchType == TouchDown || touchType == TouchUp || touchType == TouchMove {
		pm.lastInteractionTime = time.Now()
	}

	if pm.currentPage != nil {
		return pm.currentPage.HandleTouch(x, y, touchType)
	}
	return false
}

// Update 更新当前页面及全局逻辑
func (pm *PageManager) Update(deltaTime int64) {
	// 1. 检查自动归位逻辑 (30秒无操作)
	if pm.currentPage != nil && pm.currentPage.GetName() != pm.homePageName {
		if time.Since(pm.lastInteractionTime) > 30*time.Second {
			fmt.Println("⏳ 30秒无操作，自动返回首页")
			pm.SwitchTo(pm.homePageName)
			return
		}
	}

	// 2. 更新页面
	if pm.currentPage != nil {
		pm.currentPage.Update(deltaTime)
	}
}
