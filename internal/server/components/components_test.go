package components_test

import (
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