// Package workflow - 上下文构建测试
package workflow

import (
	"testing"
	"time"

	"github.com/dministrator/symphony/internal/domain"
	"github.com/stretchr/testify/assert"
)

// TestContextBuilder_BuildAgentContext 测试构建Agent上下文
func TestContextBuilder_BuildAgentContext(t *testing.T) {
	builder := NewContextBuilder()

	tests := []struct {
		name     string
		task     *domain.Issue
		history  []domain.ConversationTurn
		expected []string // 检查是否包含这些内容
	}{
		{
			name: "仅有任务信息",
			task: &domain.Issue{
				Identifier:  "TEST-1",
				Title:       "添加用户登录功能",
				State:       "Todo",
				Description: stringPtr("实现邮箱登录"),
				Priority:    intPtr(1),
				Labels:      []string{"feature", "auth"},
			},
			history: nil,
			expected: []string{
				"## 任务信息",
				"**任务标识符:** TEST-1",
				"**任务标题:** 添加用户登录功能",
				"**当前状态:** Todo",
				"**任务描述:** 实现邮箱登录",
				"**优先级:** 1",
				"**标签:** feature, auth",
			},
		},
		{
			name: "包含对话历史",
			task: &domain.Issue{
				Identifier: "TEST-2",
				Title:      "实现支付功能",
				State:      "In Progress",
			},
			history: []domain.ConversationTurn{
				{Role: "user", Content: "想要添加用户登录功能", Timestamp: time.Now()},
				{Role: "assistant", Content: "请问登录方式是邮箱还是手机号？", Timestamp: time.Now()},
				{Role: "user", Content: "邮箱", Timestamp: time.Now()},
				{Role: "assistant", Content: "请问是否需要支持第三方登录？", Timestamp: time.Now()},
				{Role: "user", Content: "暂时不需要", Timestamp: time.Now()},
			},
			expected: []string{
				"## 任务信息",
				"**任务标识符:** TEST-2",
				"## 需求澄清历史",
				"### Round 1",
				"**User:** 想要添加用户登录功能",
				"**Assistant:** 请问登录方式是邮箱还是手机号？",
				"### Round 2",
				"**User:** 邮箱",
				"**Assistant:** 请问是否需要支持第三方登录？",
				"### Round 3",
				"**User:** 暂时不需要",
			},
		},
		{
			name: "空对话历史",
			task: &domain.Issue{
				Identifier: "TEST-3",
				Title:      "修复Bug",
				State:      "Done",
			},
			history: []domain.ConversationTurn{},
			expected: []string{
				"## 任务信息",
				"**任务标识符:** TEST-3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builder.BuildAgentContext(tt.task, tt.history)

			for _, expected := range tt.expected {
				assert.Contains(t, result, expected)
			}

			// 空历史不应该包含对话历史部分
			if len(tt.history) == 0 {
				assert.NotContains(t, result, "## 需求澄清历史")
			}
		})
	}
}

// TestContextBuilder_FormatConversationHistory 测试格式化对话历史
func TestContextBuilder_FormatConversationHistory(t *testing.T) {
	builder := NewContextBuilder()

	t.Run("单轮对话", func(t *testing.T) {
		history := []domain.ConversationTurn{
			{Role: "user", Content: "Hello", Timestamp: time.Now()},
			{Role: "assistant", Content: "Hi there!", Timestamp: time.Now()},
		}

		result := builder.FormatConversationHistoryOnly(history)

		assert.Contains(t, result, "## 需求澄清历史")
		assert.Contains(t, result, "### Round 1")
		assert.Contains(t, result, "**User:** Hello")
		assert.Contains(t, result, "**Assistant:** Hi there!")
	})

	t.Run("多轮对话", func(t *testing.T) {
		history := []domain.ConversationTurn{
			{Role: "user", Content: "Q1", Timestamp: time.Now()},
			{Role: "assistant", Content: "A1", Timestamp: time.Now()},
			{Role: "user", Content: "Q2", Timestamp: time.Now()},
			{Role: "assistant", Content: "A2", Timestamp: time.Now()},
			{Role: "user", Content: "Q3", Timestamp: time.Now()},
		}

		result := builder.FormatConversationHistoryOnly(history)

		assert.Contains(t, result, "### Round 1")
		assert.Contains(t, result, "### Round 2")
		assert.Contains(t, result, "### Round 3")
		assert.Contains(t, result, "**User:** Q1")
		assert.Contains(t, result, "**Assistant:** A1")
		assert.Contains(t, result, "**User:** Q2")
		assert.Contains(t, result, "**Assistant:** A2")
		assert.Contains(t, result, "**User:** Q3")
	})

	t.Run("空历史", func(t *testing.T) {
		result := builder.FormatConversationHistoryOnly(nil)
		assert.Empty(t, result)

		result = builder.FormatConversationHistoryOnly([]domain.ConversationTurn{})
		assert.Empty(t, result)
	})
}

// TestContextBuilder_InjectHistoryIntoPrompt 测试注入对话历史到模板
func TestContextBuilder_InjectHistoryIntoPrompt(t *testing.T) {
	builder := NewContextBuilder()

	t.Run("带有占位符", func(t *testing.T) {
		template := "任务: {{ issue.title }}\n\n{{ conversation_history }}"
		history := []domain.ConversationTurn{
			{Role: "user", Content: "问题1", Timestamp: time.Now()},
			{Role: "assistant", Content: "回答1", Timestamp: time.Now()},
		}

		result := builder.InjectHistoryIntoPrompt(template, history)

		assert.Contains(t, result, "任务: {{ issue.title }}")
		assert.Contains(t, result, "## 需求澄清历史")
		assert.NotContains(t, result, "{{ conversation_history }}")
	})

	t.Run("无占位符追加到末尾", func(t *testing.T) {
		template := "任务: 执行开发任务"
		history := []domain.ConversationTurn{
			{Role: "user", Content: "澄清内容", Timestamp: time.Now()},
		}

		result := builder.InjectHistoryIntoPrompt(template, history)

		assert.Contains(t, result, "任务: 执行开发任务")
		assert.Contains(t, result, "## 需求澄清历史")
	})

	t.Run("空历史移除占位符", func(t *testing.T) {
		template := "任务描述\n\n{{ conversation_history }}\n开始执行"

		result := builder.InjectHistoryIntoPrompt(template, nil)

		assert.Contains(t, result, "任务描述")
		assert.NotContains(t, result, "{{ conversation_history }}")
		assert.Contains(t, result, "开始执行")
	})
}

// TestContextBuilder_groupConversationRounds 测试对话轮次分组
func TestContextBuilder_groupConversationRounds(t *testing.T) {
	builder := NewContextBuilder()

	tests := []struct {
		name          string
		history       []domain.ConversationTurn
		expectedCount int // 期望的轮次数量
	}{
		{
			name: "两轮对话",
			history: []domain.ConversationTurn{
				{Role: "user", Content: "Q1"},
				{Role: "assistant", Content: "A1"},
				{Role: "user", Content: "Q2"},
				{Role: "assistant", Content: "A2"},
			},
			expectedCount: 2,
		},
		{
			name: "三轮对话",
			history: []domain.ConversationTurn{
				{Role: "user", Content: "Q1"},
				{Role: "assistant", Content: "A1"},
				{Role: "user", Content: "Q2"},
				{Role: "assistant", Content: "A2"},
				{Role: "user", Content: "Q3"},
				{Role: "assistant", Content: "A3"},
			},
			expectedCount: 3,
		},
		{
			name: "单轮对话",
			history: []domain.ConversationTurn{
				{Role: "user", Content: "Q"},
				{Role: "assistant", Content: "A"},
			},
			expectedCount: 1,
		},
		{
			name: "只有user发言",
			history: []domain.ConversationTurn{
				{Role: "user", Content: "Only question"},
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rounds := builder.groupConversationRounds(tt.history)
			assert.Equal(t, tt.expectedCount, len(rounds))
		})
	}
}

// TestContextBuilder_getRoleLabel 测试角色标签获取
func TestContextBuilder_getRoleLabel(t *testing.T) {
	builder := NewContextBuilder()

	assert.Equal(t, "User", builder.getRoleLabel("user"))
	assert.Equal(t, "Assistant", builder.getRoleLabel("assistant"))
	assert.Equal(t, "unknown", builder.getRoleLabel("unknown"))
}

// 辅助函数
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}