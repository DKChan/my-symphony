// Package harness 提供 P-G-E 编排引擎
package harness

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dministrator/symphony/internal/common/errors"
)

// TestGeneratorInterface tests that Generator interface is properly defined
func TestGeneratorInterface(t *testing.T) {
	var _ Generator = (*GeneratorImpl)(nil)
}

// TestPhase1OutputStructure tests Phase1Output structure
func TestPhase1OutputStructure(t *testing.T) {
	output := &Phase1Output{
		TaskID:          "TEST-001",
		BDDTestScript:   "BDD test script",
		IntegrationTest: "Integration test",
		UnitTest:        "Unit test",
		CreatedAt:       time.Now(),
	}

	if output.TaskID != "TEST-001" {
		t.Errorf("expected TaskID 'TEST-001', got %s", output.TaskID)
	}
	if output.BDDTestScript == "" {
		t.Error("expected non-empty BDDTestScript")
	}
}

// TestPhase2OutputStructure tests Phase2Output structure
func TestPhase2OutputStructure(t *testing.T) {
	output := &Phase2Output{
		TaskID:     "TEST-001",
		CodePath:   "/workspace/TEST-001/code",
		Summary:    "Implementation summary",
		CreatedAt:  time.Now(),
		Iteration:  1,
	}

	if output.TaskID != "TEST-001" {
		t.Errorf("expected TaskID 'TEST-001', got %s", output.TaskID)
	}
	if output.Iteration != 1 {
		t.Errorf("expected Iteration 1, got %d", output.Iteration)
	}
}

// TestNewGenerator tests constructor
func TestNewGenerator(t *testing.T) {
	mockCaller := &MockAgentCaller{Available: true}
	generator := NewGenerator(mockCaller)
	if generator == nil {
		t.Fatal("expected non-nil Generator")
	}
}

// TestExecutePhase1 tests Phase 1 execution
func TestExecutePhase1(t *testing.T) {
	mockCaller := &MockAgentCaller{
		Available: true,
		CallFunc: func(ctx context.Context, input *AgentInput) (*AgentOutput, error) {
			// Return different content based on phase
			switch input.Context["phase"] {
			case "bdd_test_generation":
				return &AgentOutput{Success: true, Content: "BDD Test Script Code"}, nil
			case "integration_test_generation":
				return &AgentOutput{Success: true, Content: "Integration Test Code"}, nil
			case "unit_test_generation":
				return &AgentOutput{Success: true, Content: "Unit Test Code"}, nil
			default:
				return &AgentOutput{Success: true, Content: "Default output"}, nil
			}
		},
	}
	generator := NewGenerator(mockCaller)

	plannerOutput := &PlannerOutput{
		TaskID:         "TEST-001",
		BDDRules:       "Feature: Login",
		Architecture:   "Clean Architecture",
		APIInterfaces:  "POST /api/login",
	}

	ctx := context.Background()
	output, err := generator.ExecutePhase1(ctx, "TEST-001", plannerOutput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.TaskID != "TEST-001" {
		t.Errorf("expected TaskID 'TEST-001', got %s", output.TaskID)
	}
	if output.BDDTestScript == "" {
		t.Error("expected non-empty BDDTestScript")
	}
	if output.IntegrationTest == "" {
		t.Error("expected non-empty IntegrationTest")
	}
	if output.UnitTest == "" {
		t.Error("expected non-empty UnitTest")
	}
}

// TestExecutePhase1Parallel tests that Phase 1 executes tasks in parallel
func TestExecutePhase1Parallel(t *testing.T) {
	var callOrder []string
	var mu sync.Mutex

	mockCaller := &MockAgentCaller{
		Available: true,
		CallFunc: func(ctx context.Context, input *AgentInput) (*AgentOutput, error) {
			mu.Lock()
			callOrder = append(callOrder, input.Context["phase"])
			mu.Unlock()
			// Simulate some work
			time.Sleep(10 * time.Millisecond)
			return &AgentOutput{Success: true, Content: "Test output"}, nil
		},
	}
	generator := NewGenerator(mockCaller)

	plannerOutput := &PlannerOutput{
		TaskID:         "TEST-PARALLEL",
		BDDRules:       "BDD Rules",
		Architecture:   "Architecture",
		APIInterfaces:  "API Interfaces",
	}

	ctx := context.Background()
	start := time.Now()
	_, _ = generator.ExecutePhase1(ctx, "TEST-PARALLEL", plannerOutput)
	duration := time.Since(start)

	// If truly parallel, duration should be close to 10ms (not 30ms)
	// Allow some margin for overhead
	if duration > 50*time.Millisecond {
		t.Errorf("Phase 1 took too long (%v), tasks may not be running in parallel", duration)
	}

	// Verify all 3 tasks were called
	if len(callOrder) != 3 {
		t.Errorf("expected 3 calls, got %d", len(callOrder))
	}
}

// TestExecutePhase1Error tests error handling in Phase 1
func TestExecutePhase1Error(t *testing.T) {
	mockCaller := &MockAgentCaller{
		Available: true,
		CallFunc: func(ctx context.Context, input *AgentInput) (*AgentOutput, error) {
			// Make one task fail
			if input.Context["phase"] == "unit_test_generation" {
				return nil, errors.ErrAgentExecutionFail
			}
			return &AgentOutput{Success: true, Content: "Test output"}, nil
		},
	}
	generator := NewGenerator(mockCaller)

	plannerOutput := &PlannerOutput{
		TaskID:         "TEST-ERROR",
		BDDRules:       "BDD Rules",
		Architecture:   "Architecture",
		APIInterfaces:  "API Interfaces",
	}

	ctx := context.Background()
	output, err := generator.ExecutePhase1(ctx, "TEST-ERROR", plannerOutput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should still return partial output even if one task failed
	if output.BDDTestScript == "" || output.IntegrationTest == "" {
		t.Error("expected partial output when one task fails")
	}
	if output.UnitTest != "" {
		t.Error("expected empty UnitTest when that task fails")
	}
}

// TestExecutePhase2 tests Phase 2 execution
func TestExecutePhase2(t *testing.T) {
	mockCaller := &MockAgentCaller{
		Available: true,
		CallFunc: func(ctx context.Context, input *AgentInput) (*AgentOutput, error) {
			return &AgentOutput{
				Success:  true,
				Content:  "Code implementation content",
				Duration: 100 * time.Millisecond,
			}, nil
		},
	}
	generator := NewGenerator(mockCaller)

	phase1Output := &Phase1Output{
		TaskID:          "TEST-001",
		BDDTestScript:   "BDD Script",
		IntegrationTest: "Integration Test",
		UnitTest:        "Unit Test",
	}

	ctx := context.Background()
	output, err := generator.ExecutePhase2(ctx, "TEST-001", phase1Output, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.TaskID != "TEST-001" {
		t.Errorf("expected TaskID 'TEST-001', got %s", output.TaskID)
	}
	if output.Iteration != 1 {
		t.Errorf("expected Iteration 1, got %d", output.Iteration)
	}
}

// TestExecutePhase2WithFailureReport tests Phase 2 with failure report
func TestExecutePhase2WithFailureReport(t *testing.T) {
	mockCaller := &MockAgentCaller{
		Available: true,
		CallFunc: func(ctx context.Context, input *AgentInput) (*AgentOutput, error) {
			// Verify failure report is included in task
			if input.Task == "" {
				t.Error("expected non-empty task")
			}
			return &AgentOutput{Success: true, Content: "Fixed code"}, nil
		},
	}
	generator := NewGenerator(mockCaller)

	phase1Output := &Phase1Output{
		TaskID:          "TEST-FIX",
		BDDTestScript:   "BDD Script",
		IntegrationTest: "Integration Test",
		UnitTest:        "Unit Test",
	}

	ctx := context.Background()
	failureReport := "Test failed: login validation error"
	output, err := generator.ExecutePhase2(ctx, "TEST-FIX", phase1Output, failureReport)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be iteration 1 (first iteration)
	if output.Iteration != 1 {
		t.Errorf("expected Iteration 1, got %d", output.Iteration)
	}
}

// TestExecutePhase2Iteration tests iteration increment
func TestExecutePhase2Iteration(t *testing.T) {
	mockCaller := &MockAgentCaller{Available: true}
	generator := NewGenerator(mockCaller)

	phase1Output := &Phase1Output{
		TaskID:          "TEST-ITER",
		BDDTestScript:   "BDD",
		IntegrationTest: "Integration",
		UnitTest:        "Unit",
	}

	ctx := context.Background()

	// First execution
	output1, _ := generator.ExecutePhase2(ctx, "TEST-ITER", phase1Output, "")
	if output1.Iteration != 1 {
		t.Errorf("expected Iteration 1, got %d", output1.Iteration)
	}

	// Second execution (simulating iteration)
	output2, _ := generator.ExecutePhase2(ctx, "TEST-ITER", phase1Output, "Failure report")
	if output2.Iteration != 2 {
		t.Errorf("expected Iteration 2, got %d", output2.Iteration)
	}

	// Third execution
	output3, _ := generator.ExecutePhase2(ctx, "TEST-ITER", phase1Output, "Another failure")
	if output3.Iteration != 3 {
		t.Errorf("expected Iteration 3, got %d", output3.Iteration)
	}
}

// TestGetPhase1Output tests GetPhase1Output method
func TestGetPhase1Output(t *testing.T) {
	mockCaller := &MockAgentCaller{Available: true}
	generator := NewGenerator(mockCaller)

	plannerOutput := &PlannerOutput{
		TaskID:         "TEST-GET",
		BDDRules:       "BDD",
		Architecture:   "Arch",
		APIInterfaces:  "API",
	}

	ctx := context.Background()
	_, _ = generator.ExecutePhase1(ctx, "TEST-GET", plannerOutput)

	output := generator.GetPhase1Output("TEST-GET")
	if output == nil {
		t.Fatal("expected non-nil output")
	}
	if output.TaskID != "TEST-GET" {
		t.Errorf("expected TaskID 'TEST-GET', got %s", output.TaskID)
	}
}

// TestGetPhase2Output tests GetPhase2Output method
func TestGetPhase2Output(t *testing.T) {
	mockCaller := &MockAgentCaller{Available: true}
	generator := NewGenerator(mockCaller)

	phase1Output := &Phase1Output{
		TaskID:          "TEST-GET2",
		BDDTestScript:   "BDD",
		IntegrationTest: "Integration",
		UnitTest:        "Unit",
	}

	ctx := context.Background()
	_, _ = generator.ExecutePhase2(ctx, "TEST-GET2", phase1Output, "")

	output := generator.GetPhase2Output("TEST-GET2")
	if output == nil {
		t.Fatal("expected non-nil output")
	}
	if output.TaskID != "TEST-GET2" {
		t.Errorf("expected TaskID 'TEST-GET2', got %s", output.TaskID)
	}
}

// TestGetIteration tests GetIteration method
func TestGetIteration(t *testing.T) {
	mockCaller := &MockAgentCaller{Available: true}
	generator := NewGenerator(mockCaller)

	phase1Output := &Phase1Output{
		TaskID:          "TEST-ITER-GET",
		BDDTestScript:   "BDD",
		IntegrationTest: "Integration",
		UnitTest:        "Unit",
	}

	ctx := context.Background()

	// Initial iteration should be 0
	if generator.GetIteration("TEST-ITER-GET") != 0 {
		t.Error("expected initial iteration 0")
	}

	// After first execution
	_, _ = generator.ExecutePhase2(ctx, "TEST-ITER-GET", phase1Output, "")
	if generator.GetIteration("TEST-ITER-GET") != 1 {
		t.Errorf("expected iteration 1, got %d", generator.GetIteration("TEST-ITER-GET"))
	}

	// After second execution
	_, _ = generator.ExecutePhase2(ctx, "TEST-ITER-GET", phase1Output, "Failure")
	if generator.GetIteration("TEST-ITER-GET") != 2 {
		t.Errorf("expected iteration 2, got %d", generator.GetIteration("TEST-ITER-GET"))
	}
}