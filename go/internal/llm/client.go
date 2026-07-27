package llm

import (
	"bufio"
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

// 高并发 HTTP 连接池（流式 LLM 调用需要更长超时）
var llmHTTPClient = &http.Client{
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
	Timeout: 120 * time.Second,
}

// Client LLM 客户端（OpenAI 兼容接口）
type Client struct {
	BaseURL     string
	APIKey      string
	ModelName   string
	Temperature *float64
	TopP        *float64
	MaxTokens   *int
	httpCli     *http.Client
}

// New 创建 LLM 客户端
func New(baseURL, apiKey, modelName string) *Client {
	return &Client{
		BaseURL:   strings.TrimRight(baseURL, "/"),
		APIKey:    apiKey,
		ModelName: modelName,
		httpCli:   llmHTTPClient,
	}
}

// SetParams 设置 LLM 模型参数
func (c *Client) SetParams(temperature, topP float64, maxTokens int) {
	if temperature > 0 {
		c.Temperature = &temperature
	}
	if topP > 0 {
		c.TopP = &topP
	}
	if maxTokens > 0 {
		c.MaxTokens = &maxTokens
	}
}

// ChatStream 流式聊天，通过 channel 返回每个文本片段
func (c *Client) ChatStream(systemPrompt, userMessage string) (<-chan string, <-chan error) {
	chunkCh := make(chan string, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(chunkCh)
		defer close(errCh)

		messages := []model.ChatCompletionMsg{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMessage},
		}

			reqBody := model.ChatCompletionRequest{
				Model:       c.ModelName,
				Messages:    messages,
				Stream:      true,
				Temperature: c.Temperature,
				TopP:        c.TopP,
				MaxTokens:   c.MaxTokens,
			}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			errCh <- fmt.Errorf("序列化请求失败: %w", err)
			return
		}

		url := c.BaseURL + "/chat/completions"
		req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
		if err != nil {
			errCh <- fmt.Errorf("创建请求失败: %w", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.APIKey)

		resp, err := c.httpCli.Do(req)
		if err != nil {
			errCh <- fmt.Errorf("请求 LLM 失败: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errCh <- fmt.Errorf("LLM API 返回错误 %d: %s", resp.StatusCode, string(body))
			return
		}

		// 读取 SSE 流
		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				slog.Warn("读取流数据出错", "error", err)
				break
			}

			line = strings.TrimSpace(line)
			if line == "" || !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var chunk model.ChatCompletionChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
				chunkCh <- chunk.Choices[0].Delta.Content
			}
		}
	}()

	return chunkCh, errCh
}

// Chat 非流式聊天
func (c *Client) Chat(systemPrompt, userMessage string) (string, error) {
	messages := []model.ChatCompletionMsg{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMessage},
	}
	reqBody := model.ChatCompletionRequest{
		Model:       c.ModelName,
		Messages:    messages,
		Stream:      false,
		Temperature: c.Temperature,
		TopP:        c.TopP,
		MaxTokens:   c.MaxTokens,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	url := c.BaseURL + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.httpCli.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求 LLM 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// 解析非流式响应
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	if len(result.Choices) > 0 {
		return result.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("LLM 返回空响应")
}
