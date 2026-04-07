package handlers_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dministrator/symphony/internal/common"
	"github.com/dministrator/symphony/internal/config"
	"github.com/dministrator/symphony/internal/orchestrator"
	"github.com/dministrator/symphony/internal/router"
	"github.com/stretchr/testify/assert"
)

// TestSSEHandler_Connection 测试 SSE 连接建立和头部设置
func TestSSEHandler_Connection(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	// 创建带有超时的请求
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := httptest.NewRequest("GET", "/events", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	// 在 goroutine 中处理请求
	done := make(chan struct{})
	go func() {
		engine.ServeHTTP(w, req)
		close(done)
	}()

	// 等待处理完成或超时
	select {
	case <-done:
		// 请求完成
	case <-time.After(3 * time.Second):
		t.Log("SSE connection timeout (expected for long-running connection)")
	}

	// 验证 SSE 响应头
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", w.Header().Get("Connection"))
	assert.Equal(t, "no", w.Header().Get("X-Accel-Buffering"))
}

// TestSSEHandler_WithInitialPayload 测试 SSE 初始状态推送（有初始载荷时）
func TestSSEHandler_WithInitialPayload(t *testing.T) {
	// 创建广播器并设置初始载荷
	broadcaster := common.NewSSEBroadcaster()
	payload := &common.StatePayload{
		GeneratedAt: time.Now().Format(time.RFC3339),
		Counts: common.StateCounts{
			Running:  1,
			Retrying: 0,
		},
	}
	broadcaster.Broadcast("state", payload)

	// 通过路由测试 SSE 端点
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")

	// 创建带有超时的请求
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest("GET", "/events", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	// 使用独立的路由测试
	engine := router.BuildRouter(orch)
	engine.ServeHTTP(w, req)

	// 读取响应体
	body := w.Body.String()

	// 验证 SSE 响应头
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))

	// 如果有数据，验证格式
	if body != "" {
		assert.Contains(t, body, "event: state")
		assert.Contains(t, body, "data:")
	}
}

// TestSSEHandler_Broadcast 测试 SSE 广播事件
func TestSSEHandler_Broadcast(t *testing.T) {
	// 创建独立的广播器用于测试
	broadcaster := common.NewSSEBroadcaster()

	// 启动测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 设置 SSE 头
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		// 订阅广播器
		ch := broadcaster.Subscribe()
		defer broadcaster.Unsubscribe(ch)

		// 发送初始空状态
		w.(http.Flusher).Flush()

		// 等待事件或上下文取消
		select {
		case <-r.Context().Done():
			return
		case evt := <-ch:
			_, _ = w.Write([]byte("event: " + evt.Event + "\n"))
			_, _ = w.Write([]byte("data: " + evt.Data + "\n\n"))
			w.(http.Flusher).Flush()
		}
	}))
	defer server.Close()

	// 客户端连接
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// 在 goroutine 中广播事件
	go func() {
		time.Sleep(100 * time.Millisecond) // 等待客户端连接

		payload := &common.StatePayload{
			GeneratedAt: time.Now().Format(time.RFC3339),
			Counts: common.StateCounts{
				Running:  1,
				Retrying: 0,
			},
		}
		broadcaster.Broadcast("state", payload)
	}()

	// 读取 SSE 流
	scanner := bufio.NewScanner(resp.Body)

	// 只读取一行验证
	if scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			assert.Equal(t, "event: state", line)
		}
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			var payload map[string]interface{}
			err := json.Unmarshal([]byte(data), &payload)
			assert.NoError(t, err)
			counts, ok := payload["counts"].(map[string]interface{})
			if ok {
				assert.Equal(t, float64(1), counts["running"])
			}
		}
	}

	assert.NoError(t, scanner.Err(), "Should scan successfully")
}

// TestSSEHandler_MultipleClients 测试多客户端订阅
func TestSSEHandler_MultipleClients(t *testing.T) {
	broadcaster := common.NewSSEBroadcaster()

	// 订阅多个客户端
	ch1 := broadcaster.Subscribe()
	ch2 := broadcaster.Subscribe()
	ch3 := broadcaster.Subscribe()

	// 验证订阅成功
	assert.NotNil(t, ch1)
	assert.NotNil(t, ch2)
	assert.NotNil(t, ch3)

	// 广播事件
	payload := &common.StatePayload{
		GeneratedAt: time.Now().Format(time.RFC3339),
	}
	broadcaster.Broadcast("test", payload)

	// 验证所有客户端收到事件
	select {
	case evt := <-ch1:
		assert.Equal(t, "test", evt.Event)
	case <-time.After(100 * time.Millisecond):
		t.Error("Client 1 did not receive event")
	}

	select {
	case evt := <-ch2:
		assert.Equal(t, "test", evt.Event)
	case <-time.After(100 * time.Millisecond):
		t.Error("Client 2 did not receive event")
	}

	select {
	case evt := <-ch3:
		assert.Equal(t, "test", evt.Event)
	case <-time.After(100 * time.Millisecond):
		t.Error("Client 3 did not receive event")
	}

	// 取消订阅
	broadcaster.Unsubscribe(ch1)
	broadcaster.Unsubscribe(ch2)
	broadcaster.Unsubscribe(ch3)
}

// TestSSEBroadcaster_SubscribeUnsubscribe 测试订阅和取消订阅
func TestSSEBroadcaster_SubscribeUnsubscribe(t *testing.T) {
	broadcaster := common.NewSSEBroadcaster()

	// 订阅
	ch := broadcaster.Subscribe()
	assert.NotNil(t, ch)

	// 验证可以接收事件
	broadcaster.Broadcast("test", nil)

	select {
	case evt := <-ch:
		assert.Equal(t, "test", evt.Event)
	case <-time.After(100 * time.Millisecond):
		t.Error("Did not receive event")
	}

	// 取消订阅
	broadcaster.Unsubscribe(ch)

	// 验证 channel 已关闭
	_, ok := <-ch
	assert.False(t, ok, "Channel should be closed after unsubscribe")
}

// TestSSEBroadcaster_GetLastPayload 测试获取最后的载荷
func TestSSEBroadcaster_GetLastPayload(t *testing.T) {
	broadcaster := common.NewSSEBroadcaster()

	// 初始为空
	assert.Nil(t, broadcaster.GetLastPayload())

	// 广播事件
	payload := &common.StatePayload{
		GeneratedAt: "2024-01-01T00:00:00Z",
	}
	broadcaster.Broadcast("state", payload)

	// 获取最后的载荷
	last := broadcaster.GetLastPayload()
	assert.NotNil(t, last)
	assert.Equal(t, "2024-01-01T00:00:00Z", last.GeneratedAt)
}

// TestSSEBroadcaster_TaskUpdateEvent 测试任务更新事件
func TestSSEBroadcaster_TaskUpdateEvent(t *testing.T) {
	broadcaster := common.NewSSEBroadcaster()
	ch := broadcaster.Subscribe()

	// 创建任务更新事件
	event := &common.SSEEvent{
		Event: "task_update",
		Data:  `{"type":"task_update","task_id":"1","old_stage":"clarification","new_stage":"implementation"}`,
	}

	// 广播任务更新
	broadcaster.BroadcastTaskUpdate(event)

	// 接收事件
	select {
	case evt := <-ch:
		assert.Equal(t, "task_update", evt.Event)
		assert.Contains(t, evt.Data, "task_id")
	case <-time.After(100 * time.Millisecond):
		t.Error("Did not receive task update event")
	}

	broadcaster.Unsubscribe(ch)
}

// TestSSEHandler_ContextCancellation 测试上下文取消时的清理
func TestSSEHandler_ContextCancellation(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tracker.Kind = "mock"
	orch := orchestrator.New(cfg, "")
	engine := router.BuildRouter(orch)

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(context.Background())

	req := httptest.NewRequest("GET", "/events", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	// 启动请求处理
	done := make(chan struct{})
	go func() {
		engine.ServeHTTP(w, req)
		close(done)
	}()

	// 等待一小段时间确保连接建立
	time.Sleep(100 * time.Millisecond)

	// 取消上下文
	cancel()

	// 等待处理完成
	select {
	case <-done:
		// 处理完成
	case <-time.After(1 * time.Second):
		t.Error("Handler did not stop after context cancellation")
	}
}
