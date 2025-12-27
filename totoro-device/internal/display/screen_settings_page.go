package display

import (
	"fmt"
)

// ScreenSettingsPage 屏幕设置（亮度 + 熄屏）
type ScreenSettingsPage struct {
	BasePage
	navBar   *NavBar
	pm       *PageManager
	services *AppServices
	lastErr  string
}

func NewScreenSettingsPage(pm *PageManager) *ScreenSettingsPage {
	p := &ScreenSettingsPage{
		BasePage: BasePage{Name: "screen_settings"},
		pm:       pm,
	}
	p.navBar = NewNavBar("屏幕设置", true, 480)
	p.navBar.SetOnBack(func() { pm.Back() })
	return p
}

func (p *ScreenSettingsPage) SetServices(s *AppServices) {
	p.services = s
}

func (p *ScreenSettingsPage) getOffSeconds() int {
	if p.services == nil || p.services.Config == nil || p.services.Config.System.ScreenOffSeconds == nil {
		return 0
	}
	if *p.services.Config.System.ScreenOffSeconds < 0 {
		return 0
	}
	return *p.services.Config.System.ScreenOffSeconds
}

func (p *ScreenSettingsPage) getBrightness() int {
	if p.services == nil || p.services.Config == nil || p.services.Config.System.Brightness == nil {
		return 100
	}
	v := *p.services.Config.System.Brightness
	if v < 0 {
		return 10
	}
	if v < 10 {
		return 10
	}
	if v > 100 {
		return 100
	}
	return v
}

func (p *ScreenSettingsPage) setOffSeconds(sec int) {
	p.lastErr = ""
	if sec < 0 {
		sec = 0
	}
	if p.services == nil || p.services.Config == nil {
		p.lastErr = "服务未初始化"
		return
	}
	if err := p.services.SetScreenOffSeconds(sec); err != nil {
		p.lastErr = err.Error()
	}
}

func (p *ScreenSettingsPage) setBrightness(percent int) {
	p.lastErr = ""
	if p.services == nil {
		p.lastErr = "服务未初始化"
		return
	}
	if err := p.services.SetSystemBrightness(percent); err != nil {
		p.lastErr = err.Error()
	}
}

func (p *ScreenSettingsPage) Render(g *Graphics) error {
	g.DrawRect(0, 0, 480, 480, ColorBackgroundStart)
	p.navBar.Render(g)

	// 亮度
	_ = g.DrawTextTTF("亮度", 24, 92, ColorTextPrimary, 18, FontWeightMedium)
	_ = g.DrawTextTTF(fmt.Sprintf("%d%%", p.getBrightness()), 24, 122, ColorTextSecondary, 16, FontWeightRegular)

	// 亮度 - / +
	bY := 150
	g.DrawRectRounded(24, bY, 200, 60, 16, ColorPressed)
	_ = g.DrawTextTTF("-", 24+200/2-6, bY+18, ColorTextPrimary, 28, FontWeightMedium)
	g.DrawRectRounded(256, bY, 200, 60, 16, ColorPressed)
	_ = g.DrawTextTTF("+", 256+200/2-8, bY+16, ColorTextPrimary, 28, FontWeightMedium)

	// 快捷亮度
	qY := 220
	drawQ := func(x int, label string) {
		g.DrawRectRounded(x, qY, 136, 50, 14, ColorPressed)
		w := g.MeasureText(label, 16, FontWeightMedium)
		_ = g.DrawTextTTF(label, x+(136-w)/2, qY+16, ColorTextPrimary, 16, FontWeightMedium)
	}
	drawQ(24, "10%")
	drawQ(172, "50%")
	drawQ(320, "100%")

	_ = g.DrawTextTTF("熄屏时间", 24, 290, ColorTextPrimary, 18, FontWeightMedium)
	_ = g.DrawTextTTF(fmt.Sprintf("%d 秒", p.getOffSeconds()), 24, 320, ColorTextSecondary, 16, FontWeightRegular)

	// 快捷选项
	y := 350
	drawBtn := func(x int, label string) {
		g.DrawRectRounded(x, y, 136, 50, 14, ColorPressed)
		w := g.MeasureText(label, 16, FontWeightMedium)
		_ = g.DrawTextTTF(label, x+(136-w)/2, y+16, ColorTextPrimary, 16, FontWeightMedium)
	}
	drawBtn(24, "不熄屏(0)")
	drawBtn(172, "30 秒")
	drawBtn(320, "60 秒")

	if p.lastErr != "" {
		_ = g.DrawTextTTF(p.lastErr, 24, 455-28, ColorErrorRed, 14, FontWeightRegular)
	}
	return nil
}

func (p *ScreenSettingsPage) HandleTouch(x, y int, touchType TouchType) bool {
	if p.navBar.HandleTouch(x, y, touchType) {
		return true
	}
	if touchType != TouchUp {
		return false
	}

	// 亮度 - / +
	if x >= 24 && x <= 224 && y >= 150 && y <= 210 {
		p.setBrightness(p.getBrightness() - 10)
		return true
	}
	if x >= 256 && x <= 456 && y >= 150 && y <= 210 {
		p.setBrightness(p.getBrightness() + 10)
		return true
	}
	// 亮度 quick
	if y >= 220 && y <= 270 {
		if x >= 24 && x <= 160 {
			p.setBrightness(10)
			return true
		}
		if x >= 172 && x <= 308 {
			p.setBrightness(50)
			return true
		}
		if x >= 320 && x <= 456 {
			p.setBrightness(100)
			return true
		}
	}

	if y >= 350 && y <= 400 {
		if x >= 24 && x <= 160 {
			p.setOffSeconds(0)
			return true
		}
		if x >= 172 && x <= 308 {
			p.setOffSeconds(30)
			return true
		}
		if x >= 320 && x <= 456 {
			p.setOffSeconds(60)
			return true
		}
	}
	return false
}
