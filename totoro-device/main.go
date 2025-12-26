package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"totoro-device/config"
	"totoro-device/internal/api"
	"totoro-device/internal/bridgeclient"
	"totoro-device/internal/database"
	"totoro-device/internal/display"
	"totoro-device/internal/frp"
	"totoro-device/internal/logger"
	"totoro-device/internal/network"
	"totoro-device/internal/probe"
	"totoro-device/internal/realtime"
	"totoro-device/internal/version"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

func pickDeviceMAC(netManager network.Manager) string {
	// 优先取“当前接口”的 MAC
	if st, err := netManager.GetNetworkStatus(); err == nil && st != nil {
		ifaces, _ := netManager.GetInterfaces()
		cur := strings.TrimSpace(st.CurrentInterface)
		if cur != "" {
			for _, it := range ifaces {
				if strings.TrimSpace(it.Name) == cur && strings.TrimSpace(it.MAC) != "" {
					return strings.TrimSpace(it.MAC)
				}
			}
		}
		// 回退：取第一个有 MAC 的接口
		for _, it := range ifaces {
			if strings.TrimSpace(it.MAC) != "" {
				return strings.TrimSpace(it.MAC)
			}
		}
	}
	return ""
}

func bridgeRegisterIfConfigured(cfg *config.Config, netManager network.Manager) {
	base := strings.TrimRight(strings.TrimSpace(cfg.Bridge.URL), "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(os.Getenv("TOTOTO_BRIDGE_URL")), "/")
	}
	if base == "" || cfg == nil {
		return
	}
	deviceID := strings.TrimSpace(cfg.Device.ID)
	if deviceID == "" {
		return
	}
	mac := pickDeviceMAC(netManager)
	if mac == "" {
		logger.Warn("桥梁注册跳过：无法获取设备 MAC")
		return
	}

	// 先尝试复用 SQLite 中的 session（未过期则不重复注册）
	if db := database.GetDB(); db != nil {
		if sess, err := database.GetBridgeSession(db); err == nil && sess != nil {
			if strings.TrimRight(strings.TrimSpace(sess.BridgeURL), "/") == base &&
				strings.TrimSpace(sess.DeviceID) == deviceID &&
				!database.BridgeSessionExpired(sess, 30*time.Second) {
				cfg.Bridge.URL = base
				cfg.Bridge.DeviceToken = strings.TrimSpace(sess.DeviceToken)
				cfg.Bridge.ExpiresAt = time.Unix(sess.ExpiresAt, 0).UTC().Format(time.RFC3339)
				cfg.Bridge.LastMAC = strings.TrimSpace(sess.MAC)
				_ = cfg.Save()
				logger.Info("桥梁 session 复用：device_id=%s", deviceID)
				return
			}
		}
	}

	c := &bridgeclient.Client{BaseURL: base}
	resp, err := c.Register(deviceID, mac)
	if err != nil {
		logger.Error("桥梁注册失败: %v", err)
		return
	}
	if resp != nil && strings.TrimSpace(resp.DeviceToken) != "" {
		// 持久化到本地配置（便于 UI 展示）
		cfg.Bridge.URL = base
		cfg.Bridge.DeviceToken = strings.TrimSpace(resp.DeviceToken)
		cfg.Bridge.ExpiresAt = strings.TrimSpace(resp.ExpiresAt)
		cfg.Bridge.LastMAC = mac
		_ = cfg.Save()
		// 持久化到 SQLite（source of truth）
		if db := database.GetDB(); db != nil {
			_ = database.UpsertBridgeSession(db, database.BridgeSession{
				BridgeURL:   base,
				DeviceID:    deviceID,
				MAC:         mac,
				DeviceToken: strings.TrimSpace(resp.DeviceToken),
				ExpiresAt:   bridgeclient.ParseExpiresAt(resp.ExpiresAt),
			})
		}
		logger.Info("桥梁注册成功：device_id=%s mac=%s", deviceID, mac)
	}
}

func ensurePortAvailable(port int) error {
	// 先试探性监听（不真正启动服务），用于判断端口是否可用
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err == nil {
		_ = ln.Close()
		return nil
	}

	// 非“占用”类错误直接返回（例如权限问题）
	if !strings.Contains(strings.ToLower(err.Error()), "address already in use") {
		return err
	}

	// 端口占用：尝试杀掉占用进程（macOS/linux）
	out, e := exec.Command("lsof", "-ti", fmt.Sprintf("tcp:%d", port)).CombinedOutput()
	if e != nil {
		// lsof 不可用时直接返回原错误
		return err
	}

	pids := strings.Fields(strings.TrimSpace(string(out)))
	for _, pidStr := range pids {
		pid, pe := strconv.Atoi(strings.TrimSpace(pidStr))
		if pe != nil || pid <= 1 {
			continue
		}
		// SIGKILL 直接释放端口（避免交互）
		_ = exec.Command("kill", "-9", strconv.Itoa(pid)).Run()
	}

	// 再尝试一次
	ln2, err2 := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err2 == nil {
		_ = ln2.Close()
		return nil
	}
	return err2
}

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
	// 可选启动屏幕交互系统（macOS 预览用 SDL2；Linux 设备用 FB）
	defaultDisplay := runtime.GOOS == "linux"
	enableDisplay := flag.Bool("display", defaultDisplay, "启用屏幕交互系统（macOS 需用 -tags preview 编译）")
	flag.Parse()

	// SDL 在 macOS 必须占用主线程：如果启用 display，就锁定主线程
	if *enableDisplay && runtime.GOOS == "darwin" {
		runtime.LockOSThread()
	}

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
	// 启动时向桥梁注册（白名单校验），获取 device_token 后用于拉官方/公开节点列表
	bridgeRegisterIfConfigured(cfg, netManager)

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
	frp.SetGlobalClient(frpClient)

	// 初始化HTTP API服务器
	apiServer := api.NewServer(cfg, db, netManager, frpClient)

	// 确保端口可用：若被占用则杀掉占用进程（按要求不改端口）
	if err := ensurePortAvailable(cfg.Server.Port); err != nil {
		logger.Warn("端口 %d 可能不可用: %v", cfg.Server.Port, err)
	}

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

	// 启动后自动连接 FRP（取决于用户选择的 mode；选择后保持，并自动重连）
	go func() {
		time.Sleep(800 * time.Millisecond) // 给 node/bridge/network 留一点启动缓冲

		// 确保 Active 字段与 mode 同步
		cfg.FRPServer.SyncActiveFromMode()

		switch cfg.FRPServer.Mode {
		case config.FRPModePublic:
			// public：自动换票并连接（需要 node_api + invite_code）
			if strings.TrimSpace(cfg.FRPServer.Public.NodeAPI) == "" || strings.TrimSpace(cfg.FRPServer.Public.InviteCode) == "" {
				logger.Warn("FRP(public) 未配置 node_api/invite_code，跳过自动连接")
				return
			}
			if cfg.FRPServer.Public.TicketExpiresAt == "" || frp.TicketExpired(cfg.FRPServer.Public.TicketExpiresAt, 30*time.Second) || strings.TrimSpace(cfg.FRPServer.Public.TotoroTicket) == "" {
				res, err := frp.ResolveInviteToTicket(cfg.FRPServer.Public.NodeAPI, cfg.FRPServer.Public.InviteCode)
				if err != nil {
					cfg.FRPServer.Public.LastResolveError = err.Error()
					_ = cfg.Save()
					logger.Error("FRP(public) 自动换票失败: %v", err)
					return
				}
				cfg.FRPServer.Public.LastResolveError = ""
				cfg.FRPServer.Public.Server = res.Server
				cfg.FRPServer.Public.TotoroTicket = res.Ticket
				cfg.FRPServer.Public.TicketExpiresAt = res.ExpiresAtRFC
				// public 模式下不使用 token
				cfg.FRPServer.Public.Token = ""
				cfg.FRPServer.Mode = config.FRPModePublic
				cfg.FRPServer.SyncActiveFromMode()
				_ = cfg.Save()
			}
		default:
			// builtin/manual：直接用当前 Active 配置连接
			if strings.TrimSpace(cfg.FRPServer.Server) == "" {
				logger.Warn("FRP(%s) 未配置 server，跳过自动连接", cfg.FRPServer.Mode)
				return
			}
		}

		if err := frpClient.Connect(); err != nil {
			logger.Error("FRP自动连接失败: %v", err)
		} else {
			logger.Info("FRP自动连接成功")
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	var disp display.Display
	var mgr *display.Manager

	// 启动屏幕交互系统（与主程序共享 cfg/netManager/frpClient）
	if *enableDisplay {
		// 预览/设备：统一使用 720x720 逻辑分辨率；若设备真实 fb 非 720，会在 fb.Update 中做缩放映射
		w, h := 480, 480
		if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
			w, h = 720, 720
		}
		d, err := display.NewDisplay("NWCT Display Preview", w, h)
		if err != nil {
			logger.Error("初始化显示失败: %v", err)
		} else {
			disp = d
			services := display.NewAppServices(cfg, netManager, frpClient)
			mgr = display.NewManagerWithServices(disp, services)
		}
	} else if runtime.GOOS == "darwin" {
		// macOS 上如果你直接运行 ./nwct-client 而未加 -display，这里给个明确提示
		logger.Warn("屏幕UI未启用：请使用 -display 启动；并用 go build -tags preview 编译以启用 SDL2 预览")
	}

	// 先监听信号，再让 UI（若启用）占主线程运行
	go func() {
		<-quit
		logger.Info("正在关闭服务...")
		if mgr != nil {
			mgr.Stop()
		}
	}()

	// UI 主循环占用主线程（macOS SDL 要求）
	if mgr != nil {
		if err := mgr.Run(); err != nil {
			logger.Error("屏幕交互系统运行错误: %v", err)
		}
	} else {
		// 未启用 display：阻塞等待退出信号
		<-quit
		logger.Info("正在关闭服务...")
	}

	// 关闭显示
	if disp != nil {
		_ = disp.Close()
	}

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
