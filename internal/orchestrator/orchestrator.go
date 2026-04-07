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
	"github.com/dministrator/symphony/internal/logging"
	"github.com/dministrator/symphony/internal/tracker"
	"github.com/dministrator/symphony/internal/workflow"
	"github.com/dministrator/symphony/internal/workspace"
)

// Orchestrator 核心编排器
type Orchestrator struct {
	cfg              *config.Config
	trackerClient    tracker.Tracker
	workspaceMgr     *workspace.Manager
	agentRunner      agent.Runner
	promptTemplate   string
	workflowEngine   *workflow.Engine
	constraintMgr    *workflow.ConstraintManager // BDD 约束管理器

	// 运行时状态
	mu           sync.RWMutex
	state        *domain.OrchestratorState
	retryTimers  map[string]*time.Timer
	startTime    time.Time
	endedRuntime float64 // 已结束会话的累计运行时间
	shuttingDown bool    // 是否正在关闭

	// 取消控制
	cancelContexts map[string]context.CancelFunc // 每个任务的取消函数

	// 事件回调
	onStateChange func()
}

// New 创建新的编排器
func New(cfg *config.Config, promptTemplate string) *Orchestrator {
	workflowEngine := workflow.NewEngine()
	constraintMgr := workflow.NewConstraintManager(workflowEngine, cfg.Workspace.Root)

	return &Orchestrator{
		cfg:            cfg,
		trackerClient:  tracker.NewTracker(cfg),
		workspaceMgr:   workspace.NewManager(cfg),
		agentRunner:    agent.NewRunner(cfg),
		promptTemplate: promptTemplate,
		workflowEngine: workflowEngine,
		constraintMgr:  constraintMgr,
		retryTimers:    make(map[string]*time.Timer),
		cancelContexts: make(map[string]context.CancelFunc),
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

	// 初始化日志系统
	if err := logging.Initialize(logging.Config{
		Level:        o.cfg.Logging.Level,
		Format:       o.cfg.Logging.Format,
		FilePath:     o.cfg.Logging.FilePath,
		EnableStdout: o.cfg.Logging.EnableStdout,
	}); err != nil {
		fmt.Printf("logging initialization warning: %v\n", err)
	}

	logging.Info("orchestrator started")

	// 启动时清理终态工作空间
	if err := o.startupCleanup(ctx); err != nil {
		logging.Warn("startup cleanup warning", "error", err.Error())
	}

	// 立即执行一次tick
	o.tick(ctx)

	// 启动轮询
	ticker := time.NewTicker(time.Duration(o.cfg.Polling.IntervalMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logging.Info("orchestrator stopped", "reason", ctx.Err().Error())
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
		logging.Error("dispatch validation failed", "errors", strings.Join(validation.Errors, ", "))
		return
	}

	// 3. 获取候选问题
	issues, err := o.trackerClient.FetchCandidateIssues(ctx, o.cfg.Tracker.ActiveStates)
	if err != nil {
		logging.Error("failed to fetch candidate issues", "error", err.Error())
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
		logging.Error("failed to refresh issue states", "error", err.Error())
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
			logging.LogSessionStalled(id, entry.Identifier)
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

	// 启动worker（支持取消）
	go o.runWorkerWithCancel(ctx, issue, attempt)
}

// runWorker 运行worker
func (o *Orchestrator) runWorker(ctx context.Context, issue *domain.Issue, attempt *int) {
	var retryAttempt int
	if attempt != nil {
		retryAttempt = *attempt
	}

	logging.LogWorkerStarted(issue.ID, issue.Identifier, retryAttempt)

	// 创建工作空间
	ws, err := o.workspaceMgr.CreateForIssue(ctx, issue.Identifier)
	if err != nil {
		logging.LogWorkspaceError(issue.ID, issue.Identifier, err)
		o.onWorkerExit(issue.ID, issue.Identifier, fmt.Errorf("workspace error: %w", err), attempt)
		return
	}

	logging.LogWorkspaceCreated(issue.ID, issue.Identifier, ws.Path)

	// 运行 before_run 钩子
	if err := o.workspaceMgr.RunBeforeRunHook(ctx, ws.Path); err != nil {
		logging.Error("before_run hook failed",
			"task_id", issue.ID,
			"identifier", issue.Identifier,
			"error", err.Error(),
		)
		o.onWorkerExit(issue.ID, issue.Identifier, fmt.Errorf("before_run hook error: %w", err), attempt)
		return
	}

	// 构建包含 BDD 约束的完整 prompt
	fullPrompt := o.buildFullPrompt(ctx, issue, attempt)

	// 运行代理
	result, err := o.agentRunner.RunAttempt(
		ctx,
		issue,
		ws.Path,
		attempt,
		fullPrompt,
		func(event string, data any) {
			o.onAgentEvent(issue.ID, event, data)
		},
	)

	// 运行 after_run 钩子
	_ = o.workspaceMgr.RunAfterRunHook(ctx, ws.Path)

	if err != nil {
		logging.LogAgentError(issue.ID, issue.Identifier, err)
		o.onWorkerExit(issue.ID, issue.Identifier, err, attempt)
		return
	}

	if result.Success {
		logging.LogTaskCompleted(issue.ID, issue.Identifier, "agent", result.TurnCount)
		// 成功退出，安排续行重试
		o.scheduleContinuationRetry(issue.ID, issue.Identifier)
	} else {
		logging.LogTaskFailed(issue.ID, issue.Identifier, "agent", fmt.Errorf("run failed: %v", result.Error))
		o.onWorkerExit(issue.ID, issue.Identifier, fmt.Errorf("run failed: %v", result.Error), attempt)
	}
}

// buildFullPrompt 构建包含 BDD 约束的完整 prompt
// 按照方案 B，在调用 Agent 前构建完整 prompt
func (o *Orchestrator) buildFullPrompt(ctx context.Context, issue *domain.Issue, attempt *int) string {
	// 获取工作流状态
	taskWorkflow := o.workflowEngine.GetWorkflow(issue.ID)
	
	// 加载 BDD 约束条件
	var bddConstraints string
	if o.constraintMgr != nil && taskWorkflow != nil {
		// 检查 BDD 审核阶段是否已完成
		bddStage := taskWorkflow.GetStage(workflow.StageBDDReview)
		if bddStage != nil && bddStage.Status == workflow.StatusCompleted {
			constraints, err := o.constraintMgr.LoadBDDConstraints(issue.ID)
			if err == nil && constraints != nil {
				bddConstraints = o.constraintMgr.FormatConstraintsForPrompt(constraints)
				logging.Info("bdd constraints loaded for implementation",
					"task_id", issue.ID,
					"identifier", issue.Identifier,
					"scenarios_count", len(constraints.Scenarios),
				)
			}
		}
	}
	
	// 如果没有 BDD 约束，返回原始 prompt
	if bddConstraints == "" {
		return o.promptTemplate
	}
	
	// 注入 BDD 约束到 prompt
	return agent.InjectBDDConstraints(o.promptTemplate, bddConstraints)
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

	// 记录代理事件
	logging.LogAgentEvent(issueID, event, data)

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

	var retryAttempt int
	if attempt != nil {
		retryAttempt = *attempt
	}

	entry, ok := o.state.Running[issueID]
	if ok {
		// 累加运行时间
		o.endedRuntime += time.Since(entry.StartedAt).Seconds()
		delete(o.state.Running, issueID)
	}

	delete(o.state.Claimed, issueID)

	logging.LogWorkerExit(issueID, identifier, err, retryAttempt)

	if err != nil {
		// 检查是否达到重试上限
		if o.shouldTransitionToNeedsAttention(retryAttempt) {
			o.transitionToNeedsAttentionLocked(issueID, identifier, retryAttempt, err.Error())
		} else {
			o.scheduleRetryLocked(issueID, identifier, attempt, err.Error())
		}
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
	logging.LogTermination(issueID, reason)
}

// terminateAndCleanup 终止并清理工作空间
func (o *Orchestrator) terminateAndCleanup(issueID string, entry *domain.RunningEntry) {
	o.terminate(issueID, "terminal_state")

	// 清理工作空间
	if entry.Issue != nil {
		ctx := context.Background()
		wsPath := o.workspaceMgr.GetWorkspacePath(entry.Issue.Identifier)
		if err := o.workspaceMgr.RemoveWorkspace(ctx, wsPath); err != nil {
			logging.LogWorkspaceError(issueID, entry.Issue.Identifier, err)
		}
	}
}

// scheduleRetry 安排重试
func (o *Orchestrator) scheduleRetry(issueID, identifier string, attempt *int, errorMsg string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.scheduleRetryLocked(issueID, identifier, attempt, errorMsg)
}

// scheduleRetryLocked 安排重试（已持有锁版本）
func (o *Orchestrator) scheduleRetryLocked(issueID, identifier string, attempt *int, errorMsg string) {
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

	logging.LogRetryScheduled(issueID, identifier, nextAttempt, delay.Milliseconds())
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

// shouldTransitionToNeedsAttention 判断是否应该流转到待人工处理状态
// 当重试次数达到配置的上限时返回 true
func (o *Orchestrator) shouldTransitionToNeedsAttention(retryAttempt int) bool {
	maxRetries := o.cfg.Execution.MaxRetries
	if maxRetries <= 0 {
		return false // 无限制，不流转
	}
	return retryAttempt >= maxRetries
}

// transitionToNeedsAttentionLocked 流转到待人工处理状态
// 注意：此方法必须在持有锁的情况下调用
func (o *Orchestrator) transitionToNeedsAttentionLocked(issueID, identifier string, retryAttempt int, errorMsg string) {
	logging.Info("transitioning task to needs_attention",
		"task_id", issueID,
		"identifier", identifier,
		"retry_attempt", retryAttempt,
		"error", errorMsg,
	)

	// 获取当前阶段
	currentStage, err := o.workflowEngine.GetCurrentStage(issueID)
	if err != nil {
		logging.Error("failed to get current stage for needs_attention transition",
			"task_id", issueID,
			"error", err.Error(),
		)
		return
	}

	// 构建失败详情
	details := workflow.NeedsAttentionDetails{
		FailedStage:    string(currentStage.Name),
		FailedAt:       time.Now(),
		RetryCount:     retryAttempt,
		MaxRetries:     o.cfg.Execution.MaxRetries,
		ErrorType:      "execution_failure",
		ErrorMessage:   errorMsg,
		Suggestion:     "请检查错误信息并手动修复后继续执行",
	}

	// 流转到待人工处理状态
	_, err = o.workflowEngine.TransitionToNeedsAttention(issueID, details)
	if err != nil {
		logging.Error("failed to transition to needs_attention",
			"task_id", issueID,
			"error", err.Error(),
		)
		return
	}

	// 异步更新 Tracker 状态（不在持有锁的情况下进行网络调用）
	go func() {
		ctx := context.Background()
		if err := o.trackerClient.UpdateStage(ctx, identifier, domain.StageState{
			Name:   string(workflow.StageNeedsAttention),
			Status: string(workflow.StatusInProgress),
		}); err != nil {
			logging.Error("failed to update tracker state for needs_attention",
				"task_id", issueID,
				"identifier", identifier,
				"error", err.Error(),
			)
		}
	}()

	// 通知观察者（触发 SSE 推送）
	if o.onStateChange != nil {
		go o.onStateChange()
	}

	logging.Info("task transitioned to needs_attention",
		"task_id", issueID,
		"identifier", identifier,
		"failed_stage", currentStage.Name,
		"retry_count", retryAttempt,
	)
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
		logging.Error("retry poll failed",
			"task_id", issueID,
			"error", err.Error(),
		)
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
		logging.Warn("issue no longer found, releasing claim", "task_id", issueID)
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

	logging.Info("config updated")
}

// Shutdown 优雅关闭编排器
func (o *Orchestrator) Shutdown(ctx context.Context) error {
	o.mu.Lock()
	// 标记正在关闭，停止接受新任务
	o.shuttingDown = true
	o.mu.Unlock()

	logging.Info("orchestrator shutting down")

	// 创建关闭管理器
	shutdownMgr := NewShutdownManager(o, o.trackerClient, o.cfg)

	// 执行优雅关闭
	return shutdownMgr.Shutdown(ctx)
}

// IsShuttingDown 检查是否正在关闭
func (o *Orchestrator) IsShuttingDown() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.shuttingDown
}

// GetTracker 获取跟踪器客户端（用于 API 查询）
func (o *Orchestrator) GetTracker() tracker.Tracker {
	return o.trackerClient
}

// CancelTask 取消运行中的任务
// 返回值:
//   - cancelled: 是否成功取消
//   - notFound: 任务是否不存在
//   - err: 错误信息
func (o *Orchestrator) CancelTask(identifier string) (cancelled bool, notFound bool, err error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	// 查找运行中的任务
	var foundID string
	var foundEntry *domain.RunningEntry
	for id, entry := range o.state.Running {
		if entry.Identifier == identifier {
			foundID = id
			foundEntry = entry
			break
		}
	}

	if foundEntry == nil {
		// 检查是否在重试队列中
		var foundRetry *domain.RetryEntry
		for id, entry := range o.state.RetryAttempts {
			if entry.Identifier == identifier {
				foundRetry = entry
				foundID = id
				break
			}
		}

		if foundRetry == nil {
			return false, true, nil // 任务不存在
		}

		// 取消重试队列中的任务
		delete(o.state.RetryAttempts, foundID)
		delete(o.state.Claimed, foundID)

		// 取消重试定时器
		if timer, ok := o.retryTimers[foundID]; ok {
			timer.Stop()
			delete(o.retryTimers, foundID)
		}

		logging.Info("cancelled retry", "task_id", foundID, "identifier", identifier)
		return true, false, nil
	}

	// 取消运行中的任务
	// 1. 调用取消函数（终止 Agent 进程）
	if cancelFn, ok := o.cancelContexts[foundID]; ok {
		cancelFn()
		delete(o.cancelContexts, foundID)
	}

	// 2. 更新 Tracker 状态为 "Cancelled"
	if foundEntry.Issue != nil {
		ctx := context.Background()
		if err := o.trackerClient.UpdateStage(ctx, identifier, domain.StageState{
			Name:   "cancelled",
			Status: "completed",
		}); err != nil {
			logging.Error("failed to update tracker state",
				"task_id", foundID,
				"identifier", identifier,
				"error", err.Error(),
			)
		}
	}

	// 3. 累加运行时间
	o.endedRuntime += time.Since(foundEntry.StartedAt).Seconds()

	// 4. 从运行中移除
	delete(o.state.Running, foundID)
	delete(o.state.Claimed, foundID)

	// 5. 取消重试定时器（如果存在）
	if timer, ok := o.retryTimers[foundID]; ok {
		timer.Stop()
		delete(o.retryTimers, foundID)
	}

	logging.Info("cancelled task", "task_id", foundID, "identifier", identifier)

	// 6. 异步清理工作空间
	go func() {
		ctx := context.Background()
		wsPath := o.workspaceMgr.GetWorkspacePath(identifier)
		if err := o.workspaceMgr.RemoveWorkspace(ctx, wsPath); err != nil {
			logging.LogWorkspaceError(foundID, identifier, err)
		}
	}()

	// 7. 通知观察者
	if o.onStateChange != nil {
		o.onStateChange()
	}

	return true, false, nil
}

// GetRunningEntryByIdentifier 根据标识符获取运行中的任务条目
func (o *Orchestrator) GetRunningEntryByIdentifier(identifier string) *domain.RunningEntry {
	o.mu.RLock()
	defer o.mu.RUnlock()

	for _, entry := range o.state.Running {
		if entry.Identifier == identifier {
			return entry
		}
	}
	return nil
}

// GetRetryEntryByIdentifier 根据标识符获取重试队列中的任务条目
func (o *Orchestrator) GetRetryEntryByIdentifier(identifier string) *domain.RetryEntry {
	o.mu.RLock()
	defer o.mu.RUnlock()

	for _, entry := range o.state.RetryAttempts {
		if entry.Identifier == identifier {
			return entry
		}
	}
	return nil
}

// storeCancelContext 存储任务的取消函数
func (o *Orchestrator) storeCancelContext(issueID string, cancelFn context.CancelFunc) {
	o.mu.Lock()
	o.cancelContexts[issueID] = cancelFn
	o.mu.Unlock()
}

// runWorkerWithCancel 运行 worker，支持取消
func (o *Orchestrator) runWorkerWithCancel(parentCtx context.Context, issue *domain.Issue, attempt *int) {
	// 创建可取消的上下文
	ctx, cancelFn := context.WithCancel(parentCtx)
	o.storeCancelContext(issue.ID, cancelFn)

	// 初始化任务工作流（如果不存在）
	if o.workflowEngine.GetWorkflow(issue.ID) == nil {
		_, _ = o.workflowEngine.InitTask(issue.ID)
	}

	// 使用原有的 runWorker 逻辑
	o.runWorker(ctx, issue, attempt)

	// 清理取消函数
	o.mu.Lock()
	delete(o.cancelContexts, issue.ID)
	o.mu.Unlock()
}

// GetWorkflowEngine 获取工作流引擎
func (o *Orchestrator) GetWorkflowEngine() *workflow.Engine {
	return o.workflowEngine
}

// GetConfig 获取配置
func (o *Orchestrator) GetConfig() *config.Config {
	return o.cfg
}

// GetWorkspaceManager 获取工作空间管理器
func (o *Orchestrator) GetWorkspaceManager() *workspace.Manager {
	return o.workspaceMgr
}

// AdvanceTaskStage 推进任务到下一阶段
func (o *Orchestrator) AdvanceTaskStage(taskID string) (*workflow.TaskWorkflow, error) {
	return o.workflowEngine.AdvanceStage(taskID)
}

// FailTaskStage 标记任务当前阶段为失败
func (o *Orchestrator) FailTaskStage(taskID string, reason string) (*workflow.TaskWorkflow, error) {
	return o.workflowEngine.FailStage(taskID, reason)
}

// GetTaskWorkflow 获取任务工作流状态
func (o *Orchestrator) GetTaskWorkflow(taskID string) *workflow.TaskWorkflow {
	return o.workflowEngine.GetWorkflow(taskID)
}

// InitTaskWorkflow 初始化任务工作流
func (o *Orchestrator) InitTaskWorkflow(taskID string) (*workflow.TaskWorkflow, error) {
	return o.workflowEngine.InitTask(taskID)
}

// GetTaskCurrentStage 获取任务当前阶段
func (o *Orchestrator) GetTaskCurrentStage(taskID string) (*workflow.StageState, error) {
	return o.workflowEngine.GetCurrentStage(taskID)
}
