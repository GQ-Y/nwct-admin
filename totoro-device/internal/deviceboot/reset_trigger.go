package deviceboot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"totoro-device/config"
	"totoro-device/internal/logger"
	"totoro-device/internal/systemreset"
)

type resetTriggerState struct {
	LastUnix int64 `json:"last_unix"`
	Count    int   `json:"count"`
}

func pickResetMarkerPath() string {
	// 优先持久化分区
	cands := []string{
		"/userdata/totoro/reset_trigger.json",
		"/oem/totoro/reset_trigger.json",
		"/var/nwct/reset_trigger.json",
	}
	for _, p := range cands {
		dir := filepath.Dir(p)
		_ = os.MkdirAll(dir, 0755)
		// 尝试写入探测
		f, err := os.CreateTemp(dir, ".totoro_write_test_*")
		if err != nil {
			continue
		}
		_ = f.Close()
		_ = os.Remove(f.Name())
		return p
	}
	return "/var/nwct/reset_trigger.json"
}

func getenvInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

// MaybeTriggerFactoryResetByMultiReboot 利用“硬复位键（RESET）无法被系统感知”的现实，
// 用“短时间内多次重启”作为触发条件来执行恢复出厂设置。
//
// 默认策略：
// - 30 秒窗口内连续重启 >= 5 次 -> 触发恢复出厂并 hard reboot
// 可通过环境变量调节：
// - NWCT_FACTORY_RESET_REBOOT_COUNT（默认 5）
// - NWCT_FACTORY_RESET_WINDOW_SECONDS（默认 30）
func MaybeTriggerFactoryResetByMultiReboot(cfg *config.Config) {
	need := getenvInt("NWCT_FACTORY_RESET_REBOOT_COUNT", 5)
	win := getenvInt("NWCT_FACTORY_RESET_WINDOW_SECONDS", 30)
	if need <= 1 || win <= 0 {
		return
	}

	marker := pickResetMarkerPath()
	now := time.Now().Unix()

	st := resetTriggerState{}
	if b, err := os.ReadFile(marker); err == nil && len(b) > 0 {
		_ = json.Unmarshal(b, &st)
	}

	if st.LastUnix > 0 && (now-st.LastUnix) <= int64(win) {
		st.Count++
	} else {
		st.Count = 1
	}
	st.LastUnix = now

	_ = os.WriteFile(marker, mustJSON(st), 0644)

	if st.Count >= need {
		logger.Warn("检测到短时间多次重启：%d 次 / %d 秒，触发恢复出厂设置（按 RESET 多次触发）", st.Count, win)
		systemreset.LogFactoryResetPaths(cfg)
		// 直接执行恢复出厂并硬重启（不会返回）
		_ = systemreset.FactoryReset(cfg, "hard")
	}
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	if len(b) == 0 {
		return []byte("{}")
	}
	return b
}
