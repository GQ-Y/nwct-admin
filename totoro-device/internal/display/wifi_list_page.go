package display

import "fmt"

// WiFiListPage WiFi列表页
type WiFiListPage struct {
	BasePage
	listView *ListView
	navBar   *NavBar
	pm       *PageManager
	services *AppServices
	lastErr  string
}

func NewWiFiListPage(pm *PageManager) *WiFiListPage {
	p := &WiFiListPage{
		BasePage: BasePage{Name: "wifi_list"},
		pm:       pm,
	}
	
	p.navBar = NewNavBar("选择 WLAN", true, 480)
	p.navBar.SetOnBack(func() { pm.Back() })
	
	p.listView = NewListView(0, 60, 480, 420)
	p.refresh()
	
	return p
}

func (p *WiFiListPage) SetServices(s *AppServices) {
	p.services = s
	p.refresh()
}

func (p *WiFiListPage) OnEnter() {
	p.refresh()
}

func (p *WiFiListPage) refresh() {
	if p.listView == nil {
		return
	}
	p.listView.Clear()
	p.lastErr = ""

	if p.services == nil {
		p.lastErr = "服务未初始化"
		p.listView.AddItem(&ListItem{Title: "无法扫描 WiFi", Subtitle: p.lastErr})
		return
	}

	nets, err := p.services.ScanWiFi()
	if err != nil {
		p.lastErr = err.Error()
		p.listView.AddItem(&ListItem{Title: "WiFi 扫描失败", Subtitle: p.lastErr})
		return
	}
	if len(nets) == 0 {
		p.listView.AddItem(&ListItem{Title: "未发现可用 WiFi", Subtitle: "请确认 WLAN 已开启"})
		return
	}

	for _, n := range nets {
		sub := "信号: " + fmt.Sprintf("%d%%", n.Signal) + " | " + n.Security
		val := ""
		if n.InUse {
			val = "已连接"
		}
		ssid := n.SSID
		item := &ListItem{
			Title:     ssid,
			Subtitle:  sub,
			Value:     val,
			ShowArrow: true,
			OnClick: func() {
				if cp := p.pm.GetWiFiConnectPage(); cp != nil {
					cp.SetTargetSSID(ssid)
				}
				p.pm.NavigateTo("wifi_connect")
			},
		}
		p.listView.AddItem(item)
	}
}

func (p *WiFiListPage) Render(g *Graphics) error {
	g.DrawRect(0, 0, 480, 480, ColorBackgroundStart)
	p.listView.Render(g)
	p.navBar.Render(g)
	return nil
}

func (p *WiFiListPage) HandleTouch(x, y int, touchType TouchType) bool {
	if p.navBar.HandleTouch(x, y, touchType) { return true }
	return p.listView.HandleTouch(x, y, touchType)
}

