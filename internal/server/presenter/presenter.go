package presenter

import (
	"net/http"
	"time"

	"github.com/dministrator/symphony/internal/common"
	"github.com/dministrator/symphony/internal/domain"
)

// BuildStatePayload 从 orchestrator 状态构建完整的状态载荷
func BuildStatePayload(orch *domain.OrchestratorState) *common.StatePayload {
	running := make([]common.RunningEntryPayload, 0)
	for _, entry := range orch.Running {
		r := common.RunningEntryPayload{
			IssueIdentifier: entry.Identifier,
			TurnCount:       entry.TurnCount,
			StartedAt:       entry.StartedAt.Format(time.RFC3339),
		}

		if entry.Issue != nil {
			r.IssueID = entry.Issue.ID
			r.State = entry.Issue.State
		}

		if entry.Session != nil {
			r.SessionID = entry.Session.SessionID
			if entry.Session.LastCodexEvent != nil {
				r.LastEvent = *entry.Session.LastCodexEvent
			}
			if entry.Session.LastCodexTimestamp != nil {
				r.LastEventAt = entry.Session.LastCodexTimestamp.Format(time.RFC3339)
			}
			r.Tokens = common.Tokens{
				InputTokens:  entry.Session.CodexInputTokens,
				OutputTokens: entry.Session.CodexOutputTokens,
				TotalTokens:  entry.Session.CodexTotalTokens,
			}
		}

		running = append(running, r)
	}

	retrying := make([]common.RetryEntryPayload, 0)
	for _, entry := range orch.RetryAttempts {
		errMsg := ""
		if entry.Error != nil {
			errMsg = *entry.Error
		}
		retrying = append(retrying, common.RetryEntryPayload{
			IssueID:         entry.IssueID,
			IssueIdentifier: entry.Identifier,
			Attempt:         entry.Attempt,
			DueAt:           time.UnixMilli(entry.DueAtMs).Format(time.RFC3339),
			Error:           errMsg,
		})
	}

	totals := domain.CodexTotals{}
	if orch.CodexTotals != nil {
		totals = *orch.CodexTotals
	}

	return &common.StatePayload{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Counts: common.StateCounts{
			Running:  len(orch.Running),
			Retrying: len(orch.RetryAttempts),
		},
		Running:     running,
		Retrying:    retrying,
		CodexTotals: totals,
		RateLimits:  orch.CodexRateLimits,
	}
}

// BuildIssuePayload 为指定的问题标识符构建详细的载荷
// 返回 (payload, error)，如果问题不存在则返回 nil 和错误
func BuildIssuePayload(identifier string, state *domain.OrchestratorState) (map[string]any, error) {
	// 查找运行中的问题
	var foundEntry *domain.RunningEntry
	for _, entry := range state.Running {
		if entry.Identifier == identifier {
			foundEntry = entry
			break
		}
	}

	// 查找重试中的问题
	var foundRetry *domain.RetryEntry
	for _, entry := range state.RetryAttempts {
		if entry.Identifier == identifier {
			foundRetry = entry
			break
		}
	}

	if foundEntry == nil && foundRetry == nil {
		return nil, http.ErrNoCookie
	}

	response := map[string]any{
		"issue_identifier": identifier,
		"status":          "unknown",
	}

	if foundEntry != nil {
		response["issue_id"] = foundEntry.Issue.ID
		response["status"] = "running"

		if foundEntry.Issue != nil {
			response["workspace"] = map[string]string{
				"path": "/tmp/symphony_workspaces/" + identifier,
			}
		}

		running := map[string]any{
			"session_id": "",
			"turn_count": foundEntry.TurnCount,
			"started_at": foundEntry.StartedAt.Format(time.RFC3339),
		}

		if foundEntry.Issue != nil {
			running["state"] = foundEntry.Issue.State
		}

		if foundEntry.Session != nil {
			running["session_id"] = foundEntry.Session.SessionID
			if foundEntry.Session.LastCodexEvent != nil {
				running["last_event"] = *foundEntry.Session.LastCodexEvent
			}
			if foundEntry.Session.LastCodexTimestamp != nil {
				running["last_event_at"] = foundEntry.Session.LastCodexTimestamp.Format(time.RFC3339)
			}
			running["tokens"] = map[string]int64{
				"input_tokens":  foundEntry.Session.CodexInputTokens,
				"output_tokens": foundEntry.Session.CodexOutputTokens,
				"total_tokens":  foundEntry.Session.CodexTotalTokens,
			}
		}

		response["running"] = running
	}

	if foundRetry != nil {
		errMsg := ""
		if foundRetry.Error != nil {
			errMsg = *foundRetry.Error
		}
		response["retry"] = map[string]any{
			"attempt": foundRetry.Attempt,
			"due_at":  time.UnixMilli(foundRetry.DueAtMs).Format(time.RFC3339),
			"error":   errMsg,
		}
		if response["status"] == "unknown" {
			response["status"] = "retrying"
		}
	}

	return response, nil
}

// BuildRefreshPayload 构建刷新响应载荷
func BuildRefreshPayload() map[string]any {
	return map[string]any{
		"queued":       true,
		"coalesced":    false,
		"requested_at": time.Now().UTC().Format(time.RFC3339),
		"operations":   []string{"poll", "reconcile"},
	}
}
