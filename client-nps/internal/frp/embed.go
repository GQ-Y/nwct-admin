package frp

import (
	"fmt"
	"io"
	"nwct/client-nps/internal/logger"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// embeddedAssets 由平台特定的embed文件提供（通过build tags）
// 每个平台只嵌入对应平台的frpc二进制，减少二进制体积
// 定义在 embed_*.go 文件中

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

	// 解压到临时目录
	cacheDir := os.Getenv("NWCT_CACHE_DIR")
	if cacheDir == "" {
		if d, err := os.UserCacheDir(); err == nil && d != "" {
			cacheDir = filepath.Join(d, "nwct", "bin")
		} else {
			cacheDir = filepath.Join(os.TempDir(), "nwct", "bin")
		}
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("创建缓存目录失败: %v", err)
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
