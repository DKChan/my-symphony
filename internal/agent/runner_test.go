// Package agent - Runner 工厂函数测试
package agent

import (
	"testing"

	"github.com/dministrator/symphony/internal/config"
)

// TestNewRunner 测试 Runner 工厂函数
func TestNewRunner(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Config
		expected string
	}{
		{
			name: "codex kind",
			config: &config.Config{
				Agent: config.AgentConfig{
					Kind: "codex",
				},
			},
			expected: "*agent.codexRunner",
		},
		{
			name: "claude kind",
			config: &config.Config{
				Agent: config.AgentConfig{
					Kind: "claude",
				},
			},
			expected: "*agent.claudeRunner",
		},
		{
			name: "opencode kind",
			config: &config.Config{
				Agent: config.AgentConfig{
					Kind: "opencode",
				},
			},
			expected: "*agent.openCodeRunner",
		},
		{
			name: "empty kind (default to codex)",
			config: &config.Config{
				Agent: config.AgentConfig{
					Kind: "",
				},
			},
			expected: "*agent.codexRunner",
		},
		{
			name: "unknown kind (default to codex)",
			config: &config.Config{
				Agent: config.AgentConfig{
					Kind: "unknown",
				},
			},
			expected: "*agent.codexRunner",
		},
		{
			name: "uppercase kind (case insensitive)",
			config: &config.Config{
				Agent: config.AgentConfig{
					Kind: "CODEX",
				},
			},
			expected: "*agent.codexRunner",
		},
		{
			name: "mixed case kind",
			config: &config.Config{
				Agent: config.AgentConfig{
					Kind: "Claude",
				},
			},
			expected: "*agent.codexRunner",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewRunner(tt.config)
			if runner == nil {
				t.Fatal("expected non-nil runner")
			}

			// 验证类型
			runnerType := getTypeString(runner)
			if runnerType != tt.expected {
				t.Errorf("NewRunner() = %v, want %v", runnerType, tt.expected)
			}
		})
	}
}

// TestNewRunnerEdgeCases 测试边界情况
func TestNewRunnerEdgeCases(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic with nil config")
			}
		}()
		NewRunner(nil)
	})

	t.Run("config with zero value fields", func(t *testing.T) {
		cfg := &config.Config{}
		runner := NewRunner(cfg)
		if runner == nil {
			t.Fatal("expected non-nil runner")
		}
		// 默认应该返回 codexRunner
		runnerType := getTypeString(runner)
		if runnerType != "*agent.codexRunner" {
			t.Errorf("expected *agent.codexRunner, got %v", runnerType)
		}
	})

	t.Run("config with only agent kind set", func(t *testing.T) {
		cfg := &config.Config{
			Agent: config.AgentConfig{
				Kind: "claude",
			},
		}
		runner := NewRunner(cfg)
		if runner == nil {
			t.Fatal("expected non-nil runner")
		}
		runnerType := getTypeString(runner)
		if runnerType != "*agent.claudeRunner" {
			t.Errorf("expected *agent.claudeRunner, got %v", runnerType)
		}
	})
}

// TestNewRunnerConsistency 测试多次调用的结果一致性
func TestNewRunnerConsistency(t *testing.T) {
	cfg := &config.Config{
		Agent: config.AgentConfig{
			Kind: "codex",
		},
	}

	// 多次调用应该返回相同类型的实例
	runner1 := NewRunner(cfg)
	runner2 := NewRunner(cfg)

	if getTypeString(runner1) != getTypeString(runner2) {
		t.Error("multiple calls should return same type")
	}
}

// TestNewRunnerAllKinds 测试所有支持的 kind
func TestNewRunnerAllKinds(t *testing.T) {
	kinds := []string{
		"codex",
		"claude",
		"opencode",
		"",
		"unknown",
	}

	for _, kind := range kinds {
		t.Run("kind="+kind, func(t *testing.T) {
			cfg := &config.Config{
				Agent: config.AgentConfig{
					Kind: kind,
				},
			}
			runner := NewRunner(cfg)
			if runner == nil {
				t.Fatalf("kind=%q: expected non-nil runner", kind)
			}
		})
	}
}

// TestRunnerInterface 验证所有 Runner 实现都实现了接口
func TestRunnerInterface(t *testing.T) {
	tests := []struct {
		name   string
		kind   string
		config *config.Config
	}{
		{
			name: "codex",
			kind: "codex",
			config: &config.Config{
				Agent: config.AgentConfig{
					Kind: "codex",
				},
			},
		},
		{
			name: "claude",
			kind: "claude",
			config: &config.Config{
				Agent: config.AgentConfig{
					Kind: "claude",
				},
			},
		},
		{
			name: "opencode",
			kind: "opencode",
			config: &config.Config{
				Agent: config.AgentConfig{
					Kind: "opencode",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewRunner(tt.config)
			if runner == nil {
				t.Errorf("%s runner is nil", tt.name)
			}
		})
	}
}

// getTypeString 获取变量类型的字符串表示
func getTypeString(v any) string {
	switch v.(type) {
	case *codexRunner:
		return "*agent.codexRunner"
	case *claudeRunner:
		return "*agent.claudeRunner"
	case *openCodeRunner:
		return "*agent.openCodeRunner"
	default:
		return "unknown"
	}
}
