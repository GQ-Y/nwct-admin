package display

import (
	"strconv"
	"strings"

	appcfg "totoro-device/config"
	"totoro-device/internal/frp"
)

// TunnelEditPage 隧道详情/编辑页（真实业务：保存/删除）
type TunnelEditPage struct {
	BasePage
	navBar *NavBar
	pm     *PageManager

	services *AppServices

	originName string
	tunnel     *frp.Tunnel

	selectedType    string
	fallbackEnabled bool

	nameInput       *InputField
	localIPInput    *InputField
	localPortInput  *InputField
	remotePortInput *InputField
	domainInput     *InputField

	keyboard *VirtualKeyboard
	lastErr  string

	// 滚动：内容区可滚动，操作按钮在内容最底部（不吸底）
	scrollOffset int
	dragging     bool
	dragStartY   int
	lastDragY    int

	// 布局缓存（用于点击命中）
	saveBtnY int
	delBtnY  int
	errTextY int

	// 防止切换协议时丢失远程端口
	cachedRemotePort string
}

func NewTunnelEditPage(pm *PageManager) *TunnelEditPage {
	p := &TunnelEditPage{
		BasePage: BasePage{Name: "tunnel_edit"},
		pm:       pm,
	}
	p.navBar = NewNavBar("编辑隧道", true, 480)
	p.navBar.SetOnBack(func() { pm.Back() })

	// 默认类型
	p.selectedType = "tcp"
	p.fallbackEnabled = true

	p.nameInput = NewInputField(24, 170, 432, 46)
	p.nameInput.placeholder = "隧道名称"

	p.localIPInput = NewInputField(24, 226, 432, 46)
	p.localIPInput.placeholder = "本地 IP（如 127.0.0.1）"

	p.localPortInput = NewInputField(24, 282, 432, 46)
	p.localPortInput.placeholder = "本地端口（如 8080）"

	p.remotePortInput = NewInputField(24, 338, 432, 46)
	p.remotePortInput.placeholder = "远程端口（0=自动分配）"

	p.domainInput = NewInputField(24, 338, 432, 46)
	// placeholder 会在 layout 中根据 FRP 模式动态更新
	p.domainInput.placeholder = "域名"

	p.keyboard = NewVirtualKeyboard(480-240, 480, 240)
	p.keyboard.onEnter = func() { p.keyboard.Hide() }

	return p
}

func (p *TunnelEditPage) SetServices(s *AppServices) { p.services = s }

func (p *TunnelEditPage) contentTop() int { return 60 } // NavBar 高度
func (p *TunnelEditPage) visibleBottom() int {
	// 键盘弹出时，内容区底部受键盘遮挡
	if p.keyboard != nil && p.keyboard.isVisible {
		return p.keyboard.y
	}
	return 480
}

// clampScroll 根据“内容真实高度”（与 scrollOffset 无关）夹紧滚动范围
func (p *TunnelEditPage) clampScroll(contentHeight int) {
	visibleH := p.visibleBottom() - p.contentTop()
	minOffset := visibleH - contentHeight
	if minOffset > 0 {
		minOffset = 0
	}
	if p.scrollOffset > 0 {
		p.scrollOffset = 0
	}
	if p.scrollOffset < minOffset {
		p.scrollOffset = minOffset
	}
}

// layout 根据 scrollOffset & selectedType 更新控件 y 坐标，并返回内容底部
func (p *TunnelEditPage) layout() (contentBottom int) {
	base := p.contentTop() + p.scrollOffset

	// 协议区域（文字/按钮是用 Render 画，不是控件）
	// 让“协议类型”和“隧道名称”间距更紧凑
	nameY := base + 118
	p.nameInput.y = nameY
	p.localIPInput.y = base + 174
	p.localPortInput.y = base + 230

	// 第四个输入框：remote 或 domain
	p.remotePortInput.y = base + 286
	p.domainInput.y = base + 286

	// 下面开始计算“相对 contentTop 的真实内容高度”（与 scrollOffset 无关）
	relCursorBottom := 286 + 46
	if p.selectedType == "http" || p.selectedType == "https" {
		// 根据 FRP 模式动态更新 placeholder
		if p.services != nil && p.services.Config != nil {
			mode := p.services.Config.FRPServer.Mode
			if mode == appcfg.FRPModeManual {
				p.domainInput.placeholder = "自定义域名（如：example.com）"
			} else {
				domainSuffix := strings.TrimSpace(p.services.Config.FRPServer.DomainSuffix)
				if domainSuffix != "" {
					domainSuffix = strings.TrimPrefix(domainSuffix, ".")
					p.domainInput.placeholder = "域名前缀（如：subdomain）"
				} else {
					p.domainInput.placeholder = "域名前缀"
				}
			}
		} else {
			p.domainInput.placeholder = "域名"
		}
		// toggle 放在 domain input 下方
		relToggleLabelTop := 286 + 46 + 16
		relToggleY := relToggleLabelTop + 22 // pill top
		relCursorBottom = relToggleY + 24
	}

	// 错误提示（跟随内容）
	relErrTextY := relCursorBottom + 12
	relAfterErr := relErrTextY
	if strings.TrimSpace(p.lastErr) != "" {
		relAfterErr += 22
	}

	// 操作按钮（在内容最底部）
	relBtnY := relAfterErr + 10
	relContentHeight := relBtnY + 50 + 16

	// 写回绝对坐标（带 scrollOffset）
	p.errTextY = base + relErrTextY
	p.saveBtnY = base + relBtnY
	p.delBtnY = base + relBtnY
	contentBottom = base + relContentHeight

	// 夹紧滚动边界
	p.clampScroll(relContentHeight)
	return contentBottom
}

func (p *TunnelEditPage) showKeyboardFor(input *InputField) {
	p.keyboard.Show(input)
	// 确保输入框在可见区域内（考虑键盘遮挡）
	visibleTop := p.contentTop()
	visibleBottom := p.visibleBottom()
	inTop := input.y
	inBottom := input.y + input.height
	padding := 10
	if inBottom > visibleBottom-padding {
		delta := (visibleBottom - padding) - inBottom
		p.scrollOffset += delta
		p.layout()
	} else if inTop < visibleTop+padding {
		delta := (visibleTop + padding) - inTop
		p.scrollOffset += delta
		p.layout()
	}
}

func (p *TunnelEditPage) SetTunnel(t *frp.Tunnel) {
	p.tunnel = t
	p.lastErr = ""
	if t != nil {
		p.originName = t.Name
		p.nameInput.SetText(t.Name)
		p.localIPInput.SetText(t.LocalIP)
		p.localPortInput.SetText(strconv.Itoa(t.LocalPort))
		p.remotePortInput.SetText(strconv.Itoa(t.RemotePort))

		// 根据 FRP 模式处理域名显示：
		// - 手动模式：直接显示完整域名
		// - builtin/public 模式：如果是完整域名，提取前缀；否则直接显示
		rawDomain := strings.TrimSpace(t.Domain)
		if rawDomain != "" && (t.Type == "http" || t.Type == "https") {
			if p.services != nil && p.services.Config != nil {
				mode := p.services.Config.FRPServer.Mode
				if mode != appcfg.FRPModeManual {
					// builtin/public 模式：尝试提取前缀
					domainSuffix := strings.TrimSpace(p.services.Config.FRPServer.DomainSuffix)
					if domainSuffix != "" {
						domainSuffix = strings.TrimPrefix(domainSuffix, ".")
						if strings.HasSuffix(strings.ToLower(rawDomain), "."+strings.ToLower(domainSuffix)) {
							// 提取前缀
							prefix := rawDomain[:len(rawDomain)-len(domainSuffix)-1]
							p.domainInput.SetText(prefix)
						} else {
							// 不匹配后缀，可能是完整域名，直接显示
							p.domainInput.SetText(rawDomain)
						}
					} else {
						// 没有配置后缀，直接显示
						p.domainInput.SetText(rawDomain)
					}
				} else {
					// 手动模式：直接显示完整域名
					p.domainInput.SetText(rawDomain)
				}
			} else {
				// 无法获取配置，直接显示
				p.domainInput.SetText(rawDomain)
			}
		} else {
			p.domainInput.SetText(rawDomain)
		}

		if strings.TrimSpace(t.Type) != "" {
			p.selectedType = strings.ToLower(strings.TrimSpace(t.Type))
		} else {
			p.selectedType = "tcp"
		}
		p.fallbackEnabled = t.FallbackEnabled
		p.cachedRemotePort = p.remotePortInput.GetText()
		p.scrollOffset = 0
	}
}

// BeginCreate 进入“新增隧道”模式（清空 originName，预填默认值）
func (p *TunnelEditPage) BeginCreate() {
	p.originName = ""
	p.tunnel = nil
	p.lastErr = ""
	p.nameInput.SetText("")
	p.localIPInput.SetText("127.0.0.1")
	p.localPortInput.SetText("")
	p.remotePortInput.SetText("0")
	p.domainInput.SetText("")
	p.selectedType = "tcp"
	p.fallbackEnabled = true
	p.cachedRemotePort = p.remotePortInput.GetText()
	p.scrollOffset = 0
}

func (p *TunnelEditPage) Render(g *Graphics) error {
	g.DrawRect(0, 0, 480, 480, ColorBackgroundStart)
	p.layout()

	// 标题提示
	title := "隧道配置"
	if p.originName == "" {
		title = "新增隧道"
	}
	// 动态更新 navBar 标题
	if p.originName == "" {
		p.navBar.title = "新增隧道"
	} else {
		p.navBar.title = "编辑隧道"
	}
	base := p.contentTop() + p.scrollOffset
	_ = g.DrawTextTTF(title, 24, base+28, ColorTextSecondary, 14, FontWeightRegular)

	// 协议类型选择（五个按钮）
	_ = g.DrawTextTTF("协议类型", 24, base+52, ColorTextSecondary, 14, FontWeightRegular)
	typeY := base + 72
	typeH := 34
	types := []string{"tcp", "udp", "http", "https", "stcp"}
	gap := 8
	btnW := (432 - gap*(len(types)-1)) / len(types)
	for i, t := range types {
		x := 24 + i*(btnW+gap)
		isSel := strings.EqualFold(p.selectedType, t)
		bg := ColorSeparator
		fg := ColorTextSecondary
		if isSel {
			bg = ColorBrandBlue
			fg = ColorBackgroundStart
		}
		g.DrawRectRounded(x, typeY, btnW, typeH, 10, bg)
		label := strings.ToUpper(t)
		tw := g.MeasureText(label, 14, FontWeightMedium)
		_ = g.DrawTextTTF(label, x+(btnW-tw)/2, textTopForCenter(typeY, typeH, 14), fg, 14, FontWeightMedium)
	}

	p.nameInput.Render(g)
	p.localIPInput.Render(g)
	p.localPortInput.Render(g)

	// 根据类型显示 RemotePort 或 Domain
	if p.selectedType == "http" || p.selectedType == "https" {
		p.domainInput.Render(g)
		// 兜底页开关
		toggleW := 432
		toggleH := 24
		label := "兜底页（目标不可达时展示默认页）"
		toggleLabelTop := p.domainInput.y + p.domainInput.height + 16
		toggleY := toggleLabelTop + 22
		_ = g.DrawTextTTF(label, 24, toggleLabelTop, ColorTextSecondary, 14, FontWeightRegular)

		// 右侧 pill
		pillW := 100
		pillX := 24 + toggleW - pillW
		bg := ColorSeparator
		txt := "关闭"
		txtC := ColorTextSecondary
		if p.fallbackEnabled {
			bg = ColorSuccessGreen
			txt = "开启"
			txtC = ColorBackgroundStart
		}
		g.DrawRectRounded(pillX, toggleY, pillW, toggleH, 12, bg)
		tw := g.MeasureText(txt, 14, FontWeightMedium)
		_ = g.DrawTextTTF(txt, pillX+(pillW-tw)/2, textTopForCenter(toggleY, toggleH, 14), txtC, 14, FontWeightMedium)
	} else {
		p.remotePortInput.Render(g)
	}

	// 保存按钮
	saveY := p.saveBtnY
	g.DrawRectRounded(24, saveY, 208, 50, 25, ColorBrandBlue)
	saveW := g.MeasureText("保存", 18, FontWeightMedium)
	_ = g.DrawTextTTF("保存", 24+(208-saveW)/2, saveY+(50-int(18))/2, ColorBackgroundStart, 18, FontWeightMedium)

	// 删除按钮
	delX := 24 + 224
	// 新增模式下禁用删除（置灰）
	if p.originName == "" {
		g.DrawRectRounded(delX, saveY, 208, 50, 25, ColorSeparator)
		delW := g.MeasureText("删除", 18, FontWeightMedium)
		_ = g.DrawTextTTF("删除", delX+(208-delW)/2, saveY+(50-int(18))/2, ColorTextLight, 18, FontWeightMedium)
	} else {
		g.DrawRectRounded(delX, saveY, 208, 50, 25, ColorErrorRed)
		delW := g.MeasureText("删除", 18, FontWeightMedium)
		_ = g.DrawTextTTF("删除", delX+(208-delW)/2, saveY+(50-int(18))/2, ColorBackgroundStart, 18, FontWeightMedium)
	}

	// 错误提示（跟随内容）
	if strings.TrimSpace(p.lastErr) != "" {
		_ = g.DrawTextTTF(p.lastErr, 24, p.errTextY, ColorErrorRed, 14, FontWeightRegular)
	}

	p.navBar.Render(g)
	p.keyboard.Render(g)
	return nil
}

func (p *TunnelEditPage) HandleTouch(x, y int, touchType TouchType) bool {
	if p.keyboard.isVisible {
		return p.keyboard.HandleTouch(x, y, touchType)
	}
	if p.navBar.HandleTouch(x, y, touchType) {
		return true
	}

	// 内容区滚动（TouchMove 以 dy 驱动）
	inContent := y >= p.contentTop() && y < p.visibleBottom()
	if inContent {
		if touchType == TouchDown {
			p.dragging = false
			p.dragStartY = y
			p.lastDragY = y
			// 只要按下在内容区，就认为后续可能滚动
			return true
		}
		if touchType == TouchMove {
			dy := y - p.lastDragY
			if !p.dragging && (y-p.dragStartY > 6 || p.dragStartY-y > 6) {
				p.dragging = true
			}
			if p.dragging {
				p.scrollOffset += dy
				p.layout()
			}
			p.lastDragY = y
			return true
		}
		if touchType == TouchUp && p.dragging {
			p.dragging = false
			return true
		}
	}

	// 更新控件位置后再命中测试
	p.layout()

	// 操作按钮在内容里（跟随滚动）
	saveY := p.saveBtnY
	if y >= saveY && y <= saveY+50 {
		// 保存
		if x >= 24 && x <= 24+208 {
			if touchType == TouchUp {
				p.save()
			}
			return true
		}
		// 删除
		if x >= 24+224 && x <= 24+224+208 {
			if touchType == TouchUp {
				if p.originName != "" {
					p.delete()
				}
			}
			return true
		}
	}

	// 协议按钮点击
	if touchType == TouchUp {
		typeY := (p.contentTop() + p.scrollOffset) + 72
		typeH := 34
		if y >= typeY && y <= typeY+typeH && x >= 24 && x <= 24+432 {
			types := []string{"tcp", "udp", "http", "https", "stcp"}
			gap := 8
			btnW := (432 - gap*(len(types)-1)) / len(types)
			for i, t := range types {
				bx := 24 + i*(btnW+gap)
				if x >= bx && x <= bx+btnW {
					prev := p.selectedType
					p.selectedType = t
					// 切换协议时不丢失远程端口：从非 HTTP -> HTTP 缓存；从 HTTP -> 非 HTTP 恢复
					if (prev != "http" && prev != "https") && (t == "http" || t == "https") {
						p.cachedRemotePort = strings.TrimSpace(p.remotePortInput.GetText())
					}
					if (prev == "http" || prev == "https") && (t != "http" && t != "https") {
						if strings.TrimSpace(p.remotePortInput.GetText()) == "" || strings.TrimSpace(p.remotePortInput.GetText()) == "0" {
							if strings.TrimSpace(p.cachedRemotePort) != "" {
								p.remotePortInput.SetText(p.cachedRemotePort)
							}
						}
					}
					return true
				}
			}
		}
	}

	if p.nameInput.HandleTouch(x, y, touchType) {
		if p.nameInput.isFocused {
			p.showKeyboardFor(p.nameInput)
		}
		return true
	}
	if p.localIPInput.HandleTouch(x, y, touchType) {
		if p.localIPInput.isFocused {
			p.showKeyboardFor(p.localIPInput)
		}
		return true
	}
	if p.localPortInput.HandleTouch(x, y, touchType) {
		if p.localPortInput.isFocused {
			p.showKeyboardFor(p.localPortInput)
		}
		return true
	}
	if p.selectedType == "http" || p.selectedType == "https" {
		// domain 输入
		if p.domainInput.HandleTouch(x, y, touchType) {
			if p.domainInput.isFocused {
				p.showKeyboardFor(p.domainInput)
			}
			return true
		}
		// fallback toggle
		if touchType == TouchUp {
			toggleLabelTop := p.domainInput.y + p.domainInput.height + 16
			toggleY := toggleLabelTop + 22
			toggleH := 24
			pillW := 100
			pillX := 24 + 432 - pillW
			if y >= toggleY && y <= toggleY+toggleH && x >= pillX && x <= pillX+pillW {
				p.fallbackEnabled = !p.fallbackEnabled
				return true
			}
		}
	} else {
		// remote port 输入
		if p.remotePortInput.HandleTouch(x, y, touchType) {
			if p.remotePortInput.isFocused {
				p.showKeyboardFor(p.remotePortInput)
			}
			return true
		}
	}

	return false
}

func (p *TunnelEditPage) save() {
	p.lastErr = ""
	if p.services == nil {
		p.lastErr = "服务未初始化"
		return
	}
	tt := strings.ToLower(strings.TrimSpace(p.selectedType))
	if tt == "" {
		tt = "tcp"
	}
	switch tt {
	case "tcp", "udp", "http", "https", "stcp":
	default:
		p.lastErr = "协议类型无效"
		return
	}
	name := strings.TrimSpace(p.nameInput.GetText())
	if name == "" {
		p.lastErr = "隧道名称不能为空"
		return
	}
	localIP := strings.TrimSpace(p.localIPInput.GetText())
	if localIP == "" {
		p.lastErr = "本地IP不能为空"
		return
	}
	lp, err := strconv.Atoi(strings.TrimSpace(p.localPortInput.GetText()))
	if err != nil || lp < 0 || lp > 65535 {
		p.lastErr = "本地端口无效"
		return
	}

	rp := 0
	domain := ""
	fb := false
	if tt == "http" || tt == "https" {
		domainInput := strings.TrimSpace(p.domainInput.GetText())
		// 根据 FRP 模式处理域名：
		// - 手动模式：直接使用用户输入的完整域名
		// - builtin/public 模式：如果包含点号，视为完整域名；否则拼接默认后缀
		if p.services != nil && p.services.Config != nil {
			mode := p.services.Config.FRPServer.Mode
			if mode == appcfg.FRPModeManual {
				// 手动模式：直接使用完整域名
				domain = domainInput
			} else {
				// builtin/public 模式：若包含点号，视为完整域名；否则拼接默认后缀
				if domainInput != "" {
					if strings.Contains(domainInput, ".") {
						domain = domainInput
					} else {
						domainSuffix := strings.TrimSpace(p.services.Config.FRPServer.DomainSuffix)
						if domainSuffix != "" {
							domainSuffix = strings.TrimPrefix(domainSuffix, ".")
							domain = domainInput + "." + domainSuffix
						} else {
							domain = domainInput
						}
					}
				}
			}
		} else {
			// 无法获取配置，直接使用输入值
			domain = domainInput
		}
		fb = p.fallbackEnabled
		// http/https 不需要 remote port；强制 0
		rp = 0
	} else {
		rp, err = strconv.Atoi(strings.TrimSpace(p.remotePortInput.GetText()))
		if err != nil || rp < 0 || rp > 65535 {
			p.lastErr = "远程端口无效"
			return
		}
	}

	t := &frp.Tunnel{
		Name:            name,
		Type:            tt,
		LocalIP:         localIP,
		LocalPort:       lp,
		RemotePort:      rp,
		Domain:          domain,
		FallbackEnabled: fb,
	}
	// 新增 or 更新
	if p.originName == "" {
		if err := p.services.FRP.AddTunnel(t); err != nil {
			p.lastErr = err.Error()
			return
		}
	} else {
		if err := p.services.UpdateTunnel(p.originName, t); err != nil {
			p.lastErr = err.Error()
			return
		}
	}
	p.pm.Back()
}

func (p *TunnelEditPage) delete() {
	p.lastErr = ""
	if p.services == nil {
		p.lastErr = "服务未初始化"
		return
	}
	if p.originName == "" {
		p.lastErr = "未选择隧道"
		return
	}
	if err := p.services.DeleteTunnel(p.originName); err != nil {
		p.lastErr = err.Error()
		return
	}
	p.pm.Back()
}
