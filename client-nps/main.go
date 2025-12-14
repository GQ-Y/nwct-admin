package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"nwct/client-nps/config"
	"nwct/client-nps/internal/api"
	"nwct/client-nps/internal/database"
	"nwct/client-nps/internal/logger"
	"nwct/client-nps/internal/mqtt"
	"nwct/client-nps/internal/network"
	"nwct/client-nps/internal/nps"
	"nwct/client-nps/internal/probe"
	"nwct/client-nps/internal/realtime"
	"nwct/client-nps/internal/scanner"
	"nwct/client-nps/internal/version"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

func autoConnectWiFi(cfg *config.Config, netManager network.Manager) {
	// 没有 profiles 就跳过
	if cfg == nil || len(cfg.Network.WiFiProfiles) == 0 {
		return
	}

	// 如果当前已有 IP（有线或无线），就不折腾
	if st, err := netManager.GetNetworkStatus(); err == nil {
		if st.IP != "" && st.IP != "0.0.0.0" {
			return
		}
	}

	// 按 priority desc + last_success desc 排序，逐个尝试
	profiles := make([]config.WiFiProfile, 0, len(cfg.Network.WiFiProfiles))
	for _, p := range cfg.Network.WiFiProfiles {
		if p.SSID == "" || !p.AutoConnect {
			continue
		}
		profiles = append(profiles, p)
	}
	sort.SliceStable(profiles, func(i, j int) bool {
		if profiles[i].Priority != profiles[j].Priority {
			return profiles[i].Priority > profiles[j].Priority
		}
		return profiles[i].LastSuccessAt > profiles[j].LastSuccessAt
	})

	for _, p := range profiles {
		logger.Info("尝试自动连接WiFi: ssid=%s priority=%d", p.SSID, p.Priority)

		now := time.Now().Format(time.RFC3339)
		err := netManager.ConfigureWiFi(p.SSID, p.Password)
		// 更新状态字段并保存
		for idx := range cfg.Network.WiFiProfiles {
			if cfg.Network.WiFiProfiles[idx].SSID == p.SSID {
				cfg.Network.WiFiProfiles[idx].LastTriedAt = now
				if err != nil {
					cfg.Network.WiFiProfiles[idx].LastError = err.Error()
				} else {
					cfg.Network.WiFiProfiles[idx].LastError = ""
					cfg.Network.WiFiProfiles[idx].LastSuccessAt = now
				}
				break
			}
		}
		_ = cfg.Save()

		if err != nil {
			logger.Error("自动连接WiFi失败: ssid=%s err=%v", p.SSID, err)
			continue
		}

		// 等待获取 IP（最多 15 秒）
		ok := false
		for i := 0; i < 15; i++ {
			time.Sleep(1 * time.Second)
			if st, err := netManager.GetNetworkStatus(); err == nil && st.IP != "" && st.IP != "0.0.0.0" {
				ok = true
				break
			}
		}
		if ok {
			logger.Info("自动连接WiFi成功: ssid=%s", p.SSID)
			return
		}
		logger.Error("WiFi已配置但未获取到IP，继续尝试下一个: ssid=%s", p.SSID)
	}
}

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
	// 启动时自动连接已保存WiFi（类似电脑记忆网络）
	autoConnectWiFi(cfg, netManager)

	// 系统状态心跳（WebSocket实时推送）
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			uptimeSec, _ := host.Uptime()
			diskUsage := 0.0
			if du, err := disk.Usage("/"); err == nil && du != nil {
				diskUsage = du.UsedPercent
			}
			cpuUsage := 0.0
			if v, err := cpu.Percent(0, false); err == nil && len(v) > 0 {
				cpuUsage = v[0]
			}
			memUsage := 0.0
			if m, err := mem.VirtualMemory(); err == nil && m != nil {
				memUsage = m.UsedPercent
			}
			netStatus, _ := netManager.GetNetworkStatus()
			hostname, _ := os.Hostname()

			realtime.Default().Broadcast("system_status", map[string]interface{}{
				"hostname":     hostname,
				"firmware_version": version.Version,
				"device_id":    cfg.Device.ID,
				"uptime":       uptimeSec,
				"cpu_usage":    cpuUsage,
				"memory_usage": memUsage,
				"disk_usage":   diskUsage,
				"network": map[string]interface{}{
					"interface": netStatus.CurrentInterface,
					"ip":        netStatus.IP,
					"status":    netStatus.Status,
				},
			})

			<-ticker.C
		}
	}()

	// 设备在线/离线探测器（状态变化推送）
	probeCtx, probeCancel := context.WithCancel(context.Background())
	defer probeCancel()
	probe.StartDeviceMonitor(probeCtx, db, probe.MonitorOptions{
		Interval: 60 * time.Second,
		Timeout:  1 * time.Second,
	})

	// 初始化NPS客户端
	npsClient := nps.NewClient(&cfg.NPSServer)

	// 初始化MQTT客户端
	mqttClient := mqtt.NewClient(&cfg.MQTT)
	// 给 MQTT 命令处理注入依赖（scan/config_update 需要）
	mqtt.SetGlobalConfig(cfg)
	mqtt.SetGlobalNetManager(netManager)
	mqtt.SetGlobalScanner(scanner.NewScanner(db))

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
		// 连接MQTT（可通过 mqtt.auto_connect 控制，保证 UI “断开”后不会被启动逻辑自动拉起）
		if cfg.MQTT.AutoConnect {
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
		} else {
			logger.Info("MQTT自动连接已关闭（mqtt.auto_connect=false）")
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
