// Package agent - Codex app-server 协议实现
package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/domain"
	"github.com/dministrator/symphony/internal/logging"
)

// codexRunner 使用 Codex app-server JSON-RPC 协议的运行器
type codexRunner struct {
	cfg *config.Config
}

func newCodexRunner(cfg *config.Config) Runner {
	return &codexRunner{cfg: cfg}
}

// codexSession Codex 会话状态
type codexSession struct {
	ThreadID string
	TurnID   string
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   io.Reader
	stderr   io.Reader
	mu       sync.Mutex
}

// sessionID 返回会话 ID
func (s *codexSession) sessionID() string {
	return fmt.Sprintf("%s-%s", s.ThreadID, s.TurnID)
}

// sendRequest 发送 JSON-RPC 请求
func (s *codexSession) sendRequest(req map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	_, err = s.stdin.Write(append(data, '\n'))
	return err
}

// readLine 从 stdout 读取一行 JSON
func (s *codexSession) readLine(ctx context.Context, timeout time.Duration) (map[string]any, error) {
	type readResult struct {
		msg map[string]any
		err error
	}
	ch := make(chan readResult, 1)

	go func() {
		scanner := bufio.NewScanner(s.stdout)
		if scanner.Scan() {
			var msg map[string]any
			if err := json.Unmarshal([]byte(scanner.Text()), &msg); err != nil {
				ch <- readResult{err: fmt.Errorf("json parse error: %w", err)}
				return
			}
			ch <- readResult{msg: msg}
		} else if err := scanner.Err(); err != nil {
			ch <- readResult{err: err}
		} else {
			ch <- readResult{err: fmt.Errorf("stdout closed")}
		}
	}()

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case r := <-ch:
		return r.msg, r.err
	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("response_timeout")
	}
}

// stop 停止会话
func (s *codexSession) stop() {
	if s.stdin != nil {
		s.stdin.Close()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
		s.cmd.Wait()
	}
}

// RunAttempt 实现 Runner 接口
func (r *codexRunner) RunAttempt(
	ctx context.Context,
	issue *domain.Issue,
	workspacePath string,
	attempt *int,
	promptTemplate string,
	callback EventCallback,
) (*RunAttemptResult, error) {
	prompt := buildPrompt(issue, attempt, promptTemplate)

	var attemptNum int
	if attempt != nil {
		attemptNum = *attempt
	}

	logging.Debug("starting codex run attempt",
		"task_id", issue.ID,
		"identifier", issue.Identifier,
		"attempt", attemptNum,
		"workspace_path", workspacePath,
	)

	session, err := r.startSession(ctx, workspacePath)
	if err != nil {
		logging.Error("failed to start codex session",
			"task_id", issue.ID,
			"identifier", issue.Identifier,
			"error", err.Error(),
		)
		return nil, fmt.Errorf("failed to start codex session: %w", err)
	}
	defer session.stop()

	if callback != nil {
		callback("session_started", map[string]any{
			"session_id": session.sessionID(),
			"thread_id":  session.ThreadID,
			"agent":      "codex",
		})
	}

	logging.Info("codex session started",
		"task_id", issue.ID,
		"identifier", issue.Identifier,
		"session_id", session.sessionID(),
		"thread_id", session.ThreadID,
	)

	turnCount := 0
	tokenUsage := &TokenUsage{}

	for turnNum := 1; turnNum <= r.cfg.Agent.MaxTurns; turnNum++ {
		turnPrompt := prompt
		if turnNum > 1 {
			turnPrompt = "Continue working on the issue. Check if there's more work to do."
		}

		turnResult, err := r.runTurn(ctx, session, turnPrompt, issue, callback)
		if err != nil {
			logging.Error("turn failed with error",
				"task_id", issue.ID,
				"identifier", issue.Identifier,
				"turn_num", turnNum,
				"error", err.Error(),
			)
			return &RunAttemptResult{
				Success:   false,
				Error:     err,
				TurnCount: turnCount,
			}, nil
		}

		turnCount++
		tokenUsage.InputTokens += turnResult.inputTokens
		tokenUsage.OutputTokens += turnResult.outputTokens
		tokenUsage.TotalTokens += turnResult.totalTokens

		if !turnResult.success {
			logging.Warn("turn failed",
				"task_id", issue.ID,
				"identifier", issue.Identifier,
				"turn_num", turnNum,
				"error_message", turnResult.errMsg,
			)
			return &RunAttemptResult{
				Success:    false,
				Error:      fmt.Errorf("turn failed: %s", turnResult.errMsg),
				TurnCount:  turnCount,
				TokenUsage: tokenUsage,
			}, nil
		}

		if !turnResult.shouldContinue {
			logging.Info("turn completed, no more work needed",
				"task_id", issue.ID,
				"identifier", issue.Identifier,
				"turn_num", turnNum,
			)
			break
		}
	}

	logging.Info("codex run attempt completed",
		"task_id", issue.ID,
		"identifier", issue.Identifier,
		"turn_count", turnCount,
		"input_tokens", tokenUsage.InputTokens,
		"output_tokens", tokenUsage.OutputTokens,
		"total_tokens", tokenUsage.TotalTokens,
	)

	return &RunAttemptResult{
		Success:    true,
		TurnCount:  turnCount,
		TokenUsage: tokenUsage,
	}, nil
}

// startSession 启动 Codex app-server 会话并完成握手
func (r *codexRunner) startSession(ctx context.Context, workspacePath string) (*codexSession, error) {
	command := r.cfg.Codex.Command
	if r.cfg.Agent.Command != "" {
		command = r.cfg.Agent.Command
	}

	cmd := exec.CommandContext(ctx, "bash", "-lc", command)
	cmd.Dir = workspacePath
	cmd.Env = append(os.Environ())

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.(io.ReadCloser).Close()
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start codex: %w", err)
	}

	session := &codexSession{cmd: cmd, stdin: stdin, stdout: stdout, stderr: stderr}

	if err := r.handshake(ctx, session, workspacePath); err != nil {
		session.stop()
		return nil, fmt.Errorf("handshake: %w", err)
	}

	return session, nil
}

// handshake 执行初始化握手序列
func (r *codexRunner) handshake(ctx context.Context, session *codexSession, workspacePath string) error {
	timeout := time.Duration(r.cfg.Codex.ReadTimeoutMs) * time.Millisecond

	// 1. initialize
	if err := session.sendRequest(map[string]any{
		"id":     1,
		"method": "initialize",
		"params": map[string]any{
			"clientInfo":   map[string]any{"name": "symphony", "version": "1.0"},
			"capabilities": map[string]any{},
		},
	}); err != nil {
		return fmt.Errorf("initialize: %w", err)
	}
	if _, err := session.readLine(ctx, timeout); err != nil {
		return fmt.Errorf("initialize response: %w", err)
	}

	// 2. initialized 通知
	if err := session.sendRequest(map[string]any{
		"method": "initialized",
		"params": map[string]any{},
	}); err != nil {
		return fmt.Errorf("initialized: %w", err)
	}

	// 3. thread/start
	threadParams := map[string]any{
		"approvalPolicy": r.cfg.Codex.ApprovalPolicy,
		"sandbox":        r.cfg.Codex.ThreadSandbox,
		"cwd":            workspacePath,
	}
	if err := session.sendRequest(map[string]any{
		"id":     2,
		"method": "thread/start",
		"params": threadParams,
	}); err != nil {
		return fmt.Errorf("thread/start: %w", err)
	}

	resp, err := session.readLine(ctx, timeout)
	if err != nil {
		return fmt.Errorf("thread/start response: %w", err)
	}

	// 提取 thread_id
	if result, ok := resp["result"].(map[string]any); ok {
		if thread, ok := result["thread"].(map[string]any); ok {
			if tid, ok := thread["id"].(string); ok {
				session.ThreadID = tid
			}
		}
	}

	return nil
}

// codexTurnResult 单次 turn 结果
type codexTurnResult struct {
	success        bool
	errMsg         string
	shouldContinue bool
	inputTokens    int64
	outputTokens   int64
	totalTokens    int64
}

// runTurn 执行一次 turn
// 如果 TurnTimeoutMs <= 0，则无超时限制，允许无限等待
func (r *codexRunner) runTurn(
	ctx context.Context,
	session *codexSession,
	prompt string,
	issue *domain.Issue,
	callback EventCallback,
) (*codexTurnResult, error) {
	turnTimeoutMs := r.cfg.Codex.TurnTimeoutMs

	// 无超时限制: TurnTimeoutMs <= 0 表示永久等待
	noTimeout := turnTimeoutMs <= 0

	var turnCtx context.Context
	var cancel context.CancelFunc

	if noTimeout {
		// 无超时限制，使用原始 context
		turnCtx = ctx
		cancel = func() {} // 空取消函数，避免 nil
	} else {
		// 有超时限制
		turnCtx, cancel = context.WithTimeout(ctx, time.Duration(turnTimeoutMs)*time.Millisecond)
	}
	defer cancel()

	turnParams := map[string]any{
		"threadId":       session.ThreadID,
		"input":          []map[string]any{{"type": "text", "text": prompt}},
		"cwd":            session.cmd.Dir,
		"title":          fmt.Sprintf("%s: %s", issue.Identifier, issue.Title),
		"approvalPolicy": r.cfg.Codex.ApprovalPolicy,
	}
	if r.cfg.Codex.TurnSandboxPolicy != "" {
		turnParams["sandboxPolicy"] = map[string]any{"type": r.cfg.Codex.TurnSandboxPolicy}
	}

	if err := session.sendRequest(map[string]any{
		"id":     3,
		"method": "turn/start",
		"params": turnParams,
	}); err != nil {
		return nil, fmt.Errorf("turn/start: %w", err)
	}

	result := &codexTurnResult{}
	scanner := bufio.NewScanner(session.stdout)
	scanner.Buffer(make([]byte, 10*1024*1024), 10*1024*1024)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			var msg map[string]any
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				continue
			}

			r.processMessage(msg, result, callback, session)

			if method, ok := msg["method"].(string); ok {
				switch method {
				case "turn/completed":
					result.success = true
					result.shouldContinue = true
					return
				case "turn/failed", "turn/cancelled":
					result.success = false
					return
				}
			}

			// 提取 turn_id
			if resultMap, ok := msg["result"].(map[string]any); ok {
				if turn, ok := resultMap["turn"].(map[string]any); ok {
					if tid, ok := turn["id"].(string); ok {
						session.TurnID = tid
					}
				}
			}
		}
	}()

	select {
	case <-done:
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("scanner: %w", err)
		}
	case <-turnCtx.Done():
		// 区分超时和外部取消
		if noTimeout {
			// 无超时配置时，只可能是外部取消（如服务关闭）
			return nil, fmt.Errorf("context_cancelled: %v", turnCtx.Err())
		}
		return nil, fmt.Errorf("turn_timeout")
	}

	return result, nil
}

// processMessage 处理单条消息
func (r *codexRunner) processMessage(
	msg map[string]any,
	result *codexTurnResult,
	callback EventCallback,
	session *codexSession,
) {
	method, _ := msg["method"].(string)

	if callback != nil {
		callback(method, msg)
	}

	// 提取 token 使用量
	if params, ok := msg["params"].(map[string]any); ok {
		if usage, ok := params["usage"].(map[string]any); ok {
			if v, ok := usage["input_tokens"].(float64); ok {
				result.inputTokens = int64(v)
			}
			if v, ok := usage["output_tokens"].(float64); ok {
				result.outputTokens = int64(v)
			}
			if v, ok := usage["total_tokens"].(float64); ok {
				result.totalTokens = int64(v)
			}
		}
	}

	// 自动审批
	if method == "item/tool/call" || strings.HasPrefix(method, "approval/") {
		if id, ok := msg["id"].(float64); ok && id != 0 {
			session.sendRequest(map[string]any{
				"id":     int(id),
				"result": map[string]any{"approved": true},
			})
		}
	}

	// 用户输入请求 → 硬失败
	if method == "item/tool/requestUserInput" {
		result.success = false
		result.errMsg = "turn_input_required"
	}
}
