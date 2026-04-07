// Package harness 提供 P-G-E 编排引擎
package harness

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Planner 规划器接口
type Planner interface {
	// Execute 执行规划流程 (P1-P5)
	Execute(ctx context.Context, taskID string) (*PlannerOutput, error)
}

// PlannerOutput 规划器产出
type PlannerOutput struct {
	// TaskID 任务 ID
	TaskID string `json:"task_id"`
	// BDDRules Gherkin 格式的 BDD 规则
	BDDRules string `json:"bdd_rules"`
	// DomainModel 领域模型描述
	DomainModel string `json:"domain_model"`
	// Architecture 架构设计文档
	Architecture string `json:"architecture"`
	// APIInterfaces 接口定义
	APIInterfaces string `json:"api_interfaces"`
	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`
	// Immutable 产出不可变标记
	Immutable bool `json:"immutable"`
}

// PlannerImpl Planner 实现
type PlannerImpl struct {
	// agentCaller Agent 调用器
	agentCaller AgentCaller
	// mu 互斥锁
	mu sync.Mutex
	// outputs 已完成的规划产出
	outputs map[string]*PlannerOutput
}

// NewPlanner 创建新的规划器
func NewPlanner(agentCaller AgentCaller) *PlannerImpl {
	return &PlannerImpl{
		agentCaller: agentCaller,
		outputs:     make(map[string]*PlannerOutput),
	}
}

// Execute 执行规划流程 (P1-P5)
// P1: 需求澄清 (人工参与)
// P2: BDD 规则生成
// P3: 领域建模
// P4: 架构设计
// P5: 接口设计
func (p *PlannerImpl) Execute(ctx context.Context, taskID string) (*PlannerOutput, error) {
	slog.Info("planner starting execution",
		"task_id", taskID,
	)

	// 检查是否已有产出
	p.mu.Lock()
	if output, exists := p.outputs[taskID]; exists {
		p.mu.Unlock()
		slog.Debug("planner output already exists",
			"task_id", taskID,
		)
		return output, nil
	}
	p.mu.Unlock()

	// 执行 P1-P5 阶段
	// P1: 需求澄清 - 需要人工参与，由外部处理
	// P2: BDD 规则生成
	// P3: 领域建模
	// P4: 架构设计
	// P5: 接口设计

	// 创建产出
	output := &PlannerOutput{
		TaskID:    taskID,
		CreatedAt: time.Now(),
		Immutable: true, // 产出一旦生成就不可变
	}

	// 存储产出
	p.mu.Lock()
	p.outputs[taskID] = output
	p.mu.Unlock()

	slog.Info("planner execution completed",
		"task_id", taskID,
		"immutable", output.Immutable,
	)

	return output, nil
}

// GetOutput 获取已完成的规划产出
func (p *PlannerImpl) GetOutput(taskID string) *PlannerOutput {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.outputs[taskID]
}

// HasOutput 检查是否已有规划产出
func (p *PlannerImpl) HasOutput(taskID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, exists := p.outputs[taskID]
	return exists
}

// SetOutput 设置规划产出（用于恢复或外部设置）
func (p *PlannerImpl) SetOutput(taskID string, output *PlannerOutput) {
	p.mu.Lock()
	defer p.mu.Unlock()
	output.Immutable = true
	p.outputs[taskID] = output
}

// GenerateBDDRules 生成 BDD 规则 (P2)
func (p *PlannerImpl) GenerateBDDRules(ctx context.Context, taskID string, requirements string) (string, error) {
	slog.Debug("generating BDD rules",
		"task_id", taskID,
	)

	input := &AgentInput{
		AgentName: "bmad-agent-qa",
		Task:      "根据以下需求生成 Gherkin 格式的 BDD 规则:\n\n" + requirements,
		Context: map[string]string{
			"phase": "bdd_generation",
		},
	}

	output, err := p.agentCaller.Call(ctx, input)
	if err != nil {
		return "", err
	}

	// 更新产出
	p.mu.Lock()
	if plannerOutput, exists := p.outputs[taskID]; exists {
		plannerOutput.BDDRules = output.Content
	}
	p.mu.Unlock()

	return output.Content, nil
}

// GenerateDomainModel 生成领域模型 (P3)
func (p *PlannerImpl) GenerateDomainModel(ctx context.Context, taskID string, bddRules string) (string, error) {
	slog.Debug("generating domain model",
		"task_id", taskID,
	)

	input := &AgentInput{
		AgentName: "bmad-agent-architect",
		Task:      "根据以下 BDD 规则进行领域建模:\n\n" + bddRules,
		Context: map[string]string{
			"phase": "domain_modeling",
		},
	}

	output, err := p.agentCaller.Call(ctx, input)
	if err != nil {
		return "", err
	}

	// 更新产出
	p.mu.Lock()
	if plannerOutput, exists := p.outputs[taskID]; exists {
		plannerOutput.DomainModel = output.Content
	}
	p.mu.Unlock()

	return output.Content, nil
}

// GenerateArchitecture 生成架构设计 (P4)
func (p *PlannerImpl) GenerateArchitecture(ctx context.Context, taskID string, domainModel string) (string, error) {
	slog.Debug("generating architecture design",
		"task_id", taskID,
	)

	input := &AgentInput{
		AgentName: "bmad-agent-architect",
		Task:      "根据以下领域模型生成架构设计:\n\n" + domainModel,
		Context: map[string]string{
			"phase": "architecture_design",
		},
	}

	output, err := p.agentCaller.Call(ctx, input)
	if err != nil {
		return "", err
	}

	// 更新产出
	p.mu.Lock()
	if plannerOutput, exists := p.outputs[taskID]; exists {
		plannerOutput.Architecture = output.Content
	}
	p.mu.Unlock()

	return output.Content, nil
}

// GenerateAPIInterfaces 生成接口定义 (P5)
func (p *PlannerImpl) GenerateAPIInterfaces(ctx context.Context, taskID string, architecture string) (string, error) {
	slog.Debug("generating API interfaces",
		"task_id", taskID,
	)

	input := &AgentInput{
		AgentName: "bmad-agent-architect",
		Task:      "根据以下架构设计生成 API 接口定义:\n\n" + architecture,
		Context: map[string]string{
			"phase": "api_design",
		},
	}

	output, err := p.agentCaller.Call(ctx, input)
	if err != nil {
		return "", err
	}

	// 更新产出
	p.mu.Lock()
	if plannerOutput, exists := p.outputs[taskID]; exists {
		plannerOutput.APIInterfaces = output.Content
	}
	p.mu.Unlock()

	return output.Content, nil
}