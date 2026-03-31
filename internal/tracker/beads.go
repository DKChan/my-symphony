// Package tracker 提供问题跟踪器客户端实现
package tracker

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/dministrator/symphony/internal/domain"
	"github.com/dministrator/symphony/internal/logging"
)

const (
	// beadsDefaultTimeout Beads CLI 默认超时时间
	beadsDefaultTimeout = 30 * time.Second
	// beadsCLIName Beads CLI 命令名称
	beadsCLIName = "beads"
)

// BeadsClient Beads CLI 跟踪器客户端
type BeadsClient struct {
	cliPath string
	timeout time.Duration
}

// NewBeadsClient 创建新的 Beads 客户端
func NewBeadsClient() *BeadsClient {
	return &BeadsClient{
		cliPath: beadsCLIName,
		timeout: beadsDefaultTimeout,
	}
}

// NewBeadsClientWithPath 创建带有指定 CLI 路径的 Beads 客户端
func NewBeadsClientWithPath(cliPath string) *BeadsClient {
	return &BeadsClient{
		cliPath: cliPath,
		timeout: beadsDefaultTimeout,
	}
}

// SetTimeout 设置超时时间
func (c *BeadsClient) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}

// CheckAvailability 检查 Beads CLI 是否可用
func (c *BeadsClient) CheckAvailability() error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.cliPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tracker.unavailable: Beads CLI 不可用: %w", err)
	}

	// 检查输出是否包含版本信息
	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		return fmt.Errorf("tracker.unavailable: Beads CLI 不可用: 无版本输出")
	}

	return nil
}

// FetchCandidateIssues 获取候选问题（返回活跃状态的问题）
func (c *BeadsClient) FetchCandidateIssues(ctx context.Context, activeStates []string) ([]*domain.Issue, error) {
	return c.ListTasksByState(ctx, activeStates)
}

// FetchIssuesByStates 获取指定状态的问题
func (c *BeadsClient) FetchIssuesByStates(ctx context.Context, states []string) ([]*domain.Issue, error) {
	return c.ListTasksByState(ctx, states)
}

// FetchIssueStatesByIDs 按 ID 获取问题状态
func (c *BeadsClient) FetchIssueStatesByIDs(ctx context.Context, ids []string) ([]*domain.Issue, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var issues []*domain.Issue
	for _, id := range ids {
		issue, err := c.getTask(ctx, id)
		if err != nil {
			// 单个失败不阻断其他，记录跳过
			logging.Warn("failed to fetch issue state",
				"task_id", id,
				"error", err.Error(),
			)
			continue
		}
		issues = append(issues, issue)
	}

	return issues, nil
}

// CreateTask 创建新任务
func (c *BeadsClient) CreateTask(ctx context.Context, title, description string) (*domain.Issue, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	args := []string{"issue", "create", "--title", title, "--description", description}
	output, err := c.runCommand(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("create task failed: %w", err)
	}

	// 解析创建结果，获取任务 ID
	var result beadsIssue
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("parse create result: %w", err)
	}

	return result.toDomain(), nil
}

// UpdateStage 更新任务阶段状态
func (c *BeadsClient) UpdateStage(ctx context.Context, identifier string, stage domain.StageState) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// 将阶段状态序列化为 JSON
	stageJSON, err := json.Marshal(stage)
	if err != nil {
		return fmt.Errorf("marshal stage state: %w", err)
	}

	args := []string{"issue", "update", identifier, "--stage", string(stageJSON)}
	_, err = c.runCommand(ctx, args...)
	if err != nil {
		return fmt.Errorf("update stage failed: %w", err)
	}

	return nil
}

// GetStageState 获取任务的阶段状态（用于崩溃恢复）
func (c *BeadsClient) GetStageState(ctx context.Context, identifier string) (*domain.StageState, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// 获取任务详情，包含 Custom 字段
	args := []string{"issue", "show", identifier, "--include-custom"}
	output, err := c.runCommand(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("get task with custom failed: %w", err)
	}

	var result beadsIssue
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("parse task: %w", err)
	}

	// 如果没有 Custom 字段，返回 nil
	if len(result.Custom) == 0 {
		return nil, nil
	}

	// 尝试从 Custom 字段解析 StageState
	var customData map[string]json.RawMessage
	if err := json.Unmarshal(result.Custom, &customData); err != nil {
		// Custom 字段可能直接是 StageState
		var stage domain.StageState
		if err := json.Unmarshal(result.Custom, &stage); err != nil {
			return nil, nil // 无法解析，返回 nil
		}
		return &stage, nil
	}

	// 从 customData 中查找 stage_state 字段
	if stageRaw, ok := customData["stage_state"]; ok {
		var stage domain.StageState
		if err := json.Unmarshal(stageRaw, &stage); err != nil {
			return nil, nil
		}
		return &stage, nil
	}

	return nil, nil
}

// CreateSubTask 创建子任务（带依赖关系）
func (c *BeadsClient) CreateSubTask(ctx context.Context, parentIdentifier string, title, description string, blockedBy []string) (*domain.Issue, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	args := []string{"issue", "create", "--title", title, "--description", description}
	if parentIdentifier != "" {
		args = append(args, "--parent", parentIdentifier)
	}
	for _, blocker := range blockedBy {
		args = append(args, "--blocked-by", blocker)
	}

	output, err := c.runCommand(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("create subtask failed: %w", err)
	}

	var result beadsIssue
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("parse create result: %w", err)
	}

	return result.toDomain(), nil
}

// AppendConversation 追加对话记录
func (c *BeadsClient) AppendConversation(ctx context.Context, identifier string, turn domain.ConversationTurn) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// 将对话记录序列化为 JSON
	turnJSON, err := json.Marshal(turn)
	if err != nil {
		return fmt.Errorf("marshal conversation turn: %w", err)
	}

	args := []string{"issue", "comment", identifier, "--body", string(turnJSON)}
	_, err = c.runCommand(ctx, args...)
	if err != nil {
		return fmt.Errorf("append conversation failed: %w", err)
	}

	return nil
}

// GetConversationHistory 获取对话历史记录
func (c *BeadsClient) GetConversationHistory(ctx context.Context, identifier string) ([]domain.ConversationTurn, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	args := []string{"issue", "comments", identifier}
	output, err := c.runCommand(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("get conversation history failed: %w", err)
	}

	return c.parseConversationHistory(output)
}

// ListTasksByState 按状态获取任务列表
func (c *BeadsClient) ListTasksByState(ctx context.Context, states []string) ([]*domain.Issue, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var allIssues []*domain.Issue
	for _, state := range states {
		args := []string{"issue", "list", "--state", state}
		output, err := c.runCommand(ctx, args...)
		if err != nil {
			return nil, fmt.Errorf("list tasks failed for state %s: %w", state, err)
		}

		issues, err := c.parseIssueList(output)
		if err != nil {
			return nil, fmt.Errorf("parse issue list: %w", err)
		}

		allIssues = append(allIssues, issues...)
	}

	return allIssues, nil
}

// GetTask 获取单个任务详情
func (c *BeadsClient) GetTask(ctx context.Context, identifier string) (*domain.Issue, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.getTask(ctx, identifier)
}

// getTask 内部方法：获取单个任务
func (c *BeadsClient) getTask(ctx context.Context, identifier string) (*domain.Issue, error) {
	args := []string{"issue", "show", identifier}
	output, err := c.runCommand(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("get task failed: %w", err)
	}

	var result beadsIssue
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("parse task: %w", err)
	}

	return result.toDomain(), nil
}

// runCommand 执行 Beads CLI 命令并返回输出
func (c *BeadsClient) runCommand(ctx context.Context, args ...string) ([]byte, error) {
	logging.Debug("executing beads command",
		"command", strings.Join(args, " "),
	)

	cmd := exec.CommandContext(ctx, c.cliPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logging.Error("beads command failed",
			"command", strings.Join(args, " "),
			"error", err.Error(),
			"output", string(output),
		)
		return nil, fmt.Errorf("beads command failed: %w, output: %s", err, string(output))
	}
	return output, nil
}

// parseIssueList 解析任务列表 JSON
func (c *BeadsClient) parseIssueList(output []byte) ([]*domain.Issue, error) {
	if len(output) == 0 {
		return nil, nil
	}

	var issues []beadsIssue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("unmarshal issue list: %w", err)
	}

	result := make([]*domain.Issue, len(issues))
	for i, issue := range issues {
		result[i] = issue.toDomain()
	}

	return result, nil
}

// parseConversationHistory 解析对话历史 JSON
func (c *BeadsClient) parseConversationHistory(output []byte) ([]domain.ConversationTurn, error) {
	if len(output) == 0 {
		return nil, nil
	}

	var turns []domain.ConversationTurn
	if err := json.Unmarshal(output, &turns); err != nil {
		return nil, fmt.Errorf("unmarshal conversation history: %w", err)
	}

	return turns, nil
}

// beadsIssue Beads CLI 返回的任务结构
type beadsIssue struct {
	ID          string          `json:"id"`
	Identifier  string          `json:"identifier"`
	Title       string          `json:"title"`
	Description string          `json:"description,omitempty"`
	State       string          `json:"state"`
	Priority    int             `json:"priority,omitempty"`
	Labels      []string        `json:"labels,omitempty"`
	BranchName  string          `json:"branch_name,omitempty"`
	URL         string          `json:"url,omitempty"`
	CreatedAt   string          `json:"created_at,omitempty"`
	UpdatedAt   string          `json:"updated_at,omitempty"`
	Custom      json.RawMessage `json:"custom,omitempty"` // 存储自定义数据，包括 StageState
}

// toDomain 转换为领域模型
func (bi beadsIssue) toDomain() *domain.Issue {
	issue := &domain.Issue{
		ID:         bi.ID,
		Identifier: bi.Identifier,
		Title:      bi.Title,
		State:      bi.State,
		Labels:     bi.Labels,
		BlockedBy:  make([]domain.BlockerRef, 0),
	}

	if bi.Description != "" {
		issue.Description = &bi.Description
	}

	if bi.Priority > 0 {
		issue.Priority = &bi.Priority
	}

	if bi.BranchName != "" {
		issue.BranchName = &bi.BranchName
	}

	if bi.URL != "" {
		issue.URL = &bi.URL
	}

	// 解析时间
	if bi.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, bi.CreatedAt); err == nil {
			issue.CreatedAt = &t
		}
	}
	if bi.UpdatedAt != "" {
		if t, err := time.Parse(time.RFC3339, bi.UpdatedAt); err == nil {
			issue.UpdatedAt = &t
		}
	}

	return issue
}

// GetBDDContent 获取任务的 BDD 规则内容
func (c *BeadsClient) GetBDDContent(ctx context.Context, identifier string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	args := []string{"issue", "show", identifier, "--field", "bdd_content"}
	output, err := c.runCommand(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("get BDD content failed: %w", err)
	}

	return string(output), nil
}

// UpdateBDDContent 更新任务的 BDD 规则内容
func (c *BeadsClient) UpdateBDDContent(ctx context.Context, identifier string, content string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	args := []string{"issue", "update", identifier, "--field", "bdd_content=" + content}
	_, err := c.runCommand(ctx, args...)
	if err != nil {
		return fmt.Errorf("update BDD content failed: %w", err)
	}

	return nil
}

// ApproveBDD 通过 BDD 审核
func (c *BeadsClient) ApproveBDD(ctx context.Context, identifier string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	args := []string{"issue", "update", identifier, "--field", "bdd_status=approved"}
	_, err := c.runCommand(ctx, args...)
	if err != nil {
		return fmt.Errorf("approve BDD failed: %w", err)
	}

	return nil
}

// RejectBDD 驳回 BDD 审核
func (c *BeadsClient) RejectBDD(ctx context.Context, identifier string, reason string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	args := []string{"issue", "update", identifier, "--field", "bdd_status=rejected", "--field", "bdd_reject_reason="+reason}
	_, err := c.runCommand(ctx, args...)
	if err != nil {
		return fmt.Errorf("reject BDD failed: %w", err)
	}

	return nil
}

// GetVerificationReport 获取任务的验收报告
func (c *BeadsClient) GetVerificationReport(ctx context.Context, identifier string) (*domain.VerificationReport, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	args := []string{"issue", "show", identifier, "--field", "verification_report"}
	output, err := c.runCommand(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("get verification report failed: %w", err)
	}

	if len(output) == 0 {
		return nil, nil
	}

	var report domain.VerificationReport
	if err := json.Unmarshal(output, &report); err != nil {
		// 如果不是 JSON，可能是纯文本存储
		return &domain.VerificationReport{
			TaskIdentifier: identifier,
			RawContent:     string(output),
		}, nil
	}

	return &report, nil
}

// UpdateVerificationReport 更新任务的验收报告
func (c *BeadsClient) UpdateVerificationReport(ctx context.Context, identifier string, report *domain.VerificationReport) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	reportJSON, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshal verification report: %w", err)
	}

	args := []string{"issue", "update", identifier, "--field", "verification_report=" + string(reportJSON)}
	_, err = c.runCommand(ctx, args...)
	if err != nil {
		return fmt.Errorf("update verification report failed: %w", err)
	}

	return nil
}

// ApproveVerification 通过验收
func (c *BeadsClient) ApproveVerification(ctx context.Context, identifier string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	args := []string{"issue", "update", identifier, "--field", "verification_status=approved", "--state", "Done"}
	_, err := c.runCommand(ctx, args...)
	if err != nil {
		return fmt.Errorf("approve verification failed: %w", err)
	}

	return nil
}

// RejectVerification 驳回验收（流转回实现中）
func (c *BeadsClient) RejectVerification(ctx context.Context, identifier string, reason string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	args := []string{"issue", "update", identifier, "--field", "verification_status=rejected", "--field", "verification_reject_reason="+reason, "--state", "In Progress"}
	_, err := c.runCommand(ctx, args...)
	if err != nil {
		return fmt.Errorf("reject verification failed: %w", err)
	}

	return nil
}

// GetArchitectureContent 获取任务的架构设计内容
func (c *BeadsClient) GetArchitectureContent(ctx context.Context, identifier string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	args := []string{"issue", "show", identifier, "--field", "architecture_content"}
	output, err := c.runCommand(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("get architecture content failed: %w", err)
	}

	return string(output), nil
}

// UpdateArchitectureContent 更新任务的架构设计内容
func (c *BeadsClient) UpdateArchitectureContent(ctx context.Context, identifier string, content string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	args := []string{"issue", "update", identifier, "--field", "architecture_content=" + content}
	_, err := c.runCommand(ctx, args...)
	if err != nil {
		return fmt.Errorf("update architecture content failed: %w", err)
	}

	return nil
}

// ApproveArchitecture 通过架构审核
func (c *BeadsClient) ApproveArchitecture(ctx context.Context, identifier string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	args := []string{"issue", "update", identifier, "--field", "architecture_status=approved"}
	_, err := c.runCommand(ctx, args...)
	if err != nil {
		return fmt.Errorf("approve architecture failed: %w", err)
	}

	return nil
}

// RejectArchitecture 驳回架构审核
func (c *BeadsClient) RejectArchitecture(ctx context.Context, identifier string, reason string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	args := []string{"issue", "update", identifier, "--field", "architecture_status=rejected", "--field", "architecture_reject_reason="+reason}
	_, err := c.runCommand(ctx, args...)
	if err != nil {
		return fmt.Errorf("reject architecture failed: %w", err)
	}

	return nil
}

// GetTDDContent 获取任务的 TDD 规则内容
func (c *BeadsClient) GetTDDContent(ctx context.Context, identifier string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	args := []string{"issue", "show", identifier, "--field", "tdd_content"}
	output, err := c.runCommand(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("get TDD content failed: %w", err)
	}

	return string(output), nil
}

// UpdateTDDContent 更新任务的 TDD 规则内容
func (c *BeadsClient) UpdateTDDContent(ctx context.Context, identifier string, content string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	args := []string{"issue", "update", identifier, "--field", "tdd_content=" + content}
	_, err := c.runCommand(ctx, args...)
	if err != nil {
		return fmt.Errorf("update TDD content failed: %w", err)
	}

	return nil
}
