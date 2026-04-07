// Package orchestrator 提供优雅关闭管理
package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/domain"
	"github.com/dministrator/symphony/internal/tracker"
)

// ShutdownManager 优雅关闭管理器
type ShutdownManager struct {
	orch          *Orchestrator
	trackerClient tracker.Tracker
	cfg           *config.Config

	mu            sync.Mutex
	shuttingDown  bool
	activeCmds    map[string]*exec.Cmd // issueID -> 进程命令
	shutdownTimeout time.Duration
}

// ShutdownConfig 关闭配置
type ShutdownConfig struct {
	Timeout         time.Duration // 关闭超时时间（默认30秒）
	StateSavePath   string        // 状态保存路径
}

// DefaultShutdownConfig 默认关闭配置
func DefaultShutdownConfig() *ShutdownConfig {
	return &ShutdownConfig{
		Timeout:       30 * time.Second,
		StateSavePath: ".symphony/shutdown_state.json",
	}
}

// NewShutdownManager 创建新的关闭管理器
func NewShutdownManager(orch *Orchestrator, trackerClient tracker.Tracker, cfg *config.Config) *ShutdownManager {
	return &ShutdownManager{
		orch:            orch,
		trackerClient:   trackerClient,
		cfg:             cfg,
		activeCmds:      make(map[string]*exec.Cmd),
		shutdownTimeout: 30 * time.Second,
	}
}

// RegisterAgentProcess 注册正在运行的 Agent 进程
func (sm *ShutdownManager) RegisterAgentProcess(issueID string, cmd *exec.Cmd) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.activeCmds[issueID] = cmd
}

// UnregisterAgentProcess 移除已完成的 Agent 进程
func (sm *ShutdownManager) UnregisterAgentProcess(issueID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.activeCmds, issueID)
}

// IsShuttingDown 检查是否正在关闭
func (sm *ShutdownManager) IsShuttingDown() bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.shuttingDown
}

// Shutdown 执行优雅关闭
// 流程：
// 1. 标记 shuttingDown，停止接受新任务
// 2. 停止轮询新任务
// 3. 等待当前任务完成（30s timeout）
// 4. 如果超时 -> 终止 Agent 进程
// 5. 保存所有进行中任务状态到 Beads
// 6. 返回
func (sm *ShutdownManager) Shutdown(ctx context.Context) error {
	sm.mu.Lock()
	sm.shuttingDown = true
	activeCmds := make(map[string]*exec.Cmd)
	for k, v := range sm.activeCmds {
		activeCmds[k] = v
	}
	sm.mu.Unlock()

	fmt.Println("开始优雅关闭...")

	// 创建超时上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, sm.shutdownTimeout)
	defer cancel()

	// 等待当前任务完成
	done := sm.WaitForCompletion(timeoutCtx)

	if !done {
		fmt.Println("等待任务超时，终止 Agent 进程...")
		sm.TerminateAgents()
	}

	// 保存任务状态
	if err := sm.SaveTaskStates(); err != nil {
		fmt.Printf("保存任务状态失败: %v\n", err)
	}

	fmt.Println("优雅关闭完成")
	return nil
}

// WaitForCompletion 等待当前任务完成
func (sm *ShutdownManager) WaitForCompletion(ctx context.Context) bool {
	state := sm.orch.GetState()

	// 如果没有运行中的任务，立即返回
	if len(state.Running) == 0 {
		fmt.Println("没有运行中的任务")
		return true
	}

	fmt.Printf("等待 %d 个任务完成...\n", len(state.Running))

	// 等待所有任务完成或超时
	checkInterval := 500 * time.Millisecond
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			state := sm.orch.GetState()
			if len(state.Running) == 0 {
				return true
			}
		}
	}
}

// TerminateAgents 终止所有 Agent 进程
func (sm *ShutdownManager) TerminateAgents() {
	sm.mu.Lock()
	cmds := make(map[string]*exec.Cmd)
	for k, v := range sm.activeCmds {
		cmds[k] = v
	}
	sm.mu.Unlock()

	for issueID, cmd := range cmds {
		if cmd != nil && cmd.Process != nil {
			fmt.Printf("终止 Agent 进程: %s\n", issueID)
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
		// 无论进程是否有效，都从映射中移除
		sm.UnregisterAgentProcess(issueID)
	}
}

// SaveTaskStates 保存任务状态到文件
func (sm *ShutdownManager) SaveTaskStates() error {
	state := sm.orch.GetState()

	// 如果没有运行中的任务，不需要保存
	if len(state.Running) == 0 && len(state.RetryAttempts) == 0 {
		return nil
	}

	// 准备保存路径
	savePath := filepath.Join(sm.cfg.Workspace.Root, ".symphony")
	if err := os.MkdirAll(savePath, 0755); err != nil {
		return fmt.Errorf("创建保存目录失败: %w", err)
	}

	// 构建状态数据
	saveData := &ShutdownState{
		SavedAt:       time.Now(),
		RunningTasks:  make([]SavedTask, 0),
		RetryTasks:    make([]SavedTask, 0),
	}

	// 保存运行中的任务状态
	for issueID, entry := range state.Running {
		task := SavedTask{
			IssueID:         issueID,
			IssueIdentifier: entry.Identifier,
			StartedAt:       entry.StartedAt,
			TurnCount:       entry.TurnCount,
			RetryAttempt:    entry.RetryAttempt,
			Session:         entry.Session,
		}
		if entry.Issue != nil {
			task.IssueTitle = entry.Issue.Title
			task.IssueState = entry.Issue.State
		}
		saveData.RunningTasks = append(saveData.RunningTasks, task)
	}

	// 保存重试任务状态
	for issueID, entry := range state.RetryAttempts {
		task := SavedTask{
			IssueID:         issueID,
			IssueIdentifier: entry.Identifier,
			StartedAt:       time.UnixMilli(entry.DueAtMs),
			RetryAttempt:    &entry.Attempt,
		}
		if entry.Error != nil {
			task.LastError = *entry.Error
		}
		saveData.RetryTasks = append(saveData.RetryTasks, task)
	}

	// 序列化并保存
	data, err := json.MarshalIndent(saveData, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化状态失败: %w", err)
	}

	filePath := filepath.Join(savePath, "shutdown_state.json")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("写入状态文件失败: %w", err)
	}

	fmt.Printf("已保存 %d 个运行任务和 %d 个重试任务的状态\n",
		len(saveData.RunningTasks), len(saveData.RetryTasks))
	return nil
}

// LoadSavedStates 加载保存的任务状态
func (sm *ShutdownManager) LoadSavedStates() (*ShutdownState, error) {
	filePath := filepath.Join(sm.cfg.Workspace.Root, ".symphony", "shutdown_state.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("读取状态文件失败: %w", err)
	}

	var state ShutdownState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("解析状态文件失败: %w", err)
	}

	return &state, nil
}

// ClearSavedStates 清除保存的状态文件
func (sm *ShutdownManager) ClearSavedStates() error {
	filePath := filepath.Join(sm.cfg.Workspace.Root, ".symphony", "shutdown_state.json")
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return nil
}

// ShutdownState 关闭时保存的状态
type ShutdownState struct {
	SavedAt       time.Time    `json:"saved_at"`
	RunningTasks  []SavedTask  `json:"running_tasks"`
	RetryTasks    []SavedTask  `json:"retry_tasks"`
}

// SavedTask 保存的任务信息
type SavedTask struct {
	IssueID         string           `json:"issue_id"`
	IssueIdentifier string           `json:"issue_identifier"`
	IssueTitle      string           `json:"issue_title,omitempty"`
	IssueState      string           `json:"issue_state,omitempty"`
	StartedAt       time.Time        `json:"started_at"`
	TurnCount       int              `json:"turn_count"`
	RetryAttempt    *int             `json:"retry_attempt,omitempty"`
	LastError       string           `json:"last_error,omitempty"`
	Session         *domain.LiveSession `json:"session,omitempty"`
}

// UpdateTrackerState 更新 Tracker 中的问题状态
// 使用 Tracker 接口更新当前阶段状态（如果 Tracker 支持）
func (sm *ShutdownManager) UpdateTrackerState(ctx context.Context, issueID string, stage string, message string) error {
	// 目前 Tracker 接口不支持 UpdateStage 方法
	// 这个方法预留为未来扩展，当前仅记录日志
	fmt.Printf("任务 %s 阶段 %s: %s\n", issueID, stage, message)
	return nil
}

// GetActiveCmdsCount 获取活跃进程数量
func (sm *ShutdownManager) GetActiveCmdsCount() int {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return len(sm.activeCmds)
}