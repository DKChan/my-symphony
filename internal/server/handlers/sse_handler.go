package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dministrator/symphony/internal/common"
	"github.com/gin-gonic/gin"
)

// SSEHandler SSE 服务器发送事件处理器
type SSEHandler struct {
	broadcaster *common.SSEBroadcaster
}

// NewSSEHandler 创建新的 SSE 处理器
func NewSSEHandler(broadcaster *common.SSEBroadcaster) *SSEHandler {
	return &SSEHandler{
		broadcaster: broadcaster,
	}
}

// Handle 处理 SSE 连接请求
func (h *SSEHandler) Handle(c *gin.Context) {
	// 设置 SSE 头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// 订阅广播器
	ch := h.broadcaster.Subscribe()
	defer h.broadcaster.Unsubscribe(ch)

	// 发送初始状态
	payload := h.broadcaster.GetLastPayload()

	if payload != nil {
		data, _ := json.Marshal(payload)
		fmt.Fprintf(c.Writer, "event: state\ndata: %s\n\n", string(data))
		c.Writer.(http.Flusher).Flush()
	}

	// 流式传输
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case evt := <-ch:
			fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", evt.Event, evt.Data)
			c.Writer.(http.Flusher).Flush()
		}
	}
}

// BroadcastTaskUpdate 广播任务状态变更事件
func (h *SSEHandler) BroadcastTaskUpdate(taskID, oldStage, newStage string, task common.KanbanTaskPayload) {
	event := common.TaskUpdateEvent{
		Type:      "task_update",
		TaskID:    taskID,
		OldStage:  oldStage,
		NewStage:  newStage,
		Task:      task,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	evt := &common.SSEEvent{
		Event: "task_update",
		Data:  string(data),
	}

	// 直接广播到所有客户端
	h.broadcaster.BroadcastTaskUpdate(evt)
}
