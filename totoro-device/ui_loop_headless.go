//go:build !device_display && !preview

package main

import (
	"os"

	"totoro-device/config"
	"totoro-device/internal/frp"
	"totoro-device/internal/logger"
	"totoro-device/internal/network"
)

func uiLoop(_ bool, _ *config.Config, _ network.Manager, _ frp.Client, quit <-chan os.Signal) {
	<-quit
	logger.Info("正在关闭服务...")
}
