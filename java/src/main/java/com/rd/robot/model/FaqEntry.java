package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.time.LocalDateTime;
import java.util.List;

public class FaqEntry {
    private long id;
    private List<FaqQuestion> questions;
    private String answer;

    @JsonProperty("source_file")
    private String sourceFile;

    @JsonProperty("created_at")
    private LocalDateTime createdAt;

    public long getId() { return id; }
    public void setId(long id) { this.id = id; }
    public List<FaqQuestion> getQuestions() { return questions; }
    public void setQuestions(List<FaqQuestion> questions) { this.questions = questions; }
    public String getAnswer() { return answer; }
    public void setAnswer(String answer) { this.answer = answer; }
    public String getSourceFile() { return sourceFile; }
    public void setSourceFile(String sourceFile) { this.sourceFile = sourceFile; }
    public LocalDateTime getCreatedAt() { return createdAt; }
    public void setCreatedAt(LocalDateTime createdAt) { this.createdAt = createdAt; }
}