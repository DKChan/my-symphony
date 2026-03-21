package components

import (
	"strconv"
	"time"

	"github.com/dministrator/symphony/internal/common"
	"github.com/dministrator/symphony/internal/domain"
)

// RenderRunningSessions 渲染正在运行的会话表格
func RenderRunningSessions(state *domain.OrchestratorState, now time.Time) string {
	if len(state.Running) == 0 {
		return `<p class="empty-state">No active sessions.</p>`
	}

	html := `<div class="table-wrap">
                <table class="data-table data-table-running">
                    <colgroup>
                        <col style="width: 12rem;">
                        <col style="width: 8rem;">
                        <col style="width: 7.5rem;">
                        <col style="width: 8.5rem;">
                        <col>
                        <col style="width: 10rem;">
                    </colgroup>
                    <thead>
                        <tr>
                            <th>Issue</th>
                            <th>State</th>
                            <th>Session</th>
                            <th>Runtime / turns</th>
                            <th>Codex update</th>
                            <th>Tokens</th>
                        </tr>
                    </thead>
                    <tbody>`

	for _, entry := range state.Running {
		sessionID := ""
		if entry.Session != nil {
			sessionID = entry.Session.SessionID
		}

		stateClass := common.StateBadgeClass(entry.Issue.State)
		runtimeTurns := common.FormatRuntimeAndTurns(entry.StartedAt, entry.TurnCount, now)

		lastEvent := "n/a"
		lastMessage := ""
		lastEventAt := ""
		if entry.Session != nil {
			if entry.Session.LastCodexEvent != nil {
				lastEvent = *entry.Session.LastCodexEvent
			}
			if entry.Session.LastCodexTimestamp != nil {
				lastEventAt = entry.Session.LastCodexTimestamp.Format("15:04:05")
			}
		}

		var tokens common.Tokens
		if entry.Session != nil {
			tokens = common.Tokens{
				InputTokens:  entry.Session.CodexInputTokens,
				OutputTokens: entry.Session.CodexOutputTokens,
				TotalTokens:  entry.Session.CodexTotalTokens,
			}
		}

		html += `
                            <tr>
                                <td>
                                    <div class="issue-stack">
                                        <span class="issue-id">` + entry.Identifier + `</span>
                                        <a class="issue-link" href="/api/v1/` + entry.Identifier + `">JSON details</a>
                                    </div>
                                </td>
                                <td>
                                    <span class="` + stateClass + `">` + entry.Issue.State + `</span>
                                </td>
                                <td>
                                    <div class="session-stack">`
		if sessionID != "" {
			html += `
                                        <button type="button" class="subtle-button" data-label="Copy ID" data-copy="` + sessionID + `" onclick="navigator.clipboard.writeText(this.dataset.copy); this.textContent = 'Copied'; clearTimeout(this._copyTimer); this._copyTimer = setTimeout(() => { this.textContent = this.dataset.label }, 1200);">Copy ID</button>`
		} else {
			html += `
                                        <span class="muted">n/a</span>`
		}
		html += `
                                    </div>
                                </td>
                                <td class="numeric">` + runtimeTurns + `</td>
                                <td>
                                    <div class="detail-stack">
                                        <span class="event-text" title="` + common.EscapeHTML(lastMessage) + common.EscapeHTML(lastEvent) + `">` + common.EscapeHTML(lastMessage) + common.EscapeHTML(lastEvent) + `</span>
                                        <span class="muted event-meta">
                                            ` + common.EscapeHTML(lastEvent) + `
                                            ` + func() string {
			if lastEventAt != "" {
				return `· <span class="mono numeric">` + lastEventAt + `</span>`
			}
			return ""
		}() + `
                                        </span>
                                    </div>
                                </td>
                                <td>
                                    <div class="token-stack numeric">
                                        <span>Total: ` + common.FormatInt(tokens.TotalTokens) + `</span>
                                        <span class="muted">In ` + common.FormatInt(tokens.InputTokens) + ` / Out ` + common.FormatInt(tokens.OutputTokens) + `</span>
                                    </div>
                                </td>
                            </tr>`
	}

	html += `
                    </tbody>
                </table>
            </div>`

	return html
}

// RenderRetryQueue 渲染重试队列表格
func RenderRetryQueue(state *domain.OrchestratorState) string {
	if len(state.RetryAttempts) == 0 {
		return `<p class="empty-state">No issues are currently backing off.</p>`
	}

	html := `<div class="table-wrap">
                <table class="data-table" style="min-width: 680px;">
                    <thead>
                        <tr>
                            <th>Issue</th>
                            <th>Attempt</th>
                            <th>Due at</th>
                            <th>Error</th>
                        </tr>
                    </thead>
                    <tbody>`

	for _, entry := range state.RetryAttempts {
		dueAt := time.UnixMilli(entry.DueAtMs).Format("15:04:05")
		errMsg := ""
		if entry.Error != nil {
			errMsg = *entry.Error
		}

		html += `
                            <tr>
                                <td>
                                    <div class="issue-stack">
                                        <span class="issue-id">` + entry.Identifier + `</span>
                                        <a class="issue-link" href="/api/v1/` + entry.Identifier + `">JSON details</a>
                                    </div>
                                </td>
                                <td>` + strconv.Itoa(entry.Attempt) + `</td>
                                <td class="mono">` + dueAt + `</td>
                                <td>` + common.EscapeHTML(errMsg) + `</td>
                            </tr>`
	}

	html += `
                    </tbody>
                </table>
            </div>`

	return html
}

// RenderDashboardHTML 渲染完整的仪表板 HTML
func RenderDashboardHTML(state *domain.OrchestratorState, now time.Time) string {
	html := `<!DOCTYPE html>
<html lang="zh">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Symphony Observability</title>
    <link rel="stylesheet" href="/dashboard.css">
    <script src="https://unpkg.com/htmx.org@2.0.4/dist/htmx.min.js"></script>
    <script src="https://unpkg.com/htmx-ext-sse@2.2.2/sse.js"></script>
</head>
<body>
    <main class="app-shell" hx-ext="sse" sse-connect="/events" sse-swap="state">
        <section class="dashboard-shell">
            <header class="hero-card">
                <div class="hero-grid">
                    <div>
                        <p class="eyebrow">Symphony Observability</p>
                        <h1 class="hero-title">Operations Dashboard</h1>
                        <p class="hero-copy">Current state, retry pressure, token usage, and orchestration health for the active Symphony runtime.</p>
                    </div>
                    <div class="status-stack">
                        <span class="status-badge status-badge-live" id="live-indicator">
                            <span class="status-badge-dot"></span>
                            Live
                        </span>
                    </div>
                </div>
            </header>

            <section class="metric-grid" id="metrics">
                <article class="metric-card">
                    <p class="metric-label">Running</p>
                    <p class="metric-value numeric" id="metric-running">` + strconv.Itoa(len(state.Running)) + `</p>
                    <p class="metric-detail">Active issue sessions in the current runtime.</p>
                </article>

                <article class="metric-card">
                    <p class="metric-label">Retrying</p>
                    <p class="metric-value numeric" id="metric-retrying">` + strconv.Itoa(len(state.RetryAttempts)) + `</p>
                    <p class="metric-detail">Issues waiting for the next retry window.</p>
                </article>

                <article class="metric-card">
                    <p class="metric-label">Total tokens</p>
                    <p class="metric-value numeric" id="metric-tokens">` + common.FormatInt(state.CodexTotals.TotalTokens) + `</p>
                    <p class="metric-detail numeric" id="metric-tokens-detail">In ` + common.FormatInt(state.CodexTotals.InputTokens) + ` / Out ` + common.FormatInt(state.CodexTotals.OutputTokens) + `</p>
                </article>

                <article class="metric-card">
                    <p class="metric-label">Runtime</p>
                    <p class="metric-value numeric" id="metric-runtime">` + common.FormatRuntimeSeconds(common.TotalRuntimeSeconds(state, now)) + `</p>
                    <p class="metric-detail">Total Codex runtime across completed and active sessions.</p>
                </article>
            </section>

            <section class="section-card">
                <div class="section-header">
                    <div>
                        <h2 class="section-title">Rate limits</h2>
                        <p class="section-copy">Latest upstream rate-limit snapshot, when available.</p>
                    </div>
                </div>
                <pre class="code-panel" id="rate-limits">` + common.PrettyValue(state.CodexRateLimits) + `</pre>
            </section>

            <section class="section-card">
                <div class="section-header">
                    <div>
                        <h2 class="section-title">Running sessions</h2>
                        <p class="section-copy">Active issues, last known agent activity, and token usage.</p>
                    </div>
                </div>
                <div id="running-sessions">` + RenderRunningSessions(state, now) + `</div>
            </section>

            <section class="section-card">
                <div class="section-header">
                    <div>
                        <h2 class="section-title">Retry queue</h2>
                        <p class="section-copy">Issues waiting for the next retry window.</p>
                    </div>
                </div>
                <div id="retry-queue">` + RenderRetryQueue(state) + `</div>
            </section>
        </section>
    </main>
    <script>
    document.body.addEventListener('htmx:sseMessage', function(evt) {
        if (evt.detail.type === 'state') {
            try {
                const data = JSON.parse(evt.detail.data);
                updateDashboard(data);
            } catch (e) {
                console.error('Failed to parse SSE data:', e);
            }
        }
    });

    function updateDashboard(data) {
        // 更新指标
        document.getElementById('metric-running').textContent = data.counts.running;
        document.getElementById('metric-retrying').textContent = data.counts.retrying;
        document.getElementById('metric-tokens').textContent = formatNumber(data.codex_totals.total_tokens);
        document.getElementById('metric-tokens-detail').textContent = 'In ' + formatNumber(data.codex_totals.input_tokens) + ' / Out ' + formatNumber(data.codex_totals.output_tokens);
        document.getElementById('rate-limits').textContent = JSON.stringify(data.rate_limits || 'n/a', null, 2);

        // 更新运行会话
        document.getElementById('running-sessions').innerHTML = renderRunningTable(data.running);

        // 更新重试队列
        document.getElementById('retry-queue').innerHTML = renderRetryTable(data.retrying);

        // 闪烁Live指示器
        const indicator = document.getElementById('live-indicator');
        indicator.style.animation = 'pulse 0.3s ease';
        setTimeout(() => indicator.style.animation = '', 300);
    }

    function formatNumber(n) {
        if (n >= 1000000) return (n / 1000000).toFixed(1) + 'M';
        if (n >= 1000) return (n / 1000).toFixed(1) + 'K';
        return n.toString();
    }

    function renderRunningTable(running) {
        if (!running || running.length === 0) {
            return '<p class="empty-state">No active sessions.</p>';
        }

        let html = '<div class="table-wrap"><table class="data-table data-table-running">' +
            '<colgroup>' +
            '<col style="width: 12rem;">' +
            '<col style="width: 8rem;">' +
            '<col style="width: 7.5rem;">' +
            '<col style="width: 8.5rem;">' +
            '<col>' +
            '<col style="width: 10rem;">' +
            '</colgroup>' +
            '<thead><tr>' +
            '<th>Issue</th>' +
            '<th>State</th>' +
            '<th>Session</th>' +
            '<th>Runtime / turns</th>' +
            '<th>Codex update</th>' +
            '<th>Tokens</th>' +
            '</tr></thead><tbody>';

        running.forEach(entry => {
            const stateClass = getStateBadgeClass(entry.state);
            html += '<tr>' +
                '<td><div class="issue-stack">' +
                '<span class="issue-id">' + escapeHtml(entry.issue_identifier) + '</span>' +
                '<a class="issue-link" href="/api/v1/' + escapeHtml(entry.issue_identifier) + '">JSON details</a>' +
                '</div></td>' +
                '<td><span class="' + stateClass + '">' + escapeHtml(entry.state) + '</span></td>' +
                '<td><div class="session-stack">' +
                (entry.session_id ? '<button type="button" class="subtle-button" onclick="copyId(this, \\'' + escapeHtml(entry.session_id) + '\\')">Copy ID</button>' : '<span class="muted">n/a</span>') +
                '</div></td>' +
                '<td class="numeric">' + escapeHtml(entry.runtime_turns || 'n/a') + '</td>' +
                '<td><div class="detail-stack">' +
                '<span class="event-text" title="' + escapeHtml(entry.last_message || entry.last_event || 'n/a') + '">' + escapeHtml(entry.last_message || entry.last_event || 'n/a') + '</span>' +
                '<span class="muted event-meta">' + escapeHtml(entry.last_event || 'n/a') +
                (entry.last_event_at ? ' · <span class="mono numeric">' + escapeHtml(entry.last_event_at) + '</span>' : '') +
                '</span></div></td>' +
                '<td><div class="token-stack numeric">' +
                '<span>Total: ' + formatNumber(entry.tokens.total_tokens) + '</span>' +
                '<span class="muted">In ' + formatNumber(entry.tokens.input_tokens) + ' / Out ' + formatNumber(entry.tokens.output_tokens) + '</span>' +
                '</div></td></tr>';
        });

        html += '</tbody></table></div>';
        return html;
    }

    function renderRetryTable(retrying) {
        if (!retrying || retrying.length === 0) {
            return '<p class="empty-state">No issues are currently backing off.</p>';
        }

        let html = '<div class="table-wrap"><table class="data-table" style="min-width: 680px;">' +
            '<thead><tr>' +
            '<th>Issue</th>' +
            '<th>Attempt</th>' +
            '<th>Due at</th>' +
            '<th>Error</th>' +
            '</tr></thead><tbody>';

        retrying.forEach(entry => {
            html += '<tr>' +
                '<td><div class="issue-stack">' +
                '<span class="issue-id">' + escapeHtml(entry.issue_identifier) + '</span>' +
                '<a class="issue-link" href="/api/v1/' + escapeHtml(entry.issue_identifier) + '">JSON details</a>' +
                '</div></td>' +
                '<td>' + entry.attempt + '</td>' +
                '<td class="mono">' + escapeHtml(entry.due_at || 'n/a') + '</td>' +
                '<td>' + escapeHtml(entry.error || 'n/a') + '</td></tr>';
        });

        html += '</tbody></table></div>';
        return html;
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
        btn.textContent = 'Copied';
        clearTimeout(btn._copyTimer);
        btn._copyTimer = setTimeout(() => btn.textContent = 'Copy ID', 1200);
    }
    </script>
</body>
</html>`

	return html
}
