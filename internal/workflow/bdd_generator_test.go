// Package workflow_test 测试BDD规则自动生成功能
package workflow_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dministrator/symphony/internal/agent"
	"github.com/dministrator/symphony/internal/domain"
	"github.com/dministrator/symphony/internal/workflow"
)

// MockBDDRunner 用于测试的 Mock Agent Runner
type MockBDDRunner struct {
	Response string
	Error    error
	Success  bool
}

func (m *MockBDDRunner) RunAttempt(
	ctx context.Context,
	issue *domain.Issue,
	workspacePath string,
	attempt *int,
	promptTemplate string,
	callback agent.EventCallback,
) (*agent.RunAttemptResult, error) {
	if m.Error != nil {
		return &agent.RunAttemptResult{
			Success: false,
			Error:   m.Error,
		}, m.Error
	}

	return &agent.RunAttemptResult{
		Success:   m.Success,
		TurnCount: 1,
	}, nil
}

func TestNewBDDGenerator(t *testing.T) {
	engine := workflow.NewEngine()
	generator := workflow.NewBDDGenerator(engine)

	if generator == nil {
		t.Fatal("expected non-nil generator")
	}
}

func TestNewBDDGeneratorWithOptions(t *testing.T) {
	engine := workflow.NewEngine()
	tmpDir := t.TempDir()

	mockRunner := &MockBDDRunner{Success: true}
	generator := workflow.NewBDDGenerator(engine,
		workflow.WithBDDRunner(mockRunner),
		workflow.WithBDDDir(tmpDir),
		workflow.WithBDDPromptTemplate("custom template"),
	)

	if generator == nil {
		t.Fatal("expected non-nil generator")
	}
}

func TestGenerateBDDRulesWithoutRunner(t *testing.T) {
	engine := workflow.NewEngine()
	tmpDir := t.TempDir()

	generator := workflow.NewBDDGenerator(engine, workflow.WithBDDDir(tmpDir))

	task := &domain.Issue{
		ID:          "test-task-1",
		Identifier:  "TEST-1",
		Title:       "用户登录功能",
		Description: strPtr("实现用户登录功能，支持邮箱和密码登录"),
	}

	ctx := context.Background()
	result, err := generator.GenerateBDDRules(ctx, task, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.Rules == nil {
		t.Fatal("expected non-nil rules")
	}

	if result.Rules.Feature.Name != task.Title {
		t.Errorf("expected feature name %s, got %s", task.Title, result.Rules.Feature.Name)
	}

	if len(result.Rules.Scenarios) == 0 {
		t.Error("expected at least one scenario")
	}

	if result.FilePath == "" {
		t.Error("expected file path to be set")
	}

	// 验证文件已创建
	if _, err := os.Stat(result.FilePath); os.IsNotExist(err) {
		t.Error("expected BDD file to be created")
	}
}

func TestGenerateBDDRulesWithRunner(t *testing.T) {
	engine := workflow.NewEngine()
	tmpDir := t.TempDir()

	mockRunner := &MockBDDRunner{Success: true}
	generator := workflow.NewBDDGenerator(engine,
		workflow.WithBDDRunner(mockRunner),
		workflow.WithBDDDir(tmpDir),
	)

	task := &domain.Issue{
		ID:          "test-task-2",
		Identifier:  "TEST-2",
		Title:       "用户注册功能",
		Description: strPtr("实现用户注册功能"),
	}

	ctx := context.Background()
	result, err := generator.GenerateBDDRules(ctx, task, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.Rules == nil {
		t.Fatal("expected non-nil rules")
	}
}

func TestGenerateBDDRulesWithNilTask(t *testing.T) {
	engine := workflow.NewEngine()
	generator := workflow.NewBDDGenerator(engine)

	ctx := context.Background()
	_, err := generator.GenerateBDDRules(ctx, nil, nil)

	if err == nil {
		t.Error("expected error for nil task")
	}
}

func TestGenerateBDDRulesWithEmptyTaskID(t *testing.T) {
	engine := workflow.NewEngine()
	generator := workflow.NewBDDGenerator(engine)

	task := &domain.Issue{
		ID:    "",
		Title: "Test Task",
	}

	ctx := context.Background()
	_, err := generator.GenerateBDDRules(ctx, task, nil)

	if err == nil {
		t.Error("expected error for empty task ID")
	}
}

func TestGenerateBDDRulesWithRunnerError(t *testing.T) {
	engine := workflow.NewEngine()
	tmpDir := t.TempDir()

	mockRunner := &MockBDDRunner{
		Success: false,
		Error:   workflow.ErrBDDGenerationFailed,
	}
	generator := workflow.NewBDDGenerator(engine,
		workflow.WithBDDRunner(mockRunner),
		workflow.WithBDDDir(tmpDir),
	)

	task := &domain.Issue{
		ID:    "test-task-3",
		Title: "Test Task",
	}

	ctx := context.Background()
	result, err := generator.GenerateBDDRules(ctx, task, nil)

	if err == nil {
		t.Error("expected error when runner fails")
	}

	if result != nil && result.Error == nil {
		t.Error("expected result error to be set")
	}
}

func TestSaveBDDRules(t *testing.T) {
	engine := workflow.NewEngine()
	tmpDir := t.TempDir()

	generator := workflow.NewBDDGenerator(engine, workflow.WithBDDDir(tmpDir))

	rules := &workflow.BDDRules{
		Feature: workflow.BDDFeature{
			Name:        "用户登录功能",
			Description: "登录功能描述",
		},
		Scenarios: []workflow.BDDScenario{
			{
				Name:  "用户登录成功",
				Given: []string{"用户在登录页面"},
				When:  []string{"用户输入正确的邮箱和密码"},
				Then:  []string{"用户被重定向到首页"},
				Tags:  []string{"@happy_path"},
			},
		},
		Summary: "登录功能BDD规则",
	}

	filePath, err := generator.SaveBDDRules("test-task-4", rules)
	if err != nil {
		t.Fatalf("failed to save BDD rules: %v", err)
	}

	if filePath == "" {
		t.Error("expected file path to be returned")
	}

	// 验证文件已创建
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("expected BDD file to be created")
	}

	// 验证文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read BDD file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "Feature: 用户登录功能") {
		t.Error("expected Feature in file content")
	}

	if !strings.Contains(contentStr, "Scenario: 用户登录成功") {
		t.Error("expected Scenario in file content")
	}
}

func TestSaveBDDRulesWithNilRules(t *testing.T) {
	engine := workflow.NewEngine()
	generator := workflow.NewBDDGenerator(engine)

	_, err := generator.SaveBDDRules("test-task", nil)

	if err == nil {
		t.Error("expected error for nil rules")
	}

	if err != workflow.ErrInvalidBDDRules {
		t.Errorf("expected ErrInvalidBDDRules, got %v", err)
	}
}

func TestConvertToGherkin(t *testing.T) {
	rules := &workflow.BDDRules{
		Feature: workflow.BDDFeature{
			Name:        "用户登录",
			Description: "登录功能",
		},
		Scenarios: []workflow.BDDScenario{
			{
				Name:  "登录成功",
				Given: []string{"用户在登录页面", "用户有有效账户"},
				When:  []string{"用户输入正确密码"},
				Then:  []string{"用户进入首页", "显示欢迎消息"},
				Tags:  []string{"@happy_path"},
			},
			{
				Name:  "登录失败",
				Given: []string{"用户在登录页面"},
				When:  []string{"用户输入错误密码"},
				Then:  []string{"显示错误提示"},
			},
		},
	}

	gherkin := workflow.ConvertToGherkin(rules)

	if gherkin == "" {
		t.Fatal("expected non-empty Gherkin content")
	}

	// 验证 Feature
	if !strings.Contains(gherkin, "Feature: 用户登录") {
		t.Error("expected Feature header")
	}

	// 验证 Scenario
	if !strings.Contains(gherkin, "Scenario: 登录成功") {
		t.Error("expected first Scenario")
	}

	if !strings.Contains(gherkin, "Scenario: 登录失败") {
		t.Error("expected second Scenario")
	}

	// 验证 Given/When/Then
	if !strings.Contains(gherkin, "Given 用户在登录页面") {
		t.Error("expected Given clause")
	}

	if !strings.Contains(gherkin, "When 用户输入正确密码") {
		t.Error("expected When clause")
	}

	if !strings.Contains(gherkin, "Then 用户进入首页") {
		t.Error("expected Then clause")
	}

	// 验证 And
	if !strings.Contains(gherkin, "And 用户有有效账户") {
		t.Error("expected And clause for second Given")
	}

	if !strings.Contains(gherkin, "And 显示欢迎消息") {
		t.Error("expected And clause for second Then")
	}

	// 验证 Tags
	if !strings.Contains(gherkin, "@happy_path") {
		t.Error("expected tag in content")
	}
}

func TestConvertToGherkinWithNilRules(t *testing.T) {
	gherkin := workflow.ConvertToGherkin(nil)

	if gherkin != "" {
		t.Error("expected empty string for nil rules")
	}
}

func TestParseBDDRulesResponse(t *testing.T) {
	response := `{
		"feature": {
			"name": "用户登录功能",
			"description": "实现用户登录"
		},
		"scenarios": [
			{
				"name": "登录成功",
				"given": ["用户在登录页面"],
				"when": ["用户输入正确密码"],
				"then": ["用户进入首页"],
				"tags": ["@happy_path"]
			}
		],
		"summary": "登录功能BDD规则"
	}`

	rules, err := workflow.ParseBDDRulesResponse(response)

	if err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if rules == nil {
		t.Fatal("expected non-nil rules")
	}

	if rules.Feature.Name != "用户登录功能" {
		t.Errorf("expected feature name '用户登录功能', got '%s'", rules.Feature.Name)
	}

	if len(rules.Scenarios) != 1 {
		t.Errorf("expected 1 scenario, got %d", len(rules.Scenarios))
	}

	if rules.Scenarios[0].Name != "登录成功" {
		t.Errorf("expected scenario name '登录成功', got '%s'", rules.Scenarios[0].Name)
	}
}

func TestParseBDDRulesResponseWithMarkdownBlock(t *testing.T) {
	response := "```json\n{\"feature\":{\"name\":\"测试功能\"},\"scenarios\":[]}\n```"

	rules, err := workflow.ParseBDDRulesResponse(response)

	if err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if rules.Feature.Name != "测试功能" {
		t.Errorf("expected feature name '测试功能', got '%s'", rules.Feature.Name)
	}
}

func TestParseBDDRulesResponseWithEmptyFeature(t *testing.T) {
	response := `{"feature":{"name":""},"scenarios":[]}`

	_, err := workflow.ParseBDDRulesResponse(response)

	if err == nil {
		t.Error("expected error for empty feature name")
	}

	if err != workflow.ErrInvalidBDDRules {
		t.Errorf("expected ErrInvalidBDDRules, got %v", err)
	}
}

func TestParseBDDRulesResponseWithEmptyScenarios(t *testing.T) {
	response := `{"feature":{"name":"测试功能"},"scenarios":[]}`

	rules, err := workflow.ParseBDDRulesResponse(response)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 空场景应该添加默认场景
	if len(rules.Scenarios) == 0 {
		t.Error("expected default scenario to be added")
	}
}

func TestParseGherkinContent(t *testing.T) {
	content := `Feature: 用户登录功能

  @happy_path
  Scenario: 登录成功
    Given 用户在登录页面
    And 用户有有效账户
    When 用户输入正确密码
    Then 用户进入首页
    And 显示欢迎消息

  Scenario: 登录失败
    Given 用户在登录页面
    When 用户输入错误密码
    Then 显示错误提示
`

	rules, err := workflow.ParseGherkinContent(content)

	if err != nil {
		t.Fatalf("failed to parse Gherkin content: %v", err)
	}

	if rules == nil {
		t.Fatal("expected non-nil rules")
	}

	if rules.Feature.Name != "用户登录功能" {
		t.Errorf("expected feature name '用户登录功能', got '%s'", rules.Feature.Name)
	}

	if len(rules.Scenarios) != 2 {
		t.Errorf("expected 2 scenarios, got %d", len(rules.Scenarios))
	}

	// 验证第一个场景
	scenario1 := rules.Scenarios[0]
	if scenario1.Name != "登录成功" {
		t.Errorf("expected scenario name '登录成功', got '%s'", scenario1.Name)
	}

	if len(scenario1.Given) != 2 {
		t.Errorf("expected 2 given clauses, got %d", len(scenario1.Given))
	}

	if len(scenario1.Tags) != 1 {
		t.Errorf("expected 1 tag, got %d", len(scenario1.Tags))
	}

	// 验证第二个场景
	scenario2 := rules.Scenarios[1]
	if scenario2.Name != "登录失败" {
		t.Errorf("expected scenario name '登录失败', got '%s'", scenario2.Name)
	}
}

func TestParseGherkinContentWithEmptyFeature(t *testing.T) {
	content := `Scenario: 测试场景
    Given 前置条件
`

	_, err := workflow.ParseGherkinContent(content)

	if err == nil {
		t.Error("expected error for missing Feature")
	}
}

func TestLoadBDDRules(t *testing.T) {
	engine := workflow.NewEngine()
	tmpDir := t.TempDir()

	generator := workflow.NewBDDGenerator(engine, workflow.WithBDDDir(tmpDir))

	// 先保存一个规则文件
	rules := &workflow.BDDRules{
		Feature: workflow.BDDFeature{
			Name: "测试功能",
		},
		Scenarios: []workflow.BDDScenario{
			{
				Name:  "测试场景",
				Given: []string{"前置条件"},
				When:  []string{"动作"},
				Then:  []string{"结果"},
			},
		},
	}

	_, err := generator.SaveBDDRules("test-load-task", rules)
	if err != nil {
		t.Fatalf("failed to save BDD rules: %v", err)
	}

	// 加载规则
	loadedRules, err := generator.LoadBDDRules("test-load-task")
	if err != nil {
		t.Fatalf("failed to load BDD rules: %v", err)
	}

	if loadedRules == nil {
		t.Fatal("expected non-nil loaded rules")
	}

	if loadedRules.Feature.Name != rules.Feature.Name {
		t.Errorf("expected feature name '%s', got '%s'", rules.Feature.Name, loadedRules.Feature.Name)
	}
}

func TestLoadBDDRulesNotFound(t *testing.T) {
	engine := workflow.NewEngine()
	generator := workflow.NewBDDGenerator(engine)

	_, err := generator.LoadBDDRules("non-existent-task")

	if err == nil {
		t.Error("expected error for non-existent file")
	}

	if err != workflow.ErrBDDFileNotFound {
		t.Errorf("expected ErrBDDFileNotFound, got %v", err)
	}
}

func TestTriggerBDDGeneration(t *testing.T) {
	engine := workflow.NewEngine()
	tmpDir := t.TempDir()

	// 初始化任务
	taskID := "trigger-test-task"
	_, err := engine.InitTask(taskID)
	if err != nil {
		t.Fatalf("failed to init task: %v", err)
	}

	// 推进 clarification 阶段到完成状态
	_, err = engine.AdvanceStage(taskID)
	if err != nil {
		t.Fatalf("failed to advance to bdd_review: %v", err)
	}

	// 创建生成器
	generator := workflow.NewBDDGenerator(engine, workflow.WithBDDDir(tmpDir))

	task := &domain.Issue{
		ID:          taskID,
		Identifier:  "TRIGGER-1",
		Title:       "触发测试功能",
		Description: strPtr("测试触发BDD生成"),
	}

	clarificationHistory := []domain.ConversationTurn{
		{Role: "assistant", Content: "请描述功能需求", Timestamp: time.Now()},
		{Role: "user", Content: "需要实现登录功能", Timestamp: time.Now()},
	}

	ctx := context.Background()
	result, err := generator.TriggerBDDGeneration(ctx, task, clarificationHistory)

	if err != nil {
		t.Fatalf("failed to trigger BDD generation: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.Rules == nil {
		t.Fatal("expected non-nil rules")
	}

	// 验证工作流状态
	workflowState := engine.GetWorkflow(taskID)
	if workflowState == nil {
		t.Fatal("expected workflow to exist")
	}

	if workflowState.CurrentStage != workflow.StageBDDReview {
		t.Errorf("expected current stage bdd_review, got %s", workflowState.CurrentStage)
	}
}

func TestTriggerBDDGenerationBeforeClarificationComplete(t *testing.T) {
	engine := workflow.NewEngine()
	tmpDir := t.TempDir()

	// 初始化任务（clarification 阶段处于进行中）
	taskID := "trigger-fail-task"
	_, err := engine.InitTask(taskID)
	if err != nil {
		t.Fatalf("failed to init task: %v", err)
	}

	generator := workflow.NewBDDGenerator(engine, workflow.WithBDDDir(tmpDir))

	task := &domain.Issue{
		ID:    taskID,
		Title: "测试功能",
	}

	ctx := context.Background()
	_, err = generator.TriggerBDDGeneration(ctx, task, nil)

	if err == nil {
		t.Error("expected error when clarification not completed")
	}
}

func TestTriggerBDDGenerationWithWorkflowNotFound(t *testing.T) {
	engine := workflow.NewEngine()
	generator := workflow.NewBDDGenerator(engine)

	task := &domain.Issue{
		ID:    "non-existent-workflow",
		Title: "测试功能",
	}

	ctx := context.Background()
	_, err := generator.TriggerBDDGeneration(ctx, task, nil)

	if err == nil {
		t.Error("expected error for non-existent workflow")
	}

	if err != workflow.ErrWorkflowNotFound {
		t.Errorf("expected ErrWorkflowNotFound, got %v", err)
	}
}

func TestGetBDDFilePath(t *testing.T) {
	engine := workflow.NewEngine()
	tmpDir := t.TempDir()

	generator := workflow.NewBDDGenerator(engine, workflow.WithBDDDir(tmpDir))

	filePath := generator.GetBDDFilePath("test/task:123")

	expectedFileName := "test_task_123.feature"
	if !strings.Contains(filePath, expectedFileName) {
		t.Errorf("expected file name '%s' in path '%s'", expectedFileName, filePath)
	}

	if !strings.Contains(filePath, tmpDir) {
		t.Errorf("expected tmpDir '%s' in path '%s'", tmpDir, filePath)
	}
}

func TestGetBDDStatus(t *testing.T) {
	engine := workflow.NewEngine()
	tmpDir := t.TempDir()

	// 初始化任务并推进到 bdd_review
	taskID := "status-test-task"
	_, err := engine.InitTask(taskID)
	if err != nil {
		t.Fatalf("failed to init task: %v", err)
	}

	_, err = engine.AdvanceStage(taskID)
	if err != nil {
		t.Fatalf("failed to advance stage: %v", err)
	}

	generator := workflow.NewBDDGenerator(engine, workflow.WithBDDDir(tmpDir))

	status, err := generator.GetBDDStatus(taskID)
	if err != nil {
		t.Fatalf("failed to get BDD status: %v", err)
	}

	if status == nil {
		t.Fatal("expected non-nil status")
	}

	if status.TaskID != taskID {
		t.Errorf("expected task ID '%s', got '%s'", taskID, status.TaskID)
	}

	if status.Status != workflow.StatusInProgress {
		t.Errorf("expected status '%s', got '%s'", workflow.StatusInProgress, status.Status)
	}

	// BDD 规则尚未生成
	if status.Error != "BDD rules not generated yet" {
		t.Errorf("expected error 'BDD rules not generated yet', got '%s'", status.Error)
	}
}

func TestFormatClarificationHistory(t *testing.T) {
	history := []domain.ConversationTurn{
		{Role: "assistant", Content: "请描述需求", Timestamp: time.Now()},
		{Role: "user", Content: "需要登录功能", Timestamp: time.Now()},
		{Role: "assistant", Content: "需要什么验证方式？", Timestamp: time.Now()},
		{Role: "user", Content: "邮箱密码登录", Timestamp: time.Now()},
	}

	historyStr := workflow.FormatClarificationHistory(history)

	if historyStr == "" {
		t.Fatal("expected non-empty history string")
	}

	if !strings.Contains(historyStr, "澄清对话记录") {
		t.Error("expected header in history")
	}

	if !strings.Contains(historyStr, "用户") {
		t.Error("expected '用户' label")
	}

	if !strings.Contains(historyStr, "AI助手") {
		t.Error("expected 'AI助手' label")
	}
}

func TestFormatClarificationHistoryEmpty(t *testing.T) {
	history := []domain.ConversationTurn{}

	historyStr := workflow.FormatClarificationHistory(history)

	if historyStr != "无澄清历史" {
		t.Errorf("expected '无澄清历史', got '%s'", historyStr)
	}
}

func TestBuildBDDPrompt(t *testing.T) {
	engine := workflow.NewEngine()
	generator := workflow.NewBDDGenerator(engine)

	task := &domain.Issue{
		Title:       "用户登录",
		Description: strPtr("实现登录功能"),
	}

	history := []domain.ConversationTurn{
		{Role: "user", Content: "需要登录", Timestamp: time.Now()},
	}

	prompt := generator.BuildBDDPrompt(task, history)

	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}

	if !strings.Contains(prompt, "用户登录") {
		t.Error("expected task title in prompt")
	}

	if !strings.Contains(prompt, "实现登录功能") {
		t.Error("expected task description in prompt")
	}

	if strings.Contains(prompt, "{{ issue.title }}") {
		// 应该已经被替换
		t.Error("template placeholder should be replaced")
	}
}

func TestSanitizeTaskID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ABC-123", "ABC-123"},
		{"ABC/123", "ABC_123"},
		{"ABC:123", "ABC_123"},
		{"ABC 123", "ABC_123"},
		{"ABC/:123", "ABC__123"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := workflow.SanitizeTaskID(tt.input)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestBDDGenerationResult(t *testing.T) {
	result := &workflow.BDDGenerationResult{
		Rules: &workflow.BDDRules{
			Feature: workflow.BDDFeature{Name: "测试"},
		},
		FilePath: "/path/to/file.feature",
		Error:    nil,
	}

	if result.Rules == nil {
		t.Error("expected rules to be set")
	}

	if result.FilePath == "" {
		t.Error("expected file path to be set")
	}
}

func TestBDDRulesStruct(t *testing.T) {
	rules := &workflow.BDDRules{
		Feature: workflow.BDDFeature{
			Name:        "用户登录",
			Description: "登录功能描述",
		},
		Scenarios: []workflow.BDDScenario{
			{
				Name:  "成功登录",
				Given: []string{"在登录页"},
				When:  []string{"输入密码"},
				Then:  []string{"进入首页"},
				Tags:  []string{"@happy"},
			},
		},
		Summary: "规则摘要",
	}

	if rules.Feature.Name != "用户登录" {
		t.Error("expected feature name")
	}

	if len(rules.Scenarios) != 1 {
		t.Error("expected one scenario")
	}

	if rules.Summary != "规则摘要" {
		t.Error("expected summary")
	}
}

func TestBDDScenarioStruct(t *testing.T) {
	scenario := workflow.BDDScenario{
		Name:  "测试场景",
		Given: []string{"前置1", "前置2"},
		When:  []string{"动作1", "动作2"},
		Then:  []string{"结果1", "结果2"},
		Tags:  []string{"@tag1", "@tag2"},
	}

	if scenario.Name != "测试场景" {
		t.Error("expected scenario name")
	}

	if len(scenario.Given) != 2 {
		t.Error("expected two given clauses")
	}

	if len(scenario.When) != 2 {
		t.Error("expected two when clauses")
	}

	if len(scenario.Then) != 2 {
		t.Error("expected two then clauses")
	}

	if len(scenario.Tags) != 2 {
		t.Error("expected two tags")
	}
}

func TestBDDFeatureStruct(t *testing.T) {
	feature := workflow.BDDFeature{
		Name:        "测试功能",
		Description: "功能描述",
	}

	if feature.Name != "测试功能" {
		t.Error("expected feature name")
	}

	if feature.Description != "功能描述" {
		t.Error("expected feature description")
	}
}

func TestBDDErrorConstants(t *testing.T) {
	// 验证错误常量已定义
	if workflow.ErrInvalidBDDRules == nil {
		t.Error("ErrInvalidBDDRules should not be nil")
	}

	if workflow.ErrBDDFileNotFound == nil {
		t.Error("ErrBDDFileNotFound should not be nil")
	}

	if workflow.ErrBDDGenerationFailed == nil {
		t.Error("ErrBDDGenerationFailed should not be nil")
	}
}

func TestBDDGeneratorInterface(t *testing.T) {
	engine := workflow.NewEngine()
	generator := workflow.NewBDDGenerator(engine)

	// 验证 BDDGenerator 实现了 BDDGeneratorInterface
	var _ workflow.BDDGeneratorInterface = generator
}

func TestBDDGeneratorSetMethods(t *testing.T) {
	engine := workflow.NewEngine()
	generator := workflow.NewBDDGenerator(engine)

	// 测试 SetRunner
	mockRunner := &MockBDDRunner{Success: true}
	generator.SetRunner(mockRunner)

	// 测试 SetPromptTemplate
	generator.SetPromptTemplate("custom template")

	// 测试 SetBDDDir
	generator.SetBDDDir("custom/dir")
}

// Helper function
func strPtr(s string) *string {
	return &s
}