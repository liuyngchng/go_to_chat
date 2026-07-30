package engine

import (
	"regexp"
	"strings"
	"time"
)

// varPattern matches {{variable_name}} patterns
var varPattern = regexp.MustCompile(`\{\{(\w+(?:\.\w+)*)\}\}`)

// ResolveTemplate 替换模板中的 {{var}} 占位符。
//
// 内置系统变量（sys. 前缀）：
//
//	{{sys.user_query}} - 用户原始问题
//	{{sys.history}}    - 历史对话记录
//	{{sys.cur_date}}   - 当前日期 (YYYY-MM-DD)
//	{{sys.cur_week}}   - 当前星期几（中文）
//	{{sys.kb_context}} - 知识库检索结果（由节点绑定的 vdb_ids 检索得出）
//	{{sys.intent}}     - 意图分类结果（如有分类器）
//
// 兼容旧版变量名（无 sys. 前缀）：
//
//	{{user_query}} {{history}} {{cur_date}} {{cur_week}} {{intent}}
//
// 自定义变量来自上游节点的 OutputVar 和节点 ID。
func ResolveTemplate(tmpl string, vars map[string]string) string {
	return varPattern.ReplaceAllStringFunc(tmpl, func(match string) string {
		key := match[2 : len(match)-2] // strip {{ and }}
		if val, ok := vars[key]; ok {
			return val
		}
		return match // 未匹配的变量保留原样
	})
}

// SysVarInfo 系统变量元信息
type SysVarInfo struct {
	Name        string `json:"name"`        // 变量名，如 "sys.user_query"
	Description string `json:"description"` // 中文描述
}

// validSysVars 合法的系统变量白名单（sys. 前缀），key=变量名, value=描述
var validSysVars = map[string]string{
	"sys.user_query": "用户当前问题",
	"sys.history":    "历史对话记录",
	"sys.cur_date":   "当前日期 (YYYY-MM-DD)",
	"sys.cur_week":   "当前星期几（中文）",
	"sys.kb_context": "知识库检索结果（由智能体绑定的知识库检索得出）",
	"sys.intent":     "意图分类结果（如有分类器）",
}

// GetSystemVars 返回所有可用的系统变量列表（供前端/第三方调用）
func GetSystemVars() []SysVarInfo {
	result := make([]SysVarInfo, 0, len(validSysVars))
	for name, desc := range validSysVars {
		result = append(result, SysVarInfo{Name: name, Description: desc})
	}
	return result
}

// ValidateTemplateVars 校验模板中引用的系统变量是否合法。
// sys. 前缀的变量必须在白名单中，否则返回非法变量名列表。
// 非 sys. 前缀的变量（如 {{node_1}}、自定义变量）不校验，留给运行时解析。
func ValidateTemplateVars(tmpl string) []string {
	matches := varPattern.FindAllStringSubmatch(tmpl, -1)
	seen := map[string]bool{}
	var invalid []string

	for _, m := range matches {
		varName := m[1]
		if seen[varName] {
			continue
		}
		seen[varName] = true

		// 只校验 sys. 前缀的变量
		if strings.HasPrefix(varName, "sys.") {
			if _, ok := validSysVars[varName]; !ok {
				invalid = append(invalid, varName)
			}
		}
	}
	return invalid
}

// FormatHistory 格式化历史消息为字符串用于模板
func FormatHistory(messages []ChatMsg) string {
	if len(messages) == 0 {
		return "（无历史对话）"
	}

	var b strings.Builder
	for _, msg := range messages {
		if msg.Role == "user" {
			b.WriteString("用户：" + msg.Content + "\n")
		} else {
			b.WriteString("助手：" + msg.Content + "\n")
		}
	}
	return b.String()
}

// getWeekdayCN 返回中文星期
func getWeekdayCN(d time.Weekday) string {
	days := []string{"日", "一", "二", "三", "四", "五", "六"}
	return days[d]
}

// ChatMsg 简化的消息类型（避免循环依赖 model 包）
type ChatMsg struct {
	Role    string
	Content string
}
