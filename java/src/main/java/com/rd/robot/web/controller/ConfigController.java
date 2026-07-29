package com.rd.robot.web.controller;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.rd.robot.client.ClientFactory;
import com.rd.robot.config.RuntimeConfig;
import com.rd.robot.model.Config;
import com.rd.robot.repository.MetaStore;
import com.rd.robot.web.server.HttpServer;
import io.netty.channel.ChannelHandlerContext;
import io.netty.handler.codec.http.FullHttpRequest;
import io.netty.util.CharsetUtil;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.Map;

/**
 * System configuration controller.
 */
public class ConfigController {

    private static final Logger log = LoggerFactory.getLogger(ConfigController.class);
    private static final ObjectMapper MAPPER = new ObjectMapper();

    private final Config cfg;
    private final MetaStore metaStore;
    private final ClientFactory clientFactory;

    public ConfigController(Config cfg, MetaStore metaStore, ClientFactory clientFactory) {
        this.cfg = cfg;
        this.metaStore = metaStore;
        this.clientFactory = clientFactory;
    }

    /**
     * GET /api/config — get all configuration
     */
    public void getConfig(ChannelHandlerContext ctx, FullHttpRequest request) {
        try {
            Map<String, Object> resp = Map.of("data", Map.of(
                    "sys", Map.of(
                            "api_auth", cfg.getSys().isApiAuth() ? "true" : "false"
                    ),
                    "api", Map.of(
                            "llm_api_uri", cfg.getApi().getLlmApiUri(),
                            "llm_api_key", cfg.getApi().getLlmApiKey(),
                            "llm_model_name", cfg.getApi().getLlmModelName(),
                            "embedding_api_uri", cfg.getApi().getEmbeddingApiUri(),
                            "embedding_api_key", cfg.getApi().getEmbeddingApiKey(),
                            "embedding_model_name", cfg.getApi().getEmbeddingModelName(),
                            "rerank_api_uri", cfg.getApi().getRerankApiUri(),
                            "rerank_api_key", cfg.getApi().getRerankApiKey(),
                            "rerank_model_name", cfg.getApi().getRerankModelName()
                    ),
                    "prompt", Map.of("chat_msg", getPrompt()),
                    "kb", Map.of(
                            "chunk_size", cfg.getKb().getChunkSize(),
                            "chunk_overlap", cfg.getKb().getChunkOverlap(),
                            "top_k", cfg.getKb().getTopK(),
                            "score_threshold", cfg.getKb().getScoreThreshold(),
                            "rerank_enabled", cfg.getKb().isRerankEnabled(),
                            "rerank_retrieve_n", cfg.getKb().getRerankRetrieveN()
                    ),
                    "llm", Map.of(
                            "temperature", cfg.getLlm().getTemperature(),
                            "top_p", cfg.getLlm().getTopP(),
                            "max_tokens", cfg.getLlm().getMaxTokens()
                    ),
                    "faq", Map.of("match_threshold", cfg.getFaq().getMatchThreshold())
            ));
            HttpServer.sendJson(ctx, 200, MAPPER.writeValueAsString(resp));
        } catch (Exception e) {
            HttpServer.sendError(ctx, io.netty.handler.codec.http.HttpResponseStatus.INTERNAL_SERVER_ERROR, e.getMessage());
        }
    }

    /**
     * PUT /api/config — update configuration
     */
    public void updateConfig(ChannelHandlerContext ctx, FullHttpRequest request) {
        try {
            String body = request.content().toString(CharsetUtil.UTF_8);
            @SuppressWarnings("unchecked")
            Map<String, Object> req = MAPPER.readValue(body, Map.class);

            @SuppressWarnings("unchecked")
            Map<String, Object> sys = (Map<String, Object>) req.get("sys");
            @SuppressWarnings("unchecked")
            Map<String, Object> api = (Map<String, Object>) req.get("api");
            @SuppressWarnings("unchecked")
            Map<String, Object> prompt = (Map<String, Object>) req.get("prompt");
            @SuppressWarnings("unchecked")
            Map<String, Object> kb = (Map<String, Object>) req.get("kb");
            @SuppressWarnings("unchecked")
            Map<String, Object> llm = (Map<String, Object>) req.get("llm");
            @SuppressWarnings("unchecked")
            Map<String, Object> faq = (Map<String, Object>) req.get("faq");

            if (sys != null) {
                if (sys.get("api_auth") != null) metaStore.setConfig("sys.api_auth", (String) sys.get("api_auth"), "是否启用接口认证");
            }

            if (api != null) {
                setConfigIfPresent(api, "llm_api_uri", "api.llm_api_uri");
                setConfigIfPresent(api, "llm_api_key", "api.llm_api_key");
                setConfigIfPresent(api, "llm_model_name", "api.llm_model_name");
                setConfigIfPresent(api, "embedding_api_uri", "api.embedding_api_uri");
                setConfigIfPresent(api, "embedding_api_key", "api.embedding_api_key");
                setConfigIfPresent(api, "embedding_model_name", "api.embedding_model_name");
                setConfigIfPresent(api, "rerank_api_uri", "api.rerank_api_uri");
                setConfigIfPresent(api, "rerank_api_key", "api.rerank_api_key");
                setConfigIfPresent(api, "rerank_model_name", "api.rerank_model_name");
            }

            if (prompt != null && prompt.get("chat_msg") != null) {
                metaStore.upsertPrompt("chat_msg", (String) prompt.get("chat_msg"), 0);
            }

            if (kb != null) {
                setConfigIfPresentInt(kb, "chunk_size", "kb.chunk_size");
                setConfigIfPresentInt(kb, "chunk_overlap", "kb.chunk_overlap");
                setConfigIfPresentInt(kb, "top_k", "kb.top_k");
                setConfigIfPresentDouble(kb, "score_threshold", "kb.score_threshold");
                setConfigIfPresentBool(kb, "rerank_enabled", "kb.rerank_enabled");
                setConfigIfPresentInt(kb, "rerank_retrieve_n", "kb.rerank_retrieve_n");
            }

            if (llm != null) {
                setConfigIfPresentDouble(llm, "temperature", "llm.temperature");
                setConfigIfPresentDouble(llm, "top_p", "llm.top_p");
                setConfigIfPresentInt(llm, "max_tokens", "llm.max_tokens");
            }

            if (faq != null) {
                setConfigIfPresentDouble(faq, "match_threshold", "faq.match_threshold");
            }

            // Reload runtime config and invalidate client cache
            RuntimeConfig.reload(metaStore, cfg);
            clientFactory.invalidate();

            HttpServer.sendJson(ctx, 200, "{\"status\":\"ok\"}");

        } catch (Exception e) {
            log.error("更新配置失败", e);
            HttpServer.sendJson(ctx, 500, "{\"error\":\"更新配置失败: " + e.getMessage() + "\"}");
        }
    }

    private String getPrompt() {
        if (metaStore != null) {
            String prompt = metaStore.getPrompt("chat_msg");
            if (prompt != null && !prompt.isEmpty()) return prompt;
        }
        return "你是专业的对话机器人，负责解答客户咨询。";
    }

    private void setConfigIfPresent(Map<String, Object> map, String key, String configKey) {
        if (map.get(key) != null) {
            metaStore.setConfig(configKey, String.valueOf(map.get(key)), "");
        }
    }

    private void setConfigIfPresentInt(Map<String, Object> map, String key, String configKey) {
        if (map.get(key) != null) {
            metaStore.setConfig(configKey, String.valueOf(map.get(key)), "");
        }
    }

    private void setConfigIfPresentDouble(Map<String, Object> map, String key, String configKey) {
        if (map.get(key) != null) {
            metaStore.setConfig(configKey, String.valueOf(map.get(key)), "");
        }
    }

    private void setConfigIfPresentBool(Map<String, Object> map, String key, String configKey) {
        if (map.get(key) != null) {
            metaStore.setConfig(configKey, String.valueOf(map.get(key)), "");
        }
    }
}