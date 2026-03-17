// Package domain 定义核心领域模型
package domain

import "time"

// Issue 标准化的问题记录
type Issue struct {
	// ID 稳定的跟踪器内部ID
	ID string `json:"id"`
	// Identifier 人类可读的工单键（如 ABC-123）
	Identifier string `json:"identifier"`
	// Title 问题标题
	Title string `json:"title"`
	// Description 问题描述
	Description *string `json:"description,omitempty"`
	// Priority 优先级（数字越小优先级越高）
	Priority *int `json:"priority,omitempty"`
	// State 当前跟踪器状态名称
	State string `json:"state"`
	// BranchName 跟踪器提供的分支元数据
	BranchName *string `json:"branch_name,omitempty"`
	// URL 问题URL
	URL *string `json:"url,omitempty"`
	// Labels 标签列表（已标准化为小写）
	Labels []string `json:"labels,omitempty"`
	// BlockedBy 阻塞项列表
	BlockedBy []BlockerRef `json:"blocked_by,omitempty"`
	// CreatedAt 创建时间
	CreatedAt *time.Time `json:"created_at,omitempty"`
	// UpdatedAt 更新时间
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

// BlockerRef 阻塞项引用
type BlockerRef struct {
	ID         *string `json:"id,omitempty"`
	Identifier *string `json:"identifier,omitempty"`
	State      *string `json:"state,omitempty"`
}

// WorkflowDefinition 解析后的WORKFLOW.md内容
type WorkflowDefinition struct {
	// Config YAML前置配置
	Config map[string]interface{} `json:"config"`
	// PromptTemplate 提示模板（Markdown主体）
	PromptTemplate string `json:"prompt_template"`
}

// Workspace 工作空间
type Workspace struct {
	// Path 工作空间路径
	Path string `json:"path"`
	// WorkspaceKey 清理后的问题标识符
	WorkspaceKey string `json:"workspace_key"`
	// CreatedNow 是否刚刚创建
	CreatedNow bool `json:"created_now"`
}

// RunAttempt 一次执行尝试
type RunAttempt struct {
	// IssueID 问题ID
	IssueID string `json:"issue_id"`
	// IssueIdentifier 问题标识符
	IssueIdentifier string `json:"issue_identifier"`
	// Attempt 尝试次数（null表示首次运行）
	Attempt *int `json:"attempt,omitempty"`
	// WorkspacePath 工作空间路径
	WorkspacePath string `json:"workspace_path"`
	// StartedAt 开始时间
	StartedAt time.Time `json:"started_at"`
	// Status 状态
	Status RunStatus `json:"status"`
	// Error 错误信息
	Error *string `json:"error,omitempty"`
}

// RunStatus 运行状态
type RunStatus string

const (
	StatusPreparingWorkspace    RunStatus = "preparing_workspace"
	StatusBuildingPrompt        RunStatus = "building_prompt"
	StatusLaunchingAgentProcess RunStatus = "launching_agent_process"
	StatusInitializingSession   RunStatus = "initializing_session"
	StatusStreamingTurn         RunStatus = "streaming_turn"
	StatusFinishing             RunStatus = "finishing"
	StatusSucceeded             RunStatus = "succeeded"
	StatusFailed                RunStatus = "failed"
	StatusTimedOut              RunStatus = "timed_out"
	StatusStalled               RunStatus = "stalled"
	StatusCanceledByReconcile   RunStatus = "canceled_by_reconcile"
)

// LiveSession 编码代理会话元数据
type LiveSession struct {
	// SessionID 会话ID (<thread_id>-<turn_id>)
	SessionID string `json:"session_id"`
	// ThreadID 线程ID
	ThreadID string `json:"thread_id"`
	// TurnID 轮次ID
	TurnID string `json:"turn_id"`
	// CodexAppServerPID 进程ID
	CodexAppServerPID *string `json:"codex_app_server_pid,omitempty"`
	// LastCodexEvent 最后的Codex事件
	LastCodexEvent *string `json:"last_codex_event,omitempty"`
	// LastCodexTimestamp 最后的Codex时间戳
	LastCodexTimestamp *time.Time `json:"last_codex_timestamp,omitempty"`
	// LastCodexMessage 最后的Codex消息摘要
	LastCodexMessage any `json:"last_codex_message,omitempty"`
	// CodexInputTokens 输入token数
	CodexInputTokens int64 `json:"codex_input_tokens"`
	// CodexOutputTokens 输出token数
	CodexOutputTokens int64 `json:"codex_output_tokens"`
	// CodexTotalTokens 总token数
	CodexTotalTokens int64 `json:"codex_total_tokens"`
	// LastReportedInputTokens 最后报告的输入token
	LastReportedInputTokens int64 `json:"last_reported_input_tokens"`
	// LastReportedOutputTokens 最后报告的输出token
	LastReportedOutputTokens int64 `json:"last_reported_output_tokens"`
	// LastReportedTotalTokens 最后报告的总token
	LastReportedTotalTokens int64 `json:"last_reported_total_tokens"`
	// TurnCount 轮次计数
	TurnCount int `json:"turn_count"`
}

// RetryEntry 重试条目
type RetryEntry struct {
	// IssueID 问题ID
	IssueID string `json:"issue_id"`
	// Identifier 问题标识符
	Identifier string `json:"identifier"`
	// Attempt 尝试次数（从1开始）
	Attempt int `json:"attempt"`
	// DueAtMs 到期时间（单调时钟）
	DueAtMs int64 `json:"due_at_ms"`
	// Error 错误信息
	Error *string `json:"error,omitempty"`
}

// RunningEntry 运行中的条目
type RunningEntry struct {
	// Issue 问题信息
	Issue *Issue `json:"issue"`
	// Identifier 问题标识符
	Identifier string `json:"identifier"`
	// Session 会话信息
	Session *LiveSession `json:"session,omitempty"`
	// RetryAttempt 重试尝试次数
	RetryAttempt *int `json:"retry_attempt,omitempty"`
	// StartedAt 开始时间
	StartedAt time.Time `json:"started_at"`
	// TurnCount 轮次计数
	TurnCount int `json:"turn_count"`
}

// CodexTotals Codex总计统计
type CodexTotals struct {
	// InputTokens 输入token总计
	InputTokens int64 `json:"input_tokens"`
	// OutputTokens 输出token总计
	OutputTokens int64 `json:"output_tokens"`
	// TotalTokens 总token
	TotalTokens int64 `json:"total_tokens"`
	// SecondsRunning 运行秒数
	SecondsRunning float64 `json:"seconds_running"`
}

// OrchestratorState 编排器运行状态
type OrchestratorState struct {
	// PollIntervalMs 轮询间隔（毫秒）
	PollIntervalMs int64 `json:"poll_interval_ms"`
	// MaxConcurrentAgents 最大并发代理数
	MaxConcurrentAgents int `json:"max_concurrent_agents"`
	// Running 运行中的任务
	Running map[string]*RunningEntry `json:"running"`
	// Claimed 已声明的任务ID集合
	Claimed map[string]struct{} `json:"claimed"`
	// RetryAttempts 重试尝试映射
	RetryAttempts map[string]*RetryEntry `json:"retry_attempts"`
	// Completed 已完成的任务ID集合
	Completed map[string]struct{} `json:"completed"`
	// CodexTotals Codex统计
	CodexTotals *CodexTotals `json:"codex_totals"`
	// CodexRateLimits 速率限制快照
	CodexRateLimits interface{} `json:"codex_rate_limits,omitempty"`
}