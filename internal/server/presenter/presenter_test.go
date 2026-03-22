package presenter_test

import (
	"net/http"
	"testing"
	"time"

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