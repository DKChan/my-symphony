// Package config_test 测试配置解析
package config_test

import (
	"os"
	"testing"

	"github.com/dministrator/symphony/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	if cfg.Tracker.Kind != "mock" {
		t.Errorf("expected tracker kind 'mock', got %s", cfg.Tracker.Kind)
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
			"kind":          "github",
			"api_key":       "$TEST_API_KEY",
			"repo":          "owner/repo",
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

	if cfg.Tracker.Repo != "owner/repo" {
		t.Errorf("expected repo 'owner/repo', got %s", cfg.Tracker.Repo)
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
					Kind:    "github",
					APIKey:  "test-key",
					Repo:    "owner/repo",
				},
				Codex: config.CodexConfig{
					Command: "codex app-server",
				},
				Harness: config.HarnessConfig{
					MaxIterations: 5,
				},
			},
			wantValid: true,
		},
		{
			name: "missing tracker kind",
			config: &config.Config{
				Tracker: config.TrackerConfig{
					APIKey: "test-key",
					Repo:   "owner/repo",
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
					Kind: "github",
					Repo: "owner/repo",
				},
				Codex: config.CodexConfig{
					Command: "codex app-server",
				},
			},
			wantValid: false,
		},
		{
			name: "missing repo",
			config: &config.Config{
				Tracker: config.TrackerConfig{
					Kind:    "github",
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
					Kind:    "jira",
					APIKey:  "test-key",
					Repo:    "owner/repo",
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
				Harness: config.HarnessConfig{
					MaxIterations: 5,
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

// 5.4 测试 Harness 配置解析
func TestDefaultConfig_HarnessDefaults(t *testing.T) {
	cfg := config.DefaultConfig()

	// 测试 Harness 默认值
	if cfg.Harness.MaxIterations != 5 {
		t.Errorf("expected harness.max_iterations 5, got %d", cfg.Harness.MaxIterations)
	}

	if !cfg.Harness.BMAD.Enabled {
		t.Error("expected harness.bmad.enabled to be true by default")
	}

	// 默认 agents 应包含分组
	if len(cfg.Harness.BMAD.Agents.Planner) != 3 {
		t.Errorf("expected harness.bmad.agents.planner to have 3 agents, got %d", len(cfg.Harness.BMAD.Agents.Planner))
	}
	if len(cfg.Harness.BMAD.Agents.Generator) != 2 {
		t.Errorf("expected harness.bmad.agents.generator to have 2 agents, got %d", len(cfg.Harness.BMAD.Agents.Generator))
	}
	if len(cfg.Harness.BMAD.Agents.Evaluator) != 2 {
		t.Errorf("expected harness.bmad.agents.evaluator to have 2 agents, got %d", len(cfg.Harness.BMAD.Agents.Evaluator))
	}
}

func TestParseConfig_HarnessMaxIterations(t *testing.T) {
	tests := []struct {
		name           string
		rawConfig      map[string]any
		expectedValue  int
	}{
		{
			name: "parse max_iterations from int",
			rawConfig: map[string]any{
				"tracker": map[string]any{"kind": "mock"},
				"harness": map[string]any{
					"max_iterations": 10,
				},
			},
			expectedValue: 10,
		},
		{
			name: "parse max_iterations from string",
			rawConfig: map[string]any{
				"tracker": map[string]any{"kind": "mock"},
				"harness": map[string]any{
					"max_iterations": "15",
				},
			},
			expectedValue: 15,
		},
		{
			name: "missing harness uses default",
			rawConfig: map[string]any{
				"tracker": map[string]any{"kind": "mock"},
			},
			expectedValue: 5, // default value
		},
		{
			name: "invalid max_iterations uses default",
			rawConfig: map[string]any{
				"tracker": map[string]any{"kind": "mock"},
				"harness": map[string]any{
					"max_iterations": -1,
				},
			},
			expectedValue: 5, // default value (negative ignored)
		},
		{
			name: "zero max_iterations uses default",
			rawConfig: map[string]any{
				"tracker": map[string]any{"kind": "mock"},
				"harness": map[string]any{
					"max_iterations": 0,
				},
			},
			expectedValue: 5, // default value (zero ignored)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.ParseConfig(tt.rawConfig)
			if err != nil {
				t.Fatalf("failed to parse config: %v", err)
			}

			if cfg.Harness.MaxIterations != tt.expectedValue {
				t.Errorf("expected harness.max_iterations %d, got %d", tt.expectedValue, cfg.Harness.MaxIterations)
			}
		})
	}
}

func TestParseConfig_HarnessBMADEnabled(t *testing.T) {
	tests := []struct {
		name           string
		rawConfig      map[string]any
		expectedValue  bool
	}{
		{
			name: "bmad enabled true",
			rawConfig: map[string]any{
				"tracker": map[string]any{"kind": "mock"},
				"harness": map[string]any{
					"bmad": map[string]any{
						"enabled": true,
					},
				},
			},
			expectedValue: true,
		},
		{
			name: "bmad enabled false",
			rawConfig: map[string]any{
				"tracker": map[string]any{"kind": "mock"},
				"harness": map[string]any{
					"bmad": map[string]any{
						"enabled": false,
					},
				},
			},
			expectedValue: false,
		},
		{
			name: "missing bmad uses default true",
			rawConfig: map[string]any{
				"tracker": map[string]any{"kind": "mock"},
			},
			expectedValue: true, // default is true
		},
		{
			name: "missing enabled field uses default",
			rawConfig: map[string]any{
				"tracker": map[string]any{"kind": "mock"},
				"harness": map[string]any{
					"bmad": map[string]any{},
				},
			},
			expectedValue: true, // default is true
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.ParseConfig(tt.rawConfig)
			if err != nil {
				t.Fatalf("failed to parse config: %v", err)
			}

			if cfg.Harness.BMAD.Enabled != tt.expectedValue {
				t.Errorf("expected harness.bmad.enabled %v, got %v", tt.expectedValue, cfg.Harness.BMAD.Enabled)
			}
		})
	}
}

func TestParseConfig_HarnessBMADAgents(t *testing.T) {
	tests := []struct {
		name              string
		rawConfig         map[string]any
		expectedPlanner   []string
		expectedGenerator []string
		expectedEvaluator []string
	}{
		{
			name: "parse grouped agents",
			rawConfig: map[string]any{
				"tracker": map[string]any{"kind": "mock"},
				"harness": map[string]any{
					"bmad": map[string]any{
						"agents": map[string]any{
							"planner":   []any{"agent-pm", "agent-qa"},
							"generator": []any{"agent-dev"},
							"evaluator": []any{"agent-review"},
						},
					},
				},
			},
			expectedPlanner:   []string{"agent-pm", "agent-qa"},
			expectedGenerator: []string{"agent-dev"},
			expectedEvaluator: []string{"agent-review"},
		},
		{
			name: "partial agents config uses defaults for missing groups",
			rawConfig: map[string]any{
				"tracker": map[string]any{"kind": "mock"},
				"harness": map[string]any{
					"bmad": map[string]any{
						"agents": map[string]any{
							"planner": []any{"agent-pm"},
						},
					},
				},
			},
			expectedPlanner:   []string{"agent-pm"},  // custom
			expectedGenerator: []string{"bmad-agent-qa", "bmad-agent-dev"},  // defaults
			expectedEvaluator: []string{"bmad-code-review", "bmad-editorial-review-prose"},  // defaults
		},
		{
			name: "missing agents field uses defaults",
			rawConfig: map[string]any{
				"tracker": map[string]any{"kind": "mock"},
				"harness": map[string]any{
					"bmad": map[string]any{
						"enabled": true,
					},
				},
			},
			expectedPlanner:   []string{"bmad-agent-pm", "bmad-agent-qa", "bmad-agent-architect"},
			expectedGenerator: []string{"bmad-agent-qa", "bmad-agent-dev"},
			expectedEvaluator: []string{"bmad-code-review", "bmad-editorial-review-prose"},
		},
		{
			name: "missing bmad section uses defaults",
			rawConfig: map[string]any{
				"tracker": map[string]any{"kind": "mock"},
			},
			expectedPlanner:   []string{"bmad-agent-pm", "bmad-agent-qa", "bmad-agent-architect"},
			expectedGenerator: []string{"bmad-agent-qa", "bmad-agent-dev"},
			expectedEvaluator: []string{"bmad-code-review", "bmad-editorial-review-prose"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.ParseConfig(tt.rawConfig)
			if err != nil {
				t.Fatalf("failed to parse config: %v", err)
			}

			// Compare planner agents
			if len(cfg.Harness.BMAD.Agents.Planner) != len(tt.expectedPlanner) {
				t.Errorf("expected %d planner agents, got %d", len(tt.expectedPlanner), len(cfg.Harness.BMAD.Agents.Planner))
			}
			// Compare generator agents
			if len(cfg.Harness.BMAD.Agents.Generator) != len(tt.expectedGenerator) {
				t.Errorf("expected %d generator agents, got %d", len(tt.expectedGenerator), len(cfg.Harness.BMAD.Agents.Generator))
			}
			// Compare evaluator agents
			if len(cfg.Harness.BMAD.Agents.Evaluator) != len(tt.expectedEvaluator) {
				t.Errorf("expected %d evaluator agents, got %d", len(tt.expectedEvaluator), len(cfg.Harness.BMAD.Agents.Evaluator))
			}
		})
	}
}

func TestParseConfig_HarnessComplete(t *testing.T) {
	raw := map[string]any{
		"tracker": map[string]any{
			"kind": "mock",
		},
		"harness": map[string]any{
			"max_iterations": 7,
			"bmad": map[string]any{
				"enabled": true,
				"agents": map[string]any{
					"planner":   []any{"bmad-agent-pm", "bmad-agent-qa"},
					"generator": []any{"bmad-agent-dev"},
					"evaluator": []any{"bmad-code-review"},
				},
			},
		},
	}

	cfg, err := config.ParseConfig(raw)
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	// Verify complete harness config
	if cfg.Harness.MaxIterations != 7 {
		t.Errorf("expected harness.max_iterations 7, got %d", cfg.Harness.MaxIterations)
	}

	if !cfg.Harness.BMAD.Enabled {
		t.Error("expected harness.bmad.enabled to be true")
	}

	// Verify grouped agents
	if len(cfg.Harness.BMAD.Agents.Planner) != 2 {
		t.Errorf("expected 2 planner agents, got %d", len(cfg.Harness.BMAD.Agents.Planner))
	}
	if len(cfg.Harness.BMAD.Agents.Generator) != 1 {
		t.Errorf("expected 1 generator agent, got %d", len(cfg.Harness.BMAD.Agents.Generator))
	}
	if len(cfg.Harness.BMAD.Agents.Evaluator) != 1 {
		t.Errorf("expected 1 evaluator agent, got %d", len(cfg.Harness.BMAD.Agents.Evaluator))
	}
}