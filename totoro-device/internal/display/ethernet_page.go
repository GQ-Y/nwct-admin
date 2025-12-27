package display

// EthernetPage 以太网设置页
type EthernetPage struct {
	BasePage
	navBar *NavBar
	pm     *PageManager

	ipInput      *InputField
	maskInput    *InputField
	gatewayInput *InputField
	dnsInput     *InputField

	keyboard   *VirtualKeyboard
	saveButton *ListView // 借用 ListView 实现按钮效果，或者直接画

	services *AppServices
	iface    string
	lastErr  string
}

func NewEthernetPage(pm *PageManager) *EthernetPage {
	p := &EthernetPage{
		BasePage: BasePage{Name: "ethernet"},
		pm:       pm,
	}

	p.navBar = NewNavBar("以太网配置", true, 480)
	p.navBar.SetOnBack(func() { pm.Back() })

	// 输入框（静态 IP）
	// 表单采用「Label 在上，输入框在下」的标准间距，避免文字与输入框重叠
	p.ipInput = NewInputField(24, 110, 432, 50)
	p.ipInput.placeholder = "输入静态 IP 地址"
	p.ipInput.SetText("")

	p.maskInput = NewInputField(24, 175, 432, 50)
	p.maskInput.placeholder = "子网掩码（如 255.255.255.0 或 /24）"
	p.maskInput.SetText("")

	p.gatewayInput = NewInputField(24, 240, 432, 50)
	p.gatewayInput.placeholder = "网关（可选，如 192.168.1.1）"
	p.gatewayInput.SetText("")

	p.dnsInput = NewInputField(24, 305, 432, 50)
	p.dnsInput.placeholder = "DNS（如 8.8.8.8,1.1.1.1）"
	p.dnsInput.SetText("")

	// 键盘
	p.keyboard = NewVirtualKeyboard(480-240, 480, 240) // 底部半屏
	p.keyboard.onClose = func() {
		// 键盘关闭回调
	}
	p.keyboard.onEnter = func() {
		// 确认输入
		p.keyboard.Hide()
	}

	return p
}

func (p *EthernetPage) SetServices(s *AppServices) {
	p.services = s
}

func (p *EthernetPage) OnEnter() {
	p.lastErr = ""
	if p.services == nil {
		return
	}
	// 以太网页面固定操作 eth0，避免当前在 WiFi 时误把配置下发到 wlan0
	p.iface = "eth0"
	// 从 config 预填
	if p.services.Config != nil {
		p.ipInput.SetText(p.services.Config.Network.IP)
		p.maskInput.SetText(p.services.Config.Network.Netmask)
		p.gatewayInput.SetText(p.services.Config.Network.Gateway)
		p.dnsInput.SetText(p.services.Config.Network.DNS)
	}
	// 从当前网络状态补充接口/IP
	if st, err := p.services.GetNetworkStatus(); err == nil && st != nil {
		if st.CurrentInterface != "" {
			p.iface = st.CurrentInterface
		}
		if p.ipInput.GetText() == "" && st.IP != "" {
			p.ipInput.SetText(st.IP)
		}
		if p.gatewayInput.GetText() == "" && st.Gateway != "" {
			p.gatewayInput.SetText(st.Gateway)
		}
	}
}

func (p *EthernetPage) Render(g *Graphics) error {
	g.DrawRect(0, 0, 480, 480, ColorBackgroundStart)

	// 表单标题（放在输入框上方，留出足够间距）
	title := "IP 配置（静态）"
	if p.iface != "" {
		title = "IP 配置（" + p.iface + "）"
	}
	_ = g.DrawTextTTF(title, 24, 88, ColorTextSecondary, 14, FontWeightRegular)

	p.ipInput.Render(g)
	p.maskInput.Render(g)
	p.gatewayInput.Render(g)
	p.dnsInput.Render(g)

	// DHCP 按钮
	dhcpY := 365
	g.DrawRectRounded(24, dhcpY, 432, 50, 25, ColorPressed)
	dhcpW := g.MeasureText("使用 DHCP（自动获取）", 16, FontWeightMedium)
	_ = g.DrawTextTTF("使用 DHCP（自动获取）", 24+(432-dhcpW)/2, dhcpY+(50-int(16))/2, ColorTextPrimary, 16, FontWeightMedium)

	// 应用静态按钮
	btnY := 425
	g.DrawRectRounded(24, btnY, 432, 50, 25, ColorBrandBlue)
	labelW := g.MeasureText("应用静态IP", 18, FontWeightMedium)
	textTop := btnY + (50-int(18))/2
	_ = g.DrawTextTTF("应用静态IP", 24+(432-labelW)/2, textTop, ColorBackgroundStart, 18, FontWeightMedium)

	// 错误提示
	if p.lastErr != "" {
		_ = g.DrawTextTTF(p.lastErr, 24, 406, ColorErrorRed, 14, FontWeightRegular)
	}

	p.navBar.Render(g)
	p.keyboard.Render(g)

	return nil
}

func (p *EthernetPage) HandleTouch(x, y int, touchType TouchType) bool {
	if p.keyboard.isVisible {
		return p.keyboard.HandleTouch(x, y, touchType)
	}

	if p.navBar.HandleTouch(x, y, touchType) {
		return true
	}

	if p.ipInput.HandleTouch(x, y, touchType) {
		if p.ipInput.isFocused {
			p.keyboard.Show(p.ipInput)
		}
		return true
	}
	if p.maskInput.HandleTouch(x, y, touchType) {
		if p.maskInput.isFocused {
			p.keyboard.Show(p.maskInput)
		}
		return true
	}
	if p.gatewayInput.HandleTouch(x, y, touchType) {
		if p.gatewayInput.isFocused {
			p.keyboard.Show(p.gatewayInput)
		}
		return true
	}
	if p.dnsInput.HandleTouch(x, y, touchType) {
		if p.dnsInput.isFocused {
			p.keyboard.Show(p.dnsInput)
		}
		return true
	}

	// DHCP 按钮
	if x > 24 && x < 456 && y > 365 && y < 415 {
		if touchType == TouchUp {
			p.lastErr = ""
			if p.services == nil {
				p.lastErr = "服务未初始化"
				return true
			}
			if err := p.services.ApplyDHCP(p.iface, p.dnsInput.GetText()); err != nil {
				p.lastErr = err.Error()
				return true
			}
			p.pm.Back()
		}
		return true
	}

	// 应用静态按钮点击检测
	if x > 24 && x < 456 && y > 425 && y < 480 {
		if touchType == TouchUp {
			// 下发静态IP
			p.lastErr = ""
			if p.services == nil {
				p.lastErr = "服务未初始化"
				return true
			}
			if err := p.services.ApplyStaticIP(p.iface, p.ipInput.GetText(), p.maskInput.GetText(), p.gatewayInput.GetText(), p.dnsInput.GetText()); err != nil {
				p.lastErr = err.Error()
				return true
			}
			p.pm.Back()
		}
		return true
	}

	return false
}
