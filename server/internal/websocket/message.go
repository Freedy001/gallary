package websocket

import "time"

// MessageType 消息类型
type MessageType string

const (
	// 服务端推送消息类型
	MsgTypeAIQueueStatus MessageType = "ai_queue_status" // AI 队列状态更新
	MsgTypeStorageStats  MessageType = "storage_stats"   // 存储统计更新
	MsgTypeImageCount    MessageType = "image_count"     // 图片总数更新

	// 客户端请求消息类型
	MsgTypePing MessageType = "ping" // 心跳 ping
	MsgTypePong MessageType = "pong" // 心跳 pong
)

// Message WebSocket 消息结构
type Message struct {
	Type      MessageType `json:"type"`                 // 消息类型
	Data      any         `json:"data,omitempty"`       // 消息数据
	Timestamp int64       `json:"timestamp"`            // 时间戳（毫秒）
	RequestID string      `json:"request_id,omitempty"` // 请求 ID（用于请求-响应匹配）
}

// NewMessage 创建新消息
func NewMessage(msgType MessageType, data any) *Message {
	return &Message{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}
}
