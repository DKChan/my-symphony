// Package errors 提供结构化错误管理
package errors

import (
	"fmt"
	"strings"
)

// SymphonyError 结构化错误类型
type SymphonyError struct {
	Module      string // 模块名称 (config, tracker, agent, prompt 等)
	Type        string // 错误类型 (not_found, invalid, unavailable 等)
	Description string // 错误描述
	Cause       error  // 原始错误
}

// Error 实现 error 接口
func (e *SymphonyError) Error() string {
	var sb strings.Builder
	sb.WriteString(e.Module)
	sb.WriteString(".")
	sb.WriteString(e.Type)
	if e.Description != "" {
		sb.WriteString(": ")
		sb.WriteString(e.Description)
	}
	if e.Cause != nil {
		sb.WriteString(" (")
		sb.WriteString(e.Cause.Error())
		sb.WriteString(")")
	}
	return sb.String()
}

// Unwrap 返回原始错误
func (e *SymphonyError) Unwrap() error {
	return e.Cause
}

// Code 返回错误码 (module.type)
func (e *SymphonyError) Code() string {
	return fmt.Sprintf("%s.%s", e.Module, e.Type)
}

// NewError 创建新的结构化错误
func NewError(module, typ, description string) *SymphonyError {
	return &SymphonyError{
		Module:      module,
		Type:        typ,
		Description: description,
	}
}

// WrapError 包装现有错误
func WrapError(module, typ, description string, cause error) *SymphonyError {
	return &SymphonyError{
		Module:      module,
		Type:        typ,
		Description: description,
		Cause:       cause,
	}
}

// 预定义错误常量
var (
	// 配置相关错误
	ErrConfigNotFound     = NewError("config", "not_found", "配置文件不存在")
	ErrConfigInvalid      = NewError("config", "invalid", "配置验证失败")
	ErrConfigParseFailed  = NewError("config", "parse_failed", "配置解析失败")
	ErrConfigMissingField = NewError("config", "missing_field", "缺少必需字段")

	// Tracker 相关错误
	ErrTrackerNotFound     = NewError("tracker", "not_found", "Tracker 未找到")
	ErrTrackerUnavailable  = NewError("tracker", "unavailable", "Tracker 不可用")
	ErrTrackerInvalid      = NewError("tracker", "invalid", "Tracker 配置无效")
	ErrTrackerAuthFailed   = NewError("tracker", "auth_failed", "Tracker 认证失败")

	// Agent 相关错误
	ErrAgentNotFound      = NewError("agent", "not_found", "Agent CLI 未找到")
	ErrAgentUnavailable   = NewError("agent", "unavailable", "Agent 不可用")
	ErrAgentExecutionFail = NewError("agent", "execution_failed", "Agent 执行失败")
	ErrAgentTimeout       = NewError("agent", "timeout", "Agent 执行超时")

	// Prompt 相关错误
	ErrPromptNotFound = NewError("prompt", "not_found", "Prompt 文件不存在")
	ErrPromptInvalid  = NewError("prompt", "invalid", "Prompt 内容无效")

	// 工作空间相关错误
	ErrWorkspaceNotFound  = NewError("workspace", "not_found", "工作空间不存在")
	ErrWorkspaceInvalid   = NewError("workspace", "invalid", "工作空间配置无效")
	ErrWorkspaceCreateFail = NewError("workspace", "create_failed", "创建工作空间失败")

	// 工作流相关错误
	ErrWorkflowNotFound = NewError("workflow", "not_found", "工作流文件不存在")
	ErrWorkflowInvalid  = NewError("workflow", "invalid", "工作流内容无效")
)

// IsNotFoundError 检查是否为"未找到"类型错误
func IsNotFoundError(err error) bool {
	if se, ok := err.(*SymphonyError); ok {
		return se.Type == "not_found"
	}
	return false
}

// IsConfigError 检查是否为配置相关错误
func IsConfigError(err error) bool {
	if se, ok := err.(*SymphonyError); ok {
		return se.Module == "config"
	}
	return false
}

// IsValidationError 检查是否为验证失败错误
func IsValidationError(err error) bool {
	if se, ok := err.(*SymphonyError); ok {
		return se.Type == "invalid" || se.Type == "validation_failed"
	}
	return false
}