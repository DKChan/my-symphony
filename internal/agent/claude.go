// Package agent - Claude Code 适配器
// 使用 `claude --print --output-format=stream-json` 非交互模式
package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/domain"
)

// claudeRunner 使用 Claude Code CLI 的运行器
type claudeRunner struct {
	cfg *config.Config
}

func newClaudeRunner(cfg *config.Config) Runner {
	return &claudeRunner{cfg: cfg}
}

// claudeEvent Claude stream-json 输出事件结构
type claudeEvent struct {
	Type    string          `json:"type"`
	Message json.RawMessage `json:"message,omitempty"`
	// system 事件字段
	Subtype string `json:"subtype,omitempty"`
	// result 事件字段
	Result    string       `json:"result,omitempty"`
	SessionID string       `json:"session_id,omitempty"`
	Usage     *claudeUsage `json:"usage,omitempty"`
	IsError   bool         `json:"is_error,omitempty"`
	CostUSD   float64      `json:"cost_usd,omitempty"`
	// assistant 消息内容
	Content json.RawMessage `json:"content,omitempty"`
}

// claudeUsage token 使用量
type claudeUsage struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
}

// RunAttempt 实现 Runner 接口
func (r *claudeRunner) RunAttempt(
	ctx context.Context,
	issue *domain.Issue,
	workspacePath string,
	attempt *int,
	promptTemplate string,
	callback EventCallback,
) (*RunAttemptResult, error) {
	prompt := buildPrompt(issue, attempt, promptTemplate)

	sessionID := fmt.Sprintf("claude-%s-%d", issue.Identifier, time.Now().UnixNano())
	if callback != nil {
		callback("session_started", map[string]any{
			"session_id": sessionID,
			"agent":      "claude",
			"issue_id":   issue.ID,
		})
	}

	turnCount := 0
	tokenUsage := &TokenUsage{}

	// Claude 每次调用是独立的，不支持多 turn 续行，只执行一次
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
			Error:      fmt.Errorf("claude run failed: %s", result.errMsg),
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

// claudeRunResult 单次运行结果
type claudeRunResult struct {
	success      bool
	errMsg       string
	inputTokens  int64
	outputTokens int64
}

// runOnce 执行一次 claude 调用
// 超时配置逻辑：
// - Agent.TurnTimeoutMs < 0: 明确无超时
// - Agent.TurnTimeoutMs == 0: 使用 Codex.TurnTimeoutMs
// - Agent.TurnTimeoutMs > 0: 使用 Agent.TurnTimeoutMs
// 最终值 <= 0 表示无超时限制
func (r *claudeRunner) runOnce(
	ctx context.Context,
	workspacePath string,
	prompt string,
	sessionID string,
	callback EventCallback,
) (*claudeRunResult, error) {
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

	// 构建 claude 命令
	args := []string{
		"--print",
		"--verbose",
		"--output-format", "stream-json",
		"--no-session-persistence",
	}

	// 跳过权限检查（默认开启）
	skipPerms := true
	if r.cfg.Claude != nil {
		skipPerms = r.cfg.Claude.SkipPermissions
	}
	if skipPerms {
		args = append(args, "--dangerously-skip-permissions")
	}

	// 追加额外参数
	if r.cfg.Claude != nil && len(r.cfg.Claude.ExtraArgs) > 0 {
		args = append(args, r.cfg.Claude.ExtraArgs...)
	}

	// 支持自定义命令（例如指定模型）
	command := "claude"
	if r.cfg.Claude != nil && r.cfg.Claude.Command != "" {
		command = r.cfg.Claude.Command
	}
	if r.cfg.Agent.Command != "" {
		command = r.cfg.Agent.Command
	}

	args = append(args, prompt)
	cmd := exec.CommandContext(runCtx, command, args...)
	cmd.Dir = workspacePath

	// 清除 CLAUDECODE 环境变量，避免嵌套会话限制
	// Claude CLI 检测到此变量会拒绝在 Claude Code 会话中运行
	cmd.Env = filterEnvVars(os.Environ(), "CLAUDECODE", "CLAUDE_CODE_ENTRYPOINT")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdout.Close()
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	// 收集 stderr 输出用于错误诊断
	var stderrBuf strings.Builder
	var stderrMu sync.Mutex
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			stderrMu.Lock()
			stderrBuf.WriteString(scanner.Text())
			stderrBuf.WriteString("\n")
			stderrMu.Unlock()
		}
	}()

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start claude: %w", err)
	}

	result := &claudeRunResult{}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 10*1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var event claudeEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// 非 JSON 行可能是错误消息
			if callback != nil {
				callback("parse_error", map[string]any{
					"session_id": sessionID,
					"raw":        line,
					"error":      err.Error(),
				})
			}
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
		case "result":
			result.success = !event.IsError
			if event.IsError {
				result.errMsg = event.Result
			}
			if event.Usage != nil {
				result.inputTokens = event.Usage.InputTokens
				result.outputTokens = event.Usage.OutputTokens
			}
		case "system":
			if event.Subtype == "init" {
				// 会话初始化成功
				if callback != nil {
					callback("init", map[string]any{
						"session_id": sessionID,
					})
				}
			}
		case "error":
			result.success = false
			result.errMsg = event.Result
		}
	}

	if err := scanner.Err(); err != nil {
		// 区分超时和外部取消
		if runCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("turn_timeout (%dms)", turnTimeoutMs)
		}
		if runCtx.Err() == context.Canceled {
			// 外部取消（如服务关闭）
			return nil, fmt.Errorf("context_cancelled: %v", runCtx.Err())
		}
		return nil, fmt.Errorf("scanner: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		stderrMu.Lock()
		stderrOutput := stderrBuf.String()
		stderrMu.Unlock()

		// 非零退出但已有结果时不报错（claude 某些情况下退出码非0）
		if !result.success && result.errMsg == "" {
			if stderrOutput != "" {
				result.errMsg = fmt.Sprintf("exit: %v, stderr: %s", err, strings.TrimSpace(stderrOutput))
			} else {
				result.errMsg = fmt.Sprintf("exit: %v", err)
			}
		}
	}

	// 检查是否从未收到结果事件
	if result.errMsg == "" && !result.success {
		// 检查 stderr 是否有嵌套会话错误
		stderrMu.Lock()
		stderrOutput := stderrBuf.String()
		stderrMu.Unlock()

		if strings.Contains(stderrOutput, "Nested sessions") {
			return nil, fmt.Errorf("nested_session_blocked: claude CLI cannot run inside Claude Code session")
		}

		// 如果没有收到任何结果，但也没有错误，假设成功
		result.success = true
	}

	return result, nil
}

// filterEnvVars 过滤掉指定的环境变量
func filterEnvVars(env []string, excludeKeys ...string) []string {
	exclude := make(map[string]bool)
	for _, key := range excludeKeys {
		exclude[key] = true
	}

	result := make([]string, 0, len(env))
	for _, kv := range env {
		// 环境变量格式为 KEY=VALUE
		if idx := strings.Index(kv, "="); idx > 0 {
			key := kv[:idx]
			if !exclude[key] {
				result = append(result, kv)
			}
		} else {
			result = append(result, kv)
		}
	}
	return result
}
