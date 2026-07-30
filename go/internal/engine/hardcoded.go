package engine

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"go_to_chat/internal/model"
)

// hardcodedStep 硬编码工作流中的一个步骤
type hardcodedStep struct {
	AgentID   int64  // 对应 agent_def 表中的 ID
	InputTmpl string // 输入模板，用 {{var}} 引用上游变量
	OutputVar string // 输出变量名（空表示不存变量池）
	IsFinal   bool   // 是否最终输出节点（流式输出）
}

// hardcodedClassifier 硬编码的意图分类器配置
var hardcodedClassifier = &model.ClassifierDef{
	OutputVar: "intent",
	Categories: []model.IntentCategory{
		{
			Name:        model.IntentEmergency,
			Description: "燃气泄漏、燃气味、报警等紧急安全情况",
			Keywords:    []string{"漏气", "燃气味", "煤气味", "报警", "爆炸", "火灾", "着火", "泄漏", "冒烟", "异味", "刺鼻"},
		},
		{
			Name:        model.IntentBilling,
			Description: "账单查询、缴费、欠费、发票等财务问题",
			Keywords:    []string{"账单", "缴费", "欠费", "余额", "发票", "价格", "费用", "多少钱", "扣费", "充值", "代扣", "阶梯价"},
		},
		{
			Name:        model.IntentBusiness,
			Description: "开户、过户、改名、报装、停气等业务办理",
			Keywords:    []string{"开户", "过户", "改名", "报装", "停气", "新装", "移表", "增容", "改管", "安装", "开通", "搬迁", "换表"},
		},
		{
			Name:        model.IntentRepair,
			Description: "燃气设备维修、故障排查、保养、安检",
			Keywords:    []string{"维修", "故障", "坏了", "打不着火", "点不着", "不着火", "保养", "安检", "检查", "熄火", "红火", "小火", "自动关", "打火"},
		},
		{
			Name:        model.IntentFaq,
			Description: "常见综合咨询：营业时间、电话、地址、投诉建议等",
			Keywords:    []string{"营业时间", "电话", "地址", "投诉", "建议", "表扬", "几点", "在哪", "怎么去", "客服", "人工", "工作时间"},
		},
	},
}

// hardcodedFlows 硬编码工作流路由：每个 intent 对应一组执行步骤
//
// 如需修改工作流逻辑：
//  1. 在页面上新增/编辑 AI Agent（agent_def 表），获得 agent ID
//  2. 在此 map 中引用该 ID
//  3. 如需新增分类，在 model/types.go 的 IntentType 枚举中增加常量
//
//nolint:gochecknoglobals
var hardcodedFlows = map[model.IntentType][]hardcodedStep{
	// 紧急：一步走完，直接输出
	model.IntentEmergency: {
		{AgentID: 2, InputTmpl: "{{user_query}}", IsFinal: true},
	},
	// 账单：先检索知识库，再输出
	model.IntentBilling: {
		{AgentID: 3, InputTmpl: "{{user_query}}", OutputVar: "bill_ctx"},
		{AgentID: 4, InputTmpl: "用户问题：{{user_query}}\n检索信息：{{bill_ctx}}", IsFinal: true},
	},
	// 业务办理：一步走完，直接输出
	model.IntentBusiness: {
		{AgentID: 5, InputTmpl: "{{user_query}}", IsFinal: true},
	},
	// 维修：先检索知识库，再输出
	model.IntentRepair: {
		{AgentID: 6, InputTmpl: "{{user_query}}", OutputVar: "rep_ctx"},
		{AgentID: 7, InputTmpl: "用户问题：{{user_query}}\n检索信息：{{rep_ctx}}", IsFinal: true},
	},
	// 综合咨询：先检索知识库，再输出
	model.IntentFaq: {
		{AgentID: 8, InputTmpl: "{{user_query}}", OutputVar: "faq_ctx"},
		{AgentID: 9, InputTmpl: "用户问题：{{user_query}}\n检索信息：{{faq_ctx}}", IsFinal: true},
	},
}

// HardcodedWorkflow 执行硬编码工作流，返回事件通道。
// 事件类型与 ExecuteStream 兼容：
//   - "progress" — 进度通知
//   - "chunk"     — 流式输出片段
//   - "done"      — 执行完成
//   - "error"     — 错误
func (e *Engine) HardcodedWorkflow(
	userQuery string,
	uid string,
	messages []ChatMsg,
) <-chan EngineEvent {
	eventCh := make(chan EngineEvent, 50)

	go func() {
		defer close(eventCh)

		// 1. 意图分类
		intent := classify(hardcodedClassifier, userQuery, e.baseLLM)
		if intent == "" {
			eventCh <- EngineEvent{
				Type:    "error",
				Content: "无法识别用户意图",
			}
			return
		}

		eventCh <- EngineEvent{
			Type:  "progress",
			Step:  0,
			Total: 1,
			Agent: "意图分类: " + string(intent),
		}
		slog.Info("hardcoded workflow classify", "intent", intent, "query", userQuery[:min(50, len(userQuery))])

		// 2. 查找该 intent 对应的步骤链
		steps, ok := hardcodedFlows[intent]
		if !ok || len(steps) == 0 {
			eventCh <- EngineEvent{
				Type:    "error",
				Content: fmt.Sprintf("意图 %s 没有配置工作流步骤", intent),
			}
			return
		}

		// 3. 初始化变量池
		vars := map[string]string{
			"user_query": userQuery,
			"history":    FormatHistory(messages),
		}

		// 4. 顺序执行每个步骤
		for i, step := range steps {
			// 加载 Agent（从页面配置的 agent_def 表加载）
			agent, err := e.store.GetAgent(step.AgentID)
			if err != nil || agent == nil {
				eventCh <- EngineEvent{
					Type:    "error",
					Content: fmt.Sprintf("智能体 (ID: %d) 不存在或加载失败，请在页面中创建", step.AgentID),
				}
				return
			}

			// 发送进度事件
			eventCh <- EngineEvent{
				Type:  "progress",
				Step:  i + 1,
				Total: len(steps),
				Agent: agent.Name,
			}
			slog.Info("hardcoded workflow step", "agent", agent.Name, "step", i+1)

			// 渲染输入模板
			input := ResolveTemplate(step.InputTmpl, vars)

			// 知识库检索（如果 agent 绑定了 vdb_ids）
			kbContext := ""
			if agent.VdbIDs != "" && agent.VdbIDs != "[]" {
				var vdbIDs []int64
				if err := json.Unmarshal([]byte(agent.VdbIDs), &vdbIDs); err == nil && len(vdbIDs) > 0 {
					for _, vdbID := range vdbIDs {
						ctx, err := e.kbMgr.SearchInKB(userQuery, vdbID, uid, e.cfg.KB.TopK, e.cfg.KB.ScoreThreshold)
						if err == nil && ctx != "" {
							kbContext += ctx + "\n"
						}
					}
				}
			}

			// 构建 system prompt
			systemPrompt := buildSystemPrompt(agent.SystemPrompt, kbContext)

			// 选择 LLM 客户端（使用 Agent 特定的参数，或默认）
			llmClient := e.getLLMClient(agent)

			if step.IsFinal {
				// 最终节点：流式输出
				chunkCh, errCh := llmClient.ChatStream(systemPrompt, input)

				for chunk := range chunkCh {
					eventCh <- EngineEvent{
						Type:    "chunk",
						Step:    i + 1,
						Total:   len(steps),
						Agent:   agent.Name,
						Content: chunk,
					}
				}

				// 检查流式错误
				select {
				case err := <-errCh:
					if err != nil {
						eventCh <- EngineEvent{
							Type:    "chunk",
							Step:    i + 1,
							Total:   len(steps),
							Agent:   agent.Name,
							Content: fmt.Sprintf("[错误] %v", err),
						}
					}
				default:
				}
			} else {
				// 非最终节点：同步调用，输出存变量池
				fullOutput, err := llmClient.Chat(systemPrompt, input)
				if err != nil {
					slog.Warn("hardcoded workflow node error", "agent", agent.Name, "error", err)
					fullOutput = fmt.Sprintf("[错误] %v", err)
				}
				vars[step.OutputVar] = fullOutput
				vars[agent.Name] = fullOutput // 双 key，方便引用
			}
		}

		// 发送完成事件
		eventCh <- EngineEvent{Type: "done"}
	}()

	return eventCh
}