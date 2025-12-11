package mqtt

import (
	"encoding/json"
	"fmt"
	"nwct/client-nps/internal/logger"
	"os/exec"
	"runtime"
	"time"
)

// HandleCommand 处理MQTT命令
func (c *mqttClient) HandleCommand(topic string, payload []byte) {
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
		c.handleRestart()
	case "scan":
		// TODO: 触发设备扫描
		logger.Info("触发设备扫描")
	case "config_update":
		// TODO: 更新配置
		logger.Info("更新配置")
	default:
		logger.Warn("未知的MQTT命令: %s", cmd.Action)
	}
}

// handleRestart 处理重启命令
func (c *mqttClient) handleRestart() {
	logger.Info("执行软重启命令")

	// 发布重启响应
	deviceID := c.config.ClientID
	responseTopic := fmt.Sprintf("nwct/%s/response", deviceID)
	response := map[string]interface{}{
		"action":  "restart",
		"status":  "success",
		"message": "重启命令已执行",
	}
	c.Publish(responseTopic, response)

	// 延迟执行重启，给MQTT消息发送时间
	go func() {
		// 等待1秒后重启
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

