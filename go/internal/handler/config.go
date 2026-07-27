package handler

import (
	"fmt"
	"net/http"

	"go_to_chat/internal/config"
	"go_to_chat/internal/model"
	"go_to_chat/internal/store"

	"github.com/gin-gonic/gin"
)

// ConfigHandler 系统配置 API 处理器
type ConfigHandler struct {
	cfg   *model.Config
	store *store.SQLiteStore
}

// NewConfigHandler 创建配置处理器
func NewConfigHandler(cfg *model.Config, metaStore *store.SQLiteStore) *ConfigHandler {
	return &ConfigHandler{
		cfg:   cfg,
		store: metaStore,
	}
}

// ConfigResponse 配置响应结构
type ConfigResponse struct {
	Sys    SysConfigResp    `json:"sys"`
	API    APIConfigResp    `json:"api"`
	Prompt PromptConfigResp `json:"prompt"`
	KB     KBConfigResp     `json:"kb"`
	LLM    LLMParamsResp    `json:"llm"`
}

type SysConfigResp struct {
	Name    string `json:"name"`
	Auth    string `json:"auth"`
	ApiAuth string `json:"api_auth"`
}

type APIConfigResp struct {
	LLMAPIURI          string `json:"llm_api_uri"`
	LLMAPIKey          string `json:"llm_api_key"`
	LLMModelName       string `json:"llm_model_name"`
	EmbeddingAPIURI    string `json:"embedding_api_uri"`
	EmbeddingAPIKey    string `json:"embedding_api_key"`
	EmbeddingModelName string `json:"embedding_model_name"`
}

type PromptConfigResp struct {
	ChatMsg string `json:"chat_msg"`
}

type KBConfigResp struct {
	ChunkSize      int     `json:"chunk_size"`
	ChunkOverlap   int     `json:"chunk_overlap"`
	TopK           int     `json:"top_k"`
	ScoreThreshold float64 `json:"score_threshold"`
}

type LLMParamsResp struct {
	Temperature float64 `json:"temperature"`
	TopP        float64 `json:"top_p"`
	MaxTokens   int     `json:"max_tokens"`
}

// GetConfig 获取所有配置
func (h *ConfigHandler) GetConfig(c *gin.Context) {
	resp := ConfigResponse{
		Sys: SysConfigResp{
			Name:    h.cfg.Sys.Name,
			Auth:    boolToStr(h.cfg.Sys.Auth),
			ApiAuth: boolToStr(h.cfg.Sys.ApiAuth),
		},
		API: APIConfigResp{
			LLMAPIURI:          h.cfg.API.LLMAPIURI,
			LLMAPIKey:          h.cfg.API.LLMAPIKey,
			LLMModelName:       h.cfg.API.LLMModelName,
			EmbeddingAPIURI:    h.cfg.API.EmbeddingAPIURI,
			EmbeddingAPIKey:    h.cfg.API.EmbeddingAPIKey,
			EmbeddingModelName: h.cfg.API.EmbeddingModelName,
		},
		Prompt: PromptConfigResp{
			ChatMsg: h.getPrompt(),
		},
		KB: KBConfigResp{
			ChunkSize:      h.cfg.KB.ChunkSize,
			ChunkOverlap:   h.cfg.KB.ChunkOverlap,
			TopK:           h.cfg.KB.TopK,
			ScoreThreshold: h.cfg.KB.ScoreThreshold,
		},
		LLM: LLMParamsResp{
			Temperature: h.cfg.LLM.Temperature,
			TopP:        h.cfg.LLM.TopP,
			MaxTokens:   h.cfg.LLM.MaxTokens,
		},
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

// UpdateConfig 更新配置
func (h *ConfigHandler) UpdateConfig(c *gin.Context) {
	var req ConfigResponse
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 更新系统配置
	if req.Sys.Name != "" {
		if err := h.store.SetConfig("sys.name", req.Sys.Name, "系统名称"); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存系统名称失败: " + err.Error()})
			return
		}
	}
	if req.Sys.Auth != "" {
		if err := h.store.SetConfig("sys.auth", req.Sys.Auth, "是否启用认证"); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存认证配置失败: " + err.Error()})
			return
		}
	}
	if req.Sys.ApiAuth != "" {
		if err := h.store.SetConfig("sys.api_auth", req.Sys.ApiAuth, "是否启用接口认证"); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存接口认证配置失败: " + err.Error()})
			return
		}
	}

	// 更新大模型 API 配置
	apiUpdates := map[string]string{
		"api.llm_api_uri":          req.API.LLMAPIURI,
		"api.llm_api_key":          req.API.LLMAPIKey,
		"api.llm_model_name":       req.API.LLMModelName,
		"api.embedding_api_uri":    req.API.EmbeddingAPIURI,
		"api.embedding_api_key":    req.API.EmbeddingAPIKey,
		"api.embedding_model_name": req.API.EmbeddingModelName,
	}
	for key, value := range apiUpdates {
		if value != "" {
			if err := h.store.SetConfig(key, value, ""); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "保存配置 " + key + " 失败: " + err.Error()})
				return
			}
		}
	}

	// 更新提示词
	if req.Prompt.ChatMsg != "" {
		if err := h.store.UpsertPrompt("chat_msg", req.Prompt.ChatMsg, 0); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存提示词失败: " + err.Error()})
			return
		}
	}

	// 更新知识库参数
	if req.KB.ChunkSize > 0 {
		if err := h.store.SetConfig("kb.chunk_size", fmt.Sprintf("%d", req.KB.ChunkSize), "文本分片大小"); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存分片大小失败: " + err.Error()})
			return
		}
	}
	if req.KB.ChunkOverlap > 0 {
		if err := h.store.SetConfig("kb.chunk_overlap", fmt.Sprintf("%d", req.KB.ChunkOverlap), "分片重叠大小"); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存分片重叠失败: " + err.Error()})
			return
		}
	}
	if req.KB.TopK > 0 {
		if err := h.store.SetConfig("kb.top_k", fmt.Sprintf("%d", req.KB.TopK), "检索返回条数"); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存返回条数失败: " + err.Error()})
			return
		}
	}
	if req.KB.ScoreThreshold > 0 {
		if err := h.store.SetConfig("kb.score_threshold", fmt.Sprintf("%.3f", req.KB.ScoreThreshold), "相似度阈值"); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存阈值失败: " + err.Error()})
			return
		}
	}

	// 更新 LLM 参数
	if req.LLM.Temperature > 0 {
		if err := h.store.SetConfig("llm.temperature", fmt.Sprintf("%.2f", req.LLM.Temperature), "LLM 温度"); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存温度参数失败: " + err.Error()})
			return
		}
	}
	if req.LLM.TopP > 0 {
		if err := h.store.SetConfig("llm.top_p", fmt.Sprintf("%.2f", req.LLM.TopP), "LLM Top-P"); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存 Top-P 失败: " + err.Error()})
			return
		}
	}
	if req.LLM.MaxTokens > 0 {
		if err := h.store.SetConfig("llm.max_tokens", fmt.Sprintf("%d", req.LLM.MaxTokens), "LLM 最大 Token 数"); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存最大 Token 数失败: " + err.Error()})
			return
		}
	}

	// 重新加载运行时配置到内存
	if err := config.ReloadRuntimeConfig(h.store, h.cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "重新加载配置失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// getPrompt 从数据库获取提示词模板
func (h *ConfigHandler) getPrompt() string {
	if h.store != nil {
		prompt, err := h.store.GetPrompt("chat_msg")
		if err == nil && prompt != "" {
			return prompt
		}
	}
	return store.DefaultChatPrompt()
}

func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
