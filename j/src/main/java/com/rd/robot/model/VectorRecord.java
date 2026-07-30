package com.rd.robot.model;

import java.util.Map;

public class VectorRecord {
    private String id;
    private double[] vector;
    private String content;
    private Map<String, String> meta;

    public String getId() { return id; }
    public void setId(String id) { this.id = id; }
    public double[] getVector() { return vector; }
    public void setVector(double[] vector) { this.vector = vector; }
    public String getContent() { return content; }
    public void setContent(String content) { this.content = content; }
    public Map<String, String> getMeta() { return meta; }
    public void setMeta(Map<String, String> meta) { this.meta = meta; }
}
