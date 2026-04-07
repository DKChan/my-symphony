package components

import (
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	"github.com/dministrator/symphony/internal/common"
	"github.com/dministrator/symphony/internal/domain"
	"github.com/dministrator/symphony/internal/workflow"
)

// MaxClarificationRounds 最大澄清轮次
const MaxClarificationRounds = 5

// RenderErrorHTML 渲染错误页面
func RenderErrorHTML(title, message string) string {
	return `<!DOCTYPE html>
<html lang="zh">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Symphony · 错误</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=Fira+Code:wght@400;500&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="/dashboard.css">
</head>
<body>
    <main class="app-shell">
        <section class="dashboard-shell">
            <header class="hero-card">
                <div class="hero-grid">
                    <div>
                        <p class="eyebrow">Symphony Orchestrator</p>
                        <h1 class="hero-title" style="color: var(--danger);">` + common.EscapeHTML(title) + `</h1>
                        <p class="hero-copy">` + common.EscapeHTML(message) + `</p>
                    </div>
                </div>
            </header>
            <section class="section-card" style="background: var(--card); border: 1px solid var(--line); border-radius: var(--radius-lg); padding: 1.5rem;">
                <a href="/" class="btn-secondary" style="display: inline-flex; align-items: center; gap: 0.5rem; padding: 0.75rem 1.5rem; border: 1px solid var(--line); border-radius: var(--radius); background: transparent; color: var(--ink); text-decoration: none;">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <line x1="19" y1="12" x2="5" y2="12"></line>
                        <polyline points="12 19 5 12 12 5"></polyline>
                    </svg>
                    返回看板
                </a>
            </section>
        </section>
    </main>
</body>
</html>`
}

// RenderTaskDetailHTML 渲染任务详情页面
func RenderTaskDetailHTML(issue *domain.Issue, stageState *domain.StageState, conversationHistory []domain.ConversationTurn) string {
	// 解析状态
	state := issue.State
	stateClass := common.StateBadgeClass(state)
	stageDisplay := getStageDisplay(stageState.Name)
	statusDisplay := getStatusDisplay(stageState.Status)

	// 计算已用时间
	elapsedSeconds := int64(0)
	elapsedDisplay := ""
	if stageState.StartedAt != (time.Time{}) {
		elapsedSeconds = int64(time.Since(stageState.StartedAt).Seconds())
		elapsedDisplay = formatDurationForDetail(elapsedSeconds)
	}

	// 解析澄清进度
	currentRound := stageState.Round
	roundProgress := strconv.Itoa(currentRound) + " / " + strconv.Itoa(MaxClarificationRounds)

	// 判断是否处于等待用户回答状态
	isWaitingForAnswer := stageState.Name == "clarification" && stageState.Status == "in_progress"

	// 判断是否在实现阶段或需要人工处理阶段
	isImplementation := stageState.Name == "implementation"
	isNeedsAttention := stageState.Name == "needs_attention"
	isVerification := stageState.Name == "verification"

	// 获取当前问题（最后一条 assistant 消息）
	currentQuestion := ""
	if len(conversationHistory) > 0 {
		for i := len(conversationHistory) - 1; i >= 0; i-- {
			if conversationHistory[i].Role == "assistant" {
				currentQuestion = conversationHistory[i].Content
				break
			}
		}
	}

	// 渲染历史问答记录
	historyHTML := renderConversationHistory(conversationHistory)

	// 构建页面 HTML
	return `<!DOCTYPE html>
<html lang="zh">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Symphony · ` + common.EscapeHTML(issue.Identifier) + `</title>
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
                        <h1 class="hero-title">任务详情: ` + common.EscapeHTML(issue.Identifier) + `</h1>
                        <p class="hero-copy">` + common.EscapeHTML(issue.Title) + `</p>
                    </div>
                    <div class="status-stack">
                        <a href="/" class="btn-secondary" style="display: inline-flex; align-items: center; gap: 0.5rem; padding: 0.5rem 1rem; border: 1px solid var(--line); border-radius: var(--radius); background: transparent; color: var(--ink); text-decoration: none; font-weight: 500; font-size: 0.9rem;">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                                <line x1="19" y1="12" x2="5" y2="12"></line>
                                <polyline points="12 19 5 12 12 5"></polyline>
                            </svg>
                            返回看板
                        </a>
                    </div>
                </div>
            </header>

            <section class="section-card" style="background: var(--card); border: 1px solid var(--line); border-radius: var(--radius-lg); padding: 1.5rem;">
                <div class="task-detail-header" style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;">
                    <div>
                        <h2 style="font-size: 1.25rem; font-weight: 600; color: var(--ink-bright); margin-bottom: 0.5rem;">` + common.EscapeHTML(issue.Title) + `</h2>
                        <div style="display: flex; gap: 1rem; align-items: center;">
                            <span class="` + stateClass + `">` + common.EscapeHTML(state) + `</span>
                            <span style="color: var(--muted); font-size: 0.85rem;">阶段: ` + stageDisplay + `</span>
                            <span style="color: var(--muted); font-size: 0.85rem;">状态: ` + statusDisplay + `</span>
                            ` + func() string {
		if elapsedDisplay != "" {
			return `<span style="color: var(--muted); font-size: 0.85rem;">已用时间: <span class="mono">` + elapsedDisplay + `</span></span>`
		}
		return ""
	}() + `
                        </div>
                    </div>
                    ` + func() string {
		if isImplementation || isNeedsAttention || isVerification {
			return `<div style="display: flex; gap: 0.5rem;">
                        <a href="/tasks/` + common.EscapeHTML(issue.Identifier) + `/logs" class="btn-secondary" style="padding: 0.5rem 1rem; border: 1px solid var(--line); border-radius: var(--radius); background: transparent; color: var(--ink); font-size: 0.85rem;">
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="vertical-align: middle; margin-right: 0.25rem;">
                                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                                <polyline points="14 2 14 8 20 8"></polyline>
                                <line x1="16" y1="13" x2="8" y2="13"></line>
                                <line x1="16" y1="17" x2="8" y2="17"></line>
                            </svg>
                            执行日志
                        </a>
                    </div>`
		}
		return ""
	}() + `
                </div>

                <!-- 进度显示 -->
                ` + func() string {
		// 澄清阶段显示澄清进度
		if stageState.Name == "clarification" {
			return `<div class="progress-section" style="background: var(--surface); border-radius: var(--radius-sm); padding: 1rem; margin-bottom: 1rem;">
                    <div style="display: flex; justify-content: space-between; align-items: center;">
                        <span style="color: var(--ink); font-weight: 500;">澄清进度</span>
                        <span class="mono" style="color: var(--accent);">第 ` + roundProgress + ` 轮</span>
                    </div>
                    <div class="progress-bar" style="margin-top: 0.5rem; height: 6px; background: var(--line); border-radius: var(--radius-sm);">
                        <div class="progress-bar-fill" style="height: 100%; width: ` + strconv.Itoa(int(float64(currentRound)/float64(MaxClarificationRounds)*100)) + `%; background: var(--accent); border-radius: var(--radius-sm);"></div>
                    </div>
                </div>`
		}
		// 实现阶段显示执行进度摘要
		if isImplementation {
			progressSummary := "正在执行..."
			if stageState.Status == "failed" {
				progressSummary = "执行失败 - 查看日志了解详情"
			} else if stageState.Status == "completed" {
				progressSummary = "执行完成"
			}
			return `<div class="progress-section" style="background: linear-gradient(135deg, rgba(139, 92, 246, 0.1), rgba(139, 92, 246, 0.05)); border: 1px solid rgba(139, 92, 246, 0.3); border-radius: var(--radius-lg); padding: 1rem; margin-bottom: 1rem;">
                    <div style="display: flex; justify-content: space-between; align-items: center;">
                        <span style="color: var(--ink); font-weight: 500;">实现进度</span>
                        <span class="mono" style="color: var(--accent);">` + elapsedDisplay + `</span>
                    </div>
                    <div style="margin-top: 0.75rem; color: var(--ink-bright); font-size: 0.9rem;">
                        ` + progressSummary + `
                    </div>
                    <div style="margin-top: 0.5rem;">
                        <small style="color: var(--muted);">点击上方"执行日志"按钮查看详细执行记录</small>
                    </div>
                </div>`
		}
		// 需要人工处理阶段显示提示
		if isNeedsAttention {
			return `<div class="progress-section" style="background: linear-gradient(135deg, rgba(239, 68, 68, 0.1), rgba(239, 68, 68, 0.05)); border: 1px solid rgba(239, 68, 68, 0.3); border-radius: var(--radius-lg); padding: 1rem; margin-bottom: 1rem;">
                    <div style="display: flex; align-items: center; gap: 0.5rem; margin-bottom: 0.5rem;">
                        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: var(--danger);">
                            <circle cx="12" cy="12" r="10"></circle>
                            <line x1="12" y1="8" x2="12" y2="12"></line>
                            <line x1="12" y1="16" x2="12.01" y2="16"></line>
                        </svg>
                        <span style="color: var(--danger); font-weight: 600;">需要人工处理</span>
                    </div>
                    <div style="color: var(--ink); font-size: 0.9rem; margin-top: 0.5rem;">
                        此任务在执行过程中遇到问题，需要人工干预。
                        ` + func() string {
				if stageState.Error != "" {
					return `<div style="margin-top: 0.75rem; background: var(--surface); border-radius: var(--radius-sm); padding: 0.75rem;">
                            <span style="color: var(--muted); font-size: 0.85rem;">错误信息:</span>
                            <pre style="color: var(--danger); font-size: 0.85rem; margin: 0.25rem 0 0 0; white-space: pre-wrap;">` + common.EscapeHTML(stageState.Error) + `</pre>
                        </div>`
				}
				return ""
			}() + `
                    </div>
                    <div style="margin-top: 0.75rem; display: flex; gap: 0.5rem;">
                        <button onclick="resumeTask()" style="padding: 0.5rem 1rem; border: 1px solid var(--accent); border-radius: var(--radius); background: transparent; color: var(--accent); cursor: pointer;">
                            继续执行
                        </button>
                        <button onclick="reclarifyTask()" style="padding: 0.5rem 1rem; border: 1px solid var(--line); border-radius: var(--radius); background: transparent; color: var(--ink); cursor: pointer;">
                            重新澄清
                        </button>
                        <button onclick="abandonTask()" style="padding: 0.5rem 1rem; border: 1px solid var(--danger); border-radius: var(--radius); background: transparent; color: var(--danger); cursor: pointer;">
                            放弃任务
                        </button>
                    </div>
                </div>`
		}
		// 其他阶段显示基本状态
		return `<div class="progress-section" style="background: var(--surface); border-radius: var(--radius-sm); padding: 1rem; margin-bottom: 1rem;">
                    <div style="display: flex; justify-content: space-between; align-items: center;">
                        <span style="color: var(--ink); font-weight: 500;">当前阶段</span>
                        <span style="color: var(--accent);">` + stageDisplay + `</span>
                    </div>
                    ` + func() string {
			if elapsedDisplay != "" {
				return `<div style="margin-top: 0.5rem; color: var(--muted); font-size: 0.85rem;">已用时间: <span class="mono">` + elapsedDisplay + `</span></div>`
			}
			return ""
		}() + `
                </div>`
	}() + `

                <!-- 当前问题 -->
                ` + func() string {
	if isWaitingForAnswer && currentQuestion != "" {
		return `<div class="ai-question-section" style="background: linear-gradient(135deg, rgba(139, 92, 246, 0.1), rgba(139, 92, 246, 0.05)); border: 1px solid rgba(139, 92, 246, 0.3); border-radius: var(--radius-lg); padding: 1.5rem; margin-bottom: 1rem;">
                    <div style="display: flex; align-items: center; gap: 0.5rem; margin-bottom: 0.75rem;">
                        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: var(--accent);">
                            <circle cx="12" cy="12" r="10"></circle>
                            <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"></path>
                            <line x1="12" y1="17" x2="12.01" y2="17"></line>
                        </svg>
                        <span style="font-weight: 600; color: var(--accent);">AI 当前问题</span>
                    </div>
                    <p style="color: var(--ink-bright); font-size: 1rem; line-height: 1.6;">` + common.EscapeHTML(currentQuestion) + `</p>
                    <form id="answer-form" hx-post="/api/v1/` + common.EscapeHTML(issue.Identifier) + `/answer" hx-target="#answer-result" hx-swap="innerHTML" style="margin-top: 1rem;">
                        <div style="margin-bottom: 0.75rem;">
                            <textarea id="answer-input" name="answer" rows="3" required
                                style="width: 100%; padding: 0.75rem 1rem; border: 1px solid var(--line); border-radius: var(--radius); background: var(--bg); color: var(--ink-bright); font-size: 1rem; resize: vertical;"
                                placeholder="输入您的回答..."></textarea>
                        </div>
                        <div style="display: flex; gap: 0.75rem;">
                            <button type="submit" class="btn-primary" style="padding: 0.75rem 1.5rem; border: none; border-radius: var(--radius); background: var(--accent); color: white; font-weight: 500; cursor: pointer;">
                                提交回答
                            </button>
                            <button type="button" class="btn-secondary" onclick="skipClarification()" style="padding: 0.75rem 1.5rem; border: 1px solid var(--line); border-radius: var(--radius); background: transparent; color: var(--muted); cursor: pointer;">
                                跳过澄清
                            </button>
                        </div>
                    </form>
                    <div id="answer-result" style="margin-top: 1rem;"></div>
                </div>`
	}
	return `<div class="ai-question-section" style="background: var(--surface); border: 1px solid var(--line); border-radius: var(--radius-lg); padding: 1.5rem; margin-bottom: 1rem;">
                    <div style="display: flex; align-items: center; gap: 0.5rem; margin-bottom: 0.75rem;">
                        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: var(--muted);">
                            <circle cx="12" cy="12" r="10"></circle>
                            <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"></path>
                            <line x1="12" y1="17" x2="12.01" y2="17"></line>
                        </svg>
                        <span style="font-weight: 600; color: var(--muted);">AI 问题</span>
                    </div>
                    <p style="color: var(--muted); font-size: 1rem;">
                        当前任务不在澄清阶段，或尚未收到 AI 提问。
                    </p>
                </div>`
}() + `

                <!-- 历史问答记录 -->
                <div class="history-section" style="background: var(--surface); border: 1px solid var(--line); border-radius: var(--radius-lg); padding: 1.5rem;">
                    <div style="display: flex; align-items: center; gap: 0.5rem; margin-bottom: 1rem;">
                        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: var(--ink);">
                            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                            <polyline points="14 2 14 8 20 8"></polyline>
                            <line x1="16" y1="13" x2="8" y2="13"></line>
                            <line x1="16" y1="17" x2="8" y2="17"></line>
                            <polyline points="10 9 9 9 8 9"></polyline>
                        </svg>
                        <span style="font-weight: 600; color: var(--ink-bright);">历史问答记录</span>
                    </div>
                    ` + func() string {
	if len(conversationHistory) == 0 {
		return `<p style="color: var(--muted); font-size: 0.9rem;">暂无历史问答记录。</p>`
	}
	return historyHTML
}() + `
                </div>
            </section>

            <!-- 任务描述 -->
            ` + func() string {
	if issue.Description != nil && *issue.Description != "" {
		return `<section class="section-card" style="background: var(--card); border: 1px solid var(--line); border-radius: var(--radius-lg); padding: 1.5rem; margin-top: 1.5rem;">
                <h3 style="font-size: 1rem; font-weight: 600; color: var(--ink-bright); margin-bottom: 0.75rem;">任务描述</h3>
                <div style="background: var(--surface); border-radius: var(--radius-sm); padding: 1rem;">
                    <p style="color: var(--ink); font-size: 0.9rem; line-height: 1.6; white-space: pre-wrap;">` + common.EscapeHTML(*issue.Description) + `</p>
                </div>
            </section>`
	}
	return ""
}() + `
        </section>
    </main>
    <script>
        function skipClarification() {
            if (confirm('确定要跳过澄清阶段吗？跳过后将直接进入下一阶段。')) {
                // 发送跳过请求
                fetch('/api/v1/` + common.EscapeHTML(issue.Identifier) + `/skip-clarification', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    }
                })
                .then(response => response.json())
                .then(data => {
                    if (data.success) {
                        location.reload();
                    } else {
                        alert('跳过失败: ' + (data.error ? data.error.message : '未知错误'));
                    }
                })
                .catch(err => {
                    alert('跳过失败: 网络错误');
                });
            }
        }

        function resumeTask() {
            if (confirm('确定要继续执行此任务吗？')) {
                fetch('/api/tasks/` + common.EscapeHTML(issue.Identifier) + `/resume', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    }
                })
                .then(response => response.json())
                .then(data => {
                    if (data.success) {
                        location.reload();
                    } else {
                        alert('继续执行失败: ' + (data.error ? data.error.message : '未知错误'));
                    }
                })
                .catch(err => {
                    alert('继续执行失败: 网络错误');
                });
            }
        }

        function reclarifyTask() {
            if (confirm('确定要重新澄清此任务的需求吗？')) {
                fetch('/api/tasks/` + common.EscapeHTML(issue.Identifier) + `/reclarify', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    }
                })
                .then(response => response.json())
                .then(data => {
                    if (data.success) {
                        location.reload();
                    } else {
                        alert('重新澄清失败: ' + (data.error ? data.error.message : '未知错误'));
                    }
                })
                .catch(err => {
                    alert('重新澄清失败: 网络错误');
                });
            }
        }

        function abandonTask() {
            if (confirm('确定要放弃此任务吗？此操作不可撤销。')) {
                fetch('/api/tasks/` + common.EscapeHTML(issue.Identifier) + `/abandon', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    }
                })
                .then(response => response.json())
                .then(data => {
                    if (data.success) {
                        location.reload();
                    } else {
                        alert('放弃任务失败: ' + (data.error ? data.error.message : '未知错误'));
                    }
                })
                .catch(err => {
                    alert('放弃任务失败: 网络错误');
                });
            }
        }
    </script>
</body>
</html>`
}

// renderConversationHistory 渲染对话历史
func renderConversationHistory(history []domain.ConversationTurn) string {
	if len(history) == 0 {
		return ""
	}

	html := `<div class="conversation-list" style="max-height: 400px; overflow-y: auto;">`

	// 按轮次分组显示
	roundNum := 0
	for i, turn := range history {
		// 每2条为一个完整轮次（question + answer）
		if turn.Role == "assistant" {
			roundNum++
			html += `<div class="conversation-round" style="margin-bottom: 1rem; padding-bottom: 1rem; border-bottom: 1px solid var(--line);">
                        <div class="question-item" style="margin-bottom: 0.5rem;">
                            <span class="round-badge" style="background: rgba(139, 92, 246, 0.2); color: #8b5cf6; padding: 0.25rem 0.5rem; border-radius: var(--radius-sm); font-size: 0.75rem; font-weight: 600;">Q` + strconv.Itoa(roundNum) + `</span>
                            <span style="color: var(--ink-bright); margin-left: 0.5rem;">` + common.EscapeHTML(turn.Content) + `</span>
                        </div>`
			// 查找对应的回答
			if i+1 < len(history) && history[i+1].Role == "user" {
				answer := history[i+1]
				html += `<div class="answer-item" style="padding-left: 1rem;">
                            <span class="round-badge" style="background: rgba(34, 197, 94, 0.2); color: #22c55e; padding: 0.25rem 0.5rem; border-radius: var(--radius-sm); font-size: 0.75rem; font-weight: 600;">A` + strconv.Itoa(roundNum) + `</span>
                            <span style="color: var(--ink); margin-left: 0.5rem;">` + common.EscapeHTML(answer.Content) + `</span>
                        </div>`
			}
			html += `</div>`
		}
	}

	html += `</div>`
	return html
}

// getStageDisplay 获取阶段显示名称
func getStageDisplay(stageName string) string {
	// 先尝试使用 workflow 包的阶段名称映射（用于任务阶段）
	if displayName := workflow.GetStageDisplayName(workflow.StageName(stageName)); displayName != stageName {
		return displayName
	}
	// 如果不是任务阶段，使用 Kanban 列配置
	stageConfig := common.GetKanbanStageConfig(stageName)
	return stageConfig.Title
}

// getStatusDisplay 获取状态显示名称
func getStatusDisplay(status string) string {
	return workflow.GetStatusDisplayName(workflow.StageStatus(status))
}

// formatDurationForDetail 格式化持续时间用于详情页面显示
func formatDurationForDetail(seconds int64) string {
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

// RenderFilterBar 渲染任务状态筛选器
func RenderFilterBar(currentFilter string) string {
	html := `<section class="filter-bar" id="filter-bar">
            <div class="filter-header">
                <h3 class="filter-title">任务筛选</h3>
                <span class="filter-result-count" id="filter-result-count">共 0 个任务</span>
            </div>
            <div class="filter-buttons">`

	for _, filterState := range common.AllFilterStates() {
		label := common.TaskFilterLabel[filterState]
		stateValue := string(filterState)
		activeClass := ""
		if currentFilter == stateValue || (currentFilter == "" && filterState == common.FilterAll) {
			activeClass = " filter-btn-active"
		}

		html += `<button class="filter-btn` + activeClass + `" data-state="` + stateValue + `" onclick="applyFilter('` + stateValue + `')">
                    <span class="filter-btn-label">` + label + `</span>
                </button>`
	}

	html += `</div>
        </section>`

	return html
}

// RenderTaskList 渲染任务列表（筛选结果）
func RenderTaskList(tasks []common.TaskPayload, filter string) string {
	html := `<section class="task-list-container" id="task-list-container">
            <div class="task-list-header">
                <span class="task-list-title">筛选结果</span>
                <span class="task-list-count">` + strconv.Itoa(len(tasks)) + ` 个任务</span>
            </div>`

	if len(tasks) == 0 {
		html += `<div class="task-list-empty">
                    <p class="empty-state">没有找到匹配的任务</p>
                </div>`
	} else {
		html += `<div class="task-list" id="task-list">`
		for _, task := range tasks {
			html += renderTaskCard(task)
		}
		html += `</div>`
	}

	html += `</section>`

	return html
}

// renderTaskCard 渲染单个任务卡片
func renderTaskCard(task common.TaskPayload) string {
	stateClass := common.StateBadgeClass(task.State)
	priority := "n/a"
	if task.Priority != nil {
		priority = strconv.Itoa(*task.Priority)
	}
	labels := ""
	if len(task.Labels) > 0 {
		for _, l := range task.Labels {
			labels += `<span class="task-label">` + common.EscapeHTML(l) + `</span>`
		}
	}

	return `<div class="task-card">
                <div class="task-card-header">
                    <span class="task-id">` + common.EscapeHTML(task.Identifier) + `</span>
                    <span class="` + stateClass + `">` + common.EscapeHTML(task.State) + `</span>
                </div>
                <div class="task-card-body">
                    <h4 class="task-title">` + common.EscapeHTML(task.Title) + `</h4>
                    <div class="task-meta">
                        <span class="task-meta-item">
                            <span class="task-meta-label">优先级</span>
                            <span class="task-meta-value">` + priority + `</span>
                        </span>
                    </div>` +
		func() string {
			if labels != "" {
				return `<div class="task-labels">` + labels + `</div>`
			}
			return ""
		}() + `
                    <div class="task-card-footer">
                        <a class="task-link" href="/api/v1/` + common.EscapeHTML(task.Identifier) + `">查看详情 →</a>
                    </div>
                </div>
            </div>`
}

// RenderRunningCard 渲染单个运行中任务的 Kanban 卡片
func RenderRunningCard(entry *domain.RunningEntry, now time.Time) string {
	issueState := ""
	if entry.Issue != nil {
		issueState = entry.Issue.State
	}
	stateClass := common.StateBadgeClass(issueState)
	runtimeTurns := common.FormatRuntimeAndTurns(entry.StartedAt, entry.TurnCount, now)

	lastEvent := "n/a"
	lastEventAt := ""
	var tokens common.Tokens
	if entry.Session != nil {
		if entry.Session.LastCodexEvent != nil {
			lastEvent = *entry.Session.LastCodexEvent
		}
		if entry.Session.LastCodexTimestamp != nil {
			lastEventAt = entry.Session.LastCodexTimestamp.Format("15:04:05")
		}
		tokens = common.Tokens{
			InputTokens:  entry.Session.CodexInputTokens,
			OutputTokens: entry.Session.CodexOutputTokens,
			TotalTokens:  entry.Session.CodexTotalTokens,
		}
	}

	sessionID := ""
	if entry.Session != nil {
		sessionID = entry.Session.SessionID
	}

	// 计算token进度条
	tokenPercent := 0
	if tokens.TotalTokens > 0 {
		tokenPercent = int((float64(tokens.OutputTokens) / float64(tokens.TotalTokens)) * 100)
		if tokenPercent > 100 {
			tokenPercent = 100
		}
	}

	return `
                <div class="kanban-card">
                    <div class="card-header">
                        <span class="issue-id">` + entry.Identifier + `</span>
                        <span class="` + stateClass + `">` + issueState + `</span>
                    </div>
                    <div class="card-body">
                        <div class="card-row">
                            <span class="card-label">Session</span>
                            <span>` + func() string {
		if sessionID != "" {
			return `<button type="button" class="subtle-button" data-label="复制" data-copy="` + sessionID + `" onclick="navigator.clipboard.writeText(this.dataset.copy); this.textContent = '已复制'; clearTimeout(this._copyTimer); this._copyTimer = setTimeout(() => { this.textContent = this.dataset.label }, 1200);">复制</button>`
		}
		return `<span class="muted">n/a</span>`
	}() + `</span>
                        </div>
                        <div class="card-row">
                            <span class="card-label">Runtime</span>
                            <span class="card-value mono">` + runtimeTurns + `</span>
                        </div>
                        <div class="card-divider"></div>
                        <div class="card-row">
                            <span class="card-label">Last Event</span>
                            <span class="card-value" title="` + common.EscapeHTML(lastEvent) + `">` + common.EscapeHTML(lastEvent) + `</span>
                        </div>` + func() string {
		if lastEventAt != "" {
			return `
                        <div class="card-row">
                            <span class="card-label">At</span>
                            <span class="card-value mono">` + lastEventAt + `</span>
                        </div>`
		}
		return ""
	}() + `
                        <div class="card-divider"></div>
                        <div class="card-row">
                            <span class="card-label">Tokens</span>
                            <span class="card-value mono">` + common.FormatInt(tokens.TotalTokens) + `</span>
                        </div>
                        <div class="token-bar">
                            <div class="token-bar-fill" style="width: ` + strconv.Itoa(tokenPercent) + `%;"></div>
                            <div class="token-bar-bg"></div>
                        </div>
                        <div class="card-row">
                            <span class="card-label">In / Out</span>
                            <span class="card-value mono muted">` + common.FormatInt(tokens.InputTokens) + ` / ` + common.FormatInt(tokens.OutputTokens) + `</span>
                        </div>
                        <div class="card-divider"></div>
                        <div class="card-row" style="margin-top: 0.5rem;">
                            <a class="issue-link" href="/api/v1/` + entry.Identifier + `">查看 JSON 详情 →</a>
                        </div>
                        <div class="card-row" style="margin-top: 0.5rem;">
                            <button type="button" class="cancel-button" data-identifier="` + entry.Identifier + `" onclick="showCancelConfirm(this.dataset.identifier)">取消任务</button>
                        </div>
                    </div>
                </div>`
}

// RenderRetryCard 渲染单个重试任务的 Kanban 卡片
func RenderRetryCard(entry *domain.RetryEntry) string {
	dueAt := time.UnixMilli(entry.DueAtMs).Format("15:04:05")
	errMsg := "n/a"
	if entry.Error != nil && *entry.Error != "" {
		errMsg = *entry.Error
		if len(errMsg) > 50 {
			errMsg = errMsg[:50] + "..."
		}
	}

	return `
                <div class="kanban-card">
                    <div class="card-header">
                        <span class="issue-id">` + entry.Identifier + `</span>
                        <span class="state-badge state-badge-warning">Retry #` + strconv.Itoa(entry.Attempt) + `</span>
                    </div>
                    <div class="card-body">
                        <div class="card-row">
                            <span class="card-label">Attempt</span>
                            <span class="card-value">第 ` + strconv.Itoa(entry.Attempt) + ` 次重试</span>
                        </div>
                        <div class="card-row">
                            <span class="card-label">Due At</span>
                            <span class="card-value mono">` + dueAt + `</span>
                        </div>
                        <div class="card-divider"></div>
                        <div class="card-row">
                            <span class="card-label">Error</span>
                            <span class="card-value" title="` + common.EscapeHTML(errMsg) + `">` + common.EscapeHTML(errMsg) + `</span>
                        </div>
                        <div class="card-row" style="margin-top: 0.5rem;">
                            <a class="issue-link" href="/api/v1/` + entry.Identifier + `">查看 JSON 详情 →</a>
                        </div>
                        <div class="card-row" style="margin-top: 0.5rem;">
                            <button type="button" class="cancel-button" data-identifier="` + entry.Identifier + `" onclick="showCancelConfirm(this.dataset.identifier)">取消重试</button>
                        </div>
                    </div>
                </div>`
}

// RenderRunningSessionsKanban 渲染运行中任务的 Kanban 列
func RenderRunningSessionsKanban(state *domain.OrchestratorState, now time.Time) string {
	html := `<div class="kanban-column kanban-column-running">
            <div class="kanban-header">
                <div class="kanban-header-icon">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <polygon points="5 3 19 12 5 21 5 3"></polygon>
                    </svg>
                </div>
                <span class="kanban-header-title">Running</span>
                <span class="kanban-header-count">` + strconv.Itoa(len(state.Running)) + `</span>
            </div>
            <div class="kanban-cards" id="running-cards">`

	if len(state.Running) == 0 {
		html += `<p class="empty-state">暂无活跃 Session</p>`
	} else {
		for _, entry := range state.Running {
			html += RenderRunningCard(entry, now)
		}
	}

	html += `</div></div>`
	return html
}

// RenderRetryQueueKanban 渲染重试队列的 Kanban 列
func RenderRetryQueueKanban(state *domain.OrchestratorState) string {
	html := `<div class="kanban-column kanban-column-retrying">
            <div class="kanban-header">
                <div class="kanban-header-icon">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"></path>
                        <path d="M3 3v5h5"></path>
                        <path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16"></path>
                        <path d="M16 21h5v-5"></path>
                    </svg>
                </div>
                <span class="kanban-header-title">Retrying</span>
                <span class="kanban-header-count">` + strconv.Itoa(len(state.RetryAttempts)) + `</span>
            </div>
            <div class="kanban-cards" id="retrying-cards">`

	if len(state.RetryAttempts) == 0 {
		html += `<p class="empty-state">当前没有等待重试的 Issue</p>`
	} else {
		for _, entry := range state.RetryAttempts {
			html += RenderRetryCard(entry)
		}
	}

	html += `</div></div>`
	return html
}

// RenderStageKanban 渲染按阶段分列的看板
func RenderStageKanban(payload *common.KanbanPayload) string {
	var html string

	for _, col := range payload.Columns {
		html += renderStageColumn(col)
	}

	return html
}

// renderStageColumn 渲染单个阶段列
func renderStageColumn(col common.KanbanColumn) string {
	stageClass := "column-" + col.ID

	html := `<div class="column ` + stageClass + `" data-stage="` + col.ID + `">
            <div class="column-header">
                <span class="column-title">` + col.Icon + ` ` + col.Title + `</span>
                <span class="task-count">` + strconv.Itoa(col.TaskCount) + `</span>
            </div>
            <div class="task-list" id="stage-cards-` + col.ID + `" data-stage="` + col.ID + `">`

	if col.TaskCount == 0 {
		html += `<p class="empty-state">暂无任务</p>`
	} else {
		for _, task := range col.Tasks {
			html += renderStageKanbanCard(task, col.Color)
		}
	}

	html += `</div></div>`
	return html
}

// renderStageKanbanCard 渲染看板任务卡片
func renderStageKanbanCard(task common.KanbanTaskPayload, colColor string) string {
	// 标题处理
	title := task.Title
	if title == "" {
		title = task.IssueIdentifier
	}
	if len(title) > 60 {
		title = title[:60] + "..."
	}

	// 优先级映射 (基于 state)
	priorityClass := "priority-medium"
	priorityLabel := "中优"
	if task.State == "needs_attention" {
		priorityClass = "priority-high"
		priorityLabel = "高优"
	} else if task.State == "completed" || task.State == "cancelled" {
		priorityClass = "priority-low"
		priorityLabel = "低优"
	}

	// 时间显示
	timeDisplay := "刚刚"
	if task.LastEventAt != "" {
		timeDisplay = task.LastEventAt
	}

	return `<div class="task-card" data-task-id="` + task.IssueIdentifier + `" data-stage="` + task.Stage + `">
            <div class="task-id">` + common.EscapeHTML(task.IssueIdentifier) + `</div>
            <div class="task-title">` + common.EscapeHTML(title) + `</div>
            <div class="task-meta">
                <span class="task-priority ` + priorityClass + `">` + priorityLabel + `</span>
                <span>` + common.EscapeHTML(timeDisplay) + `</span>
            </div>
        </div>`
}

// RenderStageKanbanScript 渲染看板 SSE 脚本
func RenderStageKanbanScript() string {
	return `
    // SSE 任务状态变更事件监听
    eventSource.addEventListener('task_update', function(e) {
        try {
            const data = JSON.parse(e.data);
            handleTaskUpdate(data);
        } catch (err) {
            console.error('Failed to parse task_update data:', err);
        }
    });

    function handleTaskUpdate(data) {
        const taskId = data.task_id;
        const oldStage = data.old_stage;
        const newStage = data.new_stage;
        const task = data.task;

        // 查找旧卡片
        const oldCard = document.querySelector('.kanban-card[data-task-id="' + taskId + '"]');

        // 如果卡片存在，执行动画移动
        if (oldCard) {
            // 添加移出动画
            oldCard.classList.add('card-exiting');

            setTimeout(() => {
                // 从旧列移除
                oldCard.remove();

                // 更新列计数
                updateColumnCount(oldStage);

                // 添加到新列
                const newColumnCards = document.querySelector('#stage-cards-' + newStage);
                if (newColumnCards) {
                    const newCard = createTaskCard(task);
                    newCard.classList.add('card-entering');
                    newColumnCards.appendChild(newCard);

                    // 移除动画类
                    setTimeout(() => {
                        newCard.classList.remove('card-entering');
                    }, 300);

                    // 更新新列计数
                    updateColumnCount(newStage);
                }
            }, 200);
        } else {
            // 新任务，直接添加
            const newColumnCards = document.querySelector('#stage-cards-' + newStage);
            if (newColumnCards) {
                const newCard = createTaskCard(task);
                newCard.classList.add('card-entering');
                newColumnCards.appendChild(newCard);

                setTimeout(() => {
                    newCard.classList.remove('card-entering');
                }, 300);

                updateColumnCount(newStage);

                // 移除空状态提示
                const emptyState = newColumnCards.querySelector('.empty-state');
                if (emptyState) {
                    emptyState.remove();
                }
            }
        }

        // 闪烁指示器
        const indicator = document.getElementById('live-indicator');
        indicator.style.animation = 'pulse 0.3s ease';
        setTimeout(() => indicator.style.animation = '', 300);
    }

    function updateColumnCount(stageId) {
        const column = document.querySelector('.kanban-column[data-stage="' + stageId + '"]');
        if (column) {
            const cards = column.querySelectorAll('.kanban-card');
            const countBadge = column.querySelector('.kanban-header-count');
            if (countBadge) {
                countBadge.textContent = cards.length;
            }

            // 如果没有卡片，显示空状态
            const cardsContainer = column.querySelector('.kanban-cards');
            if (cards.length === 0 && cardsContainer) {
                const emptyState = cardsContainer.querySelector('.empty-state');
                if (!emptyState) {
                    cardsContainer.innerHTML = '<p class="empty-state">暂无任务</p>';
                }
            }
        }
    }

    function createTaskCard(task) {
        const stateClass = getStateBadgeClass(task.state);
        const stageConfig = getStageConfig(task.stage);
        const tokenPercent = task.tokens && task.tokens.total_tokens > 0
            ? Math.min(100, Math.round((task.tokens.output_tokens / task.tokens.total_tokens) * 100))
            : 0;

        let title = task.title || task.issue_identifier;
        if (title.length > 50) title = title.substring(0, 50) + '...';

        let cardHtml = '<div class="kanban-card" data-task-id="' + escapeHtml(task.issue_identifier) + '" data-stage="' + task.stage + '" style="--card-accent: ' + stageConfig.color + ';">' +
            '<div class="card-header">' +
            '<span class="issue-id">' + escapeHtml(task.issue_identifier) + '</span>' +
            '<span class="' + stateClass + '">' + escapeHtml(task.state || 'unknown') + '</span>' +
            '</div>' +
            '<div class="card-title">' + escapeHtml(title) + '</div>' +
            '<div class="card-body">';

        if (task.session_id) {
            cardHtml += '<div class="card-row"><span class="card-label">Session</span><span><button type="button" class="subtle-button" onclick="copyId(this, \'' + escapeHtml(task.session_id) + '\')">复制</button></span></div>';
        }

        if (task.runtime_turns) {
            cardHtml += '<div class="card-row"><span class="card-label">Runtime</span><span class="card-value mono">' + escapeHtml(task.runtime_turns) + '</span></div>';
        }

        if (task.last_event) {
            cardHtml += '<div class="card-divider"></div><div class="card-row"><span class="card-label">Last Event</span><span class="card-value" title="' + escapeHtml(task.last_event) + '">' + escapeHtml(task.last_event) + '</span></div>';
        }

        if (task.attempt > 0) {
            cardHtml += '<div class="card-divider"></div><div class="card-row"><span class="card-label">Attempt</span><span class="card-value">第 ' + task.attempt + ' 次</span></div>';
        }

        if (task.error) {
            let errDisplay = task.error.length > 50 ? task.error.substring(0, 50) + '...' : task.error;
            cardHtml += '<div class="card-row"><span class="card-label">Error</span><span class="card-value card-value-error" title="' + escapeHtml(task.error) + '">' + escapeHtml(errDisplay) + '</span></div>';
        }

        if (task.tokens && task.tokens.total_tokens > 0) {
            cardHtml += '<div class="card-divider"></div>' +
                '<div class="card-row"><span class="card-label">Tokens</span><span class="card-value mono">' + formatNumber(task.tokens.total_tokens) + '</span></div>' +
                '<div class="token-bar"><div class="token-bar-fill" style="width: ' + tokenPercent + '%;"></div><div class="token-bar-bg"></div></div>' +
                '<div class="card-row"><span class="card-label">In / Out</span><span class="card-value mono muted">' + formatNumber(task.tokens.input_tokens) + ' / ' + formatNumber(task.tokens.output_tokens) + '</span></div>';
        }

        cardHtml += '<div class="card-row" style="margin-top: 0.5rem;"><a class="issue-link" href="/api/v1/' + escapeHtml(task.issue_identifier) + '">查看详情 →</a></div></div></div>';

        const div = document.createElement('div');
        div.innerHTML = cardHtml.trim();
        return div.firstChild;
    }

    // 阶段配置
    const stageConfigs = {
        'backlog': { title: '待开始', color: '#6b7280' },
        'clarification': { title: '需求澄清', color: '#8b5cf6' },
        'bdd_review': { title: '待审核 BDD', color: '#f59e0b' },
        'architecture_review': { title: '待审核架构', color: '#ec4899' },
        'implementation': { title: '实现中', color: '#22d3ee' },
        'verification': { title: '待验收', color: '#10b981' },
        'completed': { title: '完成', color: '#4ade80' },
        'needs_attention': { title: '待人工处理', color: '#f87171' },
        'cancelled': { title: '已取消', color: '#9ca3af' }
    };

    function getStageConfig(stageId) {
        return stageConfigs[stageId] || stageConfigs['backlog'];
    }
`
}

// RenderDashboardHTML 渲染完整的仪表板 HTML
func RenderDashboardHTML(state *domain.OrchestratorState, now time.Time) string {
	html := `<!DOCTYPE html>
<html lang="zh">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Symphony · 任务看板</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=Fira+Code:wght@400;500&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="/dashboard.css">
</head>
<body>
    <main class="app-shell">
        <section class="dashboard-shell">
            <header class="hero-card">
                <div class="hero-grid">
                    <div>
                        <p class="eyebrow">Symphony Orchestrator</p>
                        <h1 class="hero-title">任务看板</h1>
                        <p class="hero-copy">实时监控运行中的 Agent 会话、重试队列状态和 Token 使用量。</p>
                    </div>
                    <div class="status-stack">
                        <a href="/tasks/new" class="create-task-btn" style="display: inline-flex; align-items: center; gap: 0.5rem; padding: 0.5rem 1rem; background: var(--accent); color: white; border-radius: var(--radius); text-decoration: none; font-weight: 500; font-size: 0.9rem; transition: all 0.2s;">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                                <line x1="12" y1="5" x2="12" y2="19"></line>
                                <line x1="5" y1="12" x2="19" y2="12"></line>
                            </svg>
                            创建需求
                        </a>
                        <span class="status-badge status-badge-live" id="live-indicator">
                            <span class="status-badge-dot"></span>
                            Live
                        </span>
                        <span class="status-badge status-badge-offline">
                            <span class="status-badge-dot"></span>
                            Offline
                        </span>
                    </div>
                </div>
            </header>

            <section class="metric-grid" id="metrics">
                <article class="metric-card">
                    <p class="metric-label">Running</p>
                    <p class="metric-value numeric" id="metric-running">` + strconv.Itoa(len(state.Running)) + `</p>
                    <p class="metric-detail">活跃会话</p>
                </article>

                <article class="metric-card">
                    <p class="metric-label">Retrying</p>
                    <p class="metric-value numeric" id="metric-retrying">` + strconv.Itoa(len(state.RetryAttempts)) + `</p>
                    <p class="metric-detail">等待重试</p>
                </article>

                <article class="metric-card">
                    <p class="metric-label">Total Tokens</p>
                    <p class="metric-value numeric" id="metric-tokens">` + common.FormatInt(state.CodexTotals.TotalTokens) + `</p>
                    <p class="metric-detail numeric" id="metric-tokens-detail">In ` + common.FormatInt(state.CodexTotals.InputTokens) + ` / Out ` + common.FormatInt(state.CodexTotals.OutputTokens) + `</p>
                </article>

                <article class="metric-card">
                    <p class="metric-label">Runtime</p>
                    <p class="metric-value numeric" id="metric-runtime">` + common.FormatRuntimeSeconds(common.TotalRuntimeSeconds(state, now)) + `</p>
                    <p class="metric-detail">总运行时长</p>
                </article>
            </section>

            <section class="kanban-container" id="kanban">
                ` + RenderRunningSessionsKanban(state, now) + `
                ` + RenderRetryQueueKanban(state) + `
            </section>

            ` + RenderFilterBar("") + `
            <section class="task-list-container" id="task-list-container">
                <div class="task-list-placeholder">
                    <p class="placeholder-text">点击上方筛选按钮查看任务列表</p>
                </div>
            </section>

            <section class="section-card" style="background: var(--card); border: 1px solid var(--line); border-radius: var(--radius-lg); padding: 1.25rem;">
                <div class="section-header">
                    <div>
                        <h2 class="section-title" style="font-size: 1rem; color: var(--ink-bright);">Rate Limits</h2>
                        <p class="section-copy" style="color: var(--muted); font-size: 0.85rem;">上游 API 速率限制快照</p>
                    </div>
                </div>
                <pre class="code-panel" id="rate-limits">` + common.PrettyValue(state.CodexRateLimits) + `</pre>
            </section>
        </section>
    </main>
    <script>
    // SSE 实时更新
    const eventSource = new EventSource('/events');

    eventSource.addEventListener('state', function(e) {
        document.body.classList.add('hx-connected');
        try {
            const data = JSON.parse(e.data);
            updateDashboard(data);
        } catch (err) {
            console.error('Failed to parse SSE data:', err);
        }
    });

    eventSource.onerror = function(e) {
        console.error('SSE connection error:', e);
        document.body.classList.remove('hx-connected');
    };

    function updateDashboard(data) {
        // 更新指标
        document.getElementById('metric-running').textContent = data.counts.running;
        document.getElementById('metric-retrying').textContent = data.counts.retrying;
        document.getElementById('metric-tokens').textContent = formatNumber(data.codex_totals.total_tokens);
        document.getElementById('metric-tokens-detail').textContent = 'In ' + formatNumber(data.codex_totals.input_tokens) + ' / Out ' + formatNumber(data.codex_totals.output_tokens);
        document.getElementById('rate-limits').textContent = JSON.stringify(data.rate_limits || 'n/a', null, 2);

        // 更新 Kanban
        document.getElementById('kanban').innerHTML = renderKanban(data.running, data.retrying);

        // 闪烁 Live 指示器
        const indicator = document.getElementById('live-indicator');
        indicator.style.animation = 'pulse 0.3s ease';
        setTimeout(() => indicator.style.animation = '', 300);
    }

    function formatNumber(n) {
        if (n >= 1000000) return (n / 1000000).toFixed(1) + 'M';
        if (n >= 1000) return (n / 1000).toFixed(1) + 'K';
        return n.toString();
    }

    function renderKanban(running, retrying) {
        return renderRunningColumn(running) + renderRetryingColumn(retrying);
    }

    function renderRunningColumn(running) {
        let headerCount = running ? running.length : 0;
        let html = '<div class="kanban-column kanban-column-running">' +
            '<div class="kanban-header">' +
            '<div class="kanban-header-icon"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="5 3 19 12 5 21 5 3"></polygon></svg></div>' +
            '<span class="kanban-header-title">Running</span>' +
            '<span class="kanban-header-count">' + headerCount + '</span>' +
            '</div><div class="kanban-cards" id="running-cards">';

        if (!running || running.length === 0) {
            html += '<p class="empty-state">暂无活跃 Session</p>';
        } else {
            running.forEach(entry => {
                html += renderRunningCard(entry);
            });
        }

        html += '</div></div>';
        return html;
    }

    function renderRetryingColumn(retrying) {
        let headerCount = retrying ? retrying.length : 0;
        let html = '<div class="kanban-column kanban-column-retrying">' +
            '<div class="kanban-header">' +
            '<div class="kanban-header-icon"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"></path><path d="M3 3v5h5"></path><path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16"></path><path d="M16 21h5v-5"></path></svg></div>' +
            '<span class="kanban-header-title">Retrying</span>' +
            '<span class="kanban-header-count">' + headerCount + '</span>' +
            '</div><div class="kanban-cards" id="retrying-cards">';

        if (!retrying || retrying.length === 0) {
            html += '<p class="empty-state">当前没有等待重试的 Issue</p>';
        } else {
            retrying.forEach(entry => {
                html += renderRetryCard(entry);
            });
        }

        html += '</div></div>';
        return html;
    }

    function renderRunningCard(entry) {
        const stateClass = getStateBadgeClass(entry.state);
        const tokenPercent = entry.tokens.total_tokens > 0
            ? Math.min(100, Math.round((entry.tokens.output_tokens / entry.tokens.total_tokens) * 100))
            : 0;

        return '<div class="kanban-card">' +
            '<div class="card-header">' +
            '<span class="issue-id">' + escapeHtml(entry.issue_identifier) + '</span>' +
            '<span class="' + stateClass + '">' + escapeHtml(entry.state || 'unknown') + '</span>' +
            '</div>' +
            '<div class="card-body">' +
            '<div class="card-row">' +
            '<span class="card-label">Session</span>' +
            '<span>' + (entry.session_id ? '<button type="button" class="subtle-button" onclick="copyId(this, \\'' + escapeHtml(entry.session_id) + '\\')">复制</button>' : '<span class="muted">n/a</span>') + '</span>' +
            '</div>' +
            '<div class="card-row">' +
            '<span class="card-label">Runtime</span>' +
            '<span class="card-value mono">' + escapeHtml(entry.runtime_turns || 'n/a') + '</span>' +
            '</div>' +
            '<div class="card-divider"></div>' +
            '<div class="card-row">' +
            '<span class="card-label">Last Event</span>' +
            '<span class="card-value" title="' + escapeHtml(entry.last_event || 'n/a') + '">' + escapeHtml(entry.last_event || 'n/a') + '</span>' +
            '</div>' +
            (entry.last_event_at ? '<div class="card-row"><span class="card-label">At</span><span class="card-value mono">' + escapeHtml(entry.last_event_at) + '</span></div>' : '') +
            '<div class="card-divider"></div>' +
            '<div class="card-row">' +
            '<span class="card-label">Tokens</span>' +
            '<span class="card-value mono">' + formatNumber(entry.tokens.total_tokens) + '</span>' +
            '</div>' +
            '<div class="token-bar">' +
            '<div class="token-bar-fill" style="width: ' + tokenPercent + '%;"></div>' +
            '<div class="token-bar-bg"></div>' +
            '</div>' +
            '<div class="card-row">' +
            '<span class="card-label">In / Out</span>' +
            '<span class="card-value mono muted">' + formatNumber(entry.tokens.input_tokens) + ' / ' + formatNumber(entry.tokens.output_tokens) + '</span>' +
            '</div>' +
            '<div class="card-row" style="margin-top: 0.5rem;">' +
            '<a class="issue-link" href="/api/v1/' + escapeHtml(entry.issue_identifier) + '">查看 JSON 详情 →</a>' +
            '</div>' +
            '<div class="card-row" style="margin-top: 0.5rem;">' +
            '<button type="button" class="cancel-button" onclick="showCancelConfirm(\\'' + escapeHtml(entry.issue_identifier) + '\\')">取消任务</button>' +
            '</div>' +
            '</div>' +
            '</div>';
    }

    function renderRetryCard(entry) {
        const errMsg = entry.error || 'n/a';
        const displayErr = errMsg.length > 50 ? errMsg.substring(0, 50) + '...' : errMsg;

        return '<div class="kanban-card">' +
            '<div class="card-header">' +
            '<span class="issue-id">' + escapeHtml(entry.issue_identifier) + '</span>' +
            '<span class="state-badge state-badge-warning">Retry #' + entry.attempt + '</span>' +
            '</div>' +
            '<div class="card-body">' +
            '<div class="card-row">' +
            '<span class="card-label">Attempt</span>' +
            '<span class="card-value">第 ' + entry.attempt + ' 次重试</span>' +
            '</div>' +
            '<div class="card-row">' +
            '<span class="card-label">Due At</span>' +
            '<span class="card-value mono">' + escapeHtml(entry.due_at || 'n/a') + '</span>' +
            '</div>' +
            '<div class="card-divider"></div>' +
            '<div class="card-row">' +
            '<span class="card-label">Error</span>' +
            '<span class="card-value" title="' + escapeHtml(errMsg) + '">' + escapeHtml(displayErr) + '</span>' +
            '</div>' +
            '<div class="card-row" style="margin-top: 0.5rem;">' +
            '<a class="issue-link" href="/api/v1/' + escapeHtml(entry.issue_identifier) + '">查看 JSON 详情 →</a>' +
            '</div>' +
            '<div class="card-row" style="margin-top: 0.5rem;">' +
            '<button type="button" class="cancel-button" onclick="showCancelConfirm(\\'' + escapeHtml(entry.issue_identifier) + '\\')">取消重试</button>' +
            '</div>' +
            '</div>' +
            '</div>';
    }

    function getStateBadgeClass(state) {
        if (!state) return 'state-badge';
        const s = state.toLowerCase();
        if (s.includes('progress') || s.includes('running') || s.includes('active')) {
            return 'state-badge state-badge-active';
        }
        if (s.includes('blocked') || s.includes('error') || s.includes('failed')) {
            return 'state-badge state-badge-danger';
        }
        if (s.includes('todo') || s.includes('queued') || s.includes('pending') || s.includes('retry')) {
            return 'state-badge state-badge-warning';
        }
        return 'state-badge';
    }

    function escapeHtml(s) {
        if (!s) return '';
        return s.toString()
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#39;');
    }

    function copyId(btn, id) {
        navigator.clipboard.writeText(id);
        btn.textContent = '已复制';
        clearTimeout(btn._copyTimer);
        btn._copyTimer = setTimeout(() => btn.textContent = '复制', 1200);
    }

    // 任务筛选功能
    let currentFilter = 'all';

    function applyFilter(filterState) {
        currentFilter = filterState;

        // 更新按钮状态
        document.querySelectorAll('.filter-btn').forEach(btn => {
            btn.classList.remove('filter-btn-active');
            if (btn.dataset.state === filterState) {
                btn.classList.add('filter-btn-active');
            }
        });

        // 发送请求获取筛选后的任务列表
        fetch('/api/v1/tasks?state=' + filterState)
            .then(response => response.json())
            .then(data => {
                updateTaskList(data);
            })
            .catch(err => {
                console.error('Failed to fetch tasks:', err);
                document.getElementById('task-list-container').innerHTML =
                    '<div class="task-list-error"><p class="error-text">获取任务列表失败</p></div>';
            });
    }

    function updateTaskList(data) {
        // 更新筛选结果数量
        document.getElementById('filter-result-count').textContent =
            '共 ' + data.total_count + ' 个任务';

        // 渲染任务列表
        const container = document.getElementById('task-list-container');
        if (data.tasks.length === 0) {
            container.innerHTML =
                '<div class="task-list-header"><span class="task-list-title">筛选结果</span><span class="task-list-count">0 个任务</span></div>' +
                '<div class="task-list-empty"><p class="empty-state">没有找到匹配的任务</p></div>';
        } else {
            let html = '<div class="task-list-header"><span class="task-list-title">筛选结果</span><span class="task-list-count">' + data.total_count + ' 个任务</span></div>';
            html += '<div class="task-list" id="task-list">';
            data.tasks.forEach(task => {
                html += renderTaskCard(task);
            });
            html += '</div>';
            container.innerHTML = html;
        }
    }

    function renderTaskCard(task) {
        const stateClass = getStateBadgeClass(task.state);
        const priority = task.priority || 'n/a';
        let labelsHtml = '';
        if (task.labels && task.labels.length > 0) {
            task.labels.forEach(l => {
                labelsHtml += '<span class="task-label">' + escapeHtml(l) + '</span>';
            });
        }

        return '<div class="task-card">' +
            '<div class="task-card-header">' +
            '<span class="task-id">' + escapeHtml(task.identifier) + '</span>' +
            '<span class="' + stateClass + '">' + escapeHtml(task.state) + '</span>' +
            '</div>' +
            '<div class="task-card-body">' +
            '<h4 class="task-title">' + escapeHtml(task.title) + '</h4>' +
            '<div class="task-meta">' +
            '<span class="task-meta-item">' +
            '<span class="task-meta-label">优先级</span>' +
            '<span class="task-meta-value">' + priority + '</span>' +
            '</span>' +
            '</div>' +
            (labelsHtml ? '<div class="task-labels">' + labelsHtml + '</div>' : '') +
            '<div class="task-card-footer">' +
            '<a class="task-link" href="/api/v1/' + escapeHtml(task.identifier) + '">查看详情 →</a>' +
            '</div>' +
            '</div>' +
            '</div>';
    }

    // 页面加载时自动加载全部任务
    document.addEventListener('DOMContentLoaded', function() {
        applyFilter('all');
    });

    // 取消任务功能
    let pendingCancelIdentifier = null;

    function showCancelConfirm(identifier) {
        pendingCancelIdentifier = identifier;

        // 获取确认信息
        fetch('/api/v1/' + identifier + '/cancel/confirm')
            .then(response => response.json())
            .then(data => {
                if (data.error) {
                    alert('获取取消信息失败: ' + data.error.message);
                    return;
                }

                // 显示确认对话框
                const modal = document.getElementById('cancel-modal');
                const modalIdentifier = document.getElementById('modal-identifier');
                const modalWarning = document.getElementById('modal-warning');
                const modalTaskType = document.getElementById('modal-task-type');

                modalIdentifier.textContent = data.identifier;
                modalWarning.textContent = data.warning;

                if (data.task_type === 'running') {
                    modalTaskType.innerHTML = '运行中任务<br>Session ID: ' + (data.session_id || 'n/a') + '<br>Turns: ' + (data.turn_count || 0);
                } else if (data.task_type === 'retrying') {
                    modalTaskType.innerHTML = '重试队列任务<br>Attempt: ' + data.attempt;
                }

                modal.style.display = 'flex';
            })
            .catch(err => {
                console.error('Failed to get cancel confirmation:', err);
                // 直接显示简化确认对话框
                pendingCancelIdentifier = identifier;
                document.getElementById('modal-identifier').textContent = identifier;
                document.getElementById('modal-task-type').textContent = '任务';
                document.getElementById('modal-warning').textContent = '取消操作不可逆，正在执行的 Agent 进程将被终止';
                document.getElementById('cancel-modal').style.display = 'flex';
            });
    }

    function hideCancelConfirm() {
        document.getElementById('cancel-modal').style.display = 'none';
        pendingCancelIdentifier = null;
    }

    function executeCancel() {
        if (!pendingCancelIdentifier) {
            return;
        }

        const identifier = pendingCancelIdentifier;
        hideCancelConfirm();

        // 执行取消
        fetch('/api/v1/' + identifier + '/cancel', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        })
        .then(response => response.json())
        .then(data => {
            if (data.error) {
                alert('取消失败: ' + data.error.message);
                return;
            }

            // 显示成功消息
            const toast = document.getElementById('cancel-toast');
            toast.textContent = '任务 ' + identifier + ' 已取消';
            toast.style.display = 'block';

            setTimeout(() => {
                toast.style.display = 'none';
            }, 3000);
        })
        .catch(err => {
            console.error('Failed to cancel task:', err);
            alert('取消失败: 网络错误');
        });
    }

    // 点击模态框外部关闭
    document.getElementById('cancel-modal').addEventListener('click', function(e) {
        if (e.target === this) {
            hideCancelConfirm();
        }
    });
    </script>

    <!-- 取消任务确认对话框 -->
    <div id="cancel-modal" class="modal" style="display: none;">
        <div class="modal-content">
            <div class="modal-header">
                <h3 class="modal-title">确认取消任务</h3>
                <button type="button" class="modal-close" onclick="hideCancelConfirm()">×</button>
            </div>
            <div class="modal-body">
                <div class="modal-info">
                    <div class="modal-row">
                        <span class="modal-label">任务标识符</span>
                        <span class="modal-value" id="modal-identifier"></span>
                    </div>
                    <div class="modal-row">
                        <span class="modal-label">任务类型</span>
                        <span class="modal-value" id="modal-task-type"></span>
                    </div>
                </div>
                <div class="modal-warning-box">
                    <span class="warning-icon">⚠️</span>
                    <span class="warning-text" id="modal-warning"></span>
                </div>
            </div>
            <div class="modal-footer">
                <button type="button" class="modal-btn modal-btn-cancel" onclick="hideCancelConfirm()">取消</button>
                <button type="button" class="modal-btn modal-btn-confirm" onclick="executeCancel()">确认取消</button>
            </div>
        </div>
    </div>

    <!-- 成功提示 -->
    <div id="cancel-toast" class="toast" style="display: none;"></div>

    <!-- 模态框样式 -->
    <style>
    .modal {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.5);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 1000;
    }
    .modal-content {
        background: var(--card);
        border: 1px solid var(--line);
        border-radius: var(--radius-lg);
        max-width: 400px;
        width: 90%;
        padding: 1.5rem;
    }
    .modal-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 1rem;
    }
    .modal-title {
        font-size: 1.1rem;
        font-weight: 600;
        color: var(--ink-bright);
    }
    .modal-close {
        background: none;
        border: none;
        font-size: 1.5rem;
        color: var(--muted);
        cursor: pointer;
        padding: 0;
        line-height: 1;
    }
    .modal-close:hover {
        color: var(--ink-bright);
    }
    .modal-body {
        margin-bottom: 1.5rem;
    }
    .modal-info {
        background: var(--surface);
        border-radius: var(--radius-sm);
        padding: 1rem;
        margin-bottom: 1rem;
    }
    .modal-row {
        display: flex;
        justify-content: space-between;
        margin-bottom: 0.5rem;
    }
    .modal-row:last-child {
        margin-bottom: 0;
    }
    .modal-label {
        color: var(--muted);
        font-size: 0.85rem;
    }
    .modal-value {
        color: var(--ink-bright);
        font-size: 0.85rem;
        font-weight: 500;
    }
    .modal-warning-box {
        background: rgba(239, 68, 68, 0.1);
        border: 1px solid rgba(239, 68, 68, 0.3);
        border-radius: var(--radius-sm);
        padding: 0.75rem;
        display: flex;
        align-items: center;
        gap: 0.5rem;
    }
    .warning-icon {
        font-size: 1.2rem;
    }
    .warning-text {
        color: #dc2626;
        font-size: 0.85rem;
    }
    .modal-footer {
        display: flex;
        justify-content: flex-end;
        gap: 0.75rem;
    }
    .modal-btn {
        padding: 0.5rem 1rem;
        border-radius: var(--radius-sm);
        font-size: 0.85rem;
        font-weight: 500;
        cursor: pointer;
        border: none;
    }
    .modal-btn-cancel {
        background: var(--surface);
        color: var(--ink-bright);
        border: 1px solid var(--line);
    }
    .modal-btn-cancel:hover {
        background: var(--card);
    }
    .modal-btn-confirm {
        background: #dc2626;
        color: white;
    }
    .modal-btn-confirm:hover {
        background: #b91c1c;
    }
    .cancel-button {
        background: rgba(239, 68, 68, 0.1);
        border: 1px solid rgba(239, 68, 68, 0.3);
        color: #dc2626;
        padding: 0.25rem 0.5rem;
        border-radius: var(--radius-sm);
        font-size: 0.75rem;
        cursor: pointer;
        width: 100%;
    }
    .cancel-button:hover {
        background: rgba(239, 68, 68, 0.2);
    }
    .toast {
        position: fixed;
        bottom: 2rem;
        right: 2rem;
        background: var(--card);
        border: 1px solid var(--line);
        border-radius: var(--radius-sm);
        padding: 1rem 1.5rem;
        color: var(--ink-bright);
        font-size: 0.85rem;
        z-index: 1001;
        box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
    }
    </style>
</body>
</html>`

	return html
}

// RenderBDDReviewHTML 渲染 BDD 规则审核页面
func RenderBDDReviewHTML(issue *domain.Issue, stageState *domain.StageState, bddContent string) string {
	// 解析状态
	state := issue.State
	stateClass := common.StateBadgeClass(state)
	stageDisplay := getStageDisplay(stageState.Name)

	// 格式化 BDD 内容用于显示
	formattedBDD := formatBDDContent(bddContent)

	return `<!DOCTYPE html>
<html lang="zh">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Symphony · BDD 规则审核: ` + common.EscapeHTML(issue.Identifier) + `</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=Fira+Code:wght@400;500&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="/dashboard.css">
    <script src="https://unpkg.com/htmx.org@1.9.10" crossorigin="anonymous"></script>
    <style>
    .bdd-container {
        background: var(--surface);
        border: 1px solid var(--line);
        border-radius: var(--radius-lg);
        padding: 1.5rem;
        margin-top: 1rem;
    }
    .feature-header {
        border-bottom: 1px solid var(--line);
        padding-bottom: 1rem;
        margin-bottom: 1rem;
    }
    .feature-name {
        font-size: 1.25rem;
        font-weight: 600;
        color: var(--accent);
    }
    .feature-description {
        color: var(--ink);
        margin-top: 0.5rem;
        font-size: 0.9rem;
        line-height: 1.6;
    }
    .scenario-block {
        background: var(--card);
        border: 1px solid var(--line);
        border-radius: var(--radius);
        padding: 1rem;
        margin-bottom: 1rem;
    }
    .scenario-name {
        font-weight: 600;
        color: var(--ink-bright);
        margin-bottom: 0.75rem;
        display: flex;
        align-items: center;
        gap: 0.5rem;
    }
    .scenario-badge {
        background: rgba(139, 92, 246, 0.2);
        color: #8b5cf6;
        padding: 0.25rem 0.5rem;
        border-radius: var(--radius-sm);
        font-size: 0.75rem;
        font-weight: 500;
    }
    .gherkin-content {
        font-family: 'Fira Code', monospace;
        font-size: 0.85rem;
        line-height: 1.8;
        white-space: pre-wrap;
        word-wrap: break-word;
    }
    .gherkin-feature {
        color: #8b5cf6;
        font-weight: 600;
    }
    .gherkin-scenario {
        color: #f59e0b;
        font-weight: 600;
    }
    .gherkin-step {
        display: block;
        padding-left: 1rem;
    }
    .step-given { color: #22c55e; }
    .step-when { color: #3b82f6; }
    .step-then { color: #f97316; }
    .step-and { color: var(--ink); }
    .step-but { color: var(--ink); }
    .step-other { color: var(--ink); }
    .gherkin-tags {
        color: #6b7280;
        font-style: italic;
    }
    .action-buttons {
        display: flex;
        gap: 1rem;
        margin-top: 1.5rem;
        padding-top: 1.5rem;
        border-top: 1px solid var(--line);
    }
    .btn-approve {
        display: inline-flex;
        align-items: center;
        gap: 0.5rem;
        padding: 0.75rem 1.5rem;
        background: linear-gradient(135deg, rgba(34, 197, 94, 0.9), rgba(34, 197, 94, 0.7));
        color: white;
        border: none;
        border-radius: var(--radius);
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
    }
    .btn-approve:hover {
        background: linear-gradient(135deg, rgba(34, 197, 94, 1), rgba(34, 197, 94, 0.9));
        transform: translateY(-1px);
    }
    .btn-reject {
        display: inline-flex;
        align-items: center;
        gap: 0.5rem;
        padding: 0.75rem 1.5rem;
        background: linear-gradient(135deg, rgba(239, 68, 68, 0.9), rgba(239, 68, 68, 0.7));
        color: white;
        border: none;
        border-radius: var(--radius);
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
    }
    .btn-reject:hover {
        background: linear-gradient(135deg, rgba(239, 68, 68, 1), rgba(239, 68, 68, 0.9));
        transform: translateY(-1px);
    }
    .btn-back {
        display: inline-flex;
        align-items: center;
        gap: 0.5rem;
        padding: 0.75rem 1.5rem;
        background: var(--surface);
        color: var(--ink);
        border: 1px solid var(--line);
        border-radius: var(--radius);
        font-weight: 500;
        text-decoration: none;
        transition: all 0.2s;
    }
    .btn-back:hover {
        background: var(--card);
    }
    .reject-modal {
        display: none;
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.5);
        align-items: center;
        justify-content: center;
        z-index: 1000;
    }
    .reject-modal-content {
        background: var(--card);
        border: 1px solid var(--line);
        border-radius: var(--radius-lg);
        padding: 1.5rem;
        max-width: 500px;
        width: 90%;
    }
    .reject-reason-input {
        width: 100%;
        min-height: 100px;
        padding: 0.75rem;
        border: 1px solid var(--line);
        border-radius: var(--radius);
        background: var(--surface);
        color: var(--ink-bright);
        font-size: 0.9rem;
        resize: vertical;
    }
    .no-bdd-content {
        text-align: center;
        padding: 3rem;
        color: var(--muted);
    }
    .no-bdd-icon {
        font-size: 3rem;
        margin-bottom: 1rem;
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
                        <h1 class="hero-title">BDD 规则审核: ` + common.EscapeHTML(issue.Identifier) + `</h1>
                        <p class="hero-copy">` + common.EscapeHTML(issue.Title) + `</p>
                    </div>
                    <div class="status-stack">
                        <a href="/api/v1/` + common.EscapeHTML(issue.Identifier) + `" class="btn-back" style="padding: 0.5rem 1rem;">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <line x1="19" y1="12" x2="5" y2="12"></line>
                                <polyline points="12 19 5 12 12 5"></polyline>
                            </svg>
                            返回任务详情
                        </a>
                    </div>
                </div>
            </header>

            <section class="section-card" style="background: var(--card); border: 1px solid var(--line); border-radius: var(--radius-lg); padding: 1.5rem;">
                <div class="task-detail-header" style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;">
                    <div>
                        <h2 style="font-size: 1.1rem; font-weight: 600; color: var(--ink-bright);">` + common.EscapeHTML(issue.Title) + `</h2>
                        <div style="display: flex; gap: 1rem; align-items: center; margin-top: 0.5rem;">
                            <span class="` + stateClass + `">` + common.EscapeHTML(state) + `</span>
                            <span style="color: var(--muted); font-size: 0.85rem;">阶段: ` + stageDisplay + `</span>
                        </div>
                    </div>
                </div>
            </section>

            <section class="bdd-container">
                <div style="display: flex; align-items: center; gap: 0.5rem; margin-bottom: 1rem;">
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: var(--accent);">
                        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                        <polyline points="14 2 14 8 20 8"></polyline>
                        <line x1="16" y1="13" x2="8" y2="13"></line>
                        <line x1="16" y1="17" x2="8" y2="17"></line>
                    </svg>
                    <span style="font-weight: 600; color: var(--ink-bright); font-size: 1rem;">BDD 场景规则</span>
                </div>

                ` + func() string {
		if bddContent == "" {
			return `<div class="no-bdd-content">
                        <div class="no-bdd-icon">📋</div>
                        <p>暂无 BDD 规则内容</p>
                        <p style="font-size: 0.85rem; margin-top: 0.5rem;">等待 AI 生成 BDD 场景...</p>
                    </div>`
		}
		return formattedBDD
	}() + `
            </section>

            ` + func() string {
		if bddContent == "" {
			return ``
		}
		return `<div class="action-buttons">
                    <a href="/" class="btn-back">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <line x1="19" y1="12" x2="5" y2="12"></line>
                            <polyline points="12 19 5 12 12 5"></polyline>
                        </svg>
                        返回看板
                    </a>
                    <button type="button" class="btn-approve" onclick="approveBDD()">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <polyline points="20 6 9 17 4 12"></polyline>
                        </svg>
                        通过
                    </button>
                    <button type="button" class="btn-reject" onclick="showRejectModal()">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <line x1="18" y1="6" x2="6" y2="18"></line>
                            <line x1="6" y1="6" x2="18" y2="18"></line>
                        </svg>
                        驳回
                    </button>
                </div>`
	}() + `
        </section>
    </main>

    <!-- 驳回确认对话框 -->
    <div id="reject-modal" class="reject-modal">
        <div class="reject-modal-content">
            <h3 style="font-size: 1.1rem; font-weight: 600; color: var(--ink-bright); margin-bottom: 1rem;">驳回 BDD 规则</h3>
            <p style="color: var(--ink); font-size: 0.9rem; margin-bottom: 1rem;">请输入驳回原因，AI 将根据您的反馈重新生成 BDD 规则。</p>
            <textarea id="reject-reason" class="reject-reason-input" placeholder="请描述需要修改的内容..."></textarea>
            <div style="display: flex; justify-content: flex-end; gap: 0.75rem; margin-top: 1rem;">
                <button type="button" class="btn-back" onclick="hideRejectModal()">取消</button>
                <button type="button" class="btn-reject" onclick="rejectBDD()">确认驳回</button>
            </div>
        </div>
    </div>

    <script>
    function approveBDD() {
        fetch('/api/v1/` + common.EscapeHTML(issue.Identifier) + `/bdd/approve', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                alert('BDD 规则已通过审核');
                window.location.href = '/api/v1/` + common.EscapeHTML(issue.Identifier) + `';
            } else {
                alert('操作失败: ' + (data.error ? data.error.message : '未知错误'));
            }
        })
        .catch(err => {
            alert('操作失败: 网络错误');
        });
    }

    function showRejectModal() {
        document.getElementById('reject-modal').style.display = 'flex';
    }

    function hideRejectModal() {
        document.getElementById('reject-modal').style.display = 'none';
        document.getElementById('reject-reason').value = '';
    }

    function rejectBDD() {
        const reason = document.getElementById('reject-reason').value.trim();
        if (!reason) {
            alert('请输入驳回原因');
            return;
        }

        fetch('/api/v1/` + common.EscapeHTML(issue.Identifier) + `/bdd/reject', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ reason: reason })
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                alert('BDD 规则已驳回，AI 将重新生成');
                window.location.href = '/api/v1/` + common.EscapeHTML(issue.Identifier) + `';
            } else {
                alert('操作失败: ' + (data.error ? data.error.message : '未知错误'));
            }
        })
        .catch(err => {
            alert('操作失败: 网络错误');
        });
    }

    // 点击模态框外部关闭
    document.getElementById('reject-modal').addEventListener('click', function(e) {
        if (e.target === this) {
            hideRejectModal();
        }
    });
    </script>
</body>
</html>`
}

// formatBDDContent 格式化 BDD 内容用于 HTML 显示
func formatBDDContent(content string) string {
	if content == "" {
		return ""
	}

	// 使用转义处理
	escapedContent := common.EscapeHTML(content)

	// 添加 Gherkin 语法高亮类
	lines := strings.Split(escapedContent, "\n")
	result := make([]string, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Feature 行
		if strings.HasPrefix(trimmed, "Feature:") || strings.HasPrefix(trimmed, "功能:") {
			result = append(result, `<div class="feature-header"><div class="feature-name">`+trimmed+`</div>`)
			continue
		}

		// Scenario 行
		if strings.HasPrefix(trimmed, "Scenario:") || strings.HasPrefix(trimmed, "场景:") ||
			strings.HasPrefix(trimmed, "Scenario Outline:") || strings.HasPrefix(trimmed, "场景大纲:") {
			// 关闭之前的 feature-header（如果有的话）
			if len(result) > 0 && strings.HasSuffix(result[len(result)-1], "</div>") {
				// 已经是一个完整的 div
			} else if len(result) > 0 && strings.HasPrefix(result[len(result)-1], `<div class="feature-header">`) {
				result[len(result)-1] += `</div>`
			}
			result = append(result, `<div class="scenario-block"><div class="scenario-name"><span class="scenario-badge">Scenario</span>`+trimmed+`</div><div class="gherkin-content">`)
			continue
		}

		// 步骤行
		if strings.HasPrefix(trimmed, "Given") || strings.HasPrefix(trimmed, "When") ||
			strings.HasPrefix(trimmed, "Then") || strings.HasPrefix(trimmed, "And") ||
			strings.HasPrefix(trimmed, "But") || strings.HasPrefix(trimmed, "假如") ||
			strings.HasPrefix(trimmed, "当") || strings.HasPrefix(trimmed, "那么") ||
			strings.HasPrefix(trimmed, "并且") || strings.HasPrefix(trimmed, "但是") {
			cssClass := getStepCSSClass(trimmed)
			result = append(result, `<span class="gherkin-step `+cssClass+`">`+line+`</span>`)
			continue
		}

		// Tags
		if strings.HasPrefix(trimmed, "@") {
			result = append(result, `<span class="gherkin-tags">`+line+`</span>`)
			continue
		}

		// 空行
		if trimmed == "" {
			result = append(result, "")
			continue
		}

		// 其他内容（描述等）
		result = append(result, line)
	}

	// 关闭最后打开的标签
	if len(result) > 0 {
		lastIdx := len(result) - 1
		if strings.Contains(result[lastIdx], `<div class="scenario-block">`) {
			result[lastIdx] += `</div></div>`
		} else if strings.Contains(result[lastIdx], `<div class="gherkin-content">`) {
			result[lastIdx] += `</div></div>`
		}
	}

	return strings.Join(result, "\n")
}

// getStepCSSClass 根据步骤关键词返回 CSS 类名
func getStepCSSClass(line string) string {
	switch {
	case strings.HasPrefix(line, "Given") || strings.HasPrefix(line, "假如"):
		return "step-given"
	case strings.HasPrefix(line, "When") || strings.HasPrefix(line, "当"):
		return "step-when"
	case strings.HasPrefix(line, "Then") || strings.HasPrefix(line, "那么"):
		return "step-then"
	case strings.HasPrefix(line, "And") || strings.HasPrefix(line, "并且"):
		return "step-and"
	case strings.HasPrefix(line, "But") || strings.HasPrefix(line, "但是"):
		return "step-but"
	default:
		return "step-other"
	}
}

// RenderArchitectureReviewHTML 渲染架构设计审核页面
func RenderArchitectureReviewHTML(issue *domain.Issue, stageState *domain.StageState, archContent string, tddContent string) string {
	// 解析状态
	state := issue.State
	stateClass := common.StateBadgeClass(state)
	stageDisplay := getStageDisplay(stageState.Name)

	// 格式化架构设计内容用于显示
	formattedArch := formatArchitectureContent(archContent)
	formattedTDD := formatTDDContent(tddContent)

	return `<!DOCTYPE html>
<html lang="zh">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Symphony · 架构设计审核: ` + common.EscapeHTML(issue.Identifier) + `</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=Fira+Code:wght@400;500&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="/dashboard.css">
    <script src="https://unpkg.com/htmx.org@1.9.10" crossorigin="anonymous"></script>
    <style>
    .arch-container {
        background: var(--surface);
        border: 1px solid var(--line);
        border-radius: var(--radius-lg);
        padding: 1.5rem;
        margin-top: 1rem;
    }
    .arch-header {
        border-bottom: 1px solid var(--line);
        padding-bottom: 1rem;
        margin-bottom: 1rem;
    }
    .arch-title {
        font-size: 1.25rem;
        font-weight: 600;
        color: var(--accent);
    }
    .arch-overview {
        color: var(--ink);
        margin-top: 0.5rem;
        font-size: 0.9rem;
        line-height: 1.6;
    }
    .component-block {
        background: var(--card);
        border: 1px solid var(--line);
        border-radius: var(--radius);
        padding: 1rem;
        margin-bottom: 1rem;
    }
    .component-name {
        font-weight: 600;
        color: var(--ink-bright);
        margin-bottom: 0.75rem;
        display: flex;
        align-items: center;
        gap: 0.5rem;
    }
    .component-badge {
        background: rgba(236, 72, 153, 0.2);
        color: #ec4899;
        padding: 0.25rem 0.5rem;
        border-radius: var(--radius-sm);
        font-size: 0.75rem;
        font-weight: 500;
    }
    .tdd-container {
        background: var(--surface);
        border: 1px solid var(--line);
        border-radius: var(--radius-lg);
        padding: 1.5rem;
        margin-top: 1.5rem;
    }
    .tdd-rule-block {
        background: var(--card);
        border: 1px solid var(--line);
        border-radius: var(--radius);
        padding: 1rem;
        margin-bottom: 1rem;
    }
    .tdd-rule-name {
        font-weight: 600;
        color: var(--ink-bright);
        margin-bottom: 0.75rem;
    }
    .tdd-rule-priority {
        display: inline-block;
        padding: 0.125rem 0.5rem;
        border-radius: var(--radius-sm);
        font-size: 0.7rem;
        font-weight: 600;
        margin-left: 0.5rem;
    }
    .priority-high { background: rgba(239, 68, 68, 0.2); color: #ef4444; }
    .priority-medium { background: rgba(245, 158, 11, 0.2); color: #f59e0b; }
    .priority-low { background: rgba(34, 197, 94, 0.2); color: #22c55e; }
    .arch-section {
        margin-bottom: 1.5rem;
    }
    .arch-section-title {
        font-size: 1rem;
        font-weight: 600;
        color: var(--ink-bright);
        margin-bottom: 0.75rem;
        display: flex;
        align-items: center;
        gap: 0.5rem;
    }
    .arch-content {
        font-family: 'Fira Code', monospace;
        font-size: 0.85rem;
        line-height: 1.8;
        white-space: pre-wrap;
        word-wrap: break-word;
        color: var(--ink);
    }
    .action-buttons {
        display: flex;
        gap: 1rem;
        margin-top: 1.5rem;
        padding-top: 1.5rem;
        border-top: 1px solid var(--line);
    }
    .btn-approve {
        display: inline-flex;
        align-items: center;
        gap: 0.5rem;
        padding: 0.75rem 1.5rem;
        background: linear-gradient(135deg, rgba(34, 197, 94, 0.9), rgba(34, 197, 94, 0.7));
        color: white;
        border: none;
        border-radius: var(--radius);
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
    }
    .btn-approve:hover {
        background: linear-gradient(135deg, rgba(34, 197, 94, 1), rgba(34, 197, 94, 0.9));
        transform: translateY(-1px);
    }
    .btn-reject {
        display: inline-flex;
        align-items: center;
        gap: 0.5rem;
        padding: 0.75rem 1.5rem;
        background: linear-gradient(135deg, rgba(239, 68, 68, 0.9), rgba(239, 68, 68, 0.7));
        color: white;
        border: none;
        border-radius: var(--radius);
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
    }
    .btn-reject:hover {
        background: linear-gradient(135deg, rgba(239, 68, 68, 1), rgba(239, 68, 68, 0.9));
        transform: translateY(-1px);
    }
    .btn-back {
        display: inline-flex;
        align-items: center;
        gap: 0.5rem;
        padding: 0.75rem 1.5rem;
        background: var(--surface);
        color: var(--ink);
        border: 1px solid var(--line);
        border-radius: var(--radius);
        font-weight: 500;
        text-decoration: none;
        transition: all 0.2s;
    }
    .btn-back:hover {
        background: var(--card);
    }
    .reject-modal {
        display: none;
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.5);
        align-items: center;
        justify-content: center;
        z-index: 1000;
    }
    .reject-modal-content {
        background: var(--card);
        border: 1px solid var(--line);
        border-radius: var(--radius-lg);
        padding: 1.5rem;
        max-width: 500px;
        width: 90%;
    }
    .reject-reason-input {
        width: 100%;
        min-height: 100px;
        padding: 0.75rem;
        border: 1px solid var(--line);
        border-radius: var(--radius);
        background: var(--surface);
        color: var(--ink-bright);
        font-size: 0.9rem;
        resize: vertical;
    }
    .no-content {
        text-align: center;
        padding: 3rem;
        color: var(--muted);
    }
    .no-content-icon {
        font-size: 3rem;
        margin-bottom: 1rem;
    }
    .tab-container {
        display: flex;
        gap: 0.5rem;
        margin-bottom: 1rem;
        border-bottom: 1px solid var(--line);
        padding-bottom: 0.5rem;
    }
    .tab-btn {
        padding: 0.5rem 1rem;
        border: none;
        background: transparent;
        color: var(--ink);
        cursor: pointer;
        border-radius: var(--radius-sm) var(--radius-sm) 0 0;
        font-weight: 500;
        transition: all 0.2s;
    }
    .tab-btn:hover {
        background: var(--surface);
    }
    .tab-btn.active {
        background: var(--accent);
        color: white;
    }
    .tab-content {
        display: none;
    }
    .tab-content.active {
        display: block;
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
                        <h1 class="hero-title">架构设计审核: ` + common.EscapeHTML(issue.Identifier) + `</h1>
                        <p class="hero-copy">` + common.EscapeHTML(issue.Title) + `</p>
                    </div>
                    <div class="status-stack">
                        <a href="/api/v1/` + common.EscapeHTML(issue.Identifier) + `" class="btn-back" style="padding: 0.5rem 1rem;">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <line x1="19" y1="12" x2="5" y2="12"></line>
                                <polyline points="12 19 5 12 12 5"></polyline>
                            </svg>
                            返回任务详情
                        </a>
                    </div>
                </div>
            </header>

            <section class="section-card" style="background: var(--card); border: 1px solid var(--line); border-radius: var(--radius-lg); padding: 1.5rem;">
                <div class="task-detail-header" style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;">
                    <div>
                        <h2 style="font-size: 1.1rem; font-weight: 600; color: var(--ink-bright);">` + common.EscapeHTML(issue.Title) + `</h2>
                        <div style="display: flex; gap: 1rem; align-items: center; margin-top: 0.5rem;">
                            <span class="` + stateClass + `">` + common.EscapeHTML(state) + `</span>
                            <span style="color: var(--muted); font-size: 0.85rem;">阶段: ` + stageDisplay + `</span>
                        </div>
                    </div>
                </div>
            </section>

            <section class="arch-container">
                <div class="tab-container">
                    <button class="tab-btn active" onclick="switchTab('architecture')">架构设计</button>
                    <button class="tab-btn" onclick="switchTab('tdd')">TDD 规则</button>
                </div>

                <div id="tab-architecture" class="tab-content active">
                    <div style="display: flex; align-items: center; gap: 0.5rem; margin-bottom: 1rem;">
                        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: var(--accent);">
                            <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
                            <line x1="3" y1="9" x2="21" y2="9"></line>
                            <line x1="9" y1="21" x2="9" y2="9"></line>
                        </svg>
                        <span style="font-weight: 600; color: var(--ink-bright); font-size: 1rem;">架构设计文档</span>
                    </div>

                    ` + func() string {
		if archContent == "" {
			return `<div class="no-content">
                        <div class="no-content-icon">🏗️</div>
                        <p>暂无架构设计内容</p>
                        <p style="font-size: 0.85rem; margin-top: 0.5rem;">等待 AI 生成架构设计...</p>
                    </div>`
		}
		return formattedArch
	}() + `
                </div>

                <div id="tab-tdd" class="tab-content">
                    <div style="display: flex; align-items: center; gap: 0.5rem; margin-bottom: 1rem;">
                        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: var(--accent);">
                            <polyline points="9 11 12 14 22 4"></polyline>
                            <path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"></path>
                        </svg>
                        <span style="font-weight: 600; color: var(--ink-bright); font-size: 1rem;">TDD 规则列表</span>
                    </div>

                    ` + func() string {
		if tddContent == "" {
			return `<div class="no-content">
                        <div class="no-content-icon">✅</div>
                        <p>暂无 TDD 规则内容</p>
                        <p style="font-size: 0.85rem; margin-top: 0.5rem;">等待 AI 生成 TDD 规则...</p>
                    </div>`
		}
		return formattedTDD
	}() + `
                </div>
            </section>

            ` + func() string {
		if archContent == "" {
			return ``
		}
		return `<div class="action-buttons">
                    <a href="/" class="btn-back">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <line x1="19" y1="12" x2="5" y2="12"></line>
                            <polyline points="12 19 5 12 12 5"></polyline>
                        </svg>
                        返回看板
                    </a>
                    <button type="button" class="btn-approve" onclick="approveArchitecture()">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <polyline points="20 6 9 17 4 12"></polyline>
                        </svg>
                        通过
                    </button>
                    <button type="button" class="btn-reject" onclick="showRejectModal()">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <line x1="18" y1="6" x2="6" y2="18"></line>
                            <line x1="6" y1="6" x2="18" y2="18"></line>
                        </svg>
                        驳回
                    </button>
                </div>`
	}() + `
        </section>
    </main>

    <!-- 驳回确认对话框 -->
    <div id="reject-modal" class="reject-modal">
        <div class="reject-modal-content">
            <h3 style="font-size: 1.1rem; font-weight: 600; color: var(--ink-bright); margin-bottom: 1rem;">驳回架构设计</h3>
            <p style="color: var(--ink); font-size: 0.9rem; margin-bottom: 1rem;">请输入驳回原因，AI 将根据您的反馈重新生成架构设计。</p>
            <textarea id="reject-reason" class="reject-reason-input" placeholder="请描述需要修改的内容..."></textarea>
            <div style="display: flex; justify-content: flex-end; gap: 0.75rem; margin-top: 1rem;">
                <button type="button" class="btn-back" onclick="hideRejectModal()">取消</button>
                <button type="button" class="btn-reject" onclick="rejectArchitecture()">确认驳回</button>
            </div>
        </div>
    </div>

    <script>
    function switchTab(tabId) {
        // 切换标签页按钮状态
        document.querySelectorAll('.tab-btn').forEach(btn => {
            btn.classList.remove('active');
        });
        event.target.classList.add('active');

        // 切换内容显示
        document.querySelectorAll('.tab-content').forEach(content => {
            content.classList.remove('active');
        });
        document.getElementById('tab-' + tabId).classList.add('active');
    }

    function approveArchitecture() {
        fetch('/api/v1/` + common.EscapeHTML(issue.Identifier) + `/architecture/approve', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                alert('架构设计已通过审核');
                window.location.href = '/api/v1/` + common.EscapeHTML(issue.Identifier) + `';
            } else {
                alert('操作失败: ' + (data.error ? data.error.message : '未知错误'));
            }
        })
        .catch(err => {
            alert('操作失败: 网络错误');
        });
    }

    function showRejectModal() {
        document.getElementById('reject-modal').style.display = 'flex';
    }

    function hideRejectModal() {
        document.getElementById('reject-modal').style.display = 'none';
        document.getElementById('reject-reason').value = '';
    }

    function rejectArchitecture() {
        const reason = document.getElementById('reject-reason').value.trim();
        if (!reason) {
            alert('请输入驳回原因');
            return;
        }

        fetch('/api/v1/` + common.EscapeHTML(issue.Identifier) + `/architecture/reject', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ reason: reason })
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                alert('架构设计已驳回，AI 将重新生成');
                window.location.href = '/api/v1/` + common.EscapeHTML(issue.Identifier) + `';
            } else {
                alert('操作失败: ' + (data.error ? data.error.message : '未知错误'));
            }
        })
        .catch(err => {
            alert('操作失败: 网络错误');
        });
    }

    // 点击模态框外部关闭
    document.getElementById('reject-modal').addEventListener('click', function(e) {
        if (e.target === this) {
            hideRejectModal();
        }
    });
    </script>
</body>
</html>`
}

// formatArchitectureContent 格式化架构设计内容用于 HTML 显示
func formatArchitectureContent(content string) string {
	if content == "" {
		return ""
	}

	// 使用转义处理
	escapedContent := common.EscapeHTML(content)

	// 添加架构设计语法高亮
	lines := strings.Split(escapedContent, "\n")
	result := make([]string, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 标题
		if strings.HasPrefix(trimmed, "# ") {
			result = append(result, `<div class="arch-section-title" style="font-size: 1.25rem; color: var(--accent); margin-top: 1rem;">`+strings.TrimPrefix(trimmed, "# ")+`</div>`)
			continue
		}

		// 二级标题
		if strings.HasPrefix(trimmed, "## ") {
			result = append(result, `<div class="arch-section-title" style="margin-top: 1rem;">`+strings.TrimPrefix(trimmed, "## ")+`</div>`)
			continue
		}

		// 三级标题
		if strings.HasPrefix(trimmed, "### ") {
			result = append(result, `<div class="component-name"><span class="component-badge">组件</span>`+strings.TrimPrefix(trimmed, "### ")+`</div>`)
			continue
		}

		// 加粗文本
		if strings.HasPrefix(trimmed, "**") && strings.Contains(trimmed, ":**") {
			result = append(result, `<div style="color: var(--accent); font-weight: 600; margin: 0.5rem 0;">`+trimmed+`</div>`)
			continue
		}

		// 列表项
		if strings.HasPrefix(trimmed, "- ") {
			result = append(result, `<div style="padding-left: 1rem; color: var(--ink);">`+line+`</div>`)
			continue
		}

		// 表格行
		if strings.HasPrefix(trimmed, "|") {
			result = append(result, `<div style="font-family: 'Fira Code', monospace; font-size: 0.8rem; color: var(--ink);">`+line+`</div>`)
			continue
		}

		// 空行
		if trimmed == "" {
			result = append(result, "")
			continue
		}

		// 普通文本
		result = append(result, `<div style="color: var(--ink);">`+line+`</div>`)
	}

	return strings.Join(result, "\n")
}

// formatTDDContent 格式化 TDD 规则内容用于 HTML 显示
func formatTDDContent(content string) string {
	if content == "" {
		return ""
	}

	// 使用转义处理
	escapedContent := common.EscapeHTML(content)

	// 添加 TDD 规则语法高亮
	lines := strings.Split(escapedContent, "\n")
	result := make([]string, 0, len(lines))

	var inRule bool

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 标题
		if strings.HasPrefix(trimmed, "# TDD 规则: ") {
			result = append(result, `<div class="arch-section-title" style="font-size: 1.25rem; color: var(--accent);">TDD 规则: `+strings.TrimPrefix(trimmed, "# TDD 规则: ")+`</div>`)
			continue
		}

		// 规则标题
		if strings.HasPrefix(trimmed, "## 规则 ") || strings.HasPrefix(trimmed, "## 规则:") {
			if inRule {
				result = append(result, `</div>`)
			}
			result = append(result, `<div class="tdd-rule-block">`)
			result = append(result, `<div class="tdd-rule-name">`+trimmed+`</div>`)
			inRule = true
			continue
		}

		// 摘要
		if strings.HasPrefix(trimmed, "## 摘要") {
			if inRule {
				result = append(result, `</div>`)
				inRule = false
			}
			result = append(result, `<div class="arch-section-title">摘要</div>`)
			continue
		}

		// Given/When/Then
		if strings.HasPrefix(trimmed, "**Given:**") || strings.HasPrefix(trimmed, "**When:**") || strings.HasPrefix(trimmed, "**Then:**") {
			result = append(result, `<div style="color: var(--accent); font-weight: 600; margin: 0.5rem 0;">`+trimmed+`</div>`)
			continue
		}

		// 优先级
		if strings.HasPrefix(trimmed, "**优先级:**") {
			priority := strings.TrimPrefix(trimmed, "**优先级:** ")
			priorityClass := "priority-medium"
			if priority == "high" {
				priorityClass = "priority-high"
			} else if priority == "low" {
				priorityClass = "priority-low"
			}
			result = append(result, `<div style="margin: 0.5rem 0;">优先级: <span class="tdd-rule-priority `+priorityClass+`">`+priority+`</span></div>`)
			continue
		}

		// Tags
		if strings.HasPrefix(trimmed, "**Tags:**") {
			result = append(result, `<div style="color: var(--muted); font-size: 0.85rem;">`+trimmed+`</div>`)
			continue
		}

		// 列表项
		if strings.HasPrefix(trimmed, "- ") {
			result = append(result, `<div style="padding-left: 1.5rem; color: var(--ink);">`+line+`</div>`)
			continue
		}

		// 空行
		if trimmed == "" {
			result = append(result, "")
			continue
		}

		// 普通文本
		result = append(result, `<div style="color: var(--ink);">`+line+`</div>`)
	}

	if inRule {
		result = append(result, `</div>`)
	}

	return strings.Join(result, "\n")
}

// RenderVerificationReportHTML 渲染验收报告页面
func RenderVerificationReportHTML(issue *domain.Issue, stageState *domain.StageState, report *domain.VerificationReport) string {
	// 解析状态
	state := issue.State
	stateClass := common.StateBadgeClass(state)
	stageDisplay := getStageDisplay(stageState.Name)

	// 计算测试通过率
	testPassRate := 0
	if report != nil && report.TestResults != nil && report.TestResults.Total > 0 {
		testPassRate = int(float64(report.TestResults.Passed) / float64(report.TestResults.Total) * 100)
	}

	// 计算BDD通过率
	bddPassCount := 0
	bddTotal := 0
	if report != nil {
		for _, bdd := range report.BDDValidation {
			bddTotal++
			if bdd.Status == "pass" {
				bddPassCount++
			}
		}
	}
	bddPassRate := 0
	if bddTotal > 0 {
		bddPassRate = int(float64(bddPassCount) / float64(bddTotal) * 100)
	}

	// 判断是否显示操作按钮
	showActions := stageState.Name == "verification" && (stageState.Status == "pending" || stageState.Status == "in_progress")

	return `<!DOCTYPE html>
<html lang="zh">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Symphony · 验收报告: ` + common.EscapeHTML(issue.Identifier) + `</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=Fira+Code:wght@400;500&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="/dashboard.css">
    <script src="https://unpkg.com/htmx.org@1.9.10" crossorigin="anonymous"></script>
    <style>
    .verification-container {
        background: var(--surface);
        border: 1px solid var(--line);
        border-radius: var(--radius-lg);
        padding: 1.5rem;
        margin-top: 1rem;
    }
    .summary-grid {
        display: grid;
        grid-template-columns: repeat(2, 1fr);
        gap: 1rem;
        margin-bottom: 1.5rem;
    }
    .summary-card {
        background: var(--card);
        border: 1px solid var(--line);
        border-radius: var(--radius);
        padding: 1rem;
    }
    .summary-title {
        font-size: 0.85rem;
        color: var(--muted);
        margin-bottom: 0.5rem;
    }
    .summary-value {
        font-size: 1.5rem;
        font-weight: 600;
        color: var(--ink-bright);
    }
    .summary-detail {
        font-size: 0.85rem;
        color: var(--ink);
        margin-top: 0.25rem;
    }
    .progress-bar-container {
        margin-top: 0.5rem;
        height: 8px;
        background: var(--line);
        border-radius: var(--radius-sm);
        overflow: hidden;
    }
    .progress-bar-fill {
        height: 100%;
        border-radius: var(--radius-sm);
        transition: width 0.3s ease;
    }
    .pass-rate-high { background: #22c55e; }
    .pass-rate-medium { background: #f59e0b; }
    .pass-rate-low { background: #ef4444; }
    .test-result-item, .bdd-result-item {
        background: var(--card);
        border: 1px solid var(--line);
        border-radius: var(--radius-sm);
        padding: 0.75rem 1rem;
        margin-bottom: 0.5rem;
        display: flex;
        justify-content: space-between;
        align-items: center;
    }
    .status-pass { color: #22c55e; }
    .status-fail { color: #ef4444; }
    .action-buttons {
        display: flex;
        gap: 1rem;
        margin-top: 1.5rem;
        padding-top: 1.5rem;
        border-top: 1px solid var(--line);
    }
    .btn-approve {
        display: inline-flex;
        align-items: center;
        gap: 0.5rem;
        padding: 0.75rem 1.5rem;
        background: linear-gradient(135deg, rgba(34, 197, 94, 0.9), rgba(34, 197, 94, 0.7));
        color: white;
        border: none;
        border-radius: var(--radius);
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
    }
    .btn-approve:hover {
        background: linear-gradient(135deg, rgba(34, 197, 94, 1), rgba(34, 197, 94, 0.9));
        transform: translateY(-1px);
    }
    .btn-reject {
        display: inline-flex;
        align-items: center;
        gap: 0.5rem;
        padding: 0.75rem 1.5rem;
        background: linear-gradient(135deg, rgba(239, 68, 68, 0.9), rgba(239, 68, 68, 0.7));
        color: white;
        border: none;
        border-radius: var(--radius);
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
    }
    .btn-reject:hover {
        background: linear-gradient(135deg, rgba(239, 68, 68, 1), rgba(239, 68, 68, 0.9));
        transform: translateY(-1px);
    }
    .btn-back {
        display: inline-flex;
        align-items: center;
        gap: 0.5rem;
        padding: 0.75rem 1.5rem;
        background: var(--surface);
        color: var(--ink);
        border: 1px solid var(--line);
        border-radius: var(--radius);
        font-weight: 500;
        text-decoration: none;
        transition: all 0.2s;
    }
    .btn-back:hover {
        background: var(--card);
    }
    .no-report {
        text-align: center;
        padding: 3rem;
        color: var(--muted);
    }
    .no-report-icon {
        font-size: 3rem;
        margin-bottom: 1rem;
    }
    .reject-modal {
        display: none;
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.5);
        align-items: center;
        justify-content: center;
        z-index: 1000;
    }
    .reject-modal-content {
        background: var(--card);
        border: 1px solid var(--line);
        border-radius: var(--radius-lg);
        padding: 1.5rem;
        max-width: 500px;
        width: 90%;
    }
    .reject-reason-input {
        width: 100%;
        min-height: 100px;
        padding: 0.75rem;
        border: 1px solid var(--line);
        border-radius: var(--radius);
        background: var(--surface);
        color: var(--ink-bright);
        font-size: 0.9rem;
        resize: vertical;
    }
    .overall-status {
        display: inline-flex;
        align-items: center;
        gap: 0.5rem;
        padding: 0.5rem 1rem;
        border-radius: var(--radius);
        font-weight: 600;
    }
    .status-badge-pass {
        background: rgba(34, 197, 94, 0.2);
        color: #22c55e;
    }
    .status-badge-fail {
        background: rgba(239, 68, 68, 0.2);
        color: #ef4444;
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
                        <h1 class="hero-title">验收报告: ` + common.EscapeHTML(issue.Identifier) + `</h1>
                        <p class="hero-copy">` + common.EscapeHTML(issue.Title) + `</p>
                    </div>
                    <div class="status-stack">
                        <a href="/api/v1/` + common.EscapeHTML(issue.Identifier) + `" class="btn-back" style="padding: 0.5rem 1rem;">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <line x1="19" y1="12" x2="5" y2="12"></line>
                                <polyline points="12 19 5 12 12 5"></polyline>
                            </svg>
                            返回任务详情
                        </a>
                    </div>
                </div>
            </header>

            <section class="section-card" style="background: var(--card); border: 1px solid var(--line); border-radius: var(--radius-lg); padding: 1.5rem;">
                <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;">
                    <div>
                        <h2 style="font-size: 1.1rem; font-weight: 600; color: var(--ink-bright);">` + common.EscapeHTML(issue.Title) + `</h2>
                        <div style="display: flex; gap: 1rem; align-items: center; margin-top: 0.5rem;">
                            <span class="` + stateClass + `">` + common.EscapeHTML(state) + `</span>
                            <span style="color: var(--muted); font-size: 0.85rem;">阶段: ` + stageDisplay + `</span>
                        </div>
                    </div>
                </div>
            </section>

            ` + func() string {
		if report == nil {
			return `<section class="verification-container">
                <div class="no-report">
                    <div class="no-report-icon">📋</div>
                    <p>暂无验收报告</p>
                    <p style="font-size: 0.85rem; margin-top: 0.5rem;">等待 AI 生成验收报告...</p>
                </div>
            </section>`
		}
		return renderVerificationReportContent(report, testPassRate, bddPassRate, bddPassCount, bddTotal)
	}() + `

            ` + func() string {
		if showActions && report != nil {
			return `<div class="action-buttons">
                    <a href="/" class="btn-back">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <line x1="19" y1="12" x2="5" y2="12"></line>
                            <polyline points="12 19 5 12 12 5"></polyline>
                        </svg>
                        返回看板
                    </a>
                    <button type="button" class="btn-approve" onclick="approveVerification()">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <polyline points="20 6 9 17 4 12"></polyline>
                        </svg>
                        通过验收
                    </button>
                    <button type="button" class="btn-reject" onclick="showRejectModal()">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <line x1="18" y1="6" x2="6" y2="18"></line>
                            <line x1="6" y1="6" x2="18" y2="18"></line>
                        </svg>
                        驳回验收
                    </button>
                </div>`
		}
		return ""
	}() + `
        </section>
    </main>

    ` + func() string {
		if showActions && report != nil {
			return `<!-- 驳回确认对话框 -->
    <div id="reject-modal" class="reject-modal">
        <div class="reject-modal-content">
            <h3 style="font-size: 1.1rem; font-weight: 600; color: var(--ink-bright); margin-bottom: 1rem;">驳回验收</h3>
            <p style="color: var(--ink); font-size: 0.9rem; margin-bottom: 1rem;">请输入驳回原因，任务将流转回实现中重新实现。</p>
            <textarea id="reject-reason" class="reject-reason-input" placeholder="请描述需要修改的内容..."></textarea>
            <div style="display: flex; justify-content: flex-end; gap: 0.75rem; margin-top: 1rem;">
                <button type="button" class="btn-back" onclick="hideRejectModal()">取消</button>
                <button type="button" class="btn-reject" onclick="rejectVerification()">确认驳回</button>
            </div>
        </div>
    </div>

    <script>
    function approveVerification() {
        fetch('/api/v1/` + common.EscapeHTML(issue.Identifier) + `/verification/approve', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                alert('验收通过，任务已完成' + (data.commit_hash ? '\\nCommit: ' + data.commit_hash : ''));
                window.location.href = '/';
            } else {
                alert('操作失败: ' + (data.error ? data.error.message : '未知错误'));
            }
        })
        .catch(err => {
            alert('操作失败: 网络错误');
        });
    }

    function showRejectModal() {
        document.getElementById('reject-modal').style.display = 'flex';
    }

    function hideRejectModal() {
        document.getElementById('reject-modal').style.display = 'none';
        document.getElementById('reject-reason').value = '';
    }

    function rejectVerification() {
        const reason = document.getElementById('reject-reason').value.trim();

        fetch('/api/v1/` + common.EscapeHTML(issue.Identifier) + `/verification/reject', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ reason: reason })
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                alert('验收已驳回，任务流转回实现中');
                window.location.href = '/api/v1/` + common.EscapeHTML(issue.Identifier) + `';
            } else {
                alert('操作失败: ' + (data.error ? data.error.message : '未知错误'));
            }
        })
        .catch(err => {
            alert('操作失败: 网络错误');
        });
    }

    // 点击模态框外部关闭
    document.getElementById('reject-modal').addEventListener('click', function(e) {
        if (e.target === this) {
            hideRejectModal();
        }
    });
    </script>`
		}
		return ""
	}() + `
</body>
</html>`
}

// renderVerificationReportContent 渲染验收报告内容
func renderVerificationReportContent(report *domain.VerificationReport, testPassRate, bddPassRate, bddPassCount, bddTotal int) string {
	// 确定状态样式
	overallClass := "status-badge-pass"
	if report.OverallStatus != "PASS" {
		overallClass = "status-badge-fail"
	}

	// 获取进度条样式类
	testBarClass := "pass-rate-high"
	if testPassRate < 80 {
		testBarClass = "pass-rate-medium"
	}
	if testPassRate < 50 {
		testBarClass = "pass-rate-low"
	}

	bddBarClass := "pass-rate-high"
	if bddPassRate < 80 {
		bddBarClass = "pass-rate-medium"
	}
	if bddPassRate < 50 {
		bddBarClass = "pass-rate-low"
	}

	return `<section class="verification-container">
            <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 1rem;">
                <div style="display: flex; align-items: center; gap: 0.5rem;">
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: var(--accent);">
                        <path d="M9 12l2 2 4-4"></path>
                        <circle cx="12" cy="12" r="10"></circle>
                    </svg>
                    <span style="font-weight: 600; color: var(--ink-bright); font-size: 1rem;">验收报告</span>
                </div>
                <span class="overall-status ` + overallClass + `">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        ` + func() string {
		if report.OverallStatus == "PASS" {
			return `<polyline points="20 6 9 17 4 12"></polyline>`
		}
		return `<line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line>`
	}() + `
                    </svg>
                    ` + report.OverallStatus + `
                </span>
            </div>

            <div style="margin-bottom: 1rem; color: var(--muted); font-size: 0.85rem;">
                生成时间: ` + report.GeneratedAt.Format("2006-01-02 15:04:05") + `
            </div>

            <!-- 测试结果摘要 -->
            <div class="summary-grid">
                <div class="summary-card">
                    <div class="summary-title">测试结果</div>
                    <div class="summary-value">` + strconv.Itoa(report.TestResults.Passed) + ` / ` + strconv.Itoa(report.TestResults.Total) + `</div>
                    <div class="summary-detail">通过率: ` + strconv.Itoa(testPassRate) + `%</div>
                    <div class="progress-bar-container">
                        <div class="progress-bar-fill ` + testBarClass + `" style="width: ` + strconv.Itoa(testPassRate) + `%;"></div>
                    </div>
                    <div style="margin-top: 0.5rem; font-size: 0.8rem; color: var(--muted);">
                        通过: ` + strconv.Itoa(report.TestResults.Passed) + ` | 失败: ` + strconv.Itoa(report.TestResults.Failed) + ` | 跳过: ` + strconv.Itoa(report.TestResults.Skipped) + `
                    </div>
                </div>

                <div class="summary-card">
                    <div class="summary-title">BDD 场景验证</div>
                    <div class="summary-value">` + strconv.Itoa(bddPassCount) + ` / ` + strconv.Itoa(bddTotal) + `</div>
                    <div class="summary-detail">通过率: ` + strconv.Itoa(bddPassRate) + `%</div>
                    <div class="progress-bar-container">
                        <div class="progress-bar-fill ` + bddBarClass + `" style="width: ` + strconv.Itoa(bddPassRate) + `%;"></div>
                    </div>
                </div>
            </div>

            ` + func() string {
		if len(report.TestResults.FailedTests) > 0 {
			html := `<div style="margin-bottom: 1.5rem;">
                    <h3 style="font-size: 0.9rem; font-weight: 600; color: var(--ink-bright); margin-bottom: 0.75rem;">失败的测试</h3>`
			for _, failed := range report.TestResults.FailedTests {
				html += `<div class="test-result-item" style="background: rgba(239, 68, 68, 0.05);">
                        <div>
                            <div style="font-weight: 500; color: var(--ink-bright);">` + common.EscapeHTML(failed.TestName) + `</div>
                            <div style="font-size: 0.8rem; color: #ef4444; margin-top: 0.25rem;">` + common.EscapeHTML(failed.ErrorMessage) + `</div>
                        </div>
                        <span class="status-fail">失败</span>
                    </div>`
			}
			html += `</div>`
			return html
		}
		return ""
	}() + `

            ` + func() string {
		if len(report.BDDValidation) > 0 {
			html := `<div style="margin-bottom: 1.5rem;">
                    <h3 style="font-size: 0.9rem; font-weight: 600; color: var(--ink-bright); margin-bottom: 0.75rem;">BDD 场景验证结果</h3>`
			for _, bdd := range report.BDDValidation {
				statusClass := "status-pass"
				if bdd.Status != "pass" {
					statusClass = "status-fail"
				}
				html += `<div class="bdd-result-item">
                        <div>
                            <div style="font-weight: 500; color: var(--ink-bright);">` + common.EscapeHTML(bdd.ScenarioName) + `</div>
                            ` + func() string {
					if bdd.Notes != "" {
						return `<div style="font-size: 0.8rem; color: var(--muted); margin-top: 0.25rem;">` + common.EscapeHTML(bdd.Notes) + `</div>`
					}
					return ""
				}() + `
                        </div>
                        <span class="` + statusClass + `">` + bdd.Status + `</span>
                    </div>`
			}
			html += `</div>`
			return html
		}
		return ""
	}() + `

            ` + func() string {
		if report.ImplementationSummary != "" {
			return `<div style="margin-bottom: 1.5rem;">
                    <h3 style="font-size: 0.9rem; font-weight: 600; color: var(--ink-bright); margin-bottom: 0.75rem;">实现摘要</h3>
                    <div style="background: var(--card); border: 1px solid var(--line); border-radius: var(--radius-sm); padding: 1rem;">
                        <p style="color: var(--ink); font-size: 0.9rem; line-height: 1.6; white-space: pre-wrap;">` + common.EscapeHTML(report.ImplementationSummary) + `</p>
                    </div>
                </div>`
		}
		return ""
	}() + `

            ` + func() string {
		if len(report.Recommendations) > 0 {
			html := `<div style="margin-bottom: 1.5rem;">
                    <h3 style="font-size: 0.9rem; font-weight: 600; color: var(--ink-bright); margin-bottom: 0.75rem;">建议</h3>
                    <ul style="list-style: disc; padding-left: 1.5rem; margin: 0;">`
			for _, rec := range report.Recommendations {
				html += `<li style="color: var(--ink); font-size: 0.9rem; margin-bottom: 0.5rem;">` + common.EscapeHTML(rec) + `</li>`
			}
			html += `</ul></div>`
			return html
		}
		return ""
	}() + `
        </section>`
}
// RenderNeedsAttentionHTML 渲染待人工处理页面
func RenderNeedsAttentionHTML(issue *domain.Issue, stageState *domain.StageState) string {
	_ = common.StateBadgeClass(issue.State) // state class for potential future use

	// 获取失败阶段显示名称
	failedStageDisplay := getStageDisplay(stageState.Name)
	
	// 格式化失败时间
	failedAtStr := "未知"
	if stageState.FailedAt != nil {
		failedAtStr = stageState.FailedAt.Format("2006-01-02 15:04:05")
	}

	return `<!DOCTYPE html>
<html lang="zh">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Symphony · 待人工处理: ` + common.EscapeHTML(issue.Identifier) + `</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=Fira+Code:wght@400;500&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="/dashboard.css">
    <script src="https://unpkg.com/htmx.org@1.9.10" crossorigin="anonymous"></script>
    <style>
    .needs-attention-container {
        background: var(--surface);
        border: 1px solid var(--line);
        border-radius: var(--radius-lg);
        padding: 1.5rem;
        margin-top: 1rem;
    }
    .failure-header {
        display: flex;
        align-items: center;
        gap: 1rem;
        padding: 1rem;
        background: rgba(239, 68, 68, 0.1);
        border: 1px solid rgba(239, 68, 68, 0.3);
        border-radius: var(--radius);
        margin-bottom: 1.5rem;
    }
    .failure-icon {
        font-size: 2rem;
        color: #ef4444;
    }
    .failure-title {
        font-size: 1.1rem;
        font-weight: 600;
        color: #ef4444;
    }
    .failure-subtitle {
        font-size: 0.85rem;
        color: var(--ink);
        margin-top: 0.25rem;
    }
    .detail-grid {
        display: grid;
        grid-template-columns: repeat(2, 1fr);
        gap: 1rem;
        margin-bottom: 1.5rem;
    }
    .detail-item {
        background: var(--card);
        border: 1px solid var(--line);
        border-radius: var(--radius-sm);
        padding: 1rem;
    }
    .detail-label {
        font-size: 0.85rem;
        color: var(--muted);
        margin-bottom: 0.5rem;
    }
    .detail-value {
        font-size: 1rem;
        color: var(--ink-bright);
        font-weight: 500;
    }
    .log-snippet {
        background: var(--card);
        border: 1px solid var(--line);
        border-radius: var(--radius-sm);
        padding: 1rem;
        margin-bottom: 1.5rem;
        font-family: 'Fira Code', monospace;
        font-size: 0.85rem;
        white-space: pre-wrap;
        overflow-x: auto;
        color: var(--ink);
    }
    .suggestion-box {
        background: rgba(59, 130, 246, 0.1);
        border: 1px solid rgba(59, 130, 246, 0.3);
        border-radius: var(--radius-sm);
        padding: 1rem;
        margin-bottom: 1.5rem;
    }
    .suggestion-title {
        font-size: 0.85rem;
        color: #3b82f6;
        font-weight: 600;
        margin-bottom: 0.5rem;
        display: flex;
        align-items: center;
        gap: 0.5rem;
    }
    .suggestion-content {
        font-size: 0.9rem;
        color: var(--ink);
        line-height: 1.6;
    }
    .action-buttons {
        display: flex;
        gap: 1rem;
        margin-top: 1.5rem;
        padding-top: 1.5rem;
        border-top: 1px solid var(--line);
    }
    .btn-resume {
        display: inline-flex;
        align-items: center;
        gap: 0.5rem;
        padding: 0.75rem 1.5rem;
        background: linear-gradient(135deg, rgba(34, 197, 94, 0.9), rgba(34, 197, 94, 0.7));
        color: white;
        border: none;
        border-radius: var(--radius);
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
    }
    .btn-resume:hover {
        background: linear-gradient(135deg, rgba(34, 197, 94, 1), rgba(34, 197, 94, 0.9));
        transform: translateY(-1px);
    }
    .btn-reclarify {
        display: inline-flex;
        align-items: center;
        gap: 0.5rem;
        padding: 0.75rem 1.5rem;
        background: linear-gradient(135deg, rgba(59, 130, 246, 0.9), rgba(59, 130, 246, 0.7));
        color: white;
        border: none;
        border-radius: var(--radius);
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
    }
    .btn-reclarify:hover {
        background: linear-gradient(135deg, rgba(59, 130, 246, 1), rgba(59, 130, 246, 0.9));
        transform: translateY(-1px);
    }
    .btn-abandon {
        display: inline-flex;
        align-items: center;
        gap: 0.5rem;
        padding: 0.75rem 1.5rem;
        background: linear-gradient(135deg, rgba(239, 68, 68, 0.9), rgba(239, 68, 68, 0.7));
        color: white;
        border: none;
        border-radius: var(--radius);
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
    }
    .btn-abandon:hover {
        background: linear-gradient(135deg, rgba(239, 68, 68, 1), rgba(239, 68, 68, 0.9));
        transform: translateY(-1px);
    }
    .btn-back {
        display: inline-flex;
        align-items: center;
        gap: 0.5rem;
        padding: 0.75rem 1.5rem;
        background: var(--surface);
        color: var(--ink);
        border: 1px solid var(--line);
        border-radius: var(--radius);
        font-weight: 500;
        text-decoration: none;
        transition: all 0.2s;
    }
    .btn-back:hover {
        background: var(--card);
    }
    .confirm-modal {
        display: none;
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.5);
        align-items: center;
        justify-content: center;
        z-index: 1000;
    }
    .confirm-modal-content {
        background: var(--card);
        border: 1px solid var(--line);
        border-radius: var(--radius-lg);
        padding: 1.5rem;
        max-width: 500px;
        width: 90%;
    }
    .confirm-title {
        font-size: 1.1rem;
        font-weight: 600;
        color: var(--ink-bright);
        margin-bottom: 1rem;
    }
    .confirm-message {
        font-size: 0.9rem;
        color: var(--ink);
        margin-bottom: 1.5rem;
    }
    .confirm-buttons {
        display: flex;
        justify-content: flex-end;
        gap: 0.75rem;
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
                        <h1 class="hero-title">待人工处理: ` + common.EscapeHTML(issue.Identifier) + `</h1>
                        <p class="hero-copy">` + common.EscapeHTML(issue.Title) + `</p>
                    </div>
                    <div class="status-stack">
                        <a href="/api/v1/` + common.EscapeHTML(issue.Identifier) + `" class="btn-back" style="padding: 0.5rem 1rem;">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <line x1="19" y1="12" x2="5" y2="12"></line>
                                <polyline points="12 19 5 12 12 5"></polyline>
                            </svg>
                            返回任务详情
                        </a>
                    </div>
                </div>
            </header>

            <section class="needs-attention-container">
                <div class="failure-header">
                    <span class="failure-icon">&#x26A0;</span>
                    <div>
                        <div class="failure-title">任务执行失败，需要人工干预</div>
                        <div class="failure-subtitle">已达到最大重试次数 (` + strconv.Itoa(stageState.RetryCount) + `/` + strconv.Itoa(stageState.RetryCount) + `)，无法自动恢复</div>
                    </div>
                </div>

                <!-- 失败详情 -->
                <div class="detail-grid">
                    <div class="detail-item">
                        <div class="detail-label">失败阶段</div>
                        <div class="detail-value">` + failedStageDisplay + `</div>
                    </div>
                    <div class="detail-item">
                        <div class="detail-label">失败时间</div>
                        <div class="detail-value">` + failedAtStr + `</div>
                    </div>
                    <div class="detail-item">
                        <div class="detail-label">错误类型</div>
                        <div class="detail-value">` + common.EscapeHTML(stageState.ErrorType) + `</div>
                    </div>
                    <div class="detail-item">
                        <div class="detail-label">重试次数</div>
                        <div class="detail-value">` + strconv.Itoa(stageState.RetryCount) + `</div>
                    </div>
                </div>

                <!-- 错误消息 -->
                <div class="detail-item" style="margin-bottom: 1.5rem;">
                    <div class="detail-label">错误消息</div>
                    <div class="detail-value" style="white-space: pre-wrap;">` + common.EscapeHTML(stageState.ErrorMessage) + `</div>
                </div>

                ` + func() string {
		if stageState.LastLogSnippet != "" {
			return `<div class="detail-label" style="margin-bottom: 0.5rem;">日志片段</div>
                    <div class="log-snippet">` + common.EscapeHTML(stageState.LastLogSnippet) + `</div>`
		}
		return ""
	}() + `

                ` + func() string {
		if stageState.Suggestion != "" {
			return `<div class="suggestion-box">
                    <div class="suggestion-title">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <circle cx="12" cy="12" r="10"></circle>
                            <line x1="12" y1="16" x2="12" y2="12"></line>
                            <line x1="12" y1="8" x2="12.01" y2="8"></line>
                        </svg>
                        修复建议
                    </div>
                    <div class="suggestion-content">` + common.EscapeHTML(stageState.Suggestion) + `</div>
                </div>`
		}
		return ""
	}() + `

                <!-- 操作按钮 -->
                <div class="action-buttons">
                    <a href="/" class="btn-back">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <line x1="19" y1="12" x2="5" y2="12"></line>
                            <polyline points="12 19 5 12 12 5"></polyline>
                        </svg>
                        返回看板
                    </a>
                    <button type="button" class="btn-resume" onclick="resumeTask()">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <polyline points="23 4 23 10 17 10"></polyline>
                            <path d="M20.49 15a9 9 0 1 1-2.12-5.36L23 10"></path>
                        </svg>
                        重新执行
                    </button>
                    <button type="button" class="btn-reclarify" onclick="reclarifyTask()">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path>
                            <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path>
                        </svg>
                        重新澄清需求
                    </button>
                    <button type="button" class="btn-abandon" onclick="showAbandonModal()">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <polyline points="3 6 5 6 21 6"></polyline>
                            <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                        </svg>
                        放弃任务
                    </button>
                </div>
            </section>
        </section>
    </main>

    <!-- 放弃确认对话框 -->
    <div id="abandon-modal" class="confirm-modal">
        <div class="confirm-modal-content">
            <h3 class="confirm-title">确认放弃任务</h3>
            <p class="confirm-message">放弃任务将清理工作空间并将任务状态设置为"已取消"。此操作不可撤销，请确认是否继续。</p>
            <div class="confirm-buttons">
                <button type="button" class="btn-back" onclick="hideAbandonModal()">取消</button>
                <button type="button" class="btn-abandon" onclick="abandonTask()">确认放弃</button>
            </div>
        </div>
    </div>

    <script>
    function resumeTask() {
        fetch('/api/tasks/` + common.EscapeHTML(issue.Identifier) + `/resume', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                alert('任务已恢复执行，重试计数器已清零');
                window.location.href = '/api/v1/` + common.EscapeHTML(issue.Identifier) + `';
            } else {
                alert('操作失败: ' + (data.error ? data.error.message : '未知错误'));
            }
        })
        .catch(err => {
            alert('操作失败: 网络错误');
        });
    }

    function reclarifyTask() {
        fetch('/api/tasks/` + common.EscapeHTML(issue.Identifier) + `/reclarify', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                alert('任务已流转至需求澄清阶段，BDD和架构设计已清除');
                window.location.href = '/tasks/` + common.EscapeHTML(issue.Identifier) + `';
            } else {
                alert('操作失败: ' + (data.error ? data.error.message : '未知错误'));
            }
        })
        .catch(err => {
            alert('操作失败: 网络错误');
        });
    }

    function showAbandonModal() {
        document.getElementById('abandon-modal').style.display = 'flex';
    }

    function hideAbandonModal() {
        document.getElementById('abandon-modal').style.display = 'none';
    }

    function abandonTask() {
        fetch('/api/tasks/` + common.EscapeHTML(issue.Identifier) + `/abandon', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                alert('任务已放弃，工作空间已清理');
                window.location.href = '/';
            } else {
                alert('操作失败: ' + (data.error ? data.error.message : '未知错误'));
            }
        })
        .catch(err => {
            alert('操作失败: 网络错误');
        });
    }

    // 点击模态框外部关闭
    document.getElementById('abandon-modal').addEventListener('click', function(e) {
        if (e.target === this) {
            hideAbandonModal();
        }
    });
    </script>
</body>
</html>`
}

// === 导出的辅助函数供模板引擎使用 ===

// GetStageDisplay 导出的阶段显示函数
func GetStageDisplay(stageName string) string {
	return getStageDisplay(stageName)
}

// GetStatusDisplay 导出的状态显示函数
func GetStatusDisplay(status string) string {
	return getStatusDisplay(status)
}

// FormatDurationForDetail 导出的时长格式化函数
func FormatDurationForDetail(seconds int64) string {
	return formatDurationForDetail(seconds)
}

// RenderConversationHistoryHTML 导出的对话历史渲染函数
func RenderConversationHistoryHTML(history []domain.ConversationTurn) template.HTML {
	return template.HTML(renderConversationHistory(history))
}

// FormatBDDContentHTML 导出的 BDD 内容格式化函数
func FormatBDDContentHTML(content string) template.HTML {
	return template.HTML(formatBDDContent(content))
}

// FormatArchitectureContentHTML 导出的架构内容格式化函数
func FormatArchitectureContentHTML(content string) template.HTML {
	return template.HTML(formatArchitectureContent(content))
}

// FormatTDDContentHTML 导出的 TDD 内容格式化函数
func FormatTDDContentHTML(content string) template.HTML {
	return template.HTML(formatTDDContent(content))
}
