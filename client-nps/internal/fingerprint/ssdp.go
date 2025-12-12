package fingerprint

import (
	"bufio"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// SSDPDevice 表示通过 UPnP/SSDP 发现到的设备信息（尽量通用、字段可空）
type SSDPDevice struct {
	IP           string `json:"ip"`
	Location     string `json:"location"`
	Server       string `json:"server"`
	USN          string `json:"usn"`
	ST           string `json:"st"`
	FriendlyName string `json:"friendly_name"`
	ModelName    string `json:"model_name"`
	Manufacturer string `json:"manufacturer"`
	DeviceType   string `json:"device_type"`
}

// SSDPDiscover 执行一次 M-SEARCH（ssdp:all），收集响应并尝试抓取描述 XML。
// 返回 map[ip]*SSDPDevice（同一 IP 多条响应时尽量融合/保留更完整的一条）。
func SSDPDiscover(ctx context.Context, timeout time.Duration) (map[string]*SSDPDevice, error) {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	deadline := time.Now().Add(timeout)

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	_ = conn.SetDeadline(deadline)

	// SSDP M-SEARCH 请求（UDP multicast）
	req := strings.Join([]string{
		"M-SEARCH * HTTP/1.1",
		"HOST: 239.255.255.250:1900",
		`MAN: "ssdp:discover"`,
		"MX: 1",
		"ST: ssdp:all",
		"", "",
	}, "\r\n")

	dst := &net.UDPAddr{IP: net.IPv4(239, 255, 255, 250), Port: 1900}
	if _, err := conn.WriteToUDP([]byte(req), dst); err != nil {
		return nil, err
	}

	// 收集响应
	out := map[string]*SSDPDevice{}
	buf := make([]byte, 64*1024)
	for {
		if ctx.Err() != nil {
			break
		}
		n, raddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			// deadline or other error: stop
			break
		}
		ip := ""
		if raddr != nil && raddr.IP != nil {
			ip = raddr.IP.String()
		}
		if ip == "" {
			continue
		}
		h := parseSSDPHeaders(buf[:n])
		if len(h) == 0 {
			continue
		}
		dev := &SSDPDevice{
			IP:       ip,
			Location: strings.TrimSpace(firstHeader(h, "location")),
			Server:   strings.TrimSpace(firstHeader(h, "server")),
			USN:      strings.TrimSpace(firstHeader(h, "usn")),
			ST:       strings.TrimSpace(firstHeader(h, "st")),
		}
		mergeSSDP(out, dev)
	}

	// 拉取 XML 描述（尽量不阻塞太久：并发少量抓取）
	cli := &http.Client{Timeout: 4 * time.Second}
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for _, dev := range out {
		if ctx.Err() != nil {
			break
		}
		loc := strings.TrimSpace(dev.Location)
		if loc == "" {
			continue
		}
		if u, err := url.Parse(loc); err != nil || u.Scheme == "" || u.Host == "" {
			continue
		}
		wg.Add(1)
		go func(d *SSDPDevice, location string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			info, err := fetchUPnPDescription(ctx, cli, location)
			if err != nil || info == nil {
				return
			}
			if info.FriendlyName != "" && d.FriendlyName == "" {
				d.FriendlyName = info.FriendlyName
			}
			if info.ModelName != "" && d.ModelName == "" {
				d.ModelName = info.ModelName
			}
			if info.Manufacturer != "" && d.Manufacturer == "" {
				d.Manufacturer = info.Manufacturer
			}
			if info.DeviceType != "" && d.DeviceType == "" {
				d.DeviceType = info.DeviceType
			}
		}(dev, loc)
	}
	wg.Wait()

	return out, nil
}

func mergeSSDP(m map[string]*SSDPDevice, dev *SSDPDevice) {
	old := m[dev.IP]
	if old == nil {
		m[dev.IP] = dev
		return
	}
	// 简单合并：保留非空字段；Location 优先有值
	if old.Location == "" && dev.Location != "" {
		old.Location = dev.Location
	}
	if old.Server == "" && dev.Server != "" {
		old.Server = dev.Server
	}
	if old.USN == "" && dev.USN != "" {
		old.USN = dev.USN
	}
	if old.ST == "" && dev.ST != "" {
		old.ST = dev.ST
	}
}

func parseSSDPHeaders(b []byte) map[string][]string {
	// 响应是类似 HTTP 的头部；这里做一个宽松解析
	m := map[string][]string{}
	sc := bufio.NewScanner(bytes.NewReader(b))
	first := true
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r\n")
		if first {
			first = false
			// 仅接受 HTTP/1.1 200 OK 等
			if !strings.Contains(line, "200") {
				// 也可能是 NOTIFY / 其它，这里先跳过
			}
			continue
		}
		if strings.TrimSpace(line) == "" {
			break
		}
		i := strings.IndexByte(line, ':')
		if i <= 0 {
			continue
		}
		k := strings.ToLower(strings.TrimSpace(line[:i]))
		v := strings.TrimSpace(line[i+1:])
		if k == "" || v == "" {
			continue
		}
		m[k] = append(m[k], v)
	}
	return m
}

func firstHeader(h map[string][]string, key string) string {
	k := strings.ToLower(strings.TrimSpace(key))
	v := h[k]
	if len(v) == 0 {
		return ""
	}
	return v[0]
}

type upnpDescRoot struct {
	Device upnpDescDevice `xml:"device"`
}
type upnpDescDevice struct {
	DeviceType   string `xml:"deviceType"`
	FriendlyName string `xml:"friendlyName"`
	Manufacturer string `xml:"manufacturer"`
	ModelName    string `xml:"modelName"`
}

func fetchUPnPDescription(ctx context.Context, cli *http.Client, location string) (*SSDPDevice, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "nwct/1.0 UPnP/SSDP")

	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, err
	}

	var root upnpDescRoot
	if err := xml.Unmarshal(body, &root); err != nil {
		return nil, err
	}
	return &SSDPDevice{
		Location:     location,
		FriendlyName: strings.TrimSpace(root.Device.FriendlyName),
		ModelName:    strings.TrimSpace(root.Device.ModelName),
		Manufacturer: strings.TrimSpace(root.Device.Manufacturer),
		DeviceType:   strings.TrimSpace(root.Device.DeviceType),
	}, nil
}


