package session

import (
	"sync"
	"time"

	"kb-chat-flow/internal/model"
)

const (
	MaxHistoryRounds = 5 // 最多保留 5 轮（10 条消息）
	SessionTimeout   = 30 * time.Minute
	CleanupInterval  = 10 * time.Minute
)

// sessionEntry 每个会话独立锁，避免多 uid 竞争
type sessionEntry struct {
	mu sync.Mutex
	h  *model.ChatHistory
}

// Manager 会话管理器（内存，sync.Map 分片）
type Manager struct {
	sessions sync.Map // key: uid:sessionID → *sessionEntry
}

// NewManager 创建会话管理器
func NewManager() *Manager {
	m := &Manager{}
	go m.cleanup()
	return m
}

// GetHistory 获取会话历史
func (m *Manager) GetHistory(uid, sessionID string) []model.ChatMessage {
	key := uid + ":" + sessionID
	v, ok := m.sessions.Load(key)
	if !ok {
		return nil
	}

	entry := v.(*sessionEntry)
	entry.mu.Lock()
	result := make([]model.ChatMessage, len(entry.h.Messages))
	copy(result, entry.h.Messages)
	entry.mu.Unlock()

	return result
}

// AddMessage 添加消息到会话历史
func (m *Manager) AddMessage(uid, sessionID, role, content string) {
	key := uid + ":" + sessionID

	entry := &sessionEntry{
		h: &model.ChatHistory{
			SessionID: sessionID,
			UID:       uid,
			Messages:  make([]model.ChatMessage, 0),
			CreatedAt: time.Now(),
		},
	}

	actual, loaded := m.sessions.LoadOrStore(key, entry)
	if loaded {
		// 已存在，使用旧的 entry
		entry = actual.(*sessionEntry)
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	entry.h.Messages = append(entry.h.Messages, model.ChatMessage{
		Role:    role,
		Content: content,
	})
	entry.h.UpdatedAt = time.Now()

	// 限制历史长度（保留最近 N 轮）
	maxMessages := MaxHistoryRounds * 2
	if len(entry.h.Messages) > maxMessages {
		entry.h.Messages = entry.h.Messages[len(entry.h.Messages)-maxMessages:]
	}
}

// Clear 清空会话历史
func (m *Manager) Clear(uid, sessionID string) {
	key := uid + ":" + sessionID
	m.sessions.Delete(key)
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

// cleanup 定期清理过期会话
func (m *Manager) cleanup() {
	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
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
