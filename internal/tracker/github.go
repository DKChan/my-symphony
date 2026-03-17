// Package tracker - GitHub Issues REST API 适配器
// 使用 GitHub REST API v3（api.github.com）实现 Tracker 接口
package tracker

import (
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
// GitHub 没有 Linear 的"状态"概念，使用 label 映射状态
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
