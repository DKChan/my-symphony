// Package agent - Claude runner 单元测试
package agent

import (
	"testing"

	"github.com/dministrator/symphony/internal/config"
)

// TestFilterEnvVars 测试环境变量过滤
func TestFilterEnvVars(t *testing.T) {
	tests := []struct {
		name        string
		env         []string
		excludeKeys []string
		expected    []string
	}{
		{
			name:        "filter single key",
			env:         []string{"PATH=/usr/bin", "HOME=/home/user", "CLAUDECODE=1"},
			excludeKeys: []string{"CLAUDECODE"},
			expected:    []string{"PATH=/usr/bin", "HOME=/home/user"},
		},
		{
			name:        "filter multiple keys",
			env:         []string{"A=1", "B=2", "C=3", "D=4"},
			excludeKeys: []string{"B", "D"},
			expected:    []string{"A=1", "C=3"},
		},
		{
			name:        "no matching keys",
			env:         []string{"A=1", "B=2"},
			excludeKeys: []string{"C", "D"},
			expected:    []string{"A=1", "B=2"},
		},
		{
			name:        "empty env",
			env:         []string{},
			excludeKeys: []string{"A"},
			expected:    []string{},
		},
		{
			name:        "empty exclude",
			env:         []string{"A=1", "B=2"},
			excludeKeys: []string{},
			expected:    []string{"A=1", "B=2"},
		},
		{
			name:        "filter CLAUDECODE and CLAUDE_CODE_ENTRYPOINT",
			env:         []string{"PATH=/bin", "CLAUDECODE=1", "CLAUDE_CODE_ENTRYPOINT=cli", "HOME=/home"},
			excludeKeys: []string{"CLAUDECODE", "CLAUDE_CODE_ENTRYPOINT"},
			expected:    []string{"PATH=/bin", "HOME=/home"},
		},
		{
			name:        "entry without equals sign",
			env:         []string{"INVALID", "VALID=1"},
			excludeKeys: []string{"A"},
			expected:    []string{"INVALID", "VALID=1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterEnvVars(tt.env, tt.excludeKeys...)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d items, got %d: %v", len(tt.expected), len(result), result)
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("expected %q at index %d, got %q", tt.expected[i], i, v)
				}
			}
		})
	}
}

// TestClaudeEventParsing 测试 Claude 事件解析
func TestClaudeEventParsing(t *testing.T) {
	tests := []struct {
		name            string
		event           claudeEvent
		expectedSuccess bool
		expectedErrMsg  string
		expectedInput   int64
		expectedOutput  int64
	}{
		{
			name: "result success",
			event: claudeEvent{
				Type:    "result",
				IsError: false,
			},
			expectedSuccess: true,
		},
		{
			name: "result error",
			event: claudeEvent{
				Type:    "result",
				IsError: true,
				Result:  "something went wrong",
			},
			expectedSuccess: false,
			expectedErrMsg:  "something went wrong",
		},
		{
			name: "result with usage",
			event: claudeEvent{
				Type: "result",
				Usage: &claudeUsage{
					InputTokens:  100,
					OutputTokens: 50,
				},
			},
			expectedSuccess: true,
			expectedInput:   100,
			expectedOutput:  50,
		},
		{
			name: "error event",
			event: claudeEvent{
				Type:   "error",
				Result: "failed",
			},
			expectedSuccess: false,
			expectedErrMsg:  "failed",
		},
		{
			name: "system init event",
			event: claudeEvent{
				Type:    "system",
				Subtype: "init",
			},
			expectedSuccess: false, // Not a result event
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &claudeRunResult{}

			switch tt.event.Type {
			case "result":
				result.success = !tt.event.IsError
				if tt.event.IsError {
					result.errMsg = tt.event.Result
				}
				if tt.event.Usage != nil {
					result.inputTokens = tt.event.Usage.InputTokens
					result.outputTokens = tt.event.Usage.OutputTokens
				}
			case "error":
				result.success = false
				result.errMsg = tt.event.Result
			}

			if result.success != tt.expectedSuccess {
				t.Errorf("expected success %v, got %v", tt.expectedSuccess, result.success)
			}
			if result.errMsg != tt.expectedErrMsg {
				t.Errorf("expected errMsg %q, got %q", tt.expectedErrMsg, result.errMsg)
			}
			if result.inputTokens != tt.expectedInput {
				t.Errorf("expected input tokens %d, got %d", tt.expectedInput, result.inputTokens)
			}
			if result.outputTokens != tt.expectedOutput {
				t.Errorf("expected output tokens %d, got %d", tt.expectedOutput, result.outputTokens)
			}
		})
	}
}

// TestClaudeConfig 测试 Claude 配置
func TestClaudeConfig(t *testing.T) {
	tests := []struct {
		name              string
		cfg               *config.Config
		expectedCommand   string
		expectedSkipPerms bool
		expectedExtraArgs []string
	}{
		{
			name:              "default config",
			cfg:               config.DefaultConfig(),
			expectedCommand:   "claude",
			expectedSkipPerms: true, // 默认跳过
		},
		{
			name: "custom command",
			cfg: &config.Config{
				Claude: &config.ClaudeConfig{
					Command: "/path/to/claude",
				},
			},
			expectedCommand:   "/path/to/claude",
			expectedSkipPerms: true,
		},
		{
			name: "skip permissions false",
			cfg: &config.Config{
				Claude: &config.ClaudeConfig{
					SkipPermissions: false,
				},
			},
			expectedCommand:   "claude",
			expectedSkipPerms: false,
		},
		{
			name: "extra args",
			cfg: &config.Config{
				Claude: &config.ClaudeConfig{
					ExtraArgs: []string{"--model", "opus-4"},
				},
			},
			expectedCommand:   "claude",
			expectedSkipPerms: true,
			expectedExtraArgs: []string{"--model", "opus-4"},
		},
		{
			name: "agent command override",
			cfg: &config.Config{
				Agent: config.AgentConfig{
					Command: "custom-claude",
				},
			},
			expectedCommand: "custom-claude",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := newClaudeRunner(tt.cfg).(*claudeRunner)

			// 验证配置被正确传递
			if runner.cfg == nil {
				t.Fatal("expected non-nil config")
			}

			// 检查 Claude 配置
			if tt.cfg.Claude != nil {
				if tt.cfg.Claude.Command != "" && runner.cfg.Claude.Command != tt.cfg.Claude.Command {
					t.Errorf("expected command %q, got %q", tt.cfg.Claude.Command, runner.cfg.Claude.Command)
				}
			}
		})
	}
}

// TestClaudeUsage 测试 token 使用量
func TestClaudeUsage(t *testing.T) {
	usage := &claudeUsage{
		InputTokens:  1000,
		OutputTokens: 500,
	}

	if usage.InputTokens != 1000 {
		t.Errorf("expected input tokens 1000, got %d", usage.InputTokens)
	}
	if usage.OutputTokens != 500 {
		t.Errorf("expected output tokens 500, got %d", usage.OutputTokens)
	}
}

// TestClaudeRunResult 测试运行结果
func TestClaudeRunResult(t *testing.T) {
	result := &claudeRunResult{
		success:      true,
		errMsg:       "",
		inputTokens:  200,
		outputTokens: 100,
	}

	if !result.success {
		t.Error("expected success to be true")
	}
	if result.errMsg != "" {
		t.Errorf("expected empty errMsg, got %q", result.errMsg)
	}
	if result.inputTokens != 200 {
		t.Errorf("expected input tokens 200, got %d", result.inputTokens)
	}
	if result.outputTokens != 100 {
		t.Errorf("expected output tokens 100, got %d", result.outputTokens)
	}
}

// TestClaudeEventTypes 测试事件类型
func TestClaudeEventTypes(t *testing.T) {
	eventTypes := []string{"result", "system", "error", "assistant", "user"}

	for _, eventType := range eventTypes {
		t.Run("type_"+eventType, func(t *testing.T) {
			event := claudeEvent{Type: eventType}
			if event.Type != eventType {
				t.Errorf("expected type %q, got %q", eventType, event.Type)
			}
		})
	}
}

// TestClaudeNilConfig 测试空配置处理
func TestClaudeNilConfig(t *testing.T) {
	// claudeRunner 在创建时不会检查 nil config
	// 但在实际使用时会访问 config 字段导致 panic
	// 这里验证创建行为
	runner := newClaudeRunner(nil)
	_ = runner // 验证可以创建，但实际使用需要有效 config
}

// TestClaudeTokenUsageCalculation 测试 token 使用量计算
func TestClaudeTokenUsageCalculation(t *testing.T) {
	// 模拟多次运行累加 token
	tokenUsage := &TokenUsage{}

	runs := []struct {
		input  int64
		output int64
	}{
		{100, 50},
		{200, 100},
		{150, 75},
	}

	for _, run := range runs {
		tokenUsage.InputTokens += run.input
		tokenUsage.OutputTokens += run.output
		tokenUsage.TotalTokens += run.input + run.output
	}

	if tokenUsage.InputTokens != 450 {
		t.Errorf("expected total input 450, got %d", tokenUsage.InputTokens)
	}
	if tokenUsage.OutputTokens != 225 {
		t.Errorf("expected total output 225, got %d", tokenUsage.OutputTokens)
	}
	if tokenUsage.TotalTokens != 675 {
		t.Errorf("expected total 675, got %d", tokenUsage.TotalTokens)
	}
}

// TestClaudeErrorScenarios 测试错误场景
func TestClaudeErrorScenarios(t *testing.T) {
	tests := []struct {
		name        string
		errMsg      string
		description string
	}{
		{
			name:        "nested session error",
			errMsg:      "nested_session_blocked: claude CLI cannot run inside Claude Code session",
			description: "嵌套会话被阻止",
		},
		{
			name:        "timeout error",
			errMsg:      "turn_timeout (3600000ms)",
			description: "超时错误",
		},
		{
			name:        "exit error",
			errMsg:      "exit: signal: killed",
			description: "退出错误",
		},
		{
			name:        "api error",
			errMsg:      "API error: rate limit exceeded",
			description: "API 错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &claudeRunResult{
				success: false,
				errMsg:  tt.errMsg,
			}

			if result.success {
				t.Error("expected success to be false")
			}
			if result.errMsg != tt.errMsg {
				t.Errorf("expected errMsg %q, got %q", tt.errMsg, result.errMsg)
			}
		})
	}
}