package com.rd.robot.server;

import com.rd.robot.kb.KnowledgeBaseManager;
import com.rd.robot.llm.LLMClient;
import com.rd.robot.model.ChatMessage;
import com.rd.robot.model.Config;
import com.rd.robot.session.SessionManager;
import com.rd.robot.store.SQLiteStore;

import io.netty.buffer.Unpooled;
import io.netty.channel.ChannelFutureListener;
import io.netty.channel.ChannelHandlerContext;
import io.netty.handler.codec.http.*;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.LocalDate;
import java.time.format.DateTimeFormatter;
import java.util.List;

public class ChatHandler {

    private static final Logger log = LoggerFactory.getLogger(ChatHandler.class);

    private final Config cfg;
    private final KnowledgeBaseManager kbMgr;
    private final SessionManager sessionMgr;
    private final LLMClient llmClient;
    private final SQLiteStore store;

    public ChatHandler(Config cfg, KnowledgeBaseManager kbMgr, SessionManager sessionMgr, SQLiteStore store) {
        this.cfg = cfg;
        this.kbMgr = kbMgr;
        this.sessionMgr = sessionMgr;
        this.store = store;
        this.llmClient = new LLMClient(
                cfg.getApi().getLlmApiUri(),
                cfg.getApi().getLlmApiKey(),
                cfg.getApi().getLlmModelName()
        );
    }

    // ============================================================
    // POST /chat — SSE 流式聊天
    // ============================================================

    public void chat(ChannelHandlerContext ctx, FullHttpRequest request) {
        String msg = HttpServer.getParam(request, "msg");
        String uid = getParamDefault(request, "uid", "default");
        String sessionId = getParamDefault(request, "session_id", "default");

        if (msg == null || msg.isEmpty()) {
            HttpServer.sendJson(ctx, 400, "{\"error\":\"参数错误: msg 不能为空\"}");
            return;
        }

        // 设置 SSE headers
        FullHttpResponse initResponse = new DefaultFullHttpResponse(
                HttpVersion.HTTP_1_1, HttpResponseStatus.OK, Unpooled.EMPTY_BUFFER);
        initResponse.headers()
                .set(HttpHeaderNames.CONTENT_TYPE, "text/event-stream; charset=utf-8")
                .set(HttpHeaderNames.CACHE_CONTROL, "no-cache")
                .set(HttpHeaderNames.CONNECTION, "keep-alive")
                .set("X-Accel-Buffering", "no");
        ctx.writeAndFlush(initResponse);

        // 获取历史
        List<ChatMessage> history = sessionMgr.getHistory(uid, sessionId);
        String historyStr = SessionManager.formatHistory(history);

        // 获取知识库上下文
        LocalDate today = LocalDate.now();
        String curDate = today.format(DateTimeFormatter.ofPattern("yyyy-MM-dd"));
        String curWeek = getWeekdayCN(today.getDayOfWeek().getValue());

        String contextStr = kbMgr.searchAllKBs(msg, uid, 3, 0.1);

        // 构建 system prompt
        String promptTemplate = getPromptTemplate();
        String systemPrompt = buildPrompt(promptTemplate, contextStr, historyStr, msg, curDate, curWeek);

        int previewLen = Math.min(msg.length(), 50);
        log.info("chat uid={} session={} query={} contextLen={}", uid, sessionId,
                msg.substring(0, previewLen), contextStr.length());

        // 保存用户消息
        sessionMgr.addMessage(uid, sessionId, "user", msg);

        // 发送初始化事件
        ctx.writeAndFlush(new DefaultHttpContent(Unpooled.copiedBuffer("data: \n\n", io.netty.util.CharsetUtil.UTF_8)));

        // 流式调用 LLM
        StringBuilder fullResponse = new StringBuilder();

        llmClient.chatStream(systemPrompt, "",
                chunk -> {
                    // onChunk
                    fullResponse.append(chunk);
                    ctx.writeAndFlush(new DefaultHttpContent(
                            Unpooled.copiedBuffer("data: " + chunk + "\n\n", io.netty.util.CharsetUtil.UTF_8)));
                },
                error -> {
                    // onError
                    log.error("LLM 错误 error={}", error);
                    ctx.writeAndFlush(new DefaultHttpContent(
                            Unpooled.copiedBuffer("data: [错误] " + error + "\n\n", io.netty.util.CharsetUtil.UTF_8)));
                },
                () -> {
                    // onDone
                    // 发送结束标记
                    ctx.writeAndFlush(new DefaultHttpContent(
                            Unpooled.copiedBuffer("data: [DONE]\n\n", io.netty.util.CharsetUtil.UTF_8)));

                    // 保存助手回复
                    String responseText = fullResponse.toString();
                    if (!responseText.isEmpty()) {
                        sessionMgr.addMessage(uid, sessionId, "assistant", responseText);
                    }

                    ctx.writeAndFlush(LastHttpContent.EMPTY_LAST_CONTENT)
                            .addListener(ChannelFutureListener.CLOSE);
                });
    }

    // ============================================================
    // POST /chat/clear — 清空会话
    // ============================================================

    public void clear(ChannelHandlerContext ctx, FullHttpRequest request) {
        String uid = getParamDefault(request, "uid", "default");
        String sessionId = getParamDefault(request, "session_id", "default");
        sessionMgr.clear(uid, sessionId);
        HttpServer.sendJson(ctx, 200, "{\"status\":\"ok\"}");
    }

    // ============================================================
    // 辅助方法
    // ============================================================

    private String buildPrompt(String template, String context, String history, String question,
                               String curDate, String curWeek) {
        return template
                .replace("{context}", context)
                .replace("{history}", history)
                .replace("{question}", question)
                .replace("{cur_date}", curDate)
                .replace("{cur_week}", curWeek);
    }

    private String getWeekdayCN(int dayOfWeek) {
        String[] days = {"一", "二", "三", "四", "五", "六", "日"};
        // dayOfWeek: 1=Monday ... 7=Sunday
        return days[dayOfWeek - 1];
    }

    private String getPromptTemplate() {
        if (store != null) {
            String prompt = store.getPrompt("chat_msg");
            if (prompt != null && !prompt.isEmpty()) {
                return prompt;
            }
        }
        return cfg.getPrompts().getChatMsg();
    }

    private String getParamDefault(FullHttpRequest request, String key, String defaultValue) {
        String val = HttpServer.getParam(request, key);
        return val != null && !val.isEmpty() ? val : defaultValue;
    }
}
