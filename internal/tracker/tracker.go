// Package tracker 提供问题跟踪器接口和工厂
package tracker

import (
	"context"

	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/domain"
)

// Tracker 问题跟踪器接口
type Tracker interface {
	// FetchCandidateIssues 获取处于活跃状态的候选问题
	FetchCandidateIssues(ctx context.Context, activeStates []string) ([]*domain.Issue, error)

	// FetchIssuesByStates 获取指定状态的问题（用于启动时清理）
	FetchIssuesByStates(ctx context.Context, states []string) ([]*domain.Issue, error)

	// FetchIssueStatesByIDs 按 ID 批量刷新问题状态
	FetchIssueStatesByIDs(ctx context.Context, ids []string) ([]*domain.Issue, error)
}

// NewTracker 根据配置创建对应的问题跟踪器
func NewTracker(cfg *config.Config) Tracker {
	switch cfg.Tracker.Kind {
	case "github":
		return NewGitHubClient(cfg.Tracker.APIKey, cfg.Tracker.Repo)
	case "mock":
		return NewMockClient(cfg.Tracker.MockIssues)
	default:
		// 默认 linear
		return NewLinearClient(cfg.Tracker.Endpoint, cfg.Tracker.APIKey, cfg.Tracker.ProjectSlug)
	}
}