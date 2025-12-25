package display

// TunnelListPage 隧道列表页
type TunnelListPage struct {
	BasePage
	listView *ListView
	navBar   *NavBar
	pm       *PageManager
}

func NewTunnelListPage(pm *PageManager) *TunnelListPage {
	p := &TunnelListPage{
		BasePage: BasePage{Name: "tunnel_list"},
		pm:       pm,
	}
	
	p.navBar = NewNavBar("隧道列表", true, 480)
	p.navBar.SetOnBack(func() { pm.Back() })
	
	p.listView = NewListView(0, 60, 480, 420)
	
	// 模拟隧道数据
	tunnels := []struct{ name string; status string; type_ string }{
		{"SSH Remote", "在线", "TCP"},
		{"Web Service", "在线", "HTTP"},
		{"Database", "离线", "TCP"},
		{"P2P Transfer", "在线", "P2P"},
	}
	
	for _, t := range tunnels {
		statusColor := ColorSuccessGreen
		if t.status == "离线" { statusColor = ColorTextLight }
		
		item := &ListItem{
			Title:    t.name,
			Subtitle: t.type_ + " | " + t.status,
			ShowArrow: true,
			Icon: func(g *Graphics, x, y int) {
				// 状态点
				g.DrawCircle(x, y, 6, statusColor)
			},
		}
		
		tName := t.name
		item.OnClick = func() {
			if ep := pm.GetTunnelEditPage(); ep != nil {
				ep.SetTunnelName(tName)
			}
			pm.NavigateTo("tunnel_edit")
		}
		
		p.listView.AddItem(item)
	}
	
	return p
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

