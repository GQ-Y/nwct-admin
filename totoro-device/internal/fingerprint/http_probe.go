package fingerprint

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type HTTPFingerprint struct {
	Scheme string `json:"scheme"`
	Server string `json:"server,omitempty"`
	Title  string `json:"title,omitempty"`
	Realm  string `json:"realm,omitempty"`
}

var reTitle = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
var reRealm = regexp.MustCompile(`(?i)realm=\"?([^\"\\s]+)\"?`)

// ProbeHTTPFingerprint 轻量抓取 HTTP/HTTPS 指纹：Server / Title / Basic realm
func ProbeHTTPFingerprint(ctx context.Context, host string, https bool) (*HTTPFingerprint, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 1200*time.Millisecond)
		defer cancel()
	}
	scheme := "http"
	if https {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s/", scheme, strings.TrimSpace(host))
	if strings.TrimSpace(host) == "" {
		return nil, fmt.Errorf("host 为空")
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // 仅用于指纹；不做安全校验
	}
	cli := &http.Client{Transport: tr, Timeout: 1500 * time.Millisecond}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("User-Agent", "nwct-fingerprint/1.0")
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	fp := &HTTPFingerprint{Scheme: scheme}
	fp.Server = strings.TrimSpace(resp.Header.Get("Server"))

	wwwAuth := resp.Header.Get("WWW-Authenticate")
	if wwwAuth != "" {
		if m := reRealm.FindStringSubmatch(wwwAuth); len(m) == 2 {
			fp.Realm = m[1]
		}
	}

	// 只读一点点 html 解析 title
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if m := reTitle.FindSubmatch(b); len(m) == 2 {
		title := strings.TrimSpace(string(m[1]))
		title = strings.ReplaceAll(title, "\n", " ")
		title = strings.Join(strings.Fields(title), " ")
		fp.Title = title
	}
	if fp.Server == "" && fp.Title == "" && fp.Realm == "" {
		return nil, fmt.Errorf("无可用指纹")
	}
	return fp, nil
}


