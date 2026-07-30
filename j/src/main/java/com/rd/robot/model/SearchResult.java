package com.rd.robot.model;

import java.util.Map;

public class SearchResult {
    private String id;
    private String content;
    private Map<String, String> metadata;
    private double score;

    public String getId() { return id; }
    public void setId(String id) { this.id = id; }
    public String getContent() { return content; }
    public void setContent(String content) { this.content = content; }
    public Map<String, String> getMetadata() { return metadata; }
    public void setMetadata(Map<String, String> metadata) { this.metadata = metadata; }
    public double getScore() { return score; }
    public void setScore(double score) { this.score = score; }
}
