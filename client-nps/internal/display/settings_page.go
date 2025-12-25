package display

import (
	"image/color"
	
)

// SettingsPage 设备设置页
type SettingsPage struct {
	BasePage
	list       *List
	backButton *Button
	onBack     func()
}

// NewSettingsPage 创建设置页
func NewSettingsPage() *SettingsPage {
	page := &SettingsPage{
		BasePage: BasePage{Name: "settings"},
	}

	// 创建列表
	page.list = NewList(30, 80, 420, 320)
	
	// 添加设置项
	page.list.AddItem(&ListItem{
		Title:    "网络设置",
		Subtitle: "以太网、静态IP配置",
		OnClick:  func() { /* TODO: 打开网络设置页 */ },
	})
	
	page.list.AddItem(&ListItem{
		Title:    "WiFi设置",
		Subtitle: "连接WiFi、查看已保存网络",
		OnClick:  func() { /* TODO: 打开WiFi页 */ },
	})
	
	page.list.AddItem(&ListItem{
		Title:    "隧道管理",
		Subtitle: "添加、编辑、删除隧道",
		OnClick:  func() { /* TODO: 打开隧道页 */ },
	})
	
	page.list.AddItem(&ListItem{
		Title:    "系统信息",
		Subtitle: "版本、设备ID、存储空间",
		OnClick:  func() { /* TODO: 打开系统信息页 */ },
	})

	// 创建返回按钮
	page.backButton = NewSecondaryButton(30, 410, 100, 50, "返回")
	page.backButton.OnClick = func() {
		if page.onBack != nil {
			page.onBack()
		}
	}

	return page
}

// SetOnBack 设置返回回调
func (p *SettingsPage) SetOnBack(callback func()) {
	p.onBack = callback
}

// Render 渲染设置页
func (p *SettingsPage) Render(g *Graphics) error {
	// 背景渐变
	colors := []color.Color{
		color.RGBA{18, 20, 38, 255},
		color.RGBA{28, 32, 58, 255},
		color.RGBA{38, 44, 78, 255},
	}
	g.DrawGradient(0, 0, 480, 480, colors, GradientVertical)

	// 顶部标题栏
	g.DrawRect(0, 0, 480, 60, color.RGBA{20, 24, 44, 200})
	g.DrawTextTTF("设置", 40, 40, color.RGBA{255, 255, 255, 255}, 22, FontWeightMedium)

	// 渲染列表
	p.list.Render(interface{}(g))

	// 渲染返回按钮
	p.backButton.Render(interface{}(g))

	return nil
}

// HandleTouch 处理触摸
func (p *SettingsPage) HandleTouch(x, y int, touchType TouchType) bool {
	// 转换类型
	uiTouchType := TouchType(touchType)
	
	if p.backButton.HandleTouch(x, y, uiTouchType) {
		return true
	}
	
	if p.list.HandleTouch(x, y, uiTouchType) {
		return true
	}

	return false
}

