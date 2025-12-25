package display

// NetworkPage 网络设置页
type NetworkPage struct {
	BasePage
	listView *ListView
	navBar   *NavBar
	pm       *PageManager
	services *AppServices
}

func NewNetworkPage(pm *PageManager) *NetworkPage {
	p := &NetworkPage{
		BasePage: BasePage{Name: "network"},
		pm:       pm,
	}
	
	p.navBar = NewNavBar("网络设置", true, 480)
	p.navBar.SetOnBack(func() { pm.Back() })
	
	p.listView = NewListView(0, 60, 480, 420)
	p.refreshList()
	
	return p
}

func (p *NetworkPage) SetServices(s *AppServices) {
	p.services = s
	p.refreshList()
}

func (p *NetworkPage) OnEnter() {
	p.refreshList()
}

func (p *NetworkPage) refreshList() {
	if p.listView == nil {
		return
	}
	p.listView.Clear()

	ethSubtitle := "未连接"
	ethIP := ""

	wlanSubtitle := "未连接"
	wlanIP := ""

	if p.services != nil {
		// 当前网络状态
		if st, err := p.services.GetNetworkStatus(); err == nil && st != nil && st.Status == "connected" {
			// 粗略判断：当前接口名含 wlan/wl 视为 WiFi，否则视为以太网
			ifName := st.CurrentInterface
			if len(ifName) >= 2 && (ifName[:2] == "wl" || (len(ifName) >= 4 && ifName[:4] == "wlan")) {
				wlanSubtitle = "已连接"
				wlanIP = st.IP
			} else {
				ethSubtitle = "已连接"
				ethIP = st.IP
			}
		}

		// 从配置补充 DHCP/静态标识
		if p.services.Config != nil {
			if p.services.Config.Network.IPMode == "static" {
				ethSubtitle = ethSubtitle + " (静态IP)"
			} else if p.services.Config.Network.IPMode != "" {
				ethSubtitle = ethSubtitle + " (DHCP)"
			}
		}
	}

	p.listView.AddItem(&ListItem{
		Title:     "以太网",
		Subtitle:  ethSubtitle,
		Value:     ethIP,
		ShowArrow: true,
		OnClick: func() {
			p.pm.NavigateTo("ethernet")
		},
	})

	p.listView.AddItem(&ListItem{
		Title:     "WLAN",
		Subtitle:  wlanSubtitle,
		Value:     wlanIP,
		ShowArrow: true,
		OnClick: func() {
			p.pm.NavigateTo("wifi_list")
		},
	})
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

