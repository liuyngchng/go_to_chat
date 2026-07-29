package com.rd.robot.model;

import java.util.List;

public class RerankResponse {
    private List<RerankResult> results;

    public List<RerankResult> getResults() { return results; }
    public void setResults(List<RerankResult> results) { this.results = results; }
}