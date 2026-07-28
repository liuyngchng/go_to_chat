package embedding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"go_to_chat/internal/model"
)

const MaxBatchSize = 32

// 高并发 HTTP 连接池
var embedHTTPClient = &http.Client{
	Transport: &http.Transport{
		Proxy:                 nil, // 忽略所有代理，直连
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	},
	Timeout: 60 * time.Second,
}

// Client Embedding 客户端（OpenAI 兼容接口）
type Client struct {
	BaseURL   string
	APIKey    string
	ModelName string
	httpCli   *http.Client
}

// New 创建 Embedding 客户端
func New(baseURL, apiKey, modelName string) *Client {
	return &Client{
		BaseURL:   strings.TrimRight(baseURL, "/"),
		APIKey:    apiKey,
		ModelName: modelName,
		httpCli:   embedHTTPClient,
	}
}

// Embed 将文本列表转为向量
func (c *Client) Embed(texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	allEmbeddings := make([][]float64, 0, len(texts))

	for i := 0; i < len(texts); i += MaxBatchSize {
		end := i + MaxBatchSize
		if end > len(texts) {
			end = len(texts)
		}
		batch := texts[i:end]

		embeddings, err := c.embedBatch(batch)
		if err != nil {
			return nil, fmt.Errorf("batch %d-%d embedding 失败: %w", i, end, err)
		}
		allEmbeddings = append(allEmbeddings, embeddings...)
	}

	return allEmbeddings, nil
}

// EmbedSingle 将单个文本转为向量
func (c *Client) EmbedSingle(text string) ([]float64, error) {
	results, err := c.Embed([]string{text})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("embedding 返回空结果")
	}
	return results[0], nil
}

// Dimension 探测 embedding 模型的输出维度
func (c *Client) Dimension() (int, error) {
	vec, err := c.EmbedSingle("dimension probe")
	if err != nil {
		slog.Error("embedding 维度探测失败", "model", c.ModelName, "baseURL", c.BaseURL, "error", err)
		return 0, err
	}
	dim := len(vec)
	slog.Info("embedding 模型探测成功", "model", c.ModelName, "dim", dim)
	return dim, nil
}

func (c *Client) embedBatch(texts []string) ([][]float64, error) {
	if c.BaseURL == "" {
		return nil, fmt.Errorf("Embedding API 地址未配置，请在系统管理页面中设置 embedding_api_uri")
	}

	reqBody := model.EmbeddingRequest{
		Model: c.ModelName,
		Input: texts,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := c.BaseURL + "/embeddings"
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.httpCli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 embedding API 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		slog.Error("embedding API 返回非 200",
			"url", url,
			"model", c.ModelName,
			"status", resp.StatusCode,
			"body", bodyStr,
		)
		return nil, fmt.Errorf("embedding API 返回错误 %d: %s", resp.StatusCode, bodyStr)
	}

	var result model.EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析 embedding 响应失败: %w", err)
	}

	embeddings := make([][]float64, len(result.Data))
	for i, item := range result.Data {
		embeddings[i] = item.Embedding
	}

	return embeddings, nil
}
