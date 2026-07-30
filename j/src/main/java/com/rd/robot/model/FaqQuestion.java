package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.time.LocalDateTime;

public class FaqQuestion {
    private long id;

    @JsonProperty("entry_id")
    private long entryId;

    private String question;

    @JsonProperty("created_at")
    private LocalDateTime createdAt;

    public long getId() { return id; }
    public void setId(long id) { this.id = id; }
    public long getEntryId() { return entryId; }
    public void setEntryId(long entryId) { this.entryId = entryId; }
    public String getQuestion() { return question; }
    public void setQuestion(String question) { this.question = question; }
    public LocalDateTime getCreatedAt() { return createdAt; }
    public void setCreatedAt(LocalDateTime createdAt) { this.createdAt = createdAt; }
}