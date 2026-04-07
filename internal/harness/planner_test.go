// Package harness 提供 P-G-E 编排引擎
package harness

import (
	"context"
	"testing"
	"time"

	"github.com/dministrator/symphony/internal/common/errors"
)

// MockAgentCaller 用于测试的 Mock AgentCaller
type MockAgentCaller struct {
	// CallFunc 自定义 Call 函数
	CallFunc func(ctx context.Context, input *AgentInput) (*AgentOutput, error)
	// Available 是否可用
	Available bool
}

// Call 实现 AgentCaller 接口
func (m *MockAgentCaller) Call(ctx context.Context, input *AgentInput) (*AgentOutput, error) {
	if m.CallFunc != nil {
		return m.CallFunc(ctx, input)
	}
	// 默认返回成功
	return &AgentOutput{
		Success:  true,
		Content:  "Feature: Test\n  Scenario: Mock test",
		Duration: 100 * time.Millisecond,
	}, nil
}

// CheckAvailability 实现 AgentCaller 接口
func (m *MockAgentCaller) CheckAvailability() error {
	if !m.Available {
		return errors.ErrAgentUnavailable
	}
	return nil
}

// TestPlannerInterface tests that Planner interface is properly defined
func TestPlannerInterface(t *testing.T) {
	var _ Planner = (*PlannerImpl)(nil)
}

// TestPlannerOutputStructure tests PlannerOutput structure
func TestPlannerOutputStructure(t *testing.T) {
	output := &PlannerOutput{
		TaskID:         "TEST-001",
		BDDRules:       "Feature: Login\n  Scenario: User logs in",
		DomainModel:    "User, Session, Auth",
		Architecture:   "Clean Architecture",
		APIInterfaces:  "POST /api/login",
		CreatedAt:      time.Now(),
		Immutable:      true,
	}

	if output.TaskID != "TEST-001" {
		t.Errorf("expected TaskID 'TEST-001', got %s", output.TaskID)
	}
	if output.BDDRules == "" {
		t.Error("expected non-empty BDDRules")
	}
	if !output.Immutable {
		t.Error("expected Immutable to be true")
	}
}

// TestNewPlanner tests constructor
func TestNewPlanner(t *testing.T) {
	mockCaller := &MockAgentCaller{Available: true}
	planner := NewPlanner(mockCaller)
	if planner == nil {
		t.Fatal("expected non-nil Planner")
	}
}

// TestPlannerExecute tests the Execute method
func TestPlannerExecute(t *testing.T) {
	mockCaller := &MockAgentCaller{Available: true}
	planner := NewPlanner(mockCaller)

	ctx := context.Background()
	output, err := planner.Execute(ctx, "TEST-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.TaskID != "TEST-001" {
		t.Errorf("expected TaskID 'TEST-001', got %s", output.TaskID)
	}
	if !output.Immutable {
		t.Error("expected output to be immutable")
	}
}

// TestPlannerGetOutput tests GetOutput method
func TestPlannerGetOutput(t *testing.T) {
	mockCaller := &MockAgentCaller{Available: true}
	planner := NewPlanner(mockCaller)

	ctx := context.Background()
	_, _ = planner.Execute(ctx, "TEST-001")

	output := planner.GetOutput("TEST-001")
	if output == nil {
		t.Fatal("expected non-nil output")
	}
	if output.TaskID != "TEST-001" {
		t.Errorf("expected TaskID 'TEST-001', got %s", output.TaskID)
	}
}

// TestPlannerHasOutput tests HasOutput method
func TestPlannerHasOutput(t *testing.T) {
	mockCaller := &MockAgentCaller{Available: true}
	planner := NewPlanner(mockCaller)

	if planner.HasOutput("TEST-001") {
		t.Error("expected no output for TEST-001")
	}

	ctx := context.Background()
	_, _ = planner.Execute(ctx, "TEST-001")

	if !planner.HasOutput("TEST-001") {
		t.Error("expected output for TEST-001")
	}
}

// TestPlannerSetOutput tests SetOutput method
func TestPlannerSetOutput(t *testing.T) {
	mockCaller := &MockAgentCaller{Available: true}
	planner := NewPlanner(mockCaller)

	output := &PlannerOutput{
		TaskID:    "TEST-002",
		CreatedAt: time.Now(),
	}
	planner.SetOutput("TEST-002", output)

	if !planner.HasOutput("TEST-002") {
		t.Error("expected output for TEST-002")
	}

	retrieved := planner.GetOutput("TEST-002")
	if retrieved == nil {
		t.Fatal("expected non-nil output")
	}
	if !retrieved.Immutable {
		t.Error("expected output to be immutable after SetOutput")
	}
}

// TestPlannerIdempotent tests that Execute is idempotent
func TestPlannerIdempotent(t *testing.T) {
	mockCaller := &MockAgentCaller{Available: true}
	planner := NewPlanner(mockCaller)

	ctx := context.Background()
	output1, _ := planner.Execute(ctx, "TEST-001")
	output2, _ := planner.Execute(ctx, "TEST-001")

	// Should return the same output
	if output1.CreatedAt != output2.CreatedAt {
		t.Error("expected same output on repeated Execute calls")
	}
}

// TestGenerateBDDRules tests BDD rules generation
func TestGenerateBDDRules(t *testing.T) {
	mockCaller := &MockAgentCaller{
		Available: true,
		CallFunc: func(ctx context.Context, input *AgentInput) (*AgentOutput, error) {
			// 验证调用了正确的 agent
			if input.AgentName != "bmad-agent-qa" {
				t.Errorf("expected agent name 'bmad-agent-qa', got '%s'", input.AgentName)
			}
			return &AgentOutput{
				Success:  true,
				Content:  "Feature: Login\n  Scenario: User logs in successfully\n    Given a registered user\n    When they enter valid credentials\n    Then they should be logged in",
				Duration: 100 * time.Millisecond,
			}, nil
		},
	}
	planner := NewPlanner(mockCaller)

	ctx := context.Background()
	_, _ = planner.Execute(ctx, "TEST-001")

	// Generate BDD rules
	requirements := "用户登录功能:\n- 用户可以输入用户名和密码\n- 系统验证用户身份\n- 登录成功后跳转到主页"
	bddRules, err := planner.GenerateBDDRules(ctx, "TEST-001", requirements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if bddRules == "" {
		t.Error("expected non-empty BDD rules")
	}

	// Verify output was updated
	output := planner.GetOutput("TEST-001")
	if output == nil {
		t.Fatal("expected non-nil output")
	}
	if output.BDDRules != bddRules {
		t.Error("expected BDDRules to match generated rules")
	}
}

// TestGenerateBDDRulesNoOutput tests BDD generation without existing output
func TestGenerateBDDRulesNoOutput(t *testing.T) {
	mockCaller := &MockAgentCaller{
		Available: true,
		CallFunc: func(ctx context.Context, input *AgentInput) (*AgentOutput, error) {
			return &AgentOutput{
				Success:  true,
				Content:  "Feature: Test Feature",
				Duration: 50 * time.Millisecond,
			}, nil
		},
	}
	planner := NewPlanner(mockCaller)

	ctx := context.Background()

	// Generate BDD rules without first calling Execute
	requirements := "测试需求"
	bddRules, err := planner.GenerateBDDRules(ctx, "TEST-NO-OUTPUT", requirements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should still return generated rules
	if bddRules == "" {
		t.Error("expected non-empty BDD rules even without output")
	}
}

// TestGenerateBDDRulesOutputImmutable tests that BDD rules generation preserves immutable flag
func TestGenerateBDDRulesOutputImmutable(t *testing.T) {
	mockCaller := &MockAgentCaller{
		Available: true,
	}
	planner := NewPlanner(mockCaller)

	ctx := context.Background()
	_, _ = planner.Execute(ctx, "TEST-001")

	// Generate BDD rules
	_, _ = planner.GenerateBDDRules(ctx, "TEST-001", "需求描述")

	// Verify output is still immutable
	output := planner.GetOutput("TEST-001")
	if output == nil {
		t.Fatal("expected non-nil output")
	}
	if !output.Immutable {
		t.Error("expected output to remain immutable after BDD generation")
	}
}

// TestGenerateBDDRulesAgentError tests error handling when agent fails
func TestGenerateBDDRulesAgentError(t *testing.T) {
	mockCaller := &MockAgentCaller{
		Available: true,
		CallFunc: func(ctx context.Context, input *AgentInput) (*AgentOutput, error) {
			return nil, errors.ErrAgentExecutionFail
		},
	}
	planner := NewPlanner(mockCaller)

	ctx := context.Background()
	_, _ = planner.Execute(ctx, "TEST-001")

	// Generate BDD rules - should fail
	requirements := "测试需求"
	_, err := planner.GenerateBDDRules(ctx, "TEST-001", requirements)
	if err == nil {
		t.Error("expected error when agent fails")
	}
}

// TestGenerateDomainModel tests domain model generation
func TestGenerateDomainModel(t *testing.T) {
	mockCaller := &MockAgentCaller{
		Available: true,
		CallFunc: func(ctx context.Context, input *AgentInput) (*AgentOutput, error) {
			// 验证调用了正确的 agent
			if input.AgentName != "bmad-agent-architect" {
				t.Errorf("expected agent name 'bmad-agent-architect', got '%s'", input.AgentName)
			}
			return &AgentOutput{
				Success:  true,
				Content:  "Domain Model:\n- User: 用户实体\n- Session: 会话实体\n- AuthService: 认证服务",
				Duration: 150 * time.Millisecond,
			}, nil
		},
	}
	planner := NewPlanner(mockCaller)

	ctx := context.Background()
	_, _ = planner.Execute(ctx, "TEST-001")

	// Generate domain model
	bddRules := "Feature: Login\n  Scenario: User logs in"
	domainModel, err := planner.GenerateDomainModel(ctx, "TEST-001", bddRules)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if domainModel == "" {
		t.Error("expected non-empty domain model")
	}

	// Verify output was updated
	output := planner.GetOutput("TEST-001")
	if output == nil {
		t.Fatal("expected non-nil output")
	}
	if output.DomainModel != domainModel {
		t.Error("expected DomainModel to match generated model")
	}
}

// TestGenerateDomainModelAgentError tests error handling
func TestGenerateDomainModelAgentError(t *testing.T) {
	mockCaller := &MockAgentCaller{
		Available: true,
		CallFunc: func(ctx context.Context, input *AgentInput) (*AgentOutput, error) {
			return nil, errors.ErrAgentTimeout
		},
	}
	planner := NewPlanner(mockCaller)

	ctx := context.Background()
	_, _ = planner.Execute(ctx, "TEST-001")

	_, err := planner.GenerateDomainModel(ctx, "TEST-001", "BDD rules")
	if err == nil {
		t.Error("expected error when agent fails")
	}
}

// TestGenerateArchitecture tests architecture design generation
func TestGenerateArchitecture(t *testing.T) {
	mockCaller := &MockAgentCaller{
		Available: true,
		CallFunc: func(ctx context.Context, input *AgentInput) (*AgentOutput, error) {
			if input.AgentName != "bmad-agent-architect" {
				t.Errorf("expected agent name 'bmad-agent-architect', got '%s'", input.AgentName)
			}
			if input.Context["phase"] != "architecture_design" {
				t.Errorf("expected phase 'architecture_design', got '%s'", input.Context["phase"])
			}
			return &AgentOutput{
				Success:  true,
				Content:  "Architecture: Clean Architecture\n- Domain Layer\n- Application Layer\n- Infrastructure Layer",
				Duration: 200 * time.Millisecond,
			}, nil
		},
	}
	planner := NewPlanner(mockCaller)

	ctx := context.Background()
	_, _ = planner.Execute(ctx, "TEST-001")

	architecture, err := planner.GenerateArchitecture(ctx, "TEST-001", "Domain Model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if architecture == "" {
		t.Error("expected non-empty architecture")
	}

	output := planner.GetOutput("TEST-001")
	if output.Architecture != architecture {
		t.Error("expected Architecture to match generated design")
	}
}

// TestGenerateArchitectureAgentError tests error handling
func TestGenerateArchitectureAgentError(t *testing.T) {
	mockCaller := &MockAgentCaller{
		Available: true,
		CallFunc: func(ctx context.Context, input *AgentInput) (*AgentOutput, error) {
			return nil, errors.ErrAgentExecutionFail
		},
	}
	planner := NewPlanner(mockCaller)

	ctx := context.Background()
	_, _ = planner.Execute(ctx, "TEST-001")

	_, err := planner.GenerateArchitecture(ctx, "TEST-001", "Domain Model")
	if err == nil {
		t.Error("expected error when agent fails")
	}
}

// TestGenerateAPIInterfaces tests API interfaces generation
func TestGenerateAPIInterfaces(t *testing.T) {
	mockCaller := &MockAgentCaller{
		Available: true,
		CallFunc: func(ctx context.Context, input *AgentInput) (*AgentOutput, error) {
			if input.AgentName != "bmad-agent-architect" {
				t.Errorf("expected agent name 'bmad-agent-architect', got '%s'", input.AgentName)
			}
			if input.Context["phase"] != "api_design" {
				t.Errorf("expected phase 'api_design', got '%s'", input.Context["phase"])
			}
			return &AgentOutput{
				Success:  true,
				Content:  "API Interfaces:\nPOST /api/login\nPOST /api/logout\nGET /api/session",
				Duration: 180 * time.Millisecond,
			}, nil
		},
	}
	planner := NewPlanner(mockCaller)

	ctx := context.Background()
	_, _ = planner.Execute(ctx, "TEST-001")

	apiInterfaces, err := planner.GenerateAPIInterfaces(ctx, "TEST-001", "Architecture Design")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if apiInterfaces == "" {
		t.Error("expected non-empty API interfaces")
	}

	output := planner.GetOutput("TEST-001")
	if output.APIInterfaces != apiInterfaces {
		t.Error("expected APIInterfaces to match generated interfaces")
	}
}

// TestGenerateAPIInterfacesAgentError tests error handling
func TestGenerateAPIInterfacesAgentError(t *testing.T) {
	mockCaller := &MockAgentCaller{
		Available: true,
		CallFunc: func(ctx context.Context, input *AgentInput) (*AgentOutput, error) {
			return nil, errors.ErrAgentTimeout
		},
	}
	planner := NewPlanner(mockCaller)

	ctx := context.Background()
	_, _ = planner.Execute(ctx, "TEST-001")

	_, err := planner.GenerateAPIInterfaces(ctx, "TEST-001", "Architecture Design")
	if err == nil {
		t.Error("expected error when agent fails")
	}
}

// TestFullPlannerFlow tests complete P1-P5 flow
func TestFullPlannerFlow(t *testing.T) {
	mockCaller := &MockAgentCaller{
		Available: true,
		CallFunc: func(ctx context.Context, input *AgentInput) (*AgentOutput, error) {
			// Return different content based on phase
			switch input.Context["phase"] {
			case "bdd_generation":
				return &AgentOutput{Success: true, Content: "Feature: Login\n  Scenario: User logs in"}, nil
			case "domain_modeling":
				return &AgentOutput{Success: true, Content: "Domain: User, Session, AuthService"}, nil
			case "architecture_design":
				return &AgentOutput{Success: true, Content: "Architecture: Clean Architecture"}, nil
			case "api_design":
				return &AgentOutput{Success: true, Content: "API: POST /api/login"}, nil
			default:
				return &AgentOutput{Success: true, Content: "Default output"}, nil
			}
		},
	}
	planner := NewPlanner(mockCaller)

	ctx := context.Background()
	taskID := "FULL-TEST"

	// Execute full planner flow
	_, _ = planner.Execute(ctx, taskID)
	_, _ = planner.GenerateBDDRules(ctx, taskID, "Requirements")
	_, _ = planner.GenerateDomainModel(ctx, taskID, "BDD Rules")
	_, _ = planner.GenerateArchitecture(ctx, taskID, "Domain Model")
	_, _ = planner.GenerateAPIInterfaces(ctx, taskID, "Architecture")

	// Verify all outputs are populated
	output := planner.GetOutput(taskID)
	if output == nil {
		t.Fatal("expected non-nil output")
	}
	if output.BDDRules == "" {
		t.Error("expected BDDRules to be populated")
	}
	if output.DomainModel == "" {
		t.Error("expected DomainModel to be populated")
	}
	if output.Architecture == "" {
		t.Error("expected Architecture to be populated")
	}
	if output.APIInterfaces == "" {
		t.Error("expected APIInterfaces to be populated")
	}
	if !output.Immutable {
		t.Error("expected output to be immutable")
	}
}