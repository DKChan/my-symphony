// Package tracker 提供文件系统 Tracker 实现
package tracker

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dministrator/symphony/internal/domain"
	"github.com/dministrator/symphony/internal/logging"
)

const (
	// fileTrackerDir FileTracker 存储目录
	fileTrackerDir = ".sym"
	// taskFileName 状态索引文件名
	taskFileName = "task.md"
	// fileTrackerTimeout 默认超时时间
	fileTrackerTimeout = 5 * time.Second
	// writeQueueSize 写入队列大小
	writeQueueSize = 100
)

// writeRequest 写入请求
type writeRequest struct {
	fn      func() error
	done    chan error
}

// FileClient 文件系统 Tracker 实现
type FileClient struct {
	baseDir     string
	timeout     time.Duration
	writeQueue  chan *writeRequest
	writeWg     sync.WaitGroup
	stopWriteCh chan struct{}
}

// NewFileClient 创建新的 File Tracker 客户端
func NewFileClient() *FileClient {
	return NewFileClientWithDir(fileTrackerDir)
}

// NewFileClientWithDir 创建带有指定目录的 File Tracker 客户端
func NewFileClientWithDir(baseDir string) *FileClient {
	c := &FileClient{
		baseDir:     baseDir,
		timeout:     fileTrackerTimeout,
		writeQueue:  make(chan *writeRequest, writeQueueSize),
		stopWriteCh: make(chan struct{}),
	}

	// 启动写入协程
	c.startWriteWorker()

	return c
}

// startWriteWorker 启动写入工作协程
func (c *FileClient) startWriteWorker() {
	c.writeWg.Add(1)
	go func() {
		defer c.writeWg.Done()
		for {
			select {
			case req := <-c.writeQueue:
				if req == nil {
					continue
				}
				err := req.fn()
				if req.done != nil {
					req.done <- err
				}
			case <-c.stopWriteCh:
				// 处理剩余请求
				for len(c.writeQueue) > 0 {
					req := <-c.writeQueue
					if req != nil && req.done != nil {
						req.done <- fmt.Errorf("client shutting down")
					}
				}
				return
			}
		}
	}()
}

// Stop 停止写入工作协程
func (c *FileClient) Stop() {
	close(c.stopWriteCh)
	c.writeWg.Wait()
}

// submitWrite 提交写入请求（同步等待）
func (c *FileClient) submitWrite(fn func() error) error {
	done := make(chan error, 1)
	req := &writeRequest{
		fn:   fn,
		done: done,
	}

	select {
	case c.writeQueue <- req:
		select {
		case err := <-done:
			return err
		case <-time.After(c.timeout):
			return fmt.Errorf("write timeout")
		}
	default:
		// 队列满，直接执行
		return fn()
	}
}

// SetTimeout 设置超时时间
func (c *FileClient) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}

// CheckAvailability 检查文件系统是否可用
func (c *FileClient) CheckAvailability() error {
	// 检查目录是否存在或可创建
	info, err := os.Stat(c.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			// 目录不存在，尝试创建
			if err := os.MkdirAll(c.baseDir, 0755); err != nil {
				return fmt.Errorf("tracker.unavailable: 无法创建目录 %s: %w", c.baseDir, err)
			}
			return nil
		}
		return fmt.Errorf("tracker.unavailable: 目录 %s 不可访问: %w", c.baseDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("tracker.unavailable: %s 不是目录", c.baseDir)
	}
	return nil
}

// FetchCandidateIssues 获取处于活跃状态的候选问题
func (c *FileClient) FetchCandidateIssues(ctx context.Context, activeStates []string) ([]*domain.Issue, error) {
	return c.ListTasksByState(ctx, activeStates)
}

// FetchIssuesByStates 获取指定状态的问题
func (c *FileClient) FetchIssuesByStates(ctx context.Context, states []string) ([]*domain.Issue, error) {
	return c.ListTasksByState(ctx, states)
}

// FetchIssueStatesByIDs 按 ID 获取问题状态
func (c *FileClient) FetchIssueStatesByIDs(ctx context.Context, ids []string) ([]*domain.Issue, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var issues []*domain.Issue
	for _, id := range ids {
		issue, err := c.GetTask(ctx, id)
		if err != nil {
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
func (c *FileClient) CreateTask(ctx context.Context, title, description string) (*domain.Issue, error) {
	_, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// 生成任务 ID
	identifier := c.generateIdentifier(title)
	taskDir := filepath.Join(c.baseDir, identifier)

	// 创建任务目录结构
	if err := c.createTaskStructure(taskDir, identifier, title, description); err != nil {
		return nil, fmt.Errorf("create task failed: %w", err)
	}

	// 返回创建的任务
	now := time.Now()
	return &domain.Issue{
		ID:         identifier,
		Identifier: identifier,
		Title:      title,
		State:      "backlog",
		CreatedAt:  &now,
		UpdatedAt:  &now,
	}, nil
}

// CreateSubTask 创建子任务（带依赖关系）
func (c *FileClient) CreateSubTask(ctx context.Context, parentIdentifier string, title, description string, blockedBy []string) (*domain.Issue, error) {
	_, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	taskDir := filepath.Join(c.baseDir, parentIdentifier)
	if _, err := os.Stat(taskDir); err != nil {
		return nil, fmt.Errorf("parent task not found: %w", err)
	}

	// 解析子任务类型和编号
	subTaskType, subTaskNum, subTaskName := parseSubTaskTitle(title)
	subTaskID := fmt.Sprintf("%s-%s%d", parentIdentifier, strings.ToUpper(subTaskType[:1]), subTaskNum)

	// 创建子任务详情文件
	subTaskFile := c.getSubTaskFilePath(taskDir, subTaskType, subTaskNum, subTaskName, 1)
	if err := c.createSubTaskFile(subTaskFile, subTaskID, parentIdentifier, subTaskType, subTaskName, blockedBy, description); err != nil {
		return nil, fmt.Errorf("create subtask failed: %w", err)
	}

	// 更新状态索引文件
	if err := c.updateTaskIndex(taskDir, subTaskID, subTaskType, subTaskNum, subTaskName, 1, "pending"); err != nil {
		return nil, fmt.Errorf("update task index failed: %w", err)
	}

	now := time.Now()
	return &domain.Issue{
		ID:         subTaskID,
		Identifier: subTaskID,
		Title:      title,
		State:      "pending",
		CreatedAt:  &now,
		UpdatedAt:  &now,
	}, nil
}

// GetTask 获取单个任务详情
func (c *FileClient) GetTask(ctx context.Context, identifier string) (*domain.Issue, error) {
	_, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	taskDir := filepath.Join(c.baseDir, identifier)
	taskFile := filepath.Join(taskDir, taskFileName)

	data, err := os.ReadFile(taskFile)
	if err != nil {
		return nil, fmt.Errorf("get task failed: %w", err)
	}

	// 解析 frontmatter
	fm, err := parseFrontmatter(data)
	if err != nil {
		return nil, fmt.Errorf("parse task frontmatter: %w", err)
	}

	issue := &domain.Issue{
		ID:         fm["id"].(string),
		Identifier: fm["id"].(string),
		Title:      fm["title"].(string),
		State:      fm["status"].(string),
		CreatedAt:  parseTime(fm["created"]),
		UpdatedAt:  parseTime(fm["updated"]),
	}

	if desc, ok := fm["description"].(string); ok {
		issue.Description = &desc
	}

	if labels, ok := fm["labels"].([]interface{}); ok {
		for _, l := range labels {
			issue.Labels = append(issue.Labels, l.(string))
		}
	}

	return issue, nil
}

// UpdateStage 更新任务阶段状态
func (c *FileClient) UpdateStage(ctx context.Context, identifier string, stage domain.StageState) error {
	taskDir := filepath.Join(c.baseDir, identifier)
	taskFile := filepath.Join(taskDir, taskFileName)

	// 使用写入队列确保并发安全
	return c.submitWrite(func() error {
		data, err := os.ReadFile(taskFile)
		if err != nil {
			return fmt.Errorf("update stage failed: %w", err)
		}

		// 解析并更新 frontmatter
		fm, content, err := parseFrontmatterWithContent(data)
		if err != nil {
			return fmt.Errorf("parse frontmatter: %w", err)
		}

		fm["status"] = stageToStatus(stage.Status)
		fm["phase"] = stage.Name
		fm["updated"] = time.Now().Format(time.RFC3339)

		// 写回文件
		newData := formatFrontmatter(fm, content)
		if err := os.WriteFile(taskFile, newData, 0644); err != nil {
			return fmt.Errorf("write task file: %w", err)
		}

		return nil
	})
}

// GetStageState 获取任务的阶段状态（用于崩溃恢复）
func (c *FileClient) GetStageState(ctx context.Context, identifier string) (*domain.StageState, error) {
	_, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	taskDir := filepath.Join(c.baseDir, identifier)
	taskFile := filepath.Join(taskDir, taskFileName)

	data, err := os.ReadFile(taskFile)
	if err != nil {
		return nil, fmt.Errorf("get stage state failed: %w", err)
	}

	fm, err := parseFrontmatter(data)
	if err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}

	stage := &domain.StageState{
		Name:   fm["phase"].(string),
		Status: statusToStageStatus(fm["status"].(string)),
	}

	// 处理时间字段
	if created := parseTime(fm["created"]); created != nil {
		stage.StartedAt = *created
	}
	if updated := parseTime(fm["updated"]); updated != nil {
		stage.UpdatedAt = *updated
	}

	if iter, ok := fm["iteration"].(float64); ok {
		stage.RetryCount = int(iter) - 1
	}

	return stage, nil
}

// AppendConversation 追加对话记录
func (c *FileClient) AppendConversation(ctx context.Context, identifier string, turn domain.ConversationTurn) error {
	_, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// identifier 是子任务 ID，如 "SYM-001-P1"
	parentID, subTaskType, subTaskNum := parseSubTaskID(identifier)
	taskDir := filepath.Join(c.baseDir, parentID)

	// 找到对应的子任务文件
	subTaskFiles, err := c.findSubTaskFiles(taskDir, subTaskType, subTaskNum)
	if err != nil || len(subTaskFiles) == 0 {
		return fmt.Errorf("subtask file not found: %w", err)
	}

	// 使用最新的版本文件
	subTaskFile := subTaskFiles[len(subTaskFiles)-1]

	data, err := os.ReadFile(subTaskFile)
	if err != nil {
		return fmt.Errorf("read subtask file: %w", err)
	}

	// 追加对话记录
	turnContent := formatConversationTurn(turn)
	newData := append(data, []byte(turnContent)...)

	if err := os.WriteFile(subTaskFile, newData, 0644); err != nil {
		return fmt.Errorf("write subtask file: %w", err)
	}

	// 更新时间戳
	fm, content, err := parseFrontmatterWithContent(data)
	if err == nil {
		fm["updated"] = time.Now().Format(time.RFC3339)
		newFrontmatterData := formatFrontmatter(fm, content)
		if err := os.WriteFile(subTaskFile, newFrontmatterData, 0644); err != nil {
			logging.Warn("failed to update subtask timestamp", "error", err)
		}
	}

	return nil
}

// GetConversationHistory 获取对话历史记录
func (c *FileClient) GetConversationHistory(ctx context.Context, identifier string) ([]domain.ConversationTurn, error) {
	_, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	parentID, subTaskType, subTaskNum := parseSubTaskID(identifier)
	taskDir := filepath.Join(c.baseDir, parentID)

	subTaskFiles, err := c.findSubTaskFiles(taskDir, subTaskType, subTaskNum)
	if err != nil || len(subTaskFiles) == 0 {
		return nil, fmt.Errorf("subtask file not found: %w", err)
	}

	subTaskFile := subTaskFiles[len(subTaskFiles)-1]
	data, err := os.ReadFile(subTaskFile)
	if err != nil {
		return nil, fmt.Errorf("read subtask file: %w", err)
	}

	return parseConversationHistory(data), nil
}

// ListTasksByState 按状态获取任务列表
func (c *FileClient) ListTasksByState(ctx context.Context, states []string) ([]*domain.Issue, error) {
	_, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var issues []*domain.Issue

	// 遍历 .sym 目录下的所有任务目录
	entries, err := os.ReadDir(c.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list tasks failed: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		taskFile := filepath.Join(c.baseDir, entry.Name(), taskFileName)
		data, err := os.ReadFile(taskFile)
		if err != nil {
			continue
		}

		fm, err := parseFrontmatter(data)
		if err != nil {
			continue
		}

		status, ok := fm["status"].(string)
		if !ok {
			continue
		}

		// 检查状态是否匹配
		if !containsState(states, status) {
			continue
		}

		issue := &domain.Issue{
			ID:         fm["id"].(string),
			Identifier: fm["id"].(string),
			Title:      fm["title"].(string),
			State:      status,
			CreatedAt:  parseTime(fm["created"]),
			UpdatedAt:  parseTime(fm["updated"]),
		}

		issues = append(issues, issue)
	}

	return issues, nil
}

// 内部方法

// generateIdentifier 根据标题生成任务标识符
func (c *FileClient) generateIdentifier(title string) string {
	// 简化实现：使用时间戳 + 标题哈希
	// 实际应该有更好的生成逻辑
	timestamp := time.Now().Format("0102")
	hash := simpleHash(title) % 1000
	return fmt.Sprintf("SYM-%s%03d", timestamp, hash)
}

// simpleHash 简单哈希函数
func simpleHash(s string) int {
	h := 0
	for _, c := range s {
		h = (h * 31 + int(c)) % 10000
	}
	return h
}

// createTaskStructure 创建任务目录结构
func (c *FileClient) createTaskStructure(taskDir, identifier, title, description string) error {
	// 创建目录
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return err
	}

	// 创建子任务目录
	for _, subdir := range []string{"Planner", "Generator", "Evaluator"} {
		if err := os.MkdirAll(filepath.Join(taskDir, subdir), 0755); err != nil {
			return err
		}
	}

	// 创建状态索引文件
	taskFile := filepath.Join(taskDir, taskFileName)
	now := time.Now().Format(time.RFC3339)

	fm := map[string]interface{}{
		"id":            identifier,
		"title":         title,
		"status":        "backlog",
		"phase":         "backlog",
		"iteration":     1,
		"created":       now,
		"updated":       now,
		"max_iterations": 5,
	}

	if description != "" {
		fm["description"] = description
	}

	content := `# Planner

- P1: 需求澄清 ⬜
- P2: BDD规则 ⬜
- P3: 领域建模 ⬜
- P4: 架构设计 ⬜
- P5: 接口设计 ⬜

# Generator

- G1: BDD测试脚本-v1 ⬜
- G2: 集成测试-v1 ⬜
- G3: 单元测试-v1 ⬜
- G4: 代码实现-v1 ⬜

# Evaluator

- E1: 评估验收-v1 ⬜
`

	data := formatFrontmatter(fm, content)
	return os.WriteFile(taskFile, data, 0644)
}

// createSubTaskFile 创建子任务详情文件
func (c *FileClient) createSubTaskFile(filePath, id, parent, subTaskType, name string, blockedBy []string, description string) error {
	now := time.Now().Format(time.RFC3339)

	fm := map[string]interface{}{
		"id":       id,
		"parent":   parent,
		"type":     subTaskType,
		"name":     name,
		"version":  1,
		"status":   "pending",
		"created":  now,
		"updated":  now,
	}

	if len(blockedBy) > 0 {
		fm["blocked_by"] = blockedBy
	}

	content := fmt.Sprintf(`## 任务描述

%s

## 输入

（待填充）

## 对话记录

（待填充）

## 输出

（待填充）
`, description)

	data := formatFrontmatter(fm, content)
	return os.WriteFile(filePath, data, 0644)
}

// updateTaskIndex 更新状态索引文件
func (c *FileClient) updateTaskIndex(taskDir string, subTaskID string, subTaskType string, subTaskNum int, name string, version int, status string) error {
	taskFile := filepath.Join(taskDir, taskFileName)
	data, err := os.ReadFile(taskFile)
	if err != nil {
		return err
	}

	fm, content, err := parseFrontmatterWithContent(data)
	if err != nil {
		return err
	}

	// 更新时间戳
	fm["updated"] = time.Now().Format(time.RFC3339)

	// 更新 markdown 内容中的子任务状态
	statusMark := statusToMark(status)
	content = updateSubTaskStatus(content, strings.ToUpper(subTaskType[:1]), subTaskNum, name, version, statusMark)

	newData := formatFrontmatter(fm, content)
	return os.WriteFile(taskFile, newData, 0644)
}

// getSubTaskFilePath 获取子任务文件路径
func (c *FileClient) getSubTaskFilePath(taskDir string, subTaskType string, subTaskNum int, name string, version int) string {
	switch strings.ToLower(subTaskType) {
	case "planner", "p":
		return filepath.Join(taskDir, "Planner", fmt.Sprintf("P%d-%s.md", subTaskNum, name))
	case "generator", "g":
		return filepath.Join(taskDir, "Generator", fmt.Sprintf("G%d-%s-v%d.md", subTaskNum, name, version))
	case "evaluator", "e":
		return filepath.Join(taskDir, "Evaluator", fmt.Sprintf("E%d-%s-v%d.md", subTaskNum, name, version))
	}
	return filepath.Join(taskDir, strings.ToUpper(subTaskType[:1])+subTaskType[1:], fmt.Sprintf("%s%d-%s-v%d.md", strings.ToUpper(subTaskType[:1]), subTaskNum, name, version))
}

// findSubTaskFiles 查找子任务文件（所有版本）
func (c *FileClient) findSubTaskFiles(taskDir, subTaskType string, subTaskNum int) ([]string, error) {
	typeDir := ""
	switch strings.ToLower(subTaskType) {
	case "p", "planner":
		typeDir = "Planner"
	case "g", "generator":
		typeDir = "Generator"
	case "e", "evaluator":
		typeDir = "Evaluator"
	default:
		return nil, fmt.Errorf("unknown subtask type: %s", subTaskType)
	}

	dirPath := filepath.Join(taskDir, typeDir)
	var files []string

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasPrefix(d.Name(), fmt.Sprintf("%s%d-", strings.ToUpper(subTaskType[:1]), subTaskNum)) {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 按版本号排序
	sortByVersion(files)
	return files, nil
}

// BDD 相关方法

// GetBDDContent 获取任务的 BDD 规则内容
func (c *FileClient) GetBDDContent(ctx context.Context, identifier string) (string, error) {
	parentID := identifier
	if strings.Contains(identifier, "-") {
		parentID, _, _ = parseSubTaskID(identifier)
	}

	taskDir := filepath.Join(c.baseDir, parentID)
	bddFile := filepath.Join(taskDir, "Planner", "P2-BDD规则.md")

	data, err := os.ReadFile(bddFile)
	if err != nil {
		return "", fmt.Errorf("get BDD content failed: %w", err)
	}

	_, content, err := parseFrontmatterWithContent(data)
	if err != nil {
		return string(data), nil
	}

	return content, nil
}

// UpdateBDDContent 更新任务的 BDD 规则内容
func (c *FileClient) UpdateBDDContent(ctx context.Context, identifier string, content string) error {
	parentID := identifier
	if strings.Contains(identifier, "-") {
		parentID, _, _ = parseSubTaskID(identifier)
	}

	taskDir := filepath.Join(c.baseDir, parentID)
	bddFile := filepath.Join(taskDir, "Planner", "P2-BDD规则.md")

	// 如果文件不存在，创建它
	if _, err := os.Stat(bddFile); os.IsNotExist(err) {
		now := time.Now().Format(time.RFC3339)
		fm := map[string]interface{}{
			"id":      parentID + "-P2",
			"parent":  parentID,
			"type":    "planner",
			"name":    "BDD规则",
			"version": 1,
			"status":  "pending",
			"created": now,
			"updated": now,
		}
		data := formatFrontmatter(fm, content)
		return os.WriteFile(bddFile, data, 0644)
	}

	data, err := os.ReadFile(bddFile)
	if err != nil {
		return fmt.Errorf("read BDD file: %w", err)
	}

	fm, _, err := parseFrontmatterWithContent(data)
	if err != nil {
		return fmt.Errorf("parse frontmatter: %w", err)
	}

	fm["updated"] = time.Now().Format(time.RFC3339)
	newData := formatFrontmatter(fm, content)
	return os.WriteFile(bddFile, newData, 0644)
}

// ApproveBDD 通过 BDD 审核
func (c *FileClient) ApproveBDD(ctx context.Context, identifier string) error {
	return c.updateSubTaskStatus(ctx, identifier, "completed")
}

// RejectBDD 驳回 BDD 审核
func (c *FileClient) RejectBDD(ctx context.Context, identifier string, reason string) error {
	return c.updateSubTaskStatusWithReason(ctx, identifier, "failed", reason)
}

// Architecture 相关方法

// GetArchitectureContent 获取任务的架构设计内容
func (c *FileClient) GetArchitectureContent(ctx context.Context, identifier string) (string, error) {
	parentID := identifier
	if strings.Contains(identifier, "-") {
		parentID, _, _ = parseSubTaskID(identifier)
	}

	taskDir := filepath.Join(c.baseDir, parentID)
	archFile := filepath.Join(taskDir, "Planner", "P4-架构设计.md")

	data, err := os.ReadFile(archFile)
	if err != nil {
		return "", fmt.Errorf("get architecture content failed: %w", err)
	}

	_, content, err := parseFrontmatterWithContent(data)
	if err != nil {
		return string(data), nil
	}

	return content, nil
}

// UpdateArchitectureContent 更新任务的架构设计内容
func (c *FileClient) UpdateArchitectureContent(ctx context.Context, identifier string, content string) error {
	parentID := identifier
	if strings.Contains(identifier, "-") {
		parentID, _, _ = parseSubTaskID(identifier)
	}

	taskDir := filepath.Join(c.baseDir, parentID)
	archFile := filepath.Join(taskDir, "Planner", "P4-架构设计.md")

	if _, err := os.Stat(archFile); os.IsNotExist(err) {
		now := time.Now().Format(time.RFC3339)
		fm := map[string]interface{}{
			"id":      parentID + "-P4",
			"parent":  parentID,
			"type":    "planner",
			"name":    "架构设计",
			"version": 1,
			"status":  "pending",
			"created": now,
			"updated": now,
		}
		data := formatFrontmatter(fm, content)
		return os.WriteFile(archFile, data, 0644)
	}

	data, err := os.ReadFile(archFile)
	if err != nil {
		return fmt.Errorf("read architecture file: %w", err)
	}

	fm, _, err := parseFrontmatterWithContent(data)
	if err != nil {
		return fmt.Errorf("parse frontmatter: %w", err)
	}

	fm["updated"] = time.Now().Format(time.RFC3339)
	newData := formatFrontmatter(fm, content)
	return os.WriteFile(archFile, newData, 0644)
}

// ApproveArchitecture 通过架构审核
func (c *FileClient) ApproveArchitecture(ctx context.Context, identifier string) error {
	return c.updateSubTaskStatus(ctx, identifier, "completed")
}

// RejectArchitecture 驳回架构审核
func (c *FileClient) RejectArchitecture(ctx context.Context, identifier string, reason string) error {
	return c.updateSubTaskStatusWithReason(ctx, identifier, "failed", reason)
}

// TDD 相关方法

// GetTDDContent 获取任务的 TDD 规则内容
func (c *FileClient) GetTDDContent(ctx context.Context, identifier string) (string, error) {
	parentID := identifier
	if strings.Contains(identifier, "-") {
		parentID, _, _ = parseSubTaskID(identifier)
	}

	taskDir := filepath.Join(c.baseDir, parentID)
	tddFile := filepath.Join(taskDir, "Generator", "G3-单元测试-v1.md")

	data, err := os.ReadFile(tddFile)
	if err != nil {
		return "", fmt.Errorf("get TDD content failed: %w", err)
	}

	_, content, err := parseFrontmatterWithContent(data)
	if err != nil {
		return string(data), nil
	}

	return content, nil
}

// UpdateTDDContent 更新任务的 TDD 规则内容
func (c *FileClient) UpdateTDDContent(ctx context.Context, identifier string, content string) error {
	parentID := identifier
	if strings.Contains(identifier, "-") {
		parentID, _, _ = parseSubTaskID(identifier)
	}

	taskDir := filepath.Join(c.baseDir, parentID)
	tddFile := filepath.Join(taskDir, "Generator", "G3-单元测试-v1.md")

	if _, err := os.Stat(tddFile); os.IsNotExist(err) {
		now := time.Now().Format(time.RFC3339)
		fm := map[string]interface{}{
			"id":        parentID + "-G3",
			"parent":    parentID,
			"type":      "generator",
			"name":      "单元测试",
			"version":   1,
			"status":    "pending",
			"created":   now,
			"updated":   now,
			"blocked_by": []string{parentID + "-G2"},
		}
		data := formatFrontmatter(fm, content)
		return os.WriteFile(tddFile, data, 0644)
	}

	data, err := os.ReadFile(tddFile)
	if err != nil {
		return fmt.Errorf("read TDD file: %w", err)
	}

	fm, _, err := parseFrontmatterWithContent(data)
	if err != nil {
		return fmt.Errorf("parse frontmatter: %w", err)
	}

	fm["updated"] = time.Now().Format(time.RFC3339)
	newData := formatFrontmatter(fm, content)
	return os.WriteFile(tddFile, newData, 0644)
}

// Verification 相关方法

// GetVerificationReport 获取任务的验收报告
func (c *FileClient) GetVerificationReport(ctx context.Context, identifier string) (*domain.VerificationReport, error) {
	parentID := identifier
	if strings.Contains(identifier, "-") {
		parentID, _, _ = parseSubTaskID(identifier)
	}

	taskDir := filepath.Join(c.baseDir, parentID)
	evalFile := filepath.Join(taskDir, "Evaluator", "E1-评估验收-v1.md")

	data, err := os.ReadFile(evalFile)
	if err != nil {
		return nil, fmt.Errorf("get verification report failed: %w", err)
	}

	fm, content, err := parseFrontmatterWithContent(data)
	if err != nil {
		return &domain.VerificationReport{
			TaskIdentifier: identifier,
			RawContent:     string(data),
		}, nil
	}

	report := &domain.VerificationReport{
		TaskIdentifier: identifier,
		TaskTitle:      fm["name"].(string),
		RawContent:     content,
	}

	// 处理时间字段
	if updated := parseTime(fm["updated"]); updated != nil {
		report.GeneratedAt = *updated
	}

	return report, nil
}

// UpdateVerificationReport 更新任务的验收报告
func (c *FileClient) UpdateVerificationReport(ctx context.Context, identifier string, report *domain.VerificationReport) error {
	parentID := identifier
	if strings.Contains(identifier, "-") {
		parentID, _, _ = parseSubTaskID(identifier)
	}

	taskDir := filepath.Join(c.baseDir, parentID)

	// 找到当前版本的评估文件
	evalFiles, err := c.findSubTaskFiles(taskDir, "e", 1)
	if err != nil || len(evalFiles) == 0 {
		return fmt.Errorf("evaluator file not found: %w", err)
	}

	evalFile := evalFiles[len(evalFiles)-1]
	now := time.Now().Format(time.RFC3339)

	// 将报告序列化为 JSON 存储
	reportJSON, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}

	data, err := os.ReadFile(evalFile)
	if err != nil {
		return fmt.Errorf("read eval file: %w", err)
	}

	fm, content, err := parseFrontmatterWithContent(data)
	if err != nil {
		return fmt.Errorf("parse frontmatter: %w", err)
	}

	fm["updated"] = now
	newContent := content + "\n\n## 验收报告\n\n```json\n" + string(reportJSON) + "\n```\n"
	newData := formatFrontmatter(fm, newContent)

	return os.WriteFile(evalFile, newData, 0644)
}

// ApproveVerification 通过验收
func (c *FileClient) ApproveVerification(ctx context.Context, identifier string) error {
	return c.updateSubTaskStatus(ctx, identifier, "completed")
}

// RejectVerification 驳回验收
func (c *FileClient) RejectVerification(ctx context.Context, identifier string, reason string) error {
	return c.updateSubTaskStatusWithReason(ctx, identifier, "failed", reason)
}

// updateSubTaskStatus 更新子任务状态
func (c *FileClient) updateSubTaskStatus(ctx context.Context, identifier string, status string) error {
	parentID, subTaskType, subTaskNum := parseSubTaskID(identifier)
	taskDir := filepath.Join(c.baseDir, parentID)

	// 找到最新版本的文件
	files, err := c.findSubTaskFiles(taskDir, subTaskType, subTaskNum)
	if err != nil || len(files) == 0 {
		return fmt.Errorf("subtask file not found: %w", err)
	}

	file := files[len(files)-1]
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read subtask file: %w", err)
	}

	fm, content, err := parseFrontmatterWithContent(data)
	if err != nil {
		return fmt.Errorf("parse frontmatter: %w", err)
	}

	fm["status"] = status
	fm["updated"] = time.Now().Format(time.RFC3339)

	newData := formatFrontmatter(fm, content)
	if err := os.WriteFile(file, newData, 0644); err != nil {
		return fmt.Errorf("write subtask file: %w", err)
	}

	// 更新状态索引
	version := 1
	if v, ok := fm["version"].(float64); ok {
		version = int(v)
	}
	name := fm["name"].(string)
	statusMark := statusToMark(status)

	taskFile := filepath.Join(taskDir, taskFileName)
	taskData, err := os.ReadFile(taskFile)
	if err != nil {
		return nil // 状态索引更新失败不影响主流程
	}

	taskFm, taskContent, err := parseFrontmatterWithContent(taskData)
	if err != nil {
		return nil
	}

	taskFm["updated"] = time.Now().Format(time.RFC3339)
	taskContent = updateSubTaskStatusInContent(taskContent, strings.ToUpper(subTaskType[:1]), subTaskNum, name, version, statusMark)
	newTaskData := formatFrontmatter(taskFm, taskContent)

	return os.WriteFile(taskFile, newTaskData, 0644)
}

// updateSubTaskStatusWithReason 更新子任务状态并添加原因
func (c *FileClient) updateSubTaskStatusWithReason(ctx context.Context, identifier string, status string, reason string) error {
	parentID, subTaskType, subTaskNum := parseSubTaskID(identifier)
	taskDir := filepath.Join(c.baseDir, parentID)

	files, err := c.findSubTaskFiles(taskDir, subTaskType, subTaskNum)
	if err != nil || len(files) == 0 {
		return fmt.Errorf("subtask file not found: %w", err)
	}

	file := files[len(files)-1]
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read subtask file: %w", err)
	}

	fm, content, err := parseFrontmatterWithContent(data)
	if err != nil {
		return fmt.Errorf("parse frontmatter: %w", err)
	}

	fm["status"] = status
	fm["updated"] = time.Now().Format(time.RFC3339)
	fm["error_message"] = reason

	newContent := content + "\n\n## 驳回原因\n\n" + reason + "\n"
	newData := formatFrontmatter(fm, newContent)

	if err := os.WriteFile(file, newData, 0644); err != nil {
		return fmt.Errorf("write subtask file: %w", err)
	}

	return c.updateSubTaskStatus(ctx, identifier, status)
}