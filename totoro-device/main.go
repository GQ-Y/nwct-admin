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
	"totoro-device/internal/deviceboot"
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
	if cfg == nil {
		return
	}
	base := config.ResolveBridgeBase(cfg)
	if base == "" {
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
	// 无论 bridge 是否可达，都先记录 MAC（仅用于注册/审计，不涉密）
	cfg.Bridge.URL = base
	cfg.Bridge.LastMAC = mac
	// 确保设备侧密钥对存在（用于桥梁加密返回解密）
	var privB64, pubB64 string
	if db := database.GetDB(); db != nil {
		if dc, err := database.GetOrCreateDeviceCrypto(db); err == nil && dc != nil {
			privB64 = strings.TrimSpace(dc.PrivKeyB64)
			pubB64 = strings.TrimSpace(dc.PubKeyB64)
		}
	}
	if privB64 == "" || pubB64 == "" {
		logger.Warn("桥梁注册跳过：设备密钥不可用")
		return
	}

	// 先尝试复用 SQLite 中的 session（未过期则不重复注册）
	if db := database.GetDB(); db != nil {
		if sess, err := database.GetBridgeSession(db); err == nil && sess != nil {
			if strings.TrimRight(strings.TrimSpace(sess.BridgeURL), "/") == base &&
				strings.TrimSpace(sess.DeviceID) == deviceID &&
				!database.BridgeSessionExpired(sess, 30*time.Second) {
				cfg.Bridge.URL = base
				cfg.Bridge.LastMAC = strings.TrimSpace(sess.MAC)
				logger.Info("桥梁 session 复用：device_id=%s", deviceID)
				return
			}
		}
	}

	c := &bridgeclient.Client{
		BaseURL:          base,
		DeviceID:         deviceID,
		DevicePrivKeyB64: privB64,
	}
	resp, err := c.Register(deviceID, mac, pubB64)
	if err != nil {
		logger.Error("桥梁注册失败: %v", err)
		return
	}
	if resp != nil && strings.TrimSpace(resp.DeviceToken) != "" {
		cfg.Bridge.URL = base
		cfg.Bridge.LastMAC = mac
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

		// builtin 模式：从桥梁下发 official_nodes 写入本地配置（不再硬编码默认节点）
		if cfg.FRPServer.Mode == config.FRPModeBuiltin {
			if len(resp.OfficialNodes) == 0 {
				logger.Warn("桥梁未配置官方内置节点（official_nodes 为空），builtin 暂不自动连接")
			} else {
				off := resp.OfficialNodes[0]
				cfg.FRPServer.Builtin = config.FRPProfile{
					Server:       strings.TrimSpace(off.Server),
					Token:        strings.TrimSpace(off.Token),
					AdminAddr:    strings.TrimSpace(off.AdminAddr),
					AdminUser:    strings.TrimSpace(off.AdminUser),
					AdminPwd:     strings.TrimSpace(off.AdminPwd),
					DomainSuffix: strings.TrimPrefix(strings.TrimSpace(off.DomainSuffix), "."),
					HTTPEnabled:  off.HTTPEnabled,
					HTTPSEnabled: off.HTTPSEnabled,
				}
				cfg.FRPServer.SyncActiveFromMode()
				_ = cfg.Save()
				logger.Info("已从桥梁同步官方内置节点：%s", cfg.FRPServer.Builtin.Server)
			}
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

	// 是否偏好 WiFi：如果用户选择的主接口是 wlan/wl，则即便有线已获取 IP 也会优先尝试 WiFi。
	prefer := strings.ToLower(strings.TrimSpace(cfg.Network.Interface))
	preferWiFi := strings.HasPrefix(prefer, "wlan") || strings.HasPrefix(prefer, "wl")

	// 若不偏好 WiFi：当前已有 IP（有线或无线）就不折腾
	if !preferWiFi {
		if st, err := netManager.GetNetworkStatus(); err == nil {
			if st.IP != "" && st.IP != "0.0.0.0" {
				return
			}
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

	// 开机语音（不阻塞启动）
	deviceboot.TryPlayBootAudio()

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
			// public：
			// - 若配置了邀请码：bridge 兑换邀请码 -> 下发短期 ticket
			// - 若选择了公开节点：bridge 为该 public node 签发短期 ticket（无需邀请码）
			db := database.GetDB()
			if db == nil {
				logger.Warn("FRP(public) 数据库未初始化，跳过自动连接")
				return
			}
			code, _ := database.GetPublicInviteCode(db)
			pubNodeID, _ := database.GetPublicNodeID(db)
			if strings.TrimSpace(code) == "" && strings.TrimSpace(pubNodeID) == "" {
				logger.Warn("FRP(public) 未配置邀请码/未选择公开节点，跳过自动连接")
				return
			}
			// 确保 bridge session 可用（过期则重新注册）
			if sess, _ := database.GetBridgeSession(db); sess == nil || database.BridgeSessionExpired(sess, 30*time.Second) {
				bridgeRegisterIfConfigured(cfg, netManager)
			}
			sess, _ := database.GetBridgeSession(db)
			if sess == nil || strings.TrimSpace(sess.DeviceToken) == "" {
				logger.Warn("FRP(public) 缺少 bridge device_token，跳过自动连接")
				return
			}
			dc, err := database.GetOrCreateDeviceCrypto(db)
			if err != nil || dc == nil || strings.TrimSpace(dc.PrivKeyB64) == "" {
				logger.Warn("FRP(public) 设备密钥不可用，跳过自动连接")
				return
			}
			bridgeBase := config.ResolveBridgeBase(cfg)
			if bridgeBase == "" {
				logger.Warn("FRP(public) 未配置 bridge URL，跳过自动连接")
				return
			}
			bc := &bridgeclient.Client{
				BaseURL:          bridgeBase,
				DeviceToken:      strings.TrimSpace(sess.DeviceToken),
				DeviceID:         strings.TrimSpace(cfg.Device.ID),
				DevicePrivKeyB64: strings.TrimSpace(dc.PrivKeyB64),
			}
			// 如果本地已有 ticket 且尚未临近过期，则不必每次启动都换票（避免频繁请求桥梁）
			if strings.TrimSpace(cfg.FRPServer.Public.TotoroTicket) != "" &&
				!frp.TicketExpiredByTokenOrRFC(cfg.FRPServer.Public.TicketExpiresAt, cfg.FRPServer.Public.TotoroTicket, 24*time.Hour) {
				logger.Info("FRP(public) 复用现有 ticket（未临近过期），跳过换票")
			} else if strings.TrimSpace(code) != "" {
				res, err := bc.RedeemInvite(code)
				if err != nil {
					cfg.FRPServer.Public.LastResolveError = err.Error()
					logger.Error("FRP(public) 自动换票失败: %v", err)
					return
				}
				cfg.FRPServer.Public.LastResolveError = ""
				if len(res.Node.Endpoints) == 0 {
					logger.Error("FRP(public) bridge 未返回 endpoints")
					return
				}
				ep := res.Node.Endpoints[0]
				cfg.FRPServer.Public.Server = fmt.Sprintf("%s:%d", strings.TrimSpace(ep.Addr), ep.Port)
				cfg.FRPServer.Public.TotoroTicket = strings.TrimSpace(res.ConnectionTicket)
				cfg.FRPServer.Public.TicketExpiresAt = strings.TrimSpace(res.ExpiresAt)
				cfg.FRPServer.Public.DomainSuffix = strings.TrimPrefix(strings.TrimSpace(res.Node.DomainSuffix), ".")
			} else {
				res, err := bc.ConnectPublicNode(pubNodeID)
				if err != nil {
					cfg.FRPServer.Public.LastResolveError = err.Error()
					logger.Error("FRP(public) 自动换票失败: %v", err)
					return
				}
				cfg.FRPServer.Public.LastResolveError = ""
				if len(res.Node.Endpoints) == 0 {
					logger.Error("FRP(public) bridge 未返回 endpoints")
					return
				}
				ep := res.Node.Endpoints[0]
				cfg.FRPServer.Public.Server = fmt.Sprintf("%s:%d", strings.TrimSpace(ep.Addr), ep.Port)
				cfg.FRPServer.Public.TotoroTicket = strings.TrimSpace(res.ConnectionTicket)
				cfg.FRPServer.Public.TicketExpiresAt = strings.TrimSpace(res.ExpiresAt)
				cfg.FRPServer.Public.DomainSuffix = strings.TrimPrefix(strings.TrimSpace(res.Node.DomainSuffix), ".")
			}
			// public 模式下不使用 token
			cfg.FRPServer.Public.Token = ""
			cfg.FRPServer.Mode = config.FRPModePublic
			cfg.FRPServer.SyncActiveFromMode()
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
