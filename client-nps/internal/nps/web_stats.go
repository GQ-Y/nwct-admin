package nps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type npsWebStats struct {
	ClientsOnline     int
	TrafficInBytes    int64
	TrafficOutBytes   int64
	TotalTrafficBytes int64
	Tunnels           []Tunnel
}

func fetchNPSWebStats(ctx context.Context, baseURL, username, password string) (*npsWebStats, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return nil, fmt.Errorf("nps web base_url 为空")
	}
	user := strings.TrimSpace(username)
	pass := strings.TrimSpace(password)
	if user == "" || pass == "" {
		return nil, fmt.Errorf("nps web 用户名/密码未配置")
	}

	jar, _ := cookiejar.New(nil)
	cli := &http.Client{Jar: jar, Timeout: 6 * time.Second}

	// login
	{
		form := url.Values{}
		form.Set("username", user)
		form.Set("password", pass)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/login/verify", strings.NewReader(form.Encode()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := cli.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("nps web 登录失败: %s: %s", resp.Status, strings.TrimSpace(string(b)))
		}
		var lr loginResp
		_ = json.Unmarshal(b, &lr)
		if lr.Status != 1 {
			if lr.Msg == "" {
				lr.Msg = strings.TrimSpace(string(b))
			}
			return nil, fmt.Errorf("nps web 登录失败: %s", lr.Msg)
		}
	}

	// client/list
	{
		form := url.Values{}
		form.Set("limit", "200")
		form.Set("offset", "0")
		form.Set("search", "")
		form.Set("sort", "")
		form.Set("order", "desc")

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/client/list", strings.NewReader(form.Encode()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := cli.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			return nil, fmt.Errorf("读取 client/list 失败: %s: %s", resp.Status, strings.TrimSpace(string(b)))
		}
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
		var out clientListResp
		if err := json.Unmarshal(b, &out); err != nil {
			return nil, fmt.Errorf("解析 client/list JSON 失败: %v", err)
		}

		st := &npsWebStats{}
		for _, r := range out.Rows {
			if r.IsConnect {
				st.ClientsOnline++
			}
			st.TrafficInBytes += r.Flow.InletFlow
			st.TrafficOutBytes += r.Flow.ExportFlow
		}
		st.TotalTrafficBytes = st.TrafficInBytes + st.TrafficOutBytes

		// tunnels（按类型分别拉取，再合并）
		if tunnels, err := fetchNPSTunnelsViaWeb(ctx, cli, base); err == nil {
			st.Tunnels = tunnels
		}
		return st, nil
	}
}

type tunnelListResp struct {
	Rows []struct {
		ID        int    `json:"Id"`
		Remark    string `json:"Remark"`
		Mode      string `json:"Mode"`
		Port      int    `json:"Port"`
		Status    bool   `json:"Status"`
		RunStatus bool   `json:"RunStatus"`
		Target    struct {
			TargetStr string `json:"TargetStr"`
		} `json:"Target"`
		Client struct {
			ID        int    `json:"Id"`
			IsConnect bool   `json:"IsConnect"`
			VerifyKey string `json:"VerifyKey"`
		} `json:"Client"`
	} `json:"rows"`
	Total int `json:"total"`
}

func fetchNPSTunnelsViaWeb(ctx context.Context, cli *http.Client, baseURL string) ([]Tunnel, error) {
	types := []string{"tcp", "udp", "http", "socks5", "secret", "p2p", "file"}
	out := make([]Tunnel, 0, 32)

	for _, tp := range types {
		form := url.Values{}
		form.Set("limit", "200")
		form.Set("offset", "0")
		form.Set("type", tp)
		form.Set("client_id", "")
		form.Set("search", "")

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/index/gettunnel", strings.NewReader(form.Encode()))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := cli.Do(req)
		if err != nil {
			continue
		}
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
		_ = resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			continue
		}

		var lr tunnelListResp
		if err := json.Unmarshal(b, &lr); err != nil {
			continue
		}

		for _, r := range lr.Rows {
			localPort := parsePortFromTarget(r.Target.TargetStr)
			status := "offline"
			if r.RunStatus && r.Client.IsConnect {
				status = "online"
			} else if r.Status && r.Client.IsConnect {
				// 未运行但开启：给一个更“宽松”的在线判断
				status = "online"
			}
			out = append(out, Tunnel{
				ID:         fmt.Sprintf("%d", r.ID),
				Type:       tp,
				LocalPort:  localPort,
				RemotePort: r.Port,
				Status:     status,
			})
		}
	}
	return out, nil
}

func parsePortFromTarget(target string) int {
	// 形如 "127.0.0.1:22" / "10.0.0.2:8080"；失败则返回 0
	s := strings.TrimSpace(target)
	if s == "" {
		return 0
	}
	// 去掉可能的协议头
	s = strings.TrimPrefix(s, "tcp://")
	s = strings.TrimPrefix(s, "udp://")
	// 最后一个冒号后的数字
	i := strings.LastIndex(s, ":")
	if i < 0 || i == len(s)-1 {
		return 0
	}
	p, err := strconv.Atoi(strings.TrimSpace(s[i+1:]))
	if err != nil || p < 0 || p > 65535 {
		return 0
	}
	return p
}

func formatBytesIEC(n int64) string {
	if n < 0 {
		n = 0
	}
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	suffix := []string{"K", "M", "G", "T", "P", "E"}
	if exp >= len(suffix) {
		exp = len(suffix) - 1
	}
	return fmt.Sprintf("%.1f%s", float64(n)/float64(div), suffix[exp])
}


