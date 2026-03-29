// Package errors 测试
package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSymphonyError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *SymphonyError
		expected string
	}{
		{
			name: "basic error",
			err:  NewError("config", "not_found", "配置文件不存在"),
			expected: "config.not_found: 配置文件不存在",
		},
		{
			name: "error with cause",
			err:  WrapError("config", "parse_failed", "配置解析失败", errors.New("yaml error")),
			expected: "config.parse_failed: 配置解析失败 (yaml error)",
		},
		{
			name: "error without description",
			err:  &SymphonyError{Module: "tracker", Type: "invalid"},
			expected: "tracker.invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestSymphonyError_Code(t *testing.T) {
	err := NewError("config", "invalid", "配置验证失败")
	assert.Equal(t, "config.invalid", err.Code())
}

func TestSymphonyError_Unwrap(t *testing.T) {
	cause := errors.New("original error")
	err := WrapError("config", "parse_failed", "解析失败", cause)
	assert.Equal(t, cause, err.Unwrap())
	assert.True(t, errors.Is(err, cause))
}

func TestNewError(t *testing.T) {
	err := NewError("tracker", "not_found", "Tracker 未找到")
	assert.NotNil(t, err)
	assert.Equal(t, "tracker", err.Module)
	assert.Equal(t, "not_found", err.Type)
	assert.Equal(t, "Tracker 未找到", err.Description)
	assert.Nil(t, err.Cause)
}

func TestWrapError(t *testing.T) {
	cause := errors.New("connection refused")
	err := WrapError("tracker", "unavailable", "Tracker 不可用", cause)
	assert.NotNil(t, err)
	assert.Equal(t, "tracker", err.Module)
	assert.Equal(t, "unavailable", err.Type)
	assert.Equal(t, "Tracker 不可用", err.Description)
	assert.Equal(t, cause, err.Cause)
}

func TestPredefinedErrors(t *testing.T) {
	// 验证预定义错误常量
	assert.Equal(t, "config.not_found", ErrConfigNotFound.Code())
	assert.Equal(t, "config.invalid", ErrConfigInvalid.Code())
	assert.Equal(t, "tracker.not_found", ErrTrackerNotFound.Code())
	assert.Equal(t, "tracker.unavailable", ErrTrackerUnavailable.Code())
	assert.Equal(t, "agent.not_found", ErrAgentNotFound.Code())
	assert.Equal(t, "prompt.not_found", ErrPromptNotFound.Code())
}

func TestIsNotFoundError(t *testing.T) {
	// 正例
	assert.True(t, IsNotFoundError(ErrConfigNotFound))
	assert.True(t, IsNotFoundError(ErrTrackerNotFound))
	assert.True(t, IsNotFoundError(ErrAgentNotFound))
	assert.True(t, IsNotFoundError(ErrPromptNotFound))

	// 反例
	assert.False(t, IsNotFoundError(ErrConfigInvalid))
	assert.False(t, IsNotFoundError(ErrTrackerUnavailable))
	assert.False(t, IsNotFoundError(errors.New("some error")))
}

func TestIsConfigError(t *testing.T) {
	// 正例
	assert.True(t, IsConfigError(ErrConfigNotFound))
	assert.True(t, IsConfigError(ErrConfigInvalid))
	assert.True(t, IsConfigError(NewError("config", "test", "test")))

	// 反例
	assert.False(t, IsConfigError(ErrTrackerNotFound))
	assert.False(t, IsConfigError(ErrAgentNotFound))
	assert.False(t, IsConfigError(errors.New("some error")))
}

func TestIsValidationError(t *testing.T) {
	// 正例
	assert.True(t, IsValidationError(ErrConfigInvalid))
	assert.True(t, IsValidationError(ErrTrackerInvalid))
	assert.True(t, IsValidationError(ErrPromptInvalid))

	// 反例
	assert.False(t, IsValidationError(ErrConfigNotFound))
	assert.False(t, IsValidationError(ErrTrackerNotFound))
	assert.False(t, IsValidationError(errors.New("some error")))
}