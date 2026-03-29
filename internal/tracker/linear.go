// Package tracker 提供问题跟踪器客户端实现
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

// LinearClient Linear API客户端
type LinearClient struct {
	endpoint   string
	apiKey     string
	projectSlug string
	httpClient *http.Client
}

// NewLinearClient 创建新的Linear客户端
func NewLinearClient(endpoint, apiKey, projectSlug string) *LinearClient {
	return &LinearClient{
		endpoint:    endpoint,
		apiKey:      apiKey,
		projectSlug: projectSlug,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GraphQLRequest GraphQL请求
type GraphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

// GraphQLResponse GraphQL响应
type GraphQLResponse struct {
	Data   json.RawMessage `json:"data,omitempty"`
	Errors []GraphQLError  `json:"errors,omitempty"`
}

// GraphQLError GraphQL错误
type GraphQLError struct {
	Message string `json:"message"`
}

// FetchCandidateIssues 获取候选问题
func (c *LinearClient) FetchCandidateIssues(ctx context.Context, activeStates []string) ([]*domain.Issue, error) {
	query := `
		query($filter: IssueFilter) {
			issues(filter: $filter, first: 50) {
				pageInfo {
					hasNextPage
					endCursor
				}
				nodes {
					id
					identifier
					title
					description
					priority
					state { name }
					branchName
					url
					labels { nodes { name } }
					createdAt
					updatedAt
				}
			}
		}
	`

	filter := map[string]any{
		"project": map[string]any{
			"slugId": map[string]any{"eq": c.projectSlug},
		},
		"state": map[string]any{
			"name": map[string]any{"in": activeStates},
		},
	}

	var allIssues []*domain.Issue
	var cursor *string

	for {
		variables := map[string]any{
			"filter": filter,
		}
		if cursor != nil {
			variables["after"] = *cursor
		}

		var resp struct {
			Issues struct {
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
				Nodes []LinearIssue `json:"nodes"`
			} `json:"issues"`
		}

		if err := c.doRequest(ctx, query, variables, &resp); err != nil {
			return nil, err
		}

		for _, li := range resp.Issues.Nodes {
			allIssues = append(allIssues, li.ToDomain())
		}

		if !resp.Issues.PageInfo.HasNextPage {
			break
		}
		cursor = &resp.Issues.PageInfo.EndCursor
	}

	return allIssues, nil
}

// FetchIssuesByStates 按状态获取问题
func (c *LinearClient) FetchIssuesByStates(ctx context.Context, states []string) ([]*domain.Issue, error) {
	if len(states) == 0 {
		return nil, nil
	}

	query := `
		query($filter: IssueFilter) {
			issues(filter: $filter, first: 100) {
				nodes {
					id
					identifier
					title
					state { name }
				}
			}
		}
	`

	filter := map[string]any{
		"project": map[string]any{
			"slugId": map[string]any{"eq": c.projectSlug},
		},
		"state": map[string]any{
			"name": map[string]any{"in": states},
		},
	}

	var resp struct {
		Issues struct {
			Nodes []LinearIssue `json:"nodes"`
		} `json:"issues"`
	}

	if err := c.doRequest(ctx, query, map[string]any{"filter": filter}, &resp); err != nil {
		return nil, err
	}

	issues := make([]*domain.Issue, len(resp.Issues.Nodes))
	for i, li := range resp.Issues.Nodes {
		issues[i] = li.ToDomain()
	}

	return issues, nil
}

// FetchIssueStatesByIDs 按ID获取问题状态
func (c *LinearClient) FetchIssueStatesByIDs(ctx context.Context, ids []string) ([]*domain.Issue, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query := `
		query($ids: [ID!]!) {
			issues(filter: { id: { in: $ids } }) {
				nodes {
					id
					identifier
					title
					description
					priority
					state { name }
					branchName
					url
					labels { nodes { name } }
					createdAt
					updatedAt
				}
			}
		}
	`

	var resp struct {
		Issues struct {
			Nodes []LinearIssue `json:"nodes"`
		} `json:"issues"`
	}

	if err := c.doRequest(ctx, query, map[string]any{"ids": ids}, &resp); err != nil {
		return nil, err
	}

	issues := make([]*domain.Issue, len(resp.Issues.Nodes))
	for i, li := range resp.Issues.Nodes {
		issues[i] = li.ToDomain()
	}

	return issues, nil
}

// doRequest 执行GraphQL请求
func (c *LinearClient) doRequest(ctx context.Context, query string, variables map[string]any, result any) error {
	reqBody := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("linear_api_status: %d - %s", resp.StatusCode, string(body))
	}

	var graphqlResp GraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&graphqlResp); err != nil {
		return fmt.Errorf("linear_unknown_payload: %w", err)
	}

	if len(graphqlResp.Errors) > 0 {
		return fmt.Errorf("linear_graphql_errors: %s", graphqlResp.Errors[0].Message)
	}

	if err := json.Unmarshal(graphqlResp.Data, result); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}

// LinearIssue Linear问题结构
type LinearIssue struct {
	ID          string `json:"id"`
	Identifier  string `json:"identifier"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
	State       struct {
		Name string `json:"name"`
	} `json:"state"`
	BranchName string `json:"branchName"`
	URL        string `json:"url"`
	Labels     struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// ToDomain 转换为领域模型
func (li LinearIssue) ToDomain() *domain.Issue {
	issue := &domain.Issue{
		ID:          li.ID,
		Identifier:  li.Identifier,
		Title:       li.Title,
		State:       li.State.Name,
		BranchName:  stringPtr(li.BranchName),
		URL:         stringPtr(li.URL),
		Labels:      make([]string, 0),
		BlockedBy:   make([]domain.BlockerRef, 0),
	}

	if li.Description != "" {
		issue.Description = stringPtr(li.Description)
	}

	if li.Priority > 0 {
		priority := li.Priority
		issue.Priority = &priority
	}

	// 标准化标签为小写
	for _, label := range li.Labels.Nodes {
		issue.Labels = append(issue.Labels, strings.ToLower(label.Name))
	}

	// 解析时间
	if t, err := time.Parse(time.RFC3339, li.CreatedAt); err == nil {
		issue.CreatedAt = &t
	}
	if t, err := time.Parse(time.RFC3339, li.UpdatedAt); err == nil {
		issue.UpdatedAt = &t
	}

	return issue
}

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// CheckAvailability 检查 Linear API 是否可用
func (c *LinearClient) CheckAvailability() error {
	if c.apiKey == "" {
		return fmt.Errorf("linear_api_key_required: LINEAR_API_KEY not set")
	}
	if c.projectSlug == "" {
		return fmt.Errorf("linear_project_slug_required: project_slug not configured")
	}

	// 执行一个简单的查询来验证连接
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		query {
			me {
				id
			}
		}
	`

	var resp struct {
		Me struct {
			ID string `json:"id"`
		} `json:"me"`
	}

	if err := c.doRequest(ctx, query, nil, &resp); err != nil {
		return fmt.Errorf("linear_connection_failed: %w", err)
	}

	return nil
}

// CreateTask 创建新任务
func (c *LinearClient) CreateTask(ctx context.Context, title, description string) (*domain.Issue, error) {
	query := `
		mutation($input: IssueCreateInput!) {
			issueCreate(input: $input) {
				issue {
					id
					identifier
					title
					description
					state { name }
					url
					createdAt
				}
			}
		}
	`

	variables := map[string]any{
		"input": map[string]any{
			"title":       title,
			"description": description,
			"projectId":   c.projectSlug,
		},
	}

	var resp struct {
		IssueCreate struct {
			Issue LinearIssue `json:"issue"`
		} `json:"issueCreate"`
	}

	if err := c.doRequest(ctx, query, variables, &resp); err != nil {
		return nil, fmt.Errorf("create_issue_failed: %w", err)
	}

	return resp.IssueCreate.Issue.ToDomain(), nil
}

// UpdateStage 更新任务阶段状态
func (c *LinearClient) UpdateStage(ctx context.Context, identifier string, stage domain.StageState) error {
	// Linear 不支持自定义阶段字段，这里只记录日志
	// 未来可以通过 label 或 comment 实现
	return nil
}

// GetStageState 获取任务的阶段状态（用于崩溃恢复）
func (c *LinearClient) GetStageState(ctx context.Context, identifier string) (*domain.StageState, error) {
	// Linear 不支持自定义阶段字段，返回 nil
	// 未来可以通过解析 label 或 comment 实现
	return nil, nil
}

// CreateSubTask 创建子任务（带依赖关系）
func (c *LinearClient) CreateSubTask(ctx context.Context, parentIdentifier string, title, description string, blockedBy []string) (*domain.Issue, error) {
	// Linear 支持父子关系，但这里简化处理
	return c.CreateTask(ctx, title, description)
}

// AppendConversation 追加对话记录
func (c *LinearClient) AppendConversation(ctx context.Context, identifier string, turn domain.ConversationTurn) error {
	// 通过 comment 添加对话记录
	query := `
		mutation($input: CommentCreateInput!) {
			commentCreate(input: $input) {
				comment {
					id
				}
			}
		}
	`

	variables := map[string]any{
		"input": map[string]any{
			"issueId": identifier,
			"body":    fmt.Sprintf("**%s**: %s", turn.Role, turn.Content),
		},
	}

	var resp struct {
		CommentCreate struct {
			Comment struct {
				ID string `json:"id"`
			} `json:"comment"`
		} `json:"commentCreate"`
	}

	return c.doRequest(ctx, query, variables, &resp)
}

// GetConversationHistory 获取对话历史记录
func (c *LinearClient) GetConversationHistory(ctx context.Context, identifier string) ([]domain.ConversationTurn, error) {
	// Linear 通过 comments 存储对话，这里简化返回空列表
	// 未来可以通过查询 comments API 实现
	return nil, nil
}

// ListTasksByState 按状态获取任务列表
func (c *LinearClient) ListTasksByState(ctx context.Context, states []string) ([]*domain.Issue, error) {
	return c.FetchIssuesByStates(ctx, states)
}

// GetTask 获取单个任务详情
func (c *LinearClient) GetTask(ctx context.Context, identifier string) (*domain.Issue, error) {
	query := `
		query($identifier: String!) {
			issue(identifier: $identifier) {
				id
				identifier
				title
				description
				priority
				state { name }
				branchName
				url
				labels { nodes { name } }
				createdAt
				updatedAt
			}
		}
	`

	var resp struct {
		Issue LinearIssue `json:"issue"`
	}

	if err := c.doRequest(ctx, query, map[string]any{"identifier": identifier}, &resp); err != nil {
		return nil, err
	}

	return resp.Issue.ToDomain(), nil
}

// GetBDDContent 获取任务的 BDD 规则内容
// Linear 通过 issue description 或 custom field 存储 BDD 内容
func (c *LinearClient) GetBDDContent(ctx context.Context, identifier string) (string, error) {
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
		// 如果没有标记，返回整个描述作为 BDD 内容
		return body, nil
	}

	return body[startIdx+len(startMarker) : endIdx], nil
}

// UpdateBDDContent 更新任务的 BDD 规则内容
func (c *LinearClient) UpdateBDDContent(ctx context.Context, identifier string, content string) error {
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
	query := `
		mutation($id: String!, $input: IssueUpdateInput!) {
			issueUpdate(id: $id, input: $input) {
				success
			}
		}
	`

	variables := map[string]any{
		"id": identifier,
		"input": map[string]any{
			"description": body,
		},
	}

	var resp struct {
		IssueUpdate struct {
			Success bool `json:"success"`
		} `json:"issueUpdate"`
	}

	if err := c.doRequest(ctx, query, variables, &resp); err != nil {
		return fmt.Errorf("update BDD content failed: %w", err)
	}

	return nil
}

// ApproveBDD 通过 BDD 审核
func (c *LinearClient) ApproveBDD(ctx context.Context, identifier string) error {
	query := `
		mutation($id: String!, $input: IssueUpdateInput!) {
			issueUpdate(id: $id, input: $input) {
				success
			}
		}
	`

	variables := map[string]any{
		"id": identifier,
		"input": map[string]any{
			"labelIds": []string{"bdd-approved"},
		},
	}

	var resp struct {
		IssueUpdate struct {
			Success bool `json:"success"`
		} `json:"issueUpdate"`
	}

	if err := c.doRequest(ctx, query, variables, &resp); err != nil {
		return fmt.Errorf("approve BDD failed: %w", err)
	}

	return nil
}

// RejectBDD 驳回 BDD 审核
func (c *LinearClient) RejectBDD(ctx context.Context, identifier string, reason string) error {
	query := `
		mutation($id: String!, $input: IssueUpdateInput!) {
			issueUpdate(id: $id, input: $input) {
				success
			}
		}
	`

	variables := map[string]any{
		"id": identifier,
		"input": map[string]any{
			"labelIds": []string{"bdd-rejected"},
		},
	}

	var resp struct {
		IssueUpdate struct {
			Success bool `json:"success"`
		} `json:"issueUpdate"`
	}

	if err := c.doRequest(ctx, query, variables, &resp); err != nil {
		return fmt.Errorf("reject BDD failed: %w", err)
	}

	// 添加驳回原因评论
	commentQuery := `
		mutation($input: CommentCreateInput!) {
			commentCreate(input: $input) {
				comment {
					id
				}
			}
		}
	`

	commentVariables := map[string]any{
		"input": map[string]any{
			"issueId": identifier,
			"body":    fmt.Sprintf("**BDD 审核驳回原因**: %s", reason),
		},
	}

	var commentResp struct {
		CommentCreate struct {
			Comment struct {
				ID string `json:"id"`
			} `json:"comment"`
		} `json:"commentCreate"`
	}

	if err := c.doRequest(ctx, commentQuery, commentVariables, &commentResp); err != nil {
		return fmt.Errorf("add reject comment failed: %w", err)
	}

	return nil
}