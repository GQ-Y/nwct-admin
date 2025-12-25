package display

import (
	"fmt"
	"image/color"
	"time"
)

// StatusPage 实时状态页（鸿蒙风格）
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

// Render 渲染页面（鸿蒙风格 - 响应式）
func (p *StatusPage) Render(g *Graphics) error {
	// 使用相对尺寸
	w := float64(p.width)
	h := float64(p.height)

	// 绘制渐变背景（鸿蒙风格深色）
	colors := []color.Color{
		color.RGBA{18, 20, 38, 255}, // 更深的深蓝色
		color.RGBA{28, 32, 58, 255}, // 中等深蓝
		color.RGBA{38, 44, 78, 255}, // 较浅深蓝
	}
	g.DrawGradient(0, 0, p.width, p.height, colors, GradientVertical)

	// 绘制顶部标题栏
	p.drawTopBar(g, w, h)

	// 绘制龙猫 LOGO（改进的动画）
	p.drawLogo(g, w, h)

	// 绘制数据卡片（鸿蒙风格）
	p.drawNetworkCard(g, w, h)
	p.drawStatsCards(g, w, h)

	// 底部提示（居中显示）
	tipText := "轻触屏幕进入设置"
	tipWidth := g.MeasureText(tipText, h*0.027, FontWeightRegular) // 约13px
	tipX := (p.width - tipWidth) / 2
	g.DrawTextTTF(tipText, tipX, int(h*0.958), color.RGBA{100, 110, 140, 255}, h*0.027, FontWeightRegular)

	return nil
}

// drawTopBar 绘制顶部标题栏（响应式）
func (p *StatusPage) drawTopBar(g *Graphics, w, h float64) {
	barHeight := int(h * 0.125) // 12.5% 高度

	// 顶部状态栏背景（半透明）
	g.DrawRect(0, 0, p.width, barHeight, color.RGBA{20, 24, 44, 200})

	// 标题
	titleSize := h * 0.046   // 约22px
	titleY := int(h * 0.083) // 约40px
	g.DrawTextTTF("NWCT 客户端", int(w*0.083), titleY, color.RGBA{255, 255, 255, 255}, titleSize, FontWeightMedium)

	// 状态指示器（右上角绿点）
	dotRadius := int(h * 0.0125) // 约6px
	dotX := int(w * 0.896)       // 约430px
	dotY := int(h * 0.0625)      // 约30px
	g.DrawCircle(dotX, dotY, dotRadius, color.RGBA{52, 211, 153, 255})
}

// drawLogo 绘制 LOGO 动画（响应式）
func (p *StatusPage) drawLogo(g *Graphics, w, h float64) {
	centerX := int(w * 0.5)
	centerY := int(h * 0.3125) // 约150px

	// 外圈动画（呼吸效果）
	breatheFactor := float64(p.logoFrame%120) / 120.0
	if breatheFactor > 0.5 {
		breatheFactor = 1.0 - breatheFactor
	}
	breatheFactor = breatheFactor * 2.0 // 0 到 1

	baseRadius := h * 0.104 // 基础半径约50px
	radius1 := int(baseRadius + breatheFactor*h*0.021)
	alpha1 := uint8(150 + breatheFactor*50)

	// 外圈渐变色（蓝紫到青色）
	outerHue := float64(p.logoFrame%360) / 360.0
	c1 := interpolateHarmonyColors(outerHue)
	c1.A = alpha1
	g.DrawCircle(centerX, centerY, radius1, c1)

	// 中圈
	radius2 := int(h * 0.079) // 约38px
	c2 := color.RGBA{75, 123, 236, 230}
	g.DrawCircle(centerX, centerY, radius2, c2)

	// 内圈（最亮）
	radius3 := int(h * 0.058) // 约28px
	c3 := color.RGBA{100, 149, 255, 255}
	g.DrawCircle(centerX, centerY, radius3, c3)

	// 中心白点
	centerDot := int(h * 0.017) // 约8px
	g.DrawCircle(centerX, centerY, centerDot, color.RGBA{255, 255, 255, 200})
}

// interpolateHarmonyColors 鸿蒙风格颜色插值
func interpolateHarmonyColors(t float64) color.RGBA {
	// 鸿蒙主题色：蓝紫 -> 青色 -> 蓝紫
	colors := []color.RGBA{
		{102, 126, 234, 255}, // 蓝紫
		{59, 130, 246, 255},  // 蓝色
		{14, 165, 233, 255},  // 天蓝
		{6, 182, 212, 255},   // 青色
		{20, 184, 166, 255},  // 青绿
		{59, 130, 246, 255},  // 蓝色
		{102, 126, 234, 255}, // 回到蓝紫
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

// drawNetworkCard 绘制网络速度卡片（响应式）
func (p *StatusPage) drawNetworkCard(g *Graphics, w, h float64) {
	// 卡片尺寸和位置（基于屏幕尺寸计算）
	cardX := int(w * 0.0625)     // 30px @ 480
	cardY := int(h * 0.479)      // 230px @ 480
	cardW := int(w * 0.875)      // 420px @ 480
	cardH := int(h * 0.208)      // 100px @ 480
	cardRadius := int(w * 0.042) // 20px @ 480

	// 绘制卡片背景
	g.DrawRectRounded(cardX, cardY, cardW, cardH, cardRadius, color.RGBA{45, 55, 90, 180})

	// 卡片标题 - 直接相对于卡片位置
	titleSize := h * 0.029         // 14px @ 480
	titleY := cardY + int(h*0.058) // 卡片顶部 + 28px
	g.DrawTextTTF("网络速度", int(w*0.104), titleY, color.RGBA{160, 170, 200, 255}, titleSize, FontWeightRegular)

	// 左侧 - 上传速度
	leftX := int(w * 0.146)        // 70px @ 480
	labelY := cardY + int(h*0.115) // 卡片顶部 + 55px
	valueY := cardY + int(h*0.177) // 卡片顶部 + 85px
	labelSize := h * 0.025         // 12px @ 480
	valueSize := h * 0.058         // 28px @ 480
	unitSize := h * 0.029          // 14px @ 480

	g.DrawTextTTF("上传", leftX, labelY, color.RGBA{140, 150, 180, 255}, labelSize, FontWeightRegular)
	uploadText := fmt.Sprintf("%.1f", p.uploadSpeed)
	g.DrawTextTTF(uploadText, leftX, valueY, color.RGBA{99, 179, 237, 255}, valueSize, FontWeightMedium)
	numWidth := g.MeasureText(uploadText, valueSize, FontWeightMedium)
	g.DrawTextTTF("KB/s", leftX+numWidth+int(w*0.01), valueY, color.RGBA{140, 150, 180, 255}, unitSize, FontWeightRegular)

	// 中间分隔线
	lineX := int(w * 0.5)         // 240px @ 480
	lineY := cardY + int(h*0.094) // 卡片顶部 + 45px
	lineH := int(h * 0.104)       // 50px @ 480
	g.DrawRect(lineX, lineY, 1, lineH, color.RGBA{80, 90, 120, 100})

	// 右侧 - 下载速度
	rightX := int(w * 0.583) // 280px @ 480
	g.DrawTextTTF("下载", rightX, labelY, color.RGBA{140, 150, 180, 255}, labelSize, FontWeightRegular)
	downloadText := fmt.Sprintf("%.1f", p.downloadSpeed)
	g.DrawTextTTF(downloadText, rightX, valueY, color.RGBA{52, 211, 153, 255}, valueSize, FontWeightMedium)
	numWidth2 := g.MeasureText(downloadText, valueSize, FontWeightMedium)
	g.DrawTextTTF("KB/s", rightX+numWidth2+int(w*0.01), valueY, color.RGBA{140, 150, 180, 255}, unitSize, FontWeightRegular)
}

// drawStatsCards 绘制统计卡片（响应式）
func (p *StatusPage) drawStatsCards(g *Graphics, w, h float64) {
	cardY := int(h * 0.729)      // 350px @ 480
	cardW := int(w * 0.417)      // 200px @ 480
	cardH := int(h * 0.167)      // 80px @ 480
	cardRadius := int(w * 0.033) // 16px @ 480

	labelSize := h * 0.027 // 13px @ 480
	valueSize := h * 0.067 // 32px @ 480
	unitSize := h * 0.033  // 16px @ 480

	// 隧道数量卡片
	leftCardX := int(w * 0.0625) // 30px @ 480
	g.DrawRectRounded(leftCardX, cardY, cardW, cardH, cardRadius, color.RGBA{45, 55, 90, 180})

	labelX := int(w * 0.104)        // 50px @ 480
	labelY := cardY + int(h*0.0625) // 卡片顶部 + 30px
	valueY := cardY + int(h*0.135)  // 卡片顶部 + 65px

	g.DrawTextTTF("隧道数量", labelX, labelY, color.RGBA{140, 150, 180, 255}, labelSize, FontWeightRegular)
	tunnelText := fmt.Sprintf("%d", p.tunnelCount)
	g.DrawTextTTF(tunnelText, labelX, valueY, color.RGBA{251, 191, 36, 255}, valueSize, FontWeightMedium)
	numWidth := g.MeasureText(tunnelText, valueSize, FontWeightMedium)
	g.DrawTextTTF("个", labelX+numWidth+int(w*0.017), valueY, color.RGBA{140, 150, 180, 255}, unitSize, FontWeightRegular)

	// 运行时间卡片
	rightCardX := int(w * 0.521) // 250px @ 480
	g.DrawRectRounded(rightCardX, cardY, cardW, cardH, cardRadius, color.RGBA{45, 55, 90, 180})

	rightLabelX := int(w * 0.5625) // 270px @ 480
	g.DrawTextTTF("运行时间", rightLabelX, labelY, color.RGBA{140, 150, 180, 255}, labelSize, FontWeightRegular)

	uptime := time.Since(p.startTime)
	hours := int(uptime.Hours())
	minutes := int(uptime.Minutes()) % 60
	var uptimeText string
	if hours > 0 {
		uptimeText = fmt.Sprintf("%dh %dm", hours, minutes)
	} else {
		uptimeText = fmt.Sprintf("%dm", minutes)
	}
	uptimeSize := h * 0.054 // 26px @ 480
	g.DrawTextTTF(uptimeText, rightLabelX, valueY, color.RGBA{167, 139, 250, 255}, uptimeSize, FontWeightMedium)
}
