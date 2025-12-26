package frp

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type InviteResolveResult struct {
	NodeID       string
	Endpoints    []map[string]any
	DomainSuffix string
	Server       string
	Ticket       string
	ExpiresAtRFC string
}

// ResolveInviteToTicket 调用节点的 /api/v1/invites/resolve，用邀请码换取连接票据（ticket）
// 返回 server（取 endpoints[0]），ticket，以及 expires_at（RFC3339）。
func ResolveInviteToTicket(nodeAPI, code string) (*InviteResolveResult, error) {
	nodeAPI = strings.TrimRight(strings.TrimSpace(nodeAPI), "/")
	code = strings.TrimSpace(code)
	if nodeAPI == "" || code == "" {
		return nil, fmt.Errorf("node_api/code 不能为空")
	}

	u := nodeAPI + "/api/v1/invites/resolve"
	body := fmt.Sprintf(`{"code":%q}`, code)
	req, _ := http.NewRequest(http.MethodPost, u, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 6 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("节点不可达: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var wrap struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil, fmt.Errorf("节点响应解析失败")
	}
	if wrap.Code != 0 {
		msg := strings.TrimSpace(wrap.Message)
		if msg == "" {
			msg = "节点解析邀请码失败"
		}
		return nil, fmt.Errorf("%s", msg)
	}

	var data struct {
		Node struct {
			NodeID    string `json:"node_id"`
			Endpoints []struct {
				Addr  string `json:"addr"`
				Port  int    `json:"port"`
				Proto string `json:"proto"`
			} `json:"endpoints"`
			DomainSuffix string `json:"domain_suffix"`
		} `json:"node"`
		ConnectionTicket string `json:"connection_ticket"`
		ExpiresAt        string `json:"expires_at"`
	}
	if err := json.Unmarshal(wrap.Data, &data); err != nil {
		return nil, fmt.Errorf("节点响应解析失败")
	}
	if len(data.Node.Endpoints) == 0 {
		return nil, fmt.Errorf("节点缺少 endpoints")
	}
	ep := data.Node.Endpoints[0]
	if strings.TrimSpace(ep.Addr) == "" || ep.Port <= 0 {
		return nil, fmt.Errorf("节点 endpoints 无效")
	}
	if strings.TrimSpace(data.ConnectionTicket) == "" {
		return nil, fmt.Errorf("节点未返回 connection_ticket")
	}

	eps := make([]map[string]any, 0, len(data.Node.Endpoints))
	for _, e := range data.Node.Endpoints {
		eps = append(eps, map[string]any{
			"addr":  e.Addr,
			"port":  e.Port,
			"proto": e.Proto,
		})
	}

	return &InviteResolveResult{
		NodeID:       strings.TrimSpace(data.Node.NodeID),
		Endpoints:    eps,
		DomainSuffix: strings.TrimSpace(data.Node.DomainSuffix),
		Server:       fmt.Sprintf("%s:%d", ep.Addr, ep.Port),
		Ticket:       strings.TrimSpace(data.ConnectionTicket),
		ExpiresAtRFC: strings.TrimSpace(data.ExpiresAt),
	}, nil
}

func TicketExpired(expiresAtRFC string, skew time.Duration) bool {
	expiresAtRFC = strings.TrimSpace(expiresAtRFC)
	if expiresAtRFC == "" {
		return true
	}
	t, err := time.Parse(time.RFC3339, expiresAtRFC)
	if err != nil {
		return true
	}
	return time.Now().Add(skew).After(t)
}

// TicketExpiredByTokenOrRFC 优先使用 expiresAtRFC；若为空/解析失败，则从 JWT ticket 的 exp 字段推断过期时间（无需验签）。
func TicketExpiredByTokenOrRFC(expiresAtRFC string, ticket string, skew time.Duration) bool {
	expiresAtRFC = strings.TrimSpace(expiresAtRFC)
	if expiresAtRFC != "" {
		if t, err := time.Parse(time.RFC3339, expiresAtRFC); err == nil {
			return time.Now().Add(skew).After(t)
		}
	}
	exp, ok := ticketExpUnix(ticket)
	if !ok {
		return true
	}
	return time.Now().Add(skew).After(time.Unix(exp, 0))
}

func ticketExpUnix(ticket string) (int64, bool) {
	// JWT: header.payload.signature
	parts := strings.Split(strings.TrimSpace(ticket), ".")
	if len(parts) < 2 {
		return 0, false
	}
	payloadB64 := parts[1]
	// base64url decode（补齐 padding）
	if m := len(payloadB64) % 4; m != 0 {
		payloadB64 += strings.Repeat("=", 4-m)
	}
	b, err := base64.URLEncoding.DecodeString(payloadB64)
	if err != nil {
		return 0, false
	}
	var p struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(b, &p); err != nil {
		return 0, false
	}
	if p.Exp <= 0 {
		return 0, false
	}
	return p.Exp, true
}
