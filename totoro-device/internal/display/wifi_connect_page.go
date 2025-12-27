package display

// WiFiConnectPage WiFi连接页
type WiFiConnectPage struct {
	BasePage
	navBar *NavBar
	pm     *PageManager

	targetSSID string
	pwdInput   *InputField
	keyboard   *VirtualKeyboard

	services *AppServices
	lastErr  string

	showForget bool
	showPassword bool
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
		// 触发连接
		_ = p.connect()
	}

	return p
}

func (p *WiFiConnectPage) SetServices(s *AppServices) {
	p.services = s
}

func (p *WiFiConnectPage) SetTargetSSID(ssid string) {
	p.targetSSID = ssid
	p.pwdInput.SetText("") // 默认清空（若已保存则回填）
	p.lastErr = ""
	p.showForget = false
	p.showPassword = false
	p.pwdInput.isPassword = true
	if p.services != nil && p.services.Config != nil {
		for _, it := range p.services.Config.Network.WiFiProfiles {
			if it.SSID == ssid {
				p.showForget = true
				// 已保存：回填密码（支持查看/编辑）
				p.pwdInput.SetText(it.Password)
				break
			}
		}
	}
	// 更新标题
	// p.navBar.title = ssid (需要 NavBar 支持修改 Title)
}

func (p *WiFiConnectPage) Render(g *Graphics) error {
	g.DrawRect(0, 0, 480, 480, ColorBackgroundStart)

	_ = g.DrawTextTTF("正在连接到: "+p.targetSSID, 24, 90, ColorTextPrimary, 18, FontWeightMedium)

	p.pwdInput.Render(g)

	// 显示/隐藏密码（仅在有内容时显示）
	if p.pwdInput.GetText() != "" {
		label := "显示密码"
		if p.showPassword {
			label = "隐藏密码"
		}
		w := g.MeasureText(label, 14, FontWeightRegular)
		_ = g.DrawTextTTF(label, 456-w, 175, ColorTextSecondary, 14, FontWeightRegular)
	}

	if p.lastErr != "" {
		_ = g.DrawTextTTF(p.lastErr, 24, 195, ColorErrorRed, 14, FontWeightRegular)
	}

	// 连接按钮
	btnY := 220
	g.DrawRectRounded(24, btnY, 432, 50, 25, ColorBrandBlue)
	labelW := g.MeasureText("加入网络", 18, FontWeightMedium)
	textTop := btnY + (50-int(18))/2
	_ = g.DrawTextTTF("加入网络", 24+(432-labelW)/2, textTop, ColorBackgroundStart, 18, FontWeightMedium)

	// 忘记网络按钮（仅当已保存/可忘记）
	if p.showForget {
		fy := 280
		g.DrawRectRounded(24, fy, 432, 50, 25, ColorPressed)
		fw := g.MeasureText("忘记此网络", 16, FontWeightMedium)
		_ = g.DrawTextTTF("忘记此网络", 24+(432-fw)/2, fy+(50-int(16))/2, ColorErrorRed, 16, FontWeightMedium)
	}

	p.navBar.Render(g)
	p.keyboard.Render(g)
	return nil
}

func (p *WiFiConnectPage) connect() error {
	p.lastErr = ""
	if p.services == nil {
		p.lastErr = "服务未初始化"
		return nil
	}
	if err := p.services.ConnectWiFi(p.targetSSID, p.pwdInput.GetText()); err != nil {
		p.lastErr = err.Error()
		return err
	}
	p.pm.Back()
	return nil
}

func (p *WiFiConnectPage) HandleTouch(x, y int, touchType TouchType) bool {
	if p.keyboard.isVisible {
		return p.keyboard.HandleTouch(x, y, touchType)
	}

	if p.navBar.HandleTouch(x, y, touchType) {
		return true
	}

	if p.pwdInput.HandleTouch(x, y, touchType) {
		if p.pwdInput.isFocused {
			p.keyboard.Show(p.pwdInput)
		}
		return true
	}

	// 显示/隐藏密码按钮（右上角文字区域）
	if p.pwdInput.GetText() != "" && x >= 320 && x <= 456 && y >= 168 && y <= 192 {
		if touchType == TouchUp {
			p.showPassword = !p.showPassword
			p.pwdInput.isPassword = !p.showPassword
		}
		return true
	}

	// 按钮点击
	if x > 24 && x < 456 && y > 220 && y < 270 {
		if touchType == TouchUp {
			_ = p.connect()
		}
		return true
	}

	// 忘记按钮
	if p.showForget && x > 24 && x < 456 && y > 280 && y < 330 {
		if touchType == TouchUp {
			p.lastErr = ""
			if p.services == nil {
				p.lastErr = "服务未初始化"
				return true
			}
			if err := p.services.ForgetWiFi(p.targetSSID); err != nil {
				p.lastErr = err.Error()
				return true
			}
			p.pwdInput.SetText("")
			p.showForget = false
			p.showPassword = false
			p.pwdInput.isPassword = true
			p.pm.Back()
		}
		return true
	}

	return false
}
