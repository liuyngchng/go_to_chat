package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.List;

public class ClassifierDef {
    private String prompt; // LLM classification prompt
    @JsonProperty("output_var")
    private String outputVar = "intent"; // variable to store result
    private List<IntentCategory> categories;

    public String getPrompt() { return prompt; }
    public void setPrompt(String prompt) { this.prompt = prompt; }
    public String getOutputVar() { return outputVar; }
    public void setOutputVar(String outputVar) { this.outputVar = outputVar; }
    public List<IntentCategory> getCategories() { return categories; }
    public void setCategories(List<IntentCategory> categories) { this.categories = categories; }
}