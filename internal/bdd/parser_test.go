package bdd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseGherkin_Empty(t *testing.T) {
	feature, err := ParseGherkin("")
	assert.NoError(t, err)
	assert.Nil(t, feature)
}

func TestParseGherkin_SimpleFeature(t *testing.T) {
	content := `Feature: 用户登录功能

Scenario: 邮箱登录成功
  Given 用户在登录页面
  When 用户输入有效邮箱 "test@example.com"
  Then 用户被重定向到首页`

	feature, err := ParseGherkin(content)
	assert.NoError(t, err)
	assert.NotNil(t, feature)
	assert.Equal(t, "用户登录功能", feature.Name)
	assert.Len(t, feature.Scenarios, 1)
	assert.Equal(t, "邮箱登录成功", feature.Scenarios[0].Name)
	assert.Len(t, feature.Scenarios[0].Steps, 3)
}

func TestParseGherkin_MultipleScenarios(t *testing.T) {
	content := `Feature: 用户登录功能

Scenario: 邮箱登录成功
  Given 用户在登录页面
  When 用户输入有效邮箱
  Then 用户被重定向到首页

Scenario: 无效邮箱登录失败
  Given 用户在登录页面
  When 用户输入无效邮箱 "invalid"
  Then 显示错误消息 "邮箱格式不正确"`

	feature, err := ParseGherkin(content)
	assert.NoError(t, err)
	assert.NotNil(t, feature)
	assert.Len(t, feature.Scenarios, 2)
	assert.Equal(t, 6, feature.TotalSteps())
}

func TestParseGherkin_WithDescription(t *testing.T) {
	content := `Feature: 用户登录功能
  作为系统用户
  我希望能够通过邮箱登录系统
  以便访问我的个人数据

Scenario: 邮箱登录成功
  Given 用户在登录页面`

	feature, err := ParseGherkin(content)
	assert.NoError(t, err)
	assert.NotNil(t, feature)
	assert.Contains(t, feature.Description, "作为系统用户")
	assert.Contains(t, feature.Description, "我希望能够通过邮箱登录系统")
}

func TestParseGherkin_WithTags(t *testing.T) {
	content := `Feature: 用户登录功能

@smoke @critical
Scenario: 邮箱登录成功
  Given 用户在登录页面`

	feature, err := ParseGherkin(content)
	assert.NoError(t, err)
	assert.NotNil(t, feature)
	assert.Len(t, feature.Scenarios[0].Tags, 2)
	assert.Contains(t, feature.Scenarios[0].Tags, "smoke")
	assert.Contains(t, feature.Scenarios[0].Tags, "critical")
}

func TestParseGherkin_ScenarioOutline(t *testing.T) {
	content := `Feature: 用户登录功能

Scenario Outline: 多种登录方式
  Given 用户在登录页面
  When 用户使用 <方式> 登录
  Then 登录结果为 <结果>

Examples:
  | 方式     | 结果   |
  | 邮箱     | 成功   |
  | 手机号   | 成功   |`

	feature, err := ParseGherkin(content)
	assert.NoError(t, err)
	assert.NotNil(t, feature)
	assert.Len(t, feature.Scenarios, 1)
	assert.Equal(t, "多种登录方式", feature.Scenarios[0].Name)
}

func TestParseGherkin_EnglishKeywords(t *testing.T) {
	content := `Feature: User Login

Scenario: Successful email login
  Given user is on login page
  When user enters valid email
  Then user is redirected to homepage

Scenario: Failed login
  Given user is on login page
  When user enters invalid credentials
  And submits the form
  But the login fails
  Then error message is displayed`

	feature, err := ParseGherkin(content)
	assert.NoError(t, err)
	assert.NotNil(t, feature)
	assert.Equal(t, "User Login", feature.Name)
	assert.Len(t, feature.Scenarios, 2)

	// 验证第二个场景的所有步骤关键词
	assert.Len(t, feature.Scenarios[1].Steps, 5)
	assert.Equal(t, "Given", feature.Scenarios[1].Steps[0].Keyword)
	assert.Equal(t, "When", feature.Scenarios[1].Steps[1].Keyword)
	assert.Equal(t, "And", feature.Scenarios[1].Steps[2].Keyword)
	assert.Equal(t, "But", feature.Scenarios[1].Steps[3].Keyword)
	assert.Equal(t, "Then", feature.Scenarios[1].Steps[4].Keyword)
}

func TestParseGherkin_ChineseKeywords(t *testing.T) {
	content := `功能: 用户登录

场景: 邮箱登录成功
  假如 用户在登录页面
  当 用户输入有效邮箱
  那么 用户被重定向到首页
  并且 显示欢迎消息`

	feature, err := ParseGherkin(content)
	assert.NoError(t, err)
	assert.NotNil(t, feature)
	assert.Equal(t, "用户登录", feature.Name)
	assert.Len(t, feature.Scenarios, 1)
	assert.Len(t, feature.Scenarios[0].Steps, 4)
	assert.Equal(t, "假如", feature.Scenarios[0].Steps[0].Keyword)
	assert.Equal(t, "当", feature.Scenarios[0].Steps[1].Keyword)
	assert.Equal(t, "那么", feature.Scenarios[0].Steps[2].Keyword)
	assert.Equal(t, "并且", feature.Scenarios[0].Steps[3].Keyword)
}

func TestStepKeywordCSSClass(t *testing.T) {
	tests := []struct {
		keyword string
		expected string
	}{
		{"Given", "step-given"},
		{"假如", "step-given"},
		{"When", "step-when"},
		{"当", "step-when"},
		{"Then", "step-then"},
		{"那么", "step-then"},
		{"And", "step-and"},
		{"并且", "step-and"},
		{"But", "step-but"},
		{"但是", "step-but"},
		{"Unknown", "step-other"},
	}

	for _, tt := range tests {
		result := StepKeywordCSSClass(tt.keyword)
		assert.Equal(t, tt.expected, result)
	}
}

func TestFeature_ScenarioCount(t *testing.T) {
	feature := &Feature{
		Scenarios: []Scenario{
			{Name: "Scenario 1"},
			{Name: "Scenario 2"},
			{Name: "Scenario 3"},
		},
	}
	assert.Equal(t, 3, feature.ScenarioCount())
}

func TestFeature_TotalSteps(t *testing.T) {
	feature := &Feature{
		Scenarios: []Scenario{
			{Name: "Scenario 1", Steps: []Step{{}, {}, {}}},
			{Name: "Scenario 2", Steps: []Step{{}, {}}},
		},
	}
	assert.Equal(t, 5, feature.TotalSteps())
}

func TestValidateBDDContent(t *testing.T) {
	tests := []struct {
		content  string
		expected bool
	}{
		{"", false},
		{"Some random text", false},
		{"Feature: Test\nSome description", false}, // 没有 Scenario
		{"Scenario: Test\nGiven something", false}, // 没有 Feature
		{"Feature: Test\nScenario: Test\nGiven something", true},
		{"功能: 测试\n场景: 测试\n假如 某事", true},
		{"Feature: Test\nScenario Outline: Test\nGiven something", true},
	}

	for _, tt := range tests {
		result := ValidateBDDContent(tt.content)
		assert.Equal(t, tt.expected, result, "content: %q", tt.content)
	}
}

func TestExtractFeatureName(t *testing.T) {
	tests := []struct {
		content  string
		expected string
	}{
		{"Feature: User Login\nScenario: Test", "User Login"},
		{"功能: 用户登录\n场景: 测试", "用户登录"},
		{"Feature:   Multiple Spaces   \nScenario: Test", "Multiple Spaces"},
		{"No feature here", ""},
	}

	for _, tt := range tests {
		result := ExtractFeatureName(tt.content)
		assert.Equal(t, tt.expected, result)
	}
}

func TestFormatGherkinForDisplay(t *testing.T) {
	content := `Feature: User Login

Scenario: Successful login
  Given user is on login page
  When user enters credentials
  Then user is logged in`

	result := FormatGherkinForDisplay(content)

	assert.Contains(t, result, `<span class="gherkin-feature">`)
	assert.Contains(t, result, `<span class="gherkin-scenario">`)
	assert.Contains(t, result, `<span class="gherkin-step step-given">`)
	assert.Contains(t, result, `<span class="gherkin-step step-when">`)
	assert.Contains(t, result, `<span class="gherkin-step step-then">`)
}

func TestFormatGherkinForDisplay_WithTags(t *testing.T) {
	content := `Feature: User Login

@smoke @critical
Scenario: Successful login
  Given user is on login page`

	result := FormatGherkinForDisplay(content)

	assert.Contains(t, result, `<span class="gherkin-tags">@smoke @critical</span>`)
}

func TestParseStep(t *testing.T) {
	tests := []struct {
		line     string
		expected *Step
	}{
		{"Given user is on login page", &Step{Keyword: "Given", Text: "user is on login page"}},
		{"Given: user is on login page", &Step{Keyword: "Given", Text: "user is on login page"}},
		{"When user clicks submit", &Step{Keyword: "When", Text: "user clicks submit"}},
		{"Then result is shown", &Step{Keyword: "Then", Text: "result is shown"}},
		{"And the form is valid", &Step{Keyword: "And", Text: "the form is valid"}},
		{"But the password is wrong", &Step{Keyword: "But", Text: "the password is wrong"}},
		{"No keyword here", nil},
	}

	for _, tt := range tests {
		result := parseStep(tt.line)
		if tt.expected == nil {
			assert.Nil(t, result)
		} else {
			assert.NotNil(t, result)
			assert.Equal(t, tt.expected.Keyword, result.Keyword)
			assert.Equal(t, tt.expected.Text, result.Text)
		}
	}
}

func TestParseTags(t *testing.T) {
	tests := []struct {
		line     string
		expected []string
	}{
		{"@smoke", []string{"smoke"}},
		{"@smoke @critical", []string{"smoke", "critical"}},
		{"@smoke @critical @auth", []string{"smoke", "critical", "auth"}},
	}

	for _, tt := range tests {
		result := parseTags(tt.line)
		assert.Equal(t, tt.expected, result)
	}
}