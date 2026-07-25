package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class APIConfig {
    @JsonProperty("llm_api_uri")
    private String llmApiUri;
    @JsonProperty("llm_api_key")
    private String llmApiKey;
    @JsonProperty("llm_model_name")
    private String llmModelName;
    @JsonProperty("embedding_api_uri")
    private String embeddingApiUri;
    @JsonProperty("embedding_api_key")
    private String embeddingApiKey;
    @JsonProperty("embedding_model_name")
    private String embeddingModelName;

    public String getLlmApiUri() { return llmApiUri; }
    public void setLlmApiUri(String llmApiUri) { this.llmApiUri = llmApiUri; }
    public String getLlmApiKey() { return llmApiKey; }
    public void setLlmApiKey(String llmApiKey) { this.llmApiKey = llmApiKey; }
    public String getLlmModelName() { return llmModelName; }
    public void setLlmModelName(String llmModelName) { this.llmModelName = llmModelName; }
    public String getEmbeddingApiUri() { return embeddingApiUri; }
    public void setEmbeddingApiUri(String embeddingApiUri) { this.embeddingApiUri = embeddingApiUri; }
    public String getEmbeddingApiKey() { return embeddingApiKey; }
    public void setEmbeddingApiKey(String embeddingApiKey) { this.embeddingApiKey = embeddingApiKey; }
    public String getEmbeddingModelName() { return embeddingModelName; }
    public void setEmbeddingModelName(String embeddingModelName) { this.embeddingModelName = embeddingModelName; }
}
