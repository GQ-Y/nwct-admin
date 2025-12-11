package toolkit

import (
	"net"
	"time"
)

// PingResult Ping测试结果
type PingResult struct {
	Target          string    `json:"target"`
	PacketsSent     int       `json:"packets_sent"`
	PacketsReceived int       `json:"packets_received"`
	PacketLoss      float64   `json:"packet_loss"`
	MinLatency      float64   `json:"min_latency"`
	MaxLatency      float64   `json:"max_latency"`
	AvgLatency      float64   `json:"avg_latency"`
	Results         []PingPacket `json:"results"`
}

// PingPacket 单个Ping包结果
type PingPacket struct {
	Sequence int     `json:"sequence"`
	Latency  float64 `json:"latency"`
	Status   string  `json:"status"`
}

// Ping 执行Ping测试
func Ping(target string, count int, timeout time.Duration) (*PingResult, error) {
	result := &PingResult{
		Target:      target,
		PacketsSent: count,
		Results:     make([]PingPacket, 0, count),
	}

	// TODO: 实现ICMP Ping
	// 这里使用TCP连接作为简单实现
	for i := 0; i < count; i++ {
		start := time.Now()
		conn, err := net.DialTimeout("tcp", target+":80", timeout)
		latency := time.Since(start).Seconds() * 1000 // 转换为毫秒

		packet := PingPacket{
			Sequence: i + 1,
			Latency:  latency,
		}

		if err != nil {
			packet.Status = "failed"
		} else {
			packet.Status = "success"
			result.PacketsReceived++
			conn.Close()
		}

		result.Results = append(result.Results, packet)

		if i < count-1 {
			time.Sleep(time.Second)
		}
	}

	if result.PacketsSent > 0 {
		result.PacketLoss = float64(result.PacketsSent-result.PacketsReceived) / float64(result.PacketsSent) * 100
	}

	// 计算最小、最大、平均延迟
	if len(result.Results) > 0 {
		min := result.Results[0].Latency
		max := result.Results[0].Latency
		sum := 0.0
		count := 0

		for _, r := range result.Results {
			if r.Status == "success" {
				if r.Latency < min {
					min = r.Latency
				}
				if r.Latency > max {
					max = r.Latency
				}
				sum += r.Latency
				count++
			}
		}

		result.MinLatency = min
		result.MaxLatency = max
		if count > 0 {
			result.AvgLatency = sum / float64(count)
		}
	}

	return result, nil
}

