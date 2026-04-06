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
	Harness       HarnessConfig       `json:"harness"`
}

// TrackerConfig 跟踪器配置
type TrackerConfig struct {
	// Kind 跟踪器类型：github、mock、beads 或 file
	Kind string `json:"kind"`
	// Endpoint API端点
	Endpoint string `json:"endpoint"`
	// APIKey API密钥
	APIKey string `json:"api_key"`
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
	// ProjectName 项目名称
	ProjectName string `json:"project_name,omitempty"`
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

// HarnessConfig Harness 配置 (P-G-E 架构)
type HarnessConfig struct {
	// MaxIterations 最大迭代次数
	MaxIterations int `json:"max_iterations"`
	// BMAD BMAD Agent 配置
	BMAD BMADConfig `json:"bmad"`
}

// BMADConfig BMAD Agent 配置
type BMADConfig struct {
	// Enabled 是否启用 BMAD Agent
	Enabled bool `json:"enabled"`
	// Agents 启用的 Agent 列表（按角色分组）
	Agents BMADAgentsConfig `json:"agents,omitempty"`
}

// BMADAgentsConfig BMAD Agent 分组配置
type BMADAgentsConfig struct {
	// Planner 规划阶段 Agent 列表
	Planner []string `json:"planner,omitempty"`
	// Generator 生成阶段 Agent 列表
	Generator []string `json:"generator,omitempty"`
	// Evaluator 评估阶段 Agent 列表
	Evaluator []string `json:"evaluator,omitempty"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Tracker: TrackerConfig{
			Kind:           "mock",
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
		Harness: HarnessConfig{
			MaxIterations: 5,
			BMAD: BMADConfig{
				Enabled: true,
				Agents: BMADAgentsConfig{
					Planner:   []string{"bmad-agent-pm", "bmad-agent-qa", "bmad-agent-architect"},
					Generator: []string{"bmad-agent-qa", "bmad-agent-dev"},
					Evaluator: []string{"bmad-code-review", "bmad-editorial-review-prose"},
				},
			},
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
		if projectName, ok := workspace["project_name"].(string); ok {
			cfg.Workspace.ProjectName = projectName
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

	// 解析 harness 配置
	if harness, ok := raw["harness"].(map[string]interface{}); ok {
		if maxIterations, ok := parseInt(harness["max_iterations"]); ok && maxIterations > 0 {
			cfg.Harness.MaxIterations = int(maxIterations)
		}
		if bmad, ok := harness["bmad"].(map[string]interface{}); ok {
			if enabled, ok := bmad["enabled"].(bool); ok {
				cfg.Harness.BMAD.Enabled = enabled
			}
			// 解析 agents 配置（支持分组结构）
			if agents, ok := bmad["agents"].(map[string]interface{}); ok {
				if planner := parseStringList(agents["planner"]); len(planner) > 0 {
					cfg.Harness.BMAD.Agents.Planner = planner
				}
				if generator := parseStringList(agents["generator"]); len(generator) > 0 {
					cfg.Harness.BMAD.Agents.Generator = generator
				}
				if evaluator := parseStringList(agents["evaluator"]); len(evaluator) > 0 {
					cfg.Harness.BMAD.Agents.Evaluator = evaluator
				}
			}
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
	supportedTrackers := map[string]bool{"github": true, "mock": true, "beads": true}
	if c.Tracker.Kind == "" {
		errors = append(errors, "tracker.kind is required")
	} else if !supportedTrackers[c.Tracker.Kind] {
		errors = append(errors, fmt.Sprintf("unsupported tracker.kind: %s (supported: github, mock, beads)", c.Tracker.Kind))
	}

	// mock 和 beads 类型不需要 api_key，跳过验证
	if c.Tracker.Kind == "mock" || c.Tracker.Kind == "beads" {
		return &Validation{
			Valid:  len(errors) == 0,
			Errors: errors,
		}
	}

	// 验证 tracker.api_key（mock 和 beads 不需要）
	if c.Tracker.Kind != "mock" && c.Tracker.Kind != "beads" && c.Tracker.APIKey == "" {
		switch c.Tracker.Kind {
		case "github":
			errors = append(errors, "tracker.api_key is required (set GITHUB_TOKEN env var or api_key in config)")
		default:
			errors = append(errors, "tracker.api_key is required")
		}
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

	// 验证 harness.max_iterations
	if c.Harness.MaxIterations <= 0 {
		errors = append(errors, "harness.max_iterations must be positive")
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
	supportedTrackers := map[string]bool{"github": true, "mock": true, "beads": true}
	if c.Tracker.Kind == "" {
		errors = append(errors, "tracker.kind is required")
	} else if !supportedTrackers[c.Tracker.Kind] {
		errors = append(errors, fmt.Sprintf("unsupported tracker.kind: %s (supported: github, mock, beads)", c.Tracker.Kind))
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

	// 验证 harness.max_iterations
	if c.Harness.MaxIterations <= 0 {
		errors = append(errors, "harness.max_iterations must be positive")
	}

	return &Validation{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// ConfigError 配置错误（统一错误码格式）
type ConfigError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// 预定义错误
var (
	ErrConfigInvalid         = &ConfigError{Code: "config.invalid", Message: "配置无效"}
	ErrBMADAgentUnavailable  = &ConfigError{Code: "config.bmad.unavailable", Message: "BMAD Agent 不可用"}
	ErrBeadsCLIUnavailable   = &ConfigError{Code: "tracker.unavailable", Message: "Beads CLI 不可用"}
	ErrConfigFileNotFound    = &ConfigError{Code: "config.file.not_found", Message: "配置文件不存在"}
	ErrConfigYAMLInvalid     = &ConfigError{Code: "config.yaml.invalid", Message: "YAML 格式错误"}
)

// NewConfigError 创建新的配置错误
func NewConfigError(code, message string) *ConfigError {
	return &ConfigError{Code: code, Message: message}
}

// CheckBMADAgentAvailability 检查单个 BMAD Agent 可用性
func CheckBMADAgentAvailability(agentName string) error {
	// BMAD agents 通过 claude code CLI 调用，检查 claude 是否可用
	if _, err := exec.LookPath("claude"); err != nil {
		return NewConfigError("config.bmad.unavailable",
			fmt.Sprintf("BMAD Agent '%s' 需要 claude CLI，但未找到", agentName))
	}
	return nil
}

// CheckBMADAgentsAvailability 批量检查 BMAD Agent 可用性
func CheckBMADAgentsAvailability(agents []string) error {
	for _, agent := range agents {
		if err := CheckBMADAgentAvailability(agent); err != nil {
			return err
		}
	}
	return nil
}

// CheckBeadsCLIAvailability 检查 Beads CLI 可用性
func CheckBeadsCLIAvailability(beadsPath string) error {
	if beadsPath == "" {
		beadsPath = "beads"
	}
	if _, err := exec.LookPath(beadsPath); err != nil {
		return ErrBeadsCLIUnavailable
	}
	return nil
}

// ValidateStartupConfig 启动时完整配置验证
// 验证配置格式、BMAD Agent 可用性、Beads CLI 可用性
func ValidateStartupConfig(configPath string, cfg *Config) error {
	// 1. 验证配置文件存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return ErrConfigFileNotFound
	}

	// 2. 验证配置格式
	dispatchValidation := cfg.ValidateDispatchConfig()
	if !dispatchValidation.Valid {
		return NewConfigError("config.invalid",
			fmt.Sprintf("配置验证失败: %s", strings.Join(dispatchValidation.Errors, "; ")))
	}

	symphonyValidation := cfg.ValidateSymphonyConfig()
	if !symphonyValidation.Valid {
		return NewConfigError("config.invalid",
			fmt.Sprintf("Symphony 配置验证失败: %s", strings.Join(symphonyValidation.Errors, "; ")))
	}

	// 3. 验证 BMAD Agent 可用性（如果启用）
	if cfg.Harness.BMAD.Enabled {
		allAgents := make([]string, 0)
		allAgents = append(allAgents, cfg.Harness.BMAD.Agents.Planner...)
		allAgents = append(allAgents, cfg.Harness.BMAD.Agents.Generator...)
		allAgents = append(allAgents, cfg.Harness.BMAD.Agents.Evaluator...)

		if len(allAgents) > 0 {
			if err := CheckBMADAgentsAvailability(allAgents); err != nil {
				return err
			}
		}
	}

	// 4. 验证 Beads CLI 可用性（如果使用 beads tracker）
	if cfg.Tracker.Kind == "beads" {
		if err := CheckBeadsCLIAvailability(""); err != nil {
			return err
		}
	}

	return nil
}
