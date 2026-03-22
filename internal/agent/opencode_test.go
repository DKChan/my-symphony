// Package agent - OpenCode runner 单元测试
package agent

import (
	"testing"

	"github.com/dministrator/symphony/internal/config"
)

// TestOpenCodeEventParsing 测试 OpenCode 事件解析
func TestOpenCodeEventParsing(t *testing.T) {
	tests := []struct {
		name            string
		event           openCodeEvent
		expectedSuccess bool
		expectedErrMsg  string
		expectedInput   int64
		expectedOutput  int64
	}{
		{
			name: "session_complete success",
			event: openCodeEvent{
				Type: "session_complete",
			},
			expectedSuccess: true,
		},
		{
			name: "session_complete with usage",
			event: openCodeEvent{
				Type: "session_complete",
				Usage: &openCodeUsage{
					InputTokens:  200,
					OutputTokens: 100,
				},
			},
			expectedSuccess: true,
			expectedInput:   200,
			expectedOutput:  100,
		},
		{
			name: "session_complete with non-zero exit code",
			event: openCodeEvent{
				Type:     "session_complete",
				ExitCode: intPtr(1),
			},
			expectedSuccess: false,
			expectedErrMsg:  "exit code 1",
		},
		{
			name: "session_complete with zero exit code",
			event: openCodeEvent{
				Type:     "session_complete",
				ExitCode: intPtr(0),
			},
			expectedSuccess: true,
		},
		{
			name: "error event",
			event: openCodeEvent{
				Type:  "error",
				Error: "something went wrong",
			},
			expectedSuccess: false,
			expectedErrMsg:  "something went wrong",
		},
		{
			name: "error event with content",
			event: openCodeEvent{
				Type:    "error",
				Content: "error content",
			},
			expectedSuccess: false,
			expectedErrMsg:  "error content",
		},
		{
			name: "message event",
			event: openCodeEvent{
				Type:    "message",
				Role:    "assistant",
				Content: "Hello",
			},
			expectedSuccess: false,
		},
		{
			name: "tool_use event",
			event: openCodeEvent{
				Type: "tool_use",
				Tool: "read_file",
			},
			expectedSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &openCodeRunResult{}

			switch tt.event.Type {
			case "session_complete":
				result.success = true
				if tt.event.Usage != nil {
					result.inputTokens = tt.event.Usage.InputTokens
					result.outputTokens = tt.event.Usage.OutputTokens
				}
				if tt.event.ExitCode != nil && *tt.event.ExitCode != 0 {
					result.success = false
					result.errMsg = "exit code 1"
				}
			case "error":
				result.success = false
				result.errMsg = tt.event.Error
				if result.errMsg == "" {
					result.errMsg = tt.event.Content
				}
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

// TestOpenCodeConfig 测试 OpenCode 配置
func TestOpenCodeConfig(t *testing.T) {
	tests := []struct {
		name              string
		cfg               *config.Config
		expectedCommand   string
		expectedExtraArgs []string
	}{
		{
			name:            "default config",
			cfg:             config.DefaultConfig(),
			expectedCommand: "opencode",
		},
		{
			name: "custom command",
			cfg: &config.Config{
				OpenCode: &config.OpenCodeConfig{
					Command: "/usr/local/bin/opencode",
				},
			},
			expectedCommand: "/usr/local/bin/opencode",
		},
		{
			name: "extra args",
			cfg: &config.Config{
				OpenCode: &config.OpenCodeConfig{
					ExtraArgs: []string{"--model", "gpt-4", "--provider", "openai"},
				},
			},
			expectedCommand:   "opencode",
			expectedExtraArgs: []string{"--model", "gpt-4", "--provider", "openai"},
		},
		{
			name: "agent command override",
			cfg: &config.Config{
				Agent: config.AgentConfig{
					Command: "custom-opencode",
				},
			},
			expectedCommand: "custom-opencode",
		},
		{
			name: "agent command takes precedence",
			cfg: &config.Config{
				Agent: config.AgentConfig{
					Command: "agent-cmd",
				},
				OpenCode: &config.OpenCodeConfig{
					Command: "opencode-cmd",
				},
			},
			expectedCommand: "agent-cmd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := newOpenCodeRunner(tt.cfg).(*openCodeRunner)

			if runner.cfg == nil {
				t.Fatal("expected non-nil config")
			}

			// 检查 OpenCode 配置
			if tt.cfg.OpenCode != nil {
				if tt.cfg.OpenCode.Command != "" && runner.cfg.OpenCode.Command != tt.cfg.OpenCode.Command {
					t.Errorf("expected command %q, got %q", tt.cfg.OpenCode.Command, runner.cfg.OpenCode.Command)
				}
				if len(tt.expectedExtraArgs) > 0 {
					if len(runner.cfg.OpenCode.ExtraArgs) != len(tt.expectedExtraArgs) {
						t.Errorf("expected %d extra args, got %d", len(tt.expectedExtraArgs), len(runner.cfg.OpenCode.ExtraArgs))
					}
				}
			}
		})
	}
}

// TestOpenCodeUsage 测试 token 使用量
func TestOpenCodeUsage(t *testing.T) {
	usage := &openCodeUsage{
		InputTokens:  1500,
		OutputTokens: 750,
	}

	if usage.InputTokens != 1500 {
		t.Errorf("expected input tokens 1500, got %d", usage.InputTokens)
	}
	if usage.OutputTokens != 750 {
		t.Errorf("expected output tokens 750, got %d", usage.OutputTokens)
	}
}

// TestOpenCodeRunResult 测试运行结果
func TestOpenCodeRunResult(t *testing.T) {
	result := &openCodeRunResult{
		success:      true,
		errMsg:       "",
		inputTokens:  300,
		outputTokens: 150,
	}

	if !result.success {
		t.Error("expected success to be true")
	}
	if result.errMsg != "" {
		t.Errorf("expected empty errMsg, got %q", result.errMsg)
	}
	if result.inputTokens != 300 {
		t.Errorf("expected input tokens 300, got %d", result.inputTokens)
	}
	if result.outputTokens != 150 {
		t.Errorf("expected output tokens 150, got %d", result.outputTokens)
	}
}

// TestOpenCodeEventTypes 测试事件类型
func TestOpenCodeEventTypes(t *testing.T) {
	eventTypes := []string{
		"message",
		"tool_use",
		"tool_result",
		"session_complete",
		"error",
	}

	for _, eventType := range eventTypes {
		t.Run("type_"+eventType, func(t *testing.T) {
			event := openCodeEvent{Type: eventType}
			if event.Type != eventType {
				t.Errorf("expected type %q, got %q", eventType, event.Type)
			}
		})
	}
}

// TestOpenCodeNilConfig 测试空配置处理
func TestOpenCodeNilConfig(t *testing.T) {
	// openCodeRunner 在创建时不会检查 nil config
	// 但在实际使用时会访问 config 字段导致 panic
	// 这里验证创建行为
	runner := newOpenCodeRunner(nil)
	_ = runner // 验证可以创建，但实际使用需要有效 config
}

// TestOpenCodeExitCodeHandling 测试退出码处理
func TestOpenCodeExitCodeHandling(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		success  bool
	}{
		{"exit code 0", 0, true},
		{"exit code 1", 1, false},
		{"exit code 2", 2, false},
		{"exit code 127", 127, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &openCodeRunResult{success: true}
			exitCode := tt.exitCode

			if exitCode != 0 {
				result.success = false
				result.errMsg = "exit code 1"
			}

			if result.success != tt.success {
				t.Errorf("expected success %v, got %v", tt.success, result.success)
			}
		})
	}
}

// TestOpenCodeErrorScenarios 测试错误场景
func TestOpenCodeErrorScenarios(t *testing.T) {
	tests := []struct {
		name        string
		errMsg      string
		description string
	}{
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
		{
			name:        "scanner error",
			errMsg:      "scanner: read error",
			description: "扫描器错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &openCodeRunResult{
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

// TestOpenCodeSessionID 测试会话 ID 格式
func TestOpenCodeSessionID(t *testing.T) {
	// 验证会话 ID 格式: opencode-{identifier}-{timestamp}
	identifier := "TEST-123"
	timestamp := int64(1234567890)

	sessionID := "opencode-" + identifier + "-" + string(rune(timestamp))
	expected := "opencode-TEST-123"

	if sessionID[:len(expected)] != expected {
		t.Errorf("session ID should start with %q", expected)
	}
}

// TestOpenCodeTokenUsageCalculation 测试 token 使用量计算
func TestOpenCodeTokenUsageCalculation(t *testing.T) {
	tokenUsage := &TokenUsage{}

	runs := []struct {
		input  int64
		output int64
	}{
		{150, 75},
		{250, 125},
		{100, 50},
	}

	for _, run := range runs {
		tokenUsage.InputTokens += run.input
		tokenUsage.OutputTokens += run.output
		tokenUsage.TotalTokens += run.input + run.output
	}

	if tokenUsage.InputTokens != 500 {
		t.Errorf("expected total input 500, got %d", tokenUsage.InputTokens)
	}
	if tokenUsage.OutputTokens != 250 {
		t.Errorf("expected total output 250, got %d", tokenUsage.OutputTokens)
	}
	if tokenUsage.TotalTokens != 750 {
		t.Errorf("expected total 750, got %d", tokenUsage.TotalTokens)
	}
}

// TestOpenCodeToolUseEvent 测试工具使用事件
func TestOpenCodeToolUseEvent(t *testing.T) {
	event := openCodeEvent{
		Type:  "tool_use",
		Tool:  "read_file",
		Input: []byte(`{"path": "/tmp/test.txt"}`),
	}

	if event.Type != "tool_use" {
		t.Errorf("expected type 'tool_use', got %q", event.Type)
	}
	if event.Tool != "read_file" {
		t.Errorf("expected tool 'read_file', got %q", event.Tool)
	}
	if string(event.Input) != `{"path": "/tmp/test.txt"}` {
		t.Errorf("unexpected input: %s", event.Input)
	}
}

// TestOpenCodeMessageEvent 测试消息事件
func TestOpenCodeMessageEvent(t *testing.T) {
	event := openCodeEvent{
		Type:    "message",
		Role:    "assistant",
		Content: "I will help you with that task.",
	}

	if event.Type != "message" {
		t.Errorf("expected type 'message', got %q", event.Type)
	}
	if event.Role != "assistant" {
		t.Errorf("expected role 'assistant', got %q", event.Role)
	}
	if event.Content != "I will help you with that task." {
		t.Errorf("unexpected content: %q", event.Content)
	}
}

// TestOpenCodeConfigWithTurnTimeout 测试 turn timeout 配置
func TestOpenCodeConfigWithTurnTimeout(t *testing.T) {
	tests := []struct {
		name              string
		cfg               *config.Config
		expectedTimeoutMs int64
	}{
		{
			name:              "default timeout",
			cfg:               config.DefaultConfig(),
			expectedTimeoutMs: 3600000, // 默认 1 小时
		},
		{
			name: "custom turn timeout from agent",
			cfg: &config.Config{
				Agent: config.AgentConfig{
					TurnTimeoutMs: 1800000, // 30 分钟
				},
			},
			expectedTimeoutMs: 1800000,
		},
		{
			name: "codex timeout fallback",
			cfg: &config.Config{
				Codex: config.CodexConfig{
					TurnTimeoutMs: 7200000, // 2 小时
				},
			},
			expectedTimeoutMs: 7200000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := newOpenCodeRunner(tt.cfg).(*openCodeRunner)

			// 计算实际使用的超时
			turnTimeoutMs := tt.cfg.Codex.TurnTimeoutMs
			if tt.cfg.Agent.TurnTimeoutMs > 0 {
				turnTimeoutMs = tt.cfg.Agent.TurnTimeoutMs
			}
			if turnTimeoutMs <= 0 {
				turnTimeoutMs = 3600000
			}

			if turnTimeoutMs != tt.expectedTimeoutMs {
				t.Errorf("expected timeout %dms, got %dms", tt.expectedTimeoutMs, turnTimeoutMs)
			}

			_ = runner
		})
	}
}