package com.rd.robot.engine;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.rd.robot.client.ClientFactory;
import com.rd.robot.client.EmbeddingClient;
import com.rd.robot.client.LlmClient;
import com.rd.robot.fasttext.FastTextPredictor;
import com.rd.robot.knowledge.KnowledgeBaseManager;
import com.rd.robot.model.*;
import com.rd.robot.repository.MetaStore;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;
import java.util.concurrent.BlockingQueue;
import java.util.concurrent.LinkedBlockingQueue;

/**
 * Workflow execution engine.
 * Executes a workflow's nodes sequentially, supporting streaming output,
 * variable passing, KB retrieval, and multi-tier intent classification routing.
 */
public class WorkflowEngine {

    private static final Logger log = LoggerFactory.getLogger(WorkflowEngine.class);
    private static final ObjectMapper MAPPER = new ObjectMapper();

    private final Config cfg;
    private final KnowledgeBaseManager kbMgr;
    private final MetaStore metaStore;
    private final ClientFactory clientFactory;
    private final FastTextPredictor ftPredictor;

    public WorkflowEngine(Config cfg, KnowledgeBaseManager kbMgr, MetaStore metaStore,
                          ClientFactory clientFactory) {
        this.cfg = cfg;
        this.kbMgr = kbMgr;
        this.metaStore = metaStore;
        this.clientFactory = clientFactory;
        this.ftPredictor = new FastTextPredictor();
    }

    /** Returns the embedding client (for external debug/test use). */
    public EmbeddingClient embClient() {
        return clientFactory.getEmbeddingClient();
    }

    /** Returns the fastText predictor (for external debug/test use). */
    public FastTextPredictor ftPredictor() {
        return ftPredictor;
    }

    /**
     * Execute a workflow with streaming events.
     * Returns a BlockingQueue of events that the caller can poll or iterate.
     */
    public BlockingQueue<EngineEvent> executeStream(long workflowId, String userQuery, String uid,
                                                     List<TemplateResolver.ChatMsg> messages) {
        BlockingQueue<EngineEvent> eventQueue = new LinkedBlockingQueue<>(100);

        new Thread(() -> {
            try {
                // 1. Load workflow
                WorkflowDef workflow = metaStore.getWorkflow(workflowId);
                if (workflow == null) {
                    eventQueue.offer(new EngineEvent("error", "工作流不存在"));
                    return;
                }
                if (workflow.getNodes() == null || workflow.getNodes().isEmpty()) {
                    eventQueue.offer(new EngineEvent("error", "工作流没有节点"));
                    return;
                }

                List<WorkflowNode> nodes = workflow.getNodes();
                int total = nodes.size();
                int stepCounter = 0;

                // 2. Initialize variable pool
                String curDate = java.time.LocalDate.now().toString();
                String curWeek = TemplateResolver.getWeekdayCN();

                Map<String, String> vars = new HashMap<>();
                // New names (sys. prefix)
                vars.put("sys.user_query", userQuery);
                vars.put("sys.history", TemplateResolver.formatHistory(messages));
                vars.put("sys.cur_date", curDate);
                vars.put("sys.cur_week", curWeek);
                vars.put("sys.kb_context", ""); // filled per node by KB retrieval

                // Legacy names (backward compatibility)
                vars.put("user_query", userQuery);
                vars.put("history", TemplateResolver.formatHistory(messages));
                vars.put("cur_date", curDate);
                vars.put("cur_week", curWeek);

                // 3. Intent classification (if workflow has a classifier)
                String classifierOutputVar = "intent"; // default
                if (workflow.getClassifier() != null) {
                    // Train fastText model from category keywords
                    try {
                        ftPredictor.train(workflow.getClassifier().getCategories(),
                                workflow.getClassifier().getPrompt());
                    } catch (Exception e) {
                        log.warn("fastText train failed, will skip fastText tier: {}", e.getMessage());
                    }

                    log.info("classifier start workflow={}", workflow.getName());
                    long classifyStart = System.currentTimeMillis();

                    String intent = IntentClassifier.classify(
                            workflow.getClassifier(), userQuery,
                            clientFactory.getLlmClient(),
                            clientFactory.getEmbeddingClient(),
                            ftPredictor);

                    long classifyElapsed = System.currentTimeMillis() - classifyStart;

                    classifierOutputVar = workflow.getClassifier().getOutputVar();
                    if (classifierOutputVar == null || classifierOutputVar.isEmpty()) {
                        classifierOutputVar = "intent";
                    }
                    vars.put(classifierOutputVar, intent);
                    // sys. prefixed copy for template reference
                    vars.put("sys." + classifierOutputVar, intent);

                    eventQueue.offer(new EngineEvent("progress", 0, total, "意图分类: " + intent, ""));

                    log.info("classifier done workflow={} intent={} duration_ms={} query={}",
                            workflow.getName(), intent, classifyElapsed, truncate(userQuery, 50));
                }

                // 4. Execute nodes sequentially
                log.info("workflow nodes start workflow={} total_nodes={} classifier_result={}",
                        workflow.getName(), total, vars.get(classifierOutputVar));

                for (int i = 0; i < nodes.size(); i++) {
                    WorkflowNode node = nodes.get(i);

                    // Condition routing: skip if condition doesn't match
                    if (node.getCondition() != null && !node.getCondition().isEmpty()) {
                        String currentIntent = vars.get(classifierOutputVar);
                        if (!node.getCondition().equals(currentIntent)) {
                            log.info("skip node by condition workflow={} node={} agent_name={} condition={} current_intent={}",
                                    workflow.getName(), node.getId(), node.getAgentName(),
                                    node.getCondition(), currentIntent);
                            continue;
                        }
                    }

                    // Load agent
                    AgentDef agent = metaStore.getAgent(node.getAgentId());
                    if (agent == null) {
                        eventQueue.offer(new EngineEvent("error",
                                "节点 " + node.getId() + " 引用的智能体 (ID: " + node.getAgentId() + ") 不存在"));
                        return;
                    }

                    stepCounter++;
                    eventQueue.offer(new EngineEvent("progress", stepCounter, total, agent.getName(), ""));

                    log.info("node start workflow={} step={} total={} node={} agent={} is_final={}",
                            workflow.getName(), stepCounter, total, node.getId(), agent.getName(), node.isFinal());

                    // Render input template
                    String input = TemplateResolver.resolve(node.getInputTemplate(), vars);
                    log.info("node input ready node={} agent={} input_len={} input_preview={}",
                            node.getId(), agent.getName(),
                            input != null ? input.length() : 0,
                            truncate(input, 80));

                    // KB retrieval (if agent has vdb_ids)
                    vars.put("sys.kb_context", retrieveKbContext(agent, userQuery, uid));

                    // Build system prompt (via template resolution)
                    String systemPrompt = TemplateResolver.resolve(agent.getSystemPrompt(), vars);

                    // Select LLM client (with agent-specific params or defaults)
                    LlmClient llmClient = getLlmClient(agent);
                    log.info("llm call start node={} agent={} model={} system_prompt_len={}",
                            node.getId(), agent.getName(),
                            llmClient.getModelName(),
                            systemPrompt != null ? systemPrompt.length() : 0);

                    boolean isFinalNode = node.isFinal() || (i == nodes.size() - 1);

                    long llmStart = System.currentTimeMillis();

                    if (isFinalNode) {
                        // Final node: streaming output via LLM
                        StringBuilder fullContent = new StringBuilder();
                        int[] totalChunks = {0};

                        final int currentStep = stepCounter;
                        final String currentAgentName = agent.getName();
                        final int currentTotal = total;

                        llmClient.chatStream(systemPrompt, input,
                                chunk -> {
                                    // onChunk
                                    totalChunks[0]++;
                                    fullContent.append(chunk);
                                    eventQueue.offer(new EngineEvent("chunk", currentStep, currentTotal, currentAgentName, chunk));
                                },
                                error -> {
                                    // onError
                                    eventQueue.offer(new EngineEvent("chunk", currentStep, currentTotal, currentAgentName,
                                            "[错误] " + error));
                                },
                                () -> {
                                    // onDone
                                    long llmElapsed = System.currentTimeMillis() - llmStart;
                                    log.info("node done node={} agent={} type=stream duration_ms={} chunks={}",
                                            node.getId(), currentAgentName, llmElapsed, totalChunks[0]);
                                }
                        );
                    } else {
                        // Non-final node: synchronous call
                        try {
                            String fullOutput = llmClient.chat(systemPrompt, input);
                            long llmElapsed = System.currentTimeMillis() - llmStart;

                            if (fullOutput != null) {
                                vars.put(node.getOutputVar(), fullOutput);
                                vars.put(node.getId(), fullOutput); // Use node ID as key too
                                log.info("node done node={} agent={} type=sync duration_ms={} output_len={} output_preview={}",
                                        node.getId(), agent.getName(), llmElapsed,
                                        fullOutput.length(), truncate(fullOutput, 80));
                            } else {
                                String errorOutput = "[错误] LLM 返回空结果";
                                vars.put(node.getOutputVar(), errorOutput);
                                vars.put(node.getId(), errorOutput);
                            }
                        } catch (Exception e) {
                            log.error("node error node={} agent={} error={}", node.getId(), agent.getName(), e.getMessage());
                            String errorOutput = "[错误] " + e.getMessage();
                            vars.put(node.getOutputVar(), errorOutput);
                            vars.put(node.getId(), errorOutput);
                        }
                    }
                }

                // Send completion event
                log.info("workflow nodes done workflow={} total_nodes={}", workflow.getName(), total);
                eventQueue.offer(new EngineEvent("done", total, total, "", ""));

            } catch (Exception e) {
                log.error("workflow execution error", e);
                eventQueue.offer(new EngineEvent("error", "工作流执行失败: " + e.getMessage()));
            }
        }).start();

        return eventQueue;
    }

    /**
     * Non-streaming workflow execution.
     */
    public String execute(long workflowId, String userQuery, String uid,
                          List<TemplateResolver.ChatMsg> messages) throws Exception {
        StringBuilder result = new StringBuilder();
        String lastError = null;

        BlockingQueue<EngineEvent> events = executeStream(workflowId, userQuery, uid, messages);

        while (true) {
            EngineEvent evt = events.take();
            switch (evt.getType()) {
                case "chunk":
                    result.append(evt.getContent());
                    break;
                case "error":
                    lastError = evt.getContent();
                    if (result.isEmpty()) {
                        throw new RuntimeException(lastError);
                    }
                    break;
                case "done":
                    if (lastError != null && result.isEmpty()) {
                        throw new RuntimeException(lastError);
                    }
                    return result.toString();
            }
        }
    }

    // ============================================================
    // Internal methods
    // ============================================================

    private LlmClient getLlmClient(AgentDef agent) {
        String modelName = (agent.getModelName() != null && !agent.getModelName().isEmpty())
                ? agent.getModelName() : null;
        LlmClient client = clientFactory.createLlmClient(modelName);

        // Agent-specific overrides
        if (agent.getTemperature() != null || agent.getTopP() != null || agent.getMaxTokens() != null) {
            double temp = agent.getTemperature() != null ? agent.getTemperature() : 0.7;
            double topP = agent.getTopP() != null ? agent.getTopP() : 0.9;
            int maxTok = agent.getMaxTokens() != null ? agent.getMaxTokens() : 2048;
            client.setParams(temp, topP, maxTok);
        }
        return client;
    }

    private String retrieveKbContext(AgentDef agent, String userQuery, String uid) {
        if (agent.getVdbIds() == null || agent.getVdbIds().isEmpty() || "[]".equals(agent.getVdbIds())) {
            return "";
        }

        try {
            List<Long> vdbIds = MAPPER.readValue(agent.getVdbIds(), new TypeReference<List<Long>>() {});
            if (vdbIds.isEmpty()) return "";

            log.info("kb search start node={} agent={} vdb_ids={}", "workflow", agent.getName(), vdbIds);

            long kbStart = System.currentTimeMillis();
            StringBuilder sb = new StringBuilder();
            for (long vdbId : vdbIds) {
                String ctx = kbMgr.searchInKB(userQuery, vdbId, uid,
                        cfg.getKb().getTopK(), cfg.getKb().getScoreThreshold());
                if (ctx != null && !ctx.isEmpty()) {
                    sb.append(ctx).append("\n");
                }
            }
            long kbElapsed = System.currentTimeMillis() - kbStart;

            log.info("kb search done node={} agent={} kb_context_len={} duration_ms={}",
                    "workflow", agent.getName(), sb.length(), kbElapsed);

            return sb.toString();
        } catch (Exception e) {
            log.warn("KB retrieval failed for agent {}", agent.getName(), e);
            return "";
        }
    }

    private static String truncate(String s, int maxLen) {
        if (s == null) return "";
        return s.length() <= maxLen ? s : s.substring(0, maxLen);
    }
}
