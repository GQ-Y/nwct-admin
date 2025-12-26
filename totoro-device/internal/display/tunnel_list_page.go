package display

import (
	"fmt"
	"strings"
)

// TunnelListPage 隧道列表页
type TunnelListPage struct {
	BasePage
	listView *ListView
	navBar   *NavBar
	pm       *PageManager
	services *AppServices
	lastErr  string
}

func NewTunnelListPage(pm *PageManager) *TunnelListPage {
	p := &TunnelListPage{
		BasePage: BasePage{Name: "tunnel_list"},
		pm:       pm,
	}
	
	p.navBar = NewNavBar("隧道列表", true, 480)
	p.navBar.SetOnBack(func() { pm.Back() })
	
	p.listView = NewListView(0, 60, 480, 420)
	p.refresh()
	
	return p
}

func (p *TunnelListPage) SetServices(s *AppServices) {
	p.services = s
	p.refresh()
}

func (p *TunnelListPage) OnEnter() {
	p.refresh()
}

func (p *TunnelListPage) refresh() {
	if p.listView == nil {
		return
	}
	p.listView.Clear()
	p.lastErr = ""

	// 新增入口（始终显示）
	p.listView.AddItem(&ListItem{
		Title:     "新增隧道",
		Subtitle:  "创建新的内网穿透隧道",
		ShowArrow: true,
		Icon: func(g *Graphics, x, y int) {
			// 简单 + 号
			g.DrawLine(x-6, y, x+6, y, ColorBrandBlue)
			g.DrawLine(x, y-6, x, y+6, ColorBrandBlue)
		},
		OnClick: func() {
			if ep := p.pm.GetTunnelEditPage(); ep != nil {
				ep.BeginCreate()
			}
			p.pm.NavigateTo("tunnel_edit")
		},
	})

	if p.services == nil {
		p.listView.AddItem(&ListItem{Title: "隧道服务未初始化", Subtitle: "请先启动 FRP 客户端"})
		return
	}
	tunnels, err := p.services.GetTunnels()
	if err != nil {
		p.lastErr = err.Error()
		p.listView.AddItem(&ListItem{Title: "读取隧道失败", Subtitle: p.lastErr})
		return
	}
	if len(tunnels) == 0 {
		p.listView.AddItem(&ListItem{Title: "暂无隧道", Subtitle: "点击上方“新增隧道”创建"})
		return
	}

	connected := false
	if p.services.FRP != nil {
		connected = p.services.FRP.IsConnected()
	}
	for _, t := range tunnels {
		if t == nil {
			continue
		}
		statusColor := ColorTextLight
		statusText := "离线"
		if connected {
			statusColor = ColorSuccessGreen
			statusText = "在线"
		}
		title := t.Name
		typeLabel := strings.ToUpper(strings.TrimSpace(t.Type))
		if typeLabel == "" {
			typeLabel = "TCP"
		}
		sub := typeLabel + " | " + statusText + " | "
		// HTTP/HTTPS 优先展示域名
		if strings.EqualFold(t.Type, "http") || strings.EqualFold(t.Type, "https") {
			d := strings.TrimSpace(t.Domain)
			if d == "" {
				d = "（未设置域名，将自动生成）"
			}
			sub += fmt.Sprintf("%s:%d → %s", t.LocalIP, t.LocalPort, d)
		} else {
			sub += fmt.Sprintf("%s:%d → :%d", t.LocalIP, t.LocalPort, t.RemotePort)
		}

		tunnelCopy := t // capture
		item := &ListItem{
			Title:     title,
			Subtitle:  sub,
			ShowArrow: true,
			Icon: func(g *Graphics, x, y int) {
				g.DrawCircle(x, y, 6, statusColor)
			},
			OnClick: func() {
				if ep := p.pm.GetTunnelEditPage(); ep != nil {
					ep.SetTunnel(tunnelCopy)
				}
				p.pm.NavigateTo("tunnel_edit")
			},
		}
		p.listView.AddItem(item)
	}
}

func (p *TunnelListPage) Render(g *Graphics) error {
	g.DrawRect(0, 0, 480, 480, ColorBackgroundStart)
	p.listView.Render(g)
	p.navBar.Render(g)
	return nil
}

func (p *TunnelListPage) HandleTouch(x, y int, touchType TouchType) bool {
	if p.navBar.HandleTouch(x, y, touchType) { return true }
	return p.listView.HandleTouch(x, y, touchType)
}

