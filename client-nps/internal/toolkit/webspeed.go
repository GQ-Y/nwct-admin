package toolkit

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"time"
)

type WebSpeedAttempt struct {
	OK         bool   `json:"ok"`
	Error      string `json:"error,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`

	ResolvedIP string `json:"resolved_ip,omitempty"`
	BytesRead  int64  `json:"bytes_read"`

	DNSMs      int `json:"dns_ms"`
	ConnectMs  int `json:"connect_ms"`
	TLSMs      int `json:"tls_ms"`
	TTFBMs     int `json:"ttfb_ms"`
	DownloadMs int `json:"download_ms"`
	TotalMs    int `json:"total_ms"`
}

type WebSpeedResult struct {
	URL           string            `json:"url"`
	Method        string            `json:"method"`
	Count         int               `json:"count"`
	TimeoutSec    int               `json:"timeout_sec"`
	DownloadBytes int64             `json:"download_bytes"`
	Attempts      []WebSpeedAttempt `json:"attempts"`
	Summary       map[string]any    `json:"summary"`
	TestTime      string            `json:"test_time"`
}

// WebSpeedTest 对“访问网站速度”做测量：DNS、TCP、TLS、TTFB、总耗时，并可选下载少量字节。
func WebSpeedTest(rawURL string, count int, timeout time.Duration, downloadBytes int64) (*WebSpeedResult, error) {
	return WebSpeedTestWithOptions(rawURL, "GET", count, timeout, downloadBytes)
}

// WebSpeedTestWithOptions 支持指定 method：
// - GET：会读取 downloadBytes（默认 64KB）
// - HEAD：不读取 body，只测 DNS/TCP/TLS/TTFB/Total（更贴近“打开网页速度”）
func WebSpeedTestWithOptions(rawURL string, method string, count int, timeout time.Duration, downloadBytes int64) (*WebSpeedResult, error) {
	u, err := normalizeWebURL(rawURL)
	if err != nil {
		return nil, err
	}
	if count <= 0 {
		count = 3
	}
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = "GET"
	}
	if method != "GET" && method != "HEAD" {
		return nil, fmt.Errorf("不支持的method: %s", method)
	}
	if downloadBytes <= 0 {
		downloadBytes = 64 * 1024 // 默认只读 64KB（足够衡量站点响应，不等同带宽）
	}
	if method == "HEAD" {
		downloadBytes = 0
	}

	res := &WebSpeedResult{
		URL:           u.String(),
		Method:        method,
		Count:         count,
		TimeoutSec:    int(timeout.Seconds()),
		DownloadBytes: downloadBytes,
		Attempts:      make([]WebSpeedAttempt, 0, count),
		Summary:       map[string]any{},
		TestTime:      time.Now().Format(time.RFC3339),
	}

	for i := 0; i < count; i++ {
		a := webSpeedOnce(u, method, timeout, downloadBytes)
		res.Attempts = append(res.Attempts, a)
	}

	// summary（简单平均/成功率）
	ok := 0
	sumTTFB := 0
	sumTotal := 0
	sumDNS := 0
	sumConn := 0
	sumTLS := 0
	for _, a := range res.Attempts {
		if a.OK {
			ok++
			sumDNS += a.DNSMs
			sumConn += a.ConnectMs
			sumTLS += a.TLSMs
			sumTTFB += a.TTFBMs
			sumTotal += a.TotalMs
		}
	}
	res.Summary["success"] = ok
	res.Summary["total"] = len(res.Attempts)
	if ok > 0 {
		res.Summary["avg_dns_ms"] = sumDNS / ok
		res.Summary["avg_connect_ms"] = sumConn / ok
		res.Summary["avg_tls_ms"] = sumTLS / ok
		res.Summary["avg_ttfb_ms"] = sumTTFB / ok
		res.Summary["avg_total_ms"] = sumTotal / ok
	}
	return res, nil
}

func normalizeWebURL(raw string) (*url.URL, error) {
	s := strings.TrimSpace(raw)
	if s == "" || s == "default" {
		// 默认给一个常见站点；面板通常会让用户输入
		s = "https://www.baidu.com"
	}
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" {
		u, err = url.Parse("https://" + s)
		if err != nil {
			return nil, err
		}
	}
	if u.Host == "" {
		return nil, fmt.Errorf("url 无效: %s", raw)
	}
	return u, nil
}

func webSpeedOnce(u *url.URL, method string, timeout time.Duration, downloadBytes int64) WebSpeedAttempt {
	a := WebSpeedAttempt{}
	start := time.Now()

	var dnsStart, connStart, tlsStart time.Time
	var gotConn, gotFirstByte time.Time
	var resolvedIP string

	trace := &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone: func(info httptrace.DNSDoneInfo) {
			if len(info.Addrs) > 0 {
				resolvedIP = info.Addrs[0].IP.String()
			}
			if !dnsStart.IsZero() {
				a.DNSMs = int(time.Since(dnsStart).Milliseconds())
			}
		},
		ConnectStart: func(_, _ string) { connStart = time.Now() },
		ConnectDone: func(_, addr string, _ error) {
			_ = addr
			if !connStart.IsZero() {
				a.ConnectMs = int(time.Since(connStart).Milliseconds())
			}
		},
		TLSHandshakeStart: func() { tlsStart = time.Now() },
		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			if !tlsStart.IsZero() {
				a.TLSMs = int(time.Since(tlsStart).Milliseconds())
			}
		},
		GotConn: func(_ httptrace.GotConnInfo) { gotConn = time.Now() },
		GotFirstResponseByte: func() {
			gotFirstByte = time.Now()
			if !gotConn.IsZero() {
				a.TTFBMs = int(time.Since(gotConn).Milliseconds())
			}
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, _ := http.NewRequestWithContext(httptrace.WithClientTrace(ctx, trace), method, u.String(), nil)
	req.Header.Set("User-Agent", "nwct-client/1.0")
	// 优先用 Range 控制体积；不支持 Range 时，我们也会 LimitReader。
	if method == "GET" && downloadBytes > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=0-%d", downloadBytes-1))
	}

	client := httpClientWithTransport()
	resp, err := client.Do(req)
	if err != nil {
		a.OK = false
		a.Error = err.Error()
		a.TotalMs = int(time.Since(start).Milliseconds())
		a.ResolvedIP = resolvedIP
		return a
	}
	defer resp.Body.Close()

	a.StatusCode = resp.StatusCode
	if resp.StatusCode >= 400 {
		a.OK = false
		a.Error = resp.Status
		a.TotalMs = int(time.Since(start).Milliseconds())
		a.ResolvedIP = resolvedIP
		return a
	}

	// 读下载窗口（HEAD 不读 body）
	if method == "GET" && downloadBytes > 0 {
		dlStart := time.Now()
		n, _ := io.Copy(io.Discard, io.LimitReader(resp.Body, downloadBytes))
		a.BytesRead = n
		a.DownloadMs = int(time.Since(dlStart).Milliseconds())
	}
	if gotFirstByte.IsZero() && !gotConn.IsZero() {
		// 某些情况下 GotFirstResponseByte 回调没触发，兜底用 header 到达
		a.TTFBMs = int(time.Since(gotConn).Milliseconds())
	}

	a.OK = true
	a.TotalMs = int(time.Since(start).Milliseconds())
	a.ResolvedIP = resolvedIP
	return a
}
