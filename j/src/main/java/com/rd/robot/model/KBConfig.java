package com.rd.robot.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class KBConfig {
    @JsonProperty("chunk_size")
    private int chunkSize = 300;

    @JsonProperty("chunk_overlap")
    private int chunkOverlap = 80;

    @JsonProperty("top_k")
    private int topK = 3;

    @JsonProperty("score_threshold")
    private double scoreThreshold = 0.1;

    @JsonProperty("rerank_enabled")
    private boolean rerankEnabled;

    @JsonProperty("rerank_retrieve_n")
    private int rerankRetrieveN = 15;

    public int getChunkSize() { return chunkSize; }
    public void setChunkSize(int chunkSize) { this.chunkSize = chunkSize; }
    public int getChunkOverlap() { return chunkOverlap; }
    public void setChunkOverlap(int chunkOverlap) { this.chunkOverlap = chunkOverlap; }
    public int getTopK() { return topK; }
    public void setTopK(int topK) { this.topK = topK; }
    public double getScoreThreshold() { return scoreThreshold; }
    public void setScoreThreshold(double scoreThreshold) { this.scoreThreshold = scoreThreshold; }
    public boolean isRerankEnabled() { return rerankEnabled; }
    public void setRerankEnabled(boolean rerankEnabled) { this.rerankEnabled = rerankEnabled; }
    public int getRerankRetrieveN() { return rerankRetrieveN; }
    public void setRerankRetrieveN(int rerankRetrieveN) { this.rerankRetrieveN = rerankRetrieveN; }
}