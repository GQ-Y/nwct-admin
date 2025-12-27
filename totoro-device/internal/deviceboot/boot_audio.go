package deviceboot

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"totoro-device/internal/logger"
)

// TryPlayBootAudio 尝试播放开机语音（尽量不阻塞启动）。
//
// 设计目标：
// - Buildroot 环境不一定有 mp3 播放器；优先探测 mpg123/madplay/ffplay/aplay
// - 没有可用播放器时静默跳过（只打日志）
// - 文件路径默认 /etc/nwct/boot.mp3，其次 /etc/nwct/boot.wav；也可通过环境变量覆盖
func TryPlayBootAudio() {
	if strings.TrimSpace(os.Getenv("NWCT_BOOT_AUDIO")) == "0" || strings.EqualFold(strings.TrimSpace(os.Getenv("NWCT_BOOT_AUDIO")), "false") {
		return
	}

	path := strings.TrimSpace(os.Getenv("NWCT_BOOT_AUDIO_PATH"))
	if path == "" {
		if fileExists("/etc/nwct/boot.mp3") {
			path = "/etc/nwct/boot.mp3"
		} else if fileExists("/etc/nwct/boot.wav") {
			path = "/etc/nwct/boot.wav"
		}
	}
	if path == "" || !fileExists(path) {
		return
	}

	ext := strings.ToLower(filepath.Ext(path))

	type candidate struct {
		name string
		args []string
	}

	var cands []candidate
	switch ext {
	case ".mp3":
		cands = []candidate{
			{name: "mpg123", args: []string{"-q", path}},
			{name: "madplay", args: []string{"-q", path}},
			{name: "ffplay", args: []string{"-nodisp", "-autoexit", "-loglevel", "quiet", path}},
		}
	case ".wav":
		cands = []candidate{
			{name: "aplay", args: []string{"-q", path}},
			{name: "ffplay", args: []string{"-nodisp", "-autoexit", "-loglevel", "quiet", path}},
		}
	default:
		// 尝试 ffplay 兜底
		cands = []candidate{{name: "ffplay", args: []string{"-nodisp", "-autoexit", "-loglevel", "quiet", path}}}
	}

	for _, c := range cands {
		if _, err := exec.LookPath(c.name); err != nil {
			continue
		}
		go func(name string, args []string) {
			// 最长 10 秒，避免异常卡死（比如音频设备阻塞）
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, name, args...)
			_ = cmd.Start()
			_ = cmd.Wait()
			if ctx.Err() == context.DeadlineExceeded {
				logger.Warn("开机语音播放超时，已停止: %s", name)
			}
		}(c.name, c.args)
		logger.Info("开机语音播放启动: %s (%s)", path, c.name)
		return
	}

	logger.Warn("开机语音未播放：系统缺少播放器（mpg123/madplay/ffplay/aplay）")
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}


