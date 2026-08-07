package com.rd.robot.model;

import java.util.ArrayList;
import java.util.List;

public class ChatHistory {
    private String uid;
    private List<ChatMessage> messages;
    private long createdAt;
    private long updatedAt;

    public ChatHistory() {
        this.messages = new ArrayList<>();
    }

    public ChatHistory(String uid) {
        this.uid = uid;
        this.messages = new ArrayList<>();
        this.createdAt = System.currentTimeMillis();
        this.updatedAt = this.createdAt;
    }

    public String getUid() { return uid; }
    public void setUid(String uid) { this.uid = uid; }
    public List<ChatMessage> getMessages() { return messages; }
    public void setMessages(List<ChatMessage> messages) { this.messages = messages; }
    public long getCreatedAt() { return createdAt; }
    public void setCreatedAt(long createdAt) { this.createdAt = createdAt; }
    public long getUpdatedAt() { return updatedAt; }
    public void setUpdatedAt(long updatedAt) { this.updatedAt = updatedAt; }
}
