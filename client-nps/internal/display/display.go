package display

import (
	"image"
)

// Display 显示接口
type Display interface {
	// Init 初始化显示
	Init() error

	// Close 关闭显示
	Close() error

	// GetWidth 获取宽度
	GetWidth() int

	// GetHeight 获取高度
	GetHeight() int

	// GetBackBuffer 获取后缓冲区 (用于绘图)
	GetBackBuffer() *image.RGBA

	// Update 更新显示（将后缓冲区刷新到屏幕）
	Update() error

	// PollEvents 轮询事件（返回是否需要退出）
	PollEvents() (shouldQuit bool)

	// GetTouchEvents 获取触摸事件队列
	GetTouchEvents() []TouchEvent
}
