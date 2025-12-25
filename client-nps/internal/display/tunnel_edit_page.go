package display

// TunnelEditPage 隧道详情/编辑页
type TunnelEditPage struct {
	BasePage
	navBar     *NavBar
	pm         *PageManager
	tunnelName string
}

func NewTunnelEditPage(pm *PageManager) *TunnelEditPage {
	p := &TunnelEditPage{
		BasePage: BasePage{Name: "tunnel_edit"},
		pm:       pm,
	}
	p.navBar = NewNavBar("编辑隧道", true, 480)
	p.navBar.SetOnBack(func() { pm.Back() })
	return p
}

func (p *TunnelEditPage) SetTunnelName(name string) {
	p.tunnelName = name
}

func (p *TunnelEditPage) Render(g *Graphics) error {
	g.DrawRect(0, 0, 480, 480, ColorBackgroundStart)
	
	// 详情信息
	startY := 80
	lineH := 40
	
	drawInfo := func(label, value string, y int) {
		g.DrawTextTTF(label, 24, y, ColorTextSecondary, 16, FontWeightRegular)
		g.DrawTextTTF(value, 120, y, ColorTextPrimary, 16, FontWeightMedium)
	}
	
	drawInfo("名称", p.tunnelName, startY)
	drawInfo("类型", "TCP 代理", startY+lineH)
	drawInfo("本地", "127.0.0.1:8080", startY+lineH*2)
	drawInfo("远程", "server.com:8024", startY+lineH*3)
	
	// 删除按钮 (红色)
	btnY := 300
	g.DrawRectRounded(24, btnY, 432, 50, 25, ColorErrorRed)
	labelW := g.MeasureText("删除此隧道", 18, FontWeightMedium)
	textTop := btnY + (50-int(18))/2
	_ = g.DrawTextTTF("删除此隧道", 24+(432-labelW)/2, textTop, ColorBackgroundStart, 18, FontWeightMedium)
	
	p.navBar.Render(g)
	return nil
}

func (p *TunnelEditPage) HandleTouch(x, y int, touchType TouchType) bool {
	if p.navBar.HandleTouch(x, y, touchType) { return true }
	
	// 删除按钮
	if x > 24 && x < 456 && y > 300 && y < 350 {
		if touchType == TouchUp {
			// TODO: 删除逻辑
			p.pm.Back()
		}
		return true
	}
	
	return false
}

