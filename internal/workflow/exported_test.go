// Package workflow 提供工作流管理功能的导出函数测试
package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetStageDisplayName 测试 GetStageDisplayName 函数
func TestGetStageDisplayName(t *testing.T) {
	tests := []struct {
		stage    StageName
		expected string
	}{
		{StageClarification, "需求澄清"},
		{StageBDDReview, "BDD评审"},
		{StageArchitectureReview, "架构评审"},
		{StageImplementation, "实现"},
		{StageVerification, "验证"},
		{StageNeedsAttention, "待人工处理"},
		{StageCancelled, "已取消"},
		{StageName("unknown"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.stage), func(t *testing.T) {
			result := GetStageDisplayName(tt.stage)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetStatusDisplayName 测试 GetStatusDisplayName 函数
func TestGetStatusDisplayName(t *testing.T) {
	tests := []struct {
		status   StageStatus
		expected string
	}{
		{StatusPending, "待开始"},
		{StatusInProgress, "进行中"},
		{StatusCompleted, "已完成"},
		{StatusFailed, "失败"},
		{StageStatus("unknown"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := GetStatusDisplayName(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestConstraintsFromBDDRules 测试 ConstraintsFromBDDRules 函数
func TestConstraintsFromBDDRules(t *testing.T) {
	t.Run("nil rules", func(t *testing.T) {
		result := ConstraintsFromBDDRules(nil, "task-1", "TASK-1")
		assert.Nil(t, result)
	})

	t.Run("valid rules", func(t *testing.T) {
		rules := &BDDRules{
			Feature: BDDFeature{
				Name:        "登录功能",
				Description: "用户登录系统",
			},
			Scenarios: []BDDScenario{
				{
					Name:  "登录成功",
					Given: []string{"用户在登录页"},
					When:  []string{"输入正确的密码"},
					Then:  []string{"登录成功"},
				},
			},
			Summary: "登录功能测试",
		}

		result := ConstraintsFromBDDRules(rules, "task-1", "TASK-1")
		assert.NotNil(t, result)
		assert.Equal(t, "task-1", result.TaskID)
		assert.Equal(t, "TASK-1", result.Identifier)
		assert.Equal(t, "登录功能", result.Feature.Name)
		assert.Len(t, result.Scenarios, 1)
	})
}

// TestNewArchitectureGenerator 测试 NewArchitectureGenerator 构造函数
func TestNewArchitectureGenerator(t *testing.T) {
	engine := NewEngine()

	t.Run("default options", func(t *testing.T) {
		gen := NewArchitectureGenerator(engine)
		assert.NotNil(t, gen)
	})

	t.Run("with options", func(t *testing.T) {
		gen := NewArchitectureGenerator(engine,
			WithArchitectureDir("custom/arch"),
			WithTDDDir("custom/tdd"),
		)
		assert.NotNil(t, gen)
	})
}

// TestWithArchitecturePromptTemplate 测试 WithArchitecturePromptTemplate option
func TestWithArchitecturePromptTemplate(t *testing.T) {
	engine := NewEngine()
	template := "custom template"
	gen := NewArchitectureGenerator(engine, WithArchitecturePromptTemplate(template))
	assert.NotNil(t, gen)
}

// TestWithTDDPromptTemplate 测试 WithTDDPromptTemplate option
func TestWithTDDPromptTemplate(t *testing.T) {
	engine := NewEngine()
	template := "custom tdd template"
	gen := NewArchitectureGenerator(engine, WithTDDPromptTemplate(template))
	assert.NotNil(t, gen)
}

// TestNewBDDGenerator 测试 NewBDDGenerator 构造函数
func TestNewBDDGenerator(t *testing.T) {
	engine := NewEngine()

	t.Run("default options", func(t *testing.T) {
		gen := NewBDDGenerator(engine)
		assert.NotNil(t, gen)
	})

	t.Run("with options", func(t *testing.T) {
		gen := NewBDDGenerator(engine, WithBDDDir("test/bdd"))
		assert.NotNil(t, gen)
	})
}

// TestWithBDDPromptTemplate 测试 WithBDDPromptTemplate option
func TestWithBDDPromptTemplate(t *testing.T) {
	engine := NewEngine()
	template := "custom bdd template"
	gen := NewBDDGenerator(engine, WithBDDPromptTemplate(template))
	assert.NotNil(t, gen)
}

// TestNewContextBuilder 测试 NewContextBuilder 构造函数
func TestNewContextBuilder(t *testing.T) {
	builder := NewContextBuilder()
	assert.NotNil(t, builder)
}

// TestNewLoader 测试 NewLoader 构造函数
func TestNewLoader(t *testing.T) {
	loader := NewLoader("test/path")
	assert.NotNil(t, loader)
}