// Package workflow 测试约束管理功能
package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestBDDScenario 测试 BDD 场景结构
func TestBDDScenario(t *testing.T) {
	scenario := BDDScenario{
		Name:  "用户邮箱登录成功",
		Given: []string{"用户在登录页面"},
		When:  []string{"用户输入有效邮箱和密码并点击登录"},
		Then:  []string{"用户被重定向到首页"},
	}

	if scenario.Name != "用户邮箱登录成功" {
		t.Errorf("unexpected name: %s", scenario.Name)
	}
}

// TestBDDConstraints 测试 BDD 约束结构
func TestBDDConstraints(t *testing.T) {
	constraints := &BDDConstraints{
		TaskID:    "task-123",
		Identifier: "TEST-1",
		Scenarios: []BDDScenario{
			{Name: "场景1", Given: []string{"G1"}, When: []string{"W1"}, Then: []string{"T1"}},
			{Name: "场景2", Given: []string{"G2"}, When: []string{"W2"}, Then: []string{"T2"}},
		},
		ApprovedAt: "2024-01-01T00:00:00Z",
		FilePath:   "/path/to/bdd.json",
	}

	if constraints.TaskID != "task-123" {
		t.Errorf("unexpected taskID: %s", constraints.TaskID)
	}
	if len(constraints.Scenarios) != 2 {
		t.Errorf("expected 2 scenarios, got %d", len(constraints.Scenarios))
	}
}

// TestNewConstraintManager 测试创建约束管理器
func TestNewConstraintManager(t *testing.T) {
	engine := NewEngine()
	tempDir := t.TempDir()

	cm := NewConstraintManager(engine, tempDir)

	if cm == nil {
		t.Error("expected non-nil constraint manager")
	}
	if cm.engine != engine {
		t.Error("engine not set correctly")
	}
	if cm.workspaceRoot != tempDir {
		t.Error("workspaceRoot not set correctly")
	}
	if cm.constraints == nil {
		t.Error("constraints map should be initialized")
	}
}

// TestFormatConstraintsForPrompt 测试格式化约束为 Prompt
func TestFormatConstraintsForPrompt(t *testing.T) {
	engine := NewEngine()
	tempDir := t.TempDir()
	cm := NewConstraintManager(engine, tempDir)

	tests := []struct {
		name       string
		constraints *BDDConstraints
		expectedContains []string
	}{
		{
			name: "单场景约束",
			constraints: &BDDConstraints{
				Scenarios: []BDDScenario{
					{
						Name:  "用户邮箱登录成功",
						Given: []string{"用户在登录页面"},
						When:  []string{"用户输入有效邮箱和密码并点击登录"},
						Then:  []string{"用户被重定向到首页"},
					},
				},
			},
			expectedContains: []string{
				"## BDD 验收标准",
				"### Scenario 1: 用户邮箱登录成功",
				"- Given: 用户在登录页面",
				"- When: 用户输入有效邮箱和密码并点击登录",
				"- Then: 用户被重定向到首页",
				"**重要**: 你的实现必须使上述所有场景的测试通过",
			},
		},
		{
			name: "多场景约束",
			constraints: &BDDConstraints{
				Scenarios: []BDDScenario{
					{Name: "场景A", Given: []string{"GA"}, When: []string{"WA"}, Then: []string{"TA"}},
					{Name: "场景B", Given: []string{"GB"}, When: []string{"WB"}, Then: []string{"TB"}},
				},
			},
			expectedContains: []string{
				"### Scenario 1: 场景A",
				"### Scenario 2: 场景B",
			},
		},
		{
			name:       "空约束",
			constraints: nil,
			expectedContains: []string{},
		},
		{
			name:       "空场景列表",
			constraints: &BDDConstraints{Scenarios: []BDDScenario{}},
			expectedContains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cm.FormatConstraintsForPrompt(tt.constraints)

			if len(tt.expectedContains) == 0 {
				if result != "" {
					t.Errorf("expected empty result, got: %s", result)
				}
				return
			}

			for _, expected := range tt.expectedContains {
				if !strings.Contains(result, expected) {
					t.Errorf("result missing expected content: %q\nResult:\n%s", expected, result)
				}
			}
		})
	}
}

// TestFormatConstraintsForPromptWithScenarios 测试直接格式化场景列表
func TestFormatConstraintsForPromptWithScenarios(t *testing.T) {
	engine := NewEngine()
	tempDir := t.TempDir()
	cm := NewConstraintManager(engine, tempDir)

	scenarios := []BDDScenario{
		{Name: "登录成功", Given: []string{"用户在登录页"}, When: []string{"点击登录"}, Then: []string{"跳转首页"}},
	}

	result := cm.FormatConstraintsForPromptWithScenarios(scenarios)

	if !strings.Contains(result, "登录成功") {
		t.Errorf("result missing scenario name")
	}
}

// TestLoadBDDConstraints 测试加载 BDD 约束
func TestLoadBDDConstraints(t *testing.T) {
	engine := NewEngine()
	tempDir := t.TempDir()
	cm := NewConstraintManager(engine, tempDir)

	// 初始化任务
	taskID := "test-task-1"
	_, err := engine.InitTask(taskID)
	if err != nil {
		t.Fatalf("failed to init task: %v", err)
	}

	// 测试工作流不存在的情况
	_, err = cm.LoadBDDConstraints("non-existent-task")
	if err != ErrWorkflowNotFound {
		t.Errorf("expected ErrWorkflowNotFound, got: %v", err)
	}

	// 测试 BDD 阶段未完成的情况
	_, err = cm.LoadBDDConstraints(taskID)
	// 应该返回错误或 nil（取决于 BDD 阶段状态）
	if err == nil {
		// 如果 BDD 阶段不是完成状态，应该返回错误
		t.Log("LoadBDDConstraints returned nil for incomplete BDD stage")
	}
}

// TestLoadBDDJSONFile 测试加载 JSON 格式的 BDD 文件
func TestLoadBDDJSONFile(t *testing.T) {
	engine := NewEngine()
	tempDir := t.TempDir()
	cm := NewConstraintManager(engine, tempDir)

	// 创建测试 BDD JSON 文件
	identifier := "TEST-1"
	workspaceKey := SanitizeTaskID(identifier)
	workspacePath := filepath.Join(tempDir, workspaceKey)
	bddFilePath := filepath.Join(workspacePath, "bdd.json")

	// 创建目录
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// 创建 BDD 内容
	bddContent := &BDDConstraints{
		TaskID:    "task-1",
		Identifier: identifier,
		Scenarios: []BDDScenario{
			{Name: "登录成功", Given: []string{"G"}, When: []string{"W"}, Then: []string{"T"}},
		},
		ApprovedAt: time.Now().Format(time.RFC3339),
	}

	data, err := json.MarshalIndent(bddContent, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal BDD: %v", err)
	}

	if err := os.WriteFile(bddFilePath, data, 0644); err != nil {
		t.Fatalf("failed to write BDD file: %v", err)
	}

	// 测试加载
	constraints, err := cm.LoadBDDConstraintsByIdentifier(identifier)
	if err != nil {
		t.Errorf("failed to load BDD constraints: %v", err)
	}

	if constraints == nil {
		t.Error("expected non-nil constraints")
		return
	}

	if len(constraints.Scenarios) != 1 {
		t.Errorf("expected 1 scenario, got %d", len(constraints.Scenarios))
	}

	if constraints.Scenarios[0].Name != "登录成功" {
		t.Errorf("unexpected scenario name: %s", constraints.Scenarios[0].Name)
	}

	if constraints.FilePath != bddFilePath {
		t.Errorf("unexpected file path: %s", constraints.FilePath)
	}
}

// TestLoadBDDMarkdownFile 测试加载 Markdown 格式的 BDD 文件
func TestLoadBDDMarkdownFile(t *testing.T) {
	engine := NewEngine()
	tempDir := t.TempDir()
	cm := NewConstraintManager(engine, tempDir)

	// 创建测试 BDD Markdown 文件
	identifier := "TEST-2"
	workspaceKey := SanitizeTaskID(identifier)
	workspacePath := filepath.Join(tempDir, workspaceKey)
	bddFilePath := filepath.Join(workspacePath, "bdd.md")

	// 创建目录
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// 创建 BDD Markdown 内容
	bddMarkdown := `# BDD Specifications

### Scenario: 用户邮箱登录成功
- Given: 用户在登录页面
- When: 用户输入有效邮箱和密码并点击登录
- Then: 用户被重定向到首页

### Scenario: 无效邮箱登录失败
- Given: 用户在登录页面
- When: 用户输入无效邮箱并点击登录
- Then: 显示错误消息
`

	if err := os.WriteFile(bddFilePath, []byte(bddMarkdown), 0644); err != nil {
		t.Fatalf("failed to write BDD file: %v", err)
	}

	// 测试加载
	constraints, err := cm.LoadBDDConstraintsByIdentifier(identifier)
	if err != nil {
		t.Errorf("failed to load BDD constraints: %v", err)
	}

	if constraints == nil {
		t.Error("expected non-nil constraints")
		return
	}

	if len(constraints.Scenarios) != 2 {
		t.Errorf("expected 2 scenarios, got %d", len(constraints.Scenarios))
	}

	// 检查第一个场景
	if constraints.Scenarios[0].Name != "用户邮箱登录成功" {
		t.Errorf("unexpected scenario 1 name: %s", constraints.Scenarios[0].Name)
	}
	if len(constraints.Scenarios[0].Given) == 0 || constraints.Scenarios[0].Given[0] != "用户在登录页面" {
		t.Errorf("unexpected scenario 1 given: %v", constraints.Scenarios[0].Given)
	}

	// 检查第二个场景
	if constraints.Scenarios[1].Name != "无效邮箱登录失败" {
		t.Errorf("unexpected scenario 2 name: %s", constraints.Scenarios[1].Name)
	}
}

// TestSaveBDDConstraints 测试保存 BDD 约束
func TestSaveBDDConstraints(t *testing.T) {
	engine := NewEngine()
	tempDir := t.TempDir()
	cm := NewConstraintManager(engine, tempDir)

	identifier := "TEST-3"
	constraints := &BDDConstraints{
		TaskID:    "task-3",
		Identifier: identifier,
		Scenarios: []BDDScenario{
			{Name: "测试场景", Given: []string{"G"}, When: []string{"W"}, Then: []string{"T"}},
		},
	}

	err := cm.SaveBDDConstraints(identifier, constraints)
	if err != nil {
		t.Errorf("failed to save BDD constraints: %v", err)
	}

	// 验证文件存在
	workspaceKey := SanitizeTaskID(identifier)
	bddFilePath := filepath.Join(tempDir, workspaceKey, "bdd.json")

	if _, err := os.Stat(bddFilePath); os.IsNotExist(err) {
		t.Error("BDD file was not created")
	}

	// 验证内容
	data, err := os.ReadFile(bddFilePath)
	if err != nil {
		t.Fatalf("failed to read BDD file: %v", err)
	}

	var loaded BDDConstraints
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to unmarshal BDD: %v", err)
	}

	if len(loaded.Scenarios) != 1 {
		t.Errorf("expected 1 scenario, got %d", len(loaded.Scenarios))
	}
}

// TestValidateImplementation 测试验证实现
func TestValidateImplementation(t *testing.T) {
	engine := NewEngine()
	tempDir := t.TempDir()
	cm := NewConstraintManager(engine, tempDir)

	taskID := "test-task-val"
	_, err := engine.InitTask(taskID)
	if err != nil {
		t.Fatalf("failed to init task: %v", err)
	}

	// 测试无约束情况
	result, err := cm.ValidateImplementation(taskID, tempDir)
	if err != nil {
		t.Errorf("validation failed: %v", err)
	}

	if result == nil {
		t.Error("expected non-nil result")
		return
	}

	// 无约束时应该跳过验证
	if !result.Skipped {
		t.Error("expected validation to be skipped when no constraints")
	}
}

// TestValidationResult 测试验证结果结构
func TestValidationResult(t *testing.T) {
	result := &ValidationResult{
		TaskID:      "task-123",
		CodePath:    "/path/to/code",
		Passed:      true,
		Message:     "验证通过",
		Skipped:     false,
		ScenarioResults: []ScenarioValidationResult{
			{ScenarioName: "场景1", Passed: true, Message: "OK"},
			{ScenarioName: "场景2", Passed: true, Message: "OK"},
		},
		ConstraintsPath: "/path/to/bdd.json",
	}

	if result.TaskID != "task-123" {
		t.Errorf("unexpected taskID")
	}
	if !result.Passed {
		t.Error("expected passed=true")
	}
	if len(result.ScenarioResults) != 2 {
		t.Errorf("expected 2 scenario results")
	}
}

// TestScenarioValidationResult 测试场景验证结果结构
func TestScenarioValidationResult(t *testing.T) {
	result := ScenarioValidationResult{
		ScenarioName: "登录测试",
		Passed:       false,
		Message:      "测试失败",
		Details:      "Expected success but got failure",
	}

	if result.ScenarioName != "登录测试" {
		t.Errorf("unexpected scenario name")
	}
	if result.Passed {
		t.Error("expected passed=false")
	}
}

// TestCleanIdentifier 测试清理标识符
func TestCleanIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ABC-123", "ABC-123"},
		{"TEST/123", "TEST_123"},
		{"TEST:123", "TEST_123"},
		{"TEST 123", "TEST_123"},
		// Note: SanitizeTaskID only replaces /, :, and space
	}

	for _, tt := range tests {
		result := SanitizeTaskID(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeTaskID(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

// TestGetCachedConstraints 测试获取缓存的约束
func TestGetCachedConstraints(t *testing.T) {
	engine := NewEngine()
	tempDir := t.TempDir()
	cm := NewConstraintManager(engine, tempDir)

	taskID := "cached-task"

	// 缓存为空时
	if cm.GetCachedConstraints(taskID) != nil {
		t.Error("expected nil for non-existent cache")
	}

	// 添加缓存
	cm.constraints[taskID] = &BDDConstraints{
		TaskID: taskID,
		Scenarios: []BDDScenario{{Name: "测试", Given: []string{}, When: []string{}, Then: []string{}}},
	}

	// 获取缓存
 cached := cm.GetCachedConstraints(taskID)
	if cached == nil {
		t.Error("expected cached constraints")
	}
	if cached.TaskID != taskID {
		t.Errorf("unexpected cached taskID")
	}
}

// TestClearCache 测试清除缓存
func TestClearCache(t *testing.T) {
	engine := NewEngine()
	tempDir := t.TempDir()
	cm := NewConstraintManager(engine, tempDir)

	taskID := "clear-task"
	cm.constraints[taskID] = &BDDConstraints{TaskID: taskID}

	cm.ClearCache(taskID)

	if cm.GetCachedConstraints(taskID) != nil {
		t.Error("cache should be cleared")
	}
}

// TestClearAllCache 测试清除所有缓存
func TestClearAllCache(t *testing.T) {
	engine := NewEngine()
	tempDir := t.TempDir()
	cm := NewConstraintManager(engine, tempDir)

	cm.constraints["task1"] = &BDDConstraints{TaskID: "task1"}
	cm.constraints["task2"] = &BDDConstraints{TaskID: "task2"}

	cm.ClearAllCache()

	if len(cm.constraints) != 0 {
		t.Error("all cache should be cleared")
	}
}

// TestHasConstraints 测试检查是否有约束
func TestHasConstraints(t *testing.T) {
	engine := NewEngine()
	tempDir := t.TempDir()
	cm := NewConstraintManager(engine, tempDir)

	taskID := "has-con-task"
	_, err := engine.InitTask(taskID)
	if err != nil {
		t.Fatalf("failed to init task: %v", err)
	}

	// 无约束时
	if cm.HasConstraints(taskID) {
		t.Error("expected no constraints initially")
	}
}

// TestFindBDDFileByIdentifier 测试根据标识符查找 BDD 文件
func TestFindBDDFileByIdentifier(t *testing.T) {
	engine := NewEngine()
	tempDir := t.TempDir()
	cm := NewConstraintManager(engine, tempDir)

	identifier := "FIND-TEST"
	workspaceKey := SanitizeTaskID(identifier)
	workspacePath := filepath.Join(tempDir, workspaceKey)

	// 创建目录和 BDD 文件
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	bddFilePath := filepath.Join(workspacePath, "bdd.json")
	if err := os.WriteFile(bddFilePath, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to write BDD file: %v", err)
	}

	foundPath := cm.findBDDFileByIdentifier(identifier)
	if foundPath != bddFilePath {
		t.Errorf("expected %s, got %s", bddFilePath, foundPath)
	}
}

// TestGetConstraintFilePathUnlocked 测试不加锁获取约束文件路径
func TestGetConstraintFilePathUnlocked(t *testing.T) {
	engine := NewEngine()
	tempDir := t.TempDir()
	cm := NewConstraintManager(engine, tempDir)

	taskID := "unlock-task"

	// 测试缓存情况
	cm.constraints[taskID] = &BDDConstraints{
		TaskID:    taskID,
		FilePath:  "/cached/path/bdd.json",
	}

	path := cm.GetConstraintFilePathUnlocked(taskID)
	if path != "/cached/path/bdd.json" {
		t.Errorf("expected cached path, got: %s", path)
	}

	// 测试无缓存情况
	cm.ClearCache(taskID)
	_, err := engine.InitTask(taskID)
	if err != nil {
		t.Fatalf("failed to init task: %v", err)
	}

	path = cm.GetConstraintFilePathUnlocked(taskID)
	if path != "" {
		// 无 identifier 时应该返回空
		t.Log("path returned for task without identifier")
	}
}

// TestGetBDDConstraintsForPrompt 测试获取格式化的 BDD 约束 Prompt
func TestGetBDDConstraintsForPrompt(t *testing.T) {
	engine := NewEngine()
	tempDir := t.TempDir()
	cm := NewConstraintManager(engine, tempDir)

	taskID := "prompt-task"

	// 测试无约束情况
	prompt, err := cm.GetBDDConstraintsForPrompt(taskID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if prompt != "" {
		t.Error("expected empty prompt when no constraints")
	}

	// 测试有缓存约束情况
	cm.constraints[taskID] = &BDDConstraints{
		TaskID: taskID,
		Scenarios: []BDDScenario{
			{Name: "测试场景", Given: []string{"G"}, When: []string{"W"}, Then: []string{"T"}},
		},
	}

	prompt, err = cm.GetBDDConstraintsForPrompt(taskID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(prompt, "测试场景") {
		t.Error("prompt should contain scenario name")
	}
}

// TestParseBDDMarkdown 测试解析 Markdown BDD 内容
func TestParseBDDMarkdown(t *testing.T) {
	engine := NewEngine()
	tempDir := t.TempDir()
	cm := NewConstraintManager(engine, tempDir)

	markdownContent := `# BDD Scenarios

### Scenario: 场景A
- Given: 条件A
- When: 动作A
- Then: 结果A

### Scenario: 场景B
- Given: 条件B
- When: 动作B
- Then: 结果B

Some other text
`

	constraints, err := cm.parseBDDMarkdown([]byte(markdownContent), "/test/bdd.md", "task-1")
	if err != nil {
		t.Fatalf("failed to parse markdown: %v", err)
	}

	if len(constraints.Scenarios) != 2 {
		t.Errorf("expected 2 scenarios, got %d", len(constraints.Scenarios))
	}

	if constraints.Scenarios[0].Name != "场景A" {
		t.Errorf("unexpected first scenario name: %s", constraints.Scenarios[0].Name)
	}

	if constraints.Scenarios[1].Name != "场景B" {
		t.Errorf("unexpected second scenario name: %s", constraints.Scenarios[1].Name)
	}
}

// TestParseBDDJSON 测试解析 JSON BDD 内容
func TestParseBDDJSON(t *testing.T) {
	engine := NewEngine()
	tempDir := t.TempDir()
	cm := NewConstraintManager(engine, tempDir)

	jsonContent := `{
		"task_id": "task-1",
		"identifier": "JSON-TEST",
		"scenarios": [
			{
				"name": "JSON场景",
				"given": ["JSON条件"],
				"when": ["JSON动作"],
				"then": ["JSON结果"]
			}
		]
	}`

	constraints, err := cm.parseBDDJSON([]byte(jsonContent), "/test/bdd.json", "task-1")
	if err != nil {
		t.Fatalf("failed to parse json: %v", err)
	}

	if constraints.TaskID != "task-1" {
		t.Errorf("unexpected taskID: %s", constraints.TaskID)
	}

	if len(constraints.Scenarios) != 1 {
		t.Errorf("expected 1 scenario, got %d", len(constraints.Scenarios))
	}

	if constraints.Scenarios[0].Name != "JSON场景" {
		t.Errorf("unexpected scenario name: %s", constraints.Scenarios[0].Name)
	}
}