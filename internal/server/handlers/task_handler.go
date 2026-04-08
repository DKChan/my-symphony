package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/dministrator/symphony/internal/common"
	"github.com/dministrator/symphony/internal/domain"
	"github.com/dministrator/symphony/internal/server/components"
	"github.com/dministrator/symphony/internal/tracker"
)

// TaskHandler 任务创建处理器
type TaskHandler struct {
	tracker tracker.Tracker
	engine  *components.TemplateEngine
}

// NewTaskHandler 创建新的任务处理器
func NewTaskHandler() *TaskHandler {
	return &TaskHandler{}
}

// NewTaskHandlerWithTracker 创建带有 tracker 的任务处理器
func NewTaskHandlerWithTracker(t tracker.Tracker) *TaskHandler {
	return &TaskHandler{tracker: t}
}

// NewTaskHandlerWithTrackerAndEngine 创建带有 tracker 和模板引擎的任务处理器
func NewTaskHandlerWithTrackerAndEngine(t tracker.Tracker, engine *components.TemplateEngine) *TaskHandler {
	return &TaskHandler{tracker: t, engine: engine}
}

// HandleNewTaskForm 处理任务创建表单页面请求
func (h *TaskHandler) HandleNewTaskForm(c *gin.Context) {
	html := RenderTaskFormHTML()
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// HandleTaskDetail 处理任务详情页面请求
// GET /tasks/:identifier
func (h *TaskHandler) HandleTaskDetail(c *gin.Context) {
	identifier := c.Param("identifier")

	// 检查 tracker 是否可用
	if h.tracker == nil {
		if h.engine != nil {
			data := &components.TemplateData{
				Title:        "错误",
				ErrorTitle:   "Tracker 不可用",
				ErrorMessage: "无法获取任务信息",
			}
			h.engine.RenderHTML(c, "pages/error.html", data)
			return
		}
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "tracker not available",
		})
		return
	}

	ctx := context.Background()

	// 获取任务详情
	issue, err := h.tracker.GetTask(ctx, identifier)
	if err != nil {
		// 任务不存在，显示错误页面
		if h.engine != nil {
			data := &components.TemplateData{
				Title:        "错误",
				ErrorTitle:   "任务不存在",
				ErrorMessage: fmt.Sprintf("无法找到任务 %s", identifier),
			}
			h.engine.RenderHTML(c, "pages/error.html", data)
			return
		}
		html := components.RenderErrorHTML("任务不存在", fmt.Sprintf("无法找到任务 %s", identifier))
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusNotFound, html)
		return
	}

	// 获取阶段状态
	stageState, err := h.tracker.GetStageState(ctx, identifier)
	if err != nil {
		// 获取失败时使用默认状态
		stageState = &domain.StageState{
			Name:      "unknown",
			Status:    "unknown",
			StartedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	// 获取对话历史
	conversationHistory, err := h.tracker.GetConversationHistory(ctx, identifier)
	if err != nil {
		// 获取失败时使用空历史
		conversationHistory = []domain.ConversationTurn{}
	}

	// 使用模板引擎（如果可用）
	if h.engine != nil {
		// 计算辅助数据
		elapsedDisplay := ""
		if stageState.StartedAt != (time.Time{}) {
			elapsedSeconds := int64(time.Since(stageState.StartedAt).Seconds())
			elapsedDisplay = components.FormatDurationForDetail(elapsedSeconds)
		}

		currentRound := stageState.Round
		roundProgress := fmt.Sprintf("%d / %d", currentRound, components.MaxClarificationRounds)

		// 获取当前问题
		currentQuestion := ""
		for i := len(conversationHistory) - 1; i >= 0; i-- {
			if conversationHistory[i].Role == "assistant" {
				currentQuestion = conversationHistory[i].Content
				break
			}
		}

		data := &components.TemplateData{
			Title:             fmt.Sprintf("任务详情: %s", issue.Identifier),
			PageName:          "task-detail",
			HeroCopy:          issue.Title,
			NeedsHTMX:         true,
			ShowBackButton:    true,
			BackURL:           "/",
			BackText:          "返回看板",
			Issue:             issue,
			StageState:        stageState,
			Conversation:      conversationHistory,
			ConversationHTML:  components.RenderConversationHistoryHTML(conversationHistory),
			ElapsedDisplay:    elapsedDisplay,
			StageDisplay:      components.GetStageDisplay(stageState.Name),
			StatusDisplay:     components.GetStatusDisplay(stageState.Status),
			StateClass:        common.StateBadgeClass(issue.State),
			IsWaitingForAnswer: stageState.Name == "clarification" && stageState.Status == "in_progress",
			IsImplementation:   stageState.Name == "implementation",
			IsNeedsAttention:   stageState.Name == "needs_attention",
			IsVerification:     stageState.Name == "verification",
			CurrentQuestion:    currentQuestion,
			RoundProgress:      roundProgress,
			CurrentRound:       currentRound,
		}

		h.engine.RenderHTML(c, "pages/task-detail.html", data)
		return
	}

	// 兜底：使用旧的字符串模板
	html := components.RenderTaskDetailHTML(issue, stageState, conversationHistory)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// RenderTaskFormHTML 渲染任务创建表单 HTML
func RenderTaskFormHTML() string {
	return `<!DOCTYPE html>
<html lang="zh">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Symphony · 创建需求</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=Fira+Code:wght@400;500&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="/dashboard.css">
    <script src="https://unpkg.com/htmx.org@1.9.10" crossorigin="anonymous"></script>
</head>
<body>
    <main class="app-shell">
        <section class="dashboard-shell">
            <header class="hero-card">
                <div class="hero-grid">
                    <div>
                        <p class="eyebrow">Symphony Orchestrator</p>
                        <h1 class="hero-title">创建新需求</h1>
                        <p class="hero-copy">创建一个新的需求任务，系统将自动生成 5 个子阶段任务并建立依赖关系。</p>
                    </div>
                </div>
            </header>

            <section class="section-card" style="background: var(--card); border: 1px solid var(--line); border-radius: var(--radius-lg); padding: 1.5rem;">
                <form id="task-form" hx-post="/api/tasks" hx-target="#result" hx-swap="innerHTML">
                    <div class="form-group" style="margin-bottom: 1.25rem;">
                        <label for="title" class="form-label" style="display: block; font-weight: 500; margin-bottom: 0.5rem; color: var(--ink-bright);">需求标题</label>
                        <input type="text" id="title" name="title" required
                            style="width: 100%; padding: 0.75rem 1rem; border: 1px solid var(--line); border-radius: var(--radius); background: var(--bg); color: var(--ink-bright); font-size: 1rem;"
                            placeholder="例如：实现用户登录功能">
                    </div>

                    <div class="form-group" style="margin-bottom: 1.5rem;">
                        <label for="description" class="form-label" style="display: block; font-weight: 500; margin-bottom: 0.5rem; color: var(--ink-bright);">需求描述</label>
                        <textarea id="description" name="description" rows="6" required
                            style="width: 100%; padding: 0.75rem 1rem; border: 1px solid var(--line); border-radius: var(--radius); background: var(--bg); color: var(--ink-bright); font-size: 1rem; resize: vertical; min-height: 150px;"
                            placeholder="详细描述需求的内容、背景和预期结果..."></textarea>
                    </div>

                    <div class="form-actions" style="display: flex; gap: 1rem; justify-content: flex-end;">
                        <a href="/" class="btn-secondary" style="padding: 0.75rem 1.5rem; border: 1px solid var(--line); border-radius: var(--radius); background: transparent; color: var(--muted);">
                            取消
                        </a>
                        <button type="submit" class="btn-primary" style="padding: 0.75rem 1.5rem; border: none; border-radius: var(--radius); background: var(--accent); color: white; font-weight: 500; cursor: pointer;">
                            <span id="submit-text">创建需求</span>
                            <span id="loading-text" style="display: none;">正在创建...</span>
                        </button>
                    </div>
                </form>

                <div id="result" style="margin-top: 1.5rem;"></div>
            </section>

            <section class="info-card" style="background: var(--surface); border: 1px solid var(--line); border-radius: var(--radius-lg); padding: 1.25rem; margin-top: 1.5rem;">
                <h3 style="font-size: 0.9rem; font-weight: 600; color: var(--ink-bright); margin-bottom: 0.75rem;">创建后将生成以下子任务</h3>
                <ul style="list-style: none; padding: 0; margin: 0;">
                    <li style="padding: 0.5rem 0; border-bottom: 1px solid var(--line); color: var(--ink);">
                        <span class="stage-badge" style="background: var(--warning-bg); color: var(--warning-text); padding: 0.25rem 0.5rem; border-radius: var(--radius-sm); font-size: 0.8rem; margin-right: 0.5rem;">1</span>
                        需求澄清 - 收集和澄清需求细节
                    </li>
                    <li style="padding: 0.5rem 0; border-bottom: 1px solid var(--line); color: var(--ink);">
                        <span class="stage-badge" style="background: var(--warning-bg); color: var(--warning-text); padding: 0.25rem 0.5rem; border-radius: var(--radius-sm); font-size: 0.8rem; margin-right: 0.5rem;">2</span>
                        BDD 审核 - 审核行为驱动开发规范
                    </li>
                    <li style="padding: 0.5rem 0; border-bottom: 1px solid var(--line); color: var(--ink);">
                        <span class="stage-badge" style="background: var(--warning-bg); color: var(--warning-text); padding: 0.25rem 0.5rem; border-radius: var(--radius-sm); font-size: 0.8rem; margin-right: 0.5rem;">3</span>
                        架构审核 - 审核技术架构设计
                    </li>
                    <li style="padding: 0.5rem 0; border-bottom: 1px solid var(--line); color: var(--ink);">
                        <span class="stage-badge" style="background: var(--warning-bg); color: var(--warning-text); padding: 0.25rem 0.5rem; border-radius: var(--radius-sm); font-size: 0.8rem; margin-right: 0.5rem;">4</span>
                        实现 - 编写代码并完成功能
                    </li>
                    <li style="padding: 0.5rem 0; color: var(--ink);">
                        <span class="stage-badge" style="background: var(--warning-bg); color: var(--warning-text); padding: 0.25rem 0.5rem; border-radius: var(--radius-sm); font-size: 0.8rem; margin-right: 0.5rem;">5</span>
                        验收 - 验证功能符合预期
                    </li>
                </ul>
                <p style="margin-top: 0.75rem; color: var(--muted); font-size: 0.85rem;">
                    子任务之间通过依赖关系阻塞，确保按顺序执行。
                </p>
            </section>
        </section>
    </main>
    <script>
        document.getElementById('task-form').addEventListener('htmx:beforeRequest', function(e) {
            document.getElementById('submit-text').style.display = 'none';
            document.getElementById('loading-text').style.display = 'inline';
        });

        document.getElementById('task-form').addEventListener('htmx:afterRequest', function(e) {
            document.getElementById('submit-text').style.display = 'inline';
            document.getElementById('loading-text').style.display = 'none';
        });
    </script>
</body>
</html>`
}

// TaskCreateRequest 任务创建请求结构
type TaskCreateRequest struct {
	Title       string `json:"title" form:"title" binding:"required"`
	Description string `json:"description" form:"description" binding:"required"`
}

// TaskCreateResponse 任务创建响应结构
type TaskCreateResponse struct {
	ParentTask *TaskInfo  `json:"parent_task"`
	SubTasks   []TaskInfo `json:"sub_tasks"`
	Message    string     `json:"message"`
}

// TaskInfo 任务信息
type TaskInfo struct {
	ID          string `json:"id"`
	Identifier  string `json:"identifier"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	State       string `json:"state"`
	URL         string `json:"url,omitempty"`
}

// RenderTaskCreatedHTML 渲染任务创建成功 HTML
func RenderTaskCreatedHTML(parent TaskInfo, subTasks []TaskInfo) string {
	subTasksHTML := ""
	for _, st := range subTasks {
		blockedBy := ""
		if st.Identifier != parent.Identifier+"-1" {
			prevNum := 0
			for i, t := range subTasks {
				if t.Identifier == st.Identifier {
					prevNum = i
					break
				}
			}
			if prevNum > 0 {
				blockedBy = fmt.Sprintf(` <span style="color: var(--muted); font-size: 0.8rem;">(blocked by %s)</span>`, subTasks[prevNum-1].Identifier)
			}
		}

		subTasksHTML += fmt.Sprintf(`
            <li style="padding: 0.75rem 0; border-bottom: 1px solid var(--line);">
                <span style="font-weight: 500; color: var(--accent);">%s</span>
                <span style="color: var(--ink);">%s</span>
                %s
            </li>`, st.Identifier, st.Title, blockedBy)
	}

	return fmt.Sprintf(`
<div class="success-card" style="background: linear-gradient(135deg, rgba(34, 197, 94, 0.1), rgba(34, 197, 94, 0.05)); border: 1px solid rgba(34, 197, 94, 0.3); border-radius: var(--radius-lg); padding: 1.5rem;">
    <div style="display: flex; align-items: center; gap: 0.75rem; margin-bottom: 1rem;">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: rgb(34, 197, 94);">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path>
            <polyline points="22 4 12 14.01 9 11.01"></polyline>
        </svg>
        <h3 style="font-size: 1.1rem; font-weight: 600; color: var(--ink-bright);">需求创建成功</h3>
    </div>

    <div style="margin-bottom: 1.25rem;">
        <p style="color: var(--ink); margin-bottom: 0.5rem;">
            <strong>父任务:</strong> <span style="color: var(--accent);">%s</span> - %s
        </p>
        <p style="color: var(--muted); font-size: 0.9rem;">状态: %s</p>
    </div>

    <h4 style="font-size: 0.9rem; font-weight: 600; color: var(--ink-bright); margin-bottom: 0.5rem;">生成的子任务:</h4>
    <ul style="list-style: none; padding: 0; margin: 0;">
        %s
    </ul>

    <div style="margin-top: 1.5rem; display: flex; gap: 1rem;">
        <a href="/api/v1/%s" class="btn-secondary" style="padding: 0.75rem 1.5rem; border: 1px solid var(--line); border-radius: var(--radius); background: transparent; color: var(--ink);">
            查看任务详情
        </a>
        <a href="/" class="btn-primary" style="padding: 0.75rem 1.5rem; border: none; border-radius: var(--radius); background: var(--accent); color: white; font-weight: 500;">
            返回看板
        </a>
        <button onclick="location.href='/tasks/new'" class="btn-secondary" style="padding: 0.75rem 1.5rem; border: 1px solid var(--line); border-radius: var(--radius); background: transparent; color: var(--ink);">
            创建更多
        </button>
    </div>
</div>`, parent.Identifier, parent.Title, parent.State, subTasksHTML, parent.Identifier)
}

// HandleBDDReviewPage 处理 BDD 规则审核页面请求
// GET /tasks/:identifier/bdd
func (h *TaskHandler) HandleBDDReviewPage(c *gin.Context) {
	identifier := c.Param("identifier")

	// 检查 tracker 是否可用
	if h.tracker == nil {
		html := components.RenderErrorHTML("Tracker 不可用", "无法获取任务信息")
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusInternalServerError, html)
		return
	}

	ctx := context.Background()

	// 获取任务详情
	issue, err := h.tracker.GetTask(ctx, identifier)
	if err != nil {
		// 任务不存在，显示错误页面
		html := components.RenderErrorHTML("任务不存在", fmt.Sprintf("无法找到任务 %s", identifier))
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusNotFound, html)
		return
	}

	// 获取阶段状态
	stageState, err := h.tracker.GetStageState(ctx, identifier)
	if err != nil {
		// 获取失败时使用默认状态
		stageState = &domain.StageState{
			Name:      "bdd_review",
			Status:    "pending",
			StartedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	// 获取 BDD 内容
	bddContent, err := h.tracker.GetBDDContent(ctx, identifier)
	if err != nil {
		// BDD 内容不存在时，显示空内容页面
		bddContent = ""
	}

	// RenderBDDReviewHTML 渲染 BDD 规则审核页面
	html := components.RenderBDDReviewHTML(issue, stageState, bddContent)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// HandleArchitectureReviewPage 处理架构设计审核页面请求
// GET /tasks/:identifier/architecture
func (h *TaskHandler) HandleArchitectureReviewPage(c *gin.Context) {
	identifier := c.Param("identifier")

	// 检查 tracker 是否可用
	if h.tracker == nil {
		html := components.RenderErrorHTML("Tracker 不可用", "无法获取任务信息")
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusInternalServerError, html)
		return
	}

	ctx := context.Background()

	// 获取任务详情
	issue, err := h.tracker.GetTask(ctx, identifier)
	if err != nil {
		// 任务不存在，显示错误页面
		html := components.RenderErrorHTML("任务不存在", fmt.Sprintf("无法找到任务 %s", identifier))
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusNotFound, html)
		return
	}

	// 获取阶段状态
	stageState, err := h.tracker.GetStageState(ctx, identifier)
	if err != nil {
		// 获取失败时使用默认状态
		stageState = &domain.StageState{
			Name:      "architecture_review",
			Status:    "pending",
			StartedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	// 获取架构设计内容
	archContent, err := h.tracker.GetArchitectureContent(ctx, identifier)
	if err != nil {
		// 架构设计内容不存在时，显示空内容页面
		archContent = ""
	}

	// 获取 TDD 规则内容
	tddContent, err := h.tracker.GetTDDContent(ctx, identifier)
	if err != nil {
		// TDD 规则内容不存在时，显示空内容页面
		tddContent = ""
	}

	// 渲染架构设计审核页面
	html := components.RenderArchitectureReviewHTML(issue, stageState, archContent, tddContent)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}
// HandleNeedsAttentionPage 处理待人工处理任务详情页面请求
// GET /tasks/:identifier/needs-attention
func (h *TaskHandler) HandleNeedsAttentionPage(c *gin.Context) {
	identifier := c.Param("identifier")

	// 检查 tracker 是否可用
	if h.tracker == nil {
		html := components.RenderErrorHTML("Tracker 不可用", "无法获取任务信息")
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusInternalServerError, html)
		return
	}

	ctx := context.Background()

	// 获取任务详情
	issue, err := h.tracker.GetTask(ctx, identifier)
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
			Name:      "needs_attention",
			Status:    "failed",
			StartedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	// 渲染待人工处理页面
	html := components.RenderNeedsAttentionHTML(issue, stageState)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// HandleVerificationPage 处理验收报告页面请求
// GET /tasks/:identifier/verification
func (h *TaskHandler) HandleVerificationPage(c *gin.Context) {
	identifier := c.Param("identifier")

	// 检查 tracker 是否可用
	if h.tracker == nil {
		html := components.RenderErrorHTML("Tracker 不可用", "无法获取任务信息")
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusInternalServerError, html)
		return
	}

	ctx := context.Background()

	// 获取任务详情
	issue, err := h.tracker.GetTask(ctx, identifier)
	if err != nil {
		// 任务不存在，显示错误页面
		html := components.RenderErrorHTML("任务不存在", fmt.Sprintf("无法找到任务 %s", identifier))
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusNotFound, html)
		return
	}

	// 获取阶段状态
	stageState, err := h.tracker.GetStageState(ctx, identifier)
	if err != nil {
		// 获取失败时使用默认状态
		stageState = &domain.StageState{
			Name:      "verification",
			Status:    "pending",
			StartedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	// 获取验收报告
	report, err := h.tracker.GetVerificationReport(ctx, identifier)
	if err != nil {
		// 验收报告不存在时，显示空报告
		report = nil
	}

	// 渲染验收报告页面
	html := components.RenderVerificationReportHTML(issue, stageState, report)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}
