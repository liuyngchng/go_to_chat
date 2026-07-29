package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.List;

public class CreateAgentRequest {
    private String name;
    private String description;

    @JsonProperty("system_prompt")
    private String systemPrompt;

    @JsonProperty("model_name")
    private String modelName;

    private Double temperature;
    private Double topP;

    @JsonProperty("max_tokens")
    private Integer maxTokens;

    @JsonProperty("vdb_ids")
    private List<Long> vdbIds;

    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getDescription() { return description; }
    public void setDescription(String description) { this.description = description; }
    public String getSystemPrompt() { return systemPrompt; }
    public void setSystemPrompt(String systemPrompt) { this.systemPrompt = systemPrompt; }
    public String getModelName() { return modelName; }
    public void setModelName(String modelName) { this.modelName = modelName; }
    public Double getTemperature() { return temperature; }
    public void setTemperature(Double temperature) { this.temperature = temperature; }
    public Double getTopP() { return topP; }
    public void setTopP(Double topP) { this.topP = topP; }
    public Integer getMaxTokens() { return maxTokens; }
    public void setMaxTokens(Integer maxTokens) { this.maxTokens = maxTokens; }
    public List<Long> getVdbIds() { return vdbIds; }
    public void setVdbIds(List<Long> vdbIds) { this.vdbIds = vdbIds; }
}