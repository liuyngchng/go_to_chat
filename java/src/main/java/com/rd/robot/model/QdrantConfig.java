package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class QdrantConfig {
    @JsonProperty("host")
    private String host = "localhost";

    @JsonProperty("port")
    private int port = 6334;

    @JsonProperty("api_key")
    private String apiKey;

    @JsonProperty("use_tls")
    private boolean useTls;

    public String getHost() { return host; }
    public void setHost(String host) { this.host = host; }
    public int getPort() { return port; }
    public void setPort(int port) { this.port = port; }
    public String getApiKey() { return apiKey; }
    public void setApiKey(String apiKey) { this.apiKey = apiKey; }
    public boolean isUseTls() { return useTls; }
    public void setUseTls(boolean useTls) { this.useTls = useTls; }
}