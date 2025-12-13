package fingerprint

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"net"
	"strings"
	"time"
)

// WSDiscoveryDevice 通过 WS-Discovery 发现的设备（常用于 ONVIF 摄像头/Windows 设备）
type WSDiscoveryDevice struct {
	IP     string   `json:"ip"`
	Types  []string `json:"types"`
	XAddrs []string `json:"xaddrs"`
	Scopes []string `json:"scopes"`
}

// WSDiscoveryProbe 在局域网广播 WS-Discovery Probe，并收集响应。
// 返回 map[ip]device
func WSDiscoveryProbe(ctx context.Context, timeout time.Duration) (map[string]*WSDiscoveryDevice, error) {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// UDP 3702 multicast
	raddr := &net.UDPAddr{IP: net.ParseIP("239.255.255.250"), Port: 3702}
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Probe message (minimal)
	msg := `<?xml version="1.0" encoding="UTF-8"?>
<e:Envelope xmlns:e="http://www.w3.org/2003/05/soap-envelope"
 xmlns:w="http://schemas.xmlsoap.org/ws/2004/08/addressing"
 xmlns:d="http://schemas.xmlsoap.org/ws/2005/04/discovery"
 xmlns:dn="http://www.onvif.org/ver10/network/wsdl">
 <e:Header>
  <w:MessageID>uuid:` + randomUUIDLike() + `</w:MessageID>
  <w:To>urn:schemas-xmlsoap-org:ws:2005:04:discovery</w:To>
  <w:Action>http://schemas.xmlsoap.org/ws/2005/04/discovery/Probe</w:Action>
 </e:Header>
 <e:Body>
  <d:Probe>
   <d:Types>dn:NetworkVideoTransmitter</d:Types>
  </d:Probe>
 </e:Body>
</e:Envelope>`

	_ = conn.SetWriteDeadline(time.Now().Add(500 * time.Millisecond))
	_, _ = conn.WriteToUDP([]byte(msg), raddr)

	out := map[string]*WSDiscoveryDevice{}
	buf := make([]byte, 64*1024)
	for {
		select {
		case <-ctx.Done():
			return out, nil
		default:
		}
		_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		n, from, err := conn.ReadFromUDP(buf)
		if err != nil {
			// timeout -> continue until ctx done
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			continue
		}
		ip := ""
		if from != nil && from.IP != nil {
			ip = from.IP.String()
		}
		if ip == "" {
			continue
		}
		dev := parseWSDiscoveryResponse(ip, buf[:n])
		if dev == nil {
			continue
		}
		// merge
		old := out[ip]
		if old == nil {
			out[ip] = dev
			continue
		}
		old.Types = mergeStrings(old.Types, dev.Types)
		old.XAddrs = mergeStrings(old.XAddrs, dev.XAddrs)
		old.Scopes = mergeStrings(old.Scopes, dev.Scopes)
	}
}

func mergeStrings(a, b []string) []string {
	m := map[string]bool{}
	out := make([]string, 0, len(a)+len(b))
	for _, s := range append(a, b...) {
		s = strings.TrimSpace(s)
		if s == "" || m[s] {
			continue
		}
		m[s] = true
		out = append(out, s)
	}
	return out
}

// 仅解析 ProbeMatches 中关键字段
type wsdEnvelope struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		ProbeMatches struct {
			ProbeMatch []struct {
				Types  string `xml:"Types"`
				Scopes string `xml:"Scopes"`
				XAddrs string `xml:"XAddrs"`
			} `xml:"ProbeMatch"`
		} `xml:"ProbeMatches"`
	} `xml:"Body"`
}

func parseWSDiscoveryResponse(ip string, payload []byte) *WSDiscoveryDevice {
	// 有些设备会带前缀/空白，尽量容错
	p := bytes.TrimSpace(payload)
	if len(p) == 0 || !bytes.Contains(p, []byte("ProbeMatches")) {
		return nil
	}
	var env wsdEnvelope
	if err := xml.Unmarshal(p, &env); err != nil {
		return nil
	}
	dev := &WSDiscoveryDevice{IP: ip}
	for _, pm := range env.Body.ProbeMatches.ProbeMatch {
		if pm.Types != "" {
			dev.Types = append(dev.Types, strings.Fields(pm.Types)...)
		}
		if pm.XAddrs != "" {
			dev.XAddrs = append(dev.XAddrs, strings.Fields(pm.XAddrs)...)
		}
		if pm.Scopes != "" {
			dev.Scopes = append(dev.Scopes, strings.Fields(pm.Scopes)...)
		}
	}
	dev.Types = mergeStrings(nil, dev.Types)
	dev.XAddrs = mergeStrings(nil, dev.XAddrs)
	dev.Scopes = mergeStrings(nil, dev.Scopes)
	if len(dev.Types) == 0 && len(dev.XAddrs) == 0 && len(dev.Scopes) == 0 {
		return nil
	}
	return dev
}

func randomUUIDLike() string {
	// 不需要强随机，仅用于 WS-Discovery messageId
	now := time.Now().UnixNano()
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uint32(now),
		uint16(now>>32),
		uint16(now>>16),
		uint16(now>>48),
		uint64(now)^0xabcdef,
	)
}


