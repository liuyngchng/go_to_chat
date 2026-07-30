package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.List;

public class CreateWorkflowRequest {
    private String name;
    private String description;
    private ClassifierDef classifier;
    private List<WorkflowNode> nodes;

    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getDescription() { return description; }
    public void setDescription(String description) { this.description = description; }
    public ClassifierDef getClassifier() { return classifier; }
    public void setClassifier(ClassifierDef classifier) { this.classifier = classifier; }
    public List<WorkflowNode> getNodes() { return nodes; }
    public void setNodes(List<WorkflowNode> nodes) { this.nodes = nodes; }
}