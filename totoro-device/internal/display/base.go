package display

// TouchType 触摸事件类型
type TouchType int

const (
	TouchDown TouchType = iota
	TouchUp
	TouchMove
)

// TouchEvent 触摸事件
type TouchEvent struct {
	Type      TouchType
	X         int
	Y         int
	Timestamp int64
}

type GraphicsInterface interface{}

// Page 页面接口
type Page interface {
	Render(g *Graphics) error
	HandleTouch(x, y int, touchType TouchType) bool
	Update(deltaTime int64)
	OnEnter()
	OnExit()
	GetName() string
}

// BasePage 页面基类
type BasePage struct {
	Name string
}

// GetName 获取页面名称
func (p *BasePage) GetName() string {
	return p.Name
}

// OnEnter 进入页面
func (p *BasePage) OnEnter() {
	// 默认实现
}

// OnExit 退出页面
func (p *BasePage) OnExit() {
	// 默认实现
}

// Update 更新页面
func (p *BasePage) Update(deltaTime int64) {
	// 默认实现
}

// HandleTouch 处理触摸
func (p *BasePage) HandleTouch(x, y int, touchType TouchType) bool {
	return false
}

