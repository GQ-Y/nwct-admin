package display

// NetworkPage 网络设置页
type NetworkPage struct {
	BasePage
	listView *ListView
	navBar   *NavBar
	pm       *PageManager
}

func NewNetworkPage(pm *PageManager) *NetworkPage {
	p := &NetworkPage{
		BasePage: BasePage{Name: "network"},
		pm:       pm,
	}
	
	p.navBar = NewNavBar("网络设置", true, 480)
	p.navBar.SetOnBack(func() { pm.Back() })
	
	p.listView = NewListView(0, 60, 480, 420)
	
	p.listView.AddItem(&ListItem{
		Title:    "以太网",
		Subtitle: "已连接 (DHCP)",
		Value:    "192.168.1.100",
		ShowArrow: true,
		OnClick: func() {
			pm.NavigateTo("ethernet")
		},
	})
	
	p.listView.AddItem(&ListItem{
		Title:    "WLAN",
		Subtitle: "未连接",
		ShowArrow: true,
		OnClick: func() {
			pm.NavigateTo("wifi_list")
		},
	})
	
	return p
}

func (p *NetworkPage) Render(g *Graphics) error {
	g.DrawRect(0, 0, 480, 480, ColorBackgroundStart)
	p.listView.Render(g)
	p.navBar.Render(g)
	return nil
}

func (p *NetworkPage) HandleTouch(x, y int, touchType TouchType) bool {
	if p.navBar.HandleTouch(x, y, touchType) { return true }
	return p.listView.HandleTouch(x, y, touchType)
}

