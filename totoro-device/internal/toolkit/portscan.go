package toolkit

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// PortInfo 端口信息
type PortInfo struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Service  string `json:"service"`
	Version  string `json:"version"`
	Status   string `json:"status"`
}

// PortScanResult 端口扫描结果
type PortScanResult struct {
	Target      string     `json:"target"`
	ScannedPorts int      `json:"scanned_ports"`
	OpenPorts   []PortInfo `json:"open_ports"`
	ClosedPorts []int      `json:"closed_ports"`
	ScanTime    time.Time `json:"scan_time"`
}

// PortScan 执行端口扫描
func PortScan(target string, ports interface{}, timeout time.Duration, scanType string) (*PortScanResult, error) {
	result := &PortScanResult{
		Target:   target,
		ScanTime: time.Now(),
	}

	// 解析端口列表
	portList, err := parsePorts(ports)
	if err != nil {
		return nil, err
	}

	result.ScannedPorts = len(portList)
	result.OpenPorts = []PortInfo{}
	result.ClosedPorts = []int{}

	// 扫描端口
	for _, port := range portList {
		info, err := scanPort(target, port, timeout, scanType)
		if err != nil {
			result.ClosedPorts = append(result.ClosedPorts, port)
			continue
		}

		if info.Status == "open" {
			result.OpenPorts = append(result.OpenPorts, *info)
		} else {
			result.ClosedPorts = append(result.ClosedPorts, port)
		}
	}

	return result, nil
}

// parsePorts 解析端口参数
func parsePorts(ports interface{}) ([]int, error) {
	var portList []int

	switch v := ports.(type) {
	case []int:
		return v, nil
	case []interface{}:
		for _, p := range v {
			if port, ok := p.(int); ok {
				portList = append(portList, port)
			} else if port, ok := p.(float64); ok {
				portList = append(portList, int(port))
			}
		}
		return portList, nil
	case string:
		// 支持 "80,443,8080" 或 "1-1000" 格式
		if strings.Contains(v, "-") {
			// 范围格式
			parts := strings.Split(v, "-")
			if len(parts) != 2 {
				return nil, fmt.Errorf("无效的端口范围格式")
			}
			start, err := strconv.Atoi(parts[0])
			if err != nil {
				return nil, err
			}
			end, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, err
			}
			for i := start; i <= end; i++ {
				portList = append(portList, i)
			}
		} else {
			// 逗号分隔
			parts := strings.Split(v, ",")
			for _, p := range parts {
				port, err := strconv.Atoi(strings.TrimSpace(p))
				if err != nil {
					continue
				}
				portList = append(portList, port)
			}
		}
		return portList, nil
	default:
		return nil, fmt.Errorf("不支持的端口格式")
	}
}

// scanPort 扫描单个端口
func scanPort(target string, port int, timeout time.Duration, scanType string) (*PortInfo, error) {
	info := &PortInfo{
		Port:     port,
		Protocol: "tcp",
		Status:   "closed",
	}

	if scanType == "" || scanType == "tcp" || scanType == "both" {
		address := net.JoinHostPort(target, strconv.Itoa(port))
		conn, err := net.DialTimeout("tcp", address, timeout)
		if err == nil {
			info.Status = "open"
			info.Service = identifyService(port)
			conn.Close()
			return info, nil
		}
	}

	if scanType == "udp" || scanType == "both" {
		// UDP扫描（简化实现）
		address := net.JoinHostPort(target, strconv.Itoa(port))
		conn, err := net.DialTimeout("udp", address, timeout)
		if err == nil {
			info.Protocol = "udp"
			info.Status = "open"
			info.Service = identifyService(port)
			conn.Close()
			return info, nil
		}
	}

	return info, fmt.Errorf("端口关闭")
}

// identifyService 识别服务
func identifyService(port int) string {
	services := map[int]string{
		21:   "ftp",
		22:   "ssh",
		23:   "telnet",
		25:   "smtp",
		53:   "dns",
		80:   "http",
		110:  "pop3",
		143:  "imap",
		443:  "https",
		3306: "mysql",
		3389: "rdp",
		5432: "postgresql",
		6379: "redis",
		8080: "http-proxy",
	}

	if service, ok := services[port]; ok {
		return service
	}
	return "unknown"
}

