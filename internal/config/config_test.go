// Package config_test 测试配置解析
package config_test

import (
	"os"
	"testing"

	"github.com/dministrator/symphony/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	if cfg.Tracker.Kind != "linear" {
		t.Errorf("expected tracker kind 'linear', got %s", cfg.Tracker.Kind)
	}

	if cfg.Polling.IntervalMs != 30000 {
		t.Errorf("expected polling interval 30000, got %d", cfg.Polling.IntervalMs)
	}

	if cfg.Agent.MaxConcurrentAgents != 10 {
		t.Errorf("expected max concurrent agents 10, got %d", cfg.Agent.MaxConcurrentAgents)
	}

	// 测试新配置字段的默认值
	if cfg.Clarification.MaxRounds != 5 {
		t.Errorf("expected clarification.max_rounds 5, got %d", cfg.Clarification.MaxRounds)
	}

	if cfg.Execution.MaxRetries != 3 {
		t.Errorf("expected execution.max_retries 3, got %d", cfg.Execution.MaxRetries)
	}
}

func TestParseConfig(t *testing.T) {
	raw := map[string]any{
		"tracker": map[string]any{
			"kind":          "linear",
			"api_key":       "$TEST_API_KEY",
			"project_slug":  "TEST-PROJECT",
			"active_states": []any{"Todo", "In Progress"},
		},
		"polling": map[string]any{
			"interval_ms": "60000",
		},
		"agent": map[string]any{
			"max_concurrent_agents": 5,
		},
	}

	// 设置测试环境变量
	os.Setenv("TEST_API_KEY", "test-key-123")
	defer os.Unsetenv("TEST_API_KEY")

	cfg, err := config.ParseConfig(raw)
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	if cfg.Tracker.APIKey != "test-key-123" {
		t.Errorf("expected resolved API key 'test-key-123', got %s", cfg.Tracker.APIKey)
	}

	if cfg.Tracker.ProjectSlug != "TEST-PROJECT" {
		t.Errorf("expected project slug 'TEST-PROJECT', got %s", cfg.Tracker.ProjectSlug)
	}

	if cfg.Polling.IntervalMs != 60000 {
		t.Errorf("expected polling interval 60000, got %d", cfg.Polling.IntervalMs)
	}

	if cfg.Agent.MaxConcurrentAgents != 5 {
		t.Errorf("expected max concurrent agents 5, got %d", cfg.Agent.MaxConcurrentAgents)
	}
}

func TestValidateDispatchConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Config
		wantValid bool
	}{
		{
			name: "valid config",
			config: &config.Config{
				Tracker: config.TrackerConfig{
					Kind:         "linear",
					APIKey:       "test-key",
					ProjectSlug:  "TEST",
				},
				Codex: config.CodexConfig{
					Command: "codex app-server",
				},
			},
			wantValid: true,
		},
		{
			name: "missing tracker kind",
			config: &config.Config{
				Tracker: config.TrackerConfig{
					APIKey:      "test-key",
					ProjectSlug: "TEST",
				},
				Codex: config.CodexConfig{
					Command: "codex app-server",
				},
			},
			wantValid: false,
		},
		{
			name: "missing api key",
			config: &config.Config{
				Tracker: config.TrackerConfig{
					Kind:        "linear",
					ProjectSlug: "TEST",
				},
				Codex: config.CodexConfig{
					Command: "codex app-server",
				},
			},
			wantValid: false,
		},
		{
			name: "missing project slug",
			config: &config.Config{
				Tracker: config.TrackerConfig{
					Kind:    "linear",
					APIKey:  "test-key",
				},
				Codex: config.CodexConfig{
					Command: "codex app-server",
				},
			},
			wantValid: false,
		},
		{
			name: "unsupported tracker kind",
			config: &config.Config{
				Tracker: config.TrackerConfig{
					Kind:         "jira",
					APIKey:       "test-key",
					ProjectSlug:  "TEST",
				},
				Codex: config.CodexConfig{
					Command: "codex app-server",
				},
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validation := tt.config.ValidateDispatchConfig()
			if validation.Valid != tt.wantValid {
				t.Errorf("expected valid=%v, got valid=%v, errors=%v", tt.wantValid, validation.Valid, validation.Errors)
			}
		})
	}
}

func TestIsActiveState(t *testing.T) {
	cfg := config.DefaultConfig()

	if !cfg.IsActiveState("Todo") {
		t.Error("expected 'Todo' to be active state")
	}

	if !cfg.IsActiveState("In Progress") {
		t.Error("expected 'In Progress' to be active state")
	}

	if cfg.IsActiveState("Done") {
		t.Error("expected 'Done' to not be active state")
	}

	// 测试大小写不敏感
	if !cfg.IsActiveState("todo") {
		t.Error("expected 'todo' (lowercase) to be active state")
	}
}

func TestIsTerminalState(t *testing.T) {
	cfg := config.DefaultConfig()

	if !cfg.IsTerminalState("Done") {
		t.Error("expected 'Done' to be terminal state")
	}

	if !cfg.IsTerminalState("Cancelled") {
		t.Error("expected 'Cancelled' to be terminal state")
	}

	if cfg.IsTerminalState("Todo") {
		t.Error("expected 'Todo' to not be terminal state")
	}
}

func TestSanitizeWorkspaceKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ABC-123", "ABC-123"},
		{"ABC/123", "ABC_123"},
		{"ABC 123", "ABC_123"},
		{"ABC@#$%123", "ABC____123"},
		{"test.issue", "test.issue"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := config.SanitizeWorkspaceKey(tt.input)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestParseConfigClarificationAndExecution(t *testing.T) {
	raw := map[string]any{
		"tracker": map[string]any{
			"kind": "mock",
		},
		"clarification": map[string]any{
			"max_rounds": 10,
		},
		"execution": map[string]any{
			"max_retries": 5,
		},
	}

	cfg, err := config.ParseConfig(raw)
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	if cfg.Clarification.MaxRounds != 10 {
		t.Errorf("expected clarification.max_rounds 10, got %d", cfg.Clarification.MaxRounds)
	}

	if cfg.Execution.MaxRetries != 5 {
		t.Errorf("expected execution.max_retries 5, got %d", cfg.Execution.MaxRetries)
	}
}

func TestValidateSymphonyConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *config.Config
		wantValid bool
	}{
		{
			name: "valid mock tracker config",
			config: &config.Config{
				Tracker: config.TrackerConfig{
					Kind: "mock",
				},
				Agent: config.AgentConfig{
					Kind: "codex",
				},
				Codex: config.CodexConfig{
					Command: "echo", // Use echo command which is always available
				},
				Workspace: config.WorkspaceConfig{
					Root: "/tmp/workspaces",
				},
				Clarification: config.ClarificationConfig{
					MaxRounds: 5,
				},
				Execution: config.ExecutionConfig{
					MaxRetries: 3,
				},
			},
			wantValid: true,
		},
		{
			name: "invalid tracker kind",
			config: &config.Config{
				Tracker: config.TrackerConfig{
					Kind: "invalid_tracker",
				},
				Agent: config.AgentConfig{
					Kind: "codex",
				},
				Workspace: config.WorkspaceConfig{
					Root: "/tmp/workspaces",
				},
				Clarification: config.ClarificationConfig{
					MaxRounds: 5,
				},
				Execution: config.ExecutionConfig{
					MaxRetries: 3,
				},
			},
			wantValid: false,
		},
		{
			name: "missing workspace root",
			config: &config.Config{
				Tracker: config.TrackerConfig{
					Kind: "mock",
				},
				Agent: config.AgentConfig{
					Kind: "codex",
				},
				Workspace: config.WorkspaceConfig{
					Root: "",
				},
				Clarification: config.ClarificationConfig{
					MaxRounds: 5,
				},
				Execution: config.ExecutionConfig{
					MaxRetries: 3,
				},
			},
			wantValid: false,
		},
		{
			name: "invalid clarification max_rounds",
			config: &config.Config{
				Tracker: config.TrackerConfig{
					Kind: "mock",
				},
				Agent: config.AgentConfig{
					Kind: "codex",
				},
				Workspace: config.WorkspaceConfig{
					Root: "/tmp/workspaces",
				},
				Clarification: config.ClarificationConfig{
					MaxRounds: 0,
				},
				Execution: config.ExecutionConfig{
					MaxRetries: 3,
				},
			},
			wantValid: false,
		},
		{
			name: "negative execution max_retries",
			config: &config.Config{
				Tracker: config.TrackerConfig{
					Kind: "mock",
				},
				Agent: config.AgentConfig{
					Kind: "codex",
				},
				Workspace: config.WorkspaceConfig{
					Root: "/tmp/workspaces",
				},
				Clarification: config.ClarificationConfig{
					MaxRounds: 5,
				},
				Execution: config.ExecutionConfig{
					MaxRetries: -1,
				},
			},
			wantValid: false,
		},
		{
			name: "invalid agent kind",
			config: &config.Config{
				Tracker: config.TrackerConfig{
					Kind: "mock",
				},
				Agent: config.AgentConfig{
					Kind: "invalid_agent",
				},
				Workspace: config.WorkspaceConfig{
					Root: "/tmp/workspaces",
				},
				Clarification: config.ClarificationConfig{
					MaxRounds: 5,
				},
				Execution: config.ExecutionConfig{
					MaxRetries: 3,
				},
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validation := tt.config.ValidateSymphonyConfig()
			if validation.Valid != tt.wantValid {
				t.Errorf("expected valid=%v, got valid=%v, errors=%v", tt.wantValid, validation.Valid, validation.Errors)
			}
		})
	}
}