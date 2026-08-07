package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;

/**
 * 对象存储配置（仅 server.mode 设为 "cluster" 时需要）
 */
public class OSSConfig {
    @JsonProperty("type")
    private String type = "minio";

    @JsonProperty("endpoint")
    private String endpoint = "localhost:9000";

    @JsonProperty("access_key")
    private String accessKey;

    @JsonProperty("secret_key")
    private String secretKey;

    @JsonProperty("bucket")
    private String bucket = "kb-chat-flow";

    public String getType() { return type; }
    public void setType(String type) { this.type = type; }
    public String getEndpoint() { return endpoint; }
    public void setEndpoint(String endpoint) { this.endpoint = endpoint; }
    public String getAccessKey() { return accessKey; }
    public void setAccessKey(String accessKey) { this.accessKey = accessKey; }
    public String getSecretKey() { return secretKey; }
    public void setSecretKey(String secretKey) { this.secretKey = secretKey; }
    public String getBucket() { return bucket; }
    public void setBucket(String bucket) { this.bucket = bucket; }
}