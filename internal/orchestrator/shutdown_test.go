// Package orchestrator_test 测试优雅关闭功能
package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/domain"
)

// 辅助函数：创建测试用的配置
func createShutdownTestConfig() *config.Config {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	cfg.Agent.MaxConcurrentAgents = 2
	cfg.Workspace.Root = filepath.Join(os.TempDir(), "symphony_shutdown_test")
	return cfg
}

// TestDefaultShutdownConfig 测试默认关闭配置
func TestDefaultShutdownConfig(t *testing.T) {
	cfg := DefaultShutdownConfig()
	assert.NotNil(t, cfg)
	assert.Equal(t, 30*time.Second, cfg.Timeout)
	assert.Equal(t, ".symphony/shutdown_state.json", cfg.StateSavePath)
}

// TestNewShutdownManager 测试创建关闭管理器
func TestNewShutdownManager(t *testing.T) {
	cfg := createShutdownTestConfig()
	orch := New(cfg, "test prompt")

	sm := NewShutdownManager(orch, orch.trackerClient, cfg)

	assert.NotNil(t, sm)
	assert.NotNil(t, sm.orch)
	assert.Equal(t, 30*time.Second, sm.shutdownTimeout)
	assert.NotNil(t, sm.activeCmds)
}

// TestShutdownManagerIsShuttingDown 测试关闭状态检查
func TestShutdownManagerIsShuttingDown(t *testing.T) {
	cfg := createShutdownTestConfig()
	orch := New(cfg, "test prompt")
	sm := NewShutdownManager(orch, orch.trackerClient, cfg)

	// 初始状态不应在关闭中
	assert.False(t, sm.IsShuttingDown())
}

// TestShutdownManagerRegisterUnregisterProcess 测试进程注册和移除
func TestShutdownManagerRegisterUnregisterProcess(t *testing.T) {
	cfg := createShutdownTestConfig()
	orch := New(cfg, "test prompt")
	sm := NewShutdownManager(orch, orch.trackerClient, cfg)

	// 注册进程（使用 nil 模拟）
	sm.RegisterAgentProcess("issue-1", nil)
	sm.RegisterAgentProcess("issue-2", nil)

	assert.Equal(t, 2, sm.GetActiveCmdsCount())

	// 移除进程
	sm.UnregisterAgentProcess("issue-1")
	assert.Equal(t, 1, sm.GetActiveCmdsCount())

	sm.UnregisterAgentProcess("issue-2")
	assert.Equal(t, 0, sm.GetActiveCmdsCount())
}

// TestShutdownManagerWaitForCompletion 测试等待任务完成
func TestShutdownManagerWaitForCompletion(t *testing.T) {
	cfg := createShutdownTestConfig()
	orch := New(cfg, "test prompt")
	sm := NewShutdownManager(orch, orch.trackerClient, cfg)

	// 没有运行任务时应该立即返回
	ctx := context.Background()
	done := sm.WaitForCompletion(ctx)
	assert.True(t, done)

	// 添加一个运行中的任务
	issue := &domain.Issue{
		ID:         "test-1",
		Identifier: "TEST-1",
		Title:      "Test Issue",
		State:      "In Progress",
	}
	entry := &domain.RunningEntry{
		Issue:      issue,
		Identifier: "TEST-1",
		StartedAt:  time.Now(),
		TurnCount:  0,
	}
	orch.mu.Lock()
	orch.state.Running["test-1"] = entry
	orch.mu.Unlock()

	// 创建超时上下文（短超时）
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// 等待应该超时（因为任务还在运行）
	done = sm.WaitForCompletion(timeoutCtx)
	assert.False(t, done)
}

// TestShutdownManagerSaveTaskStates 测试保存任务状态
func TestShutdownManagerSaveTaskStates(t *testing.T) {
	cfg := createShutdownTestConfig()
	orch := New(cfg, "test prompt")

	// 添加运行中的任务
	issue := &domain.Issue{
		ID:         "test-1",
		Identifier: "TEST-1",
		Title:      "Test Issue",
		State:      "In Progress",
	}
	now := time.Now()
	entry := &domain.RunningEntry{
		Issue:      issue,
		Identifier: "TEST-1",
		StartedAt:  now,
		TurnCount:  5,
	}
	orch.mu.Lock()
	orch.state.Running["test-1"] = entry

	// 添加重试任务
	retryEntry := &domain.RetryEntry{
		IssueID:    "test-2",
		Identifier: "TEST-2",
		Attempt:    2,
		DueAtMs:    time.Now().Add(10 * time.Second).UnixMilli(),
	}
	errMsg := "connection timeout"
	retryEntry.Error = &errMsg
	orch.state.RetryAttempts["test-2"] = retryEntry
	orch.mu.Unlock()

	sm := NewShutdownManager(orch, orch.trackerClient, cfg)

	// 保存状态
	err := sm.SaveTaskStates()
	require.NoError(t, err)

	// 验证文件已创建
	savePath := filepath.Join(cfg.Workspace.Root, ".symphony", "shutdown_state.json")
	assert.FileExists(t, savePath)

	// 加载状态验证
	loadedState, err := sm.LoadSavedStates()
	require.NoError(t, err)
	require.NotNil(t, loadedState)

	assert.Len(t, loadedState.RunningTasks, 1)
	assert.Len(t, loadedState.RetryTasks, 1)

	// 验证运行任务内容
	runningTask := loadedState.RunningTasks[0]
	assert.Equal(t, "test-1", runningTask.IssueID)
	assert.Equal(t, "TEST-1", runningTask.IssueIdentifier)
	assert.Equal(t, "Test Issue", runningTask.IssueTitle)
	assert.Equal(t, 5, runningTask.TurnCount)

	// 验证重试任务内容
	retryTask := loadedState.RetryTasks[0]
	assert.Equal(t, "test-2", retryTask.IssueID)
	assert.Equal(t, "TEST-2", retryTask.IssueIdentifier)
	assert.Equal(t, "connection timeout", retryTask.LastError)

	// 清理测试文件
	os.Remove(savePath)
	os.RemoveAll(cfg.Workspace.Root)
}

// TestShutdownManagerLoadSavedStatesNotFound 测试加载不存在的状态文件
func TestShutdownManagerLoadSavedStatesNotFound(t *testing.T) {
	cfg := createShutdownTestConfig()
	orch := New(cfg, "test prompt")
	sm := NewShutdownManager(orch, orch.trackerClient, cfg)

	// 文件不存在时应返回 nil
	loadedState, err := sm.LoadSavedStates()
	assert.NoError(t, err)
	assert.Nil(t, loadedState)

	// 清理
	os.RemoveAll(cfg.Workspace.Root)
}

// TestShutdownManagerClearSavedStates 测试清除状态文件
func TestShutdownManagerClearSavedStates(t *testing.T) {
	cfg := createShutdownTestConfig()
	orch := New(cfg, "test prompt")
	sm := NewShutdownManager(orch, orch.trackerClient, cfg)

	// 创建状态文件
	savePath := filepath.Join(cfg.Workspace.Root, ".symphony", "shutdown_state.json")
	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(savePath, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	assert.FileExists(t, savePath)

	// 清除
	err := sm.ClearSavedStates()
	assert.NoError(t, err)

	// 文件应不存在
	assert.NoFileExists(t, savePath)

	// 清理
	os.RemoveAll(cfg.Workspace.Root)
}

// TestShutdownStateStruct 测试关闭状态结构
func TestShutdownStateStruct(t *testing.T) {
	now := time.Now()
	state := &ShutdownState{
		SavedAt: now,
		RunningTasks: []SavedTask{
			{
				IssueID:         "test-1",
				IssueIdentifier: "TEST-1",
				IssueTitle:      "Test Issue",
				IssueState:      "In Progress",
				StartedAt:       now,
				TurnCount:       5,
			},
		},
		RetryTasks: []SavedTask{
			{
				IssueID:         "test-2",
				IssueIdentifier: "TEST-2",
				StartedAt:       now,
				LastError:       "timeout",
			},
		},
	}

	assert.Equal(t, now, state.SavedAt)
	assert.Len(t, state.RunningTasks, 1)
	assert.Len(t, state.RetryTasks, 1)

	task := state.RunningTasks[0]
	assert.Equal(t, "test-1", task.IssueID)
	assert.Equal(t, "TEST-1", task.IssueIdentifier)
	assert.Equal(t, "Test Issue", task.IssueTitle)
	assert.Equal(t, 5, task.TurnCount)
}

// TestSavedTaskStruct 测试保存任务结构
func TestSavedTaskStruct(t *testing.T) {
	now := time.Now()
	attempt := 3
	session := &domain.LiveSession{
		SessionID:        "session-1",
		ThreadID:         "thread-1",
		TurnID:           "turn-1",
		CodexInputTokens: 100,
		CodexOutputTokens: 50,
	}

	task := SavedTask{
		IssueID:         "test-1",
		IssueIdentifier: "TEST-1",
		IssueTitle:      "Test Issue",
		IssueState:      "In Progress",
		StartedAt:       now,
		TurnCount:       5,
		RetryAttempt:    &attempt,
		LastError:       "connection error",
		Session:         session,
	}

	assert.Equal(t, "test-1", task.IssueID)
	assert.Equal(t, "TEST-1", task.IssueIdentifier)
	assert.Equal(t, 5, task.TurnCount)
	assert.Equal(t, 3, *task.RetryAttempt)
	assert.Equal(t, "connection error", task.LastError)
	assert.NotNil(t, task.Session)
	assert.Equal(t, "session-1", task.Session.SessionID)
}

// TestOrchestratorShutdown 测试编排器关闭方法
func TestOrchestratorShutdown(t *testing.T) {
	cfg := createShutdownTestConfig()
	orch := New(cfg, "test prompt")

	// 初始状态不应在关闭中
	assert.False(t, orch.IsShuttingDown())

	// 创建关闭上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 执行关闭
	err := orch.Shutdown(ctx)
	assert.NoError(t, err)

	// 关闭后应标记为关闭状态
	assert.True(t, orch.IsShuttingDown())

	// 清理
	os.RemoveAll(cfg.Workspace.Root)
}

// TestShutdownManagerTerminateAgents 测试终止 Agent 进程
func TestShutdownManagerTerminateAgents(t *testing.T) {
	cfg := createShutdownTestConfig()
	orch := New(cfg, "test prompt")
	sm := NewShutdownManager(orch, orch.trackerClient, cfg)

	// 没有进程时应该正常执行
	sm.TerminateAgents()
	assert.Equal(t, 0, sm.GetActiveCmdsCount())

	// 注册 nil 进程（模拟已完成的进程）
	sm.RegisterAgentProcess("issue-1", nil)
	sm.TerminateAgents()
	assert.Equal(t, 0, sm.GetActiveCmdsCount())
}

// TestShutdownManagerShutdownEmpty 测试空状态的关闭
func TestShutdownManagerShutdownEmpty(t *testing.T) {
	cfg := createShutdownTestConfig()
	orch := New(cfg, "test prompt")
	sm := NewShutdownManager(orch, orch.trackerClient, cfg)

	ctx := context.Background()
	err := sm.Shutdown(ctx)
	assert.NoError(t, err)

	// 清理
	os.RemoveAll(cfg.Workspace.Root)
}

// TestShutdownManagerUpdateTrackerState 测试更新 Tracker 状态
func TestShutdownManagerUpdateTrackerState(t *testing.T) {
	cfg := createShutdownTestConfig()
	orch := New(cfg, "test prompt")
	sm := NewShutdownManager(orch, orch.trackerClient, cfg)

	ctx := context.Background()
	err := sm.UpdateTrackerState(ctx, "test-1", "preparing", "正在准备工作空间")
	assert.NoError(t, err)
}