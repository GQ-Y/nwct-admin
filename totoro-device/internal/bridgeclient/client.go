package bridgeclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"totoro-device/internal/cryptobox"
)

type Client struct {
	BaseURL          string
	DeviceToken      string
	HTTP             *http.Client
	DeviceID         string
	DevicePrivKeyB64 string
}

type RegisterReq struct {
	DeviceID string `json:"device_id"`
	MAC      string `json:"mac"`
	PubKey   string `json:"pub_key"`
}

type RegisterResp struct {
	DeviceToken   string         `json:"device_token"`
	ExpiresAt     string         `json:"expires_at"`
	OfficialNodes []OfficialNode `json:"official_nodes"`
	PublicNodes   []PublicNode   `json:"public_nodes"`
}

type OfficialNode struct {
	NodeID       string `json:"node_id"`
	Name         string `json:"name"`
	Server       string `json:"server"`
	Token        string `json:"token"`
	AdminAddr    string `json:"admin_addr"`
	AdminUser    string `json:"admin_user"`
	AdminPwd     string `json:"admin_pwd"`
	NodeAPI      string `json:"node_api"`
	DomainSuffix string `json:"domain_suffix"`
	HTTPEnabled  bool   `json:"http_enabled"`
	HTTPSEnabled bool   `json:"https_enabled"`
	UpdatedAt    string `json:"updated_at"`
}

type PublicNode struct {
	NodeID        string   `json:"node_id"`
	Name          string   `json:"name"`
	Public        bool     `json:"public"`
	Status        string   `json:"status"`
	Region        string   `json:"region"`
	ISP           string   `json:"isp"`
	Tags          []string `json:"tags"`
	Endpoints     []any    `json:"endpoints"`
	NodeAPI       string   `json:"node_api"`
	DomainSuffix  string   `json:"domain_suffix"`
	UpdatedAt     string   `json:"updated_at"`
	HeartbeatAgeS int64    `json:"heartbeat_age_s"`
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

func (c *Client) Register(deviceID, mac string, pubKeyB64 string) (*RegisterResp, error) {
	c.ensureHTTP()
	base := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if base == "" {
		return nil, fmt.Errorf("bridge base url empty")
	}
	u := base + "/api/v1/device/register"
	reqBody := RegisterReq{DeviceID: deviceID, MAC: mac, PubKey: strings.TrimSpace(pubKeyB64)}
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
	pt, err := c.decodeEncryptedOrPlainBytes(wrap.Data, deviceID)
	if err != nil {
		return nil, err
	}
	var out RegisterResp
	if err := json.Unmarshal(pt, &out); err != nil {
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
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&wrap)
	if wrap.Code != 0 {
		return nil, fmt.Errorf("bridge public nodes failed")
	}
	pt, err := c.decodeEncryptedOrPlainBytes(wrap.Data, c.DeviceID)
	if err != nil {
		return nil, err
	}
	var out struct {
		Nodes []any `json:"nodes"`
	}
	if err := json.Unmarshal(pt, &out); err != nil {
		return nil, err
	}
	return out.Nodes, nil
}

func (c *Client) GetOfficialNodes() ([]OfficialNode, error) {
	c.ensureHTTP()
	base := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	u := base + "/api/v1/official/nodes"
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	req.Header.Set("X-Device-Token", strings.TrimSpace(c.DeviceToken))
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("bridge official nodes status=%s", resp.Status)
	}
	var wrap struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&wrap)
	if wrap.Code != 0 {
		return nil, fmt.Errorf("bridge official nodes failed")
	}
	pt, err := c.decodeEncryptedOrPlainBytes(wrap.Data, c.DeviceID)
	if err != nil {
		return nil, err
	}
	var out struct {
		Nodes []OfficialNode `json:"nodes"`
	}
	if err := json.Unmarshal(pt, &out); err != nil {
		return nil, err
	}
	return out.Nodes, nil
}

type RedeemResp struct {
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

type PublicNodeConnectResp struct {
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

func (c *Client) ConnectPublicNode(nodeID string) (*PublicNodeConnectResp, error) {
	c.ensureHTTP()
	base := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	u := base + "/api/v1/public/nodes/connect"
	body := fmt.Sprintf(`{"node_id":%q}`, strings.TrimSpace(nodeID))
	req, _ := http.NewRequest(http.MethodPost, u, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Device-Token", strings.TrimSpace(c.DeviceToken))
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("bridge public connect status=%s", resp.Status)
	}
	var wrap struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&wrap)
	if wrap.Code != 0 {
		return nil, fmt.Errorf("bridge public connect failed")
	}
	pt, err := c.decodeEncryptedOrPlainBytes(wrap.Data, c.DeviceID)
	if err != nil {
		return nil, err
	}
	var out PublicNodeConnectResp
	if err := json.Unmarshal(pt, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type PreviewResp struct {
	Node struct {
		NodeID    string `json:"node_id"`
		Endpoints []struct {
			Addr  string `json:"addr"`
			Port  int    `json:"port"`
			Proto string `json:"proto"`
		} `json:"endpoints"`
		DomainSuffix string `json:"domain_suffix"`
	} `json:"node"`
	InviteID  string `json:"invite_id"`
	ExpiresAt string `json:"expires_at"`
}

func (c *Client) PreviewInvite(code string) (*PreviewResp, error) {
	c.ensureHTTP()
	base := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	u := base + "/api/v1/invites/preview"
	body := fmt.Sprintf(`{"code":%q}`, strings.TrimSpace(code))
	req, _ := http.NewRequest(http.MethodPost, u, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Device-Token", strings.TrimSpace(c.DeviceToken))
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("bridge preview status=%s", resp.Status)
	}
	var wrap struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&wrap)
	if wrap.Code != 0 {
		return nil, fmt.Errorf("bridge preview failed")
	}
	pt, err := c.decodeEncryptedOrPlainBytes(wrap.Data, c.DeviceID)
	if err != nil {
		return nil, err
	}
	var out PreviewResp
	if err := json.Unmarshal(pt, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RedeemInvite(code string) (*RedeemResp, error) {
	c.ensureHTTP()
	base := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	u := base + "/api/v1/invites/redeem"
	body := fmt.Sprintf(`{"code":%q}`, strings.TrimSpace(code))
	req, _ := http.NewRequest(http.MethodPost, u, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Device-Token", strings.TrimSpace(c.DeviceToken))
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("bridge redeem status=%s", resp.Status)
	}
	var wrap struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&wrap)
	if wrap.Code != 0 {
		return nil, fmt.Errorf("bridge redeem failed")
	}
	pt, err := c.decodeEncryptedOrPlainBytes(wrap.Data, c.DeviceID)
	if err != nil {
		return nil, err
	}
	var out RedeemResp
	if err := json.Unmarshal(pt, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) decodeEncryptedOrPlainBytes(data json.RawMessage, deviceID string) ([]byte, error) {
	// encrypted
	var env struct {
		Encrypted cryptobox.EncryptedPayload `json:"encrypted"`
	}
	if err := json.Unmarshal(data, &env); err == nil && strings.TrimSpace(env.Encrypted.Ciphertext) != "" {
		if strings.TrimSpace(c.DevicePrivKeyB64) == "" || strings.TrimSpace(deviceID) == "" {
			return nil, fmt.Errorf("device decrypt not configured")
		}
		pt, err := cryptobox.DecryptFromBridge(c.DevicePrivKeyB64, deviceID, env.Encrypted)
		if err != nil {
			return nil, err
		}
		return pt, nil
	}
	// plain (admin or legacy)
	return data, nil
}
