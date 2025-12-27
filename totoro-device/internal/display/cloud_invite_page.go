package display

import (
	"fmt"
	"strings"

	"totoro-device/internal/database"
	"totoro-device/internal/bridgeclient"
)

// CloudInvitePage 私有分享邀请码（MVP：预览/保存邀请码；连接动作后续加按钮）
type CloudInvitePage struct {
	BasePage
	navBar    *NavBar
	pm        *PageManager
	services  *AppServices
	codeInput *InputField
	keyboard  *VirtualKeyboard

	lastMsg string
	lastErr string
}

func NewCloudInvitePage(pm *PageManager) *CloudInvitePage {
	p := &CloudInvitePage{
		BasePage: BasePage{Name: "cloud_invite"},
		pm:       pm,
	}
	p.navBar = NewNavBar("邀请码", true, 480)
	p.navBar.SetOnBack(func() { pm.Back() })

	p.codeInput = NewInputField(24, 120, 432, 50)
	p.codeInput.placeholder = "请输入邀请码"
	p.codeInput.isPassword = false

	p.keyboard = NewVirtualKeyboard(480-240, 480, 240)
	p.keyboard.onEnter = func() {
		p.keyboard.Hide()
		_ = p.preview()
	}
	return p
}

func (p *CloudInvitePage) SetServices(s *AppServices) {
	p.services = s
}

func (p *CloudInvitePage) preview() error {
	p.lastErr = ""
	p.lastMsg = ""
	if p.services == nil || p.services.Config == nil {
		p.lastErr = "服务未初始化"
		return nil
	}
	code := strings.TrimSpace(p.codeInput.GetText())
	if code == "" {
		p.lastErr = "邀请码不能为空"
		return nil
	}

	var res *bridgeclient.PreviewResp
	err := p.services.RegisterBridgeAndRetryOn401(func(bc *bridgeclient.Client) error {
		r, e := bc.PreviewInvite(code)
		if e != nil {
			return e
		}
		res = r
		return nil
	})
	if err != nil {
		// 友好化错误提示
		msg := err.Error()
		low := strings.ToLower(msg)
		switch {
		case strings.Contains(low, "invalid_code"):
			p.lastErr = "邀请码无效"
		case strings.Contains(low, "expired"):
			p.lastErr = "邀请码已过期"
		case strings.Contains(low, "status=401"):
			p.lastErr = "桥梁鉴权失败（设备未注册/不在白名单）"
		default:
			p.lastErr = msg
		}
		return err
	}

	// 仅保存邀请码（用于后续“一键连接”）
	db := database.GetDB()
	if db != nil {
		_ = database.SetPublicInviteCode(db, code)
	}
	p.lastMsg = fmt.Sprintf("预览成功：node=%s", strings.TrimSpace(res.Node.NodeID))
	return nil
}

func (p *CloudInvitePage) Render(g *Graphics) error {
	g.DrawRect(0, 0, 480, 480, ColorBackgroundStart)
	p.navBar.Render(g)

	_ = g.DrawTextTTF("输入邀请码获取私有分享云节点", 24, 90, ColorTextSecondary, 14, FontWeightRegular)
	p.codeInput.Render(g)

	// 按钮
	btnY := 200
	g.DrawRectRounded(24, btnY, 432, 50, 25, ColorBrandBlue)
	w := g.MeasureText("预览并保存", 18, FontWeightMedium)
	_ = g.DrawTextTTF("预览并保存", 24+(432-w)/2, btnY+(50-int(18))/2, ColorBackgroundStart, 18, FontWeightMedium)

	if p.lastMsg != "" {
		_ = g.DrawTextTTF(p.lastMsg, 24, 270, ColorSuccessGreen, 14, FontWeightRegular)
	}
	if p.lastErr != "" {
		_ = g.DrawTextTTF(p.lastErr, 24, 270, ColorErrorRed, 14, FontWeightRegular)
	}

	p.keyboard.Render(g)
	return nil
}

func (p *CloudInvitePage) HandleTouch(x, y int, touchType TouchType) bool {
	if p.keyboard.isVisible {
		return p.keyboard.HandleTouch(x, y, touchType)
	}
	if p.navBar.HandleTouch(x, y, touchType) {
		return true
	}
	if p.codeInput.HandleTouch(x, y, touchType) {
		if p.codeInput.isFocused {
			p.keyboard.Show(p.codeInput)
		}
		return true
	}
	if x >= 24 && x <= 456 && y >= 200 && y <= 250 {
		if touchType == TouchUp {
			_ = p.preview()
		}
		return true
	}
	return false
}


