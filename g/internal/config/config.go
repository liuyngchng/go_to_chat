package config

import (
	"fmt"
	"os"
	"strconv"

	"kb-chat-flow/internal/model"
	"kb-chat-flow/internal/store"

	"gopkg.in/yaml.v3"
)

// Load 从 YAML 文件加载配置（server + milvus 部分）
// sys、api 配置将在后续从 SQLite 数据库加载
func Load(path string) (*model.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件 %s 失败: %w", path, err)
	}

	var cfg model.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 设置默认值
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 19007
	}
	if cfg.KB.ChunkSize == 0 {
		cfg.KB.ChunkSize = 300
	}
	if cfg.KB.ChunkOverlap == 0 {
		cfg.KB.ChunkOverlap = 80
	}
	if cfg.KB.TopK == 0 {
		cfg.KB.TopK = 3
	}
	if cfg.KB.ScoreThreshold == 0 {
		cfg.KB.ScoreThreshold = 0.1
	}
	if cfg.LLM.Temperature == 0 {
		cfg.LLM.Temperature = 0.7
	}
	if cfg.LLM.TopP == 0 {
		cfg.LLM.TopP = 0.9
	}
	if cfg.LLM.MaxTokens == 0 {
		cfg.LLM.MaxTokens = 2048
	}
	if cfg.Faq.MatchThreshold == 0 {
		cfg.Faq.MatchThreshold = 0.85
	}

	return &cfg, nil
}

// LoadRuntimeConfig 从数据库加载运行时配置（sys.name、api 等），覆盖 YAML 中的初始值。
// 注意：sys.auth 只从 cfg.yml 读取，不从数据库加载，也不在页面上设置。
// 如果 sys_config 表为空，则用 cfg 中的 YAML 值作为种子写入数据库。
func LoadRuntimeConfig(s store.MetaStore, cfg *model.Config) error {
	if err := s.SeedDefaultConfigs(cfg.Sys.Name); err != nil {
		return fmt.Errorf("初始化默认配置失败: %w", err)
	}

	// 从数据库读取所有配置
	configs, err := s.GetAllConfigs()
	if err != nil {
		return fmt.Errorf("读取系统配置失败: %w", err)
	}

	// 应用配置到 cfg 结构体
	applyConfig(configs, cfg)

	return nil
}

// ReloadRuntimeConfig 从数据库重新加载运行时配置（用于配置更新后刷新）
func ReloadRuntimeConfig(s store.MetaStore, cfg *model.Config) error {
	configs, err := s.GetAllConfigs()
	if err != nil {
		return fmt.Errorf("读取系统配置失败: %w", err)
	}
	applyConfig(configs, cfg)
	return nil
}

// applyConfig 将 key-value 配置映射到 Config 结构体
func applyConfig(configs map[string]string, cfg *model.Config) {
	if v, ok := configs["sys.name"]; ok && v != "" {
		cfg.Sys.Name = v
	}
	// sys.auth 只从 cfg.yml 读取，不从数据库覆盖
	if v, ok := configs["sys.api_auth"]; ok {
		cfg.Sys.ApiAuth = v == "true"
	}
	if v, ok := configs["api.llm_api_uri"]; ok && v != "" {
		cfg.API.LLMAPIURI = v
	}
	if v, ok := configs["api.llm_api_key"]; ok && v != "" {
		cfg.API.LLMAPIKey = v
	}
	if v, ok := configs["api.llm_model_name"]; ok && v != "" {
		cfg.API.LLMModelName = v
	}
	if v, ok := configs["api.embedding_api_uri"]; ok && v != "" {
		cfg.API.EmbeddingAPIURI = v
	}
	if v, ok := configs["api.embedding_api_key"]; ok && v != "" {
		cfg.API.EmbeddingAPIKey = v
	}
	if v, ok := configs["api.embedding_model_name"]; ok && v != "" {
		cfg.API.EmbeddingModelName = v
	}

	// 知识库参数
	if v, ok := configs["kb.chunk_size"]; ok && v != "" {
		cfg.KB.ChunkSize, _ = strconv.Atoi(v)
	}
	if v, ok := configs["kb.chunk_overlap"]; ok && v != "" {
		cfg.KB.ChunkOverlap, _ = strconv.Atoi(v)
	}
	// 兜底：overlap 必须严格小于 chunkSize 的一定比例，否则文本切分会死循环
	if cfg.KB.ChunkOverlap >= cfg.KB.ChunkSize {
		cfg.KB.ChunkOverlap = cfg.KB.ChunkSize / 3
	}
	if v, ok := configs["kb.top_k"]; ok && v != "" {
		cfg.KB.TopK, _ = strconv.Atoi(v)
	}
	if v, ok := configs["kb.score_threshold"]; ok && v != "" {
		cfg.KB.ScoreThreshold, _ = strconv.ParseFloat(v, 64)
	}
	if v, ok := configs["kb.rerank_enabled"]; ok && v != "" {
		cfg.KB.RerankEnabled = v == "true"
	}
	if v, ok := configs["kb.rerank_retrieve_n"]; ok && v != "" {
		cfg.KB.RerankRetrieveN, _ = strconv.Atoi(v)
	}

	// Rerank API 配置
	if v, ok := configs["api.rerank_api_uri"]; ok && v != "" {
		cfg.API.RerankAPIURI = v
	}
	if v, ok := configs["api.rerank_api_key"]; ok && v != "" {
		cfg.API.RerankAPIKey = v
	}
	if v, ok := configs["api.rerank_model_name"]; ok && v != "" {
		cfg.API.RerankModelName = v
	}

	// LLM 参数
	if v, ok := configs["llm.temperature"]; ok && v != "" {
		cfg.LLM.Temperature, _ = strconv.ParseFloat(v, 64)
	}
	if v, ok := configs["llm.top_p"]; ok && v != "" {
		cfg.LLM.TopP, _ = strconv.ParseFloat(v, 64)
	}
	if v, ok := configs["llm.max_tokens"]; ok && v != "" {
		cfg.LLM.MaxTokens, _ = strconv.Atoi(v)
	}

	// FAQ 参数
	if v, ok := configs["faq.match_threshold"]; ok && v != "" {
		cfg.Faq.MatchThreshold, _ = strconv.ParseFloat(v, 64)
	}
}
