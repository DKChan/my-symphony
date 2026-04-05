// Package config_test 配置解析和验证的补充测试
package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dministrator/symphony/internal/config"
)

// TestResolveEnvVar 测试环境变量解析
func TestResolveEnvVar(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		setup    func()
		cleanup  func()
		expected string
	}{
		{
			name:     "环境变量存在",
			input:    "$TEST_VAR",
			setup:    func() { os.Setenv("TEST_VAR", "test-value") },
			cleanup:  func() { os.Unsetenv("TEST_VAR") },
			expected: "test-value",
		},
		{
			name:     "环境变量不存在",
			input:    "$NON_EXISTENT_VAR",
			setup:    func() {},
			cleanup:  func() {},
			expected: "",
		},
		{
			name:     "无环境变量前缀",
			input:    "plain-text",
			setup:    func() {},
			cleanup:  func() {},
			expected: "plain-text",
		},
		{
			name:     "带$符号但非环境变量",
			input:    "$100",
			setup:    func() {},
			cleanup:  func() {},
			expected: "", // $100 会被当作环境变量名，而不是 "100"
		},
		{
			name:     "空字符串",
			input:    "",
			setup:    func() {},
			cleanup:  func() {},
			expected: "",
		},
		{
			name:     "多个环境变量（只解析第一个）",
			input:    "$VAR1$VAR2",
			setup:    func() { os.Setenv("VAR1", "value1") },
			cleanup:  func() { os.Unsetenv("VAR1") },
			expected: "",
		},
		{
			name:     "环境变量值为空",
			input:    "$EMPTY_VAR",
			setup:    func() { os.Setenv("EMPTY_VAR", "") },
			cleanup:  func() { os.Unsetenv("EMPTY_VAR") },
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			if tt.cleanup != nil {
				defer tt.cleanup()
			}

			result := resolveEnvVar(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// 辅助函数：访问 resolveEnvVar（未导出）
func resolveEnvVar(s string) string {
	// 通过 ParseConfig 间接测试
	raw := map[string]any{
		"tracker": map[string]any{
			"api_key": s,
		},
	}
	cfg, err := config.ParseConfig(raw)
	if err != nil {
		return ""
	}
	return cfg.Tracker.APIKey
}

// TestExpandPath 测试路径展开
func TestExpandPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "带 ~ 前缀",
			input:    "~/test",
			expected: filepath.Join(os.Getenv("HOME"), "/test"),
		},
		{
			name:     "带 ~ 后跟路径分隔符",
			input:    "~/Documents/work",
			expected: filepath.Join(os.Getenv("HOME"), "/Documents/work"),
		},
		{
			name:     "绝对路径",
			input:    "/absolute/path",
			expected: "/absolute/path",
		},
		{
			name:     "相对路径",
			input:    "relative/path",
			expected: "relative/path",
		},
		{
			name:     "空字符串",
			input:    "",
			expected: "",
		},
		{
			name:     "只有 ~ 符号",
			input:    "~",
			expected: filepath.Join(os.Getenv("HOME"), "/"),
		},
		{
			name:     "普通字符串",
			input:    "normal-string",
			expected: "normal-string",
		},
		{
			name:     "路径包含 $ 符号（不展开）",
			input:    "/path/$VAR/test",
			expected: "/path/$VAR/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPath(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// 辅助函数：访问 expandPath（未导出）
func expandPath(path string) string {
	// 通过 ParseConfig 间接测试
	raw := map[string]any{
		"workspace": map[string]any{
			"root": path,
		},
	}
	cfg, err := config.ParseConfig(raw)
	if err != nil {
		return ""
	}
	return cfg.Workspace.Root
}

// TestParseStringList 测试字符串列表解析
func TestParseStringList(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected []string
	}{
		{
			name:     "字符串数组",
			input:    []any{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "逗号分隔字符串",
			input:    "a,b,c",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "带空格的逗号分隔字符串",
			input:    "a, b, c",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "空字符串",
			input:    "",
			expected: nil, // 空字符串会解析为空数组，不会覆盖默认值
		},
		{
			name:     "nil",
			input:    nil,
			expected: nil, // nil 不会覆盖默认值
		},
		{
			name:     "空数组",
			input:    []any{},
			expected: nil, // 空数组不会覆盖默认值
		},
		{
			name:     "数组包含非字符串元素",
			input:    []any{"a", 123, "c"},
			expected: []string{"a", "c"},
		},
		{
			name:     "单个字符串",
			input:    "single",
			expected: []string{"single"},
		},
		{
			name:     "逗号分隔包含空元素",
			input:    "a,,c",
			expected: []string{"a", "c"},
		},
		{
			name:     "逗号分隔只有空格",
			input:    "a,  ,c",
			expected: []string{"a", "c"},
		},
		{
			name:     "整数类型",
			input:    123,
			expected: nil,
		},
		{
			name:     "map 类型",
			input:    map[string]any{"key": "value"},
			expected: nil,
		},
		{
			name:     "bool 类型",
			input:    true,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 通过 ParseConfig 间接测试 parseStringList
			raw := map[string]any{
				"tracker": map[string]any{
					"active_states": tt.input,
				},
			}
			cfg, _ := config.ParseConfig(raw)

			result := cfg.Tracker.ActiveStates

			// 如果期望是 nil 或空数组，应该保持默认值（2个元素）
			if tt.expected == nil || len(tt.expected) == 0 {
				// 应该保持默认值
				if len(result) != 2 {
					t.Errorf("expected to keep default values (2 elements), got %d elements: %v", len(result), result)
				}
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected len=%d, got %d", len(tt.expected), len(result))
				return
			}

			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("at index %d: expected %q, got %q", i, tt.expected[i], v)
				}
			}
		})
	}
}

// TestParseInt 测试整数解析
func TestParseInt(t *testing.T) {
	tests := []struct {
		name       string
		input      interface{}
		expected   int64
		expectedOK bool
	}{
		{
			name:       "int 类型",
			input:      123,
			expected:   123,
			expectedOK: true,
		},
		{
			name:       "int64 类型",
			input:      int64(456),
			expected:   456,
			expectedOK: true,
		},
		{
			name:       "float64 类型",
			input:      789.0,
			expected:   789,
			expectedOK: true,
		},
		{
			name:       "字符串数字",
			input:      "100",
			expected:   100,
			expectedOK: true,
		},
		{
			name:       "字符串数字带符号",
			input:      "-50",
			expected:   -50,
			expectedOK: true,
		},
		{
			name:       "字符串零",
			input:      "0",
			expected:   0,
			expectedOK: true,
		},
		{
			name:       "空字符串",
			input:      "",
			expected:   0,
			expectedOK: false,
		},
		{
			name:       "无效字符串",
			input:      "abc",
			expected:   0,
			expectedOK: false,
		},
		{
			name:       "nil",
			input:      nil,
			expected:   0,
			expectedOK: false,
		},
		{
			name:       "bool 类型",
			input:      true,
			expected:   0,
			expectedOK: false,
		},
		{
			name:       "map 类型",
			input:      map[string]any{"key": "value"},
			expected:   0,
			expectedOK: false,
		},
		{
			name:       "浮点数字符串",
			input:      "12.34",
			expected:   0,
			expectedOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 通过 ParseConfig 间接测试 parseInt
			raw := map[string]any{
				"polling": map[string]any{
					"interval_ms": tt.input,
				},
			}
			cfg, _ := config.ParseConfig(raw)

			if tt.expectedOK {
				if cfg.Polling.IntervalMs != tt.expected {
					t.Errorf("expected %d, got %d", tt.expected, cfg.Polling.IntervalMs)
				}
			} else {
				// 无效输入时应该保持默认值
				if cfg.Polling.IntervalMs != 30000 {
					t.Errorf("expected default value 30000, got %d", cfg.Polling.IntervalMs)
				}
			}
		})
	}
}

// TestParseConfigEdgeCases 测试配置解析的边界情况
func TestParseConfigEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		raw      map[string]any
		validate func(*testing.T, *config.Config)
	}{
		{
			name: "空配置",
			raw:  map[string]any{},
			validate: func(t *testing.T, cfg *config.Config) {
				if cfg == nil {
					t.Fatal("expected non-nil config")
				}
				// 应该使用默认值
				if cfg.Tracker.Kind != "mock" {
					t.Errorf("expected default tracker kind 'mock', got %s", cfg.Tracker.Kind)
				}
			},
		},
		{
			name: "GitHub tracker 配置",
			raw: map[string]any{
				"tracker": map[string]any{
					"kind":   "github",
					"api_key": "test-token",
					"repo":   "owner/repo",
				},
			},
			validate: func(t *testing.T, cfg *config.Config) {
				if cfg.Tracker.Kind != "github" {
					t.Errorf("expected 'github', got %s", cfg.Tracker.Kind)
				}
				if cfg.Tracker.Repo != "owner/repo" {
					t.Errorf("expected 'owner/repo', got %s", cfg.Tracker.Repo)
				}
			},
		},
		{
			name: "Mock tracker 配置",
			raw: map[string]any{
				"tracker": map[string]any{
					"kind": "mock",
					"mock_issues": []any{
						map[string]any{
							"id":         "1",
							"identifier": "TEST-1",
							"title":      "Test Issue",
							"state":      "Todo",
							"priority":   1,
							"labels":     []any{"bug", "feature"},
						},
					},
				},
			},
			validate: func(t *testing.T, cfg *config.Config) {
				if len(cfg.Tracker.MockIssues) != 1 {
					t.Fatalf("expected 1 mock issue, got %d", len(cfg.Tracker.MockIssues))
				}
				issue := cfg.Tracker.MockIssues[0]
				if issue.ID != "1" {
					t.Errorf("expected id '1', got %s", issue.ID)
				}
				if issue.Priority != 1 {
					t.Errorf("expected priority 1, got %d", issue.Priority)
				}
				if len(issue.Labels) != 2 {
					t.Errorf("expected 2 labels, got %d", len(issue.Labels))
				}
			},
		},
		{
			name: "Workspace 配置带环境变量",
			raw: map[string]any{
				"workspace": map[string]any{
					"root": "$WORKSPACE_ROOT",
				},
			},
			validate: func(t *testing.T, cfg *config.Config) {
				// 由于 WORKSPACE_ROOT 未设置，应该为空字符串
				if cfg.Workspace.Root != "" {
					t.Errorf("expected empty string for unset env var, got %s", cfg.Workspace.Root)
				}
			},
		},
		{
			name: "Agent 配置",
			raw: map[string]any{
				"agent": map[string]any{
					"kind":                   "claude",
					"max_concurrent_agents":  5,
					"max_turns":              30,
					"command":                "claude code",
					"max_retry_backoff_ms":   600000,
					"turn_timeout_ms":        7200000,
					"max_concurrent_agents_by_state": map[string]any{
						"Todo": 3,
						"In Progress": 2,
					},
				},
			},
			validate: func(t *testing.T, cfg *config.Config) {
				if cfg.Agent.Kind != "claude" {
					t.Errorf("expected 'claude', got %s", cfg.Agent.Kind)
				}
				if cfg.Agent.MaxConcurrentAgents != 5 {
					t.Errorf("expected 5, got %d", cfg.Agent.MaxConcurrentAgents)
				}
				if cfg.Agent.MaxTurns != 30 {
					t.Errorf("expected 30, got %d", cfg.Agent.MaxTurns)
				}
				if cfg.Agent.Command != "claude code" {
					t.Errorf("expected 'claude code', got %s", cfg.Agent.Command)
				}
				if cfg.Agent.MaxRetryBackoffMs != 600000 {
					t.Errorf("expected 600000, got %d", cfg.Agent.MaxRetryBackoffMs)
				}
				if cfg.Agent.TurnTimeoutMs != 7200000 {
					t.Errorf("expected 7200000, got %d", cfg.Agent.TurnTimeoutMs)
				}
				// 验证按状态的并发限制
				todoLimit, ok := cfg.Agent.MaxConcurrentAgentsByState["todo"]
				if !ok || todoLimit != 3 {
					t.Errorf("expected todo limit 3, got %d", todoLimit)
				}
			},
		},
		{
			name: "Claude 配置",
			raw: map[string]any{
				"claude": map[string]any{
					"command":          "claude-code",
					"skip_permissions": false,
					"extra_args":       []any{"--max-tokens", "100000"},
				},
			},
			validate: func(t *testing.T, cfg *config.Config) {
				if cfg.Claude == nil {
					t.Fatal("expected non-nil Claude config")
				}
				if cfg.Claude.Command != "claude-code" {
					t.Errorf("expected 'claude-code', got %s", cfg.Claude.Command)
				}
				if cfg.Claude.SkipPermissions != false {
					t.Errorf("expected false, got %v", cfg.Claude.SkipPermissions)
				}
				if len(cfg.Claude.ExtraArgs) != 2 {
					t.Errorf("expected 2 extra args, got %d", len(cfg.Claude.ExtraArgs))
				}
			},
		},
		{
			name: "OpenCode 配置",
			raw: map[string]any{
				"opencode": map[string]any{
					"command":    "opencode",
					"extra_args": []any{"--model", "gpt-4"},
				},
			},
			validate: func(t *testing.T, cfg *config.Config) {
				if cfg.OpenCode == nil {
					t.Fatal("expected non-nil OpenCode config")
				}
				if cfg.OpenCode.Command != "opencode" {
					t.Errorf("expected 'opencode', got %s", cfg.OpenCode.Command)
				}
				if len(cfg.OpenCode.ExtraArgs) != 2 {
					t.Errorf("expected 2 extra args, got %d", len(cfg.OpenCode.ExtraArgs))
				}
			},
		},
		{
			name: "Codex 配置",
			raw: map[string]any{
				"codex": map[string]any{
					"command":              "codex app-server --custom",
					"approval_policy":       "always",
					"thread_sandbox":        "thread",
					"turn_sandbox_policy":   "thread",
					"turn_timeout_ms":       7200000,
				},
			},
			validate: func(t *testing.T, cfg *config.Config) {
				if cfg.Codex.Command != "codex app-server --custom" {
					t.Errorf("expected 'codex app-server --custom', got %s", cfg.Codex.Command)
				}
				if cfg.Codex.ApprovalPolicy != "always" {
					t.Errorf("expected 'always', got %s", cfg.Codex.ApprovalPolicy)
				}
				if cfg.Codex.ThreadSandbox != "thread" {
					t.Errorf("expected 'thread', got %s", cfg.Codex.ThreadSandbox)
				}
				if cfg.Codex.TurnSandboxPolicy != "thread" {
					t.Errorf("expected 'thread', got %s", cfg.Codex.TurnSandboxPolicy)
				}
				if cfg.Codex.TurnTimeoutMs != 7200000 {
					t.Errorf("expected 7200000, got %d", cfg.Codex.TurnTimeoutMs)
				}
			},
		},
		{
			name: "Hooks 配置",
			raw: map[string]any{
				"hooks": map[string]any{
					"after_create":  "/path/to/after_create.sh",
					"before_run":    "/path/to/before_run.sh",
					"after_run":     "/path/to/after_run.sh",
					"before_remove": "/path/to/before_remove.sh",
					"timeout_ms":    120000,
				},
			},
			validate: func(t *testing.T, cfg *config.Config) {
				if cfg.Hooks.AfterCreate == nil || *cfg.Hooks.AfterCreate != "/path/to/after_create.sh" {
					t.Errorf("expected after_create hook, got %v", cfg.Hooks.AfterCreate)
				}
				if cfg.Hooks.BeforeRun == nil || *cfg.Hooks.BeforeRun != "/path/to/before_run.sh" {
					t.Errorf("expected before_run hook, got %v", cfg.Hooks.BeforeRun)
				}
				if cfg.Hooks.AfterRun == nil || *cfg.Hooks.AfterRun != "/path/to/after_run.sh" {
					t.Errorf("expected after_run hook, got %v", cfg.Hooks.AfterRun)
				}
				if cfg.Hooks.BeforeRemove == nil || *cfg.Hooks.BeforeRemove != "/path/to/before_remove.sh" {
					t.Errorf("expected before_remove hook, got %v", cfg.Hooks.BeforeRemove)
				}
				if cfg.Hooks.TimeoutMs != 120000 {
					t.Errorf("expected 120000, got %d", cfg.Hooks.TimeoutMs)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.ParseConfig(tt.raw)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

// TestValidateDispatchConfigGitHub 测试 GitHub tracker 的验证
func TestValidateDispatchConfigGitHub(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Config
		expected bool
	}{
		{
			name: "valid GitHub config",
			config: &config.Config{
				Tracker: config.TrackerConfig{
					Kind:   "github",
					APIKey: "test-token",
					Repo:   "owner/repo",
				},
				Codex: config.CodexConfig{
					Command: "codex app-server",
				},
				Harness: config.HarnessConfig{
					MaxIterations: 5,
				},
			},
			expected: true,
		},
		{
			name: "missing API key",
			config: &config.Config{
				Tracker: config.TrackerConfig{
					Kind:   "github",
					Repo:   "owner/repo",
				},
				Codex: config.CodexConfig{
					Command: "codex app-server",
				},
				Harness: config.HarnessConfig{
					MaxIterations: 5,
				},
			},
			expected: false,
		},
		{
			name: "missing repo",
			config: &config.Config{
				Tracker: config.TrackerConfig{
					Kind:   "github",
					APIKey: "test-token",
				},
				Codex: config.CodexConfig{
					Command: "codex app-server",
				},
				Harness: config.HarnessConfig{
					MaxIterations: 5,
				},
			},
			expected: false, // repo 是必需的
		},
		{
			name: "invalid repo format",
			config: &config.Config{
				Tracker: config.TrackerConfig{
					Kind:   "github",
					APIKey: "test-token",
					Repo:   "invalid-repo",
				},
				Codex: config.CodexConfig{
					Command: "codex app-server",
				},
				Harness: config.HarnessConfig{
					MaxIterations: 5,
				},
			},
			expected: true, // 目前没有格式验证
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validation := tt.config.ValidateDispatchConfig()
			if validation.Valid != tt.expected {
				t.Errorf("expected valid=%v, got valid=%v, errors=%v", tt.expected, validation.Valid, validation.Errors)
			}
		})
	}
}

// TestValidateDispatchConfigMock 测试 Mock tracker 的验证
func TestValidateDispatchConfigMock(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Config
		expected bool
	}{
		{
			name: "valid mock config without api_key",
			config: &config.Config{
				Tracker: config.TrackerConfig{
					Kind: "mock",
				},
				Codex: config.CodexConfig{
					Command: "codex app-server",
				},
			},
			expected: true,
		},
		{
			name: "mock config with api_key",
			config: &config.Config{
				Tracker: config.TrackerConfig{
					Kind:   "mock",
					APIKey: "test-key",
				},
				Codex: config.CodexConfig{
					Command: "codex app-server",
				},
			},
			expected: true,
		},
		{
			name: "mock config with mock_issues",
			config: &config.Config{
				Tracker: config.TrackerConfig{
					Kind: "mock",
					MockIssues: []config.MockIssueConfig{
						{ID: "1", Identifier: "TEST-1", Title: "Test", State: "Todo"},
					},
				},
				Codex: config.CodexConfig{
					Command: "codex app-server",
				},
			},
			expected: true,
		},
		{
			name: "mock config without codex command",
			config: &config.Config{
				Tracker: config.TrackerConfig{
					Kind: "mock",
				},
			},
			expected: true, // codex command 不是必需的
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validation := tt.config.ValidateDispatchConfig()
			if validation.Valid != tt.expected {
				t.Errorf("expected valid=%v, got valid=%v, errors=%v", tt.expected, validation.Valid, validation.Errors)
			}
		})
	}
}

// TestDefaultConfigFields 测试默认配置的所有字段
func TestDefaultConfigFields(t *testing.T) {
	cfg := config.DefaultConfig()

	// 验证 Tracker 配置
	if cfg.Tracker.Kind != "mock" {
		t.Errorf("expected tracker kind 'mock', got %s", cfg.Tracker.Kind)
	}
	if len(cfg.Tracker.ActiveStates) == 0 {
		t.Error("expected non-empty active states")
	}
	if len(cfg.Tracker.TerminalStates) == 0 {
		t.Error("expected non-empty terminal states")
	}

	// 验证 Polling 配置
	if cfg.Polling.IntervalMs != 30000 {
		t.Errorf("expected interval 30000, got %d", cfg.Polling.IntervalMs)
	}

	// 验证 Workspace 配置
	if cfg.Workspace.Root == "" {
		t.Error("expected non-empty workspace root")
	}

	// 验证 Hooks 配置
	if cfg.Hooks.TimeoutMs != 60000 {
		t.Errorf("expected timeout 60000, got %d", cfg.Hooks.TimeoutMs)
	}

	// 验证 Agent 配置
	if cfg.Agent.Kind != "codex" {
		t.Errorf("expected agent kind 'codex', got %s", cfg.Agent.Kind)
	}
	if cfg.Agent.MaxConcurrentAgents != 10 {
		t.Errorf("expected max concurrent agents 10, got %d", cfg.Agent.MaxConcurrentAgents)
	}
	if cfg.Agent.MaxTurns != 20 {
		t.Errorf("expected max turns 20, got %d", cfg.Agent.MaxTurns)
	}
	if cfg.Agent.MaxRetryBackoffMs != 300000 {
		t.Errorf("expected max retry backoff 300000, got %d", cfg.Agent.MaxRetryBackoffMs)
	}

	// 验证 Codex 配置
	if cfg.Codex.Command != "codex app-server" {
		t.Errorf("expected command 'codex app-server', got %s", cfg.Codex.Command)
	}
	if cfg.Codex.TurnTimeoutMs != 3600000 {
		t.Errorf("expected turn timeout 3600000, got %d", cfg.Codex.TurnTimeoutMs)
	}
	if cfg.Codex.ReadTimeoutMs != 5000 {
		t.Errorf("expected read timeout 5000, got %d", cfg.Codex.ReadTimeoutMs)
	}
	if cfg.Codex.StallTimeoutMs != 300000 {
		t.Errorf("expected stall timeout 300000, got %d", cfg.Codex.StallTimeoutMs)
	}
}
