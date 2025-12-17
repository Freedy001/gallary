package websocket

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"gallary/server/pkg/logger"
)

// Hub WebSocket 连接中心
type Hub struct {
	// 已注册的客户端
	clients   map[*Client]bool
	clientsMu sync.RWMutex

	// 注册/注销通道
	register   chan *Client
	unregister chan *Client

	// 广播通道
	broadcast chan *Message

	// 运行状态
	ctx    context.Context
	cancel context.CancelFunc

	// 初始状态获取器（连接时推送当前状态）
	setup []func(*Client)
}

// NewHub 创建新的 Hub
func NewHub() *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	h := &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Message, 256),
		ctx:        ctx,
		cancel:     cancel,
	}
	go h.eventLoop()
	return h
}

// eventLoop 启动 Hub
func (h *Hub) eventLoop() {
	for {
		select {
		case <-h.ctx.Done():
			// 关闭所有客户端连接
			h.clientsMu.Lock()
			for client := range h.clients {
				close(client.send)
				delete(h.clients, client)
			}
			h.clientsMu.Unlock()
			return

		case client := <-h.register:
			h.clientsMu.Lock()
			h.clients[client] = true
			h.clientsMu.Unlock()

			logger.Info("WebSocket 客户端已连接",
				zap.String("client_id", client.clientID),
				zap.Int("total_clients", len(h.clients)))

			go client.WritePump()
			go client.ReadPump()
			// 连接成功后立即推送当前状态
			go func() {
				for i := range h.setup {
					h.setup[i](client)
				}
			}()

		case client := <-h.unregister:
			h.clientsMu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				logger.Info("WebSocket 客户端已断开",
					zap.String("client_id", client.clientID),
					zap.Int("total_clients", len(h.clients)))
			}
			h.clientsMu.Unlock()
		case message := <-h.broadcast:
			h.doBoardCast(message)
		}
	}
}

func (h *Hub) doBoardCast(message *Message) {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()

	// broadcastMessage 广播消息给所有客户端
	for client := range h.clients {
		select {
		case client.send <- message:
		default:
			// 客户端缓冲已满，断开连接
			close(client.send)
			delete(h.clients, client)
		}
	}
}

// Stop 停止 Hub
func (h *Hub) Stop() {
	h.cancel()
}

// ClientCount 获取当前连接数
func (h *Hub) ClientCount() int {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()
	return len(h.clients)
}

// Register 注册客户端（供外部调用）
func (h *Hub) Register(conn *websocket.Conn) {
	h.register <- NewClient(h, conn, uuid.New().String())
}

func (h *Hub) Broadcast(msg *Message) {
	h.broadcast <- msg
}
