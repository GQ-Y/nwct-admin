package mqtt

import (
	"encoding/json"
	"fmt"
	"nwct/client-nps/internal/logger"
	"os/exec"
	"runtime"
	"time"
)

var globalMQTTClient Client

// SetGlobalClient 设置全局MQTT客户端（用于命令处理）
func SetGlobalClient(client Client) {
	globalMQTTClient = client
}

// HandleCommandMessage 处理MQTT命令消息
func HandleCommandMessage(topic string, payload []byte) {
	var cmd struct {
		Action string                 `json:"action"`
		Params map[string]interface{} `json:"params"`
	}

	if err := json.Unmarshal(payload, &cmd); err != nil {
		logger.Error("解析MQTT命令失败: %v", err)
		return
	}

	logger.Info("收到MQTT命令: action=%s", cmd.Action)

	switch cmd.Action {
	case "restart":
		handleRestartCommand()
	case "scan":
		// TODO: 触发设备扫描
		logger.Info("触发设备扫描")
	case "config_update":
		// TODO: 更新配置
		logger.Info("更新配置")
	default:
		logger.Warn("未知的MQTT命令: %s", cmd.Action)
	}

	// 发送响应
	if globalMQTTClient != nil {
		responseTopic := fmt.Sprintf("nwct/device_001/response")
		response := map[string]interface{}{
			"action":  cmd.Action,
			"status":  "success",
			"message": "命令已执行",
		}
		globalMQTTClient.Publish(responseTopic, response)
	}
}

// handleRestartCommand 处理重启命令
func handleRestartCommand() {
	logger.Info("执行软重启命令")

	// 延迟执行重启，给MQTT消息发送时间
	go func() {
		time.Sleep(1 * time.Second)

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("shutdown", "/r", "/t", "0")
		} else {
			cmd = exec.Command("reboot")
		}

		if err := cmd.Run(); err != nil {
			logger.Error("执行重启命令失败: %v", err)
		}
	}()
}

