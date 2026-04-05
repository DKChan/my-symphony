// Package common - SSE 广播器和辅助函数测试
package common

import (
	"sync"
	"testing"
	"time"

	"github.com/dministrator/symphony/internal/domain"
)

// TestNewSSEBroadcaster 测试创建 SSE 广播器
func TestNewSSEBroadcaster(t *testing.T) {
	broadcaster := NewSSEBroadcaster()
	if broadcaster == nil {
		t.Fatal("expected non-nil broadcaster")
	}

	if broadcaster.clients == nil {
		t.Error("expected non-nil clients map")
	}

	if broadcaster.lastPayload != nil {
		t.Error("expected nil lastPayload initially")
	}
}

// TestSSEBroadcasterSubscribe 测试订阅功能
func TestSSEBroadcasterSubscribe(t *testing.T) {
	broadcaster := NewSSEBroadcaster()

	ch := broadcaster.Subscribe()
	if ch == nil {
		t.Fatal("expected non-nil channel")
	}

	// 验证 channel 缓冲区大小
	if cap(ch) != 10 {
		t.Errorf("expected channel buffer size 10, got %d", cap(ch))
	}

	// 验证客户端已注册
	broadcaster.mu.RLock()
	if _, ok := broadcaster.clients[ch]; !ok {
		t.Error("expected channel to be registered")
	}
	broadcaster.mu.RUnlock()

	// 清理
	broadcaster.Unsubscribe(ch)
}

// TestSSEBroadcasterUnsubscribe 测试取消订阅功能
func TestSSEBroadcasterUnsubscribe(t *testing.T) {
	broadcaster := NewSSEBroadcaster()

	ch := broadcaster.Subscribe()

	// 取消订阅
	broadcaster.Unsubscribe(ch)

	// 验证客户端已移除
	broadcaster.mu.RLock()
	if _, ok := broadcaster.clients[ch]; ok {
		t.Error("expected channel to be unregistered")
	}
	broadcaster.mu.RUnlock()

	// 验证 channel 已关闭
	select {
	case <-ch:
		// channel 应该已关闭，可以读取
	default:
		t.Error("expected channel to be closed")
	}
}

// TestSSEBroadcasterBroadcast 测试广播功能
func TestSSEBroadcasterBroadcast(t *testing.T) {
	broadcaster := NewSSEBroadcaster()

	// 订阅多个客户端
	client1 := broadcaster.Subscribe()
	client2 := broadcaster.Subscribe()
	client3 := broadcaster.Subscribe()

	payload := &StatePayload{
		GeneratedAt: "2024-01-01T00:00:00Z",
		Counts: StateCounts{
			Running:  2,
			Retrying: 1,
		},
		Running: []RunningEntryPayload{},
		Retrying: []RetryEntryPayload{},
		CodexTotals: domain.CodexTotals{
			SecondsRunning: 3600,
		},
	}

	// 广播事件
	broadcaster.Broadcast("test", payload)

	// 验证所有客户端都收到事件
	timeout := time.After(100 * time.Millisecond)
	received := 0

	for i := 0; i < 3; i++ {
		select {
		case event := <-client1:
			if event.Event != "test" {
				t.Errorf("expected event type 'test', got %s", event.Event)
			}
			received++
		case event := <-client2:
			if event.Event != "test" {
				t.Errorf("expected event type 'test', got %s", event.Event)
			}
			received++
		case event := <-client3:
			if event.Event != "test" {
				t.Errorf("expected event type 'test', got %s", event.Event)
			}
			received++
		case <-timeout:
			t.Error("expected to receive event within timeout")
		}
	}

	if received != 3 {
		t.Errorf("expected to receive 3 events, got %d", received)
	}

	// 清理
	broadcaster.Unsubscribe(client1)
	broadcaster.Unsubscribe(client2)
	broadcaster.Unsubscribe(client3)
}

// TestSSEBroadcasterBroadcastConcurrent 测试并发广播
func TestSSEBroadcasterBroadcastConcurrent(t *testing.T) {
	broadcaster := NewSSEBroadcaster()

	// 订阅多个客户端
	clients := make([]chan *SSEEvent, 10)
	for i := range clients {
		clients[i] = broadcaster.Subscribe()
	}

	payload := &StatePayload{
		GeneratedAt: "2024-01-01T00:00:00Z",
		Counts: StateCounts{
			Running:  1,
			Retrying: 0,
		},
	}

	var wg sync.WaitGroup
	wg.Add(10)

	// 并发广播
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			broadcaster.Broadcast("concurrent", payload)
		}()
	}

	wg.Wait()

	// 验证每个客户端都收到至少一个事件
	for _, ch := range clients {
		select {
		case event := <-ch:
			if event.Event != "concurrent" {
				t.Errorf("expected event type 'concurrent', got %s", event.Event)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("expected to receive event within timeout")
		}
	}

	// 清理
	for _, ch := range clients {
		broadcaster.Unsubscribe(ch)
	}
}

// TestSSEBroadcasterGetLastPayload 测试获取最后载荷
func TestSSEBroadcasterGetLastPayload(t *testing.T) {
	broadcaster := NewSSEBroadcaster()

	// 初始应该为 nil
	payload := broadcaster.GetLastPayload()
	if payload != nil {
		t.Error("expected nil payload initially")
	}

	// 广播事件
	expectedPayload := &StatePayload{
		GeneratedAt: "2024-01-01T00:00:00Z",
		Counts: StateCounts{
			Running:  1,
			Retrying: 0,
		},
	}
	broadcaster.Broadcast("test", expectedPayload)

	// 获取最后载荷
	payload = broadcaster.GetLastPayload()
	if payload == nil {
		t.Fatal("expected non-nil payload after broadcast")
	}

	if payload.GeneratedAt != expectedPayload.GeneratedAt {
		t.Errorf("expected generated_at %s, got %s", expectedPayload.GeneratedAt, payload.GeneratedAt)
	}
}

// TestSSEBroadcasterGetMu 测试获取互斥锁
func TestSSEBroadcasterGetMu(t *testing.T) {
	broadcaster := NewSSEBroadcaster()

	mu := broadcaster.GetMu()
	if mu == nil {
		t.Fatal("expected non-nil mutex")
	}

	// 测试锁功能
	mu.RLock()
	clientCount := len(broadcaster.clients)
	mu.RUnlock()

	if clientCount < 0 {
		t.Error("expected non-negative client count")
	}

	// 不要在 RLock 中调用 Subscribe，否则会死锁
	ch := broadcaster.Subscribe()
	broadcaster.Unsubscribe(ch)
}

// TestSSEBroadcasterMultipleSubscriptions 测试多次订阅
func TestSSEBroadcasterMultipleSubscriptions(t *testing.T) {
	broadcaster := NewSSEBroadcaster()

	// 同一个客户端多次订阅
	ch1 := broadcaster.Subscribe()
	ch2 := broadcaster.Subscribe()
	ch3 := broadcaster.Subscribe()

	// 验证所有 channel 都已注册
	broadcaster.mu.RLock()
	clientCount := len(broadcaster.clients)
	broadcaster.mu.RUnlock()

	if clientCount != 3 {
		t.Errorf("expected 3 clients, got %d", clientCount)
	}

	// 清理
	broadcaster.Unsubscribe(ch1)
	broadcaster.Unsubscribe(ch2)
	broadcaster.Unsubscribe(ch3)
}

// TestSSEBroadcasterBroadcastWithNilPayload 测试广播空载荷
func TestSSEBroadcasterBroadcastWithNilPayload(t *testing.T) {
	broadcaster := NewSSEBroadcaster()

	ch := broadcaster.Subscribe()

	// 广播空载荷
	broadcaster.Broadcast("test", nil)

	// 验证收到事件（数据为空）
	timeout := time.After(100 * time.Millisecond)
	select {
	case event := <-ch:
		if event.Event != "test" {
			t.Errorf("expected event type 'test', got %s", event.Event)
		}
		if event.Data != "null" {
			t.Errorf("expected data 'null', got %s", event.Data)
		}
	case <-timeout:
		t.Error("expected to receive event within timeout")
	}

	broadcaster.Unsubscribe(ch)
}

// TestTotalRuntimeSeconds 测试计算总运行时间
func TestTotalRuntimeSeconds(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		state    *domain.OrchestratorState
		expected int
	}{
		{
			name: "no running sessions",
			state: &domain.OrchestratorState{
				Running: map[string]*domain.RunningEntry{},
				CodexTotals: &domain.CodexTotals{
					SecondsRunning: 3600,
				},
			},
			expected: 3600,
		},
		{
			name: "one running session",
			state: &domain.OrchestratorState{
				Running: map[string]*domain.RunningEntry{
					"test-1": {
						StartedAt: now.Add(-30 * time.Second),
					},
				},
				CodexTotals: &domain.CodexTotals{
					SecondsRunning: 3600,
				},
			},
			expected: 3630,
		},
		{
			name: "multiple running sessions",
			state: &domain.OrchestratorState{
				Running: map[string]*domain.RunningEntry{
					"test-1": {
						StartedAt: now.Add(-30 * time.Second),
					},
					"test-2": {
						StartedAt: now.Add(-60 * time.Second),
					},
				},
				CodexTotals: &domain.CodexTotals{
					SecondsRunning: 3600,
				},
			},
			expected: 3690,
		},
		{
			name: "zero seconds running",
			state: &domain.OrchestratorState{
				Running: map[string]*domain.RunningEntry{
					"test-1": {
						StartedAt: now,
					},
				},
				CodexTotals: &domain.CodexTotals{
					SecondsRunning: 0,
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TotalRuntimeSeconds(tt.state, now)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// TestFormatRuntimeSeconds 测试格式化运行时间
func TestFormatRuntimeSeconds(t *testing.T) {
	tests := []struct {
		name     string
		seconds  int
		expected string
	}{
		{
			name:     "zero seconds",
			seconds:  0,
			expected: "0m 0s",
		},
		{
			name:     "less than a minute",
			seconds:  30,
			expected: "0m 30s",
		},
		{
			name:     "one minute",
			seconds:  60,
			expected: "1m 0s",
		},
		{
			name:     "one minute and thirty seconds",
			seconds:  90,
			expected: "1m 30s",
		},
		{
			name:     "hour",
			seconds:  3600,
			expected: "60m 0s",
		},
		{
			name:     "two hours and thirty seconds",
			seconds:  7230,
			expected: "120m 30s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatRuntimeSeconds(tt.seconds)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestFormatRuntimeAndTurns 测试格式化运行时间和轮次
func TestFormatRuntimeAndTurns(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		startedAt  time.Time
		turnCount  int
		now        time.Time
		expected   string
	}{
		{
			name:      "no turns",
			startedAt: now.Add(-30 * time.Second),
			turnCount: 0,
			now:       now,
			expected:  "0m 30s",
		},
		{
			name:      "with turns",
			startedAt: now.Add(-30 * time.Second),
			turnCount: 5,
			now:       now,
			expected:  "0m 30s / 5",
		},
		{
			name:      "one minute with turns",
			startedAt: now.Add(-60 * time.Second),
			turnCount: 10,
			now:       now,
			expected:  "1m 0s / 10",
		},
		{
			name:      "zero time with turns",
			startedAt: now,
			turnCount: 1,
			now:       now,
			expected:  "0m 0s / 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatRuntimeAndTurns(tt.startedAt, tt.turnCount, tt.now)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestStateBadgeClass 测试状态徽章样式
func TestStateBadgeClass(t *testing.T) {
	tests := []struct {
		name     string
		state    string
		expected string
	}{
		{
			name:     "In Progress - active",
			state:    "In Progress",
			expected: "state-badge state-badge-active",
		},
		{
			name:     "Running - active",
			state:    "Running",
			expected: "state-badge state-badge-active",
		},
		{
			name:     "Active - active",
			state:    "Active",
			expected: "state-badge state-badge-active",
		},
		{
			name:     "in progress - lowercase active",
			state:    "in progress",
			expected: "state-badge state-badge-active",
		},
		{
			name:     "Blocked - danger",
			state:    "Blocked",
			expected: "state-badge state-badge-danger",
		},
		{
			name:     "Error - danger",
			state:    "Error",
			expected: "state-badge state-badge-danger",
		},
		{
			name:     "Failed - danger",
			state:    "Failed",
			expected: "state-badge state-badge-danger",
		},
		{
			name:     "Todo - warning",
			state:    "Todo",
			expected: "state-badge state-badge-warning",
		},
		{
			name:     "Queued - warning",
			state:    "Queued",
			expected: "state-badge state-badge-warning",
		},
		{
			name:     "Pending - warning",
			state:    "Pending",
			expected: "state-badge state-badge-warning",
		},
		{
			name:     "Retry - warning",
			state:    "Retry",
			expected: "state-badge state-badge-warning",
		},
		{
			name:     "Done - default",
			state:    "Done",
			expected: "state-badge",
		},
		{
			name:     "Cancelled - default",
			state:    "Cancelled",
			expected: "state-badge",
		},
		{
			name:     "empty state - default",
			state:    "",
			expected: "state-badge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StateBadgeClass(tt.state)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestFormatInt 测试整数格式化
func TestFormatInt(t *testing.T) {
	tests := []struct {
		name     string
		value    int64
		expected string
	}{
		{
			name:     "small number",
			value:    123,
			expected: "123",
		},
		{
			name:     "one thousand",
			value:    1000,
			expected: "1.0K",
		},
		{
			name:     "one thousand five hundred",
			value:    1500,
			expected: "1.5K",
		},
		{
			name:     "ten thousand",
			value:    10000,
			expected: "10.0K",
		},
		{
			name:     "one million",
			value:    1000000,
			expected: "1.0M",
		},
		{
			name:     "one million five hundred",
			value:    1500000,
			expected: "1.5M",
		},
		{
			name:     "ten million",
			value:    10000000,
			expected: "10.0M",
		},
		{
			name:     "zero",
			value:    0,
			expected: "0",
		},
		{
			name:     "negative number",
			value:    -100,
			expected: "-100",
		},
		{
			name:     "999 - just below K",
			value:    999,
			expected: "999",
		},
		{
			name:     "999999 - just below M",
			value:    999999,
			expected: "1000.0K",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatInt(tt.value)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestPrettyValue 测试 JSON 格式化
func TestPrettyValue(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name     string
		value    any
		expected string
		contains string
	}{
		{
			name:     "nil value",
			value:    nil,
			expected: "n/a",
		},
		{
			name:  "string",
			value: "test",
			contains: `"test"`,
		},
		{
			name:  "integer",
			value: 123,
			contains: "123",
		},
		{
			name:  "struct",
			value: TestStruct{Name: "test", Value: 123},
			contains: `"name": "test"`,
		},
		{
			name:  "map",
			value: map[string]int{"a": 1, "b": 2},
			contains: `"a": 1`,
		},
		{
			name:  "slice",
			value: []int{1, 2, 3},
			contains: `1`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PrettyValue(tt.value)
			if tt.expected != "" && result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
			if tt.contains != "" {
				if !contains(result, tt.contains) {
					t.Errorf("expected result to contain %s, got %s", tt.contains, result)
				}
			}
		})
	}
}

// TestEscapeHTML 测试 HTML 转义
func TestEscapeHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ampersand",
			input:    "A & B",
			expected: "A &amp; B",
		},
		{
			name:     "less than",
			input:    "A < B",
			expected: "A &lt; B",
		},
		{
			name:     "greater than",
			input:    "A > B",
			expected: "A &gt; B",
		},
		{
			name:     "double quote",
			input:    `A "B" C`,
			expected: "A &quot;B&quot; C",
		},
		{
			name:     "single quote",
			input:    "A 'B' C",
			expected: "A &#39;B&#39; C",
		},
		{
			name:     "all special characters",
			input:    `&<>"'`,
			expected: "&amp;&lt;&gt;&quot;&#39;",
		},
		{
			name:     "no special characters",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "multiple occurrences",
			input:    "A & B & C",
			expected: "A &amp; B &amp; C",
		},
		{
			name:     "HTML tag",
			input:    "<script>alert('XSS')</script>",
			expected: "&lt;script&gt;alert(&#39;XSS&#39;)&lt;/script&gt;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapeHTML(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestSSEBroadcasterSubscribeUnsubscribeCycle 测试订阅-取消订阅循环
func TestSSEBroadcasterSubscribeUnsubscribeCycle(t *testing.T) {
	broadcaster := NewSSEBroadcaster()

	// 多次订阅和取消订阅
	for i := 0; i < 10; i++ {
		ch := broadcaster.Subscribe()

		broadcaster.mu.RLock()
		clientCount := len(broadcaster.clients)
		broadcaster.mu.RUnlock()

		if clientCount != 1 {
			t.Errorf("iteration %d: expected 1 client, got %d", i, clientCount)
		}

		broadcaster.Unsubscribe(ch)

		broadcaster.mu.RLock()
		clientCount = len(broadcaster.clients)
		broadcaster.mu.RUnlock()

		if clientCount != 0 {
			t.Errorf("iteration %d: expected 0 clients, got %d", i, clientCount)
		}
	}
}

// 辅助函数：检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || len(s) > len(substr) && containsHelper(s[1:], substr)
}

func containsHelper(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestSSEBroadcasterBroadcastTaskUpdate 测试广播任务更新事件
func TestSSEBroadcasterBroadcastTaskUpdate(t *testing.T) {
	broadcaster := NewSSEBroadcaster()

	// 订阅客户端
	client1 := broadcaster.Subscribe()
	client2 := broadcaster.Subscribe()

	// 创建任务更新事件
	evt := &SSEEvent{
		Event: "task_update",
		Data:  `{"type":"task_update","task_id":"TEST-1","old_stage":"backlog","new_stage":"implementation"}`,
	}

	// 广播任务更新事件
	broadcaster.BroadcastTaskUpdate(evt)

	// 验证所有客户端都收到事件
	timeout := time.After(100 * time.Millisecond)
	received := 0

	for i := 0; i < 2; i++ {
		select {
		case event := <-client1:
			if event.Event != "task_update" {
				t.Errorf("expected event type 'task_update', got %s", event.Event)
			}
			received++
		case event := <-client2:
			if event.Event != "task_update" {
				t.Errorf("expected event type 'task_update', got %s", event.Event)
			}
			received++
		case <-timeout:
			t.Error("expected to receive event within timeout")
		}
	}

	if received != 2 {
		t.Errorf("expected to receive 2 events, got %d", received)
	}

	// 清理
	broadcaster.Unsubscribe(client1)
	broadcaster.Unsubscribe(client2)
}

// TestKanbanStages 测试看板阶段配置
func TestKanbanStages(t *testing.T) {
	// 验证阶段数量 (5列: 未开始、规划器、生成器、评估器、已完成)
	if len(KanbanStages) != 5 {
		t.Errorf("expected 5 stages, got %d", len(KanbanStages))
	}

	// 验证阶段顺序
	expectedOrder := []string{
		"backlog",
		"planner",
		"generator",
		"evaluator",
		"done",
	}

	for i, stage := range KanbanStages {
		if stage.ID != expectedOrder[i] {
			t.Errorf("expected stage %d to be %s, got %s", i, expectedOrder[i], stage.ID)
		}
		if stage.Title == "" {
			t.Errorf("expected non-empty title for stage %s", stage.ID)
		}
		if stage.Color == "" {
			t.Errorf("expected non-empty color for stage %s", stage.ID)
		}
	}
}

// TestTaskStageToKanbanColumn 测试任务阶段到看板列的映射
func TestTaskStageToKanbanColumn(t *testing.T) {
	tests := []struct {
		taskStage    string
		expectedCol  string
	}{
		// 规划器阶段
		{"clarification", "planner"},
		{"bdd_review", "planner"},
		{"architecture_review", "planner"},
		// 生成器阶段
		{"implementation", "generator"},
		{"testing", "generator"},
		// 评估器阶段
		{"verification", "evaluator"},
		{"review", "evaluator"},
		// 已完成
		{"completed", "done"},
		{"cancelled", "done"},
		{"needs_attention", "done"},
		// 默认
		{"unknown", "backlog"},
		{"", "backlog"},
	}

	for _, tt := range tests {
		result := TaskStageToKanbanColumn(tt.taskStage)
		if result != tt.expectedCol {
			t.Errorf("TaskStageToKanbanColumn(%q) = %q, want %q", tt.taskStage, result, tt.expectedCol)
		}
	}
}

// TestGetKanbanStageConfig 测试获取看板阶段配置
func TestGetKanbanStageConfig(t *testing.T) {
	tests := []struct {
		name          string
		stageID       string
		expectedTitle string
		expectedColor string
	}{
		{
			name:          "backlog",
			stageID:       "backlog",
			expectedTitle: "未开始",
			expectedColor: "#6b7280",
		},
		{
			name:          "planner",
			stageID:       "planner",
			expectedTitle: "规划器",
			expectedColor: "#ffd43b",
		},
		{
			name:          "generator",
			stageID:       "generator",
			expectedTitle: "生成器",
			expectedColor: "#339af0",
		},
		{
			name:          "evaluator",
			stageID:       "evaluator",
			expectedTitle: "评估器",
			expectedColor: "#da77f2",
		},
		{
			name:          "done",
			stageID:       "done",
			expectedTitle: "已完成",
			expectedColor: "#51cf66",
		},
		{
			name:          "unknown stage - defaults to backlog",
			stageID:       "unknown_stage",
			expectedTitle: "未开始",
			expectedColor: "#6b7280",
		},
		{
			name:          "empty stage - defaults to backlog",
			stageID:       "",
			expectedTitle: "未开始",
			expectedColor: "#6b7280",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := GetKanbanStageConfig(tt.stageID)
			if config.Title != tt.expectedTitle {
				t.Errorf("expected title %s, got %s", tt.expectedTitle, config.Title)
			}
			if config.Color != tt.expectedColor {
				t.Errorf("expected color %s, got %s", tt.expectedColor, config.Color)
			}
		})
	}
}

// TestTaskUpdateEvent 测试任务更新事件结构
func TestTaskUpdateEvent(t *testing.T) {
	event := TaskUpdateEvent{
		Type:      "task_update",
		TaskID:    "TEST-1",
		OldStage:  "backlog",
		NewStage:  "implementation",
		Timestamp: "2024-01-01T00:00:00Z",
		Task: KanbanTaskPayload{
			IssueIdentifier: "TEST-1",
			Title:           "Test Task",
			Stage:           "implementation",
			State:           "In Progress",
		},
	}

	if event.Type != "task_update" {
		t.Errorf("expected type 'task_update', got %s", event.Type)
	}
	if event.TaskID != "TEST-1" {
		t.Errorf("expected task_id 'TEST-1', got %s", event.TaskID)
	}
	if event.OldStage != "backlog" {
		t.Errorf("expected old_stage 'backlog', got %s", event.OldStage)
	}
	if event.NewStage != "implementation" {
		t.Errorf("expected new_stage 'implementation', got %s", event.NewStage)
	}
}

// TestKanbanColumn 测试看板列结构
func TestKanbanColumn(t *testing.T) {
	column := KanbanColumn{
		ID:        "implementation",
		Title:     "实现中",
		Icon:      "<svg>...</svg>",
		Color:     "#22d3ee",
		TaskCount: 5,
		Tasks: []KanbanTaskPayload{
			{
				IssueIdentifier: "TEST-1",
				Title:           "Task 1",
				Stage:           "implementation",
			},
		},
	}

	if column.ID != "implementation" {
		t.Errorf("expected ID 'implementation', got %s", column.ID)
	}
	if column.TaskCount != 5 {
		t.Errorf("expected TaskCount 5, got %d", column.TaskCount)
	}
	if len(column.Tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(column.Tasks))
	}
}

// TestKanbanPayload 测试看板载荷结构
func TestKanbanPayload(t *testing.T) {
	payload := KanbanPayload{
		GeneratedAt: "2024-01-01T00:00:00Z",
		TotalTasks:  10,
		Columns: []KanbanColumn{
			{
				ID:        "backlog",
				Title:     "待开始",
				TaskCount: 3,
			},
			{
				ID:        "implementation",
				Title:     "实现中",
				TaskCount: 7,
			},
		},
	}

	if payload.GeneratedAt != "2024-01-01T00:00:00Z" {
		t.Errorf("expected GeneratedAt '2024-01-01T00:00:00Z', got %s", payload.GeneratedAt)
	}
	if payload.TotalTasks != 10 {
		t.Errorf("expected TotalTasks 10, got %d", payload.TotalTasks)
	}
	if len(payload.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(payload.Columns))
	}
}

// TestKanbanTaskPayload 测试看板任务载荷结构
func TestKanbanTaskPayload(t *testing.T) {
	task := KanbanTaskPayload{
		IssueID:         "issue-1",
		IssueIdentifier: "TEST-1",
		Title:           "Implement Feature",
		State:           "In Progress",
		Stage:           "implementation",
		SessionID:       "sess-abc123",
		TurnCount:       10,
		RuntimeTurns:    "5m 30s / 10",
		Tokens: Tokens{
			InputTokens:  1000,
			OutputTokens: 500,
			TotalTokens:  1500,
		},
		Attempt: 2,
		Error:   "Connection timeout",
		DueAt:   "2024-01-01T00:05:00Z",
	}

	if task.IssueIdentifier != "TEST-1" {
		t.Errorf("expected IssueIdentifier 'TEST-1', got %s", task.IssueIdentifier)
	}
	if task.Stage != "implementation" {
		t.Errorf("expected Stage 'implementation', got %s", task.Stage)
	}
	if task.Tokens.TotalTokens != 1500 {
		t.Errorf("expected TotalTokens 1500, got %d", task.Tokens.TotalTokens)
	}
}
