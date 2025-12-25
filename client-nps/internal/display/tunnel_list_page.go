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
		p.listView.AddItem(&ListItem{Title: "暂无隧道", Subtitle: "请在主程序或后续页面添加隧道"})
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
		sub := strings.ToUpper(t.Type) + " | " + statusText + " | " + fmt.Sprintf("%s:%d → :%d", t.LocalIP, t.LocalPort, t.RemotePort)

		tt := t // capture
		item := &ListItem{
			Title:     title,
			Subtitle:  sub,
			ShowArrow: true,
			Icon: func(g *Graphics, x, y int) {
				g.DrawCircle(x, y, 6, statusColor)
			},
			OnClick: func() {
				if ep := p.pm.GetTunnelEditPage(); ep != nil {
					ep.SetTunnel(tt)
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

