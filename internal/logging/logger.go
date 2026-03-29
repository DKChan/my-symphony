// Package logging 提供结构化日志系统
package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// Config 日志配置
type Config struct {
	// Level 日志级别: debug, info, warn, error
	Level string `json:"level"`
	// Format 输出格式: json, text
	Format string `json:"format"`
	// FilePath 输出文件路径（可选）
	FilePath string `json:"file_path,omitempty"`
	// EnableStdout 是否输出到标准输出
	EnableStdout bool `json:"enable_stdout"`
}

// DefaultLoggingConfig 返回默认日志配置
func DefaultLoggingConfig() Config {
	return Config{
		Level:        "info",
		Format:       "json",
		EnableStdout: true,
	}
}

// ParseLoggingConfig 从原始配置映射解析日志配置
func ParseLoggingConfig(raw map[string]interface{}) Config {
	cfg := DefaultLoggingConfig()

	if logging, ok := raw["logging"].(map[string]interface{}); ok {
		if level, ok := logging["level"].(string); ok {
			cfg.Level = level
		}
		if format, ok := logging["format"].(string); ok {
			cfg.Format = format
		}
		if filePath, ok := logging["file_path"].(string); ok {
			cfg.FilePath = filePath
		}
		if enableStdout, ok := logging["enable_stdout"].(bool); ok {
			cfg.EnableStdout = enableStdout
		}
	}

	return cfg
}

var (
	defaultLogger *slog.Logger
	once          sync.Once
)

// Initialize 初始化默认日志器
func Initialize(cfg Config) error {
	var err error
	once.Do(func() {
		err = initializeLogger(cfg)
	})
	return err
}

// initializeLogger 初始化日志器
func initializeLogger(cfg Config) error {
	// 解析日志级别
	level := parseLevel(cfg.Level)

	// 创建输出目标
	var writers []io.Writer

	if cfg.EnableStdout {
		writers = append(writers, os.Stdout)
	}

	if cfg.FilePath != "" {
		// 确保目录存在
		dir := filepath.Dir(cfg.FilePath)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
		}

		file, err := os.OpenFile(cfg.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		writers = append(writers, file)
	}

	// 如果没有输出目标，默认使用 stdout
	if len(writers) == 0 {
		writers = append(writers, os.Stdout)
	}

	// 组合输出目标
	var writer io.Writer
	if len(writers) == 1 {
		writer = writers[0]
	} else {
		writer = io.MultiWriter(writers...)
	}

	// 创建 Handler
	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: level}

	if cfg.Format == "text" {
		handler = slog.NewTextHandler(writer, opts)
	} else {
		handler = slog.NewJSONHandler(writer, opts)
	}

	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)

	return nil
}

// parseLevel 解析日志级别字符串
func parseLevel(levelStr string) slog.Level {
	switch levelStr {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// GetLogger 获取默认日志器
func GetLogger() *slog.Logger {
	if defaultLogger == nil {
		// 如果未初始化，使用默认配置
		Initialize(DefaultLoggingConfig())
	}
	return defaultLogger
}

// Debug 记录调试级别日志
func Debug(msg string, args ...any) {
	GetLogger().Debug(msg, args...)
}

// Info 记录信息级别日志
func Info(msg string, args ...any) {
	GetLogger().Info(msg, args...)
}

// Warn 记录警告级别日志
func Warn(msg string, args ...any) {
	GetLogger().Warn(msg, args...)
}

// Error 记录错误级别日志
func Error(msg string, args ...any) {
	GetLogger().Error(msg, args...)
}

// ErrorWithStack 记录带堆栈的错误日志
func ErrorWithStack(msg string, errorCode, errorMessage, stackTrace string, args ...any) {
	fullArgs := []any{
		"error_code", errorCode,
		"error_message", errorMessage,
		"stack_trace", stackTrace,
	}
	fullArgs = append(fullArgs, args...)
	GetLogger().Error(msg, fullArgs...)
}