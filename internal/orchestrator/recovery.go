// Package orchestrator 提供任务状态恢复功能
package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/domain"
	"github.com/dministrator/symphony/internal/tracker"
)

// RecoveryManager 任务状态恢复管理器
type RecoveryManager struct {
	cfg           *config.Config
	trackerClient tracker.Tracker
	orch          *Orchestrator

	mu            sync.Mutex
	recoveredTasks map[string]*domain.RecoveredTask // 已恢复的任务
	maxRetries    int                               // 最大重试次数
}

// NewRecoveryManager 创建新的恢复管理器
func NewRecoveryManager(cfg *config.Config, trackerClient tracker.Tracker, orch *Orchestrator) *RecoveryManager {
	maxRetries := cfg.Execution.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	return &RecoveryManager{
		cfg:           cfg,
		trackerClient: trackerClient,
		orch:          orch,
		recoveredTasks: make(map[string]*domain.RecoveredTask),
		maxRetries:    maxRetries,
	}
}

// RestoreAll 扫描所有活跃状态任务并恢复执行
// 流程：
// 1. 获取所有活跃状态的任务
// 2. 对每个任务获取 StageState
// 3. 根据 StageState 决定恢复策略
// 4. 执行恢复动作
func (r *RecoveryManager) RestoreAll(ctx context.Context) ([]*domain.RecoveredTask, error) {
	fmt.Println("开始扫描需要恢复的任务...")

	// 获取所有活跃状态的任务
	issues, err := r.trackerClient.FetchCandidateIssues(ctx, r.cfg.Tracker.ActiveStates)
	if err != nil {
		return nil, fmt.Errorf("获取活跃任务失败: %w", err)
	}

	if len(issues) == 0 {
		fmt.Println("没有活跃状态的任务，无需恢复")
		return nil, nil
	}

	fmt.Printf("发现 %d 个活跃状态的任务\n", len(issues))

	var recovered []*domain.RecoveredTask
	var errors []string

	for _, issue := range issues {
		task, err := r.RestoreFromBeads(ctx, issue.Identifier)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", issue.Identifier, err))
			continue
		}

		if task != nil && task.Action != domain.ActionSkip {
			recovered = append(recovered, task)
			r.mu.Lock()
			r.recoveredTasks[issue.ID] = task
			r.mu.Unlock()
		}
	}

	if len(errors) > 0 {
		fmt.Printf("恢复过程中遇到 %d 个错误:\n", len(errors))
		for _, e := range errors {
			fmt.Printf("  - %s\n", e)
		}
	}

	if len(recovered) > 0 {
		fmt.Printf("成功恢复 %d 个任务\n", len(recovered))
	} else {
		fmt.Println("没有需要恢复的任务")
	}

	return recovered, nil
}

// RestoreFromBeads 从 Beads 任务恢复状态
// 流程：
// 1. 获取任务的 StageState
// 2. 根据状态决定恢复动作
// 3. 返回恢复结果
func (r *RecoveryManager) RestoreFromBeads(ctx context.Context, identifier string) (*domain.RecoveredTask, error) {
	// 获取任务详情
	issue, err := r.trackerClient.GetTask(ctx, identifier)
	if err != nil {
		return nil, fmt.Errorf("获取任务 %s 失败: %w", identifier, err)
	}

	// 检查是否为活跃状态
	if !r.cfg.IsActiveState(issue.State) {
		return &domain.RecoveredTask{
			Issue:  issue,
			Action: domain.ActionSkip,
		}, nil
	}

	// 获取阶段状态
	stageState, err := r.trackerClient.GetStageState(ctx, identifier)
	if err != nil {
		// 获取阶段状态失败，但不阻断恢复
		fmt.Printf("获取任务 %s 阶段状态失败: %v，将作为新任务处理\n", identifier, err)
		return &domain.RecoveredTask{
			Issue:  issue,
			Action: domain.ActionStart,
		}, nil
	}

	// 如果没有阶段状态，作为新任务处理
	if stageState == nil {
		return &domain.RecoveredTask{
			Issue:  issue,
			Action: domain.ActionStart,
		}, nil
	}

	// 决定恢复动作
	action := r.determineRecoveryAction(*stageState)

	return &domain.RecoveredTask{
		Issue:      issue,
		StageState: stageState,
		Action:     action,
	}, nil
}

// determineRecoveryAction 根据阶段状态决定恢复动作
func (r *RecoveryManager) determineRecoveryAction(state domain.StageState) domain.RecoveryAction {
	switch state.Status {
	case "in_progress":
		// 继续执行当前阶段
		return domain.ActionContinue
	case "pending":
		// 开始执行该阶段
		return domain.ActionStart
	case "waiting_review":
		// 等待用户审核
		return domain.ActionWaitForReview
	case "failed":
		// 检查重试次数
		if state.RetryCount >= r.maxRetries {
			// 已达到最大重试次数，跳过
			return domain.ActionSkip
		}
		return domain.ActionRetry
	case "completed":
		// 阶段已完成，跳过
		return domain.ActionSkip
	default:
		return domain.ActionUnknown
	}
}

// ExecuteRecovery 执行恢复动作
// 根据恢复动作类型执行相应的恢复操作
func (r *RecoveryManager) ExecuteRecovery(ctx context.Context, task *domain.RecoveredTask) error {
	switch task.Action {
	case domain.ActionContinue:
		// 继续执行当前阶段 - 重新调度任务
		fmt.Printf("任务 %s: 继续执行阶段 %s\n", task.Issue.Identifier, task.StageState.Name)
		// 这里可以触发 orchestrator 的调度逻辑
		return nil

	case domain.ActionStart:
		// 开始执行该阶段 - 作为新任务调度
		fmt.Printf("任务 %s: 开始执行\n", task.Issue.Identifier)
		return nil

	case domain.ActionWaitForReview:
		// 等待用户审核 - 保持任务状态
		fmt.Printf("任务 %s: 等待审核（阶段 %s）\n", task.Issue.Identifier, task.StageState.Name)
		return nil

	case domain.ActionRetry:
		// 重试该阶段
		fmt.Printf("任务 %s: 重试阶段 %s（重试次数: %d）\n", task.Issue.Identifier, task.StageState.Name, task.StageState.RetryCount+1)
		// 更新重试次数
		if task.StageState != nil {
			task.StageState.RetryCount++
			task.StageState.Status = "pending"
			task.StageState.UpdatedAt = time.Now()
			// 保存更新后的状态
			return r.trackerClient.UpdateStage(ctx, task.Issue.Identifier, *task.StageState)
		}
		return nil

	case domain.ActionSkip:
		// 跳过该任务
		fmt.Printf("任务 %s: 跳过恢复（已完成或达到最大重试次数）\n", task.Issue.Identifier)
		return nil

	default:
		return fmt.Errorf("未知的恢复动作: %s", task.Action)
	}
}

// GetRecoveredTask 获取已恢复的任务信息
func (r *RecoveryManager) GetRecoveredTask(issueID string) *domain.RecoveredTask {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.recoveredTasks[issueID]
}

// ClearRecoveredTasks 清除已恢复的任务记录
func (r *RecoveryManager) ClearRecoveredTasks() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recoveredTasks = make(map[string]*domain.RecoveredTask)
}

// RecoveryStats 恢复统计信息
type RecoveryStats struct {
	TotalScanned     int `json:"total_scanned"`
	TotalRecovered   int `json:"total_recovered"`
	ContinueCount    int `json:"continue_count"`
	StartCount       int `json:"start_count"`
	WaitReviewCount  int `json:"wait_review_count"`
	RetryCount       int `json:"retry_count"`
	SkipCount        int `json:"skip_count"`
	ErrorCount       int `json:"error_count"`
}

// GetRecoveryStats 获取恢复统计信息
func (r *RecoveryManager) GetRecoveryStats() *RecoveryStats {
	r.mu.Lock()
	defer r.mu.Unlock()

	stats := &RecoveryStats{}
	for _, task := range r.recoveredTasks {
		stats.TotalRecovered++
		switch task.Action {
		case domain.ActionContinue:
			stats.ContinueCount++
		case domain.ActionStart:
			stats.StartCount++
		case domain.ActionWaitForReview:
			stats.WaitReviewCount++
		case domain.ActionRetry:
			stats.RetryCount++
		case domain.ActionSkip:
			stats.SkipCount++
		default:
			stats.ErrorCount++
		}
	}
	return stats
}

// ValidateStageState 验证阶段状态的序列化格式
func ValidateStageState(state domain.StageState) error {
	if state.Name == "" {
		return fmt.Errorf("stage name is required")
	}
	if state.Status == "" {
		return fmt.Errorf("stage status is required")
	}
	validStatuses := map[string]bool{
		"pending":        true,
		"in_progress":    true,
		"completed":      true,
		"failed":         true,
		"waiting_review": true,
	}
	if !validStatuses[state.Status] {
		return fmt.Errorf("invalid stage status: %s", state.Status)
	}
	return nil
}

// SerializeStageState 序列化阶段状态为 JSON
func SerializeStageState(state domain.StageState) ([]byte, error) {
	if err := ValidateStageState(state); err != nil {
		return nil, err
	}
	return json.Marshal(state)
}

// DeserializeStageState 从 JSON 反序列化阶段状态
func DeserializeStageState(data []byte) (*domain.StageState, error) {
	var state domain.StageState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("解析阶段状态失败: %w", err)
	}
	if err := ValidateStageState(state); err != nil {
		return nil, err
	}
	return &state, nil
}