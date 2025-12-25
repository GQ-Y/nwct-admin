package display

// WiFiListPage WiFi列表页
type WiFiListPage struct {
	BasePage
	listView *ListView
	navBar   *NavBar
	pm       *PageManager
}

func NewWiFiListPage(pm *PageManager) *WiFiListPage {
	p := &WiFiListPage{
		BasePage: BasePage{Name: "wifi_list"},
		pm:       pm,
	}
	
	p.navBar = NewNavBar("选择 WLAN", true, 480)
	p.navBar.SetOnBack(func() { pm.Back() })
	
	p.listView = NewListView(0, 60, 480, 420)
	
	// 模拟数据
	wifis := []struct{ ssid string; signal string; locked bool }{
		{"ChinaNet-Home", "强", true},
		{"Office-5G", "强", true},
		{"Guest-WiFi", "中", false},
		{"TP-LINK_8888", "弱", true},
	}
	
	for _, wifi := range wifis {
		item := &ListItem{
			Title:    wifi.ssid,
			Subtitle: "信号: " + wifi.signal,
			ShowArrow: true,
			// Icon: ... 锁图标
		}
		
		// 闭包捕获
		ssid := wifi.ssid
		item.OnClick = func() {
			// 跳转到连接页 (带参数传递通常通过设置目标页面的属性)
			// 这里简单处理：假设 ConnectPage 有一个 SetTargetSSID 方法
			if cp := pm.GetWiFiConnectPage(); cp != nil {
				cp.SetTargetSSID(ssid)
			}
			pm.NavigateTo("wifi_connect")
		}
		
		p.listView.AddItem(item)
	}
	
	return p
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

