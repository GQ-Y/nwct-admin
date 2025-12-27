package display

// ConfirmDialog 简易确认弹窗（遮罩 + 两按钮）
type ConfirmDialog struct {
	Visible bool
	Title   string
	Message string

	ConfirmText string
	CancelText  string

	OnConfirm func()
	OnCancel  func()
}

func (d *ConfirmDialog) Render(g *Graphics) {
	if d == nil || !d.Visible {
		return
	}
	// overlay
	g.DrawRect(0, 0, 480, 480, ColorOverlay)

	// dialog box
	x, y, w, h := 40, 140, 400, 200
	g.DrawRectRounded(x, y, w, h, 18, ColorBackgroundStart)
	g.DrawRect(x, y+58, w, 1, ColorSeparator)

	title := d.Title
	if title == "" {
		title = "确认"
	}
	msg := d.Message

	_ = g.DrawTextTTF(title, x+16, y+20, ColorTextPrimary, 18, FontWeightMedium)
	_ = g.DrawTextTTF(msg, x+16, y+82, ColorTextSecondary, 14, FontWeightRegular)

	cancel := d.CancelText
	if cancel == "" {
		cancel = "取消"
	}
	confirm := d.ConfirmText
	if confirm == "" {
		confirm = "连接"
	}

	btnY := y + h - 62
	btnH := 46
	btnW := (w - 16*3) / 2
	cx := x + 16
	okx := x + 16*2 + btnW

	g.DrawRectRounded(cx, btnY, btnW, btnH, 14, ColorPressed)
	cw := g.MeasureText(cancel, 16, FontWeightMedium)
	_ = g.DrawTextTTF(cancel, cx+(btnW-cw)/2, btnY+(btnH-int(16))/2, ColorTextPrimary, 16, FontWeightMedium)

	g.DrawRectRounded(okx, btnY, btnW, btnH, 14, ColorBrandBlue)
	ow := g.MeasureText(confirm, 16, FontWeightMedium)
	_ = g.DrawTextTTF(confirm, okx+(btnW-ow)/2, btnY+(btnH-int(16))/2, ColorBackgroundStart, 16, FontWeightMedium)
}

func (d *ConfirmDialog) HandleTouch(x, y int, touchType TouchType) bool {
	if d == nil || !d.Visible {
		return false
	}
	if touchType != TouchUp {
		return true
	}
	// dialog geometry must match Render
	dx, dy, dw, dh := 40, 140, 400, 200
	btnY := dy + dh - 62
	btnH := 46
	btnW := (dw - 16*3) / 2
	cx := dx + 16
	okx := dx + 16*2 + btnW

	// click outside -> cancel
	if x < dx || x > dx+dw || y < dy || y > dy+dh {
		d.Visible = false
		if d.OnCancel != nil {
			d.OnCancel()
		}
		return true
	}

	// cancel
	if x >= cx && x <= cx+btnW && y >= btnY && y <= btnY+btnH {
		d.Visible = false
		if d.OnCancel != nil {
			d.OnCancel()
		}
		return true
	}
	// confirm
	if x >= okx && x <= okx+btnW && y >= btnY && y <= btnY+btnH {
		d.Visible = false
		if d.OnConfirm != nil {
			d.OnConfirm()
		}
		return true
	}
	return true
}


