package display

// Display 显示接口
type Display interface {
	Close() error
	GetWidth() int
	GetHeight() int
	Update() error
	GetTouchEvents() []TouchEvent
}
