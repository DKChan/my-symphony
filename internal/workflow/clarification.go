// Package workflow 提供澄清轮次限制、跳过功能和AI Agent需求理解调用
package workflow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dministrator/symphony/internal/agent"
	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/domain"
	"github.com/dministrator/symphony/internal/tracker"
)

// IncompleteReason 不完整原因
type IncompleteReason string

const (
	// ReasonRoundLimit 澄清轮次达到上限
	ReasonRoundLimit IncompleteReason = "澄清轮次已达上限"
	// ReasonUserSkip 用户跳过澄清
	ReasonUserSkip IncompleteReason = "用户跳过澄清"
)

// ClarificationStatusType 澄清状态类型（AI Agent响应）
type ClarificationStatusType string

const (
	// StatusNeedsClarification 需要澄清
	StatusNeedsClarification ClarificationStatusType = "needs_clarification"
	// StatusClear 需求已明确
	StatusClear ClarificationStatusType = "clear"
)

// ClarificationQuestion 澄清问题
type ClarificationQuestion struct {
	// ID 问题ID
	ID string `json:"id"`
	// Question 问题内容
	Question string `json:"question"`
}

// ClarificationResponse AI Agent 返回的澄清响应
type ClarificationResponse struct {
	// Status 状态
	Status ClarificationStatusType `json:"status"`
	// Questions 澄清问题列表
	Questions []ClarificationQuestion `json:"questions"`
	// Summary 需求摘要
	Summary string `json:"summary"`
}

// ClarificationResult 澄清结果
type ClarificationResult struct {
	// Status 澄清状态
	Status ClarificationStatusType
	// Questions 需要用户回答的问题
	Questions []ClarificationQuestion
	// Summary 需求摘要
	Summary string
	// Error 错误信息
	Error error
	// WaitingForUser 是否等待用户回答
	WaitingForUser bool
}

// ClarificationManager 澄清管理器，负责处理澄清轮次限制、跳过逻辑和AI Agent需求理解调用
type ClarificationManager struct {
	engine     *Engine
	config     *config.Config
	tracker    tracker.Tracker
	runner     agent.Runner
	promptTmpl string
}

// NewClarificationManager 创建新的澄清管理器
func NewClarificationManager(engine *Engine, cfg *config.Config) *ClarificationManager {
	return &ClarificationManager{
		engine: engine,
		config: cfg,
	}
}

// NewClarificationManagerWithTracker 创建带 tracker 的澄清管理器
func NewClarificationManagerWithTracker(engine *Engine, cfg *config.Config, t tracker.Tracker) *ClarificationManager {
	return &ClarificationManager{
		engine:  engine,
		config:  cfg,
		tracker: t,
	}
}

// SetTracker 设置 tracker（用于依赖注入）
func (cm *ClarificationManager) SetTracker(t tracker.Tracker) {
	cm.tracker = t
}

// SetRunner 设置 AI Agent 运行器（用于依赖注入）
func (cm *ClarificationManager) SetRunner(r agent.Runner) {
	cm.runner = r
}

// CheckRoundLimit 检查澄清轮次是否达到上限
// 返回值:
// - reached: 是否达到上限
// - currentRound: 当前轮次
// - maxRounds: 最大轮次
// - err: 错误信息
func (cm *ClarificationManager) CheckRoundLimit(taskID string) (reached bool, currentRound int, maxRounds int, err error) {
	// 获取工作流
	workflow := cm.engine.GetWorkflow(taskID)
	if workflow == nil {
		return false, 0, 0, ErrWorkflowNotFound
	}

	// 获取澄清阶段状态
	clarificationStage := workflow.Stages[StageClarification]
	if clarificationStage == nil {
		return false, 0, 0, ErrInvalidStage
	}

	// 检查是否在澄清阶段
	if workflow.CurrentStage != StageClarification {
		return false, clarificationStage.Round, cm.config.Clarification.MaxRounds, nil
	}

	currentRound = clarificationStage.Round
	maxRounds = cm.config.Clarification.MaxRounds

	// 检查是否达到上限（轮次从0开始，达到maxRounds时触发）
	reached = currentRound >= maxRounds

	return reached, currentRound, maxRounds, nil
}

// SkipClarification 跳过澄清阶段
// 标记需求为不完整，直接流转到下一阶段
func (cm *ClarificationManager) SkipClarification(taskID string) (*TaskWorkflow, error) {
	// 获取工作流
	workflow := cm.engine.GetWorkflow(taskID)
	if workflow == nil {
		return nil, ErrWorkflowNotFound
	}

	// 检查是否在澄清阶段
	if workflow.CurrentStage != StageClarification {
		return nil, fmt.Errorf("%w: current stage is %s, not clarification", ErrInvalidTransition, workflow.CurrentStage)
	}

	// 检查澄清阶段状态
	clarificationStage := workflow.Stages[StageClarification]
	if clarificationStage == nil {
		return nil, ErrInvalidStage
	}

	// 标记为不完整并推进到下一阶段
	return cm.markIncompleteAndAdvance(taskID, ReasonUserSkip)
}

// HandleRoundLimitReached 处理澄清轮次达到上限的情况
// 标记需求为不完整，流转到 needs_attention 状态
func (cm *ClarificationManager) HandleRoundLimitReached(taskID string) (*TaskWorkflow, error) {
	// 获取工作流
	workflow := cm.engine.GetWorkflow(taskID)
	if workflow == nil {
		return nil, ErrWorkflowNotFound
	}

	// 检查是否在澄清阶段
	if workflow.CurrentStage != StageClarification {
		return nil, fmt.Errorf("%w: current stage is %s, not clarification", ErrInvalidTransition, workflow.CurrentStage)
	}

	// 标记为不完整并设置 needs_attention 状态
	return cm.markIncompleteAndNeedsAttention(taskID, ReasonRoundLimit)
}

// MarkIncomplete 标记需求为不完整
// 设置不完整标记和原因
func (cm *ClarificationManager) MarkIncomplete(taskID string, reason IncompleteReason) (*TaskWorkflow, error) {
	// 获取工作流
	workflow := cm.engine.GetWorkflow(taskID)
	if workflow == nil {
		return nil, ErrWorkflowNotFound
	}

	// 使用内部方法标记不完整
	return cm.markIncompleteInternal(taskID, reason, false)
}

// markIncompleteAndAdvance 标记不完整并推进到下一阶段
func (cm *ClarificationManager) markIncompleteAndAdvance(taskID string, reason IncompleteReason) (*TaskWorkflow, error) {
	return cm.markIncompleteInternal(taskID, reason, false)
}

// markIncompleteAndNeedsAttention 标记不完整并设置 needs_attention 状态
func (cm *ClarificationManager) markIncompleteAndNeedsAttention(taskID string, reason IncompleteReason) (*TaskWorkflow, error) {
	return cm.markIncompleteInternal(taskID, reason, true)
}

// markIncompleteInternal 内部方法：标记不完整
func (cm *ClarificationManager) markIncompleteInternal(taskID string, reason IncompleteReason, needsAttention bool) (*TaskWorkflow, error) {
	// 使用 engine 的锁定方法
	workflow, err := cm.engine.SetIncompleteMark(taskID, reason, needsAttention)
	if err != nil {
		return nil, err
	}

	return workflow, nil
}

// IncrementRound 增加澄清轮次并检查是否达到上限
// 返回值:
// - newRound: 新的轮次数
// - reachedLimit: 是否达到上限
// - err: 错误信息
func (cm *ClarificationManager) IncrementRound(taskID string) (newRound int, reachedLimit bool, err error) {
	// 增加轮次
	newRound, err = cm.engine.IncrementRound(taskID)
	if err != nil {
		return 0, false, err
	}

	// 检查是否达到上限
	reachedLimit, _, _, err = cm.CheckRoundLimit(taskID)
	if err != nil {
		return newRound, false, err
	}

	return newRound, reachedLimit, nil
}

// GetClarificationStatus 获取澄清状态详情
func (cm *ClarificationManager) GetClarificationStatus(taskID string) (*ClarificationStatus, error) {
	workflow := cm.engine.GetWorkflow(taskID)
	if workflow == nil {
		return nil, ErrWorkflowNotFound
	}

	clarificationStage := workflow.Stages[StageClarification]
	if clarificationStage == nil {
		return nil, ErrInvalidStage
	}

	status := &ClarificationStatus{
		TaskID:           taskID,
		CurrentStage:     workflow.CurrentStage,
		CurrentRound:     clarificationStage.Round,
		MaxRounds:        cm.config.Clarification.MaxRounds,
		RoundRemaining:   cm.config.Clarification.MaxRounds - clarificationStage.Round,
		Status:           clarificationStage.Status,
		IsIncomplete:     workflow.IsIncomplete,
		IncompleteReason: IncompleteReason(workflow.IncompleteReason),
		NeedsAttention:   workflow.NeedsAttention,
	}

	// 计算是否达到上限
	status.RoundLimitReached = clarificationStage.Round >= cm.config.Clarification.MaxRounds

	return status, nil
}

// ClarificationStatus 澄清状态详情
type ClarificationStatus struct {
	TaskID            string           `json:"task_id"`
	CurrentStage      StageName        `json:"current_stage"`
	CurrentRound      int              `json:"current_round"`
	MaxRounds         int              `json:"max_rounds"`
	RoundRemaining    int              `json:"round_remaining"`
	RoundLimitReached bool             `json:"round_limit_reached"`
	Status            StageStatus      `json:"status"`
	IsIncomplete      bool             `json:"is_incomplete"`
	IncompleteReason  IncompleteReason `json:"incomplete_reason,omitempty"`
	NeedsAttention    bool             `json:"needs_attention"`
}

// BDDReviewStatus BDD 审核状态详情
type BDDReviewStatus struct {
	TaskID        string      `json:"task_id"`
	CurrentStage  StageName   `json:"current_stage"`
	Status        StageStatus `json:"status"`
	CanApprove    bool        `json:"can_approve"`
	CanReject     bool        `json:"can_reject"`
	Approved      bool        `json:"approved"`
	Rejected      bool        `json:"rejected"`
	RejectReason  string      `json:"reject_reason,omitempty"`
	NeedsAttention bool       `json:"needs_attention"`
}

// ShouldAdvanceToNeedsAttention 判断是否应该流转到 needs_attention
// 当达到轮次上限时返回 true
func (cm *ClarificationManager) ShouldAdvanceToNeedsAttention(taskID string) (bool, error) {
	reached, _, _, err := cm.CheckRoundLimit(taskID)
	if err != nil {
		return false, err
	}
	return reached, nil
}

// CanSkipClarification 判断是否可以跳过澄清
// 仅在澄清阶段进行中时可以跳过
func (cm *ClarificationManager) CanSkipClarification(taskID string) (bool, error) {
	workflow := cm.engine.GetWorkflow(taskID)
	if workflow == nil {
		return false, ErrWorkflowNotFound
	}

	// 必须在澄清阶段且进行中状态
	if workflow.CurrentStage != StageClarification {
		return false, nil
	}

	clarificationStage := workflow.Stages[StageClarification]
	if clarificationStage == nil {
		return false, ErrInvalidStage
	}

	// 阶段必须是进行中状态
	return clarificationStage.Status == StatusInProgress, nil
}

// GetMaxRounds 获取配置的最大轮次
func (cm *ClarificationManager) GetMaxRounds() int {
	return cm.config.Clarification.MaxRounds
}

// CompleteClarification 完成澄清阶段（正常完成，不标记不完整）
func (cm *ClarificationManager) CompleteClarification(taskID string) (*TaskWorkflow, error) {
	workflow := cm.engine.GetWorkflow(taskID)
	if workflow == nil {
		return nil, ErrWorkflowNotFound
	}

	// 检查是否在澄清阶段
	if workflow.CurrentStage != StageClarification {
		return nil, fmt.Errorf("%w: current stage is %s, not clarification", ErrInvalidTransition, workflow.CurrentStage)
	}

	// 正常推进阶段
	return cm.engine.AdvanceStage(taskID)
}

// UpdateStageTimestamp 更新阶段时间戳（用于每轮开始时）
func (cm *ClarificationManager) UpdateStageTimestamp(taskID string) error {
	return cm.engine.UpdateStageTime(taskID)
}

// IsInClarificationStage 判断任务是否在澄清阶段
func (cm *ClarificationManager) IsInClarificationStage(taskID string) (bool, error) {
	workflow := cm.engine.GetWorkflow(taskID)
	if workflow == nil {
		return false, ErrWorkflowNotFound
	}

	return workflow.CurrentStage == StageClarification, nil
}

// TimeSinceLastUpdate 获取自上次更新以来的时间
func (cm *ClarificationManager) TimeSinceLastUpdate(taskID string) (time.Duration, error) {
	workflow := cm.engine.GetWorkflow(taskID)
	if workflow == nil {
		return 0, ErrWorkflowNotFound
	}

	clarificationStage := workflow.Stages[StageClarification]
	if clarificationStage == nil {
		return 0, ErrInvalidStage
	}

	if clarificationStage.UpdatedAt == nil {
		return 0, nil
	}

	return time.Since(*clarificationStage.UpdatedAt), nil
}

// SubmitAnswerResult 提交回答结果
type SubmitAnswerResult struct {
	// NeedsMoreClarification 是否需要更多澄清
	NeedsMoreClarification bool `json:"needs_more_clarification"`
	// Question 新问题（如果需要更多澄清）
	Question string `json:"question,omitempty"`
	// Summary 需求摘要（如果澄清完成）
	Summary string `json:"summary,omitempty"`
	// Round 当前轮次
	Round int `json:"round"`
	// Stage 当前阶段状态
	Stage *StageState `json:"stage"`
	// Status 澄清状态
	Status ClarificationStatusType `json:"status"`
}

// SubmitAnswer 提交回答并继续澄清流程
// 流程:
// 1. AppendConversation(user, answer) 保存回答
// 2. 分析回答内容
// 3. 如果需要更多澄清: AppendConversation(assistant, question) 返回新问题
// 4. 否则: UpdateStage(bdd_review, pending) 返回"需求已明确"
func (cm *ClarificationManager) SubmitAnswer(ctx context.Context, taskID, identifier, answer string) (*SubmitAnswerResult, error) {
	// 检查 tracker 是否可用
	if cm.tracker == nil {
		return nil, fmt.Errorf("tracker not configured")
	}

	// 获取当前工作流状态
	workflow := cm.engine.GetWorkflow(taskID)
	if workflow == nil {
		return nil, ErrWorkflowNotFound
	}

	// 检查当前阶段是否为澄清阶段
	if workflow.CurrentStage != StageClarification {
		return nil, fmt.Errorf("%w: current stage is %s, not clarification", ErrInvalidTransition, workflow.CurrentStage)
	}

	currentStage := workflow.Stages[StageClarification]
	if currentStage == nil {
		return nil, ErrInvalidStage
	}

	// 检查阶段状态
	if currentStage.Status != StatusInProgress {
		return nil, fmt.Errorf("clarification stage is not in progress: %s", currentStage.Status)
	}

	// 保存用户回答
	userTurn := domain.ConversationTurn{
		Role:      "user",
		Content:   answer,
		Timestamp: time.Now(),
	}

	if err := cm.tracker.AppendConversation(ctx, identifier, userTurn); err != nil {
		return nil, fmt.Errorf("failed to save user answer: %w", err)
	}

	// 增加轮次
	newRound, reachedLimit, err := cm.IncrementRound(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to increment round: %w", err)
	}

	// 更新阶段时间戳
	_ = cm.UpdateStageTimestamp(taskID)

	result := &SubmitAnswerResult{
		Round:  newRound,
		Stage:  currentStage,
		Status: StatusNeedsClarification,
	}

	// 检查是否达到轮次上限
	if reachedLimit {
		// 达到轮次上限，标记为不完整并流转到 needs_attention
		_, err := cm.HandleRoundLimitReached(taskID)
		if err != nil {
			return nil, fmt.Errorf("failed to handle round limit: %w", err)
		}

		result.NeedsMoreClarification = false
		result.Summary = "澄清轮次已达上限，请人工介入"
		result.Status = StatusClear
		return result, nil
	}

	// 分析回答内容（模拟 AI Agent 的逻辑）
	// 实际实现中，这里会调用 AI Agent 来分析回答并生成新问题
	needsMore, question, summary := cm.analyzeAnswer(answer, currentStage.Round)

	if needsMore {
		// 需要更多澄清，保存 AI 问题
		assistantTurn := domain.ConversationTurn{
			Role:      "assistant",
			Content:   question,
			Timestamp: time.Now(),
		}

		if err := cm.tracker.AppendConversation(ctx, identifier, assistantTurn); err != nil {
			return nil, fmt.Errorf("failed to save assistant question: %w", err)
		}

		result.NeedsMoreClarification = true
		result.Question = question
		result.Status = StatusNeedsClarification
	} else {
		// 澄清完成，正常推进到下一阶段
		_, err := cm.CompleteClarification(taskID)
		if err != nil {
			return nil, fmt.Errorf("failed to complete clarification: %w", err)
		}

		// 更新 tracker 阶段状态
		_ = cm.tracker.UpdateStage(ctx, identifier, domain.StageState{
			Name:   "bdd_review",
			Status: "pending",
		})

		result.NeedsMoreClarification = false
		result.Summary = summary
		result.Status = StatusClear
	}

	// 更新当前阶段状态
	workflow = cm.engine.GetWorkflow(taskID)
	if workflow != nil {
		result.Stage = workflow.Stages[workflow.CurrentStage]
		if result.Stage == nil {
			result.Stage = currentStage
		}
	}

	return result, nil
}

// analyzeAnswer 分析回答内容（模拟 AI Agent 逻辑）
// 实际实现中，这里会调用 AI Agent 来分析回答
func (cm *ClarificationManager) analyzeAnswer(answer string, currentRound int) (needsMore bool, question string, summary string) {
	// 模拟分析逻辑：
	// - 简短回答（少于10字符）通常需要更多澄清
	// - 包含"不清楚"、"不确定"等关键词需要更多澄清
	// - 包含明确需求描述则认为澄清完成

	// 检查回答长度
	if len(answer) < 10 {
		return true, "您的回答较短，能否详细说明一下具体需求？", ""
	}

	// 检查关键词
	unclearKeywords := []string{"不清楚", "不确定", "待定", "不知道", "?", "？"}
	for _, keyword := range unclearKeywords {
		if strings.Contains(answer, keyword) {
			return true, "您提到还有一些不确定的地方，请详细说明哪些方面需要进一步澄清？", ""
		}
	}

	// 检查是否有明确的需求描述
	clearKeywords := []string{"功能", "需求", "实现", "具体", "详细", "明确", "完成", "是的", "确定"}
	for _, keyword := range clearKeywords {
		if strings.Contains(answer, keyword) && len(answer) > 20 {
			// 看起来是明确的回答，但还可以再确认一轮
			if currentRound < 2 {
				return true, "感谢您的回答。请问还有其他需要澄清的细节吗？例如性能要求、约束条件等？", ""
			}
			return false, "", "需求已明确，可以开始编写 BDD 测试规范"
		}
	}

	// 默认情况：根据轮次决定
	if currentRound < 2 {
		return true, "感谢您的回答。请问还有什么补充信息吗？", ""
	}

	return false, "", "需求澄清已完成，可以进入下一阶段"
}

// GetClarificationState 获取澄清状态（包含最后一个问题）
func (cm *ClarificationManager) GetClarificationState(ctx context.Context, taskID, identifier string) (*SubmitAnswerResult, error) {
	// 检查 tracker 是否可用
	if cm.tracker == nil {
		return nil, fmt.Errorf("tracker not configured")
	}

	workflow := cm.engine.GetWorkflow(taskID)
	if workflow == nil {
		return nil, ErrWorkflowNotFound
	}

	stage := workflow.Stages[StageClarification]
	if stage == nil {
		return nil, ErrInvalidStage
	}

	// 获取对话历史
	history, err := cm.tracker.GetConversationHistory(ctx, identifier)
	if err != nil {
		history = []domain.ConversationTurn{}
	}

	// 查找最后一个 AI 问题
	var lastQuestion string
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "assistant" {
			lastQuestion = history[i].Content
			break
		}
	}

	return &SubmitAnswerResult{
		NeedsMoreClarification: stage.Status == StatusInProgress,
		Question:               lastQuestion,
		Round:                  stage.Round,
		Stage:                  stage,
		Status:                 StatusNeedsClarification,
	}, nil
}

// StartClarification 开始澄清流程
func (cm *ClarificationManager) StartClarification(ctx context.Context, taskID, identifier string, initialQuestion string) (*SubmitAnswerResult, error) {
	// 检查 tracker 是否可用
	if cm.tracker == nil {
		return nil, fmt.Errorf("tracker not configured")
	}

	// 初始化工作流（如果不存在）
	workflow := cm.engine.GetWorkflow(taskID)
	if workflow == nil {
		_, err := cm.engine.InitTask(taskID)
		if err != nil {
			return nil, fmt.Errorf("failed to init workflow: %w", err)
		}
	}

	// 保存初始问题
	if initialQuestion != "" {
		assistantTurn := domain.ConversationTurn{
			Role:      "assistant",
			Content:   initialQuestion,
			Timestamp: time.Now(),
		}

		if err := cm.tracker.AppendConversation(ctx, identifier, assistantTurn); err != nil {
			return nil, fmt.Errorf("failed to save initial question: %w", err)
		}
	}

	// 获取当前状态
	return cm.GetClarificationState(ctx, taskID, identifier)
}

// BDDReviewManager BDD 审核管理器，负责处理 BDD 规则审核的通过和驳回
type BDDReviewManager struct {
	engine  *Engine
	tracker tracker.Tracker
}

// NewBDDReviewManager 创建新的 BDD 审核管理器
func NewBDDReviewManager(engine *Engine) *BDDReviewManager {
	return &BDDReviewManager{
		engine: engine,
	}
}

// NewBDDReviewManagerWithTracker 创建带 tracker 的 BDD 审核管理器
func NewBDDReviewManagerWithTracker(engine *Engine, t tracker.Tracker) *BDDReviewManager {
	return &BDDReviewManager{
		engine:  engine,
		tracker: t,
	}
}

// SetTracker 设置 tracker（用于依赖注入）
func (bm *BDDReviewManager) SetTracker(t tracker.Tracker) {
	bm.tracker = t
}

// ApproveBDD 通过 BDD 规则审核
// 状态流转: bdd_review (pending/in_progress) -> architecture_review (pending)
func (bm *BDDReviewManager) ApproveBDD(taskID string) (*TaskWorkflow, error) {
	return bm.engine.ApproveBDD(taskID)
}

// RejectBDD 驳回 BDD 规则审核
// 状态流转: bdd_review (pending/in_progress) -> clarification (in_progress)
func (bm *BDDReviewManager) RejectBDD(taskID string, reason string) (*TaskWorkflow, error) {
	return bm.engine.RejectBDD(taskID, reason)
}

// GetBDDReviewStatus 获取 BDD 审核状态详情
func (bm *BDDReviewManager) GetBDDReviewStatus(taskID string) (*BDDReviewStatus, error) {
	workflow := bm.engine.GetWorkflow(taskID)
	if workflow == nil {
		return nil, ErrWorkflowNotFound
	}

	bddStage := workflow.Stages[StageBDDReview]
	if bddStage == nil {
		return nil, ErrInvalidStage
	}

	status := &BDDReviewStatus{
		TaskID:       taskID,
		CurrentStage: workflow.CurrentStage,
		Status:       bddStage.Status,
		Approved:     bddStage.Status == StatusCompleted,
		Rejected:     bddStage.Status == StatusFailed,
		RejectReason: bddStage.Error,
		NeedsAttention: workflow.NeedsAttention,
	}

	// 判断是否可以进行审核操作
	canApproveOrReject := workflow.CurrentStage == StageBDDReview &&
		(bddStage.Status == StatusPending || bddStage.Status == StatusInProgress)
	status.CanApprove = canApproveOrReject
	status.CanReject = canApproveOrReject

	return status, nil
}

// CanApproveOrReject 判断是否可以进行审核操作
// 仅在 BDD 审核阶段进行中时可以审核
func (bm *BDDReviewManager) CanApproveOrReject(taskID string) (bool, error) {
	workflow := bm.engine.GetWorkflow(taskID)
	if workflow == nil {
		return false, ErrWorkflowNotFound
	}

	// 必须在 BDD 审核阶段
	if workflow.CurrentStage != StageBDDReview {
		return false, nil
	}

	bddStage := workflow.Stages[StageBDDReview]
	if bddStage == nil {
		return false, ErrInvalidStage
	}

	// 阶段必须是 pending 或 in_progress 状态
	return bddStage.Status == StatusPending || bddStage.Status == StatusInProgress, nil
}

// IsInBDDReviewStage 判断任务是否在 BDD 审核阶段
func (bm *BDDReviewManager) IsInBDDReviewStage(taskID string) (bool, error) {
	workflow := bm.engine.GetWorkflow(taskID)
	if workflow == nil {
		return false, ErrWorkflowNotFound
	}

	return workflow.CurrentStage == StageBDDReview, nil
}