package display

import "image/color"

// AppsPage 应用入口页：隧道管理 / 云平台 / 网络配置 / 系统设置
type AppsPage struct {
	BasePage
	navBar *NavBar
	pm     *PageManager
}

func NewAppsPage(pm *PageManager) *AppsPage {
	p := &AppsPage{
		BasePage: BasePage{Name: "apps"},
		pm:       pm,
	}
	p.navBar = NewNavBar("应用", true, 480)
	p.navBar.SetOnBack(func() { pm.Back() })
	return p
}

func (p *AppsPage) Render(g *Graphics) error {
	g.DrawRect(0, 0, 480, 480, ColorBackgroundStart)

	// 四宫格卡片
	type card struct {
		title string
		sub   string
		x, y  int
		w, h  int
		bg    color.Color
		to    string
	}
	cards := []card{
		{title: "隧道管理", sub: "查看与编辑隧道", x: 24, y: 90, w: 208, h: 150, bg: ColorPressed, to: "tunnel_list"},
		{title: "云平台", sub: "公开节点/邀请码", x: 248, y: 90, w: 208, h: 150, bg: ColorPressed, to: "cloud"},
		{title: "网络配置", sub: "以太网 / WLAN", x: 24, y: 260, w: 208, h: 150, bg: ColorPressed, to: "network"},
		{title: "系统设置", sub: "声音/屏幕/关于", x: 248, y: 260, w: 208, h: 150, bg: ColorPressed, to: "system_settings"},
	}

	for _, c := range cards {
		g.DrawRectRounded(c.x, c.y, c.w, c.h, 18, c.bg)
		_ = g.DrawTextTTF(c.title, c.x+16, c.y+22, ColorTextPrimary, 20, FontWeightMedium)
		_ = g.DrawTextTTF(c.sub, c.x+16, c.y+56, ColorTextSecondary, 14, FontWeightRegular)
		// 右下角箭头
		g.DrawLine(c.x+c.w-28, c.y+c.h-26, c.x+c.w-18, c.y+c.h-18, ColorTextLight)
		g.DrawLine(c.x+c.w-28, c.y+c.h-10, c.x+c.w-18, c.y+c.h-18, ColorTextLight)
	}

	p.navBar.Render(g)
	return nil
}

func (p *AppsPage) HandleTouch(x, y int, touchType TouchType) bool {
	if p.navBar.HandleTouch(x, y, touchType) {
		return true
	}
	if touchType != TouchUp {
		return false
	}

	// 命中四宫格
	if x >= 24 && x <= 232 && y >= 90 && y <= 240 {
		_ = p.pm.NavigateTo("tunnel_list")
		return true
	}
	if x >= 248 && x <= 456 && y >= 90 && y <= 240 {
		_ = p.pm.NavigateTo("cloud")
		return true
	}
	if x >= 24 && x <= 232 && y >= 260 && y <= 410 {
		_ = p.pm.NavigateTo("network")
		return true
	}
	if x >= 248 && x <= 456 && y >= 260 && y <= 410 {
		_ = p.pm.NavigateTo("system_settings")
		return true
	}
	return false
}


