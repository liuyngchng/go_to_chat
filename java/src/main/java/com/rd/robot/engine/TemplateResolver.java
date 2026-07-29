package com.rd.robot.engine;

import java.util.Map;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

/**
 * Template resolver — replaces {{var}} placeholders with values from a variable map.
 * Built-in variables: {{user_query}}, {{history}}
 * Custom variables come from upstream node OutputVar.
 */
public class TemplateResolver {

    private static final Pattern VAR_PATTERN = Pattern.compile("\\{\\{(\\w+)\\}\\}");

    /**
     * Resolve placeholders in the template string.
     * Unmatched placeholders are preserved as-is.
     */
    public static String resolve(String template, Map<String, String> vars) {
        if (template == null || template.isEmpty()) return template;

        StringBuffer sb = new StringBuffer();
        Matcher m = VAR_PATTERN.matcher(template);
        while (m.find()) {
            String key = m.group(1);
            String value = vars.get(key);
            if (value != null) {
                m.appendReplacement(sb, Matcher.quoteReplacement(value));
            }
        }
        m.appendTail(sb);
        return sb.toString();
    }

    /**
     * Format history messages as a string for template use.
     */
    public static String formatHistory(java.util.List<ChatMsg> messages) {
        if (messages == null || messages.isEmpty()) {
            return "（无历史对话）";
        }

        StringBuilder sb = new StringBuilder();
        for (ChatMsg msg : messages) {
            if ("user".equals(msg.role)) {
                sb.append("用户：").append(msg.content).append("\n");
            } else {
                sb.append("助手：").append(msg.content).append("\n");
            }
        }
        return sb.toString();
    }

    /**
     * Simplified chat message for the engine (avoids circular dependency with model package).
     */
    public static class ChatMsg {
        public String role;
        public String content;

        public ChatMsg() {}

        public ChatMsg(String role, String content) {
            this.role = role;
            this.content = content;
        }
    }
}