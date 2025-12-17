package websocket

import (
	"gallary/server/internal/model"
	"gallary/server/internal/storage"
)

// Notifier WebSocket 通知器接口
// 供其他服务调用，用于触发 WebSocket 推送
type Notifier interface {
	OnClientSetup(fun func(notifier Notifier))

	// NotifyAIQueueStatus 通知 AI 队列状态变化
	NotifyAIQueueStatus(status *model.AIQueueStatus)

	// NotifyStorageStats 通知存储统计变化
	NotifyStorageStats(stats *storage.MultiStorageStats)

	// NotifyImageCount 通知图片总数变化
	NotifyImageCount(count int64)
}

// hubNotifier Hub 实现的通知器
type hubNotifier struct {
	hub *Hub
}

// NewNotifier 创建通知器
func NewNotifier(hub *Hub) Notifier {
	return &hubNotifier{hub: hub}
}

func (n *hubNotifier) OnClientSetup(fun func(notifier Notifier)) {
	n.hub.setup = append(n.hub.setup, func(client *Client) { fun(n) })
}

func (n *hubNotifier) NotifyAIQueueStatus(status *model.AIQueueStatus) {
	n.hub.Broadcast(NewMessage(MsgTypeAIQueueStatus, status))
}

func (n *hubNotifier) NotifyStorageStats(stats *storage.MultiStorageStats) {
	n.hub.Broadcast(NewMessage(MsgTypeStorageStats, stats))
}

func (n *hubNotifier) NotifyImageCount(count int64) {
	n.hub.Broadcast(NewMessage(MsgTypeImageCount, count))
}

// NoopNotifier 空实现（用于不需要 WebSocket 的场景）
type NoopNotifier struct{}

func (n *NoopNotifier) OnClientSetup(func(notifier Notifier))         {}
func (n *NoopNotifier) NotifyAIQueueStatus(*model.AIQueueStatus)      {}
func (n *NoopNotifier) NotifyStorageStats(*storage.MultiStorageStats) {}
func (n *NoopNotifier) NotifyImageCount(int64)                        {}
