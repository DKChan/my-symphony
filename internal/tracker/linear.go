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