package frp

import (
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
