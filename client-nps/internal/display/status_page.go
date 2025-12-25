package display

import (
	"fmt"
	"image/color"
	"math"
	"time"
)

// StatusPage 实时状态页（鸿蒙风格 - 浅色简约）
type StatusPage struct {
	BasePage
	logoFrame        int
	uploadSpeed      float64
	downloadSpeed    float64
	tunnelCount      int
	startTime        time.Time
	lastActivityTime time.Time
	onEnterSettings  func()
	width            int // 屏幕宽度
	height           int // 屏幕高度
}

// NewStatusPage 创建状态页
func NewStatusPage() *StatusPage {
	return &StatusPage{
		BasePage:         BasePage{Name: "status"},
		startTime:        time.Now(),
		lastActivityTime: time.Now(),
		width:            480,
		height:           480,
	}
}

// SetOnEnterSettings 设置进入设置回调
func (p *StatusPage) SetOnEnterSettings(callback func()) {
	p.onEnterSettings = callback
}

// SetUploadSpeed 设置上传速度
func (p *StatusPage) SetUploadSpeed(speed float64) {
	p.uploadSpeed = speed
}

// SetDownloadSpeed 设置下载速度
func (p *StatusPage) SetDownloadSpeed(speed float64) {
	p.downloadSpeed = speed
}

// SetTunnelCount 设置隧道数量
func (p *StatusPage) SetTunnelCount(count int) {
	p.tunnelCount = count
}

// Update 更新页面状态
func (p *StatusPage) Update(deltaTime int64) {
	p.logoFrame++
}

// HandleTouch 处理触摸（点击任意位置进入设置）
func (p *StatusPage) HandleTouch(x, y int, touchType TouchType) bool {
	if touchType == TouchUp {
		p.lastActivityTime = time.Now()
		if p.onEnterSettings != nil {
			p.onEnterSettings()
		}
		return true
	}
	return false
}

// Render 渲染页面（鸿蒙风格 - 浅色简约响应式）
func (p *StatusPage) Render(g *Graphics) error {
	w := float64(p.width)
	h := float64(p.height)

	// 1. 浅色渐变背景 (模拟纸张质感)
	// 纯白 -> 极浅灰蓝
	colors := []color.Color{
		color.RGBA{255, 255, 255, 255}, // 纯白
		color.RGBA{248, 250, 252, 255}, // 极浅灰
		color.RGBA{241, 245, 249, 255}, // 浅灰蓝
	}
	g.DrawGradient(0, 0, p.width, p.height, colors, GradientVertical)

	// 2. 绘制顶部标题栏 (深色文字)
	p.drawTopBar(g, w, h)

	// 3. 绘制龙猫 LOGO (调整透明度适应浅色)
	p.drawLogo(g, w, h)

	// 4. 绘制数据区域 (无卡片背景，使用分割线)
	p.drawNetworkArea(g, w, h)
	p.drawStatsArea(g, w, h)

	// 5. 底部提示 (深灰色)
	tipText := "轻触屏幕进入设置"
	tipSize := h * 0.027 // 13px
	tipWidth := g.MeasureText(tipText, tipSize, FontWeightRegular)
	tipX := (p.width - tipWidth) / 2
	g.DrawTextTTF(tipText, tipX, int(h*0.94), color.RGBA{148, 163, 184, 255}, tipSize, FontWeightRegular)

	return nil
}

// drawTopBar 绘制顶部标题栏
func (p *StatusPage) drawTopBar(g *Graphics, w, h float64) {
	// 标题 (深黑色 #1E293B)
	titleSize := h * 0.046   // 22px
	titleY := int(h * 0.083) // 40px
	g.DrawTextTTF("NWCT 客户端", int(w*0.0625), titleY, color.RGBA{30, 41, 59, 255}, titleSize, FontWeightMedium)

	// 状态指示器（保持绿色，加个浅灰色描边让它在白底更清晰）
	dotRadius := int(h * 0.0125) // 6px
	dotX := int(w * 0.896)       // 430px
	dotY := int(h * 0.0625)      // 30px
	// 描边
	g.DrawCircle(dotX, dotY, dotRadius+1, color.RGBA{226, 232, 240, 255})
	// 绿点
	g.DrawCircle(dotX, dotY, dotRadius, color.RGBA{34, 197, 94, 255})
}

// drawLogo 绘制 LOGO 动画 - 悬浮灵动球 (Smooth & Cute)
func (p *StatusPage) drawLogo(g *Graphics, w, h float64) {
	centerX := int(w * 0.5)
	centerY := int(h * 0.33) // 基础中心位置

	// 动画参数 (使用 float64 保证计算平滑)
	t := float64(p.logoFrame) * 0.08 // 时间步长，控制速度

	// 1. 悬浮运动 (正弦波)
	// 上下浮动范围 +/- 8像素
	hoverOffset := math.Sin(t) * 8.0

	// 2. 呼吸效果 (轻微缩放)
	// 大小变化 +/- 1像素
	scaleOffset := math.Sin(t*0.5) * 1.0

	// 实际绘制坐标
	drawY := centerY + int(hoverOffset)

	// --- 绘制阴影 ---
	// 阴影随悬浮高度变化：越高越小越淡，越低越大越深
	shadowScale := (hoverOffset + 8.0) / 16.0 // 0.0 ~ 1.0
	shadowW := 40 - int(shadowScale*5)
	shadowH := 6
	shadowAlpha := uint8(40 + (1.0-shadowScale)*20)
	// 阴影位于球体下方
	g.DrawRectRounded(centerX-shadowW/2, centerY+50, shadowW, shadowH, 3, color.RGBA{148, 163, 184, shadowAlpha})

	// --- 绘制身体 ---
	// 鸿蒙蓝灵动球
	bodyRadius := 42 + int(scaleOffset)
	g.DrawCircle(centerX, drawY, bodyRadius, color.RGBA{59, 130, 246, 255})

	// --- 绘制肚子 (白色半圆) ---
	bellyRadius := 28
	g.DrawCircle(centerX, drawY+12, bellyRadius, color.RGBA{255, 255, 255, 240})

	// --- 绘制眼睛 ---
	eyeOffsetX := 14
	eyeOffsetY := drawY - 8
	eyeRadius := 4

	// 眨眼动画：每 4 秒眨一次眼
	// 周期约 300 帧 (假设 60fps)
	isBlinking := (p.logoFrame % 240) < 10

	if isBlinking {
		// 闭眼 (画线)
		g.DrawRect(centerX-eyeOffsetX-4, eyeOffsetY, 8, 2, color.RGBA{30, 41, 59, 255})
		g.DrawRect(centerX+eyeOffsetX-4, eyeOffsetY, 8, 2, color.RGBA{30, 41, 59, 255})
	} else {
		// 睁眼 (眼白 + 眼珠)
		// 眼白
		g.DrawCircle(centerX-eyeOffsetX, eyeOffsetY, eyeRadius+2, color.RGBA{255, 255, 255, 255})
		g.DrawCircle(centerX+eyeOffsetX, eyeOffsetY, eyeRadius+2, color.RGBA{255, 255, 255, 255})
		// 眼珠 (黑色)
		g.DrawCircle(centerX-eyeOffsetX, eyeOffsetY, eyeRadius-1, color.RGBA{30, 41, 59, 255})
		g.DrawCircle(centerX+eyeOffsetX, eyeOffsetY, eyeRadius-1, color.RGBA{30, 41, 59, 255})
	}

	// --- 绘制小装饰 (光泽) ---
	g.DrawCircle(centerX+18, drawY-22, 6, color.RGBA{255, 255, 255, 60})
}

// drawNetworkArea 绘制网络区域 (无卡片，大数字)
func (p *StatusPage) drawNetworkArea(g *Graphics, w, h float64) {
	startY := int(h * 0.52) // 250px 开始

	// 区域标题
	title := "实时速率"
	titleSize := h * 0.029 // ~14px
	titleX := int(w * 0.0625)
	// DrawTextTTF 的 y 是 topY（内部会再 +size），因此这里直接传 topY
	_ = g.DrawTextTTF(title, titleX, startY, color.RGBA{100, 116, 139, 255}, titleSize, FontWeightRegular)

	// 分隔线：放在标题下方，留出固定间距，避免与文字重叠
	sepY := startY + int(titleSize) + 8
	g.DrawRect(int(w*0.0625), sepY, int(w*0.875), 1, color.RGBA{226, 232, 240, 255})

	// 内容Y坐标
	labelY := sepY + 16
	valueY := labelY + 28

	labelSize := h * 0.025  // 12px
	valueSize := h * 0.0625 // 30px
	unitSize := h * 0.029   // 14px

	// 左侧 - 上传 (蓝色)
	leftX := int(w * 0.0625) // 30px
	_ = g.DrawTextTTF("上传", leftX, labelY, color.RGBA{148, 163, 184, 255}, labelSize, FontWeightRegular)

	uploadText := fmt.Sprintf("%.1f", p.uploadSpeed)
	_ = g.DrawTextTTF(uploadText, leftX, valueY, color.RGBA{59, 130, 246, 255}, valueSize, FontWeightMedium)
	numWidth := g.MeasureText(uploadText, valueSize, FontWeightMedium)
	_ = g.DrawTextTTF("KB/s", leftX+numWidth+8, valueY, color.RGBA{148, 163, 184, 255}, unitSize, FontWeightRegular)

	// 右侧 - 下载 (绿色)
	rightX := int(w * 0.55) // 264px
	_ = g.DrawTextTTF("下载", rightX, labelY, color.RGBA{148, 163, 184, 255}, labelSize, FontWeightRegular)

	downloadText := fmt.Sprintf("%.1f", p.downloadSpeed)
	_ = g.DrawTextTTF(downloadText, rightX, valueY, color.RGBA{16, 185, 129, 255}, valueSize, FontWeightMedium)
	numWidth2 := g.MeasureText(downloadText, valueSize, FontWeightMedium)
	_ = g.DrawTextTTF("KB/s", rightX+numWidth2+8, valueY, color.RGBA{148, 163, 184, 255}, unitSize, FontWeightRegular)
}

// drawStatsArea 绘制统计区域
func (p *StatusPage) drawStatsArea(g *Graphics, w, h float64) {
	startY := int(h * 0.73) // 350px

	// 竖向分隔线
	g.DrawRect(int(w*0.5), startY, 1, int(h*0.125), color.RGBA{226, 232, 240, 255})

	labelSize := h * 0.027 // 13px
	valueSize := h * 0.058 // 28px
	unitSize := h * 0.033  // 16px

	// 左侧 - 隧道 (橙色)
	leftX := int(w * 0.0625)
	labelY := startY + int(h*0.03) // 365px
	valueY := startY + int(h*0.1)  // 400px

	g.DrawTextTTF("隧道数量", leftX, labelY, color.RGBA{100, 116, 139, 255}, labelSize, FontWeightRegular)

	tunnelText := fmt.Sprintf("%d", p.tunnelCount)
	g.DrawTextTTF(tunnelText, leftX, valueY, color.RGBA{245, 158, 11, 255}, valueSize, FontWeightMedium)
	numWidth := g.MeasureText(tunnelText, valueSize, FontWeightMedium)
	g.DrawTextTTF("个", leftX+numWidth+5, valueY, color.RGBA{148, 163, 184, 255}, unitSize, FontWeightRegular)

	// 右侧 - 运行时间 (紫色)
	rightX := int(w * 0.55)
	g.DrawTextTTF("运行时间", rightX, labelY, color.RGBA{100, 116, 139, 255}, labelSize, FontWeightRegular)

	uptime := time.Since(p.startTime)
	hours := int(uptime.Hours())
	minutes := int(uptime.Minutes()) % 60
	var uptimeText string
	if hours > 0 {
		uptimeText = fmt.Sprintf("%dh %dm", hours, minutes)
	} else {
		uptimeText = fmt.Sprintf("%dm", minutes)
	}
	g.DrawTextTTF(uptimeText, rightX, valueY, color.RGBA{139, 92, 246, 255}, valueSize, FontWeightMedium)
}

// interpolateHarmonyColorsLight 浅色系颜色插值
func interpolateHarmonyColorsLight(t float64) color.RGBA {
	colors := []color.RGBA{
		{147, 197, 253, 255}, // 浅蓝
		{167, 139, 250, 255}, // 浅紫
		{52, 211, 153, 255},  // 浅绿
		{147, 197, 253, 255}, // 回到浅蓝
	}

	segmentCount := len(colors) - 1
	segment := t * float64(segmentCount)
	segmentIndex := int(segment)
	if segmentIndex >= segmentCount {
		segmentIndex = segmentCount - 1
	}

	localT := segment - float64(segmentIndex)
	c1 := colors[segmentIndex]
	c2 := colors[segmentIndex+1]

	return color.RGBA{
		R: uint8(float64(c1.R)*(1-localT) + float64(c2.R)*localT),
		G: uint8(float64(c1.G)*(1-localT) + float64(c2.G)*localT),
		B: uint8(float64(c1.B)*(1-localT) + float64(c2.B)*localT),
		A: 255,
	}
}
