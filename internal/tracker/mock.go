// Package tracker 提供问题跟踪器客户端实现
package tracker

import (
	"context"
	"fmt"
	"time"

	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/domain"
)

// MockIssue Mock问题结构（用于配置）
type MockIssue struct {
	ID          string   `json:"id"`
	Identifier  string   `json:"identifier"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	State       string   `json:"state"`
	Priority    int      `json:"priority,omitempty"`
	Labels      []string `json:"labels,omitempty"`
}

// MockClient Mock跟踪器客户端（用于本地测试）
type MockClient struct {
	issues         []*domain.Issue                    // 所有问题
	stateHist      map[string][]string                // 状态变更历史
	conversations  map[string][]domain.ConversationTurn // 对话历史
	bddContents    map[string]string                  // BDD 规则内容
}

// NewMockClient 创建新的Mock客户端
func NewMockClient(mockIssues []config.MockIssueConfig) *MockClient {
	issues := make([]*domain.Issue, len(mockIssues))
	for i, mi := range mockIssues {
		issue := &domain.Issue{
			ID:          mi.ID,
			Identifier:  mi.Identifier,
			Title:       mi.Title,
			State:       mi.State,
			Labels:      mi.Labels,
			BlockedBy:   make([]domain.BlockerRef, 0),
		}
		if mi.Description != "" {
			issue.Description = &mi.Description
		}
		if mi.Priority > 0 {
			issue.Priority = &mi.Priority
		}
		now := time.Now()
		issue.CreatedAt = &now
		issue.UpdatedAt = &now
		issues[i] = issue
	}
	return &MockClient{
		issues:        issues,
		stateHist:     make(map[string][]string),
		conversations: make(map[string][]domain.ConversationTurn),
		bddContents:   make(map[string]string),
	}
}

// FetchCandidateIssues 获取候选问题（返回活跃状态的问题）
func (c *MockClient) FetchCandidateIssues(ctx context.Context, activeStates []string) ([]*domain.Issue, error) {
	activeSet := make(map[string]bool)
	for _, s := range activeStates {
		activeSet[s] = true
	}

	var result []*domain.Issue
	for _, issue := range c.issues {
		if activeSet[issue.State] {
			result = append(result, issue)
		}
	}
	return result, nil
}

// FetchIssuesByStates 获取指定状态的问题
func (c *MockClient) FetchIssuesByStates(ctx context.Context, states []string) ([]*domain.Issue, error) {
	stateSet := make(map[string]bool)
	for _, s := range states {
		stateSet[s] = true
	}

	var result []*domain.Issue
	for _, issue := range c.issues {
		if stateSet[issue.State] {
			result = append(result, issue)
		}
	}
	return result, nil
}

// FetchIssueStatesByIDs 按ID获取问题状态
func (c *MockClient) FetchIssueStatesByIDs(ctx context.Context, ids []string) ([]*domain.Issue, error) {
	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
	}

	var result []*domain.Issue
	for _, issue := range c.issues {
		if idSet[issue.ID] {
			result = append(result, issue)
		}
	}
	return result, nil
}

// UpdateIssueState 更新问题的状态（用于测试时模拟状态变化）
func (c *MockClient) UpdateIssueState(id string, newState string) {
	for _, issue := range c.issues {
		if issue.ID == id {
			// 记录状态变更历史
			c.stateHist[id] = append(c.stateHist[id], issue.State)
			issue.State = newState
			now := time.Now()
			issue.UpdatedAt = &now
			break
		}
	}
}

// GetStateHistory 获取问题的状态变更历史
func (c *MockClient) GetStateHistory(id string) []string {
	return c.stateHist[id]
}

// AddIssue 添加新的Mock问题（用于动态测试）
func (c *MockClient) AddIssue(issue *domain.Issue) {
	c.issues = append(c.issues, issue)
}

// ClearIssues 清除所有问题
func (c *MockClient) ClearIssues() {
	c.issues = nil
}

// CheckAvailability 检查跟踪器可用性（Mock 始终可用）
func (c *MockClient) CheckAvailability() error {
	return nil
}

// CreateTask 创建新任务
func (c *MockClient) CreateTask(ctx context.Context, title, description string) (*domain.Issue, error) {
	now := time.Now()
	id := fmt.Sprintf("mock-%d", len(c.issues)+1)
	identifier := fmt.Sprintf("MOCK-%d", len(c.issues)+1)
	issue := &domain.Issue{
		ID:          id,
		Identifier:  identifier,
		Title:       title,
		Description: &description,
		State:       "Todo",
		Labels:      []string{},
		BlockedBy:   []domain.BlockerRef{},
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}
	c.issues = append(c.issues, issue)
	return issue, nil
}

// CreateSubTask 创建子任务（带依赖关系）
func (c *MockClient) CreateSubTask(ctx context.Context, parentIdentifier string, title, description string, blockedBy []string) (*domain.Issue, error) {
	now := time.Now()
	// 子任务 ID 使用父任务标识符作为前缀
	id := fmt.Sprintf("mock-%d", len(c.issues)+1)
	// 子任务标识符格式：PARENT-1, PARENT-2 等
	childNum := 1
	for _, issue := range c.issues {
		if issue.Identifier == parentIdentifier {
			// 计算已有子任务数量
			prefix := parentIdentifier + "-"
			for _, existing := range c.issues {
				if len(existing.Identifier) > len(prefix) && existing.Identifier[:len(prefix)] == prefix {
					childNum++
				}
			}
			break
		}
	}
	identifier := fmt.Sprintf("%s-%d", parentIdentifier, childNum)

	// 构建阻塞关系
	blockedByRefs := make([]domain.BlockerRef, 0, len(blockedBy))
	for _, blockerID := range blockedBy {
		// 查找阻塞项
		for _, issue := range c.issues {
			if issue.Identifier == blockerID {
				blockedByRefs = append(blockedByRefs, domain.BlockerRef{
					ID:         &issue.ID,
					Identifier: &issue.Identifier,
					State:      &issue.State,
				})
				break
			}
		}
	}

	issue := &domain.Issue{
		ID:          id,
		Identifier:  identifier,
		Title:       title,
		Description: &description,
		State:       "Todo",
		Labels:      []string{"subtask"},
		BlockedBy:   blockedByRefs,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}
	c.issues = append(c.issues, issue)
	return issue, nil
}

// UpdateStage 更新任务阶段状态
func (c *MockClient) UpdateStage(ctx context.Context, identifier string, stage domain.StageState) error {
	for _, issue := range c.issues {
		if issue.Identifier == identifier {
			now := time.Now()
			issue.UpdatedAt = &now
			return nil
		}
	}
	return fmt.Errorf("issue not found: %s", identifier)
}

// GetStageState 获取任务的阶段状态（用于崩溃恢复）
func (c *MockClient) GetStageState(ctx context.Context, identifier string) (*domain.StageState, error) {
	for _, issue := range c.issues {
		if issue.Identifier == identifier {
			// Mock 返回一个默认的阶段状态
			return &domain.StageState{
				Name:      "clarification",
				Status:    "pending",
				StartedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		}
	}
	return nil, fmt.Errorf("issue not found: %s", identifier)
}

// AppendConversation 追加对话记录
func (c *MockClient) AppendConversation(ctx context.Context, identifier string, turn domain.ConversationTurn) error {
	for _, issue := range c.issues {
		if issue.Identifier == identifier {
			// 存储对话记录
			c.conversations[identifier] = append(c.conversations[identifier], turn)
			now := time.Now()
			issue.UpdatedAt = &now
			return nil
		}
	}
	return fmt.Errorf("issue not found: %s", identifier)
}

// GetConversationHistory 获取对话历史记录
func (c *MockClient) GetConversationHistory(ctx context.Context, identifier string) ([]domain.ConversationTurn, error) {
	for _, issue := range c.issues {
		if issue.Identifier == identifier {
			return c.conversations[identifier], nil
		}
	}
	return nil, fmt.Errorf("issue not found: %s", identifier)
}

// ListTasksByState 按状态获取任务列表
func (c *MockClient) ListTasksByState(ctx context.Context, states []string) ([]*domain.Issue, error) {
	return c.FetchIssuesByStates(ctx, states)
}

// GetTask 获取单个任务详情
func (c *MockClient) GetTask(ctx context.Context, identifier string) (*domain.Issue, error) {
	for _, issue := range c.issues {
		if issue.Identifier == identifier {
			return issue, nil
		}
	}
	return nil, fmt.Errorf("issue not found: %s", identifier)
}

// GetBDDContent 获取任务的 BDD 规则内容
func (c *MockClient) GetBDDContent(ctx context.Context, identifier string) (string, error) {
	for _, issue := range c.issues {
		if issue.Identifier == identifier {
			content, exists := c.bddContents[identifier]
			if !exists {
				return "", fmt.Errorf("bdd content not found for issue: %s", identifier)
			}
			return content, nil
		}
	}
	return "", fmt.Errorf("issue not found: %s", identifier)
}

// UpdateBDDContent 更新任务的 BDD 规则内容
func (c *MockClient) UpdateBDDContent(ctx context.Context, identifier string, content string) error {
	for _, issue := range c.issues {
		if issue.Identifier == identifier {
			c.bddContents[identifier] = content
			now := time.Now()
			issue.UpdatedAt = &now
			return nil
		}
	}
	return fmt.Errorf("issue not found: %s", identifier)
}

// ApproveBDD 通过 BDD 审核
func (c *MockClient) ApproveBDD(ctx context.Context, identifier string) error {
	for _, issue := range c.issues {
		if issue.Identifier == identifier {
			// 更新状态为已通过 BDD 审核
			c.stateHist[issue.ID] = append(c.stateHist[issue.ID], issue.State)
			issue.State = "BDD Approved"
			now := time.Now()
			issue.UpdatedAt = &now
			return nil
		}
	}
	return fmt.Errorf("issue not found: %s", identifier)
}

// RejectBDD 驳回 BDD 审核
func (c *MockClient) RejectBDD(ctx context.Context, identifier string, reason string) error {
	for _, issue := range c.issues {
		if issue.Identifier == identifier {
			// 更新状态为已驳回 BDD 审核
			c.stateHist[issue.ID] = append(c.stateHist[issue.ID], issue.State)
			issue.State = "BDD Rejected"
			// 可以将驳回原因存储在描述中
			if issue.Description != nil {
				*issue.Description = *issue.Description + "\n\n驳回原因: " + reason
			} else {
				desc := "驳回原因: " + reason
				issue.Description = &desc
			}
			now := time.Now()
			issue.UpdatedAt = &now
			return nil
		}
	}
	return fmt.Errorf("issue not found: %s", identifier)
}

// SetBDDContent 设置 BDD 内容（用于测试）
func (c *MockClient) SetBDDContent(identifier string, content string) {
	c.bddContents[identifier] = content
}