package com.rd.robot.model;

import java.util.List;

public class ChatCompletionRequest {
    private String model;
    private List<ChatCompletionMsg> messages;
    private boolean stream;

    public String getModel() { return model; }
    public void setModel(String model) { this.model = model; }
    public List<ChatCompletionMsg> getMessages() { return messages; }
    public void setMessages(List<ChatCompletionMsg> messages) { this.messages = messages; }
    public boolean isStream() { return stream; }
    public void setStream(boolean stream) { this.stream = stream; }
}