package handler

import (
	"sync"
	"time"

	"kb-chat-flow/internal/store"
)

// MemoryPresence 单例模式在线状态存储：进程内存 sync.Map。
type MemoryPresence struct {
	agents sync.Map // key: userName (string), value: loginTime (time.Time)
	db     store.MetaStore
}

// NewMemoryPresence 创建内存在线状态存储
func NewMemoryPresence(db store.MetaStore) *MemoryPresence {
	return &MemoryPresence{db: db}
}

// SetPresence 记录座席上线
func (p *MemoryPresence) SetPresence(userName string, loginTime time.Time) {
	p.agents.Store(userName, loginTime)
}

// RemovePresence 移除座席在线状态
func (p *MemoryPresence) RemovePresence(userName string) {
	p.agents.Delete(userName)
}

// GetOnlineAgents 获取所有在线座席列表
func (p *MemoryPresence) GetOnlineAgents() []OnlineAgent {
	var agents []OnlineAgent
	p.agents.Range(func(key, value interface{}) bool {
		userName := key.(string)
		loginTime := value.(time.Time)

		note := ""
		if p.db != nil {
			user, err := p.db.GetUserByName(userName)
			if err == nil && user != nil {
				note = user.Note
			}
		}

		agents = append(agents, OnlineAgent{
			UserName:  userName,
			LoginTime: loginTime.Format("2006-01-02 15:04:05"),
			Note:      note,
		})
		return true
	})
	return agents
}

// HasPresence 检查座席是否在线
func (p *MemoryPresence) HasPresence(userName string) bool {
	_, ok := p.agents.Load(userName)
	return ok
}
