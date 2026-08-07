package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class ServerConfig {
    private int port;
    @JsonProperty("debug")
    private boolean debug;

    // 运行模式: "singleton" (默认) 或 "cluster"
    @JsonProperty("mode")
    private String mode = "singleton";

    // 服务角色: "all" (默认) | "admin" | "chat"
    @JsonProperty("role")
    private String role = "all";

    // HMAC token 签名密钥，多节点部署时需一致
    @JsonProperty("token_secret")
    private String tokenSecret;

    public int getPort() { return port; }
    public void setPort(int port) { this.port = port; }
    public boolean isDebug() { return debug; }
    public void setDebug(boolean debug) { this.debug = debug; }

    public String getMode() { return mode; }
    public void setMode(String mode) { this.mode = mode; }
    public String getRole() { return role; }
    public void setRole(String role) { this.role = role; }
    public String getTokenSecret() { return tokenSecret; }
    public void setTokenSecret(String tokenSecret) { this.tokenSecret = tokenSecret; }

    // 便捷判断方法
    public boolean isClusterMode() { return "cluster".equals(mode); }
    public boolean isSingletonMode() { return mode == null || "singleton".equals(mode); }
    public boolean isAdminRole() { return "admin".equals(role) || "all".equals(role); }
    public boolean isChatRole() { return "chat".equals(role) || "all".equals(role); }
    public boolean isAdminOnly() { return "admin".equals(role); }
    public boolean isChatOnly() { return "chat".equals(role); }
}