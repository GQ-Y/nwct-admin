package display

// EthernetPage 以太网设置页
type EthernetPage struct {
	BasePage
	navBar   *NavBar
	pm       *PageManager
	
	ipInput     *InputField
	maskInput   *InputField // 预留
	gatewayInput *InputField // 预留
	
	keyboard    *VirtualKeyboard
	saveButton  *ListView // 借用 ListView 实现按钮效果，或者直接画
}

func NewEthernetPage(pm *PageManager) *EthernetPage {
	p := &EthernetPage{
		BasePage: BasePage{Name: "ethernet"},
		pm:       pm,
	}
	
	p.navBar = NewNavBar("以太网配置", true, 480)
	p.navBar.SetOnBack(func() { pm.Back() })
	
	// 输入框
	// 表单采用「Label 在上，输入框在下」的标准间距，避免文字与输入框重叠
	p.ipInput = NewInputField(24, 120, 432, 50)
	p.ipInput.placeholder = "输入静态 IP 地址"
	p.ipInput.SetText("192.168.1.100")
	
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

func (p *EthernetPage) Render(g *Graphics) error {
	g.DrawRect(0, 0, 480, 480, ColorBackgroundStart)
	
	// 表单标题（放在输入框上方，留出足够间距）
	_ = g.DrawTextTTF("IP 地址", 24, 88, ColorTextSecondary, 14, FontWeightRegular)
	
	p.ipInput.Render(g)
	
	// 保存按钮 (简单绘制)
	btnY := 240
	g.DrawRectRounded(24, btnY, 432, 50, 25, ColorBrandBlue)
	labelW := g.MeasureText("保存配置", 18, FontWeightMedium)
	textTop := btnY + (50-int(18))/2
	_ = g.DrawTextTTF("保存配置", 24+(432-labelW)/2, textTop, ColorBackgroundStart, 18, FontWeightMedium)
	
	p.navBar.Render(g)
	p.keyboard.Render(g)
	
	return nil
}

func (p *EthernetPage) HandleTouch(x, y int, touchType TouchType) bool {
	if p.keyboard.isVisible {
		return p.keyboard.HandleTouch(x, y, touchType)
	}
	
	if p.navBar.HandleTouch(x, y, touchType) { return true }
	
	if p.ipInput.HandleTouch(x, y, touchType) {
		if p.ipInput.isFocused {
			p.keyboard.Show(p.ipInput)
		}
		return true
	}
	
	// 保存按钮点击检测
	if x > 24 && x < 456 && y > 240 && y < 290 {
		if touchType == TouchUp {
			// TODO: 保存逻辑
			p.pm.Back()
		}
		return true
	}
	
	return false
}

