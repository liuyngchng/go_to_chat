package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class FaqQuestionWithEmbedding {
    private long id;

    @JsonProperty("entry_id")
    private long entryId;

    private String question;
    private double[] embedding;

    public long getId() { return id; }
    public void setId(long id) { this.id = id; }
    public long getEntryId() { return entryId; }
    public void setEntryId(long entryId) { this.entryId = entryId; }
    public String getQuestion() { return question; }
    public void setQuestion(String question) { this.question = question; }
    public double[] getEmbedding() { return embedding; }
    public void setEmbedding(double[] embedding) { this.embedding = embedding; }
}