package com.rd.robot.web.controller;

import com.rd.robot.model.User;
import com.rd.robot.redis.RedisClient;
import com.rd.robot.repository.MetaStore;

import java.time.Instant;
import java.util.*;

/**
 * Redis 实现在线座席状态存储（集群模式）。
 * 使用 Redis Hash 存储在线座席信息，所有节点共享。
 */
public class RedisPresenceStore implements PresenceStore {

    private static final String KEY = "presence:online_agents";

    private final RedisClient redisClient;
    private final MetaStore metaStore;

    public RedisPresenceStore(RedisClient redisClient, MetaStore metaStore) {
        this.redisClient = redisClient;
        this.metaStore = metaStore;
    }

    @Override
    public void setPresence(String userName, long loginTimeMs) {
        redisClient.hset(KEY, userName, String.valueOf(loginTimeMs));
    }

    @Override
    public void removePresence(String userName) {
        redisClient.hdel(KEY, userName);
    }

    @Override
    public List<Map<String, Object>> getOnlineAgents() {
        Map<String, String> all = redisClient.hgetAll(KEY);
        if (all == null || all.isEmpty()) {
            return Collections.emptyList();
        }

        List<Map<String, Object>> agents = new ArrayList<>();
        for (var entry : all.entrySet()) {
            Map<String, Object> info = new LinkedHashMap<>();
            info.put("user_name", entry.getKey());
            try {
                long epochMs = Long.parseLong(entry.getValue());
                info.put("login_time", Instant.ofEpochMilli(epochMs).toString());
            } catch (NumberFormatException e) {
                info.put("login_time", entry.getValue());
            }
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