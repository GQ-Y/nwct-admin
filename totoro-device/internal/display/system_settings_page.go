package display

// SystemSettingsPage 系统设置入口：声音/屏幕/关于
type SystemSettingsPage struct {
	BasePage
	listView  *ListView
	navBar    *NavBar
	pm        *PageManager
	services  *AppServices
}

func NewSystemSettingsPage(pm *PageManager) *SystemSettingsPage {
	p := &SystemSettingsPage{
		BasePage: BasePage{Name: "system_settings"},
		pm:       pm,
	}
	p.navBar = NewNavBar("系统设置", true, 480)
	p.navBar.SetOnBack(func() { pm.Back() })
	p.listView = NewListView(0, 60, 480, 420)
	p.refresh()
	return p
}

func (p *SystemSettingsPage) SetServices(s *AppServices) {
	p.services = s
	p.refresh()
}

func (p *SystemSettingsPage) refresh() {
	if p.listView == nil {
		return
	}
	p.listView.Clear()

	p.listView.AddItem(&ListItem{
		Title:     "声音设置",
		Subtitle:  "音量大小",
		ShowArrow: true,
		OnClick: func() {
			_ = p.pm.NavigateTo("sound_settings")
		},
	})
	p.listView.AddItem(&ListItem{
		Title:     "屏幕设置",
		Subtitle:  "亮度与熄屏时间",
		ShowArrow: true,
		OnClick: func() {
			_ = p.pm.NavigateTo("screen_settings")
		},
	})
	p.listView.AddItem(&ListItem{
		Title:     "关于设备",
		Subtitle:  "设备信息与版本",
		ShowArrow: true,
		OnClick: func() {
			_ = p.pm.NavigateTo("about")
		},
	})
}

func (p *SystemSettingsPage) Render(g *Graphics) error {
	g.DrawRect(0, 0, 480, 480, ColorBackgroundStart)
	p.listView.Render(g)
	p.navBar.Render(g)
	return nil
}

func (p *SystemSettingsPage) HandleTouch(x, y int, touchType TouchType) bool {
	if p.navBar.HandleTouch(x, y, touchType) {
		return true
	}
	return p.listView.HandleTouch(x, y, touchType)
}


