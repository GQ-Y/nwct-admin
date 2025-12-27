package display

import (
	"fmt"
)

// ScreenSettingsPage 屏幕设置（熄屏时间 MVP：先落盘，后续接入亮度/真正背光控制）
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

func (p *ScreenSettingsPage) setOffSeconds(sec int) {
	p.lastErr = ""
	if sec < 0 {
		sec = 0
	}
	if p.services == nil || p.services.Config == nil {
		p.lastErr = "服务未初始化"
		return
	}
	p.services.mu.Lock()
	p.services.Config.System.ScreenOffSeconds = &sec
	_ = p.services.Config.Save()
	p.services.mu.Unlock()
}

func (p *ScreenSettingsPage) Render(g *Graphics) error {
	g.DrawRect(0, 0, 480, 480, ColorBackgroundStart)
	p.navBar.Render(g)

	_ = g.DrawTextTTF("熄屏时间", 24, 92, ColorTextPrimary, 18, FontWeightMedium)
	_ = g.DrawTextTTF(fmt.Sprintf("%d 秒", p.getOffSeconds()), 24, 122, ColorTextSecondary, 16, FontWeightRegular)

	// 快捷选项
	y := 170
	drawBtn := func(x int, label string) {
		g.DrawRectRounded(x, y, 136, 50, 14, ColorPressed)
		w := g.MeasureText(label, 16, FontWeightMedium)
		_ = g.DrawTextTTF(label, x+(136-w)/2, y+16, ColorTextPrimary, 16, FontWeightMedium)
	}
	drawBtn(24, "不熄屏(0)")
	drawBtn(172, "30 秒")
	drawBtn(320, "60 秒")

	// 提示：亮度后续接入（不同屏幕背光实现差异大）
	_ = g.DrawTextTTF("亮度：待接入（取决于屏幕背光接口）", 24, 250, ColorTextLight, 12, FontWeightRegular)

	if p.lastErr != "" {
		_ = g.DrawTextTTF(p.lastErr, 24, 280, ColorErrorRed, 14, FontWeightRegular)
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
	if y >= 170 && y <= 220 {
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


