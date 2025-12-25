package ui

import (
	"image/color"
	"nwct/client-nps/internal/display"
)

// Darken 颜色变暗
func Darken(c color.Color, factor float64) color.Color {
	r, g, b, a := c.RGBA()
	return color.RGBA{
		uint8(float64(r>>8) * (1 - factor)),
		uint8(float64(g>>8) * (1 - factor)),
		uint8(float64(b>>8) * (1 - factor)),
		uint8(a >> 8),
	}
}

// Button 按钮组件（鸿蒙风格）
type Button struct {
	BaseComponent
	Text      string
	TextColor color.Color
	OnClick   func()
	Pressed   bool
	Disabled  bool
	Radius    int
}

// NewButton 创建按钮
func NewButton(x, y, w, h int, text string) *Button {
	return &Button{
		BaseComponent: BaseComponent{
			X: x, Y: y, Width: w, Height: h,
			Background: color.RGBA{75, 123, 236, 255}, // 鸿蒙蓝
			Visible:    true,
		},
		Text:      text,
		TextColor: color.White,
		Radius:    12,
	}
}

// NewSecondaryButton 创建次要按钮
func NewSecondaryButton(x, y, w, h int, text string) *Button {
	return &Button{
		BaseComponent: BaseComponent{
			X: x, Y: y, Width: w, Height: h,
			Background: color.RGBA{55, 65, 100, 200},
			Visible:    true,
		},
		Text:      text,
		TextColor: color.RGBA{200, 210, 240, 255},
		Radius:    12,
	}
}

// Render 渲染按钮
func (b *Button) Render(g Graphics) error {
	gfx := g.(*display.Graphics)
	
	if !b.Visible {
		return nil
	}

	// 计算背景色
	bgColor := b.Background
	if b.Disabled {
		bgColor = color.RGBA{60, 70, 100, 100}
	} else if b.Pressed {
		bgColor = Darken(bgColor, 0.2)
	}

	// 绘制按钮背景（圆角矩形）
	gfx.DrawRectRounded(b.X, b.Y, b.Width, b.Height, b.Radius, bgColor)

	// 绘制文本（居中）
	textWidth := gfx.MeasureText(b.Text, 16, display.FontWeightMedium)
	textX := b.X + (b.Width-textWidth)/2
	textY := b.Y + b.Height/2 + 6

	textColor := b.TextColor
	if b.Disabled {
		textColor = color.RGBA{120, 130, 160, 255}
	}

	gfx.DrawTextTTF(b.Text, textX, textY, textColor, 16, display.FontWeightMedium)

	return nil
}

// HandleTouch 处理触摸事件
func (b *Button) HandleTouch(x, y int, touchType TouchType) bool {
	if !b.Visible || b.Disabled {
		return false
	}

	// 检查触摸点是否在按钮内
	if x >= b.X && x < b.X+b.Width && y >= b.Y && y < b.Y+b.Height {
		switch touchType {
		case TouchDown:
			b.Pressed = true
			return true
		case TouchUp:
			if b.Pressed {
				b.Pressed = false
				if b.OnClick != nil {
					b.OnClick()
				}
				return true
			}
		case TouchMove:
			// 如果移出按钮区域，取消按下状态
			if b.Pressed {
				return true
			}
		}
	} else {
		// 触摸点在按钮外，取消按下状态
		if b.Pressed {
			b.Pressed = false
		}
	}

	return false
}

// SetEnabled 设置启用状态
func (b *Button) SetEnabled(enabled bool) {
	b.Disabled = !enabled
}

// SetText 设置文本
func (b *Button) SetText(text string) {
	b.Text = text
}

