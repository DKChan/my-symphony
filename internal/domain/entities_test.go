// Package domain_test 测试领域模型
package domain_test

import (
	"testing"
	"time"

	"github.com/dministrator/symphony/internal/domain"
)

func TestIssueFields(t *testing.T) {
	now := time.Now()
	desc := "Test description"
	priority := 1
	branch := "feature/test"
	url := "https://github.com/owner/repo/issues/1"

	issue := &domain.Issue{
		ID:          "abc123",
		Identifier:  "TEST-1",
		Title:       "Test Issue",
		Description: &desc,
		Priority:    &priority,
		State:       "Todo",
		BranchName:  &branch,
		URL:         &url,
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

func TestWorkflowDefinition(t *testing.T) {
	def := &domain.WorkflowDefinition{
		Config: map[string]any{
			"tracker": map[string]any{
				"kind": "github",
			},
		},
		PromptTemplate: "# Task\n\nPlease work on this.",
	}

	if def.Config == nil {
		t.Error("expected non-nil config")
	}

	if def.PromptTemplate == "" {
		t.Error("expected non-empty prompt template")
	}
}

func TestWorkspaceFields(t *testing.T) {
	ws := &domain.Workspace{
		Path:         "/tmp/workspaces/TEST-1",
		WorkspaceKey: "TEST-1",
		CreatedNow:   true,
	}

	if ws.Path != "/tmp/workspaces/TEST-1" {
		t.Errorf("unexpected path: %s", ws.Path)
	}

	if !ws.CreatedNow {
		t.Error("expected CreatedNow to be true")
	}
}

func TestRunAttempt(t *testing.T) {
	attempt := &domain.RunAttempt{
		IssueID:         "abc123",
		IssueIdentifier: "TEST-1",
		Attempt:         intPtr(2),
		WorkspacePath:   "/tmp/workspaces/TEST-1",
		StartedAt:       time.Now(),
		Status:          domain.StatusStreamingTurn,
	}

	if attempt.IssueID != "abc123" {
		t.Errorf("expected IssueID 'abc123', got %s", attempt.IssueID)
	}

	if attempt.Attempt == nil || *attempt.Attempt != 2 {
		t.Errorf("expected Attempt 2, got %v", attempt.Attempt)
	}

	if attempt.Status != domain.StatusStreamingTurn {
		t.Errorf("expected Status 'streaming_turn', got %s", attempt.Status)
	}
}

func TestRunStatusConstants(t *testing.T) {
	statuses := []domain.RunStatus{
		domain.StatusPreparingWorkspace,
		domain.StatusBuildingPrompt,
		domain.StatusLaunchingAgentProcess,
		domain.StatusInitializingSession,
		domain.StatusStreamingTurn,
		domain.StatusFinishing,
		domain.StatusSucceeded,
		domain.StatusFailed,
		domain.StatusTimedOut,
		domain.StatusStalled,
		domain.StatusCanceledByReconcile,
	}

	for _, status := range statuses {
		if status == "" {
			t.Error("status should not be empty")
		}
	}
}

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

func TestRetryEntry(t *testing.T) {
	errMsg := "connection timeout"
	entry := &domain.RetryEntry{
		IssueID:    "abc123",
		Identifier: "TEST-1",
		Attempt:    3,
		DueAtMs:    time.Now().Add(30 * time.Second).UnixMilli(),
		Error:      &errMsg,
	}

	if entry.Attempt != 3 {
		t.Errorf("expected Attempt 3, got %d", entry.Attempt)
	}

	if entry.Error == nil || *entry.Error != "connection timeout" {
		t.Errorf("unexpected error: %v", entry.Error)
	}
}

func TestOrchestratorState(t *testing.T) {
	state := &domain.OrchestratorState{
		PollIntervalMs:        30000,
		MaxConcurrentAgents:   10,
		Running:               make(map[string]*domain.RunningEntry),
		Claimed:               make(map[string]struct{}),
		RetryAttempts:         make(map[string]*domain.RetryEntry),
		Completed:             make(map[string]struct{}),
		CodexTotals:          &domain.CodexTotals{},
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

	if totals.SecondsRunning != 3600.5 {
		t.Errorf("expected 3600.5 seconds, got %f", totals.SecondsRunning)
	}
}

// 辅助函数
func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}