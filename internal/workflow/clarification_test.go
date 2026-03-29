// Package workflow_test 测试澄清轮次限制和跳过功能
package workflow_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/domain"
	"github.com/dministrator/symphony/internal/workflow"
)

func TestNewClarificationManager(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := workflow.NewEngine()
	manager := workflow.NewClarificationManager(engine, cfg)

	if manager == nil {
		t.Fatal("expected non-nil manager")
	}
}

func TestCheckRoundLimit(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Clarification.MaxRounds = 5
	engine := workflow.NewEngine()
	manager := workflow.NewClarificationManager(engine, cfg)

	// 初始化任务
	taskID := "test-round-limit"
	engine.InitTask(taskID)

	// 检查初始状态（轮次为0，不应达到上限）
	reached, currentRound, maxRounds, err := manager.CheckRoundLimit(taskID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if reached {
		t.Error("expected not reached at round 0")
	}
	if currentRound != 0 {
		t.Errorf("expected current round 0, got %d", currentRound)
	}
	if maxRounds != 5 {
		t.Errorf("expected max rounds 5, got %d", maxRounds)
	}

	// 增加轮次到上限
	for i := 0; i < 5; i++ {
		engine.IncrementRound(taskID)
	}

	// 再次检查（轮次为5，应达到上限）
	reached, currentRound, _, err = manager.CheckRoundLimit(taskID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reached {
		t.Error("expected reached at round 5")
	}
	if currentRound != 5 {
		t.Errorf("expected current round 5, got %d", currentRound)
	}
}

func TestCheckRoundLimitNotFound(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := workflow.NewEngine()
	manager := workflow.NewClarificationManager(engine, cfg)

	// 检查不存在的任务
	_, _, _, err := manager.CheckRoundLimit("non-existent")
	if err != workflow.ErrWorkflowNotFound {
		t.Errorf("expected ErrWorkflowNotFound, got %v", err)
	}
}

func TestSkipClarification(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := workflow.NewEngine()
	manager := workflow.NewClarificationManager(engine, cfg)

	// 初始化任务
	taskID := "test-skip"
	engine.InitTask(taskID)

	// 跳过澄清
	wf, err := manager.SkipClarification(taskID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证状态
	if !wf.IsIncomplete {
		t.Error("expected IsIncomplete to be true")
	}
	if wf.IncompleteReason != string(workflow.ReasonUserSkip) {
		t.Errorf("expected incomplete reason '%s', got '%s'", workflow.ReasonUserSkip, wf.IncompleteReason)
	}
	if wf.NeedsAttention {
		t.Error("expected NeedsAttention to be false (should advance to next stage)")
	}
	if wf.CurrentStage != workflow.StageBDDReview {
		t.Errorf("expected current stage bdd_review, got %s", wf.CurrentStage)
	}

	// 验证澄清阶段状态
	clarification := wf.Stages[workflow.StageClarification]
	if clarification.Status != workflow.StatusCompleted {
		t.Errorf("expected clarification completed, got %s", clarification.Status)
	}

	// 验证BDD阶段状态
	bddReview := wf.Stages[workflow.StageBDDReview]
	if bddReview.Status != workflow.StatusInProgress {
		t.Errorf("expected bdd_review in_progress, got %s", bddReview.Status)
	}
}

func TestSkipClarificationNotInClarificationStage(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := workflow.NewEngine()
	manager := workflow.NewClarificationManager(engine, cfg)

	// 初始化任务并推进到下一阶段
	taskID := "test-skip-invalid"
	engine.InitTask(taskID)
	engine.AdvanceStage(taskID) // 推进到BDD评审

	// 尝试跳过澄清（应该失败）
	_, err := manager.SkipClarification(taskID)
	if err == nil {
		t.Error("expected error when skipping from non-clarification stage")
	}
}

func TestSkipClarificationNotFound(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := workflow.NewEngine()
	manager := workflow.NewClarificationManager(engine, cfg)

	// 跳过不存在的任务
	_, err := manager.SkipClarification("non-existent")
	if err != workflow.ErrWorkflowNotFound {
		t.Errorf("expected ErrWorkflowNotFound, got %v", err)
	}
}

func TestHandleRoundLimitReached(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Clarification.MaxRounds = 3
	engine := workflow.NewEngine()
	manager := workflow.NewClarificationManager(engine, cfg)

	// 初始化任务
	taskID := "test-limit-reached"
	engine.InitTask(taskID)

	// 增加轮次到上限
	for i := 0; i < 3; i++ {
		engine.IncrementRound(taskID)
	}

	// 处理达到上限
	wf, err := manager.HandleRoundLimitReached(taskID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证状态
	if !wf.IsIncomplete {
		t.Error("expected IsIncomplete to be true")
	}
	if wf.IncompleteReason != string(workflow.ReasonRoundLimit) {
		t.Errorf("expected incomplete reason '%s', got '%s'", workflow.ReasonRoundLimit, wf.IncompleteReason)
	}
	if !wf.NeedsAttention {
		t.Error("expected NeedsAttention to be true (should not advance)")
	}

	// 验证澄清阶段状态（应该是失败）
	clarification := wf.Stages[workflow.StageClarification]
	if clarification.Status != workflow.StatusFailed {
		t.Errorf("expected clarification failed, got %s", clarification.Status)
	}

	// 验证当前阶段（应该还在澄清阶段）
	if wf.CurrentStage != workflow.StageClarification {
		t.Errorf("expected current stage clarification, got %s", wf.CurrentStage)
	}
}

func TestIncrementRoundWithLimitCheck(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Clarification.MaxRounds = 3
	engine := workflow.NewEngine()
	manager := workflow.NewClarificationManager(engine, cfg)

	// 初始化任务
	taskID := "test-increment"
	engine.InitTask(taskID)

	// 增加轮次并检查上限
	for i := 1; i <= 3; i++ {
		newRound, reachedLimit, err := manager.IncrementRound(taskID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if newRound != i {
			t.Errorf("expected new round %d, got %d", i, newRound)
		}

		// 第3轮时应该达到上限
		expectedReached := i >= 3
		if reachedLimit != expectedReached {
			t.Errorf("round %d: expected reached %v, got %v", i, expectedReached, reachedLimit)
		}
	}
}

func TestGetClarificationStatus(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Clarification.MaxRounds = 5
	engine := workflow.NewEngine()
	manager := workflow.NewClarificationManager(engine, cfg)

	// 初始化任务
	taskID := "test-status"
	engine.InitTask(taskID)

	// 增加轮次
	engine.IncrementRound(taskID)
	engine.IncrementRound(taskID)

	// 获取状态
	status, err := manager.GetClarificationStatus(taskID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.TaskID != taskID {
		t.Errorf("expected taskID %s, got %s", taskID, status.TaskID)
	}
	if status.CurrentRound != 2 {
		t.Errorf("expected current round 2, got %d", status.CurrentRound)
	}
	if status.MaxRounds != 5 {
		t.Errorf("expected max rounds 5, got %d", status.MaxRounds)
	}
	if status.RoundRemaining != 3 {
		t.Errorf("expected round remaining 3, got %d", status.RoundRemaining)
	}
	if status.RoundLimitReached {
		t.Error("expected round limit not reached at round 2")
	}
	if status.Status != workflow.StatusInProgress {
		t.Errorf("expected status in_progress, got %s", status.Status)
	}
	if status.IsIncomplete {
		t.Error("expected not incomplete initially")
	}
}

func TestGetClarificationStatusAfterSkip(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := workflow.NewEngine()
	manager := workflow.NewClarificationManager(engine, cfg)

	// 初始化任务并跳过
	taskID := "test-status-skip"
	engine.InitTask(taskID)
	manager.SkipClarification(taskID)

	// 获取状态
	status, err := manager.GetClarificationStatus(taskID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !status.IsIncomplete {
		t.Error("expected incomplete after skip")
	}
	if status.IncompleteReason != workflow.ReasonUserSkip {
		t.Errorf("expected reason '%s', got '%s'", workflow.ReasonUserSkip, status.IncompleteReason)
	}
	if status.NeedsAttention {
		t.Error("expected not needs_attention after skip")
	}
}

func TestCanSkipClarification(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := workflow.NewEngine()
	manager := workflow.NewClarificationManager(engine, cfg)

	// 初始化任务
	taskID := "test-can-skip"
	engine.InitTask(taskID)

	// 应该可以跳过
	canSkip, err := manager.CanSkipClarification(taskID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !canSkip {
		t.Error("expected can skip in clarification stage in_progress")
	}

	// 推进到下一阶段
	engine.AdvanceStage(taskID)

	// 不应该可以跳过
	canSkip, err = manager.CanSkipClarification(taskID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if canSkip {
		t.Error("expected cannot skip in non-clarification stage")
	}
}

func TestCanSkipClarificationAfterSkip(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := workflow.NewEngine()
	manager := workflow.NewClarificationManager(engine, cfg)

	// 初始化任务并跳过
	taskID := "test-can-skip-after-skip"
	engine.InitTask(taskID)
	manager.SkipClarification(taskID)

	// 不应该可以跳过（已经在BDD阶段）
	canSkip, err := manager.CanSkipClarification(taskID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if canSkip {
		t.Error("expected cannot skip after already skipped")
	}
}

func TestIsInClarificationStage(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := workflow.NewEngine()
	manager := workflow.NewClarificationManager(engine, cfg)

	// 初始化任务
	taskID := "test-in-stage"
	engine.InitTask(taskID)

	// 应该在澄清阶段
	inStage, err := manager.IsInClarificationStage(taskID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !inStage {
		t.Error("expected in clarification stage")
	}

	// 推进到下一阶段
	engine.AdvanceStage(taskID)

	// 不应该在澄清阶段
	inStage, err = manager.IsInClarificationStage(taskID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inStage {
		t.Error("expected not in clarification stage after advance")
	}
}

func TestCompleteClarification(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := workflow.NewEngine()
	manager := workflow.NewClarificationManager(engine, cfg)

	// 初始化任务
	taskID := "test-complete"
	engine.InitTask(taskID)

	// 正常完成澄清（不标记不完整）
	wf, err := manager.CompleteClarification(taskID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证状态
	if wf.IsIncomplete {
		t.Error("expected not incomplete after normal completion")
	}
	if wf.NeedsAttention {
		t.Error("expected not needs_attention after normal completion")
	}
	if wf.CurrentStage != workflow.StageBDDReview {
		t.Errorf("expected current stage bdd_review, got %s", wf.CurrentStage)
	}

	// 验证澄清阶段状态
	clarification := wf.Stages[workflow.StageClarification]
	if clarification.Status != workflow.StatusCompleted {
		t.Errorf("expected clarification completed, got %s", clarification.Status)
	}
}

func TestShouldAdvanceToNeedsAttention(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Clarification.MaxRounds = 3
	engine := workflow.NewEngine()
	manager := workflow.NewClarificationManager(engine, cfg)

	// 初始化任务
	taskID := "test-should-advance"
	engine.InitTask(taskID)

	// 初始状态不应该需要 attention
	should, err := manager.ShouldAdvanceToNeedsAttention(taskID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if should {
		t.Error("expected should not advance to needs_attention at round 0")
	}

	// 增加轮次到上限
	for i := 0; i < 3; i++ {
		engine.IncrementRound(taskID)
	}

	// 达到上限后应该需要 attention
	should, err = manager.ShouldAdvanceToNeedsAttention(taskID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !should {
		t.Error("expected should advance to needs_attention at round 3")
	}
}

func TestMarkIncomplete(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := workflow.NewEngine()
	manager := workflow.NewClarificationManager(engine, cfg)

	// 初始化任务
	taskID := "test-mark-incomplete"
	engine.InitTask(taskID)

	// 标记不完整（使用自定义原因）
	wf, err := manager.MarkIncomplete(taskID, workflow.ReasonUserSkip)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证状态
	if !wf.IsIncomplete {
		t.Error("expected IsIncomplete to be true")
	}
	if wf.IncompleteReason != string(workflow.ReasonUserSkip) {
		t.Errorf("expected incomplete reason '%s', got '%s'", workflow.ReasonUserSkip, wf.IncompleteReason)
	}
}

func TestGetMaxRounds(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Clarification.MaxRounds = 10
	engine := workflow.NewEngine()
	manager := workflow.NewClarificationManager(engine, cfg)

	maxRounds := manager.GetMaxRounds()
	if maxRounds != 10 {
		t.Errorf("expected max rounds 10, got %d", maxRounds)
	}
}

func TestClarificationReasons(t *testing.T) {
	// 验证原因常量
	if workflow.ReasonRoundLimit != "澄清轮次已达上限" {
		t.Errorf("unexpected ReasonRoundLimit: %s", workflow.ReasonRoundLimit)
	}
	if workflow.ReasonUserSkip != "用户跳过澄清" {
		t.Errorf("unexpected ReasonUserSkip: %s", workflow.ReasonUserSkip)
	}
}

func TestEngineIsIncomplete(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := workflow.NewEngine()
	manager := workflow.NewClarificationManager(engine, cfg)

	// 初始化任务
	taskID := "test-engine-incomplete"
	engine.InitTask(taskID)

	// 初始状态不应该是不完整
	incomplete, err := engine.IsIncomplete(taskID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if incomplete {
		t.Error("expected not incomplete initially")
	}

	// 跳过澄清
	manager.SkipClarification(taskID)

	// 应该标记为不完整
	incomplete, err = engine.IsIncomplete(taskID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !incomplete {
		t.Error("expected incomplete after skip")
	}
}

func TestEngineIsNeedsAttention(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Clarification.MaxRounds = 2
	engine := workflow.NewEngine()
	manager := workflow.NewClarificationManager(engine, cfg)

	// 初始化任务
	taskID := "test-engine-needs-attention"
	engine.InitTask(taskID)

	// 增加轮次到上限
	engine.IncrementRound(taskID)
	engine.IncrementRound(taskID)

	// 处理达到上限
	manager.HandleRoundLimitReached(taskID)

	// 应该标记为需要人工处理
	needsAttention, err := engine.IsNeedsAttention(taskID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !needsAttention {
		t.Error("expected needs_attention after round limit")
	}
}

func TestEngineGetIncompleteReason(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := workflow.NewEngine()
	manager := workflow.NewClarificationManager(engine, cfg)

	// 初始化任务
	taskID := "test-engine-reason"
	engine.InitTask(taskID)

	// 跳过澄清
	manager.SkipClarification(taskID)

	// 获取原因
	reason, err := engine.GetIncompleteReason(taskID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reason != string(workflow.ReasonUserSkip) {
		t.Errorf("expected reason '%s', got '%s'", workflow.ReasonUserSkip, reason)
	}
}

func TestIncompleteReasonSerialization(t *testing.T) {
	// 测试不完整原因可以正确转换为字符串
	reasons := []workflow.IncompleteReason{
		workflow.ReasonRoundLimit,
		workflow.ReasonUserSkip,
	}

	for _, reason := range reasons {
		str := string(reason)
		if str == "" {
			t.Errorf("reason %v should not be empty string", reason)
		}
	}
}

func TestClarificationStatusType(t *testing.T) {
	// 验证状态类型常量
	if workflow.StatusNeedsClarification != "needs_clarification" {
		t.Errorf("unexpected StatusNeedsClarification: %s", workflow.StatusNeedsClarification)
	}
	if workflow.StatusClear != "clear" {
		t.Errorf("unexpected StatusClear: %s", workflow.StatusClear)
	}
}

func TestClarificationQuestionStruct(t *testing.T) {
	question := workflow.ClarificationQuestion{
		ID:       "q1",
		Question: "What is the main requirement?",
	}

	if question.ID != "q1" {
		t.Errorf("expected ID q1, got %s", question.ID)
	}
	if question.Question != "What is the main requirement?" {
		t.Errorf("expected question, got %s", question.Question)
	}
}

func TestClarificationResponseStruct(t *testing.T) {
	response := workflow.ClarificationResponse{
		Status: workflow.StatusNeedsClarification,
		Questions: []workflow.ClarificationQuestion{
			{ID: "q1", Question: "Test question"},
		},
		Summary: "Test summary",
	}

	if response.Status != workflow.StatusNeedsClarification {
		t.Errorf("expected status needs_clarification, got %s", response.Status)
	}
	if len(response.Questions) != 1 {
		t.Errorf("expected 1 question, got %d", len(response.Questions))
	}
	if response.Summary != "Test summary" {
		t.Errorf("expected summary 'Test summary', got '%s'", response.Summary)
	}
}

func TestClarificationResultStruct(t *testing.T) {
	result := workflow.ClarificationResult{
		Status:          workflow.StatusClear,
		Questions:       nil,
		Summary:         "Clear requirements",
		Error:           nil,
		WaitingForUser:  false,
	}

	if result.Status != workflow.StatusClear {
		t.Errorf("expected status clear, got %s", result.Status)
	}
	if result.WaitingForUser {
		t.Error("expected not waiting for user")
	}
}

// MockClarificationTracker 用于测试的 Mock Tracker
type MockClarificationTracker struct {
	conversations map[string][]domain.ConversationTurn
	stages        map[string]*domain.StageState
	tasks         map[string]*domain.Issue
}

func NewMockClarificationTracker() *MockClarificationTracker {
	return &MockClarificationTracker{
		conversations: make(map[string][]domain.ConversationTurn),
		stages:        make(map[string]*domain.StageState),
		tasks:         make(map[string]*domain.Issue),
	}
}

func (m *MockClarificationTracker) FetchCandidateIssues(ctx context.Context, activeStates []string) ([]*domain.Issue, error) {
	return nil, nil
}

func (m *MockClarificationTracker) FetchIssuesByStates(ctx context.Context, states []string) ([]*domain.Issue, error) {
	return nil, nil
}

func (m *MockClarificationTracker) FetchIssueStatesByIDs(ctx context.Context, ids []string) ([]*domain.Issue, error) {
	return nil, nil
}

func (m *MockClarificationTracker) GetTask(ctx context.Context, identifier string) (*domain.Issue, error) {
	if task, ok := m.tasks[identifier]; ok {
		return task, nil
	}
	return nil, fmt.Errorf("task not found: %s", identifier)
}

func (m *MockClarificationTracker) CheckAvailability() error {
	return nil
}

func (m *MockClarificationTracker) CreateTask(ctx context.Context, title, description string) (*domain.Issue, error) {
	return nil, nil
}

func (m *MockClarificationTracker) CreateSubTask(ctx context.Context, parentIdentifier string, title, description string, blockedBy []string) (*domain.Issue, error) {
	return nil, nil
}

func (m *MockClarificationTracker) UpdateStage(ctx context.Context, identifier string, stage domain.StageState) error {
	m.stages[identifier] = &stage
	return nil
}

func (m *MockClarificationTracker) GetStageState(ctx context.Context, identifier string) (*domain.StageState, error) {
	return m.stages[identifier], nil
}

func (m *MockClarificationTracker) AppendConversation(ctx context.Context, identifier string, turn domain.ConversationTurn) error {
	m.conversations[identifier] = append(m.conversations[identifier], turn)
	return nil
}

func (m *MockClarificationTracker) GetConversationHistory(ctx context.Context, identifier string) ([]domain.ConversationTurn, error) {
	return m.conversations[identifier], nil
}

func (m *MockClarificationTracker) ListTasksByState(ctx context.Context, states []string) ([]*domain.Issue, error) {
	return nil, nil
}

func (m *MockClarificationTracker) AddTask(identifier, id, title string) {
	m.tasks[identifier] = &domain.Issue{
		ID:         id,
		Identifier: identifier,
		Title:      title,
		State:      "Todo",
	}
}

func TestSubmitAnswer_NeedsMoreClarification(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Clarification.MaxRounds = 5
	engine := workflow.NewEngine()
	mockTracker := NewMockClarificationTracker()
	manager := workflow.NewClarificationManagerWithTracker(engine, cfg, mockTracker)

	// 初始化任务
	taskID := "test-submit-needs-more"
	identifier := "TEST-1"
	mockTracker.AddTask(identifier, taskID, "Test Task")
	engine.InitTask(taskID)

	// 提交一个简短回答（应该需要更多澄清）
	result, err := manager.SubmitAnswer(context.Background(), taskID, identifier, "yes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证结果
	if !result.NeedsMoreClarification {
		t.Error("expected needs more clarification for short answer")
	}
	if result.Question == "" {
		t.Error("expected question to be generated")
	}
	if result.Round != 1 {
		t.Errorf("expected round 1, got %d", result.Round)
	}
	if result.Status != workflow.StatusNeedsClarification {
		t.Errorf("expected status needs_clarification, got %s", result.Status)
	}
}

func TestSubmitAnswer_CompleteClarification(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Clarification.MaxRounds = 5
	engine := workflow.NewEngine()
	mockTracker := NewMockClarificationTracker()
	manager := workflow.NewClarificationManagerWithTracker(engine, cfg, mockTracker)

	// 初始化任务
	taskID := "test-submit-complete"
	identifier := "TEST-2"
	mockTracker.AddTask(identifier, taskID, "Test Task")
	engine.InitTask(taskID)

	// 增加轮次以便满足完成条件
	engine.IncrementRound(taskID)
	engine.IncrementRound(taskID)

	// 提交一个明确的回答
	result, err := manager.SubmitAnswer(context.Background(), taskID, identifier, "我需要实现用户登录功能，包括用户名密码登录和第三方登录")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证结果
	if result.NeedsMoreClarification {
		t.Error("expected clarification to be complete")
	}
	if result.Summary == "" {
		t.Error("expected summary to be provided")
	}
	if result.Status != workflow.StatusClear {
		t.Errorf("expected status clear, got %s", result.Status)
	}
}

func TestSubmitAnswer_RoundLimitReached(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Clarification.MaxRounds = 2
	engine := workflow.NewEngine()
	mockTracker := NewMockClarificationTracker()
	manager := workflow.NewClarificationManagerWithTracker(engine, cfg, mockTracker)

	// 初始化任务
	taskID := "test-submit-limit"
	identifier := "TEST-3"
	mockTracker.AddTask(identifier, taskID, "Test Task")
	engine.InitTask(taskID)

	// 增加轮次到上限-1
	engine.IncrementRound(taskID)

	// 提交回答（将达到上限）
	result, err := manager.SubmitAnswer(context.Background(), taskID, identifier, "test answer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证达到上限后的行为
	if result.NeedsMoreClarification {
		t.Error("expected no more clarification when limit reached")
	}
	if result.Summary == "" {
		t.Error("expected summary when limit reached")
	}
}

func TestSubmitAnswer_NoTracker(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := workflow.NewEngine()
	manager := workflow.NewClarificationManager(engine, cfg)

	// 初始化任务
	taskID := "test-submit-no-tracker"
	identifier := "TEST-4"
	engine.InitTask(taskID)

	// 提交回答（应该失败，因为没有 tracker）
	_, err := manager.SubmitAnswer(context.Background(), taskID, identifier, "test answer")
	if err == nil {
		t.Error("expected error when tracker not configured")
	}
}

func TestSubmitAnswer_WorkflowNotFound(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := workflow.NewEngine()
	mockTracker := NewMockClarificationTracker()
	manager := workflow.NewClarificationManagerWithTracker(engine, cfg, mockTracker)

	// 不初始化任务，直接提交回答
	_, err := manager.SubmitAnswer(context.Background(), "non-existent", "TEST-5", "test answer")
	if err != workflow.ErrWorkflowNotFound {
		t.Errorf("expected ErrWorkflowNotFound, got %v", err)
	}
}

func TestSubmitAnswer_WrongStage(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := workflow.NewEngine()
	mockTracker := NewMockClarificationTracker()
	manager := workflow.NewClarificationManagerWithTracker(engine, cfg, mockTracker)

	// 初始化任务并推进到下一阶段
	taskID := "test-submit-wrong-stage"
	identifier := "TEST-6"
	mockTracker.AddTask(identifier, taskID, "Test Task")
	engine.InitTask(taskID)
	engine.AdvanceStage(taskID) // 推进到 BDD 评审阶段

	// 尝试提交回答（应该失败，因为不在澄清阶段）
	_, err := manager.SubmitAnswer(context.Background(), taskID, identifier, "test answer")
	if err == nil {
		t.Error("expected error when not in clarification stage")
	}
}

func TestGetClarificationState(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := workflow.NewEngine()
	mockTracker := NewMockClarificationTracker()
	manager := workflow.NewClarificationManagerWithTracker(engine, cfg, mockTracker)

	// 初始化任务
	taskID := "test-get-state"
	identifier := "TEST-7"
	mockTracker.AddTask(identifier, taskID, "Test Task")
	engine.InitTask(taskID)

	// 获取澄清状态
	result, err := manager.GetClarificationState(context.Background(), taskID, identifier)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.NeedsMoreClarification {
		t.Error("expected needs more clarification initially")
	}
	if result.Round != 0 {
		t.Errorf("expected round 0, got %d", result.Round)
	}
}

func TestStartClarification(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := workflow.NewEngine()
	mockTracker := NewMockClarificationTracker()
	manager := workflow.NewClarificationManagerWithTracker(engine, cfg, mockTracker)

	// 开始澄清
	taskID := "test-start"
	identifier := "TEST-8"
	mockTracker.AddTask(identifier, taskID, "Test Task")

	result, err := manager.StartClarification(context.Background(), taskID, identifier, "请描述您的需求")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Question != "请描述您的需求" {
		t.Errorf("expected initial question, got %s", result.Question)
	}
}

func TestAnalyzeAnswer_ShortAnswer(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := workflow.NewEngine()
	mockTracker := NewMockClarificationTracker()
	manager := workflow.NewClarificationManagerWithTracker(engine, cfg, mockTracker)

	taskID := "test-analyze-short"
	identifier := "TEST-9"
	mockTracker.AddTask(identifier, taskID, "Test Task")
	engine.InitTask(taskID)

	// 提交短回答
	result, err := manager.SubmitAnswer(context.Background(), taskID, identifier, "ok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 短回答应该需要更多澄清
	if !result.NeedsMoreClarification {
		t.Error("expected needs more clarification for short answer")
	}
}

func TestAnalyzeAnswer_UnclearKeywords(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := workflow.NewEngine()
	mockTracker := NewMockClarificationTracker()
	manager := workflow.NewClarificationManagerWithTracker(engine, cfg, mockTracker)

	taskID := "test-analyze-unclear"
	identifier := "TEST-10"
	mockTracker.AddTask(identifier, taskID, "Test Task")
	engine.InitTask(taskID)

	// 提交包含不确定关键词的回答
	result, err := manager.SubmitAnswer(context.Background(), taskID, identifier, "这个需求我不太清楚具体怎么实现")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 应该需要更多澄清
	if !result.NeedsMoreClarification {
		t.Error("expected needs more clarification for unclear answer")
	}
}

// BDDReviewManager 测试
func TestNewBDDReviewManager(t *testing.T) {
	engine := workflow.NewEngine()
	manager := workflow.NewBDDReviewManager(engine)

	if manager == nil {
		t.Fatal("expected non-nil manager")
	}
}

func TestNewBDDReviewManagerWithTracker(t *testing.T) {
	engine := workflow.NewEngine()
	mockTracker := NewMockClarificationTracker()
	manager := workflow.NewBDDReviewManagerWithTracker(engine, mockTracker)

	if manager == nil {
		t.Fatal("expected non-nil manager")
	}
}

func TestBDDReviewManager_ApproveBDD(t *testing.T) {
	engine := workflow.NewEngine()
	manager := workflow.NewBDDReviewManager(engine)

	taskID := "test-bdd-approve"
	engine.InitTask(taskID)
	engine.AdvanceStage(taskID) // clarification -> bdd_review

	// 验证当前阶段
	wf := engine.GetWorkflow(taskID)
	if wf.CurrentStage != workflow.StageBDDReview {
		t.Fatalf("expected bdd_review, got %s", wf.CurrentStage)
	}

	// 通过 BDD 审核
	wf, err := manager.ApproveBDD(taskID)
	if err != nil {
		t.Fatalf("failed to approve BDD: %v", err)
	}

	// 验证状态流转
	if wf.CurrentStage != workflow.StageArchitectureReview {
		t.Errorf("expected architecture_review, got %s", wf.CurrentStage)
	}

	// 验证 BDD 阶段已完成
	bddStage := wf.Stages[workflow.StageBDDReview]
	if bddStage.Status != workflow.StatusCompleted {
		t.Errorf("expected bdd_review completed, got %s", bddStage.Status)
	}
}

func TestBDDReviewManager_ApproveBDD_NotInBDDReviewStage(t *testing.T) {
	engine := workflow.NewEngine()
	manager := workflow.NewBDDReviewManager(engine)

	taskID := "test-bdd-approve-invalid"
	engine.InitTask(taskID) // 在澄清阶段

	// 尝试通过 BDD 审核（应该失败）
	_, err := manager.ApproveBDD(taskID)
	if err == nil {
		t.Error("expected error when not in bdd_review stage")
	}
}

func TestBDDReviewManager_ApproveBDD_WorkflowNotFound(t *testing.T) {
	engine := workflow.NewEngine()
	manager := workflow.NewBDDReviewManager(engine)

	_, err := manager.ApproveBDD("non-existent")
	if err != workflow.ErrWorkflowNotFound {
		t.Errorf("expected ErrWorkflowNotFound, got %v", err)
	}
}

func TestBDDReviewManager_RejectBDD(t *testing.T) {
	engine := workflow.NewEngine()
	manager := workflow.NewBDDReviewManager(engine)

	taskID := "test-bdd-reject"
	engine.InitTask(taskID)
	engine.AdvanceStage(taskID) // clarification -> bdd_review

	// 驳回 BDD 审核
	reason := "BDD 规则不符合要求"
	wf, err := manager.RejectBDD(taskID, reason)
	if err != nil {
		t.Fatalf("failed to reject BDD: %v", err)
	}

	// 验证状态流转回澄清阶段
	if wf.CurrentStage != workflow.StageClarification {
		t.Errorf("expected clarification, got %s", wf.CurrentStage)
	}

	// 验证 BDD 阶段已失败
	bddStage := wf.Stages[workflow.StageBDDReview]
	if bddStage.Status != workflow.StatusFailed {
		t.Errorf("expected bdd_review failed, got %s", bddStage.Status)
	}

	// 验证驳回原因已记录
	if bddStage.Error != reason {
		t.Errorf("expected error '%s', got '%s'", reason, bddStage.Error)
	}

	// 验证澄清阶段重新开始
	clarificationStage := wf.Stages[workflow.StageClarification]
	if clarificationStage.Status != workflow.StatusInProgress {
		t.Errorf("expected clarification in_progress, got %s", clarificationStage.Status)
	}

	// 验证轮次已重置
	if clarificationStage.Round != 0 {
		t.Errorf("expected round 0, got %d", clarificationStage.Round)
	}
}

func TestBDDReviewManager_RejectBDD_NotInBDDReviewStage(t *testing.T) {
	engine := workflow.NewEngine()
	manager := workflow.NewBDDReviewManager(engine)

	taskID := "test-bdd-reject-invalid"
	engine.InitTask(taskID) // 在澄清阶段

	_, err := manager.RejectBDD(taskID, "test reason")
	if err == nil {
		t.Error("expected error when not in bdd_review stage")
	}
}

func TestBDDReviewManager_RejectBDD_WorkflowNotFound(t *testing.T) {
	engine := workflow.NewEngine()
	manager := workflow.NewBDDReviewManager(engine)

	_, err := manager.RejectBDD("non-existent", "test reason")
	if err != workflow.ErrWorkflowNotFound {
		t.Errorf("expected ErrWorkflowNotFound, got %v", err)
	}
}

func TestBDDReviewManager_GetBDDReviewStatus(t *testing.T) {
	engine := workflow.NewEngine()
	manager := workflow.NewBDDReviewManager(engine)

	taskID := "test-bdd-status"
	engine.InitTask(taskID)
	engine.AdvanceStage(taskID) // clarification -> bdd_review

	// 获取 BDD 审核状态
	status, err := manager.GetBDDReviewStatus(taskID)
	if err != nil {
		t.Fatalf("failed to get BDD review status: %v", err)
	}

	if status.TaskID != taskID {
		t.Errorf("expected taskID %s, got %s", taskID, status.TaskID)
	}
	if status.CurrentStage != workflow.StageBDDReview {
		t.Errorf("expected bdd_review, got %s", status.CurrentStage)
	}
	if !status.CanApprove {
		t.Error("expected can approve in bdd_review stage")
	}
	if !status.CanReject {
		t.Error("expected can reject in bdd_review stage")
	}
	if status.Approved {
		t.Error("expected not approved initially")
	}
	if status.Rejected {
		t.Error("expected not rejected initially")
	}
}

func TestBDDReviewManager_GetBDDReviewStatus_WorkflowNotFound(t *testing.T) {
	engine := workflow.NewEngine()
	manager := workflow.NewBDDReviewManager(engine)

	_, err := manager.GetBDDReviewStatus("non-existent")
	if err != workflow.ErrWorkflowNotFound {
		t.Errorf("expected ErrWorkflowNotFound, got %v", err)
	}
}

func TestBDDReviewManager_CanApproveOrReject(t *testing.T) {
	engine := workflow.NewEngine()
	manager := workflow.NewBDDReviewManager(engine)

	taskID := "test-can-approve-reject"
	engine.InitTask(taskID)

	// 在澄清阶段，不能审核
	canApproveOrReject, err := manager.CanApproveOrReject(taskID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if canApproveOrReject {
		t.Error("expected cannot approve/reject in clarification stage")
	}

	// 推进到 BDD 审核阶段
	engine.AdvanceStage(taskID)

	// 在 BDD 审核阶段，可以审核
	canApproveOrReject, err = manager.CanApproveOrReject(taskID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !canApproveOrReject {
		t.Error("expected can approve/reject in bdd_review stage")
	}
}

func TestBDDReviewManager_CanApproveOrReject_WorkflowNotFound(t *testing.T) {
	engine := workflow.NewEngine()
	manager := workflow.NewBDDReviewManager(engine)

	_, err := manager.CanApproveOrReject("non-existent")
	if err != workflow.ErrWorkflowNotFound {
		t.Errorf("expected ErrWorkflowNotFound, got %v", err)
	}
}

func TestBDDReviewManager_IsInBDDReviewStage(t *testing.T) {
	engine := workflow.NewEngine()
	manager := workflow.NewBDDReviewManager(engine)

	taskID := "test-is-in-bdd"
	engine.InitTask(taskID)

	// 在澄清阶段
	inStage, err := manager.IsInBDDReviewStage(taskID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inStage {
		t.Error("expected not in bdd_review stage initially")
	}

	// 推进到 BDD 审核阶段
	engine.AdvanceStage(taskID)

	// 在 BDD 审核阶段
	inStage, err = manager.IsInBDDReviewStage(taskID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !inStage {
		t.Error("expected in bdd_review stage after advance")
	}
}

func TestBDDReviewManager_SetTracker(t *testing.T) {
	engine := workflow.NewEngine()
	manager := workflow.NewBDDReviewManager(engine)
	mockTracker := NewMockClarificationTracker()

	manager.SetTracker(mockTracker)

	// 设置 tracker 后应该可以正常使用
	if manager == nil {
		t.Error("expected manager to be valid after setting tracker")
	}
}

func TestBDDReviewStatus_Struct(t *testing.T) {
	status := workflow.BDDReviewStatus{
		TaskID:        "test-task",
		CurrentStage:  workflow.StageBDDReview,
		Status:        workflow.StatusInProgress,
		CanApprove:    true,
		CanReject:     true,
		Approved:      false,
		Rejected:      false,
		RejectReason:  "",
		NeedsAttention: false,
	}

	if status.TaskID != "test-task" {
		t.Errorf("expected taskID test-task, got %s", status.TaskID)
	}
	if status.CurrentStage != workflow.StageBDDReview {
		t.Errorf("expected bdd_review, got %s", status.CurrentStage)
	}
	if !status.CanApprove {
		t.Error("expected can approve")
	}
	if !status.CanReject {
		t.Error("expected can reject")
	}
}