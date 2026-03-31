// Package workflow 提供实现阶段的管理功能
package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dministrator/symphony/internal/agent"
	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/domain"
	"github.com/dministrator/symphony/internal/tracker"
)

// ExecutionLog 执行日志条目
type ExecutionLog struct {
	// Timestamp 日志时间戳
	Timestamp time.Time `json:"timestamp"`
	// Event 事件类型
	Event string `json:"event"`
	// Message 日志消息
	Message string `json:"message"`
	// Data 附加数据
	Data map[string]interface{} `json:"data,omitempty"`
}

// ExecutionProgress 执行进度
type ExecutionProgress struct {
	// TaskID 任务ID
	TaskID string `json:"task_id"`
	// Identifier 任务标识符
	Identifier string `json:"identifier"`
	// CurrentStage 当前阶段
	CurrentStage StageName `json:"current_stage"`
	// Status 执行状态
	Status StageStatus `json:"status"`
	// StartedAt 开始时间
	StartedAt *time.Time `json:"started_at,omitempty"`
	// UpdatedAt 更新时间
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	// ElapsedSeconds 已用时间（秒）
	ElapsedSeconds int64 `json:"elapsed_seconds"`
	// TurnCount 轮次计数
	TurnCount int `json:"turn_count"`
	// TokenUsage Token使用量
	TokenUsage *TokenUsageInfo `json:"token_usage,omitempty"`
	// LastEvent 最后事件
	LastEvent string `json:"last_event,omitempty"`
	// LastMessage 最后消息
	LastMessage string `json:"last_message,omitempty"`
	// ProgressSummary 进度摘要
	ProgressSummary string `json:"progress_summary,omitempty"`
	// Error 错误信息
	Error string `json:"error,omitempty"`
	// RetryCount 重试次数
	RetryCount int `json:"retry_count"`
	// MaxRetries 最大重试次数
	MaxRetries int `json:"max_retries"`
}

// TokenUsageInfo Token使用信息
type TokenUsageInfo struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	TotalTokens  int64 `json:"total_tokens"`
}

// ImplementationManager 实现阶段管理器
type ImplementationManager struct {
	engine            *Engine
	config            *config.Config
	tracker           tracker.Tracker
	runner            agent.Runner
	constraintManager *ConstraintManager
	contextBuilder    *ContextBuilder

	// 执行日志存储
	mu           sync.RWMutex
	execLogs     map[string][]ExecutionLog     // taskID -> []ExecutionLog
	execProgress map[string]*ExecutionProgress // taskID -> ExecutionProgress
}

// NewImplementationManager 创建新的实现阶段管理器
func NewImplementationManager(engine *Engine, cfg *config.Config) *ImplementationManager {
	return &ImplementationManager{
		engine:         engine,
		config:         cfg,
		contextBuilder: NewContextBuilder(),
		execLogs:       make(map[string][]ExecutionLog),
		execProgress:   make(map[string]*ExecutionProgress),
	}
}

// NewImplementationManagerWithTracker 创建带 tracker 的实现阶段管理器
func NewImplementationManagerWithTracker(engine *Engine, cfg *config.Config, t tracker.Tracker) *ImplementationManager {
	im := NewImplementationManager(engine, cfg)
	im.tracker = t
	return im
}

// SetTracker 设置 tracker
func (im *ImplementationManager) SetTracker(t tracker.Tracker) {
	im.tracker = t
}

// SetRunner 设置 AI Agent 运行器
func (im *ImplementationManager) SetRunner(r agent.Runner) {
	im.runner = r
}

// SetConstraintManager 设置约束管理器
func (im *ImplementationManager) SetConstraintManager(cm *ConstraintManager) {
	im.constraintManager = cm
}

// StartImplementation 开始实现阶段
// 构建包含需求、BDD、架构、TDD 的完整 prompt，调用 AI Agent 执行
func (im *ImplementationManager) StartImplementation(ctx context.Context, taskID, identifier string, issue *domain.Issue, promptTemplate string) (*ExecutionProgress, error) {
	// 获取工作流
	workflow := im.engine.GetWorkflow(taskID)
	if workflow == nil {
		return nil, ErrWorkflowNotFound
	}

	// 检查当前阶段
	if workflow.CurrentStage != StageImplementation {
		return nil, fmt.Errorf("%w: current stage is %s, not implementation", ErrInvalidTransition, workflow.CurrentStage)
	}

	// 初始化执行进度
	now := time.Now()
	progress := &ExecutionProgress{
		TaskID:         taskID,
		Identifier:     identifier,
		CurrentStage:   StageImplementation,
		Status:         StatusInProgress,
		StartedAt:      &now,
		UpdatedAt:      &now,
		TurnCount:      0,
		RetryCount:     0,
		MaxRetries:     im.config.Execution.MaxRetries,
		ProgressSummary: "准备执行...",
	}

	im.mu.Lock()
	im.execProgress[taskID] = progress
	im.execLogs[taskID] = []ExecutionLog{}
	im.mu.Unlock()

	// 记录开始日志
	im.appendLog(taskID, "implementation_started", "开始实现阶段", map[string]interface{}{
		"identifier": identifier,
		"title":      issue.Title,
	})

	// 构建 prompt 上下文
	promptContext, err := im.buildPromptContext(ctx, taskID, identifier, issue)
	if err != nil {
		im.appendLog(taskID, "prompt_build_error", err.Error(), nil)
		return nil, fmt.Errorf("failed to build prompt context: %w", err)
	}

	// 合并 prompt 模板和上下文
	fullPrompt := im.mergePromptWithContext(promptTemplate, promptContext)

	// 记录 prompt 构建完成
	im.appendLog(taskID, "prompt_ready", "Prompt 构建完成", map[string]interface{}{
		"prompt_length": len(fullPrompt),
	})

	// 更新进度
	im.updateProgress(taskID, "正在执行 AI Agent...", "")

	return progress, nil
}

// buildPromptContext 构建完整的 prompt 上下文
// 包含需求信息、对话历史、BDD 约束、架构设计等
func (im *ImplementationManager) buildPromptContext(ctx context.Context, taskID, identifier string, issue *domain.Issue) (string, error) {
	var contextParts []string

	// 1. 添加任务基本信息
	taskInfo := im.contextBuilder.BuildAgentContext(issue, nil)
	contextParts = append(contextParts, taskInfo)

	// 2. 获取对话历史
	if im.tracker != nil {
		history, err := im.tracker.GetConversationHistory(ctx, identifier)
		if err == nil && len(history) > 0 {
			historySection := im.contextBuilder.FormatConversationHistoryOnly(history)
			if historySection != "" {
				contextParts = append(contextParts, historySection)
			}
		}
	}

	// 3. 添加 BDD 约束条件
	if im.constraintManager != nil {
		bddPrompt, err := im.constraintManager.GetBDDConstraintsForPrompt(taskID)
		if err == nil && bddPrompt != "" {
			contextParts = append(contextParts, bddPrompt)
		}
	}

	// 4. 获取 BDD 约束文件路径
	bddFilePath := ""
	if im.constraintManager != nil {
		bddFilePath = im.constraintManager.GetConstraintFilePath(taskID)
		if bddFilePath != "" {
			im.appendLog(taskID, "bdd_constraints_loaded", "BDD 约束条件已加载", map[string]interface{}{
				"file_path": bddFilePath,
			})
		}
	}

	// 保存 BDD 文件路径到工作流 metadata
	if bddFilePath != "" {
		_ = im.engine.SetBDDConstraintsPath(taskID, bddFilePath)
	}

	return joinContextParts(contextParts), nil
}

// mergePromptWithContext 合并 prompt 模板和上下文
func (im *ImplementationManager) mergePromptWithContext(template, context string) string {
	if template == "" {
		return context
	}
	if context == "" {
		return template
	}

	// 检查模板是否有上下文占位符
	if contains(template, "{{ context }}") || contains(template, "{{task_context}}") {
		result := replaceAll(template, "{{ context }}", context)
		result = replaceAll(result, "{{task_context}}", context)
		return result
	}

	// 没有占位符时追加到末尾
	return template + "\n\n" + context
}

// RecordAgentEvent 记录 Agent 事件
// 用于实时记录 Agent 执行过程中的事件
func (im *ImplementationManager) RecordAgentEvent(taskID, event string, data map[string]interface{}) {
	im.appendLog(taskID, event, formatEventMessage(event, data), data)

	// 更新进度
	im.mu.Lock()
	if progress, ok := im.execProgress[taskID]; ok {
		progress.LastEvent = event
		if msg, ok := data["message"].(string); ok {
			progress.LastMessage = msg
		}
		now := time.Now()
		progress.UpdatedAt = &now
		if progress.StartedAt != nil {
			progress.ElapsedSeconds = int64(now.Sub(*progress.StartedAt).Seconds())
		}

		// 更新 Token 使用量
		if usage, ok := data["usage"].(map[string]interface{}); ok {
			if progress.TokenUsage == nil {
				progress.TokenUsage = &TokenUsageInfo{}
			}
			if input, ok := usage["input_tokens"].(float64); ok {
				progress.TokenUsage.InputTokens = int64(input)
			}
			if output, ok := usage["output_tokens"].(float64); ok {
				progress.TokenUsage.OutputTokens = int64(output)
			}
			if total, ok := usage["total_tokens"].(float64); ok {
				progress.TokenUsage.TotalTokens = int64(total)
			}
		}

		// 更新进度摘要
		progress.ProgressSummary = im.generateProgressSummary(progress, event, data)
	}
	im.mu.Unlock()
}

// RecordTurnComplete 记录 Turn 完成
func (im *ImplementationManager) RecordTurnComplete(taskID string, turnCount int) {
	im.mu.Lock()
	if progress, ok := im.execProgress[taskID]; ok {
		progress.TurnCount = turnCount
		now := time.Now()
		progress.UpdatedAt = &now
		if progress.StartedAt != nil {
			progress.ElapsedSeconds = int64(now.Sub(*progress.StartedAt).Seconds())
		}
	}
	im.mu.Unlock()

	im.appendLog(taskID, "turn_complete", fmt.Sprintf("Turn %d 完成", turnCount), map[string]interface{}{
		"turn_count": turnCount,
	})
}

// CompleteImplementation 完成实现阶段
func (im *ImplementationManager) CompleteImplementation(taskID string) (*ExecutionProgress, error) {
	// 更新进度状态
	im.mu.Lock()
	if progress, ok := im.execProgress[taskID]; ok {
		progress.Status = StatusCompleted
		now := time.Now()
		progress.UpdatedAt = &now
		if progress.StartedAt != nil {
			progress.ElapsedSeconds = int64(now.Sub(*progress.StartedAt).Seconds())
		}
		progress.ProgressSummary = "实现完成"
	}
	im.mu.Unlock()

	// 推进工作流到下一阶段
	_, err := im.engine.AdvanceStage(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to advance stage: %w", err)
	}

	im.appendLog(taskID, "implementation_completed", "实现阶段完成", nil)

	return im.GetExecutionProgress(taskID)
}

// FailImplementation 实现阶段失败
func (im *ImplementationManager) FailImplementation(taskID, reason string, retryCount int) (*ExecutionProgress, error) {
	// 检查是否达到重试上限
	maxRetries := im.config.Execution.MaxRetries
	needsAttention := retryCount >= maxRetries

	im.mu.Lock()
	if progress, ok := im.execProgress[taskID]; ok {
		progress.Status = StatusFailed
		progress.Error = reason
		progress.RetryCount = retryCount
		progress.MaxRetries = maxRetries
		now := time.Now()
		progress.UpdatedAt = &now
		if progress.StartedAt != nil {
			progress.ElapsedSeconds = int64(now.Sub(*progress.StartedAt).Seconds())
		}

		if needsAttention {
			progress.ProgressSummary = "需要人工处理"
		} else {
			progress.ProgressSummary = fmt.Sprintf("准备重试 (%d/%d)", retryCount+1, maxRetries)
		}
	}
	im.mu.Unlock()

	// 根据重试次数决定下一步
	if needsAttention {
		// 标记为需要人工处理
		_, err := im.engine.SetIncompleteMark(taskID, reason, true)
		if err != nil {
			return nil, fmt.Errorf("failed to mark needs attention: %w", err)
		}

		im.appendLog(taskID, "implementation_failed_needs_attention", "实现失败，需要人工处理", map[string]interface{}{
			"error":       reason,
			"retry_count": retryCount,
		})
	} else {
		// 标记阶段失败（可以重试）
		_, err := im.engine.FailStage(taskID, reason)
		if err != nil {
			return nil, fmt.Errorf("failed to mark stage failed: %w", err)
		}

		im.appendLog(taskID, "implementation_failed_retry", fmt.Sprintf("实现失败，准备重试 (%d/%d)", retryCount+1, maxRetries), map[string]interface{}{
			"error":       reason,
			"retry_count": retryCount,
		})
	}

	return im.GetExecutionProgress(taskID)
}

// GetExecutionProgress 获取执行进度
func (im *ImplementationManager) GetExecutionProgress(taskID string) (*ExecutionProgress, error) {
	im.mu.RLock()
	defer im.mu.RUnlock()

	progress, ok := im.execProgress[taskID]
	if !ok {
		return nil, fmt.Errorf("execution progress not found for task: %s", taskID)
	}

	return progress, nil
}

// GetExecutionLogs 获取执行日志
// 支持分页，每页 pageSize 条，返回第 page 页
func (im *ImplementationManager) GetExecutionLogs(taskID string, page, pageSize int) ([]ExecutionLog, int, error) {
	im.mu.RLock()
	defer im.mu.RUnlock()

	logs, ok := im.execLogs[taskID]
	if !ok {
		return nil, 0, fmt.Errorf("execution logs not found for task: %s", taskID)
	}

	total := len(logs)

	// 默认每页 100 条
	if pageSize <= 0 {
		pageSize = 100
	}

	// 计算分页
	start := page * pageSize
	if start >= total {
		return []ExecutionLog{}, total, nil
	}

	end := start + pageSize
	if end > total {
		end = total
	}

	return logs[start:end], total, nil
}

// GetAllExecutionLogs 获取所有执行日志
func (im *ImplementationManager) GetAllExecutionLogs(taskID string) ([]ExecutionLog, error) {
	im.mu.RLock()
	defer im.mu.RUnlock()

	logs, ok := im.execLogs[taskID]
	if !ok {
		return nil, fmt.Errorf("execution logs not found for task: %s", taskID)
	}

	// 返回副本
	result := make([]ExecutionLog, len(logs))
	copy(result, logs)
	return result, nil
}

// appendLog 追加日志
func (im *ImplementationManager) appendLog(taskID, event, message string, data map[string]interface{}) {
	log := ExecutionLog{
		Timestamp: time.Now(),
		Event:     event,
		Message:   message,
		Data:      data,
	}

	im.mu.Lock()
	im.execLogs[taskID] = append(im.execLogs[taskID], log)
	im.mu.Unlock()
}

// updateProgress 更新进度
func (im *ImplementationManager) updateProgress(taskID, summary, lastEvent string) {
	im.mu.Lock()
	defer im.mu.Unlock()

	if progress, ok := im.execProgress[taskID]; ok {
		progress.ProgressSummary = summary
		if lastEvent != "" {
			progress.LastEvent = lastEvent
		}
		now := time.Now()
		progress.UpdatedAt = &now
		if progress.StartedAt != nil {
			progress.ElapsedSeconds = int64(now.Sub(*progress.StartedAt).Seconds())
		}
	}
}

// generateProgressSummary 生成进度摘要
func (im *ImplementationManager) generateProgressSummary(progress *ExecutionProgress, event string, data map[string]interface{}) string {
	// 根据事件类型生成不同的进度摘要
	switch event {
	case "session_started":
		return "AI Agent 会话已启动"
	case "turn_started":
		return fmt.Sprintf("正在执行 Turn %d...", progress.TurnCount+1)
	case "turn_complete":
		return fmt.Sprintf("Turn %d 完成", progress.TurnCount)
	case "item/message":
		if msg, ok := data["content"].(string); ok && len(msg) > 50 {
			return "正在思考: " + msg[:50] + "..."
		}
		return "正在处理..."
	case "item/tool/call":
		if tool, ok := data["tool_name"].(string); ok {
			return "正在调用工具: " + tool
		}
		return "正在调用工具..."
	case "turn/progress":
		return "执行中..."
	default:
		if progress.LastMessage != "" {
			if len(progress.LastMessage) > 80 {
				return progress.LastMessage[:80] + "..."
			}
			return progress.LastMessage
		}
		return "执行中..."
	}
}

// ClearExecutionLogs 清除执行日志
func (im *ImplementationManager) ClearExecutionLogs(taskID string) {
	im.mu.Lock()
	defer im.mu.Unlock()

	delete(im.execLogs, taskID)
	delete(im.execProgress, taskID)
}

// GetImplementationStatus 获取实现阶段状态
func (im *ImplementationManager) GetImplementationStatus(taskID string) (*ImplementationStatus, error) {
	workflow := im.engine.GetWorkflow(taskID)
	if workflow == nil {
		return nil, ErrWorkflowNotFound
	}

	implStage := workflow.Stages[StageImplementation]
	if implStage == nil {
		return nil, ErrInvalidStage
	}

	status := &ImplementationStatus{
		TaskID:       taskID,
		CurrentStage: workflow.CurrentStage,
		Status:       implStage.Status,
		StartedAt:    implStage.StartedAt,
		UpdatedAt:    implStage.UpdatedAt,
		Error:        implStage.Error,
	}

	// 获取执行进度
	im.mu.RLock()
	if progress, ok := im.execProgress[taskID]; ok {
		status.TurnCount = progress.TurnCount
		status.ElapsedSeconds = progress.ElapsedSeconds
		status.ProgressSummary = progress.ProgressSummary
		status.LastEvent = progress.LastEvent
		status.RetryCount = progress.RetryCount
		status.MaxRetries = progress.MaxRetries
		if progress.TokenUsage != nil {
			status.TokenUsage = &TokenUsageInfo{
				InputTokens:  progress.TokenUsage.InputTokens,
				OutputTokens: progress.TokenUsage.OutputTokens,
				TotalTokens:  progress.TokenUsage.TotalTokens,
			}
		}
	}
	im.mu.RUnlock()

	return status, nil
}

// ImplementationStatus 实现阶段状态
type ImplementationStatus struct {
	TaskID          string            `json:"task_id"`
	CurrentStage    StageName         `json:"current_stage"`
	Status          StageStatus       `json:"status"`
	StartedAt       *time.Time        `json:"started_at,omitempty"`
	UpdatedAt       *time.Time        `json:"updated_at,omitempty"`
	ElapsedSeconds  int64             `json:"elapsed_seconds"`
	TurnCount       int               `json:"turn_count"`
	ProgressSummary string            `json:"progress_summary"`
	LastEvent       string            `json:"last_event,omitempty"`
	TokenUsage      *TokenUsageInfo   `json:"token_usage,omitempty"`
	Error           string            `json:"error,omitempty"`
	RetryCount      int               `json:"retry_count"`
	MaxRetries      int               `json:"max_retries"`
}

// CanContinueImplementation 判断是否可以继续实现
func (im *ImplementationManager) CanContinueImplementation(taskID string) (bool, error) {
	workflow := im.engine.GetWorkflow(taskID)
	if workflow == nil {
		return false, ErrWorkflowNotFound
	}

	// 必须在实现阶段且失败状态
	if workflow.CurrentStage != StageImplementation {
		return false, nil
	}

	implStage := workflow.Stages[StageImplementation]
	if implStage == nil {
		return false, ErrInvalidStage
	}

	// 阶段必须是失败状态
	return implStage.Status == StatusFailed, nil
}

// RetryImplementation 重试实现
func (im *ImplementationManager) RetryImplementation(ctx context.Context, taskID, identifier string) (*ExecutionProgress, error) {
	// 检查是否可以重试
	canContinue, err := im.CanContinueImplementation(taskID)
	if err != nil {
		return nil, err
	}
	if !canContinue {
		return nil, fmt.Errorf("cannot retry implementation: not in retryable state")
	}

	// 重置阶段状态
	err = im.engine.ResetStage(taskID, StageImplementation)
	if err != nil {
		return nil, fmt.Errorf("failed to reset implementation stage: %w", err)
	}

	// 清除旧日志
	im.ClearExecutionLogs(taskID)

	// 初始化新进度
	now := time.Now()
	progress := &ExecutionProgress{
		TaskID:          taskID,
		Identifier:      identifier,
		CurrentStage:    StageImplementation,
		Status:          StatusInProgress,
		StartedAt:       &now,
		UpdatedAt:       &now,
		TurnCount:       0,
		RetryCount:      0,
		MaxRetries:      im.config.Execution.MaxRetries,
		ProgressSummary: "准备重试执行...",
	}

	im.mu.Lock()
	im.execProgress[taskID] = progress
	im.execLogs[taskID] = []ExecutionLog{}
	im.mu.Unlock()

	im.appendLog(taskID, "implementation_retry", "开始重试实现阶段", nil)

	return progress, nil
}

// 辅助函数

func joinContextParts(parts []string) string {
	result := ""
	for i, part := range parts {
		if part == "" {
			continue
		}
		if i > 0 {
			result += "\n\n"
		}
		result += part
	}
	return result
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsInString(s, substr))
}

func containsInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func replaceAll(s, old, new string) string {
	result := ""
	for {
		idx := indexOf(s, old)
		if idx == -1 {
			return result + s
		}
		result += s[:idx] + new
		s = s[idx+len(old):]
	}
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func formatEventMessage(event string, data map[string]interface{}) string {
	switch event {
	case "session_started":
		return "AI Agent 会话已启动"
	case "session_ended":
		return "AI Agent 会话已结束"
	case "turn_started":
		return "开始执行新 Turn"
	case "turn_complete":
		return "Turn 执行完成"
	case "turn_failed":
		if errMsg, ok := data["error"].(string); ok {
			return "Turn 执行失败: " + errMsg
		}
		return "Turn 执行失败"
	case "item/message":
		return "收到消息"
	case "item/tool/call":
		if tool, ok := data["tool_name"].(string); ok {
			return "调用工具: " + tool
		}
		return "调用工具"
	case "item/tool/result":
		return "工具执行完成"
	case "error":
		if errMsg, ok := data["error"].(string); ok {
			return "错误: " + errMsg
		}
		return "发生错误"
	default:
		return event
	}
}