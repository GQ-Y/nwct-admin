package display

import (
	"image/color"
)



// Component 组件接口
type Component interface {
	Render(g Graphics) error
	HandleTouch(x, y int, touchType TouchType) bool
	GetBounds() (x, y, w, h int)
	SetVisible(visible bool)
	IsVisible() bool
}

// BaseComponent 组件基类
type BaseComponent struct {
	X, Y, Width, Height int
	Background          color.Color
	Visible             bool
}

// GetBounds 获取边界
func (c *BaseComponent) GetBounds() (x, y, w, h int) {
	return c.X, c.Y, c.Width, c.Height
}

// SetVisible 设置可见性
func (c *BaseComponent) SetVisible(visible bool) {
	c.Visible = visible
}

// IsVisible 是否可见
func (c *BaseComponent) IsVisible() bool {
	return c.Visible
}

// HandleTouch 默认触摸处理
func (c *BaseComponent) HandleTouch(x, y int, touchType TouchType) bool {
	return false
}

