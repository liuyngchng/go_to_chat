package com.rd.robot.redis;

import com.rd.robot.model.Config;
import redis.clients.jedis.Jedis;
import redis.clients.jedis.JedisPool;
import redis.clients.jedis.JedisPoolConfig;
import redis.clients.jedis.params.SetParams;

import java.time.Duration;

/**
 * Redis 客户端封装（集群模式使用）。
 * 单例模式不需要初始化。
 */
public class RedisClient implements AutoCloseable {

    private final JedisPool pool;

    public RedisClient(Config cfg) {
        JedisPoolConfig poolConfig = new JedisPoolConfig();
        poolConfig.setMaxTotal(20);
        poolConfig.setMaxIdle(5);
        poolConfig.setMinIdle(1);

        String addr = cfg.getRedis().getAddr();
        String[] parts = addr.split(":");
        String host = parts[0];
        int port = parts.length > 1 ? Integer.parseInt(parts[1]) : 6379;

        String password = cfg.getRedis().getPassword();
        int db = cfg.getRedis().getDb();

        if (password != null && !password.isEmpty()) {
            this.pool = new JedisPool(poolConfig, host, port, 2000, password, db);
        } else {
            this.pool = new JedisPool(poolConfig, host, port, 2000, null, db);
        }
    }

    /** SETNX — 仅当 key 不存在时设置，返回 true 表示设置成功（获取锁成功） */
    public boolean setNX(String key, String value, long ttlSeconds) {
        try (Jedis jedis = pool.getResource()) {
            SetParams params = SetParams.setParams().nx().ex(ttlSeconds);
            return "OK".equals(jedis.set(key, value, params));
        }
    }

    /** DEL — 删除 key */
    public void del(String key) {
        try (Jedis jedis = pool.getResource()) {
            jedis.del(key);
        }
    }

    /** SET — 设置 key */
    public void set(String key, String value) {
        try (Jedis jedis = pool.getResource()) {
            jedis.set(key, value);
        }
    }

    /** GET — 获取 key */
    public String get(String key) {
        try (Jedis jedis = pool.getResource()) {
            return jedis.get(key);
        }
    }

    /** SET with TTL */
    public void setex(String key, long ttlSeconds, String value) {
        try (Jedis jedis = pool.getResource()) {
            jedis.setex(key, ttlSeconds, value);
        }
    }

    /** DEL by pattern (使用 SCAN，安全不阻塞) */
    public void delByPattern(String pattern) {
        try (Jedis jedis = pool.getResource()) {
            var keys = jedis.keys(pattern);
            if (keys != null && !keys.isEmpty()) {
                jedis.del(keys.toArray(new String[0]));
            }
        }
    }

    /** HGETALL — 获取 Hash 全部字段 */
    public java.util.Map<String, String> hgetAll(String key) {
        try (Jedis jedis = pool.getResource()) {
            return jedis.hgetAll(key);
        }
    }

    /** HSET — 设置 Hash 字段 */
    public void hset(String key, String field, String value) {
        try (Jedis jedis = pool.getResource()) {
            jedis.hset(key, field, value);
        }
    }

    /** HDEL — 删除 Hash 字段 */
    public void hdel(String key, String... fields) {
        try (Jedis jedis = pool.getResource()) {
            jedis.hdel(key, fields);
        }
    }

    /** PUBLISH — 发布消息到频道 */
    public void publish(String channel, String message) {
        try (Jedis jedis = pool.getResource()) {
            jedis.publish(channel, message);
        }
    }

    /** Return the underlying pool for subscription use */
    public JedisPool getPool() {
        return pool;
    }

    @Override
    public void close() {
        if (pool != null) {
            pool.close();
        }
    }
}