package com.rd.robot.model;

import java.util.List;

public class ChatCompletionRequest {
    private String model;
    private List<ChatCompletionMsg> messages;
    private boolean stream;
    private Double temperature;
    private Double topP;
    private Integer maxTokens;

    public String getModel() { return model; }
    public void setModel(String model) { this.model = model; }
    public List<ChatCompletionMsg> getMessages() { return messages; }
    public void setMessages(List<ChatCompletionMsg> messages) { this.messages = messages; }
    public boolean isStream() { return stream; }
    public void setStream(boolean stream) { this.stream = stream; }
    public Double getTemperature() { return temperature; }
    public void setTemperature(Double temperature) { this.temperature = temperature; }
    public Double getTopP() { return topP; }
    public void setTopP(Double topP) { this.topP = topP; }
    public Integer getMaxTokens() { return maxTokens; }
    public void setMaxTokens(Integer maxTokens) { this.maxTokens = maxTokens; }
}