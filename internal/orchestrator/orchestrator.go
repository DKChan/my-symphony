// Package orchestrator 提供核心编排功能
package orchestrator

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dministrator/symphony/internal/agent"
	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/domain"
	"github.com/dministrator/symphony/internal/tracker"
	"github.com/dministrator/symphony/internal/workspace"
)

// Orchestrator 核心编排器
type Orchestrator struct {
	cfg            *config.Config
	trackerClient  tracker.Tracker
	workspaceMgr   *workspace.Manager
	agentRunner    agent.Runner
	promptTemplate string

	// 运行时状态
	mu           sync.RWMutex
	state        *domain.OrchestratorState
	retryTimers  map[string]*time.Timer
	startTime    time.Time
	endedRuntime float64 // 已结束会话的累计运行时间

	// 事件回调
	onStateChange func()
}

// New 创建新的编排器
func New(cfg *config.Config, promptTemplate string) *Orchestrator {
	return &Orchestrator{
		cfg:            cfg,
		trackerClient:  tracker.NewTracker(cfg),
		workspaceMgr:   workspace.NewManager(cfg),
		agentRunner:    agent.NewRunner(cfg),
		promptTemplate: promptTemplate,
		retryTimers:    make(map[string]*time.Timer),
		state: &domain.OrchestratorState{
			PollIntervalMs:      cfg.Polling.IntervalMs,
			MaxConcurrentAgents: cfg.Agent.MaxConcurrentAgents,
			Running:             make(map[string]*domain.RunningEntry),
			Claimed:             make(map[string]struct{}),
			RetryAttempts:       make(map[string]*domain.RetryEntry),
			Completed:           make(map[string]struct{}),
			CodexTotals:         &domain.CodexTotals{},
			CodexRateLimits:     nil,
		},
	}
}

// SetOnStateChange 设置状态变更回调
func (o *Orchestrator) SetOnStateChange(callback func()) {
	o.onStateChange = callback
}

// Run 运行编排器
func (o *Orchestrator) Run(ctx context.Context) error {
	o.startTime = time.Now()

	// 启动时清理终态工作空间
	if err := o.startupCleanup(ctx); err != nil {
		fmt.Printf("startup cleanup warning: %v\n", err)
	}

	// 立即执行一次tick
	o.tick(ctx)

	// 启动轮询
	ticker := time.NewTicker(time.Duration(o.cfg.Polling.IntervalMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			o.tick(ctx)
		}
	}
}

// tick 执行一次轮询
func (o *Orchestrator) tick(ctx context.Context) {
	// 1. 协调运行中的问题
	o.reconcile(ctx)

	// 2. 验证配置
	validation := o.cfg.ValidateDispatchConfig()
	if !validation.Valid {
		fmt.Printf("dispatch validation failed: %v\n", validation.Errors)
		return
	}

	// 3. 获取候选问题
	issues, err := o.trackerClient.FetchCandidateIssues(ctx, o.cfg.Tracker.ActiveStates)
	if err != nil {
		fmt.Printf("failed to fetch candidate issues: %v\n", err)
		return
	}

	// 4. 排序
	sorted := o.sortForDispatch(issues)

	// 5. 调度
	for _, issue := range sorted {
		if !o.hasAvailableSlots() {
			break
		}

		if o.shouldDispatch(issue) {
			o.dispatch(ctx, issue, nil)
		}
	}

	// 通知观察者
	if o.onStateChange != nil {
		o.onStateChange()
	}
}

// reconcile 协调运行中的问题
func (o *Orchestrator) reconcile(ctx context.Context) {
	o.mu.Lock()
	defer o.mu.Unlock()

	// 检查停滞
	o.checkStalled()

	// 获取运行中问题的ID
	if len(o.state.Running) == 0 {
		return
	}

	runningIDs := make([]string, 0, len(o.state.Running))
	for id := range o.state.Running {
		runningIDs = append(runningIDs, id)
	}

	// 刷新状态
	refreshed, err := o.trackerClient.FetchIssueStatesByIDs(ctx, runningIDs)
	if err != nil {
		fmt.Printf("failed to refresh issue states: %v\n", err)
		return
	}

	// 构建ID到问题的映射
	issueMap := make(map[string]*domain.Issue)
	for _, issue := range refreshed {
		issueMap[issue.ID] = issue
	}

	// 检查每个运行中的问题
	for id, entry := range o.state.Running {
		issue, ok := issueMap[id]
		if !ok {
			continue
		}

		if o.cfg.IsTerminalState(issue.State) {
			// 终态：终止并清理工作空间
			go o.terminateAndCleanup(id, entry)
		} else if !o.cfg.IsActiveState(issue.State) {
			// 非活跃状态：终止但不清理
			go o.terminate(id, "non_active_state")
		} else {
			// 更新问题快照
			entry.Issue = issue
		}
	}
}

// checkStalled 检查停滞的会话
func (o *Orchestrator) checkStalled() {
	if o.cfg.Codex.StallTimeoutMs <= 0 {
		return
	}

	now := time.Now()
	stallTimeout := time.Duration(o.cfg.Codex.StallTimeoutMs) * time.Millisecond

	for id, entry := range o.state.Running {
		var lastActivity time.Time
		if entry.Session != nil && entry.Session.LastCodexTimestamp != nil {
			lastActivity = *entry.Session.LastCodexTimestamp
		} else {
			lastActivity = entry.StartedAt
		}

		if now.Sub(lastActivity) > stallTimeout {
			fmt.Printf("session %s appears stalled, terminating\n", id)
			go o.terminate(id, "stalled")
		}
	}
}

// shouldDispatch 判断是否应该调度
func (o *Orchestrator) shouldDispatch(issue *domain.Issue) bool {
	// 检查必填字段
	if issue.ID == "" || issue.Identifier == "" || issue.Title == "" || issue.State == "" {
		return false
	}

	// 检查状态
	if !o.cfg.IsActiveState(issue.State) {
		return false
	}
	if o.cfg.IsTerminalState(issue.State) {
		return false
	}

	// 检查是否已在运行或已声明
	o.mu.RLock()
	_, running := o.state.Running[issue.ID]
	_, claimed := o.state.Claimed[issue.ID]
	o.mu.RUnlock()

	if running || claimed {
		return false
	}

	// 检查 Todo 状态的阻塞规则
	if strings.ToLower(issue.State) == "todo" {
		for _, blocker := range issue.BlockedBy {
			if blocker.State != nil && !o.cfg.IsTerminalState(*blocker.State) {
				return false // 有非终态阻塞项
			}
		}
	}

	return true
}

// hasAvailableSlots 检查是否有可用槽位
func (o *Orchestrator) hasAvailableSlots() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()

	runningCount := len(o.state.Running)
	return runningCount < o.state.MaxConcurrentAgents
}

// getAvailableSlotsForState 获取特定状态的可用槽位
func (o *Orchestrator) getAvailableSlotsForState(state string) int {
	normalizedState := strings.ToLower(strings.TrimSpace(state))

	// 检查按状态的并发限制
	if limit, ok := o.cfg.Agent.MaxConcurrentAgentsByState[normalizedState]; ok {
		o.mu.RLock()
		defer o.mu.RUnlock()

		count := 0
		for _, entry := range o.state.Running {
			if entry.Issue != nil && strings.ToLower(entry.Issue.State) == normalizedState {
				count++
			}
		}
		return limit - count
	}

	// 使用全局限制
	o.mu.RLock()
	defer o.mu.RUnlock()

	runningCount := len(o.state.Running)
	return o.state.MaxConcurrentAgents - runningCount
}

// sortForDispatch 为调度排序问题
func (o *Orchestrator) sortForDispatch(issues []*domain.Issue) []*domain.Issue {
	sort.Slice(issues, func(i, j int) bool {
		// 优先级升序
		prioI := 999
		prioJ := 999
		if issues[i].Priority != nil {
			prioI = *issues[i].Priority
		}
		if issues[j].Priority != nil {
			prioJ = *issues[j].Priority
		}
		if prioI != prioJ {
			return prioI < prioJ
		}

		// 创建时间升序
		if issues[i].CreatedAt != nil && issues[j].CreatedAt != nil {
			if !issues[i].CreatedAt.Equal(*issues[j].CreatedAt) {
				return issues[i].CreatedAt.Before(*issues[j].CreatedAt)
			}
		}

		// 标识符字典序
		return issues[i].Identifier < issues[j].Identifier
	})

	return issues
}

// dispatch 调度一个问题
func (o *Orchestrator) dispatch(ctx context.Context, issue *domain.Issue, attempt *int) {
	o.mu.Lock()

	// 再次检查
	if _, running := o.state.Running[issue.ID]; running {
		o.mu.Unlock()
		return
	}
	if _, claimed := o.state.Claimed[issue.ID]; claimed {
		o.mu.Unlock()
		return
	}

	// 创建运行条目
	entry := &domain.RunningEntry{
		Issue:        issue,
		Identifier:   issue.Identifier,
		RetryAttempt: attempt,
		StartedAt:    time.Now(),
		TurnCount:    0,
	}

	o.state.Running[issue.ID] = entry
	o.state.Claimed[issue.ID] = struct{}{}
	delete(o.state.RetryAttempts, issue.ID)

	o.mu.Unlock()

	// 启动worker
	go o.runWorker(ctx, issue, attempt)
}

// runWorker 运行worker
func (o *Orchestrator) runWorker(ctx context.Context, issue *domain.Issue, attempt *int) {
	var retryAttempt int
	if attempt != nil {
		retryAttempt = *attempt
	}

	fmt.Printf("starting worker for %s (attempt %d)\n", issue.Identifier, retryAttempt)

	// 创建工作空间
	ws, err := o.workspaceMgr.CreateForIssue(ctx, issue.Identifier)
	if err != nil {
		o.onWorkerExit(issue.ID, issue.Identifier, fmt.Errorf("workspace error: %w", err), attempt)
		return
	}

	// 运行 before_run 钩子
	if err := o.workspaceMgr.RunBeforeRunHook(ctx, ws.Path); err != nil {
		o.onWorkerExit(issue.ID, issue.Identifier, fmt.Errorf("before_run hook error: %w", err), attempt)
		return
	}

	// 运行代理
	result, err := o.agentRunner.RunAttempt(
		ctx,
		issue,
		ws.Path,
		attempt,
		o.promptTemplate,
		func(event string, data any) {
			o.onAgentEvent(issue.ID, event, data)
		},
	)

	// 运行 after_run 钩子
	o.workspaceMgr.RunAfterRunHook(ctx, ws.Path)

	if err != nil {
		o.onWorkerExit(issue.ID, issue.Identifier, err, attempt)
		return
	}

	if result.Success {
		fmt.Printf("worker for %s completed successfully (turns: %d)\n", issue.Identifier, result.TurnCount)
		// 成功退出，安排续行重试
		o.scheduleContinuationRetry(issue.ID, issue.Identifier)
	} else {
		o.onWorkerExit(issue.ID, issue.Identifier, fmt.Errorf("run failed: %v", result.Error), attempt)
	}
}

// onAgentEvent 处理代理事件
func (o *Orchestrator) onAgentEvent(issueID, event string, data any) {
	o.mu.Lock()
	defer o.mu.Unlock()

	entry, ok := o.state.Running[issueID]
	if !ok {
		return
	}

	if entry.Session == nil {
		entry.Session = &domain.LiveSession{}
	}

	now := time.Now()
	entry.Session.LastCodexEvent = &event
	entry.Session.LastCodexTimestamp = &now
	entry.Session.LastCodexMessage = data

	// 更新token统计
	if params, ok := data.(map[string]any); ok {
		if usage, ok := params["usage"].(map[string]any); ok {
			if input, ok := usage["input_tokens"].(float64); ok {
				entry.Session.CodexInputTokens = int64(input)
			}
			if output, ok := usage["output_tokens"].(float64); ok {
				entry.Session.CodexOutputTokens = int64(output)
			}
			if total, ok := usage["total_tokens"].(float64); ok {
				entry.Session.CodexTotalTokens = int64(total)
			}
		}
	}

	// 更新全局统计
	o.state.CodexTotals.InputTokens = entry.Session.CodexInputTokens
	o.state.CodexTotals.OutputTokens = entry.Session.CodexOutputTokens
	o.state.CodexTotals.TotalTokens = entry.Session.CodexTotalTokens
}

// onWorkerExit worker退出处理
func (o *Orchestrator) onWorkerExit(issueID, identifier string, err error, attempt *int) {
	o.mu.Lock()
	defer o.mu.Unlock()

	entry, ok := o.state.Running[issueID]
	if ok {
		// 累加运行时间
		o.endedRuntime += time.Since(entry.StartedAt).Seconds()
		delete(o.state.Running, issueID)
	}

	delete(o.state.Claimed, issueID)

	if err != nil {
		fmt.Printf("worker for %s exited with error: %v\n", identifier, err)
		o.scheduleRetry(issueID, identifier, attempt, err.Error())
	}
}

// terminate 终止运行
func (o *Orchestrator) terminate(issueID, reason string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if entry, ok := o.state.Running[issueID]; ok {
		o.endedRuntime += time.Since(entry.StartedAt).Seconds()
		delete(o.state.Running, issueID)
	}

	delete(o.state.Claimed, issueID)
	fmt.Printf("terminated %s: %s\n", issueID, reason)
}

// terminateAndCleanup 终止并清理工作空间
func (o *Orchestrator) terminateAndCleanup(issueID string, entry *domain.RunningEntry) {
	o.terminate(issueID, "terminal_state")

	// 清理工作空间
	if entry.Issue != nil {
		ctx := context.Background()
		wsPath := o.workspaceMgr.GetWorkspacePath(entry.Issue.Identifier)
		if err := o.workspaceMgr.RemoveWorkspace(ctx, wsPath); err != nil {
			fmt.Printf("failed to cleanup workspace for %s: %v\n", entry.Issue.Identifier, err)
		}
	}
}

// scheduleRetry 安排重试
func (o *Orchestrator) scheduleRetry(issueID, identifier string, attempt *int, errorMsg string) {
	var nextAttempt int
	if attempt != nil {
		nextAttempt = *attempt + 1
	} else {
		nextAttempt = 1
	}

	// 计算退避时间
	delay := o.calculateBackoff(nextAttempt)

	errCopy := errorMsg
	entry := &domain.RetryEntry{
		IssueID:    issueID,
		Identifier: identifier,
		Attempt:    nextAttempt,
		DueAtMs:    time.Now().Add(delay).UnixMilli(),
		Error:      &errCopy,
	}

	o.state.RetryAttempts[issueID] = entry

	// 取消现有定时器
	if timer, ok := o.retryTimers[issueID]; ok {
		timer.Stop()
	}

	// 安排新定时器
	o.retryTimers[issueID] = time.AfterFunc(delay, func() {
		o.onRetryTimer(issueID)
	})

	fmt.Printf("scheduled retry for %s in %v (attempt %d)\n", identifier, delay, nextAttempt)
}

// scheduleContinuationRetry 安排续行重试
func (o *Orchestrator) scheduleContinuationRetry(issueID, identifier string) {
	attempt := 1 // 续行重试使用 attempt 1

	entry := &domain.RetryEntry{
		IssueID:    issueID,
		Identifier: identifier,
		Attempt:    attempt,
		DueAtMs:    time.Now().Add(1 * time.Second).UnixMilli(),
	}

	o.mu.Lock()
	o.state.RetryAttempts[issueID] = entry
	o.mu.Unlock()

	// 安排短延迟定时器
	if timer, ok := o.retryTimers[issueID]; ok {
		timer.Stop()
	}

	o.retryTimers[issueID] = time.AfterFunc(1*time.Second, func() {
		o.onRetryTimer(issueID)
	})
}

// calculateBackoff 计算退退避时间
func (o *Orchestrator) calculateBackoff(attempt int) time.Duration {
	// delay = min(10000 * 2^(attempt-1), max_backoff)
	baseMs := int64(10000 * (1 << (attempt - 1)))
	maxMs := o.cfg.Agent.MaxRetryBackoffMs

	if baseMs > maxMs {
		baseMs = maxMs
	}

	return time.Duration(baseMs) * time.Millisecond
}

// onRetryTimer 重试定时器触发
func (o *Orchestrator) onRetryTimer(issueID string) {
	ctx := context.Background()

	o.mu.Lock()
	retryEntry, ok := o.state.RetryAttempts[issueID]
	if !ok {
		o.mu.Unlock()
		return
	}
	delete(o.state.RetryAttempts, issueID)
	o.mu.Unlock()

	// 获取候选问题
	issues, err := o.trackerClient.FetchCandidateIssues(ctx, o.cfg.Tracker.ActiveStates)
	if err != nil {
		fmt.Printf("retry poll failed for %s: %v\n", issueID, err)
		o.scheduleRetry(issueID, retryEntry.Identifier, &retryEntry.Attempt, "retry_poll_failed")
		return
	}

	// 查找特定问题
	var issue *domain.Issue
	for _, i := range issues {
		if i.ID == issueID {
			issue = i
			break
		}
	}

	if issue == nil {
		// 问题不再存在，释放声明
		o.mu.Lock()
		delete(o.state.Claimed, issueID)
		o.mu.Unlock()
		fmt.Printf("issue %s no longer found, releasing claim\n", issueID)
		return
	}

	// 检查槽位
	if !o.hasAvailableSlots() {
		o.scheduleRetry(issueID, retryEntry.Identifier, &retryEntry.Attempt, "no available orchestrator slots")
		return
	}

	// 检查是否应该调度
	if !o.shouldDispatch(issue) {
		o.mu.Lock()
		delete(o.state.Claimed, issueID)
		o.mu.Unlock()
		return
	}

	// 调度
	o.dispatch(ctx, issue, &retryEntry.Attempt)
}

// startupCleanup 启动时清理终态工作空间
func (o *Orchestrator) startupCleanup(ctx context.Context) error {
	terminalIssues, err := o.trackerClient.FetchIssuesByStates(ctx, o.cfg.Tracker.TerminalStates)
	if err != nil {
		return err
	}

	return o.workspaceMgr.CleanupTerminalWorkspaces(ctx, terminalIssues)
}

// GetState 获取当前状态快照
func (o *Orchestrator) GetState() *domain.OrchestratorState {
	o.mu.RLock()
	defer o.mu.RUnlock()

	// 创建状态副本
	state := &domain.OrchestratorState{
		PollIntervalMs:      o.state.PollIntervalMs,
		MaxConcurrentAgents: o.state.MaxConcurrentAgents,
		Running:             make(map[string]*domain.RunningEntry),
		Claimed:             make(map[string]struct{}),
		RetryAttempts:       make(map[string]*domain.RetryEntry),
		Completed:           make(map[string]struct{}),
		CodexTotals:         &domain.CodexTotals{},
		CodexRateLimits:     o.state.CodexRateLimits,
	}

	for k, v := range o.state.Running {
		state.Running[k] = v
	}
	for k, v := range o.state.Claimed {
		state.Claimed[k] = v
	}
	for k, v := range o.state.RetryAttempts {
		state.RetryAttempts[k] = v
	}
	for k, v := range o.state.Completed {
		state.Completed[k] = v
	}

	// 计算运行时间
	state.CodexTotals = &domain.CodexTotals{
		InputTokens:    o.state.CodexTotals.InputTokens,
		OutputTokens:   o.state.CodexTotals.OutputTokens,
		TotalTokens:    o.state.CodexTotals.TotalTokens,
		SecondsRunning: o.endedRuntime + time.Since(o.startTime).Seconds(),
	}

	return state
}

// UpdateConfig 更新配置
func (o *Orchestrator) UpdateConfig(cfg *config.Config, promptTemplate string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.cfg = cfg
	o.promptTemplate = promptTemplate
	o.state.PollIntervalMs = cfg.Polling.IntervalMs
	o.state.MaxConcurrentAgents = cfg.Agent.MaxConcurrentAgents

	fmt.Println("config updated")
}
