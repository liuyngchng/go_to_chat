package com.rd.robot.session;

import com.rd.robot.model.ChatHistory;
import com.rd.robot.model.ChatMessage;
import com.rd.robot.repository.MetaStore;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;

/**
 * Session manager — in-memory with optional DB persistence.
 * TODO: 后续迁移至 Redis 替代 SQLite 持久化
 */
public class SessionManager {

    private static final Logger log = LoggerFactory.getLogger(SessionManager.class);
    private static final int MAX_HISTORY_ROUNDS = 5;
    private static final long SESSION_TIMEOUT_MS = 30 * 60 * 1000L;
    private static final int PERSIST_LOAD_LIMIT = 20;

    private final ConcurrentHashMap<String, ChatHistory> sessions = new ConcurrentHashMap<>();
    private final ScheduledExecutorService cleanupScheduler;
    private final MetaStore store;

    public SessionManager(MetaStore store) {
        this.store = store;
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

    public List<ChatMessage> getHistory(String uid) {
        ChatHistory history = sessions.get(uid);
        if (history != null) {
            synchronized (history) {
                return new ArrayList<>(history.getMessages());
            }
        }

        // Fallback: load from DB
        if (store != null) {
            try {
                List<ChatMessage> msgs = store.getChatMessages(uid, PERSIST_LOAD_LIMIT);
                if (msgs != null && !msgs.isEmpty()) {
                    ChatHistory loaded = new ChatHistory(uid);
                    loaded.setMessages(new ArrayList<>(msgs));
                    loaded.setUpdatedAt(System.currentTimeMillis());
                    sessions.put(uid, loaded);
                    log.info("从 DB 恢复聊天历史 uid={} count={}", uid, msgs.size());
                    return msgs;
                }
            } catch (Exception e) {
                log.warn("从 DB 加载聊天历史失败 uid={} error={}", uid, e.getMessage());
            }
        }

        return Collections.emptyList();
    }

    public void addMessage(String uid, String role, String content) {
        ChatHistory history = sessions.computeIfAbsent(uid,
                k -> new ChatHistory(uid));

        synchronized (history) {
            history.getMessages().add(new ChatMessage(role, content));
            history.setUpdatedAt(System.currentTimeMillis());

            int maxMessages = MAX_HISTORY_ROUNDS * 2;
            if (history.getMessages().size() > maxMessages) {
                int start = history.getMessages().size() - maxMessages;
                history.setMessages(new ArrayList<>(history.getMessages().subList(start, history.getMessages().size())));
            }
        }

        // Async persist to DB
        if (store != null) {
            final String u = uid;
            new Thread(() -> {
                try {
                    store.saveChatMessage(u, role, content);
                } catch (Exception e) {
                    log.warn("持久化聊天消息失败 uid={} error={}", u, e.getMessage());
                }
            }).start();
        }
    }

    public void clear(String uid) {
        sessions.remove(uid);
        if (store != null) {
            try {
                store.clearChatMessages(uid);
            } catch (Exception e) {
                log.warn("清空 DB 聊天历史失败 uid={} error={}", uid, e.getMessage());
            }
        }
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
