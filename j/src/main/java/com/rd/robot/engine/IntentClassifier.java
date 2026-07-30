package com.rd.robot.engine;

import com.rd.robot.client.LlmClient;
import com.rd.robot.model.ClassifierDef;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * Intent classifier.
 * First tries keyword matching, then falls back to LLM classification.
 */
public class IntentClassifier {

    private static final Logger log = LoggerFactory.getLogger(IntentClassifier.class);

    /**
     * Classify user intent:
     * 1. Try keyword matching first
     * 2. Fall back to LLM classification
     * 3. Final fallback: return the last category name
     *
     * @param cfg       classifier definition
     * @param userQuery user input
     * @param llmClient LLM client for fallback classification
     * @return matched category name, or empty string if none matches
     */
    public static String classify(ClassifierDef cfg, String userQuery, LlmClient llmClient) {
        if (cfg == null || cfg.getCategories() == null || cfg.getCategories().isEmpty()) {
            return "";
        }

        // 1. Keyword matching
        String name = matchKeyword(userQuery, cfg);
        if (name != null && !name.isEmpty()) {
            return name;
        }

        // 2. LLM classification fallback
        if (llmClient != null) {
            name = llmClassify(cfg, userQuery, llmClient);
            if (name != null && !name.isEmpty()) {
                return name;
            }
        }

        // 3. Final fallback: return the last category (usually "general" category)
        var categories = cfg.getCategories();
        String fallback = categories.get(categories.size() - 1).getName();
        log.info("classifier fallback intent={} query={}", fallback, truncate(userQuery, 50));
        return fallback;
    }

    /**
     * Match user query against category keywords.
     * Picks the category with the longest keyword list when multiple match.
     */
    private static String matchKeyword(String query, ClassifierDef cfg) {
        String queryLower = query.toLowerCase();
        String bestMatch = null;
        int bestLen = 0;

        for (var cat : cfg.getCategories()) {
            if (cat.getKeywords() == null) continue;
            for (String kw : cat.getKeywords()) {
                if (queryLower.contains(kw.toLowerCase())) {
                    if (cat.getKeywords().size() > bestLen) {
                        bestMatch = cat.getName();
                        bestLen = cat.getKeywords().size();
                    }
                    break; // one keyword hit is enough
                }
            }
        }

        return bestMatch;
    }

    /**
     * Use LLM to classify intent.
     */
    private static String llmClassify(ClassifierDef cfg, String userQuery, LlmClient llmClient) {
        String systemPrompt = buildClassifierPrompt(cfg);
        String userMessage = "用户输入：" + userQuery + "\n\n请输出最匹配的类别名称：";

        try {
            String result = llmClient.chat(systemPrompt, userMessage);
            if (result == null) return "";

            String name = result.trim();
            // Remove punctuation
            name = name.replaceAll("[\"'。，,.：:]", "").trim();

            // Validate against known categories
            for (var cat : cfg.getCategories()) {
                if (name.equalsIgnoreCase(cat.getName())) {
                    log.info("classifier LLM matched intent={}", cat.getName());
                    return cat.getName();
                }
                // Fuzzy match: check if category name is contained in LLM output
                if (result.toLowerCase().contains(cat.getName().toLowerCase())) {
                    log.info("classifier LLM fuzzy matched intent={}", cat.getName());
                    return cat.getName();
                }
            }

            log.warn("classifier LLM returned unknown category result={}", result);
        } catch (Exception e) {
            log.warn("classifier LLM call failed", e);
        }

        return "";
    }

    private static String buildClassifierPrompt(ClassifierDef cfg) {
        StringBuilder sb = new StringBuilder();

        if (cfg.getPrompt() != null && !cfg.getPrompt().isEmpty()) {
            sb.append(cfg.getPrompt()).append("\n\n");
        } else {
            sb.append("你是一个意图分类器。根据用户输入，判断其意图属于以下哪个类别。\n");
            sb.append("请只输出类别名称，不要输出任何其他内容。\n\n");
        }

        sb.append("可选类别：\n");
        for (var cat : cfg.getCategories()) {
            sb.append("- ").append(cat.getName()).append("：").append(cat.getDescription()).append("\n");
        }

        sb.append("\n请只输出类别名称，不要解释。");
        return sb.toString();
    }

    private static String truncate(String s, int maxLen) {
        if (s == null) return "";
        return s.length() <= maxLen ? s : s.substring(0, maxLen);
    }
}