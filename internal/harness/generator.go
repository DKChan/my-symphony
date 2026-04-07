// Package harness 提供 P-G-E 编排引擎
package harness

import (
	"context"
	"log/slog"
	"strconv"
	"sync"
	"time"
)

// Generator 生成器接口
type Generator interface {
	// ExecutePhase1 执行 Phase 1 (测试编码，并行)
	ExecutePhase1(ctx context.Context, taskID string, plannerOutput *PlannerOutput) (*Phase1Output, error)

	// ExecutePhase2 执行 Phase 2 (代码实现，顺序)
	ExecutePhase2(ctx context.Context, taskID string, phase1Output *Phase1Output, failureReport string) (*Phase2Output, error)
}

// Phase1Output Phase 1 产出
type Phase1Output struct {
	// TaskID 任务 ID
	TaskID string `json:"task_id"`
	// BDDTestScript BDD 测试脚本
	BDDTestScript string `json:"bdd_test_script"`
	// IntegrationTest 集成测试代码
	IntegrationTest string `json:"integration_test"`
	// UnitTest 单元测试代码
	UnitTest string `json:"unit_test"`
	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`
}

// Phase2Output Phase 2 产出
type Phase2Output struct {
	// TaskID 任务 ID
	TaskID string `json:"task_id"`
	// CodePath 实现代码路径
	CodePath string `json:"code_path"`
	// Summary 实现摘要
	Summary string `json:"summary"`
	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`
	// Iteration 迭代版本号
	Iteration int `json:"iteration"`
}

// GeneratorImpl Generator 实现
type GeneratorImpl struct {
	// agentCaller Agent 调用器
	agentCaller AgentCaller
	// mu 互斥锁
	mu sync.Mutex
	// phase1Outputs Phase 1 产出缓存
	phase1Outputs map[string]*Phase1Output
	// phase2Outputs Phase 2 产出缓存
	phase2Outputs map[string]*Phase2Output
	// iterations 迭代计数
	iterations map[string]int
}

// NewGenerator 创建新的生成器
func NewGenerator(agentCaller AgentCaller) *GeneratorImpl {
	return &GeneratorImpl{
		agentCaller:    agentCaller,
		phase1Outputs:  make(map[string]*Phase1Output),
		phase2Outputs:  make(map[string]*Phase2Output),
		iterations:     make(map[string]int),
	}
}

// ExecutePhase1 执行 Phase 1 (测试编码，并行)
// G1: BDD 测试脚本生成
// G2: 集成测试生成
// G3: 单元测试生成
func (g *GeneratorImpl) ExecutePhase1(ctx context.Context, taskID string, plannerOutput *PlannerOutput) (*Phase1Output, error) {
	slog.Info("generator phase 1 starting",
		"task_id", taskID,
	)

	// 检查是否已有产出
	g.mu.Lock()
	if output, exists := g.phase1Outputs[taskID]; exists {
		g.mu.Unlock()
		slog.Debug("generator phase 1 output already exists",
			"task_id", taskID,
		)
		return output, nil
	}
	g.mu.Unlock()

	// 并行执行 G1, G2, G3
	var wg sync.WaitGroup
	var results [3]string
	var errors [3]error

	// G1: BDD 测试脚本
	wg.Add(1)
	go func() {
		defer wg.Done()
		results[0], errors[0] = g.generateBDDTestScript(ctx, taskID, plannerOutput)
	}()

	// G2: 集成测试
	wg.Add(1)
	go func() {
		defer wg.Done()
		results[1], errors[1] = g.generateIntegrationTest(ctx, taskID, plannerOutput)
	}()

	// G3: 单元测试
	wg.Add(1)
	go func() {
		defer wg.Done()
		results[2], errors[2] = g.generateUnitTest(ctx, taskID, plannerOutput)
	}()

	// 等待所有任务完成
	wg.Wait()

	// 检查是否有错误
	for i, err := range errors {
		if err != nil {
			slog.Error("generator phase 1 task failed",
				"task_id", taskID,
				"task", i+1,
				"error", err,
			)
			// 即使有错误也继续，返回部分产出
		}
	}

	// 创建产出
	output := &Phase1Output{
		TaskID:         taskID,
		BDDTestScript:  results[0],
		IntegrationTest: results[1],
		UnitTest:       results[2],
		CreatedAt:      time.Now(),
	}

	// 存储产出
	g.mu.Lock()
	g.phase1Outputs[taskID] = output
	g.mu.Unlock()

	slog.Info("generator phase 1 completed",
		"task_id", taskID,
		"bdd_script_length", len(output.BDDTestScript),
		"integration_test_length", len(output.IntegrationTest),
		"unit_test_length", len(output.UnitTest),
	)

	return output, nil
}

// ExecutePhase2 执行 Phase 2 (代码实现，顺序)
// G4: 代码实现
func (g *GeneratorImpl) ExecutePhase2(ctx context.Context, taskID string, phase1Output *Phase1Output, failureReport string) (*Phase2Output, error) {
	slog.Info("generator phase 2 starting",
		"task_id", taskID,
		"has_failure_report", failureReport != "",
	)

	// 获取当前迭代版本
	g.mu.Lock()
	iteration := g.iterations[taskID] + 1
	g.iterations[taskID] = iteration
	g.mu.Unlock()

	// G4: 代码实现
	codePath, summary, err := g.generateCode(ctx, taskID, phase1Output, failureReport, iteration)
	if err != nil {
		slog.Error("generator phase 2 failed",
			"task_id", taskID,
			"iteration", iteration,
			"error", err,
		)
		return nil, err
	}

	// 创建产出
	output := &Phase2Output{
		TaskID:     taskID,
		CodePath:   codePath,
		Summary:    summary,
		CreatedAt:  time.Now(),
		Iteration:  iteration,
	}

	// 存储产出
	g.mu.Lock()
	g.phase2Outputs[taskID] = output
	g.mu.Unlock()

	slog.Info("generator phase 2 completed",
		"task_id", taskID,
		"iteration", iteration,
		"code_path", codePath,
	)

	return output, nil
}

// generateBDDTestScript 生成 BDD 测试脚本 (G1)
func (g *GeneratorImpl) generateBDDTestScript(ctx context.Context, taskID string, plannerOutput *PlannerOutput) (string, error) {
	slog.Debug("generating BDD test script",
		"task_id", taskID,
	)

	input := &AgentInput{
		AgentName: "bmad-agent-qa",
		Task:      "根据以下 BDD 规则生成可执行的 BDD 测试脚本:\n\n" + plannerOutput.BDDRules,
		Context: map[string]string{
			"phase": "bdd_test_generation",
		},
	}

	output, err := g.agentCaller.Call(ctx, input)
	if err != nil {
		return "", err
	}

	return output.Content, nil
}

// generateIntegrationTest 生成集成测试 (G2)
func (g *GeneratorImpl) generateIntegrationTest(ctx context.Context, taskID string, plannerOutput *PlannerOutput) (string, error) {
	slog.Debug("generating integration test",
		"task_id", taskID,
	)

	input := &AgentInput{
		AgentName: "bmad-agent-qa",
		Task:      "根据以下接口定义生成集成测试:\n\n" + plannerOutput.APIInterfaces,
		Context: map[string]string{
			"phase": "integration_test_generation",
		},
	}

	output, err := g.agentCaller.Call(ctx, input)
	if err != nil {
		return "", err
	}

	return output.Content, nil
}

// generateUnitTest 生成单元测试 (G3)
func (g *GeneratorImpl) generateUnitTest(ctx context.Context, taskID string, plannerOutput *PlannerOutput) (string, error) {
	slog.Debug("generating unit test",
		"task_id", taskID,
	)

	input := &AgentInput{
		AgentName: "bmad-agent-dev",
		Task:      "根据以下架构设计生成单元测试:\n\n" + plannerOutput.Architecture,
		Context: map[string]string{
			"phase": "unit_test_generation",
		},
	}

	output, err := g.agentCaller.Call(ctx, input)
	if err != nil {
		return "", err
	}

	return output.Content, nil
}

// generateCode 生成代码实现 (G4)
func (g *GeneratorImpl) generateCode(ctx context.Context, taskID string, phase1Output *Phase1Output, failureReport string, iteration int) (string, string, error) {
	slog.Debug("generating code implementation",
		"task_id", taskID,
		"iteration", iteration,
	)

	task := "根据以下测试实现代码:\n\n"
	task += "BDD 测试脚本:\n" + phase1Output.BDDTestScript + "\n\n"
	task += "集成测试:\n" + phase1Output.IntegrationTest + "\n\n"
	task += "单元测试:\n" + phase1Output.UnitTest

	if failureReport != "" {
		task += "\n\n失败报告 (需要修复的问题):\n" + failureReport
	}

	input := &AgentInput{
		AgentName: "bmad-agent-dev",
		Task:      task,
		Context: map[string]string{
			"phase":     "code_implementation",
			"iteration": strconv.Itoa(iteration),
		},
	}

	output, err := g.agentCaller.Call(ctx, input)
	if err != nil {
		return "", "", err
	}

	// 返回代码路径和摘要
	// 简化处理：返回内容作为摘要，路径为工作目录
	return "/workspace/" + taskID + "/code", output.Content, nil
}

// GetPhase1Output 获取 Phase 1 产出
func (g *GeneratorImpl) GetPhase1Output(taskID string) *Phase1Output {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.phase1Outputs[taskID]
}

// GetPhase2Output 获取 Phase 2 产出
func (g *GeneratorImpl) GetPhase2Output(taskID string) *Phase2Output {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.phase2Outputs[taskID]
}

// GetIteration 获取当前迭代次数
func (g *GeneratorImpl) GetIteration(taskID string) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.iterations[taskID]
}