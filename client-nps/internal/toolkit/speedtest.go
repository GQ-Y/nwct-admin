package toolkit

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// SpeedResult 网速测试结果
type SpeedResult struct {
	Server       string  `json:"server"`
	UploadSpeed  float64 `json:"upload_speed"`  // Mbps
	DownloadSpeed float64 `json:"download_speed"` // Mbps
	Latency      int     `json:"latency"`      // ms
	TestTime     string  `json:"test_time"`
	Duration     int     `json:"duration"`     // 秒
}

// SpeedTest 执行网速测试
func SpeedTest(server string, testType string) (*SpeedResult, error) {
	if server == "" || server == "default" {
		server = "http://speedtest.tele2.net"
	}

	result := &SpeedResult{
		Server:   server,
		TestTime: time.Now().Format(time.RFC3339),
	}

	start := time.Now()

	// 测试延迟
	latency, err := testLatency(server)
	if err == nil {
		result.Latency = latency
	}

	// 测试下载速度
	if testType == "download" || testType == "all" {
		downloadSpeed, err := testDownloadSpeed(server)
		if err == nil {
			result.DownloadSpeed = downloadSpeed
		}
	}

	// 测试上传速度
	if testType == "upload" || testType == "all" {
		uploadSpeed, err := testUploadSpeed(server)
		if err == nil {
			result.UploadSpeed = uploadSpeed
		}
	}

	result.Duration = int(time.Since(start).Seconds())

	return result, nil
}

// testLatency 测试延迟
func testLatency(server string) (int, error) {
	start := time.Now()
	resp, err := http.Get(server + "/1MB.zip")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return int(time.Since(start).Milliseconds()), nil
}

// testDownloadSpeed 测试下载速度
func testDownloadSpeed(server string) (float64, error) {
	url := server + "/10MB.zip"
	
	start := time.Now()
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// 读取数据
	buffer := make([]byte, 1024*1024) // 1MB buffer
	totalBytes := int64(0)
	
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			totalBytes += int64(n)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
	}

	duration := time.Since(start).Seconds()
	if duration == 0 {
		return 0, fmt.Errorf("测试时间过短")
	}

	// 计算速度 (Mbps)
	speed := (float64(totalBytes) * 8) / (duration * 1000000)
	return speed, nil
}

// testUploadSpeed 测试上传速度
func testUploadSpeed(server string) (float64, error) {
	// 创建测试数据
	testData := make([]byte, 10*1024*1024) // 10MB

	start := time.Now()
	resp, err := http.Post(server+"/upload", "application/octet-stream", nil)
	if err != nil {
		// 如果上传端点不存在，使用简化方法
		// 实际应该使用支持上传的服务器
		return 0, fmt.Errorf("上传测试暂不支持")
	}
	defer resp.Body.Close()

	duration := time.Since(start).Seconds()
	_ = testData // 避免未使用变量警告

	if duration == 0 {
		return 0, fmt.Errorf("测试时间过短")
	}

	// 简化实现
	return 0, fmt.Errorf("上传速度测试需要支持上传的服务器")
}

