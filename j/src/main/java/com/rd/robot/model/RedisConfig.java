package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;

/**
 * Redis 配置（仅 server.mode 设为 "cluster" 时需要）
 */
public class RedisConfig {
    @JsonProperty("addr")
    private String addr = "localhost:6379";

    @JsonProperty("password")
    private String password;

    @JsonProperty("db")
    private int db;

    public String getAddr() { return addr; }
    public void setAddr(String addr) { this.addr = addr; }
    public String getPassword() { return password; }
    public void setPassword(String password) { this.password = password; }
    public int getDb() { return db; }
    public void setDb(int db) { this.db = db; }
}