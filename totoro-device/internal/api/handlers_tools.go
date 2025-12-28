//go:build !device_minimal

package api

import (
	"net/http"
	"strings"
	"time"

	"totoro-device/internal/toolkit"
	"totoro-device/models"

	"github.com/gin-gonic/gin"
)

// handlePing 处理Ping测试请求
func (s *Server) handlePing(c *gin.Context) {
	var req struct {
		Target  string `json:"target" binding:"required"`
		Count   int    `json:"count"`
		Timeout int    `json:"timeout"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "参数错误: "+err.Error()))
		return
	}

	if req.Count <= 0 {
		req.Count = 4
	}
	if req.Timeout <= 0 {
		req.Timeout = 5
	}

	// 使用toolkit的Ping实现
	result, err := toolkit.Ping(req.Target, req.Count, time.Duration(req.Timeout)*time.Second)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(result))
}

// handleTraceroute 处理Traceroute请求
func (s *Server) handleTraceroute(c *gin.Context) {
	var req struct {
		Target  string `json:"target" binding:"required"`
		MaxHops int    `json:"max_hops"`
		Timeout int    `json:"timeout"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "参数错误: "+err.Error()))
		return
	}

	if req.MaxHops <= 0 {
		req.MaxHops = 30
	}
	if req.Timeout <= 0 {
		req.Timeout = 5
	}

	target := strings.TrimSpace(req.Target)
	if target == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "target 不能为空"))
		return
	}

	result, err := toolkit.Traceroute(target, req.MaxHops, time.Duration(req.Timeout)*time.Second)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(result))
}

// handleSpeedTest 处理网速测试请求
func (s *Server) handleSpeedTest(c *gin.Context) {
	var req struct {
		// mode:
		// - web: 访问网站测速（DNS/TCP/TLS/TTFB/Total），默认
		// - download: 下载带宽测速（旧逻辑）
		Mode          string `json:"mode"`
		URL           string `json:"url"`
		Method        string `json:"method"` // GET(默认)/HEAD
		Count         int    `json:"count"`
		Timeout       int    `json:"timeout"` // 秒
		DownloadBytes int64  `json:"download_bytes"`

		// 旧字段兼容（download 模式使用）
		Server   string `json:"server"`
		TestType string `json:"test_type"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许空请求体
		req.Mode = "web"
	}

	mode := strings.TrimSpace(req.Mode)
	if mode == "" {
		mode = "web"
	}

	switch mode {
	case "download":
		// 使用toolkit的下载测速（兼容旧面板/脚本）
		if req.TestType == "" {
			req.TestType = "download"
		}
		if req.Server == "" {
			req.Server = "default"
		}
		result, err := toolkit.SpeedTest(req.Server, req.TestType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
			return
		}
		c.JSON(http.StatusOK, models.SuccessResponse(result))
		return
	case "web":
		fallthrough
	default:
		if req.Count <= 0 {
			req.Count = 3
		}
		if req.Timeout <= 0 {
			req.Timeout = 8
		}
		// 兼容：如果 url 为空但 server 有值，把 server 当作 url
		targetURL := strings.TrimSpace(req.URL)
		if targetURL == "" {
			targetURL = strings.TrimSpace(req.Server)
		}
		method := strings.TrimSpace(req.Method)
		if method == "" {
			method = "GET"
		}
		result, err := toolkit.WebSpeedTestWithOptions(targetURL, method, req.Count, time.Duration(req.Timeout)*time.Second, req.DownloadBytes)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse(500, err.Error()))
			return
		}
		c.JSON(http.StatusOK, models.SuccessResponse(result))
		return
	}
}
