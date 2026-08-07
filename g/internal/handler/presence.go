package handler

import "time"

// PresenceStore 在线状态存储接口。
// 单例模式：MemoryPresence（进程内存 sync.Map）。
// 集群模式：RedisPresence（Redis HSET + TTL）。
type PresenceStore interface {
	// SetPresence 记录座席上线
	SetPresence(userName string, loginTime time.Time)

	// RemovePresence 移除座席在线状态
	RemovePresence(userName string)

	// GetOnlineAgents 获取所有在线座席列表（含备注信息）
	GetOnlineAgents() []OnlineAgent

	// HasPresence 检查座席是否在线
	HasPresence(userName string) bool
}
