package com.rd.robot.model;

import java.util.List;

public class IntentCategory {
    private String name; // category identifier, e.g. "emergency"
    private String description; // category description
    private List<String> keywords; // keyword list

    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getDescription() { return description; }
    public void setDescription(String description) { this.description = description; }
    public List<String> getKeywords() { return keywords; }
    public void setKeywords(List<String> keywords) { this.keywords = keywords; }
}