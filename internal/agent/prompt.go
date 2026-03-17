// Package agent - 共享工具函数
package agent

import (
	"fmt"
	"strings"

	"github.com/dministrator/symphony/internal/domain"
)

// buildPrompt 从模板和 issue 信息构建提示词
// 支持 {{ issue.xxx }} 和 {{ attempt }} 占位符
// 同时处理 {% if attempt %}...{% endif %} 简单条件块
func buildPrompt(issue *domain.Issue, attempt *int, tmpl string) string {
	prompt := tmpl

	// 替换基本字段
	prompt = strings.ReplaceAll(prompt, "{{ issue.id }}", issue.ID)
	prompt = strings.ReplaceAll(prompt, "{{ issue.identifier }}", issue.Identifier)
	prompt = strings.ReplaceAll(prompt, "{{ issue.title }}", issue.Title)
	prompt = strings.ReplaceAll(prompt, "{{ issue.state }}", issue.State)

	desc := ""
	if issue.Description != nil {
		desc = *issue.Description
	}
	prompt = strings.ReplaceAll(prompt, "{{ issue.description }}", desc)

	url := ""
	if issue.URL != nil {
		url = *issue.URL
	}
	prompt = strings.ReplaceAll(prompt, "{{ issue.url }}", url)

	// 替换 attempt
	if attempt != nil {
		prompt = strings.ReplaceAll(prompt, "{{ attempt }}", fmt.Sprintf("%d", *attempt))
		// 保留 {% if attempt %}...{% endif %} 块内容
		prompt = removeBlock(prompt, "{% if attempt %}", "{% endif %}", true)
	} else {
		prompt = strings.ReplaceAll(prompt, "{{ attempt }}", "")
		// 移除 {% if attempt %}...{% endif %} 块
		prompt = removeBlock(prompt, "{% if attempt %}", "{% endif %}", false)
	}

	return strings.TrimSpace(prompt)
}

// removeBlock 处理简单的条件块
// keep=true 时保留块内容（去除标签），keep=false 时移除整个块
func removeBlock(s, startTag, endTag string, keep bool) string {
	for {
		start := strings.Index(s, startTag)
		if start == -1 {
			break
		}
		end := strings.Index(s[start:], endTag)
		if end == -1 {
			break
		}
		end += start + len(endTag)

		inner := s[start+len(startTag) : end-len(endTag)]
		if keep {
			s = s[:start] + strings.TrimSpace(inner) + s[end:]
		} else {
			s = s[:start] + s[end:]
		}
	}
	return s
}
