// Package tracker 提供问题跟踪器客户端实现
package tracker

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// beadsTestCLI 模拟 Beads CLI 的测试脚本路径
const beadsTestCLI = "beads_mock_cli"

// setupMockBeadsCLI 创建模拟 Beads CLI 脚本
func setupMockBeadsCLI(t *testing.T) (string, func()) {
	tmpDir := t.TempDir()
	cliPath := filepath.Join(tmpDir, beadsTestCLI)

	// 创建模拟 CLI 脚本
	script := `#!/bin/bash
case "$1" in
  --version)
    echo "beads v1.0.0"
    ;;
  issue)
    case "$2" in
      list)
        # beads issue list --state <state>
        state=""
        for arg in "$@"; do
          if [[ "$prev_arg" == "--state" ]]; then
            state="$arg"
          fi
          prev_arg="$arg"
        done
        if [[ "$state" == "Todo" ]]; then
          echo '[{"id":"1","identifier":"BEADS-1","title":"Task 1","state":"Todo","priority":1,"labels":["bug"]}]'
        elif [[ "$state" == "In Progress" ]]; then
          echo '[{"id":"2","identifier":"BEADS-2","title":"Task 2","state":"In Progress","priority":2}]'
        else
          echo '[]'
        fi
        ;;
      show)
        # beads issue show <identifier>
        identifier="$3"
        if [[ "$identifier" == "BEADS-1" ]]; then
          echo '{"id":"1","identifier":"BEADS-1","title":"Task 1","description":"Description 1","state":"Todo","priority":1,"labels":["bug"],"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-02T00:00:00Z"}'
        elif [[ "$identifier" == "BEADS-2" ]]; then
          echo '{"id":"2","identifier":"BEADS-2","title":"Task 2","state":"In Progress"}'
        else
          echo "Error: issue not found"
          exit 1
        fi
        ;;
      create)
        # beads issue create --title <title> --description <desc>
        title=""
        desc=""
        for arg in "$@"; do
          if [[ "$prev_arg" == "--title" ]]; then
            title="$arg"
          fi
          if [[ "$prev_arg" == "--description" ]]; then
            desc="$arg"
          fi
          prev_arg="$arg"
        done
        echo '{"id":"new-1","identifier":"BEADS-NEW","title":"'"$title"'","description":"'"$desc"'","state":"Todo"}'
        ;;
      update)
        # beads issue update <identifier> --stage <stage>
        echo '{"success":true}'
        ;;
      comment)
        # beads issue comment <identifier> --body <body>
        echo '{"success":true}'
        ;;
      comments)
        # beads issue comments <identifier>
        identifier="$3"
        if [[ "$identifier" == "BEADS-1" ]]; then
          echo '[{"role":"user","content":"想要添加用户登录功能","timestamp":"2024-01-01T10:00:00Z"},{"role":"assistant","content":"请问登录方式是邮箱还是手机号？","timestamp":"2024-01-01T10:01:00Z"},{"role":"user","content":"邮箱","timestamp":"2024-01-01T10:02:00Z"}]'
        elif [[ "$identifier" == "BEADS-2" ]]; then
          echo '[]'
        else
          echo "Error: issue not found"
          exit 1
        fi
        ;;
      *)
        echo "Unknown issue command: $2"
        exit 1
        ;;
    esac
    ;;
  *)
    echo "Unknown command: $1"
    exit 1
    ;;
esac
`

	err := os.WriteFile(cliPath, []byte(script), 0755)
	require.NoError(t, err)

	return cliPath, func() {
		// tmpDir is automatically cleaned up by t.TempDir()
	}
}

// TestBeadsClient_CheckAvailability_Success 测试 CLI 可用性检查成功
func TestBeadsClient_CheckAvailability_Success(t *testing.T) {
	cliPath, cleanup := setupMockBeadsCLI(t)
	defer cleanup()

	client := NewBeadsClientWithPath(cliPath)
	err := client.CheckAvailability()
	assert.NoError(t, err)
}

// TestBeadsClient_CheckAvailability_CLINotAvailable 测试 CLI 不可用
func TestBeadsClient_CheckAvailability_CLINotAvailable(t *testing.T) {
	client := NewBeadsClientWithPath("/nonexistent/beads")
	err := client.CheckAvailability()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tracker.unavailable")
	assert.Contains(t, err.Error(), "Beads CLI 不可用")
}

// TestBeadsClient_FetchCandidateIssues 测试获取候选问题
func TestBeadsClient_FetchCandidateIssues(t *testing.T) {
	cliPath, cleanup := setupMockBeadsCLI(t)
	defer cleanup()

	client := NewBeadsClientWithPath(cliPath)
	ctx := context.Background()

	issues, err := client.FetchCandidateIssues(ctx, []string{"Todo"})
	require.NoError(t, err)
	require.Len(t, issues, 1)

	assert.Equal(t, "1", issues[0].ID)
	assert.Equal(t, "BEADS-1", issues[0].Identifier)
	assert.Equal(t, "Task 1", issues[0].Title)
	assert.Equal(t, "Todo", issues[0].State)
}

// TestBeadsClient_FetchIssuesByStates 测试按状态获取问题
func TestBeadsClient_FetchIssuesByStates(t *testing.T) {
	cliPath, cleanup := setupMockBeadsCLI(t)
	defer cleanup()

	client := NewBeadsClientWithPath(cliPath)
	ctx := context.Background()

	issues, err := client.FetchIssuesByStates(ctx, []string{"Todo", "In Progress"})
	require.NoError(t, err)
	require.Len(t, issues, 2)

	// 验证两种状态的问题都被获取
	stateSet := make(map[string]bool)
	for _, issue := range issues {
		stateSet[issue.State] = true
	}
	assert.True(t, stateSet["Todo"])
	assert.True(t, stateSet["In Progress"])
}

// TestBeadsClient_FetchIssueStatesByIDs 测试按 ID 获取问题状态
func TestBeadsClient_FetchIssueStatesByIDs(t *testing.T) {
	cliPath, cleanup := setupMockBeadsCLI(t)
	defer cleanup()

	client := NewBeadsClientWithPath(cliPath)
	ctx := context.Background()

	issues, err := client.FetchIssueStatesByIDs(ctx, []string{"BEADS-1"})
	require.NoError(t, err)
	require.Len(t, issues, 1)

	assert.Equal(t, "BEADS-1", issues[0].Identifier)
	assert.Equal(t, "Todo", issues[0].State)
}

// TestBeadsClient_FetchIssueStatesByIDs_Empty 测试空 ID 列表
func TestBeadsClient_FetchIssueStatesByIDs_Empty(t *testing.T) {
	client := NewBeadsClient()
	ctx := context.Background()

	issues, err := client.FetchIssueStatesByIDs(ctx, []string{})
	require.NoError(t, err)
	assert.Nil(t, issues)
}

// TestBeadsClient_CreateTask 测试创建任务
func TestBeadsClient_CreateTask(t *testing.T) {
	cliPath, cleanup := setupMockBeadsCLI(t)
	defer cleanup()

	client := NewBeadsClientWithPath(cliPath)
	ctx := context.Background()

	issue, err := client.CreateTask(ctx, "New Task", "New Description")
	require.NoError(t, err)
	require.NotNil(t, issue)

	assert.Equal(t, "new-1", issue.ID)
	assert.Equal(t, "BEADS-NEW", issue.Identifier)
	assert.Equal(t, "New Task", issue.Title)
	assert.Equal(t, "Todo", issue.State)
	assert.NotNil(t, issue.Description)
	assert.Equal(t, "New Description", *issue.Description)
}

// TestBeadsClient_UpdateStage 测试更新阶段状态
func TestBeadsClient_UpdateStage(t *testing.T) {
	cliPath, cleanup := setupMockBeadsCLI(t)
	defer cleanup()

	client := NewBeadsClientWithPath(cliPath)
	ctx := context.Background()

	stage := domain.StageState{
		Name:       "clarification",
		Status:     "in_progress",
		StartedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Round:      1,
	}

	err := client.UpdateStage(ctx, "BEADS-1", stage)
	assert.NoError(t, err)
}

// TestBeadsClient_AppendConversation 测试追加对话记录
func TestBeadsClient_AppendConversation(t *testing.T) {
	cliPath, cleanup := setupMockBeadsCLI(t)
	defer cleanup()

	client := NewBeadsClientWithPath(cliPath)
	ctx := context.Background()

	turn := domain.ConversationTurn{
		Role:      "user",
		Content:   "Hello, this is a test message",
		Timestamp: time.Now(),
	}

	err := client.AppendConversation(ctx, "BEADS-1", turn)
	assert.NoError(t, err)
}

// TestBeadsClient_ListTasksByState 测试按状态获取任务列表
func TestBeadsClient_ListTasksByState(t *testing.T) {
	cliPath, cleanup := setupMockBeadsCLI(t)
	defer cleanup()

	client := NewBeadsClientWithPath(cliPath)
	ctx := context.Background()

	issues, err := client.ListTasksByState(ctx, []string{"Todo"})
	require.NoError(t, err)
	require.Len(t, issues, 1)

	assert.Equal(t, "BEADS-1", issues[0].Identifier)
}

// TestBeadsClient_GetTask 测试获取单个任务
func TestBeadsClient_GetTask(t *testing.T) {
	cliPath, cleanup := setupMockBeadsCLI(t)
	defer cleanup()

	client := NewBeadsClientWithPath(cliPath)
	ctx := context.Background()

	issue, err := client.GetTask(ctx, "BEADS-1")
	require.NoError(t, err)
	require.NotNil(t, issue)

	assert.Equal(t, "BEADS-1", issue.Identifier)
	assert.Equal(t, "Task 1", issue.Title)
	assert.NotNil(t, issue.Description)
	assert.Equal(t, "Description 1", *issue.Description)
}

// TestBeadsClient_Timeout 测试超时控制
func TestBeadsClient_Timeout(t *testing.T) {
	// 创建一个会延迟响应的模拟 CLI
	tmpDir := t.TempDir()
	cliPath := filepath.Join(tmpDir, "slow_beads")

	script := `#!/bin/bash
sleep 10
echo "done"
`
	err := os.WriteFile(cliPath, []byte(script), 0755)
	require.NoError(t, err)

	client := NewBeadsClientWithPath(cliPath)
	client.SetTimeout(1 * time.Second) // 设置 1 秒超时

	ctx := context.Background()
	_, err = client.ListTasksByState(ctx, []string{"Todo"})
	assert.Error(t, err)
	// 超时可能表现为 signal: killed 或 context deadline exceeded
	assert.True(t, strings.Contains(err.Error(), "killed") ||
		strings.Contains(err.Error(), "deadline exceeded") ||
		strings.Contains(err.Error(), "timeout"),
		"Expected timeout-related error, got: %s", err.Error())
}

// TestBeadsClient_ParseIssueList 测试解析任务列表
func TestBeadsClient_ParseIssueList(t *testing.T) {
	client := NewBeadsClient()

	// 测试空输出
	issues, err := client.parseIssueList([]byte{})
	require.NoError(t, err)
	assert.Nil(t, issues)

	// 测试有效 JSON
	jsonData := `[{"id":"1","identifier":"BEADS-1","title":"Task 1","state":"Todo","priority":1}]`
	issues, err = client.parseIssueList([]byte(jsonData))
	require.NoError(t, err)
	require.Len(t, issues, 1)
	assert.Equal(t, "BEADS-1", issues[0].Identifier)

	// 测试无效 JSON
	_, err = client.parseIssueList([]byte("invalid json"))
	assert.Error(t, err)
}

// TestBeadsIssue_ToDomain 测试转换为领域模型
func TestBeadsIssue_ToDomain(t *testing.T) {
	bi := beadsIssue{
		ID:          "123",
		Identifier:  "BEADS-123",
		Title:       "Test Issue",
		Description: "Test Description",
		State:       "In Progress",
		Priority:    1,
		Labels:      []string{"bug", "urgent"},
		BranchName:  "feature/test",
		URL:         "https://example.com/beads/BEADS-123",
		CreatedAt:   "2024-01-01T00:00:00Z",
		UpdatedAt:   "2024-01-02T12:00:00Z",
	}

	issue := bi.toDomain()

	assert.Equal(t, "123", issue.ID)
	assert.Equal(t, "BEADS-123", issue.Identifier)
	assert.Equal(t, "Test Issue", issue.Title)
	assert.Equal(t, "In Progress", issue.State)
	assert.NotNil(t, issue.Description)
	assert.Equal(t, "Test Description", *issue.Description)
	assert.NotNil(t, issue.Priority)
	assert.Equal(t, 1, *issue.Priority)
	assert.Equal(t, []string{"bug", "urgent"}, issue.Labels)
	assert.NotNil(t, issue.BranchName)
	assert.Equal(t, "feature/test", *issue.BranchName)
	assert.NotNil(t, issue.URL)
	assert.Equal(t, "https://example.com/beads/BEADS-123", *issue.URL)
	assert.NotNil(t, issue.CreatedAt)
	assert.NotNil(t, issue.UpdatedAt)
}

// TestBeadsIssue_ToDomain_EmptyFields 测试空字段处理
func TestBeadsIssue_ToDomain_EmptyFields(t *testing.T) {
	bi := beadsIssue{
		ID:         "1",
		Identifier: "BEADS-1",
		Title:      "Minimal Issue",
		State:      "Todo",
	}

	issue := bi.toDomain()

	assert.Equal(t, "1", issue.ID)
	assert.Equal(t, "BEADS-1", issue.Identifier)
	assert.Equal(t, "Minimal Issue", issue.Title)
	assert.Equal(t, "Todo", issue.State)
	assert.Nil(t, issue.Description)
	assert.Nil(t, issue.Priority)
	assert.Nil(t, issue.BranchName)
	assert.Nil(t, issue.URL)
	assert.Nil(t, issue.CreatedAt)
	assert.Nil(t, issue.UpdatedAt)
	assert.Empty(t, issue.Labels)
}

// TestNewBeadsClient 测试创建客户端
func TestNewBeadsClient(t *testing.T) {
	client := NewBeadsClient()
	assert.NotNil(t, client)
	assert.Equal(t, beadsCLIName, client.cliPath)
	assert.Equal(t, beadsDefaultTimeout, client.timeout)
}

// TestNewBeadsClientWithPath 测试带路径创建客户端
func TestNewBeadsClientWithPath(t *testing.T) {
	client := NewBeadsClientWithPath("/custom/path/to/beads")
	assert.NotNil(t, client)
	assert.Equal(t, "/custom/path/to/beads", client.cliPath)
	assert.Equal(t, beadsDefaultTimeout, client.timeout)
}

// TestBeadsClient_SetTimeout 测试设置超时
func TestBeadsClient_SetTimeout(t *testing.T) {
	client := NewBeadsClient()
	client.SetTimeout(10 * time.Second)
	assert.Equal(t, 10*time.Second, client.timeout)
}

// TestBeadsClient_Integration 测试集成场景
func TestBeadsClient_Integration(t *testing.T) {
	if os.Getenv("BEADS_INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test; set BEADS_INTEGRATION_TEST=true to run")
	}

	// 这个测试需要真实的 Beads CLI 可用
	client := NewBeadsClient()

	// 检查可用性
	err := client.CheckAvailability()
	if err != nil {
		t.Fatalf("Beads CLI not available: %v", err)
	}

	ctx := context.Background()

	// 测试获取任务列表
	issues, err := client.ListTasksByState(ctx, []string{"Todo"})
	require.NoError(t, err)
	t.Logf("Found %d issues in Todo state", len(issues))
}

// TestTrackerInterfaceCompliance 测试 BeadsClient 实现 Tracker 接口
func TestTrackerInterfaceCompliance(t *testing.T) {
	// 编译时检查接口实现
	var _ Tracker = (*BeadsClient)(nil)
	var _ Tracker = (*MockClient)(nil)
	var _ Tracker = (*LinearClient)(nil)
	var _ Tracker = (*GitHubClient)(nil)
}

// TestMockClient_NewMethods 测试 MockClient 新方法
func TestMockClient_NewMethods(t *testing.T) {
	client := NewMockClient([]config.MockIssueConfig{
		{
			ID:         "1",
			Identifier: "MOCK-1",
			Title:      "Test Issue",
			State:      "Todo",
		},
	})

	ctx := context.Background()

	// CheckAvailability
	err := client.CheckAvailability()
	assert.NoError(t, err)

	// CreateTask
	issue, err := client.CreateTask(ctx, "New Task", "Description")
	require.NoError(t, err)
	assert.Equal(t, "New Task", issue.Title)
	assert.Equal(t, "Todo", issue.State)

	// UpdateStage
	stage := domain.StageState{
		Name:      "clarification",
		Status:    "in_progress",
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = client.UpdateStage(ctx, "MOCK-1", stage)
	assert.NoError(t, err)

	// UpdateStage - issue not found
	err = client.UpdateStage(ctx, "NONEXISTENT", stage)
	assert.Error(t, err)

	// AppendConversation
	turn := domain.ConversationTurn{
		Role:      "user",
		Content:   "Test message",
		Timestamp: time.Now(),
	}
	err = client.AppendConversation(ctx, "MOCK-1", turn)
	assert.NoError(t, err)

	// AppendConversation - issue not found
	err = client.AppendConversation(ctx, "NONEXISTENT", turn)
	assert.Error(t, err)

	// ListTasksByState
	issues, err := client.ListTasksByState(ctx, []string{"Todo"})
	require.NoError(t, err)
	assert.Len(t, issues, 2) // 原有一个 + 新创建一个
}

// TestBeadsClient_ListTasksByState_EmptyStates 测试空状态列表
func TestBeadsClient_ListTasksByState_EmptyStates(t *testing.T) {
	cliPath, cleanup := setupMockBeadsCLI(t)
	defer cleanup()

	client := NewBeadsClientWithPath(cliPath)
	ctx := context.Background()

	issues, err := client.ListTasksByState(ctx, []string{})
	require.NoError(t, err)
	assert.Empty(t, issues)
}

// TestBeadsClient_ListTasksByState_Error 测试错误处理
func TestBeadsClient_ListTasksByState_Error(t *testing.T) {
	// 创建一个会返回错误的模拟 CLI
	tmpDir := t.TempDir()
	cliPath := filepath.Join(tmpDir, "error_beads")

	script := `#!/bin/bash
echo "Error: something went wrong"
exit 1
`
	err := os.WriteFile(cliPath, []byte(script), 0755)
	require.NoError(t, err)

	client := NewBeadsClientWithPath(cliPath)
	ctx := context.Background()

	_, err = client.ListTasksByState(ctx, []string{"Todo"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list tasks failed")
}

// TestBeadsClient_CreateTask_Error 测试创建任务错误
func TestBeadsClient_CreateTask_Error(t *testing.T) {
	// 创建一个会返回错误的模拟 CLI
	tmpDir := t.TempDir()
	cliPath := filepath.Join(tmpDir, "error_beads")

	script := `#!/bin/bash
case "$1" in
  --version)
    echo "beads v1.0.0"
    ;;
  issue)
    if [[ "$2" == "create" ]]; then
      echo "Error: failed to create"
      exit 1
    fi
    ;;
esac
`
	err := os.WriteFile(cliPath, []byte(script), 0755)
	require.NoError(t, err)

	client := NewBeadsClientWithPath(cliPath)
	ctx := context.Background()

	_, err = client.CreateTask(ctx, "Test", "Description")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create task failed")
}

// TestBeadsClient_CreateTask_InvalidJSON 测试创建任务返回无效JSON
func TestBeadsClient_CreateTask_InvalidJSON(t *testing.T) {
	// 创建一个会返回无效JSON的模拟 CLI
	tmpDir := t.TempDir()
	cliPath := filepath.Join(tmpDir, "invalid_json_beads")

	script := `#!/bin/bash
case "$1" in
  --version)
    echo "beads v1.0.0"
    ;;
  issue)
    if [[ "$2" == "create" ]]; then
      echo "not valid json"
    fi
    ;;
esac
`
	err := os.WriteFile(cliPath, []byte(script), 0755)
	require.NoError(t, err)

	client := NewBeadsClientWithPath(cliPath)
	ctx := context.Background()

	_, err = client.CreateTask(ctx, "Test", "Description")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse create result")
}

// TestBeadsClient_GetTask_Error 测试获取任务错误
func TestBeadsClient_GetTask_Error(t *testing.T) {
	// 创建一个会返回错误的模拟 CLI
	tmpDir := t.TempDir()
	cliPath := filepath.Join(tmpDir, "error_beads")

	script := `#!/bin/bash
case "$1" in
  --version)
    echo "beads v1.0.0"
    ;;
  issue)
    if [[ "$2" == "show" ]]; then
      echo "Error: issue not found"
      exit 1
    fi
    ;;
esac
`
	err := os.WriteFile(cliPath, []byte(script), 0755)
	require.NoError(t, err)

	client := NewBeadsClientWithPath(cliPath)
	ctx := context.Background()

	_, err = client.GetTask(ctx, "NONEXISTENT")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get task failed")
}

// TestBeadsClient_UpdateStage_Error 测试更新阶段错误
func TestBeadsClient_UpdateStage_Error(t *testing.T) {
	// 创建一个会返回错误的模拟 CLI
	tmpDir := t.TempDir()
	cliPath := filepath.Join(tmpDir, "error_beads")

	script := `#!/bin/bash
case "$1" in
  --version)
    echo "beads v1.0.0"
    ;;
  issue)
    if [[ "$2" == "update" ]]; then
      echo "Error: update failed"
      exit 1
    fi
    ;;
esac
`
	err := os.WriteFile(cliPath, []byte(script), 0755)
	require.NoError(t, err)

	client := NewBeadsClientWithPath(cliPath)
	ctx := context.Background()

	stage := domain.StageState{
		Name:   "test",
		Status: "pending",
	}

	err = client.UpdateStage(ctx, "BEADS-1", stage)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update stage failed")
}

// TestBeadsClient_AppendConversation_Error 测试追加对话错误
func TestBeadsClient_AppendConversation_Error(t *testing.T) {
	// 创建一个会返回错误的模拟 CLI
	tmpDir := t.TempDir()
	cliPath := filepath.Join(tmpDir, "error_beads")

	script := `#!/bin/bash
case "$1" in
  --version)
    echo "beads v1.0.0"
    ;;
  issue)
    if [[ "$2" == "comment" ]]; then
      echo "Error: comment failed"
      exit 1
    fi
    ;;
esac
`
	err := os.WriteFile(cliPath, []byte(script), 0755)
	require.NoError(t, err)

	client := NewBeadsClientWithPath(cliPath)
	ctx := context.Background()

	turn := domain.ConversationTurn{
		Role:    "user",
		Content: "test",
	}

	err = client.AppendConversation(ctx, "BEADS-1", turn)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "append conversation failed")
}

// TestBeadsClient_GetConversationHistory 测试获取对话历史
func TestBeadsClient_GetConversationHistory(t *testing.T) {
	cliPath, cleanup := setupMockBeadsCLI(t)
	defer cleanup()

	client := NewBeadsClientWithPath(cliPath)
	ctx := context.Background()

	history, err := client.GetConversationHistory(ctx, "BEADS-1")
	require.NoError(t, err)
	require.Len(t, history, 3)

	assert.Equal(t, "user", history[0].Role)
	assert.Equal(t, "想要添加用户登录功能", history[0].Content)
	assert.Equal(t, "assistant", history[1].Role)
	assert.Equal(t, "请问登录方式是邮箱还是手机号？", history[1].Content)
	assert.Equal(t, "user", history[2].Role)
	assert.Equal(t, "邮箱", history[2].Content)
}

// TestBeadsClient_GetConversationHistory_Empty 测试获取空对话历史
func TestBeadsClient_GetConversationHistory_Empty(t *testing.T) {
	cliPath, cleanup := setupMockBeadsCLI(t)
	defer cleanup()

	client := NewBeadsClientWithPath(cliPath)
	ctx := context.Background()

	history, err := client.GetConversationHistory(ctx, "BEADS-2")
	require.NoError(t, err)
	assert.Empty(t, history)
}

// TestBeadsClient_GetConversationHistory_Error 测试获取对话历史错误
func TestBeadsClient_GetConversationHistory_Error(t *testing.T) {
	// 创建一个会返回错误的模拟 CLI
	tmpDir := t.TempDir()
	cliPath := filepath.Join(tmpDir, "error_beads")

	script := `#!/bin/bash
case "$1" in
  --version)
    echo "beads v1.0.0"
    ;;
  issue)
    if [[ "$2" == "comments" ]]; then
      echo "Error: comments failed"
      exit 1
    fi
    ;;
esac
`
	err := os.WriteFile(cliPath, []byte(script), 0755)
	require.NoError(t, err)

	client := NewBeadsClientWithPath(cliPath)
	ctx := context.Background()

	_, err = client.GetConversationHistory(ctx, "BEADS-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get conversation history failed")
}

// TestBeadsClient_GetConversationHistory_NotFound 测试获取不存在任务的对话历史
func TestBeadsClient_GetConversationHistory_NotFound(t *testing.T) {
	cliPath, cleanup := setupMockBeadsCLI(t)
	defer cleanup()

	client := NewBeadsClientWithPath(cliPath)
	ctx := context.Background()

	_, err := client.GetConversationHistory(ctx, "BEADS-NONEXISTENT")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get conversation history failed")
}

// TestBeadsClient_parseConversationHistory 测试解析对话历史
func TestBeadsClient_parseConversationHistory(t *testing.T) {
	client := NewBeadsClient()

	// 测试空输出
	turns, err := client.parseConversationHistory([]byte{})
	require.NoError(t, err)
	assert.Nil(t, turns)

	// 测试有效 JSON
	jsonData := `[{"role":"user","content":"Hello","timestamp":"2024-01-01T00:00:00Z"},{"role":"assistant","content":"Hi","timestamp":"2024-01-01T00:01:00Z"}]`
	turns, err = client.parseConversationHistory([]byte(jsonData))
	require.NoError(t, err)
	require.Len(t, turns, 2)
	assert.Equal(t, "user", turns[0].Role)
	assert.Equal(t, "Hello", turns[0].Content)
	assert.Equal(t, "assistant", turns[1].Role)
	assert.Equal(t, "Hi", turns[1].Content)

	// 测试无效 JSON
	_, err = client.parseConversationHistory([]byte("invalid json"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal conversation history")
}

// TestBeadsClient_FetchIssueStatesByIDs_Error 测试按ID获取问题时的错误处理
func TestBeadsClient_FetchIssueStatesByIDs_Error(t *testing.T) {
	// 创建一个对特定ID返回错误的模拟 CLI
	tmpDir := t.TempDir()
	cliPath := filepath.Join(tmpDir, "mixed_beads")

	script := `#!/bin/bash
case "$1" in
  --version)
    echo "beads v1.0.0"
    ;;
  issue)
    case "$2" in
      show)
        identifier="$3"
        if [[ "$identifier" == "BEADS-1" ]]; then
          echo '{"id":"1","identifier":"BEADS-1","title":"Task 1","state":"Todo"}'
        else
          echo "Error: issue not found"
          exit 1
        fi
        ;;
    esac
    ;;
esac
`
	err := os.WriteFile(cliPath, []byte(script), 0755)
	require.NoError(t, err)

	client := NewBeadsClientWithPath(cliPath)
	ctx := context.Background()

	// 请求包含一个有效和一个无效的ID
	issues, err := client.FetchIssueStatesByIDs(ctx, []string{"BEADS-1", "BEADS-INVALID"})
	require.NoError(t, err)
	// 只有 BEADS-1 成功返回
	require.Len(t, issues, 1)
	assert.Equal(t, "BEADS-1", issues[0].Identifier)
}

// TestBeadsClient_CheckAvailability_EmptyOutput 测试空版本输出
func TestBeadsClient_CheckAvailability_EmptyOutput(t *testing.T) {
	// 创建一个返回空输出的模拟 CLI
	tmpDir := t.TempDir()
	cliPath := filepath.Join(tmpDir, "empty_beads")

	script := `#!/bin/bash
# 无输出
`
	err := os.WriteFile(cliPath, []byte(script), 0755)
	require.NoError(t, err)

	client := NewBeadsClientWithPath(cliPath)
	err = client.CheckAvailability()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "无版本输出")
}

// TestBeadsIssue_ToDomain_WithPriority 测试带优先级的转换
func TestBeadsIssue_ToDomain_WithPriority(t *testing.T) {
	bi := beadsIssue{
		ID:         "1",
		Identifier: "BEADS-1",
		Title:      "High Priority Task",
		State:      "Todo",
		Priority:   5,
	}

	issue := bi.toDomain()

	assert.NotNil(t, issue.Priority)
	assert.Equal(t, 5, *issue.Priority)
}

// TestBeadsIssue_ToDomain_ZeroPriority 测试零优先级（不设置）
func TestBeadsIssue_ToDomain_ZeroPriority(t *testing.T) {
	bi := beadsIssue{
		ID:         "1",
		Identifier: "BEADS-1",
		Title:      "Zero Priority Task",
		State:      "Todo",
		Priority:   0,
	}

	issue := bi.toDomain()

	// 零优先级不设置 priority 字段
	assert.Nil(t, issue.Priority)
}

// TestBeadsClient_FetchCandidateIssues_AllStates 测试获取所有活跃状态的问题
func TestBeadsClient_FetchCandidateIssues_AllStates(t *testing.T) {
	cliPath, cleanup := setupMockBeadsCLI(t)
	defer cleanup()

	client := NewBeadsClientWithPath(cliPath)
	ctx := context.Background()

	// 获取 Todo 和 In Progress 状态的问题
	issues, err := client.FetchCandidateIssues(ctx, []string{"Todo", "In Progress"})
	require.NoError(t, err)
	require.Len(t, issues, 2)

	// 验证包含两种状态
	assert.Contains(t, []string{issues[0].State, issues[1].State}, "Todo")
	assert.Contains(t, []string{issues[0].State, issues[1].State}, "In Progress")
}

// TestBeadsClient_FetchCandidateIssues_NoMatching 测试无匹配状态的问题
func TestBeadsClient_FetchCandidateIssues_NoMatching(t *testing.T) {
	cliPath, cleanup := setupMockBeadsCLI(t)
	defer cleanup()

	client := NewBeadsClientWithPath(cliPath)
	ctx := context.Background()

	// 获取不存在状态的问题
	issues, err := client.FetchCandidateIssues(ctx, []string{"Nonexistent"})
	require.NoError(t, err)
	assert.Empty(t, issues)
}

// TestBeadsClient_runCommand 测试 runCommand 内部方法
func TestBeadsClient_runCommand(t *testing.T) {
	cliPath, cleanup := setupMockBeadsCLI(t)
	defer cleanup()

	client := NewBeadsClientWithPath(cliPath)
	ctx := context.Background()

	// 测试正常命令
	output, err := client.runCommand(ctx, "--version")
	require.NoError(t, err)
	assert.Contains(t, string(output), "beads")
}