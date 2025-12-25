package display

// 本文件不直接依赖标准库颜色，统一使用 ui_theme.go 中的颜色变量

// ListItem 列表项数据
type ListItem struct {
	Title    string
	Subtitle string
	Value    string      // 右侧文字
	Icon     func(*Graphics, int, int) // 自定义图标绘制函数 (可选)
	OnClick  func()
	ShowArrow bool       // 是否显示右侧箭头
}

// ListView 列表组件
type ListView struct {
	items      []*ListItem
	x, y       int
	width      int
	height     int
	itemHeight int
	offsetY    int // 滚动偏移量
	
	// 交互状态
	pressedIndex int
	dragging     bool
	dragStartY   int
	lastDragY    int
}

// NewListView 创建列表
func NewListView(x, y, width, height int) *ListView {
	return &ListView{
		items:      make([]*ListItem, 0),
		x:          x,
		y:          y,
		width:      width,
		height:     height,
		itemHeight: 72,
		pressedIndex: -1,
		dragging:   false,
		dragStartY: 0,
		lastDragY:  0,
	}
}

// AddItem 添加列表项
func (lv *ListView) AddItem(item *ListItem) {
	lv.items = append(lv.items, item)
}

// Clear 清空列表
func (lv *ListView) Clear() {
	lv.items = make([]*ListItem, 0)
	lv.offsetY = 0
	lv.pressedIndex = -1
	lv.dragging = false
}

// Render 渲染列表
func (lv *ListView) Render(g *Graphics) {
	// 裁剪区域 (模拟) - 实际 Graphics 库可能不支持 Clip，只能画在范围内
	// 这里假设 ListView 占据除了 NavBar 外的区域
	
	startY := lv.y + lv.offsetY
	
	for i, item := range lv.items {
		itemY := startY + i*lv.itemHeight
		
		// 简单的可见性剔除
		if itemY+lv.itemHeight < lv.y || itemY > lv.y+lv.height {
			continue
		}
		
		// 绘制按压背景
		if i == lv.pressedIndex {
			g.DrawRect(lv.x, itemY, lv.width, lv.itemHeight, ColorPressed)
		}
		
		// 图标区域 (预留 50px)
		contentX := lv.x + 24
		if item.Icon != nil {
			item.Icon(g, contentX, itemY+lv.itemHeight/2)
			contentX += 40
		}
		
		// 标题
		titleY := itemY + 30
		if item.Subtitle != "" {
			titleY = itemY + 24
		}
		g.DrawTextTTF(item.Title, contentX, titleY, ColorTextPrimary, 18, FontWeightRegular)
		
		// 副标题
		if item.Subtitle != "" {
			g.DrawTextTTF(item.Subtitle, contentX, itemY+52, ColorTextSecondary, 14, FontWeightRegular)
		}
		
		// 右侧区域
		rightX := lv.x + lv.width - 24
		
		// 箭头 ">"
		if item.ShowArrow {
			arrowY := itemY + lv.itemHeight/2
			arrowSize := 6
			// 简单的 > 形
			for k := 0; k < 2; k++ {
				g.DrawLine(rightX-arrowSize+k, arrowY-arrowSize, rightX+k, arrowY, ColorTextLight)
				g.DrawLine(rightX+k, arrowY, rightX-arrowSize+k, arrowY+arrowSize, ColorTextLight)
			}
			rightX -= 20
		}
		
		// 右侧数值
		if item.Value != "" {
			valW := g.MeasureText(item.Value, 16, FontWeightRegular)
			g.DrawTextTTF(item.Value, rightX-valW, itemY+lv.itemHeight/2+6, ColorTextSecondary, 16, FontWeightRegular)
		}
		
		// 分隔线
		if i < len(lv.items)-1 {
			lineX := contentX
			g.DrawRect(lineX, itemY+lv.itemHeight-1, lv.width-lineX, 1, ColorSeparator)
		}
	}
}

// HandleTouch 处理触摸
func (lv *ListView) HandleTouch(x, y int, touchType TouchType) bool {
	// 简单的点击处理，暂不支持惯性滚动
	if x < lv.x || x > lv.x+lv.width || y < lv.y || y > lv.y+lv.height {
		return false
	}
	
	// 拖动滚动：基于 TouchMove 的 dy
	if touchType == TouchDown {
		lv.dragging = false
		lv.dragStartY = y
		lv.lastDragY = y
	}
	if touchType == TouchMove {
		dy := y - lv.lastDragY
		if !lv.dragging {
			d := y - lv.dragStartY
			if d < 0 {
				d = -d
			}
			if d > 6 {
				lv.dragging = true
			}
		}
		if lv.dragging {
			lv.Scroll(dy)
			lv.pressedIndex = -1
			lv.lastDragY = y
			return true
		}
		lv.lastDragY = y
	}
	if touchType == TouchUp && lv.dragging {
		lv.dragging = false
		lv.pressedIndex = -1
		return true
	}

	// 计算点击了哪个 Item
	// y = startY + index * h => index = (y - startY) / h
	// startY = lv.y + lv.offsetY
	relativeY := y - (lv.y + lv.offsetY)
	index := relativeY / lv.itemHeight
	
	if index >= 0 && index < len(lv.items) {
		if touchType == TouchDown {
			lv.pressedIndex = index
			return true
		} else if touchType == TouchUp {
			if lv.pressedIndex == index {
				// 触发点击
				if lv.items[index].OnClick != nil {
					lv.items[index].OnClick()
				}
			}
			lv.pressedIndex = -1
			return true
		}
	}
	
	return false
}

// Scroll 滚动方法
func (lv *ListView) Scroll(deltaY int) {
	lv.offsetY += deltaY
	// 边界检查
	maxOffset := 0
	minOffset := -(len(lv.items)*lv.itemHeight - lv.height)
	if minOffset > 0 { minOffset = 0 }
	
	if lv.offsetY > maxOffset { lv.offsetY = maxOffset }
	if lv.offsetY < minOffset { lv.offsetY = minOffset }
}

