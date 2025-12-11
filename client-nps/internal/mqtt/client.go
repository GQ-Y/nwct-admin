package mqtt

import (
	"encoding/json"
	"fmt"
	"nwct/client-nps/config"
	"nwct/client-nps/internal/database"
	"nwct/client-nps/internal/logger"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Client MQTT客户端接口
type Client interface {
	Connect() error
	Disconnect() error
	IsConnected() bool
	Publish(topic string, message interface{}) error
	Subscribe(topic string, handler MessageHandler) error
	GetStatus() (*MQTTStatus, error)
}

// MessageHandler 消息处理器
type MessageHandler func(topic string, payload []byte)

// MQTTStatus MQTT状态
type MQTTStatus struct {
	Connected        bool     `json:"connected"`
	Server          string    `json:"server"`
	ClientID        string    `json:"client_id"`
	ConnectedAt     string   `json:"connected_at"`
	SubscribedTopics []string `json:"subscribed_topics"`
	PublishedTopics  []string `json:"published_topics"`
}

// mqttClient MQTT客户端实现
type mqttClient struct {
	config            *config.MQTTConfig
	client            mqtt.Client
	connected         bool
	connectedAt       time.Time
	subscribedTopics map[string]MessageHandler
	publishedTopics   map[string]bool
	mu                sync.RWMutex
}

// NewClient 创建MQTT客户端
func NewClient(cfg *config.MQTTConfig) Client {
	return &mqttClient{
		config:            cfg,
		connected:         false,
		subscribedTopics: make(map[string]MessageHandler),
		publishedTopics:   make(map[string]bool),
	}
}

// Connect 连接到MQTT服务器
func (c *mqttClient) Connect() error {
	if c.config.Server == "" {
		return fmt.Errorf("MQTT服务器地址未配置")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", c.config.Server, c.config.Port))
	opts.SetClientID(c.config.ClientID)
	
	if c.config.Username != "" {
		opts.SetUsername(c.config.Username)
		opts.SetPassword(c.config.Password)
	}

	// 设置连接选项
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)

	// 设置Last Will
	deviceID := c.config.ClientID
	opts.SetWill(fmt.Sprintf("nwct/%s/status", deviceID), `{"status":"offline"}`, 1, true)

	// 连接回调
	opts.OnConnect = func(client mqtt.Client) {
		logger.Info("MQTT连接成功")
		c.connected = true
		c.connectedAt = time.Now()

		// 发布上线消息
		c.publishStatus("online")

		// 重新订阅之前的主题
		for topic, handler := range c.subscribedTopics {
			c.subscribeInternal(topic, handler)
		}
	}

	opts.OnConnectionLost = func(client mqtt.Client, err error) {
		logger.Error("MQTT连接丢失: %v", err)
		c.connected = false
	}

	// 创建客户端
	client := mqtt.NewClient(opts)

	// 连接
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("MQTT连接失败: %v", token.Error())
	}

	c.client = client
	return nil
}

// Disconnect 断开MQTT连接
func (c *mqttClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.client == nil {
		return nil
	}

	// 发布离线消息
	c.publishStatus("offline")

	c.client.Disconnect(250)
	c.connected = false
	logger.Info("MQTT连接已断开")
	return nil
}

// IsConnected 检查是否已连接
func (c *mqttClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected && c.client != nil && c.client.IsConnected()
}

// Publish 发布消息
func (c *mqttClient) Publish(topic string, message interface{}) error {
	if !c.IsConnected() {
		return fmt.Errorf("MQTT未连接")
	}

	var payload []byte
	var err error

	switch v := message.(type) {
	case string:
		payload = []byte(v)
	case []byte:
		payload = v
	default:
		payload, err = json.Marshal(message)
		if err != nil {
			return fmt.Errorf("序列化消息失败: %v", err)
		}
	}

	token := c.client.Publish(topic, 1, false, payload)
	if !token.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("发布消息超时")
	}
	if token.Error() != nil {
		return fmt.Errorf("发布消息失败: %v", token.Error())
	}

	// 记录日志（异步）
	go c.logMessage("publish", topic, payload, 1, "success")

	// 记录发布的主题
	c.mu.Lock()
	c.publishedTopics[topic] = true
	c.mu.Unlock()

	return nil
}

// Subscribe 订阅主题
func (c *mqttClient) Subscribe(topic string, handler MessageHandler) error {
	if !c.IsConnected() {
		return fmt.Errorf("MQTT未连接")
	}

	c.mu.Lock()
	c.subscribedTopics[topic] = handler
	c.mu.Unlock()

	return c.subscribeInternal(topic, handler)
}

// subscribeInternal 内部订阅方法
func (c *mqttClient) subscribeInternal(topic string, handler MessageHandler) error {
	token := c.client.Subscribe(topic, 1, func(client mqtt.Client, msg mqtt.Message) {
		// 记录日志（异步）
		go c.logMessage("subscribe", topic, msg.Payload(), int(msg.Qos()), "success")

		// 调用处理器
		handler(topic, msg.Payload())
	})

	if !token.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("订阅主题超时")
	}
	if token.Error() != nil {
		return fmt.Errorf("订阅主题失败: %v", token.Error())
	}

	logger.Info("订阅MQTT主题: %s", topic)
	return nil
}

// GetStatus 获取MQTT状态
func (c *mqttClient) GetStatus() (*MQTTStatus, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	subscribed := make([]string, 0, len(c.subscribedTopics))
	for topic := range c.subscribedTopics {
		subscribed = append(subscribed, topic)
	}

	published := make([]string, 0, len(c.publishedTopics))
	for topic := range c.publishedTopics {
		published = append(published, topic)
	}

	connectedAt := ""
	if !c.connectedAt.IsZero() {
		connectedAt = c.connectedAt.Format(time.RFC3339)
	}

	return &MQTTStatus{
		Connected:        c.IsConnected(),
		Server:          fmt.Sprintf("%s:%d", c.config.Server, c.config.Port),
		ClientID:        c.config.ClientID,
		ConnectedAt:     connectedAt,
		SubscribedTopics: subscribed,
		PublishedTopics:  published,
	}, nil
}

// publishStatus 发布状态消息
func (c *mqttClient) publishStatus(status string) {
	deviceID := c.config.ClientID
	topic := fmt.Sprintf("nwct/%s/status", deviceID)
	message := map[string]interface{}{
		"status":    status,
		"timestamp": time.Now().Format(time.RFC3339),
		"device_id": deviceID,
	}
	c.Publish(topic, message)
}

// logMessage 记录MQTT消息日志（在goroutine中调用）
func (c *mqttClient) logMessage(direction, topic string, payload []byte, qos int, status string) {
	log := database.MQTTLog{
		Timestamp: time.Now(),
		Direction: direction,
		Topic:     topic,
		QoS:       qos,
		Payload:   string(payload),
		Status:    status,
	}

	db := database.GetDB()
	if db == nil {
		return
	}

	_, err := db.Exec(
		"INSERT INTO mqtt_logs (timestamp, direction, topic, qos, payload, status) VALUES (?, ?, ?, ?, ?, ?)",
		log.Timestamp, log.Direction, log.Topic, log.QoS, log.Payload, log.Status,
	)
	if err != nil {
		logger.Error("保存MQTT日志失败: %v", err)
	}
}
