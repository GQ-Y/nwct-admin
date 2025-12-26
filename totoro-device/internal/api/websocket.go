package api

import (
	"net/http"
	"totoro-device/models"
	"totoro-device/internal/realtime"
	"totoro-device/utils"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源
	},
}

// handleWebSocket 处理WebSocket连接
func (s *Server) handleWebSocket(c *gin.Context) {
	// 鉴权：支持 token query 或 Authorization Bearer
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		authHeader := c.GetHeader("Authorization")
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			token = parts[1]
		}
	}
	if token == "" {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse(401, "未授权"))
		return
	}
	if _, err := utils.VerifyJWT(token); err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse(401, "Token无效"))
		return
	}

	// 升级HTTP连接为WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "WebSocket升级失败"))
		return
	}
	// 连接参数
	conn.SetReadLimit(64 * 1024)
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	hub := realtime.Default()
	cl := hub.Register(conn)
	defer func() {
		hub.Unregister(cl)
		_ = conn.Close()
	}()

	// hello（一次性快照）
	netStatus, _ := s.netManager.GetNetworkStatus()
	scanStatus := s.scanner.GetScanStatus()
	frpStatus, _ := s.frpClient.GetStatus()
	hub.Hello(cl, gin.H{
		"message":     "WebSocket连接成功",
		"device_id":   s.config.Device.ID,
		"network":     netStatus,
		"scan_status": scanStatus,
		"frp_status":  frpStatus,
	})

	// 写循环
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		defer close(done)
		for {
			select {
			case msg, ok := <-cl.Send:
				_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if !ok {
					_ = conn.WriteMessage(websocket.CloseMessage, []byte{})
					return
				}
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			case <-ticker.C:
				_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	// 读循环（当前只消费 ping/控制帧；文本消息忽略）
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			<-done
			return
		}
	}
}

