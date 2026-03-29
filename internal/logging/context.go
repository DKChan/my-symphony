// Package logging 提供带上下文的日志记录器
package logging

import (
	"log/slog"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// TaskLogger 创建带任务上下文的日志器
// 预设字段: task_id, stage, timestamp
func TaskLogger(taskID, stage string) *slog.Logger {
	return GetLogger().With(
		"task_id", taskID,
		"stage", stage,
	)
}

// StageLogger 创建带阶段信息的日志器
func StageLogger(taskID, stage string, fromStatus, toStatus string) *slog.Logger {
	return TaskLogger(taskID, stage).With(
		"from_status", fromStatus,
		"to_status", toStatus,
	)
}

// LogStageChange 记录阶段状态变更
func LogStageChange(taskID, stage string, fromStatus, toStatus string) {
	StageLogger(taskID, stage, fromStatus, toStatus).Info("task stage changed")
}

// LogTaskStarted 记录任务开始
func LogTaskStarted(taskID, identifier, stage string) {
	TaskLogger(taskID, stage).Info("task started",
		"identifier", identifier,
		"timestamp", time.Now().Format(time.RFC3339),
	)
}

// LogTaskCompleted 记录任务完成
func LogTaskCompleted(taskID, identifier, stage string, turnCount int) {
	TaskLogger(taskID, stage).Info("task completed",
		"identifier", identifier,
		"turn_count", turnCount,
		"timestamp", time.Now().Format(time.RFC3339),
	)
}

// LogTaskFailed 记录任务失败
func LogTaskFailed(taskID, identifier, stage string, err error) {
	errorCode := "task.failed"
	errorMessage := err.Error()
	stackTrace := captureStackTrace()

	TaskLogger(taskID, stage).Error("task failed",
		"identifier", identifier,
		"error_code", errorCode,
		"error_message", errorMessage,
		"stack_trace", stackTrace,
		"timestamp", time.Now().Format(time.RFC3339),
	)
}

// LogAgentEvent 记录代理事件
func LogAgentEvent(taskID, event string, data interface{}) {
	TaskLogger(taskID, "agent").Info("agent event",
		"event_type", event,
		"timestamp", time.Now().Format(time.RFC3339),
	)
}

// LogAgentError 记录代理错误
func LogAgentError(taskID, identifier string, err error) {
	errorCode := extractErrorCode(err)
	errorMessage := err.Error()
	stackTrace := captureStackTrace()

	TaskLogger(taskID, "agent").Error("agent execution failed",
		"identifier", identifier,
		"error_code", errorCode,
		"error_message", errorMessage,
		"stack_trace", stackTrace,
		"timestamp", time.Now().Format(time.RFC3339),
	)
}

// LogWorkerStarted 记录 worker 开始
func LogWorkerStarted(taskID, identifier string, attempt int) {
	TaskLogger(taskID, "worker").Info("worker started",
		"identifier", identifier,
		"attempt", attempt,
		"timestamp", time.Now().Format(time.RFC3339),
	)
}

// LogWorkerExit 记录 worker 退出
func LogWorkerExit(taskID, identifier string, err error, attempt int) {
	if err != nil {
		errorCode := extractErrorCode(err)
		errorMessage := err.Error()
		stackTrace := captureStackTrace()

		TaskLogger(taskID, "worker").Error("worker exited with error",
			"identifier", identifier,
			"attempt", attempt,
			"error_code", errorCode,
			"error_message", errorMessage,
			"stack_trace", stackTrace,
			"timestamp", time.Now().Format(time.RFC3339),
		)
	} else {
		TaskLogger(taskID, "worker").Info("worker exited successfully",
			"identifier", identifier,
			"attempt", attempt,
			"timestamp", time.Now().Format(time.RFC3339),
		)
	}
}

// LogRetryScheduled 记录重试调度
func LogRetryScheduled(taskID, identifier string, attempt int, delayMs int64) {
	TaskLogger(taskID, "retry").Info("retry scheduled",
		"identifier", identifier,
		"attempt", attempt,
		"delay_ms", delayMs,
		"timestamp", time.Now().Format(time.RFC3339),
	)
}

// LogSessionStalled 记录会话停滞
func LogSessionStalled(taskID, identifier string) {
	TaskLogger(taskID, "session").Warn("session appears stalled",
		"identifier", identifier,
		"timestamp", time.Now().Format(time.RFC3339),
	)
}

// LogTermination 记录终止
func LogTermination(taskID, reason string) {
	TaskLogger(taskID, "termination").Info("terminated",
		"reason", reason,
		"timestamp", time.Now().Format(time.RFC3339),
	)
}

// LogWorkspaceCreated 记录工作空间创建
func LogWorkspaceCreated(taskID, identifier string, path string) {
	TaskLogger(taskID, "workspace").Info("workspace created",
		"identifier", identifier,
		"workspace_path", path,
		"timestamp", time.Now().Format(time.RFC3339),
	)
}

// LogWorkspaceError 记录工作空间错误
func LogWorkspaceError(taskID, identifier string, err error) {
	errorCode := "workspace.error"
	errorMessage := err.Error()
	stackTrace := captureStackTrace()

	TaskLogger(taskID, "workspace").Error("workspace error",
		"identifier", identifier,
		"error_code", errorCode,
		"error_message", errorMessage,
		"stack_trace", stackTrace,
		"timestamp", time.Now().Format(time.RFC3339),
	)
}

// captureStackTrace 捕获堆栈跟踪
func captureStackTrace() string {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])

	var sb strings.Builder
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		sb.WriteString(frame.Function)
		sb.WriteString("\n\t")
		sb.WriteString(frame.File)
		sb.WriteString(":")
		sb.WriteString(strconv.Itoa(frame.Line))
		if !more {
			break
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// extractErrorCode 从错误中提取错误码
func extractErrorCode(err error) string {
	// 检查是否为 SymphonyError 类型
	if err != nil {
		// 尝试从错误字符串中提取错误码格式 (module.type)
		errStr := err.Error()
		if idx := strings.Index(errStr, ":"); idx > 0 {
			prefix := errStr[:idx]
			if strings.Contains(prefix, ".") {
				return prefix
			}
		}
	}
	return "unknown.error"
}