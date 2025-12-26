package toolkit

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Hop Traceroute跳转信息
type Hop struct {
	Hop      int     `json:"hop"`
	IP       string  `json:"ip"`
	Hostname string  `json:"hostname"`
	Latency  float64 `json:"latency"`
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
	if timeout <= 0 {
		timeout = 3 * time.Second
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

	// 仅使用系统 traceroute/tracert（结果真实）；失败直接返回错误
	hops, err := tracerouteWithSystemCommand(target, maxHops, timeout)
	if err != nil {
		return nil, err
	}
	result.Hops = hops
	return result, nil
}

func tracerouteWithSystemCommand(target string, maxHops int, timeout time.Duration) ([]Hop, error) {
	switch runtime.GOOS {
	case "darwin", "linux":
		path, err := exec.LookPath("traceroute")
		if err != nil {
			return nil, fmt.Errorf("traceroute 未安装或不可用")
		}
		sec := int(timeout.Seconds())
		if sec <= 0 {
			sec = 1
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(maxHops)*timeout+3*time.Second)
		defer cancel()
		// -n 数字形式；-m max hops；-w wait seconds
		cmd := exec.CommandContext(ctx, path, "-n", "-m", strconv.Itoa(maxHops), "-w", strconv.Itoa(sec), target)
		out, err := cmd.CombinedOutput()
		hops := parseTracerouteOutput(out)
		if len(hops) == 0 {
			msg := strings.TrimSpace(string(out))
			if msg == "" && err != nil {
				msg = err.Error()
			}
			return nil, fmt.Errorf("traceroute 执行失败: %s", msg)
		}
		return hops, nil
	case "windows":
		path, err := exec.LookPath("tracert")
		if err != nil {
			return nil, fmt.Errorf("tracert 未安装或不可用")
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(maxHops)*timeout+5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, path, "-d", "-h", strconv.Itoa(maxHops), target)
		out, err := cmd.CombinedOutput()
		hops := parseTracertOutput(out)
		if len(hops) == 0 {
			msg := strings.TrimSpace(string(out))
			if msg == "" && err != nil {
				msg = err.Error()
			}
			return nil, fmt.Errorf("tracert 执行失败: %s", msg)
		}
		return hops, nil
	default:
		return nil, fmt.Errorf("当前系统不支持 traceroute")
	}
}

func parseTracerouteOutput(out []byte) []Hop {
	// 典型输出行：
	//  1  192.168.1.1  1.123 ms  1.045 ms  1.031 ms
	//  2  * * *
	sc := bufio.NewScanner(bytes.NewReader(out))
	reLine := regexp.MustCompile(`^\s*(\d+)\s+(.+)$`)
	reIP := regexp.MustCompile(`\b(\d{1,3}(?:\.\d{1,3}){3})\b`)
	reMS := regexp.MustCompile(`([\d.]+)\s*ms`)
	hops := []Hop{}
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		m := reLine.FindStringSubmatch(line)
		if len(m) != 3 {
			continue
		}
		hopNum, _ := strconv.Atoi(m[1])
		rest := m[2]
		h := Hop{Hop: hopNum, IP: "*"}
		if strings.Contains(rest, "*") && !reIP.MatchString(rest) {
			hops = append(hops, h)
			continue
		}
		if ipm := reIP.FindStringSubmatch(rest); len(ipm) == 2 {
			h.IP = ipm[1]
		}
		if msm := reMS.FindStringSubmatch(rest); len(msm) == 2 {
			if v, err := strconv.ParseFloat(msm[1], 64); err == nil {
				h.Latency = v
			}
		}
		hops = append(hops, h)
	}
	return hops
}

func parseTracertOutput(out []byte) []Hop {
	// 简化解析 windows tracert
	//  1    <1 ms    <1 ms    <1 ms  192.168.1.1
	//  2     *        *        *     Request timed out.
	sc := bufio.NewScanner(bytes.NewReader(out))
	reLine := regexp.MustCompile(`^\s*(\d+)\s+(.+)$`)
	reIP := regexp.MustCompile(`\b(\d{1,3}(?:\.\d{1,3}){3})\b`)
	reMS := regexp.MustCompile(`(\d+)\s*ms`)
	hops := []Hop{}
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		m := reLine.FindStringSubmatch(line)
		if len(m) != 3 {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		rest := m[2]
		h := Hop{Hop: n, IP: "*"}
		if ipm := reIP.FindStringSubmatch(rest); len(ipm) == 2 {
			h.IP = ipm[1]
		}
		if msm := reMS.FindStringSubmatch(rest); len(msm) == 2 {
			if v, err := strconv.ParseFloat(msm[1], 64); err == nil {
				h.Latency = v
			}
		}
		hops = append(hops, h)
	}
	return hops
}
