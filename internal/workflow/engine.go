// Package workflow 提供工作流引擎功能
package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Engine 工作流引擎，管理任务状态流转
type Engine struct {
	mu          sync.RWMutex
	workflows   map[string]*TaskWorkflow // taskID -> TaskWorkflow
	persistPath string                    // 持久化路径（可选）
}

// EngineOption 引擎选项
type EngineOption func(*Engine)

// WithPersistPath 设置持久化路径
func WithPersistPath(path string) EngineOption {
	return func(e *Engine) {
		e.persistPath = path
	}
}

// NewEngine 创建新的工作流引擎
func NewEngine(opts ...EngineOption) *Engine {
	engine := &Engine{
		workflows: make(map[string]*TaskWorkflow),
	}

	for _, opt := range opts {
		opt(engine)
	}

	// 尝试加载已保存的工作流
	if engine.persistPath != "" {
		if err := engine.load(); err != nil {
			// 忽略加载错误，使用空的引擎
			fmt.Printf("warning: failed to load workflows: %v\n", err)
		}
	}

	return engine
}

// InitTask 初始化任务工作流
// 创建新任务时调用，设置第一个阶段为"进行中"
func (e *Engine) InitTask(taskID string) (*TaskWorkflow, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 检查是否已存在
	if _, exists := e.workflows[taskID]; exists {
		return nil, fmt.Errorf("workflow already exists for task: %s", taskID)
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

	// 设置第一个阶段（clarification）为进行中
	workflow.Stages[StageClarification].Status = StatusInProgress
	workflow.Stages[StageClarification].StartedAt = &now
	workflow.Stages[StageClarification].UpdatedAt = &now

	e.workflows[taskID] = workflow

	// 持久化
	if err := e.persist(); err != nil {
		fmt.Printf("warning: failed to persist workflow: %v\n", err)
	}

	return workflow, nil
}

// AdvanceStage 推进任务到下一阶段
// 当当前阶段成功完成时调用
func (e *Engine) AdvanceStage(taskID string) (*TaskWorkflow, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return nil, ErrWorkflowNotFound
	}

	currentStage := workflow.Stages[workflow.CurrentStage]
	if currentStage == nil {
		return nil, ErrInvalidStage
	}

	// 检查当前状态
	if currentStage.Status == StatusFailed {
		return nil, fmt.Errorf("%w: cannot advance from failed stage", ErrInvalidTransition)
	}

	now := time.Now()

	// 标记当前阶段为完成
	if currentStage.Status != StatusCompleted {
		currentStage.Status = StatusCompleted
		currentStage.CompletedAt = &now
		currentStage.UpdatedAt = &now
	}

	// 查找下一阶段
	nextStageName := ""
	currentIdx := -1
	for i, name := range StageOrder {
		if name == workflow.CurrentStage {
			currentIdx = i
			break
		}
	}

	if currentIdx >= 0 && currentIdx < len(StageOrder)-1 {
		nextStageName = string(StageOrder[currentIdx+1])
	}

	if nextStageName == "" {
		// 已经是最后一个阶段，工作流完成
		workflow.UpdatedAt = now
		return workflow, nil
	}

	// 更新下一阶段状态
	nextStage := workflow.Stages[StageName(nextStageName)]
	if nextStage != nil {
		nextStage.Status = StatusInProgress
		nextStage.StartedAt = &now
		nextStage.UpdatedAt = &now
	}

	// 更新当前阶段指针
	workflow.CurrentStage = StageName(nextStageName)
	workflow.UpdatedAt = now

	// 持久化
	if err := e.persist(); err != nil {
		fmt.Printf("warning: failed to persist workflow: %v\n", err)
	}

	return workflow, nil
}

// FailStage 标记阶段失败
// 当阶段执行失败时调用
func (e *Engine) FailStage(taskID string, reason string) (*TaskWorkflow, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workflows[taskID]
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

	// 持久化
	if err := e.persist(); err != nil {
		fmt.Printf("warning: failed to persist workflow: %v\n", err)
	}

	return workflow, nil
}

// GetWorkflow 获取任务工作流
func (e *Engine) GetWorkflow(taskID string) *TaskWorkflow {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.workflows[taskID]
}

// GetCurrentStage 获取当前阶段
func (e *Engine) GetCurrentStage(taskID string) (*StageState, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return nil, ErrWorkflowNotFound
	}

	return workflow.Stages[workflow.CurrentStage], nil
}

// SetStageStatus 设置指定阶段状态
func (e *Engine) SetStageStatus(taskID string, stageName StageName, status StageStatus) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return ErrWorkflowNotFound
	}

	stage, ok := workflow.Stages[stageName]
	if !ok {
		return ErrInvalidStage
	}

	now := time.Now()

	// 更新状态
	switch status {
	case StatusInProgress:
		if stage.StartedAt == nil {
			stage.StartedAt = &now
		}
	case StatusCompleted:
		stage.CompletedAt = &now
	case StatusFailed:
		// 失败状态不需要额外处理时间
	}

	stage.Status = status
	stage.UpdatedAt = &now
	workflow.UpdatedAt = now

	// 持久化
	if err := e.persist(); err != nil {
		fmt.Printf("warning: failed to persist workflow: %v\n", err)
	}

	return nil
}

// IncrementRound 增加当前阶段轮次
func (e *Engine) IncrementRound(taskID string) (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return 0, ErrWorkflowNotFound
	}

	stage := workflow.Stages[workflow.CurrentStage]
	if stage == nil {
		return 0, ErrInvalidStage
	}

	now := time.Now()
	stage.Round++
	stage.UpdatedAt = &now
	workflow.UpdatedAt = now

	// 持久化
	if err := e.persist(); err != nil {
		fmt.Printf("warning: failed to persist workflow: %v\n", err)
	}

	return stage.Round, nil
}

// SetRound 设置当前阶段轮次
func (e *Engine) SetRound(taskID string, round int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return ErrWorkflowNotFound
	}

	stage := workflow.Stages[workflow.CurrentStage]
	if stage == nil {
		return ErrInvalidStage
	}

	now := time.Now()
	stage.Round = round
	stage.UpdatedAt = &now
	workflow.UpdatedAt = now

	// 持久化
	if err := e.persist(); err != nil {
		fmt.Printf("warning: failed to persist workflow: %v\n", err)
	}

	return nil
}

// GetStageHistory 获取阶段历史
func (e *Engine) GetStageHistory(taskID string) ([]*StageState, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return nil, ErrWorkflowNotFound
	}

	history := make([]*StageState, 0, len(StageOrder))
	for _, name := range StageOrder {
		if stage, ok := workflow.Stages[name]; ok {
			history = append(history, stage)
		}
	}

	return history, nil
}

// RemoveTask 移除任务工作流
func (e *Engine) RemoveTask(taskID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.workflows, taskID)

	// 持久化
	if err := e.persist(); err != nil {
		fmt.Printf("warning: failed to persist workflow: %v\n", err)
	}
}

// ListWorkflows 列出所有工作流
func (e *Engine) ListWorkflows() []*TaskWorkflow {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*TaskWorkflow, 0, len(e.workflows))
	for _, wf := range e.workflows {
		result = append(result, wf)
	}
	return result
}

// ListActiveWorkflows 列出所有活跃的工作流（未完成且未失败）
func (e *Engine) ListActiveWorkflows() []*TaskWorkflow {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*TaskWorkflow, 0)
	for _, wf := range e.workflows {
		if !wf.IsComplete() && !wf.IsFailed() {
			result = append(result, wf)
		}
	}
	return result
}

// ListFailedWorkflows 列出所有失败的工作流
func (e *Engine) ListFailedWorkflows() []*TaskWorkflow {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*TaskWorkflow, 0)
	for _, wf := range e.workflows {
		if wf.IsFailed() {
			result = append(result, wf)
		}
	}
	return result
}

// ListCompletedWorkflows 列出所有完成的工作流
func (e *Engine) ListCompletedWorkflows() []*TaskWorkflow {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*TaskWorkflow, 0)
	for _, wf := range e.workflows {
		if wf.IsComplete() {
			result = append(result, wf)
		}
	}
	return result
}

// GetWorkflowStats 获取工作流统计信息
func (e *Engine) GetWorkflowStats() map[string]int {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := map[string]int{
		"total":     len(e.workflows),
		"active":    0,
		"completed": 0,
		"failed":    0,
	}

	for _, wf := range e.workflows {
		if wf.IsComplete() {
			stats["completed"]++
		} else if wf.IsFailed() {
			stats["failed"]++
		} else {
			stats["active"]++
		}
	}

	return stats
}

// persist 持久化工作流状态
func (e *Engine) persist() error {
	if e.persistPath == "" {
		return nil
	}

	// 确保目录存在
	dir := filepath.Dir(e.persistPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(e.workflows, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal workflows: %w", err)
	}

	if err := os.WriteFile(e.persistPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write workflows file: %w", err)
	}

	return nil
}

// load 加载已保存的工作流状态
func (e *Engine) load() error {
	if e.persistPath == "" {
		return nil
	}

	data, err := os.ReadFile(e.persistPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在是正常情况
		}
		return fmt.Errorf("failed to read workflows file: %w", err)
	}

	if len(data) == 0 {
		return nil
	}

	if err := json.Unmarshal(data, &e.workflows); err != nil {
		return fmt.Errorf("failed to unmarshal workflows: %w", err)
	}

	return nil
}

// RecoverTask 恢复任务工作流（用于崩溃恢复）
func (e *Engine) RecoverTask(taskID string, stage StageName, status StageStatus) (*TaskWorkflow, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 检查是否已存在
	if _, exists := e.workflows[taskID]; exists {
		return nil, fmt.Errorf("workflow already exists for task: %s", taskID)
	}

	now := time.Now()
	workflow := &TaskWorkflow{
		TaskID:       taskID,
		CurrentStage: stage,
		Stages:       make(map[StageName]*StageState),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// 初始化所有阶段
	for _, name := range StageOrder {
		workflow.Stages[name] = &StageState{
			Name:   name,
			Status: StatusPending,
		}
	}

	// 设置阶段状态
	currentIdx := -1
	for i, name := range StageOrder {
		if name == stage {
			currentIdx = i
			break
		}
	}

	// 标记之前的阶段为已完成
	for i := 0; i < currentIdx; i++ {
		workflow.Stages[StageOrder[i]].Status = StatusCompleted
		workflow.Stages[StageOrder[i]].UpdatedAt = &now
	}

	// 设置当前阶段状态
	if currentStage := workflow.Stages[stage]; currentStage != nil {
		currentStage.Status = status
		currentStage.StartedAt = &now
		currentStage.UpdatedAt = &now
	}

	e.workflows[taskID] = workflow

	// 持久化
	if err := e.persist(); err != nil {
		fmt.Printf("warning: failed to persist workflow: %v\n", err)
	}

	return workflow, nil
}

// SetIncompleteMark 设置不完整标记
// 当澄清轮次达到上限或用户跳过时调用
func (e *Engine) SetIncompleteMark(taskID string, reason interface{}, needsAttention bool) (*TaskWorkflow, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return nil, ErrWorkflowNotFound
	}

	currentStage := workflow.Stages[workflow.CurrentStage]
	if currentStage == nil {
		return nil, ErrInvalidStage
	}

	now := time.Now()

	// 设置不完整标记
	workflow.IsIncomplete = true

	// 转换 reason 为字符串
	reasonStr := ""
	switch r := reason.(type) {
	case string:
		reasonStr = r
	case IncompleteReason:
		reasonStr = string(r)
	default:
		reasonStr = fmt.Sprintf("%v", r)
	}
	workflow.IncompleteReason = reasonStr

	// 设置 needs_attention 标记
	workflow.NeedsAttention = needsAttention

	// 标记当前阶段完成
	if needsAttention {
		// 达到上限：标记失败，不推进
		currentStage.Status = StatusFailed
		currentStage.Error = reasonStr
	} else {
		// 用户跳过：标记完成，推进到下一阶段
		currentStage.Status = StatusCompleted
		currentStage.CompletedAt = &now
	}
	currentStage.UpdatedAt = &now

	// 如果不是 needs_attention，推进到下一阶段
	if !needsAttention {
		// 查找下一阶段
		nextStageName := ""
		currentIdx := -1
		for i, name := range StageOrder {
			if name == workflow.CurrentStage {
				currentIdx = i
				break
			}
		}

		if currentIdx >= 0 && currentIdx < len(StageOrder)-1 {
			nextStageName = string(StageOrder[currentIdx+1])
		}

		if nextStageName != "" {
			// 更新下一阶段状态
			nextStage := workflow.Stages[StageName(nextStageName)]
			if nextStage != nil {
				nextStage.Status = StatusInProgress
				nextStage.StartedAt = &now
				nextStage.UpdatedAt = &now
			}

			// 更新当前阶段指针
			workflow.CurrentStage = StageName(nextStageName)
		}
	}

	workflow.UpdatedAt = now

	// 持久化
	if err := e.persist(); err != nil {
		fmt.Printf("warning: failed to persist workflow: %v\n", err)
	}

	return workflow, nil
}

// UpdateStageTime 更新阶段时间戳
func (e *Engine) UpdateStageTime(taskID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return ErrWorkflowNotFound
	}

	stage := workflow.Stages[workflow.CurrentStage]
	if stage == nil {
		return ErrInvalidStage
	}

	now := time.Now()
	stage.UpdatedAt = &now
	workflow.UpdatedAt = now

	// 持久化
	if err := e.persist(); err != nil {
		fmt.Printf("warning: failed to persist workflow: %v\n", err)
	}

	return nil
}

// IsNeedsAttention 判断任务是否需要人工处理
func (e *Engine) IsNeedsAttention(taskID string) (bool, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return false, ErrWorkflowNotFound
	}

	return workflow.NeedsAttention, nil
}

// IsIncomplete 判断任务是否标记为不完整
func (e *Engine) IsIncomplete(taskID string) (bool, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return false, ErrWorkflowNotFound
	}

	return workflow.IsIncomplete, nil
}

// GetIncompleteReason 获取不完整原因
func (e *Engine) GetIncompleteReason(taskID string) (string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return "", ErrWorkflowNotFound
	}

	return workflow.IncompleteReason, nil
}

// ApproveBDD 通过 BDD 规则审核
// 状态流转: bdd_review (pending/in_progress) -> architecture_review (pending)
// BDD 文件标记为 approved
func (e *Engine) ApproveBDD(taskID string) (*TaskWorkflow, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return nil, ErrWorkflowNotFound
	}

	// 检查当前阶段是否为 BDD 审核阶段
	if workflow.CurrentStage != StageBDDReview {
		return nil, fmt.Errorf("%w: current stage is %s, not bdd_review", ErrInvalidStage, workflow.CurrentStage)
	}

	bddStage := workflow.Stages[StageBDDReview]
	if bddStage == nil {
		return nil, ErrInvalidStage
	}

	// 检查 BDD 阶段状态（必须是 pending 或 in_progress）
	if bddStage.Status != StatusPending && bddStage.Status != StatusInProgress {
		return nil, fmt.Errorf("%w: bdd_review stage status is %s", ErrInvalidTransition, bddStage.Status)
	}

	now := time.Now()

	// 标记 BDD 阶段为完成
	bddStage.Status = StatusCompleted
	bddStage.CompletedAt = &now
	bddStage.UpdatedAt = &now

	// 推进到架构审核阶段
	nextStage := workflow.Stages[StageArchitectureReview]
	if nextStage != nil {
		nextStage.Status = StatusInProgress
		nextStage.StartedAt = &now
		nextStage.UpdatedAt = &now
	}

	// 更新当前阶段指针
	workflow.CurrentStage = StageArchitectureReview
	workflow.UpdatedAt = now

	// 持久化
	if err := e.persist(); err != nil {
		fmt.Printf("warning: failed to persist workflow: %v\n", err)
	}

	return workflow, nil
}

// RejectBDD 驳回 BDD 规则审核
// 状态流转: bdd_review (pending/in_progress) -> clarification (in_progress)
// 记录驳回原因，触发重新澄清
func (e *Engine) RejectBDD(taskID string, reason string) (*TaskWorkflow, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return nil, ErrWorkflowNotFound
	}

	// 检查当前阶段是否为 BDD 审核阶段
	if workflow.CurrentStage != StageBDDReview {
		return nil, fmt.Errorf("%w: current stage is %s, not bdd_review", ErrInvalidStage, workflow.CurrentStage)
	}

	bddStage := workflow.Stages[StageBDDReview]
	if bddStage == nil {
		return nil, ErrInvalidStage
	}

	// 检查 BDD 阶段状态（必须是 pending 或 in_progress）
	if bddStage.Status != StatusPending && bddStage.Status != StatusInProgress {
		return nil, fmt.Errorf("%w: bdd_review stage status is %s", ErrInvalidTransition, bddStage.Status)
	}

	now := time.Now()

	// 标记 BDD 阶段为失败，记录驳回原因
	bddStage.Status = StatusFailed
	bddStage.Error = reason
	bddStage.UpdatedAt = &now

	// 回退到澄清阶段进行中状态
	clarificationStage := workflow.Stages[StageClarification]
	if clarificationStage != nil {
		clarificationStage.Status = StatusInProgress
		clarificationStage.UpdatedAt = &now
		// 重置轮次，准备重新澄清
		clarificationStage.Round = 0
	}

	// 更新当前阶段指针回澄清阶段
	workflow.CurrentStage = StageClarification
	workflow.UpdatedAt = now

	// 持久化
	if err := e.persist(); err != nil {
		fmt.Printf("warning: failed to persist workflow: %v\n", err)
	}

	return workflow, nil
}

// AdvanceToImplementation 推进到实现阶段，同时加载 BDD 约束
// 这是一个特殊的方法，用于在从架构评审阶段进入实现阶段时，
// 确保 BDD 约束条件已正确加载并可以传递给 Agent
// 返回工作流状态和 BDD 约束文件路径
func (e *Engine) AdvanceToImplementation(taskID string, constraintManager *ConstraintManager) (*TaskWorkflow, string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return nil, "", ErrWorkflowNotFound
	}

	currentStage := workflow.Stages[workflow.CurrentStage]
	if currentStage == nil {
		return nil, "", ErrInvalidStage
	}

	// 检查当前状态
	if currentStage.Status == StatusFailed {
		return nil, "", fmt.Errorf("%w: cannot advance from failed stage", ErrInvalidTransition)
	}

	// 当前阶段必须是架构评审或已完成的架构评审
	if workflow.CurrentStage != StageArchitectureReview {
		return nil, "", fmt.Errorf("%w: must be in architecture_review stage to advance to implementation", ErrInvalidTransition)
	}

	now := time.Now()

	// 标记架构评审阶段为完成
	if currentStage.Status != StatusCompleted {
		currentStage.Status = StatusCompleted
		currentStage.CompletedAt = &now
		currentStage.UpdatedAt = &now
	}

	// 更新实现阶段状态
	implStage := workflow.Stages[StageImplementation]
	if implStage != nil {
		implStage.Status = StatusInProgress
		implStage.StartedAt = &now
		implStage.UpdatedAt = &now
	}

	// 更新当前阶段指针
	workflow.CurrentStage = StageImplementation
	workflow.UpdatedAt = now

	// 加载 BDD 约束文件路径（在不持有锁的情况下）
	bddFilePath := ""
	if constraintManager != nil {
		// 直接从缓存或文件中获取
		bddFilePath = constraintManager.GetConstraintFilePathUnlocked(taskID)
	}

	// 存储约束文件路径到 Metadata
	if bddFilePath != "" {
		if workflow.Metadata == nil {
			workflow.Metadata = make(map[string]string)
		}
		workflow.Metadata["bdd_constraints_path"] = bddFilePath
	}

	// 持久化
	if err := e.persist(); err != nil {
		fmt.Printf("warning: failed to persist workflow: %v\n", err)
	}

	return workflow, bddFilePath, nil
}

// SetBDDConstraintsPath 设置任务的 BDD 约束文件路径
// 用于在工作流状态中记录约束文件位置
func (e *Engine) SetBDDConstraintsPath(taskID string, path string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return ErrWorkflowNotFound
	}

	now := time.Now()

	// 存储到 workflow 的 metadata
	if workflow.Metadata == nil {
		workflow.Metadata = make(map[string]string)
	}
	workflow.Metadata["bdd_constraints_path"] = path
	workflow.UpdatedAt = now

	// 持久化
	if err := e.persist(); err != nil {
		fmt.Printf("warning: failed to persist workflow: %v\n", err)
	}

	return nil
}

// GetBDDConstraintsPath 获取任务的 BDD 约束文件路径
func (e *Engine) GetBDDConstraintsPath(taskID string) (string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return "", ErrWorkflowNotFound
	}

	if workflow.Metadata == nil {
		return "", nil
	}

	return workflow.Metadata["bdd_constraints_path"], nil
}

// GetIdentifierFromWorkflow 获取工作流关联的任务标识符
// 通过 Metadata 中的 identifier 字段获取
func (e *Engine) GetIdentifierFromWorkflow(taskID string) string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return ""
	}

	if workflow.Metadata == nil {
		return ""
	}

	return workflow.Metadata["identifier"]
}

// SetIdentifierForWorkflow 设置工作流关联的任务标识符
func (e *Engine) SetIdentifierForWorkflow(taskID string, identifier string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return ErrWorkflowNotFound
	}

	now := time.Now()

	if workflow.Metadata == nil {
		workflow.Metadata = make(map[string]string)
	}
	workflow.Metadata["identifier"] = identifier
	workflow.UpdatedAt = now

	// 持久化
	if err := e.persist(); err != nil {
		fmt.Printf("warning: failed to persist workflow: %v\n", err)
	}

	return nil
}

// ResetStage 重置阶段状态（用于重试）
// 将指定阶段重置为待开始状态，清除错误和时间戳
func (e *Engine) ResetStage(taskID string, stage StageName) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return ErrWorkflowNotFound
	}

	stageState, ok := workflow.Stages[stage]
	if !ok {
		return ErrInvalidStage
	}

	now := time.Now()

	// 重置阶段状态
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

	// 持久化
	if err := e.persist(); err != nil {
		fmt.Printf("warning: failed to persist workflow: %v\n", err)
	}

	return nil
}

// ApproveVerification 通过验收
// 状态流转: verification (pending/in_progress) -> completed
// 触发 Git 提交
func (e *Engine) ApproveVerification(taskID string) (*TaskWorkflow, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return nil, ErrWorkflowNotFound
	}

	// 检查当前阶段是否为验收阶段
	if workflow.CurrentStage != StageVerification {
		return nil, fmt.Errorf("%w: current stage is %s, not verification", ErrInvalidStage, workflow.CurrentStage)
	}

	verificationStage := workflow.Stages[StageVerification]
	if verificationStage == nil {
		return nil, ErrInvalidStage
	}

	// 检验收阶段状态（必须是 pending 或 in_progress）
	if verificationStage.Status != StatusPending && verificationStage.Status != StatusInProgress {
		return nil, fmt.Errorf("%w: verification stage status is %s", ErrInvalidTransition, verificationStage.Status)
	}

	now := time.Now()

	// 标记验收阶段为完成
	verificationStage.Status = StatusCompleted
	verificationStage.CompletedAt = &now
	verificationStage.UpdatedAt = &now

	// 更新工作流状态
	workflow.UpdatedAt = now

	// 标记工作流完成（通过设置 Metadata）
	if workflow.Metadata == nil {
		workflow.Metadata = make(map[string]string)
	}
	workflow.Metadata["verification_status"] = "approved"
	workflow.Metadata["completed_at"] = now.Format(time.RFC3339)

	// 持久化
	if err := e.persist(); err != nil {
		fmt.Printf("warning: failed to persist workflow: %v\n", err)
	}

	return workflow, nil
}

// RejectVerification 驳回验收
// 状态流转: verification (pending/in_progress) -> implementation (in_progress)
// 记录驳回原因，触发重新实现
func (e *Engine) RejectVerification(taskID string, reason string) (*TaskWorkflow, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return nil, ErrWorkflowNotFound
	}

	// 检查当前阶段是否为验收阶段
	if workflow.CurrentStage != StageVerification {
		return nil, fmt.Errorf("%w: current stage is %s, not verification", ErrInvalidStage, workflow.CurrentStage)
	}

	verificationStage := workflow.Stages[StageVerification]
	if verificationStage == nil {
		return nil, ErrInvalidStage
	}

	// 检验收阶段状态（必须是 pending 或 in_progress）
	if verificationStage.Status != StatusPending && verificationStage.Status != StatusInProgress {
		return nil, fmt.Errorf("%w: verification stage status is %s", ErrInvalidTransition, verificationStage.Status)
	}

	now := time.Now()

	// 标记验收阶段为失败，记录驳回原因
	verificationStage.Status = StatusFailed
	verificationStage.Error = reason
	verificationStage.UpdatedAt = &now

	// 重置实现阶段状态，准备重新实现
	implementationStage := workflow.Stages[StageImplementation]
	if implementationStage != nil {
		implementationStage.Status = StatusInProgress
		implementationStage.Error = "" // 清除之前的错误
		implementationStage.UpdatedAt = &now
		// 重置重试计数
		implementationStage.StartedAt = &now
	}

	// 更新当前阶段指针回实现阶段
	workflow.CurrentStage = StageImplementation
	workflow.UpdatedAt = now

	// 记录驳回信息到 Metadata
	if workflow.Metadata == nil {
		workflow.Metadata = make(map[string]string)
	}
	workflow.Metadata["verification_status"] = "rejected"
	workflow.Metadata["verification_reject_reason"] = reason
	workflow.Metadata["verification_rejected_at"] = now.Format(time.RFC3339)

	// 持久化
	if err := e.persist(); err != nil {
		fmt.Printf("warning: failed to persist workflow: %v\n", err)
	}

	return workflow, nil
}

// GetVerificationStatus 获取验收状态
func (e *Engine) GetVerificationStatus(taskID string) (*VerificationStatus, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return nil, ErrWorkflowNotFound
	}

	status := &VerificationStatus{
		TaskID:       taskID,
		CurrentStage: workflow.CurrentStage,
		Status:       "",
		CanApprove:   false,
		CanReject:    false,
		Approved:     false,
		Rejected:     false,
	}

	// 检查验收阶段状态
	verificationStage := workflow.Stages[StageVerification]
	if verificationStage != nil {
		status.Status = string(verificationStage.Status)

		// 判断是否可以进行审核操作
		if workflow.CurrentStage == StageVerification &&
			(verificationStage.Status == StatusPending || verificationStage.Status == StatusInProgress) {
			status.CanApprove = true
			status.CanReject = true
		}

		// 判断已审核状态
		if verificationStage.Status == StatusCompleted {
			status.Approved = true
		}
		if verificationStage.Status == StatusFailed {
			status.Rejected = true
			status.RejectReason = verificationStage.Error
		}
	}

	// 从 Metadata 获取额外信息
	if workflow.Metadata != nil {
		if v, ok := workflow.Metadata["verification_status"]; ok {
			if v == "approved" {
				status.Approved = true
			} else if v == "rejected" {
				status.Rejected = true
			}
		}
		if v, ok := workflow.Metadata["verification_reject_reason"]; ok {
			status.RejectReason = v
		}
	}

	return status, nil
}

// CanApproveOrRejectVerification 判断是否可以进行验收审核操作
func (e *Engine) CanApproveOrRejectVerification(taskID string) (bool, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return false, ErrWorkflowNotFound
	}

	// 必须在验收阶段
	if workflow.CurrentStage != StageVerification {
		return false, nil
	}

	verificationStage := workflow.Stages[StageVerification]
	if verificationStage == nil {
		return false, nil
	}

	// 验收阶段必须是 pending 或 in_progress
	return verificationStage.Status == StatusPending || verificationStage.Status == StatusInProgress, nil
}

// VerificationStatus 验收状态
type VerificationStatus struct {
	TaskID       string `json:"task_id"`
	CurrentStage StageName `json:"current_stage"`
	Status       string `json:"status"`
	CanApprove   bool   `json:"can_approve"`
	CanReject    bool   `json:"can_reject"`
	Approved     bool   `json:"approved"`
	Rejected     bool   `json:"rejected"`
	RejectReason string `json:"reject_reason,omitempty"`
}

// ApproveArchitecture 通过架构设计审核
// 状态流转: architecture_review (pending/in_progress) -> implementation (pending)
// 架构设计标记为 approved
func (e *Engine) ApproveArchitecture(taskID string) (*TaskWorkflow, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return nil, ErrWorkflowNotFound
	}

	// 检查当前阶段是否为架构审核阶段
	if workflow.CurrentStage != StageArchitectureReview {
		return nil, fmt.Errorf("%w: current stage is %s, not architecture_review", ErrInvalidStage, workflow.CurrentStage)
	}

	archStage := workflow.Stages[StageArchitectureReview]
	if archStage == nil {
		return nil, ErrInvalidStage
	}

	// 检查架构审核阶段状态（必须是 pending 或 in_progress）
	if archStage.Status != StatusPending && archStage.Status != StatusInProgress {
		return nil, fmt.Errorf("%w: architecture_review stage status is %s", ErrInvalidTransition, archStage.Status)
	}

	now := time.Now()

	// 标记架构审核阶段为完成
	archStage.Status = StatusCompleted
	archStage.CompletedAt = &now
	archStage.UpdatedAt = &now

	// 推进到实现阶段
	implStage := workflow.Stages[StageImplementation]
	if implStage != nil {
		implStage.Status = StatusInProgress
		implStage.StartedAt = &now
		implStage.UpdatedAt = &now
	}

	// 更新当前阶段指针
	workflow.CurrentStage = StageImplementation
	workflow.UpdatedAt = now

	// 标记架构审核通过
	if workflow.Metadata == nil {
		workflow.Metadata = make(map[string]string)
	}
	workflow.Metadata["architecture_status"] = "approved"

	// 持久化
	if err := e.persist(); err != nil {
		fmt.Printf("warning: failed to persist workflow: %v\n", err)
	}

	return workflow, nil
}

// RejectArchitecture 驳回架构设计审核
// 状态流转: architecture_review (pending/in_progress) -> clarification (in_progress)
// 记录驳回原因，触发重新澄清
func (e *Engine) RejectArchitecture(taskID string, reason string) (*TaskWorkflow, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return nil, ErrWorkflowNotFound
	}

	// 检查当前阶段是否为架构审核阶段
	if workflow.CurrentStage != StageArchitectureReview {
		return nil, fmt.Errorf("%w: current stage is %s, not architecture_review", ErrInvalidStage, workflow.CurrentStage)
	}

	archStage := workflow.Stages[StageArchitectureReview]
	if archStage == nil {
		return nil, ErrInvalidStage
	}

	// 检查架构审核阶段状态（必须是 pending 或 in_progress）
	if archStage.Status != StatusPending && archStage.Status != StatusInProgress {
		return nil, fmt.Errorf("%w: architecture_review stage status is %s", ErrInvalidTransition, archStage.Status)
	}

	now := time.Now()

	// 标记架构审核阶段为失败，记录驳回原因
	archStage.Status = StatusFailed
	archStage.Error = reason
	archStage.UpdatedAt = &now

	// 回退到澄清阶段进行中状态
	clarificationStage := workflow.Stages[StageClarification]
	if clarificationStage != nil {
		clarificationStage.Status = StatusInProgress
		clarificationStage.UpdatedAt = &now
		// 重置轮次，准备重新澄清
		clarificationStage.Round = 0
	}

	// 更新当前阶段指针回澄清阶段
	workflow.CurrentStage = StageClarification
	workflow.UpdatedAt = now

	// 标记架构审核驳回
	if workflow.Metadata == nil {
		workflow.Metadata = make(map[string]string)
	}
	workflow.Metadata["architecture_status"] = "rejected"
	workflow.Metadata["architecture_reject_reason"] = reason

	// 持久化
	if err := e.persist(); err != nil {
		fmt.Printf("warning: failed to persist workflow: %v\n", err)
	}

	return workflow, nil
}

// ArchitectureReviewStatus 架构审核状态
type ArchitectureReviewStatus struct {
	TaskID       string    `json:"task_id"`
	CurrentStage StageName `json:"current_stage"`
	Status       string    `json:"status"`
	CanApprove   bool      `json:"can_approve"`
	CanReject    bool      `json:"can_reject"`
	Approved     bool      `json:"approved"`
	Rejected     bool      `json:"rejected"`
	RejectReason string    `json:"reject_reason,omitempty"`
	NeedsAttention bool    `json:"needs_attention"`
}

// GetArchitectureReviewStatus 获取架构审核状态
func (e *Engine) GetArchitectureReviewStatus(taskID string) (*ArchitectureReviewStatus, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return nil, ErrWorkflowNotFound
	}

	status := &ArchitectureReviewStatus{
		TaskID:       taskID,
		CurrentStage: workflow.CurrentStage,
		Status:       "",
		CanApprove:   false,
		CanReject:    false,
		Approved:     false,
		Rejected:     false,
		NeedsAttention: workflow.NeedsAttention,
	}

	// 检查架构审核阶段状态
	archStage := workflow.Stages[StageArchitectureReview]
	if archStage != nil {
		status.Status = string(archStage.Status)

		// 判断是否可以进行审核操作
		if workflow.CurrentStage == StageArchitectureReview &&
			(archStage.Status == StatusPending || archStage.Status == StatusInProgress) {
			status.CanApprove = true
			status.CanReject = true
		}

		// 判断已审核状态
		if archStage.Status == StatusCompleted {
			status.Approved = true
		}
		if archStage.Status == StatusFailed {
			status.Rejected = true
			status.RejectReason = archStage.Error
		}
	}

	// 从 Metadata 获取额外信息
	if workflow.Metadata != nil {
		if v, ok := workflow.Metadata["architecture_status"]; ok {
			if v == "approved" {
				status.Approved = true
			} else if v == "rejected" {
				status.Rejected = true
			}
		}
		if v, ok := workflow.Metadata["architecture_reject_reason"]; ok {
			status.RejectReason = v
		}
	}

	return status, nil
}

// CanApproveOrRejectArchitecture 判断是否可以进行架构审核操作
func (e *Engine) CanApproveOrRejectArchitecture(taskID string) (bool, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return false, ErrWorkflowNotFound
	}

	// 必须在架构审核阶段
	if workflow.CurrentStage != StageArchitectureReview {
		return false, nil
	}

	archStage := workflow.Stages[StageArchitectureReview]
	if archStage == nil {
		return false, nil
	}

	// 架构审核阶段必须是 pending 或 in_progress
	return archStage.Status == StatusPending || archStage.Status == StatusInProgress, nil
}

// SetArchitecturePaths 设置架构设计和TDD规则文件路径
func (e *Engine) SetArchitecturePaths(taskID string, archPath string, tddPath string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return ErrWorkflowNotFound
	}

	now := time.Now()

	if workflow.Metadata == nil {
		workflow.Metadata = make(map[string]string)
	}
	workflow.Metadata["architecture_path"] = archPath
	workflow.Metadata["tdd_path"] = tddPath
	workflow.UpdatedAt = now

	// 持久化
	if err := e.persist(); err != nil {
		fmt.Printf("warning: failed to persist workflow: %v\n", err)
	}

	return nil
}

// GetArchitecturePaths 获取架构设计和TDD规则文件路径
func (e *Engine) GetArchitecturePaths(taskID string) (string, string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return "", "", ErrWorkflowNotFound
	}

	if workflow.Metadata == nil {
		return "", "", nil
	}

	return workflow.Metadata["architecture_path"], workflow.Metadata["tdd_path"], nil
}
// NeedsAttentionDetails 需要人工处理的详细信息
type NeedsAttentionDetails struct {
	FailedStage    string    `json:"failed_stage"`
	FailedAt       time.Time `json:"failed_at"`
	RetryCount     int       `json:"retry_count"`
	MaxRetries     int       `json:"max_retries"`
	ErrorType      string    `json:"error_type"`
	ErrorMessage   string    `json:"error_message"`
	LastLogSnippet string    `json:"last_log_snippet,omitempty"`
	Suggestion     string    `json:"suggestion,omitempty"`
}

// TransitionToNeedsAttention 流转到待人工处理状态
// 当任务执行失败且达到重试上限时调用
func (e *Engine) TransitionToNeedsAttention(taskID string, details NeedsAttentionDetails) (*TaskWorkflow, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return nil, ErrWorkflowNotFound
	}

	now := time.Now()

	// 记录失败阶段
	workflow.FailedStage = StageName(details.FailedStage)
	workflow.FailedAt = &now
	workflow.FailureReason = details.ErrorMessage
	workflow.RetryCount = details.RetryCount
	workflow.MaxRetries = details.MaxRetries
	workflow.NeedsAttention = true

	// 更新失败阶段的状态
	if failedStageName := StageName(details.FailedStage); failedStageName != "" {
		if stage, ok := workflow.Stages[failedStageName]; ok {
			stage.Status = StatusFailed
			stage.Error = details.ErrorMessage
			stage.RetryCount = details.RetryCount
			stage.FailedAt = &now
			stage.ErrorType = details.ErrorType
			stage.ErrorMessage = details.ErrorMessage
			stage.LastLogSnippet = details.LastLogSnippet
			stage.Suggestion = details.Suggestion
			stage.UpdatedAt = &now
		}
	}

	// 流转到待人工处理阶段
	workflow.CurrentStage = StageNeedsAttention
	workflow.UpdatedAt = now

	// 持久化
	if err := e.persist(); err != nil {
		fmt.Printf("warning: failed to persist workflow: %v\n", err)
	}

	return workflow, nil
}

// ResumeTask 恢复任务执行
// 用户手动修复后继续执行
func (e *Engine) ResumeTask(taskID string) (*TaskWorkflow, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return nil, ErrWorkflowNotFound
	}

	// 检查当前是否处于待人工处理状态
	if workflow.CurrentStage != StageNeedsAttention {
		return nil, fmt.Errorf("%w: current stage is %s, not needs_attention", ErrInvalidTransition, workflow.CurrentStage)
	}

	now := time.Now()

	// 清除人工处理标记
	workflow.NeedsAttention = false
	workflow.RetryCount = 0 // 重置重试计数器

	// 恢复到失败的阶段重新执行
	if workflow.FailedStage != "" {
		workflow.CurrentStage = workflow.FailedStage
		if stage, ok := workflow.Stages[workflow.FailedStage]; ok {
			stage.Status = StatusInProgress
			stage.StartedAt = &now
			stage.UpdatedAt = &now
			stage.Error = ""
			stage.ErrorType = ""
			stage.ErrorMessage = ""
			stage.RetryCount = 0
			stage.FailedAt = nil
		}
	}

	workflow.UpdatedAt = now

	// 持久化
	if err := e.persist(); err != nil {
		fmt.Printf("warning: failed to persist workflow: %v\n", err)
	}

	return workflow, nil
}

// ReclarifyTask 重新澄清需求
// 流转到需求澄清阶段，清除 BDD 和架构设计
func (e *Engine) ReclarifyTask(taskID string) (*TaskWorkflow, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return nil, ErrWorkflowNotFound
	}

	// 检查当前是否处于待人工处理状态
	if workflow.CurrentStage != StageNeedsAttention {
		return nil, fmt.Errorf("%w: current stage is %s, not needs_attention", ErrInvalidTransition, workflow.CurrentStage)
	}

	now := time.Now()

	// 清除人工处理标记和失败信息
	workflow.NeedsAttention = false
	workflow.FailedStage = ""
	workflow.FailedAt = nil
	workflow.FailureReason = ""
	workflow.RetryCount = 0
	workflow.IsIncomplete = false
	workflow.IncompleteReason = ""

	// 清除 BDD 约束路径
	if workflow.Metadata != nil {
		delete(workflow.Metadata, "bdd_constraints_path")
	}

	// 重置所有阶段状态
	for _, name := range StageOrder {
		if stage, ok := workflow.Stages[name]; ok {
			stage.Status = StatusPending
			stage.StartedAt = nil
			stage.CompletedAt = nil
			stage.UpdatedAt = &now
			stage.Error = ""
			stage.ErrorType = ""
			stage.ErrorMessage = ""
			stage.RetryCount = 0
			stage.FailedAt = nil
			stage.LastLogSnippet = ""
			stage.Suggestion = ""
			stage.Round = 0
		}
	}

	// 流转到需求澄清阶段
	workflow.CurrentStage = StageClarification
	if stage, ok := workflow.Stages[StageClarification]; ok {
		stage.Status = StatusInProgress
		stage.StartedAt = &now
		stage.UpdatedAt = &now
	}

	workflow.UpdatedAt = now

	// 持久化
	if err := e.persist(); err != nil {
		fmt.Printf("warning: failed to persist workflow: %v\n", err)
	}

	return workflow, nil
}

// AbandonTask 放弃任务
// 流转到已取消状态
func (e *Engine) AbandonTask(taskID string) (*TaskWorkflow, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return nil, ErrWorkflowNotFound
	}

	// 检查当前是否处于待人工处理状态
	if workflow.CurrentStage != StageNeedsAttention {
		return nil, fmt.Errorf("%w: current stage is %s, not needs_attention", ErrInvalidTransition, workflow.CurrentStage)
	}

	now := time.Now()

	// 清除所有状态
	workflow.NeedsAttention = false
	workflow.FailedStage = ""
	workflow.FailedAt = nil
	workflow.FailureReason = ""
	workflow.RetryCount = 0

	// 流转到已取消阶段
	workflow.CurrentStage = StageCancelled
	workflow.UpdatedAt = now

	// 持久化
	if err := e.persist(); err != nil {
		fmt.Printf("warning: failed to persist workflow: %v\n", err)
	}

	return workflow, nil
}

// GetNeedsAttentionTasks 获取所有待人工处理的任务
func (e *Engine) GetNeedsAttentionTasks() []*TaskWorkflow {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*TaskWorkflow, 0)
	for _, wf := range e.workflows {
		if wf.CurrentStage == StageNeedsAttention || wf.NeedsAttention {
			result = append(result, wf)
		}
	}
	return result
}

// GetFailureDetails 获取任务的失败详情
func (e *Engine) GetFailureDetails(taskID string) (*NeedsAttentionDetails, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return nil, ErrWorkflowNotFound
	}

	if !workflow.NeedsAttention && workflow.CurrentStage != StageNeedsAttention {
		return nil, fmt.Errorf("task is not in needs_attention state")
	}

	details := &NeedsAttentionDetails{
		FailedStage:  string(workflow.FailedStage),
		RetryCount:   workflow.RetryCount,
		MaxRetries:   workflow.MaxRetries,
		ErrorMessage: workflow.FailureReason,
	}

	if workflow.FailedAt != nil {
		details.FailedAt = *workflow.FailedAt
	}

	// 从失败阶段获取更多详情
	if workflow.FailedStage != "" {
		if stage, ok := workflow.Stages[workflow.FailedStage]; ok {
			details.ErrorType = stage.ErrorType
			details.LastLogSnippet = stage.LastLogSnippet
			details.Suggestion = stage.Suggestion
		}
	}

	return details, nil
}

// IncrementRetryCount 增加重试计数
func (e *Engine) IncrementRetryCount(taskID string) (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return 0, ErrWorkflowNotFound
	}

	now := time.Now()
	workflow.RetryCount++
	workflow.UpdatedAt = now

	// 更新当前阶段的 retry count
	if stage, ok := workflow.Stages[workflow.CurrentStage]; ok {
		stage.RetryCount = workflow.RetryCount
		stage.UpdatedAt = &now
	}

	// 持久化
	if err := e.persist(); err != nil {
		fmt.Printf("warning: failed to persist workflow: %v\n", err)
	}

	return workflow.RetryCount, nil
}

// SetMaxRetries 设置最大重试次数
func (e *Engine) SetMaxRetries(taskID string, maxRetries int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return ErrWorkflowNotFound
	}

	now := time.Now()
	workflow.MaxRetries = maxRetries
	workflow.UpdatedAt = now

	// 持久化
	if err := e.persist(); err != nil {
		fmt.Printf("warning: failed to persist workflow: %v\n", err)
	}

	return nil
}

// HasReachedMaxRetries 检查是否达到最大重试次数
func (e *Engine) HasReachedMaxRetries(taskID string) (bool, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	workflow, ok := e.workflows[taskID]
	if !ok {
		return false, ErrWorkflowNotFound
	}

	if workflow.MaxRetries <= 0 {
		return false, nil // 无限制
	}

	return workflow.RetryCount >= workflow.MaxRetries, nil
}
