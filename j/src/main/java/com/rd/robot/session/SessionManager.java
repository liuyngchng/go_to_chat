package com.rd.robot.session;

import com.rd.robot.model.ChatMessage;
import com.rd.robot.repository.MetaStore;

import java.util.List;

/**
 * 会话管理器，委托给 SessionStore 实现。
 * 单例模式 → MemorySessionStore（进程内存 + 异步 DB 落盘）。
 * 集群模式 → RedisSessionStore（Redis 存储 + 自动过期）。
 */
public class SessionManager {

    private final SessionStore store;

    /** 创建会话管理器（单例模式） */
    public SessionManager(MetaStore metaStore) {
        this.store = new MemorySessionStore(metaStore);
    }

    /** 创建会话管理器（指定底层存储实现） */
    public SessionManager(SessionStore sessionStore) {
        this.store = sessionStore;
    }

    public List<ChatMessage> getHistory(String uid) {
        return store.getHistory(uid);
    }

    public void addMessage(String uid, String role, String content) {
        store.addMessage(uid, role, content);
    }

    public void clear(String uid) {
        store.clear(uid);
    }

    public void stop() {
        store.stop();
    }

    /** 格式化历史消息为字符串 */
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
}