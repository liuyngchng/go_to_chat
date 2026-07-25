package model

import "time"

// ============================================================
// 配置相关
// ============================================================

// Config 应用配置
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Sys     SysConfig     `yaml:"sys"`
	API     APIConfig     `yaml:"api"`
	Milvus  MilvusConfig  `yaml:"milvus"`
	Prompts PromptsConfig `yaml:"prompts"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port  int  `yaml:"port"`
	Debug bool `yaml:"debug"`
}

// SysConfig 系统配置
type SysConfig struct {
	Name string `yaml:"name"`
	Auth bool   `yaml:"auth"`
}

// APIConfig API 配置
type APIConfig struct {
	LLMAPIURI         string `yaml:"llm_api_uri"`
	LLMAPIKey         string `yaml:"llm_api_key"`
	LLMModelName       string `yaml:"llm_model_name"`
	EmbeddingAPIURI   string `yaml:"embedding_api_uri"`
	EmbeddingAPIKey   string `yaml:"embedding_api_key"`
	EmbeddingModelName string `yaml:"embedding_model_name"`
}

// MilvusConfig Milvus 配置
type MilvusConfig struct {
	URI   string `yaml:"uri"`
	Token string `yaml:"token"`
}

// PromptsConfig 提示词配置
type PromptsConfig struct {
	ChatMsg string `yaml:"chat_msg"`
}

// ============================================================
// 知识库相关
// ============================================================

// VdbInfo 知识库元数据
type VdbInfo struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	UID        string    `json:"uid"`
	IsPublic   bool      `json:"is_public"`
	IsDefault  bool      `json:"is_default"`
	CreateTime time.Time `json:"create_time"`
}

// VdbFileInfo 知识库文件信息
type VdbFileInfo struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	UID         string    `json:"uid"`
	VdbID       int64     `json:"vdb_id"`
	TaskID      string    `json:"task_id"`
	FilePath    string    `json:"file_path"`
	Percent     float64   `json:"percent"`
	ProcessInfo string    `json:"process_info"`
	FileMD5     string    `json:"file_md5"`
	CreateTime  time.Time `json:"create_time"`
}

// ============================================================
// 聊天相关
// ============================================================

// ChatMessage 聊天消息
type ChatMessage struct {
	Role    string `json:"role"`    // "user" or "assistant"
	Content string `json:"content"`
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Msg        string `form:"msg" binding:"required"`
	UID        string `form:"uid"`
	SessionID  string `form:"session_id"`
	AppSource  string `form:"app_source"`
}

// ChatHistory 会话历史
type ChatHistory struct {
	SessionID string        `json:"session_id"`
	UID       string        `json:"uid"`
	Messages  []ChatMessage `json:"messages"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// ============================================================
// LLM / Embedding 相关
// ============================================================

// ChatCompletionRequest OpenAI 兼容的聊天请求
type ChatCompletionRequest struct {
	Model    string              `json:"model"`
	Messages []ChatCompletionMsg `json:"messages"`
	Stream   bool                `json:"stream"`
}

// ChatCompletionMsg 消息
type ChatCompletionMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionChunk 流式响应片段
type ChatCompletionChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

// EmbeddingRequest 向量化请求
type EmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// EmbeddingResponse 向量化响应
type EmbeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

// ============================================================
// 向量存储相关
// ============================================================

// SearchResult 检索结果
type SearchResult struct {
	ID      string            `json:"id"`
	Content string            `json:"content"`
	Meta    map[string]string `json:"metadata"`
	Score   float64           `json:"score"`
}

// VectorRecord 向量记录（用于插入）
type VectorRecord struct {
	ID      string            `json:"id"`
	Vector  []float64         `json:"vector"`
	Content string            `json:"content"`
	Meta    map[string]string `json:"metadata"`
}
