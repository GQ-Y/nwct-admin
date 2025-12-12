package mqtt

import (
	"encoding/json"
	"fmt"
	"net"
	"nwct/client-nps/config"
	"nwct/client-nps/internal/logger"
	"nwct/client-nps/internal/network"
	"nwct/client-nps/internal/realtime"
	"nwct/client-nps/internal/scanner"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

var globalMQTTClient Client
var globalScanner scanner.Scanner
var globalConfig *config.Config
var globalNetManager network.Manager

// SetGlobalClient 设置全局MQTT客户端（用于命令处理）
func SetGlobalClient(client Client) {
	globalMQTTClient = client
}

// SetGlobalScanner 设置全局扫描器（用于 MQTT scan 命令）
func SetGlobalScanner(s scanner.Scanner) {
	globalScanner = s
}

// SetGlobalConfig 设置全局配置（用于 MQTT config_update 命令）
func SetGlobalConfig(c *config.Config) {
	globalConfig = c
}

// SetGlobalNetManager 设置全局网络管理器（用于推断扫描网段）
func SetGlobalNetManager(nm network.Manager) {
	globalNetManager = nm
}

type mqttCommand struct {
	Action    string                 `json:"action"`
	Params    map[string]interface{} `json:"params"`
	RequestID string                 `json:"request_id,omitempty"`
}

func deviceIDForResponse() string {
	if globalConfig != nil && strings.TrimSpace(globalConfig.Device.ID) != "" {
		return strings.TrimSpace(globalConfig.Device.ID)
	}
	// fallback（兼容旧逻辑）
	return "device_001"
}

func publishResponse(action, status, message string, data map[string]interface{}, requestID string) {
	if globalMQTTClient == nil {
		return
	}
	deviceID := deviceIDForResponse()
	responseTopic := fmt.Sprintf("nwct/%s/response", deviceID)
	resp := map[string]interface{}{
		"action":     action,
		"status":     status,
		"message":    message,
		"request_id": requestID,
		"ts":         time.Now().Format(time.RFC3339),
	}
	if data != nil {
		resp["data"] = data
	}
	_ = globalMQTTClient.Publish(responseTopic, resp)
	// 同步推送到 WebSocket（控制面板实时日志）
	realtime.Default().Broadcast("mqtt_response", resp)
}

func publishEvent(event string, data map[string]interface{}, requestID string) {
	if globalMQTTClient == nil {
		return
	}
	deviceID := deviceIDForResponse()
	topic := fmt.Sprintf("nwct/%s/event", deviceID)
	msg := map[string]interface{}{
		"event":      event,
		"request_id": requestID,
		"ts":         time.Now().Format(time.RFC3339),
		"data":       data,
	}
	_ = globalMQTTClient.Publish(topic, msg)
	realtime.Default().Broadcast("mqtt_event", msg)
}

func guessSubnetFromIP(ipStr string) (string, error) {
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return "", fmt.Errorf("无法解析IP: %s", ipStr)
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return "", fmt.Errorf("仅支持IPv4推断网段: %s", ipStr)
	}
	// 默认 /24（后续可以根据接口 netmask 精确推断）
	return fmt.Sprintf("%d.%d.%d.0/24", ip4[0], ip4[1], ip4[2]), nil
}

// HandleCommandMessage 处理MQTT命令消息
func HandleCommandMessage(topic string, payload []byte) {
	var cmd mqttCommand
	if err := json.Unmarshal(payload, &cmd); err != nil {
		logger.Error("解析MQTT命令失败: %v", err)
		return
	}

	logger.Info("收到MQTT命令: action=%s", cmd.Action)

	switch cmd.Action {
	case "restart":
		handleRestartCommand(cmd.Params, cmd.RequestID)
	case "scan":
		handleScanCommand(cmd.Params, cmd.RequestID)
	case "config_update":
		handleConfigUpdateCommand(cmd.Params, cmd.RequestID)
	default:
		logger.Warn("未知的MQTT命令: %s", cmd.Action)
		publishResponse(cmd.Action, "error", "未知命令", nil, cmd.RequestID)
		return
	}
}

// handleRestartCommand 处理重启命令
func handleRestartCommand(params map[string]interface{}, requestID string) {
	typ := "soft"
	if params != nil {
		if v, ok := params["type"].(string); ok && strings.TrimSpace(v) != "" {
			typ = strings.TrimSpace(v)
		}
	}
	logger.Info("执行重启命令: type=%s", typ)

	publishResponse("restart", "success", "重启命令已接受", map[string]interface{}{"type": typ}, requestID)

	// 延迟执行重启，给MQTT消息发送时间
	go func() {
		time.Sleep(800 * time.Millisecond)

		if typ == "soft" {
			// soft：退出进程，由外部守护进程拉起（或容器策略重启）
			os.Exit(0)
			return
		}

		// hard：系统级重启（可能需要root）
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			cmd = exec.Command("shutdown", "/r", "/t", "0")
		case "linux":
			cmd = exec.Command("shutdown", "-r", "now")
		case "darwin":
			cmd = exec.Command("shutdown", "-r", "now")
		default:
			logger.Error("不支持的重启平台: %s", runtime.GOOS)
			return
		}

		if err := cmd.Run(); err != nil {
			logger.Error("执行重启命令失败: %v", err)
			publishEvent("restart_failed", map[string]interface{}{"type": typ, "error": err.Error()}, requestID)
			return
		}
	}()
}

func handleScanCommand(params map[string]interface{}, requestID string) {
	if globalScanner == nil {
		publishResponse("scan", "error", "scanner 未初始化", nil, requestID)
		return
	}

	subnet := ""
	if params != nil {
		if v, ok := params["subnet"].(string); ok {
			subnet = strings.TrimSpace(v)
		}
	}
	if subnet == "" {
		if globalNetManager == nil {
			publishResponse("scan", "error", "未指定 subnet 且 netManager 未初始化", nil, requestID)
			return
		}
		if st, err := globalNetManager.GetNetworkStatus(); err == nil {
			if st.IP != "" {
				if s, err := guessSubnetFromIP(st.IP); err == nil {
					subnet = s
				}
			}
		}
	}
	if subnet == "" {
		publishResponse("scan", "error", "无法确定扫描网段，请在 params.subnet 指定 CIDR（如 192.168.1.0/24）", nil, requestID)
		return
	}

	if err := globalScanner.StartScan(subnet); err != nil {
		publishResponse("scan", "error", err.Error(), map[string]interface{}{"subnet": subnet}, requestID)
		return
	}

	publishResponse("scan", "success", "扫描已启动", map[string]interface{}{"subnet": subnet}, requestID)

	// 后台推送进度/完成事件
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			st := globalScanner.GetScanStatus()
			if st == nil {
				return
			}
			publishEvent("scan_progress", map[string]interface{}{
				"status":        st.Status,
				"progress":      st.Progress,
				"scanned_count": st.ScannedCount,
				"found_count":   st.FoundCount,
				"start_time":    st.StartTime.Format(time.RFC3339),
			}, requestID)
			if st.Status == "completed" || st.Status == "stopped" {
				publishEvent("scan_done", map[string]interface{}{
					"status":        st.Status,
					"progress":      st.Progress,
					"scanned_count": st.ScannedCount,
					"found_count":   st.FoundCount,
				}, requestID)
				return
			}
			<-ticker.C
		}
	}()
}

func mergeMap(dst map[string]interface{}, patch map[string]interface{}) {
	for k, v := range patch {
		if vmap, ok := v.(map[string]interface{}); ok {
			if cur, ok := dst[k].(map[string]interface{}); ok {
				mergeMap(cur, vmap)
				dst[k] = cur
			} else {
				dst[k] = vmap
			}
			continue
		}
		dst[k] = v
	}
}

func handleConfigUpdateCommand(params map[string]interface{}, requestID string) {
	if globalConfig == nil {
		publishResponse("config_update", "error", "config 未初始化", nil, requestID)
		return
	}
	if params == nil {
		publishResponse("config_update", "error", "params 不能为空", nil, requestID)
		return
	}

	// 支持两种格式：
	// 1) params.config = 完整/部分 config 对象
	// 2) params = 直接 patch（顶层字段）
	patch := params
	if cobj, ok := params["config"].(map[string]interface{}); ok {
		patch = cobj
	}

	// 做一次 map merge：globalConfig -> map -> merge patch -> unmarshal
	baseBytes, _ := json.Marshal(globalConfig)
	var base map[string]interface{}
	_ = json.Unmarshal(baseBytes, &base)
	mergeMap(base, patch)

	// 安全：禁止通过 MQTT 覆盖 password_hash
	if auth, ok := base["auth"].(map[string]interface{}); ok {
		delete(auth, "password_hash")
		base["auth"] = auth
	}

	mergedBytes, _ := json.Marshal(base)
	var next config.Config
	if err := json.Unmarshal(mergedBytes, &next); err != nil {
		publishResponse("config_update", "error", "配置解析失败: "+err.Error(), nil, requestID)
		return
	}
	// 继续保留现有 password_hash
	next.Auth.PasswordHash = globalConfig.Auth.PasswordHash

	if err := next.Validate(); err != nil {
		publishResponse("config_update", "error", "配置校验失败: "+err.Error(), nil, requestID)
		return
	}

	*globalConfig = next
	if err := globalConfig.Save(); err != nil {
		publishResponse("config_update", "error", "保存失败: "+err.Error(), nil, requestID)
		return
	}
	publishResponse("config_update", "success", "配置已更新并保存", nil, requestID)
}

