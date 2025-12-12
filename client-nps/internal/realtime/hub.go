package realtime

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Message 是推送到前端的统一消息结构
type Message struct {
	Type  string      `json:"type"`           // event / hello / error
	Event string      `json:"event,omitempty"` // system_status / scan_progress / ...
	Data  interface{} `json:"data,omitempty"`
	TS    string      `json:"ts"`
}

type Client struct {
	Conn *websocket.Conn
	Send chan []byte
}

type Hub struct {
	mu sync.RWMutex

	clients    map[*Client]struct{}
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
}

func NewHub() *Hub {
	h := &Hub{
		clients:    make(map[*Client]struct{}),
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
		broadcast:  make(chan []byte, 256),
	}
	go h.run()
	return h
}

func (h *Hub) run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = struct{}{}
			h.mu.Unlock()
		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.Send)
			}
			h.mu.Unlock()
		case msg := <-h.broadcast:
			h.mu.RLock()
			for c := range h.clients {
				select {
				case c.Send <- msg:
				default:
					// 客户端写入慢：踢掉
					h.mu.RUnlock()
					h.unregister <- c
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Register(conn *websocket.Conn) *Client {
	c := &Client{
		Conn: conn,
		Send: make(chan []byte, 64),
	}
	h.register <- c
	return c
}

func (h *Hub) Unregister(c *Client) {
	h.unregister <- c
}

func (h *Hub) Broadcast(event string, data interface{}) {
	b, _ := json.Marshal(Message{
		Type:  "event",
		Event: event,
		Data:  data,
		TS:    time.Now().Format(time.RFC3339),
	})
	select {
	case h.broadcast <- b:
	default:
		// broadcast 堵住了就丢弃，避免拖垮主流程
	}
}

func (h *Hub) Hello(c *Client, data interface{}) {
	b, _ := json.Marshal(Message{
		Type: "hello",
		Data: data,
		TS:   time.Now().Format(time.RFC3339),
	})
	select {
	case c.Send <- b:
	default:
	}
}

var defaultHub *Hub
var once sync.Once

func Default() *Hub {
	once.Do(func() {
		defaultHub = NewHub()
	})
	return defaultHub
}


