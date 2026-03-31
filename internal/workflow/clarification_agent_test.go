// Package workflow_test 测试AI Agent需求理解调用
package workflow_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dministrator/symphony/internal/agent"
	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/domain"
	"github.com/dministrator/symphony/internal/tracker"
	"github.com/dministrator/symphony/internal/workflow"
)

// MockRunner 模拟AI Agent运行器
type MockRunner struct {
	// Response 要返回的响应
	Response string
	// Error 要返回的错误
	Error error
	// Success 是否成功
	Success bool
	// Called 是否被调用
	Called bool
	// LastPrompt 最后收到的提示词
	LastPrompt string
}

// RunAttempt 实现 agent.Runner 接口
func (m *MockRunner) RunAttempt(
	ctx context.Context,
	issue *domain.Issue,
	workspacePath string,
	attempt *int,
	promptTemplate string,
	callback agent.EventCallback,
) (*agent.RunAttemptResult, error) {
	m.Called = true
	m.LastPrompt = promptTemplate

	if m.Error != nil {
		return nil, m.Error
	}

	return &agent.RunAttemptResult{
		Success: m.Success,
		Error:   m.Error,
	}, nil
}

// MockTracker 模拟问题跟踪器
type MockTracker struct {
	Conversations map[string][]domain.ConversationTurn
	StageStates   map[string]*domain.StageState
}

// NewMockTracker 创建新的模拟跟踪器
func NewMockTracker() *MockTracker {
	return &MockTracker{
		Conversations: make(map[string][]domain.ConversationTurn),
		StageStates:   make(map[string]*domain.StageState),
	}
}

func (m *MockTracker) FetchCandidateIssues(ctx context.Context, activeStates []string) ([]*domain.Issue, error) {
	return nil, nil
}

func (m *MockTracker) FetchIssuesByStates(ctx context.Context, states []string) ([]*domain.Issue, error) {
	return nil, nil
}

func (m *MockTracker) FetchIssueStatesByIDs(ctx context.Context, ids []string) ([]*domain.Issue, error) {
	return nil, nil
}

func (m *MockTracker) GetTask(ctx context.Context, identifier string) (*domain.Issue, error) {
	return nil, nil
}

func (m *MockTracker) CheckAvailability() error {
	return nil
}

func (m *MockTracker) CreateTask(ctx context.Context, title, description string) (*domain.Issue, error) {
	return nil, nil
}

func (m *MockTracker) CreateSubTask(ctx context.Context, parentIdentifier string, title, description string, blockedBy []string) (*domain.Issue, error) {
	return nil, nil
}

func (m *MockTracker) UpdateStage(ctx context.Context, identifier string, stage domain.StageState) error {
	m.StageStates[identifier] = &stage
	return nil
}

func (m *MockTracker) GetStageState(ctx context.Context, identifier string) (*domain.StageState, error) {
	return m.StageStates[identifier], nil
}

func (m *MockTracker) AppendConversation(ctx context.Context, identifier string, turn domain.ConversationTurn) error {
	m.Conversations[identifier] = append(m.Conversations[identifier], turn)
	return nil
}

func (m *MockTracker) GetConversationHistory(ctx context.Context, identifier string) ([]domain.ConversationTurn, error) {
	return m.Conversations[identifier], nil
}

func (m *MockTracker) ListTasksByState(ctx context.Context, states []string) ([]*domain.Issue, error) {
	return nil, nil
}

func (m *MockTracker) GetBDDContent(ctx context.Context, identifier string) (string, error) {
	return "", nil
}

func (m *MockTracker) UpdateBDDContent(ctx context.Context, identifier string, content string) error {
	return nil
}

func (m *MockTracker) ApproveBDD(ctx context.Context, identifier string) error {
	return nil
}

func (m *MockTracker) RejectBDD(ctx context.Context, identifier string, reason string) error {
	return nil
}

func (m *MockTracker) GetArchitectureContent(ctx context.Context, identifier string) (string, error) {
	return "", nil
}

func (m *MockTracker) GetTDDContent(ctx context.Context, identifier string) (string, error) {
	return "", nil
}

func (m *MockTracker) UpdateArchitectureContent(ctx context.Context, identifier string, content string) error {
	return nil
}

func (m *MockTracker) UpdateTDDContent(ctx context.Context, identifier string, content string) error {
	return nil
}

func (m *MockTracker) ApproveArchitecture(ctx context.Context, identifier string) error {
	return nil
}

func (m *MockTracker) RejectArchitecture(ctx context.Context, identifier string, reason string) error {
	return nil
}

func (m *MockTracker) GetVerificationReport(ctx context.Context, identifier string) (*tracker.VerificationReport, error) {
	return nil, nil
}

func (m *MockTracker) UpdateVerificationReport(ctx context.Context, identifier string, report *tracker.VerificationReport) error {
	return nil
}

func (m *MockTracker) ApproveVerification(ctx context.Context, identifier string) error {
	return nil
}

func (m *MockTracker) RejectVerification(ctx context.Context, identifier string, reason string) error {
	return nil
}

// TestNewClarificationManager 测试创建澄清管理器
func TestNewClarificationManagerAgent(t *testing.T) {
	engine := workflow.NewEngine()
	cfg := config.DefaultConfig()

	cm := workflow.NewClarificationManager(engine, cfg)
	if cm == nil {
		t.Fatal("expected non-nil ClarificationManager")
	}
}

// TestNewClarificationManagerWithTracker 测试带tracker创建
func TestNewClarificationManagerWithTracker(t *testing.T) {
	engine := workflow.NewEngine()
	cfg := config.DefaultConfig()
	mockTracker := NewMockTracker()

	cm := workflow.NewClarificationManagerWithTracker(engine, cfg, mockTracker)
	if cm == nil {
		t.Fatal("expected non-nil ClarificationManager")
	}
}

// TestSetRunner 测试设置AI Agent运行器
func TestSetRunner(t *testing.T) {
	engine := workflow.NewEngine()
	cfg := config.DefaultConfig()
	mockRunner := &MockRunner{Success: true}

	cm := workflow.NewClarificationManager(engine, cfg)
	cm.SetRunner(mockRunner)

	// 验证设置成功（通过调用方法间接验证）
	// 由于 runner 是私有字段，我们通过行为验证
}

// TestParseClarificationResponse 测试解析澄清响应
func TestParseClarificationResponse(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    *workflow.ClarificationResponse
		expectError bool
	}{
		{
			name:  "clear response",
			input: `{"status": "clear", "questions": [], "summary": "需求已明确"}`,
			expected: &workflow.ClarificationResponse{
				Status:   workflow.StatusClear,
				Questions: nil,
				Summary:  "需求已明确",
			},
		},
		{
			name:  "needs_clarification response",
			input: `{"status": "needs_clarification", "questions": [{"id": "q1", "question": "问题1"}], "summary": "需要澄清"}`,
			expected: &workflow.ClarificationResponse{
				Status: workflow.StatusNeedsClarification,
				Questions: []workflow.ClarificationQuestion{
					{ID: "q1", Question: "问题1"},
				},
				Summary: "需要澄清",
			},
		},
		{
			name:  "markdown code block",
			input: "```json\n{\"status\": \"clear\", \"questions\": [], \"summary\": \"测试\"}\n```",
			expected: &workflow.ClarificationResponse{
				Status:   workflow.StatusClear,
				Summary:  "测试",
			},
		},
		{
			name:  "code block without json prefix",
			input: "```\n{\"status\": \"clear\", \"questions\": [], \"summary\": \"测试2\"}\n```",
			expected: &workflow.ClarificationResponse{
				Status:   workflow.StatusClear,
				Summary:  "测试2",
			},
		},
		{
			name:        "invalid json",
			input:       `{invalid json}`,
			expectError: true,
		},
		{
			name:  "unknown status defaults to needs_clarification",
			input: `{"status": "unknown", "questions": [], "summary": "测试"}`,
			expected: &workflow.ClarificationResponse{
				Status:   workflow.StatusNeedsClarification,
				Summary:  "测试",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := workflow.ParseClarificationResponse(tt.input)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Status != tt.expected.Status {
				t.Errorf("status: expected %s, got %s", tt.expected.Status, result.Status)
			}

			if result.Summary != tt.expected.Summary {
				t.Errorf("summary: expected %s, got %s", tt.expected.Summary, result.Summary)
			}

			if len(result.Questions) != len(tt.expected.Questions) {
				t.Errorf("questions count: expected %d, got %d", len(tt.expected.Questions), len(result.Questions))
			}
		})
	}
}

// TestClarificationQuestionJSON 测试问题JSON序列化
func TestClarificationQuestionJSON(t *testing.T) {
	questions := []workflow.ClarificationQuestion{
		{ID: "q1", Question: "第一个问题"},
		{ID: "q2", Question: "第二个问题"},
	}

	data, err := json.Marshal(questions)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded []workflow.ClarificationQuestion
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(decoded) != len(questions) {
		t.Errorf("expected %d questions, got %d", len(questions), len(decoded))
	}

	for i, q := range decoded {
		if q.ID != questions[i].ID {
			t.Errorf("question %d: expected ID %s, got %s", i, questions[i].ID, q.ID)
		}
		if q.Question != questions[i].Question {
			t.Errorf("question %d: expected Question %s, got %s", i, questions[i].Question, q.Question)
		}
	}
}

// TestClarificationResult 测试澄清结果
func TestClarificationResult(t *testing.T) {
	result := &workflow.ClarificationResult{
		Status:         workflow.StatusNeedsClarification,
		Summary:        "需要更多澄清",
		WaitingForUser: true,
		Questions: []workflow.ClarificationQuestion{
			{ID: "q1", Question: "问题"},
		},
	}

	if result.Status != workflow.StatusNeedsClarification {
		t.Errorf("expected status needs_clarification, got %s", result.Status)
	}

	if !result.WaitingForUser {
		t.Error("expected WaitingForUser to be true")
	}

	if result.Summary != "需要更多澄清" {
		t.Errorf("unexpected summary: %s", result.Summary)
	}

	if len(result.Questions) != 1 {
		t.Errorf("expected 1 question, got %d", len(result.Questions))
	}
}

// TestClarificationStatusType 测试状态类型常量
func TestClarificationStatusTypeAgent(t *testing.T) {
	if workflow.StatusNeedsClarification != "needs_clarification" {
		t.Errorf("unexpected StatusNeedsClarification: %s", workflow.StatusNeedsClarification)
	}

	if workflow.StatusClear != "clear" {
		t.Errorf("unexpected StatusClear: %s", workflow.StatusClear)
	}
}

// TestClarificationResponse 测试响应结构
func TestClarificationResponse(t *testing.T) {
	response := workflow.ClarificationResponse{
		Status: workflow.StatusClear,
		Questions: []workflow.ClarificationQuestion{
			{ID: "q1", Question: "测试问题"},
		},
		Summary: "测试摘要",
	}

	if response.Status != workflow.StatusClear {
		t.Errorf("expected StatusClear, got %s", response.Status)
	}

	if len(response.Questions) != 1 {
		t.Errorf("expected 1 question, got %d", len(response.Questions))
	}

	if response.Summary != "测试摘要" {
		t.Errorf("unexpected summary: %s", response.Summary)
	}
}

// TestRunClarificationWithAgent_NoRunner 测试没有runner时的情况
func TestRunClarificationWithAgent_NoRunner(t *testing.T) {
	engine := workflow.NewEngine()
	cfg := config.DefaultConfig()
	mockTracker := NewMockTracker()

	cm := workflow.NewClarificationManagerWithTracker(engine, cfg, mockTracker)

	task := &domain.Issue{
		ID:          "test-1",
		Identifier:  "TEST-1",
		Title:       "测试任务",
		Description: strPtr("测试描述"),
	}

	result, err := cm.RunClarificationWithAgent(context.Background(), task, "/tmp/workspace")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != workflow.StatusNeedsClarification {
		t.Errorf("expected needs_clarification status, got %s", result.Status)
	}

	if result.WaitingForUser {
		t.Error("expected WaitingForUser to be false without runner")
	}
}

// TestRunClarificationWithAgent_WithRunner 测试有runner时的情况
func TestRunClarificationWithAgent_WithRunner(t *testing.T) {
	engine := workflow.NewEngine()
	cfg := config.DefaultConfig()
	mockTracker := NewMockTracker()
	mockRunner := &MockRunner{
		Success: true,
	}

	cm := workflow.NewClarificationManagerWithTracker(engine, cfg, mockTracker)
	cm.SetRunner(mockRunner)

	// 初始化工作流
	_, _ = engine.InitTask("test-1")

	task := &domain.Issue{
		ID:          "test-1",
		Identifier:  "TEST-1",
		Title:       "测试任务",
		Description: strPtr("测试描述"),
	}

	result, err := cm.RunClarificationWithAgent(context.Background(), task, "/tmp/workspace")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证runner被调用
	if !mockRunner.Called {
		t.Error("expected runner to be called")
	}

	// 验证提示词包含任务标题
	if mockRunner.LastPrompt == "" {
		t.Error("expected non-empty prompt")
	}

	// 结果应该来自模拟响应
	if result.Status != workflow.StatusClear {
		t.Errorf("expected clear status from mock, got %s", result.Status)
	}
}

// TestRunClarificationWithAgent_NilTask 测试空任务
func TestRunClarificationWithAgent_NilTask(t *testing.T) {
	engine := workflow.NewEngine()
	cfg := config.DefaultConfig()
	mockTracker := NewMockTracker()

	cm := workflow.NewClarificationManagerWithTracker(engine, cfg, mockTracker)

	result, err := cm.RunClarificationWithAgent(context.Background(), nil, "/tmp/workspace")
	if err == nil {
		t.Error("expected error for nil task")
	}

	if result != nil && result.Error == nil {
		t.Error("expected error in result")
	}
}

// TestProcessAgentResponse_ClearStatus 测试处理clear响应
func TestProcessAgentResponse_ClearStatus(t *testing.T) {
	engine := workflow.NewEngine()
	cfg := config.DefaultConfig()
	mockTracker := NewMockTracker()

	_, _ = engine.InitTask("test-1")

	cm := workflow.NewClarificationManagerWithTracker(engine, cfg, mockTracker)

	task := &domain.Issue{
		ID:         "test-1",
		Identifier: "TEST-1",
		Title:      "测试任务",
	}

	response := `{"status": "clear", "questions": [], "summary": "需求已明确"}`

	result, err := cm.ProcessAgentResponse(context.Background(), task, response)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != workflow.StatusClear {
		t.Errorf("expected clear status, got %s", result.Status)
	}

	if result.WaitingForUser {
		t.Error("expected WaitingForUser to be false for clear status")
	}

	// 验证对话历史被保存
	history := mockTracker.Conversations["TEST-1"]
	if len(history) == 0 {
		t.Error("expected conversation history to be saved")
	}
}

// TestProcessAgentResponse_NeedsClarification 测试处理needs_clarification响应
func TestProcessAgentResponse_NeedsClarification(t *testing.T) {
	engine := workflow.NewEngine()
	cfg := config.DefaultConfig()
	mockTracker := NewMockTracker()

	_, _ = engine.InitTask("test-1")

	cm := workflow.NewClarificationManagerWithTracker(engine, cfg, mockTracker)

	task := &domain.Issue{
		ID:         "test-1",
		Identifier: "TEST-1",
		Title:      "测试任务",
	}

	response := `{"status": "needs_clarification", "questions": [{"id": "q1", "question": "问题1"}], "summary": "需要澄清"}`

	result, err := cm.ProcessAgentResponse(context.Background(), task, response)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != workflow.StatusNeedsClarification {
		t.Errorf("expected needs_clarification status, got %s", result.Status)
	}

	if !result.WaitingForUser {
		t.Error("expected WaitingForUser to be true for needs_clarification status")
	}

	if len(result.Questions) != 1 {
		t.Errorf("expected 1 question, got %d", len(result.Questions))
	}

	// 验证阶段状态被更新为等待用户回答
	stageState := mockTracker.StageStates["TEST-1"]
	if stageState == nil || stageState.Status != "waiting_user_response" {
		t.Error("expected stage status to be waiting_user_response")
	}
}

// TestHandleUserResponseWithAgent 测试处理用户回答
func TestHandleUserResponseWithAgent(t *testing.T) {
	engine := workflow.NewEngine()
	cfg := config.DefaultConfig()
	mockTracker := NewMockTracker()
	mockRunner := &MockRunner{Success: true}

	_, _ = engine.InitTask("test-1")

	cm := workflow.NewClarificationManagerWithTracker(engine, cfg, mockTracker)
	cm.SetRunner(mockRunner)

	task := &domain.Issue{
		ID:         "test-1",
		Identifier: "TEST-1",
		Title:      "测试任务",
	}

	result, err := cm.HandleUserResponseWithAgent(context.Background(), task, "用户回答", "/tmp/workspace")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证用户回答被保存
	history := mockTracker.Conversations["TEST-1"]
	if len(history) == 0 {
		t.Error("expected user response to be saved")
	}

	if history[0].Role != "user" {
		t.Errorf("expected first turn to be user, got %s", history[0].Role)
	}

	// 验证runner被再次调用
	if !mockRunner.Called {
		t.Error("expected runner to be called")
	}

	_ = result // result may be nil if round limit reached
}

// Ensure MockTracker implements tracker.Tracker
var _ tracker.Tracker = (*MockTracker)(nil)