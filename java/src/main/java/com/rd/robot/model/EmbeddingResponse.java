package com.rd.robot.model;

import java.util.List;

public class EmbeddingResponse {
    private List<EmbeddingData> data;

    public List<EmbeddingData> getData() { return data; }
    public void setData(List<EmbeddingData> data) { this.data = data; }

    public static class EmbeddingData {
        private List<Double> embedding;

        public List<Double> getEmbedding() { return embedding; }
        public void setEmbedding(List<Double> embedding) { this.embedding = embedding; }
    }
}
