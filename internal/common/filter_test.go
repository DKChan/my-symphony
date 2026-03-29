package common

import (
	"testing"
)

func TestParseFilterState(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TaskFilterState
	}{
		{
			name:     "empty input returns all",
			input:    "",
			expected: []TaskFilterState{FilterAll},
		},
		{
			name:     "single state",
			input:    "backlog",
			expected: []TaskFilterState{FilterBacklog},
		},
		{
			name:     "multiple states",
			input:    "active,review",
			expected: []TaskFilterState{FilterActive, FilterReview},
		},
		{
			name:     "invalid state returns all",
			input:    "invalid",
			expected: []TaskFilterState{FilterAll},
		},
		{
			name:     "case insensitive",
			input:    "ACTIVE",
			expected: []TaskFilterState{FilterActive},
		},
		{
			name:     "whitespace trimming",
			input:    " backlog , active ",
			expected: []TaskFilterState{FilterBacklog, FilterActive},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseFilterState(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("ParseFilterState(%s) returned %d states, expected %d", tt.input, len(result), len(tt.expected))
			}
			for i, state := range result {
				if state != tt.expected[i] {
					t.Errorf("ParseFilterState(%s)[%d] = %s, expected %s", tt.input, i, state, tt.expected[i])
				}
			}
		})
	}
}

func TestFilterStateToTrackerStates(t *testing.T) {
	tests := []struct {
		name     string
		filter   TaskFilterState
		expected []string
	}{
		{
			name:     "all returns nil",
			filter:   FilterAll,
			expected: nil,
		},
		{
			name:     "backlog returns todo states",
			filter:   FilterBacklog,
			expected: []string{"Todo", "Backlog", "待开始"},
		},
		{
			name:     "active returns in progress states",
			filter:   FilterActive,
			expected: []string{"In Progress", "clarification", "implementation", "进行中"},
		},
		{
			name:     "completed returns done states",
			filter:   FilterCompleted,
			expected: []string{"Done", "Closed", "Completed", "完成"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterStateToTrackerStates(tt.filter)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("FilterStateToTrackerStates(%s) expected nil, got %v", tt.filter, result)
				}
				return
			}
			if len(result) != len(tt.expected) {
				t.Errorf("FilterStateToTrackerStates(%s) returned %d states, expected %d", tt.filter, len(result), len(tt.expected))
			}
		})
	}
}

func TestMergeFilterStates(t *testing.T) {
	tests := []struct {
		name     string
		filters  []TaskFilterState
		expected int // expected number of unique states
	}{
		{
			name:     "empty returns nil",
			filters:  []TaskFilterState{},
			expected: 0,
		},
		{
			name:     "all returns nil",
			filters:  []TaskFilterState{FilterAll},
			expected: 0,
		},
		{
			name:     "single filter",
			filters:  []TaskFilterState{FilterBacklog},
			expected: 3, // Todo, Backlog, 待开始
		},
		{
			name:     "multiple filters",
			filters:  []TaskFilterState{FilterActive, FilterReview},
			expected: 9, // active: 4 (In Progress, clarification, implementation, 进行中) + review: 5 (bdd_review, architecture_review, verification, Review, 待审核)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeFilterStates(tt.filters)
			if tt.expected == 0 {
				if result != nil {
					t.Errorf("MergeFilterStates(%v) expected nil, got %v", tt.filters, result)
				}
				return
			}
			if len(result) != tt.expected {
				t.Errorf("MergeFilterStates(%v) returned %d states, expected %d", tt.filters, len(result), tt.expected)
			}
		})
	}
}

func TestTaskFilterLabel(t *testing.T) {
	tests := []struct {
		filter   TaskFilterState
		expected string
	}{
		{FilterAll, "全部"},
		{FilterBacklog, "待开始"},
		{FilterActive, "进行中"},
		{FilterReview, "待审核"},
		{FilterNeedsAttention, "待人工处理"},
		{FilterCompleted, "完成"},
		{FilterCancelled, "已取消"},
	}

	for _, tt := range tests {
		t.Run(string(tt.filter), func(t *testing.T) {
			label := TaskFilterLabel[tt.filter]
			if label != tt.expected {
				t.Errorf("TaskFilterLabel[%s] = %s, expected %s", tt.filter, label, tt.expected)
			}
		})
	}
}

func TestAllFilterStates(t *testing.T) {
	states := AllFilterStates()
	if len(states) != 7 {
		t.Errorf("AllFilterStates() returned %d states, expected 7", len(states))
	}
}