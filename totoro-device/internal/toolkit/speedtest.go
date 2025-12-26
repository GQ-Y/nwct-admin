package toolkit

import (
	"context"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// SpeedResult 网速测试结果
type SpeedResult struct {
	Server       string  `json:"server"`
	UploadSpeed  float64 `json:"upload_speed"`  // Mbps
	DownloadSpeed float64 `json:"download_speed"` // Mbps
	Latency      int     `json:"latency"`      // ms
	TestTime     string  `json:"test_time"`
	Duration     int     `json:"duration"`     // 秒
}

// SpeedTest 执行网速测试
func SpeedTest(server string, testType string) (*SpeedResult, error) {
	// 默认测速源：清华大学 TUNA 镜像站（更适合国内环境）；失败会自动多级兜底
	if server == "" || server == "default" {
		// 选用相对稳定的大文件路径（我们只在固定时间窗口内读取，不会强制下完整文件）
		server = "https://mirrors.tuna.tsinghua.edu.cn/ubuntu-releases/24.04/ubuntu-24.04-desktop-amd64.iso"
	}
	if testType == "" {
		testType = "all"
	}

	result := &SpeedResult{
		Server:   server,
		TestTime: time.Now().Format(time.RFC3339),
	}

	start := time.Now()

	// download 测试用的候选 URL（按顺序尝试）
	downloadCandidates := []string{}
	downloadCandidates = append(downloadCandidates, server)
	// 如果用户选择默认/清华源，提供更多清华的候选（避免单一路径变动）
	if strings.Contains(server, "mirrors.tuna.tsinghua.edu.cn") {
		downloadCandidates = append(downloadCandidates,
			"https://mirrors.tuna.tsinghua.edu.cn/ubuntu-releases/22.04/ubuntu-22.04.5-desktop-amd64.iso",
			"https://mirrors.tuna.tsinghua.edu.cn/archlinux/iso/latest/archlinux-x86_64.iso",
		)
	}
	// 兜底：Cloudflare/Hetzner/ThinkBroadband/Tele2
	downloadCandidates = append(downloadCandidates,
		"https://speed.cloudflare.com",
		"https://speed.hetzner.de/10MB.bin",
		"https://download.thinkbroadband.com/10MB.zip",
		"http://speedtest.tele2.net/10MB.zip",
	)

	// 测试下载速度
	if testType == "download" || testType == "all" {
		var lastErr error
		for _, cand := range downloadCandidates {
			downloadSpeed, latency, used, err := testDownloadSpeedAny(cand)
			if err != nil {
				lastErr = err
				continue
			}
			result.DownloadSpeed = downloadSpeed
			result.Latency = latency
			result.Server = used
			lastErr = nil
			break
		}
		if lastErr != nil {
			return nil, lastErr
		}
	}

	// 测试上传速度
	if testType == "upload" || testType == "all" {
		uploadSpeed, err := testUploadSpeed(server)
		if err == nil {
			result.UploadSpeed = uploadSpeed
		}
	}

	result.Duration = int(time.Since(start).Seconds())

	return result, nil
}

func normalizeServer(server string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(server))
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" {
		// 兼容旧配置：没写 scheme 就按 http
		u, err = url.Parse("http://" + strings.TrimSpace(server))
		if err != nil {
			return nil, err
		}
	}
	return u, nil
}

func httpClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
	}
}

func httpClientWithTransport() *http.Client {
	// 使用 ctx 控制整体时长，这里不给 Client.Timeout，避免“读 body 超时”导致直接失败
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          20,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &http.Client{Transport: tr}
}

// testDownloadSpeedAny 支持：
// - 传入 https://speed.cloudflare.com（会使用 /__down）
// - 传入具体文件URL（如 http://speedtest.tele2.net/10MB.zip）
// - 传入 base URL（会自动拼接一个常见下载文件名）
func testDownloadSpeedAny(serverOrURL string) (float64, int, string, error) {
	s := strings.TrimSpace(serverOrURL)
	if s == "" {
		return 0, 0, "", fmt.Errorf("测速服务器不能为空")
	}

	// Cloudflare 特殊端点
	if strings.Contains(s, "speed.cloudflare.com") && !strings.Contains(s, "/__down") {
		base, err := normalizeServer(s)
		if err != nil {
			return 0, 0, "", err
		}
		downURL := *base
		downURL.Path = "/__down"
		q := downURL.Query()
		q.Set("bytes", "100000000")
		downURL.RawQuery = q.Encode()
		return testDownloadSpeedURL(downURL.String(), 2, 12*time.Second)
	}

	// 如果用户给的是 base（没有明显文件后缀），给它拼一个常见文件
	if u, err := normalizeServer(s); err == nil {
		if u.Path == "" || strings.HasSuffix(u.Path, "/") {
			// 默认拼 /10MB.zip（多数公开测速源兼容）
			u.Path = strings.TrimRight(u.Path, "/") + "/10MB.zip"
			return testDownloadSpeedURL(u.String(), 1, 12*time.Second)
		}
		// 看起来像文件URL，直接用
		return testDownloadSpeedURL(u.String(), 1, 12*time.Second)
	}

	// 最后尝试当作原始 URL
	return testDownloadSpeedURL(s, 1, 12*time.Second)
}

func testDownloadSpeedURL(urlStr string, concurrency int, duration time.Duration) (float64, int, string, error) {
	latency, _ := testLatencyByRange(urlStr)
	totalBytes, dur, err := parallelDownloadForDuration(urlStr, concurrency, duration)
	if err != nil {
		return 0, latency, urlStr, err
	}
	if dur <= 0 {
		return 0, latency, urlStr, fmt.Errorf("测试时间过短")
	}
	mbps := (float64(totalBytes) * 8) / (dur * 1000000)
	if math.IsInf(mbps, 0) || math.IsNaN(mbps) {
		return 0, latency, urlStr, fmt.Errorf("测速结果异常")
	}
	return mbps, latency, urlStr, nil
}

func testLatencyByRange(urlStr string) (int, error) {
	client := httpClientWithTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequest("GET", urlStr, nil)
	req = req.WithContext(ctx)
	req.Header.Set("Range", "bytes=0-0")
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode >= 400 {
		_ = resp.Body.Close()
		return 0, fmt.Errorf("latency请求失败: %s", resp.Status)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	return int(time.Since(start).Milliseconds()), nil
}

func parallelDownloadForDuration(urlStr string, concurrency int, duration time.Duration) (int64, float64, error) {
	if concurrency <= 0 {
		concurrency = 1
	}
	if duration <= 0 {
		duration = 8 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	client := httpClientWithTransport()

	var total int64
	var mu sync.Mutex
	var wg sync.WaitGroup
	var firstErr error
	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
			resp, err := client.Do(req)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode >= 400 {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("下载请求失败: %s", resp.Status)
				}
				mu.Unlock()
				return
			}
			buf := make([]byte, 64*1024)
			var n int64
			for {
				readN, rerr := resp.Body.Read(buf)
				if readN > 0 {
					n += int64(readN)
				}
				if rerr == io.EOF {
					break
				}
				if rerr != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = rerr
					}
					mu.Unlock()
					return
				}
				select {
				case <-ctx.Done():
					// 到达测速窗口：正常结束
					mu.Lock()
					total += n
					mu.Unlock()
					return
				default:
				}
			}
			mu.Lock()
			total += n
			mu.Unlock()
		}()
	}
	wg.Wait()
	dur := time.Since(start).Seconds()
	if firstErr != nil && total == 0 {
		return 0, dur, firstErr
	}
	if total == 0 {
		return 0, dur, fmt.Errorf("下载测速失败：未读取到数据")
	}
	return total, dur, nil
}

// 保留以兼容旧接口（不再直接调用）
func testDownloadSpeedTele2(server string) (float64, int, error) {
	speed, latency, _, err := testDownloadSpeedURL(strings.TrimRight(server, "/")+"/10MB.zip", 1, 12*time.Second)
	return speed, latency, err
}

// testUploadSpeed 测试上传速度
func testUploadSpeed(server string) (float64, error) {
	// 仅当测速服务端明确支持上传接口时才启用（否则返回明确错误）
	return 0, fmt.Errorf("上传速度测试需要测速服务端提供上传接口（当前默认不支持）")
}

