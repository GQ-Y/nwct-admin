package nps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

type ensureVKeyOptions struct {
	BaseURL  string
	Username string
	Password string
	Remark   string
	Timeout  time.Duration
}

type loginResp struct {
	Status int    `json:"status"`
	Msg    string `json:"msg"`
}

type clientListResp struct {
	Rows []struct {
		ID        int    `json:"Id"`
		Remark    string `json:"Remark"`
		VerifyKey string `json:"VerifyKey"`
		Addr      string `json:"Addr"`
		IsConnect bool   `json:"IsConnect"`
		Status    bool   `json:"Status"`
		Flow      struct {
			ExportFlow int64 `json:"ExportFlow"`
			InletFlow  int64 `json:"InletFlow"`
			FlowLimit  int64 `json:"FlowLimit"`
		} `json:"Flow"`
	} `json:"rows"`
	Total int `json:"total"`
}

// EnsureVKey 通过 NPS Web 管理端确保 remark 对应的 client 存在并返回 vkey。
// 备注：这是为“一键连接”测试体验提供的自动化能力；生产环境建议显式配置 vkey。
func EnsureVKey(ctx context.Context, baseURL, username, password, remark string) (string, error) {
	return ensureVKeyViaWeb(ctx, ensureVKeyOptions{
		BaseURL:  baseURL,
		Username: username,
		Password: password,
		Remark:   remark,
	})
}

// ensureVKeyViaWeb 尝试通过 NPS Web 管理端自动创建/查找 client 并返回 vkey。
// 这仅用于“测试/内置默认服务”场景：当设备端尚未配置 vkey 时，实现一键连接。
func ensureVKeyViaWeb(ctx context.Context, opts ensureVKeyOptions) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(opts.BaseURL), "/")
	if base == "" {
		return "", fmt.Errorf("nps web base_url 为空")
	}
	remark := strings.TrimSpace(opts.Remark)
	if remark == "" {
		return "", fmt.Errorf("remark 为空")
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	user := strings.TrimSpace(opts.Username)
	pass := opts.Password
	if user == "" || strings.TrimSpace(pass) == "" {
		return "", fmt.Errorf("nps web 用户名/密码未配置")
	}

	jar, _ := cookiejar.New(nil)
	cli := &http.Client{Jar: jar, Timeout: timeout}

	// 1) login
	{
		form := url.Values{}
		form.Set("username", user)
		form.Set("password", pass)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/login/verify", strings.NewReader(form.Encode()))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := cli.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", fmt.Errorf("nps web 登录失败: %s: %s", resp.Status, strings.TrimSpace(string(b)))
		}
		var lr loginResp
		_ = json.Unmarshal(b, &lr)
		if lr.Status != 1 {
			if lr.Msg == "" {
				lr.Msg = strings.TrimSpace(string(b))
			}
			return "", fmt.Errorf("nps web 登录失败: %s", lr.Msg)
		}
	}

	// 2) find existing
	if v, _ := findVKeyInClientList(ctx, cli, base, remark); v != "" {
		return v, nil
	}

	// 3) create new client (vkey 留空 => 自动生成)
	{
		form := url.Values{}
		form.Set("remark", remark)
		form.Set("u", "")
		form.Set("p", "")
		form.Set("vkey", "")
		form.Set("config_conn_allow", "1")
		form.Set("compress", "0")
		form.Set("crypt", "0")

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/client/add", strings.NewReader(form.Encode()))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := cli.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		// 兼容：页面通常返回 JSON/重定向/文本，这里不强依赖响应体，只要请求成功即可
		if resp.StatusCode < 200 || resp.StatusCode >= 400 {
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			return "", fmt.Errorf("nps web 创建 client 失败: %s: %s", resp.Status, strings.TrimSpace(string(b)))
		}
	}

	// 4) re-fetch list
	if v, _ := findVKeyInClientList(ctx, cli, base, remark); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("已尝试创建 client，但未能从列表解析到 vkey（remark=%s）", remark)
}

func findVKeyInClientList(ctx context.Context, cli *http.Client, baseURL, remark string) (string, error) {
	// NPS Web 的 bootstrap-table 使用 POST /client/list 返回 JSON
	// 参数：limit/offset/search/sort/order
	form := url.Values{}
	form.Set("limit", "200")
	form.Set("offset", "0")
	form.Set("search", "")
	form.Set("sort", "")
	form.Set("order", "desc")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/client/list", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := cli.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("读取 client/list 失败: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	var out clientListResp
	if err := json.Unmarshal(b, &out); err != nil {
		// 有些版本可能返回带额外字段的 JSON，这里给出可读错误
		return "", fmt.Errorf("解析 client/list JSON 失败: %v", err)
	}
	// 选取“remark 匹配且状态开启”的条目；如果有多个，优先返回在线的
	var bestOnline, bestAny string
	for _, r := range out.Rows {
		if strings.TrimSpace(r.Remark) != strings.TrimSpace(remark) {
			continue
		}
		if strings.TrimSpace(r.VerifyKey) == "" {
			continue
		}
		if r.IsConnect {
			bestOnline = r.VerifyKey
			break
		}
		if bestAny == "" {
			bestAny = r.VerifyKey
		}
	}
	if bestOnline != "" {
		return bestOnline, nil
	}
	if bestAny != "" {
		return bestAny, nil
	}
	return "", nil
}
