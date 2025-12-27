package display

import (
	"fmt"
)

// CloudPage 云平台入口：Totoro 云 / 公开节点 / 私有邀请码
type CloudPage struct {
	BasePage
	navBar   *NavBar
	pm       *PageManager
	services *AppServices
	lastErr  string
}

func NewCloudPage(pm *PageManager) *CloudPage {
	p := &CloudPage{
		BasePage: BasePage{Name: "cloud"},
		pm:       pm,
	}
	p.navBar = NewNavBar("云平台", true, 480)
	p.navBar.SetOnBack(func() { pm.Back() })
	return p
}

func (p *CloudPage) SetServices(s *AppServices) {
	p.services = s
}

func (p *CloudPage) Render(g *Graphics) error {
	g.DrawRect(0, 0, 480, 480, ColorBackgroundStart)
	p.navBar.Render(g)

	// 当前 FRP 状态概览
	lineY := 86
	if p.services != nil {
		if st, err := p.services.GetFRPStatus(); err == nil && st != nil {
			status := "未连接"
			if st.Connected {
				status = "已连接"
			}
			_ = g.DrawTextTTF("穿透状态: "+status, 24, lineY, ColorTextSecondary, 14, FontWeightRegular)
		}
	}

	// 三张卡片
	drawCard := func(x, y int, title, sub string) {
		g.DrawRectRounded(x, y, 432, 90, 18, ColorPressed)
		_ = g.DrawTextTTF(title, x+16, y+18, ColorTextPrimary, 18, FontWeightMedium)
		_ = g.DrawTextTTF(sub, x+16, y+46, ColorTextSecondary, 14, FontWeightRegular)
		// 右侧箭头
		g.DrawLine(x+432-26, y+45-8, x+432-18, y+45, ColorTextLight)
		g.DrawLine(x+432-26, y+45+8, x+432-18, y+45, ColorTextLight)
	}
	drawCard(24, 120, "Totoro 云服务", "官方内置节点 / 服务状态")
	drawCard(24, 230, "公开节点", "公开节点列表与连接")
	drawCard(24, 340, "私有分享云节点", "输入邀请码获取节点")

	if p.lastErr != "" {
		_ = g.DrawTextTTF(p.lastErr, 24, 455-28, ColorErrorRed, 14, FontWeightRegular)
	}
	return nil
}

func (p *CloudPage) HandleTouch(x, y int, touchType TouchType) bool {
	if p.navBar.HandleTouch(x, y, touchType) {
		return true
	}
	if touchType != TouchUp {
		return false
	}
	// 卡片点击区域
	if x >= 24 && x <= 456 {
		switch {
		case y >= 120 && y <= 210:
			_ = p.pm.NavigateTo("cloud_status")
			return true
		case y >= 230 && y <= 320:
			_ = p.pm.NavigateTo("cloud_public_nodes")
			return true
		case y >= 340 && y <= 430:
			_ = p.pm.NavigateTo("cloud_invite")
			return true
		}
	}
	return false
}

// CloudStatusPage 简单展示当前 FRP 状态（MVP）
type CloudStatusPage struct {
	BasePage
	navBar   *NavBar
	pm       *PageManager
	services *AppServices
}

func NewCloudStatusPage(pm *PageManager) *CloudStatusPage {
	p := &CloudStatusPage{
		BasePage: BasePage{Name: "cloud_status"},
		pm:       pm,
	}
	p.navBar = NewNavBar("连接状态", true, 480)
	p.navBar.SetOnBack(func() { pm.Back() })
	return p
}

func (p *CloudStatusPage) SetServices(s *AppServices) { p.services = s }

func (p *CloudStatusPage) Render(g *Graphics) error {
	g.DrawRect(0, 0, 480, 480, ColorBackgroundStart)
	p.navBar.Render(g)
	if p.services == nil {
		_ = g.DrawTextTTF("服务未初始化", 24, 92, ColorErrorRed, 16, FontWeightRegular)
		return nil
	}
	st, err := p.services.GetFRPStatus()
	if err != nil || st == nil {
		_ = g.DrawTextTTF("无法获取状态: "+fmt.Sprintf("%v", err), 24, 92, ColorErrorRed, 14, FontWeightRegular)
		return nil
	}
	_ = g.DrawTextTTF(fmt.Sprintf("Connected: %v", st.Connected), 24, 92, ColorTextPrimary, 16, FontWeightRegular)
	_ = g.DrawTextTTF("Server: "+st.Server, 24, 120, ColorTextSecondary, 14, FontWeightRegular)
	_ = g.DrawTextTTF("PID: "+fmt.Sprintf("%d", st.PID), 24, 146, ColorTextSecondary, 14, FontWeightRegular)
	if st.LastError != "" {
		_ = g.DrawTextTTF("LastError: "+st.LastError, 24, 174, ColorErrorRed, 12, FontWeightRegular)
	}
	return nil
}

func (p *CloudStatusPage) HandleTouch(x, y int, touchType TouchType) bool {
	return p.navBar.HandleTouch(x, y, touchType)
}


