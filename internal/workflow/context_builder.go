// Package workflow 提供AI Agent上下文构建功能
package workflow

import (
	"fmt"
	"strings"

	"github.com/dministrator/symphony/internal/domain"
)

// ContextBuilder 构建AI Agent的prompt上下文
type ContextBuilder struct{}

// NewContextBuilder 创建新的上下文构建器
func NewContextBuilder() *ContextBuilder {
	return &ContextBuilder{}
}

// BuildAgentContext 构建Agent上下文，将对话历史格式化为prompt上下文
func (b *ContextBuilder) BuildAgentContext(task *domain.Issue, history []domain.ConversationTurn) string {
	var builder strings.Builder

	// 添加任务基本信息
	builder.WriteString(b.formatTaskInfo(task))

	// 如果有对话历史，添加格式化的对话历史
	if len(history) > 0 {
		builder.WriteString("\n")
		builder.WriteString(b.formatConversationHistory(history))
	}

	return builder.String()
}

// formatTaskInfo 格式化任务基本信息
func (b *ContextBuilder) formatTaskInfo(task *domain.Issue) string {
	var builder strings.Builder

	builder.WriteString("## 任务信息\n\n")
	builder.WriteString(fmt.Sprintf("**任务标识符:** %s\n", task.Identifier))
	builder.WriteString(fmt.Sprintf("**任务标题:** %s\n", task.Title))
	builder.WriteString(fmt.Sprintf("**当前状态:** %s\n", task.State))

	if task.Description != nil && *task.Description != "" {
		builder.WriteString(fmt.Sprintf("**任务描述:** %s\n", *task.Description))
	}

	if task.Priority != nil {
		builder.WriteString(fmt.Sprintf("**优先级:** %d\n", *task.Priority))
	}

	if len(task.Labels) > 0 {
		builder.WriteString(fmt.Sprintf("**标签:** %s\n", strings.Join(task.Labels, ", ")))
	}

	if task.URL != nil {
		builder.WriteString(fmt.Sprintf("**链接:** %s\n", *task.URL))
	}

	return builder.String()
}

// formatConversationHistory 格式化对话历史为prompt上下文
func (b *ContextBuilder) formatConversationHistory(history []domain.ConversationTurn) string {
	var builder strings.Builder

	builder.WriteString("## 需求澄清历史\n\n")

	// 按轮次分组对话
	rounds := b.groupConversationRounds(history)

	for roundNum, turns := range rounds {
		builder.WriteString(fmt.Sprintf("### Round %d\n", roundNum))
		for _, turn := range turns {
			roleLabel := b.getRoleLabel(turn.Role)
			builder.WriteString(fmt.Sprintf("**%s:** %s\n", roleLabel, turn.Content))
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

// groupConversationRounds 将对话历史按轮次分组
func (b *ContextBuilder) groupConversationRounds(history []domain.ConversationTurn) map[int][]domain.ConversationTurn {
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
func (b *ContextBuilder) getRoleLabel(role string) string {
	switch role {
	case "user":
		return "User"
	case "assistant":
		return "Assistant"
	default:
		return role
	}
}

// FormatConversationHistoryOnly 仅格式化对话历史（不包含任务信息）
func (b *ContextBuilder) FormatConversationHistoryOnly(history []domain.ConversationTurn) string {
	if len(history) == 0 {
		return ""
	}
	return b.formatConversationHistory(history)
}

// InjectHistoryIntoPrompt 将对话历史注入到现有prompt模板中
func (b *ContextBuilder) InjectHistoryIntoPrompt(template string, history []domain.ConversationTurn) string {
	// 检查模板是否已经有对话历史占位符
	if strings.Contains(template, "{{ conversation_history }}") {
		if len(history) == 0 {
			// 空历史时移除占位符
			return strings.ReplaceAll(template, "{{ conversation_history }}", "")
		}
		// 有历史时替换占位符
		historySection := b.formatConversationHistory(history)
		return strings.ReplaceAll(template, "{{ conversation_history }}", historySection)
	}

	// 没有占位符
	if len(history) == 0 {
		// 空历史直接返回模板
		return template
	}

	// 有历史时追加到模板末尾
	historySection := b.formatConversationHistory(history)
	return template + "\n\n" + historySection
}