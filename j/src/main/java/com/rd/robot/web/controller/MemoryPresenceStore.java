package com.rd.robot.web.controller;

import com.rd.robot.model.User;
import com.rd.robot.repository.MetaStore;

import java.time.Instant;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;

/**
 * 进程内存实现的在线座席状态存储（单例模式）。
 */
public class MemoryPresenceStore implements PresenceStore {

    private final ConcurrentHashMap<String, Instant> onlineAgents = new ConcurrentHashMap<>();
    private final MetaStore metaStore;

    public MemoryPresenceStore(MetaStore metaStore) {
        this.metaStore = metaStore;
    }

    @Override
    public void setPresence(String userName, long loginTimeMs) {
        onlineAgents.put(userName, Instant.ofEpochMilli(loginTimeMs));
    }

    @Override
    public void removePresence(String userName) {
        onlineAgents.remove(userName);
    }

    @Override
    public List<Map<String, Object>> getOnlineAgents() {
        List<Map<String, Object>> agents = new ArrayList<>();
        for (var entry : onlineAgents.entrySet()) {
            Map<String, Object> info = new LinkedHashMap<>();
            info.put("user_name", entry.getKey());
            info.put("login_time", entry.getValue().toString());
            try {
                User user = metaStore.getUserByName(entry.getKey());
                if (user != null && user.getNote() != null) {
                    info.put("note", user.getNote());
                }
            } catch (Exception ignored) {}
            agents.add(info);
        }
        return agents;
    }
}