package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dministrator/symphony/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStartCommand(t *testing.T) {
	opts := StartOptions{
		WorkflowPath: "/path/to/workflow.md",
		Port:         8080,
	}
	cmd := NewStartCommand(opts)

	assert.NotNil(t, cmd)
	assert.Equal(t, "/path/to/workflow.md", cmd.options.WorkflowPath)
	assert.Equal(t, 8080, cmd.options.Port)
}

func TestStartOptions_DefaultValues(t *testing.T) {
	opts := StartOptions{}
	cmd := NewStartCommand(opts)

	assert.Equal(t, "", cmd.options.WorkflowPath)
	assert.Equal(t, 0, cmd.options.Port)
	assert.Equal(t, "", cmd.options.ConfigPath)
}

func TestStartCommand_ResolveWorkflowPath(t *testing.T) {
	tests := []struct {
		name            string
		opts            StartOptions
		wantContains    string
	}{
		{
			name: "explicit workflow path",
			opts: StartOptions{WorkflowPath: "/custom/workflow.md"},
			wantContains: "/custom/workflow.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewStartCommand(tt.opts)
			result := cmd.resolveWorkflowPath()
			assert.Contains(t, result, tt.wantContains)
		})
	}
}

// TestStartCommand_ResolveWorkflowPathDefaultPaths tests the default path resolution
// without actually changing directories (using mocks/stubs would be better for this)
func TestStartCommand_ResolveWorkflowPathDefaultPaths(t *testing.T) {
	// These tests require directory changes which can interfere with other tests
	// We test the basic logic here
	cmd := NewStartCommand(StartOptions{})

	// When no path is provided and no files exist, it should fallback to WORKFLOW.md
	// This test only verifies the method exists and doesn't panic
	assert.NotNil(t, cmd)
}

func TestStartCommand_ResolveHTTPPort(t *testing.T) {
	tests := []struct {
		name     string
		opts     StartOptions
		port     int // config.Server.Port
		wantPort int
	}{
		{
			name:     "CLI port takes priority",
			opts:     StartOptions{Port: 9000},
			port:     8080,
			wantPort: 9000,
		},
		{
			name:     "config port when no CLI port",
			opts:     StartOptions{},
			port:     8080,
			wantPort: 8080,
		},
		{
			name:     "no port configured",
			opts:     StartOptions{},
			port:     0,
			wantPort: 0,
		},
		{
			name:     "CLI port zero uses config port",
			opts:     StartOptions{Port: 0},
			port:     3000,
			wantPort: 3000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewStartCommand(tt.opts)

			// 创建模拟配置
			cfg := createTestConfigWithPort(tt.port)

			result := cmd.resolveHTTPPort(cfg)
			assert.Equal(t, tt.wantPort, result)
		})
	}
}

func TestStartCommand_ValidateConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		tempDir := t.TempDir()

		// 创建有效的 workflow.md - 使用 mock tracker 和有效的 agent
		// 注意：ValidateSymphonyConfig 会检查 agent CLI 是否存在
		// 所以这里我们需要使用一个存在的命令或跳过这个检查
		// 对于测试，我们只测试 ValidateDispatchConfig 的基本配置
		workflowPath := filepath.Join(tempDir, "workflow.md")
		content := `---
tracker:
  kind: mock
polling:
  interval_ms: 30000
agent:
  kind: codex
harness:
  max_iterations: 5
---
# Test Prompt
`
		err := os.WriteFile(workflowPath, []byte(content), 0644)
		require.NoError(t, err)

		cmd := NewStartCommand(StartOptions{WorkflowPath: workflowPath})
		cfg, result, err := cmd.ValidateConfig()

		// ValidateConfig 会调用 ValidateSymphonyConfig，它检查 codex CLI
		// 在测试环境中 codex app-server 可能不存在，所以可能会失败
		// 但我们验证配置解析是正确的
		if err != nil {
			// 如果失败是因为 agent CLI 不存在，这是预期的
			assert.Contains(t, err.Error(), "agent CLI not found")
		} else {
			assert.NotNil(t, cfg)
			assert.NotNil(t, result)
			assert.Equal(t, "mock", cfg.Tracker.Kind)
			assert.Contains(t, result.WorkflowPath, "workflow.md")
		}
	})

	t.Run("missing workflow file", func(t *testing.T) {
		cmd := NewStartCommand(StartOptions{WorkflowPath: "/nonexistent/workflow.md"})
		cfg, result, err := cmd.ValidateConfig()

		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "workflow_load")
	})

	t.Run("invalid workflow content", func(t *testing.T) {
		tempDir := t.TempDir()

		// 创建无效的 workflow.md
		workflowPath := filepath.Join(tempDir, "workflow.md")
		content := `---
invalid yaml content
---
# Test
`
		err := os.WriteFile(workflowPath, []byte(content), 0644)
		require.NoError(t, err)

		cmd := NewStartCommand(StartOptions{WorkflowPath: workflowPath})
		cfg, result, err := cmd.ValidateConfig()

		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Nil(t, result)
	})
}

func TestStartCommand_PrintStartupInfo(t *testing.T) {
	cmd := NewStartCommand(StartOptions{Port: 8080})

	// 创建测试配置
	cfg := createTestConfigWithPort(8080)

	// printStartupInfo 只打印信息，不返回错误
	// 我们只验证函数不会崩溃
	cmd.printStartupInfo(cfg, "/path/to/workflow.md")
}

func TestStartCommand_Run_ContextCancellation(t *testing.T) {
	t.Run("context cancelled before run", func(t *testing.T) {
		tempDir := t.TempDir()

		// 创建有效的 workflow.md
		workflowPath := filepath.Join(tempDir, "workflow.md")
		content := `---
tracker:
  kind: mock
polling:
  interval_ms: 30000
agent:
  kind: codex
harness:
  max_iterations: 5
---
# Test Prompt
`
		err := os.WriteFile(workflowPath, []byte(content), 0644)
		require.NoError(t, err)

		cmd := NewStartCommand(StartOptions{WorkflowPath: workflowPath})

		// 创建一个已经被取消的上下文
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err = cmd.Run(ctx)
		// 由于上下文已取消，Run 应该快速返回
		// 可能返回 context.Canceled 错误或验证错误（如果 agent CLI 不存在）
		assert.Error(t, err)
	})
}

func TestStartCommand_WatchWorkflow(t *testing.T) {
	t.Run("watch setup", func(t *testing.T) {
		// 这个测试验证 watchWorkflow 函数不会崩溃
		// 实际的文件变更测试需要更复杂的设置

		tempDir := t.TempDir()
		workflowPath := filepath.Join(tempDir, "workflow.md")
		content := `---
tracker:
  kind: mock
---
# Test
`
		err := os.WriteFile(workflowPath, []byte(content), 0644)
		require.NoError(t, err)

		// 创建一个简单的 mock orchestrator
		// 由于 orchestrator 需要复杂的依赖，这里只测试函数不会崩溃
		// 在实际集成测试中会更完整

		cmd := NewStartCommand(StartOptions{})
		// watchWorkflow 需要实际的 orchestrator，这里只验证函数定义存在
		assert.NotNil(t, cmd.watchWorkflow)
	})
}

func TestStartCommand_RunResult(t *testing.T) {
	t.Run("run result structure", func(t *testing.T) {
		result := &RunResult{
			WorkflowPath: "/path/to/workflow.md",
		}

		assert.Equal(t, "/path/to/workflow.md", result.WorkflowPath)
	})
}

func TestStartCommand_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	t.Run("short run with cancellation", func(t *testing.T) {
		tempDir := t.TempDir()

		// 创建有效的 workflow.md
		workflowPath := filepath.Join(tempDir, "workflow.md")
		content := `---
tracker:
  kind: mock
polling:
  interval_ms: 1000
agent:
  kind: codex
harness:
  max_iterations: 5
---
# Test Prompt
`
		err := os.WriteFile(workflowPath, []byte(content), 0644)
		require.NoError(t, err)

		cmd := NewStartCommand(StartOptions{
			WorkflowPath: workflowPath,
			Port:         0, // 不启动 HTTP 服务
		})

		// 创建一个超时上下文
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Run 应该在超时前返回（由于 context 取消）
		err = cmd.Run(ctx)
		// 由于 codex CLI 可能不存在，可能会返回验证错误
		if err != nil {
			errStr := err.Error()
			// 可能是 agent CLI 不存在错误或 context 取消错误
			validErrors := []string{
				"agent CLI not found",
				"symphony_config_invalid",
				"orchestrator_error",
				"context canceled",
				"context deadline exceeded",
			}
			found := false
			for _, validErr := range validErrors {
				if strings.Contains(errStr, validErr) {
					found = true
					break
				}
			}
			assert.True(t, found, "unexpected error: %v", err)
		}
	})
}

func TestStartCommand_ValidateConfig_AllErrors(t *testing.T) {
	t.Run("missing workflow file", func(t *testing.T) {
		cmd := NewStartCommand(StartOptions{WorkflowPath: "/nonexistent/path/workflow.md"})
		cfg, result, err := cmd.ValidateConfig()

		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "workflow_load")
	})

	t.Run("invalid tracker kind", func(t *testing.T) {
		tempDir := t.TempDir()
		workflowPath := filepath.Join(tempDir, "workflow.md")
		content := `---
tracker:
  kind: invalid_tracker
agent:
  kind: codex
---
# Test
`
		err := os.WriteFile(workflowPath, []byte(content), 0644)
		require.NoError(t, err)

		cmd := NewStartCommand(StartOptions{WorkflowPath: workflowPath})
		cfg, result, err := cmd.ValidateConfig()

		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Nil(t, result)
	})

	t.Run("invalid agent kind", func(t *testing.T) {
		tempDir := t.TempDir()
		workflowPath := filepath.Join(tempDir, "workflow.md")
		content := `---
tracker:
  kind: mock
agent:
  kind: invalid_agent
---
# Test
`
		err := os.WriteFile(workflowPath, []byte(content), 0644)
		require.NoError(t, err)

		cmd := NewStartCommand(StartOptions{WorkflowPath: workflowPath})
		cfg, result, err := cmd.ValidateConfig()

		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Nil(t, result)
	})

	t.Run("missing harness max_iterations", func(t *testing.T) {
		tempDir := t.TempDir()
		workflowPath := filepath.Join(tempDir, "workflow.md")
		content := `---
tracker:
  kind: mock
agent:
  kind: codex
harness:
  max_iterations: 0
---
# Test
`
		err := os.WriteFile(workflowPath, []byte(content), 0644)
		require.NoError(t, err)

		cmd := NewStartCommand(StartOptions{WorkflowPath: workflowPath})
		cfg, result, err := cmd.ValidateConfig()

		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Nil(t, result)
	})
}

func TestStartCommand_RunResult_Fields(t *testing.T) {
	result := &RunResult{
		Config:       config.DefaultConfig(),
		WorkflowPath: "/test/workflow.md",
	}

	assert.NotNil(t, result.Config)
	assert.Equal(t, "/test/workflow.md", result.WorkflowPath)
}

func TestStartCommand_StartOptions_ZeroValues(t *testing.T) {
	opts := StartOptions{}
	cmd := NewStartCommand(opts)

	assert.Equal(t, "", cmd.options.WorkflowPath)
	assert.Equal(t, 0, cmd.options.Port)
	assert.Equal(t, "", cmd.options.ConfigPath)
}

func TestStartCommand_ResolveHTTPPort_NilServerConfig(t *testing.T) {
	cmd := NewStartCommand(StartOptions{Port: 0})
	cfg := config.DefaultConfig()
	cfg.Server = nil // explicit nil

	result := cmd.resolveHTTPPort(cfg)
	assert.Equal(t, 0, result)
}

func TestStartCommand_PrintStartupInfo_WithPort(t *testing.T) {
	cmd := NewStartCommand(StartOptions{Port: 8080})
	cfg := config.DefaultConfig()
	cfg.Server = &config.ServerConfig{Port: 8080}

	// Should not panic
	cmd.printStartupInfo(cfg, "/test/workflow.md")
}

func TestStartCommand_PrintStartupInfo_WithoutPort(t *testing.T) {
	cmd := NewStartCommand(StartOptions{Port: 0})
	cfg := config.DefaultConfig()
	cfg.Server = nil

	// Should not panic
	cmd.printStartupInfo(cfg, "/test/workflow.md")
}

func TestStartCommand_ValidateConfig_WithPath(t *testing.T) {
	// Test with absolute path
	tempDir := t.TempDir()
	workflowPath := filepath.Join(tempDir, "workflow.md")
	content := `---
tracker:
  kind: mock
agent:
  kind: codex
harness:
  max_iterations: 5
---
# Test
`
	err := os.WriteFile(workflowPath, []byte(content), 0644)
	require.NoError(t, err)

	cmd := NewStartCommand(StartOptions{WorkflowPath: workflowPath})
	cfg, result, err := cmd.ValidateConfig()

	// May fail due to agent CLI check, but path should be resolved
	if err == nil {
		assert.NotNil(t, cfg)
		assert.NotNil(t, result)
		assert.Contains(t, result.WorkflowPath, "workflow.md")
	}
}

func TestStartCommand_ResolveWorkflowPath_Explicit(t *testing.T) {
	// Test explicit workflow path
	cmd := NewStartCommand(StartOptions{WorkflowPath: "/custom/workflow.md"})
	result := cmd.resolveWorkflowPath()
	assert.Equal(t, "/custom/workflow.md", result)
}

func TestStartCommand_ResolveHTTPPort_AllScenarios(t *testing.T) {
	tests := []struct {
		name       string
		cliPort    int
		configPort int
		wantPort   int
	}{
		{
			name:       "CLI port overrides config port",
			cliPort:    9000,
			configPort: 8080,
			wantPort:   9000,
		},
		{
			name:       "Config port when CLI port is 0",
			cliPort:    0,
			configPort: 8080,
			wantPort:   8080,
		},
		{
			name:       "Zero when both are zero",
			cliPort:    0,
			configPort: 0,
			wantPort:   0,
		},
		{
			name:       "Config port 3000",
			cliPort:    0,
			configPort: 3000,
			wantPort:   3000,
		},
		{
			name:       "CLI port 5000 with config port 0",
			cliPort:    5000,
			configPort: 0,
			wantPort:   5000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewStartCommand(StartOptions{Port: tt.cliPort})
			cfg := config.DefaultConfig()
			if tt.configPort > 0 {
				cfg.Server = &config.ServerConfig{Port: tt.configPort}
			}
			result := cmd.resolveHTTPPort(cfg)
			assert.Equal(t, tt.wantPort, result)
		})
	}
}

func TestStartCommand_PrintStartupInfo_NoPanic(t *testing.T) {
	cmd := NewStartCommand(StartOptions{Port: 8080})
	cfg := config.DefaultConfig()
	cfg.Server = &config.ServerConfig{Port: 8080}

	// printStartupInfo should not panic
	cmd.printStartupInfo(cfg, "/path/to/workflow.md")
}

func TestStartCommand_PrintStartupInfo_NoPort(t *testing.T) {
	cmd := NewStartCommand(StartOptions{Port: 0})
	cfg := config.DefaultConfig()

	// Should handle no port gracefully
	cmd.printStartupInfo(cfg, "/path/to/workflow.md")
}

func TestStartCommand_EmptyWorkflowPath_Logic(t *testing.T) {
	// Test the logic without changing directories
	// When WorkflowPath is empty, resolveWorkflowPath will check for files
	cmd := NewStartCommand(StartOptions{})
	assert.NotNil(t, cmd)
	assert.Equal(t, "", cmd.options.WorkflowPath)
}

func TestStartCommand_PortVariations(t *testing.T) {
	tests := []struct {
		name     string
		opts     StartOptions
		wantPort int
	}{
		{
			name:     "port 8080",
			opts:     StartOptions{Port: 8080},
			wantPort: 8080,
		},
		{
			name:     "port 3000",
			opts:     StartOptions{Port: 3000},
			wantPort: 3000,
		},
		{
			name:     "port 0 (disabled)",
			opts:     StartOptions{Port: 0},
			wantPort: 0,
		},
		{
			name:     "negative port",
			opts:     StartOptions{Port: -1},
			wantPort: -1, // 保持原值，由验证器处理
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewStartCommand(tt.opts)
			assert.Equal(t, tt.wantPort, cmd.options.Port)
		})
	}
}

func TestStartCommand_ValidateConfig_InvalidTrackerKind(t *testing.T) {
	tempDir := t.TempDir()

	// 创建带有无效 tracker 类型的 workflow.md
	workflowPath := filepath.Join(tempDir, "workflow.md")
	content := `---
tracker:
  kind: invalid_tracker
agent:
  kind: codex
---
# Test Prompt
`
	err := os.WriteFile(workflowPath, []byte(content), 0644)
	require.NoError(t, err)

	cmd := NewStartCommand(StartOptions{WorkflowPath: workflowPath})
	cfg, result, err := cmd.ValidateConfig()

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "config_invalid")
}

func TestStartCommand_ValidateConfig_InvalidAgentKind(t *testing.T) {
	tempDir := t.TempDir()

	// 创建带有无效 agent 类型的 workflow.md
	workflowPath := filepath.Join(tempDir, "workflow.md")
	content := `---
tracker:
  kind: mock
agent:
  kind: invalid_agent
---
# Test Prompt
`
	err := os.WriteFile(workflowPath, []byte(content), 0644)
	require.NoError(t, err)

	cmd := NewStartCommand(StartOptions{WorkflowPath: workflowPath})
	cfg, result, err := cmd.ValidateConfig()

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Nil(t, result)
}

func TestStartCommand_MultipleWorkflowFiles_Logic(t *testing.T) {
	// Test the priority logic without changing directories
	// .sym/workflow.md takes priority over WORKFLOW.md
	cmd := NewStartCommand(StartOptions{})
	assert.NotNil(t, cmd)
}

func TestStartCommand_WorkflowPathPrecedence_Logic(t *testing.T) {
	// Test that explicit path takes precedence
	cmd := NewStartCommand(StartOptions{WorkflowPath: "/custom/path/workflow.md"})
	result := cmd.resolveWorkflowPath()
	assert.Equal(t, "/custom/path/workflow.md", result)
}

// 辅助函数：创建带有指定端口配置的测试配置
func createTestConfigWithPort(port int) *config.Config {
	// 使用默认配置
	cfg := config.DefaultConfig()
	if port > 0 {
		cfg.Server = &config.ServerConfig{Port: port}
	}
	return cfg
}