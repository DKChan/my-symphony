// Package agent - OpenCode CLI 适配器
// 使用 `opencode run "prompt" --output-format json` 非交互模式
package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/domain"
)

// openCodeRunner 使用 OpenCode CLI 的运行器
type openCodeRunner struct {
	cfg *config.Config
}

func newOpenCodeRunner(cfg *config.Config) Runner {
	return &openCodeRunner{cfg: cfg}
}

// openCodeEvent OpenCode JSON 输出事件结构
// opencode run --output-format json 输出 NDJSON，每行一个事件
type openCodeEvent struct {
	// 事件类型：message | tool_use | tool_result | session_complete | error
	Type string `json:"type"`

	// message 事件字段
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`

	// tool_use 事件字段
	Tool  string          `json:"tool,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`

	// session_complete / error 事件字段
	SessionID string         `json:"session_id,omitempty"`
	Error     string         `json:"error,omitempty"`
	Usage     *openCodeUsage `json:"usage,omitempty"`
	ExitCode  *int           `json:"exit_code,omitempty"`
}

// openCodeUsage token 使用量
type openCodeUsage struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
}

// RunAttempt 实现 Runner 接口
func (r *openCodeRunner) RunAttempt(
	ctx context.Context,
	issue *domain.Issue,
	workspacePath string,
	attempt *int,
	promptTemplate string,
	callback EventCallback,
) (*RunAttemptResult, error) {
	prompt := buildPrompt(issue, attempt, promptTemplate)

	sessionID := fmt.Sprintf("opencode-%s-%d", issue.Identifier, time.Now().UnixNano())
	if callback != nil {
		callback("session_started", map[string]any{
			"session_id": sessionID,
			"agent":      "opencode",
			"issue_id":   issue.ID,
		})
	}

	turnCount := 0
	tokenUsage := &TokenUsage{}

	// OpenCode 每次调用独立执行完整任务，单次即可
	result, err := r.runOnce(ctx, workspacePath, prompt, sessionID, callback)
	if err != nil {
		return &RunAttemptResult{
			Success:    false,
			Error:      err,
			TurnCount:  turnCount,
			TokenUsage: tokenUsage,
		}, nil
	}

	turnCount++
	tokenUsage.InputTokens += result.inputTokens
	tokenUsage.OutputTokens += result.outputTokens
	tokenUsage.TotalTokens += result.inputTokens + result.outputTokens

	if !result.success {
		return &RunAttemptResult{
			Success:    false,
			Error:      fmt.Errorf("opencode run failed: %s", result.errMsg),
			TurnCount:  turnCount,
			TokenUsage: tokenUsage,
		}, nil
	}

	return &RunAttemptResult{
		Success:    true,
		TurnCount:  turnCount,
		TokenUsage: tokenUsage,
	}, nil
}

// openCodeRunResult 单次运行结果
type openCodeRunResult struct {
	success      bool
	errMsg       string
	inputTokens  int64
	outputTokens int64
}

// runOnce 执行一次 opencode 调用
// 超时配置逻辑：
// - Agent.TurnTimeoutMs < 0: 明确无超时
// - Agent.TurnTimeoutMs == 0: 使用 Codex.TurnTimeoutMs
// - Agent.TurnTimeoutMs > 0: 使用 Agent.TurnTimeoutMs
// 最终值 <= 0 表示无超时限制
func (r *openCodeRunner) runOnce(
	ctx context.Context,
	workspacePath string,
	prompt string,
	sessionID string,
	callback EventCallback,
) (*openCodeRunResult, error) {
	// 获取超时配置
	var turnTimeoutMs int64

	// Agent.TurnTimeoutMs < 0 表示明确无超时
	// Agent.TurnTimeoutMs == 0 表示使用 Codex 配置
	// Agent.TurnTimeoutMs > 0 表示使用 Agent 配置
	if r.cfg.Agent.TurnTimeoutMs < 0 {
		// 明确无超时
		turnTimeoutMs = -1
	} else if r.cfg.Agent.TurnTimeoutMs > 0 {
		// 使用 Agent 配置
		turnTimeoutMs = r.cfg.Agent.TurnTimeoutMs
	} else {
		// Agent == 0, fallback 到 Codex
		turnTimeoutMs = r.cfg.Codex.TurnTimeoutMs
	}

	// 无超时限制: TurnTimeoutMs <= 0 表示永久等待
	noTimeout := turnTimeoutMs <= 0

	var runCtx context.Context
	var cancel context.CancelFunc

	if noTimeout {
		// 无超时限制，使用原始 context
		runCtx = ctx
		cancel = func() {} // 空取消函数，避免 nil
	} else {
		// 有超时限制
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(turnTimeoutMs)*time.Millisecond)
	}
	defer cancel()

	// 构建 opencode 命令
	// opencode run "<prompt>" --output-format json
	command := "opencode"
	if r.cfg.OpenCode != nil && r.cfg.OpenCode.Command != "" {
		command = r.cfg.OpenCode.Command
	}
	if r.cfg.Agent.Command != "" {
		command = r.cfg.Agent.Command
	}

	args := []string{"run", prompt, "--output-format", "json"}

	// 追加额外参数
	if r.cfg.OpenCode != nil && len(r.cfg.OpenCode.ExtraArgs) > 0 {
		args = append(args, r.cfg.OpenCode.ExtraArgs...)
	}
	cmd := exec.CommandContext(runCtx, command, args...)
	cmd.Dir = workspacePath

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdout.(interface{ Close() error }).Close()
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	// 丢弃 stderr（诊断日志）
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
		}
	}()

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start opencode: %w", err)
	}

	result := &openCodeRunResult{}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 10*1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var event openCodeEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// 非 JSON 行跳过（可能是诊断输出）
			continue
		}

		if callback != nil {
			callback(event.Type, map[string]any{
				"session_id": sessionID,
				"event":      event,
				"raw":        line,
			})
		}

		switch event.Type {
		case "session_complete":
			result.success = true
			if event.Usage != nil {
				result.inputTokens = event.Usage.InputTokens
				result.outputTokens = event.Usage.OutputTokens
			}
			// exit_code != 0 视为失败
			if event.ExitCode != nil && *event.ExitCode != 0 {
				result.success = false
				result.errMsg = fmt.Sprintf("exit code %d", *event.ExitCode)
			}
		case "error":
			result.success = false
			result.errMsg = event.Error
			if result.errMsg == "" {
				result.errMsg = event.Content
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		// 未从事件流获得 session_complete 时，以退出码判断
		if !result.success && result.errMsg == "" {
			result.errMsg = fmt.Sprintf("exit: %v", err)
		}
	}

	// 若从未收到 session_complete，视为成功（某些版本不输出该事件）
	// 但如有错误标记则保持失败
	if !result.success && result.errMsg == "" {
		result.success = true
	}

	return result, nil
}
