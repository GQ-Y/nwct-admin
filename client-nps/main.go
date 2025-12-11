package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nwct/client-nps/config"
	"nwct/client-nps/internal/api"
	"nwct/client-nps/internal/database"
	"nwct/client-nps/internal/logger"
	"nwct/client-nps/internal/mqtt"
	"nwct/client-nps/internal/network"
	"nwct/client-nps/internal/nps"
)

func main() {
	// 初始化日志
	if err := logger.InitLogger(); err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}
	defer logger.Close()
	logger.Info("启动内网穿透盒子客户端...")

	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal("加载配置失败: %v", err)
	}

	// 初始化数据库
	db, err := database.InitDB(cfg.Database.Path)
	if err != nil {
		logger.Fatal("初始化数据库失败: %v", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			logger.Error("关闭数据库失败: %v", err)
		}
	}()

	// 初始化网络管理器
	netManager := network.NewManager()

	// 初始化NPS客户端
	npsClient := nps.NewClient(&cfg.NPSServer)

	// 初始化MQTT客户端
	mqttClient := mqtt.NewClient(&cfg.MQTT)

	// 初始化HTTP API服务器
	apiServer := api.NewServer(cfg, db, netManager, npsClient, mqttClient)

	// 创建HTTP服务器
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: apiServer.Router(),
	}

	// 启动HTTP服务器
	go func() {
		logger.Info("HTTP服务器启动在端口 %d", cfg.Server.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP服务器启动失败: %v", err)
		}
	}()

	// 如果已初始化，启动服务
	if cfg.Initialized {
		// 连接MQTT
		if err := mqttClient.Connect(); err != nil {
			logger.Error("MQTT连接失败: %v", err)
		} else {
			logger.Info("MQTT连接成功")

			// 设置全局客户端用于命令处理
			mqtt.SetGlobalClient(mqttClient)

			// 订阅命令主题
			deviceID := cfg.Device.ID
			commandTopic := fmt.Sprintf("nwct/%s/command", deviceID)
			mqttClient.Subscribe(commandTopic, mqtt.HandleCommandMessage)
		}

		// 连接NPS
		if err := npsClient.Connect(); err != nil {
			logger.Error("NPS连接失败: %v", err)
		} else {
			logger.Info("NPS连接成功")
		}
	}

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭服务...")

	// 关闭MQTT连接
	if mqttClient.IsConnected() {
		mqttClient.Disconnect()
	}

	// 关闭NPS连接
	if npsClient.IsConnected() {
		npsClient.Disconnect()
	}

	// 关闭HTTP服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatal("HTTP服务器关闭失败:", err)
	}

	logger.Info("服务已关闭")
}
