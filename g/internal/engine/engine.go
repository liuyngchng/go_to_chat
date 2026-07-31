package engine

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"kb-chat-flow/internal/kb"
	"kb-chat-flow/internal/llm"
	"kb-chat-flow/internal/model"
	"kb-chat-flow/internal/store"
)

// EngineEvent 工作流执行事件
type EngineEvent struct {
	Type    string `json:"type"` // "progress" | "chunk" | "done" | "error"
	Step    int    `json:"step"`
	Total   int    `json:"total"`
	Agent   string `json:"agent"`
	Content string `json:"content"` // chunk 或 error 内容
	Error   error  `json:"-"`       // 内部使用
}

// Engine 工作流执行引擎
type Engine struct {
	cfg     *model.Config
	kbMgr   *kb.Manager
	store   store.MetaStore
	baseLLM *llm.Client
}

// NewEngine 创建引擎
func NewEngine(cfg *model.Config, kbMgr *kb.Manager, metaStore store.MetaStore) *Engine {
	llmClient := llm.New(
		cfg.API.LLMAPIURI,
		cfg.API.LLMAPIKey,
		cfg.API.LLMModelName,
	)
	llmClient.SetParams(cfg.LLM.Temperature, cfg.LLM.TopP, cfg.LLM.MaxTokens)

	return &Engine{
		cfg:     cfg,
		kbMgr:   kbMgr,
		store:   metaStore,
		baseLLM: llmClient,
	}
}

// ExecuteStream 执行工作流，返回事件通道。
// 非最终节点：同步执行，发 progress 事件。
// 最终节点：流式执行，发 progress + chunk 事件。
// 最后发 done 事件。
func (e *Engine) ExecuteStream(
	workflowID int64,
	userQuery string,
	uid string,
	messages []ChatMsg,
) <-chan EngineEvent {
	eventCh := make(chan EngineEvent, 50)

	go func() {
		defer close(eventCh)

		// 1. 加载工作流
		workflow, err := e.store.GetWorkflow(workflowID)
		if err != nil {
			slog.Error("load workflow failed", "workflow_id", workflowID, "error", err)
			eventCh <- EngineEvent{Type: "error", Content: "加载工作流失败: " + err.Error(), Error: err}
			return
		}
		if workflow == nil {
			slog.Error("workflow not found", "workflow_id", workflowID)
			eventCh <- EngineEvent{Type: "error", Content: "工作流不存在", Error: fmt.Errorf("workflow %d not found", workflowID)}
			return
		}
		if len(workflow.Nodes) == 0 {
			slog.Error("workflow has no nodes", "workflow", workflow.Name)
			eventCh <- EngineEvent{Type: "error", Content: "工作流没有节点", Error: fmt.Errorf("empty workflow")}
			return
		}

		slog.Info("workflow loaded", "workflow", workflow.Name, "id", workflowID, "nodes", len(workflow.Nodes))

		// 2. 排序节点（按 OrderIndex）
		nodes := workflow.Nodes
		total := len(nodes)

		// 3. 初始化变量池
		curDate := time.Now().Format("2006-01-02")
		curWeek := getWeekdayCN(time.Now().Weekday())
		vars := map[string]string{
			// 新版命名（sys. 前缀）
			"sys.user_query": userQuery,
			"sys.history":    FormatHistory(messages),
			"sys.cur_date":   curDate,
			"sys.cur_week":   curWeek,
			"sys.kb_context": "", // 知识库检索结果，由节点执行时填充
			// 兼容旧版变量名
			"user_query": userQuery,
			"history":    FormatHistory(messages),
			"cur_date":   curDate,
			"cur_week":   curWeek,
		}
		classifierOutputVar := "intent" // 默认变量名，分类器可能覆盖

		// 4. 意图分类（如果工作流配置了 Classifier）
		if workflow.Classifier != nil {
			slog.Info("classifier start", "workflow", workflow.Name)
			classifyStart := time.Now()
			intent := classify(workflow.Classifier, userQuery, e.baseLLM)
			classifyElapsed := time.Since(classifyStart)

			classifierOutputVar = workflow.Classifier.OutputVar
			if classifierOutputVar == "" {
				classifierOutputVar = "intent"
			}
			vars[classifierOutputVar] = string(intent)
			// sys. 前缀版本（供模板引用）
			vars["sys."+classifierOutputVar] = string(intent)

			eventCh <- EngineEvent{
				Type:  "progress",
				Step:  0,
				Total: total,
				Agent: "意图分类: " + string(intent),
			}

			slog.Info("classifier done", "workflow", workflow.Name, "intent", intent, "duration_ms", classifyElapsed.Milliseconds(), "query", userQuery[:min(50, len(userQuery))])
		}

		// 5. 顺序执行每个节点（跳过 Condition 不匹配的）
		slog.Info("workflow nodes start", "workflow", workflow.Name, "total_nodes", total, "classifier_result", vars[classifierOutputVar])
		for i, node := range nodes {
			// 条件路由：有 Condition 但不匹配 → 跳过
			if node.Condition != "" {
				if model.IntentType(vars[classifierOutputVar]) != node.Condition {
					slog.Info("skip node by condition", "workflow", workflow.Name, "node", node.ID, "agent_name", node.AgentName, "condition", node.Condition, "current_intent", vars[classifierOutputVar])
					continue
				}
			}

			// 加载 Agent
			agent, err := e.store.GetAgent(node.AgentID)
			if err != nil || agent == nil {
				slog.Error("agent not found", "node", node.ID, "agent_id", node.AgentID, "error", err)
				eventCh <- EngineEvent{
					Type:    "error",
					Content: fmt.Sprintf("节点 %s 引用的智能体 (ID: %d) 不存在", node.ID, node.AgentID),
					Error:   fmt.Errorf("agent %d not found", node.AgentID),
				}
				return
			}

			// 发送进度事件
			eventCh <- EngineEvent{
				Type:  "progress",
				Step:  i + 1,
				Total: total,
				Agent: agent.Name,
			}

			slog.Info("node start", "workflow", workflow.Name, "step", i+1, "total", total, "node", node.ID, "agent", agent.Name, "is_final", node.IsFinal)

			// 渲染输入模板
			input := ResolveTemplate(node.InputTemplate, vars)
			slog.Info("node input ready", "node", node.ID, "agent", agent.Name, "input_len", len(input), "input_preview", input[:min(80, len(input))])

			// 知识库检索（如果 agent 绑定了 vdb_ids）
			if agent.VdbIDs != "" && agent.VdbIDs != "[]" {
				var vdbIDs []int64
				if err := json.Unmarshal([]byte(agent.VdbIDs), &vdbIDs); err == nil && len(vdbIDs) > 0 {
					slog.Info("kb search start", "node", node.ID, "agent", agent.Name, "vdb_ids", vdbIDs)
					kbStart := time.Now()
					var kbContext strings.Builder
					for _, vdbID := range vdbIDs {
						ctx, err := e.kbMgr.SearchInKB(userQuery, vdbID, uid, e.cfg.KB.TopK, e.cfg.KB.ScoreThreshold)
						if err == nil && ctx != "" {
							kbContext.WriteString(ctx)
							kbContext.WriteString("\n")
						}
					}
					vars["sys.kb_context"] = kbContext.String()
					kbElapsed := time.Since(kbStart)
					slog.Info("kb search done", "node", node.ID, "agent", agent.Name, "kb_context_len", len(vars["sys.kb_context"]), "duration_ms", kbElapsed.Milliseconds())
				}
			}

			// 构建 system prompt（走模板渲染，支持 {{sys.xxx}} 变量）
			systemPrompt := ResolveTemplate(agent.SystemPrompt, vars)

			// 选择 LLM 客户端（使用 Agent 特定的参数，或默认）
			llmClient := e.getLLMClient(agent)
			slog.Info("llm call start", "node", node.ID, "agent", agent.Name, "model", llmClient.ModelName, "system_prompt_len", len(systemPrompt))

			llmStart := time.Now()

			if node.IsFinal || i == total-1 {
				// 最终节点：流式输出
				chunkCh, errCh := llmClient.ChatStream(systemPrompt, input)

				totalChunks := 0
				for chunk := range chunkCh {
					totalChunks++
					eventCh <- EngineEvent{
						Type:    "chunk",
						Step:    i + 1,
						Total:   total,
						Agent:   agent.Name,
						Content: chunk,
					}
				}

				// 检查流式错误
				select {
				case err := <-errCh:
					if err != nil {
						slog.Error("node stream error", "node", node.ID, "agent", agent.Name, "error", err)
						eventCh <- EngineEvent{
							Type:    "chunk",
							Step:    i + 1,
							Total:   total,
							Agent:   agent.Name,
							Content: fmt.Sprintf("[错误] %v", err),
						}
					}
				default:
				}

				llmElapsed := time.Since(llmStart)
				slog.Info("node done", "node", node.ID, "agent", agent.Name, "type", "stream", "duration_ms", llmElapsed.Milliseconds(), "chunks", totalChunks)
			} else {
				// 非最终节点：同步调用
				fullOutput, err := llmClient.Chat(systemPrompt, input)
				llmElapsed := time.Since(llmStart)

				if err != nil {
					slog.Error("node error", "node", node.ID, "agent", agent.Name, "error", err, "duration_ms", llmElapsed.Milliseconds())
					fullOutput = fmt.Sprintf("[错误] %v", err)
				} else {
					outputPreview := fullOutput
					if len(outputPreview) > 80 {
						outputPreview = outputPreview[:80]
					}
					slog.Info("node done", "node", node.ID, "agent", agent.Name, "type", "sync", "duration_ms", llmElapsed.Milliseconds(), "output_len", len(fullOutput), "output_preview", outputPreview)
				}

				// 存储输出到变量池（供下游引用）
				vars[node.OutputVar] = fullOutput
				// 也用 node.ID 做 key（双 key，灵活引用）
				vars[node.ID] = fullOutput
			}
		}

		// 发送完成事件
		slog.Info("workflow nodes done", "workflow", workflow.Name, "total_nodes", total)
		eventCh <- EngineEvent{Type: "done", Total: total}
	}()

	return eventCh
}

// getLLMClient 获取 LLM 客户端（使用 Agent 特定参数或默认）
func (e *Engine) getLLMClient(agent *model.AgentDef) *llm.Client {
	modelName := e.cfg.API.LLMModelName
	apiURI := e.cfg.API.LLMAPIURI
	apiKey := e.cfg.API.LLMAPIKey

	if agent.ModelName != "" {
		modelName = agent.ModelName
	}

	client := llm.New(apiURI, apiKey, modelName)

	// 使用 Agent 特定参数或全局默认
	temp := e.cfg.LLM.Temperature
	topP := e.cfg.LLM.TopP
	maxTok := e.cfg.LLM.MaxTokens

	if agent.Temperature != nil {
		temp = *agent.Temperature
	}
	if agent.TopP != nil {
		topP = *agent.TopP
	}
	if agent.MaxTokens != nil {
		maxTok = *agent.MaxTokens
	}

	client.SetParams(temp, topP, maxTok)
	return client
}

// Execute 非流式执行工作流，返回最终结果
func (e *Engine) Execute(
	workflowID int64,
	userQuery string,
	uid string,
	messages []ChatMsg,
) (string, error) {
	var result strings.Builder
	var lastErr error

	for evt := range e.ExecuteStream(workflowID, userQuery, uid, messages) {
		switch evt.Type {
		case "chunk":
			result.WriteString(evt.Content)
		case "error":
			lastErr = evt.Error
			if result.Len() == 0 {
				return "", evt.Error
			}
		case "done":
			return result.String(), lastErr
		}
	}

	return result.String(), lastErr
}
