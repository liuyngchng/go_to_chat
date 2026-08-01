package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;

/**
 * Workflow engine execution event (for SSE streaming).
 */
public class EngineEvent {
    private String type; // "progress" | "chunk" | "done" | "error"
    private int step;
    private int total;
    private String agent;
    private String content;
    @JsonProperty("node_id")
    private String nodeId;       // DAG mode: node ID
    @JsonProperty("parallel_group")
    private String parallelGroup; // DAG mode: parallel group name

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
    public String getNodeId() { return nodeId; }
    public void setNodeId(String nodeId) { this.nodeId = nodeId; }
    public String getParallelGroup() { return parallelGroup; }
    public void setParallelGroup(String parallelGroup) { this.parallelGroup = parallelGroup; }
}