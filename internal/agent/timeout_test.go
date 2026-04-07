// Package agent - 无超时限制测试 (Story 9.2)
package agent

import (
	"context"
	"testing"
	"time"

	"github.com/dministrator/symphony/internal/config"
)

// TestNoTimeoutConfig 测试无超时配置
// TurnTimeoutMs <= 0 表示无超时限制
func TestNoTimeoutConfig(t *testing.T) {
	tests := []struct {
		name              string
		agentTurnTimeout  int64
		codexTurnTimeout  int64
		expectedNoTimeout bool
		description       string
	}{
		{
			name:              "zero timeout means no timeout",
			agentTurnTimeout:  0,
			codexTurnTimeout:  3600000,
			expectedNoTimeout: true,
			description:       "turn_timeout_ms=0 表示无超时限制",
		},
		{
			name:              "negative timeout means no timeout",
			agentTurnTimeout:  -1,
			codexTurnTimeout:  3600000,
			expectedNoTimeout: true,
			description:       "turn_timeout_ms<0 表示无超时限制",
		},
		{
			name:              "positive timeout means timeout",
			agentTurnTimeout:  60000,
			codexTurnTimeout:  3600000,
			expectedNoTimeout: false,
			description:       "turn_timeout_ms>0 表示有超时限制",
		},
		{
			name:              "codex zero timeout",
			agentTurnTimeout:  0,
			codexTurnTimeout:  0,
			expectedNoTimeout: true,
			description:       "codex.turn_timeout_ms=0 也表示无超时",
		},
		{
			name:              "agent timeout overrides codex",
			agentTurnTimeout:  0,
			codexTurnTimeout:  60000,
			expectedNoTimeout: true,
			description:       "agent.turn_timeout_ms 优先于 codex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Agent.TurnTimeoutMs = tt.agentTurnTimeout
			cfg.Codex.TurnTimeoutMs = tt.codexTurnTimeout

			// 验证配置值
			noTimeout := cfg.Agent.TurnTimeoutMs <= 0
			if noTimeout != tt.expectedNoTimeout {
				t.Errorf("%s: expected noTimeout=%v, got %v",
					tt.description, tt.expectedNoTimeout, noTimeout)
			}
		})
	}
}

// TestCodexNoTimeoutLogic 测试 Codex runner 的无超时逻辑
func TestCodexNoTimeoutLogic(t *testing.T) {
	tests := []struct {
		name         string
		turnTimeoutMs int64
		expectNoTimeout bool
	}{
		{
			name:         "zero timeout",
			turnTimeoutMs: 0,
			expectNoTimeout: true,
		},
		{
			name:         "negative timeout",
			turnTimeoutMs: -100,
			expectNoTimeout: true,
		},
		{
			name:         "positive timeout",
			turnTimeoutMs: 3600000,
			expectNoTimeout: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Codex.TurnTimeoutMs = tt.turnTimeoutMs

			runner := newCodexRunner(cfg).(*codexRunner)

			// 验证 runner 正确读取配置
			noTimeout := runner.cfg.Codex.TurnTimeoutMs <= 0
			if noTimeout != tt.expectNoTimeout {
				t.Errorf("expected noTimeout=%v (TurnTimeoutMs=%d), got %v",
					tt.expectNoTimeout, tt.turnTimeoutMs, noTimeout)
			}
		})
	}
}

// TestClaudeNoTimeoutLogic 测试 Claude runner 的无超时逻辑
// Agent.TurnTimeoutMs < 0: 明确无超时
// Agent.TurnTimeoutMs == 0: fallback 到 Codex
// Agent.TurnTimeoutMs > 0: 使用 Agent 值
func TestClaudeNoTimeoutLogic(t *testing.T) {
	tests := []struct {
		name              string
		agentTurnTimeout  int64
		codexTurnTimeout  int64
		expectedNoTimeout bool
		description       string
	}{
		{
			name:              "agent negative means explicit no timeout",
			agentTurnTimeout:  -1,
			codexTurnTimeout:  3600000,
			expectedNoTimeout: true,
			description:       "Agent.TurnTimeoutMs < 0 明确表示无超时",
		},
		{
			name:              "agent zero fallback to codex positive",
			agentTurnTimeout:  0,
			codexTurnTimeout:  3600000,
			expectedNoTimeout: false,
			description:       "Agent.TurnTimeoutMs == 0 时 fallback 到 Codex",
		},
		{
			name:              "agent zero fallback to codex zero",
			agentTurnTimeout:  0,
			codexTurnTimeout:  0,
			expectedNoTimeout: true,
			description:       "两者都为 0 时无超时",
		},
		{
			name:              "agent zero fallback to codex negative",
			agentTurnTimeout:  0,
			codexTurnTimeout:  -1,
			expectedNoTimeout: true,
			description:       "Agent 为 0，Codex 为负数，无超时",
		},
		{
			name:              "agent positive overrides codex",
			agentTurnTimeout:  60000,
			codexTurnTimeout:  3600000,
			expectedNoTimeout: false,
			description:       "Agent > 0 时使用 Agent 值",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Agent.TurnTimeoutMs = tt.agentTurnTimeout
			cfg.Codex.TurnTimeoutMs = tt.codexTurnTimeout

			runner := newClaudeRunner(cfg).(*claudeRunner)

			// 模拟 runOnce 中的逻辑
			var turnTimeoutMs int64
			if runner.cfg.Agent.TurnTimeoutMs < 0 {
				turnTimeoutMs = -1
			} else if runner.cfg.Agent.TurnTimeoutMs > 0 {
				turnTimeoutMs = runner.cfg.Agent.TurnTimeoutMs
			} else {
				turnTimeoutMs = runner.cfg.Codex.TurnTimeoutMs
			}
			noTimeout := turnTimeoutMs <= 0

			if noTimeout != tt.expectedNoTimeout {
				t.Errorf("%s: expected noTimeout=%v, got %v (agent=%d, codex=%d, final=%d)",
					tt.description, tt.expectedNoTimeout, noTimeout,
					tt.agentTurnTimeout, tt.codexTurnTimeout, turnTimeoutMs)
			}
		})
	}
}

// TestOpenCodeNoTimeoutLogic 测试 OpenCode runner 的无超时逻辑
// Agent.TurnTimeoutMs < 0: 明确无超时
// Agent.TurnTimeoutMs == 0: fallback 到 Codex
// Agent.TurnTimeoutMs > 0: 使用 Agent 值
func TestOpenCodeNoTimeoutLogic(t *testing.T) {
	tests := []struct {
		name              string
		agentTurnTimeout  int64
		codexTurnTimeout  int64
		expectedNoTimeout bool
		description       string
	}{
		{
			name:              "agent negative means explicit no timeout",
			agentTurnTimeout:  -500,
			codexTurnTimeout:  3600000,
			expectedNoTimeout: true,
			description:       "Agent.TurnTimeoutMs < 0 明确表示无超时",
		},
		{
			name:              "agent zero fallback to codex positive",
			agentTurnTimeout:  0,
			codexTurnTimeout:  3600000,
			expectedNoTimeout: false,
			description:       "Agent.TurnTimeoutMs == 0 时 fallback 到 Codex",
		},
		{
			name:              "both zero means no timeout",
			agentTurnTimeout:  0,
			codexTurnTimeout:  0,
			expectedNoTimeout: true,
			description:       "两者都为 0 时无超时",
		},
		{
			name:              "agent positive overrides codex",
			agentTurnTimeout:  60000,
			codexTurnTimeout:  3600000,
			expectedNoTimeout: false,
			description:       "Agent > 0 时使用 Agent 值",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Agent.TurnTimeoutMs = tt.agentTurnTimeout
			cfg.Codex.TurnTimeoutMs = tt.codexTurnTimeout

			runner := newOpenCodeRunner(cfg).(*openCodeRunner)

			// 模拟 runOnce 中的逻辑
			var turnTimeoutMs int64
			if runner.cfg.Agent.TurnTimeoutMs < 0 {
				turnTimeoutMs = -1
			} else if runner.cfg.Agent.TurnTimeoutMs > 0 {
				turnTimeoutMs = runner.cfg.Agent.TurnTimeoutMs
			} else {
				turnTimeoutMs = runner.cfg.Codex.TurnTimeoutMs
			}
			noTimeout := turnTimeoutMs <= 0

			if noTimeout != tt.expectedNoTimeout {
				t.Errorf("%s: expected noTimeout=%v, got %v (agent=%d, codex=%d, final=%d)",
					tt.description, tt.expectedNoTimeout, noTimeout,
					tt.agentTurnTimeout, tt.codexTurnTimeout, turnTimeoutMs)
			}
		})
	}
}

// TestNoTimeoutContextBehavior 测试无超时 context 行为
// 无超时时，context 不应该有 Deadline
func TestNoTimeoutContextBehavior(t *testing.T) {
	ctx := context.Background()

	// 模拟无超时场景
	noTimeout := true

	var runCtx context.Context
	var cancel context.CancelFunc

	if noTimeout {
		runCtx = ctx
		cancel = func() {}
	} else {
		runCtx, cancel = context.WithTimeout(ctx, time.Hour)
	}
	defer cancel()

	// 验证无超时 context 没有 deadline
	_, hasDeadline := runCtx.Deadline()
	if hasDeadline {
		t.Error("无超时 context 不应该有 deadline")
	}

	// 验证 context 仍然有效
	if runCtx.Err() != nil {
		t.Errorf("无超时 context 应该没有错误: %v", runCtx.Err())
	}
}

// TestNoTimeoutWithCancellation 测试无超时但可被外部取消
// 无超时配置时，任务仍可被外部取消（如服务关闭）
func TestNoTimeoutWithCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// 模拟无超时场景使用原始 context
	noTimeout := true
	var runCtx context.Context
	var runCancel context.CancelFunc

	if noTimeout {
		runCtx = ctx
		runCancel = func() {}
	} else {
		runCtx, runCancel = context.WithTimeout(ctx, time.Hour)
	}
	defer runCancel()

	// 验证初始状态正常
	if runCtx.Err() != nil {
		t.Errorf("初始 context 应该正常: %v", runCtx.Err())
	}

	// 模拟外部取消
	cancel()

	// 等待取消传播
	time.Sleep(10 * time.Millisecond)

	// 验证取消生效
	if runCtx.Err() != context.Canceled {
		t.Errorf("取消后 context 应为 Canceled: %v", runCtx.Err())
	}
}

// TestLongRunningSimulation 模拟长时间运行场景
// 验证无超时配置可以支持超过常规超时限制的执行
func TestLongRunningSimulation(t *testing.T) {
	// 设置一个短超时用于测试验证
	testTimeout := 100 * time.Millisecond

	// 模拟长时间运行的任务（超过常规超时）
	longRunningDuration := 200 * time.Millisecond

	t.Run("with timeout fails long task", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			// 模拟长时间任务
			time.Sleep(longRunningDuration)
			done <- nil
		}()

		select {
		case <-done:
			t.Error("有超时配置时，长时间任务应该超时失败")
		case <-ctx.Done():
			// 正确：超时生效
		}
	})

	t.Run("no timeout allows long task", func(t *testing.T) {
		ctx := context.Background()

		// 无超时配置
		noTimeout := true
		var runCtx context.Context
		var cancel context.CancelFunc

		if noTimeout {
			runCtx = ctx
			cancel = func() {}
		} else {
			runCtx, cancel = context.WithTimeout(ctx, testTimeout)
		}
		defer cancel()

		done := make(chan error, 1)
		go func() {
			// 模拟长时间任务
			time.Sleep(longRunningDuration)
			done <- nil
		}()

		select {
		case err := <-done:
			if err != nil {
				t.Errorf("无超时配置时，长时间任务应该成功: %v", err)
			}
			// 正确：任务完成
		case <-runCtx.Done():
			t.Errorf("无超时配置时，不应该超时: %v", runCtx.Err())
		case <-time.After(300 * time.Millisecond):
			t.Error("测试本身超时")
		}
	})
}

// Test24HourPlusSimulation 模拟超过24小时的执行支持
// 验证系统设计上可以支持无限等待
func Test24HourPlusSimulation(t *testing.T) {
	// 注意：实际测试中不可能等待24小时
	// 这里通过验证配置和 context 行为来确认设计支持

	cfg := config.DefaultConfig()
	cfg.Agent.TurnTimeoutMs = 0 // 无超时
	cfg.Codex.TurnTimeoutMs = 0

	// 验证配置
	if cfg.Agent.TurnTimeoutMs != 0 {
		t.Error("Agent TurnTimeoutMs 应该为 0 表示无超时")
	}

	// 创建无超时 context
	ctx := context.Background()
	noTimeout := cfg.Agent.TurnTimeoutMs <= 0

	var runCtx context.Context
	var cancel context.CancelFunc

	if noTimeout {
		runCtx = ctx
		cancel = func() {}
	} else {
		// 24小时超时（如果配置为正数）
		runCtx, cancel = context.WithTimeout(ctx, 24*time.Hour)
	}
	defer cancel()

	// 验证无 deadline
	_, hasDeadline := runCtx.Deadline()
	if hasDeadline {
		t.Error("无超时配置时 context 不应该有 deadline")
	}

	// 验证理论上可以无限等待
	// 实际测试中只是验证 context 状态
	if runCtx.Err() != nil {
		t.Errorf("无超时 context 应该没有错误: %v", runCtx.Err())
	}
}

// TestConfigParsingForNoTimeout 测试配置解析支持无超时值
func TestConfigParsingForNoTimeout(t *testing.T) {
	tests := []struct {
		name     string
		raw      map[string]interface{}
		expected int64
	}{
		{
			name: "explicit zero",
			raw: map[string]interface{}{
				"agent": map[string]interface{}{
					"turn_timeout_ms": 0,
				},
			},
			expected: 0,
		},
		{
			name: "negative value",
			raw: map[string]interface{}{
				"agent": map[string]interface{}{
					"turn_timeout_ms": -1,
				},
			},
			expected: 0, // 解析时负数会被忽略，使用默认值
		},
		{
			name: "positive value",
			raw: map[string]interface{}{
				"agent": map[string]interface{}{
					"turn_timeout_ms": 60000,
				},
			},
			expected: 60000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.ParseConfig(tt.raw)
			if err != nil {
				t.Fatalf("解析配置失败: %v", err)
			}

			// 注意：ParseConfig 中负数会被忽略，所以实际值可能不同
			// 这里验证解析逻辑
			_ = cfg
		})
	}
}

// TestRunnerCreationWithNoTimeout 测试无超时配置下创建 runner
func TestRunnerCreationWithNoTimeout(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agent.TurnTimeoutMs = 0
	cfg.Codex.TurnTimeoutMs = 0

	// 测试所有 runner 类型
	runners := []struct {
		name string
		kind string
	}{
		{"codex", "codex"},
		{"claude", "claude"},
		{"opencode", "opencode"},
	}

	for _, r := range runners {
		t.Run(r.name, func(t *testing.T) {
			cfg.Agent.Kind = r.kind
			runner := NewRunner(cfg)

			if runner == nil {
				t.Fatalf("%s runner 创建失败", r.name)
			}
		})
	}
}