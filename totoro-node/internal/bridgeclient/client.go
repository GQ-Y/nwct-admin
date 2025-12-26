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
	Region       string        `json:"region"`
	ISP          string        `json:"isp"`
	Tags         []string      `json:"tags"`
	Endpoints    []any         `json:"endpoints"`
	DomainSuffix string        `json:"domain_suffix"`
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


