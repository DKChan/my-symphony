package components

import (
	"strconv"
	"time"

	"github.com/dministrator/symphony/internal/common"
	"github.com/dministrator/symphony/internal/domain"
)

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
                        <div class="card-row" style="margin-top: 0.5rem;">
                            <a class="issue-link" href="/api/v1/` + entry.Identifier + `">查看 JSON 详情 →</a>
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
    </script>
</body>
</html>`

	return html
}