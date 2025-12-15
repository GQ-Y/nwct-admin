package fingerprint

import (
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

// MDNSReverseLookup 尝试通过 mDNS 做反向解析（PTR: x.x.x.x.in-addr.arpa），成功返回主机名（去掉末尾点）。
// 典型用于补全 Apple/iOS/IoT 设备的 `.local` 名称。
func MDNSReverseLookup(ip string, timeout time.Duration) (string, error) {
	if timeout <= 0 {
		timeout = 400 * time.Millisecond
	}
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return "", fmt.Errorf("invalid ip")
	}
	v4 := parsed.To4()
	if v4 == nil {
		return "", fmt.Errorf("only ipv4 supported")
	}

	qname := fmt.Sprintf("%d.%d.%d.%d.in-addr.arpa.", v4[3], v4[2], v4[1], v4[0])
	name, err := dnsmessage.NewName(qname)
	if err != nil {
		return "", err
	}

	var b dnsmessage.Builder
	msg := make([]byte, 0, 512)
	b = dnsmessage.NewBuilder(msg, dnsmessage.Header{
		// mDNS uses ID=0
		ID:                 0,
		Response:           false,
		OpCode:             0,
		RecursionDesired:   false,
		RecursionAvailable: false,
	})
	b.EnableCompression()
	if err := b.StartQuestions(); err != nil {
		return "", err
	}
	if err := b.Question(dnsmessage.Question{
		Name:  name,
		Type:  dnsmessage.TypePTR,
		Class: dnsmessage.ClassINET,
	}); err != nil {
		return "", err
	}
	wire, err := b.Finish()
	if err != nil {
		return "", err
	}

	raddr := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: 5353}
	conn, err := net.DialUDP("udp4", nil, raddr)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	if _, err := conn.Write(wire); err != nil {
		return "", err
	}

	buf := make([]byte, 512) // 从 2048 降到 512，mDNS 响应通常很小
	n, err := conn.Read(buf)
	if err != nil {
		return "", err
	}
	buf = buf[:n]

	var p dnsmessage.Parser
	if _, err := p.Start(buf); err != nil {
		return "", err
	}
	// skip questions
	if err := p.SkipAllQuestions(); err != nil {
		return "", err
	}

	for {
		ah, err := p.AnswerHeader()
		if err != nil {
			break
		}
		if ah.Type != dnsmessage.TypePTR {
			if err := p.SkipAnswer(); err != nil {
				break
			}
			continue
		}
		ptr, err := p.PTRResource()
		if err != nil {
			return "", err
		}
		// 只取我们查询的 PTR 记录
		if strings.EqualFold(ah.Name.String(), qname) {
			host := strings.TrimSuffix(ptr.PTR.String(), ".")
			return host, nil
		}
	}

	return "", fmt.Errorf("no mdns ptr")
}


