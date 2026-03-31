package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/dministrator/symphony/internal/common"
	"github.com/dministrator/symphony/internal/domain"
	"github.com/dministrator/symphony/internal/server/presenter"
	"github.com/dministrator/symphony/internal/tracker"
	"github.com/dministrator/symphony/internal/workflow"
	"github.com/gin-gonic/gin"
)

// OrchestratorGetter 定义获取 orchestrator 状态的接口
type OrchestratorGetter interface {
	GetState() *domain.OrchestratorState
	GetTracker() tracker.Tracker
}

// OrchestratorCanceler 定义取消任务的接口
type OrchestratorCanceler interface {
	CancelTask(identifier string) (cancelled bool, notFound bool, err error)
	GetRunningEntryByIdentifier(identifier string) *domain.RunningEntry
	GetRetryEntryByIdentifier(identifier string) *domain.RetryEntry
}

// ClarificationManager 定义澄清管理接口
type ClarificationManager interface {
	SkipClarification(taskID string) (*workflow.TaskWorkflow, error)
	GetClarificationStatus(taskID string) (*workflow.ClarificationStatus, error)
	CanSkipClarification(taskID string) (bool, error)
	CheckRoundLimit(taskID string) (reached bool, currentRound int, maxRounds int, err error)
	SubmitAnswer(ctx context.Context, taskID, identifier, answer string) (*workflow.SubmitAnswerResult, error)
	GetClarificationState(ctx context.Context, taskID, identifier string) (*workflow.SubmitAnswerResult, error)
}

// BDDReviewManager 定义 BDD 审核管理接口
type BDDReviewManager interface {
	ApproveBDD(taskID string) (*workflow.TaskWorkflow, error)
	RejectBDD(taskID string, reason string) (*workflow.TaskWorkflow, error)
	GetBDDReviewStatus(taskID string) (*workflow.BDDReviewStatus, error)
	CanApproveOrReject(taskID string) (bool, error)
}

// ArchitectureReviewManager 定义架构审核管理接口
type ArchitectureReviewManager interface {
	ApproveArchitecture(taskID string) (*workflow.TaskWorkflow, error)
	RejectArchitecture(taskID string, reason string) (*workflow.TaskWorkflow, error)
	GetArchitectureReviewStatus(taskID string) (*workflow.ArchitectureReviewStatus, error)
	CanApproveOrRejectArchitecture(taskID string) (bool, error)
}

// VerificationManager 定义验收审核管理接口
type VerificationManager interface {
	ApproveVerification(taskID string) (*workflow.TaskWorkflow, error)
	RejectVerification(taskID string, reason string) (*workflow.TaskWorkflow, error)
	GetVerificationStatus(taskID string) (*workflow.VerificationStatus, error)
	CanApproveOrRejectVerification(taskID string) (bool, error)
}

// GitCommitter 定义 Git 提交接口
type GitCommitter interface {
	GitCommit(ctx context.Context, workspacePath string, identifier string, title string) (string, error)
	GetWorkspacePath(identifier string) string
}

// NeedsAttentionManager 定义人工干预管理接口
type NeedsAttentionManager interface {
	TransitionToNeedsAttention(taskID string, details workflow.NeedsAttentionDetails) (*workflow.TaskWorkflow, error)
	ResumeTask(taskID string) (*workflow.TaskWorkflow, error)
	ReclarifyTask(taskID string) (*workflow.TaskWorkflow, error)
	AbandonTask(taskID string) (*workflow.TaskWorkflow, error)
	GetFailureDetails(taskID string) (*workflow.NeedsAttentionDetails, error)
	GetNeedsAttentionTasks() []*workflow.TaskWorkflow
}

// WorkspaceCleaner 定义工作空间清理接口
type WorkspaceCleaner interface {
	RemoveWorkspace(ctx context.Context, path string) error
	GetWorkspacePath(identifier string) string
}

// APIHandler API 处理器，提供状态、问题和刷新相关的 API 端点
type APIHandler struct {
	orchestrator             OrchestratorGetter
	canceler                 OrchestratorCanceler
	clarificationManager     ClarificationManager
	bddReviewManager         BDDReviewManager
	architectureReviewManager ArchitectureReviewManager
	verificationManager      VerificationManager
	gitCommitter             GitCommitter
	needsAttentionManager    NeedsAttentionManager
	workspaceCleaner         WorkspaceCleaner
}

// NewAPIHandler 创建新的 API 处理器
func NewAPIHandler(orch OrchestratorGetter) *APIHandler {
	return &APIHandler{
		orchestrator: orch,
	}
}

// NewAPIHandlerWithCanceler 创建带取消功能的 API 处理器
func NewAPIHandlerWithCanceler(orch OrchestratorGetter, canceler OrchestratorCanceler) *APIHandler {
	return &APIHandler{
		orchestrator: orch,
		canceler:     canceler,
	}
}

// NewAPIHandlerWithClarification 创建带澄清管理功能的 API 处理器
func NewAPIHandlerWithClarification(orch OrchestratorGetter, canceler OrchestratorCanceler, clarificationManager ClarificationManager) *APIHandler {
	return &APIHandler{
		orchestrator:         orch,
		canceler:             canceler,
		clarificationManager: clarificationManager,
	}
}

// NewAPIHandlerWithBDDReview 创建带 BDD 审核管理功能的 API 处理器
func NewAPIHandlerWithBDDReview(orch OrchestratorGetter, canceler OrchestratorCanceler, clarificationManager ClarificationManager, bddReviewManager BDDReviewManager) *APIHandler {
	return &APIHandler{
		orchestrator:         orch,
		canceler:             canceler,
		clarificationManager: clarificationManager,
		bddReviewManager:     bddReviewManager,
	}
}

// NewAPIHandlerWithVerification 创建带验收审核管理功能的 API 处理器
func NewAPIHandlerWithVerification(orch OrchestratorGetter, canceler OrchestratorCanceler, clarificationManager ClarificationManager, bddReviewManager BDDReviewManager, verificationManager VerificationManager, gitCommitter GitCommitter) *APIHandler {
	return &APIHandler{
		orchestrator:         orch,
		canceler:             canceler,
		clarificationManager: clarificationManager,
		bddReviewManager:     bddReviewManager,
		verificationManager:  verificationManager,
		gitCommitter:         gitCommitter,
	}
}

// NewAPIHandlerWithNeedsAttention 创建带人工干预管理功能的 API 处理器
func NewAPIHandlerWithNeedsAttention(orch OrchestratorGetter, canceler OrchestratorCanceler, clarificationManager ClarificationManager, bddReviewManager BDDReviewManager, verificationManager VerificationManager, gitCommitter GitCommitter, needsAttentionManager NeedsAttentionManager, workspaceCleaner WorkspaceCleaner) *APIHandler {
	return &APIHandler{
		orchestrator:          orch,
		canceler:              canceler,
		clarificationManager:  clarificationManager,
		bddReviewManager:      bddReviewManager,
		verificationManager:   verificationManager,
		gitCommitter:          gitCommitter,
		needsAttentionManager: needsAttentionManager,
		workspaceCleaner:      workspaceCleaner,
	}
}

// HandleGetState 处理获取状态的请求
func (h *APIHandler) HandleGetState(c *gin.Context) {
	state := h.orchestrator.GetState()
	payload := presenter.BuildStatePayload(state)
	c.JSON(http.StatusOK, payload)
}

// HandleGetIssue 处理获取单个问题的请求
func (h *APIHandler) HandleGetIssue(c *gin.Context) {
	identifier := c.Param("identifier")
	state := h.orchestrator.GetState()

	response, err := presenter.BuildIssuePayload(identifier, state)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "issue_not_found",
				"message": "issue not found in current state",
			},
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// HandleRefresh 处理刷新请求
func (h *APIHandler) HandleRefresh(c *gin.Context) {
	response := presenter.BuildRefreshPayload()
	c.JSON(http.StatusAccepted, response)
}

// HandleGetTasks 处理获取任务列表的请求（支持状态筛选）
func (h *APIHandler) HandleGetTasks(c *gin.Context) {
	stateParam := c.Query("state")

	// 解析筛选状态
	filterStates := common.ParseFilterState(stateParam)

	// 合并跟踪器状态
	trackerStates := common.MergeFilterStates(filterStates)

	// 构建筛选标签
	filterLabel := buildFilterLabel(filterStates)

	// 获取跟踪器客户端
	trackerClient := h.orchestrator.GetTracker()

	// 查询任务
	ctx := context.Background()
	issues, err := trackerClient.ListTasksByState(ctx, trackerStates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "tracker_error",
				"message": err.Error(),
			},
		})
		return
	}

	// 构建响应
	payload := presenter.BuildTasksPayload(issues, stateParam, filterLabel)
	c.JSON(http.StatusOK, payload)
}

// buildFilterLabel 构建筛选状态标签
func buildFilterLabel(filters []common.TaskFilterState) string {
	if len(filters) == 0 {
		return common.TaskFilterLabel[common.FilterAll]
	}

	labels := make([]string, 0, len(filters))
	for _, f := range filters {
		if label, ok := common.TaskFilterLabel[f]; ok {
			labels = append(labels, label)
		}
	}

	if len(labels) == 0 {
		return common.TaskFilterLabel[common.FilterAll]
	}

	return strings.Join(labels, ", ")
}

// HandleCancelTask 处理取消任务的请求
// POST /api/v1/:identifier/cancel
func (h *APIHandler) HandleCancelTask(c *gin.Context) {
	identifier := c.Param("identifier")

	// 检查是否有取消器
	if h.canceler == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "cancel_not_supported",
				"message": "cancel functionality not available",
			},
		})
		return
	}

	// 检查任务是否存在
	runningEntry := h.canceler.GetRunningEntryByIdentifier(identifier)
	retryEntry := h.canceler.GetRetryEntryByIdentifier(identifier)

	if runningEntry == nil && retryEntry == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "task_not_found",
				"message": "task not found in running or retry queue",
			},
		})
		return
	}

	// 执行取消
	cancelled, notFound, err := h.canceler.CancelTask(identifier)

	if notFound {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "task_not_found",
				"message": "task no longer exists",
			},
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "cancel_failed",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        cancelled,
		"identifier":     identifier,
		"previous_state": getStateDisplayName(runningEntry, retryEntry),
		"new_state":      "cancelled",
		"message":        "task has been cancelled successfully",
		"warning":        "this operation is irreversible",
	})
}

// HandleCancelConfirm 处理取消确认请求（返回确认提示）
// GET /api/v1/:identifier/cancel/confirm
func (h *APIHandler) HandleCancelConfirm(c *gin.Context) {
	identifier := c.Param("identifier")

	// 检查是否有取消器
	if h.canceler == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "cancel_not_supported",
				"message": "cancel functionality not available",
			},
		})
		return
	}

	// 检查任务是否存在
	runningEntry := h.canceler.GetRunningEntryByIdentifier(identifier)
	retryEntry := h.canceler.GetRetryEntryByIdentifier(identifier)

	if runningEntry == nil && retryEntry == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "task_not_found",
				"message": "task not found in running or retry queue",
			},
		})
		return
	}

	// 返回确认提示
	response := gin.H{
		"identifier":       identifier,
		"current_state":    getStateDisplayName(runningEntry, retryEntry),
		"requires_confirm": true,
		"warning":          "取消操作不可逆，正在执行的 Agent 进程将被终止，子任务状态将同步更新为已取消",
		"actions": []gin.H{
			{
				"method":      "POST",
				"url":         "/api/v1/" + identifier + "/cancel",
				"description": "确认取消",
			},
		},
	}

	// 添加任务详情
	if runningEntry != nil {
		response["task_type"] = "running"
		response["started_at"] = runningEntry.StartedAt.Format("2006-01-02 15:04:05")
		if runningEntry.Issue != nil {
			response["title"] = runningEntry.Issue.Title
		}
		if runningEntry.Session != nil {
			response["session_id"] = runningEntry.Session.SessionID
			response["turn_count"] = runningEntry.TurnCount
		}
	} else if retryEntry != nil {
		response["task_type"] = "retrying"
		response["attempt"] = retryEntry.Attempt
		if retryEntry.Error != nil {
			response["last_error"] = *retryEntry.Error
		}
	}

	c.JSON(http.StatusOK, response)
}

// getStateDisplayName 获取状态显示名称
func getStateDisplayName(running *domain.RunningEntry, retry *domain.RetryEntry) string {
	if running != nil {
		if running.Issue != nil {
			return running.Issue.State
		}
		return "running"
	}
	if retry != nil {
		return "retrying"
	}
	return "unknown"
}

// HandleCreateTask 处理任务创建请求
// POST /api/tasks
func (h *APIHandler) HandleCreateTask(c *gin.Context) {
	var req TaskCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]string{
				"code":    "task.validation_failed",
				"message": "标题和描述为必填字段",
			},
		})
		return
	}

	// 验证字段不为空
	if req.Title == "" || req.Description == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]string{
				"code":    "task.validation_failed",
				"message": "标题和描述不能为空",
			},
		})
		return
	}

	ctx := context.Background()
	trackerClient := h.orchestrator.GetTracker()

	// 创建父任务
	parentTask, err := trackerClient.CreateTask(ctx, req.Title, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "task.create_failed",
				"message": "创建父任务失败: " + err.Error(),
			},
		})
		return
	}

	// 子阶段定义
	subTaskTitles := []string{
		"需求澄清",
		"BDD 审核",
		"架构审核",
		"实现",
		"验收",
	}

	subTaskDescriptions := []string{
		"收集和澄清需求细节，明确业务目标和用户期望",
		"审核行为驱动开发规范，确保测试覆盖关键场景",
		"审核技术架构设计，确保方案可行且符合最佳实践",
		"编写代码并完成功能实现，包括单元测试",
		"验证功能符合预期，确保需求完整交付",
	}

	// 创建子任务并建立依赖关系
	var subTasks []TaskInfo
	var previousIdentifier string

	for i, title := range subTaskTitles {
		// 构建子任务标题
		subTitle := fmt.Sprintf("%s-%d: %s", parentTask.Identifier, i+1, title)
		subDesc := subTaskDescriptions[i]

		// 构建阻塞关系（当前任务被前一个任务阻塞）
		var blockedBy []string
		if previousIdentifier != "" {
			blockedBy = []string{previousIdentifier}
		}

		// 创建子任务
		subTask, err := trackerClient.CreateSubTask(ctx, parentTask.Identifier, subTitle, subDesc, blockedBy)
		if err != nil {
			// 子任务创建失败，记录但继续
			continue
		}

		subTaskInfo := TaskInfo{
			ID:          subTask.ID,
			Identifier:  subTask.Identifier,
			Title:       subTask.Title,
			Description: subDesc,
			State:       subTask.State,
		}
		if subTask.URL != nil {
			subTaskInfo.URL = *subTask.URL
		}

		subTasks = append(subTasks, subTaskInfo)
		previousIdentifier = subTask.Identifier
	}

	// 构建父任务信息
	parentInfo := TaskInfo{
		ID:          parentTask.ID,
		Identifier:  parentTask.Identifier,
		Title:       parentTask.Title,
		Description: req.Description,
		State:       parentTask.State,
	}
	if parentTask.URL != nil {
		parentInfo.URL = *parentTask.URL
	}

	// 返回响应（支持 HTML 和 JSON）
	if c.GetHeader("Accept") == "text/html" || c.GetHeader("HX-Request") == "true" {
		// HTMX 请求返回 HTML
		html := RenderTaskCreatedHTML(parentInfo, subTasks)
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, html)
	} else {
		// JSON API 请求
		c.JSON(http.StatusOK, TaskCreateResponse{
			ParentTask:  &parentInfo,
			SubTasks:    subTasks,
			Message:     "需求创建成功，已生成 5 个子阶段任务",
		})
	}
}

// HandleSkipClarification 处理跳过澄清的请求
// POST /api/v1/:identifier/skip 或 POST /api/tasks/:identifier/skip
func (h *APIHandler) HandleSkipClarification(c *gin.Context) {
	identifier := c.Param("identifier")

	// 检查是否有澄清管理器
	if h.clarificationManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "clarification_not_supported",
				"message": "澄清管理功能不可用",
			},
		})
		return
	}

	// 检查是否可以跳过澄清
	canSkip, err := h.clarificationManager.CanSkipClarification(identifier)
	if err != nil {
		if err == workflow.ErrWorkflowNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": map[string]string{
					"code":    "task_not_found",
					"message": "任务未找到或未初始化工作流",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "clarification_check_failed",
				"message": err.Error(),
			},
		})
		return
	}

	if !canSkip {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]string{
				"code":    "cannot_skip_clarification",
				"message": "当前状态不允许跳过澄清（必须在澄清阶段进行中状态）",
			},
		})
		return
	}

	// 执行跳过澄清
	wf, err := h.clarificationManager.SkipClarification(identifier)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "skip_failed",
				"message": "跳过澄清失败: " + err.Error(),
			},
		})
		return
	}

	// 获取澄清状态
	status, err := h.clarificationManager.GetClarificationStatus(identifier)
	if err != nil {
		// 使用工作流信息构建响应
		c.JSON(http.StatusOK, gin.H{
			"success":           true,
			"identifier":        identifier,
			"previous_stage":    workflow.StageClarification,
			"current_stage":     wf.CurrentStage,
			"is_incomplete":     wf.IsIncomplete,
			"incomplete_reason": wf.IncompleteReason,
			"needs_attention":   wf.NeedsAttention,
			"message":           "已跳过澄清阶段，需求标记为不完整",
		})
		return
	}

	// 返回完整响应
	c.JSON(http.StatusOK, gin.H{
		"success":            true,
		"identifier":         identifier,
		"previous_stage":     workflow.StageClarification,
		"current_stage":      wf.CurrentStage,
		"is_incomplete":      wf.IsIncomplete,
		"incomplete_reason":  wf.IncompleteReason,
		"needs_attention":    wf.NeedsAttention,
		"clarification_round": status.CurrentRound,
		"max_rounds":         status.MaxRounds,
		"message":            "已跳过澄清阶段，需求标记为不完整",
	})
}

// HandleGetClarificationStatus 处理获取澄清状态的请求
// GET /api/v1/:identifier/clarification
func (h *APIHandler) HandleGetClarificationStatus(c *gin.Context) {
	identifier := c.Param("identifier")

	// 检查是否有澄清管理器
	if h.clarificationManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "clarification_not_supported",
				"message": "澄清管理功能不可用",
			},
		})
		return
	}

	// 获取澄清状态
	status, err := h.clarificationManager.GetClarificationStatus(identifier)
	if err != nil {
		if err == workflow.ErrWorkflowNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": map[string]string{
					"code":    "task_not_found",
					"message": "任务未找到或未初始化工作流",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "status_fetch_failed",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"task_id":             status.TaskID,
		"current_stage":       status.CurrentStage,
		"current_round":       status.CurrentRound,
		"max_rounds":          status.MaxRounds,
		"round_remaining":     status.RoundRemaining,
		"round_limit_reached": status.RoundLimitReached,
		"status":              status.Status,
		"is_incomplete":       status.IsIncomplete,
		"incomplete_reason":   status.IncompleteReason,
		"needs_attention":     status.NeedsAttention,
		"can_skip":            status.CurrentStage == workflow.StageClarification && status.Status == workflow.StatusInProgress && !status.IsIncomplete,
	})
}

// SubmitAnswerRequest 提交回答请求结构
type SubmitAnswerRequest struct {
	Answer string `json:"answer" binding:"required"`
}

// HandleSubmitAnswer 处理提交回答的请求
// POST /api/tasks/:identifier/answer
func (h *APIHandler) HandleSubmitAnswer(c *gin.Context) {
	identifier := c.Param("identifier")

	// 解析请求体
	var req SubmitAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]string{
				"code":    "answer.validation_failed",
				"message": "回答内容为必填字段",
			},
		})
		return
	}

	// 验证回答不为空
	if strings.TrimSpace(req.Answer) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]string{
				"code":    "answer.empty",
				"message": "回答内容不能为空",
			},
		})
		return
	}

	// 检查是否有澄清管理器
	if h.clarificationManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "clarification_not_supported",
				"message": "澄清管理功能不可用",
			},
		})
		return
	}

	// 获取任务信息以获取 taskID
	ctx := context.Background()
	trackerClient := h.orchestrator.GetTracker()
	task, err := trackerClient.GetTask(ctx, identifier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "task_not_found",
				"message": "任务未找到: " + err.Error(),
			},
		})
		return
	}

	// 提交回答
	result, err := h.clarificationManager.SubmitAnswer(ctx, task.ID, identifier, req.Answer)
	if err != nil {
		if err == workflow.ErrWorkflowNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": map[string]string{
					"code":    "workflow_not_found",
					"message": "任务工作流未初始化",
				},
			})
			return
		}
		if err == workflow.ErrInvalidStage || err == workflow.ErrInvalidTransition {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": map[string]string{
					"code":    "invalid_stage",
					"message": err.Error(),
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "submit_answer_failed",
				"message": "提交回答失败: " + err.Error(),
			},
		})
		return
	}

	// 构建响应
	response := gin.H{
		"success":                true,
		"identifier":             identifier,
		"needs_more_clarification": result.NeedsMoreClarification,
		"round":                  result.Round,
		"status":                 result.Status,
	}

	if result.NeedsMoreClarification {
		response["question"] = result.Question
		response["message"] = "回答已提交，请继续澄清"
	} else {
		response["summary"] = result.Summary
		response["message"] = "需求已明确，进入下一阶段"
		if result.Stage != nil {
			response["current_stage"] = result.Stage.Name
			response["stage_status"] = result.Stage.Status
		}
	}

	// 支持HTML响应（用于HTMX）
	if c.GetHeader("Accept") == "text/html" || c.GetHeader("HX-Request") == "true" {
		html := RenderAnswerSubmittedHTML(response)
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, html)
		return
	}

	c.JSON(http.StatusOK, response)
}

// HandleGetClarificationState 处理获取澄清状态的请求（包含最后一个问题）
// GET /api/tasks/:identifier/clarification
func (h *APIHandler) HandleGetClarificationState(c *gin.Context) {
	identifier := c.Param("identifier")

	// 检查是否有澄清管理器
	if h.clarificationManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "clarification_not_supported",
				"message": "澄清管理功能不可用",
			},
		})
		return
	}

	// 获取任务信息以获取 taskID
	ctx := context.Background()
	trackerClient := h.orchestrator.GetTracker()
	task, err := trackerClient.GetTask(ctx, identifier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "task_not_found",
				"message": "任务未找到: " + err.Error(),
			},
		})
		return
	}

	// 获取澄清状态
	result, err := h.clarificationManager.GetClarificationState(ctx, task.ID, identifier)
	if err != nil {
		if err == workflow.ErrWorkflowNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": map[string]string{
					"code":    "workflow_not_found",
					"message": "任务工作流未初始化",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "state_fetch_failed",
				"message": "获取澄清状态失败: " + err.Error(),
			},
		})
		return
	}

	response := gin.H{
		"identifier":               identifier,
		"needs_more_clarification": result.NeedsMoreClarification,
		"question":                 result.Question,
		"round":                    result.Round,
		"status":                   result.Status,
	}

	if result.Stage != nil {
		response["stage_name"] = result.Stage.Name
		response["stage_status"] = result.Stage.Status
	}

	c.JSON(http.StatusOK, response)
}

// RenderAnswerSubmittedHTML 渲染回答提交成功的 HTML
func RenderAnswerSubmittedHTML(response gin.H) string {
	return fmt.Sprintf(`
<div class="answer-result-card" style="background: linear-gradient(135deg, rgba(34, 197, 94, 0.1), rgba(34, 197, 94, 0.05)); border: 1px solid rgba(34, 197, 94, 0.3); border-radius: var(--radius-lg); padding: 1.5rem;">
    <div style="display: flex; align-items: center; gap: 0.75rem; margin-bottom: 1rem;">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: rgb(34, 197, 94);">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path>
            <polyline points="22 4 12 14.01 9 11.01"></polyline>
        </svg>
        <h3 style="font-size: 1.1rem; font-weight: 600; color: var(--ink-bright);">回答已提交</h3>
    </div>
    <div style="margin-bottom: 1rem;">
        <p style="color: var(--ink);">%s</p>
    </div>
    <div style="margin-top: 1rem;">
        <span style="color: var(--muted); font-size: 0.85rem;">轮次: %d</span>
    </div>
</div>`, response["message"], response["round"])
}

// HandleApproveBDD 处理通过 BDD 规则审核的请求
// POST /api/tasks/:identifier/bdd/approve
func (h *APIHandler) HandleApproveBDD(c *gin.Context) {
	identifier := c.Param("identifier")

	// 检查是否有 BDD 审核管理器
	if h.bddReviewManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "bdd_review_not_supported",
				"message": "BDD 审核管理功能不可用",
			},
		})
		return
	}

	// 获取任务信息以获取 taskID
	ctx := context.Background()
	trackerClient := h.orchestrator.GetTracker()
	task, err := trackerClient.GetTask(ctx, identifier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "task_not_found",
				"message": "任务未找到: " + err.Error(),
			},
		})
		return
	}

	// 检查是否可以进行审核操作
	canApprove, err := h.bddReviewManager.CanApproveOrReject(task.ID)
	if err != nil {
		if err == workflow.ErrWorkflowNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": map[string]string{
					"code":    "workflow_not_found",
					"message": "任务工作流未初始化",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "check_failed",
				"message": err.Error(),
			},
		})
		return
	}

	if !canApprove {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]string{
				"code":    "cannot_approve_bdd",
				"message": "当前状态不允许通过 BDD 审核（必须在 BDD 审核阶段进行中状态）",
			},
		})
		return
	}

	// 执行通过 BDD 审核
	wf, err := h.bddReviewManager.ApproveBDD(task.ID)
	if err != nil {
		if err == workflow.ErrWorkflowNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": map[string]string{
					"code":    "workflow_not_found",
					"message": "任务工作流未找到",
				},
			})
			return
		}
		if err == workflow.ErrInvalidStage || err == workflow.ErrInvalidTransition {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": map[string]string{
					"code":    "invalid_stage",
					"message": err.Error(),
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "approve_failed",
				"message": "通过 BDD 审核失败: " + err.Error(),
			},
		})
		return
	}

	// 更新 tracker 阶段状态
	_ = trackerClient.UpdateStage(ctx, identifier, domain.StageState{
		Name:   string(workflow.StageArchitectureReview),
		Status: "pending",
	})

	// 构建响应
	response := gin.H{
		"success":        true,
		"identifier":     identifier,
		"previous_stage": workflow.StageBDDReview,
		"current_stage":  wf.CurrentStage,
		"message":        "BDD 规则审核通过",
	}

	// 支持 HTML 响应（用于 HTMX）
	if c.GetHeader("Accept") == "text/html" || c.GetHeader("HX-Request") == "true" {
		html := RenderBDDApprovedHTML(response)
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, html)
		return
	}

	c.JSON(http.StatusOK, response)
}

// RejectBDDRequest 驳回 BDD 规则审核请求结构
type RejectBDDRequest struct {
	Reason string `json:"reason"` // 驳回原因（可选）
}

// HandleRejectBDD 处理驳回 BDD 规则审核的请求
// POST /api/tasks/:identifier/bdd/reject
func (h *APIHandler) HandleRejectBDD(c *gin.Context) {
	identifier := c.Param("identifier")

	// 解析请求体（可选）
	var req RejectBDDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 驳回原因可选，忽略解析错误
		req.Reason = ""
	}

	// 检查是否有 BDD 审核管理器
	if h.bddReviewManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "bdd_review_not_supported",
				"message": "BDD 审核管理功能不可用",
			},
		})
		return
	}

	// 获取任务信息以获取 taskID
	ctx := context.Background()
	trackerClient := h.orchestrator.GetTracker()
	task, err := trackerClient.GetTask(ctx, identifier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "task_not_found",
				"message": "任务未找到: " + err.Error(),
			},
		})
		return
	}

	// 检查是否可以进行审核操作
	canReject, err := h.bddReviewManager.CanApproveOrReject(task.ID)
	if err != nil {
		if err == workflow.ErrWorkflowNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": map[string]string{
					"code":    "workflow_not_found",
					"message": "任务工作流未初始化",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "check_failed",
				"message": err.Error(),
			},
		})
		return
	}

	if !canReject {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]string{
				"code":    "cannot_reject_bdd",
				"message": "当前状态不允许驳回 BDD 审核（必须在 BDD 审核阶段进行中状态）",
			},
		})
		return
	}

	// 设置默认驳回原因
	reason := req.Reason
	if reason == "" {
		reason = "BDD 规则不符合要求，需要重新生成"
	}

	// 执行驳回 BDD 审核
	wf, err := h.bddReviewManager.RejectBDD(task.ID, reason)
	if err != nil {
		if err == workflow.ErrWorkflowNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": map[string]string{
					"code":    "workflow_not_found",
					"message": "任务工作流未找到",
				},
			})
			return
		}
		if err == workflow.ErrInvalidStage || err == workflow.ErrInvalidTransition {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": map[string]string{
					"code":    "invalid_stage",
					"message": err.Error(),
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "reject_failed",
				"message": "驳回 BDD 审核失败: " + err.Error(),
			},
		})
		return
	}

	// 更新 tracker 阶段状态
	_ = trackerClient.UpdateStage(ctx, identifier, domain.StageState{
		Name:   string(workflow.StageClarification),
		Status: "in_progress",
	})

	// 构建响应
	response := gin.H{
		"success":        true,
		"identifier":     identifier,
		"previous_stage": workflow.StageBDDReview,
		"current_stage":  wf.CurrentStage,
		"reject_reason":  reason,
		"message":        "BDD 规则已驳回，需重新生成",
	}

	// 支持 HTML 响应（用于 HTMX）
	if c.GetHeader("Accept") == "text/html" || c.GetHeader("HX-Request") == "true" {
		html := RenderBDDRejectedHTML(response)
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, html)
		return
	}

	c.JSON(http.StatusOK, response)
}

// HandleGetBDDReviewStatus 处理获取 BDD 审核状态的请求
// GET /api/tasks/:identifier/bdd
func (h *APIHandler) HandleGetBDDReviewStatus(c *gin.Context) {
	identifier := c.Param("identifier")

	// 检查是否有 BDD 审核管理器
	if h.bddReviewManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "bdd_review_not_supported",
				"message": "BDD 审核管理功能不可用",
			},
		})
		return
	}

	// 获取任务信息以获取 taskID
	ctx := context.Background()
	trackerClient := h.orchestrator.GetTracker()
	task, err := trackerClient.GetTask(ctx, identifier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "task_not_found",
				"message": "任务未找到: " + err.Error(),
			},
		})
		return
	}

	// 获取 BDD 审核状态
	status, err := h.bddReviewManager.GetBDDReviewStatus(task.ID)
	if err != nil {
		if err == workflow.ErrWorkflowNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": map[string]string{
					"code":    "workflow_not_found",
					"message": "任务工作流未初始化",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "status_fetch_failed",
				"message": err.Error(),
			},
		})
		return
	}

	response := gin.H{
		"identifier":      identifier,
		"current_stage":   status.CurrentStage,
		"status":          status.Status,
		"can_approve":     status.CanApprove,
		"can_reject":      status.CanReject,
		"approved":        status.Approved,
		"rejected":        status.Rejected,
		"reject_reason":   status.RejectReason,
		"needs_attention": status.NeedsAttention,
	}

	c.JSON(http.StatusOK, response)
}

// RenderBDDApprovedHTML 渲染 BDD 审核通过的 HTML
func RenderBDDApprovedHTML(response gin.H) string {
	return fmt.Sprintf(`
<div class="bdd-approved-card" style="background: linear-gradient(135deg, rgba(34, 197, 94, 0.1), rgba(34, 197, 94, 0.05)); border: 1px solid rgba(34, 197, 94, 0.3); border-radius: var(--radius-lg); padding: 1.5rem;">
    <div style="display: flex; align-items: center; gap: 0.75rem; margin-bottom: 1rem;">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: rgb(34, 197, 94);">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path>
            <polyline points="22 4 12 14.01 9 11.01"></polyline>
        </svg>
        <h3 style="font-size: 1.1rem; font-weight: 600; color: var(--ink-bright);">BDD 规则审核通过</h3>
    </div>
    <div style="margin-bottom: 1rem;">
        <p style="color: var(--ink);">%s</p>
    </div>
    <div style="margin-top: 1rem;">
        <span style="color: var(--muted); font-size: 0.85rem;">当前阶段: %s</span>
    </div>
</div>`, response["message"], response["current_stage"])
}

// RenderBDDRejectedHTML 渲染 BDD 审核驳回的 HTML
func RenderBDDRejectedHTML(response gin.H) string {
	return fmt.Sprintf(`
<div class="bdd-rejected-card" style="background: linear-gradient(135deg, rgba(239, 68, 68, 0.1), rgba(239, 68, 68, 0.05)); border: 1px solid rgba(239, 68, 68, 0.3); border-radius: var(--radius-lg); padding: 1.5rem;">
    <div style="display: flex; align-items: center; gap: 0.75rem; margin-bottom: 1rem;">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: rgb(239, 68, 68);">
            <circle cx="12" cy="12" r="10"></circle>
            <line x1="15" y1="9" x2="9" y2="15"></line>
            <line x1="9" y1="9" x2="15" y2="15"></line>
        </svg>
        <h3 style="font-size: 1.1rem; font-weight: 600; color: var(--ink-bright);">BDD 规则已驳回</h3>
    </div>
    <div style="margin-bottom: 1rem;">
        <p style="color: var(--ink);">%s</p>
    </div>
    <div style="margin-top: 1rem;">
        <span style="color: var(--muted); font-size: 0.85rem;">驳回原因: %s</span>
    </div>
</div>`, response["message"], response["reject_reason"])
}

// ========== Epic 5: 架构设计与 TDD 规则审核 ==========

// HandleApproveArchitecture 处理通过架构设计审核的请求
// POST /api/tasks/:identifier/architecture/approve
func (h *APIHandler) HandleApproveArchitecture(c *gin.Context) {
	identifier := c.Param("identifier")

	// 检查是否有架构审核管理器
	if h.architectureReviewManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "architecture_review_not_supported",
				"message": "架构审核管理功能不可用",
			},
		})
		return
	}

	// 获取任务信息以获取 taskID
	ctx := context.Background()
	trackerClient := h.orchestrator.GetTracker()
	task, err := trackerClient.GetTask(ctx, identifier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "task_not_found",
				"message": "任务未找到: " + err.Error(),
			},
		})
		return
	}

	// 检查是否可以进行审核操作
	canApprove, err := h.architectureReviewManager.CanApproveOrRejectArchitecture(task.ID)
	if err != nil {
		if err == workflow.ErrWorkflowNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": map[string]string{
					"code":    "workflow_not_found",
					"message": "任务工作流未初始化",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "check_failed",
				"message": err.Error(),
			},
		})
		return
	}

	if !canApprove {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]string{
				"code":    "cannot_approve_architecture",
				"message": "当前状态不允许通过架构审核（必须在架构审核阶段进行中状态）",
			},
		})
		return
	}

	// 执行通过架构审核
	wf, err := h.architectureReviewManager.ApproveArchitecture(task.ID)
	if err != nil {
		if err == workflow.ErrWorkflowNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": map[string]string{
					"code":    "workflow_not_found",
					"message": "任务工作流未找到",
				},
			})
			return
		}
		if err == workflow.ErrInvalidStage || err == workflow.ErrInvalidTransition {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": map[string]string{
					"code":    "invalid_stage",
					"message": err.Error(),
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "approve_failed",
				"message": "通过架构审核失败: " + err.Error(),
			},
		})
		return
	}

	// 更新 tracker 阶段状态
	_ = trackerClient.UpdateStage(ctx, identifier, domain.StageState{
		Name:   string(workflow.StageImplementation),
		Status: "pending",
	})

	// 更新 tracker 架构审核状态
	_ = trackerClient.ApproveArchitecture(ctx, identifier)

	// 构建响应
	response := gin.H{
		"success":        true,
		"identifier":     identifier,
		"previous_stage": workflow.StageArchitectureReview,
		"current_stage":  wf.CurrentStage,
		"message":        "架构设计审核通过",
	}

	// 支持 HTML 响应（用于 HTMX）
	if c.GetHeader("Accept") == "text/html" || c.GetHeader("HX-Request") == "true" {
		html := RenderArchitectureApprovedHTML(response)
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, html)
		return
	}

	c.JSON(http.StatusOK, response)
}

// RejectArchitectureRequest 驳回架构设计审核请求结构
type RejectArchitectureRequest struct {
	Reason string `json:"reason"` // 驳回原因（可选）
}

// HandleRejectArchitecture 处理驳回架构设计审核的请求
// POST /api/tasks/:identifier/architecture/reject
func (h *APIHandler) HandleRejectArchitecture(c *gin.Context) {
	identifier := c.Param("identifier")

	// 解析请求体（可选）
	var req RejectArchitectureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 驳回原因可选，忽略解析错误
		req.Reason = ""
	}

	// 检查是否有架构审核管理器
	if h.architectureReviewManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "architecture_review_not_supported",
				"message": "架构审核管理功能不可用",
			},
		})
		return
	}

	// 获取任务信息以获取 taskID
	ctx := context.Background()
	trackerClient := h.orchestrator.GetTracker()
	task, err := trackerClient.GetTask(ctx, identifier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "task_not_found",
				"message": "任务未找到: " + err.Error(),
			},
		})
		return
	}

	// 检查是否可以进行审核操作
	canReject, err := h.architectureReviewManager.CanApproveOrRejectArchitecture(task.ID)
	if err != nil {
		if err == workflow.ErrWorkflowNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": map[string]string{
					"code":    "workflow_not_found",
					"message": "任务工作流未初始化",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "check_failed",
				"message": err.Error(),
			},
		})
		return
	}

	if !canReject {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]string{
				"code":    "cannot_reject_architecture",
				"message": "当前状态不允许驳回架构审核（必须在架构审核阶段进行中状态）",
			},
		})
		return
	}

	// 设置默认驳回原因
	reason := req.Reason
	if reason == "" {
		reason = "架构设计不符合要求，需要重新生成"
	}

	// 执行驳回架构审核
	wf, err := h.architectureReviewManager.RejectArchitecture(task.ID, reason)
	if err != nil {
		if err == workflow.ErrWorkflowNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": map[string]string{
					"code":    "workflow_not_found",
					"message": "任务工作流未找到",
				},
			})
			return
		}
		if err == workflow.ErrInvalidStage || err == workflow.ErrInvalidTransition {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": map[string]string{
					"code":    "invalid_stage",
					"message": err.Error(),
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "reject_failed",
				"message": "驳回架构审核失败: " + err.Error(),
			},
		})
		return
	}

	// 更新 tracker 阶段状态
	_ = trackerClient.UpdateStage(ctx, identifier, domain.StageState{
		Name:   string(workflow.StageClarification),
		Status: "in_progress",
	})

	// 更新 tracker 架构审核状态
	_ = trackerClient.RejectArchitecture(ctx, identifier, reason)

	// 构建响应
	response := gin.H{
		"success":        true,
		"identifier":     identifier,
		"previous_stage": workflow.StageArchitectureReview,
		"current_stage":  wf.CurrentStage,
		"reject_reason":  reason,
		"message":        "架构设计已驳回，需重新生成",
	}

	// 支持 HTML 响应（用于 HTMX）
	if c.GetHeader("Accept") == "text/html" || c.GetHeader("HX-Request") == "true" {
		html := RenderArchitectureRejectedHTML(response)
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, html)
		return
	}

	c.JSON(http.StatusOK, response)
}

// HandleGetArchitectureReviewStatus 处理获取架构审核状态的请求
// GET /api/tasks/:identifier/architecture
func (h *APIHandler) HandleGetArchitectureReviewStatus(c *gin.Context) {
	identifier := c.Param("identifier")

	// 检查是否有架构审核管理器
	if h.architectureReviewManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "architecture_review_not_supported",
				"message": "架构审核管理功能不可用",
			},
		})
		return
	}

	// 获取任务信息以获取 taskID
	ctx := context.Background()
	trackerClient := h.orchestrator.GetTracker()
	task, err := trackerClient.GetTask(ctx, identifier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "task_not_found",
				"message": "任务未找到: " + err.Error(),
			},
		})
		return
	}

	// 获取架构审核状态
	status, err := h.architectureReviewManager.GetArchitectureReviewStatus(task.ID)
	if err != nil {
		if err == workflow.ErrWorkflowNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": map[string]string{
					"code":    "workflow_not_found",
					"message": "任务工作流未初始化",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "status_fetch_failed",
				"message": err.Error(),
			},
		})
		return
	}

	response := gin.H{
		"identifier":      identifier,
		"current_stage":   status.CurrentStage,
		"status":          status.Status,
		"can_approve":     status.CanApprove,
		"can_reject":      status.CanReject,
		"approved":        status.Approved,
		"rejected":        status.Rejected,
		"reject_reason":   status.RejectReason,
		"needs_attention": status.NeedsAttention,
	}

	c.JSON(http.StatusOK, response)
}

// RenderArchitectureApprovedHTML 渲染架构审核通过的 HTML
func RenderArchitectureApprovedHTML(response gin.H) string {
	return fmt.Sprintf(`
<div class="architecture-approved-card" style="background: linear-gradient(135deg, rgba(34, 197, 94, 0.1), rgba(34, 197, 94, 0.05)); border: 1px solid rgba(34, 197, 94, 0.3); border-radius: var(--radius-lg); padding: 1.5rem;">
    <div style="display: flex; align-items: center; gap: 0.75rem; margin-bottom: 1rem;">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: rgb(34, 197, 94);">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path>
            <polyline points="22 4 12 14.01 9 11.01"></polyline>
        </svg>
        <h3 style="font-size: 1.1rem; font-weight: 600; color: var(--ink-bright);">架构设计审核通过</h3>
    </div>
    <div style="margin-bottom: 1rem;">
        <p style="color: var(--ink);">%s</p>
    </div>
    <div style="margin-top: 1rem;">
        <span style="color: var(--muted); font-size: 0.85rem;">当前阶段: %s</span>
    </div>
</div>`, response["message"], response["current_stage"])
}

// RenderArchitectureRejectedHTML 渲染架构审核驳回的 HTML
func RenderArchitectureRejectedHTML(response gin.H) string {
	return fmt.Sprintf(`
<div class="architecture-rejected-card" style="background: linear-gradient(135deg, rgba(239, 68, 68, 0.1), rgba(239, 68, 68, 0.05)); border: 1px solid rgba(239, 68, 68, 0.3); border-radius: var(--radius-lg); padding: 1.5rem;">
    <div style="display: flex; align-items: center; gap: 0.75rem; margin-bottom: 1rem;">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: rgb(239, 68, 68);">
            <circle cx="12" cy="12" r="10"></circle>
            <line x1="15" y1="9" x2="9" y2="15"></line>
            <line x1="9" y1="9" x2="15" y2="15"></line>
        </svg>
        <h3 style="font-size: 1.1rem; font-weight: 600; color: var(--ink-bright);">架构设计已驳回</h3>
    </div>
    <div style="margin-bottom: 1rem;">
        <p style="color: var(--ink);">%s</p>
    </div>
    <div style="margin-top: 1rem;">
        <span style="color: var(--muted); font-size: 0.85rem;">驳回原因: %s</span>
    </div>
</div>`, response["message"], response["reject_reason"])
}

// HandleApproveVerification 处理通过验收的请求
// POST /api/tasks/:identifier/verification/approve
func (h *APIHandler) HandleApproveVerification(c *gin.Context) {
	identifier := c.Param("identifier")

	// 检查是否有验收审核管理器
	if h.verificationManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "verification_not_supported",
				"message": "验收审核管理功能不可用",
			},
		})
		return
	}

	// 获取任务信息以获取 taskID
	ctx := context.Background()
	trackerClient := h.orchestrator.GetTracker()
	task, err := trackerClient.GetTask(ctx, identifier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "task_not_found",
				"message": "任务未找到: " + err.Error(),
			},
		})
		return
	}

	// 检查是否可以进行审核操作
	canApprove, err := h.verificationManager.CanApproveOrRejectVerification(task.ID)
	if err != nil {
		if err == workflow.ErrWorkflowNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": map[string]string{
					"code":    "workflow_not_found",
					"message": "任务工作流未初始化",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "check_failed",
				"message": err.Error(),
			},
		})
		return
	}

	if !canApprove {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]string{
				"code":    "cannot_approve_verification",
				"message": "当前状态不允许通过验收（必须在验收阶段进行中状态）",
			},
		})
		return
	}

	// 执行通过验收
	wf, err := h.verificationManager.ApproveVerification(task.ID)
	if err != nil {
		if err == workflow.ErrWorkflowNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": map[string]string{
					"code":    "workflow_not_found",
					"message": "任务工作流未找到",
				},
			})
			return
		}
		if err == workflow.ErrInvalidStage || err == workflow.ErrInvalidTransition {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": map[string]string{
					"code":    "invalid_stage",
					"message": err.Error(),
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "approve_failed",
				"message": "通过验收失败: " + err.Error(),
			},
		})
		return
	}

	// 更新 tracker 状态
	_ = trackerClient.ApproveVerification(ctx, identifier)

	// 执行 Git 提交
	var commitHash string
	if h.gitCommitter != nil {
		workspacePath := h.gitCommitter.GetWorkspacePath(identifier)
		title := ""
		if task.Title != "" {
			title = task.Title
		}
		hash, err := h.gitCommitter.GitCommit(ctx, workspacePath, identifier, title)
		if err != nil {
			// Git 提交失败不影响任务完成状态，只记录
			fmt.Printf("warning: git commit failed: %v\n", err)
		} else if hash != "" {
			commitHash = hash
		}
	}

	// 构建响应
	response := gin.H{
		"success":        true,
		"identifier":     identifier,
		"previous_stage": workflow.StageVerification,
		"current_stage":  wf.CurrentStage,
		"message":        "验收通过，任务已完成",
	}

	if commitHash != "" {
		response["commit_hash"] = commitHash
	}

	// 支持 HTML 响应（用于 HTMX）
	if c.GetHeader("Accept") == "text/html" || c.GetHeader("HX-Request") == "true" {
		html := RenderVerificationApprovedHTML(response)
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, html)
		return
	}

	c.JSON(http.StatusOK, response)
}

// RejectVerificationRequest 驳回验收请求结构
type RejectVerificationRequest struct {
	Reason string `json:"reason"` // 驳回原因（可选）
}

// HandleRejectVerification 处理驳回验收的请求
// POST /api/tasks/:identifier/verification/reject
func (h *APIHandler) HandleRejectVerification(c *gin.Context) {
	identifier := c.Param("identifier")

	// 解析请求体（可选）
	var req RejectVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 驳回原因可选，忽略解析错误
		req.Reason = ""
	}

	// 检查是否有验收审核管理器
	if h.verificationManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "verification_not_supported",
				"message": "验收审核管理功能不可用",
			},
		})
		return
	}

	// 获取任务信息以获取 taskID
	ctx := context.Background()
	trackerClient := h.orchestrator.GetTracker()
	task, err := trackerClient.GetTask(ctx, identifier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "task_not_found",
				"message": "任务未找到: " + err.Error(),
			},
		})
		return
	}

	// 检查是否可以进行审核操作
	canReject, err := h.verificationManager.CanApproveOrRejectVerification(task.ID)
	if err != nil {
		if err == workflow.ErrWorkflowNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": map[string]string{
					"code":    "workflow_not_found",
					"message": "任务工作流未初始化",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "check_failed",
				"message": err.Error(),
			},
		})
		return
	}

	if !canReject {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]string{
				"code":    "cannot_reject_verification",
				"message": "当前状态不允许驳回验收（必须在验收阶段进行中状态）",
			},
		})
		return
	}

	// 设置默认驳回原因
	reason := req.Reason
	if reason == "" {
		reason = "验收不通过，需要重新实现"
	}

	// 执行驳回验收
	wf, err := h.verificationManager.RejectVerification(task.ID, reason)
	if err != nil {
		if err == workflow.ErrWorkflowNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": map[string]string{
					"code":    "workflow_not_found",
					"message": "任务工作流未找到",
				},
			})
			return
		}
		if err == workflow.ErrInvalidStage || err == workflow.ErrInvalidTransition {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": map[string]string{
					"code":    "invalid_stage",
					"message": err.Error(),
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "reject_failed",
				"message": "驳回验收失败: " + err.Error(),
			},
		})
		return
	}

	// 更新 tracker 状态
	_ = trackerClient.RejectVerification(ctx, identifier, reason)

	// 构建响应
	response := gin.H{
		"success":        true,
		"identifier":     identifier,
		"previous_stage": workflow.StageVerification,
		"current_stage":  wf.CurrentStage,
		"reject_reason":  reason,
		"message":        "验收已驳回，任务流转回实现中",
	}

	// 支持 HTML 响应（用于 HTMX）
	if c.GetHeader("Accept") == "text/html" || c.GetHeader("HX-Request") == "true" {
		html := RenderVerificationRejectedHTML(response)
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, html)
		return
	}

	c.JSON(http.StatusOK, response)
}

// HandleGetVerificationStatus 处理获取验收状态的请求
// GET /api/tasks/:identifier/verification
func (h *APIHandler) HandleGetVerificationStatus(c *gin.Context) {
	identifier := c.Param("identifier")

	// 检查是否有验收审核管理器
	if h.verificationManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "verification_not_supported",
				"message": "验收审核管理功能不可用",
			},
		})
		return
	}

	// 获取任务信息以获取 taskID
	ctx := context.Background()
	trackerClient := h.orchestrator.GetTracker()
	task, err := trackerClient.GetTask(ctx, identifier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "task_not_found",
				"message": "任务未找到: " + err.Error(),
			},
		})
		return
	}

	// 获取验收状态
	status, err := h.verificationManager.GetVerificationStatus(task.ID)
	if err != nil {
		if err == workflow.ErrWorkflowNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": map[string]string{
					"code":    "workflow_not_found",
					"message": "任务工作流未初始化",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "status_fetch_failed",
				"message": err.Error(),
			},
		})
		return
	}

	// 获取验收报告
	report, err := trackerClient.GetVerificationReport(ctx, identifier)
	if err != nil {
		// 获取失败不影响状态返回
		report = nil
	}

	response := gin.H{
		"identifier":     identifier,
		"current_stage":  status.CurrentStage,
		"status":         status.Status,
		"can_approve":    status.CanApprove,
		"can_reject":     status.CanReject,
		"approved":       status.Approved,
		"rejected":       status.Rejected,
		"reject_reason":  status.RejectReason,
	}

	// 添加验收报告数据
	if report != nil {
		response["report"] = gin.H{
			"task_identifier":     report.TaskIdentifier,
			"task_title":          report.TaskTitle,
			"generated_at":        report.GeneratedAt,
			"overall_status":      report.OverallStatus,
			"test_results":        report.TestResults,
			"bdd_validation":      report.BDDValidation,
			"implementation_summary": report.ImplementationSummary,
			"recommendations":     report.Recommendations,
			"raw_content":         report.RawContent,
		}
	}

	c.JSON(http.StatusOK, response)
}

// RenderVerificationApprovedHTML 渲染验收通过的 HTML
func RenderVerificationApprovedHTML(response gin.H) string {
	commitHashInfo := ""
	if hash, ok := response["commit_hash"].(string); ok && hash != "" {
		commitHashInfo = fmt.Sprintf(`<div style="margin-top: 0.5rem;"><span style="color: var(--muted); font-size: 0.85rem;">Commit: <code style="background: var(--surface); padding: 0.15rem 0.3rem; border-radius: var(--radius-sm);">%s</code></span></div>`, hash[:8])
	}

	return fmt.Sprintf(`
<div class="verification-approved-card" style="background: linear-gradient(135deg, rgba(34, 197, 94, 0.1), rgba(34, 197, 94, 0.05)); border: 1px solid rgba(34, 197, 94, 0.3); border-radius: var(--radius-lg); padding: 1.5rem;">
    <div style="display: flex; align-items: center; gap: 0.75rem; margin-bottom: 1rem;">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: rgb(34, 197, 94);">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path>
            <polyline points="22 4 12 14.01 9 11.01"></polyline>
        </svg>
        <h3 style="font-size: 1.1rem; font-weight: 600; color: var(--ink-bright);">验收通过</h3>
    </div>
    <div style="margin-bottom: 1rem;">
        <p style="color: var(--ink);">%s</p>
    </div>
    %s
    <div style="margin-top: 1rem;">
        <span style="color: var(--muted); font-size: 0.85rem;">任务已完成</span>
    </div>
</div>`, response["message"], commitHashInfo)
}

// RenderVerificationRejectedHTML 渲染验收驳回的 HTML
func RenderVerificationRejectedHTML(response gin.H) string {
	return fmt.Sprintf(`
<div class="verification-rejected-card" style="background: linear-gradient(135deg, rgba(239, 68, 68, 0.1), rgba(239, 68, 68, 0.05)); border: 1px solid rgba(239, 68, 68, 0.3); border-radius: var(--radius-lg); padding: 1.5rem;">
    <div style="display: flex; align-items: center; gap: 0.75rem; margin-bottom: 1rem;">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: rgb(239, 68, 68);">
            <circle cx="12" cy="12" r="10"></circle>
            <line x1="15" y1="9" x2="9" y2="15"></line>
            <line x1="9" y1="9" x2="15" y2="15"></line>
        </svg>
        <h3 style="font-size: 1.1rem; font-weight: 600; color: var(--ink-bright);">验收已驳回</h3>
    </div>
    <div style="margin-bottom: 1rem;">
        <p style="color: var(--ink);">%s</p>
    </div>
    <div style="margin-top: 1rem;">
        <span style="color: var(--muted); font-size: 0.85rem;">驳回原因: %s</span>
    </div>
    <div style="margin-top: 0.5rem;">
        <span style="color: var(--muted); font-size: 0.85rem;">当前阶段: 实现中</span>
    </div>
</div>`, response["message"], response["reject_reason"])
}

// ========== Epic 8: 异常处理与人工干预 ==========

// HandleGetNeedsAttentionStatus 处理获取待人工处理状态的请求
// GET /api/tasks/:identifier/needs-attention
func (h *APIHandler) HandleGetNeedsAttentionStatus(c *gin.Context) {
	identifier := c.Param("identifier")

	// 检查是否有人工干预管理器
	if h.needsAttentionManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "needs_attention_not_supported",
				"message": "人工干预管理功能不可用",
			},
		})
		return
	}

	// 获取任务信息以获取 taskID
	ctx := context.Background()
	trackerClient := h.orchestrator.GetTracker()
	task, err := trackerClient.GetTask(ctx, identifier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "task_not_found",
				"message": "任务未找到: " + err.Error(),
			},
		})
		return
	}

	// 获取失败详情
	details, err := h.needsAttentionManager.GetFailureDetails(task.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "details_fetch_failed",
				"message": "获取失败详情失败: " + err.Error(),
			},
		})
		return
	}

	response := gin.H{
		"identifier":       identifier,
		"failed_stage":     details.FailedStage,
		"failed_at":        details.FailedAt,
		"retry_count":      details.RetryCount,
		"max_retries":      details.MaxRetries,
		"error_type":       details.ErrorType,
		"error_message":    details.ErrorMessage,
		"last_log_snippet": details.LastLogSnippet,
		"suggestion":       details.Suggestion,
		"can_resume":       true,
		"can_reclarify":    true,
		"can_abandon":      true,
	}

	c.JSON(http.StatusOK, response)
}

// HandleResumeTask 处理继续执行任务的请求
// POST /api/tasks/:identifier/resume
func (h *APIHandler) HandleResumeTask(c *gin.Context) {
	identifier := c.Param("identifier")

	// 检查是否有人工干预管理器
	if h.needsAttentionManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "needs_attention_not_supported",
				"message": "人工干预管理功能不可用",
			},
		})
		return
	}

	// 获取任务信息以获取 taskID
	ctx := context.Background()
	trackerClient := h.orchestrator.GetTracker()
	task, err := trackerClient.GetTask(ctx, identifier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "task_not_found",
				"message": "任务未找到: " + err.Error(),
			},
		})
		return
	}

	// 恢复任务执行
	wf, err := h.needsAttentionManager.ResumeTask(task.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "resume_failed",
				"message": "恢复任务执行失败: " + err.Error(),
			},
		})
		return
	}

	// 更新 tracker 阶段状态
	_ = trackerClient.UpdateStage(ctx, identifier, domain.StageState{
		Name:   string(wf.CurrentStage),
		Status: "in_progress",
	})

	response := gin.H{
		"success":        true,
		"identifier":     identifier,
		"current_stage":  wf.CurrentStage,
		"message":        "任务已恢复执行，重试计数器已重置",
		"retry_count":    wf.RetryCount,
	}

	// 支持 HTML 响应（用于 HTMX）
	if c.GetHeader("Accept") == "text/html" || c.GetHeader("HX-Request") == "true" {
		html := RenderTaskResumedHTML(response)
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, html)
		return
	}

	c.JSON(http.StatusOK, response)
}

// HandleReclarifyTask 处理重新澄清需求的请求
// POST /api/tasks/:identifier/reclarify
func (h *APIHandler) HandleReclarifyTask(c *gin.Context) {
	identifier := c.Param("identifier")

	// 检查是否有人工干预管理器
	if h.needsAttentionManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "needs_attention_not_supported",
				"message": "人工干预管理功能不可用",
			},
		})
		return
	}

	// 获取任务信息以获取 taskID
	ctx := context.Background()
	trackerClient := h.orchestrator.GetTracker()
	task, err := trackerClient.GetTask(ctx, identifier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "task_not_found",
				"message": "任务未找到: " + err.Error(),
			},
		})
		return
	}

	// 重新澄清需求
	wf, err := h.needsAttentionManager.ReclarifyTask(task.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "reclarify_failed",
				"message": "重新澄清需求失败: " + err.Error(),
			},
		})
		return
	}

	// 更新 tracker 阶段状态
	_ = trackerClient.UpdateStage(ctx, identifier, domain.StageState{
		Name:   string(workflow.StageClarification),
		Status: "in_progress",
	})

	response := gin.H{
		"success":        true,
		"identifier":     identifier,
		"current_stage":  wf.CurrentStage,
		"message":        "任务已流转到需求澄清阶段，BDD 和架构设计已清除",
	}

	// 支持 HTML 响应（用于 HTMX）
	if c.GetHeader("Accept") == "text/html" || c.GetHeader("HX-Request") == "true" {
		html := RenderTaskReclarifiedHTML(response)
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, html)
		return
	}

	c.JSON(http.StatusOK, response)
}

// HandleAbandonTask 处理放弃任务的请求
// POST /api/tasks/:identifier/abandon
func (h *APIHandler) HandleAbandonTask(c *gin.Context) {
	identifier := c.Param("identifier")

	// 检查是否有人工干预管理器
	if h.needsAttentionManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "needs_attention_not_supported",
				"message": "人工干预管理功能不可用",
			},
		})
		return
	}

	// 获取任务信息以获取 taskID
	ctx := context.Background()
	trackerClient := h.orchestrator.GetTracker()
	task, err := trackerClient.GetTask(ctx, identifier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "task_not_found",
				"message": "任务未找到: " + err.Error(),
			},
		})
		return
	}

	// 放弃任务
	wf, err := h.needsAttentionManager.AbandonTask(task.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"code":    "abandon_failed",
				"message": "放弃任务失败: " + err.Error(),
			},
		})
		return
	}

	// 更新 tracker 阶段状态
	_ = trackerClient.UpdateStage(ctx, identifier, domain.StageState{
		Name:   string(workflow.StageCancelled),
		Status: "completed",
	})

	// 清理工作空间
	if h.workspaceCleaner != nil {
		wsPath := h.workspaceCleaner.GetWorkspacePath(identifier)
		if wsPath != "" {
			_ = h.workspaceCleaner.RemoveWorkspace(ctx, wsPath)
		}
	}

	response := gin.H{
		"success":        true,
		"identifier":     identifier,
		"current_stage":  wf.CurrentStage,
		"message":        "任务已取消，相关工作空间已清理",
	}

	// 支持 HTML 响应（用于 HTMX）
	if c.GetHeader("Accept") == "text/html" || c.GetHeader("HX-Request") == "true" {
		html := RenderTaskAbandonedHTML(response)
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, html)
		return
	}

	c.JSON(http.StatusOK, response)
}

// HandleAbandonConfirm 处理放弃任务确认请求
// GET /api/tasks/:identifier/abandon/confirm
func (h *APIHandler) HandleAbandonConfirm(c *gin.Context) {
	identifier := c.Param("identifier")

	// 获取任务信息
	ctx := context.Background()
	trackerClient := h.orchestrator.GetTracker()
	task, err := trackerClient.GetTask(ctx, identifier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "task_not_found",
				"message": "任务未找到: " + err.Error(),
			},
		})
		return
	}

	response := gin.H{
		"identifier":       identifier,
		"title":            task.Title,
		"requires_confirm": true,
		"warning":          "放弃操作不可逆，相关工作空间将被清理，任务状态将变为已取消",
		"actions": []gin.H{
			{
				"method":      "POST",
				"url":         "/api/tasks/" + identifier + "/abandon",
				"description": "确认放弃",
			},
		},
	}

	c.JSON(http.StatusOK, response)
}

// RenderTaskResumedHTML 渲染任务恢复执行的 HTML
func RenderTaskResumedHTML(response gin.H) string {
	return fmt.Sprintf(`
<div class="task-resumed-card" style="background: linear-gradient(135deg, rgba(34, 197, 94, 0.1), rgba(34, 197, 94, 0.05)); border: 1px solid rgba(34, 197, 94, 0.3); border-radius: var(--radius-lg); padding: 1.5rem;">
    <div style="display: flex; align-items: center; gap: 0.75rem; margin-bottom: 1rem;">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: rgb(34, 197, 94);">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path>
            <polyline points="22 4 12 14.01 9 11.01"></polyline>
        </svg>
        <h3 style="font-size: 1.1rem; font-weight: 600; color: var(--ink-bright);">任务已恢复执行</h3>
    </div>
    <div style="margin-bottom: 1rem;">
        <p style="color: var(--ink);">%s</p>
    </div>
    <div style="margin-top: 1rem;">
        <span style="color: var(--muted); font-size: 0.85rem;">当前阶段: %s</span>
    </div>
    <div style="margin-top: 0.5rem;">
        <span style="color: var(--muted); font-size: 0.85rem;">重试计数: %d</span>
    </div>
</div>`, response["message"], response["current_stage"], response["retry_count"])
}

// RenderTaskReclarifiedHTML 渲染任务重新澄清的 HTML
func RenderTaskReclarifiedHTML(response gin.H) string {
	return fmt.Sprintf(`
<div class="task-reclarified-card" style="background: linear-gradient(135deg, rgba(139, 92, 246, 0.1), rgba(139, 92, 246, 0.05)); border: 1px solid rgba(139, 92, 246, 0.3); border-radius: var(--radius-lg); padding: 1.5rem;">
    <div style="display: flex; align-items: center; gap: 0.75rem; margin-bottom: 1rem;">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: rgb(139, 92, 246);">
            <circle cx="12" cy="12" r="10"></circle>
            <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"></path>
            <line x1="12" y1="17" x2="12.01" y2="17"></line>
        </svg>
        <h3 style="font-size: 1.1rem; font-weight: 600; color: var(--ink-bright);">任务已重新进入需求澄清</h3>
    </div>
    <div style="margin-bottom: 1rem;">
        <p style="color: var(--ink);">%s</p>
    </div>
    <div style="margin-top: 1rem;">
        <span style="color: var(--muted); font-size: 0.85rem;">当前阶段: 需求澄清</span>
    </div>
</div>`, response["message"])
}

// RenderTaskAbandonedHTML 渲染任务放弃的 HTML
func RenderTaskAbandonedHTML(response gin.H) string {
	return fmt.Sprintf(`
<div class="task-abandoned-card" style="background: linear-gradient(135deg, rgba(156, 163, 175, 0.1), rgba(156, 163, 175, 0.05)); border: 1px solid rgba(156, 163, 175, 0.3); border-radius: var(--radius-lg); padding: 1.5rem;">
    <div style="display: flex; align-items: center; gap: 0.75rem; margin-bottom: 1rem;">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: rgb(156, 163, 175);">
            <circle cx="12" cy="12" r="10"></circle>
            <line x1="15" y1="9" x2="9" y2="15"></line>
            <line x1="9" y1="9" x2="15" y2="15"></line>
        </svg>
        <h3 style="font-size: 1.1rem; font-weight: 600; color: var(--ink-bright);">任务已取消</h3>
    </div>
    <div style="margin-bottom: 1rem;">
        <p style="color: var(--ink);">%s</p>
    </div>
    <div style="margin-top: 1rem;">
        <span style="color: var(--muted); font-size: 0.85rem;">工作空间已清理</span>
    </div>
</div>`, response["message"])
}
