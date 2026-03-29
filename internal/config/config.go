// Package config 提供配置解析和管理功能
package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Config 服务配置（类型化视图）
type Config struct {
	Tracker       TrackerConfig       `json:"tracker"`
	Polling       PollingConfig       `json:"polling"`
	Workspace     WorkspaceConfig     `json:"workspace"`
	Hooks         HooksConfig         `json:"hooks"`
	Agent         AgentConfig         `json:"agent"`
	Claude        *ClaudeConfig       `json:"claude,omitempty"`
	OpenCode      *OpenCodeConfig     `json:"opencode,omitempty"`
	Codex         CodexConfig         `json:"codex"`
	Server        *ServerConfig       `json:"server,omitempty"`
	Clarification ClarificationConfig `json:"clarification"`
	Execution     ExecutionConfig     `json:"execution"`
	Logging       LoggingConfig       `json:"logging"`
}

// TrackerConfig 跟踪器配置
type TrackerConfig struct {
	// Kind 跟踪器类型：linear、github 或 mock
	Kind string `json:"kind"`
	// Endpoint API端点
	Endpoint string `json:"endpoint"`
	// APIKey API密钥
	APIKey string `json:"api_key"`
	// ProjectSlug 项目标识（Linear专用）
	ProjectSlug string `json:"project_slug"`
	// Repo 仓库（GitHub专用，格式：owner/repo）
	Repo string `json:"repo,omitempty"`
	// ActiveStates 活跃状态列表
	ActiveStates []string `json:"active_states"`
	// TerminalStates 终态列表
	TerminalStates []string `json:"terminal_states"`
	// MockIssues Mock问题列表（mock tracker专用）
	MockIssues []MockIssueConfig `json:"mock_issues,omitempty"`
}

// MockIssueConfig Mock问题配置
type MockIssueConfig struct {
	ID          string   `json:"id"`
	Identifier  string   `json:"identifier"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	State       string   `json:"state"`
	Priority    int      `json:"priority,omitempty"`
	Labels      []string `json:"labels,omitempty"`
}

// PollingConfig 轮询配置
type PollingConfig struct {
	// IntervalMs 轮询间隔（毫秒）
	IntervalMs int64 `json:"interval_ms"`
}

// WorkspaceConfig 工作空间配置
type WorkspaceConfig struct {
	// Root 工作空间根目录
	Root string `json:"root"`
}

// HooksConfig 钩子配置
type HooksConfig struct {
	// AfterCreate 创建后钩子
	AfterCreate *string `json:"after_create,omitempty"`
	// BeforeRun 运行前钩子
	BeforeRun *string `json:"before_run,omitempty"`
	// AfterRun 运行后钩子
	AfterRun *string `json:"after_run,omitempty"`
	// BeforeRemove 删除前钩子
	BeforeRemove *string `json:"before_remove,omitempty"`
	// TimeoutMs 钩子超时（毫秒）
	TimeoutMs int64 `json:"timeout_ms"`
}

// AgentConfig 代理配置
type AgentConfig struct {
	// Kind 代理类型：codex（默认）、claude、opencode
	Kind string `json:"kind"`
	// MaxConcurrentAgents 最大并发代理数
	MaxConcurrentAgents int `json:"max_concurrent_agents"`
	// MaxTurns 最大轮次
	MaxTurns int `json:"max_turns"`
	// MaxRetryBackoffMs 最大重试退避时间（毫秒）
	MaxRetryBackoffMs int64 `json:"max_retry_backoff_ms"`
	// MaxConcurrentAgentsByState 按状态的并发限制
	MaxConcurrentAgentsByState map[string]int `json:"max_concurrent_agents_by_state,omitempty"`
	// Command 代理命令（覆盖默认命令）
	Command string `json:"command,omitempty"`
	// TurnTimeoutMs 轮次超时（毫秒，用于非 codex agent）
	TurnTimeoutMs int64 `json:"turn_timeout_ms,omitempty"`
}

// CodexConfig Codex配置
type CodexConfig struct {
	// Command Codex命令
	Command string `json:"command"`
	// ApprovalPolicy 审批策略
	ApprovalPolicy string `json:"approval_policy"`
	// ThreadSandbox 线程沙箱模式
	ThreadSandbox string `json:"thread_sandbox"`
	// TurnSandboxPolicy 轮次沙箱策略
	TurnSandboxPolicy string `json:"turn_sandbox_policy"`
	// TurnTimeoutMs 轮次超时（毫秒）
	TurnTimeoutMs int64 `json:"turn_timeout_ms"`
	// ReadTimeoutMs 读取超时（毫秒）
	ReadTimeoutMs int64 `json:"read_timeout_ms"`
	// StallTimeoutMs 停滞超时（毫秒）
	StallTimeoutMs int64 `json:"stall_timeout_ms"`
}

// ClaudeConfig Claude Code CLI配置（当 agent.kind: "claude" 时使用）
type ClaudeConfig struct {
	// Command CLI命令（默认: claude）
	Command string `json:"command,omitempty"`
	// ExtraArgs 额外命令行参数（会追加到默认参数之后）
	// 示例: ["--model", "opus-4", "--max-tokens", "4096"]
	ExtraArgs []string `json:"extra_args,omitempty"`
	// SkipPermissions 跳过权限检查（默认: true）
	SkipPermissions bool `json:"skip_permissions,omitempty"`
}

// OpenCodeConfig OpenCode CLI配置（当 agent.kind: "opencode" 时使用）
type OpenCodeConfig struct {
	// Command CLI命令（默认: opencode）
	Command string `json:"command,omitempty"`
	// ExtraArgs 额外命令行参数（会追加到默认参数之后）
	// 示例: ["--model", "gpt-4", "--provider", "openai"]
	ExtraArgs []string `json:"extra_args,omitempty"`
}

// ServerConfig HTTP服务器配置
type ServerConfig struct {
	// Port 端口号
	Port int `json:"port"`
}

// ClarificationConfig 澄清配置
type ClarificationConfig struct {
	// MaxRounds 最大澄清轮次
	MaxRounds int `json:"max_rounds"`
}

// ExecutionConfig 执行配置
type ExecutionConfig struct {
	// MaxRetries 最大重试次数
	MaxRetries int `json:"max_retries"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	// Level 日志级别: debug, info, warn, error
	Level string `json:"level"`
	// Format 输出格式: json, text
	Format string `json:"format"`
	// FilePath 输出文件路径（可选）
	FilePath string `json:"file_path,omitempty"`
	// EnableStdout 是否输出到标准输出
	EnableStdout bool `json:"enable_stdout"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Tracker: TrackerConfig{
			Kind:           "linear",
			Endpoint:       "https://api.linear.app/graphql",
			ActiveStates:   []string{"Todo", "In Progress"},
			TerminalStates: []string{"Closed", "Cancelled", "Canceled", "Duplicate", "Done"},
		},
		Polling: PollingConfig{
			IntervalMs: 30000,
		},
		Workspace: WorkspaceConfig{
			Root: filepath.Join(os.TempDir(), "symphony_workspaces"),
		},
		Hooks: HooksConfig{
			TimeoutMs: 60000,
		},
		Agent: AgentConfig{
			Kind:                       "codex",
			MaxConcurrentAgents:        10,
			MaxTurns:                   20,
			MaxRetryBackoffMs:          300000,
			MaxConcurrentAgentsByState: make(map[string]int),
		},
		Codex: CodexConfig{
			Command:        "codex app-server",
			TurnTimeoutMs:  3600000,
			ReadTimeoutMs:  5000,
			StallTimeoutMs: 300000,
		},
		Clarification: ClarificationConfig{
			MaxRounds: 5,
		},
		Execution: ExecutionConfig{
			MaxRetries: 3,
		},
		Logging: LoggingConfig{
			Level:        "info",
			Format:       "json",
			EnableStdout: true,
		},
	}
}

// ParseConfig 从原始配置映射解析类型化配置
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	cfg := DefaultConfig()

	// 解析 tracker 配置
	if tracker, ok := raw["tracker"].(map[string]interface{}); ok {
		if kind, ok := tracker["kind"].(string); ok {
			cfg.Tracker.Kind = kind
		}
		if endpoint, ok := tracker["endpoint"].(string); ok {
			cfg.Tracker.Endpoint = endpoint
		}
		if apiKey, ok := tracker["api_key"].(string); ok {
			cfg.Tracker.APIKey = resolveEnvVar(apiKey)
		}
		if projectSlug, ok := tracker["project_slug"].(string); ok {
			cfg.Tracker.ProjectSlug = projectSlug
		}
		if repo, ok := tracker["repo"].(string); ok {
			cfg.Tracker.Repo = repo
		}
		if activeStates := parseStringList(tracker["active_states"]); len(activeStates) > 0 {
			cfg.Tracker.ActiveStates = activeStates
		}
		if terminalStates := parseStringList(tracker["terminal_states"]); len(terminalStates) > 0 {
			cfg.Tracker.TerminalStates = terminalStates
		}
		// 解析 mock_issues
		if mockIssues, ok := tracker["mock_issues"].([]interface{}); ok {
			for _, item := range mockIssues {
				if mi, ok := item.(map[string]interface{}); ok {
					mockIssue := MockIssueConfig{}
					if id, ok := mi["id"].(string); ok {
						mockIssue.ID = id
					}
					if identifier, ok := mi["identifier"].(string); ok {
						mockIssue.Identifier = identifier
					}
					if title, ok := mi["title"].(string); ok {
						mockIssue.Title = title
					}
					if desc, ok := mi["description"].(string); ok {
						mockIssue.Description = desc
					}
					if state, ok := mi["state"].(string); ok {
						mockIssue.State = state
					}
					if priority, ok := parseInt(mi["priority"]); ok && priority > 0 {
						mockIssue.Priority = int(priority)
					}
					if labels := parseStringList(mi["labels"]); len(labels) > 0 {
						mockIssue.Labels = labels
					}
					cfg.Tracker.MockIssues = append(cfg.Tracker.MockIssues, mockIssue)
				}
			}
		}
	}

	// 解析 polling 配置
	if polling, ok := raw["polling"].(map[string]interface{}); ok {
		if intervalMs, ok := parseInt(polling["interval_ms"]); ok {
			cfg.Polling.IntervalMs = intervalMs
		}
	}

	// 解析 workspace 配置
	if workspace, ok := raw["workspace"].(map[string]interface{}); ok {
		if root, ok := workspace["root"].(string); ok {
			cfg.Workspace.Root = expandPath(resolveEnvVar(root))
		}
	}

	// 解析 hooks 配置
	if hooks, ok := raw["hooks"].(map[string]interface{}); ok {
		if afterCreate, ok := hooks["after_create"].(string); ok {
			cfg.Hooks.AfterCreate = &afterCreate
		}
		if beforeRun, ok := hooks["before_run"].(string); ok {
			cfg.Hooks.BeforeRun = &beforeRun
		}
		if afterRun, ok := hooks["after_run"].(string); ok {
			cfg.Hooks.AfterRun = &afterRun
		}
		if beforeRemove, ok := hooks["before_remove"].(string); ok {
			cfg.Hooks.BeforeRemove = &beforeRemove
		}
		if timeoutMs, ok := parseInt(hooks["timeout_ms"]); ok && timeoutMs > 0 {
			cfg.Hooks.TimeoutMs = timeoutMs
		}
	}

	// 解析 agent 配置
	if agent, ok := raw["agent"].(map[string]interface{}); ok {
		if kind, ok := agent["kind"].(string); ok && kind != "" {
			cfg.Agent.Kind = kind
		}
		if command, ok := agent["command"].(string); ok && command != "" {
			cfg.Agent.Command = command
		}
		if maxConcurrent, ok := parseInt(agent["max_concurrent_agents"]); ok && maxConcurrent > 0 {
			cfg.Agent.MaxConcurrentAgents = int(maxConcurrent)
		}
		if maxTurns, ok := parseInt(agent["max_turns"]); ok && maxTurns > 0 {
			cfg.Agent.MaxTurns = int(maxTurns)
		}
		if maxRetryBackoff, ok := parseInt(agent["max_retry_backoff_ms"]); ok && maxRetryBackoff > 0 {
			cfg.Agent.MaxRetryBackoffMs = maxRetryBackoff
		}
		if turnTimeout, ok := parseInt(agent["turn_timeout_ms"]); ok {
				// 0 或负数表示无超时限制，允许设置任何值
				cfg.Agent.TurnTimeoutMs = turnTimeout
			}
		if byState, ok := agent["max_concurrent_agents_by_state"].(map[string]interface{}); ok {
			for state, val := range byState {
				if limit, ok := parseInt(val); ok && limit > 0 {
					cfg.Agent.MaxConcurrentAgentsByState[strings.ToLower(strings.TrimSpace(state))] = int(limit)
				}
			}
		}
	}

	// 解析 claude 配置
	if claude, ok := raw["claude"].(map[string]interface{}); ok {
		cfg.Claude = &ClaudeConfig{}
		if command, ok := claude["command"].(string); ok {
			cfg.Claude.Command = command
		}
		if skipPerms, ok := claude["skip_permissions"].(bool); ok {
			cfg.Claude.SkipPermissions = skipPerms
		} else {
			// 默认跳过权限检查
			cfg.Claude.SkipPermissions = true
		}
		if extraArgs := parseStringList(claude["extra_args"]); len(extraArgs) > 0 {
			cfg.Claude.ExtraArgs = extraArgs
		}
	}

	// 解析 opencode 配置
	if opencode, ok := raw["opencode"].(map[string]interface{}); ok {
		cfg.OpenCode = &OpenCodeConfig{}
		if command, ok := opencode["command"].(string); ok {
			cfg.OpenCode.Command = command
		}
		if extraArgs := parseStringList(opencode["extra_args"]); len(extraArgs) > 0 {
			cfg.OpenCode.ExtraArgs = extraArgs
		}
	}

	// 解析 codex 配置
	if codex, ok := raw["codex"].(map[string]interface{}); ok {
		if command, ok := codex["command"].(string); ok {
			cfg.Codex.Command = command
		}
		if approvalPolicy, ok := codex["approval_policy"].(string); ok {
			cfg.Codex.ApprovalPolicy = approvalPolicy
		}
		if threadSandbox, ok := codex["thread_sandbox"].(string); ok {
			cfg.Codex.ThreadSandbox = threadSandbox
		}
		if turnSandboxPolicy, ok := codex["turn_sandbox_policy"].(string); ok {
			cfg.Codex.TurnSandboxPolicy = turnSandboxPolicy
		}
		if turnTimeout, ok := parseInt(codex["turn_timeout_ms"]); ok {
				// 0 或负数表示无超时限制，允许设置任何值
				cfg.Codex.TurnTimeoutMs = turnTimeout
			}
		if readTimeout, ok := parseInt(codex["read_timeout_ms"]); ok && readTimeout > 0 {
			cfg.Codex.ReadTimeoutMs = readTimeout
		}
		if stallTimeout, ok := parseInt(codex["stall_timeout_ms"]); ok {
			cfg.Codex.StallTimeoutMs = stallTimeout
		}
	}

	// 解析 server 配置（扩展）
	if server, ok := raw["server"].(map[string]interface{}); ok {
		if port, ok := parseInt(server["port"]); ok {
			cfg.Server = &ServerConfig{Port: int(port)}
		}
	}

	// 解析 clarification 配置
	if clarification, ok := raw["clarification"].(map[string]interface{}); ok {
		if maxRounds, ok := parseInt(clarification["max_rounds"]); ok && maxRounds > 0 {
			cfg.Clarification.MaxRounds = int(maxRounds)
		}
	}

	// 解析 execution 配置
	if execution, ok := raw["execution"].(map[string]interface{}); ok {
		if maxRetries, ok := parseInt(execution["max_retries"]); ok && maxRetries >= 0 {
			cfg.Execution.MaxRetries = int(maxRetries)
		}
	}

	// 解析 logging 配置
	if logging, ok := raw["logging"].(map[string]interface{}); ok {
		if level, ok := logging["level"].(string); ok {
			cfg.Logging.Level = level
		}
		if format, ok := logging["format"].(string); ok {
			cfg.Logging.Format = format
		}
		if filePath, ok := logging["file_path"].(string); ok {
			cfg.Logging.FilePath = filePath
		}
		if enableStdout, ok := logging["enable_stdout"].(bool); ok {
			cfg.Logging.EnableStdout = enableStdout
		}
	}

	return cfg, nil
}

// resolveEnvVar 解析环境变量引用（$VAR_NAME 格式）
func resolveEnvVar(s string) string {
	if strings.HasPrefix(s, "$") {
		varName := s[1:]
		return os.Getenv(varName)
	}
	return s
}

// expandPath 展开路径（~ 和环境变量）
func expandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[1:])
		}
	}
	return path
}

// parseStringList 解析字符串列表（支持数组和逗号分隔字符串）
func parseStringList(v interface{}) []string {
	switch val := v.(type) {
	case []interface{}:
		result := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case string:
		parts := strings.Split(val, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}
	return nil
}

// parseInt 解析整数（支持int和string类型）
func parseInt(v interface{}) (int64, bool) {
	switch val := v.(type) {
	case int:
		return int64(val), true
	case int64:
		return val, true
	case float64:
		return int64(val), true
	case string:
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			return i, true
		}
	}
	return 0, false
}

// Validation 验证结果
type Validation struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

// ValidateDispatchConfig 验证调度配置
func (c *Config) ValidateDispatchConfig() *Validation {
	var errors []string

	// 验证 tracker.kind
	supportedTrackers := map[string]bool{"linear": true, "github": true, "mock": true, "beads": true}
	if c.Tracker.Kind == "" {
		errors = append(errors, "tracker.kind is required")
	} else if !supportedTrackers[c.Tracker.Kind] {
		errors = append(errors, fmt.Sprintf("unsupported tracker.kind: %s (supported: linear, github, mock, beads)", c.Tracker.Kind))
	}

	// mock 和 beads 类型不需要 api_key，跳过验证
	if c.Tracker.Kind == "mock" || c.Tracker.Kind == "beads" {
		return &Validation{
			Valid:  len(errors) == 0,
			Errors: errors,
		}
	}

	// 验证 tracker.api_key
	if c.Tracker.APIKey == "" {
		switch c.Tracker.Kind {
		case "linear":
			errors = append(errors, "tracker.api_key is required (set LINEAR_API_KEY env var or api_key in config)")
		case "github":
			errors = append(errors, "tracker.api_key is required (set GITHUB_TOKEN env var or api_key in config)")
		default:
			errors = append(errors, "tracker.api_key is required")
		}
	}

	// 验证 tracker.project_slug（Linear必需）
	if c.Tracker.Kind == "linear" && c.Tracker.ProjectSlug == "" {
		errors = append(errors, "tracker.project_slug is required for linear tracker")
	}

	// 验证 tracker.repo（GitHub必需）
	if c.Tracker.Kind == "github" && c.Tracker.Repo == "" {
		errors = append(errors, "tracker.repo is required for github tracker (format: owner/repo)")
	}

	// 验证 agent.kind
	supportedAgents := map[string]bool{"codex": true, "claude": true, "opencode": true}
	agentKind := c.Agent.Kind
	if agentKind == "" {
		agentKind = "codex"
	}
	if !supportedAgents[agentKind] {
		errors = append(errors, fmt.Sprintf("unsupported agent.kind: %s (supported: codex, claude, opencode)", agentKind))
	}

	// codex agent 需要 codex.command
	if agentKind == "codex" && c.Codex.Command == "" {
		errors = append(errors, "codex.command is required for codex agent")
	}

	return &Validation{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// IsActiveState 检查状态是否为活跃状态
func (c *Config) IsActiveState(state string) bool {
	normalized := strings.ToLower(strings.TrimSpace(state))
	for _, s := range c.Tracker.ActiveStates {
		if strings.ToLower(strings.TrimSpace(s)) == normalized {
			return true
		}
	}
	return false
}

// IsTerminalState 检查状态是否为终态
func (c *Config) IsTerminalState(state string) bool {
	normalized := strings.ToLower(strings.TrimSpace(state))
	for _, s := range c.Tracker.TerminalStates {
		if strings.ToLower(strings.TrimSpace(s)) == normalized {
			return true
		}
	}
	return false
}

// SanitizeWorkspaceKey 清理工作空间键名（只保留 [A-Za-z0-9._-]）
var workspaceKeyRe = regexp.MustCompile(`[^A-Za-z0-9._-]`)

func SanitizeWorkspaceKey(identifier string) string {
	return workspaceKeyRe.ReplaceAllString(identifier, "_")
}

// ValidateSymphonyConfig 验证 Symphony 特定配置
// 验证 tracker 配置、AI Agent CLI 路径、prompt 文件等
func (c *Config) ValidateSymphonyConfig() *Validation {
	var errors []string

	// 验证 tracker 配置有效性
	supportedTrackers := map[string]bool{"linear": true, "github": true, "mock": true, "beads": true}
	if c.Tracker.Kind == "" {
		errors = append(errors, "tracker.kind is required")
	} else if !supportedTrackers[c.Tracker.Kind] {
		errors = append(errors, fmt.Sprintf("unsupported tracker.kind: %s (supported: linear, github, mock, beads)", c.Tracker.Kind))
	}

	// 验证 tracker API 配置
	// mock 和 beads 类型不需要 API 配置
	if c.Tracker.Kind != "mock" && c.Tracker.Kind != "beads" {
		if c.Tracker.APIKey == "" {
			switch c.Tracker.Kind {
			case "linear":
				errors = append(errors, "tracker.api_key is required for linear tracker")
			case "github":
				errors = append(errors, "tracker.api_key is required for github tracker")
			}
		}
	}

	// 验证 Linear 特定配置
	if c.Tracker.Kind == "linear" && c.Tracker.ProjectSlug == "" {
		errors = append(errors, "tracker.project_slug is required for linear tracker")
	}

	// 验证 GitHub 特定配置
	if c.Tracker.Kind == "github" && c.Tracker.Repo == "" {
		errors = append(errors, "tracker.repo is required for github tracker (format: owner/repo)")
	}

	// 验证 agent 配置
	supportedAgents := map[string]bool{"codex": true, "claude": true, "opencode": true}
	agentKind := c.Agent.Kind
	if agentKind == "" {
		agentKind = "codex"
	}
	if !supportedAgents[agentKind] {
		errors = append(errors, fmt.Sprintf("unsupported agent.kind: %s (supported: codex, claude, opencode)", agentKind))
	}

	// 验证 agent CLI 命令是否存在
	var agentCmd string
	switch agentKind {
	case "codex":
		agentCmd = c.Codex.Command
		if agentCmd == "" {
			agentCmd = "codex"
		}
	case "claude":
		if c.Claude != nil && c.Claude.Command != "" {
			agentCmd = c.Claude.Command
		} else {
			agentCmd = "claude"
		}
	case "opencode":
		if c.OpenCode != nil && c.OpenCode.Command != "" {
			agentCmd = c.OpenCode.Command
		} else {
			agentCmd = "opencode"
		}
	}

	// 检查 agent CLI 是否在 PATH 中
	if _, err := exec.LookPath(agentCmd); err != nil {
		// 对于带参数的命令，只取第一个部分
		cmdParts := strings.Fields(agentCmd)
		if len(cmdParts) > 0 {
			if _, err := exec.LookPath(cmdParts[0]); err != nil {
				errors = append(errors, fmt.Sprintf("agent CLI not found: %s", agentCmd))
			}
		} else {
			errors = append(errors, fmt.Sprintf("agent CLI not found: %s", agentCmd))
		}
	}

	// 验证工作空间根目录
	if c.Workspace.Root == "" {
		errors = append(errors, "workspace.root is required")
	}

	// 验证 clarification.max_rounds
	if c.Clarification.MaxRounds <= 0 {
		errors = append(errors, "clarification.max_rounds must be positive")
	}

	// 验证 execution.max_retries
	if c.Execution.MaxRetries < 0 {
		errors = append(errors, "execution.max_retries must be non-negative")
	}

	return &Validation{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}
