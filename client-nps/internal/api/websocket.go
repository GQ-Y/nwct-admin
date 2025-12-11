package api

import (
	"net/http"
	"nwct/client-nps/models"

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
	// 升级HTTP连接为WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse(400, "WebSocket升级失败"))
		return
	}
	defer conn.Close()

	// 发送欢迎消息
	conn.WriteJSON(gin.H{
		"type": "connected",
		"data": gin.H{
			"message": "WebSocket连接成功",
		},
	})

	// 处理消息
	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}

		// 处理不同类型的消息
		msgType, ok := msg["type"].(string)
		if !ok {
			continue
		}

		switch msgType {
		case "ping":
			conn.WriteJSON(gin.H{
				"type": "pong",
			})
		default:
			// 回显消息
			conn.WriteJSON(gin.H{
				"type": "echo",
				"data": msg,
			})
		}
	}
}

