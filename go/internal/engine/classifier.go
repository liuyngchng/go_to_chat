package engine

import (
	"fmt"
	"log/slog"
	"strings"

	"go_to_chat/internal/llm"
	"go_to_chat/internal/model"
)

// classify 意图分类：先走关键词匹配，命中直接返回；否则走 LLM 分类。
// 返回匹配到的 category name，如果全都没命中则返回空串。
func classify(cfg *model.ClassifierDef, userQuery string, llmClient *llm.Client) model.IntentType {
	if cfg == nil || len(cfg.Categories) == 0 {
		return ""
	}

	// 1. 关键词匹配
	if name := matchKeyword(userQuery, cfg.Categories); name != "" {
		return name
	}

	// 2. LLM 分类兜底
	if llmClient != nil {
		if name := llmClassify(cfg, userQuery, llmClient); name != "" {
			return name
		}
	}

	// 3. 最终 fallback：返回最后一个类别（通常是一般咨询类）
	if len(cfg.Categories) > 0 {
		fallback := cfg.Categories[len(cfg.Categories)-1].Name
		slog.Info("classifier fallback", "intent", fallback, "query", userQuery[:min(50, len(userQuery))])
		return fallback
	}

	return ""
}

// matchKeyword 用关键词字典做匹配，返回命中的 category name。
func matchKeyword(query string, categories []model.IntentCategory) model.IntentType {
	query = strings.ToLower(query)
	var bestMatch model.IntentType
	var bestLen int

	for _, cat := range categories {
		for _, kw := range cat.Keywords {
			if strings.Contains(query, strings.ToLower(kw)) {
				if len(cat.Keywords) > bestLen {
					bestMatch = cat.Name
					bestLen = len(cat.Keywords)
				}
				break // 命中一个 keyword 就够了，跳出内层循环
			}
		}
	}

	return bestMatch
}

// llmClassify 用 LLM 做意图分类，要求模型输出类别名。
func llmClassify(cfg *model.ClassifierDef, userQuery string, llmClient *llm.Client) model.IntentType {
	systemPrompt := buildClassifierPrompt(cfg)
	userMessage := fmt.Sprintf("用户输入：%s\n\n请输出最匹配的类别名称：", userQuery)

	result, err := llmClient.Chat(systemPrompt, userMessage)
	if err != nil {
		slog.Warn("classifier LLM call failed", "error", err)
		return ""
	}

	// 清理结果（去掉空格、引号、标点）
	name := strings.TrimSpace(result)
	name = strings.Trim(name, "\"'。，,.：: ")

	// 校验是否在已知类别列表中
	for _, cat := range cfg.Categories {
		if strings.EqualFold(name, string(cat.Name)) {
			slog.Info("classifier LLM matched", "intent", cat.Name)
			return cat.Name
		}
		// 检查类别名是否包含在 LLM 输出中（模糊匹配）
		if strings.Contains(strings.ToLower(result), strings.ToLower(string(cat.Name))) {
			slog.Info("classifier LLM fuzzy matched", "intent", cat.Name)
			return cat.Name
		}
	}

	slog.Warn("classifier LLM returned unknown category", "result", result)
	return ""
}

// buildClassifierPrompt 构建分类器的 system prompt
func buildClassifierPrompt(cfg *model.ClassifierDef) string {
	var b strings.Builder

	if cfg.Prompt != "" {
		b.WriteString(cfg.Prompt)
		b.WriteString("\n\n")
	} else {
		b.WriteString("你是一个意图分类器。根据用户输入，判断其意图属于以下哪个类别。\n")
		b.WriteString("请只输出类别名称，不要输出任何其他内容。\n\n")
	}

	b.WriteString("可选类别：\n")
	for _, cat := range cfg.Categories {
		b.WriteString(fmt.Sprintf("- %s：%s\n", cat.Name, cat.Description))
	}

	b.WriteString("\n请只输出类别名称，不要解释。")
	return b.String()
}
