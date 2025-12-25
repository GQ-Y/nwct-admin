package display

// WiFiConnectPage WiFi连接页
type WiFiConnectPage struct {
	BasePage
	navBar   *NavBar
	pm       *PageManager
	
	targetSSID  string
	pwdInput    *InputField
	keyboard    *VirtualKeyboard
}

func NewWiFiConnectPage(pm *PageManager) *WiFiConnectPage {
	p := &WiFiConnectPage{
		BasePage: BasePage{Name: "wifi_connect"},
		pm:       pm,
	}
	
	p.navBar = NewNavBar("输入密码", true, 480)
	p.navBar.SetOnBack(func() { pm.Back() })
	
	p.pwdInput = NewInputField(24, 120, 432, 50)
	p.pwdInput.placeholder = "请输入密码"
	p.pwdInput.isPassword = true
	
	p.keyboard = NewVirtualKeyboard(480-240, 480, 240)
	p.keyboard.onEnter = func() {
		p.keyboard.Hide()
		// 模拟连接...
		pm.Back() // 连接成功返回
	}
	
	return p
}

func (p *WiFiConnectPage) SetTargetSSID(ssid string) {
	p.targetSSID = ssid
	p.pwdInput.SetText("") // 清空密码
	// 更新标题
	// p.navBar.title = ssid (需要 NavBar 支持修改 Title)
}

func (p *WiFiConnectPage) Render(g *Graphics) error {
	g.DrawRect(0, 0, 480, 480, ColorBackgroundStart)
	
	_ = g.DrawTextTTF("正在连接到: "+p.targetSSID, 24, 90, ColorTextPrimary, 18, FontWeightMedium)
	
	p.pwdInput.Render(g)
	
	// 连接按钮
	btnY := 220
	g.DrawRectRounded(24, btnY, 432, 50, 25, ColorBrandBlue)
	labelW := g.MeasureText("加入网络", 18, FontWeightMedium)
	textTop := btnY + (50-int(18))/2
	_ = g.DrawTextTTF("加入网络", 24+(432-labelW)/2, textTop, ColorBackgroundStart, 18, FontWeightMedium)
	
	p.navBar.Render(g)
	p.keyboard.Render(g)
	return nil
}

func (p *WiFiConnectPage) HandleTouch(x, y int, touchType TouchType) bool {
	if p.keyboard.isVisible {
		return p.keyboard.HandleTouch(x, y, touchType)
	}
	
	if p.navBar.HandleTouch(x, y, touchType) { return true }
	
	if p.pwdInput.HandleTouch(x, y, touchType) {
		if p.pwdInput.isFocused {
			p.keyboard.Show(p.pwdInput)
		}
		return true
	}
	
	// 按钮点击
	if x > 24 && x < 456 && y > 220 && y < 270 {
		if touchType == TouchUp {
			p.pm.Back()
		}
		return true
	}
	
	return false
}

