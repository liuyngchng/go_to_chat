package com.rd.robot.model;

/**
 * Workflow engine execution event (for SSE streaming).
 */
public class EngineEvent {
    private String type; // "progress" | "chunk" | "done" | "error"
    private int step;
    private int total;
    private String agent;
    private String content;

    public EngineEvent() {}

    public EngineEvent(String type, String content) {
        this.type = type;
        this.content = content;
    }

    public EngineEvent(String type, int step, int total, String agent, String content) {
        this.type = type;
        this.step = step;
        this.total = total;
        this.agent = agent;
        this.content = content;
    }

    public String getType() { return type; }
    public void setType(String type) { this.type = type; }
    public int getStep() { return step; }
    public void setStep(int step) { this.step = step; }
    public int getTotal() { return total; }
    public void setTotal(int total) { this.total = total; }
    public String getAgent() { return agent; }
    public void setAgent(String agent) { this.agent = agent; }
    public String getContent() { return content; }
    public void setContent(String content) { this.content = content; }
}