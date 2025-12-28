package systemreset

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"totoro-device/config"
	"totoro-device/internal/logger"
)

// FactoryReset 清理 totoro-device 的持久化数据，并按 rebootMode 执行重启策略。
// rebootMode: hard|soft|none（默认 hard）
func FactoryReset(cfg *config.Config, rebootMode string) error {
	rebootMode = strings.ToLower(strings.TrimSpace(rebootMode))
	if rebootMode == "" {
		rebootMode = "hard"
	}

	paths := ResetPaths(cfg)
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		_ = os.RemoveAll(p) // best-effort
	}
	_ = exec.Command("sync").Run()

	switch rebootMode {
	case "none":
		return nil
	case "soft":
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
		return nil
	default:
		return HardRebootBestEffort()
	}
}

// ResetPaths 返回“恢复出厂”需要清理的路径集合（仅清理 totoro-device 相关数据）。
func ResetPaths(cfg *config.Config) []string {
	paths := make([]string, 0, 32)

	// config
	if cfgPath := strings.TrimSpace(config.GetConfigPath()); cfgPath != "" {
		paths = append(paths, cfgPath)
	}

	// db
	dbPath := strings.TrimSpace(os.Getenv("NWCT_DB_PATH"))
	if dbPath == "" && cfg != nil {
		dbPath = strings.TrimSpace(cfg.Database.Path)
	}
	if dbPath == "" && runtime.GOOS == "linux" {
		dbPath = "/var/nwct/devices.db"
	}
	if dbPath != "" {
		paths = append(paths, dbPath, dbPath+"-wal", dbPath+"-shm")
	}

	// logs
	logDir := strings.TrimSpace(os.Getenv("NWCT_LOG_DIR"))
	if logDir == "" {
		logDir = "/var/log/nwct"
	}
	paths = append(paths,
		filepath.Join(logDir, "system.log"),
		filepath.Join(logDir, "error.log"),
		filepath.Join(logDir, "info.log"),
		"/var/log/totoro-device.log",
	)

	// runtime temp
	paths = append(paths,
		"/tmp/nwct",
		"/tmp/nwct.pid",
		"/tmp/nwct.out",
		"/run/nwct",
		"/var/run/totoro-device.pid",
	)

	// caches
	if d := strings.TrimSpace(os.Getenv("NWCT_CACHE_DIR")); d != "" {
		paths = append(paths, d)
	}
	paths = append(paths,
		"/root/.cache/nwct",
		"/oem/.cache/nwct",
		"/userdata/.cache/nwct",
		"/mnt/sdcard/.cache/nwct",
	)

	// reset marker（如果存在）
	paths = append(paths,
		"/userdata/totoro/reset_trigger.json",
		"/oem/totoro/reset_trigger.json",
		"/var/nwct/reset_trigger.json",
	)

	return paths
}

func HardRebootBestEffort() error {
	switch runtime.GOOS {
	case "linux":
		_ = exec.Command("sync").Run()

		if p, err := exec.LookPath("reboot"); err == nil && strings.TrimSpace(p) != "" {
			_ = exec.Command(p, "-f").Run()
			_ = exec.Command(p).Run()
		}
		if p, err := exec.LookPath("busybox"); err == nil && strings.TrimSpace(p) != "" {
			_ = exec.Command(p, "reboot", "-f").Run()
			_ = exec.Command(p, "reboot").Run()
		}
		if p, err := exec.LookPath("shutdown"); err == nil && strings.TrimSpace(p) != "" {
			_ = exec.Command(p, "-r", "now").Run()
		}
		return nil
	case "darwin":
		return exec.Command("shutdown", "-r", "now").Run()
	default:
		return fmt.Errorf("不支持的系统重启平台: %s", runtime.GOOS)
	}
}

func LogFactoryResetPaths(cfg *config.Config) {
	paths := ResetPaths(cfg)
	logger.Warn("恢复出厂将清理以下路径（共 %d 项）：", len(paths))
	for _, p := range paths {
		if strings.TrimSpace(p) == "" {
			continue
		}
		logger.Warn("- %s", p)
	}
}
