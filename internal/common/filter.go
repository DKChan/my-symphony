// Package common 提供通用工具和类型定义
package common

import (
	"strings"
)

// TaskFilterState 任务筛选状态枚举
type TaskFilterState string

const (
	FilterAll            TaskFilterState = "all"
	FilterBacklog        TaskFilterState = "backlog"
	FilterActive         TaskFilterState = "active"
	FilterReview         TaskFilterState = "review"
	FilterNeedsAttention TaskFilterState = "needs_attention"
	FilterCompleted      TaskFilterState = "completed"
	FilterCancelled      TaskFilterState = "cancelled"
)

// TaskFilterLabel 筛选状态的中文标签
var TaskFilterLabel = map[TaskFilterState]string{
	FilterAll:            "全部",
	FilterBacklog:        "待开始",
	FilterActive:         "进行中",
	FilterReview:         "待审核",
	FilterNeedsAttention: "待人工处理",
	FilterCompleted:      "完成",
	FilterCancelled:      "已取消",
}

// FilterStateToTrackerStates 将筛选状态映射到跟踪器原始状态
// 返回需要查询的跟踪器状态列表
func FilterStateToTrackerStates(filter TaskFilterState) []string {
	switch filter {
	case FilterAll:
		return nil // nil 表示查询所有状态
	case FilterBacklog:
		return []string{"Todo", "Backlog", "待开始"}
	case FilterActive:
		return []string{"In Progress", "clarification", "implementation", "进行中"}
	case FilterReview:
		return []string{"bdd_review", "architecture_review", "verification", "Review", "待审核"}
	case FilterNeedsAttention:
		return []string{"Blocked", "needs_attention", "待人工处理"}
	case FilterCompleted:
		return []string{"Done", "Closed", "Completed", "完成"}
	case FilterCancelled:
		return []string{"Cancelled", "Canceled", "已取消"}
	default:
		return nil
	}
}

// ParseFilterState 解析筛选状态参数
// 支持单状态和多状态（逗号分隔）
func ParseFilterState(stateParam string) []TaskFilterState {
	if stateParam == "" {
		return []TaskFilterState{FilterAll}
	}

	parts := strings.Split(stateParam, ",")
	result := make([]TaskFilterState, 0, len(parts))
	for _, part := range parts {
		normalized := strings.TrimSpace(strings.ToLower(part))
		switch normalized {
		case "all":
			result = append(result, FilterAll)
		case "backlog":
			result = append(result, FilterBacklog)
		case "active":
			result = append(result, FilterActive)
		case "review":
			result = append(result, FilterReview)
		case "needs_attention":
			result = append(result, FilterNeedsAttention)
		case "completed":
			result = append(result, FilterCompleted)
		case "cancelled":
			result = append(result, FilterCancelled)
		}
	}

	if len(result) == 0 {
		return []TaskFilterState{FilterAll}
	}
	return result
}

// MergeFilterStates 合并多个筛选状态的跟踪器状态
func MergeFilterStates(filters []TaskFilterState) []string {
	if len(filters) == 0 {
		return nil
	}

	// 检查是否包含 "all"
	for _, f := range filters {
		if f == FilterAll {
			return nil
		}
	}

	// 合并所有状态
	result := make([]string, 0)
	seen := make(map[string]bool)

	for _, f := range filters {
		states := FilterStateToTrackerStates(f)
		for _, s := range states {
			if !seen[s] {
				seen[s] = true
				result = append(result, s)
			}
		}
	}

	return result
}

// TaskPayload 任务载荷，用于 API 响应
type TaskPayload struct {
	ID          string   `json:"id"`
	Identifier  string   `json:"identifier"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	State       string   `json:"state"`
	Priority    *int     `json:"priority,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	URL         *string  `json:"url,omitempty"`
	CreatedAt   *string  `json:"created_at,omitempty"`
	UpdatedAt   *string  `json:"updated_at,omitempty"`
}

// TasksPayload 任务列表载荷
type TasksPayload struct {
	Filter      string         `json:"filter"`
	FilterLabel string         `json:"filter_label"`
	TotalCount  int            `json:"total_count"`
	Tasks       []TaskPayload  `json:"tasks"`
}

// AllFilterStates 返回所有可用的筛选状态
func AllFilterStates() []TaskFilterState {
	return []TaskFilterState{
		FilterAll,
		FilterBacklog,
		FilterActive,
		FilterReview,
		FilterNeedsAttention,
		FilterCompleted,
		FilterCancelled,
	}
}