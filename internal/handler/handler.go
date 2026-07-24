package handler

import (
	"go_to_chat/internal/kb"
	"go_to_chat/internal/model"
	"go_to_chat/internal/session"
	"go_to_chat/internal/store"
)

// Handler 聚合所有处理器
type Handler struct {
	Page *PageHandler
	Chat *ChatHandler
	Vdb  *VdbHandler
}

// New 创建处理器
func New(cfg *model.Config, kbMgr *kb.Manager, sessionMgr *session.Manager, metaStore *store.SQLiteStore) *Handler {
	return &Handler{
		Page: NewPageHandler(cfg),
		Chat: NewChatHandler(cfg, kbMgr, sessionMgr, metaStore),
		Vdb:  NewVdbHandler(cfg, kbMgr, metaStore),
	}
}
