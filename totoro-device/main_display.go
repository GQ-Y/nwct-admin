//go:build (linux && device_display) || preview

package main

import (
	"flag"
	"runtime"

	"totoro-device/internal/logger"
)

// Display 版本：支持屏幕/UI（用于 Pico Ultra；macOS 预览用 -tags preview）。
func main() {
	// 可选启动屏幕交互系统（macOS 预览用 SDL2；Linux 设备用 FB）
	defaultDisplay := runtime.GOOS == "linux"
	enableDisplay := flag.Bool("display", defaultDisplay, "启用屏幕交互系统（macOS 需用 -tags preview 编译）")
	flag.Parse()

	// SDL 在 macOS 必须占用主线程：如果启用 display，就锁定主线程
	if *enableDisplay && runtime.GOOS == "darwin" {
		runtime.LockOSThread()
	}

	// 早期日志初始化失败时也能有输出（core 里会再次 InitLogger，但那里会 Fatal）
	_ = logger.InitLogger()

	runCore(*enableDisplay)
}
