// Package harness 提供 P-G-E 编排引擎
package harness

import (
	"context"
	"testing"
	"time"

	"github.com/dministrator/symphony/internal/common/errors"
)

// TestEvaluatorInterface tests that Evaluator interface is properly defined
func TestEvaluatorInterface(t *testing.T) {
	var _ Evaluator = (*EvaluatorImpl)(nil)
}

// TestEvaluatorOutputStructure tests EvaluatorOutput structure
func TestEvaluatorOutputStructure(t *testing.T) {
	output := &EvaluatorOutput{
		TaskID:     "TEST-001",
		Passed:     true,
		BDDResult:  TestResult{Passed: true, Total: 5, PassedCount: 5},
		TDDResult:  TestResult{Passed: true, Total: 3, PassedCount: 3},
		CodeReview: ReviewResult{Passed: true},
		StyleReview: ReviewResult{Passed: true},
		CreatedAt:  time.Now(),
		Iteration:  1,
	}

	if output.TaskID != "TEST-001" {
		t.Errorf("expected TaskID 'TEST-001', got %s", output.TaskID)
	}
	if !output.Passed {
		t.Error("expected Passed to be true")
	}
}

// TestTestResultStructure tests TestResult structure
func TestTestResultStructure(t *testing.T) {
	result := TestResult{
		Passed:      false,
		Total:       5,
		PassedCount: 3,
		FailedCases: []string{"Test1", "Test2"},
	}

	if result.Passed {
		t.Error("expected Passed to be false")
	}
	if result.Total != 5 {
		t.Errorf("expected Total 5, got %d", result.Total)
	}
	if len(result.FailedCases) != 2 {
		t.Errorf("expected 2 failed cases, got %d", len(result.FailedCases))
	}
}

// TestReviewResultStructure tests ReviewResult structure
func TestReviewResultStructure(t *testing.T) {
	result := ReviewResult{
		Passed: false,
		Issues: []string{"Issue1", "Issue2"},
	}

	if result.Passed {
		t.Error("expected Passed to be false")
	}
	if len(result.Issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(result.Issues))
	}
}

// TestNewEvaluator tests constructor
func TestNewEvaluator(t *testing.T) {
	mockCaller := &MockAgentCaller{Available: true}
	evaluator := NewEvaluator(mockCaller)
	if evaluator == nil {
		t.Fatal("expected non-nil Evaluator")
	}
}

// TestEvaluatorExecute tests Execute method
func TestEvaluatorExecute(t *testing.T) {
	mockCaller := &MockAgentCaller{
		Available: true,
		CallFunc: func(ctx context.Context, input *AgentInput) (*AgentOutput, error) {
			// Return passing results
			return &AgentOutput{
				Success:  true,
				Content:  "All tests passed",
				Duration: 100 * time.Millisecond,
			}, nil
		},
	}
	evaluator := NewEvaluator(mockCaller)

	generatorOutput := &Phase2Output{
		TaskID:     "TEST-001",
		CodePath:   "/workspace/TEST-001/code",
		Summary:    "Implementation",
		Iteration:  1,
	}

	ctx := context.Background()
	output, err := evaluator.Execute(ctx, "TEST-001", generatorOutput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.TaskID != "TEST-001" {
		t.Errorf("expected TaskID 'TEST-001', got %s", output.TaskID)
	}
	if !output.Passed {
		t.Error("expected Passed to be true")
	}
	if output.Iteration != 1 {
		t.Errorf("expected Iteration 1, got %d", output.Iteration)
	}
}

// TestEvaluatorExecuteWithFailures tests Execute with failures
func TestEvaluatorExecuteWithFailures(t *testing.T) {
	mockCaller := &MockAgentCaller{
		Available: true,
		CallFunc: func(ctx context.Context, input *AgentInput) (*AgentOutput, error) {
			// Return failing results based on phase
			switch input.Context["phase"] {
			case "bdd_acceptance":
				return &AgentOutput{Success: true, Content: "Tests failed: 2 failures"}, nil
			case "code_review":
				return &AgentOutput{Success: true, Content: "Found issues in code"}, nil
			default:
				return &AgentOutput{Success: true, Content: "All passed"}, nil
			}
		},
	}
	evaluator := NewEvaluator(mockCaller)

	generatorOutput := &Phase2Output{
		TaskID:     "TEST-FAIL",
		CodePath:   "/workspace/TEST-FAIL/code",
		Summary:    "Implementation",
		Iteration:  1,
	}

	ctx := context.Background()
	output, err := evaluator.Execute(ctx, "TEST-FAIL", generatorOutput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Passed {
		t.Error("expected Passed to be false when there are failures")
	}
	if output.FailureReport == "" {
		t.Error("expected FailureReport to be generated")
	}
}

// TestEvaluatorExecuteAgentError tests error handling
func TestEvaluatorExecuteAgentError(t *testing.T) {
	mockCaller := &MockAgentCaller{
		Available: true,
		CallFunc: func(ctx context.Context, input *AgentInput) (*AgentOutput, error) {
			return nil, errors.ErrAgentExecutionFail
		},
	}
	evaluator := NewEvaluator(mockCaller)

	generatorOutput := &Phase2Output{
		TaskID:     "TEST-ERROR",
		CodePath:   "/workspace/TEST-ERROR/code",
		Summary:    "Implementation",
		Iteration:  1,
	}

	ctx := context.Background()
	output, err := evaluator.Execute(ctx, "TEST-ERROR", generatorOutput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should still return output with errors
	if output.Passed {
		t.Error("expected Passed to be false when agent fails")
	}
}

// TestEvaluatorGetOutput tests GetOutput method
func TestEvaluatorGetOutput(t *testing.T) {
	mockCaller := &MockAgentCaller{Available: true}
	evaluator := NewEvaluator(mockCaller)

	generatorOutput := &Phase2Output{
		TaskID:     "TEST-GET",
		CodePath:   "/workspace",
		Summary:    "Test",
		Iteration:  1,
	}

	ctx := context.Background()
	_, _ = evaluator.Execute(ctx, "TEST-GET", generatorOutput)

	output := evaluator.GetOutput("TEST-GET")
	if output == nil {
		t.Fatal("expected non-nil output")
	}
	if output.TaskID != "TEST-GET" {
		t.Errorf("expected TaskID 'TEST-GET', got %s", output.TaskID)
	}
}

// TestEvaluatorHasOutput tests HasOutput method
func TestEvaluatorHasOutput(t *testing.T) {
	mockCaller := &MockAgentCaller{Available: true}
	evaluator := NewEvaluator(mockCaller)

	if evaluator.HasOutput("TEST-001") {
		t.Error("expected no output initially")
	}

	generatorOutput := &Phase2Output{
		TaskID:     "TEST-001",
		CodePath:   "/workspace",
		Summary:    "Test",
		Iteration:  1,
	}

	ctx := context.Background()
	_, _ = evaluator.Execute(ctx, "TEST-001", generatorOutput)

	if !evaluator.HasOutput("TEST-001") {
		t.Error("expected output after Execute")
	}
}

// TestParseTestResult tests parseTestResult function
func TestParseTestResult(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{"passing", "All tests PASS", true},
		{"passing lowercase", "all tests passed", true},
		{"failing", "Tests failed", false},
		{"with failures", "PASS with failures", false},
		{"neutral", "Test output", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTestResult(tt.content)
			if result.Passed != tt.expected {
				t.Errorf("expected Passed %v, got %v", tt.expected, result.Passed)
			}
		})
	}
}

// TestParseReviewResult tests parseReviewResult function
func TestParseReviewResult(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{"passing", "Code looks good", true},
		{"with issue", "Found issue in code", false},
		{"with problem", "Problem detected", false},
		{"with error", "Error in implementation", false},
		{"neutral", "Review complete", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseReviewResult(tt.content)
			if result.Passed != tt.expected {
				t.Errorf("expected Passed %v, got %v", tt.expected, result.Passed)
			}
		})
	}
}

// TestGenerateFailureReport tests failure report generation
func TestGenerateFailureReport(t *testing.T) {
	evaluator := NewEvaluator(&MockAgentCaller{Available: true})

	bddResult := TestResult{
		Passed:      false,
		Total:       5,
		PassedCount: 3,
		FailedCases: []string{"Test1", "Test2"},
	}
	tddResult := TestResult{Passed: true, Total: 3, PassedCount: 3}
	codeReview := ReviewResult{
		Passed: false,
		Issues: []string{"Missing error handling"},
	}
	styleReview := ReviewResult{Passed: true}

	report := evaluator.generateFailureReport(bddResult, tddResult, codeReview, styleReview)

	if report == "" {
		t.Error("expected non-empty failure report")
	}
	if !contains(report, "BDD 测试失败") {
		t.Error("expected BDD failure section in report")
	}
	if !contains(report, "代码审计问题") {
		t.Error("expected code review issues section in report")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) >= len(substr) &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				len(s) > len(substr) && containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}