package main

import (
	"fmt"
	"log"
	"runtime"

	"totoro-device/internal/display"
)

func init() {
	// 锁定主线程用于 SDL（macOS 必须）
	runtime.LockOSThread()
}

func main() {
	fmt.Println("🚀 启动 NWCT 显示预览...")

	// 创建显示实例
	w, h := 480, 480
	if runtime.GOOS == "darwin" {
		w, h = 720, 720
	}
	disp, err := display.NewDisplay("NWCT Display Preview", w, h)
	if err != nil {
		log.Fatalf("❌ 初始化显示失败: %v", err)
	}
	defer disp.Close()

	// 创建显示管理器
	manager := display.NewManager(disp)

	// 运行显示主循环
	fmt.Printf("✅ 显示系统已启动，%dx%d 窗口\n", w, h)
	fmt.Println("💡 按 ESC 或关闭窗口退出")

	if err := manager.Run(); err != nil {
		log.Fatalf("❌ 显示运行错误: %v", err)
	}

	fmt.Println("👋 显示系统已关闭")
}
