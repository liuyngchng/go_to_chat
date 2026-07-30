package com.rd.robot.session;

import com.rd.robot.model.ChatHistory;
import com.rd.robot.model.ChatMessage;

import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;

public class SessionManager {

    private static final int MAX_HISTORY_ROUNDS = 5;
    private static final long SESSION_TIMEOUT_MS = 30 * 60 * 1000L;

    private final ConcurrentHashMap<String, ChatHistory> sessions = new ConcurrentHashMap<>();
    private final ScheduledExecutorService cleanupScheduler;

    public SessionManager() {
        cleanupScheduler = Executors.newSingleThreadScheduledExecutor(r -> {
            Thread t = new Thread(r, "session-cleanup");
            t.setDaemon(true);
            return t;
        });
        cleanupScheduler.scheduleWithFixedDelay(this::cleanup, 10, 10, TimeUnit.MINUTES);
    }

    // ============================================================
    // 会话操作
    // ============================================================

    public List<ChatMessage> getHistory(String uid, String sessionId) {
        String key = uid + ":" + sessionId;
        ChatHistory history = sessions.get(key);
        if (history == null) {
            return Collections.emptyList();
        }

        synchronized (history) {
            return new ArrayList<>(history.getMessages());
        }
    }

    public void addMessage(String uid, String sessionId, String role, String content) {
        String key = uid + ":" + sessionId;
        ChatHistory history = sessions.computeIfAbsent(key,
                k -> new ChatHistory(sessionId, uid));

        synchronized (history) {
            history.getMessages().add(new ChatMessage(role, content));
            history.setUpdatedAt(System.currentTimeMillis());

            // 限制历史长度
            int maxMessages = MAX_HISTORY_ROUNDS * 2;
            if (history.getMessages().size() > maxMessages) {
                int start = history.getMessages().size() - maxMessages;
                history.setMessages(new ArrayList<>(history.getMessages().subList(start, history.getMessages().size())));
            }
        }
    }

    public void clear(String uid, String sessionId) {
        String key = uid + ":" + sessionId;
        sessions.remove(key);
    }

    // ============================================================
    // 历史格式化
    // ============================================================

    public static String formatHistory(List<ChatMessage> messages) {
        if (messages == null || messages.isEmpty()) {
            return "（无历史对话）";
        }

        StringBuilder result = new StringBuilder();
        for (ChatMessage msg : messages) {
            if ("user".equals(msg.getRole())) {
                result.append("用户：").append(msg.getContent()).append("\n");
            } else {
                result.append("机器人：").append(msg.getContent()).append("\n");
            }
        }
        return result.toString();
    }

    // ============================================================
    // 过期清理
    // ============================================================

    private void cleanup() {
        long now = System.currentTimeMillis();
        Iterator<Map.Entry<String, ChatHistory>> it = sessions.entrySet().iterator();
        while (it.hasNext()) {
            Map.Entry<String, ChatHistory> entry = it.next();
            ChatHistory history = entry.getValue();
            synchronized (history) {
                if (now - history.getUpdatedAt() > SESSION_TIMEOUT_MS) {
                    it.remove();
                }
            }
        }
    }
}
