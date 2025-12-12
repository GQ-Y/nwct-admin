package toolkit

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// PingResult Ping测试结果
type PingResult struct {
	Target          string       `json:"target"`
	PacketsSent     int          `json:"packets_sent"`
	PacketsReceived int          `json:"packets_received"`
	PacketLoss      float64      `json:"packet_loss"`
	MinLatency      float64      `json:"min_latency"`
	MaxLatency      float64      `json:"max_latency"`
	AvgLatency      float64      `json:"avg_latency"`
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
	if count <= 0 {
		count = 4
	}
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	// 仅使用系统 ping（结果真实）；失败直接返回错误
	return pingWithSystemCommand(target, count, timeout)
}

func pingWithSystemCommand(target string, count int, timeout time.Duration) (*PingResult, error) {
	pingPath, err := exec.LookPath("ping")
	if err != nil {
		return nil, fmt.Errorf("ping not found")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Duration(count)+2*time.Second)
	defer cancel()

	args := []string{}
	switch runtime.GOOS {
	case "darwin":
		// -c count, -n no DNS, -W waittime (ms)
		args = append(args, "-c", strconv.Itoa(count), "-n", "-W", strconv.Itoa(int(timeout.Milliseconds())), target)
	case "linux":
		// -c count, -n no DNS, -W timeout (seconds per reply)
		sec := int(timeout.Seconds())
		if sec <= 0 {
			sec = 1
		}
		args = append(args, "-c", strconv.Itoa(count), "-n", "-W", strconv.Itoa(sec), target)
	default:
		// 其他平台先不实现，交给回退逻辑
		return nil, fmt.Errorf("unsupported platform for system ping")
	}

	cmd := exec.CommandContext(ctx, pingPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// ping 非 0 也可能代表部分丢包，这里仍尝试解析；但如果完全无法解析则报错
	}

	lines := bytes.Split(out, []byte("\n"))
	re := regexp.MustCompile(`icmp[_-]?seq[= ](\d+).*time[= ]([\d.]+)\s*ms`)

	results := make(map[int]float64)
	for _, ln := range lines {
		m := re.FindSubmatch(ln)
		if len(m) == 3 {
			seq, _ := strconv.Atoi(string(m[1]))
			lat, _ := strconv.ParseFloat(string(m[2]), 64)
			// mac 的 icmp_seq 从 0 开始；统一映射为 1..count
			if seq == 0 {
				seq = 1
			}
			if seq >= 1 && seq <= count {
				results[seq] = lat
			}
		}
	}

	r := &PingResult{
		Target:      target,
		PacketsSent: count,
		Results:     make([]PingPacket, 0, count),
	}

	for i := 1; i <= count; i++ {
		if lat, ok := results[i]; ok {
			r.PacketsReceived++
			r.Results = append(r.Results, PingPacket{Sequence: i, Latency: lat, Status: "success"})
		} else {
			r.Results = append(r.Results, PingPacket{Sequence: i, Latency: 0, Status: "failed"})
		}
	}
	if r.PacketsSent > 0 {
		r.PacketLoss = float64(r.PacketsSent-r.PacketsReceived) / float64(r.PacketsSent) * 100
	}

	// 统计
	min := 0.0
	max := 0.0
	sum := 0.0
	okCnt := 0
	for _, p := range r.Results {
		if p.Status != "success" {
			continue
		}
		if okCnt == 0 || p.Latency < min {
			min = p.Latency
		}
		if okCnt == 0 || p.Latency > max {
			max = p.Latency
		}
		sum += p.Latency
		okCnt++
	}
	if okCnt > 0 {
		r.MinLatency = min
		r.MaxLatency = max
		r.AvgLatency = sum / float64(okCnt)
	}

	// sanity：如果完全解析不到（不同语言输出/格式），直接报错（不回退）
	if okCnt == 0 {
		return nil, fmt.Errorf("系统 ping 执行或解析失败: %s", strings.TrimSpace(string(out)))
	}
	return r, nil
}
