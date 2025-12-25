package display

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/math/fixed"
)

const (
	// 设计稿基准尺寸：所有页面/组件以 480x480 作为逻辑坐标系
	designW = 480
	designH = 480
)

// Graphics 图形绘制库
type Graphics struct {
	buffer *image.RGBA
	scaleX float64
	scaleY float64
}

// NewGraphics 创建图形库实例
func NewGraphics(buffer *image.RGBA) *Graphics {
	w := buffer.Bounds().Dx()
	h := buffer.Bounds().Dy()
	sx := float64(w) / float64(designW)
	sy := float64(h) / float64(designH)
	if sx <= 0 {
		sx = 1
	}
	if sy <= 0 {
		sy = 1
	}
	return &Graphics{
		buffer: buffer,
		scaleX: sx,
		scaleY: sy,
	}
}

func (g *Graphics) sx(v int) int { return int(math.Round(float64(v) * g.scaleX)) }
func (g *Graphics) sy(v int) int { return int(math.Round(float64(v) * g.scaleY)) }
func (g *Graphics) sr(v int) int {
	// 半径取平均缩放（目标是方形缩放）
	s := (g.scaleX + g.scaleY) * 0.5
	return int(math.Round(float64(v) * s))
}

func (g *Graphics) drawRectPx(x, y, w, h int, c color.Color) {
	if w <= 0 || h <= 0 {
		return
	}
	rect := image.Rect(x, y, x+w, y+h)
	draw.Draw(g.buffer, rect, &image.Uniform{c}, image.Point{}, draw.Src)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func (g *Graphics) blendPixelRGBA(x, y int, src color.RGBA) {
	if x < 0 || y < 0 || x >= g.buffer.Bounds().Dx() || y >= g.buffer.Bounds().Dy() {
		return
	}
	i := y*g.buffer.Stride + x*4
	dstR := float64(g.buffer.Pix[i+0])
	dstG := float64(g.buffer.Pix[i+1])
	dstB := float64(g.buffer.Pix[i+2])
	dstA := float64(g.buffer.Pix[i+3]) / 255.0

	sa := float64(src.A) / 255.0
	if sa <= 0 {
		return
	}

	// Porter-Duff over
	outA := sa + dstA*(1-sa)
	if outA <= 0 {
		return
	}
	outR := (float64(src.R)*sa + dstR*dstA*(1-sa)) / outA
	outG := (float64(src.G)*sa + dstG*dstA*(1-sa)) / outA
	outB := (float64(src.B)*sa + dstB*dstA*(1-sa)) / outA

	g.buffer.Pix[i+0] = uint8(clamp01(outR/255.0) * 255.0)
	g.buffer.Pix[i+1] = uint8(clamp01(outG/255.0) * 255.0)
	g.buffer.Pix[i+2] = uint8(clamp01(outB/255.0) * 255.0)
	g.buffer.Pix[i+3] = uint8(clamp01(outA) * 255.0)
}

// DrawCircleAA 绘制抗锯齿实心圆（用于状态页动画/指示点的平滑边缘）
func (g *Graphics) DrawCircleAA(cx, cy, r int, c color.Color) {
	cx = g.sx(cx)
	cy = g.sy(cy)
	r = g.sr(r)

	if r <= 0 {
		return
	}
	cr, cg, cb, ca := c.RGBA()
	base := color.RGBA{uint8(cr >> 8), uint8(cg >> 8), uint8(cb >> 8), uint8(ca >> 8)}

	rr := float64(r)
	inner := rr - 0.5
	outer := rr + 0.5
	inner2 := inner * inner
	outer2 := outer * outer

	minX := cx - r - 1
	maxX := cx + r + 1
	minY := cy - r - 1
	maxY := cy + r + 1

	for y := minY; y <= maxY; y++ {
		dy := float64(y) - float64(cy)
		for x := minX; x <= maxX; x++ {
			dx := float64(x) - float64(cx)
			d2 := dx*dx + dy*dy
			if d2 <= inner2 {
				// 全覆盖
				g.blendPixelRGBA(x, y, base)
				continue
			}
			if d2 >= outer2 {
				continue
			}
			// 边缘：线性覆盖估算
			d := math.Sqrt(d2)
			cover := clamp01(outer - d) // 0..1
			if cover <= 0 {
				continue
			}
			s := base
			s.A = uint8(float64(base.A) * cover)
			g.blendPixelRGBA(x, y, s)
		}
	}
}

// DrawEllipseAA 绘制抗锯齿实心椭圆
func (g *Graphics) DrawEllipseAA(cx, cy, rx, ry int, c color.Color) {
	cx = g.sx(cx)
	cy = g.sy(cy)
	rx = g.sx(rx)
	ry = g.sy(ry)
	if rx <= 0 || ry <= 0 {
		return
	}

	cr, cg, cb, ca := c.RGBA()
	base := color.RGBA{uint8(cr >> 8), uint8(cg >> 8), uint8(cb >> 8), uint8(ca >> 8)}

	frx := float64(rx)
	fry := float64(ry)

	// 用“距离到边界”的近似做 1px 的抗锯齿带
	// 椭圆方程： (dx/rx)^2 + (dy/ry)^2 = 1
	// 取 d = sqrt(v) - 1，v 为上式左侧；d<0 在内部
	minX := cx - rx - 2
	maxX := cx + rx + 2
	minY := cy - ry - 2
	maxY := cy + ry + 2

	for y := minY; y <= maxY; y++ {
		dy := float64(y - cy)
		ny := dy / fry
		for x := minX; x <= maxX; x++ {
			dx := float64(x - cx)
			nx := dx / frx
			v := nx*nx + ny*ny
			if v <= 0.0 {
				g.blendPixelRGBA(x, y, base)
				continue
			}
			d := math.Sqrt(v) - 1.0
			if d <= -0.5 {
				// 内部
				g.blendPixelRGBA(x, y, base)
				continue
			}
			if d >= 0.5 {
				continue
			}
			cover := clamp01(0.5 - d) // 0..1
			if cover <= 0 {
				continue
			}
			s := base
			s.A = uint8(float64(base.A) * cover)
			g.blendPixelRGBA(x, y, s)
		}
	}
}

// Clear 清空画布
func (g *Graphics) Clear(c color.Color) {
	draw.Draw(g.buffer, g.buffer.Bounds(), &image.Uniform{c}, image.Point{}, draw.Src)
}

// DrawRect 绘制矩形
func (g *Graphics) DrawRect(x, y, w, h int, c color.Color) {
	x2 := g.sx(x)
	y2 := g.sy(y)
	w2 := g.sx(w)
	h2 := g.sy(h)
	g.drawRectPx(x2, y2, w2, h2, c)
}

// DrawLine 绘制直线（Bresenham）
func (g *Graphics) DrawLine(x0, y0, x1, y1 int, c color.Color) {
	x0 = g.sx(x0)
	y0 = g.sy(y0)
	x1 = g.sx(x1)
	y1 = g.sy(y1)

	dx := x1 - x0
	if dx < 0 {
		dx = -dx
	}
	dy := y1 - y0
	if dy < 0 {
		dy = -dy
	}

	sx := -1
	if x0 < x1 {
		sx = 1
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}

	err := dx - dy

	for {
		if x0 >= 0 && x0 < g.buffer.Bounds().Dx() && y0 >= 0 && y0 < g.buffer.Bounds().Dy() {
			g.buffer.Set(x0, y0, c)
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := err * 2
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

// DrawRectRounded 绘制圆角矩形
func (g *Graphics) DrawRectRounded(x, y, w, h, radius int, c color.Color) {
	x = g.sx(x)
	y = g.sy(y)
	w = g.sx(w)
	h = g.sy(h)
	radius = g.sr(radius)
	// 绘制中心矩形
	g.drawRectPx(x+radius, y, w-2*radius, h, c)
	g.drawRectPx(x, y+radius, w, h-2*radius, c)

	// 绘制四个圆角
	g.drawFilledCircleCorner(x+radius, y+radius, radius, c, 2)     // 左上
	g.drawFilledCircleCorner(x+w-radius, y+radius, radius, c, 1)   // 右上
	g.drawFilledCircleCorner(x+radius, y+h-radius, radius, c, 3)   // 左下
	g.drawFilledCircleCorner(x+w-radius, y+h-radius, radius, c, 4) // 右下
}

// drawFilledCircleCorner 绘制圆角（四分之一圆）
func (g *Graphics) drawFilledCircleCorner(cx, cy, r int, c color.Color, quadrant int) {
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy <= r*r {
				var px, py int
				switch quadrant {
				case 1: // 右上
					if dx >= 0 && dy <= 0 {
						px, py = cx+dx, cy+dy
					} else {
						continue
					}
				case 2: // 左上
					if dx <= 0 && dy <= 0 {
						px, py = cx+dx, cy+dy
					} else {
						continue
					}
				case 3: // 左下
					if dx <= 0 && dy >= 0 {
						px, py = cx+dx, cy+dy
					} else {
						continue
					}
				case 4: // 右下
					if dx >= 0 && dy >= 0 {
						px, py = cx+dx, cy+dy
					} else {
						continue
					}
				}

				if px >= 0 && px < g.buffer.Bounds().Dx() && py >= 0 && py < g.buffer.Bounds().Dy() {
					g.buffer.Set(px, py, c)
				}
			}
		}
	}
}

// DrawCircle 绘制实心圆
func (g *Graphics) DrawCircle(cx, cy, r int, c color.Color) {
	cx = g.sx(cx)
	cy = g.sy(cy)
	r = g.sr(r)
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy <= r*r {
				if x >= 0 && x < g.buffer.Bounds().Dx() && y >= 0 && y < g.buffer.Bounds().Dy() {
					g.buffer.Set(x, y, c)
				}
			}
		}
	}
}

// GradientDirection 渐变方向
type GradientDirection int

const (
	GradientVertical GradientDirection = iota
	GradientHorizontal
)

// DrawGradient 绘制渐变
func (g *Graphics) DrawGradient(x, y, w, h int, colors []color.Color, direction GradientDirection) {
	if len(colors) < 2 {
		return
	}

	x = g.sx(x)
	y = g.sy(y)
	w = g.sx(w)
	h = g.sy(h)

	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			var t float64
			if direction == GradientVertical {
				t = float64(py) / float64(h)
			} else {
				t = float64(px) / float64(w)
			}

			// 计算当前位置在哪两个颜色之间
			segmentCount := len(colors) - 1
			segment := t * float64(segmentCount)
			segmentIndex := int(segment)
			if segmentIndex >= segmentCount {
				segmentIndex = segmentCount - 1
			}

			localT := segment - float64(segmentIndex)

			c1 := colors[segmentIndex]
			c2 := colors[segmentIndex+1]

			r1, g1, b1, a1 := c1.RGBA()
			r2, g2, b2, a2 := c2.RGBA()

			r := uint8((float64(r1>>8)*(1-localT) + float64(r2>>8)*localT))
			gb := uint8((float64(g1>>8)*(1-localT) + float64(g2>>8)*localT))
			b := uint8((float64(b1>>8)*(1-localT) + float64(b2>>8)*localT))
			a := uint8((float64(a1>>8)*(1-localT) + float64(a2>>8)*localT))

			finalX, finalY := x+px, y+py
			if finalX >= 0 && finalX < g.buffer.Bounds().Dx() && finalY >= 0 && finalY < g.buffer.Bounds().Dy() {
				g.buffer.Set(finalX, finalY, color.RGBA{r, gb, b, a})
			}
		}
	}
}

// DrawText 绘制文本（简单位图字体）
func (g *Graphics) DrawText(text string, x, y int, c color.Color, size int) {
	// 使用简单的 8x8 位图字体
	for i, ch := range text {
		g.drawChar(ch, x+i*size, y, c, size)
	}
}

// DrawTextTTF 使用 TrueType 字体绘制文本
func (g *Graphics) DrawTextTTF(text string, x, y int, c color.Color, size float64, weight FontWeight) error {
	fm := GetFontManager()
	ttfFont := fm.GetFont(weight)
	
	if ttfFont == nil {
		// 回退到位图字体
		g.DrawText(text, x, y, c, int(size))
		return nil
	}

	// 渲染层按屏幕缩放，保持布局仍以 480 逻辑坐标计算
	sx := float64(x) * g.scaleX
	sy := float64(y) * g.scaleY
	sz := size * g.scaleX
	if sz < 1 {
		sz = 1
	}

	// 创建 freetype context
	ctx := freetype.NewContext()
	ctx.SetDPI(72)
	ctx.SetFont(ttfFont)
	ctx.SetFontSize(sz)
	ctx.SetClip(g.buffer.Bounds())
	ctx.SetDst(g.buffer)
	ctx.SetSrc(&image.Uniform{c})

	// 绘制文本
	pt := freetype.Pt(int(math.Round(sx)), int(math.Round(sy+sz)))
	_, err := ctx.DrawString(text, pt)
	
	return err
}

// MeasureText 测量文本宽度
func (g *Graphics) MeasureText(text string, size float64, weight FontWeight) int {
	fm := GetFontManager()
	ttfFont := fm.GetFont(weight)
	
	if ttfFont == nil {
		// 回退到位图字体估算
		return len(text) * int(size) / 2
	}

	face := truetype.NewFace(ttfFont, &truetype.Options{
		Size: size,
		DPI:  72,
	})
	defer face.Close()

	width := fixed.Int26_6(0)
	for _, ch := range text {
		advance, ok := face.GlyphAdvance(ch)
		if !ok {
			width += fixed.Int26_6(int(size) * 64 / 2) // 估算
			continue
		}
		width += advance
	}

	return int(width >> 6)
}

// drawChar 绘制单个字符
func (g *Graphics) drawChar(ch rune, x, y int, c color.Color, size int) {
	if ch < 32 || ch > 126 {
		ch = '?' // 不支持的字符显示为 ?
	}

}

// 简单的 8x8 ASCII 位图字体
var font8x8 = [95][8]byte{
	{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, // Space
	{0x18, 0x3C, 0x3C, 0x18, 0x18, 0x00, 0x18, 0x00}, // !
	{0x36, 0x36, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, // "
	{0x36, 0x36, 0x7F, 0x36, 0x7F, 0x36, 0x36, 0x00}, // #
	{0x0C, 0x3E, 0x03, 0x1E, 0x30, 0x1F, 0x0C, 0x00}, // $
	{0x00, 0x63, 0x33, 0x18, 0x0C, 0x66, 0x63, 0x00}, // %
	{0x1C, 0x36, 0x1C, 0x6E, 0x3B, 0x33, 0x6E, 0x00}, // &
	{0x06, 0x06, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00}, // '
	{0x18, 0x0C, 0x06, 0x06, 0x06, 0x0C, 0x18, 0x00}, // (
	{0x06, 0x0C, 0x18, 0x18, 0x18, 0x0C, 0x06, 0x00}, // )
	{0x00, 0x66, 0x3C, 0xFF, 0x3C, 0x66, 0x00, 0x00}, // *
	{0x00, 0x0C, 0x0C, 0x3F, 0x0C, 0x0C, 0x00, 0x00}, // +
	{0x00, 0x00, 0x00, 0x00, 0x00, 0x0C, 0x0C, 0x06}, // ,
	{0x00, 0x00, 0x00, 0x3F, 0x00, 0x00, 0x00, 0x00}, // -
	{0x00, 0x00, 0x00, 0x00, 0x00, 0x0C, 0x0C, 0x00}, // .
	{0x60, 0x30, 0x18, 0x0C, 0x06, 0x03, 0x01, 0x00}, // /
	{0x3E, 0x63, 0x73, 0x7B, 0x6F, 0x67, 0x3E, 0x00}, // 0
	{0x0C, 0x0E, 0x0C, 0x0C, 0x0C, 0x0C, 0x3F, 0x00}, // 1
	{0x1E, 0x33, 0x30, 0x1C, 0x06, 0x33, 0x3F, 0x00}, // 2
	{0x1E, 0x33, 0x30, 0x1C, 0x30, 0x33, 0x1E, 0x00}, // 3
	{0x38, 0x3C, 0x36, 0x33, 0x7F, 0x30, 0x78, 0x00}, // 4
	{0x3F, 0x03, 0x1F, 0x30, 0x30, 0x33, 0x1E, 0x00}, // 5
	{0x1C, 0x06, 0x03, 0x1F, 0x33, 0x33, 0x1E, 0x00}, // 6
	{0x3F, 0x33, 0x30, 0x18, 0x0C, 0x0C, 0x0C, 0x00}, // 7
	{0x1E, 0x33, 0x33, 0x1E, 0x33, 0x33, 0x1E, 0x00}, // 8
	{0x1E, 0x33, 0x33, 0x3E, 0x30, 0x18, 0x0E, 0x00}, // 9
	{0x00, 0x0C, 0x0C, 0x00, 0x00, 0x0C, 0x0C, 0x00}, // :
	{0x00, 0x0C, 0x0C, 0x00, 0x00, 0x0C, 0x0C, 0x06}, // ;
	{0x18, 0x0C, 0x06, 0x03, 0x06, 0x0C, 0x18, 0x00}, // <
	{0x00, 0x00, 0x3F, 0x00, 0x00, 0x3F, 0x00, 0x00}, // =
	{0x06, 0x0C, 0x18, 0x30, 0x18, 0x0C, 0x06, 0x00}, // >
	{0x1E, 0x33, 0x30, 0x18, 0x0C, 0x00, 0x0C, 0x00}, // ?
	{0x3E, 0x63, 0x7B, 0x7B, 0x7B, 0x03, 0x1E, 0x00}, // @
	{0x0C, 0x1E, 0x33, 0x33, 0x3F, 0x33, 0x33, 0x00}, // A
	{0x3F, 0x66, 0x66, 0x3E, 0x66, 0x66, 0x3F, 0x00}, // B
	{0x3C, 0x66, 0x03, 0x03, 0x03, 0x66, 0x3C, 0x00}, // C
	{0x1F, 0x36, 0x66, 0x66, 0x66, 0x36, 0x1F, 0x00}, // D
	{0x7F, 0x46, 0x16, 0x1E, 0x16, 0x46, 0x7F, 0x00}, // E
	{0x7F, 0x46, 0x16, 0x1E, 0x16, 0x06, 0x0F, 0x00}, // F
	{0x3C, 0x66, 0x03, 0x03, 0x73, 0x66, 0x7C, 0x00}, // G
	{0x33, 0x33, 0x33, 0x3F, 0x33, 0x33, 0x33, 0x00}, // H
	{0x1E, 0x0C, 0x0C, 0x0C, 0x0C, 0x0C, 0x1E, 0x00}, // I
	{0x78, 0x30, 0x30, 0x30, 0x33, 0x33, 0x1E, 0x00}, // J
	{0x67, 0x66, 0x36, 0x1E, 0x36, 0x66, 0x67, 0x00}, // K
	{0x0F, 0x06, 0x06, 0x06, 0x46, 0x66, 0x7F, 0x00}, // L
	{0x63, 0x77, 0x7F, 0x7F, 0x6B, 0x63, 0x63, 0x00}, // M
	{0x63, 0x67, 0x6F, 0x7B, 0x73, 0x63, 0x63, 0x00}, // N
	{0x1C, 0x36, 0x63, 0x63, 0x63, 0x36, 0x1C, 0x00}, // O
	{0x3F, 0x66, 0x66, 0x3E, 0x06, 0x06, 0x0F, 0x00}, // P
	{0x1E, 0x33, 0x33, 0x33, 0x3B, 0x1E, 0x38, 0x00}, // Q
	{0x3F, 0x66, 0x66, 0x3E, 0x36, 0x66, 0x67, 0x00}, // R
	{0x1E, 0x33, 0x07, 0x0E, 0x38, 0x33, 0x1E, 0x00}, // S
	{0x3F, 0x2D, 0x0C, 0x0C, 0x0C, 0x0C, 0x1E, 0x00}, // T
	{0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x3F, 0x00}, // U
	{0x33, 0x33, 0x33, 0x33, 0x33, 0x1E, 0x0C, 0x00}, // V
	{0x63, 0x63, 0x63, 0x6B, 0x7F, 0x77, 0x63, 0x00}, // W
	{0x63, 0x63, 0x36, 0x1C, 0x1C, 0x36, 0x63, 0x00}, // X
	{0x33, 0x33, 0x33, 0x1E, 0x0C, 0x0C, 0x1E, 0x00}, // Y
	{0x7F, 0x63, 0x31, 0x18, 0x4C, 0x66, 0x7F, 0x00}, // Z
	{0x1E, 0x06, 0x06, 0x06, 0x06, 0x06, 0x1E, 0x00}, // [
	{0x03, 0x06, 0x0C, 0x18, 0x30, 0x60, 0x40, 0x00}, // \
	{0x1E, 0x18, 0x18, 0x18, 0x18, 0x18, 0x1E, 0x00}, // ]
	{0x08, 0x1C, 0x36, 0x63, 0x00, 0x00, 0x00, 0x00}, // ^
	{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF}, // _
	{0x0C, 0x0C, 0x18, 0x00, 0x00, 0x00, 0x00, 0x00}, // `
	{0x00, 0x00, 0x1E, 0x30, 0x3E, 0x33, 0x6E, 0x00}, // a
	{0x07, 0x06, 0x06, 0x3E, 0x66, 0x66, 0x3B, 0x00}, // b
	{0x00, 0x00, 0x1E, 0x33, 0x03, 0x33, 0x1E, 0x00}, // c
	{0x38, 0x30, 0x30, 0x3e, 0x33, 0x33, 0x6E, 0x00}, // d
	{0x00, 0x00, 0x1E, 0x33, 0x3f, 0x03, 0x1E, 0x00}, // e
	{0x1C, 0x36, 0x06, 0x0f, 0x06, 0x06, 0x0F, 0x00}, // f
	{0x00, 0x00, 0x6E, 0x33, 0x33, 0x3E, 0x30, 0x1F}, // g
	{0x07, 0x06, 0x36, 0x6E, 0x66, 0x66, 0x67, 0x00}, // h
	{0x0C, 0x00, 0x0E, 0x0C, 0x0C, 0x0C, 0x1E, 0x00}, // i
	{0x30, 0x00, 0x30, 0x30, 0x30, 0x33, 0x33, 0x1E}, // j
	{0x07, 0x06, 0x66, 0x36, 0x1E, 0x36, 0x67, 0x00}, // k
	{0x0E, 0x0C, 0x0C, 0x0C, 0x0C, 0x0C, 0x1E, 0x00}, // l
	{0x00, 0x00, 0x33, 0x7F, 0x7F, 0x6B, 0x63, 0x00}, // m
	{0x00, 0x00, 0x1F, 0x33, 0x33, 0x33, 0x33, 0x00}, // n
	{0x00, 0x00, 0x1E, 0x33, 0x33, 0x33, 0x1E, 0x00}, // o
	{0x00, 0x00, 0x3B, 0x66, 0x66, 0x3E, 0x06, 0x0F}, // p
	{0x00, 0x00, 0x6E, 0x33, 0x33, 0x3E, 0x30, 0x78}, // q
	{0x00, 0x00, 0x3B, 0x6E, 0x66, 0x06, 0x0F, 0x00}, // r
	{0x00, 0x00, 0x3E, 0x03, 0x1E, 0x30, 0x1F, 0x00}, // s
	{0x08, 0x0C, 0x3E, 0x0C, 0x0C, 0x2C, 0x18, 0x00}, // t
	{0x00, 0x00, 0x33, 0x33, 0x33, 0x33, 0x6E, 0x00}, // u
	{0x00, 0x00, 0x33, 0x33, 0x33, 0x1E, 0x0C, 0x00}, // v
	{0x00, 0x00, 0x63, 0x6B, 0x7F, 0x7F, 0x36, 0x00}, // w
	{0x00, 0x00, 0x63, 0x36, 0x1C, 0x36, 0x63, 0x00}, // x
	{0x00, 0x00, 0x33, 0x33, 0x33, 0x3E, 0x30, 0x1F}, // y
	{0x00, 0x00, 0x3F, 0x19, 0x0C, 0x26, 0x3F, 0x00}, // z
	{0x38, 0x0C, 0x0C, 0x07, 0x0C, 0x0C, 0x38, 0x00}, // {
	{0x18, 0x18, 0x18, 0x00, 0x18, 0x18, 0x18, 0x00}, // |
	{0x07, 0x0C, 0x0C, 0x38, 0x0C, 0x0C, 0x07, 0x00}, // }
	{0x6E, 0x3B, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, // ~
}
