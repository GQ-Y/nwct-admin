package display

// 统一使用 ui_theme.go 中的颜色变量

// NavBar 通用导航栏
type NavBar struct {
	title    string
	onBack   func()
	height   int
	width    int
	hasBack  bool // 是否显示返回按钮
}

// textTopForCenter 计算 DrawTextTTF 所需的 topY，使文字在给定高度内垂直居中
// 注意：Graphics.DrawTextTTF 内部会使用 Pt(x, y+size)，因此这里传入的是“文字顶部”而非基线。
func textTopForCenter(containerY, containerH int, fontSize float64) int {
	return containerY + (containerH-int(fontSize))/2
}

// NewNavBar 创建导航栏
func NewNavBar(title string, hasBack bool, width int) *NavBar {
	return &NavBar{
		title:   title,
		hasBack: hasBack,
		width:   width,
		height:  60, // 固定高度
	}
}

// SetOnBack 设置返回回调
func (nb *NavBar) SetOnBack(callback func()) {
	nb.onBack = callback
}

// Render 渲染导航栏
func (nb *NavBar) Render(g *Graphics) {
	// 背景：必须是实体底色，否则页面内容上滑会“透上来”导致返回箭头/标题重叠
	// 这里用纯白 + 底部分割线，贴近鸿蒙/系统导航栏观感
	g.DrawRect(0, 0, nb.width, nb.height, ColorBackgroundStart)
	g.DrawRect(0, nb.height-1, nb.width, 1, ColorSeparator)
	
	// 返回按钮
	if nb.hasBack {
		// 绘制 "<" 符号
		// 简单的线条模拟箭头
		arrowX := 24
		arrowY := nb.height / 2
		size := 14
		
		// 两条线组成箭头
		// 注意：在 720x720 预览缩放下，用“多条偏移线加粗”会产生明显重影感
		// 这里改为单笔画，观感更干净
		g.DrawLine(arrowX+size, arrowY-size, arrowX, arrowY, ColorTextPrimary)
		g.DrawLine(arrowX, arrowY, arrowX+size, arrowY+size, ColorTextPrimary)
		
		// 点击区域提示 (可选)
		// g.DrawRectRounded(10, 10, 40, 40, 8, color.RGBA{0,0,0,10})
	}

	// 标题
	titleX := 24
	if nb.hasBack {
		titleX = 60 // 给返回按钮留出空间
	}
	
	// 标题文字
	titleSize := 22.0
	titleTop := textTopForCenter(0, nb.height, titleSize)
	_ = g.DrawTextTTF(nb.title, titleX, titleTop, ColorTextPrimary, titleSize, FontWeightMedium)
}

// HandleTouch 处理触摸
func (nb *NavBar) HandleTouch(x, y int, touchType TouchType) bool {
	// 如果点击了返回按钮区域 (0,0) -> (60, 60)
	if nb.hasBack && x < 60 && y < nb.height {
		if touchType == TouchUp && nb.onBack != nil {
			nb.onBack()
			return true
		}
		return true // 拦截该区域的所有触摸
	}
	return false
}

