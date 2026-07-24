package session

import (
	"sync"
	"time"

	"go_to_chat/internal/model"
)

const (
	MaxHistoryRounds   = 5    // 最多保留 5 轮（10 条消息）
	SessionTimeout     = 30 * time.Minute
	CleanupInterval    = 10 * time.Minute
)

// Manager 会话管理器（内存）
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*model.ChatHistory // key: uid + ":" + sessionID
}

// NewManager 创建会话管理器
func NewManager() *Manager {
	m := &Manager{
		sessions: make(map[string]*model.ChatHistory),
	}
	// 启动过期清理
	go m.cleanup()
	return m
}

// GetHistory 获取会话历史
func (m *Manager) GetHistory(uid, sessionID string) []model.ChatMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := uid + ":" + sessionID
	if h, ok := m.sessions[key]; ok {
		// 复制一份返回，避免外部修改
		result := make([]model.ChatMessage, len(h.Messages))
		copy(result, h.Messages)
		return result
	}
	return nil
}

// AddMessage 添加消息到会话历史
func (m *Manager) AddMessage(uid, sessionID, role, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := uid + ":" + sessionID
	h, ok := m.sessions[key]
	if !ok {
		h = &model.ChatHistory{
			SessionID: sessionID,
			UID:       uid,
			Messages:  make([]model.ChatMessage, 0),
			CreatedAt: time.Now(),
		}
		m.sessions[key] = h
	}

	h.Messages = append(h.Messages, model.ChatMessage{
		Role:    role,
		Content: content,
	})
	h.UpdatedAt = time.Now()

	// 限制历史长度（保留最近 N 轮）
	maxMessages := MaxHistoryRounds * 2
	if len(h.Messages) > maxMessages {
		h.Messages = h.Messages[len(h.Messages)-maxMessages:]
	}
}

// Clear 清空会话历史
func (m *Manager) Clear(uid, sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := uid + ":" + sessionID
	delete(m.sessions, key)
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
			result += "客服：" + msg.Content + "\n"
		}
	}
	return result
}

// cleanup 定期清理过期会话
func (m *Manager) cleanup() {
	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for key, h := range m.sessions {
			if now.Sub(h.UpdatedAt) > SessionTimeout {
				delete(m.sessions, key)
			}
		}
		m.mu.Unlock()
	}
}
