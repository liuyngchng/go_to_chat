package com.rd.robot.web.controller;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.rd.robot.client.ClientFactory;
import com.rd.robot.client.LlmClient;
import com.rd.robot.engine.TemplateResolver;
import com.rd.robot.engine.WorkflowEngine;
import com.rd.robot.knowledge.KnowledgeBaseManager;
import com.rd.robot.model.*;
import com.rd.robot.repository.MetaStore;
import com.rd.robot.session.SessionManager;
import com.rd.robot.web.server.HttpServer;
import io.netty.buffer.Unpooled;
import io.netty.channel.ChannelFutureListener;
import io.netty.channel.ChannelHandlerContext;
import io.netty.handler.codec.http.*;
import io.netty.util.CharsetUtil;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.LocalDate;
import java.time.format.DateTimeFormatter;
import java.util.List;
import java.util.stream.Collectors;

/**
 * Chat controller — SSE streaming chat and workflow execution.
 */
public class ChatController {

    private static final Logger log = LoggerFactory.getLogger(ChatController.class);
    private static final ObjectMapper MAPPER = new ObjectMapper();

    private final Config cfg;
    private final KnowledgeBaseManager kbMgr;
    private final SessionManager sessionMgr;
    private final ClientFactory clientFactory;
    private final MetaStore metaStore;
    private final WorkflowEngine workflowEngine;

    public ChatController(Config cfg, KnowledgeBaseManager kbMgr, SessionManager sessionMgr,
                          MetaStore metaStore, ClientFactory clientFactory) {
        this.cfg = cfg;
        this.kbMgr = kbMgr;
        this.sessionMgr = sessionMgr;
        this.metaStore = metaStore;
        this.clientFactory = clientFactory;
        this.workflowEngine = new WorkflowEngine(cfg, kbMgr, metaStore, clientFactory);
    }

    private LlmClient getLlmClient() {
        return clientFactory.getLlmClient();
    }

    /**
     * POST /api/chat — SSE streaming chat
     */
    public void chat(ChannelHandlerContext ctx, FullHttpRequest request) {
        try {
            String body = request.content().toString(CharsetUtil.UTF_8);
            ChatRequest req = MAPPER.readValue(body, ChatRequest.class);

            if (req.getMsg() == null || req.getMsg().isEmpty()) {
                HttpServer.sendJson(ctx, 400, "{\"error\":\"msg 不能为空\"}");
                return;
            }

            String uid = getUid(request);
            String sessionId = req.getSessionId() != null ? req.getSessionId() : "default";

            // Set SSE headers
            FullHttpResponse initResponse = new DefaultFullHttpResponse(
                    HttpVersion.HTTP_1_1, HttpResponseStatus.OK, Unpooled.EMPTY_BUFFER);
            initResponse.headers()
                    .set(HttpHeaderNames.CONTENT_TYPE, "text/event-stream; charset=utf-8")
                    .set(HttpHeaderNames.CACHE_CONTROL, "no-cache")
                    .set(HttpHeaderNames.CONNECTION, "keep-alive")
                    .set("X-Accel-Buffering", "no");
            ctx.writeAndFlush(initResponse);

            // If workflow_id is specified, use workflow engine
            if (req.getWorkflowId() > 0) {
                chatWithWorkflow(ctx, req, uid, sessionId);
                return;
            }

            // Get history
            List<ChatMessage> history = sessionMgr.getHistory(uid, sessionId);
            String historyStr = SessionManager.formatHistory(history);

            // Try FAQ matching first
            double faqThreshold = cfg.getFaq().getMatchThreshold();
            int faqCount = getFaqCount();
            if (faqCount > 0) {
                try {
                    FaqMatchResult faqResult = matchFaq(req.getMsg(), faqThreshold);
                    if (faqResult != null && faqResult.answer != null) {
                        log.info("faq-matched uid={} query={} score={}",
                                uid, truncate(req.getMsg(), 50), faqResult.score);
                        sessionMgr.addMessage(uid, sessionId, "user", req.getMsg());
                        ctx.writeAndFlush(new DefaultHttpContent(
                                Unpooled.copiedBuffer("data: \n\n", CharsetUtil.UTF_8)));
                        ctx.writeAndFlush(new DefaultHttpContent(
                                Unpooled.copiedBuffer("data: " + faqResult.answer + "\n\n", CharsetUtil.UTF_8)));
                        ctx.writeAndFlush(new DefaultHttpContent(
                                Unpooled.copiedBuffer("data: [DONE]\n\n", CharsetUtil.UTF_8)));
                        ctx.writeAndFlush(LastHttpContent.EMPTY_LAST_CONTENT)
                                .addListener(ChannelFutureListener.CLOSE);
                        sessionMgr.addMessage(uid, sessionId, "assistant", faqResult.answer);
                        return;
                    }
                } catch (Exception e) {
                    log.warn("FAQ 匹配失败", e);
                }
            }

            // Get KB context
            LocalDate today = LocalDate.now();
            String curDate = today.format(DateTimeFormatter.ofPattern("yyyy-MM-dd"));
            String curWeek = getWeekdayCN(today.getDayOfWeek().getValue());

            String contextStr = kbMgr.searchAllKBs(req.getMsg(), uid,
                    cfg.getKb().getTopK(), cfg.getKb().getScoreThreshold());

            // Build prompt
            String promptTemplate = getPromptTemplate();
            String systemPrompt = buildPrompt(promptTemplate, contextStr, historyStr, req.getMsg(), curDate, curWeek);

            log.info("chat uid={} session={} query={} contextLen={}",
                    uid, sessionId, truncate(req.getMsg(), 50), contextStr.length());

            // Save user message
            sessionMgr.addMessage(uid, sessionId, "user", req.getMsg());

            // Send initial event
            ctx.writeAndFlush(new DefaultHttpContent(
                    Unpooled.copiedBuffer("data: \n\n", CharsetUtil.UTF_8)));

            // Stream LLM response
            StringBuilder fullResponse = new StringBuilder();

            getLlmClient().chatStream(systemPrompt, "",
                    chunk -> {
                        fullResponse.append(chunk);
                        ctx.writeAndFlush(new DefaultHttpContent(
                                Unpooled.copiedBuffer("data: " + chunk + "\n\n", CharsetUtil.UTF_8)));
                    },
                    error -> {
                        log.error("LLM 错误 error={}", error);
                        ctx.writeAndFlush(new DefaultHttpContent(
                                Unpooled.copiedBuffer("data: [错误] " + error + "\n\n", CharsetUtil.UTF_8)));
                    },
                    () -> {
                        // Send DONE
                        ctx.writeAndFlush(new DefaultHttpContent(
                                Unpooled.copiedBuffer("data: [DONE]\n\n", CharsetUtil.UTF_8)));

                        // Save assistant response
                        String responseText = fullResponse.toString();
                        if (!responseText.isEmpty()) {
                            sessionMgr.addMessage(uid, sessionId, "assistant", responseText);
                        }

                        ctx.writeAndFlush(LastHttpContent.EMPTY_LAST_CONTENT)
                                .addListener(ChannelFutureListener.CLOSE);
                    });

        } catch (Exception e) {
            log.error("chat error", e);
            HttpServer.sendJson(ctx, 500, "{\"error\":\"聊天请求失败: " + e.getMessage() + "\"}");
        }
    }

    /**
     * Chat with workflow engine.
     */
    private void chatWithWorkflow(ChannelHandlerContext ctx, ChatRequest req, String uid, String sessionId) {
        List<ChatMessage> history = sessionMgr.getHistory(uid, sessionId);
        List<TemplateResolver.ChatMsg> historyMsgs = history.stream()
                .map(h -> new TemplateResolver.ChatMsg(h.getRole(), h.getContent()))
                .collect(Collectors.toList());

        // Save user message
        sessionMgr.addMessage(uid, sessionId, "user", req.getMsg());

        log.info("workflow-chat uid={} session={} workflow={} query={}",
                uid, sessionId, req.getWorkflowId(), truncate(req.getMsg(), 50));

        // Send initial event
        ctx.writeAndFlush(new DefaultHttpContent(
                Unpooled.copiedBuffer("data: \n\n", CharsetUtil.UTF_8)));

        StringBuilder fullResponse = new StringBuilder();
        var eventQueue = workflowEngine.executeStream(req.getWorkflowId(), req.getMsg(), uid, historyMsgs);

        // Process events in a background thread
        new Thread(() -> {
            try {
                while (true) {
                    EngineEvent evt = eventQueue.take();
                    switch (evt.getType()) {
                        case "progress":
                            ctx.writeAndFlush(new DefaultHttpContent(Unpooled.copiedBuffer(
                                    "data: [步骤 " + evt.getStep() + "/" + evt.getTotal() + "] " + evt.getAgent() + "\n\n",
                                    CharsetUtil.UTF_8)));
                            break;
                        case "chunk":
                            fullResponse.append(evt.getContent());
                            ctx.writeAndFlush(new DefaultHttpContent(Unpooled.copiedBuffer(
                                    "data: " + evt.getContent() + "\n\n", CharsetUtil.UTF_8)));
                            break;
                        case "error":
                            log.error("workflow error error={}", evt.getContent());
                            ctx.writeAndFlush(new DefaultHttpContent(Unpooled.copiedBuffer(
                                    "data: [错误] " + evt.getContent() + "\n\n", CharsetUtil.UTF_8)));
                            break;
                        case "done":
                            // Send DONE
                            ctx.writeAndFlush(new DefaultHttpContent(Unpooled.copiedBuffer(
                                    "data: [DONE]\n\n", CharsetUtil.UTF_8)));

                            String responseText = fullResponse.toString();
                            if (!responseText.isEmpty()) {
                                sessionMgr.addMessage(uid, sessionId, "assistant", responseText);
                            }

                            ctx.writeAndFlush(LastHttpContent.EMPTY_LAST_CONTENT)
                                    .addListener(ChannelFutureListener.CLOSE);
                            return;
                    }
                }
            } catch (Exception e) {
                log.error("workflow event processing error", e);
            }
        }).start();
    }

    /**
     * POST /api/chat/clear — clear session
     */
    public void clear(ChannelHandlerContext ctx, FullHttpRequest request) {
        try {
            String body = request.content().toString(CharsetUtil.UTF_8);
            ChatRequest req = MAPPER.readValue(body, ChatRequest.class);
            String uid = getUid(request);
            String sessionId = req.getSessionId() != null ? req.getSessionId() : "default";
            sessionMgr.clear(uid, sessionId);
            HttpServer.sendJson(ctx, 200, "{\"status\":\"ok\"}");
        } catch (Exception e) {
            HttpServer.sendJson(ctx, 500, "{\"error\":\"清空会话失败: " + e.getMessage() + "\"}");
        }
    }

    // ============================================================
    // Helpers
    // ============================================================

    private String buildPrompt(String template, String context, String history, String question,
                               String curDate, String curWeek) {
        return template
                .replace("{context}", context != null ? context : "")
                .replace("{history}", history != null ? history : "")
                .replace("{question}", question != null ? question : "")
                .replace("{cur_date}", curDate)
                .replace("{cur_week}", curWeek);
    }

    private String getWeekdayCN(int dayOfWeek) {
        String[] days = {"一", "二", "三", "四", "五", "六", "日"};
        return days[dayOfWeek - 1];
    }

    private String getPromptTemplate() {
        if (metaStore != null) {
            String prompt = metaStore.getPrompt("chat_msg");
            if (prompt != null && !prompt.isEmpty()) return prompt;
        }
        return cfg.getPrompts() != null && cfg.getPrompts().getChatMsg() != null
                ? cfg.getPrompts().getChatMsg()
                : "你是专业的对话机器人，负责解答客户咨询。\n\n知识库内容：\n---\n{context}\n---\n\n历史对话：\n{history}\n\n用户问题：{question}\n\n请用亲切、专业的中文回答：";
    }

    private String getUid(FullHttpRequest request) {
        // Try Authorization header first
        User user = AuthController.parseUserFromRequest(request);
        if (user != null) return user.getUserName();
        // Fallback to query param
        String uid = HttpServer.getQueryParam(request, "uid");
        return uid != null ? uid : "default";
    }

    // ============================================================
    // FAQ matching (inline, avoids circular dependency)
    // ============================================================

    private static class FaqMatchResult {
        String answer;
        double score;
    }

    private int getFaqCount() {
        try {
            return metaStore.getFaqEntries().size();
        } catch (Exception e) {
            return 0;
        }
    }

    private FaqMatchResult matchFaq(String query, double threshold) {
        try {
            var questions = metaStore.getAllFaqQuestionsWithEmbedding();
            if (questions.isEmpty()) return null;

            // Compute query embedding
            var embClient = new com.rd.robot.client.EmbeddingClient(
                    cfg.getApi().getEmbeddingApiUri(),
                    cfg.getApi().getEmbeddingApiKey(),
                    cfg.getApi().getEmbeddingModelName()
            );
            double[] queryVec = embClient.embedSingle(query);

            double bestScore = 0;
            long bestEntryId = 0;

            for (var q : questions) {
                if (q.getEmbedding() == null || q.getEmbedding().length == 0) continue;
                double score = cosineSimilarity(queryVec, q.getEmbedding());
                if (score > bestScore) {
                    bestScore = score;
                    bestEntryId = q.getEntryId();
                }
            }

            if (bestScore < threshold) return null;

            // Get answer from entry
            var entries = metaStore.getFaqEntries();
            for (var e : entries) {
                if (e.getId() == bestEntryId) {
                    FaqMatchResult result = new FaqMatchResult();
                    result.answer = e.getAnswer();
                    result.score = bestScore;
                    return result;
                }
            }
        } catch (Exception e) {
            log.warn("FAQ 匹配失败", e);
        }
        return null;
    }

    private static double cosineSimilarity(double[] a, double[] b) {
        if (a.length != b.length || a.length == 0) return 0;
        double dotProd = 0, normA = 0, normB = 0;
        for (int i = 0; i < a.length; i++) {
            dotProd += a[i] * b[i];
            normA += a[i] * a[i];
            normB += b[i] * b[i];
        }
        if (normA == 0 || normB == 0) return 0;
        return dotProd / (Math.sqrt(normA) * Math.sqrt(normB));
    }

    private static String truncate(String s, int maxLen) {
        if (s == null) return "";
        return s.length() <= maxLen ? s : s.substring(0, maxLen);
    }
}