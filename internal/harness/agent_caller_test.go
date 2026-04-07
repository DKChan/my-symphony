// Package harness 提供 P-G-E 编排引擎
package harness

import (
	"context"
	"testing"
	"time"
)

// TestAgentCallerInterface tests that AgentCaller interface is properly defined
func TestAgentCallerInterface(t *testing.T) {
	// This test verifies the interface exists and has correct method signatures
	var _ AgentCaller = (*AgentCallerImpl)(nil)
}

// TestAgentInputStructure tests AgentInput structure
func TestAgentInputStructure(t *testing.T) {
	input := AgentInput{
		AgentName:  "bmad-agent-pm",
		Task:       "设计用户认证模块",
		Context:    map[string]string{"project": "symphony"},
		WorkingDir: "/tmp/workspace",
	}

	if input.AgentName != "bmad-agent-pm" {
		t.Errorf("expected AgentName 'bmad-agent-pm', got %s", input.AgentName)
	}
	if input.Task != "设计用户认证模块" {
		t.Errorf("expected Task '设计用户认证模块', got %s", input.Task)
	}
	if input.Context["project"] != "symphony" {
		t.Errorf("expected Context[project] 'symphony', got %s", input.Context["project"])
	}
	if input.WorkingDir != "/tmp/workspace" {
		t.Errorf("expected WorkingDir '/tmp/workspace', got %s", input.WorkingDir)
	}
}

// TestAgentOutputStructure tests AgentOutput structure
func TestAgentOutputStructure(t *testing.T) {
	output := AgentOutput{
		Success:  true,
		Content:  "任务完成",
		Duration: 5 * time.Second,
		Error:    "",
	}

	if !output.Success {
		t.Errorf("expected Success true, got %v", output.Success)
	}
	if output.Content != "任务完成" {
		t.Errorf("expected Content '任务完成', got %s", output.Content)
	}
	if output.Duration != 5*time.Second {
		t.Errorf("expected Duration 5s, got %v", output.Duration)
	}
}

// TestNewAgentCaller tests constructor
func TestNewAgentCaller(t *testing.T) {
	caller := NewAgentCaller("claude", 30*time.Second)
	if caller == nil {
		t.Fatal("expected non-nil AgentCaller")
	}
}

// TestCallWithInvalidAgent tests calling with invalid agent name
func TestCallWithInvalidAgent(t *testing.T) {
	caller := NewAgentCaller("nonexistent-cli", 1*time.Second)

	input := &AgentInput{
		AgentName:  "bmad-agent-pm",
		Task:       "test task",
		WorkingDir: t.TempDir(),
	}

	ctx := context.Background()
	_, err := caller.Call(ctx, input)
	if err == nil {
		t.Error("expected error for nonexistent CLI")
	}
}

// TestCallWithTimeout tests timeout handling
func TestCallWithTimeout(t *testing.T) {
	caller := NewAgentCaller("claude", 1*time.Nanosecond) // Very short timeout

	input := &AgentInput{
		AgentName:  "bmad-agent-pm",
		Task:       "test task",
		WorkingDir: t.TempDir(),
	}

	ctx := context.Background()
	_, err := caller.Call(ctx, input)
	if err == nil {
		t.Error("expected timeout error")
	}
}

// TestCheckAvailability tests availability check
func TestCheckAvailability(t *testing.T) {
	// Test with nonexistent CLI
	caller := NewAgentCaller("nonexistent-cli-12345", 1*time.Second)
	err := caller.CheckAvailability()
	if err == nil {
		t.Error("expected error for nonexistent CLI")
	}
}