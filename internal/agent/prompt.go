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

// buildPromptWithHistory 从模板、issue信息和对话历史构建提示词
func buildPromptWithHistory(issue *domain.Issue, attempt *int, history []domain.ConversationTurn, tmpl string) string {
	// 先构建基础prompt
	prompt := buildPrompt(issue, attempt, tmpl)

	// 如果有对话历史，注入到prompt中
	if len(history) > 0 {
		historySection := formatConversationHistory(history)
		// 检查是否有对话历史占位符
		if strings.Contains(prompt, "{{ conversation_history }}") {
			prompt = strings.ReplaceAll(prompt, "{{ conversation_history }}", historySection)
		} else {
			// 追加到末尾
			prompt = prompt + "\n\n" + historySection
		}
	} else {
		// 移除空的对话历史占位符
		prompt = strings.ReplaceAll(prompt, "{{ conversation_history }}", "")
	}

	return strings.TrimSpace(prompt)
}

// buildPromptWithBDDConstraints 从模板、issue信息和BDD约束构建提示词
// 支持注入 BDD 约束条件到 Agent Prompt
func buildPromptWithBDDConstraints(issue *domain.Issue, attempt *int, bddConstraints string, tmpl string) string {
	// 先构建基础prompt
	prompt := buildPrompt(issue, attempt, tmpl)

	// 如果有 BDD 约束，注入到 prompt 中
	if bddConstraints != "" {
		// 检查是否有 BDD 约束占位符
		if strings.Contains(prompt, "{{ bdd_constraints }}") {
			prompt = strings.ReplaceAll(prompt, "{{ bdd_constraints }}", bddConstraints)
		} else {
			// 没有占位符时，追加到末尾
			prompt = prompt + "\n\n" + bddConstraints
		}
	} else {
		// 移除空的 BDD 约束占位符
		prompt = strings.ReplaceAll(prompt, "{{ bdd_constraints }}", "")
	}

	return strings.TrimSpace(prompt)
}

// buildPromptWithHistoryAndBDD 构建包含历史对话和BDD约束的完整提示词
func buildPromptWithHistoryAndBDD(issue *domain.Issue, attempt *int, history []domain.ConversationTurn, bddConstraints string, tmpl string) string {
	// 先构建包含历史的 prompt
	prompt := buildPromptWithHistory(issue, attempt, history, tmpl)

	// 如果有 BDD 约束，注入到 prompt 中
	if bddConstraints != "" {
		// 检查是否有 BDD 约束占位符
		if strings.Contains(prompt, "{{ bdd_constraints }}") {
			prompt = strings.ReplaceAll(prompt, "{{ bdd_constraints }}", bddConstraints)
		} else {
			// 没有占位符时，追加到末尾
			prompt = prompt + "\n\n" + bddConstraints
		}
	} else {
		// 移除空的 BDD 约束占位符
		prompt = strings.ReplaceAll(prompt, "{{ bdd_constraints }}", "")
	}

	return strings.TrimSpace(prompt)
}

// InjectBDDConstraints 注入 BDD 约束到已有 prompt
// 这是一个便捷方法，用于在已有 prompt 中追加 BDD 约束
func InjectBDDConstraints(prompt string, bddConstraints string) string {
	if bddConstraints == "" {
		return prompt
	}

	// 检查是否有 BDD 约束占位符
	if strings.Contains(prompt, "{{ bdd_constraints }}") {
		return strings.ReplaceAll(prompt, "{{ bdd_constraints }}", bddConstraints)
	}

	// 追加到末尾，添加分隔符
	if prompt != "" {
		return prompt + "\n\n---\n\n" + bddConstraints
	}

	return bddConstraints
}

// formatConversationHistory 格式化对话历史为prompt上下文
func formatConversationHistory(history []domain.ConversationTurn) string {
	var builder strings.Builder

	builder.WriteString("## 需求澄清历史\n\n")

	// 按轮次分组对话
	rounds := groupConversationRounds(history)

	for roundNum, turns := range rounds {
		builder.WriteString(fmt.Sprintf("### Round %d\n", roundNum))
		for _, turn := range turns {
			roleLabel := getRoleLabel(turn.Role)
			builder.WriteString(fmt.Sprintf("**%s:** %s\n", roleLabel, turn.Content))
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

// groupConversationRounds 将对话历史按轮次分组
func groupConversationRounds(history []domain.ConversationTurn) map[int][]domain.ConversationTurn {
	rounds := make(map[int][]domain.ConversationTurn)
	currentRound := 1

	for _, turn := range history {
		// 每当角色从 assistant 变为 user 时，增加轮次
		// 这表示一个新的澄清回合开始
		if len(rounds[currentRound]) > 0 {
			lastTurn := rounds[currentRound][len(rounds[currentRound])-1]
			if lastTurn.Role == "assistant" && turn.Role == "user" {
				currentRound++
			}
		}
		rounds[currentRound] = append(rounds[currentRound], turn)
	}

	return rounds
}

// getRoleLabel 获取角色的显示标签
func getRoleLabel(role string) string {
	switch role {
	case "user":
		return "User"
	case "assistant":
		return "Assistant"
	default:
		return role
	}
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
