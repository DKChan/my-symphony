package presenter_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/dministrator/symphony/internal/common"
	"github.com/dministrator/symphony/internal/domain"
	"github.com/dministrator/symphony/internal/server/presenter"
	"github.com/stretchr/testify/assert"
)

func TestBuildStatePayload(t *testing.T) {
	state := &domain.OrchestratorState{
		PollIntervalMs:      5000,
		MaxConcurrentAgents: 3,
		Running:             make(map[string]*domain.RunningEntry),
		RetryAttempts:       make(map[string]*domain.RetryEntry),
		Completed:           make(map[string]struct{}),
		CodexTotals:         &domain.CodexTotals{},
	}

	payload := presenter.BuildStatePayload(state)

	assert.NotNil(t, payload)
	assert.NotEmpty(t, payload.GeneratedAt)
	assert.Equal(t, 0, payload.Counts.Running)
	assert.Equal(t, 0, payload.Counts.Retrying)
	assert.NotNil(t, payload.Running)
	assert.NotNil(t, payload.Retrying)
}

func TestBuildStatePayload_WithRunningEntry(t *testing.T) {
	now := time.Now()
	state := &domain.OrchestratorState{
		PollIntervalMs:      5000,
		MaxConcurrentAgents: 3,
		Running: map[string]*domain.RunningEntry{
			"TEST-1": {
				Identifier: "TEST-1",
				StartedAt: now,
				TurnCount: 5,
				Issue: &domain.Issue{
					ID:    "1",
					State: "In Progress",
				},
			},
		},
		RetryAttempts: make(map[string]*domain.RetryEntry),
		Completed:     make(map[string]struct{}),
		CodexTotals:   &domain.CodexTotals{},
	}

	payload := presenter.BuildStatePayload(state)

	assert.Equal(t, 1, payload.Counts.Running)
	assert.Len(t, payload.Running, 1)
	assert.Equal(t, "TEST-1", payload.Running[0].IssueIdentifier)
	assert.Equal(t, "1", payload.Running[0].IssueID)
	assert.Equal(t, "In Progress", payload.Running[0].State)
}

func TestBuildIssuePayload_NotFound(t *testing.T) {
	state := &domain.OrchestratorState{
		Running:      make(map[string]*domain.RunningEntry),
		RetryAttempts: make(map[string]*domain.RetryEntry),
	}

	_, err := presenter.BuildIssuePayload("NONEXISTENT", state)

	assert.Error(t, err)
	assert.ErrorIs(t, err, http.ErrNoCookie)
}

func TestBuildIssuePayload_FoundInRunning(t *testing.T) {
	now := time.Now()
	state := &domain.OrchestratorState{
		Running: map[string]*domain.RunningEntry{
			"TEST-1": {
				Identifier: "TEST-1",
				StartedAt:  now,
				TurnCount:  3,
				Issue: &domain.Issue{
					ID:    "issue-123",
					State: "In Progress",
				},
			},
		},
		RetryAttempts: make(map[string]*domain.RetryEntry),
	}

	payload, err := presenter.BuildIssuePayload("TEST-1", state)

	assert.NoError(t, err)
	assert.NotNil(t, payload)
	assert.Equal(t, "TEST-1", payload["issue_identifier"])
	assert.Equal(t, "running", payload["status"])
	assert.Equal(t, "issue-123", payload["issue_id"])

	running := payload["running"].(map[string]any)
	assert.Equal(t, 3, running["turn_count"])
}

func TestBuildIssuePayload_FoundInRetry(t *testing.T) {
	now := time.Now()
	state := &domain.OrchestratorState{
		Running: make(map[string]*domain.RunningEntry),
		RetryAttempts: map[string]*domain.RetryEntry{
			"TEST-2": {
				IssueID:     "issue-456",
				Identifier:  "TEST-2",
				Attempt:     2,
				DueAtMs:     now.Add(5 * time.Minute).UnixMilli(),
			},
		},
	}

	payload, err := presenter.BuildIssuePayload("TEST-2", state)

	assert.NoError(t, err)
	assert.NotNil(t, payload)
	assert.Equal(t, "TEST-2", payload["issue_identifier"])
	assert.Equal(t, "retrying", payload["status"])

	retry := payload["retry"].(map[string]any)
	assert.Equal(t, 2, retry["attempt"])
}

func TestBuildRefreshPayload(t *testing.T) {
	payload := presenter.BuildRefreshPayload()

	assert.NotNil(t, payload)
	assert.Equal(t, true, payload["queued"])
	assert.Equal(t, false, payload["coalesced"])
	assert.Contains(t, payload, "requested_at")
	assert.Contains(t, payload, "operations")
}

func TestBuildKanbanPayload_Empty(t *testing.T) {
	now := time.Now()
	state := &domain.OrchestratorState{
		PollIntervalMs:      5000,
		MaxConcurrentAgents: 3,
		Running:             make(map[string]*domain.RunningEntry),
		RetryAttempts:       make(map[string]*domain.RetryEntry),
		Completed:           make(map[string]struct{}),
		CodexTotals:         &domain.CodexTotals{},
	}

	payload := presenter.BuildKanbanPayload(state, now)

	assert.NotNil(t, payload)
	assert.NotEmpty(t, payload.GeneratedAt)
	assert.Equal(t, 9, len(payload.Columns)) // 9 个阶段列
	assert.Equal(t, 0, payload.TotalTasks)

	// 检查所有列都是空的
	for _, col := range payload.Columns {
		assert.Equal(t, 0, col.TaskCount)
		assert.Len(t, col.Tasks, 0)
	}
}

func TestBuildKanbanPayload_WithRunningEntries(t *testing.T) {
	now := time.Now()
	state := &domain.OrchestratorState{
		PollIntervalMs:      5000,
		MaxConcurrentAgents: 3,
		Running: map[string]*domain.RunningEntry{
			"TEST-1": {
				Identifier: "TEST-1",
				Title:      "Implement Feature",
				Stage:      "implementation",
				StartedAt:  now.Add(-5 * time.Minute),
				TurnCount:  10,
				Issue: &domain.Issue{
					ID:    "1",
					State: "In Progress",
					Title: "Implement Feature",
				},
				Session: &domain.LiveSession{
					SessionID:         "sess-abc123",
					CodexInputTokens:  1000,
					CodexOutputTokens: 500,
					CodexTotalTokens:  1500,
				},
			},
			"TEST-2": {
				Identifier: "TEST-2",
				Title:      "Clarify Requirements",
				Stage:      "clarification",
				StartedAt:  now.Add(-2 * time.Minute),
				TurnCount:  3,
				Issue: &domain.Issue{
					ID:    "2",
					State: "Active",
				},
			},
		},
		RetryAttempts: make(map[string]*domain.RetryEntry),
		Completed:     make(map[string]struct{}),
		CodexTotals:   &domain.CodexTotals{},
	}

	payload := presenter.BuildKanbanPayload(state, now)

	assert.NotNil(t, payload)
	assert.Equal(t, 2, payload.TotalTasks)

	// 检查 implementation 列有 1 个任务
	var implCol *common.KanbanColumn
	for i, col := range payload.Columns {
		if col.ID == "implementation" {
			implCol = &payload.Columns[i]
			break
		}
	}
	assert.NotNil(t, implCol)
	assert.Equal(t, 1, implCol.TaskCount)
	assert.Len(t, implCol.Tasks, 1)
	assert.Equal(t, "TEST-1", implCol.Tasks[0].IssueIdentifier)
	assert.Equal(t, "Implement Feature", implCol.Tasks[0].Title)
	assert.Equal(t, "implementation", implCol.Tasks[0].Stage)

	// 检查 clarification 列有 1 个任务
	var clarCol *common.KanbanColumn
	for i, col := range payload.Columns {
		if col.ID == "clarification" {
			clarCol = &payload.Columns[i]
			break
		}
	}
	assert.NotNil(t, clarCol)
	assert.Equal(t, 1, clarCol.TaskCount)
	assert.Equal(t, "TEST-2", clarCol.Tasks[0].IssueIdentifier)
	assert.Equal(t, "clarification", clarCol.Tasks[0].Stage)
}

func TestBuildKanbanPayload_WithRetryEntries(t *testing.T) {
	now := time.Now()
	errMsg := "Connection timeout"
	state := &domain.OrchestratorState{
		PollIntervalMs:      5000,
		MaxConcurrentAgents: 3,
		Running:             make(map[string]*domain.RunningEntry),
		RetryAttempts: map[string]*domain.RetryEntry{
			"TEST-3": {
				IssueID:     "3",
				Identifier:  "TEST-3",
				Attempt:     2,
				DueAtMs:     now.Add(5 * time.Minute).UnixMilli(),
				Error:       &errMsg,
			},
		},
		Completed:   make(map[string]struct{}),
		CodexTotals: &domain.CodexTotals{},
	}

	payload := presenter.BuildKanbanPayload(state, now)

	assert.NotNil(t, payload)
	assert.Equal(t, 1, payload.TotalTasks)

	// 重试任务应该放在 needs_attention 列
	var needsCol *common.KanbanColumn
	for i, col := range payload.Columns {
		if col.ID == "needs_attention" {
			needsCol = &payload.Columns[i]
			break
		}
	}
	assert.NotNil(t, needsCol)
	assert.Equal(t, 1, needsCol.TaskCount)
	assert.Equal(t, "TEST-3", needsCol.Tasks[0].IssueIdentifier)
	assert.Equal(t, "needs_attention", needsCol.Tasks[0].Stage)
	assert.Equal(t, 2, needsCol.Tasks[0].Attempt)
	assert.Equal(t, errMsg, needsCol.Tasks[0].Error)
}

func TestBuildKanbanPayload_UnknownStage(t *testing.T) {
	now := time.Now()
	state := &domain.OrchestratorState{
		PollIntervalMs:      5000,
		MaxConcurrentAgents: 3,
		Running: map[string]*domain.RunningEntry{
			"TEST-4": {
				Identifier: "TEST-4",
				Stage:      "unknown_stage", // 未知的阶段
				StartedAt:  now,
				Issue: &domain.Issue{
					ID:    "4",
					State: "Active",
				},
			},
		},
		RetryAttempts: make(map[string]*domain.RetryEntry),
		Completed:     make(map[string]struct{}),
		CodexTotals:   &domain.CodexTotals{},
	}

	payload := presenter.BuildKanbanPayload(state, now)

	assert.NotNil(t, payload)
	assert.Equal(t, 1, payload.TotalTasks)

	// 未知阶段应该放入 implementation 列
	var implCol *common.KanbanColumn
	for i, col := range payload.Columns {
		if col.ID == "implementation" {
			implCol = &payload.Columns[i]
			break
		}
	}
	assert.NotNil(t, implCol)
	assert.Equal(t, 1, implCol.TaskCount)
}

func TestBuildKanbanPayload_EmptyStage(t *testing.T) {
	now := time.Now()
	state := &domain.OrchestratorState{
		PollIntervalMs:      5000,
		MaxConcurrentAgents: 3,
		Running: map[string]*domain.RunningEntry{
			"TEST-5": {
				Identifier: "TEST-5",
				Stage:      "", // 空阶段
				StartedAt:  now,
				Issue: &domain.Issue{
					ID:    "5",
					State: "Active",
				},
			},
		},
		RetryAttempts: make(map[string]*domain.RetryEntry),
		Completed:     make(map[string]struct{}),
		CodexTotals:   &domain.CodexTotals{},
	}

	payload := presenter.BuildKanbanPayload(state, now)

	assert.NotNil(t, payload)
	assert.Equal(t, 1, payload.TotalTasks)

	// 空阶段默认放入 implementation 列
	var implCol *common.KanbanColumn
	for i, col := range payload.Columns {
		if col.ID == "implementation" {
			implCol = &payload.Columns[i]
			break
		}
	}
	assert.NotNil(t, implCol)
	assert.Equal(t, 1, implCol.TaskCount)
}