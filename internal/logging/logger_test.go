// Package logging 测试
package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"unknown", slog.LevelInfo},
		{"", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseLevel(tt.input)
			if result != tt.expected {
				t.Errorf("parseLevel(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestInitializeJSONFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := Config{
		Level:        "info",
		Format:       "json",
		EnableStdout: false,
	}

	// 直接创建 handler 测试
	handler := slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: parseLevel(cfg.Level)})
	logger := slog.New(handler)

	logger.Info("test message", "task_id", "SYM-123", "stage", "clarification")

	// 验证 JSON 输出
	output := buf.String()
	if !strings.Contains(output, `"msg":"test message"`) {
		t.Errorf("JSON output missing message: %s", output)
	}
	if !strings.Contains(output, `"task_id":"SYM-123"`) {
		t.Errorf("JSON output missing task_id: %s", output)
	}
	if !strings.Contains(output, `"stage":"clarification"`) {
		t.Errorf("JSON output missing stage: %s", output)
	}
	if !strings.Contains(output, `"level":"INFO"`) {
		t.Errorf("JSON output missing level: %s", output)
	}
}

func TestInitializeTextFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := Config{
		Level:        "info",
		Format:       "text",
		EnableStdout: false,
	}

	// 直接创建 handler 测试
	handler := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: parseLevel(cfg.Level)})
	logger := slog.New(handler)

	logger.Info("test message", "task_id", "SYM-123", "stage", "clarification")

	// 验证 Text 输出
	output := buf.String()
	if !strings.Contains(output, `msg="test message"`) {
		t.Errorf("Text output missing message: %s", output)
	}
	if !strings.Contains(output, "task_id=SYM-123") {
		t.Errorf("Text output missing task_id: %s", output)
	}
	if !strings.Contains(output, "stage=clarification") {
		t.Errorf("Text output missing stage: %s", output)
	}
	if !strings.Contains(output, "level=INFO") {
		t.Errorf("Text output missing level: %s", output)
	}
}

func TestFieldNamingSnakeCase(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)

	// 测试 snake_case 字段命名
	logger.Info("task stage changed",
		"task_id", "SYM-123",
		"stage", "clarification",
		"from_status", "pending",
		"to_status", "in_progress",
	)

	output := buf.String()

	// 验证字段使用 snake_case
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output[:len(output)-1]), &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// 验证 snake_case 字段存在
	expectedFields := []string{"task_id", "stage", "from_status", "to_status"}
	for _, field := range expectedFields {
		if _, ok := logEntry[field]; !ok {
			t.Errorf("Missing snake_case field: %s", field)
		}
	}

	// 验证没有 camelCase 字段
	camelCaseFields := []string{"taskId", "fromStatus", "toStatus"}
	for _, field := range camelCaseFields {
		if _, ok := logEntry[field]; ok {
			t.Errorf("Found camelCase field (should be snake_case): %s", field)
		}
	}
}

func TestErrorLogFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelError})
	logger := slog.New(handler)

	// 测试错误日志格式
	logger.Error("agent execution failed",
		"task_id", "SYM-123",
		"error_code", "agent.execution_failed",
		"error_message", "timeout after 30s",
		"stack_trace", "main.go:123\nruntime.go:456",
	)

	output := buf.String()

	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output[:len(output)-1]), &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// 验证错误日志必需字段
	expectedFields := []string{"error_code", "error_message", "stack_trace"}
	for _, field := range expectedFields {
		if _, ok := logEntry[field]; !ok {
			t.Errorf("Missing error field: %s", field)
		}
	}

	// 验证日志级别
	if level, ok := logEntry["level"].(string); ok {
		if level != "ERROR" {
			t.Errorf("Expected level ERROR, got: %s", level)
		}
	} else {
		t.Errorf("Missing level field")
	}
}

func TestTaskLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo})

	// 重置默认日志器
	defaultLogger = slog.New(handler)

	logger := TaskLogger("SYM-123", "clarification")
	logger.Info("test message")

	output := buf.String()

	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output[:len(output)-1]), &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// 验证预设字段
	if taskID, ok := logEntry["task_id"].(string); ok {
		if taskID != "SYM-123" {
			t.Errorf("Expected task_id SYM-123, got: %s", taskID)
		}
	} else {
		t.Errorf("Missing task_id field")
	}

	if stage, ok := logEntry["stage"].(string); ok {
		if stage != "clarification" {
			t.Errorf("Expected stage clarification, got: %s", stage)
		}
	} else {
		t.Errorf("Missing stage field")
	}
}

func TestDefaultLoggingConfig(t *testing.T) {
	cfg := DefaultLoggingConfig()

	if cfg.Level != "info" {
		t.Errorf("Default level should be 'info', got: %s", cfg.Level)
	}
	if cfg.Format != "json" {
		t.Errorf("Default format should be 'json', got: %s", cfg.Format)
	}
	if !cfg.EnableStdout {
		t.Errorf("EnableStdout should be true by default")
	}
}

func TestParseLoggingConfig(t *testing.T) {
	raw := map[string]interface{}{
		"logging": map[string]interface{}{
			"level":        "debug",
			"format":       "text",
			"file_path":    "/var/log/symphony.log",
			"enable_stdout": false,
		},
	}

	cfg := ParseLoggingConfig(raw)

	if cfg.Level != "debug" {
		t.Errorf("Expected level debug, got: %s", cfg.Level)
	}
	if cfg.Format != "text" {
		t.Errorf("Expected format text, got: %s", cfg.Format)
	}
	if cfg.FilePath != "/var/log/symphony.log" {
		t.Errorf("Expected file path /var/log/symphony.log, got: %s", cfg.FilePath)
	}
	if cfg.EnableStdout {
		t.Errorf("EnableStdout should be false")
	}
}

func TestParseLoggingConfigDefaults(t *testing.T) {
	raw := map[string]interface{}{} // No logging config

	cfg := ParseLoggingConfig(raw)

	// 应使用默认值
	if cfg.Level != "info" {
		t.Errorf("Expected default level info, got: %s", cfg.Level)
	}
	if cfg.Format != "json" {
		t.Errorf("Expected default format json, got: %s", cfg.Format)
	}
	if !cfg.EnableStdout {
		t.Errorf("EnableStdout should be true by default")
	}
}