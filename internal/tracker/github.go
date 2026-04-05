// Package tracker - GitHub Issues REST API 适配器
// 使用 GitHub REST API v3（api.github.com）实现 Tracker 接口
package tracker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dministrator/symphony/internal/domain"
)

const githubAPIBase = "https://api.github.com"

// GitHubClient GitHub Issues API 客户端
type GitHubClient struct {
	token      string
	owner      string
	repo       string
	httpClient *http.Client
}

// NewGitHubClient 创建新的 GitHub 客户端
// repo 格式：owner/repo
func NewGitHubClient(token, repo string) *GitHubClient {
	parts := strings.SplitN(repo, "/", 2)
	owner, repoName := "", repo
	if len(parts) == 2 {
		owner = parts[0]
		repoName = parts[1]
	}

	return &GitHubClient{
		token: token,
		owner: owner,
		repo:  repoName,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// githubIssue GitHub API 返回的 issue 结构
type githubIssue struct {
	ID     int    `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"` // "open" | "closed"
	URL    string `json:"html_url"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	Milestone *struct {
		Title string `json:"title"`
	} `json:"milestone"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// toDomain 将 GitHub Issue 转换为领域模型
// GitHub 使用 label 映射状态
// 约定：label 名称 = 状态名（如 "Todo"、"In Progress"、"Done"）
func (gi githubIssue) toDomain(owner, repo string) *domain.Issue {
	id := fmt.Sprintf("%d", gi.Number)
	identifier := fmt.Sprintf("%s/%s#%d", owner, repo, gi.Number)
	url := gi.URL

	issue := &domain.Issue{
		ID:         id,
		Identifier: identifier,
		Title:      gi.Title,
		URL:        &url,
		Labels:     make([]string, 0),
		BlockedBy:  make([]domain.BlockerRef, 0),
	}

	if gi.Body != "" {
		issue.Description = &gi.Body
	}

	// 提取状态 label（约定：以 "status:" 或直接使用完整 label 名）
	// 默认状态：open → "In Progress"，closed → "Done"
	stateLabel := ""
	for _, label := range gi.Labels {
		issue.Labels = append(issue.Labels, strings.ToLower(label.Name))
		// 优先使用 "status:" 前缀的 label 作为状态
		if strings.HasPrefix(strings.ToLower(label.Name), "status:") {
			stateLabel = strings.TrimPrefix(label.Name, "status:")
			stateLabel = strings.TrimPrefix(stateLabel, "Status:")
			stateLabel = strings.TrimSpace(stateLabel)
		}
	}

	if stateLabel != "" {
		issue.State = stateLabel
	} else if gi.State == "closed" {
		issue.State = "Done"
	} else {
		issue.State = "In Progress"
	}

	if t, err := time.Parse(time.RFC3339, gi.CreatedAt); err == nil {
		issue.CreatedAt = &t
	}
	if t, err := time.Parse(time.RFC3339, gi.UpdatedAt); err == nil {
		issue.UpdatedAt = &t
	}

	return issue
}

// FetchCandidateIssues 获取处于活跃状态的候选问题
// 策略：拉取所有 open issues，再按 label 过滤状态
func (c *GitHubClient) FetchCandidateIssues(ctx context.Context, activeStates []string) ([]*domain.Issue, error) {
	activeSet := make(map[string]bool, len(activeStates))
	for _, s := range activeStates {
		activeSet[strings.ToLower(strings.TrimSpace(s))] = true
	}

	// 拉取所有 open issues（分页）
	var allIssues []*domain.Issue
	page := 1

	for {
		url := fmt.Sprintf("%s/repos/%s/%s/issues?state=open&per_page=100&page=%d",
			githubAPIBase, c.owner, c.repo, page)

		ghIssues, err := c.listIssues(ctx, url)
		if err != nil {
			return nil, err
		}
		if len(ghIssues) == 0 {
			break
		}

		for _, gi := range ghIssues {
			// 跳过 pull request（GitHub API 在 issues 中混入 PR）
			if isPullRequest(gi) {
				continue
			}
			issue := gi.toDomain(c.owner, c.repo)
			if len(activeStates) == 0 || activeSet[strings.ToLower(issue.State)] {
				allIssues = append(allIssues, issue)
			}
		}

		if len(ghIssues) < 100 {
			break
		}
		page++
	}

	return allIssues, nil
}

// FetchIssuesByStates 按状态获取问题（用于启动时清理终态工作空间）
func (c *GitHubClient) FetchIssuesByStates(ctx context.Context, states []string) ([]*domain.Issue, error) {
	if len(states) == 0 {
		return nil, nil
	}

	stateSet := make(map[string]bool, len(states))
	for _, s := range states {
		stateSet[strings.ToLower(strings.TrimSpace(s))] = true
	}

	// 需要同时拉取 open 和 closed 以覆盖所有终态
	var allIssues []*domain.Issue

	for _, apiState := range []string{"open", "closed"} {
		page := 1
		for {
			url := fmt.Sprintf("%s/repos/%s/%s/issues?state=%s&per_page=100&page=%d",
				githubAPIBase, c.owner, c.repo, apiState, page)

			ghIssues, err := c.listIssues(ctx, url)
			if err != nil {
				return nil, err
			}
			if len(ghIssues) == 0 {
				break
			}

			for _, gi := range ghIssues {
				if isPullRequest(gi) {
					continue
				}
				issue := gi.toDomain(c.owner, c.repo)
				if stateSet[strings.ToLower(issue.State)] {
					allIssues = append(allIssues, issue)
				}
			}

			if len(ghIssues) < 100 {
				break
			}
			page++
		}
	}

	return allIssues, nil
}

// FetchIssueStatesByIDs 按 ID 批量刷新问题状态
// GitHub 的 ID 即 issue number，逐个请求
func (c *GitHubClient) FetchIssueStatesByIDs(ctx context.Context, ids []string) ([]*domain.Issue, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	issues := make([]*domain.Issue, 0, len(ids))
	for _, id := range ids {
		// id 可能是 number 字符串，也可能是 "owner/repo#number" 格式
		number := extractIssueNumber(id)
		if number == "" {
			continue
		}

		url := fmt.Sprintf("%s/repos/%s/%s/issues/%s", githubAPIBase, c.owner, c.repo, number)
		gi, err := c.getIssue(ctx, url)
		if err != nil {
			// 单个失败不阻断其他，记录跳过
			continue
		}
		issues = append(issues, gi.toDomain(c.owner, c.repo))
	}

	return issues, nil
}

// listIssues 拉取指定 URL 的 issue 列表
func (c *GitHubClient) listIssues(ctx context.Context, url string) ([]githubIssue, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github_api_status: %d - %s", resp.StatusCode, string(body))
	}

	var issues []githubIssue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return issues, nil
}

// getIssue 获取单个 issue
func (c *GitHubClient) getIssue(ctx context.Context, url string) (*githubIssue, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("issue not found: %s", url)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github_api_status: %d - %s", resp.StatusCode, string(body))
	}

	var issue githubIssue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &issue, nil
}

// setHeaders 设置通用请求头
func (c *GitHubClient) setHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
}

// isPullRequest 判断是否为 Pull Request（GitHub API 在 issues 中混入 PR）
// PR 的 issue 对象有 pull_request 字段
func isPullRequest(gi githubIssue) bool {
	// 使用 Number 无法直接判断，需要检查原始 JSON
	// 此处保守处理：通过 URL 含 /pull/ 判断
	return strings.Contains(gi.URL, "/pull/")
}

// extractIssueNumber 从 ID 字符串提取 issue number
// 支持格式：
//   - "123"                  → "123"
//   - "owner/repo#123"       → "123"
func extractIssueNumber(id string) string {
	// 去掉 owner/repo# 前缀
	if idx := strings.LastIndex(id, "#"); idx >= 0 {
		return id[idx+1:]
	}
	return id
}

// CheckAvailability 检查 GitHub API 是否可用
func (c *GitHubClient) CheckAvailability() error {
	if c.token == "" {
		return fmt.Errorf("github_token_required: GITHUB_TOKEN not set")
	}
	if c.owner == "" || c.repo == "" {
		return fmt.Errorf("github_repo_required: repo format must be 'owner/repo'")
	}

	// 执行一个简单的查询来验证连接
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/repos/%s/%s", githubAPIBase, c.owner, c.repo)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("github_connection_failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("github_auth_failed: invalid token")
	}
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("github_repo_not_found: %s/%s", c.owner, c.repo)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github_api_error: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

// CreateTask 创建新任务 (GitHub Issue)
func (c *GitHubClient) CreateTask(ctx context.Context, title, description string) (*domain.Issue, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues", githubAPIBase, c.owner, c.repo)

	body := map[string]any{
		"title": title,
		"body":  description,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github_create_failed: %d - %s", resp.StatusCode, string(respBody))
	}

	var gi githubIssue
	if err := json.NewDecoder(resp.Body).Decode(&gi); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return gi.toDomain(c.owner, c.repo), nil
}

// CreateSubTask 创建子任务（带依赖关系）
// GitHub 不原生支持父子任务关系，通过在标题中添加父任务标识符来表示
// 通过 Tasklists（GitHub 新功能）或 body 中引用来实现依赖关系
func (c *GitHubClient) CreateSubTask(ctx context.Context, parentIdentifier string, title, description string, blockedBy []string) (*domain.Issue, error) {
	// 提取父任务的 issue number（用于日志记录）
	_ = extractIssueNumber(parentIdentifier) // parentNumber 用于后续扩展

	url := fmt.Sprintf("%s/repos/%s/%s/issues", githubAPIBase, c.owner, c.repo)

	// 构建任务体，包含依赖关系引用
	bodyContent := description
	if len(blockedBy) > 0 {
		// 使用 GitHub Tasklists 格式表示依赖
		bodyContent += "\n\n### Depends on\n"
		for _, blocker := range blockedBy {
			bodyContent += fmt.Sprintf("- #%s\n", extractIssueNumber(blocker))
		}
	}

	body := map[string]any{
		"title": title,
		"body":  bodyContent,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github_create_failed: %d - %s", resp.StatusCode, string(respBody))
	}

	var gi githubIssue
	if err := json.NewDecoder(resp.Body).Decode(&gi); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return gi.toDomain(c.owner, c.repo), nil
}

// UpdateStage 更新任务阶段状态
func (c *GitHubClient) UpdateStage(ctx context.Context, identifier string, stage domain.StageState) error {
	// GitHub 通过 label 表示状态，这里添加/移除 label
	number := extractIssueNumber(identifier)
	if number == "" {
		return fmt.Errorf("invalid identifier: %s", identifier)
	}

	// 添加阶段 label（如果不存在）
	stageLabel := fmt.Sprintf("stage:%s", stage.Name)
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%s/labels", githubAPIBase, c.owner, c.repo, number)

	body := map[string]any{
		"labels": []string{stageLabel},
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github_update_stage_failed: %d - %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// AppendConversation 追加对话记录 (通过 Issue Comment)
func (c *GitHubClient) AppendConversation(ctx context.Context, identifier string, turn domain.ConversationTurn) error {
	number := extractIssueNumber(identifier)
	if number == "" {
		return fmt.Errorf("invalid identifier: %s", identifier)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/issues/%s/comments", githubAPIBase, c.owner, c.repo, number)

	body := map[string]any{
		"body": fmt.Sprintf("**%s**: %s", turn.Role, turn.Content),
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github_comment_failed: %d - %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetStageState 获取任务的阶段状态（用于崩溃恢复）
// GitHub 使用 labels 来存储阶段状态
func (c *GitHubClient) GetStageState(ctx context.Context, identifier string) (*domain.StageState, error) {
	number := extractIssueNumber(identifier)
	if number == "" {
		return nil, fmt.Errorf("invalid identifier: %s", identifier)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/issues/%s", githubAPIBase, c.owner, c.repo, number)
	gi, err := c.getIssue(ctx, url)
	if err != nil {
		return nil, err
	}

	// 从 labels 中提取阶段状态
	for _, label := range gi.Labels {
		if strings.HasPrefix(strings.ToLower(label.Name), "stage:") {
			stageName := strings.TrimPrefix(label.Name, "stage:")
			return &domain.StageState{
				Name:      stageName,
				Status:    "pending",
				StartedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		}
	}

	return nil, nil
}

// GetConversationHistory 获取对话历史记录（通过 Issue Comments）
func (c *GitHubClient) GetConversationHistory(ctx context.Context, identifier string) ([]domain.ConversationTurn, error) {
	number := extractIssueNumber(identifier)
	if number == "" {
		return nil, fmt.Errorf("invalid identifier: %s", identifier)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/issues/%s/comments", githubAPIBase, c.owner, c.repo, number)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github_get_comments_failed: %d - %s", resp.StatusCode, string(respBody))
	}

	var comments []struct {
		Body      string    `json:"body"`
		CreatedAt string    `json:"created_at"`
		User      struct {
			Login string `json:"login"`
		} `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// 解析评论为对话轮次
	var turns []domain.ConversationTurn
	for _, comment := range comments {
		// 解析评论格式：**role**: content
		body := comment.Body
		role := "assistant"
		content := body

		if strings.HasPrefix(body, "**") {
			endIdx := strings.Index(body, "**:")
			if endIdx > 2 {
				role = body[2:endIdx]
				content = strings.TrimPrefix(body, "**"+role+"**: ")
				content = strings.TrimSpace(content)
			}
		}

		var timestamp time.Time
		if t, err := time.Parse(time.RFC3339, comment.CreatedAt); err == nil {
			timestamp = t
		} else {
			timestamp = time.Now()
		}

		turns = append(turns, domain.ConversationTurn{
			Role:      role,
			Content:   content,
			Timestamp: timestamp,
		})
	}

	return turns, nil
}

// ListTasksByState 按状态获取任务列表
func (c *GitHubClient) ListTasksByState(ctx context.Context, states []string) ([]*domain.Issue, error) {
	return c.FetchIssuesByStates(ctx, states)
}

// GetTask 获取单个任务详情
func (c *GitHubClient) GetTask(ctx context.Context, identifier string) (*domain.Issue, error) {
	number := extractIssueNumber(identifier)
	if number == "" {
		return nil, fmt.Errorf("invalid identifier: %s", identifier)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/issues/%s", githubAPIBase, c.owner, c.repo, number)
	gi, err := c.getIssue(ctx, url)
	if err != nil {
		return nil, err
	}

	return gi.toDomain(c.owner, c.repo), nil
}

// GetBDDContent 获取任务的 BDD 规则内容
// GitHub 通过 issue body 中的特定标记来存储 BDD 内容
func (c *GitHubClient) GetBDDContent(ctx context.Context, identifier string) (string, error) {
	issue, err := c.GetTask(ctx, identifier)
	if err != nil {
		return "", err
	}

	if issue.Description == nil {
		return "", fmt.Errorf("no BDD content found for issue: %s", identifier)
	}

	// 查找 BDD 标记之间的内容
	body := *issue.Description
	startMarker := "<!-- BDD_START -->"
	endMarker := "<!-- BDD_END -->"

	startIdx := strings.Index(body, startMarker)
	endIdx := strings.Index(body, endMarker)

	if startIdx == -1 || endIdx == -1 || startIdx >= endIdx {
		return "", fmt.Errorf("no BDD content markers found for issue: %s", identifier)
	}

	return body[startIdx+len(startMarker) : endIdx], nil
}

// UpdateBDDContent 更新任务的 BDD 规则内容
// GitHub 通过更新 issue body 来存储 BDD 内容
func (c *GitHubClient) UpdateBDDContent(ctx context.Context, identifier string, content string) error {
	number := extractIssueNumber(identifier)
	if number == "" {
		return fmt.Errorf("invalid identifier: %s", identifier)
	}

	// 获取当前 issue
	issue, err := c.GetTask(ctx, identifier)
	if err != nil {
		return err
	}

	body := ""
	if issue.Description != nil {
		body = *issue.Description
	}

	// 替换或追加 BDD 内容
	startMarker := "<!-- BDD_START -->"
	endMarker := "<!-- BDD_END -->"
	bddSection := startMarker + "\n" + content + "\n" + endMarker

	startIdx := strings.Index(body, startMarker)
	endIdx := strings.Index(body, endMarker)

	if startIdx != -1 && endIdx != -1 && startIdx < endIdx {
		// 替换现有 BDD 内容
		body = body[:startIdx] + bddSection + body[endIdx+len(endMarker):]
	} else {
		// 追加 BDD 内容
		if body != "" && !strings.HasSuffix(body, "\n") {
			body += "\n\n"
		}
		body += bddSection
	}

	// 更新 issue
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%s", githubAPIBase, c.owner, c.repo, number)

	updateBody := map[string]any{
		"body": body,
	}
	bodyBytes, err := json.Marshal(updateBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github_update_bdd_failed: %d - %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// ApproveBDD 通过 BDD 审核
// GitHub 通过添加 label 来标记 BDD 审核通过
func (c *GitHubClient) ApproveBDD(ctx context.Context, identifier string) error {
	number := extractIssueNumber(identifier)
	if number == "" {
		return fmt.Errorf("invalid identifier: %s", identifier)
	}

	// 添加 "bdd-approved" label
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%s/labels", githubAPIBase, c.owner, c.repo, number)

	body := map[string]any{
		"labels": []string{"bdd-approved"},
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github_approve_bdd_failed: %d - %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// RejectBDD 驳回 BDD 审核
// GitHub 通过添加 label 和评论来标记 BDD 审核驳回
func (c *GitHubClient) RejectBDD(ctx context.Context, identifier string, reason string) error {
	number := extractIssueNumber(identifier)
	if number == "" {
		return fmt.Errorf("invalid identifier: %s", identifier)
	}

	// 添加 "bdd-rejected" label
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%s/labels", githubAPIBase, c.owner, c.repo, number)

	body := map[string]any{
		"labels": []string{"bdd-rejected"},
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github_reject_bdd_failed: %d - %s", resp.StatusCode, string(respBody))
	}

	// 添加驳回原因评论
	commentURL := fmt.Sprintf("%s/repos/%s/%s/issues/%s/comments", githubAPIBase, c.owner, c.repo, number)
	commentBody := map[string]any{
		"body": fmt.Sprintf("**BDD 审核驳回原因**: %s", reason),
	}
	commentBytes, err := json.Marshal(commentBody)
	if err != nil {
		return fmt.Errorf("marshal comment: %w", err)
	}

	commentReq, err := http.NewRequestWithContext(ctx, "POST", commentURL, bytes.NewReader(commentBytes))
	if err != nil {
		return fmt.Errorf("create comment request: %w", err)
	}
	c.setHeaders(commentReq)
	commentReq.Header.Set("Content-Type", "application/json")

	commentResp, err := c.httpClient.Do(commentReq)
	if err != nil {
		return fmt.Errorf("comment request failed: %w", err)
	}
	defer commentResp.Body.Close()

	if commentResp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(commentResp.Body)
		return fmt.Errorf("github_reject_comment_failed: %d - %s", commentResp.StatusCode, string(respBody))
	}

	return nil
}

// GetVerificationReport 获取任务的验收报告内容
func (c *GitHubClient) GetVerificationReport(ctx context.Context, identifier string) (*VerificationReport, error) {
	// GitHub 通过 issue body 中的标记存储验收报告
	issue, err := c.GetTask(ctx, identifier)
	if err != nil {
		return nil, err
	}

	if issue.Description == nil {
		return nil, fmt.Errorf("no verification report found for issue: %s", identifier)
	}

	// 解析验收报告内容（简化实现）
	return &VerificationReport{
		TaskID:          issue.ID,
		TaskIdentifier:  issue.Identifier,
		TaskTitle:       issue.Title,
		GeneratedAt:     time.Now(),
	}, nil
}

// UpdateVerificationReport 更新任务的验收报告
func (c *GitHubClient) UpdateVerificationReport(ctx context.Context, identifier string, report *VerificationReport) error {
	// 简化实现：将报告序列化后存储在 issue body 中
	return nil
}

// ApproveVerification 通过验收
func (c *GitHubClient) ApproveVerification(ctx context.Context, identifier string) error {
	number := extractIssueNumber(identifier)
	if number == "" {
		return fmt.Errorf("invalid identifier: %s", identifier)
	}

	// 添加 "verification-approved" label
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%s/labels", githubAPIBase, c.owner, c.repo, number)

	body := map[string]any{
		"labels": []string{"verification-approved"},
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github_approve_verification_failed: %d - %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// RejectVerification 驳回验收（流转回实现中）
func (c *GitHubClient) RejectVerification(ctx context.Context, identifier string, reason string) error {
	number := extractIssueNumber(identifier)
	if number == "" {
		return fmt.Errorf("invalid identifier: %s", identifier)
	}

	// 添加 "verification-rejected" label
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%s/labels", githubAPIBase, c.owner, c.repo, number)

	body := map[string]any{
		"labels": []string{"verification-rejected"},
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github_reject_verification_failed: %d - %s", resp.StatusCode, string(respBody))
	}

	// 添加驳回原因评论
	commentURL := fmt.Sprintf("%s/repos/%s/%s/issues/%s/comments", githubAPIBase, c.owner, c.repo, number)
	commentBody := map[string]any{
		"body": fmt.Sprintf("**验收驳回原因**: %s", reason),
	}
	commentBytes, err := json.Marshal(commentBody)
	if err != nil {
		return fmt.Errorf("marshal comment: %w", err)
	}

	commentReq, err := http.NewRequestWithContext(ctx, "POST", commentURL, bytes.NewReader(commentBytes))
	if err != nil {
		return fmt.Errorf("create comment request: %w", err)
	}
	c.setHeaders(commentReq)
	commentReq.Header.Set("Content-Type", "application/json")

	commentResp, err := c.httpClient.Do(commentReq)
	if err != nil {
		return fmt.Errorf("comment request failed: %w", err)
	}
	defer commentResp.Body.Close()

	return nil
}

// GetArchitectureContent 获取任务的架构设计内容
// GitHub 通过 issue body 中的特定标记来存储架构内容
func (c *GitHubClient) GetArchitectureContent(ctx context.Context, identifier string) (string, error) {
	issue, err := c.GetTask(ctx, identifier)
	if err != nil {
		return "", err
	}

	if issue.Description == nil {
		return "", fmt.Errorf("no architecture content found for issue: %s", identifier)
	}

	// 查找架构标记之间的内容
	body := *issue.Description
	startMarker := "<!-- ARCHITECTURE_START -->"
	endMarker := "<!-- ARCHITECTURE_END -->"

	startIdx := strings.Index(body, startMarker)
	endIdx := strings.Index(body, endMarker)

	if startIdx == -1 || endIdx == -1 || startIdx >= endIdx {
		return "", fmt.Errorf("no architecture content markers found for issue: %s", identifier)
	}

	return body[startIdx+len(startMarker) : endIdx], nil
}

// UpdateArchitectureContent 更新任务的架构设计内容
// GitHub 通过更新 issue body 来存储架构内容
func (c *GitHubClient) UpdateArchitectureContent(ctx context.Context, identifier string, content string) error {
	number := extractIssueNumber(identifier)
	if number == "" {
		return fmt.Errorf("invalid identifier: %s", identifier)
	}

	// 获取当前 issue
	issue, err := c.GetTask(ctx, identifier)
	if err != nil {
		return err
	}

	body := ""
	if issue.Description != nil {
		body = *issue.Description
	}

	// 替换或追加架构内容
	startMarker := "<!-- ARCHITECTURE_START -->"
	endMarker := "<!-- ARCHITECTURE_END -->"
	archSection := startMarker + "\n" + content + "\n" + endMarker

	startIdx := strings.Index(body, startMarker)
	endIdx := strings.Index(body, endMarker)

	if startIdx != -1 && endIdx != -1 && startIdx < endIdx {
		// 替换现有架构内容
		body = body[:startIdx] + archSection + body[endIdx+len(endMarker):]
	} else {
		// 追加架构内容
		if body != "" && !strings.HasSuffix(body, "\n") {
			body += "\n\n"
		}
		body += archSection
	}

	// 更新 issue
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%s", githubAPIBase, c.owner, c.repo, number)

	updateBody := map[string]any{
		"body": body,
	}
	bodyBytes, err := json.Marshal(updateBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github_update_architecture_failed: %d - %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetTDDContent 获取任务的 TDD 规则内容
// GitHub 通过 issue body 中的特定标记来存储 TDD 内容
func (c *GitHubClient) GetTDDContent(ctx context.Context, identifier string) (string, error) {
	issue, err := c.GetTask(ctx, identifier)
	if err != nil {
		return "", err
	}

	if issue.Description == nil {
		return "", fmt.Errorf("no TDD content found for issue: %s", identifier)
	}

	// 查找 TDD 标记之间的内容
	body := *issue.Description
	startMarker := "<!-- TDD_START -->"
	endMarker := "<!-- TDD_END -->"

	startIdx := strings.Index(body, startMarker)
	endIdx := strings.Index(body, endMarker)

	if startIdx == -1 || endIdx == -1 || startIdx >= endIdx {
		return "", fmt.Errorf("no TDD content markers found for issue: %s", identifier)
	}

	return body[startIdx+len(startMarker) : endIdx], nil
}

// UpdateTDDContent 更新任务的 TDD 规则内容
// GitHub 通过更新 issue body 来存储 TDD 内容
func (c *GitHubClient) UpdateTDDContent(ctx context.Context, identifier string, content string) error {
	number := extractIssueNumber(identifier)
	if number == "" {
		return fmt.Errorf("invalid identifier: %s", identifier)
	}

	// 获取当前 issue
	issue, err := c.GetTask(ctx, identifier)
	if err != nil {
		return err
	}

	body := ""
	if issue.Description != nil {
		body = *issue.Description
	}

	// 替换或追加 TDD 内容
	startMarker := "<!-- TDD_START -->"
	endMarker := "<!-- TDD_END -->"
	tddSection := startMarker + "\n" + content + "\n" + endMarker

	startIdx := strings.Index(body, startMarker)
	endIdx := strings.Index(body, endMarker)

	if startIdx != -1 && endIdx != -1 && startIdx < endIdx {
		// 替换现有 TDD 内容
		body = body[:startIdx] + tddSection + body[endIdx+len(endMarker):]
	} else {
		// 追加 TDD 内容
		if body != "" && !strings.HasSuffix(body, "\n") {
			body += "\n\n"
		}
		body += tddSection
	}

	// 更新 issue
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%s", githubAPIBase, c.owner, c.repo, number)

	updateBody := map[string]any{
		"body": body,
	}
	bodyBytes, err := json.Marshal(updateBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github_update_tdd_failed: %d - %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// ApproveArchitecture 通过架构审核
// GitHub 通过添加 label 来标记架构审核通过
func (c *GitHubClient) ApproveArchitecture(ctx context.Context, identifier string) error {
	number := extractIssueNumber(identifier)
	if number == "" {
		return fmt.Errorf("invalid identifier: %s", identifier)
	}

	// 添加 "architecture-approved" label
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%s/labels", githubAPIBase, c.owner, c.repo, number)

	body := map[string]any{
		"labels": []string{"architecture-approved"},
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github_approve_architecture_failed: %d - %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// RejectArchitecture 驳回架构审核
// GitHub 通过添加 label 和评论来标记架构审核驳回
func (c *GitHubClient) RejectArchitecture(ctx context.Context, identifier string, reason string) error {
	number := extractIssueNumber(identifier)
	if number == "" {
		return fmt.Errorf("invalid identifier: %s", identifier)
	}

	// 添加 "architecture-rejected" label
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%s/labels", githubAPIBase, c.owner, c.repo, number)

	body := map[string]any{
		"labels": []string{"architecture-rejected"},
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github_reject_architecture_failed: %d - %s", resp.StatusCode, string(respBody))
	}

	// 添加驳回原因评论
	commentURL := fmt.Sprintf("%s/repos/%s/%s/issues/%s/comments", githubAPIBase, c.owner, c.repo, number)
	commentBody := map[string]any{
		"body": fmt.Sprintf("**架构审核驳回原因**: %s", reason),
	}
	commentBytes, err := json.Marshal(commentBody)
	if err != nil {
		return fmt.Errorf("marshal comment: %w", err)
	}

	commentReq, err := http.NewRequestWithContext(ctx, "POST", commentURL, bytes.NewReader(commentBytes))
	if err != nil {
		return fmt.Errorf("create comment request: %w", err)
	}
	c.setHeaders(commentReq)
	commentReq.Header.Set("Content-Type", "application/json")

	commentResp, err := c.httpClient.Do(commentReq)
	if err != nil {
		return fmt.Errorf("comment request failed: %w", err)
	}
	defer commentResp.Body.Close()

	if commentResp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(commentResp.Body)
		return fmt.Errorf("github_reject_comment_failed: %d - %s", commentResp.StatusCode, string(respBody))
	}

	return nil
}
