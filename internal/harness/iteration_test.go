// Package harness 提供 P-G-E 编排引擎
package harness

import (
	"context"
	"testing"
	"time"
)

// TestIterationManager tests IterationManager creation
func TestIterationManager(t *testing.T) {
	config := IterationConfig{MaxIterations: 5}
	manager := NewIterationManager(config)

	if manager == nil {
		t.Fatal("expected non-nil manager")
	}
}

// TestIterationManagerGetIteration tests GetIteration
func TestIterationManagerGetIteration(t *testing.T) {
	manager := NewIterationManager(DefaultIterationConfig)

	// Initial iteration should be 0
	if manager.GetIteration("TEST-001") != 0 {
		t.Error("expected initial iteration 0")
	}

	// After increment
	manager.IncrementIteration("TEST-001")
	if manager.GetIteration("TEST-001") != 1 {
		t.Errorf("expected iteration 1, got %d", manager.GetIteration("TEST-001"))
	}
}

// TestIncrementIteration tests IncrementIteration
func TestIncrementIteration(t *testing.T) {
	manager := NewIterationManager(DefaultIterationConfig)

	// First increment
	iter := manager.IncrementIteration("TEST-001")
	if iter != 1 {
		t.Errorf("expected 1, got %d", iter)
	}

	// Second increment
	iter = manager.IncrementIteration("TEST-001")
	if iter != 2 {
		t.Errorf("expected 2, got %d", iter)
	}
}

// TestCheckLimit tests CheckLimit
func TestCheckLimit(t *testing.T) {
	config := IterationConfig{MaxIterations: 3}
	manager := NewIterationManager(config)

	// Under limit
	if manager.CheckLimit("TEST-001") {
		t.Error("expected not at limit with 0 iterations")
	}

	manager.IncrementIteration("TEST-001") // 1
	manager.IncrementIteration("TEST-001") // 2
	manager.IncrementIteration("TEST-001") // 3

	// At limit
	if !manager.CheckLimit("TEST-001") {
		t.Error("expected at limit with 3 iterations")
	}
}

// TestShouldContinue tests ShouldContinue
func TestShouldContinue(t *testing.T) {
	config := IterationConfig{MaxIterations: 3}
	manager := NewIterationManager(config)

	// Evaluator passed
	evaluatorOutput := &EvaluatorOutput{Passed: true}
	shouldContinue, reason := manager.ShouldContinue("TEST-001", evaluatorOutput)
	if shouldContinue {
		t.Error("expected not to continue when passed")
	}
	if reason != "evaluation passed" {
		t.Errorf("expected 'evaluation passed', got '%s'", reason)
	}

	// Evaluator failed, under limit
	evaluatorOutput = &EvaluatorOutput{Passed: false}
	shouldContinue, _ = manager.ShouldContinue("TEST-002", evaluatorOutput)
	if !shouldContinue {
		t.Error("expected to continue when failed and under limit")
	}

	// Evaluator failed, at limit
	manager.iterations["TEST-003"] = 3
	shouldContinue, reason = manager.ShouldContinue("TEST-003", evaluatorOutput)
	if shouldContinue {
		t.Error("expected not to continue when at limit")
	}
	if reason == "" {
		t.Error("expected reason for limit reached")
	}
}

// TestRecordIteration tests RecordIteration
func TestRecordIteration(t *testing.T) {
	manager := NewIterationManager(DefaultIterationConfig)

	evaluatorOutput := &EvaluatorOutput{
		TaskID:         "TEST-001",
		Passed:         false,
		FailureReport:  "Test failed",
		Iteration:      1,
	}

	manager.RecordIteration("TEST-001", evaluatorOutput)

	history := manager.GetHistory("TEST-001")
	if len(history) != 1 {
		t.Fatalf("expected 1 history record, got %d", len(history))
	}
	if history[0].Iteration != 1 {
		t.Errorf("expected iteration 1, got %d", history[0].Iteration)
	}
	if history[0].Passed {
		t.Error("expected Passed to be false")
	}
}

// TestGetStatus tests GetStatus
func TestGetStatus(t *testing.T) {
	manager := NewIterationManager(IterationConfig{MaxIterations: 5})

	// Initial status
	status := manager.GetStatus("TEST-001")
	if status.TaskID != "TEST-001" {
		t.Errorf("expected TaskID 'TEST-001', got '%s'", status.TaskID)
	}
	if status.Max != 5 {
		t.Errorf("expected Max 5, got %d", status.Max)
	}
	if status.NeedsAttention {
		t.Error("expected NeedsAttention to be false initially")
	}

	// After iterations
	manager.iterations["TEST-002"] = 5
	status = manager.GetStatus("TEST-002")
	if !status.NeedsAttention {
		t.Error("expected NeedsAttention to be true at limit")
	}
}

// TestOrchestrator tests Orchestrator creation
func TestOrchestrator(t *testing.T) {
	mockCaller := &MockAgentCaller{Available: true}
	planner := NewPlanner(mockCaller)
	generator := NewGenerator(mockCaller)
	evaluator := NewEvaluator(mockCaller)
	iterationManager := NewIterationManager(DefaultIterationConfig)

	orchestrator := NewOrchestrator(planner, generator, evaluator, iterationManager, mockCaller)
	if orchestrator == nil {
		t.Fatal("expected non-nil orchestrator")
	}
}

// TestOrchestratorExecute tests Orchestrator.Execute
func TestOrchestratorExecute(t *testing.T) {
	mockCaller := &MockAgentCaller{
		Available: true,
		CallFunc: func(ctx context.Context, input *AgentInput) (*AgentOutput, error) {
			return &AgentOutput{
				Success: true,
				Content: "All tests passed",
			}, nil
		},
	}
	planner := NewPlanner(mockCaller)
	generator := NewGenerator(mockCaller)
	evaluator := NewEvaluator(mockCaller)
	iterationManager := NewIterationManager(DefaultIterationConfig)

	orchestrator := NewOrchestrator(planner, generator, evaluator, iterationManager, mockCaller)

	ctx := context.Background()
	err := orchestrator.Execute(ctx, "TEST-ORCH-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	status := orchestrator.GetStatus("TEST-ORCH-001")
	if status == nil {
		t.Fatal("expected non-nil status")
	}
	if status.Status != "completed" {
		t.Errorf("expected status 'completed', got '%s'", status.Status)
	}
}

// TestOrchestratorExecuteWithIteration tests iteration flow
func TestOrchestratorExecuteWithIteration(t *testing.T) {
	callCount := 0
	mockCaller := &MockAgentCaller{
		Available: true,
		CallFunc: func(ctx context.Context, input *AgentInput) (*AgentOutput, error) {
			callCount++
			// First evaluation fails, second passes
			if input.Context["phase"] == "bdd_acceptance" && callCount < 10 {
				return &AgentOutput{Success: true, Content: "Tests passed"}, nil
			}
			if input.Context["phase"] == "code_review" && callCount < 10 {
				return &AgentOutput{Success: true, Content: "Found issue in code"}, nil
			}
			return &AgentOutput{Success: true, Content: "All passed"}, nil
		},
	}
	planner := NewPlanner(mockCaller)
	generator := NewGenerator(mockCaller)
	evaluator := NewEvaluator(mockCaller)
	iterationManager := NewIterationManager(IterationConfig{MaxIterations: 3})

	orchestrator := NewOrchestrator(planner, generator, evaluator, iterationManager, mockCaller)

	ctx := context.Background()
	_ = orchestrator.Execute(ctx, "TEST-ITER")

	// Should complete (either passed or hit limit)
	status := orchestrator.GetStatus("TEST-ITER")
	if status == nil {
		t.Fatal("expected non-nil status")
	}
	t.Logf("Final status: %s, iteration: %d", status.Status, status.Iteration)
}

// TestExecutionStatus tests ExecutionStatus structure
func TestExecutionStatus(t *testing.T) {
	now := time.Now()
	status := &ExecutionStatus{
		TaskID:      "TEST-001",
		Phase:       "generator",
		Status:      "running",
		Iteration:   2,
		StartTime:   now,
	}

	if status.TaskID != "TEST-001" {
		t.Errorf("expected TaskID 'TEST-001', got '%s'", status.TaskID)
	}
	if status.Phase != "generator" {
		t.Errorf("expected Phase 'generator', got '%s'", status.Phase)
	}
	if status.Iteration != 2 {
		t.Errorf("expected Iteration 2, got %d", status.Iteration)
	}
}

// TestIterationRecord tests IterationRecord structure
func TestIterationRecord(t *testing.T) {
	record := IterationRecord{
		Iteration:     2,
		FailureReport: "Test failed",
		Timestamp:     time.Now(),
		Passed:        false,
	}

	if record.Iteration != 2 {
		t.Errorf("expected Iteration 2, got %d", record.Iteration)
	}
	if record.Passed {
		t.Error("expected Passed to be false")
	}
	if record.FailureReport == "" {
		t.Error("expected non-empty FailureReport")
	}
}