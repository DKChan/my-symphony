package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/dministrator/symphony/internal/common"
	"github.com/dministrator/symphony/internal/domain"
	"github.com/dministrator/symphony/internal/server/components"
	"github.com/gin-gonic/gin"
)

// DashboardHandler 仪表板页面处理器
type DashboardHandler struct {
	orchestrator OrchestratorGetter
	engine       *components.TemplateEngine
}

// NewDashboardHandler 创建新的仪表板处理器
func NewDashboardHandler(orch OrchestratorGetter) *DashboardHandler {
	return &DashboardHandler{
		orchestrator: orch,
	}
}

// NewDashboardHandlerWithEngine 创建带模板引擎的仪表板处理器
func NewDashboardHandlerWithEngine(orch OrchestratorGetter, engine *components.TemplateEngine) *DashboardHandler {
	return &DashboardHandler{
		orchestrator: orch,
		engine:       engine,
	}
}

// Handle 处理仪表板页面请求
func (h *DashboardHandler) Handle(c *gin.Context) {
	state := h.orchestrator.GetState()
	now := time.Now()

	// 构建看板数据
	kanbanPayload := buildKanbanPayload(state)

	// 使用模板引擎（如果可用）
	if h.engine != nil {
		data := &components.TemplateData{
			Title:         "任务看板",
			PageName:      "dashboard",
			State:         state,
			Now:           now,
			HeroCopy:      "实时监控运行中的 Agent 会话、重试队列状态和 Token 使用量。",
			NeedsSSE:      true,
			NeedsHTMX:     true,
			ShowBackButton: false,
			MetricRunning:  len(state.Running),
			MetricRetrying: len(state.RetryAttempts),
			MetricTokens:   common.FormatInt(state.CodexTotals.TotalTokens),
			MetricRuntime:  common.FormatRuntimeSeconds(common.TotalRuntimeSeconds(state, now)),
			RateLimitsHTML: template.HTML(common.PrettyValue(state.CodexRateLimits)),
			KanbanHTML:     template.HTML(components.RenderStageKanban(kanbanPayload)),
		}

		h.engine.RenderHTML(c, "dashboard-page", data)
		return
	}

	// 兜底：使用旧的字符串模板
	html := components.RenderDashboardHTML(state, now)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// buildKanbanPayload 构建看板数据载荷
func buildKanbanPayload(state *domain.OrchestratorState) *common.KanbanPayload {
	payload := &common.KanbanPayload{
		GeneratedAt: time.Now().Format(time.RFC3339),
		Columns:     []common.KanbanColumn{},
		TotalTasks:  0,
	}

	// 按看板列分组任务 (使用 TaskStageToKanbanColumn 映射)
	columnTasks := map[string][]common.KanbanTaskPayload{}

	// 处理 Running 任务
	for _, entry := range state.Running {
		taskStage := entry.Stage
		if taskStage == "" {
			taskStage = "implementation"
		}
		// 映射到看板列
		kanbanColumn := common.TaskStageToKanbanColumn(taskStage)

		sessionID := ""
		if entry.Session != nil {
			sessionID = entry.Session.SessionID
		}

		columnTasks[kanbanColumn] = append(columnTasks[kanbanColumn], common.KanbanTaskPayload{
			IssueID:         entry.Issue.ID,
			IssueIdentifier: entry.Identifier,
			Title:           entry.Title,
			State:           entry.Stage,
			Stage:           taskStage,
			SessionID:       sessionID,
			RuntimeTurns:    formatRuntimeTurns(entry.StartedAt, entry.TurnCount),
			LastEvent:       "",
			LastEventAt:     entry.StartedAt.Format(time.RFC3339),
			Tokens:          common.Tokens{},
		})
	}

	// 处理 Retry 任务 (映射到 done 列，因为 needs_attention 属于终态)
	for _, entry := range state.RetryAttempts {
		errMsg := ""
		if entry.Error != nil {
			errMsg = *entry.Error
		}
		columnTasks["done"] = append(columnTasks["done"], common.KanbanTaskPayload{
			IssueID:         entry.IssueID,
			IssueIdentifier: entry.Identifier,
			Title:           "",
			State:           "retrying",
			Stage:           "needs_attention",
			Attempt:         entry.Attempt,
			Error:           errMsg,
			DueAt:           time.Now().Add(time.Duration(entry.DueAtMs) * time.Millisecond).Format(time.RFC3339),
		})
	}

	// 构建各看板列
	for _, cfg := range common.KanbanStages {
		tasks := columnTasks[cfg.ID]
		col := common.KanbanColumn{
			ID:        cfg.ID,
			Title:     cfg.Title,
			Icon:      iconToEmoji(cfg.Icon),
			Color:     cfg.Color,
			Tasks:     tasks,
			TaskCount: len(tasks),
		}
		payload.Columns = append(payload.Columns, col)
		payload.TotalTasks += len(tasks)
	}

	return payload
}

// formatRuntimeTurns 格式化运行时间和轮次
func formatRuntimeTurns(startedAt time.Time, turnCount int) string {
	duration := time.Since(startedAt)
	mins := int(duration.Minutes())
	if mins < 1 {
		return fmt.Sprintf("%d轮", turnCount)
	}
	return fmt.Sprintf("%dm %d轮", mins, turnCount)
}

// iconToEmoji 将图标名称转换为 emoji
func iconToEmoji(icon string) string {
	emojiMap := map[string]string{
		"inbox":        "⏸",
		"clipboard":    "📋",
		"code":         "⚙️",
		"check-circle": "🔍",
		"check":        "✅",
	}
	if emoji, ok := emojiMap[icon]; ok {
		return emoji
	}
	return "📌"
}
