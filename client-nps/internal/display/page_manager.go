package display

import (
	"fmt"
)

// PageManager 页面管理器
type PageManager struct {
	pages       map[string]Page
	currentPage Page
	prevPage    Page
}

// NewPageManager 创建页面管理器
func NewPageManager() *PageManager {
	return &PageManager{
		pages: make(map[string]Page),
	}
}

// RegisterPage 注册页面
func (pm *PageManager) RegisterPage(name string, page Page) {
	pm.pages[name] = page
}

// NavigateTo 导航到页面
func (pm *PageManager) NavigateTo(name string) error {
	page, ok := pm.pages[name]
	if !ok {
		return fmt.Errorf("页面不存在: %s", name)
	}

	if pm.currentPage != nil {
		pm.currentPage.OnExit()
		pm.prevPage = pm.currentPage
	}

	pm.currentPage = page
	page.OnEnter()

	return nil
}

// Back 返回上一页
func (pm *PageManager) Back() {
	if pm.prevPage != nil {
		prevName := pm.prevPage.GetName()
		pm.NavigateTo(prevName)
	}
}

// GetCurrentPage 获取当前页面
func (pm *PageManager) GetCurrentPage() Page {
	return pm.currentPage
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
	if pm.currentPage != nil {
		return pm.currentPage.HandleTouch(x, y, touchType)
	}
	return false
}

// Update 更新当前页面
func (pm *PageManager) Update(deltaTime int64) {
	if pm.currentPage != nil {
		pm.currentPage.Update(deltaTime)
	}
}
