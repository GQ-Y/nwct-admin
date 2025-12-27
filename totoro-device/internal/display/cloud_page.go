package display

import (
	"fmt"
	"strings"
	"time"
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

	lastLatency int
	lastLatencyAt time.Time
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

	// 顶部状态卡片
	cardX, cardY, cardW, cardH := 24, 92, 432, 150
	g.DrawRectRounded(cardX, cardY, cardW, cardH, 18, ColorPressed)
	statusTxt := "未连接"
	statusColor := ColorErrorRed
	if st.Connected {
		statusTxt = "已连接"
		statusColor = ColorSuccessGreen
	}
	_ = g.DrawTextTTF("连接状态", cardX+16, cardY+22, ColorTextPrimary, 16, FontWeightMedium)
	_ = g.DrawTextTTF(statusTxt, cardX+16, cardY+52, statusColor, 22, FontWeightMedium)

	mode := ""
	if p.services != nil && p.services.Config != nil {
		mode = string(p.services.Config.FRPServer.Mode)
	}
	modeCN := map[string]string{
		"builtin": "Totoro 云服务",
		"public":  "公开/私有分享节点",
		"manual":  "手动节点",
	}[strings.TrimSpace(mode)]
	if modeCN == "" {
		modeCN = "未知"
	}
	_ = g.DrawTextTTF("模式: "+modeCN, cardX+16, cardY+84, ColorTextSecondary, 14, FontWeightRegular)

	server := strings.TrimSpace(st.Server)
	if server == "" && p.services.Config != nil {
		server = strings.TrimSpace(p.services.Config.FRPServer.Server)
	}
	if server == "" {
		server = "—"
	}
	_ = g.DrawTextTTF("节点: "+server, cardX+16, cardY+110, ColorTextSecondary, 14, FontWeightRegular)

	// 支持协议（基于当前 Active 能力字段）
	protos := []string{"TCP", "UDP"}
	if p.services.Config != nil && p.services.Config.FRPServer.HTTPEnabled {
		protos = append(protos, "HTTP")
	}
	if p.services.Config != nil && p.services.Config.FRPServer.HTTPSEnabled {
		protos = append(protos, "HTTPS")
	}
	_ = g.DrawTextTTF("支持协议: "+strings.Join(protos, "/"), cardX+16, cardY+136, ColorTextSecondary, 14, FontWeightRegular)

	// 延迟卡片
	latY := 260
	g.DrawRectRounded(24, latY, 432, 90, 18, ColorPressed)
	_ = g.DrawTextTTF("延迟", 40, latY+20, ColorTextPrimary, 16, FontWeightMedium)
	lat := p.latencyText()
	_ = g.DrawTextTTF(lat, 40, latY+52, ColorTextSecondary, 18, FontWeightMedium)

	if st.LastError != "" {
		_ = g.DrawTextTTF("错误: "+st.LastError, 24, 370, ColorErrorRed, 12, FontWeightRegular)
	}
	return nil
}

func (p *CloudStatusPage) HandleTouch(x, y int, touchType TouchType) bool {
	return p.navBar.HandleTouch(x, y, touchType)
}

func (p *CloudStatusPage) Update(deltaTime int64) {
	if p.services == nil || p.services.Config == nil {
		return
	}
	// 每 2 秒刷新一次延迟
	if !p.lastLatencyAt.IsZero() && time.Since(p.lastLatencyAt) < 2*time.Second {
		return
	}
	p.lastLatencyAt = time.Now()

	server := strings.TrimSpace(p.services.Config.FRPServer.Server)
	if server == "" {
		p.lastLatency = -1
		return
	}
	if ms, err := p.services.MeasureLatencyToServer(server); err == nil {
		p.lastLatency = ms
	} else {
		p.lastLatency = -1
	}
}

func (p *CloudStatusPage) latencyText() string {
	if p.lastLatency <= 0 {
		return "—"
	}
	return fmt.Sprintf("%d ms", p.lastLatency)
}


