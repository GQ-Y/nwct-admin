package nps

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type InstallOptions struct {
	// Version 形如 "v0.26.10"；为空则使用默认版本（尽量稳定）
	Version string
	// OS/Arch 默认使用 runtime.GOOS/GOARCH
	OS   string
	Arch string
	// InstallDir 安装目录（会创建）；为空则使用 NWCT_NPC_DIR 或用户缓存目录
	InstallDir string
	// Timeout 下载超时
	Timeout time.Duration
}

type InstallResult struct {
	Version     string `json:"version"`
	AssetName   string `json:"asset_name"`
	DownloadURL string `json:"download_url"`
	Path        string `json:"path"`
}

const defaultNPSVersion = "v0.26.10"

// InstallNPC 从 GitHub release 下载并解压 npc 可执行文件到本地（用于设备端一键集成）。
func InstallNPC(ctx context.Context, opts InstallOptions) (*InstallResult, error) {
	version := strings.TrimSpace(opts.Version)
	if version == "" {
		version = defaultNPSVersion
	}
	goos := strings.TrimSpace(opts.OS)
	if goos == "" {
		goos = runtime.GOOS
	}
	arch := strings.TrimSpace(opts.Arch)
	if arch == "" {
		arch = runtime.GOARCH
	}

	asset, err := npcAssetName(goos, arch)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://github.com/ehang-io/nps/releases/download/%s/%s", version, asset)

	installDir := strings.TrimSpace(opts.InstallDir)
	if installDir == "" {
		installDir = strings.TrimSpace(os.Getenv("NWCT_NPC_DIR"))
	}
	if installDir == "" {
		if d, err := os.UserCacheDir(); err == nil && d != "" {
			installDir = filepath.Join(d, "nwct", "bin")
		} else {
			installDir = filepath.Join(os.TempDir(), "nwct", "bin")
		}
	}
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return nil, err
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	cli := &http.Client{Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("下载 npc 失败: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}

	tmpArchive := filepath.Join(installDir, asset+".tmp")
	f, err := os.Create(tmpArchive)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpArchive)
		return nil, err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmpArchive)
		return nil, err
	}

	dest := filepath.Join(installDir, "npc")
	if goos == "windows" {
		dest = filepath.Join(installDir, "npc.exe")
	}

	if strings.HasSuffix(asset, ".tar.gz") {
		if err := extractNPCFromTarGz(tmpArchive, dest); err != nil {
			_ = os.Remove(tmpArchive)
			return nil, err
		}
	} else if strings.HasSuffix(asset, ".zip") {
		if err := extractNPCFromZip(tmpArchive, dest); err != nil {
			_ = os.Remove(tmpArchive)
			return nil, err
		}
	} else {
		_ = os.Remove(tmpArchive)
		return nil, fmt.Errorf("不支持的压缩格式: %s", asset)
	}
	_ = os.Remove(tmpArchive)

	_ = os.Chmod(dest, 0o755)

	return &InstallResult{
		Version:     version,
		AssetName:   asset,
		DownloadURL: url,
		Path:        dest,
	}, nil
}

func npcAssetName(goos, arch string) (string, error) {
	// NPS release 常见命名：linux_amd64_client.tar.gz / darwin_arm64_client.tar.gz 等
	goos = strings.ToLower(goos)
	arch = strings.ToLower(arch)

	switch goos {
	case "linux", "darwin", "windows", "freebsd":
	default:
		return "", fmt.Errorf("不支持的 OS: %s", goos)
	}

	// 归一化 arch
	switch arch {
	case "amd64", "arm64":
		// ok
	case "arm":
		// 没法可靠区分 v5/v6/v7，这里先按 v7 处理（多数树莓派/路由器）
		arch = "arm_v7"
	case "386":
		// ok
	case "mips", "mipsle", "mips64", "mips64le":
		// ok（用于部分路由器）
	default:
		return "", fmt.Errorf("不支持的 ARCH: %s", arch)
	}

	// v0.26.10 及常见版本：所有 client 资产均为 .tar.gz（包括 windows）
	ext := "tar.gz"

	// 兼容：macOS arm64 release 可能缺失，此时回退到 amd64（Rosetta 环境可运行）
	if goos == "darwin" && arch == "arm64" {
		arch = "amd64"
	}
	return fmt.Sprintf("%s_%s_client.%s", goos, arch, ext), nil
}

func extractNPCFromTarGz(archivePath, dest string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)

	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := filepath.Base(h.Name)
		if name != "npc" && name != "npc.exe" {
			continue
		}
		out, err := os.Create(dest)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			_ = out.Close()
			return err
		}
		return out.Close()
	}
	return fmt.Errorf("压缩包中未找到 npc")
}

func extractNPCFromZip(archivePath, dest string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer zr.Close()

	for _, f := range zr.File {
		name := filepath.Base(f.Name)
		if name != "npc" && name != "npc.exe" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		out, err := os.Create(dest)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, rc); err != nil {
			_ = out.Close()
			return err
		}
		return out.Close()
	}
	return fmt.Errorf("压缩包中未找到 npc")
}


