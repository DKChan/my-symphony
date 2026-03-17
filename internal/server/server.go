// Package server 提供HTTP服务器实现
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dministrator/symphony/internal/domain"
	"github.com/dministrator/symphony/internal/orchestrator"
	"github.com/gin-gonic/gin"
)

// Server HTTP服务器
type Server struct {
	orchestrator *orchestrator.Orchestrator
	port         int
	engine       *gin.Engine

	// SSE 广播器
	broadcaster *SSEBroadcaster
}

// SSEBroadcaster SSE广播器
type SSEBroadcaster struct {
	mu          sync.RWMutex
	clients     map[chan *SSEEvent]struct{}
	lastPayload *StatePayload
}

// SSEEvent SSE事件
type SSEEvent struct {
	Event string
	Data  string
}

// StatePayload 状态载荷
type StatePayload struct {
	GeneratedAt string                 `json:"generated_at"`
	Counts      StateCounts            `json:"counts"`
	Running     []RunningEntryPayload  `json:"running"`
	Retrying    []RetryEntryPayload    `json:"retrying"`
	CodexTotals domain.CodexTotals     `json:"codex_totals"`
	RateLimits  any                    `json:"rate_limits"`
}

// StateCounts 状态计数
type StateCounts struct {
	Running  int `json:"running"`
	Retrying int `json:"retrying"`
}

// RunningEntryPayload 运行条目载荷
type RunningEntryPayload struct {
	IssueID         string `json:"issue_id"`
	IssueIdentifier string `json:"issue_identifier"`
	State           string `json:"state"`
	SessionID       string `json:"session_id"`
	TurnCount       int    `json:"turn_count"`
	LastEvent       string `json:"last_event"`
	LastMessage     string `json:"last_message"`
	StartedAt       string `json:"started_at"`
	LastEventAt     string `json:"last_event_at"`
	Tokens          Tokens `json:"tokens"`
}

// RetryEntryPayload 重试条目载荷
type RetryEntryPayload struct {
	IssueID         string `json:"issue_id"`
	IssueIdentifier string `json:"issue_identifier"`
	Attempt         int    `json:"attempt"`
	DueAt           string `json:"due_at"`
	Error           string `json:"error"`
}

// Tokens Token统计
type Tokens struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	TotalTokens  int64 `json:"total_tokens"`
}

// NewSSEBroadcaster 创建SSE广播器
func NewSSEBroadcaster() *SSEBroadcaster {
	return &SSEBroadcaster{
		clients: make(map[chan *SSEEvent]struct{}),
	}
}

// Subscribe 订阅
func (b *SSEBroadcaster) Subscribe() chan *SSEEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan *SSEEvent, 10)
	b.clients[ch] = struct{}{}
	return ch
}

// Unsubscribe 取消订阅
func (b *SSEBroadcaster) Unsubscribe(ch chan *SSEEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.clients, ch)
	close(ch)
}

// Broadcast 广播
func (b *SSEBroadcaster) Broadcast(event string, payload *StatePayload) {
	b.mu.Lock()
	b.lastPayload = payload
	clients := make(map[chan *SSEEvent]struct{})
	for k, v := range b.clients {
		clients[k] = v
	}
	b.mu.Unlock()

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	evt := &SSEEvent{
		Event: event,
		Data:  string(data),
	}

	for ch := range clients {
		select {
		case ch <- evt:
		default:
			// 客户端阻塞，跳过
		}
	}
}

// NewServer 创建新的HTTP服务器
func NewServer(orch *orchestrator.Orchestrator, port int) *Server {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())

	s := &Server{
		orchestrator: orch,
		port:         port,
		engine:       engine,
		broadcaster:  NewSSEBroadcaster(),
	}

	// 设置状态变更回调
	orch.SetOnStateChange(s.onStateChange)

	s.setupRoutes()
	return s
}

// onStateChange 状态变更回调
func (s *Server) onStateChange() {
	payload := s.buildStatePayload()
	s.broadcaster.Broadcast("state", payload)
}

// buildStatePayload 构建状态载荷
func (s *Server) buildStatePayload() *StatePayload {
	state := s.orchestrator.GetState()

	running := make([]RunningEntryPayload, 0)
	for _, entry := range state.Running {
		r := RunningEntryPayload{
			IssueIdentifier: entry.Identifier,
			TurnCount:       entry.TurnCount,
			StartedAt:       entry.StartedAt.Format(time.RFC3339),
		}

		if entry.Issue != nil {
			r.IssueID = entry.Issue.ID
			r.State = entry.Issue.State
		}

		if entry.Session != nil {
			r.SessionID = entry.Session.SessionID
			if entry.Session.LastCodexEvent != nil {
				r.LastEvent = *entry.Session.LastCodexEvent
			}
			if entry.Session.LastCodexTimestamp != nil {
				r.LastEventAt = entry.Session.LastCodexTimestamp.Format(time.RFC3339)
			}
			r.Tokens = Tokens{
				InputTokens:  entry.Session.CodexInputTokens,
				OutputTokens: entry.Session.CodexOutputTokens,
				TotalTokens:  entry.Session.CodexTotalTokens,
			}
		}

		running = append(running, r)
	}

	retrying := make([]RetryEntryPayload, 0)
	for _, entry := range state.RetryAttempts {
		errMsg := ""
		if entry.Error != nil {
			errMsg = *entry.Error
		}
		retrying = append(retrying, RetryEntryPayload{
			IssueID:         entry.IssueID,
			IssueIdentifier: entry.Identifier,
			Attempt:         entry.Attempt,
			DueAt:           time.UnixMilli(entry.DueAtMs).Format(time.RFC3339),
			Error:           errMsg,
		})
	}

	totals := domain.CodexTotals{}
	if state.CodexTotals != nil {
		totals = *state.CodexTotals
	}

	return &StatePayload{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Counts: StateCounts{
			Running:  len(state.Running),
			Retrying: len(state.RetryAttempts),
		},
		Running:     running,
		Retrying:    retrying,
		CodexTotals: totals,
		RateLimits:  state.CodexRateLimits,
	}
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// 静态文件
	s.engine.GET("/dashboard.css", s.handleDashboardCSS)

	// 主页 - 仪表板
	s.engine.GET("/", s.handleDashboard)

	// SSE 端点
	s.engine.GET("/events", s.handleSSE)

	// API路由
	api := s.engine.Group("/api/v1")
	{
		api.GET("/state", s.handleGetState)
		api.GET("/:identifier", s.handleGetIssue)
		api.POST("/refresh", s.handleRefresh)
	}
}

// Run 运行服务器
func (s *Server) Run() error {
	return s.engine.Run(":" + strconv.Itoa(s.port))
}

// handleDashboardCSS 处理CSS文件请求
func (s *Server) handleDashboardCSS(c *gin.Context) {
	c.Header("Content-Type", "text/css; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=3600")
	c.String(http.StatusOK, dashboardCSS)
}

// handleSSE 处理SSE请求
func (s *Server) handleSSE(c *gin.Context) {
	// 设置SSE头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// 订阅
	ch := s.broadcaster.Subscribe()
	defer s.broadcaster.Unsubscribe(ch)

	// 发送初始状态
	s.broadcaster.mu.RLock()
	payload := s.broadcaster.lastPayload
	s.broadcaster.mu.RUnlock()

	if payload != nil {
		data, _ := json.Marshal(payload)
		fmt.Fprintf(c.Writer, "event: state\ndata: %s\n\n", string(data))
		c.Writer.(http.Flusher).Flush()
	}

	// 流式传输
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case evt := <-ch:
			fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", evt.Event, evt.Data)
			c.Writer.(http.Flusher).Flush()
		}
	}
}

// handleDashboard 处理仪表板请求
func (s *Server) handleDashboard(c *gin.Context) {
	state := s.orchestrator.GetState()
	now := time.Now()

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
                    <p class="metric-value numeric" id="metric-tokens">` + formatInt(state.CodexTotals.TotalTokens) + `</p>
                    <p class="metric-detail numeric" id="metric-tokens-detail">In ` + formatInt(state.CodexTotals.InputTokens) + ` / Out ` + formatInt(state.CodexTotals.OutputTokens) + `</p>
                </article>

                <article class="metric-card">
                    <p class="metric-label">Runtime</p>
                    <p class="metric-value numeric" id="metric-runtime">` + formatRuntimeSeconds(totalRuntimeSeconds(state, now)) + `</p>
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
                <pre class="code-panel" id="rate-limits">` + prettyValue(state.CodexRateLimits) + `</pre>
            </section>

            <section class="section-card">
                <div class="section-header">
                    <div>
                        <h2 class="section-title">Running sessions</h2>
                        <p class="section-copy">Active issues, last known agent activity, and token usage.</p>
                    </div>
                </div>
                <div id="running-sessions">` + renderRunningSessions(state, now) + `</div>
            </section>

            <section class="section-card">
                <div class="section-header">
                    <div>
                        <h2 class="section-title">Retry queue</h2>
                        <p class="section-copy">Issues waiting for the next retry window.</p>
                    </div>
                </div>
                <div id="retry-queue">` + renderRetryQueue(state) + `</div>
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

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// renderRunningSessions 渲染运行会话
func renderRunningSessions(state *domain.OrchestratorState, now time.Time) string {
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

		stateClass := stateBadgeClass(entry.Issue.State)
		runtimeTurns := formatRuntimeAndTurns(entry.StartedAt, entry.TurnCount, now)

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

		var tokens Tokens
		if entry.Session != nil {
			tokens = Tokens{
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
                                        <span class="event-text" title="` + escapeHTML(lastMessage) + escapeHTML(lastEvent) + `">` + escapeHTML(lastMessage) + escapeHTML(lastEvent) + `</span>
                                        <span class="muted event-meta">
                                            ` + escapeHTML(lastEvent) + `
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
                                        <span>Total: ` + formatInt(tokens.TotalTokens) + `</span>
                                        <span class="muted">In ` + formatInt(tokens.InputTokens) + ` / Out ` + formatInt(tokens.OutputTokens) + `</span>
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

// renderRetryQueue 渲染重试队列
func renderRetryQueue(state *domain.OrchestratorState) string {
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
                                <td>` + escapeHTML(errMsg) + `</td>
                            </tr>`
	}

	html += `
                    </tbody>
                </table>
            </div>`

	return html
}

// handleGetState 处理状态请求
func (s *Server) handleGetState(c *gin.Context) {
	payload := s.buildStatePayload()
	c.JSON(http.StatusOK, payload)
}

// handleGetIssue 处理单个问题请求
func (s *Server) handleGetIssue(c *gin.Context) {
	identifier := c.Param("identifier")

	state := s.orchestrator.GetState()

	// 查找问题
	var foundEntry *domain.RunningEntry
	for _, entry := range state.Running {
		if entry.Identifier == identifier {
			foundEntry = entry
			break
		}
	}

	// 查找重试条目
	var foundRetry *domain.RetryEntry
	for _, entry := range state.RetryAttempts {
		if entry.Identifier == identifier {
			foundRetry = entry
			break
		}
	}

	if foundEntry == nil && foundRetry == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"code":    "issue_not_found",
				"message": "issue not found in current state",
			},
		})
		return
	}

	response := map[string]any{
		"issue_identifier": identifier,
		"status":          "unknown",
	}

	if foundEntry != nil {
		response["issue_id"] = foundEntry.Issue.ID
		response["status"] = "running"

		if foundEntry.Issue != nil {
			response["workspace"] = map[string]string{
				"path": "/tmp/symphony_workspaces/" + identifier,
			}
		}

		running := map[string]any{
			"session_id":  "",
			"turn_count":  foundEntry.TurnCount,
			"started_at":  foundEntry.StartedAt.Format(time.RFC3339),
		}

		if foundEntry.Issue != nil {
			running["state"] = foundEntry.Issue.State
		}

		if foundEntry.Session != nil {
			running["session_id"] = foundEntry.Session.SessionID
			if foundEntry.Session.LastCodexEvent != nil {
				running["last_event"] = *foundEntry.Session.LastCodexEvent
			}
			if foundEntry.Session.LastCodexTimestamp != nil {
				running["last_event_at"] = foundEntry.Session.LastCodexTimestamp.Format(time.RFC3339)
			}
			running["tokens"] = map[string]int64{
				"input_tokens":  foundEntry.Session.CodexInputTokens,
				"output_tokens": foundEntry.Session.CodexOutputTokens,
				"total_tokens":  foundEntry.Session.CodexTotalTokens,
			}
		}

		response["running"] = running
	}

	if foundRetry != nil {
		errMsg := ""
		if foundRetry.Error != nil {
			errMsg = *foundRetry.Error
		}
		response["retry"] = map[string]any{
			"attempt": foundRetry.Attempt,
			"due_at":  time.UnixMilli(foundRetry.DueAtMs).Format(time.RFC3339),
			"error":   errMsg,
		}
		if response["status"] == "unknown" {
			response["status"] = "retrying"
		}
	}

	c.JSON(http.StatusOK, response)
}

// handleRefresh 处理刷新请求
func (s *Server) handleRefresh(c *gin.Context) {
	response := map[string]any{
		"queued":       true,
		"coalesced":    false,
		"requested_at": time.Now().UTC().Format(time.RFC3339),
		"operations":   []string{"poll", "reconcile"},
	}
	c.JSON(http.StatusAccepted, response)
}

// 辅助函数
func formatInt(value int64) string {
	if value >= 1000000 {
		return strconv.FormatFloat(float64(value)/1000000, 'f', 1, 64) + "M"
	}
	if value >= 1000 {
		return strconv.FormatFloat(float64(value)/1000, 'f', 1, 64) + "K"
	}
	return strconv.FormatInt(value, 10)
}

func formatRuntimeSeconds(seconds int) string {
	mins := seconds / 60
	secs := seconds % 60
	return strconv.Itoa(mins) + "m " + strconv.Itoa(secs) + "s"
}

func totalRuntimeSeconds(state *domain.OrchestratorState, now time.Time) int {
	completed := int(state.CodexTotals.SecondsRunning)
	for _, entry := range state.Running {
		completed += int(now.Sub(entry.StartedAt).Seconds())
	}
	return completed
}

func formatRuntimeAndTurns(startedAt time.Time, turnCount int, now time.Time) string {
	seconds := int(now.Sub(startedAt).Seconds())
	runtime := formatRuntimeSeconds(seconds)
	if turnCount > 0 {
		return runtime + " / " + strconv.Itoa(turnCount)
	}
	return runtime
}

func stateBadgeClass(state string) string {
	base := "state-badge"
	normalized := strings.ToLower(state)

	switch {
	case strings.Contains(normalized, "progress") ||
		strings.Contains(normalized, "running") ||
		strings.Contains(normalized, "active"):
		return base + " state-badge-active"
	case strings.Contains(normalized, "blocked") ||
		strings.Contains(normalized, "error") ||
		strings.Contains(normalized, "failed"):
		return base + " state-badge-danger"
	case strings.Contains(normalized, "todo") ||
		strings.Contains(normalized, "queued") ||
		strings.Contains(normalized, "pending") ||
		strings.Contains(normalized, "retry"):
		return base + " state-badge-warning"
	default:
		return base
	}
}

func prettyValue(v any) string {
	if v == nil {
		return "n/a"
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "n/a"
	}
	return string(b)
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

// dashboardCSS 内联CSS样式
const dashboardCSS = `:root {
  color-scheme: light;
  --page: #f7f7f8;
  --page-soft: #fbfbfc;
  --page-deep: #ececf1;
  --card: rgba(255, 255, 255, 0.94);
  --card-muted: #f3f4f6;
  --ink: #202123;
  --muted: #6e6e80;
  --line: #ececf1;
  --line-strong: #d9d9e3;
  --accent: #10a37f;
  --accent-ink: #0f513f;
  --accent-soft: #e8faf4;
  --danger: #b42318;
  --danger-soft: #fef3f2;
  --shadow-sm: 0 1px 2px rgba(16, 24, 40, 0.05);
  --shadow-lg: 0 20px 50px rgba(15, 23, 42, 0.08);
}

* {
  box-sizing: border-box;
}

html {
  background: var(--page);
}

body {
  margin: 0;
  min-height: 100vh;
  background:
    radial-gradient(circle at top, rgba(16, 163, 127, 0.12) 0%, rgba(16, 163, 127, 0) 30%),
    linear-gradient(180deg, var(--page-soft) 0%, var(--page) 24%, #f3f4f6 100%);
  color: var(--ink);
  font-family: "Sohne", "SF Pro Text", "Helvetica Neue", "Segoe UI", sans-serif;
  line-height: 1.5;
}

a {
  color: var(--ink);
  text-decoration: none;
  transition: color 140ms ease;
}

a:hover {
  color: var(--accent);
}

button {
  appearance: none;
  border: 1px solid var(--accent);
  background: var(--accent);
  color: white;
  border-radius: 999px;
  padding: 0.72rem 1.08rem;
  cursor: pointer;
  font: inherit;
  font-weight: 600;
  letter-spacing: -0.01em;
  box-shadow: 0 8px 20px rgba(16, 163, 127, 0.18);
  transition:
    transform 140ms ease,
    box-shadow 140ms ease,
    background 140ms ease,
    border-color 140ms ease;
}

button:hover {
  transform: translateY(-1px);
  box-shadow: 0 12px 24px rgba(16, 163, 127, 0.22);
}

button.secondary {
  background: var(--card);
  color: var(--ink);
  border-color: var(--line-strong);
  box-shadow: var(--shadow-sm);
}

button.secondary:hover {
  box-shadow: 0 6px 16px rgba(15, 23, 42, 0.08);
}

.subtle-button {
  appearance: none;
  border: 1px solid var(--line-strong);
  background: rgba(255, 255, 255, 0.72);
  color: var(--muted);
  border-radius: 999px;
  padding: 0.34rem 0.72rem;
  cursor: pointer;
  font: inherit;
  font-size: 0.82rem;
  font-weight: 600;
  letter-spacing: 0.01em;
  box-shadow: none;
  transition:
    background 140ms ease,
    border-color 140ms ease,
    color 140ms ease;
}

.subtle-button:hover {
  transform: none;
  box-shadow: none;
  background: white;
  border-color: var(--muted);
  color: var(--ink);
}

pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
}

code,
pre,
.mono {
  font-family: "Sohne Mono", "SFMono-Regular", "SF Mono", Consolas, "Liberation Mono", monospace;
}

.mono,
.numeric {
  font-variant-numeric: tabular-nums slashed-zero;
  font-feature-settings: "tnum" 1, "zero" 1;
}

.app-shell {
  max-width: 1280px;
  margin: 0 auto;
  padding: 2rem 1rem 3.5rem;
}

.dashboard-shell {
  display: grid;
  gap: 1rem;
}

.hero-card,
.section-card,
.metric-card,
.error-card {
  background: var(--card);
  border: 1px solid rgba(217, 217, 227, 0.82);
  box-shadow: var(--shadow-sm);
  backdrop-filter: blur(18px);
}

.hero-card {
  border-radius: 28px;
  padding: clamp(1.25rem, 3vw, 2rem);
  box-shadow: var(--shadow-lg);
}

.hero-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 1.25rem;
  align-items: start;
}

.eyebrow {
  margin: 0;
  color: var(--muted);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-size: 0.76rem;
  font-weight: 600;
}

.hero-title {
  margin: 0.35rem 0 0;
  font-size: clamp(2rem, 4vw, 3.3rem);
  line-height: 0.98;
  letter-spacing: -0.04em;
}

.hero-copy {
  margin: 0.75rem 0 0;
  max-width: 46rem;
  color: var(--muted);
  font-size: 1rem;
}

.status-stack {
  display: grid;
  justify-items: end;
  align-content: start;
  min-width: min(100%, 9rem);
}

.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  min-height: 2rem;
  padding: 0.35rem 0.78rem;
  border-radius: 999px;
  border: 1px solid var(--line);
  background: var(--card-muted);
  color: var(--muted);
  font-size: 0.82rem;
  font-weight: 700;
  letter-spacing: 0.01em;
}

.status-badge-dot {
  width: 0.52rem;
  height: 0.52rem;
  border-radius: 999px;
  background: currentColor;
  opacity: 0.9;
}

.status-badge-live {
  background: var(--accent-soft);
  border-color: rgba(16, 163, 127, 0.18);
  color: var(--accent-ink);
}

.metric-grid {
  display: grid;
  gap: 0.85rem;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
}

.metric-card {
  border-radius: 22px;
  padding: 1rem 1.05rem 1.1rem;
}

.metric-label {
  margin: 0;
  color: var(--muted);
  font-size: 0.82rem;
  font-weight: 600;
  letter-spacing: 0.01em;
}

.metric-value {
  margin: 0.35rem 0 0;
  font-size: clamp(1.6rem, 2vw, 2.1rem);
  line-height: 1.05;
  letter-spacing: -0.03em;
}

.metric-detail {
  margin: 0.45rem 0 0;
  color: var(--muted);
  font-size: 0.88rem;
}

.section-card {
  border-radius: 24px;
  padding: 1.15rem;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
  flex-wrap: wrap;
}

.section-title {
  margin: 0;
  font-size: 1.08rem;
  line-height: 1.2;
  letter-spacing: -0.02em;
}

.section-copy {
  margin: 0.35rem 0 0;
  color: var(--muted);
  font-size: 0.94rem;
}

.table-wrap {
  overflow-x: auto;
  margin-top: 1rem;
}

.data-table {
  width: 100%;
  min-width: 720px;
  border-collapse: collapse;
}

.data-table-running {
  table-layout: fixed;
  min-width: 980px;
}

.data-table th {
  padding: 0 0.5rem 0.75rem 0;
  text-align: left;
  color: var(--muted);
  font-size: 0.78rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.data-table td {
  padding: 0.9rem 0.5rem 0.9rem 0;
  border-top: 1px solid var(--line);
  vertical-align: top;
  font-size: 0.94rem;
}

.issue-stack,
.session-stack,
.detail-stack,
.token-stack {
  display: grid;
  gap: 0.24rem;
  min-width: 0;
}

.event-text {
  font-weight: 500;
  line-height: 1.45;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.event-meta {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.state-badge {
  display: inline-flex;
  align-items: center;
  min-height: 1.85rem;
  padding: 0.3rem 0.68rem;
  border-radius: 999px;
  border: 1px solid var(--line);
  background: var(--card-muted);
  color: var(--ink);
  font-size: 0.8rem;
  font-weight: 600;
  line-height: 1;
}

.state-badge-active {
  background: var(--accent-soft);
  border-color: rgba(16, 163, 127, 0.18);
  color: var(--accent-ink);
}

.state-badge-warning {
  background: #fff7e8;
  border-color: #f1d8a6;
  color: #8a5a00;
}

.state-badge-danger {
  background: var(--danger-soft);
  border-color: #f6d3cf;
  color: var(--danger);
}

.issue-id {
  font-weight: 600;
  letter-spacing: -0.01em;
}

.issue-link {
  color: var(--muted);
  font-size: 0.86rem;
}

.muted {
  color: var(--muted);
}

.code-panel {
  margin-top: 1rem;
  padding: 1rem;
  border-radius: 18px;
  background: #f5f5f7;
  border: 1px solid var(--line);
  color: #353740;
  font-size: 0.9rem;
}

.empty-state {
  margin: 1rem 0 0;
  color: var(--muted);
}

.error-card {
  border-radius: 24px;
  padding: 1.25rem;
  background: linear-gradient(180deg, #fff8f7 0%, var(--danger-soft) 100%);
  border-color: #f6d3cf;
}

.error-title {
  margin: 0;
  color: var(--danger);
  font-size: 1.15rem;
  letter-spacing: -0.02em;
}

.error-copy {
  margin: 0.45rem 0 0;
  color: var(--danger);
}

@keyframes pulse {
  0% { transform: scale(1); }
  50% { transform: scale(1.05); }
  100% { transform: scale(1); }
}

@media (max-width: 860px) {
  .app-shell {
    padding: 1rem 0.85rem 2rem;
  }

  .hero-grid {
    grid-template-columns: 1fr;
  }

  .status-stack {
    justify-items: start;
  }

  .metric-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 560px) {
  .metric-grid {
    grid-template-columns: 1fr;
  }

  .section-card,
  .hero-card,
  .error-card {
    border-radius: 20px;
    padding: 1rem;
  }
}
`