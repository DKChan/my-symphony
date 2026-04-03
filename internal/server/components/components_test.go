package components_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/dministrator/symphony/internal/common"
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

// Epic 8: 待人工处理页面渲染测试

func TestRenderNeedsAttentionHTML_Basic(t *testing.T) {
	failedAt := time.Now().Add(-1 * time.Hour)
	issue := &domain.Issue{
		ID:          "1",
		Identifier:  "FAIL-123",
		Title:       "失败的任务",
		Description: nil,
		State:       "Needs Attention",
	}

	stageState := &domain.StageState{
		Name:          "implementation",
		Status:        "failed",
		StartedAt:     time.Now().Add(-2 * time.Hour),
		UpdatedAt:     time.Now(),
		FailedAt:      &failedAt,
		RetryCount:    3,
		ErrorType:     "execution_error",
		ErrorMessage:  "测试执行失败：无法连接数据库",
		LastLogSnippet: "[ERROR] connection refused\n[ERROR] retry 1/3 failed\n[ERROR] max retries exceeded",
		Suggestion:     "请检查数据库连接配置，确保数据库服务正在运行",
	}

	html := components.RenderNeedsAttentionHTML(issue, stageState)

	// 验证基本信息
	assert.Contains(t, html, "FAIL-123")
	assert.Contains(t, html, "失败的任务")
	assert.Contains(t, html, "待人工处理")

	// 验证失败详情
	assert.Contains(t, html, "失败阶段")
	assert.Contains(t, html, "失败时间")
	assert.Contains(t, html, "错误类型")
	assert.Contains(t, html, "重试次数")
	assert.Contains(t, html, "3")

	// 验证错误信息
	assert.Contains(t, html, "错误消息")
	assert.Contains(t, html, "测试执行失败：无法连接数据库")

	// 验证日志片段
	assert.Contains(t, html, "日志片段")
	assert.Contains(t, html, "[ERROR] connection refused")

	// 验证修复建议
	assert.Contains(t, html, "修复建议")
	assert.Contains(t, html, "请检查数据库连接配置")

	// 验证操作按钮
	assert.Contains(t, html, "重新执行")
	assert.Contains(t, html, "重新澄清需求")
	assert.Contains(t, html, "放弃任务")
}

func TestRenderNeedsAttentionHTML_WithoutLogSnippet(t *testing.T) {
	failedAt := time.Now()
	issue := &domain.Issue{
		ID:         "1",
		Identifier: "FAIL-456",
		Title:      "无日志片段的失败任务",
		State:      "Needs Attention",
	}

	stageState := &domain.StageState{
		Name:          "verification",
		Status:        "failed",
		StartedAt:     time.Now().Add(-30 * time.Minute),
		UpdatedAt:     time.Now(),
		FailedAt:      &failedAt,
		RetryCount:    1,
		ErrorType:     "test_failure",
		ErrorMessage:  "验收测试失败",
		LastLogSnippet: "",
		Suggestion:     "请手动检查测试用例",
	}

	html := components.RenderNeedsAttentionHTML(issue, stageState)

	// 验证基本信息
	assert.Contains(t, html, "FAIL-456")
	assert.Contains(t, html, "无日志片段的失败任务")

	// 日志片段区域不应显示（因为为空）
	assert.NotContains(t, html, "log-snippet\">")
}

func TestRenderNeedsAttentionHTML_WithoutSuggestion(t *testing.T) {
	failedAt := time.Now()
	issue := &domain.Issue{
		ID:         "1",
		Identifier: "FAIL-789",
		Title:      "无建议的失败任务",
		State:      "Needs Attention",
	}

	stageState := &domain.StageState{
		Name:          "implementation",
		Status:        "failed",
		StartedAt:     time.Now().Add(-1 * time.Hour),
		UpdatedAt:     time.Now(),
		FailedAt:      &failedAt,
		RetryCount:    2,
		ErrorType:     "unknown_error",
		ErrorMessage:  "未知错误",
		LastLogSnippet: "some log content",
		Suggestion:     "",
	}

	html := components.RenderNeedsAttentionHTML(issue, stageState)

	// 验证基本信息
	assert.Contains(t, html, "FAIL-789")

	// 建议区域不应显示（因为为空） - 检查 suggestion-box div 元素不存在
	assert.NotContains(t, html, `<div class="suggestion-box">`)
}

func TestRenderNeedsAttentionHTML_MaxRetries(t *testing.T) {
	failedAt := time.Now()
	issue := &domain.Issue{
		ID:         "1",
		Identifier: "FAIL-MAX",
		Title:      "达到最大重试次数",
		State:      "Needs Attention",
	}

	stageState := &domain.StageState{
		Name:          "implementation",
		Status:        "failed",
		StartedAt:     time.Now().Add(-3 * time.Hour),
		UpdatedAt:     time.Now(),
		FailedAt:      &failedAt,
		RetryCount:    5,
		ErrorType:     "max_retries_exceeded",
		ErrorMessage:  "已达到最大重试次数",
		LastLogSnippet: "Final retry attempt failed",
		Suggestion:     "任务需要人工干预",
	}

	html := components.RenderNeedsAttentionHTML(issue, stageState)

	// 验证重试次数显示
	assert.Contains(t, html, "已达到最大重试次数 (5/5)")
	assert.Contains(t, html, "Final retry attempt failed")
}

// TestRenderFilterBar 测试 RenderFilterBar 函数
func TestRenderFilterBar(t *testing.T) {
	tests := []struct {
		name         string
		currentFilter string
		wantContains []string
	}{
		{
			name:         "empty filter",
			currentFilter: "",
			wantContains: []string{"filter-bar", "全部", "任务筛选"},
		},
		{
			name:         "running filter",
			currentFilter: "running",
			wantContains: []string{"filter-bar", "进行中"},
		},
		{
			name:         "retrying filter",
			currentFilter: "retrying",
			wantContains: []string{"filter-bar", "待人工处理"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := components.RenderFilterBar(tt.currentFilter)
			for _, want := range tt.wantContains {
				assert.Contains(t, html, want)
			}
		})
	}
}

// TestRenderTaskList 测试 RenderTaskList 函数
func TestRenderTaskList(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		html := components.RenderTaskList([]common.TaskPayload{}, "")
		assert.Contains(t, html, "task-list")
	})

	t.Run("with tasks", func(t *testing.T) {
		tasks := []common.TaskPayload{
			{
				Identifier: "TEST-1",
				Title:      "Task 1",
				State:      "In Progress",
			},
		}
		html := components.RenderTaskList(tasks, "")
		assert.Contains(t, html, "TEST-1")
		assert.Contains(t, html, "Task 1")
	})
}

// TestRenderRunningCard 测试 RenderRunningCard 函数
func TestRenderRunningCard(t *testing.T) {
	now := time.Now()
	entry := &domain.RunningEntry{
		Identifier: "RUN-1",
		StartedAt:  now.Add(-10 * time.Minute),
		TurnCount:  5,
		Issue: &domain.Issue{
			ID:    "1",
			Title: "Running Task",
			State: "In Progress",
		},
		Session: &domain.LiveSession{
			SessionID: "session-123",
		},
	}

	html := components.RenderRunningCard(entry, now)
	assert.Contains(t, html, "RUN-1")
	assert.Contains(t, html, "In Progress")
	assert.Contains(t, html, "session-123")
	assert.Contains(t, html, "kanban-card")
}

// TestRenderRetryCard 测试 RenderRetryCard 函数
func TestRenderRetryCard(t *testing.T) {
	entry := &domain.RetryEntry{
		Identifier: "RETRY-1",
		Attempt:    2,
		DueAtMs:    time.Now().Add(5 * time.Minute).UnixMilli(),
	}

	html := components.RenderRetryCard(entry)
	assert.Contains(t, html, "RETRY-1")
	assert.Contains(t, html, "Retry")
}

// TestRenderStageKanban 测试 RenderStageKanban 函数
func TestRenderStageKanban(t *testing.T) {
	payload := &common.KanbanPayload{
		Columns: []common.KanbanColumn{
			{
				ID:    "clarification",
				Title: "需求澄清",
			},
		},
	}

	html := components.RenderStageKanban(payload)
	assert.Contains(t, html, "kanban")
}

// TestRenderStageKanbanScript 测试 RenderStageKanbanScript 函数
func TestRenderStageKanbanScript(t *testing.T) {
	html := components.RenderStageKanbanScript()
	assert.Contains(t, html, "function")
	assert.Contains(t, html, "task_update")
}

// TestRenderBDDReviewHTML 测试 RenderBDDReviewHTML 函数
func TestRenderBDDReviewHTML(t *testing.T) {
	issue := &domain.Issue{
		ID:         "1",
		Identifier: "BDD-1",
		Title:      "BDD Review Task",
		State:      "In Progress",
	}

	stageState := &domain.StageState{
		Name:      "bdd_review",
		Status:    "waiting_review",
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	bddContent := "Feature: Login\n  Scenario: User logs in"

	html := components.RenderBDDReviewHTML(issue, stageState, bddContent)
	assert.Contains(t, html, "BDD-1")
	assert.Contains(t, html, "BDD Review Task")
	assert.Contains(t, html, "BDD 规则")
	assert.Contains(t, html, "Feature: Login")
}

// TestRenderArchitectureReviewHTML 测试 RenderArchitectureReviewHTML 函数
func TestRenderArchitectureReviewHTML(t *testing.T) {
	issue := &domain.Issue{
		ID:         "1",
		Identifier: "ARCH-1",
		Title:      "Architecture Review Task",
		State:      "In Progress",
	}

	stageState := &domain.StageState{
		Name:      "architecture_review",
		Status:    "waiting_review",
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	archContent := "# Architecture\n## Design"
	tddContent := "# TDD Rules"

	html := components.RenderArchitectureReviewHTML(issue, stageState, archContent, tddContent)
	assert.Contains(t, html, "ARCH-1")
	assert.Contains(t, html, "Architecture Review Task")
	assert.Contains(t, html, "架构设计")
	assert.Contains(t, html, "Architecture")
}

// TestRenderVerificationReportHTML 测试 RenderVerificationReportHTML 函数
func TestRenderVerificationReportHTML(t *testing.T) {
	issue := &domain.Issue{
		ID:         "1",
		Identifier: "VERIFY-1",
		Title:      "Verification Task",
		State:      "In Progress",
	}

	stageState := &domain.StageState{
		Name:      "verification",
		Status:    "waiting_review",
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	report := &domain.VerificationReport{
		TaskID:         "1",
		TaskIdentifier: "VERIFY-1",
		TaskTitle:      "Verification Task",
		GeneratedAt:    time.Now(),
		OverallStatus:  "PASS",
		TestResults:    &domain.TestResults{Total: 10, Passed: 10, Failed: 0},
	}

	html := components.RenderVerificationReportHTML(issue, stageState, report)
	assert.Contains(t, html, "VERIFY-1")
	assert.Contains(t, html, "Verification Task")
	assert.Contains(t, html, "验证")
}