// Package workflow 提供工作流阶段管理功能
package workflow

import (
	"fmt"
	"sync"
	"time"
)

// StageName 阶段名称
type StageName string

const (
	// StageClarification 需求澄清阶段
	StageClarification StageName = "clarification"
	// StageBDDReview BDD评审阶段
	StageBDDReview StageName = "bdd_review"
	// StageArchitectureReview 架构评审阶段
	StageArchitectureReview StageName = "architecture_review"
	// StageImplementation 实现阶段
	StageImplementation StageName = "implementation"
	// StageVerification 验证阶段
	StageVerification StageName = "verification"
	// StageNeedsAttention 待人工处理阶段
	StageNeedsAttention StageName = "needs_attention"
	// StageCancelled 已取消阶段
	StageCancelled StageName = "cancelled"
)

// StageStatus 阶段状态
type StageStatus string

const (
	// StatusPending 待开始
	StatusPending StageStatus = "pending"
	// StatusInProgress 进行中
	StatusInProgress StageStatus = "in_progress"
	// StatusCompleted 已完成
	StatusCompleted StageStatus = "completed"
	// StatusFailed 失败
	StatusFailed StageStatus = "failed"
)

// StageOrder 定义阶段顺序
var StageOrder = []StageName{
	StageClarification,
	StageBDDReview,
	StageArchitectureReview,
	StageImplementation,
	StageVerification,
}

// TerminalStages 终态阶段列表
var TerminalStages = []StageName{
	StageNeedsAttention,
	StageCancelled,
}

// StageState 阶段状态
type StageState struct {
	// Name 阶段名称
	Name StageName `json:"name"`
	// Status 状态
	Status StageStatus `json:"status"`
	// StartedAt 开始时间
	StartedAt *time.Time `json:"started_at,omitempty"`
	// UpdatedAt 更新时间
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	// CompletedAt 完成时间
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	// Round 澄清轮次（仅用于clarification阶段）
	Round int `json:"round,omitempty"`
	// Error 失败时的错误信息
	Error string `json:"error,omitempty"`
	// RetryCount 重试次数
	RetryCount int `json:"retry_count,omitempty"`
	// FailedAt 失败时间
	FailedAt *time.Time `json:"failed_at,omitempty"`
	// ErrorType 错误类型
	ErrorType string `json:"error_type,omitempty"`
	// ErrorMessage 错误消息
	ErrorMessage string `json:"error_message,omitempty"`
	// LastLogSnippet 最后的日志片段
	LastLogSnippet string `json:"last_log_snippet,omitempty"`
	// Suggestion 修复建议
	Suggestion string `json:"suggestion,omitempty"`
}

// IsTerminal 判断阶段是否处于终态
func (s *StageState) IsTerminal() bool {
	return s.Status == StatusCompleted || s.Status == StatusFailed
}

// TaskWorkflow 任务工作流状态
type TaskWorkflow struct {
	// TaskID 任务ID
	TaskID string `json:"task_id"`
	// CurrentStage 当前阶段
	CurrentStage StageName `json:"current_stage"`
	// Stages 阶段状态映射
	Stages map[StageName]*StageState `json:"stages"`
	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at"`
	// IsIncomplete 需求是否标记为不完整
	IsIncomplete bool `json:"is_incomplete"`
	// IncompleteReason 不完整原因
	IncompleteReason string `json:"incomplete_reason,omitempty"`
	// NeedsAttention 是否需要人工处理
	NeedsAttention bool `json:"needs_attention"`
	// Metadata 扩展元数据（用于存储 BDD 约束路径等）
	Metadata map[string]string `json:"metadata,omitempty"`
	// RetryCount 当前重试次数
	RetryCount int `json:"retry_count,omitempty"`
	// MaxRetries 最大重试次数
	MaxRetries int `json:"max_retries,omitempty"`
	// FailedStage 失败的阶段
	FailedStage StageName `json:"failed_stage,omitempty"`
	// FailureReason 失败原因
	FailureReason string `json:"failure_reason,omitempty"`
	// FailedAt 失败时间
	FailedAt *time.Time `json:"failed_at,omitempty"`
	// Identifier 任务标识符
	Identifier string `json:"identifier,omitempty"`
	// Title 任务标题
	Title string `json:"title,omitempty"`
}

// GetStage 获取指定阶段的状态
func (tw *TaskWorkflow) GetStage(name StageName) *StageState {
	if tw.Stages == nil {
		return nil
	}
	return tw.Stages[name]
}

// GetAllStages 获取所有阶段状态（按顺序）
func (tw *TaskWorkflow) GetAllStages() []*StageState {
	result := make([]*StageState, 0, len(StageOrder))
	for _, name := range StageOrder {
		if stage, ok := tw.Stages[name]; ok {
			result = append(result, stage)
		}
	}
	return result
}

// GetNextStage 获取下一个阶段
func (tw *TaskWorkflow) GetNextStage() StageName {
	for i, name := range StageOrder {
		if name == tw.CurrentStage && i < len(StageOrder)-1 {
			return StageOrder[i+1]
		}
	}
	return ""
}

// IsComplete 判断工作流是否完成
func (tw *TaskWorkflow) IsComplete() bool {
	lastStage := tw.Stages[StageVerification]
	return lastStage != nil && lastStage.Status == StatusCompleted
}

// IsFailed 判断工作流是否失败
func (tw *TaskWorkflow) IsFailed() bool {
	for _, stage := range tw.Stages {
		if stage.Status == StatusFailed {
			return true
		}
	}
	return false
}

// GetFailedStage 获取失败的阶段
func (tw *TaskWorkflow) GetFailedStage() *StageState {
	for _, stage := range tw.Stages {
		if stage.Status == StatusFailed {
			return stage
		}
	}
	return nil
}

// WorkflowEngine 工作流引擎
type WorkflowEngine struct {
	mu      sync.RWMutex
	workers map[string]*TaskWorkflow // taskID -> TaskWorkflow
}

// NewWorkflowEngine 创建新的工作流引擎
func NewWorkflowEngine() *WorkflowEngine {
	return &WorkflowEngine{
		workers: make(map[string]*TaskWorkflow),
	}
}

// InitWorkflow 初始化任务工作流
func (e *WorkflowEngine) InitWorkflow(taskID string) *TaskWorkflow {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 检查是否已存在
	if existing, ok := e.workers[taskID]; ok {
		return existing
	}

	now := time.Now()
	workflow := &TaskWorkflow{
		TaskID:       taskID,
		CurrentStage: StageClarification,
		Stages:       make(map[StageName]*StageState),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// 初始化所有阶段为待开始状态
	for _, name := range StageOrder {
		workflow.Stages[name] = &StageState{
			Name:   name,
			Status: StatusPending,
		}
	}

	// 设置第一个阶段为进行中
	workflow.Stages[StageClarification].Status = StatusInProgress
	workflow.Stages[StageClarification].StartedAt = &now
	workflow.Stages[StageClarification].UpdatedAt = &now

	e.workers[taskID] = workflow
	return workflow
}

// GetWorkflow 获取任务工作流
func (e *WorkflowEngine) GetWorkflow(taskID string) *TaskWorkflow {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.workers[taskID]
}

// AdvanceStage 推进到下一阶段
func (e *WorkflowEngine) AdvanceStage(taskID string) (*TaskWorkflow, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workers[taskID]
	if !ok {
		return nil, ErrWorkflowNotFound
	}

	currentStage := workflow.Stages[workflow.CurrentStage]
	if currentStage == nil {
		return nil, ErrInvalidStage
	}

	// 当前阶段必须在进行中或已完成状态
	if currentStage.Status != StatusInProgress && currentStage.Status != StatusCompleted {
		return nil, fmt.Errorf("%w: current stage status is %s", ErrInvalidTransition, currentStage.Status)
	}

	now := time.Now()

	// 如果当前阶段还在进行中，标记为完成
	if currentStage.Status == StatusInProgress {
		currentStage.Status = StatusCompleted
		currentStage.CompletedAt = &now
		currentStage.UpdatedAt = &now
	}

	// 获取下一阶段
	nextStageName := workflow.GetNextStage()
	if nextStageName == "" {
		// 已经是最后一个阶段
		return workflow, nil
	}

	// 更新下一阶段状态
	nextStage := workflow.Stages[nextStageName]
	if nextStage != nil {
		nextStage.Status = StatusInProgress
		nextStage.StartedAt = &now
		nextStage.UpdatedAt = &now
	}

	// 更新当前阶段
	workflow.CurrentStage = nextStageName
	workflow.UpdatedAt = now

	return workflow, nil
}

// FailStage 标记当前阶段为失败
func (e *WorkflowEngine) FailStage(taskID string, reason string) (*TaskWorkflow, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workers[taskID]
	if !ok {
		return nil, ErrWorkflowNotFound
	}

	currentStage := workflow.Stages[workflow.CurrentStage]
	if currentStage == nil {
		return nil, ErrInvalidStage
	}

	now := time.Now()

	// 标记为失败
	currentStage.Status = StatusFailed
	currentStage.Error = reason
	currentStage.UpdatedAt = &now

	workflow.UpdatedAt = now

	return workflow, nil
}

// CompleteStage 完成当前阶段并推进到下一阶段
func (e *WorkflowEngine) CompleteStage(taskID string) (*TaskWorkflow, error) {
	return e.AdvanceStage(taskID)
}

// GetCurrentStage 获取当前阶段状态
func (e *WorkflowEngine) GetCurrentStage(taskID string) (*StageState, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	workflow, ok := e.workers[taskID]
	if !ok {
		return nil, ErrWorkflowNotFound
	}

	return workflow.Stages[workflow.CurrentStage], nil
}

// GetStageStatus 获取指定阶段状态
func (e *WorkflowEngine) GetStageStatus(taskID string, stageName StageName) (*StageState, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	workflow, ok := e.workers[taskID]
	if !ok {
		return nil, ErrWorkflowNotFound
	}

	stageState := workflow.Stages[stageName]
	if stageState == nil {
		return nil, ErrInvalidStage
	}

	return stageState, nil
}

// SetStageRound 设置阶段轮次（用于澄清阶段的多轮对话）
func (e *WorkflowEngine) SetStageRound(taskID string, round int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workers[taskID]
	if !ok {
		return ErrWorkflowNotFound
	}

	currentStage := workflow.Stages[workflow.CurrentStage]
	if currentStage == nil {
		return ErrInvalidStage
	}

	now := time.Now()
	currentStage.Round = round
	currentStage.UpdatedAt = &now
	workflow.UpdatedAt = now

	return nil
}

// IncrementStageRound 增加阶段轮次
func (e *WorkflowEngine) IncrementStageRound(taskID string) (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workers[taskID]
	if !ok {
		return 0, ErrWorkflowNotFound
	}

	currentStage := workflow.Stages[workflow.CurrentStage]
	if currentStage == nil {
		return 0, ErrInvalidStage
	}

	now := time.Now()
	currentStage.Round++
	currentStage.UpdatedAt = &now
	workflow.UpdatedAt = now

	return currentStage.Round, nil
}

// ResetStage 重置阶段状态（用于重试）
func (e *WorkflowEngine) ResetStage(taskID string, stage StageName) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workers[taskID]
	if !ok {
		return ErrWorkflowNotFound
	}

	stageState := workflow.Stages[stage]
	if stageState == nil {
		return ErrInvalidStage
	}

	now := time.Now()
	stageState.Status = StatusPending
	stageState.StartedAt = nil
	stageState.CompletedAt = nil
	stageState.Error = ""
	stageState.Round = 0
	stageState.UpdatedAt = &now

	// 如果重置的是当前阶段，设置为进行中
	if stage == workflow.CurrentStage {
		stageState.Status = StatusInProgress
		stageState.StartedAt = &now
	}

	workflow.UpdatedAt = now

	return nil
}

// RemoveWorkflow 移除工作流
func (e *WorkflowEngine) RemoveWorkflow(taskID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.workers, taskID)
}

// GetAllWorkflows 获取所有工作流
func (e *WorkflowEngine) GetAllWorkflows() map[string]*TaskWorkflow {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make(map[string]*TaskWorkflow, len(e.workers))
	for k, v := range e.workers {
		result[k] = v
	}
	return result
}

// GetStageName 从字符串解析阶段名称
func GetStageName(s string) (StageName, bool) {
	stage := StageName(s)
	for _, name := range StageOrder {
		if name == stage {
			return stage, true
		}
	}
	return "", false
}

// MustGetStageName 从字符串解析阶段名称，失败时panic
func MustGetStageName(s string) StageName {
	stage, ok := GetStageName(s)
	if !ok {
		panic(fmt.Sprintf("invalid stage name: %s", s))
	}
	return stage
}

// GetStageDisplayName 获取阶段的显示名称
func GetStageDisplayName(stage StageName) string {
	switch stage {
	case StageClarification:
		return "需求澄清"
	case StageBDDReview:
		return "BDD评审"
	case StageArchitectureReview:
		return "架构评审"
	case StageImplementation:
		return "实现"
	case StageVerification:
		return "验证"
	case StageNeedsAttention:
		return "待人工处理"
	case StageCancelled:
		return "已取消"
	default:
		return string(stage)
	}
}

// GetStatusDisplayName 获取状态的显示名称
func GetStatusDisplayName(status StageStatus) string {
	switch status {
	case StatusPending:
		return "待开始"
	case StatusInProgress:
		return "进行中"
	case StatusCompleted:
		return "已完成"
	case StatusFailed:
		return "失败"
	default:
		return string(status)
	}
}