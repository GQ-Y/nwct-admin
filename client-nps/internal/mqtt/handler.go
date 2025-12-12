package mqtt

import (
	"encoding/json"
	"nwct/client-nps/internal/logger"
)

// HandleCommand 处理MQTT命令
func (c *mqttClient) HandleCommand(topic string, payload []byte) {
	// 兼容旧入口：转发到统一实现
	var _cmd map[string]interface{}
	if err := json.Unmarshal(payload, &_cmd); err != nil {
		logger.Error("解析MQTT命令失败: %v", err)
		return
	}
	HandleCommandMessage(topic, payload)
}

