package deviceboot

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"totoro-device/config"
	"totoro-device/internal/logger"
	"totoro-device/internal/system"
)

// ApplySystemSettings 启动时应用系统设置（音量/亮度等）。
// 设计：best-effort，不阻塞启动流程。
func ApplySystemSettings(cfg *config.Config) {
	if cfg == nil {
		return
	}
	if runtime.GOOS != "linux" {
		return
	}

	// 音量：Luckfox Buildroot 推荐用 amixer cset name='DAC LINEOUT Volume'
	if cfg.System.Volume != nil {
		v := *cfg.System.Volume
		if v < 0 {
			v = 0
		}
		if v > 30 {
			v = 30
		}
		go func(vol int) {
			if err := runCmd(2*time.Second, "amixer", "cset", "name=DAC LINEOUT Volume", strconv.Itoa(vol)); err != nil {
				logger.Warn("应用音量失败: %v", err)
			} else {
				logger.Info("已应用音量: %d", vol)
			}
		}(v)
	}

	// 亮度：不同屏幕实现差异大，这里只做预留（后续在面板里做“可用性检测”）
	if cfg.System.Brightness != nil {
		p := *cfg.System.Brightness
		go func(percent int) {
			bl, err := system.DiscoverBacklight()
			if err != nil || bl == nil {
				logger.Warn("应用亮度失败: 未检测到背光")
				return
			}
			if err := bl.SetPercent(percent); err != nil {
				logger.Warn("应用亮度失败: %v", err)
				return
			}
			logger.Info("已应用亮度: %d%%", percent)
		}(p)
	}
}

func runCmd(timeout time.Duration, name string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("%s 超时", name)
	}
	if err != nil {
		s := strings.TrimSpace(string(out))
		if s != "" {
			return fmt.Errorf("%s 失败: %s", name, s)
		}
		return err
	}
	return nil
}


