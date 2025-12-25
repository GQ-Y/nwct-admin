package display

import (
	"strconv"
	"strings"

	"nwct/client-nps/internal/frp"
)

// TunnelEditPage 隧道详情/编辑页（真实业务：保存/删除）
type TunnelEditPage struct {
	BasePage
	navBar *NavBar
	pm     *PageManager

	services *AppServices

	originName string
	tunnel     *frp.Tunnel

	nameInput      *InputField
	localIPInput   *InputField
	localPortInput *InputField
	remotePortInput *InputField

	keyboard *VirtualKeyboard
	lastErr  string
}

func NewTunnelEditPage(pm *PageManager) *TunnelEditPage {
	p := &TunnelEditPage{
		BasePage: BasePage{Name: "tunnel_edit"},
		pm:       pm,
	}
	p.navBar = NewNavBar("编辑隧道", true, 480)
	p.navBar.SetOnBack(func() { pm.Back() })

	p.nameInput = NewInputField(24, 120, 432, 50)
	p.nameInput.placeholder = "隧道名称"

	p.localIPInput = NewInputField(24, 200, 432, 50)
	p.localIPInput.placeholder = "本地 IP（如 127.0.0.1）"

	p.localPortInput = NewInputField(24, 280, 432, 50)
	p.localPortInput.placeholder = "本地端口（如 8080）"

	p.remotePortInput = NewInputField(24, 360, 432, 50)
	p.remotePortInput.placeholder = "远程端口（0=自动分配）"

	p.keyboard = NewVirtualKeyboard(480-240, 480, 240)
	p.keyboard.onEnter = func() { p.keyboard.Hide() }

	return p
}

func (p *TunnelEditPage) SetServices(s *AppServices) { p.services = s }

func (p *TunnelEditPage) SetTunnel(t *frp.Tunnel) {
	p.tunnel = t
	p.lastErr = ""
	if t != nil {
		p.originName = t.Name
		p.nameInput.SetText(t.Name)
		p.localIPInput.SetText(t.LocalIP)
		p.localPortInput.SetText(strconv.Itoa(t.LocalPort))
		p.remotePortInput.SetText(strconv.Itoa(t.RemotePort))
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
}

func (p *TunnelEditPage) Render(g *Graphics) error {
	g.DrawRect(0, 0, 480, 480, ColorBackgroundStart)

	// 标题提示
	title := "隧道配置"
	if p.originName == "" {
		title = "新增隧道"
	}
	_ = g.DrawTextTTF(title, 24, 88, ColorTextSecondary, 14, FontWeightRegular)

	p.nameInput.Render(g)
	p.localIPInput.Render(g)
	p.localPortInput.Render(g)
	p.remotePortInput.Render(g)

	// 保存按钮
	saveY := 430
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

	if p.lastErr != "" {
		_ = g.DrawTextTTF(p.lastErr, 24, 406, ColorErrorRed, 14, FontWeightRegular)
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

	if p.nameInput.HandleTouch(x, y, touchType) {
		if p.nameInput.isFocused {
			p.keyboard.Show(p.nameInput)
		}
		return true
	}
	if p.localIPInput.HandleTouch(x, y, touchType) {
		if p.localIPInput.isFocused {
			p.keyboard.Show(p.localIPInput)
		}
		return true
	}
	if p.localPortInput.HandleTouch(x, y, touchType) {
		if p.localPortInput.isFocused {
			p.keyboard.Show(p.localPortInput)
		}
		return true
	}
	if p.remotePortInput.HandleTouch(x, y, touchType) {
		if p.remotePortInput.isFocused {
			p.keyboard.Show(p.remotePortInput)
		}
		return true
	}

	// 按钮区域
	saveY := 430
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
	return false
}

func (p *TunnelEditPage) save() {
	p.lastErr = ""
	if p.services == nil {
		p.lastErr = "服务未初始化"
		return
	}
	lp, err := strconv.Atoi(strings.TrimSpace(p.localPortInput.GetText()))
	if err != nil || lp < 0 || lp > 65535 {
		p.lastErr = "本地端口无效"
		return
	}
	rp, err := strconv.Atoi(strings.TrimSpace(p.remotePortInput.GetText()))
	if err != nil || rp < 0 || rp > 65535 {
		p.lastErr = "远程端口无效"
		return
	}
	t := &frp.Tunnel{
		Name:       strings.TrimSpace(p.nameInput.GetText()),
		Type:       "tcp",
		LocalIP:    strings.TrimSpace(p.localIPInput.GetText()),
		LocalPort:  lp,
		RemotePort: rp,
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

