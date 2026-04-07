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
			Title:           entry.Title,
			Stage:           entry.Stage,
			TurnCount:       entry.TurnCount,
			StartedAt:       entry.StartedAt.Format(time.RFC3339),
		}

		if entry.Issue != nil {
			r.IssueID = entry.Issue.ID
			r.State = entry.Issue.State
			if entry.Title == "" && entry.Issue.Title != "" {
				r.Title = entry.Issue.Title
			}
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

// BuildKanbanPayload 从 orchestrator 状态构建看板载荷，按阶段分列展示任务
func BuildKanbanPayload(orch *domain.OrchestratorState, now time.Time) *common.KanbanPayload {
	// 初始化各列
	columnsMap := make(map[string]*common.KanbanColumn)
	for _, stage := range common.KanbanStages {
		columnsMap[stage.ID] = &common.KanbanColumn{
			ID:        stage.ID,
			Title:     stage.Title,
			Icon:      getStageIconSVG(stage.ID),
			Color:     stage.Color,
			Tasks:     []common.KanbanTaskPayload{},
			TaskCount: 0,
		}
	}

	// 处理运行中的任务
	for _, entry := range orch.Running {
		stage := entry.Stage
		if stage == "" {
			stage = "implementation" // 默认放入实现中
		}

		task := buildKanbanTaskFromRunning(entry, now)
		// 将任务阶段映射到看板列
			kanbanCol := common.TaskStageToKanbanColumn(stage)

			if col, ok := columnsMap[kanbanCol]; ok {
			col.Tasks = append(col.Tasks, task)
			col.TaskCount++
		} else {
			// 未知阶段放入生成器中
			if col, ok := columnsMap["generator"]; ok {
				col.Tasks = append(col.Tasks, task)
				col.TaskCount++
			}
		}
	}

	// 处理重试中的任务，放入待人工处理列
	for _, entry := range orch.RetryAttempts {
		task := buildKanbanTaskFromRetry(entry)
		if col, ok := columnsMap[common.TaskStageToKanbanColumn("needs_attention")]; ok {
			col.Tasks = append(col.Tasks, task)
			col.TaskCount++
		}
	}

	// 按顺序构建列列表
	columns := make([]common.KanbanColumn, 0, len(common.KanbanStages))
	for _, stage := range common.KanbanStages {
		if col, ok := columnsMap[stage.ID]; ok {
			columns = append(columns, *col)
		}
	}

	// 计算总任务数
	totalTasks := 0
	for _, col := range columns {
		totalTasks += col.TaskCount
	}

	return &common.KanbanPayload{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Columns:     columns,
		TotalTasks:  totalTasks,
	}
}

// buildKanbanTaskFromRunning 从运行条目构建看板任务
func buildKanbanTaskFromRunning(entry *domain.RunningEntry, now time.Time) common.KanbanTaskPayload {
	task := common.KanbanTaskPayload{
		IssueIdentifier: entry.Identifier,
		Title:           entry.Title,
		Stage:           entry.Stage,
		TurnCount:       entry.TurnCount,
		StartedAt:       entry.StartedAt.Format(time.RFC3339),
		RuntimeTurns:    common.FormatRuntimeAndTurns(entry.StartedAt, entry.TurnCount, now),
	}

	if entry.Issue != nil {
		task.IssueID = entry.Issue.ID
		task.State = entry.Issue.State
		if task.Title == "" && entry.Issue.Title != "" {
			task.Title = entry.Issue.Title
		}
	}

	if entry.Session != nil {
		task.SessionID = entry.Session.SessionID
		if entry.Session.LastCodexEvent != nil {
			task.LastEvent = *entry.Session.LastCodexEvent
		}
		if entry.Session.LastCodexTimestamp != nil {
			task.LastEventAt = entry.Session.LastCodexTimestamp.Format(time.RFC3339)
		}
		task.Tokens = common.Tokens{
			InputTokens:  entry.Session.CodexInputTokens,
			OutputTokens: entry.Session.CodexOutputTokens,
			TotalTokens:  entry.Session.CodexTotalTokens,
		}
	}

	if entry.RetryAttempt != nil {
		task.Attempt = *entry.RetryAttempt
	}

	return task
}

// buildKanbanTaskFromRetry 从重试条目构建看板任务
func buildKanbanTaskFromRetry(entry *domain.RetryEntry) common.KanbanTaskPayload {
	task := common.KanbanTaskPayload{
		IssueID:         entry.IssueID,
		IssueIdentifier: entry.Identifier,
		Stage:           "needs_attention",
		Attempt:         entry.Attempt,
		DueAt:           time.UnixMilli(entry.DueAtMs).Format(time.RFC3339),
	}

	if entry.Error != nil {
		task.Error = *entry.Error
	}

	return task
}

// getStageIconSVG 获取阶段图标 SVG
func getStageIconSVG(stageID string) string {
	icons := map[string]string{
		// 看板列图标 (P-G-E 模式)
		"backlog":           `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2"/><path d="M3 9h18"/><path d="M9 21V9"/></svg>`,
		"planner":           `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M2 3h6a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H2"/><path d="M22 3h-6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h6"/><line x1="12" y1="3" x2="12" y2="21"/></svg>`,
		"generator":         `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>`,
		"evaluator":         `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>`,
		"done":              `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="16 6 9 17 4 12"/></svg>`,
		// 任务阶段图标 (用于任务详情显示)
		"clarification":     `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>`,
		"bdd_review":        `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/></svg>`,
		"architecture_review": `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="3" y1="9" x2="21" y2="9"/><line x1="9" y1="21" x2="9" y2="9"/></svg>`,
		"implementation":    `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>`,
		"verification":      `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>`,
		"completed":         `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>`,
		"needs_attention":   `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>`,
		"cancelled":         `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>`,
	}
	if icon, ok := icons[stageID]; ok {
		return icon
	}
	return icons["backlog"]
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

// BuildTasksPayload 构建任务列表载荷
func BuildTasksPayload(issues []*domain.Issue, filter string, filterLabel string) *common.TasksPayload {
	tasks := make([]common.TaskPayload, 0, len(issues))
	for _, issue := range issues {
		task := common.TaskPayload{
			ID:         issue.ID,
			Identifier: issue.Identifier,
			Title:      issue.Title,
			State:      issue.State,
			Priority:   issue.Priority,
			Labels:     issue.Labels,
			URL:        issue.URL,
		}
		if issue.Description != nil {
			task.Description = *issue.Description
		}
		if issue.CreatedAt != nil {
			task.CreatedAt = strPtr(issue.CreatedAt.Format(time.RFC3339))
		}
		if issue.UpdatedAt != nil {
			task.UpdatedAt = strPtr(issue.UpdatedAt.Format(time.RFC3339))
		}
		tasks = append(tasks, task)
	}

	return &common.TasksPayload{
		Filter:      filter,
		FilterLabel: filterLabel,
		TotalCount:  len(tasks),
		Tasks:       tasks,
	}
}

// strPtr 辅助函数：字符串转指针
func strPtr(s string) *string {
	return &s
}
