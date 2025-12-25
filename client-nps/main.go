package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"nwct/client-nps/config"
	"nwct/client-nps/internal/api"
	"nwct/client-nps/internal/database"
	"nwct/client-nps/internal/frp"
	"nwct/client-nps/internal/logger"
	"nwct/client-nps/internal/network"
	"nwct/client-nps/internal/probe"
	"nwct/client-nps/internal/realtime"
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
		ticker := time.NewTicker(30 * time.Second) // 从 10 秒增加到 30 秒，减少内存占用和 CPU 使用
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
			sshListening := false
			if conn, err := net.DialTimeout("tcp", "127.0.0.1:22", 200*time.Millisecond); err == nil {
				sshListening = true
				_ = conn.Close()
			}

			realtime.Default().Broadcast("system_status", map[string]interface{}{
				"hostname":         hostname,
				"firmware_version": version.Version,
				"device_id":        cfg.Device.ID,
				"uptime":           uptimeSec,
				"cpu_usage":        cpuUsage,
				"memory_usage":     memUsage,
				"disk_usage":       diskUsage,
				"ssh": map[string]interface{}{
					"listening": sshListening,
					"port":      22,
				},
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

	// 初始化FRP客户端
	frpClient := frp.NewClient(&cfg.FRPServer)

	// 初始化HTTP API服务器
	apiServer := api.NewServer(cfg, db, netManager, frpClient)

	// 创建HTTP服务器，设置内存优化参数
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: apiServer.Router(),
		// 限制连接参数以节省内存
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 12, // 4KB 头部限制（从默认 1MB 降到 4KB）
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
		// 连接FRP
		if err := frpClient.Connect(); err != nil {
			logger.Error("FRP连接失败: %v", err)
		} else {
			logger.Info("FRP连接成功")
		}
	}

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭服务...")

	// 关闭FRP连接
	if frpClient.IsConnected() {
		frpClient.Disconnect()
	}

	// 关闭HTTP服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatal("HTTP服务器关闭失败:", err)
	}

	logger.Info("服务已关闭")
}
