// Package harness 提供 P-G-E 编排引擎
package harness

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// Evaluator 评估器接口
type Evaluator interface {
	// Execute 执行评估
	Execute(ctx context.Context, taskID string, generatorOutput *Phase2Output) (*EvaluatorOutput, error)
}

// EvaluatorOutput 评估器产出
type EvaluatorOutput struct {
	// TaskID 任务 ID
	TaskID string `json:"task_id"`
	// Passed 是否全部通过
	Passed bool `json:"passed"`
	// BDDResult BDD 测试结果
	BDDResult TestResult `json:"bdd_result"`
	// TDDResult TDD 测试结果
	TDDResult TestResult `json:"tdd_result"`
	// CodeReview 代码审计结果
	CodeReview ReviewResult `json:"code_review"`
	// StyleReview 代码风格评审结果
	StyleReview ReviewResult `json:"style_review"`
	// FailureReport 失败报告 (对话上下文)
	FailureReport string `json:"failure_report,omitempty"`
	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`
	// Iteration 迭代版本
	Iteration int `json:"iteration"`
}

// TestResult 测试结果
type TestResult struct {
	// Passed 是否通过
	Passed bool `json:"passed"`
	// Total 总测试数
	Total int `json:"total"`
	// PassedCount 通过数
	PassedCount int `json:"passed_count"`
	// FailedCases 失败用例
	FailedCases []string `json:"failed_cases,omitempty"`
}

// ReviewResult 审查结果
type ReviewResult struct {
	// Passed 是否通过
	Passed bool `json:"passed"`
	// Issues 问题列表
	Issues []string `json:"issues,omitempty"`
}

// EvaluatorImpl Evaluator 实现
type EvaluatorImpl struct {
	// agentCaller Agent 调用器
	agentCaller AgentCaller
	// mu 互斥锁
	mu sync.Mutex
	// outputs 已完成的评估产出
	outputs map[string]*EvaluatorOutput
}

// NewEvaluator 创建新的评估器
func NewEvaluator(agentCaller AgentCaller) *EvaluatorImpl {
	return &EvaluatorImpl{
		agentCaller: agentCaller,
		outputs:     make(map[string]*EvaluatorOutput),
	}
}

// Execute 执行评估
// E1: BDD 验收执行
// E2: TDD 验收执行
// E3: 代码审计
// E4: 代码风格评审
func (e *EvaluatorImpl) Execute(ctx context.Context, taskID string, generatorOutput *Phase2Output) (*EvaluatorOutput, error) {
	slog.Info("evaluator starting execution",
		"task_id", taskID,
		"iteration", generatorOutput.Iteration,
	)

	// 检查是否已有产出
	e.mu.Lock()
	if output, exists := e.outputs[taskID]; exists {
		e.mu.Unlock()
		slog.Debug("evaluator output already exists",
			"task_id", taskID,
		)
		return output, nil
	}
	e.mu.Unlock()

	// 执行 E1-E4
	bddResult := e.executeBDDTest(ctx, taskID, generatorOutput)
	tddResult := e.executeTDDTest(ctx, taskID, generatorOutput)
	codeReview := e.executeCodeReview(ctx, taskID, generatorOutput)
	styleReview := e.executeStyleReview(ctx, taskID, generatorOutput)

	// 判断是否通过
	passed := bddResult.Passed && tddResult.Passed && codeReview.Passed && styleReview.Passed

	// 生成失败报告
	var failureReport string
	if !passed {
		failureReport = e.generateFailureReport(bddResult, tddResult, codeReview, styleReview)
	}

	// 创建产出
	output := &EvaluatorOutput{
		TaskID:        taskID,
		Passed:        passed,
		BDDResult:     bddResult,
		TDDResult:     tddResult,
		CodeReview:    codeReview,
		StyleReview:   styleReview,
		FailureReport: failureReport,
		CreatedAt:     time.Now(),
		Iteration:     generatorOutput.Iteration,
	}

	// 存储产出
	e.mu.Lock()
	e.outputs[taskID] = output
	e.mu.Unlock()

	slog.Info("evaluator execution completed",
		"task_id", taskID,
		"passed", passed,
		"iteration", output.Iteration,
	)

	return output, nil
}

// executeBDDTest 执行 BDD 验收测试 (E1)
func (e *EvaluatorImpl) executeBDDTest(ctx context.Context, taskID string, generatorOutput *Phase2Output) TestResult {
	slog.Debug("executing BDD acceptance test",
		"task_id", taskID,
	)

	input := &AgentInput{
		AgentName: "bmad-agent-qa",
		Task:      "执行以下代码路径的 BDD 验收测试:\n" + generatorOutput.CodePath,
		Context: map[string]string{
			"phase": "bdd_acceptance",
		},
	}

	output, err := e.agentCaller.Call(ctx, input)
	if err != nil {
		slog.Error("BDD test execution failed",
			"task_id", taskID,
			"error", err,
		)
		return TestResult{
			Passed:   false,
			Total:    1,
			PassedCount: 0,
			FailedCases: []string{"Agent execution failed: " + err.Error()},
		}
	}

	// 解析测试结果 (简化处理)
	result := parseTestResult(output.Content)
	return result
}

// executeTDDTest 执行 TDD 验收测试 (E2)
func (e *EvaluatorImpl) executeTDDTest(ctx context.Context, taskID string, generatorOutput *Phase2Output) TestResult {
	slog.Debug("executing TDD acceptance test",
		"task_id", taskID,
	)

	input := &AgentInput{
		AgentName: "bmad-agent-qa",
		Task:      "执行以下代码路径的单元测试验收:\n" + generatorOutput.CodePath,
		Context: map[string]string{
			"phase": "tdd_acceptance",
		},
	}

	output, err := e.agentCaller.Call(ctx, input)
	if err != nil {
		slog.Error("TDD test execution failed",
			"task_id", taskID,
			"error", err,
		)
		return TestResult{
			Passed:   false,
			Total:    1,
			PassedCount: 0,
			FailedCases: []string{"Agent execution failed: " + err.Error()},
		}
	}

	result := parseTestResult(output.Content)
	return result
}

// executeCodeReview 执行代码审计 (E3)
func (e *EvaluatorImpl) executeCodeReview(ctx context.Context, taskID string, generatorOutput *Phase2Output) ReviewResult {
	slog.Debug("executing code review",
		"task_id", taskID,
	)

	input := &AgentInput{
		AgentName: "bmad-code-review",
		Task:      "审计以下代码实现:\n" + generatorOutput.Summary,
		Context: map[string]string{
			"phase": "code_review",
		},
	}

	output, err := e.agentCaller.Call(ctx, input)
	if err != nil {
		slog.Error("code review failed",
			"task_id", taskID,
			"error", err,
		)
		return ReviewResult{
			Passed: false,
			Issues: []string{"Code review failed: " + err.Error()},
		}
	}

	result := parseReviewResult(output.Content)
	return result
}

// executeStyleReview 执行代码风格评审 (E4)
func (e *EvaluatorImpl) executeStyleReview(ctx context.Context, taskID string, generatorOutput *Phase2Output) ReviewResult {
	slog.Debug("executing style review",
		"task_id", taskID,
	)

	input := &AgentInput{
		AgentName: "bmad-editorial-review-prose",
		Task:      "评审以下代码的风格:\n" + generatorOutput.Summary,
		Context: map[string]string{
			"phase": "style_review",
		},
	}

	output, err := e.agentCaller.Call(ctx, input)
	if err != nil {
		slog.Error("style review failed",
			"task_id", taskID,
			"error", err,
		)
		return ReviewResult{
			Passed: false,
			Issues: []string{"Style review failed: " + err.Error()},
		}
	}

	result := parseReviewResult(output.Content)
	return result
}

// generateFailureReport 生成失败报告
func (e *EvaluatorImpl) generateFailureReport(bddResult, tddResult TestResult, codeReview, styleReview ReviewResult) string {
	var sb strings.Builder

	sb.WriteString("# 评估失败报告\n\n")

	if !bddResult.Passed {
		sb.WriteString("## BDD 测试失败\n")
		sb.WriteString(fmt.Sprintf("- 通过: %d/%d\n", bddResult.PassedCount, bddResult.Total))
		for _, failed := range bddResult.FailedCases {
			sb.WriteString(fmt.Sprintf("- 失败用例: %s\n", failed))
		}
		sb.WriteString("\n")
	}

	if !tddResult.Passed {
		sb.WriteString("## TDD 测试失败\n")
		sb.WriteString(fmt.Sprintf("- 通过: %d/%d\n", tddResult.PassedCount, tddResult.Total))
		for _, failed := range tddResult.FailedCases {
			sb.WriteString(fmt.Sprintf("- 失败用例: %s\n", failed))
		}
		sb.WriteString("\n")
	}

	if !codeReview.Passed {
		sb.WriteString("## 代码审计问题\n")
		for _, issue := range codeReview.Issues {
			sb.WriteString(fmt.Sprintf("- %s\n", issue))
		}
		sb.WriteString("\n")
	}

	if !styleReview.Passed {
		sb.WriteString("## 代码风格问题\n")
		for _, issue := range styleReview.Issues {
			sb.WriteString(fmt.Sprintf("- %s\n", issue))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// parseTestResult 解析测试结果 (简化处理)
func parseTestResult(content string) TestResult {
	// 简化处理：如果包含 "PASS" 或 "passed" 则认为通过
	content = strings.ToLower(content)
	if strings.Contains(content, "pass") && !strings.Contains(content, "fail") {
		return TestResult{
			Passed:      true,
			Total:       1,
			PassedCount: 1,
			FailedCases: nil,
		}
	}

	// 检查是否有失败
	if strings.Contains(content, "fail") {
		return TestResult{
			Passed:      false,
			Total:       1,
			PassedCount: 0,
			FailedCases: []string{"Test failed"},
		}
	}

	// 默认通过
	return TestResult{
		Passed:      true,
		Total:       1,
		PassedCount: 1,
		FailedCases: nil,
	}
}

// parseReviewResult 解析审查结果 (简化处理)
func parseReviewResult(content string) ReviewResult {
	// 简化处理：如果包含 "issue" 或 "problem" 则认为有问题
	content = strings.ToLower(content)
	if strings.Contains(content, "issue") || strings.Contains(content, "problem") || strings.Contains(content, "error") {
		return ReviewResult{
			Passed: false,
			Issues: []string{"Found issues in review"},
		}
	}

	// 默认通过
	return ReviewResult{
		Passed: true,
		Issues: nil,
	}
}

// GetOutput 获取已完成的评估产出
func (e *EvaluatorImpl) GetOutput(taskID string) *EvaluatorOutput {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.outputs[taskID]
}

// HasOutput 检查是否已有评估产出
func (e *EvaluatorImpl) HasOutput(taskID string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, exists := e.outputs[taskID]
	return exists
}