package session

import "kb-chat-flow/internal/model"

// SessionStore 会话存储接口。
// 单例模式：MemoryStore（进程内存 + 异步 SQLite 落盘）。
// 集群模式：RedisStore（Redis 存储 + 自动过期）。
type SessionStore interface {
	// GetHistory 获取会话历史。无历史时返回 nil。
	GetHistory(uid string) []model.ChatMessage

	// AddMessage 追加一条消息到会话历史。
	AddMessage(uid, role, content string)

	// Clear 清空指定用户的会话历史。
	Clear(uid string)

	// Stop 停止后台任务（清理 goroutine、关闭连接等）。
	Stop()
}
