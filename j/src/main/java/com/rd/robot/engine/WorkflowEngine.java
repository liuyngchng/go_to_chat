package com.rd.robot.engine;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.rd.robot.client.ClientFactory;
import com.rd.robot.client.LlmClient;
import com.rd.robot.knowledge.KnowledgeBaseManager;
import com.rd.robot.model.*;
import com.rd.robot.repository.MetaStore;
import com.rd.robot.vector.VectorStore;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;
import java.util.concurrent.BlockingQueue;
import java.util.concurrent.LinkedBlockingQueue;
import java.util.function.Consumer;
import java.util.stream.Collectors;

/**
 * Workflow execution engine.
 * Executes a workflow's nodes sequentially, supporting streaming output,
 * variable passing, KB retrieval, and intent classification routing.
 */
public class WorkflowEngine {

    private static final Logger log = LoggerFactory.getLogger(WorkflowEngine.class);
    private static final ObjectMapper MAPPER = new ObjectMapper();

    private final Config cfg;
    private final KnowledgeBaseManager kbMgr;
    private final MetaStore metaStore;
    private final ClientFactory clientFactory;

    public WorkflowEngine(Config cfg, KnowledgeBaseManager kbMgr, MetaStore metaStore,
                          ClientFactory clientFactory) {
        this.cfg = cfg;
        this.kbMgr = kbMgr;
        this.metaStore = metaStore;
        this.clientFactory = clientFactory;
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
                Map<String, String> vars = new HashMap<>();
                vars.put("user_query", userQuery);
                vars.put("history", TemplateResolver.formatHistory(messages));

                // 3. Intent classification (if workflow has a classifier)
                if (workflow.getClassifier() != null) {
                    String intent = IntentClassifier.classify(workflow.getClassifier(), userQuery, clientFactory.getLlmClient());
                    String outputVar = workflow.getClassifier().getOutputVar();
                    if (outputVar == null || outputVar.isEmpty()) {
                        outputVar = "intent";
                    }
                    vars.put(outputVar, intent);

                    eventQueue.offer(new EngineEvent("progress", 0, total, "意图分类: " + intent, ""));

                    log.info("workflow classify workflow={} intent={} query={}",
                            workflow.getName(), intent, truncate(userQuery, 50));
                }

                // 4. Execute nodes sequentially
                for (int i = 0; i < nodes.size(); i++) {
                    WorkflowNode node = nodes.get(i);

                    // Condition routing: skip if condition doesn't match
                    if (node.getCondition() != null && !node.getCondition().isEmpty()
                            && !node.getCondition().equals(vars.get("intent"))) {
                        continue;
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

                    log.info("workflow step workflow={} step={} agent={}",
                            workflow.getName(), stepCounter, agent.getName());

                    // Render input template
                    String input = TemplateResolver.resolve(node.getInputTemplate(), vars);

                    // KB retrieval (if agent has vdb_ids)
                    String kbContext = retrieveKbContext(agent, userQuery, uid);

                    // Build system prompt
                    String systemPrompt = buildSystemPrompt(agent.getSystemPrompt(), kbContext);

                    // Select LLM client (with agent-specific params or defaults)
                    LlmClient llmClient = getLlmClient(agent);

                    boolean isFinalNode = node.isFinal() || (i == nodes.size() - 1);

                    if (isFinalNode) {
                        // Final node: streaming output via LLM
                        StringBuilder fullContent = new StringBuilder();

                        final int currentStep = stepCounter;
                        final String currentAgentName = agent.getName();
                        final int currentTotal = total;

                        llmClient.chatStream(systemPrompt, input,
                                chunk -> {
                                    // onChunk
                                    fullContent.append(chunk);
                                    eventQueue.offer(new EngineEvent("chunk", currentStep, currentTotal, currentAgentName, chunk));
                                },
                                error -> {
                                    // onError
                                    eventQueue.offer(new EngineEvent("chunk", currentStep, currentTotal, currentAgentName,
                                            "[错误] " + error));
                                },
                                () -> {
                                    // onDone - nothing special needed
                                }
                        );
                    } else {
                        // Non-final node: synchronous call
                        try {
                            String fullOutput = llmClient.chat(systemPrompt, input);
                            vars.put(node.getOutputVar(), fullOutput);
                            vars.put(node.getId(), fullOutput); // Use node ID as key too
                        } catch (Exception e) {
                            log.warn("workflow node error node={} agent={} error={}", node.getId(), agent.getName(), e.getMessage());
                            String errorOutput = "[错误] " + e.getMessage();
                            vars.put(node.getOutputVar(), errorOutput);
                            vars.put(node.getId(), errorOutput);
                        }
                    }
                }

                // Send completion event
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
        String modelName = agent.getModelName() != null && !agent.getModelName().isEmpty()
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

    private String buildSystemPrompt(String systemPrompt, String kbContext) {
        if (kbContext == null || kbContext.isEmpty()) {
            return systemPrompt;
        }
        return systemPrompt + "\n\n参考知识库内容：\n---\n" + kbContext + "\n---";
    }

    private String retrieveKbContext(AgentDef agent, String userQuery, String uid) {
        if (agent.getVdbIds() == null || agent.getVdbIds().isEmpty() || "[]".equals(agent.getVdbIds())) {
            return "";
        }

        try {
            List<Long> vdbIds = MAPPER.readValue(agent.getVdbIds(), new TypeReference<List<Long>>() {});
            if (vdbIds.isEmpty()) return "";

            StringBuilder sb = new StringBuilder();
            for (long vdbId : vdbIds) {
                String ctx = kbMgr.searchInKB(userQuery, vdbId, uid,
                        cfg.getKb().getTopK(), cfg.getKb().getScoreThreshold());
                if (ctx != null && !ctx.isEmpty()) {
                    sb.append(ctx).append("\n");
                }
            }
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