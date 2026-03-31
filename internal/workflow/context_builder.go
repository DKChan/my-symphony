// Package workflow 提供AI Agent上下文构建功能
package workflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/dministrator/symphony/internal/domain"
)

// ContextBuilder 构建AI Agent的prompt上下文
type ContextBuilder struct {
	constraintManager *ConstraintManager
}

// NewContextBuilder 创建新的上下文构建器
func NewContextBuilder() *ContextBuilder {
	return &ContextBuilder{}
}

// NewContextBuilderWithConstraints 创建带约束管理器的上下文构建器
func NewContextBuilderWithConstraints(cm *ConstraintManager) *ContextBuilder {
	return &ContextBuilder{
		constraintManager: cm,
	}
}

// SetConstraintManager 设置约束管理器（用于依赖注入）
func (b *ContextBuilder) SetConstraintManager(cm *ConstraintManager) {
	b.constraintManager = cm
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

// BuildImplementationContext 构建实现阶段的上下文
// Story 4.4: 审核通过的 BDD 规则作为约束条件传递给 AI Agent
func (b *ContextBuilder) BuildImplementationContext(task *domain.Issue, history []domain.ConversationTurn, bddConstraints *BDDConstraints) string {
	var builder strings.Builder

	// 添加任务基本信息
	builder.WriteString(b.formatTaskInfo(task))

	// 如果有对话历史，添加格式化的对话历史
	if len(history) > 0 {
		builder.WriteString("\n")
		builder.WriteString(b.formatConversationHistory(history))
	}

	// 如果有 BDD 约束，添加约束条件
	if bddConstraints != nil && len(bddConstraints.Scenarios) > 0 {
		builder.WriteString("\n")
		builder.WriteString(b.formatBDDConstraints(bddConstraints))
	}

	return builder.String()
}

// formatBDDConstraints 格式化 BDD 约束条件为 prompt
func (b *ContextBuilder) formatBDDConstraints(constraints *BDDConstraints) string {
	var builder strings.Builder

	builder.WriteString("## BDD 验收标准\n\n")
	builder.WriteString("**重要：以下 BDD 场景必须全部通过，实现必须满足这些验收标准。**\n\n")

	// Feature 信息
	if constraints.Feature.Name != "" {
		builder.WriteString(fmt.Sprintf("**功能名称:** %s\n\n", constraints.Feature.Name))
	}
	if constraints.Feature.Description != "" {
		builder.WriteString(fmt.Sprintf("**功能描述:** %s\n\n", constraints.Feature.Description))
	}

	// 场景列表
	for i, scenario := range constraints.Scenarios {
		builder.WriteString(fmt.Sprintf("### 场景 %d: %s\n\n", i+1, scenario.Name))

		// Given
		if len(scenario.Given) > 0 {
			builder.WriteString("**前置条件 (Given):**\n")
			for _, given := range scenario.Given {
				builder.WriteString(fmt.Sprintf("- Given: %s\n", given))
			}
			builder.WriteString("\n")
		}

		// When
		if len(scenario.When) > 0 {
			builder.WriteString("**触发动作 (When):**\n")
			for _, when := range scenario.When {
				builder.WriteString(fmt.Sprintf("- When: %s\n", when))
			}
			builder.WriteString("\n")
		}

		// Then
		if len(scenario.Then) > 0 {
			builder.WriteString("**预期结果 (Then):**\n")
			for _, then := range scenario.Then {
				builder.WriteString(fmt.Sprintf("- Then: %s\n", then))
			}
			builder.WriteString("\n")
		}

		// Tags
		if len(scenario.Tags) > 0 {
			builder.WriteString(fmt.Sprintf("**标签:** %s\n\n", strings.Join(scenario.Tags, ", ")))
		}
	}

	// Summary
	if constraints.Summary != "" {
		builder.WriteString(fmt.Sprintf("**规则摘要:** %s\n", constraints.Summary))
	}

	return builder.String()
}

// BuildImplementationContextFromManager 从约束管理器构建实现上下文
// 便捷方法，自动从约束管理器加载 BDD 约束
func (b *ContextBuilder) BuildImplementationContextFromManager(ctx context.Context, taskID string, task *domain.Issue, history []domain.ConversationTurn) (string, error) {
	var constraints *BDDConstraints

	// 如果有约束管理器，尝试加载 BDD 约束
	if b.constraintManager != nil {
		var err error
		constraints, err = b.constraintManager.LoadBDDConstraints(taskID)
		if err != nil {
			// 加载失败，继续执行但不包含约束
			constraints = nil
		}
	}

	return b.BuildImplementationContext(task, history, constraints), nil
}

// GetBDDConstraintsForTask 获取任务的 BDD 约束
func (b *ContextBuilder) GetBDDConstraintsForTask(taskID string) (*BDDConstraints, error) {
	if b.constraintManager == nil {
		return nil, nil
	}
	return b.constraintManager.LoadBDDConstraints(taskID)
}

// GetBDDConstraintsPath 获取任务的 BDD 约束文件路径
func (b *ContextBuilder) GetBDDConstraintsPath(taskID string) string {
	if b.constraintManager == nil {
		return ""
	}
	return b.constraintManager.GetConstraintFilePath(taskID)
}