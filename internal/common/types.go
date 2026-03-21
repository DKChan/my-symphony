package common

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dministrator/symphony/internal/domain"
)

// SSEBroadcaster SSE 广播器，用于向所有连接的客户端推送实时状态更新
type SSEBroadcaster struct {
	mu          sync.RWMutex
	clients     map[chan *SSEEvent]struct{}
	lastPayload *StatePayload
}

// SSEEvent SSE 事件，包含事件类型和 JSON 数据
type SSEEvent struct {
	Event string // 事件类型，如 "state"
	Data  string // JSON 格式的数据载荷
}

// StatePayload 状态载荷，包含 orchestrator 的完整状态快照
type StatePayload struct {
	GeneratedAt string                 `json:"generated_at"` // 生成时间 (RFC3339)
	Counts      StateCounts            `json:"counts"`       // 计数统计
	Running     []RunningEntryPayload  `json:"running"`      // 正在运行的任务列表
	Retrying    []RetryEntryPayload    `json:"retrying"`     // 重试队列
	CodexTotals domain.CodexTotals    `json:"codex_totals"` // Codex token 总计
	RateLimits  any                    `json:"rate_limits"`  // 速率限制快照
}

// StateCounts 状态计数，记录运行中和重试中的任务数量
type StateCounts struct {
	Running  int `json:"running"`  // 运行中的任务数
	Retrying int `json:"retrying"` // 重试中的任务数
}

// RunningEntryPayload 运行条目载荷，描述一个正在运行的任务
type RunningEntryPayload struct {
	IssueID         string `json:"issue_id"`          // 问题 ID
	IssueIdentifier string `json:"issue_identifier"` // 问题标识符
	State           string `json:"state"`            // 当前状态
	SessionID       string `json:"session_id"`       // 会话 ID
	TurnCount       int    `json:"turn_count"`       // 轮次计数
	LastEvent       string `json:"last_event"`       // 最后事件名称
	LastMessage     string `json:"last_message"`     // 最后消息摘要
	StartedAt       string `json:"started_at"`       // 开始时间 (RFC3339)
	LastEventAt     string `json:"last_event_at"`   // 最后事件时间 (RFC3339)
	Tokens          Tokens `json:"tokens"`           // Token 统计
}

// RetryEntryPayload 重试条目载荷，描述一个等待重试的任务
type RetryEntryPayload struct {
	IssueID         string `json:"issue_id"`          // 问题 ID
	IssueIdentifier string `json:"issue_identifier"` // 问题标识符
	Attempt         int    `json:"attempt"`           // 当前尝试次数
	DueAt           string `json:"due_at"`           // 下次重试时间 (RFC3339)
	Error           string `json:"error"`            // 错误消息
}

// Tokens Token 统计，记录输入、输出和总 token 数量
type Tokens struct {
	InputTokens  int64 `json:"input_tokens"`  // 输入 token 数
	OutputTokens int64 `json:"output_tokens"` // 输出 token 数
	TotalTokens  int64 `json:"total_tokens"`  // 总 token 数
}

// NewSSEBroadcaster 创建新的 SSE 广播器实例
func NewSSEBroadcaster() *SSEBroadcaster {
	return &SSEBroadcaster{
		clients: make(map[chan *SSEEvent]struct{}),
	}
}

// Subscribe 订阅 SSE 事件流，返回用于接收事件的 channel
func (b *SSEBroadcaster) Subscribe() chan *SSEEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan *SSEEvent, 10)
	b.clients[ch] = struct{}{}
	return ch
}

// Unsubscribe 取消订阅 SSE 事件流，关闭 channel 并从广播器中移除
func (b *SSEBroadcaster) Unsubscribe(ch chan *SSEEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.clients, ch)
	close(ch)
}

// Broadcast 广播事件到所有订阅的客户端
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

// GetMu 获取互斥锁的指针（用于外部 RLock/RUnlock）
func (b *SSEBroadcaster) GetMu() *sync.RWMutex {
	return &b.mu
}

// GetLastPayload 获取最后发送的载荷
func (b *SSEBroadcaster) GetLastPayload() *StatePayload {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.lastPayload
}

// 运行时辅助函数

// TotalRuntimeSeconds 计算总运行时间（秒），包括已完成和正在运行的会话
func TotalRuntimeSeconds(state *domain.OrchestratorState, now time.Time) int {
	completed := int(state.CodexTotals.SecondsRunning)
	for _, entry := range state.Running {
		completed += int(now.Sub(entry.StartedAt).Seconds())
	}
	return completed
}

// FormatRuntimeSeconds 将秒数格式化为 "Xm Ys" 格式
func FormatRuntimeSeconds(seconds int) string {
	mins := seconds / 60
	secs := seconds % 60
	return strconv.Itoa(mins) + "m " + strconv.Itoa(secs) + "s"
}

// FormatRuntimeAndTurns 格式化运行时间和轮次
func FormatRuntimeAndTurns(startedAt time.Time, turnCount int, now time.Time) string {
	seconds := int(now.Sub(startedAt).Seconds())
	runtime := FormatRuntimeSeconds(seconds)
	if turnCount > 0 {
		return runtime + " / " + strconv.Itoa(turnCount)
	}
	return runtime
}

// StateBadgeClass 根据状态返回对应的 CSS 类名
func StateBadgeClass(state string) string {
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

// FormatInt 格式化整数，添加 K/M 后缀
func FormatInt(value int64) string {
	if value >= 1000000 {
		return strconv.FormatFloat(float64(value)/1000000, 'f', 1, 64) + "M"
	}
	if value >= 1000 {
		return strconv.FormatFloat(float64(value)/1000, 'f', 1, 64) + "K"
	}
	return strconv.FormatInt(value, 10)
}

// PrettyValue 将任意值格式化为 JSON 字符串，nil 返回 "n/a"
func PrettyValue(v any) string {
	if v == nil {
		return "n/a"
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "n/a"
	}
	return string(b)
}

// EscapeHTML HTML 转义，防止 XSS 攻击
func EscapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}
