//go:build !linux || preview

package display

// 非 Linux 或 preview 构建下不启用 evdev 触摸
func newLinuxEvdevTouch(screenW, screenH int) touchReader { return nil }


