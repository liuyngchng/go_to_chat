package handler

import (
	"kb-chat-flow/internal/embedding"
	"kb-chat-flow/internal/engine"
	"kb-chat-flow/internal/kb"
	"kb-chat-flow/internal/model"
	"kb-chat-flow/internal/session"
	"kb-chat-flow/internal/store"

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
	Page     *PageHandler
	Chat     *ChatHandler
	Vdb      *VdbHandler
	Config   *ConfigHandler
	Auth     *AuthHandler
	User     *UserHandler
	Agent    *AgentHandler
	Workflow *WorkflowHandler
	Faq      *FaqHandler
}

// New 创建处理器
func New(cfg *model.Config, kbMgr *kb.Manager, sessionMgr *session.Manager, metaStore store.MetaStore) *Handler {
	embClient := embedding.New(
		cfg.API.EmbeddingAPIURI,
		cfg.API.EmbeddingAPIKey,
		cfg.API.EmbeddingModelName,
	)

	faqHandler := NewFaqHandler(metaStore, embClient)

	// 共享 engine 实例：聊天执行 + 知识库绑定热加载用同一个
	eng := engine.NewEngine(cfg, kbMgr, metaStore)

	return &Handler{
		Page:     NewPageHandler(cfg),
		Chat:     NewChatHandler(cfg, kbMgr, sessionMgr, metaStore, faqHandler, eng),
		Vdb:      NewVdbHandler(cfg, kbMgr, metaStore, eng),
		Config:   NewConfigHandler(cfg, metaStore),
		Auth:     NewAuthHandler(cfg, metaStore),
		User:     NewUserHandler(metaStore),
		Agent:    NewAgentHandler(metaStore),
		Workflow: NewWorkflowHandler(metaStore),
		Faq:      faqHandler,
	}
}
