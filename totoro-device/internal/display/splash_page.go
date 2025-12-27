package display

import (
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"
	"time"

	"totoro-device/internal/logger"
)

// SplashPage 启动页（品牌 LOGO + 简单动画）
//
// 资源可在设备侧替换（无需重编译）：
// - 环境变量 NWCT_BRANDING_PATH 指定图片路径
// - 默认读取 /etc/nwct/branding.png （存在则使用）
type SplashPage struct {
	BasePage

	pm *PageManager

	startAt time.Time
	done    bool

	branding image.Image
}

func NewSplashPage(pm *PageManager) *SplashPage {
	return &SplashPage{
		BasePage: BasePage{Name: "splash"},
		pm:       pm,
	}
}

func (p *SplashPage) OnEnter() {
	p.startAt = time.Now()
	p.done = false

	// 加载品牌图
	path := strings.TrimSpace(os.Getenv("NWCT_BRANDING_PATH"))
	if path == "" && fileExists("/etc/nwct/branding.png") {
		path = "/etc/nwct/branding.png"
	}
	if path != "" {
		if img, err := loadImage(path); err == nil {
			p.branding = img
			logger.Info("启动页品牌图已加载: %s", path)
		} else {
			logger.Warn("启动页品牌图加载失败: %v", err)
			p.branding = nil
		}
	}
}

func (p *SplashPage) Update(deltaTime int64) {
	if p.done {
		return
	}
	// 展示 2.2 秒后自动进入首页
	if time.Since(p.startAt) >= 2200*time.Millisecond {
		p.done = true
		_ = p.pm.SwitchTo("status")
	}
}

func (p *SplashPage) HandleTouch(x, y int, touchType TouchType) bool {
	if touchType == TouchDown {
		p.done = true
		_ = p.pm.SwitchTo("status")
		return true
	}
	return false
}

func (p *SplashPage) Render(g *Graphics) error {
	// 背景渐变（与主题一致）
	g.DrawGradient(0, 0, 480, 480, []color.Color{ColorBackgroundStart, ColorBackgroundEnd}, GradientVertical)

	// 中心 LOGO（如果没有图片，就用文字占位）
	if p.branding != nil {
		// 逻辑坐标中，logo 最大宽高 220
		g.DrawImageFitCenter(p.branding, 240, 200, 220, 220)
	} else {
		_ = g.DrawTextTTF("Totoro", 150, 190, ColorTextPrimary, 44, FontWeightBold)
		_ = g.DrawTextTTF("Device", 188, 235, ColorTextSecondary, 22, FontWeightRegular)
	}

	// 简单的“加载动画”：三点跳动
	elapsed := time.Since(p.startAt).Milliseconds()
	phase := int(elapsed/250) % 3
	baseX := 240
	y := 330
	for i := 0; i < 3; i++ {
		r := 6
		if i == phase {
			r = 9
		}
		g.DrawCircleAA(baseX+(i-1)*22, y, r, ColorBrandBlue)
	}

	_ = g.DrawTextTTF("启动中…", 185, 360, ColorTextSecondary, 16, FontWeightRegular)
	return nil
}

func loadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}
