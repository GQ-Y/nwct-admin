//go:build (linux && device_display) || preview

package main

import (
	"os"
	"runtime"

	"totoro-device/config"
	"totoro-device/internal/display"
	"totoro-device/internal/frp"
	"totoro-device/internal/logger"
	"totoro-device/internal/network"
)

func uiLoop(enableDisplay bool, cfg *config.Config, netManager network.Manager, frpClient frp.Client, quit <-chan os.Signal) {
	var disp display.Display
	var mgr *display.Manager

	// 启动屏幕交互系统（与主程序共享 cfg/netManager/frpClient）
	if enableDisplay {
		// 预览/设备：统一使用 720x720 逻辑分辨率；若设备真实 fb 非 720，会在 fb.Update 中做缩放映射
		w, h := 480, 480
		if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
			w, h = 720, 720
		}
		d, err := display.NewDisplay("Totoro Device", w, h)
		if err != nil {
			logger.Error("初始化显示失败: %v", err)
		} else {
			disp = d
			services := display.NewAppServices(cfg, netManager, frpClient)
			mgr = display.NewManagerWithServices(disp, services)
		}
	} else if runtime.GOOS == "darwin" {
		// macOS 上如果你直接运行而未加 -display，这里给个明确提示
		logger.Warn("屏幕UI未启用：请使用 -display 启动；并用 go build -tags preview 编译以启用 SDL2 预览")
	}

	// UI 主循环占用主线程（macOS SDL 要求）。收到退出信号后 Stop。
	if mgr != nil {
		go func() {
			<-quit
			logger.Info("正在关闭服务...")
			mgr.Stop()
		}()
		if err := mgr.Run(); err != nil {
			logger.Error("屏幕交互系统运行错误: %v", err)
		}
	} else {
		<-quit
		logger.Info("正在关闭服务...")
	}

	// 关闭显示（best-effort）
	if disp != nil {
		_ = disp.Close()
	}
}
