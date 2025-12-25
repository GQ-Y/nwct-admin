package display

// SettingsPage 设置主页
type SettingsPage struct {
	BasePage
	listView *ListView
	navBar   *NavBar
	pm       *PageManager
}

// NewSettingsPage 创建设置页
func NewSettingsPage(pm *PageManager) *SettingsPage {
	sp := &SettingsPage{
		BasePage: BasePage{Name: "settings"},
		pm:       pm,
	}
	
	// 导航栏
	sp.navBar = NewNavBar("设置", true, 480)
	sp.navBar.SetOnBack(func() {
		pm.Back()
	})
	
	// 列表
	sp.listView = NewListView(0, 60, 480, 420)
	
	// 添加设置项
	sp.listView.AddItem(&ListItem{
		Title:    "网络设置",
		Subtitle: "以太网与 WLAN 配置",
		ShowArrow: true,
		OnClick: func() {
			pm.NavigateTo("network")
		},
	})
	
	sp.listView.AddItem(&ListItem{
		Title:    "隧道管理",
		Subtitle: "查看与编辑传输隧道",
		ShowArrow: true,
		OnClick: func() {
			pm.NavigateTo("tunnel_list")
		},
	})
	
	sp.listView.AddItem(&ListItem{
		Title:    "关于设备",
		Subtitle: "Luckfox Pico Ultra RV1106",
		ShowArrow: true,
		OnClick: func() {
			// TODO: 关于页面
		},
	})
	
	return sp
}

func (p *SettingsPage) OnEnter() {
	// 刷新状态等
}

func (p *SettingsPage) Render(g *Graphics) error {
	// 背景
	g.DrawRect(0, 0, 480, 480, ColorBackgroundStart)
	
	p.listView.Render(g)
	p.navBar.Render(g)
	
	return nil
}

func (p *SettingsPage) HandleTouch(x, y int, touchType TouchType) bool {
	if p.navBar.HandleTouch(x, y, touchType) {
		return true
	}
	if p.listView.HandleTouch(x, y, touchType) {
		return true
	}
	return false
}

func (p *SettingsPage) Update(deltaTime int64) {
	// 可以在这里处理列表动画等
}
