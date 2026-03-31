// Package tracker 提供问题跟踪器接口和工厂
package tracker

import (
	"context"

	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/domain"
)

// VerificationReport 验收报告类型别名
type VerificationReport = domain.VerificationReport

// Tracker 问题跟踪器接口
type Tracker interface {
	// FetchCandidateIssues 获取处于活跃状态的候选问题
	FetchCandidateIssues(ctx context.Context, activeStates []string) ([]*domain.Issue, error)

	// FetchIssuesByStates 获取指定状态的问题（用于启动时清理）
	FetchIssuesByStates(ctx context.Context, states []string) ([]*domain.Issue, error)

	// FetchIssueStatesByIDs 按 ID 批量刷新问题状态
	FetchIssueStatesByIDs(ctx context.Context, ids []string) ([]*domain.Issue, error)

	// GetTask 获取单个任务详情
	GetTask(ctx context.Context, identifier string) (*domain.Issue, error)

	// CheckAvailability 检查跟踪器是否可用
	CheckAvailability() error

	// CreateTask 创建新任务
	CreateTask(ctx context.Context, title, description string) (*domain.Issue, error)

	// CreateSubTask 创建子任务（带依赖关系）
	CreateSubTask(ctx context.Context, parentIdentifier string, title, description string, blockedBy []string) (*domain.Issue, error)

	// UpdateStage 更新任务阶段状态
	UpdateStage(ctx context.Context, identifier string, stage domain.StageState) error

	// GetStageState 获取任务的阶段状态（用于崩溃恢复）
	GetStageState(ctx context.Context, identifier string) (*domain.StageState, error)

	// AppendConversation 追加对话记录
	AppendConversation(ctx context.Context, identifier string, turn domain.ConversationTurn) error

	// GetConversationHistory 获取对话历史记录
	GetConversationHistory(ctx context.Context, identifier string) ([]domain.ConversationTurn, error)

	// ListTasksByState 按状态获取任务列表
	ListTasksByState(ctx context.Context, states []string) ([]*domain.Issue, error)

	// GetBDDContent 获取任务的 BDD 规则内容
	GetBDDContent(ctx context.Context, identifier string) (string, error)

	// UpdateBDDContent 更新任务的 BDD 规则内容
	UpdateBDDContent(ctx context.Context, identifier string, content string) error

	// ApproveBDD 通过 BDD 审核
	ApproveBDD(ctx context.Context, identifier string) error

	// RejectBDD 驳回 BDD 审核（附带驳回原因）
	RejectBDD(ctx context.Context, identifier string, reason string) error

	// GetArchitectureContent 获取任务的架构设计内容
	GetArchitectureContent(ctx context.Context, identifier string) (string, error)

	// GetTDDContent 获取任务的 TDD 规则内容
	GetTDDContent(ctx context.Context, identifier string) (string, error)

	// UpdateArchitectureContent 更新任务的架构设计内容
	UpdateArchitectureContent(ctx context.Context, identifier string, content string) error

	// UpdateTDDContent 更新任务的 TDD 规则内容
	UpdateTDDContent(ctx context.Context, identifier string, content string) error

	// ApproveArchitecture 通过架构审核
	ApproveArchitecture(ctx context.Context, identifier string) error

	// RejectArchitecture 驳回架构审核（附带驳回原因）
	RejectArchitecture(ctx context.Context, identifier string, reason string) error

	// GetVerificationReport 获取任务的验收报告内容
	GetVerificationReport(ctx context.Context, identifier string) (*VerificationReport, error)

	// UpdateVerificationReport 更新任务的验收报告
	UpdateVerificationReport(ctx context.Context, identifier string, report *VerificationReport) error

	// ApproveVerification 通过验收
	ApproveVerification(ctx context.Context, identifier string) error

	// RejectVerification 驳回验收（流转回实现中）
	RejectVerification(ctx context.Context, identifier string, reason string) error
}

// NewTracker 根据配置创建对应的问题跟踪器
func NewTracker(cfg *config.Config) Tracker {
	switch cfg.Tracker.Kind {
	case "github":
		return NewGitHubClient(cfg.Tracker.APIKey, cfg.Tracker.Repo)
	case "mock":
		return NewMockClient(cfg.Tracker.MockIssues)
	case "beads":
		return NewBeadsClient()
	default:
		// 默认 linear
		return NewLinearClient(cfg.Tracker.Endpoint, cfg.Tracker.APIKey, cfg.Tracker.ProjectSlug)
	}
}