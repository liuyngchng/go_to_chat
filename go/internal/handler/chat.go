package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"go_to_chat/internal/kb"
	"go_to_chat/internal/llm"
	"go_to_chat/internal/model"
	"go_to_chat/internal/session"
	"go_to_chat/internal/store"

	"github.com/gin-gonic/gin"
)

// ChatHandler 聊天处理器
type ChatHandler struct {
	cfg        *model.Config
	kbMgr      *kb.Manager
	sessionMgr *session.Manager
	llmClient  *llm.Client
	store      *store.SQLiteStore
}

// NewChatHandler 创建聊天处理器
func NewChatHandler(cfg *model.Config, kbMgr *kb.Manager, sessionMgr *session.Manager, metaStore *store.SQLiteStore) *ChatHandler {
	llmClient := llm.New(
		cfg.API.LLMAPIURI,
		cfg.API.LLMAPIKey,
		cfg.API.LLMModelName,
	)
	llmClient.SetParams(cfg.LLM.Temperature, cfg.LLM.TopP, cfg.LLM.MaxTokens)

	return &ChatHandler{
		cfg:        cfg,
		kbMgr:      kbMgr,
		sessionMgr: sessionMgr,
		llmClient:  llmClient,
		store:      metaStore,
	}
}

// Chat 处理聊天请求，SSE 流式返回
func (h *ChatHandler) Chat(c *gin.Context) {
	var req model.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	uid := getAuthUID(c)
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = "default"
	}

	// 设置 SSE 头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "不支持流式传输"})
		return
	}

	// 获取历史
	history := h.sessionMgr.GetHistory(uid, sessionID)
	historyStr := session.FormatHistory(history)

	// 获取知识库上下文
	curDate := time.Now().Format("2006-01-02")
	curWeek := getWeekdayCN(time.Now().Weekday())

	contextStr := h.kbMgr.SearchAllKBs(req.Msg, uid, h.cfg.KB.TopK, h.cfg.KB.ScoreThreshold)

	// 构建提示词：优先从数据库读取，fallback 到 YAML 配置
	promptTemplate := h.getPromptTemplate()
	systemPrompt := buildPrompt(promptTemplate, contextStr, historyStr, req.Msg, curDate, curWeek)

	slog.Info("chat", "uid", uid, "session", sessionID, "query", req.Msg[:min(50, len(req.Msg))], "contextLen", len(contextStr))

	// 保存用户消息
	h.sessionMgr.AddMessage(uid, sessionID, "user", req.Msg)

	// 调用 LLM 流式
	chunkCh, errCh := h.llmClient.ChatStream(systemPrompt, "")

	var fullResponse strings.Builder

	// 先发送一个初始事件
	fmt.Fprintf(c.Writer, "data: \n\n")
	flusher.Flush()

	for chunk := range chunkCh {
		fullResponse.WriteString(chunk)
		fmt.Fprintf(c.Writer, "data: %s\n\n", chunk)
		flusher.Flush()
	}

	// 检查错误
	select {
	case err := <-errCh:
		if err != nil {
			slog.Error("LLM 错误", "error", err)
			fmt.Fprintf(c.Writer, "data: [错误] %v\n\n", err)
			flusher.Flush()
		}
	default:
	}

	// 发送结束标记
	fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
	flusher.Flush()

	// 保存助手回复
	responseText := fullResponse.String()
	if responseText != "" {
		h.sessionMgr.AddMessage(uid, sessionID, "assistant", responseText)
	}
}

// Clear 清空会话
func (h *ChatHandler) Clear(c *gin.Context) {
	var req model.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	uid := getAuthUID(c)
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = "default"
	}

	h.sessionMgr.Clear(uid, sessionID)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ============================================================
// 辅助函数
// ============================================================

func buildPrompt(template, context, history, question, curDate, curWeek string) string {
	result := template
	result = strings.ReplaceAll(result, "{context}", context)
	result = strings.ReplaceAll(result, "{history}", history)
	result = strings.ReplaceAll(result, "{question}", question)
	result = strings.ReplaceAll(result, "{cur_date}", curDate)
	result = strings.ReplaceAll(result, "{cur_week}", curWeek)
	return result
}

func getWeekdayCN(d time.Weekday) string {
	days := []string{"日", "一", "二", "三", "四", "五", "六"}
	return days[d]
}

// getPromptTemplate 从 SQLite 数据库获取提示词模板，不存在则返回默认模板
func (h *ChatHandler) getPromptTemplate() string {
	if h.store != nil {
		prompt, err := h.store.GetPrompt("chat_msg")
		if err == nil && prompt != "" {
			return prompt
		}
	}
	return store.DefaultChatPrompt()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
