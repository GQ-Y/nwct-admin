package display

// touchReader 触摸事件读取器（用于 Linux evdev；非 Linux 下为 nil）
type touchReader interface {
	Init() error
	Poll() []TouchEvent
	Close() error
}


