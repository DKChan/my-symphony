// Package tracker 提供问题跟踪器客户端实现
package tracker

import (
	"context"
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
	issues    []*domain.Issue      // 所有问题
	stateHist map[string][]string  // 状态变更历史
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
		issues:    issues,
		stateHist: make(map[string][]string),
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