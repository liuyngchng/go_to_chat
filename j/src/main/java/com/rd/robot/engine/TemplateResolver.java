package com.rd.robot.engine;

import java.time.LocalDate;
import java.time.DayOfWeek;
import java.time.format.DateTimeFormatter;
import java.util.*;

/**
 * Template resolver — replaces {{var}} placeholders with values from a variable map.
 *
 * <h3>Built-in system variables (sys. prefix)</h3>
 * <ul>
 *   <li>{@code {{sys.user_query}}} — user's original question</li>
 *   <li>{@code {{sys.history}}}    — conversation history</li>
 *   <li>{@code {{sys.cur_date}}}   — current date (YYYY-MM-DD)</li>
 *   <li>{@code {{sys.cur_week}}}   — current weekday (Chinese)</li>
 *   <li>{@code {{sys.kb_context}}} — KB retrieval result</li>
 *   <li>{@code {{sys.intent}}}     — intent classification result</li>
 * </ul>
 *
 * <h3>Legacy variable names (no prefix, kept for compatibility)</h3>
 * {@code {{user_query}} {{history}} {{cur_date}} {{cur_week}} {{intent}}}
 *
 * <h3>Custom variables</h3>
 * Any variable <em>without</em> {@code sys.} prefix is treated as a
 * workflow-level variable (node output or user-defined) and is NOT
 * validated — it will be resolved at runtime.
 */
public class TemplateResolver {

    // Matches {{var_name}} and {{sys.var_name}}
    private static final java.util.regex.Pattern VAR_PATTERN =
            java.util.regex.Pattern.compile("\\{\\{(\\w+(?:\\.\\w+)*)\\}\\}");

    /** System variable name → description */
    private static final Map<String, String> VALID_SYS_VARS = Map.of(
            "sys.user_query", "用户当前问题",
            "sys.history",    "历史对话记录",
            "sys.cur_date",   "当前日期 (YYYY-MM-DD)",
            "sys.cur_week",   "当前星期几（中文）",
            "sys.kb_context", "知识库检索结果（由智能体绑定的知识库检索得出）",
            "sys.intent",     "意图分类结果（如有分类器）"
    );

    /** System variable metadata record */
    public record SysVarInfo(String name, String description) {}

    /**
     * Return all available system variables for API consumers.
     */
    public static List<SysVarInfo> getSystemVars() {
        return VALID_SYS_VARS.entrySet().stream()
                .map(e -> new SysVarInfo(e.getKey(), e.getValue()))
                .toList();
    }

    private static final String[] WEEKDAY_CN = {"日", "一", "二", "三", "四", "五", "六"};

    /**
     * Resolve placeholders in the template string.
     * Unmatched placeholders are preserved as-is.
     */
    public static String resolve(String template, Map<String, String> vars) {
        if (template == null || template.isEmpty()) return template;

        StringBuffer sb = new StringBuffer();
        java.util.regex.Matcher m = VAR_PATTERN.matcher(template);
        while (m.find()) {
            String key = m.group(1);
            String value = vars.get(key);
            if (value != null) {
                m.appendReplacement(sb, java.util.regex.Matcher.quoteReplacement(value));
            }
        }
        m.appendTail(sb);
        return sb.toString();
    }

    /**
     * Validate system variable references in a template.
     * Variables with {@code sys.} prefix must be in the whitelist.
     * Variables without the prefix are allowed (workflow-level).
     *
     * @return list of invalid variable names (empty if all valid)
     */
    public static List<String> validateTemplateVars(String template) {
        if (template == null || template.isEmpty()) return List.of();

        java.util.regex.Matcher m = VAR_PATTERN.matcher(template);
        Set<String> seen = new HashSet<>();
        List<String> invalid = new ArrayList<>();

        while (m.find()) {
            String varName = m.group(1);
            if (!seen.add(varName)) continue; // skip duplicates

            if (varName.startsWith("sys.") && !VALID_SYS_VARS.containsKey(varName)) {
                invalid.add(varName);
            }
        }
        return invalid;
    }

    /**
     * Return current weekday in Chinese.
     */
    public static String getWeekdayCN() {
        DayOfWeek dow = LocalDate.now().getDayOfWeek();
        // DayOfWeek: MONDAY=1 … SUNDAY=7, our array: 日=0,一=1 … 六=6
        return WEEKDAY_CN[dow.getValue() % 7];
    }

    /**
     * Format history messages as a string for template use.
     */
    public static String formatHistory(List<ChatMsg> messages) {
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
