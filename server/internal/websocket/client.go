package websocket

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"gallary/server/pkg/logger"
)

const (
	// 写超时
	writeWait = 10 * time.Second
	// 读超时（心跳间隔）
	pongWait = 60 * time.Second
	// ping 发送间隔（必须小于 pongWait）
	pingPeriod = (pongWait * 9) / 10
	// 最大消息大小
	maxMessageSize = 512
)

// Client WebSocket 客户端连接
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan *Message // 发送消息缓冲
	clientID string        // 客户端唯一标识

	ctx    context.Context
	cancel context.CancelFunc
}

// NewClient 创建新客户端
func NewClient(hub *Hub, conn *websocket.Conn, clientID string) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan *Message, 256),
		clientID: clientID,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// ReadPump 读取客户端消息的 goroutine
func (c *Client) ReadPump() {
	defer c.clean()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure) {
				logger.Error("WebSocket 读取错误", zap.Error(err))
			}
			break
		}
		// 处理客户端消息
		switch (&msg).Type {
		case MsgTypePing:
			// 响应 pong
			c.send <- NewMessage(MsgTypePong, nil)
		}
	}
}

// WritePump 写入消息到客户端的 goroutine
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	defer c.clean()

	for {
		select {
		case <-c.ctx.Done():
			return
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub 关闭了连接
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				logger.Error("WebSocket 写入错误", zap.Error(err))
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) clean() {
	c.hub.unregister <- c
	_ = c.conn.Close()
	c.cancel()
	select {
	case _ = <-c.send:
	default:
		close(c.send)
	}
}
