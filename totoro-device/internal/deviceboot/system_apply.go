package deviceboot

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

	// 设备名/hostname：让路由器/局域网里更容易识别（DHCP 会带 option 12）
	// - cfg.Device.Name 允许包含空格，但 hostname 不允许：需要做规范化
	// - 兼容老配置：若还是默认的 Luckfox，则强制更新为编译期默认名（config.DefaultDeviceName）
	name := strings.TrimSpace(cfg.Device.Name)
	if name == "" || strings.EqualFold(name, "luckfox") {
		name = config.DefaultDeviceName
		cfg.Device.Name = name
		_ = cfg.Save() // best-effort
	}
	// 系统 hostname 不能包含空格，但允许大写；这里用 '-' 替代空格，并保留大小写
	host := sanitizeHostname(name)
	if host != "" {
		_ = os.WriteFile("/etc/hostname", []byte(host+"\n"), 0644)
		_ = runCmd(2*time.Second, "hostname", host)
	}
	// 确保 sshd 运行（best-effort；避免某些镜像默认没启动导致 WiFi 下“22 不可达”的误判）
	_ = runCmd(2*time.Second, "sh", "-lc", "/etc/init.d/S50sshd start >/dev/null 2>&1 || true")
	// DHCP hostname 宣告：很多路由器只在 DHCP 租约更新时才显示“设备名称”
	// - 如果系统 init 早于本程序启动拿到了 DHCP（且没带 hostname），路由器会显示“*”
	// - 这里在不阻塞启动的前提下，做一次短促的 udhcpc 请求，让路由器刷新显示名
	// DHCP 上报名称：路由器显示用，允许包含空格（如 "Totoro S1 Pro"）
	go dhcpAnnounceHostnameOnce(name, host)

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
		// 限制：最低 10%，避免黑屏
		if p < 10 {
			p = 10
			cfg.System.Brightness = &p
			_ = cfg.Save() // best-effort
		}
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

func sanitizeHostname(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = config.DefaultDeviceName
	}
	var b []rune
	lastHyphen := false
	for _, r := range name {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			b = append(b, r)
			lastHyphen = false
		case r >= '0' && r <= '9':
			b = append(b, r)
			lastHyphen = false
		case r == '-' || r == ' ' || r == '_':
			if len(b) == 0 || lastHyphen {
				continue
			}
			b = append(b, '-')
			lastHyphen = true
		default:
			// 跳过其他字符（包括中文/标点）
		}
	}
	s := strings.Trim(string(b), "-")
	if s == "" {
		s = strings.Trim(string(b), "-")
		if s == "" {
			// 最终兜底
			s = "Totoro-S1"
		}
	}
	if len(s) > 63 {
		s = strings.TrimRight(s[:63], "-")
		if s == "" {
			s = "Totoro-S1"
		}
	}
	return s
}

func dhcpAnnounceHostnameOnce(displayName, host string) {
	displayName = strings.TrimSpace(displayName)
	host = strings.TrimSpace(host)
	if host == "" {
		return
	}
	// 用 /run 做一次性标记，避免反复打断网络
	_ = os.MkdirAll("/run/nwct", 0755)
	marker := "/run/nwct/dhcp_hostname_announced"
	// 若 marker 已存在且内容匹配，则跳过；否则允许再次宣告（比如从小写升级为大写显示名）
	if b, err := os.ReadFile(marker); err == nil {
		s := strings.TrimSpace(string(b))
		want := "display=" + displayName + "\nhost=" + host
		if s == want {
			return
		}
	}
	// 仅在存在 udhcpc 时尝试
	if _, err := exec.LookPath("udhcpc"); err != nil {
		return
	}
	// 修复现实情况：系统自身会常驻启动 udhcpc（但不带 hostname），导致路由器列表显示“*”
	// 这里的策略是：按接口重启 udhcpc，并带上 hostname 选项（option 12）。
	for _, iface := range []string{"eth0", "wlan0"} {
		restartUdhcpcWithHostnameBestEffort(iface, displayName, host)
	}
	_ = os.WriteFile(marker, []byte("display="+displayName+"\nhost="+host+"\n"), 0644)
}

func restartUdhcpcWithHostnameBestEffort(iface, displayName, host string) {
	iface = strings.TrimSpace(iface)
	displayName = strings.TrimSpace(displayName)
	host = strings.TrimSpace(host)
	if iface == "" || host == "" {
		return
	}
	// 若接口不存在则跳过
	if _, err := os.Stat("/sys/class/net/" + iface); err != nil {
		return
	}

	// 1) 尽量 kill 掉旧的 udhcpc（同接口）
	killUdhcpcByIfaceBestEffort(iface)

	// 2) 启动一个带 hostname 的 udhcpc（常驻）
	// -p：pidfile，便于后续排障
	pidfile := filepath.Join("/run", "udhcpc."+iface+".pid")
	args := []string{
		"-i", iface,
		"-p", pidfile,
		"-T", "3",
		"-t", "3",
		// option 12: hostname（路由器显示名）
		"-x", "hostname:" + pickDHCPHostname(displayName, host),
		// -F：请求更新 DNS 映射（多数路由器只接受无空格 hostname）
		"-F", host,
	}
	cmd := exec.Command("udhcpc", args...)
	_ = cmd.Start() // best-effort，不阻塞启动
}

func pickDHCPHostname(displayName, host string) string {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return host
	}
	// DHCP option 12 允许字符串；这里优先使用用户可读名（如 "Totoro S1 Pro"）
	// 如果包含非 ASCII，可能会导致部分路由器显示异常，此时回退到 host
	for _, r := range displayName {
		if r > 127 {
			return host
		}
	}
	return displayName
}

func killUdhcpcByIfaceBestEffort(iface string) {
	// 遍历 /proc/<pid>/cmdline，筛选出 "udhcpc ... -i <iface>" 的进程并 kill
	ents, err := os.ReadDir("/proc")
	if err != nil {
		return
	}
	for _, ent := range ents {
		if !ent.IsDir() {
			continue
		}
		pid := ent.Name()
		// 仅数字目录
		if pid == "" || pid[0] < '0' || pid[0] > '9' {
			continue
		}
		cmdlinePath := filepath.Join("/proc", pid, "cmdline")
		b, err := os.ReadFile(cmdlinePath)
		if err != nil || len(b) == 0 {
			continue
		}
		// cmdline 是 \0 分隔
		s := strings.ReplaceAll(string(b), "\x00", " ")
		s = strings.TrimSpace(s)
		if !strings.Contains(s, "udhcpc") {
			continue
		}
		if !strings.Contains(s, "-i "+iface) {
			continue
		}
		// kill -9（best-effort）
		_ = runCmd(500*time.Millisecond, "sh", "-lc", "kill -9 "+pid+" >/dev/null 2>&1 || true")
	}
}
