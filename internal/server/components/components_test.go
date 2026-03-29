package components_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/dministrator/symphony/internal/domain"
	"github.com/dministrator/symphony/internal/server/components"
	"github.com/stretchr/testify/assert"
)

func TestRenderRunningSessionsKanban_Empty(t *testing.T) {
	state := &domain.OrchestratorState{
		Running:     make(map[string]*domain.RunningEntry),
		CodexTotals: &domain.CodexTotals{},
	}
	now := time.Now()

	html := components.RenderRunningSessionsKanban(state, now)

	assert.Contains(t, html, "暂无活跃 Session")
	assert.Contains(t, html, "kanban-column-running")
}

func TestRenderRunningSessionsKanban_WithEntry(t *testing.T) {
	now := time.Now()
	state := &domain.OrchestratorState{
		Running: map[string]*domain.RunningEntry{
			"TEST-1": {
				Identifier: "TEST-1",
				StartedAt:  now.Add(-5 * time.Minute),
				TurnCount:  10,
				Issue: &domain.Issue{
					ID:    "1",
					State: "In Progress",
				},
				Session: &domain.LiveSession{
					SessionID: "sess-abc123",
				},
			},
		},
		CodexTotals: &domain.CodexTotals{},
	}

	html := components.RenderRunningSessionsKanban(state, now)

	assert.Contains(t, html, "TEST-1")
	assert.Contains(t, html, "In Progress")
	assert.Contains(t, html, "sess-abc123")
	assert.Contains(t, html, "kanban-card")
}

func TestRenderRetryQueueKanban_Empty(t *testing.T) {
	state := &domain.OrchestratorState{
		RetryAttempts: make(map[string]*domain.RetryEntry),
		CodexTotals:   &domain.CodexTotals{},
	}

	html := components.RenderRetryQueueKanban(state)

	assert.Contains(t, html, "当前没有等待重试的 Issue")
	assert.Contains(t, html, "kanban-column-retrying")
}

func TestRenderRetryQueueKanban_WithEntry(t *testing.T) {
	state := &domain.OrchestratorState{
		RetryAttempts: map[string]*domain.RetryEntry{
			"RETRY-1": {
				Identifier: "RETRY-1",
				Attempt:    2,
				DueAtMs:    time.Now().Add(5 * time.Minute).UnixMilli(),
			},
		},
		CodexTotals: &domain.CodexTotals{},
	}

	html := components.RenderRetryQueueKanban(state)

	assert.Contains(t, html, "RETRY-1")
	assert.Contains(t, html, "Retry #2")
	assert.Contains(t, html, "kanban-card")
}

func TestRenderDashboardHTML(t *testing.T) {
	now := time.Now()
	state := &domain.OrchestratorState{
		Running:       make(map[string]*domain.RunningEntry),
		RetryAttempts: make(map[string]*domain.RetryEntry),
		CodexTotals:   &domain.CodexTotals{},
	}

	html := components.RenderDashboardHTML(state, now)

	assert.Contains(t, html, "任务看板")
	assert.Contains(t, html, "kanban-container")
	assert.Contains(t, html, "metric-running")
}

func TestRenderErrorHTML(t *testing.T) {
	html := components.RenderErrorHTML("任务不存在", "无法找到任务 NONEXISTENT-123")

	assert.Contains(t, html, "任务不存在")
	assert.Contains(t, html, "无法找到任务 NONEXISTENT-123")
	assert.Contains(t, html, "返回看板")
	assert.Contains(t, html, "错误")
}

func TestRenderTaskDetailHTML_Basic(t *testing.T) {
	description := "这是一个测试任务的描述"
	issue := &domain.Issue{
		ID:          "1",
		Identifier:  "SYM-123",
		Title:       "添加用户登录功能",
		Description: &description,
		State:       "In Progress",
	}

	stageState := &domain.StageState{
		Name:      "clarification",
		Status:    "pending",
		Round:     1,
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	html := components.RenderTaskDetailHTML(issue, stageState, nil)

	assert.Contains(t, html, "SYM-123")
	assert.Contains(t, html, "添加用户登录功能")
	assert.Contains(t, html, "In Progress")
	assert.Contains(t, html, "需求澄清")
	assert.Contains(t, html, "第 1 / 5 轮")
	assert.Contains(t, html, "暂无历史问答记录")
	assert.Contains(t, html, "这是一个测试任务的描述")
}

func TestRenderTaskDetailHTML_WithConversationHistory(t *testing.T) {
	description := "任务描述"
	issue := &domain.Issue{
		ID:          "1",
		Identifier:  "SYM-456",
		Title:       "实现数据导出功能",
		Description: &description,
		State:       "In Progress",
	}

	stageState := &domain.StageState{
		Name:      "clarification",
		Status:    "in_progress",
		Round:     2,
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	conversationHistory := []domain.ConversationTurn{
		{
			Role:      "assistant",
			Content:   "请问需要导出哪些数据格式？",
			Timestamp: time.Now().Add(-10 * time.Minute),
		},
		{
			Role:      "user",
			Content:   "需要支持 CSV 和 JSON 格式",
			Timestamp: time.Now().Add(-5 * time.Minute),
		},
		{
			Role:      "assistant",
			Content:   "数据量预计有多大？",
			Timestamp: time.Now().Add(-2 * time.Minute),
		},
	}

	html := components.RenderTaskDetailHTML(issue, stageState, conversationHistory)

	assert.Contains(t, html, "SYM-456")
	assert.Contains(t, html, "实现数据导出功能")
	assert.Contains(t, html, "第 2 / 5 轮")
	assert.Contains(t, html, "Q1")
	assert.Contains(t, html, "请问需要导出哪些数据格式？")
	assert.Contains(t, html, "A1")
	assert.Contains(t, html, "需要支持 CSV 和 JSON 格式")
	assert.Contains(t, html, "Q2")
	assert.Contains(t, html, "数据量预计有多大？")
	assert.Contains(t, html, "AI 当前问题")
}

func TestRenderTaskDetailHTML_WaitingForAnswer(t *testing.T) {
	issue := &domain.Issue{
		ID:         "1",
		Identifier: "SYM-789",
		Title:      "优化搜索性能",
		State:      "In Progress",
	}

	stageState := &domain.StageState{
		Name:      "clarification",
		Status:    "in_progress",
		Round:     3,
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	conversationHistory := []domain.ConversationTurn{
		{
			Role:      "assistant",
			Content:   "请问登录方式是邮箱还是手机号？",
			Timestamp: time.Now(),
		},
	}

	html := components.RenderTaskDetailHTML(issue, stageState, conversationHistory)

	assert.Contains(t, html, "AI 当前问题")
	assert.Contains(t, html, "请问登录方式是邮箱还是手机号？")
	assert.Contains(t, html, "输入您的回答")
	assert.Contains(t, html, "提交回答")
	assert.Contains(t, html, "跳过澄清")
	assert.Contains(t, html, "第 3 / 5 轮")
}

func TestRenderTaskDetailHTML_NotInClarification(t *testing.T) {
	issue := &domain.Issue{
		ID:         "1",
		Identifier: "SYM-999",
		Title:      "完成代码重构",
		State:      "Done",
	}

	stageState := &domain.StageState{
		Name:      "implementation",
		Status:    "completed",
		Round:     0,
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	html := components.RenderTaskDetailHTML(issue, stageState, nil)

	assert.Contains(t, html, "SYM-999")
	assert.Contains(t, html, "完成代码重构")
	assert.Contains(t, html, "当前任务不在澄清阶段")
	assert.NotContains(t, html, "提交回答")
}

func TestRenderTaskDetailHTML_NoDescription(t *testing.T) {
	issue := &domain.Issue{
		ID:         "1",
		Identifier: "SYM-111",
		Title:      "无描述任务",
		State:      "Todo",
	}

	stageState := &domain.StageState{
		Name:      "backlog",
		Status:    "pending",
		Round:     0,
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	html := components.RenderTaskDetailHTML(issue, stageState, nil)

	assert.Contains(t, html, "SYM-111")
	assert.Contains(t, html, "无描述任务")
	// 任务描述区域不会被渲染（没有 h3 标签显示"任务描述"）
	assert.NotContains(t, html, "<h3 style=\"font-size: 1rem; font-weight: 600; color: var(--ink-bright); margin-bottom: 0.75rem;\">任务描述</h3>")
}

func TestRenderTaskDetailHTML_ProgressBar(t *testing.T) {
	issue := &domain.Issue{
		ID:         "1",
		Identifier: "SYM-222",
		Title:      "进度测试",
		State:      "In Progress",
	}

	// 测试不同轮次的进度条宽度
	for round := 1; round <= 5; round++ {
		stageState := &domain.StageState{
			Name:      "clarification",
			Status:    "in_progress",
			Round:     round,
			StartedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		html := components.RenderTaskDetailHTML(issue, stageState, nil)

		expectedWidth := round * 20 // 1/5 = 20%, 2/5 = 40%, etc.
		assert.Contains(t, html, "width: "+strconv.Itoa(expectedWidth))
	}
}