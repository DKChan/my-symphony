package components_test

import (
	"testing"
	"time"

	"github.com/dministrator/symphony/internal/domain"
	"github.com/dministrator/symphony/internal/server/components"
	"github.com/stretchr/testify/assert"
)

func TestRenderRunningSessions_Empty(t *testing.T) {
	state := &domain.OrchestratorState{
		Running: make(map[string]*domain.RunningEntry),
	}
	now := time.Now()

	html := components.RenderRunningSessions(state, now)

	assert.Contains(t, html, "暂无活跃 Session")
}

func TestRenderRunningSessions_WithEntry(t *testing.T) {
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
	}

	html := components.RenderRunningSessions(state, now)

	assert.Contains(t, html, "TEST-1")
	assert.Contains(t, html, "In Progress")
	assert.Contains(t, html, "sess-abc123")
	assert.Contains(t, html, "data-table-running")
}

func TestRenderRetryQueue_Empty(t *testing.T) {
	state := &domain.OrchestratorState{
		RetryAttempts: make(map[string]*domain.RetryEntry),
	}

	html := components.RenderRetryQueue(state)

	assert.Contains(t, html, "当前没有 Issues 在等待 Retry")
}