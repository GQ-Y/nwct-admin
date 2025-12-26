package display

import (
	"image/color"
	"strings"
)

// InputField 输入框
type InputField struct {
	x, y, width, height int
	text                string
	placeholder         string
	isFocused           bool
	isPassword          bool
	maxLength           int
	
	// 光标动画
	cursorFrame int
}

func NewInputField(x, y, width, height int) *InputField {
	return &InputField{
		x: x, y: y, width: width, height: height,
		maxLength: 32,
	}
}

func (input *InputField) SetText(text string) {
	input.text = text
}

func (input *InputField) GetText() string {
	return input.text
}

func (input *InputField) SetFocus(focused bool) {
	input.isFocused = focused
}

func (input *InputField) Render(g *Graphics) {
	// 背景
	g.DrawRect(input.x, input.y, input.width, input.height, ColorBackgroundStart)
	
	// 底部线条
	lineColor := ColorSeparator
	lineHeight := 1
	if input.isFocused {
		lineColor = ColorBrandBlue
		lineHeight = 2
	}
	g.DrawRect(input.x, input.y+input.height-lineHeight, input.width, lineHeight, lineColor)
	
	// 文字内容
	displayText := input.text
	if input.isPassword {
		displayText = strings.Repeat("•", len(input.text))
	}
	
	textColor := ColorTextPrimary
	if len(input.text) == 0 {
		displayText = input.placeholder
		textColor = ColorTextLight
	}
	
	// 简单的光标闪烁
	showCursor := input.isFocused && (input.cursorFrame/30)%2 == 0
	
	textX := input.x + 8
	fontSize := 20.0
	// DrawTextTTF 的 y 是“文字顶部”（内部会再 +size），因此用统一的居中计算
	textTop := textTopForCenter(input.y, input.height, fontSize)
	
	_ = g.DrawTextTTF(displayText, textX, textTop, textColor, fontSize, FontWeightRegular)
	
	if showCursor {
		width := g.MeasureText(displayText, fontSize, FontWeightRegular)
		if len(input.text) == 0 { width = 0 }
		// 光标同样在输入框内垂直居中
		cursorTop := input.y + 10
		cursorH := input.height - 20
		g.DrawRect(textX+width+2, cursorTop, 2, cursorH, ColorBrandBlue)
	}
	
	if input.isFocused {
		input.cursorFrame++
	}
}

func (input *InputField) HandleTouch(x, y int, touchType TouchType) bool {
	if x >= input.x && x <= input.x+input.width && y >= input.y && y <= input.y+input.height {
		if touchType == TouchUp {
			input.isFocused = true
			return true
		}
	}
	return false
}

// -----------------------------------------------------------------------------

// VirtualKeyboard 虚拟键盘
type VirtualKeyboard struct {
	x, y, width, height int
	isVisible           bool
	targetInput         *InputField
	mode                int // 0: Lower, 1: Upper, 2: Number
	
	// 按键布局
	keysLower [][]string
	keysUpper [][]string
	keysNum   [][]string
	
	pressedKey string
	onClose    func()
	onEnter    func()
}

func NewVirtualKeyboard(y, width, height int) *VirtualKeyboard {
	vk := &VirtualKeyboard{
		x: 0, y: y, width: width, height: height,
		isVisible: false,
		mode:      0,
	}
	
	// 初始化布局
	vk.keysLower = [][]string{
		{"q", "w", "e", "r", "t", "y", "u", "i", "o", "p"},
		{"a", "s", "d", "f", "g", "h", "j", "k", "l"},
		{"UP", "z", "x", "c", "v", "b", "n", "m", "DEL"},
		{"123", "SPACE", "ENTER"},
	}
	
	vk.keysUpper = [][]string{
		{"Q", "W", "E", "R", "T", "Y", "U", "I", "O", "P"},
		{"A", "S", "D", "F", "G", "H", "J", "K", "L"},
		{"LOW", "Z", "X", "C", "V", "B", "N", "M", "DEL"},
		{"123", "SPACE", "ENTER"},
	}
	
	vk.keysNum = [][]string{
		{"1", "2", "3", "4", "5", "6", "7", "8", "9", "0"},
		{"-", "/", ":", ";", "(", ")", "$", "&", "@", "\""},
		{"ABC", ".", ",", "?", "!", "'", "#", "+", "DEL"},
		{"ABC", "SPACE", "ENTER"},
	}
	
	return vk
}

func (vk *VirtualKeyboard) Show(input *InputField) {
	vk.targetInput = input
	vk.isVisible = true
	vk.mode = 0 // Reset to lowercase
}

func (vk *VirtualKeyboard) Hide() {
	vk.isVisible = false
	if vk.targetInput != nil {
		vk.targetInput.SetFocus(false)
		vk.targetInput = nil
	}
	if vk.onClose != nil {
		vk.onClose()
	}
}

func (vk *VirtualKeyboard) Render(g *Graphics) {
	if !vk.isVisible {
		return
	}
	
	// 背景
	g.DrawRect(vk.x, vk.y, vk.width, vk.height, ColorKeyboardBg)
	
	// 渲染按键
	currentLayout := vk.keysLower
	if vk.mode == 1 {
		currentLayout = vk.keysUpper
	} else if vk.mode == 2 {
		currentLayout = vk.keysNum
	}
	
	rowHeight := vk.height / 4
	padding := 4
	
	for r, row := range currentLayout {
		rowY := vk.y + r*rowHeight + padding
		keyHeight := rowHeight - padding*2
		
		// 计算每行的总宽度单位
		totalUnits := 0.0
		for _, k := range row {
			totalUnits += vk.getKeyWidthUnit(k)
		}
		
		unitWidth := float64(vk.width - padding*2) / totalUnits
		currentX := float64(vk.x + padding)
		
		for _, key := range row {
			keyWidth := int(unitWidth * vk.getKeyWidthUnit(key)) - padding
			
			// 绘制按键背景
			bg := ColorKeyBg
			if key == "UP" || key == "LOW" || key == "123" || key == "ABC" || key == "DEL" || key == "ENTER" {
				bg = color.RGBA{180, 190, 200, 255} // 功能键深一点
			}
			if key == "ENTER" {
				bg = ColorBrandBlue // 确认键蓝色
			}
			if vk.pressedKey == key {
				bg = ColorPressed
			}
			
			g.DrawRectRounded(int(currentX), rowY, keyWidth, keyHeight, 6, bg)
			
			// 绘制按键文字
			text := key
			if key == "SPACE" { text = "空格" }
			if key == "UP" { text = "⇧" }
			if key == "LOW" { text = "⇩" }
			if key == "DEL" { text = "⌫" }
			if key == "ENTER" { text = "确定" }
			
			textColor := ColorTextPrimary
			if key == "ENTER" { textColor = ColorBackgroundStart }
			
			textSize := 18.0
			// 按键内垂直居中（DrawTextTTF 传 topY）
			tw := g.MeasureText(text, textSize, FontWeightMedium)
			textTop := textTopForCenter(rowY, keyHeight, textSize)
			_ = g.DrawTextTTF(text, int(currentX)+keyWidth/2-tw/2, textTop, textColor, textSize, FontWeightMedium)
			
			currentX += float64(keyWidth + padding)
		}
	}
}

func (vk *VirtualKeyboard) getKeyWidthUnit(key string) float64 {
	switch key {
	case "SPACE": return 4.0
	case "ENTER": return 2.0
	case "UP", "LOW", "DEL", "123", "ABC": return 1.5
	default: return 1.0
	}
}

func (vk *VirtualKeyboard) HandleTouch(x, y int, touchType TouchType) bool {
	if !vk.isVisible || x < vk.x || y < vk.y {
		return false
	}
	
	// 简单的按键检测逻辑 (这里只是简化版，实际需要精确计算)
	if touchType == TouchDown {
		// 查找按下的键
		vk.pressedKey = vk.hitTest(x, y)
		return true
	} else if touchType == TouchUp {
		key := vk.hitTest(x, y)
		if key != "" && key == vk.pressedKey {
			vk.handleKeyPress(key)
		}
		vk.pressedKey = ""
		return true
	}
	
	return true // 吞掉键盘区域的事件
}

func (vk *VirtualKeyboard) hitTest(x, y int) string {
	// 反向计算点击了哪个键
	// 逻辑与 Render 类似
	currentLayout := vk.keysLower
	if vk.mode == 1 {
		currentLayout = vk.keysUpper
	} else if vk.mode == 2 {
		currentLayout = vk.keysNum
	}
	
	rowHeight := vk.height / 4
	padding := 4
	
	r := (y - vk.y) / rowHeight
	if r < 0 || r >= len(currentLayout) { return "" }
	
	row := currentLayout[r]
	totalUnits := 0.0
	for _, k := range row { totalUnits += vk.getKeyWidthUnit(k) }
	
	unitWidth := float64(vk.width - padding*2) / totalUnits
	currentX := float64(vk.x + padding)
	
	for _, key := range row {
		keyWidth := int(unitWidth * vk.getKeyWidthUnit(key)) - padding
		if x >= int(currentX) && x <= int(currentX)+keyWidth {
			return key
		}
		currentX += float64(keyWidth + padding)
	}
	return ""
}

func (vk *VirtualKeyboard) handleKeyPress(key string) {
	if vk.targetInput == nil { return }
	
	switch key {
	case "UP": vk.mode = 1
	case "LOW": vk.mode = 0
	case "123": vk.mode = 2
	case "ABC": vk.mode = 0
	case "DEL":
		text := vk.targetInput.text
		if len(text) > 0 {
			vk.targetInput.SetText(text[:len(text)-1])
		}
	case "ENTER":
		vk.Hide()
		if vk.onEnter != nil { vk.onEnter() }
	case "SPACE":
		if len(vk.targetInput.text) < vk.targetInput.maxLength {
			vk.targetInput.SetText(vk.targetInput.text + " ")
		}
	default:
		if len(vk.targetInput.text) < vk.targetInput.maxLength {
			vk.targetInput.SetText(vk.targetInput.text + key)
		}
	}
}

