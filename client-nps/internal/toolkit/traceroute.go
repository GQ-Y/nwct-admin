package toolkit

import (
	"fmt"
	"net"
	"time"
)

// Hop Traceroute跳转信息
type Hop struct {
	Hop     int     `json:"hop"`
	IP      string  `json:"ip"`
	Hostname string `json:"hostname"`
	Latency float64 `json:"latency"`
}

// TracerouteResult Traceroute结果
type TracerouteResult struct {
	Target string `json:"target"`
	Hops   []Hop  `json:"hops"`
}

// Traceroute 执行Traceroute
func Traceroute(target string, maxHops int, timeout time.Duration) (*TracerouteResult, error) {
	if maxHops <= 0 {
		maxHops = 30
	}

	result := &TracerouteResult{
		Target: target,
		Hops:   []Hop{},
	}

	// 解析目标地址（验证目标是否有效）
	_, err := net.ResolveIPAddr("ip", target)
	if err != nil {
		return nil, fmt.Errorf("解析目标地址失败: %v", err)
	}

	// 简化实现：使用TCP连接模拟traceroute
	// 实际traceroute需要使用ICMP或UDP，需要root权限
	for hop := 1; hop <= maxHops; hop++ {
		// 尝试连接目标（简化实现）
		start := time.Now()
		conn, err := net.DialTimeout("tcp", target+":80", timeout)
		latency := time.Since(start).Seconds() * 1000

		hopInfo := Hop{
			Hop:     hop,
			Latency: latency,
		}

		if err == nil {
			// 连接成功，到达目标
			hopInfo.IP = conn.RemoteAddr().(*net.TCPAddr).IP.String()
			conn.Close()

			// 尝试反向DNS
			names, _ := net.LookupAddr(hopInfo.IP)
			if len(names) > 0 {
				hopInfo.Hostname = names[0]
			}

			result.Hops = append(result.Hops, hopInfo)
			break
		}

		// 连接失败，记录中间跳
		hopInfo.IP = "*"
		result.Hops = append(result.Hops, hopInfo)

		// 如果已经尝试多次，停止
		if hop > 10 {
			break
		}
	}

	return result, nil
}

