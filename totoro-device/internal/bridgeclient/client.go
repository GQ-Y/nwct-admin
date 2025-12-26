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
	BaseURL      string
	DeviceToken  string
	HTTP         *http.Client
}

type RegisterReq struct {
	DeviceID string `json:"device_id"`
	MAC      string `json:"mac"`
}

type RegisterResp struct {
	DeviceToken   string `json:"device_token"`
	ExpiresAt     string `json:"expires_at"`
	OfficialNodes []any  `json:"official_nodes"`
	PublicNodes   []any  `json:"public_nodes"`
}

func ParseExpiresAt(expiresAt string) int64 {
	expiresAt = strings.TrimSpace(expiresAt)
	if expiresAt == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return 0
	}
	return t.Unix()
}

func (c *Client) ensureHTTP() {
	if c.HTTP == nil {
		c.HTTP = &http.Client{Timeout: 6 * time.Second}
	}
}

func (c *Client) Register(deviceID, mac string) (*RegisterResp, error) {
	c.ensureHTTP()
	base := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if base == "" {
		return nil, fmt.Errorf("bridge base url empty")
	}
	u := base + "/api/v1/device/register"
	reqBody := RegisterReq{DeviceID: deviceID, MAC: mac}
	b, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, u, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("bridge register status=%s", resp.Status)
	}
	var wrap struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
		Msg  string          `json:"message"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&wrap)
	if wrap.Code != 0 {
		return nil, fmt.Errorf("bridge register failed")
	}
	var out RegisterResp
	if err := json.Unmarshal(wrap.Data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetPublicNodes() ([]any, error) {
	c.ensureHTTP()
	base := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	u := base + "/api/v1/public/nodes"
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	req.Header.Set("X-Device-Token", strings.TrimSpace(c.DeviceToken))
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("bridge public nodes status=%s", resp.Status)
	}
	var wrap struct {
		Code int `json:"code"`
		Data struct {
			Nodes []any `json:"nodes"`
		} `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&wrap)
	return wrap.Data.Nodes, nil
}


