package bridgeclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL string
	NodeID  string
	NodeKey string
	HTTP    *http.Client
}

type Heartbeat struct {
	Ts           string        `json:"ts"`
	NodeID       string        `json:"node_id"`
	Public       bool          `json:"public"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Region       string        `json:"region"`
	ISP          string        `json:"isp"`
	Tags         []string      `json:"tags"`
	Endpoints    []any         `json:"endpoints"`
	NodeAPI      string        `json:"node_api,omitempty"`
	DomainSuffix string        `json:"domain_suffix"`
	HTTPEnabled  bool          `json:"http_enabled"`
	HTTPSEnabled bool          `json:"https_enabled"`
	TCPPortPool  map[string]int `json:"tcp_port_pool,omitempty"`
	UDPPortPool  map[string]int `json:"udp_port_pool,omitempty"`
	Version      any           `json:"version,omitempty"`
	Metrics      any           `json:"metrics,omitempty"`
	Extra        any           `json:"extra,omitempty"`
}

func (c *Client) SendHeartbeat(hb Heartbeat) error {
	if c.HTTP == nil {
		c.HTTP = &http.Client{Timeout: 5 * time.Second}
	}
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		return fmt.Errorf("bridge base url empty")
	}
	u := base + "/api/v1/nodes/heartbeat"

	b, err := json.Marshal(hb)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Node-Id", c.NodeID)
	req.Header.Set("X-Node-Key", c.NodeKey)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("bridge heartbeat status=%s", resp.Status)
	}
	return nil
}

type CreateInviteReq struct {
	ScopeJSON  string `json:"scope_json"`
	TTLSeconds int    `json:"ttl_s"`
	MaxUses    int    `json:"max_uses"`
}

type CreateInviteResp struct {
	InviteID  string `json:"invite_id"`
	Code      string `json:"code"`
	ExpiresAt string `json:"expires_at"`
}

func (c *Client) CreateInvite(reqBody CreateInviteReq) (*CreateInviteResp, error) {
	if c.HTTP == nil {
		c.HTTP = &http.Client{Timeout: 5 * time.Second}
	}
	base := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if base == "" {
		return nil, fmt.Errorf("bridge base url empty")
	}
	u := base + "/api/v1/nodes/invites/create"
	b, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, u, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Node-Id", c.NodeID)
	req.Header.Set("X-Node-Key", c.NodeKey)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("bridge create invite status=%s", resp.Status)
	}
	var wrap struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
		Message string       `json:"message"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&wrap)
	if wrap.Code != 0 {
		return nil, fmt.Errorf("bridge create invite failed")
	}
	var out CreateInviteResp
	if err := json.Unmarshal(wrap.Data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RevokeInvite(inviteID string) error {
	if c.HTTP == nil {
		c.HTTP = &http.Client{Timeout: 5 * time.Second}
	}
	base := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if base == "" {
		return fmt.Errorf("bridge base url empty")
	}
	u := base + "/api/v1/nodes/invites/revoke"
	body := fmt.Sprintf(`{"invite_id":%q}`, strings.TrimSpace(inviteID))
	req, _ := http.NewRequest(http.MethodPost, u, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Node-Id", c.NodeID)
	req.Header.Set("X-Node-Key", c.NodeKey)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("bridge revoke invite status=%s", resp.Status)
	}
	var wrap struct {
		Code int `json:"code"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&wrap)
	if wrap.Code != 0 {
		return fmt.Errorf("bridge revoke invite failed")
	}
	return nil
}


