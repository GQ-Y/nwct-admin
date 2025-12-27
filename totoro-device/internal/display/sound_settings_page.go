package display

import (
	"fmt"
	"strconv"
	"strings"
)

// SoundSettingsPage 声音设置（音量 0-30）
type SoundSettingsPage struct {
	BasePage
	navBar   *NavBar
	pm       *PageManager
	services *AppServices

	lastErr string
}

func NewSoundSettingsPage(pm *PageManager) *SoundSettingsPage {
	p := &SoundSettingsPage{
		BasePage: BasePage{Name: "sound_settings"},
		pm:       pm,
	}
	p.navBar = NewNavBar("声音设置", true, 480)
	p.navBar.SetOnBack(func() { pm.Back() })
	return p
}

func (p *SoundSettingsPage) SetServices(s *AppServices) {
	p.services = s
}

func (p *SoundSettingsPage) getVolume() int {
	if p.services == nil || p.services.Config == nil || p.services.Config.System.Volume == nil {
		return 15
	}
	v := *p.services.Config.System.Volume
	if v < 0 {
		return 0
	}
	if v > 30 {
		return 30
	}
	return v
}

func (p *SoundSettingsPage) setVolume(v int) {
	p.lastErr = ""
	if v < 0 {
		v = 0
	}
	if v > 30 {
		v = 30
	}
	if p.services == nil {
		p.lastErr = "服务未初始化"
		return
	}
	if err := p.services.SetSystemVolume(v); err != nil {
		p.lastErr = err.Error()
	}
}

func (p *SoundSettingsPage) Render(g *Graphics) error {
	g.DrawRect(0, 0, 480, 480, ColorBackgroundStart)
	p.navBar.Render(g)

	_ = g.DrawTextTTF("音量", 24, 92, ColorTextPrimary, 18, FontWeightMedium)
	v := p.getVolume()
	_ = g.DrawTextTTF(fmt.Sprintf("%d / 30", v), 24, 122, ColorTextSecondary, 16, FontWeightRegular)

	// - 按钮
	minusX, minusY := 24, 170
	btnW, btnH := 200, 60
	g.DrawRectRounded(minusX, minusY, btnW, btnH, 16, ColorPressed)
	_ = g.DrawTextTTF("-", minusX+btnW/2-6, minusY+18, ColorTextPrimary, 28, FontWeightMedium)

	// + 按钮
	plusX := 256
	g.DrawRectRounded(plusX, minusY, btnW, btnH, 16, ColorPressed)
	_ = g.DrawTextTTF("+", plusX+btnW/2-8, minusY+16, ColorTextPrimary, 28, FontWeightMedium)

	// 快速设置（0/15/30）
	quickY := 250
	drawQuick := func(x int, label string) {
		g.DrawRectRounded(x, quickY, 136, 50, 14, ColorPressed)
		w := g.MeasureText(label, 16, FontWeightMedium)
		_ = g.DrawTextTTF(label, x+(136-w)/2, quickY+16, ColorTextPrimary, 16, FontWeightMedium)
	}
	drawQuick(24, "静音(0)")
	drawQuick(172, "中等(15)")
	drawQuick(320, "最大(30)")

	// 允许查看/编辑 config 值（调试用：显示原始值）
	if p.services != nil && p.services.Config != nil && p.services.Config.System.Volume != nil {
		raw := strconv.Itoa(*p.services.Config.System.Volume)
		_ = g.DrawTextTTF("已保存: "+strings.TrimSpace(raw), 24, 320, ColorTextLight, 12, FontWeightRegular)
	}

	if strings.TrimSpace(p.lastErr) != "" {
		_ = g.DrawTextTTF(p.lastErr, 24, 350, ColorErrorRed, 14, FontWeightRegular)
	}
	return nil
}

func (p *SoundSettingsPage) HandleTouch(x, y int, touchType TouchType) bool {
	if p.navBar.HandleTouch(x, y, touchType) {
		return true
	}
	if touchType != TouchUp {
		return false
	}

	// - / +
	if x >= 24 && x <= 224 && y >= 170 && y <= 230 {
		p.setVolume(p.getVolume() - 1)
		return true
	}
	if x >= 256 && x <= 456 && y >= 170 && y <= 230 {
		p.setVolume(p.getVolume() + 1)
		return true
	}

	// quick
	if y >= 250 && y <= 300 {
		if x >= 24 && x <= 160 {
			p.setVolume(0)
			return true
		}
		if x >= 172 && x <= 308 {
			p.setVolume(15)
			return true
		}
		if x >= 320 && x <= 456 {
			p.setVolume(30)
			return true
		}
	}
	return false
}


