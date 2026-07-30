package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.List;

public class RerankRequest {
    private String model;
    private String query;
    private List<String> documents;

    @JsonProperty("top_n")
    private int topN;

    public RerankRequest() {}

    public RerankRequest(String model, String query, List<String> documents, int topN) {
        this.model = model;
        this.query = query;
        this.documents = documents;
        this.topN = topN;
    }

    public String getModel() { return model; }
    public void setModel(String model) { this.model = model; }
    public String getQuery() { return query; }
    public void setQuery(String query) { this.query = query; }
    public List<String> getDocuments() { return documents; }
    public void setDocuments(List<String> documents) { this.documents = documents; }
    public int getTopN() { return topN; }
    public void setTopN(int topN) { this.topN = topN; }
}