package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class LLMParams {
    @JsonProperty("temperature")
    private double temperature = 0.7;

    @JsonProperty("top_p")
    private double topP = 0.9;

    @JsonProperty("max_tokens")
    private int maxTokens = 2048;

    public double getTemperature() { return temperature; }
    public void setTemperature(double temperature) { this.temperature = temperature; }
    public double getTopP() { return topP; }
    public void setTopP(double topP) { this.topP = topP; }
    public int getMaxTokens() { return maxTokens; }
    public void setMaxTokens(int maxTokens) { this.maxTokens = maxTokens; }
}