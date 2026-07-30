package engine

import (
	"regexp"
	"strings"
)

// varPattern matches {{variable_name}} patterns
var varPattern = regexp.MustCompile(`\{\{(\w+)\}\}`)

// ResolveTemplate 替换模板中的 {{var}} 占位符
// 内置变量：
//
//	{{user_query}} - 用户原始问题
//	{{history}}    - 历史对话
//	自定义变量来自上游节点的 OutputVar
func ResolveTemplate(tmpl string, vars map[string]string) string {
	return varPattern.ReplaceAllStringFunc(tmpl, func(match string) string {
		key := match[2 : len(match)-2] // strip {{ and }}
		if val, ok := vars[key]; ok {
			return val
		}
		return match // 未匹配的变量保留原样
	})
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

// ChatMsg 简化的消息类型（避免循环依赖 model 包）
type ChatMsg struct {
	Role    string
	Content string
}
