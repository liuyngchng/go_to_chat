package handler

import (
	"go_to_chat/internal/kb"
	"go_to_chat/internal/model"
	"go_to_chat/internal/session"
	"go_to_chat/internal/store"

	"github.com/gin-gonic/gin"
)

// getAuthUID 从认证上下文中获取用户名作为 uid
func getAuthUID(c *gin.Context) string {
	userVal, exists := c.Get("user")
	if !exists {
		return "default"
	}
	user, ok := userVal.(*model.User)
	if !ok {
		return "default"
	}
	return user.UserName
}

// Handler 聚合所有处理器
type Handler struct {
	Page   *PageHandler
	Chat   *ChatHandler
	Vdb    *VdbHandler
	Config *ConfigHandler
	Auth   *AuthHandler
	User   *UserHandler
}

// New 创建处理器
func New(cfg *model.Config, kbMgr *kb.Manager, sessionMgr *session.Manager, metaStore *store.SQLiteStore) *Handler {
	return &Handler{
		Page:   NewPageHandler(cfg),
		Chat:   NewChatHandler(cfg, kbMgr, sessionMgr, metaStore),
		Vdb:    NewVdbHandler(cfg, kbMgr, metaStore),
		Config: NewConfigHandler(cfg, metaStore),
		Auth:   NewAuthHandler(cfg, metaStore),
		User:   NewUserHandler(metaStore),
	}
}
