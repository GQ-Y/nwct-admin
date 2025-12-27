//go:build !device_display && !preview

package main

// Headless 版本：不编译屏幕/UI 相关代码（用于 Pico Plus/Pro 等无屏板子）。
func main() {
	runCore(false)
}
