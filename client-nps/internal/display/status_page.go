package display

import (
	"fmt"
	"image/color"
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

// drawLogo 绘制 LOGO 动画
func (p *StatusPage) drawLogo(g *Graphics, w, h float64) {
	centerX := int(w * 0.5)
	centerY := int(h * 0.33) // 158px (稍微下移一点，让上方留白均衡)

	// 呼吸动画
	breatheFactor := float64(p.logoFrame%120) / 120.0
	if breatheFactor > 0.5 {
		breatheFactor = 1.0 - breatheFactor
	}
	breatheFactor = breatheFactor * 2.0

	baseRadius := h * 0.115 // 55px (稍微加大)
	radius1 := int(baseRadius + breatheFactor*h*0.021)

	// 外圈 (使用较浅的蓝色，适应浅色背景)
	// 这里的颜色需要比深色模式更轻盈
	outerHue := float64(p.logoFrame%360) / 360.0
	c1 := interpolateHarmonyColorsLight(outerHue)
	c1.A = 100 // 降低透明度
	g.DrawCircle(centerX, centerY, radius1, c1)

	// 中圈
	radius2 := int(h * 0.085) // 41px
	g.DrawCircle(centerX, centerY, radius2, color.RGBA{59, 130, 246, 200})

	// 内圈
	radius3 := int(h * 0.0625) // 30px
	g.DrawCircle(centerX, centerY, radius3, color.RGBA{37, 99, 235, 255})

	// 中心白点
	centerDot := int(h * 0.019) // 9px
	g.DrawCircle(centerX, centerY, centerDot, color.RGBA{255, 255, 255, 255})
}

// drawNetworkArea 绘制网络区域 (无卡片，大数字)
func (p *StatusPage) drawNetworkArea(g *Graphics, w, h float64) {
	startY := int(h * 0.52) // 250px 开始

	// 区域标题
	g.DrawTextTTF("实时速率", int(w*0.0625), startY, color.RGBA{100, 116, 139, 255}, h*0.029, FontWeightRegular)

	// 分隔线 (横跨屏幕)
	g.DrawRect(int(w*0.0625), startY+10, int(w*0.875), 1, color.RGBA{226, 232, 240, 255})

	// 内容Y坐标
	labelY := startY + int(h*0.083) // 290px
	valueY := startY + int(h*0.146) // 320px

	labelSize := h * 0.025  // 12px
	valueSize := h * 0.0625 // 30px
	unitSize := h * 0.029   // 14px

	// 左侧 - 上传 (蓝色)
	leftX := int(w * 0.0625) // 30px
	g.DrawTextTTF("上传", leftX, labelY, color.RGBA{148, 163, 184, 255}, labelSize, FontWeightRegular)

	uploadText := fmt.Sprintf("%.1f", p.uploadSpeed)
	g.DrawTextTTF(uploadText, leftX, valueY, color.RGBA{59, 130, 246, 255}, valueSize, FontWeightMedium)
	numWidth := g.MeasureText(uploadText, valueSize, FontWeightMedium)
	g.DrawTextTTF("KB/s", leftX+numWidth+8, valueY, color.RGBA{148, 163, 184, 255}, unitSize, FontWeightRegular)

	// 右侧 - 下载 (绿色)
	rightX := int(w * 0.55) // 264px
	g.DrawTextTTF("下载", rightX, labelY, color.RGBA{148, 163, 184, 255}, labelSize, FontWeightRegular)

	downloadText := fmt.Sprintf("%.1f", p.downloadSpeed)
	g.DrawTextTTF(downloadText, rightX, valueY, color.RGBA{16, 185, 129, 255}, valueSize, FontWeightMedium)
	numWidth2 := g.MeasureText(downloadText, valueSize, FontWeightMedium)
	g.DrawTextTTF("KB/s", rightX+numWidth2+8, valueY, color.RGBA{148, 163, 184, 255}, unitSize, FontWeightRegular)
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
