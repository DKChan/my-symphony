package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/dministrator/symphony/internal/common"
	"github.com/dministrator/symphony/internal/domain"
	"github.com/dministrator/symphony/internal/server/components"
	"github.com/dministrator/symphony/internal/tracker"
	"github.com/dministrator/symphony/internal/workflow"
	"github.com/gin-gonic/gin"
)

// ExecutionManager 定义执行管理接口
type ExecutionManager interface {
	GetExecutionProgress(taskID string) (*workflow.ExecutionProgress, error)
	GetExecutionLogs(taskID string, page, pageSize int) ([]workflow.ExecutionLog, int, error)
	GetAllExecutionLogs(taskID string) ([]workflow.ExecutionLog, error)
	GetImplementationStatus(taskID string) (*workflow.ImplementationStatus, error)
}

// ExecutionHandler 执行相关 API 处理器
type ExecutionHandler struct {
	tracker          tracker.Tracker
	executionManager ExecutionManager
	workflowEngine   *workflow.Engine
}

// NewExecutionHandler 创建新的执行处理器
func NewExecutionHandler(t tracker.Tracker, execMgr ExecutionManager, engine *workflow.Engine) *ExecutionHandler {
	return &ExecutionHandler{
		tracker:          t,
		executionManager: execMgr,
		workflowEngine:   engine,
	}
}

// HandleGetProgress 处理获取执行进度的请求
// GET /api/v1/:identifier/progress
func (h *ExecutionHandler) HandleGetProgress(c *gin.Context) {
	identifier := c.Param("identifier")

	// 获取任务信息
	ctx := context.Background()
	task, err := h.tracker.GetTask(ctx, identifier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "task_not_found",
				"message": "任务未找到: " + err.Error(),
			},
		})
		return
	}

	// 获取工作流
	wf := h.workflowEngine.GetWorkflow(task.ID)
	if wf == nil {
		c.JSON(http.StatusOK, gin.H{
			"identifier":     identifier,
			"current_stage":  "unknown",
			"status":         "unknown",
			"progress_summary": "工作流未初始化",
		})
		return
	}

	// 获取当前阶段
	currentStage := wf.Stages[wf.CurrentStage]
	if currentStage == nil {
		c.JSON(http.StatusOK, gin.H{
			"identifier":     identifier,
			"current_stage":  wf.CurrentStage,
			"status":         "unknown",
			"progress_summary": "阶段状态未知",
		})
		return
	}

	// 计算已用时间
	elapsedSeconds := int64(0)
	if currentStage.StartedAt != nil {
		elapsedSeconds = int64(time.Since(*currentStage.StartedAt).Seconds())
	}

	// 构建进度摘要
	progressSummary := buildProgressSummary(wf.CurrentStage, currentStage.Status, elapsedSeconds)

	response := gin.H{
		"identifier":       identifier,
		"current_stage":    wf.CurrentStage,
		"stage_display":    workflow.GetStageDisplayName(wf.CurrentStage),
		"status":           currentStage.Status,
		"status_display":   workflow.GetStatusDisplayName(currentStage.Status),
		"started_at":       formatTime(currentStage.StartedAt),
		"updated_at":       formatTime(currentStage.UpdatedAt),
		"elapsed_seconds":  elapsedSeconds,
		"elapsed_display":  formatDuration(elapsedSeconds),
		"progress_summary": progressSummary,
		"round":            currentStage.Round,
		"error":            currentStage.Error,
		"is_incomplete":    wf.IsIncomplete,
		"needs_attention":  wf.NeedsAttention,
	}

	// 如果有执行管理器，获取更详细的进度
	if h.executionManager != nil && wf.CurrentStage == workflow.StageImplementation {
		progress, err := h.executionManager.GetExecutionProgress(task.ID)
		if err == nil && progress != nil {
			response["turn_count"] = progress.TurnCount
			response["last_event"] = progress.LastEvent
			response["last_message"] = progress.LastMessage
			response["retry_count"] = progress.RetryCount
			response["max_retries"] = progress.MaxRetries
			if progress.TokenUsage != nil {
				response["token_usage"] = gin.H{
					"input_tokens":  progress.TokenUsage.InputTokens,
					"output_tokens": progress.TokenUsage.OutputTokens,
					"total_tokens":  progress.TokenUsage.TotalTokens,
				}
			}
			response["progress_summary"] = progress.ProgressSummary
		}
	}

	c.JSON(http.StatusOK, response)
}

// HandleGetLogs 处理获取执行日志的请求
// GET /api/v1/:identifier/logs?page=0&pageSize=100
func (h *ExecutionHandler) HandleGetLogs(c *gin.Context) {
	identifier := c.Param("identifier")

	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "0"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "100"))

	// 获取任务信息
	ctx := context.Background()
	task, err := h.tracker.GetTask(ctx, identifier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "task_not_found",
				"message": "任务未找到: " + err.Error(),
			},
		})
		return
	}

	// 检查执行管理器
	if h.executionManager == nil {
		c.JSON(http.StatusOK, gin.H{
			"identifier": identifier,
			"logs":       []workflow.ExecutionLog{},
			"total":      0,
			"page":       page,
			"page_size":  pageSize,
		})
		return
	}

	// 获取日志
	logs, total, err := h.executionManager.GetExecutionLogs(task.ID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"identifier": identifier,
			"logs":       []workflow.ExecutionLog{},
			"total":      0,
			"page":       page,
			"page_size":  pageSize,
			"message":    "暂无执行日志",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"identifier": identifier,
		"logs":       logs,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"has_more":   (page+1)*pageSize < total,
	})
}

// HandleGetLogsPage 处理获取执行日志页面的请求
// GET /tasks/:identifier/logs
func (h *ExecutionHandler) HandleGetLogsPage(c *gin.Context) {
	identifier := c.Param("identifier")

	// 获取任务信息
	ctx := context.Background()
	task, err := h.tracker.GetTask(ctx, identifier)
	if err != nil {
		html := components.RenderErrorHTML("任务不存在", fmt.Sprintf("无法找到任务 %s", identifier))
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusNotFound, html)
		return
	}

	// 获取阶段状态
	stageState, err := h.tracker.GetStageState(ctx, identifier)
	if err != nil {
		stageState = &domain.StageState{
			Name:      "unknown",
			Status:    "unknown",
			StartedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	// 获取执行日志
	var logs []workflow.ExecutionLog
	var total int
	if h.executionManager != nil {
		logs, total, _ = h.executionManager.GetExecutionLogs(task.ID, 0, 100)
	}

	// 获取工作流状态
	var progress *workflow.ExecutionProgress
	if h.executionManager != nil {
		progress, _ = h.executionManager.GetExecutionProgress(task.ID)
	}

	// 渲染页面
	html := RenderExecutionLogsHTML(task, stageState, logs, total, progress)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// HandleGetStatusDetail 处理获取任务状态详情的请求
// GET /api/v1/:identifier/status
func (h *ExecutionHandler) HandleGetStatusDetail(c *gin.Context) {
	identifier := c.Param("identifier")

	// 获取任务信息
	ctx := context.Background()
	task, err := h.tracker.GetTask(ctx, identifier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "task_not_found",
				"message": "任务未找到: " + err.Error(),
			},
		})
		return
	}

	// 获取阶段状态
	stageState, err := h.tracker.GetStageState(ctx, identifier)
	if err != nil {
		stageState = &domain.StageState{
			Name:      "unknown",
			Status:    "unknown",
			StartedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	// 获取工作流
	wf := h.workflowEngine.GetWorkflow(task.ID)

	// 构建状态详情
	response := gin.H{
		"identifier":     identifier,
		"title":          task.Title,
		"state":          task.State,
		"stage":          stageState.Name,
		"stage_status":   stageState.Status,
		"stage_display":  workflow.GetStageDisplayName(workflow.StageName(stageState.Name)),
		"updated_at":      formatTime(&stageState.UpdatedAt),
	}

	// 添加阶段历史
	if wf != nil {
		stages := wf.GetAllStages()
		stageHistory := make([]gin.H, 0, len(stages))
		for _, stage := range stages {
			stageHistory = append(stageHistory, gin.H{
				"name":          stage.Name,
				"display_name":  workflow.GetStageDisplayName(stage.Name),
				"status":        stage.Status,
				"status_display": workflow.GetStatusDisplayName(stage.Status),
				"started_at":    formatTime(stage.StartedAt),
				"completed_at":  formatTime(stage.CompletedAt),
				"updated_at":    formatTime(stage.UpdatedAt),
				"error":         stage.Error,
				"round":         stage.Round,
			})
		}
		response["stages"] = stageHistory
		response["current_stage"] = wf.CurrentStage
		response["is_incomplete"] = wf.IsIncomplete
		response["needs_attention"] = wf.NeedsAttention
	}

	c.JSON(http.StatusOK, response)
}

// RenderExecutionLogsHTML 渲染执行日志页面 HTML
func RenderExecutionLogsHTML(task *domain.Issue, stageState *domain.StageState, logs []workflow.ExecutionLog, total int, progress *workflow.ExecutionProgress) string {
	_ = common.StateBadgeClass(task.State) // state class for potential future use
	stageDisplay := workflow.GetStageDisplayName(workflow.StageName(stageState.Name))

	// 计算已用时间
	elapsedSeconds := int64(0)
	if stageState.StartedAt != (time.Time{}) {
		elapsedSeconds = int64(time.Since(stageState.StartedAt).Seconds())
	}
	elapsedDisplay := formatDuration(elapsedSeconds)

	// 进度摘要
	progressSummary := "执行中..."
	if progress != nil && progress.ProgressSummary != "" {
		progressSummary = progress.ProgressSummary
	}

	// 渲染日志列表
	logsHTML := renderLogsList(logs)

	return `<!DOCTYPE html>
<html lang="zh">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Symphony · 执行日志: ` + common.EscapeHTML(task.Identifier) + `</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=Fira+Code:wght@400;500&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="/dashboard.css">
    <script src="https://unpkg.com/htmx.org@1.9.10" crossorigin="anonymous"></script>
    <style>
    .logs-container {
        background: var(--surface);
        border: 1px solid var(--line);
        border-radius: var(--radius-lg);
        margin-top: 1rem;
    }
    .logs-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 1rem 1.5rem;
        border-bottom: 1px solid var(--line);
    }
    .logs-list {
        max-height: 600px;
        overflow-y: auto;
        padding: 1rem;
    }
    .log-entry {
        padding: 0.75rem 1rem;
        border-radius: var(--radius-sm);
        margin-bottom: 0.5rem;
        background: var(--card);
        border: 1px solid var(--line);
    }
    .log-entry:hover {
        background: var(--surface);
    }
    .log-timestamp {
        color: var(--muted);
        font-family: 'Fira Code', monospace;
        font-size: 0.8rem;
    }
    .log-event {
        display: inline-block;
        padding: 0.125rem 0.5rem;
        border-radius: var(--radius-sm);
        font-size: 0.75rem;
        font-weight: 500;
        margin-left: 0.5rem;
    }
    .log-event-info { background: rgba(59, 130, 246, 0.2); color: #3b82f6; }
    .log-event-success { background: rgba(34, 197, 94, 0.2); color: #22c55e; }
    .log-event-warning { background: rgba(245, 158, 11, 0.2); color: #f59e0b; }
    .log-event-error { background: rgba(239, 68, 68, 0.2); color: #ef4444; }
    .log-message {
        color: var(--ink);
        margin-top: 0.25rem;
        font-size: 0.9rem;
    }
    .log-data {
        background: var(--surface);
        border-radius: var(--radius-sm);
        padding: 0.5rem;
        margin-top: 0.5rem;
        font-family: 'Fira Code', monospace;
        font-size: 0.8rem;
        overflow-x: auto;
        white-space: pre-wrap;
        word-break: break-all;
    }
    .progress-card {
        background: linear-gradient(135deg, rgba(139, 92, 246, 0.1), rgba(139, 92, 246, 0.05));
        border: 1px solid rgba(139, 92, 246, 0.3);
        border-radius: var(--radius-lg);
        padding: 1.5rem;
        margin-bottom: 1rem;
    }
    .progress-item {
        display: flex;
        justify-content: space-between;
        padding: 0.5rem 0;
        border-bottom: 1px solid rgba(139, 92, 246, 0.2);
    }
    .progress-item:last-child {
        border-bottom: none;
    }
    .progress-label {
        color: var(--ink);
        font-size: 0.9rem;
    }
    .progress-value {
        color: var(--ink-bright);
        font-weight: 500;
        font-size: 0.9rem;
    }
    .empty-logs {
        text-align: center;
        padding: 3rem;
        color: var(--muted);
    }
    </style>
</head>
<body>
    <main class="app-shell">
        <section class="dashboard-shell">
            <header class="hero-card">
                <div class="hero-grid">
                    <div>
                        <p class="eyebrow">Symphony Orchestrator</p>
                        <h1 class="hero-title">执行日志: ` + common.EscapeHTML(task.Identifier) + `</h1>
                        <p class="hero-copy">` + common.EscapeHTML(task.Title) + `</p>
                    </div>
                    <div class="status-stack">
                        <a href="/api/v1/` + common.EscapeHTML(task.Identifier) + `" class="btn-secondary" style="padding: 0.5rem 1rem;">
                            返回任务详情
                        </a>
                    </div>
                </div>
            </header>

            <!-- 进度摘要卡片 -->
            <div class="progress-card">
                <h3 style="font-size: 1rem; font-weight: 600; color: var(--ink-bright); margin-bottom: 1rem;">执行进度</h3>
                <div class="progress-item">
                    <span class="progress-label">当前阶段</span>
                    <span class="progress-value">` + stageDisplay + `</span>
                </div>
                <div class="progress-item">
                    <span class="progress-label">状态</span>
                    <span class="progress-value">` + workflow.GetStatusDisplayName(workflow.StageStatus(stageState.Status)) + `</span>
                </div>
                <div class="progress-item">
                    <span class="progress-label">已用时间</span>
                    <span class="progress-value mono">` + elapsedDisplay + `</span>
                </div>
                <div class="progress-item">
                    <span class="progress-label">进度摘要</span>
                    <span class="progress-value">` + common.EscapeHTML(progressSummary) + `</span>
                </div>
                ` + func() string {
		if progress != nil && progress.TurnCount > 0 {
			return `<div class="progress-item">
                    <span class="progress-label">Turn 次数</span>
                    <span class="progress-value mono">` + strconv.Itoa(progress.TurnCount) + `</span>
                </div>`
		}
		return ""
	}() + `
                ` + func() string {
		if progress != nil && progress.TokenUsage != nil && progress.TokenUsage.TotalTokens > 0 {
			return `<div class="progress-item">
                    <span class="progress-label">Token 使用</span>
                    <span class="progress-value mono">` + common.FormatInt(progress.TokenUsage.TotalTokens) + `</span>
                </div>`
		}
		return ""
	}() + `
            </div>

            <!-- 执行日志 -->
            <div class="logs-container">
                <div class="logs-header">
                    <div>
                        <h3 style="font-size: 1rem; font-weight: 600; color: var(--ink-bright);">执行日志</h3>
                        <span style="color: var(--muted); font-size: 0.85rem;">共 ` + strconv.Itoa(total) + ` 条记录</span>
                    </div>
                    <div style="display: flex; gap: 0.5rem;">
                        <button onclick="refreshLogs()" class="btn-secondary" style="padding: 0.5rem 1rem; border: 1px solid var(--line); border-radius: var(--radius); background: transparent; color: var(--ink);">
                            刷新
                        </button>
                    </div>
                </div>
                <div class="logs-list" id="logs-list">
                    ` + logsHTML + `
                </div>
            </div>
        </section>
    </main>
    <script>
    function refreshLogs() {
        location.reload();
    }

    // SSE 实时更新日志
    const eventSource = new EventSource('/events');
    eventSource.addEventListener('state', function(e) {
        // 检查是否有新的日志需要刷新
        // 这里可以添加更智能的增量更新逻辑
    });
    </script>
</body>
</html>`
}

// renderLogsList 渲染日志列表
func renderLogsList(logs []workflow.ExecutionLog) string {
	if len(logs) == 0 {
		return `<div class="empty-logs">
            <p>暂无执行日志</p>
            <p style="font-size: 0.85rem; margin-top: 0.5rem;">执行开始后将在此显示日志</p>
        </div>`
	}

	html := ""
	for _, log := range logs {
		eventClass := getLogEventClass(log.Event)
		dataHTML := ""
		if len(log.Data) > 0 {
			dataHTML = fmt.Sprintf(`<div class="log-data">%s</div>`, common.PrettyValue(log.Data))
		}

		html += fmt.Sprintf(`
            <div class="log-entry">
                <div>
                    <span class="log-timestamp">%s</span>
                    <span class="log-event %s">%s</span>
                </div>
                <div class="log-message">%s</div>
                %s
            </div>`,
			log.Timestamp.Format("15:04:05.000"),
			eventClass,
			common.EscapeHTML(log.Event),
			common.EscapeHTML(log.Message),
			dataHTML,
		)
	}

	return html
}

// getLogEventClass 获取日志事件样式类
func getLogEventClass(event string) string {
	switch {
	case containsStr(event, "error"), containsStr(event, "failed"):
		return "log-event-error"
	case containsStr(event, "completed"), containsStr(event, "success"):
		return "log-event-success"
	case containsStr(event, "warning"), containsStr(event, "retry"):
		return "log-event-warning"
	default:
		return "log-event-info"
	}
}

// buildProgressSummary 构建进度摘要
func buildProgressSummary(stage workflow.StageName, status workflow.StageStatus, elapsedSeconds int64) string {
	stageDisplay := workflow.GetStageDisplayName(stage)
	statusDisplay := workflow.GetStatusDisplayName(status)
	elapsedDisplay := formatDuration(elapsedSeconds)

	switch stage {
	case workflow.StageClarification:
		return fmt.Sprintf("正在进行%s - %s", stageDisplay, elapsedDisplay)
	case workflow.StageBDDReview:
		if status == workflow.StatusPending || status == workflow.StatusInProgress {
			return "等待 BDD 规则审核"
		}
		return fmt.Sprintf("%s - %s", stageDisplay, statusDisplay)
	case workflow.StageArchitectureReview:
		if status == workflow.StatusPending || status == workflow.StatusInProgress {
			return "等待架构设计审核"
		}
		return fmt.Sprintf("%s - %s", stageDisplay, statusDisplay)
	case workflow.StageImplementation:
		return fmt.Sprintf("正在实现 - %s", elapsedDisplay)
	case workflow.StageVerification:
		return fmt.Sprintf("正在验证 - %s", elapsedDisplay)
	default:
		return fmt.Sprintf("%s - %s", stageDisplay, statusDisplay)
	}
}

// formatDuration 格式化持续时间
func formatDuration(seconds int64) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	minutes := seconds / 60
	secs := seconds % 60
	if minutes < 60 {
		return fmt.Sprintf("%dm %ds", minutes, secs)
	}
	hours := minutes / 60
	mins := minutes % 60
	return fmt.Sprintf("%dh %dm", hours, mins)
}

// formatTime 格式化时间
func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

// containsStr 检查字符串是否包含子串
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}