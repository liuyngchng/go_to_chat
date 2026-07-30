package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.time.LocalDateTime;

public class ApiToken {
    private long id;

    @JsonProperty("user_name")
    private String userName;

    @JsonProperty("token_preview")
    private String tokenPreview;

    @JsonProperty("expires_at")
    private LocalDateTime expiresAt;

    @JsonProperty("create_time")
    private LocalDateTime createTime;

    public long getId() { return id; }
    public void setId(long id) { this.id = id; }
    public String getUserName() { return userName; }
    public void setUserName(String userName) { this.userName = userName; }
    public String getTokenPreview() { return tokenPreview; }
    public void setTokenPreview(String tokenPreview) { this.tokenPreview = tokenPreview; }
    public LocalDateTime getExpiresAt() { return expiresAt; }
    public void setExpiresAt(LocalDateTime expiresAt) { this.expiresAt = expiresAt; }
    public LocalDateTime getCreateTime() { return createTime; }
    public void setCreateTime(LocalDateTime createTime) { this.createTime = createTime; }
}