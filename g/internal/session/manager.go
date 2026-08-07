package session

import (
	"log/slog"
	"sync"
	"time"

	"kb-chat-flow/internal/model"
	"kb-chat-flow/internal/store"
)

const (
	MaxHistoryRounds       = 5            // 最多保留 5 轮（10 条消息）
	MaxMessages            = MaxHistoryRounds * 2
	SessionTimeout         = 30 * time.Minute
	CleanupInterval        = 10 * time.Minute
	PersistInterval        = 5 * time.Minute // TODO: 后续迁移至 Redis 替代 SQLite 持久化
	PersistLoadLimit       = 20            // 从 DB 加载时最多取多少条
)

// sessionEntry 每个会话独立锁，避免多 uid 竞争
type sessionEntry struct {
	mu sync.Mutex
	h  *model.ChatHistory
}

// Manager 会话管理器（内存 + SQLite 持久化）
// TODO: 后续迁移至 Redis 替代 SQLite 持久化
type Manager struct {
	sessions sync.Map // key: uid → *sessionEntry
	store    store.MetaStore
	stopCh   chan struct{}
}

// NewManager 创建会话管理器
func NewManager(s store.MetaStore) *Manager {
	m := &Manager{
		store:  s,
		stopCh: make(chan struct{}),
	}
	go m.cleanup()
	go m.persistLoop()
	return m
}

// Stop 停止后台任务
func (m *Manager) Stop() {
	close(m.stopCh)
}

// GetHistory 获取会话历史（优先内存，fallback DB）
func (m *Manager) GetHistory(uid string) []model.ChatMessage {
	v, ok := m.sessions.Load(uid)
	if ok {
		entry := v.(*sessionEntry)
		entry.mu.Lock()
		result := make([]model.ChatMessage, len(entry.h.Messages))
		copy(result, entry.h.Messages)
		entry.mu.Unlock()
		return result
	}

	// 内存没有，尝试从 DB 加载
	if m.store != nil {
		msgs, err := m.store.GetChatMessages(uid, PersistLoadLimit)
		if err != nil {
			slog.Warn("从 DB 加载聊天历史失败", "uid", uid, "error", err)
			return nil
		}
		if len(msgs) > 0 {
			// 恢复到内存
			entry := &sessionEntry{
				h: &model.ChatHistory{
					UID:       uid,
					Messages:  msgs,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			}
			m.sessions.Store(uid, entry)
			slog.Info("从 DB 恢复聊天历史", "uid", uid, "count", len(msgs))
			return msgs
		}
	}

	return nil
}

// AddMessage 添加消息到会话历史（异步持久化）
func (m *Manager) AddMessage(uid, role, content string) {
	entry := &sessionEntry{
		h: &model.ChatHistory{
			UID:       uid,
			Messages:  make([]model.ChatMessage, 0),
			CreatedAt: time.Now(),
		},
	}

	actual, loaded := m.sessions.LoadOrStore(uid, entry)
	if loaded {
		entry = actual.(*sessionEntry)
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	entry.h.Messages = append(entry.h.Messages, model.ChatMessage{
		Role:    role,
		Content: content,
	})
	entry.h.UpdatedAt = time.Now()

	if len(entry.h.Messages) > MaxMessages {
		entry.h.Messages = entry.h.Messages[len(entry.h.Messages)-MaxMessages:]
	}

	// 异步持久化到 DB
	if m.store != nil {
		go func() {
			if err := m.store.SaveChatMessage(uid, role, content); err != nil {
				slog.Warn("持久化聊天消息失败", "uid", uid, "error", err)
			}
		}()
	}
}

// Clear 清空会话历史（内存 + DB）
func (m *Manager) Clear(uid string) {
	m.sessions.Delete(uid)
	if m.store != nil {
		if err := m.store.ClearChatMessages(uid); err != nil {
			slog.Warn("清空 DB 聊天历史失败", "uid", uid, "error", err)
		}
	}
}

// FormatHistory 格式化历史消息为字符串
func FormatHistory(messages []model.ChatMessage) string {
	if len(messages) == 0 {
		return "（无历史对话）"
	}

	var result string
	for _, msg := range messages {
		if msg.Role == "user" {
			result += "用户：" + msg.Content + "\n"
		} else {
			result += "机器人：" + msg.Content + "\n"
		}
	}
	return result
}

// persistLoop 定时全量持久化内存会话到 DB
func (m *Manager) persistLoop() {
	ticker := time.NewTicker(PersistInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			slog.Info("会话持久化循环已停止")
			return
		case <-ticker.C:
			m.persistAll()
		}
	}
}

// persistAll 全量持久化所有内存会话
func (m *Manager) persistAll() {
	if m.store == nil {
		return
	}
	count := 0
	m.sessions.Range(func(key, value any) bool {
		uid := key.(string)
		entry := value.(*sessionEntry)
		entry.mu.Lock()
		messages := make([]model.ChatMessage, len(entry.h.Messages))
		copy(messages, entry.h.Messages)
		entry.mu.Unlock()

		// 全量替换：先删后插
		if err := m.store.ClearChatMessages(uid); err != nil {
			slog.Warn("持久化前清空失败", "uid", uid, "error", err)
			return true
		}
		for _, msg := range messages {
			if err := m.store.SaveChatMessage(uid, msg.Role, msg.Content); err != nil {
				slog.Warn("全量持久化失败", "uid", uid, "error", err)
				return true
			}
		}
		count++
		return true
	})
	if count > 0 {
		slog.Info("会话全量持久化完成", "sessions", count)
	}
}

// cleanup 定期清理过期会话
func (m *Manager) cleanup() {
	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			now := time.Now()
			m.sessions.Range(func(key, value any) bool {
				entry := value.(*sessionEntry)
				entry.mu.Lock()
				if now.Sub(entry.h.UpdatedAt) > SessionTimeout {
					entry.mu.Unlock()
					m.sessions.Delete(key)
				} else {
					entry.mu.Unlock()
				}
				return true
			})
		}
	}
}
