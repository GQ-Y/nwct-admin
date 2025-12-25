package display

import (
	"image/color"
)

// ListItem 列表项
type ListItem struct {
	Title    string
	Subtitle string
	Icon     string
	OnClick  func()
	Data     interface{}
}

// List 列表组件（鸿蒙风格）
type List struct {
	BaseComponent
	Items         []*ListItem
	ItemHeight    int
	SelectedIndex int
	ScrollOffset  int
}

// NewList 创建列表
func NewList(x, y, w, h int) *List {
	return &List{
		BaseComponent: BaseComponent{
			X: x, Y: y, Width: w, Height: h,
			Background: color.RGBA{0, 0, 0, 0}, // 透明背景
			Visible:    true,
		},
		ItemHeight:    70,
		SelectedIndex: -1,
	}
}

// AddItem 添加列表项
func (l *List) AddItem(item *ListItem) {
	l.Items = append(l.Items, item)
}

// Clear 清空列表
func (l *List) Clear() {
	l.Items = nil
	l.SelectedIndex = -1
	l.ScrollOffset = 0
}

// Render 渲染列表
func (l *List) Render(g interface{}) error {
	gfx := g.(*Graphics)
	
	if !l.Visible {
		return nil
	}

	visibleCount := l.Height / l.ItemHeight
	startIndex := l.ScrollOffset / l.ItemHeight
	if startIndex < 0 {
		startIndex = 0
	}

	for i := startIndex; i < len(l.Items) && i < startIndex+visibleCount+1; i++ {
		itemY := l.Y + i*l.ItemHeight - l.ScrollOffset
		
		// 裁剪超出区域的项
		if itemY+l.ItemHeight < l.Y || itemY > l.Y+l.Height {
			continue
		}

		l.renderItem(gfx, l.Items[i], itemY, i == l.SelectedIndex)
	}

	return nil
}

// renderItem 渲染单个列表项
func (l *List) renderItem(g *Graphics, item *ListItem, y int, selected bool) {
	// 背景
	bgColor := color.RGBA{45, 55, 90, 180}
	if selected {
		bgColor = color.RGBA{75, 123, 236, 200}
	}
	g.DrawRectRounded(l.X, y, l.Width, l.ItemHeight-4, 12, bgColor)

	// 标题
	titleColor := color.RGBA{255, 255, 255, 255}
	if !selected {
		titleColor = color.RGBA{220, 230, 255, 255}
	}
	g.DrawTextTTF(item.Title, l.X+20, y+28, titleColor, 16, FontWeightMedium)

	// 副标题
	if item.Subtitle != "" {
		subtitleColor := color.RGBA{160, 170, 200, 255}
		if selected {
			subtitleColor = color.RGBA{200, 210, 240, 255}
		}
		g.DrawTextTTF(item.Subtitle, l.X+20, y+50, subtitleColor, 12, FontWeightRegular)
	}

	// 箭头指示器
	if item.OnClick != nil {
		arrowColor := color.RGBA{140, 150, 180, 255}
		if selected {
			arrowColor = color.RGBA{255, 255, 255, 255}
		}
		g.DrawText(">", l.X+l.Width-30, y+35, arrowColor, 16)
	}
}

// HandleTouch 处理触摸事件
func (l *List) HandleTouch(x, y int, touchType TouchType) bool {
	if !l.Visible {
		return false
	}

	// 检查触摸点是否在列表区域内
	if x < l.X || x >= l.X+l.Width || y < l.Y || y >= l.Y+l.Height {
		return false
	}

	if touchType == TouchDown {
		// 计算点击的列表项
		relativeY := y - l.Y + l.ScrollOffset
		itemIndex := relativeY / l.ItemHeight

		if itemIndex >= 0 && itemIndex < len(l.Items) {
			l.SelectedIndex = itemIndex
			return true
		}
	} else if touchType == TouchUp {
		if l.SelectedIndex >= 0 && l.SelectedIndex < len(l.Items) {
			item := l.Items[l.SelectedIndex]
			if item.OnClick != nil {
				item.OnClick()
			}
			l.SelectedIndex = -1
			return true
		}
	}

	return false
}

// GetSelectedItem 获取选中的列表项
func (l *List) GetSelectedItem() *ListItem {
	if l.SelectedIndex >= 0 && l.SelectedIndex < len(l.Items) {
		return l.Items[l.SelectedIndex]
	}
	return nil
}

