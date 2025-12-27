package display

import "totoro-device/config"

// AboutPage 关于设备详情页（鸿蒙浅色风格）
type AboutPage struct {
	BasePage
	navBar *NavBar
	pm     *PageManager
}

func NewAboutPage(pm *PageManager) *AboutPage {
	p := &AboutPage{
		BasePage: BasePage{Name: "about"},
		pm:       pm,
	}

	p.navBar = NewNavBar("关于设备", true, 480)
	p.navBar.SetOnBack(func() { pm.Back() })

	return p
}

func (p *AboutPage) Render(g *Graphics) error {
	// 背景（浅色）
	g.DrawRect(0, 0, 480, 480, ColorBackgroundStart)

	// 顶部导航栏
	p.navBar.Render(g)

	// 设备主标题（品牌）
	brand := config.DefaultDeviceName
	brandSize := 30.0
	brandTop := textTopForCenter(80, 48, brandSize)
	_ = g.DrawTextTTF(brand, 24, brandTop, ColorTextPrimary, brandSize, FontWeightMedium)

	// 分隔线
	g.DrawRect(24, 140, 432, 1, ColorSeparator)

	// 信息区（两列：Label / Value）
	type kv struct {
		k string
		v string
	}
	items := []kv{
		{"设备品牌", config.DefaultDeviceName},
		{"作者", "Hook"},
		{"邮箱", "1959595510@qq.com"},
		{"固件版本", "20251225231"},
	}

	startY := 160
	rowH := 56
	labelX := 24
	valueX := 140

	for i, it := range items {
		y := startY + i*rowH
		// label
		_ = g.DrawTextTTF(it.k, labelX, textTopForCenter(y, rowH, 16), ColorTextSecondary, 16, FontWeightRegular)
		// value
		_ = g.DrawTextTTF(it.v, valueX, textTopForCenter(y, rowH, 18), ColorTextPrimary, 18, FontWeightMedium)
		// separator
		if i != len(items)-1 {
			g.DrawRect(24, y+rowH-1, 432, 1, ColorSeparator)
		}
	}

	// 底部提示
	_ = g.DrawTextTTF("© Hook", 24, textTopForCenter(430, 40, 14), ColorTextLight, 14, FontWeightRegular)

	return nil
}

func (p *AboutPage) HandleTouch(x, y int, touchType TouchType) bool {
	return p.navBar.HandleTouch(x, y, touchType)
}
