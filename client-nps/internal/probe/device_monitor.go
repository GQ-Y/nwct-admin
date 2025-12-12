package probe

import (
	"context"
	"database/sql"
	"net"
	"strings"
	"time"

	"nwct/client-nps/internal/database"
	"nwct/client-nps/internal/logger"
	"nwct/client-nps/internal/realtime"
)

type MonitorOptions struct {
	Interval time.Duration
	Timeout  time.Duration
}

func StartDeviceMonitor(ctx context.Context, db *sql.DB, opts MonitorOptions) {
	if opts.Interval <= 0 {
		opts.Interval = 60 * time.Second
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 1 * time.Second
	}

	go func() {
		ticker := time.NewTicker(opts.Interval)
		defer ticker.Stop()

		// 启动后先跑一轮
		runOnce(db, opts.Timeout)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runOnce(db, opts.Timeout)
			}
		}
	}()
}

func runOnce(db *sql.DB, timeout time.Duration) {
	if db == nil {
		return
	}
	devs, _, err := database.GetDevices(db, "all", "", 5000, 0)
	if err != nil {
		logger.Error("设备探测：读取设备列表失败: %v", err)
		return
	}

	for _, d := range devs {
		if strings.TrimSpace(d.IP) == "" {
			continue
		}
		online := probeReachable(d.IP, timeout)
		newStatus := "offline"
		if online {
			newStatus = "online"
		}
		if d.Status == newStatus {
			// online 的设备也更新 last_seen（避免 UI 误判 오래没见）
			if newStatus == "online" {
				_ = database.TouchDeviceLastSeen(db, d.IP)
			}
			continue
		}

		if err := database.UpdateDeviceStatus(db, d.IP, newStatus); err != nil {
			logger.Error("设备探测：更新状态失败: ip=%s err=%v", d.IP, err)
			continue
		}

		realtime.Default().Broadcast("device_status_changed", map[string]interface{}{
			"ip":     d.IP,
			"status": newStatus,
			"ts":     time.Now().Format(time.RFC3339),
		})
	}
}

func probeReachable(ip string, timeout time.Duration) bool {
	ports := []string{"80", "443", "22"}
	for _, p := range ports {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, p), timeout)
		if err == nil {
			_ = conn.Close()
			return true
		}
		// connection refused 代表主机可达但端口关闭
		if isConnRefused(err) {
			return true
		}
	}
	return false
}

func isConnRefused(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection refused")
}
