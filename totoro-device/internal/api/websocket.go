package api

import (
	"net/http"
	"totoro-device/models"
	"totoro-device/internal/realtime"
	"totoro-device/config"
	"totoro-device/internal/database"
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

	// 注意：前端优先使用 hello.frp_status 作为初始展示，因此这里也要遵循“非手动不暴露 IP/端口”的规则
	frpStatusUI := any(frpStatus)
	if frpStatus != nil {
		mode := s.config.FRPServer.Mode
		display := ""
		source := ""
		switch mode {
		case config.FRPModeManual:
			display = strings.TrimSpace(frpStatus.Server)
			source = "manual"
		case config.FRPModeBuiltin:
			display = "Totoro云节点"
			source = "builtin"
		case config.FRPModePublic:
			db := database.GetDB()
			code := ""
			nodeID := ""
			if db != nil {
				code, _ = database.GetPublicInviteCode(db)
				nodeID, _ = database.GetPublicNodeID(db)
			}
			if strings.TrimSpace(code) != "" {
				display = "私有分享云节点"
				source = "invite"
			} else if strings.TrimSpace(nodeID) != "" {
				display = "公开云节点"
				source = "public"
			} else {
				display = "公开云节点"
				source = "public"
			}
		default:
			display = "Totoro云节点"
			source = "unknown"
		}
		serverOut := frpStatus.Server
		if mode != config.FRPModeManual {
			serverOut = ""
		}
		frpStatusUI = gin.H{
			"connected":      frpStatus.Connected,
			"server":         serverOut,
			"connected_at":   frpStatus.ConnectedAt,
			"pid":            frpStatus.PID,
			"last_error":     frpStatus.LastError,
			"tunnels":        frpStatus.Tunnels,
			"log_path":       frpStatus.LogPath,
			"display_server": display,
			"mode":           string(mode),
			"source":         source,
		}
	}
	hub.Hello(cl, gin.H{
		"message":     "WebSocket连接成功",
		"device_id":   s.config.Device.ID,
		"network":     netStatus,
		"scan_status": scanStatus,
		"frp_status":  frpStatusUI,
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

