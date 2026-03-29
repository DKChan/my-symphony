// Package orchestrator 提供任务状态恢复功能测试
package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRecoveryTracker 模拟 Tracker 用于恢复测试
type mockRecoveryTracker struct {
	issues     map[string]*domain.Issue
	stageState map[string]*domain.StageState
}

func newMockRecoveryTracker() *mockRecoveryTracker {
	return &mockRecoveryTracker{
		issues:     make(map[string]*domain.Issue),
		stageState: make(map[string]*domain.StageState),
	}
}

func (m *mockRecoveryTracker) FetchCandidateIssues(ctx context.Context, activeStates []string) ([]*domain.Issue, error) {
	var result []*domain.Issue
	for _, issue := range m.issues {
		for _, state := range activeStates {
			if issue.State == state {
				result = append(result, issue)
				break
			}
		}
	}
	return result, nil
}

func (m *mockRecoveryTracker) FetchIssuesByStates(ctx context.Context, states []string) ([]*domain.Issue, error) {
	return m.FetchCandidateIssues(ctx, states)
}

func (m *mockRecoveryTracker) FetchIssueStatesByIDs(ctx context.Context, ids []string) ([]*domain.Issue, error) {
	var result []*domain.Issue
	for _, id := range ids {
		if issue, ok := m.issues[id]; ok {
			result = append(result, issue)
		}
	}
	return result, nil
}

func (m *mockRecoveryTracker) GetTask(ctx context.Context, identifier string) (*domain.Issue, error) {
	for _, issue := range m.issues {
		if issue.Identifier == identifier {
			return issue, nil
		}
	}
	return nil, assert.AnError
}

func (m *mockRecoveryTracker) CheckAvailability() error {
	return nil
}

func (m *mockRecoveryTracker) CreateTask(ctx context.Context, title, description string) (*domain.Issue, error) {
	return nil, nil
}

func (m *mockRecoveryTracker) CreateSubTask(ctx context.Context, parentIdentifier string, title, description string, blockedBy []string) (*domain.Issue, error) {
	return nil, nil
}

func (m *mockRecoveryTracker) UpdateStage(ctx context.Context, identifier string, stage domain.StageState) error {
	m.stageState[identifier] = &stage
	return nil
}

func (m *mockRecoveryTracker) GetStageState(ctx context.Context, identifier string) (*domain.StageState, error) {
	return m.stageState[identifier], nil
}

func (m *mockRecoveryTracker) AppendConversation(ctx context.Context, identifier string, turn domain.ConversationTurn) error {
	return nil
}

func (m *mockRecoveryTracker) GetConversationHistory(ctx context.Context, identifier string) ([]domain.ConversationTurn, error) {
	return nil, nil
}

func (m *mockRecoveryTracker) ListTasksByState(ctx context.Context, states []string) ([]*domain.Issue, error) {
	return m.FetchIssuesByStates(ctx, states)
}

func (m *mockRecoveryTracker) addIssue(issue *domain.Issue) {
	m.issues[issue.ID] = issue
}

func (m *mockRecoveryTracker) setStageState(identifier string, state *domain.StageState) {
	m.stageState[identifier] = state
}

// TestNewRecoveryManager 测试创建恢复管理器
func TestNewRecoveryManager(t *testing.T) {
	cfg := &config.Config{
		Execution: config.ExecutionConfig{
			MaxRetries: 5,
		},
	}
	tracker := newMockRecoveryTracker()

	mgr := NewRecoveryManager(cfg, tracker, nil)
	assert.NotNil(t, mgr)
	assert.Equal(t, 5, mgr.maxRetries)
}

// TestNewRecoveryManager_DefaultMaxRetries 测试默认最大重试次数
func TestNewRecoveryManager_DefaultMaxRetries(t *testing.T) {
	cfg := &config.Config{
		Execution: config.ExecutionConfig{
			MaxRetries: 0,
		},
	}
	tracker := newMockRecoveryTracker()

	mgr := NewRecoveryManager(cfg, tracker, nil)
	assert.NotNil(t, mgr)
	assert.Equal(t, 3, mgr.maxRetries)
}

// TestDetermineRecoveryAction 测试决定恢复动作
func TestDetermineRecoveryAction(t *testing.T) {
	cfg := &config.Config{
		Execution: config.ExecutionConfig{
			MaxRetries: 3,
		},
	}
	tracker := newMockRecoveryTracker()
	mgr := NewRecoveryManager(cfg, tracker, nil)

	tests := []struct {
		name     string
		state    domain.StageState
		expected domain.RecoveryAction
	}{
		{
			name: "in_progress",
			state: domain.StageState{
				Status: "in_progress",
			},
			expected: domain.ActionContinue,
		},
		{
			name: "pending",
			state: domain.StageState{
				Status: "pending",
			},
			expected: domain.ActionStart,
		},
		{
			name: "waiting_review",
			state: domain.StageState{
				Status: "waiting_review",
			},
			expected: domain.ActionWaitForReview,
		},
		{
			name: "failed_under_max",
			state: domain.StageState{
				Status:     "failed",
				RetryCount: 2,
			},
			expected: domain.ActionRetry,
		},
		{
			name: "failed_at_max",
			state: domain.StageState{
				Status:     "failed",
				RetryCount: 3,
			},
			expected: domain.ActionSkip,
		},
		{
			name: "failed_over_max",
			state: domain.StageState{
				Status:     "failed",
				RetryCount: 5,
			},
			expected: domain.ActionSkip,
		},
		{
			name: "completed",
			state: domain.StageState{
				Status: "completed",
			},
			expected: domain.ActionSkip,
		},
		{
			name: "unknown",
			state: domain.StageState{
				Status: "unknown_status",
			},
			expected: domain.ActionUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := mgr.determineRecoveryAction(tt.state)
			assert.Equal(t, tt.expected, action)
		})
	}
}

// TestRestoreAll_NoActiveTasks 测试没有活跃任务
func TestRestoreAll_NoActiveTasks(t *testing.T) {
	cfg := &config.Config{
		Tracker: config.TrackerConfig{
			ActiveStates: []string{"Todo", "In Progress"},
		},
	}
	tracker := newMockRecoveryTracker()
	mgr := NewRecoveryManager(cfg, tracker, nil)

	tasks, err := mgr.RestoreAll(context.Background())
	assert.NoError(t, err)
	assert.Nil(t, tasks)
}

// TestRestoreAll_WithActiveTasks 测试有活跃任务
func TestRestoreAll_WithActiveTasks(t *testing.T) {
	cfg := &config.Config{
		Tracker: config.TrackerConfig{
			ActiveStates: []string{"Todo", "In Progress"},
		},
		Execution: config.ExecutionConfig{
			MaxRetries: 3,
		},
	}
	tracker := newMockRecoveryTracker()

	// 添加活跃任务
	tracker.addIssue(&domain.Issue{
		ID:         "1",
		Identifier: "TEST-1",
		Title:      "Active Task 1",
		State:      "In Progress",
	})
	tracker.setStageState("TEST-1", &domain.StageState{
		Name:      "clarification",
		Status:    "in_progress",
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	tracker.addIssue(&domain.Issue{
		ID:         "2",
		Identifier: "TEST-2",
		Title:      "Active Task 2",
		State:      "Todo",
	})
	// TEST-2 没有阶段状态

	mgr := NewRecoveryManager(cfg, tracker, nil)

	tasks, err := mgr.RestoreAll(context.Background())
	assert.NoError(t, err)
	assert.Len(t, tasks, 2)

	// 构建任务映射，以便按标识符查找
	taskMap := make(map[string]*domain.RecoveredTask)
	for _, task := range tasks {
		taskMap[task.Issue.Identifier] = task
	}

	// 验证 TEST-1 的恢复动作
	task1, ok := taskMap["TEST-1"]
	require.True(t, ok)
	assert.Equal(t, domain.ActionContinue, task1.Action)
	require.NotNil(t, task1.StageState)
	assert.Equal(t, "clarification", task1.StageState.Name)

	// 验证 TEST-2 的恢复动作（没有阶段状态，应该是 start）
	task2, ok := taskMap["TEST-2"]
	require.True(t, ok)
	assert.Equal(t, domain.ActionStart, task2.Action)
}

// TestRestoreFromBeads 测试从 Beads 恢复
func TestRestoreFromBeads(t *testing.T) {
	cfg := &config.Config{
		Tracker: config.TrackerConfig{
			ActiveStates: []string{"In Progress"},
		},
		Execution: config.ExecutionConfig{
			MaxRetries: 3,
		},
	}
	tracker := newMockRecoveryTracker()

	tracker.addIssue(&domain.Issue{
		ID:         "1",
		Identifier: "TEST-1",
		Title:      "Test Task",
		State:      "In Progress",
	})
	tracker.setStageState("TEST-1", &domain.StageState{
		Name:      "implementation",
		Status:    "waiting_review",
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	mgr := NewRecoveryManager(cfg, tracker, nil)

	task, err := mgr.RestoreFromBeads(context.Background(), "TEST-1")
	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, "TEST-1", task.Issue.Identifier)
	assert.Equal(t, domain.ActionWaitForReview, task.Action)
	assert.Equal(t, "implementation", task.StageState.Name)
}

// TestRestoreFromBeads_NonActiveState 测试非活跃状态的任务
func TestRestoreFromBeads_NonActiveState(t *testing.T) {
	cfg := &config.Config{
		Tracker: config.TrackerConfig{
			ActiveStates:   []string{"Todo", "In Progress"},
			TerminalStates: []string{"Done", "Cancelled"},
		},
	}
	tracker := newMockRecoveryTracker()

	tracker.addIssue(&domain.Issue{
		ID:         "1",
		Identifier: "TEST-1",
		Title:      "Done Task",
		State:      "Done",
	})

	mgr := NewRecoveryManager(cfg, tracker, nil)

	task, err := mgr.RestoreFromBeads(context.Background(), "TEST-1")
	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, domain.ActionSkip, task.Action)
}

// TestExecuteRecovery_Retry 测试重试恢复
func TestExecuteRecovery_Retry(t *testing.T) {
	cfg := &config.Config{
		Execution: config.ExecutionConfig{
			MaxRetries: 3,
		},
	}
	tracker := newMockRecoveryTracker()

	tracker.addIssue(&domain.Issue{
		ID:         "1",
		Identifier: "TEST-1",
		Title:      "Failed Task",
		State:      "In Progress",
	})

	mgr := NewRecoveryManager(cfg, tracker, nil)

	task := &domain.RecoveredTask{
		Issue: &domain.Issue{
			Identifier: "TEST-1",
		},
		StageState: &domain.StageState{
			Name:       "clarification",
			Status:     "failed",
			RetryCount: 1,
		},
		Action: domain.ActionRetry,
	}

	err := mgr.ExecuteRecovery(context.Background(), task)
	assert.NoError(t, err)

	// 验证重试次数增加
	assert.Equal(t, 2, task.StageState.RetryCount)
	assert.Equal(t, "pending", task.StageState.Status)

	// 验证状态已更新到 tracker
	savedState := tracker.stageState["TEST-1"]
	require.NotNil(t, savedState)
	assert.Equal(t, 2, savedState.RetryCount)
}

// TestValidateStageState 测试验证阶段状态
func TestValidateStageState(t *testing.T) {
	tests := []struct {
		name    string
		state   domain.StageState
		wantErr bool
	}{
		{
			name: "valid pending",
			state: domain.StageState{
				Name:   "clarification",
				Status: "pending",
			},
			wantErr: false,
		},
		{
			name: "valid in_progress",
			state: domain.StageState{
				Name:   "implementation",
				Status: "in_progress",
			},
			wantErr: false,
		},
		{
			name: "valid completed",
			state: domain.StageState{
				Name:   "verification",
				Status: "completed",
			},
			wantErr: false,
		},
		{
			name: "valid failed",
			state: domain.StageState{
				Name:   "clarification",
				Status: "failed",
			},
			wantErr: false,
		},
		{
			name: "valid waiting_review",
			state: domain.StageState{
				Name:   "bdd_review",
				Status: "waiting_review",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			state: domain.StageState{
				Status: "pending",
			},
			wantErr: true,
		},
		{
			name: "missing status",
			state: domain.StageState{
				Name: "clarification",
			},
			wantErr: true,
		},
		{
			name: "invalid status",
			state: domain.StageState{
				Name:   "clarification",
				Status: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStageState(tt.state)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSerializeDeserializeStageState 测试序列化和反序列化
func TestSerializeDeserializeStageState(t *testing.T) {
	original := domain.StageState{
		Name:       "clarification",
		Status:     "in_progress",
		StartedAt:  time.Date(2026, 3, 30, 10, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2026, 3, 30, 10, 5, 0, 0, time.UTC),
		Round:      2,
		RetryCount: 1,
		Error:      "previous error",
	}

	// 序列化
	data, err := SerializeStageState(original)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// 反序列化
	recovered, err := DeserializeStageState(data)
	require.NoError(t, err)
	assert.Equal(t, original.Name, recovered.Name)
	assert.Equal(t, original.Status, recovered.Status)
	assert.Equal(t, original.Round, recovered.Round)
	assert.Equal(t, original.RetryCount, recovered.RetryCount)
	assert.Equal(t, original.Error, recovered.Error)
}

// TestGetRecoveryStats 测试获取恢复统计
func TestGetRecoveryStats(t *testing.T) {
	cfg := &config.Config{
		Execution: config.ExecutionConfig{
			MaxRetries: 3,
		},
	}
	tracker := newMockRecoveryTracker()
	mgr := NewRecoveryManager(cfg, tracker, nil)

	// 添加一些恢复的任务
	mgr.recoveredTasks["1"] = &domain.RecoveredTask{Action: domain.ActionContinue}
	mgr.recoveredTasks["2"] = &domain.RecoveredTask{Action: domain.ActionStart}
	mgr.recoveredTasks["3"] = &domain.RecoveredTask{Action: domain.ActionStart}
	mgr.recoveredTasks["4"] = &domain.RecoveredTask{Action: domain.ActionWaitForReview}
	mgr.recoveredTasks["5"] = &domain.RecoveredTask{Action: domain.ActionRetry}
	mgr.recoveredTasks["6"] = &domain.RecoveredTask{Action: domain.ActionSkip}

	stats := mgr.GetRecoveryStats()
	assert.Equal(t, 6, stats.TotalRecovered)
	assert.Equal(t, 1, stats.ContinueCount)
	assert.Equal(t, 2, stats.StartCount)
	assert.Equal(t, 1, stats.WaitReviewCount)
	assert.Equal(t, 1, stats.RetryCount)
	assert.Equal(t, 1, stats.SkipCount)
}