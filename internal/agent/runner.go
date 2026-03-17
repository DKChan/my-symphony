// Package agent 提供编码代理运行器接口和工厂
package agent

import (
	"context"

	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/domain"
)

// EventCallback 事件回调函数
type EventCallback func(event string, data any)

// RunAttemptResult 运行尝试结果
type RunAttemptResult struct {
	Success    bool
	Error      error
	TurnCount  int
	TokenUsage *TokenUsage
}

// TokenUsage token 使用统计
type TokenUsage struct {
	InputTokens  int64
	OutputTokens int64
	TotalTokens  int64
}

// Runner 代理运行器接口
type Runner interface {
	// RunAttempt 执行一次运行尝试
	RunAttempt(
		ctx context.Context,
		issue *domain.Issue,
		workspacePath string,
		attempt *int,
		promptTemplate string,
		callback EventCallback,
	) (*RunAttemptResult, error)
}

// NewRunner 根据配置创建对应的代理运行器
func NewRunner(cfg *config.Config) Runner {
	kind := cfg.Agent.Kind
	if kind == "" {
		kind = "codex"
	}
	switch kind {
	case "claude":
		return newClaudeRunner(cfg)
	case "opencode":
		return newOpenCodeRunner(cfg)
	default:
		return newCodexRunner(cfg)
	}
}
