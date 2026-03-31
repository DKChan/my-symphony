// Package agent - 提示词构建和模板处理测试
package agent

import (
	"strings"
	"testing"
	"time"

	"github.com/dministrator/symphony/internal/domain"
)

// TestBuildPrompt 测试提示词构建函数
func TestBuildPrompt(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		issue    *domain.Issue
		attempt  *int
		template string
		expected string
	}{
		{
			name: "基本字段替换",
			issue: &domain.Issue{
				ID:         "123",
				Identifier: "TEST-1",
				Title:      "Test Issue",
				State:      "Todo",
			},
			attempt:  nil,
			template: "{{ issue.id }} - {{ issue.identifier }} - {{ issue.title }} - {{ issue.state }}",
			expected: "123 - TEST-1 - Test Issue - Todo",
		},
		{
			name: "包含描述字段",
			issue: &domain.Issue{
				ID:          "123",
				Identifier:  "TEST-1",
				Title:       "Test Issue",
				State:       "Todo",
				Description: stringPtr("This is a description"),
			},
			attempt:  nil,
			template: "{{ issue.description }}",
			expected: "This is a description",
		},
		{
			name: "nil 描述字段",
			issue: &domain.Issue{
				ID:         "123",
				Identifier: "TEST-1",
				Title:      "Test Issue",
				State:      "Todo",
			},
			attempt:  nil,
			template: "Desc: {{ issue.description }}",
			expected: "Desc:",
		},
		{
			name: "包含 URL 字段",
			issue: &domain.Issue{
				ID:         "123",
				Identifier: "TEST-1",
				Title:      "Test Issue",
				State:      "Todo",
				URL:        stringPtr("https://example.com/issue/123"),
			},
			attempt:  nil,
			template: "URL: {{ issue.url }}",
			expected: "URL: https://example.com/issue/123",
		},
		{
			name: "nil URL 字段",
			issue: &domain.Issue{
				ID:         "123",
				Identifier: "TEST-1",
				Title:      "Test Issue",
				State:      "Todo",
			},
			attempt:  nil,
			template: "URL: {{ issue.url }}",
			expected: "URL:",
		},
		{
			name: "包含 attempt 参数",
			issue: &domain.Issue{
				ID:         "123",
				Identifier: "TEST-1",
				Title:      "Test Issue",
				State:      "Todo",
			},
			attempt:  intPtr(1),
			template: "Attempt {{ attempt }}",
			expected: "Attempt 1",
		},
		{
			name: "nil attempt 参数",
			issue: &domain.Issue{
				ID:         "123",
				Identifier: "TEST-1",
				Title:      "Test Issue",
				State:      "Todo",
			},
			attempt:  nil,
			template: "Attempt {{ attempt }}",
			expected: "Attempt",
		},
		{
			name: "保留条件块内容（有 attempt）",
			issue: &domain.Issue{
				ID:         "123",
				Identifier: "TEST-1",
				Title:      "Test Issue",
				State:      "Todo",
			},
			attempt:  intPtr(1),
			template: "Start{% if attempt %}This is attempt {{ attempt }}{% endif %}End",
			expected: "StartThis is attempt 1End",
		},
		{
			name: "移除条件块（无 attempt）",
			issue: &domain.Issue{
				ID:         "123",
				Identifier: "TEST-1",
				Title:      "Test Issue",
				State:      "Todo",
			},
			attempt:  nil,
			template: "Start{% if attempt %}This is attempt {{ attempt }}{% endif %}End",
			expected: "StartEnd",
		},
		{
			name: "复杂模板",
			issue: &domain.Issue{
				ID:          "123",
				Identifier:  "TEST-1",
				Title:       "Test Issue",
				State:       "Todo",
				Description: stringPtr("Description here"),
				URL:         stringPtr("https://example.com/123"),
				CreatedAt:   &now,
			},
			attempt: intPtr(2),
			template: `Issue {{ issue.identifier }} ({{ issue.id }}): {{ issue.title }}
State: {{ issue.state }}
Description: {{ issue.description }}
URL: {{ issue.url }}
{% if attempt %}This is attempt {{ attempt }}{% endif %}`,
			expected: `Issue TEST-1 (123): Test Issue
State: Todo
Description: Description here
URL: https://example.com/123
This is attempt 2`,
		},
		{
			name: "多个条件块",
			issue: &domain.Issue{
				ID:         "123",
				Identifier: "TEST-1",
				Title:      "Test Issue",
				State:      "Todo",
			},
			attempt: intPtr(1),
			template: "{% if attempt %}Block 1: {{ attempt }}{% endif %}Middle{% if attempt %}Block 2: {{ attempt }}{% endif %}",
			expected: "Block 1: 1MiddleBlock 2: 1",
		},
		{
			name: "没有占位符的模板",
			issue: &domain.Issue{
				ID:         "123",
				Identifier: "TEST-1",
				Title:      "Test Issue",
				State:      "Todo",
			},
			attempt:  nil,
			template: "This is just static text",
			expected: "This is just static text",
		},
		{
			name: "空模板",
			issue: &domain.Issue{
				ID:         "123",
				Identifier: "TEST-1",
				Title:      "Test Issue",
				State:      "Todo",
			},
			attempt:  nil,
			template: "",
			expected: "",
		},
		{
			name: "包含换行符",
			issue: &domain.Issue{
				ID:         "123",
				Identifier: "TEST-1",
				Title:      "Test Issue",
				State:      "Todo",
			},
			attempt:  nil,
			template: "{{ issue.id }}\n{{ issue.identifier }}\n{{ issue.title }}",
			expected: "123\nTEST-1\nTest Issue",
		},
		{
			name: "包含多个相同占位符",
			issue: &domain.Issue{
				ID:         "123",
				Identifier: "TEST-1",
				Title:      "Test Issue",
				State:      "Todo",
			},
			attempt:  nil,
			template: "{{ issue.id }} and {{ issue.id }} again",
			expected: "123 and 123 again",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildPrompt(tt.issue, tt.attempt, tt.template)
			if result != tt.expected {
				t.Errorf("buildPrompt() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestRemoveBlock 测试条件块处理函数
func TestRemoveBlock(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		startTag string
		endTag   string
		keep     bool
		expected string
	}{
		{
			name:     "保留块内容",
			s:        "Start{% if attempt %}Content{% endif %}End",
			startTag: "{% if attempt %}",
			endTag:   "{% endif %}",
			keep:     true,
			expected: "StartContentEnd",
		},
		{
			name:     "移除整个块",
			s:        "Start{% if attempt %}Content{% endif %}End",
			startTag: "{% if attempt %}",
			endTag:   "{% endif %}",
			keep:     false,
			expected: "StartEnd",
		},
		{
			name:     "多个块全部保留",
			s:        "{% if a %}A{% endif %} and {% if a %}B{% endif %}",
			startTag: "{% if a %}",
			endTag:   "{% endif %}",
			keep:     true,
			expected: "A and B",
		},
		{
			name:     "多个块全部移除",
			s:        "{% if a %}A{% endif %} and {% if a %}B{% endif %}",
			startTag: "{% if a %}",
			endTag:   "{% endif %}",
			keep:     false,
			expected: " and ",
		},
		{
			name:     "块内有换行符",
			s:        "Start{% if a %}\nLine 1\nLine 2\n{% endif %}End",
			startTag: "{% if a %}",
			endTag:   "{% endif %}",
			keep:     true,
			expected: "StartLine 1\nLine 2End",
		},
		{
			name:     "空内容块保留",
			s:        "Start{% if a %}{% endif %}End",
			startTag: "{% if a %}",
			endTag:   "{% endif %}",
			keep:     true,
			expected: "StartEnd",
		},
		{
			name:     "空内容块移除",
			s:        "Start{% if a %}{% endif %}End",
			startTag: "{% if a %}",
			endTag:   "{% endif %}",
			keep:     false,
			expected: "StartEnd",
		},
		{
			name:     "块在开头",
			s:        "{% if a %}Content{% endif %}End",
			startTag: "{% if a %}",
			endTag:   "{% endif %}",
			keep:     true,
			expected: "ContentEnd",
		},
		{
			name:     "块在结尾",
			s:        "Start{% if a %}Content{% endif %}",
			startTag: "{% if a %}",
			endTag:   "{% endif %}",
			keep:     true,
			expected: "StartContent",
		},
		{
			name:     "只有一个块",
			s:        "{% if a %}Content{% endif %}",
			startTag: "{% if a %}",
			endTag:   "{% endif %}",
			keep:     true,
			expected: "Content",
		},
		{
			name:     "没有匹配的块",
			s:        "No blocks here",
			startTag: "{% if a %}",
			endTag:   "{% endif %}",
			keep:     true,
			expected: "No blocks here",
		},
		{
			name:     "只有开始标签没有结束标签",
			s:        "Start{% if a %}Content",
			startTag: "{% if a %}",
			endTag:   "{% endif %}",
			keep:     true,
			expected: "Start{% if a %}Content",
		},
		{
			name:     "只有结束标签没有开始标签",
			s:        "Start{% endif %}Content",
			startTag: "{% if a %}",
			endTag:   "{% endif %}",
			keep:     true,
			expected: "Start{% endif %}Content",
		},
		{
			name:     "嵌套块保留",
			s:        "Start{% if a %}Outer{% if b %}Inner{% endif %}{% endif %}End",
			startTag: "{% if a %}",
			endTag:   "{% endif %}",
			keep:     true,
			expected: "StartOuter{% if b %}Inner{% endif %}End",
		},
		{
			name:     "块内容包含空格",
			s:        "Start{% if a %}  Content with spaces  {% endif %}End",
			startTag: "{% if a %}",
			endTag:   "{% endif %}",
			keep:     true,
			expected: "StartContent with spacesEnd",
		},
		{
			name:     "块内容包含制表符",
			s:        "Start{% if a %}\t\tContent\t\t{% endif %}End",
			startTag: "{% if a %}",
			endTag:   "{% endif %}",
			keep:     true,
			expected: "StartContentEnd",
		},
		{
			name:     "多个块保留",
			s:        "{% if a %}A{% endif %}1{% if a %}B{% endif %}2{% if a %}C{% endif %}",
			startTag: "{% if a %}",
			endTag:   "{% endif %}",
			keep:     true,
			expected: "A1B2C",
		},
		{
			name:     "自定义标签",
			s:        "Start<!--test-->Content<!--endtest-->End",
			startTag: "<!--test-->",
			endTag:   "<!--endtest-->",
			keep:     true,
			expected: "StartContentEnd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeBlock(tt.s, tt.startTag, tt.endTag, tt.keep)
			if result != tt.expected {
				t.Errorf("removeBlock() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestBuildPromptEdgeCases 测试 buildPrompt 边界情况
func TestBuildPromptEdgeCases(t *testing.T) {
	now := time.Now()

	t.Run("nil issue 基本字段", func(t *testing.T) {
		// nil issue 会触发 panic，这是预期的行为
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic with nil issue")
			}
		}()
		buildPrompt(nil, nil, "test")
	})

	t.Run("零值 issue 字段", func(t *testing.T) {
		issue := &domain.Issue{
			ID:         "",
			Identifier: "",
			Title:      "",
			State:      "",
		}
		result := buildPrompt(issue, nil, "{{ issue.id }}-{{ issue.identifier }}-{{ issue.title }}-{{ issue.state }}")
		// 4个空字段用 "-" 连接，得到 3 个 "-"
		if result != "---" {
			t.Errorf("expected '---', got %q", result)
		}
	})

	t.Run("attempt 为 0", func(t *testing.T) {
		issue := &domain.Issue{
			ID:         "123",
			Identifier: "TEST-1",
			Title:      "Test Issue",
			State:      "Todo",
		}
		attempt := 0
		result := buildPrompt(issue, &attempt, "Attempt {{ attempt }}")
		if result != "Attempt 0" {
			t.Errorf("expected 'Attempt 0', got %q", result)
		}
	})

	t.Run("负数 attempt", func(t *testing.T) {
		issue := &domain.Issue{
			ID:         "123",
			Identifier: "TEST-1",
			Title:      "Test Issue",
			State:      "Todo",
		}
		attempt := -1
		result := buildPrompt(issue, &attempt, "Attempt {{ attempt }}")
		if result != "Attempt -1" {
			t.Errorf("expected 'Attempt -1', got %q", result)
		}
	})

	t.Run("大数字 attempt", func(t *testing.T) {
		issue := &domain.Issue{
			ID:         "123",
			Identifier: "TEST-1",
			Title:      "Test Issue",
			State:      "Todo",
		}
		attempt := 999999
		result := buildPrompt(issue, &attempt, "Attempt {{ attempt }}")
		if result != "Attempt 999999" {
			t.Errorf("expected 'Attempt 999999', got %q", result)
		}
	})

	t.Run("描述包含特殊字符", func(t *testing.T) {
		issue := &domain.Issue{
			ID:          "123",
			Identifier:  "TEST-1",
			Title:       "Test Issue",
			State:       "Todo",
			Description: stringPtr("Line 1\nLine 2\nLine 3"),
		}
		result := buildPrompt(issue, nil, "{{ issue.description }}")
		if result != "Line 1\nLine 2\nLine 3" {
			t.Errorf("unexpected result: %q", result)
		}
	})

	t.Run("URL 包含特殊字符", func(t *testing.T) {
		issue := &domain.Issue{
			ID:         "123",
			Identifier: "TEST-1",
			Title:      "Test Issue",
			State:      "Todo",
			URL:        stringPtr("https://example.com/path?param=value&other=123"),
		}
		result := buildPrompt(issue, nil, "{{ issue.url }}")
		if result != "https://example.com/path?param=value&other=123" {
			t.Errorf("unexpected result: %q", result)
		}
	})

	t.Run("条件块内包含占位符", func(t *testing.T) {
		issue := &domain.Issue{
			ID:         "123",
			Identifier: "TEST-1",
			Title:      "Test Issue",
			State:      "Todo",
		}
		attempt := 1
		result := buildPrompt(issue, &attempt, "{% if attempt %}{{ issue.id }}-{{ issue.identifier }}{% endif %}")
		if result != "123-TEST-1" {
			t.Errorf("expected '123-TEST-1', got %q", result)
		}
	})

	t.Run("字符串首尾空格处理", func(t *testing.T) {
		issue := &domain.Issue{
			ID:         "123",
			Identifier: "TEST-1",
			Title:      "Test Issue",
			State:      "Todo",
		}
		result := buildPrompt(issue, nil, "  {{ issue.id }}  ")
		if result != "123" {
			t.Errorf("expected trimmed '123', got %q", result)
		}
	})

	t.Run("CreatedAt 字段不会被替换", func(t *testing.T) {
		issue := &domain.Issue{
			ID:         "123",
			Identifier: "TEST-1",
			Title:      "Test Issue",
			State:      "Todo",
			CreatedAt:  &now,
		}
		result := buildPrompt(issue, nil, "{{ issue.created_at }}")
		if result != "{{ issue.created_at }}" {
			t.Errorf("created_at should not be replaced, got %q", result)
		}
	})

	t.Run("UpdatedAt 字段不会被替换", func(t *testing.T) {
		issue := &domain.Issue{
			ID:         "123",
			Identifier: "TEST-1",
			Title:      "Test Issue",
			State:      "Todo",
			UpdatedAt:  &now,
		}
		result := buildPrompt(issue, nil, "{{ issue.updated_at }}")
		if result != "{{ issue.updated_at }}" {
			t.Errorf("updated_at should not be replaced, got %q", result)
		}
	})
}

// TestRemoveBlockEdgeCases 测试 removeBlock 边界情况
func TestRemoveBlockEdgeCases(t *testing.T) {
	t.Run("空字符串", func(t *testing.T) {
		result := removeBlock("", "{% if a %}", "{% endif %}", true)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("只有开始标签", func(t *testing.T) {
		result := removeBlock("{% if a %}", "{% if a %}", "{% endif %}", true)
		if result != "{% if a %}" {
			t.Errorf("expected unchanged, got %q", result)
		}
	})

	t.Run("只有结束标签", func(t *testing.T) {
		result := removeBlock("{% endif %}", "{% if a %}", "{% endif %}", true)
		if result != "{% endif %}" {
			t.Errorf("expected unchanged, got %q", result)
		}
	})

	t.Run("标签不存在于字符串", func(t *testing.T) {
		result := removeBlock("Content", "{% if a %}", "{% endif %}", true)
		if result != "Content" {
			t.Errorf("expected unchanged, got %q", result)
		}
	})

	t.Run("块内容非常长", func(t *testing.T) {
		longContent := ""
		for i := 0; i < 10000; i++ {
			longContent += "A"
		}
		result := removeBlock("Start{% if a %}"+longContent+"{% endif %}End",
			"{% if a %}", "{% endif %}", true)
		if len(result) != len("Start"+longContent+"End") {
			t.Errorf("unexpected length: %d", len(result))
		}
	})

	t.Run("块内容包含结束标签", func(t *testing.T) {
		result := removeBlock("Start{% if a %}Text with {% endif %} inside{% endif %}End",
			"{% if a %}", "{% endif %}", true)
		// 应该匹配第一个开始标签和第一个结束标签
		// 输入: Start{% if a %}Text with {% endif %} inside{% endif %}End
		// inner = "Text with " (从 15 到 29)
		// TrimSpace(inner) = "Text with"
		// 结果: StartText with inside{% endif %}End
		expected := "StartText with inside" + "{% endif %}" + "End"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("块内容包含开始标签", func(t *testing.T) {
		result := removeBlock("Start{% if a %}{% if a %}nested{% endif %}{% endif %}End",
			"{% if a %}", "{% endif %}", false)
		// 应该从第一个开始标签匹配到第一个结束标签并移除
		// 输入: Start{% if a %}{% if a %}nested{% endif %}{% endif %}End
		// 结果: Start{% endif %}End
		expected := "Start" + "{% endif %}" + "End"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}

// 辅助函数：字符串指针
func stringPtr(s string) *string {
	return &s
}

// 辅助函数：整数指针
func intPtr(i int) *int {
	return &i
}

// TestBuildPromptWithHistory 测试带对话历史的提示词构建
func TestBuildPromptWithHistory(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		issue    *domain.Issue
		attempt  *int
		history  []domain.ConversationTurn
		template string
		expected []string // 检查是否包含这些内容
	}{
		{
			name: "带对话历史占位符",
			issue: &domain.Issue{
				ID:         "123",
				Identifier: "TEST-1",
				Title:      "开发登录功能",
				State:      "Todo",
			},
			attempt: nil,
			history: []domain.ConversationTurn{
				{Role: "user", Content: "需要登录功能", Timestamp: now},
				{Role: "assistant", Content: "使用什么登录方式？", Timestamp: now},
				{Role: "user", Content: "邮箱登录", Timestamp: now},
			},
			template: "任务: {{ issue.title }}\n\n{{ conversation_history }}",
			expected: []string{
				"任务: 开发登录功能",
				"## 需求澄清历史",
				"**User:** 需要登录功能",
				"**Assistant:** 使用什么登录方式？",
				"**User:** 邮箱登录",
			},
		},
		{
			name: "无占位符自动追加",
			issue: &domain.Issue{
				ID:         "124",
				Identifier: "TEST-2",
				Title:      "实现支付",
				State:      "In Progress",
			},
			attempt: intPtr(1),
			history: []domain.ConversationTurn{
				{Role: "user", Content: "支付方式选择", Timestamp: now},
				{Role: "assistant", Content: "建议支付宝", Timestamp: now},
			},
			template: "任务 {{ issue.identifier }}: {{ issue.title }} (attempt {{ attempt }})",
			expected: []string{
				"任务 TEST-2: 实现支付",
				"attempt 1",
				"## 需求澄清历史",
				"**User:** 支付方式选择",
				"**Assistant:** 建议支付宝",
			},
		},
		{
			name: "空历史移除占位符",
			issue: &domain.Issue{
				ID:         "125",
				Identifier: "TEST-3",
				Title:      "简单任务",
				State:      "Todo",
			},
			attempt:  nil,
			history:  nil,
			template: "任务: {{ issue.title }}\n{{ conversation_history }}\n开始执行",
			expected: []string{
				"任务: 简单任务",
				"开始执行",
			},
		},
		{
			name: "空历史不添加对话部分",
			issue: &domain.Issue{
				ID:         "126",
				Identifier: "TEST-4",
				Title:      "无澄清任务",
				State:      "Todo",
			},
			attempt:  nil,
			history:  []domain.ConversationTurn{},
			template: "任务: {{ issue.title }}",
			expected: []string{
				"任务: 无澄清任务",
			},
		},
		{
			name: "多轮对话按轮次分组",
			issue: &domain.Issue{
				ID:         "127",
				Identifier: "TEST-5",
				Title:      "复杂需求",
				State:      "Todo",
			},
			attempt:  nil,
			history: []domain.ConversationTurn{
				{Role: "user", Content: "需求A", Timestamp: now},
				{Role: "assistant", Content: "澄清A", Timestamp: now},
				{Role: "user", Content: "需求B", Timestamp: now},
				{Role: "assistant", Content: "澄清B", Timestamp: now},
				{Role: "user", Content: "需求C", Timestamp: now},
			},
			template: "{{ conversation_history }}",
			expected: []string{
				"### Round 1",
				"### Round 2",
				"### Round 3",
				"**User:** 需求A",
				"**Assistant:** 澄清A",
				"**User:** 需求B",
				"**Assistant:** 澄清B",
				"**User:** 需求C",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildPromptWithHistory(tt.issue, tt.attempt, tt.history, tt.template)

			for _, expected := range tt.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("buildPromptWithHistory() result missing expected content: %q\nResult:\n%s", expected, result)
				}
			}

			// 空历史不应该包含对话历史部分
			if len(tt.history) == 0 && strings.Contains(result, "## 需求澄清历史") {
				t.Errorf("buildPromptWithHistory() should not include conversation history section for empty history")
			}
		})
	}
}

// TestFormatConversationHistory 测试对话历史格式化
func TestFormatConversationHistory(t *testing.T) {
	now := time.Now()

	t.Run("格式化单轮对话", func(t *testing.T) {
		history := []domain.ConversationTurn{
			{Role: "user", Content: "想要添加用户登录功能", Timestamp: now},
			{Role: "assistant", Content: "请问登录方式是邮箱还是手机号？", Timestamp: now},
			{Role: "user", Content: "邮箱", Timestamp: now},
		}

		result := formatConversationHistory(history)

		if !strings.Contains(result, "## 需求澄清历史") {
			t.Error("missing header")
		}
		if !strings.Contains(result, "### Round 1") {
			t.Error("missing round 1 header")
		}
		if !strings.Contains(result, "**User:** 想要添加用户登录功能") {
			t.Error("missing user message")
		}
		if !strings.Contains(result, "**Assistant:** 请问登录方式是邮箱还是手机号？") {
			t.Error("missing assistant message")
		}
	})

	t.Run("格式化多轮对话", func(t *testing.T) {
		history := []domain.ConversationTurn{
			{Role: "user", Content: "问题1", Timestamp: now},
			{Role: "assistant", Content: "回答1", Timestamp: now},
			{Role: "user", Content: "问题2", Timestamp: now},
			{Role: "assistant", Content: "回答2", Timestamp: now},
			{Role: "user", Content: "问题3", Timestamp: now},
		}

		result := formatConversationHistory(history)

		if !strings.Contains(result, "### Round 1") {
			t.Error("missing round 1")
		}
		if !strings.Contains(result, "### Round 2") {
			t.Error("missing round 2")
		}
		if !strings.Contains(result, "### Round 3") {
			t.Error("missing round 3")
		}
	})
}

// TestGroupConversationRounds 测试对话轮次分组
func TestGroupConversationRounds(t *testing.T) {
	now := time.Now()

	t.Run("assistant-user切换增加轮次", func(t *testing.T) {
		history := []domain.ConversationTurn{
			{Role: "user", Content: "Q1", Timestamp: now},
			{Role: "assistant", Content: "A1", Timestamp: now},
			{Role: "user", Content: "Q2", Timestamp: now},
			{Role: "assistant", Content: "A2", Timestamp: now},
		}

		rounds := groupConversationRounds(history)

		if len(rounds) != 2 {
			t.Errorf("expected 2 rounds, got %d", len(rounds))
		}
		if len(rounds[1]) != 2 {
			t.Errorf("expected 2 turns in round 1, got %d", len(rounds[1]))
		}
		if len(rounds[2]) != 2 {
			t.Errorf("expected 2 turns in round 2, got %d", len(rounds[2]))
		}
	})

	t.Run("单轮对话", func(t *testing.T) {
		history := []domain.ConversationTurn{
			{Role: "user", Content: "Q", Timestamp: now},
			{Role: "assistant", Content: "A", Timestamp: now},
		}

		rounds := groupConversationRounds(history)

		if len(rounds) != 1 {
			t.Errorf("expected 1 round, got %d", len(rounds))
		}
	})

	t.Run("空历史", func(t *testing.T) {
		rounds := groupConversationRounds(nil)

		if len(rounds) != 0 {
			t.Errorf("expected 0 rounds for nil history, got %d", len(rounds))
		}
	})
}

// TestGetRoleLabel 测试角色标签
func TestGetRoleLabel(t *testing.T) {
	if getRoleLabel("user") != "User" {
		t.Error("user label incorrect")
	}
	if getRoleLabel("assistant") != "Assistant" {
		t.Error("assistant label incorrect")
	}
	if getRoleLabel("other") != "other" {
		t.Error("other role should remain unchanged")
	}
}

// TestBuildPromptWithBDDConstraints 测试带 BDD 约束的提示词构建
func TestBuildPromptWithBDDConstraints(t *testing.T) {
	_ = time.Now() // 使用 time 包但不直接使用变量

	tests := []struct {
		name           string
		issue          *domain.Issue
		attempt        *int
		bddConstraints string
		template       string
		expectedContains []string
		notExpected    []string
	}{
		{
			name: "带 BDD 约束占位符",
			issue: &domain.Issue{
				ID:         "123",
				Identifier: "TEST-1",
				Title:      "开发登录功能",
				State:      "Todo",
			},
			attempt: nil,
			bddConstraints: "## BDD 验收标准\n\n### Scenario 1: 登录成功\n- Given: 用户在登录页\n- When: 点击登录\n- Then: 跳转首页",
			template: "任务: {{ issue.title }}\n\n{{ bdd_constraints }}",
			expectedContains: []string{
				"任务: 开发登录功能",
				"## BDD 验收标准",
				"### Scenario 1: 登录成功",
				"- Given: 用户在登录页",
			},
		},
		{
			name: "无占位符自动追加 BDD 约束",
			issue: &domain.Issue{
				ID:         "124",
				Identifier: "TEST-2",
				Title:      "实现支付",
				State:      "In Progress",
			},
			attempt: intPtr(1),
			bddConstraints: "## BDD 验收标准\n\n重要约束",
			template: "任务 {{ issue.identifier }}: {{ issue.title }} (attempt {{ attempt }})",
			expectedContains: []string{
				"任务 TEST-2: 实现支付",
				"attempt 1",
				"## BDD 验收标准",
				"重要约束",
			},
		},
		{
			name: "空 BDD 约束移除占位符",
			issue: &domain.Issue{
				ID:         "125",
				Identifier: "TEST-3",
				Title:      "简单任务",
				State:      "Todo",
			},
			attempt:        nil,
			bddConstraints: "",
			template:       "任务: {{ issue.title }}\n{{ bdd_constraints }}\n开始执行",
			expectedContains: []string{
				"任务: 简单任务",
				"开始执行",
			},
			notExpected: []string{
				"{{ bdd_constraints }}",
			},
		},
		{
			name: "空 BDD 约束不添加额外内容",
			issue: &domain.Issue{
				ID:         "126",
				Identifier: "TEST-4",
				Title:      "无约束任务",
				State:      "Todo",
			},
			attempt:        nil,
			bddConstraints: "",
			template:       "任务: {{ issue.title }}",
			expectedContains: []string{
				"任务: 无约束任务",
			},
			notExpected: []string{
				"BDD",
				"验收",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildPromptWithBDDConstraints(tt.issue, tt.attempt, tt.bddConstraints, tt.template)

			for _, expected := range tt.expectedContains {
				if !strings.Contains(result, expected) {
					t.Errorf("result missing expected content: %q\nResult:\n%s", expected, result)
				}
			}

			for _, notExpected := range tt.notExpected {
				if strings.Contains(result, notExpected) {
					t.Errorf("result contains unexpected content: %q\nResult:\n%s", notExpected, result)
				}
			}
		})
	}
}

// TestBuildPromptWithHistoryAndBDD 测试同时包含历史和BDD约束的提示词构建
func TestBuildPromptWithHistoryAndBDD(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		issue          *domain.Issue
		attempt        *int
		history        []domain.ConversationTurn
		bddConstraints string
		template       string
		expectedContains []string
	}{
		{
			name: "完整提示词包含历史和BDD",
			issue: &domain.Issue{
				ID:         "127",
				Identifier: "TEST-5",
				Title:      "复杂需求",
				State:      "Todo",
			},
			attempt: intPtr(2),
			history: []domain.ConversationTurn{
				{Role: "user", Content: "需求A", Timestamp: now},
				{Role: "assistant", Content: "澄清A", Timestamp: now},
			},
			bddConstraints: "## BDD 验收标准\n\n场景列表",
			template: "任务 {{ issue.identifier }}: {{ issue.title }}\n\n{{ conversation_history }}\n\n{{ bdd_constraints }}",
			expectedContains: []string{
				"任务 TEST-5: 复杂需求",
				"attempt 2",
				"## 需求澄清历史",
				"**User:** 需求A",
				"## BDD 验收标准",
				"场景列表",
			},
		},
		{
			name: "无历史无BDD约束",
			issue: &domain.Issue{
				ID:         "128",
				Identifier: "TEST-6",
				Title:      "简单任务",
				State:      "Todo",
			},
			attempt:        nil,
			history:        nil,
			bddConstraints: "",
			template:       "任务: {{ issue.title }}",
			expectedContains: []string{
				"任务: 简单任务",
			},
		},
		{
			name: "只有历史无BDD",
			issue: &domain.Issue{
				ID:         "129",
				Identifier: "TEST-7",
				Title:      "历史任务",
				State:      "Todo",
			},
			attempt: nil,
			history: []domain.ConversationTurn{
				{Role: "user", Content: "问题", Timestamp: now},
				{Role: "assistant", Content: "回答", Timestamp: now},
			},
			bddConstraints: "",
			template:       "任务: {{ issue.title }}\n{{ conversation_history }}",
			expectedContains: []string{
				"任务: 历史任务",
				"## 需求澄清历史",
				"**User:** 问题",
			},
		},
		{
			name: "只有BDD无历史",
			issue: &domain.Issue{
				ID:         "130",
				Identifier: "TEST-8",
				Title:      "BDD任务",
				State:      "Todo",
			},
			attempt:        nil,
			history:        nil,
			bddConstraints: "## BDD 验收标准",
			template:       "任务: {{ issue.title }}\n{{ bdd_constraints }}",
			expectedContains: []string{
				"任务: BDD任务",
				"## BDD 验收标准",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildPromptWithHistoryAndBDD(tt.issue, tt.attempt, tt.history, tt.bddConstraints, tt.template)

			for _, expected := range tt.expectedContains {
				if !strings.Contains(result, expected) {
					t.Errorf("result missing expected content: %q\nResult:\n%s", expected, result)
				}
			}
		})
	}
}

// TestInjectBDDConstraints 测试注入 BDD 约束到已有 prompt
func TestInjectBDDConstraints(t *testing.T) {
	tests := []struct {
		name           string
		prompt         string
		bddConstraints string
		expectedContains []string
		notExpected    []string
	}{
		{
			name:   "有占位符注入",
			prompt: "任务描述\n\n{{ bdd_constraints }}\n\n执行指令",
			bddConstraints: "## BDD 验收标准\n\n场景1",
			expectedContains: []string{
				"任务描述",
				"## BDD 验收标准",
				"场景1",
				"执行指令",
			},
			notExpected: []string{
				"{{ bdd_constraints }}",
			},
		},
		{
			name:   "无占位符追加",
			prompt: "任务描述\n执行指令",
			bddConstraints: "## BDD 验收标准\n\n场景列表",
			expectedContains: []string{
				"任务描述",
				"执行指令",
				"---",
				"## BDD 验收标准",
				"场景列表",
			},
		},
		{
			name:           "空约束不修改",
			prompt:         "任务描述",
			bddConstraints: "",
			expectedContains: []string{
				"任务描述",
			},
			notExpected: []string{
				"---",
				"BDD",
			},
		},
		{
			name:           "空 prompt 使用约束",
			prompt:         "",
			bddConstraints: "## BDD 验收标准",
			expectedContains: []string{
				"## BDD 验收标准",
			},
		},
		{
			name:   "约束包含分隔符",
			prompt: "任务描述",
			bddConstraints: "---\n## BDD 验收标准",
			expectedContains: []string{
				"任务描述",
				"## BDD 验收标准",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InjectBDDConstraints(tt.prompt, tt.bddConstraints)

			for _, expected := range tt.expectedContains {
				if !strings.Contains(result, expected) {
					t.Errorf("result missing expected content: %q\nResult:\n%s", expected, result)
				}
			}

			for _, notExpected := range tt.notExpected {
				if strings.Contains(result, notExpected) {
					t.Errorf("result contains unexpected content: %q\nResult:\n%s", notExpected, result)
				}
			}
		})
	}
}
