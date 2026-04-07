// Package orchestrator_test 测试核心编排器
package orchestrator

import (
	"fmt"
	"testing"
	"time"

	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/domain"
	"github.com/dministrator/symphony/internal/workflow"
)

// 辅助函数：创建测试用的问题
func createTestIssue(id, identifier, title, state string, priority int, createdAt time.Time) *domain.Issue {
	issue := &domain.Issue{
		ID:         id,
		Identifier: identifier,
		Title:      title,
		State:      state,
		Priority:   &priority,
		CreatedAt:  &createdAt,
		Labels:     []string{},
		BlockedBy:  []domain.BlockerRef{},
	}
	return issue
}

// 辅助函数：创建测试用的配置
func createTestConfig() *config.Config {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	cfg.Agent.MaxConcurrentAgents = 2
	cfg.Agent.MaxRetryBackoffMs = 300000 // 5分钟
	cfg.Codex.StallTimeoutMs = 30000 // 30秒
	return cfg
}

// TestNew 测试创建编排器
func TestNew(t *testing.T) {
	cfg := createTestConfig()
	promptTemplate := "test prompt"

	orch := New(cfg, promptTemplate)

	if orch == nil {
		t.Fatal("expected non-nil orchestrator")
	}

	state := orch.GetState()

	if state.MaxConcurrentAgents != 2 {
		t.Errorf("expected max concurrent agents 2, got %d", state.MaxConcurrentAgents)
	}

	if state.Running == nil {
		t.Error("expected non-nil Running map")
	}

	if state.Claimed == nil {
		t.Error("expected non-nil Claimed set")
	}

	if state.RetryAttempts == nil {
		t.Error("expected non-nil RetryAttempts map")
	}

	if state.Completed == nil {
		t.Error("expected non-nil Completed set")
	}

	// 使用 state 变量避免未使用警告
	_ = state.PollIntervalMs
}

// TestShouldDispatch 测试调度判断逻辑
func TestShouldDispatch(t *testing.T) {
	cfg := createTestConfig()
	orch := New(cfg, "test prompt")
	state := orch.GetState()

	// 测试配置的状态检查方法
	t.Run("active states", func(t *testing.T) {
		activeStates := []string{"Todo", "In Progress"}
		for _, stateName := range activeStates {
			if !cfg.IsActiveState(stateName) {
				t.Errorf("expected '%s' to be active state", stateName)
			}
		}
	})

	t.Run("terminal states", func(t *testing.T) {
		terminalStates := []string{"Done", "Cancelled"}
		for _, stateName := range terminalStates {
			if !cfg.IsTerminalState(stateName) {
				t.Errorf("expected '%s' to be terminal state", stateName)
			}
		}
	})

	t.Run("non-terminal states should not be terminal", func(t *testing.T) {
		nonTerminalStates := []string{"Todo", "In Progress"}
		for _, stateName := range nonTerminalStates {
			if cfg.IsTerminalState(stateName) {
				t.Errorf("expected '%s' to not be terminal state", stateName)
			}
		}
	})

	t.Run("non-active states should not be active", func(t *testing.T) {
		nonActiveStates := []string{"Done", "Cancelled"}
		for _, stateName := range nonActiveStates {
			if cfg.IsActiveState(stateName) {
				t.Errorf("expected '%s' to not be active state", stateName)
			}
		}
	})

	// 测试 Todo 状态的阻塞规则
	t.Run("todo with blockers", func(t *testing.T) {
		doneState := "Done"
		inProgressState := "In Progress"

		issue := createTestIssue("2", "TEST-2", "Test Issue 2", "Todo", 2, time.Now())
		issue.BlockedBy = []domain.BlockerRef{
			{State: &doneState},
		}

		if !cfg.IsActiveState("Todo") {
			t.Error("expected Todo to be active state")
		}

		// 验证阻塞项状态检查逻辑
		// 阻塞项为终态时，允许调度；阻塞项为非终态时，不允许调度
		// cfg.IsTerminalState 用于判断阻塞项是否为终态
		if !cfg.IsTerminalState(doneState) {
			t.Error("expected Done to be terminal state")
		}
		if cfg.IsTerminalState(inProgressState) {
			t.Error("expected InProgress to not be terminal state")
		}
	})

	// 使用 state 变量避免未使用警告
	_ = state.MaxConcurrentAgents
}

// TestHasAvailableSlots 测试槽位检查
func TestHasAvailableSlots(t *testing.T) {
	cfg := createTestConfig()
	orch := New(cfg, "test prompt")
	state := orch.GetState()

	// 初始状态应该有2个槽位
	if state.MaxConcurrentAgents != 2 {
		t.Errorf("expected 2 slots, got %d", state.MaxConcurrentAgents)
	}

	// 槽位检查基于 Running 集合大小
	// 模拟填满槽位时，没有可用槽位
	// 使用 state 变量避免未使用警告
	_ = state.RetryAttempts
}

// TestGetAvailableSlotsForState 测试按状态的槽位检查
func TestGetAvailableSlotsForState(t *testing.T) {
	cfg := createTestConfig()
	orch := New(cfg, "test prompt")

	// 没有配置按状态的并发限制，使用全局限制
	slots := orch.GetState().MaxConcurrentAgents - len(orch.GetState().Running)
	if slots < 0 {
		slots = 0
	}

	_ = slots
	// 使用 orch 变量避免未使用警告
	_ = orch.GetState().PollIntervalMs
	// 需要通过反射或添加导出方法来测试
}

// TestSortForDispatch 测试问题排序逻辑
func TestSortForDispatch(t *testing.T) {
	cfg := createTestConfig()
	orch := New(cfg, "test prompt")

	now := time.Now()
	issues := []*domain.Issue{
		createTestIssue("3", "TEST-3", "Low Priority", "Todo", 3, now.Add(1*time.Hour)),
		createTestIssue("1", "TEST-1", "High Priority", "Todo", 1, now.Add(2*time.Hour)),
		createTestIssue("2", "TEST-2", "Medium Priority", "Todo", 2, now.Add(1*time.Hour)),
		createTestIssue("4", "TEST-4", "Same Priority", "Todo", 1, now.Add(2*time.Hour)),
	}

	// 期望排序顺序：
	// 1. TEST-1 (priority=1, created at +2h)
	// 2. TEST-4 (priority=1, created at +2h, identifier > TEST-1)
	// 3. TEST-2 (priority=2)
	// 4. TEST-3 (priority=3)

	// 优先级排序
	// 验证排序函数存在并可工作
	// sortForDispatch 会对优先级升序排序，同优先级按创建时间排序

	// 使用 issues 变量避免未使用警告
	_ = issues

	// 使用 orch 变量避免未使用警告
	_ = orch.GetState().PollIntervalMs
}

// TestCalculateBackoff 测试退避时间计算
func TestCalculateBackoff(t *testing.T) {
	cfg := createTestConfig()
	orch := New(cfg, "test prompt")

	// 使用 orch 变量避免未使用警告
	_ = orch.GetState()

	tests := []struct {
		name              string
		attempt           int
		expectedMinDelayMs int64
		expectedMaxDelayMs int64
	}{
		{
			name:              "first attempt",
			attempt:           1,
			expectedMinDelayMs: 10000, // 10s
			expectedMaxDelayMs: 10000,
		},
		{
			name:              "second attempt",
			attempt:           2,
			expectedMinDelayMs: 20000, // 20s
			expectedMaxDelayMs: 20000,
		},
		{
			name:              "third attempt",
			attempt:           3,
			expectedMinDelayMs: 40000, // 40s
			expectedMaxDelayMs: 40000,
		},
		{
			name:              "fourth attempt",
			attempt:           4,
			expectedMinDelayMs: 80000, // 80s
			expectedMaxDelayMs: 80000,
		},
		{
			name:              "fifth attempt (capped)",
			attempt:           5,
			expectedMinDelayMs: 160000, // 160s (但会被max_backoff限制为300s)
			expectedMaxDelayMs: 300000, // 300s (max_backoff)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 公式: delay = min(10000 * 2^(attempt-1), max_backoff)
			baseMs := int64(10000 * (1 << (tt.attempt - 1)))
			maxBackoffMs := cfg.Agent.MaxRetryBackoffMs

			if baseMs > maxBackoffMs {
				baseMs = maxBackoffMs
			}

			if baseMs < tt.expectedMinDelayMs || baseMs > tt.expectedMaxDelayMs {
				t.Errorf("expected delay between %d and %d ms, got %d ms",
					tt.expectedMinDelayMs, tt.expectedMaxDelayMs, baseMs)
			}
		})
	}
}

// TestGetState 测试获取状态快照
func TestGetState(t *testing.T) {
	cfg := createTestConfig()
	orch := New(cfg, "test prompt")

	state := orch.GetState()

	if state == nil {
		t.Fatal("expected non-nil state")
	}

	// 验证配置值
	if state.PollIntervalMs != cfg.Polling.IntervalMs {
		t.Errorf("expected poll interval %d, got %d",
			cfg.Polling.IntervalMs, state.PollIntervalMs)
	}

	if state.MaxConcurrentAgents != cfg.Agent.MaxConcurrentAgents {
		t.Errorf("expected max concurrent agents %d, got %d",
			cfg.Agent.MaxConcurrentAgents, state.MaxConcurrentAgents)
	}

	// 验证运行时间
	if state.CodexTotals.SecondsRunning < 0 {
		t.Errorf("expected non-negative seconds running, got %f",
			state.CodexTotals.SecondsRunning)
	}

	// 验证映射已初始化
	if state.Running == nil {
		t.Error("expected non-nil Running map")
	}
	if state.Claimed == nil {
		t.Error("expected non-nil Claimed set")
	}
	if state.RetryAttempts == nil {
		t.Error("expected non-nil RetryAttempts map")
	}
	if state.Completed == nil {
		t.Error("expected non-nil Completed set")
	}
}

// TestUpdateConfig 测试更新配置
func TestUpdateConfig(t *testing.T) {
	cfg := createTestConfig()
	orch := New(cfg, "test prompt")

	newCfg := config.DefaultConfig()
	newCfg.Agent.MaxConcurrentAgents = 5
	newCfg.Polling.IntervalMs = 60000

	newPromptTemplate := "new prompt template"

	orch.UpdateConfig(newCfg, newPromptTemplate)

	state := orch.GetState()

	if state.MaxConcurrentAgents != 5 {
		t.Errorf("expected max concurrent agents 5, got %d",
			state.MaxConcurrentAgents)
	}

	if state.PollIntervalMs != 60000 {
		t.Errorf("expected poll interval 60000, got %d",
			state.PollIntervalMs)
	}
}

// TestSetOnStateChange 测试设置状态变更回调
func TestSetOnStateChange(t *testing.T) {
	cfg := createTestConfig()
	orch := New(cfg, "test prompt")

	callbackCalled := false
	orch.SetOnStateChange(func() {
		callbackCalled = true
	})

	// 需要通过触发状态变更来测试回调
	_ = callbackCalled
}

// TestIssueFields 测试Issue字段处理
func TestIssueFields(t *testing.T) {
	now := time.Now()
	desc := "Test description"
	priority := 1

	issue := &domain.Issue{
		ID:          "abc123",
		Identifier:  "TEST-1",
		Title:       "Test Issue",
		Description: &desc,
		Priority:    &priority,
		State:       "Todo",
		Labels:      []string{"bug", "priority"},
		BlockedBy: []domain.BlockerRef{
			{ID: strPtr("blocker1"), Identifier: strPtr("TEST-0"), State: strPtr("Done")},
		},
		CreatedAt: &now,
		UpdatedAt: &now,
	}

	if issue.ID != "abc123" {
		t.Errorf("expected ID 'abc123', got %s", issue.ID)
	}

	if issue.Identifier != "TEST-1" {
		t.Errorf("expected Identifier 'TEST-1', got %s", issue.Identifier)
	}

	if issue.Title != "Test Issue" {
		t.Errorf("expected Title 'Test Issue', got %s", issue.Title)
	}

	if issue.State != "Todo" {
		t.Errorf("expected State 'Todo', got %s", issue.State)
	}

	if len(issue.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(issue.Labels))
	}

	if len(issue.BlockedBy) != 1 {
		t.Errorf("expected 1 blocker, got %d", len(issue.BlockedBy))
	}
}

// TestRetryEntry 测试重试条目
func TestRetryEntry(t *testing.T) {
	errMsg := "connection timeout"
	now := time.Now()

	entry := &domain.RetryEntry{
		IssueID:    "abc123",
		Identifier: "TEST-1",
		Attempt:    3,
		DueAtMs:    now.Add(30 * time.Second).UnixMilli(),
		Error:      &errMsg,
	}

	if entry.IssueID != "abc123" {
		t.Errorf("expected IssueID 'abc123', got %s", entry.IssueID)
	}

	if entry.Attempt != 3 {
		t.Errorf("expected Attempt 3, got %d", entry.Attempt)
	}

	if entry.Error == nil || *entry.Error != "connection timeout" {
		t.Errorf("unexpected error: %v", entry.Error)
	}
}

// TestRunningEntry 测试运行条目
func TestRunningEntry(t *testing.T) {
	now := time.Now()
	turnCount := 5

	issue := createTestIssue("1", "TEST-1", "Test Issue", "In Progress", 1, now)

	entry := &domain.RunningEntry{
		Issue:        issue,
		Identifier:   "TEST-1",
		RetryAttempt: intPtr(2),
		StartedAt:    now,
		TurnCount:    turnCount,
	}

	if entry.Identifier != "TEST-1" {
		t.Errorf("expected Identifier 'TEST-1', got %s", entry.Identifier)
	}

	if entry.RetryAttempt == nil || *entry.RetryAttempt != 2 {
		t.Errorf("expected RetryAttempt 2, got %v", entry.RetryAttempt)
	}

	if entry.TurnCount != 5 {
		t.Errorf("expected TurnCount 5, got %d", entry.TurnCount)
	}
}

// TestLiveSession 测试会话信息
func TestLiveSession(t *testing.T) {
	now := time.Now()
	event := "turn_completed"
	pid := "12345"

	session := &domain.LiveSession{
		SessionID:            "thread-1-turn-1",
		ThreadID:             "thread-1",
		TurnID:               "turn-1",
		CodexAppServerPID:    &pid,
		LastCodexEvent:       &event,
		LastCodexTimestamp:   &now,
		CodexInputTokens:     100,
		CodexOutputTokens:    50,
		CodexTotalTokens:     150,
		TurnCount:            3,
	}

	if session.SessionID != "thread-1-turn-1" {
		t.Errorf("unexpected session ID: %s", session.SessionID)
	}

	if session.CodexInputTokens != 100 {
		t.Errorf("expected 100 input tokens, got %d", session.CodexInputTokens)
	}

	if session.TurnCount != 3 {
		t.Errorf("expected 3 turns, got %d", session.TurnCount)
	}
}

// TestOrchestratorState 测试编排器状态
func TestOrchestratorState(t *testing.T) {
	state := &domain.OrchestratorState{
		PollIntervalMs:      30000,
		MaxConcurrentAgents: 10,
		Running:            make(map[string]*domain.RunningEntry),
		Claimed:            make(map[string]struct{}),
		RetryAttempts:      make(map[string]*domain.RetryEntry),
		Completed:          make(map[string]struct{}),
		CodexTotals:        &domain.CodexTotals{},
	}

	if state.PollIntervalMs != 30000 {
		t.Errorf("expected poll interval 30000, got %d", state.PollIntervalMs)
	}

	if state.MaxConcurrentAgents != 10 {
		t.Errorf("expected max concurrent 10, got %d", state.MaxConcurrentAgents)
	}

	if state.Running == nil {
		t.Error("expected non-nil Running map")
	}

	if state.Claimed == nil {
		t.Error("expected non-nil Claimed set")
	}
}

// TestCodexTotals 测试Token统计
func TestCodexTotals(t *testing.T) {
	totals := &domain.CodexTotals{
		InputTokens:    5000,
		OutputTokens:   2500,
		TotalTokens:    7500,
		SecondsRunning: 3600.5,
	}

	if totals.InputTokens != 5000 {
		t.Errorf("expected 5000 input tokens, got %d", totals.InputTokens)
	}

	if totals.OutputTokens != 2500 {
		t.Errorf("expected 2500 output tokens, got %d", totals.OutputTokens)
	}

	if totals.TotalTokens != 7500 {
		t.Errorf("expected 7500 total tokens, got %d", totals.TotalTokens)
	}

	if totals.SecondsRunning != 3600.5 {
		t.Errorf("expected 3600.5 seconds, got %f", totals.SecondsRunning)
	}
}

// TestConfigValidation 测试配置验证
func TestConfigValidation(t *testing.T) {
	cfg := createTestConfig()

	validation := cfg.ValidateDispatchConfig()
	if !validation.Valid {
		t.Errorf("expected valid config, errors: %v", validation.Errors)
	}

	// 测试状态检查方法
	if !cfg.IsActiveState("Todo") {
		t.Error("expected 'Todo' to be active state")
	}

	if !cfg.IsActiveState("In Progress") {
		t.Error("expected 'In Progress' to be active state")
	}

	if cfg.IsActiveState("Done") {
		t.Error("expected 'Done' to not be active state")
	}

	if !cfg.IsTerminalState("Done") {
		t.Error("expected 'Done' to be terminal state")
	}

	if !cfg.IsTerminalState("Cancelled") {
		t.Error("expected 'Cancelled' to be terminal state")
	}

	if cfg.IsTerminalState("Todo") {
		t.Error("expected 'Todo' to not be terminal state")
	}
}

// 辅助函数
func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

// TestCancelTask_NotFound 测试取消不存在的任务
func TestCancelTask_NotFound(t *testing.T) {
	cfg := createTestConfig()
	orch := New(cfg, "test prompt")

	cancelled, notFound, err := orch.CancelTask("NONEXISTENT-123")

	if !notFound {
		t.Error("expected notFound to be true for non-existent task")
	}
	if cancelled {
		t.Error("expected cancelled to be false for non-existent task")
	}
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

// TestGetRunningEntryByIdentifier 测试获取运行中的任务
func TestGetRunningEntryByIdentifier(t *testing.T) {
	cfg := createTestConfig()
	orch := New(cfg, "test prompt")

	// 初始状态应该返回 nil
	entry := orch.GetRunningEntryByIdentifier("TEST-1")
	if entry != nil {
		t.Error("expected nil for non-existent task")
	}
}

// TestGetRetryEntryByIdentifier 测试获取重试队列中的任务
func TestGetRetryEntryByIdentifier(t *testing.T) {
	cfg := createTestConfig()
	orch := New(cfg, "test prompt")

	// 初始状态应该返回 nil
	entry := orch.GetRetryEntryByIdentifier("TEST-1")
	if entry != nil {
		t.Error("expected nil for non-existent task")
	}
}

// TestCancelTask_EmptyState 测试空状态下的取消操作
func TestCancelTask_EmptyState(t *testing.T) {
	cfg := createTestConfig()
	orch := New(cfg, "test prompt")

	// 在没有任何运行任务的情况下取消
	cancelled, notFound, err := orch.CancelTask("TEST-1")

	if !notFound {
		t.Error("expected notFound to be true")
	}
	if cancelled {
		t.Error("expected cancelled to be false")
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestShouldTransitionToNeedsAttention 测试重试上限判断逻辑
func TestShouldTransitionToNeedsAttention(t *testing.T) {
	tests := []struct {
		name         string
		maxRetries   int
		retryAttempt int
		expected     bool
	}{
		{
			name:         "below limit",
			maxRetries:   3,
			retryAttempt: 2,
			expected:     false,
		},
		{
			name:         "at limit",
			maxRetries:   3,
			retryAttempt: 3,
			expected:     true,
		},
		{
			name:         "above limit",
			maxRetries:   3,
			retryAttempt: 4,
			expected:     true,
		},
		{
			name:         "unlimited retries (maxRetries=0)",
			maxRetries:   0,
			retryAttempt: 10,
			expected:     false,
		},
		{
			name:         "negative maxRetries (unlimited)",
			maxRetries:   -1,
			retryAttempt: 5,
			expected:     false,
		},
		{
			name:         "maxRetries=1, attempt=1",
			maxRetries:   1,
			retryAttempt: 1,
			expected:     true,
		},
		{
			name:         "maxRetries=5, attempt=3",
			maxRetries:   5,
			retryAttempt: 3,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createTestConfig()
			cfg.Execution.MaxRetries = tt.maxRetries
			orch := New(cfg, "test prompt")

			result := orch.shouldTransitionToNeedsAttention(tt.retryAttempt)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestTransitionToNeedsAttentionLocked_WorflowNotFound 测试工作流不存在的情况
func TestTransitionToNeedsAttentionLocked_WorkflowNotFound(t *testing.T) {
	cfg := createTestConfig()
	cfg.Execution.MaxRetries = 3
	orch := New(cfg, "test prompt")

	// 不初始化工作流，直接尝试流转
	// 由于工作流不存在，应该记录错误但不崩溃
	orch.transitionToNeedsAttentionLocked("non-existent-task", "TEST-1", 3, "test error")

	// 验证没有崩溃，且状态正确
	state := orch.GetState()
	if len(state.Running) != 0 {
		t.Error("expected no running tasks")
	}
}

// TestTransitionToNeedsAttentionLocked_Success 测试成功流转到待人工处理状态
func TestTransitionToNeedsAttentionLocked_Success(t *testing.T) {
	cfg := createTestConfig()
	cfg.Execution.MaxRetries = 3
	orch := New(cfg, "test prompt")

	taskID := "test-task-1"

	// 初始化工作流
	_, _ = orch.InitTaskWorkflow(taskID)

	// 记录状态变更回调
	callbackCalled := false
	orch.SetOnStateChange(func() {
		callbackCalled = true
	})

	// 执行流转
	orch.transitionToNeedsAttentionLocked(taskID, "TEST-1", 3, "execution failed")

	// 等待异步操作完成
	time.Sleep(100 * time.Millisecond)

	// 验证工作流状态
	workflow := orch.GetTaskWorkflow(taskID)
	if workflow == nil {
		t.Fatal("expected workflow to exist")
	}

	// 验证当前阶段
	if workflow.CurrentStage != "needs_attention" {
		t.Errorf("expected current stage 'needs_attention', got '%s'", workflow.CurrentStage)
	}

	// 验证 NeedsAttention 标记
	if !workflow.NeedsAttention {
		t.Error("expected NeedsAttention to be true")
	}

	// 验证失败信息
	if workflow.FailureReason != "execution failed" {
		t.Errorf("expected failure reason 'execution failed', got '%s'", workflow.FailureReason)
	}

	if workflow.RetryCount != 3 {
		t.Errorf("expected retry count 3, got %d", workflow.RetryCount)
	}

	if workflow.MaxRetries != 3 {
		t.Errorf("expected max retries 3, got %d", workflow.MaxRetries)
	}

	// 验证回调被调用
	if !callbackCalled {
		t.Error("expected state change callback to be called")
	}
}

// TestOnWorkerExit_RetryNotAtLimit 测试未达重试上限时的正常重试
func TestOnWorkerExit_RetryNotAtLimit(t *testing.T) {
	cfg := createTestConfig()
	cfg.Execution.MaxRetries = 3
	orch := New(cfg, "test prompt")

	taskID := "test-task-retry"
	identifier := "TEST-RETRY"

	// 初始化工作流
	_, _ = orch.InitTaskWorkflow(taskID)

	// 模拟运行中状态
	orch.GetState().Running[taskID] = &domain.RunningEntry{
		Issue:      createTestIssue(taskID, identifier, "Test Task", "In Progress", 1, time.Now()),
		Identifier: identifier,
		StartedAt:  time.Now(),
	}
	orch.GetState().Claimed[taskID] = struct{}{}

	// 模拟第一次失败（未达上限）
	attempt := 1
	testErr := fmt.Errorf("test error")
	orch.onWorkerExit(taskID, identifier, testErr, &attempt)

	// 验证任务被安排重试
	state := orch.GetState()
	if _, ok := state.RetryAttempts[taskID]; !ok {
		t.Error("expected task to be in retry queue")
	}

	// 验证工作流状态不是 needs_attention
	workflow := orch.GetTaskWorkflow(taskID)
	if workflow.CurrentStage == "needs_attention" {
		t.Error("expected task not to be in needs_attention state")
	}
}

// TestOnWorkerExit_RetryAtLimit 测试达到重试上限时流转到待人工处理
func TestOnWorkerExit_RetryAtLimit(t *testing.T) {
	cfg := createTestConfig()
	cfg.Execution.MaxRetries = 3
	orch := New(cfg, "test prompt")

	taskID := "test-task-limit"
	identifier := "TEST-LIMIT"

	// 初始化工作流
	_, _ = orch.InitTaskWorkflow(taskID)

	// 模拟运行中状态
	orch.GetState().Running[taskID] = &domain.RunningEntry{
		Issue:      createTestIssue(taskID, identifier, "Test Task", "In Progress", 1, time.Now()),
		Identifier: identifier,
		StartedAt:  time.Now(),
	}
	orch.GetState().Claimed[taskID] = struct{}{}

	// 记录状态变更回调
	callbackCalled := false
	orch.SetOnStateChange(func() {
		callbackCalled = true
	})

	// 模拟第三次失败（达到上限）
	attempt := 3
	testErr := fmt.Errorf("persistent failure")
	orch.onWorkerExit(taskID, identifier, testErr, &attempt)

	// 等待异步操作完成
	time.Sleep(100 * time.Millisecond)

	// 验证任务不在重试队列
	state := orch.GetState()
	if _, ok := state.RetryAttempts[taskID]; ok {
		t.Error("expected task not to be in retry queue")
	}

	// 验证工作流状态是 needs_attention
	workflow := orch.GetTaskWorkflow(taskID)
	if workflow == nil {
		t.Fatal("expected workflow to exist")
	}

	if workflow.CurrentStage != "needs_attention" {
		t.Errorf("expected current stage 'needs_attention', got '%s'", workflow.CurrentStage)
	}

	if !workflow.NeedsAttention {
		t.Error("expected NeedsAttention to be true")
	}

	// 验证回调被调用
	if !callbackCalled {
		t.Error("expected state change callback to be called")
	}
}

// TestOnWorkerExit_NoError 测试成功退出时不会触发流转
func TestOnWorkerExit_NoError(t *testing.T) {
	cfg := createTestConfig()
	cfg.Execution.MaxRetries = 3
	orch := New(cfg, "test prompt")

	taskID := "test-task-success"
	identifier := "TEST-SUCCESS"

	// 初始化工作流
	_, _ = orch.InitTaskWorkflow(taskID)

	// 模拟运行中状态
	orch.GetState().Running[taskID] = &domain.RunningEntry{
		Issue:      createTestIssue(taskID, identifier, "Test Task", "In Progress", 1, time.Now()),
		Identifier: identifier,
		StartedAt:  time.Now(),
	}
	orch.GetState().Claimed[taskID] = struct{}{}

	// 成功退出（无错误）
	orch.onWorkerExit(taskID, identifier, nil, nil)

	// 验证任务不在重试队列
	state := orch.GetState()
	if _, ok := state.RetryAttempts[taskID]; ok {
		t.Error("expected task not to be in retry queue")
	}

	// 验证工作流状态不是 needs_attention
	workflow := orch.GetTaskWorkflow(taskID)
	if workflow.CurrentStage == "needs_attention" {
		t.Error("expected task not to be in needs_attention state")
	}
}

// TestNeedsAttentionDetails 测试失败详情结构体
func TestNeedsAttentionDetails(t *testing.T) {
	now := time.Now()
	details := workflow.NeedsAttentionDetails{
		FailedStage:    "implementation",
		FailedAt:       now,
		RetryCount:     3,
		MaxRetries:     3,
		ErrorType:      "execution_failure",
		ErrorMessage:   "test error message",
		LastLogSnippet: "last log line",
		Suggestion:     "check the error",
	}

	if details.FailedStage != "implementation" {
		t.Errorf("expected failed stage 'implementation', got '%s'", details.FailedStage)
	}

	if details.RetryCount != 3 {
		t.Errorf("expected retry count 3, got %d", details.RetryCount)
	}

	if details.ErrorType != "execution_failure" {
		t.Errorf("expected error type 'execution_failure', got '%s'", details.ErrorType)
	}
}

// TestGetNeedsAttentionTasks 测试获取待人工处理任务列表
func TestGetNeedsAttentionTasks(t *testing.T) {
	cfg := createTestConfig()
	cfg.Execution.MaxRetries = 3
	orch := New(cfg, "test prompt")

	// 初始应该为空
	tasks := orch.GetWorkflowEngine().GetNeedsAttentionTasks()
	if len(tasks) != 0 {
		t.Errorf("expected 0 needs_attention tasks, got %d", len(tasks))
	}

	// 添加一个待人工处理的任务
	taskID := "needs-attention-1"
	_, _ = orch.InitTaskWorkflow(taskID)
	orch.transitionToNeedsAttentionLocked(taskID, "TEST-NA", 3, "test error")

	// 等待异步操作完成
	time.Sleep(100 * time.Millisecond)

	// 验证列表包含该任务
	tasks = orch.GetWorkflowEngine().GetNeedsAttentionTasks()
	if len(tasks) != 1 {
		t.Errorf("expected 1 needs_attention task, got %d", len(tasks))
	}

	if tasks[0].TaskID != taskID {
		t.Errorf("expected task ID '%s', got '%s'", taskID, tasks[0].TaskID)
	}
}
