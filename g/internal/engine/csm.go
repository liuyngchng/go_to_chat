package engine

import (
	"log/slog"
	"strings"
	"time"

	"kb-chat-flow/internal/model"
)

// ============================================================
// csm.go — 硬编码客服问答逻辑（CSM = Customer Service Module）
// ============================================================
//
// 背景：动态工作流配置（cfg.db workflow_def）已能满足复杂编排，但日常
// 业务上配置成本高。本文件把"燃气客服"这一条问答逻辑用代码写死，
// 作为简单快速的业务实现，绕过数据库中的工作流配置。
//
// 逻辑与 cfg.db 中"燃气客服工作流"(workflow 1) 一致：
//   意图分类(emergency/billing/business/repair/faq)
//   → 按意图路由 → (紧急/业务直接回答，账单/维修/FAQ 先检索知识库)
//   → LLM 流式生成最终回答
//
// 动态配置逻辑（engine.go / classifier.go / template.go）不受影响。
//
// 关键节点日志（检索 key 便于链路追踪）：
//   csm_run_start        流程入口（uid / query）
//   csm_classify_done    意图分类完成（intent / 耗时）
//   csm_route            路由结果（intent → branch）
//   csm_kb_search_start  开始检索知识库（vdb_ids）
//   csm_kb_search_done   检索完成（耗时 / 上下文长度）
//   csm_kb_search_failed 单个知识库检索失败
//   csm_llm_start        开始请求 LLM（agent / model / 输入长度）
//   csm_llm_done         LLM 请求完成（耗时 / chunk 数 / 输出长度）
//   csm_llm_error        LLM 请求失败
//   csm_run_done         整条流程结束（总耗时）

// csmTotalStep 硬编码流程的总步骤数（0=意图分类, 1=检索, 2=回答）
// 供前端进度展示（EngineEvent.Total）使用
const csmTotalStep = 3

// csmClassifier 硬编码的意图分类器配置。
// 与 cfg.db workflow 1 的 classifier 完全一致。
var csmClassifier = &model.ClassifierDef{
	OutputVar: "intent",
	Prompt:    "你是一个燃气公司客服意图分类器。根据用户输入，判断其意图属于以下哪个类别。\n请只输出类别名称，不要输出任何其他内容。",
	Categories: []model.IntentCategory{
		{Name: model.IntentEmergency, Description: "燃气泄漏、燃气味、报警等紧急安全情况", Keywords: []string{"漏气", "燃气味", "煤气味", "报警", "爆炸", "火灾", "着火", "泄漏", "冒烟", "异味", "刺鼻"}},
		{Name: model.IntentBilling, Description: "账单查询、缴费、欠费、发票等财务问题", Keywords: []string{"账单", "缴费", "欠费", "余额", "发票", "价格", "费用", "多少钱", "扣费", "充值", "代扣", "阶梯价"}},
		{Name: model.IntentBusiness, Description: "开户、过户、改名、报装、停气等业务办理", Keywords: []string{"开户", "过户", "改名", "报装", "停气", "新装", "移表", "增容", "改管", "安装", "开通", "搬迁", "换表"}},
		{Name: model.IntentRepair, Description: "燃气设备维修、故障排查、保养、安检", Keywords: []string{"维修", "故障", "坏了", "打不着火", "点不着", "不着火", "保养", "安检", "检查", "熄火", "红火", "小火", "自动关", "打火"}},
		{Name: model.IntentFaq, Description: "常见综合咨询：营业时间、电话、地址、投诉建议等", Keywords: []string{"营业时间", "电话", "地址", "投诉", "建议", "表扬", "几点", "在哪", "怎么去", "客服", "人工", "工作时间"}},
	},
}

// csmVdbIDs 硬编码的知识库 ID 列表（账单/维修/FAQ 检索时使用）。
// 与 cfg.db 中 agent_def 各检索智能体绑定的 vdb_ids 一致：
//
//	vdb 3 = new_test（燃气客服知识汇总，admin 上传）
var csmVdbIDs = []int64{3}

// ============================================================
// 各意图智能体系统提示词（与 cfg.db agent_def 表内容一致）
// ============================================================

// csmEmergencyPrompt 紧急调度：不检索，直接回答。
const csmEmergencyPrompt = `你是燃气公司紧急调度员。用户遇到了紧急情况，你必须优先处理。
请引导用户立即采取安全措施：关闭燃气阀门、开窗通风、禁止明火、撤离现场，
同时告知用户已安排紧急维修人员尽快到达。
语气要冷静、专业，给用户安全感。`

// csmBillingPrompt 账单客服：基于检索结果回答。
const csmBillingPrompt = `你是燃气公司账单客服。根据检索到的账单信息，帮助用户解决账单查询、缴费方式、欠费处理等问题。
用亲切专业的中文回答，引导用户完成缴费操作。`

// csmBusinessPrompt 业务办理：不检索，直接回答。
const csmBusinessPrompt = `你是燃气公司业务办理专员。帮助用户办理开户、过户、改名、报装、停气等业务。
请告知用户所需材料、办理流程和注意事项。
语气亲切、专业，一步步引导用户完成业务办理。`

// csmRepairPrompt 维修客服：基于检索结果回答。
const csmRepairPrompt = `你是燃气公司维修客服。根据检索到的维修信息，帮助用户进行故障诊断、保养指导、报修登记。
对于简单故障给出排查建议，无法解决的安排维修人员上门。
语气专业、耐心。`

// csmFaqPrompt 综合FAQ：基于检索结果回答。
const csmFaqPrompt = `你是燃气公司综合客服。根据检索到的FAQ信息，回答用户的各种常见问题，
如营业时间、服务电话、地址、投诉渠道等。
语气亲切、专业，解答清晰明了。`

// ExecuteStreamCSM 硬编码客服问答的流式执行入口。
//
// 签名与 ExecuteStream 保持一致，便于 handler 侧直接替换：
//
//	eventCh := h.engine.ExecuteStreamCSM(req.WorkflowID, req.Msg, uid, historyMsgs)
//
// workflowID 已不再用于加载数据库配置，仅保留参数以维持调用兼容。
// 事件协议完全复用 EngineEvent（progress / chunk / done / error），
// handler 的 SSE 循环无需任何改动。
func (e *Engine) ExecuteStreamCSM(
	workflowID int64,
	userQuery string,
	uid string,
	messages []ChatMsg,
) <-chan EngineEvent {
	eventCh := make(chan EngineEvent, 50)

	go func() {
		defer close(eventCh)
		e.csmRun(eventCh, userQuery, uid, len(messages))
	}()

	return eventCh
}

// csmRun 硬编码流程主逻辑。
func (e *Engine) csmRun(eventCh chan<- EngineEvent, userQuery, uid string, historyCount int) {
	runStart := time.Now()
	slog.Info("csm_run_start", "uid", uid, "query", truncateStr(userQuery, 80), "query_len", len(userQuery), "history", historyCount)

	// 1. 意图分类（复用 engine.classify，多级匹配：关键词 → fastText → 语义 → LLM → fallback）
	if err := e.ftPredictor.Train(csmClassifier.Categories, csmClassifier.Prompt); err != nil {
		slog.Warn("fastText train failed, will skip fastText tier", "error", err)
	}
	classifyStart := time.Now()
	intent := classify(csmClassifier, userQuery, e.baseLLM, e.embClient, e.ftPredictor)
	if intent == "" {
		// 理论上 classify 至少会 fallback 到最后一个类别；此处兜底防御
		intent = model.IntentFaq
	}
	slog.Info("csm_classify_done", "intent", string(intent), "duration_ms", time.Since(classifyStart).Milliseconds(), "query", truncateStr(userQuery, 80))
	eventCh <- EngineEvent{Type: "progress", Step: 0, Total: csmTotalStep, Agent: "意图分类: " + string(intent)}

	// 2. 按意图路由
	branch := csmBranchName(intent)
	slog.Info("csm_route", "intent", string(intent), "branch", branch)

	switch intent {
	case model.IntentEmergency:
		e.csmAnswerDirect(eventCh, "紧急调度", csmEmergencyPrompt, userQuery)
	case model.IntentBilling:
		e.csmAnswerWithKB(eventCh, "账单检索", "账单客服", csmBillingPrompt, userQuery, uid)
	case model.IntentBusiness:
		e.csmAnswerDirect(eventCh, "业务办理", csmBusinessPrompt, userQuery)
	case model.IntentRepair:
		e.csmAnswerWithKB(eventCh, "维修检索", "维修客服", csmRepairPrompt, userQuery, uid)
	default: // faq / 未识别
		e.csmAnswerWithKB(eventCh, "FAQ检索", "综合FAQ", csmFaqPrompt, userQuery, uid)
	}

	// 3. 完成
	eventCh <- EngineEvent{Type: "done", Total: csmTotalStep}
	slog.Info("csm_run_done", "intent", string(intent), "total_ms", time.Since(runStart).Milliseconds())
}

// csmBranchName 返回意图对应的路由分支描述（仅用于日志）。
func csmBranchName(intent model.IntentType) string {
	switch intent {
	case model.IntentEmergency:
		return "emergency -> 紧急调度（直接回答）"
	case model.IntentBilling:
		return "billing -> 账单检索 + 账单客服"
	case model.IntentBusiness:
		return "business -> 业务办理（直接回答）"
	case model.IntentRepair:
		return "repair -> 维修检索 + 维修客服"
	default:
		return "faq -> FAQ检索 + 综合FAQ"
	}
}

// csmAnswerDirect 直接回答（不检索知识库），用于紧急调度 / 业务办理。
func (e *Engine) csmAnswerDirect(eventCh chan<- EngineEvent, agentName, systemPrompt, userQuery string) {
	eventCh <- EngineEvent{Type: "progress", Step: 2, Total: csmTotalStep, Agent: agentName}
	e.csmStream(eventCh, agentName, systemPrompt, userQuery)
}

// csmAnswerWithKB 先检索知识库，再基于检索结果回答，用于账单 / 维修 / FAQ。
// 检索步骤与回答步骤分别发送 progress，与动态工作流的节点展示一致。
func (e *Engine) csmAnswerWithKB(eventCh chan<- EngineEvent, retrieveAgent, answerAgent, systemPrompt, userQuery, uid string) {
	eventCh <- EngineEvent{Type: "progress", Step: 1, Total: csmTotalStep, Agent: retrieveAgent}

	kbContext := e.csmSearchKB(userQuery, uid)

	// 与 workflow 节点 InputTemplate "用户问题：{{user_query}}\n检索信息：{{xx_ctx}}" 保持一致
	userMessage := "用户问题：" + userQuery + "\n检索信息：" + kbContext

	eventCh <- EngineEvent{Type: "progress", Step: 2, Total: csmTotalStep, Agent: answerAgent}
	e.csmStream(eventCh, answerAgent, systemPrompt, userMessage)
}

// csmSearchKB 在硬编码的知识库列表中检索用户问题，拼接上下文。
func (e *Engine) csmSearchKB(userQuery, uid string) string {
	start := time.Now()
	slog.Info("csm_kb_search_start", "vdb_ids", csmVdbIDs, "query", truncateStr(userQuery, 80))

	var sb strings.Builder
	for _, vdbID := range csmVdbIDs {
		ctx, err := e.kbMgr.SearchInKB(userQuery, vdbID, uid, e.cfg.KB.TopK, e.cfg.KB.ScoreThreshold)
		if err != nil {
			slog.Warn("csm_kb_search_failed", "vdb_id", vdbID, "error", err)
			continue
		}
		if ctx != "" {
			sb.WriteString(ctx)
			sb.WriteString("\n")
		}
	}
	slog.Info("csm_kb_search_done", "duration_ms", time.Since(start).Milliseconds(), "context_len", len(sb.String()))
	return sb.String()
}

// csmStream 流式调用 LLM，将输出以 chunk 事件逐段发出。
func (e *Engine) csmStream(eventCh chan<- EngineEvent, agentName, systemPrompt, userMessage string) {
	start := time.Now()
	modelName := e.cfg.API.LLMModelName
	if e.baseLLM != nil && e.baseLLM.ModelName != "" {
		modelName = e.baseLLM.ModelName
	}
	slog.Info("csm_llm_start", "agent", agentName, "model", modelName, "prompt_len", len(systemPrompt), "input_len", len(userMessage))

	chunkCh, errCh := e.baseLLM.ChatStream(systemPrompt, userMessage)

	var output strings.Builder
	chunkCount := 0
	for chunk := range chunkCh {
		output.WriteString(chunk)
		chunkCount++
		eventCh <- EngineEvent{Type: "chunk", Content: chunk, Step: 2, Total: csmTotalStep, Agent: agentName}
	}

	// 检查错误（errCh 为缓冲通道，range 结束后读它不会阻塞）
	err := <-errCh
	if err != nil {
		slog.Error("csm_llm_error", "agent", agentName, "error", err, "duration_ms", time.Since(start).Milliseconds(), "chunks", chunkCount, "output_len", output.Len())
		eventCh <- EngineEvent{Type: "error", Content: "[错误] " + err.Error(), Error: err}
		return
	}

	slog.Info("csm_llm_done", "agent", agentName, "duration_ms", time.Since(start).Milliseconds(), "chunks", chunkCount, "output_len", output.Len(), "output_preview", truncateStr(output.String(), 80))
}

// truncateStr 截断字符串用于日志预览（按 rune 截断避免切坏中文）。
func truncateStr(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}
