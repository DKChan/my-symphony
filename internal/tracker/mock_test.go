// Package tracker 提供问题跟踪器客户端实现
package tracker

import (
	"context"
	"testing"

	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/domain"
)

func TestMockClient(t *testing.T) {
	mockIssues := []config.MockIssueConfig{
		{ID: "1", Identifier: "TEST-1", Title: "测试任务1", State: "Todo"},
		{ID: "2", Identifier: "TEST-2", Title: "测试任务2", State: "In Progress"},
		{ID: "3", Identifier: "TEST-3", Title: "测试任务3", State: "Done"},
		{ID: "4", Identifier: "TEST-4", Title: "测试任务4", State: "Todo", Priority: 1, Labels: []string{"bug", "urgent"}},
	}

	client := NewMockClient(mockIssues)

	t.Run("FetchCandidateIssues", func(t *testing.T) {
		ctx := context.Background()
		issues, err := client.FetchCandidateIssues(ctx, []string{"Todo", "In Progress"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(issues) != 3 {
			t.Fatalf("expected 3 issues, got %d", len(issues))
		}
	})

	t.Run("FetchIssuesByStates", func(t *testing.T) {
		ctx := context.Background()
		issues, err := client.FetchIssuesByStates(ctx, []string{"Done"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		if issues[0].Identifier != "TEST-3" {
			t.Errorf("expected TEST-3, got %s", issues[0].Identifier)
		}
	})

	t.Run("FetchIssueStatesByIDs", func(t *testing.T) {
		ctx := context.Background()
		issues, err := client.FetchIssueStatesByIDs(ctx, []string{"1", "2"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(issues) != 2 {
			t.Fatalf("expected 2 issues, got %d", len(issues))
		}
	})

	t.Run("UpdateIssueState", func(t *testing.T) {
		// 使用独立的 client 避免影响其他测试
		client2 := NewMockClient(mockIssues)
		client2.UpdateIssueState("1", "In Progress")
		ctx := context.Background()
		issues, _ := client2.FetchIssueStatesByIDs(ctx, []string{"1"})
		if len(issues) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(issues))
		}
		if issues[0].State != "In Progress" {
			t.Errorf("expected state 'In Progress', got '%s'", issues[0].State)
		}
		history := client2.GetStateHistory("1")
		if len(history) != 1 || history[0] != "Todo" {
			t.Errorf("unexpected state history: %v", history)
		}
	})

	t.Run("AddIssue", func(t *testing.T) {
		client.AddIssue(&domain.Issue{
			ID:         "5",
			Identifier: "TEST-5",
			Title:      "新任务",
			State:      "Todo",
		})
		ctx := context.Background()
		issues, _ := client.FetchCandidateIssues(ctx, []string{"Todo"})
		// client 中: TEST-1(Todo), TEST-2(In Progress), TEST-3(Done), TEST-4(Todo), TEST-5(Todo 新添加)
		// Todo 状态有: TEST-1, TEST-4, TEST-5 = 3 个
		if len(issues) != 3 {
			t.Errorf("expected 3 Todo issues, got %d", len(issues))
		}
	})
}