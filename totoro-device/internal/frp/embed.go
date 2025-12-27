package frp

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"totoro-device/internal/logger"
)

// embeddedAssets 由平台特定的embed文件提供（通过build tags）
// 每个平台只嵌入对应平台的frpc二进制，减少二进制体积
// 定义在 embed_*.go 文件中

func statfsAvailBytes(dir string) (int64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return 0, err
	}
	return int64(st.Bavail) * int64(st.Bsize), nil
}

func canExecInDir(dir string) bool {
	// 用 execve 的方式验证“可执行”，从而识别 vfat/noexec 等场景
	// 仅对类 Unix 有意义；其它平台直接返回 true
	if runtime.GOOS == "windows" {
		return true
	}
	_ = os.MkdirAll(dir, 0755)

	testPath := filepath.Join(dir, ".totoro_exec_test.sh")
	_ = os.WriteFile(testPath, []byte("#!/bin/sh\nexit 0\n"), 0755)
	defer os.Remove(testPath)

	cmd := exec.Command(testPath)
	_ = cmd.Run()
	return cmd.ProcessState != nil && cmd.ProcessState.Success()
}

func pickFrpcCacheDir(requiredBytes int64) (string, error) {
	// 目标：选择一个“可写 + 可执行 + 空间足够”的目录来落盘 frpc
	// 候选优先级：NWCT_CACHE_DIR > UserCacheDir > /root/.cache > /userdata/.cache > /oem/.cache > /tmp > /mnt/sdcard
	candidates := make([]string, 0, 8)

	if d := strings.TrimSpace(os.Getenv("NWCT_CACHE_DIR")); d != "" {
		candidates = append(candidates, d)
	}
	if d, err := os.UserCacheDir(); err == nil && strings.TrimSpace(d) != "" {
		candidates = append(candidates, filepath.Join(d, "nwct", "bin"))
	}
	candidates = append(candidates,
		"/root/.cache/nwct/bin",
		"/userdata/.cache/nwct/bin",
		"/oem/.cache/nwct/bin",
		filepath.Join(os.TempDir(), "nwct", "bin"),
		"/mnt/sdcard/.cache/nwct/bin",
	)

	seen := map[string]bool{}
	margin := requiredBytes / 10 // +10% 余量
	if margin < 2*1024*1024 {
		margin = 2 * 1024 * 1024 // 至少 +2MB
	}
	need := requiredBytes + margin

	var lastErr error
	for _, d := range candidates {
		d = strings.TrimSpace(d)
		if d == "" || seen[d] {
			continue
		}
		seen[d] = true

		if err := os.MkdirAll(d, 0755); err != nil {
			lastErr = err
			continue
		}
		// 写入探测
		if f, err := os.CreateTemp(d, ".totoro_write_test_*"); err == nil {
			_ = f.Close()
			_ = os.Remove(f.Name())
		} else {
			lastErr = err
			continue
		}

		// 空间探测（失败则保守跳过）
		if avail, err := statfsAvailBytes(d); err == nil {
			if avail < need {
				lastErr = fmt.Errorf("目录可用空间不足: dir=%s avail=%d need=%d", d, avail, need)
				continue
			}
		} else {
			lastErr = err
			continue
		}

		// 可执行探测
		if !canExecInDir(d) {
			lastErr = fmt.Errorf("目录不可执行（可能是 vfat/noexec）: %s", d)
			continue
		}

		return d, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("未找到可用的 frpc 缓存目录")
	}
	return "", lastErr
}

// getEmbeddedFRCPath 从嵌入的资源中获取 frpc 二进制路径
func getEmbeddedFRCPath() (string, error) {
	// 确定平台和架构
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// 构建资源路径
	assetName := fmt.Sprintf("frpc_%s_%s", goos, goarch)
	if goos == "windows" {
		assetName += ".exe"
	}

	assetPath := filepath.Join("assets", assetName)

	// 检查资源是否存在
	if _, err := embeddedAssets.Open(assetPath); err != nil {
		return "", fmt.Errorf("未找到嵌入的 frpc 二进制: %s (平台: %s/%s)", assetPath, goos, goarch)
	}

	// 智能选择落盘目录（Buildroot 上 /oem 常常很小，会导致 no space left）
	var assetSize int64 = 0
	if f, err := embeddedAssets.Open(assetPath); err == nil {
		if st, err := f.Stat(); err == nil {
			assetSize = st.Size()
		}
		_ = f.Close()
	}

	cacheDir := ""
	if assetSize > 0 {
		if d, err := pickFrpcCacheDir(assetSize); err == nil {
			cacheDir = d
		} else {
			logger.Warn("选择 frpc 缓存目录失败: %v，回退到 /tmp", err)
		}
	}
	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "nwct", "bin")
	}

	destPath := filepath.Join(cacheDir, "frpc")
	if goos == "windows" {
		destPath += ".exe"
	}

	// 检查是否已存在且是最新的（通过文件大小简单判断）
	if info, err := os.Stat(destPath); err == nil {
		// 读取嵌入文件的大小
		if f, err := embeddedAssets.Open(assetPath); err == nil {
			if stat, err := f.Stat(); err == nil {
				if stat.Size() == info.Size() {
					// 大小相同，认为是最新的，直接返回
					f.Close()
					logger.Info("使用已存在的 frpc: %s", destPath)
					return destPath, nil
				}
			}
			f.Close()
		}
	}

	// 从嵌入资源复制到临时目录
	src, err := embeddedAssets.Open(assetPath)
	if err != nil {
		return "", fmt.Errorf("打开嵌入资源失败: %v", err)
	}
	defer src.Close()

	dst, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("创建目标文件失败: %v", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(destPath)
		return "", fmt.Errorf("复制文件失败: %v", err)
	}

	// 设置可执行权限
	if err := os.Chmod(destPath, 0755); err != nil {
		logger.Warn("设置可执行权限失败: %v", err)
	}

	logger.Info("已解压嵌入的 frpc 到: %s", destPath)
	return destPath, nil
}

// ensureFRCPath 确保 frpc 可执行文件可用，优先使用嵌入的，否则使用系统 PATH
func ensureFRCPath() (string, error) {
	// 尝试使用嵌入的二进制
	if path, err := getEmbeddedFRCPath(); err == nil {
		return path, nil
	} else {
		logger.Warn("使用嵌入的 frpc 失败: %v，尝试系统 PATH", err)
	}

	// 回退到系统 PATH
	if path, err := exec.LookPath("frpc"); err == nil {
		logger.Info("使用系统 PATH 中的 frpc: %s", path)
		return path, nil
	}

	return "", fmt.Errorf("未找到 frpc 可执行文件（请确保已安装 frpc 或使用嵌入版本）")
}
