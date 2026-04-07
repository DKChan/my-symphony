// Package harness 提供 P-G-E 编排引擎
package harness

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/dministrator/symphony/internal/common/errors"
)

// AgentCaller BMAD Agent 调用接口
type AgentCaller interface {
	// Call 调用 Agent
	Call(ctx context.Context, input *AgentInput) (*AgentOutput, error)

	// CheckAvailability 检查 Agent 可用性
	CheckAvailability() error
}

// AgentInput Agent 输入
type AgentInput struct {
	// AgentName Agent 名称 (如 "bmad-agent-pm")
	AgentName string `json:"agent_name"`
	// Task 任务描述
	Task string `json:"task"`
	// Context 上下文信息
	Context map[string]string `json:"context,omitempty"`
	// WorkingDir 工作目录
	WorkingDir string `json:"working_dir"`
}

// AgentOutput Agent 输出
type AgentOutput struct {
	// Success 是否成功
	Success bool `json:"success"`
	// Content 输出内容
	Content string `json:"content"`
	// Duration 执行时长
	Duration time.Duration `json:"duration"`
	// Error 错误信息
	Error string `json:"error,omitempty"`
}

// AgentCallerImpl AgentCaller 实现
type AgentCallerImpl struct {
	// cliPath CLI 路径 (如 "claude")
	cliPath string
	// timeout 默认超时时间
	timeout time.Duration
}

// NewAgentCaller 创建新的 AgentCaller
func NewAgentCaller(cliPath string, timeout time.Duration) *AgentCallerImpl {
	return &AgentCallerImpl{
		cliPath: cliPath,
		timeout: timeout,
	}
}

// Call 调用 Agent
func (a *AgentCallerImpl) Call(ctx context.Context, input *AgentInput) (*AgentOutput, error) {
	startTime := time.Now()

	// 检查可用性
	if err := a.CheckAvailability(); err != nil {
		return nil, err
	}

	// 设置超时
	timeout := a.timeout
	if timeout <= 0 {
		// 无超时限制，使用一个很长的超时作为兜底
		timeout = 24 * time.Hour * 365 // 1 year
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 构建 prompt
	prompt := a.buildPrompt(input)

	// 构建 CLI 命令
	// BMAD agent 使用 /agent-name 语法
	// 例如: claude --print /bmad-agent-pm "任务描述"
	args := []string{
		"--print",
		"--output-format", "stream-json",
		fmt.Sprintf("/%s", input.AgentName),
		prompt,
	}

	slog.Debug("calling agent",
		"agent_name", input.AgentName,
		"cli_path", a.cliPath,
		"working_dir", input.WorkingDir,
	)

	// 执行命令
	cmd := exec.CommandContext(ctx, a.cliPath, args...)
	if input.WorkingDir != "" {
		cmd.Dir = input.WorkingDir
	}

	output, err := cmd.Output()
	duration := time.Since(startTime)

	// 处理超时
	if ctx.Err() == context.DeadlineExceeded {
		slog.Error("agent timeout",
			"agent_name", input.AgentName,
			"duration", duration,
		)
		return nil, errors.ErrAgentTimeout
	}

	// 处理执行错误
	if err != nil {
		errMsg := err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok {
			errMsg = string(exitErr.Stderr)
		}
		slog.Error("agent execution failed",
			"agent_name", input.AgentName,
			"error", errMsg,
		)
		return &AgentOutput{
			Success:  false,
			Content:  "",
			Duration: duration,
			Error:    errMsg,
		}, errors.WrapError("agent", "execution_failed", errMsg, err)
	}

	// 解析输出
	content := a.parseOutput(output)

	slog.Info("agent call completed",
		"agent_name", input.AgentName,
		"duration", duration,
		"content_length", len(content),
	)

	return &AgentOutput{
		Success:  true,
		Content:  content,
		Duration: duration,
	}, nil
}

// CheckAvailability 检查 Agent 可用性
func (a *AgentCallerImpl) CheckAvailability() error {
	// 检查 CLI 是否在 PATH 中
	path, err := exec.LookPath(a.cliPath)
	if err != nil {
		slog.Error("agent cli not found",
			"cli_path", a.cliPath,
			"error", err,
		)
		return errors.ErrAgentUnavailable
	}

	slog.Debug("agent cli available", "cli_path", a.cliPath, "resolved_path", path)
	return nil
}

// buildPrompt 构建 prompt
func (a *AgentCallerImpl) buildPrompt(input *AgentInput) string {
	var sb strings.Builder

	// 添加任务描述
	sb.WriteString(input.Task)

	// 添加上下文
	if len(input.Context) > 0 {
		sb.WriteString("\n\n上下文信息:\n")
		for k, v := range input.Context {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
		}
	}

	return sb.String()
}

// parseOutput 解析 Claude CLI 的 stream-json 输出
func (a *AgentCallerImpl) parseOutput(output []byte) string {
	// Claude CLI 使用 --output-format stream-json 时输出 JSON 行
	// 每行是一个 JSON 对象，包含 type 字段
	// 我们需要提取所有 content 类型的消息

	lines := strings.Split(string(output), "\n")
	var content strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 尝试解析为 JSON
		// 简化处理：直接收集非空行
		// 实际实现需要根据 Claude CLI 的输出格式解析
		if strings.HasPrefix(line, "{") {
			// JSON 行，尝试提取 content
			// 这里简化处理，实际需要完整解析
			content.WriteString(line)
			content.WriteString("\n")
		}
	}

	// 如果没有解析出 JSON 内容，直接返回原始输出
	if content.Len() == 0 {
		return string(output)
	}

	return content.String()
}