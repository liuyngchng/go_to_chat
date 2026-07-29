package com.rd.robot.client;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.rd.robot.model.RerankRequest;
import com.rd.robot.model.RerankResponse;
import com.rd.robot.model.RerankResult;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.List;

/**
 * Rerank client (OpenAI/Cohere compatible API).
 */
public class RerankClient {

    private static final Logger log = LoggerFactory.getLogger(RerankClient.class);
    private static final ObjectMapper MAPPER = new ObjectMapper();

    private final String baseUrl;
    private final String apiKey;
    private final String modelName;
    private final HttpClient httpClient;

    public RerankClient(String baseUrl, String apiKey, String modelName) {
        this.baseUrl = baseUrl.endsWith("/") ? baseUrl.substring(0, baseUrl.length() - 1) : baseUrl;
        this.apiKey = apiKey;
        this.modelName = modelName;
        this.httpClient = HttpClient.newBuilder()
                .connectTimeout(Duration.ofSeconds(10))
                .build();
    }

    /**
     * Rerank documents by relevance to the query.
     *
     * @param query     the search query
     * @param documents list of candidate documents
     * @param topN      number of top results to return
     * @return reranked results with indices and scores
     */
    public List<RerankResult> rerank(String query, List<String> documents, int topN) {
        if (documents == null || documents.isEmpty()) {
            return List.of();
        }

        try {
            RerankRequest reqBody = new RerankRequest(modelName, query, documents, topN);
            String json = MAPPER.writeValueAsString(reqBody);

            HttpRequest request = HttpRequest.newBuilder()
                    .uri(URI.create(baseUrl + "/rerank"))
                    .timeout(Duration.ofSeconds(60))
                    .header("Content-Type", "application/json")
                    .header("Authorization", "Bearer " + apiKey)
                    .POST(HttpRequest.BodyPublishers.ofString(json))
                    .build();

            HttpResponse<String> response = httpClient.send(request,
                    HttpResponse.BodyHandlers.ofString());

            if (response.statusCode() != 200) {
                throw new RuntimeException("Rerank API 返回错误 " + response.statusCode() + ": " + response.body());
            }

            RerankResponse result = MAPPER.readValue(response.body(), RerankResponse.class);
            return result.getResults() != null ? result.getResults() : List.of();

        } catch (Exception e) {
            throw new RuntimeException("Rerank 请求失败", e);
        }
    }
}