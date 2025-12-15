package fingerprint

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// 默认来源：IEEE OUI 文本库（权威）
const defaultIEEEOUIURL = "https://standards-oui.ieee.org/oui/oui.txt"

var reHex2 = regexp.MustCompile(`^[0-9a-f]{2}$`)

type OUIOptions struct {
	// FilePath 指定本地 OUI 文件路径（优先）。为空则使用默认缓存路径。
	FilePath string
	// AutoUpdateMissing 当本地文件不存在时，是否自动从 SourceURL 下载并缓存。
	AutoUpdateMissing bool
	// SourceURL 下载源；为空则使用 IEEE 官方默认地址。
	SourceURL string
	// CacheDir 缓存目录；仅当 FilePath 为空时生效。
	CacheDir string
	// HTTPTimeout 下载超时
	HTTPTimeout time.Duration
}

type OUIResolver struct {
	mu      sync.RWMutex
	loaded  bool
	data    map[string]string // 6 hex (upper) => vendor
	path    string
	opts    OUIOptions
	lastErr error
}

var (
	defaultOnce sync.Once
	defaultOUI  *OUIResolver
)

func DefaultOUI() *OUIResolver {
	defaultOnce.Do(func() {
		defaultOUI = NewOUIResolver(OUIOptions{
			FilePath:          strings.TrimSpace(os.Getenv("NWCT_OUI_PATH")),
			AutoUpdateMissing: os.Getenv("NWCT_OUI_AUTO_UPDATE") == "1" || strings.EqualFold(os.Getenv("NWCT_OUI_AUTO_UPDATE"), "true"),
			SourceURL:         strings.TrimSpace(os.Getenv("NWCT_OUI_URL")),
			CacheDir:          strings.TrimSpace(os.Getenv("NWCT_OUI_CACHE_DIR")),
			HTTPTimeout:       10 * time.Second,
		})
	})
	return defaultOUI
}

func NewOUIResolver(opts OUIOptions) *OUIResolver {
	r := &OUIResolver{opts: opts}
	if r.opts.SourceURL == "" {
		r.opts.SourceURL = defaultIEEEOUIURL
	}
	if r.opts.HTTPTimeout <= 0 {
		r.opts.HTTPTimeout = 10 * time.Second
	}
	r.path = strings.TrimSpace(r.opts.FilePath)
	if r.path == "" {
		// 1) 优先使用工作目录下的 assets/oui.txt（最符合部署：二进制与 assets 同目录）
		if _, err := os.Stat(filepath.Join("assets", "oui.txt")); err == nil {
			r.path = filepath.Join("assets", "oui.txt")
			return r
		}

		// 2) 其次尝试：repo root 下的 client-nps/assets/oui.txt（开发态从仓库根目录启动）
		if _, err := os.Stat(filepath.Join("client-nps", "assets", "oui.txt")); err == nil {
			r.path = filepath.Join("client-nps", "assets", "oui.txt")
			return r
		}

		// 3) 再尝试：可执行文件所在目录下的 assets/oui.txt（无论 cwd 在哪都更稳）
		if exe, err := os.Executable(); err == nil && exe != "" {
			if _, err := os.Stat(filepath.Join(filepath.Dir(exe), "assets", "oui.txt")); err == nil {
				r.path = filepath.Join(filepath.Dir(exe), "assets", "oui.txt")
				return r
			}
		}

		// 4) 最后使用系统缓存目录
		cacheDir := strings.TrimSpace(r.opts.CacheDir)
		if cacheDir == "" {
			if d, err := os.UserCacheDir(); err == nil && d != "" {
				cacheDir = filepath.Join(d, "nwct")
			} else {
				cacheDir = filepath.Join(os.TempDir(), "nwct")
			}
		}
		r.path = filepath.Join(cacheDir, "oui.txt")
	}
	return r
}

func (r *OUIResolver) ensureLoaded() {
	r.mu.RLock()
	if r.loaded {
		r.mu.RUnlock()
		return
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.loaded {
		return
	}

	// 尝试从文件加载；不存在可选下载
	if _, err := os.Stat(r.path); err != nil {
		if errors.Is(err, os.ErrNotExist) && r.opts.AutoUpdateMissing {
			_ = os.MkdirAll(filepath.Dir(r.path), 0o755)
			if err := r.download(context.Background()); err != nil {
				r.lastErr = err
				// 继续：允许无库运行
				r.data = map[string]string{}
				r.loaded = true
				return
			}
		} else {
			r.lastErr = err
			r.data = map[string]string{}
			r.loaded = true
			return
		}
	}

	f, err := os.Open(r.path)
	if err != nil {
		r.lastErr = err
		r.data = map[string]string{}
		r.loaded = true
		return
	}
	defer f.Close()

	m, err := parseIEEEOUI(f)
	if err != nil {
		r.lastErr = err
		r.data = map[string]string{}
		r.loaded = true
		return
	}

	r.data = m
	r.loaded = true
}

func (r *OUIResolver) download(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.opts.SourceURL, nil)
	if err != nil {
		return err
	}
	cli := &http.Client{Timeout: r.opts.HTTPTimeout}
	resp, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("下载OUI失败: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}

	tmp := r.path + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, resp.Body); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, r.path)
}

// Lookup 根据 MAC 返回厂商名；找不到返回空字符串
func (r *OUIResolver) Lookup(mac string) string {
	r.ensureLoaded()

	prefix := normalizeOUI(mac)
	if prefix == "" {
		return ""
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.data[prefix]
}

// LastError 返回最近一次加载失败原因（可用于调试），可能为空
func (r *OUIResolver) LastError() error {
	r.ensureLoaded()
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lastErr
}

func normalizeOUI(mac string) string {
	// 兼容 macOS arp 的 “省略前导 0” 输出：
	// 例如 18:aa:f:f7:9e:62 实际是 18:aa:0f:f7:9e:62
	s := strings.TrimSpace(mac)
	if s == "" {
		return ""
	}
	sep := ":"
	if strings.Contains(s, "-") && !strings.Contains(s, ":") {
		sep = "-"
	}
	raw := strings.Split(s, sep)
	if len(raw) < 3 {
		return ""
	}
	n := make([]string, 0, 3)
	for i := 0; i < 3; i++ {
		p := strings.TrimSpace(raw[i])
		p = strings.TrimPrefix(strings.ToLower(p), "0x")
		if p == "" {
			return ""
		}
		// 保留最后两位并左侧补 0
		if len(p) > 2 {
			p = p[len(p)-2:]
		}
		if len(p) == 1 {
			p = "0" + p
		}
		// 只允许 hex
		if !reHex2.MatchString(p) {
			return ""
		}
		n = append(n, strings.ToUpper(p))
	}
	return strings.Join(n, ":")
}

func parseIEEEOUI(r io.Reader) (map[string]string, error) {
	// IEEE oui.txt 典型行：
	// FC-D0-8C   (hex)                Huawei Technologies Co.,Ltd
	// FC D0 8C   (base 16)            Huawei Technologies Co.,Ltd
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 16*1024), 256*1024) // 从 64KB/2MB 降到 16KB/256KB，节省内存

	m := make(map[string]string, 16*1024) // 从 64K 降到 16K，减少预分配
	reLine := regexp.MustCompile(`(?i)^\s*([0-9A-F]{2})[-\s:]?([0-9A-F]{2})[-\s:]?([0-9A-F]{2})\s+\((hex|base\s+16)\)\s+(.+?)\s*$`)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		mm := reLine.FindStringSubmatch(line)
		if len(mm) != 6 {
			continue
		}
		prefix := strings.ToUpper(mm[1] + ":" + mm[2] + ":" + mm[3])
		vendor := strings.TrimSpace(mm[5])
		if vendor == "" {
			continue
		}
		m[prefix] = vendor
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return m, nil
}
