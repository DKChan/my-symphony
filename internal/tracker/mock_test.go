// Package tracker 提供问题跟踪器客户端实现
package tracker

import (
	"context"
	"testing"
	"time"

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

// TestMockClient_ConversationHistory 测试 MockClient 对话历史功能
func TestMockClient_ConversationHistory(t *testing.T) {
	mockIssues := []config.MockIssueConfig{
		{ID: "1", Identifier: "TEST-1", Title: "测试任务1", State: "Todo"},
		{ID: "2", Identifier: "TEST-2", Title: "测试任务2", State: "In Progress"},
	}

	client := NewMockClient(mockIssues)
	ctx := context.Background()

	t.Run("AppendConversation_成功", func(t *testing.T) {
		turn := domain.ConversationTurn{
			Role:      "user",
			Content:   "想要添加用户登录功能",
			Timestamp: time.Now(),
		}

		err := client.AppendConversation(ctx, "TEST-1", turn)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// 验证对话记录被存储
		history, err := client.GetConversationHistory(ctx, "TEST-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(history) != 1 {
			t.Fatalf("expected 1 turn, got %d", len(history))
		}
		if history[0].Role != "user" {
			t.Errorf("expected role 'user', got '%s'", history[0].Role)
		}
		if history[0].Content != "想要添加用户登录功能" {
			t.Errorf("unexpected content: %s", history[0].Content)
		}
	})

	t.Run("AppendConversation_多条记录", func(t *testing.T) {
		turns := []domain.ConversationTurn{
			{Role: "assistant", Content: "请问登录方式是邮箱还是手机号？", Timestamp: time.Now()},
			{Role: "user", Content: "邮箱", Timestamp: time.Now()},
		}

		for _, turn := range turns {
			err := client.AppendConversation(ctx, "TEST-1", turn)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}

		history, err := client.GetConversationHistory(ctx, "TEST-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(history) != 3 { // 包括之前的 1 条
			t.Fatalf("expected 3 turns, got %d", len(history))
		}
	})

	t.Run("AppendConversation_任务不存在", func(t *testing.T) {
		turn := domain.ConversationTurn{
			Role:    "user",
			Content: "test",
		}

		err := client.AppendConversation(ctx, "TEST-NONEXISTENT", turn)
		if err == nil {
			t.Error("expected error for nonexistent issue")
		}
	})

	t.Run("GetConversationHistory_任务不存在", func(t *testing.T) {
		_, err := client.GetConversationHistory(ctx, "TEST-NONEXISTENT")
		if err == nil {
			t.Error("expected error for nonexistent issue")
		}
	})

	t.Run("GetConversationHistory_空历史", func(t *testing.T) {
		// TEST-2 没有任何对话记录
		history, err := client.GetConversationHistory(ctx, "TEST-2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if history != nil && len(history) != 0 {
			t.Errorf("expected empty history, got %d turns", len(history))
		}
	})
}
// TestMockClient_CreateTask 测试 MockClient 创建任务功能
func TestMockClient_CreateTask(t *testing.T) {
	client := NewMockClient(nil)
	ctx := context.Background()

	t.Run("CreateTask_成功", func(t *testing.T) {
		issue, err := client.CreateTask(ctx, "测试任务", "这是一个测试任务描述")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if issue.Title != "测试任务" {
			t.Errorf("expected title '测试任务', got '%s'", issue.Title)
		}
		if issue.State != "Todo" {
			t.Errorf("expected state 'Todo', got '%s'", issue.State)
		}
		if issue.Identifier == "" {
			t.Error("expected non-empty identifier")
		}
	})

	t.Run("CreateTask_多个任务", func(t *testing.T) {
		issue1, _ := client.CreateTask(ctx, "任务1", "描述1")
		issue2, _ := client.CreateTask(ctx, "任务2", "描述2")

		if issue1.Identifier == issue2.Identifier {
			t.Error("expected different identifiers for different tasks")
		}
	})
}

// TestMockClient_CreateSubTask 测试 MockClient 创建子任务功能
func TestMockClient_CreateSubTask(t *testing.T) {
	mockIssues := []config.MockIssueConfig{
		{ID: "1", Identifier: "SYM-100", Title: "父任务", State: "Todo"},
	}
	client := NewMockClient(mockIssues)
	ctx := context.Background()

	t.Run("CreateSubTask_无依赖", func(t *testing.T) {
		subTask, err := client.CreateSubTask(ctx, "SYM-100", "SYM-100-1: 需求澄清", "收集需求", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if subTask.Title != "SYM-100-1: 需求澄清" {
			t.Errorf("unexpected title: %s", subTask.Title)
		}
		if subTask.State != "Todo" {
			t.Errorf("expected state 'Todo', got '%s'", subTask.State)
		}
		if len(subTask.BlockedBy) != 0 {
			t.Errorf("expected no blockers, got %d", len(subTask.BlockedBy))
		}
	})

	t.Run("CreateSubTask_带依赖", func(t *testing.T) {
		// 创建第二个子任务，依赖第一个
		subTask1, _ := client.CreateSubTask(ctx, "SYM-100", "SYM-100-2: BDD审核", "审核BDD", []string{"SYM-100-1"})

		if len(subTask1.BlockedBy) != 1 {
			t.Fatalf("expected 1 blocker, got %d", len(subTask1.BlockedBy))
		}
		if subTask1.BlockedBy[0].Identifier == nil || *subTask1.BlockedBy[0].Identifier != "SYM-100-1" {
			t.Error("expected blocker to be SYM-100-1")
		}
	})

	t.Run("CreateSubTask_标识符递增", func(t *testing.T) {
		subTask1, _ := client.CreateSubTask(ctx, "SYM-100", "任务1", "描述", nil)
		subTask2, _ := client.CreateSubTask(ctx, "SYM-100", "任务2", "描述", nil)

		// 验证标识符递增
		if subTask1.Identifier == subTask2.Identifier {
			t.Error("expected different identifiers for different subtasks")
		}
	})
}
