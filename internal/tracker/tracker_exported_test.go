// Package tracker 提供问题跟踪器客户端实现
package tracker

import (
	"testing"

	"github.com/dministrator/symphony/internal/config"
	"github.com/stretchr/testify/assert"
)

// TestNewTracker 测试 NewTracker 工厂函数
func TestNewTracker(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.Config
		wantType string
	}{
		{
			name: "mock tracker",
			cfg: &config.Config{
				Tracker: config.TrackerConfig{
					Kind: "mock",
					MockIssues: []config.MockIssueConfig{
						{ID: "1", Identifier: "TEST-1", Title: "Test", State: "Todo"},
					},
				},
			},
			wantType: "*tracker.MockClient",
		},
		{
			name: "beads tracker",
			cfg: &config.Config{
				Tracker: config.TrackerConfig{
					Kind: "beads",
				},
			},
			wantType: "*tracker.BeadsClient",
		},
		{
			name: "github tracker",
			cfg: &config.Config{
				Tracker: config.TrackerConfig{
					Kind:   "github",
					APIKey: "test-token",
					Repo:   "owner/repo",
				},
			},
			wantType: "*tracker.GitHubClient",
		},
		{
			name: "unknown kind defaults to mock",
			cfg: &config.Config{
				Tracker: config.TrackerConfig{
					Kind: "unknown",
				},
			},
			wantType: "*tracker.MockClient",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := NewTracker(tt.cfg)
			assert.NotNil(t, tracker, "NewTracker should return non-nil Tracker")
			assert.IsType(t, tracker, getZeroValueForType(tt.wantType), "NewTracker should return correct type")
		})
	}
}

// TestNewMockClient 测试 NewMockClient 构造函数
func TestNewMockClient(t *testing.T) {
	t.Run("with issues", func(t *testing.T) {
		mockIssues := []config.MockIssueConfig{
			{ID: "1", Identifier: "TEST-1", Title: "Task 1", State: "Todo"},
			{ID: "2", Identifier: "TEST-2", Title: "Task 2", State: "In Progress"},
		}

		client := NewMockClient(mockIssues)
		assert.NotNil(t, client, "NewMockClient should return non-nil client")
	})

	t.Run("nil issues", func(t *testing.T) {
		client := NewMockClient(nil)
		assert.NotNil(t, client, "NewMockClient should handle nil issues")
	})

	t.Run("empty issues", func(t *testing.T) {
		client := NewMockClient([]config.MockIssueConfig{})
		assert.NotNil(t, client, "NewMockClient should handle empty issues")
	})
}

// TestNewGitHubClient 测试 NewGitHubClient 构造函数
func TestNewGitHubClient(t *testing.T) {
	t.Run("creates client with valid params", func(t *testing.T) {
		client := NewGitHubClient("test-token", "owner/repo")
		assert.NotNil(t, client, "NewGitHubClient should return non-nil client")
	})

	t.Run("creates client with empty params", func(t *testing.T) {
		client := NewGitHubClient("", "")
		assert.NotNil(t, client, "NewGitHubClient should handle empty params")
	})
}

// getZeroValueForType 返回类型名称对应的零值，用于 assert.IsType
func getZeroValueForType(typeName string) interface{} {
	switch typeName {
	case "*tracker.MockClient":
		return &MockClient{}
	case "*tracker.BeadsClient":
		return &BeadsClient{}
	case "*tracker.GitHubClient":
		return &GitHubClient{}
	default:
		return nil
	}
}