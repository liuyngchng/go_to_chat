package com.rd.robot.web.controller;

import java.util.List;
import java.util.Map;

/**
 * 在线座席状态存储，支持内存和 Redis 两种实现。
 */
public interface PresenceStore {

    /** 记录座席在线 */
    void setPresence(String userName, long loginTimeMs);

    /** 移除座席在线状态 */
    void removePresence(String userName);

    /** 获取所有在线座席列表 */
    List<Map<String, Object>> getOnlineAgents();
}