// Package harness 提供 P-G-E 编排引擎
package harness

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// IterationManager 迭代管理器接口
type IterationManager interface {
	// GetIteration 获取当前迭代次数
	GetIteration(taskID string) int

	// IncrementIteration 增加迭代次数
	IncrementIteration(taskID string) int

	// CheckLimit 检查是否达到迭代上限
	CheckLimit(taskID string) bool

	// ShouldContinue 判断是否应该继续迭代
	ShouldContinue(taskID string, evaluatorOutput *EvaluatorOutput) (bool, string)
}

// IterationConfig 迭代配置
type IterationConfig struct {
	// MaxIterations 最大迭代次数
	MaxIterations int `json:"max_iterations"`
}

// DefaultIterationConfig 默认迭代配置
var DefaultIterationConfig = IterationConfig{
	MaxIterations: 5,
}

// IterationManagerImpl 迭代管理器实现
type IterationManagerImpl struct {
	// config 迭代配置
	config IterationConfig
	// mu 互斥锁
	mu sync.Mutex
	// iterations 迭代计数
	iterations map[string]int
	// histories 迭代历史
	histories map[string][]IterationRecord
}

// IterationRecord 迭代记录
type IterationRecord struct {
	// Iteration 迭代次数
	Iteration int `json:"iteration"`
	// FailureReport 失败报告
	FailureReport string `json:"failure_report"`
	// Timestamp 时间戳
	Timestamp time.Time `json:"timestamp"`
	// Passed 是否通过
	Passed bool `json:"passed"`
}

// NewIterationManager 创建新的迭代管理器
func NewIterationManager(config IterationConfig) *IterationManagerImpl {
	return &IterationManagerImpl{
		config:     config,
		iterations: make(map[string]int),
		histories:  make(map[string][]IterationRecord),
	}
}

// GetIteration 获取当前迭代次数
func (m *IterationManagerImpl) GetIteration(taskID string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.iterations[taskID]
}

// IncrementIteration 增加迭代次数
func (m *IterationManagerImpl) IncrementIteration(taskID string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.iterations[taskID]++
	slog.Debug("iteration incremented",
		"task_id", taskID,
		"iteration", m.iterations[taskID],
	)
	return m.iterations[taskID]
}

// CheckLimit 检查是否达到迭代上限
func (m *IterationManagerImpl) CheckLimit(taskID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	current := m.iterations[taskID]
	return current >= m.config.MaxIterations
}

// ShouldContinue 判断是否应该继续迭代
// 返回: (是否继续, 原因)
func (m *IterationManagerImpl) ShouldContinue(taskID string, evaluatorOutput *EvaluatorOutput) (bool, string) {
	m.mu.Lock()
	current := m.iterations[taskID]
	m.mu.Unlock()

	// 如果评估通过，不需要继续
	if evaluatorOutput.Passed {
		return false, "evaluation passed"
	}

	// 检查是否达到上限
	if current >= m.config.MaxIterations {
		return false, fmt.Sprintf("iteration limit reached (%d/%d)", current, m.config.MaxIterations)
	}

	// 可以继续迭代
	return true, "evaluation failed, starting iteration"
}

// RecordIteration 记录迭代历史
func (m *IterationManagerImpl) RecordIteration(taskID string, evaluatorOutput *EvaluatorOutput) {
	m.mu.Lock()
	defer m.mu.Unlock()

	record := IterationRecord{
		Iteration:     evaluatorOutput.Iteration,
		FailureReport: evaluatorOutput.FailureReport,
		Timestamp:     time.Now(),
		Passed:        evaluatorOutput.Passed,
	}

	m.histories[taskID] = append(m.histories[taskID], record)

	slog.Info("iteration recorded",
		"task_id", taskID,
		"iteration", record.Iteration,
		"passed", record.Passed,
	)
}

// GetHistory 获取迭代历史
func (m *IterationManagerImpl) GetHistory(taskID string) []IterationRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.histories[taskID]
}

// GetStatus 获取迭代状态
func (m *IterationManagerImpl) GetStatus(taskID string) IterationStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	current := m.iterations[taskID]
	history := m.histories[taskID]

	return IterationStatus{
		TaskID:        taskID,
		Current:       current,
		Max:           m.config.MaxIterations,
		Remaining:     m.config.MaxIterations - current,
		History:       history,
		NeedsAttention: current >= m.config.MaxIterations,
	}
}

// IterationStatus 迭代状态
type IterationStatus struct {
	// TaskID 任务 ID
	TaskID string `json:"task_id"`
	// Current 当前迭代次数
	Current int `json:"current"`
	// Max 最大迭代次数
	Max int `json:"max"`
	// Remaining 剩余迭代次数
	Remaining int `json:"remaining"`
	// History 迭代历史
	History []IterationRecord `json:"history"`
	// NeedsAttention 是否需要人工处理
	NeedsAttention bool `json:"needs_attention"`
}

// Orchestrator 编排器 (整合 P-G-E)
type Orchestrator struct {
	// planner 规划器
	planner Planner
	// generator 生成器
	generator Generator
	// evaluator 评估器
	evaluator Evaluator
	// iterationManager 迭代管理器
	iterationManager *IterationManagerImpl
	// agentCaller Agent 调用器
	agentCaller AgentCaller
	// mu 互斥锁
	mu sync.Mutex
	// statuses 执行状态
	statuses map[string]*ExecutionStatus
}

// ExecutionStatus 执行状态
type ExecutionStatus struct {
	// TaskID 任务 ID
	TaskID string `json:"task_id"`
	// Phase 当前阶段
	Phase string `json:"phase"` // "planner", "generator", "evaluator"
	// Status 状态
	Status string `json:"status"` // "running", "completed", "failed", "needs_attention"
	// Iteration 当前迭代
	Iteration int `json:"iteration"`
	// StartTime 开始时间
	StartTime time.Time `json:"start_time"`
	// EndTime 结束时间
	EndTime *time.Time `json:"end_time,omitempty"`
	// PlannerOutput 规划产出
	PlannerOutput *PlannerOutput `json:"planner_output,omitempty"`
	// Phase1Output Phase 1 产出
	Phase1Output *Phase1Output `json:"phase1_output,omitempty"`
	// Phase2Output Phase 2 产出
	Phase2Output *Phase2Output `json:"phase2_output,omitempty"`
	// EvaluatorOutput 评估产出
	EvaluatorOutput *EvaluatorOutput `json:"evaluator_output,omitempty"`
}

// NewOrchestrator 创建新的编排器
func NewOrchestrator(planner Planner, generator Generator, evaluator Evaluator, iterationManager *IterationManagerImpl, agentCaller AgentCaller) *Orchestrator {
	return &Orchestrator{
		planner:          planner,
		generator:        generator,
		evaluator:        evaluator,
		iterationManager: iterationManager,
		agentCaller:      agentCaller,
		statuses:         make(map[string]*ExecutionStatus),
	}
}

// Execute 执行完整的 P-G-E 流程
func (o *Orchestrator) Execute(ctx context.Context, taskID string) error {
	slog.Info("orchestrator starting execution",
		"task_id", taskID,
	)

	// 初始化状态
	status := &ExecutionStatus{
		TaskID:    taskID,
		Phase:     "planner",
		Status:    "running",
		StartTime: time.Now(),
	}
	o.mu.Lock()
	o.statuses[taskID] = status
	o.mu.Unlock()

	// Phase 1: Planner
	plannerOutput, err := o.planner.Execute(ctx, taskID)
	if err != nil {
		return o.handleError(taskID, "planner", err)
	}
	status.PlannerOutput = plannerOutput
	slog.Info("planner phase completed", "task_id", taskID)

	// Phase 2: Generator
	status.Phase = "generator"

	// Phase 2.1: 测试编码 (并行)
	phase1Output, err := o.generator.ExecutePhase1(ctx, taskID, plannerOutput)
	if err != nil {
		return o.handleError(taskID, "generator_phase1", err)
	}
	status.Phase1Output = phase1Output

	// 迭代循环
	for {
		// Phase 2.2: 代码实现
		phase2Output, err := o.generator.ExecutePhase2(ctx, taskID, phase1Output, o.getFailureReport(taskID))
		if err != nil {
			return o.handleError(taskID, "generator_phase2", err)
		}
		status.Phase2Output = phase2Output
		status.Iteration = phase2Output.Iteration

		// Phase 3: Evaluator
		status.Phase = "evaluator"
		evaluatorOutput, err := o.evaluator.Execute(ctx, taskID, phase2Output)
		if err != nil {
			return o.handleError(taskID, "evaluator", err)
		}
		status.EvaluatorOutput = evaluatorOutput

		// 记录迭代
		o.iterationManager.RecordIteration(taskID, evaluatorOutput)

		// 检查是否通过
		if evaluatorOutput.Passed {
			status.Status = "completed"
			now := time.Now()
			status.EndTime = &now
			slog.Info("orchestrator execution completed",
				"task_id", taskID,
				"iterations", status.Iteration,
			)
			return nil
		}

		// 检查是否应该继续迭代
		shouldContinue, reason := o.iterationManager.ShouldContinue(taskID, evaluatorOutput)
		if !shouldContinue {
			status.Status = "needs_attention"
			now := time.Now()
			status.EndTime = &now
			slog.Warn("orchestrator execution needs attention",
				"task_id", taskID,
				"reason", reason,
				"iterations", status.Iteration,
			)
			return fmt.Errorf("iteration limit reached: %s", reason)
		}

		// 继续迭代
		o.iterationManager.IncrementIteration(taskID)
		status.Phase = "generator"
		slog.Info("starting next iteration",
			"task_id", taskID,
			"iteration", o.iterationManager.GetIteration(taskID),
		)
	}
}

// handleError 处理错误
func (o *Orchestrator) handleError(taskID, phase string, err error) error {
	o.mu.Lock()
	if status, exists := o.statuses[taskID]; exists {
		status.Status = "failed"
		now := time.Now()
		status.EndTime = &now
	}
	o.mu.Unlock()

	slog.Error("orchestrator execution failed",
		"task_id", taskID,
		"phase", phase,
		"error", err,
	)
	return err
}

// getFailureReport 获取失败报告
func (o *Orchestrator) getFailureReport(taskID string) string {
	o.mu.Lock()
	defer o.mu.Unlock()
	if status, exists := o.statuses[taskID]; exists && status.EvaluatorOutput != nil {
		return status.EvaluatorOutput.FailureReport
	}
	return ""
}

// GetStatus 获取执行状态
func (o *Orchestrator) GetStatus(taskID string) *ExecutionStatus {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.statuses[taskID]
}