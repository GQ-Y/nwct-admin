package pages

import (
	"fmt"
	"image/color"
	"nwct/client-nps/internal/display"
	"time"
)

// StatusPage 实时状态页（鸿蒙风格）
type StatusPage struct {
	BasePage
	logoFrame       int
	uploadSpeed     float64
	downloadSpeed   float64
	tunnelCount     int
	startTime       time.Time
	lastActivityTime time.Time
	onEnterSettings func()
}

// NewStatusPage 创建状态页
func NewStatusPage() *StatusPage {
	return &StatusPage{
		BasePage:         BasePage{Name: "status"},
		startTime:        time.Now(),
		lastActivityTime: time.Now(),
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

// Render 渲染页面（鸿蒙风格）
func (p *StatusPage) Render(g Graphics) error {
	gfx := g.(*display.Graphics)
	
	// 绘制渐变背景（鸿蒙风格深色）
	colors := []color.Color{
		color.RGBA{18, 20, 38, 255},   // 更深的深蓝色
		color.RGBA{28, 32, 58, 255},   // 中等深蓝
		color.RGBA{38, 44, 78, 255},   // 较浅深蓝
	}
	gfx.DrawGradient(0, 0, 480, 480, colors, display.GradientVertical)

	// 绘制顶部标题栏
	p.drawTopBar(gfx)

	// 绘制龙猫 LOGO（改进的动画）
	p.drawLogo(gfx)

	// 绘制数据卡片（鸿蒙风格）
	p.drawNetworkCard(gfx)
	p.drawStatsCards(gfx)

	// 底部提示（更小，更低调）
	gfx.DrawTextTTF("轻触屏幕进入设置", 180, 450, color.RGBA{100, 110, 140, 255}, 12, display.FontWeightRegular)

	return nil
}

// drawTopBar 绘制顶部标题栏
func (p *StatusPage) drawTopBar(g *display.Graphics) {
	// 顶部状态栏背景（半透明）
	g.DrawRect(0, 0, 480, 60, color.RGBA{20, 24, 44, 200})
	
	// 标题
	g.DrawTextTTF("NWCT 客户端", 40, 40, color.RGBA{255, 255, 255, 255}, 22, display.FontWeightMedium)
	
	// 状态指示器（右上角绿点）
	g.DrawCircle(430, 30, 6, color.RGBA{52, 211, 153, 255})
}

// drawLogo 绘制 LOGO 动画（改进版）
func (p *StatusPage) drawLogo(g *display.Graphics) {
	centerX, centerY := 240, 150
	
	// 外圈动画（呼吸效果）
	breatheFactor := float64(p.logoFrame%120) / 120.0
	if breatheFactor > 0.5 {
		breatheFactor = 1.0 - breatheFactor
	}
	breatheFactor = breatheFactor * 2.0 // 0 到 1
	
	radius1 := 50 + int(breatheFactor*10)
	alpha1 := uint8(150 + breatheFactor*50)
	
	// 外圈渐变色（蓝紫到青色）
	outerHue := float64(p.logoFrame%360) / 360.0
	c1 := interpolateHarmonyColors(outerHue)
	c1.A = alpha1
	g.DrawCircle(centerX, centerY, radius1, c1)

	// 中圈
	radius2 := 38
	c2 := color.RGBA{75, 123, 236, 230}
	g.DrawCircle(centerX, centerY, radius2, c2)
	
	// 内圈（最亮）
	radius3 := 28
	c3 := color.RGBA{100, 149, 255, 255}
	g.DrawCircle(centerX, centerY, radius3, c3)
	
	// 中心白点
	g.DrawCircle(centerX, centerY, 8, color.RGBA{255, 255, 255, 200})
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

// drawNetworkCard 绘制网络速度卡片
func (p *StatusPage) drawNetworkCard(g *display.Graphics) {
	// 大卡片背景（毛玻璃效果模拟）
	cardY := 230
	g.DrawRectRounded(30, cardY, 420, 100, 20, color.RGBA{45, 55, 90, 180})
	
	// 卡片标题
	g.DrawTextTTF("网络速度", 50, cardY+30, color.RGBA{160, 170, 200, 255}, 14, display.FontWeightRegular)
	
	// 上传速度
	g.DrawTextTTF("上传", 50, cardY+60, color.RGBA{140, 150, 180, 255}, 12, display.FontWeightRegular)
	uploadText := fmt.Sprintf("%.1f", p.uploadSpeed)
	g.DrawTextTTF(uploadText, 50, cardY+90, color.RGBA{99, 179, 237, 255}, 24, display.FontWeightMedium)
	g.DrawTextTTF("KB/s", 140, cardY+90, color.RGBA{140, 150, 180, 255}, 14, display.FontWeightRegular)
	
	// 分隔线
	g.DrawRect(240, cardY+50, 1, 60, color.RGBA{80, 90, 120, 100})
	
	// 下载速度
	g.DrawTextTTF("下载", 260, cardY+60, color.RGBA{140, 150, 180, 255}, 12, display.FontWeightRegular)
	downloadText := fmt.Sprintf("%.1f", p.downloadSpeed)
	g.DrawTextTTF(downloadText, 260, cardY+90, color.RGBA{52, 211, 153, 255}, 24, display.FontWeightMedium)
	g.DrawTextTTF("KB/s", 360, cardY+90, color.RGBA{140, 150, 180, 255}, 14, display.FontWeightRegular)
}

// drawStatsCards 绘制统计卡片
func (p *StatusPage) drawStatsCards(g *display.Graphics) {
	cardY := 350
	
	// 隧道数量卡片
	g.DrawRectRounded(30, cardY, 200, 80, 16, color.RGBA{45, 55, 90, 180})
	g.DrawTextTTF("隧道数量", 50, cardY+28, color.RGBA{140, 150, 180, 255}, 12, display.FontWeightRegular)
	tunnelText := fmt.Sprintf("%d", p.tunnelCount)
	g.DrawTextTTF(tunnelText, 50, cardY+60, color.RGBA{251, 191, 36, 255}, 28, display.FontWeightMedium)
	g.DrawTextTTF("个", 110, cardY+60, color.RGBA{140, 150, 180, 255}, 14, display.FontWeightRegular)
	
	// 运行时间卡片
	g.DrawRectRounded(250, cardY, 200, 80, 16, color.RGBA{45, 55, 90, 180})
	g.DrawTextTTF("运行时间", 270, cardY+28, color.RGBA{140, 150, 180, 255}, 12, display.FontWeightRegular)
	
	uptime := time.Since(p.startTime)
	hours := int(uptime.Hours())
	minutes := int(uptime.Minutes()) % 60
	var uptimeText string
	if hours > 0 {
		uptimeText = fmt.Sprintf("%dh %dm", hours, minutes)
	} else {
		uptimeText = fmt.Sprintf("%dm", minutes)
	}
	g.DrawTextTTF(uptimeText, 270, cardY+60, color.RGBA{167, 139, 250, 255}, 22, display.FontWeightMedium)
}

