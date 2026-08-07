package com.rd.robot.session;

import com.rd.robot.model.ChatMessage;

import java.util.List;

/**
 * 会话存储接口，支持内存和 Redis 两种实现。
 */
public interface SessionStore {

    /** 获取会话历史 */
    List<ChatMessage> getHistory(String uid);

    /** 添加消息到会话 */
    void addMessage(String uid, String role, String content);

    /** 清空会话 */
    void clear(String uid);

    /** 停止后台任务 */
    void stop();
}