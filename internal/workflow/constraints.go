// Package workflow 提供约束管理功能
package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BDDConstraints BDD 约束集合
type BDDConstraints struct {
	// TaskID 任务ID
	TaskID string `json:"task_id"`
	// Identifier 任务标识符
	Identifier string `json:"identifier"`
	// Feature 功能定义
	Feature BDDFeature `json:"feature"`
	// Scenarios BDD 场景列表
	Scenarios []BDDScenario `json:"scenarios"`
	// Summary 规则摘要
	Summary string `json:"summary,omitempty"`
	// ApprovedAt 审核通过时间
	ApprovedAt string `json:"approved_at,omitempty"`
	// FilePath BDD 文件路径
	FilePath string `json:"file_path,omitempty"`
}

// ConstraintManager 约束管理器，负责加载和管理 BDD 约束条件
type ConstraintManager struct {
	engine        *Engine
	workspaceRoot string
	constraints   map[string]*BDDConstraints // taskID -> BDDConstraints
}

// NewConstraintManager 创建新的约束管理器
func NewConstraintManager(engine *Engine, workspaceRoot string) *ConstraintManager {
	return &ConstraintManager{
		engine:        engine,
		workspaceRoot: workspaceRoot,
		constraints:   make(map[string]*BDDConstraints),
	}
}

// LoadBDDConstraints 加载任务的 BDD 约束条件
// 从工作空间中的 BDD 规则文件加载已审核通过的约束
// 返回 BDD 约束集合，如果文件不存在则返回 nil
func (cm *ConstraintManager) LoadBDDConstraints(taskID string) (*BDDConstraints, error) {
	// 先检查缓存（优先级最高，避免不必要的文件/工作流检查）
	if cached, exists := cm.constraints[taskID]; exists {
		return cached, nil
	}

	// 获取工作流
	workflow := cm.engine.GetWorkflow(taskID)
	if workflow == nil {
		return nil, ErrWorkflowNotFound
	}

	// 先查找 BDD 文件路径，如果不存在则直接返回 nil（无约束）
	bddFilePath := cm.findBDDFile(taskID)
	if bddFilePath == "" {
		return nil, nil // 没有 BDD 文件，返回 nil 表示无约束
	}

	// BDD 文件存在时，检查 BDD 评审阶段是否已完成
	bddStage := workflow.Stages[StageBDDReview]
	if bddStage == nil {
		return nil, ErrInvalidStage
	}

	// BDD 评审阶段必须是已完成状态
	if bddStage.Status != StatusCompleted {
		return nil, fmt.Errorf("BDD review stage not completed: status is %s", bddStage.Status)
	}

	// 加载 BDD 文件
	constraints, err := cm.loadBDDFile(bddFilePath, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to load BDD file: %w", err)
	}

	// 缓存约束
	cm.constraints[taskID] = constraints

	return constraints, nil
}

// LoadBDDConstraintsByIdentifier 根据任务标识符加载 BDD 约束
func (cm *ConstraintManager) LoadBDDConstraintsByIdentifier(identifier string) (*BDDConstraints, error) {
	// 查找 BDD 文件
	bddFilePath := cm.findBDDFileByIdentifier(identifier)
	if bddFilePath == "" {
		return nil, nil
	}

	// 加载 BDD 文件
	constraints, err := cm.loadBDDFile(bddFilePath, "")
	if err != nil {
		return nil, fmt.Errorf("failed to load BDD file: %w", err)
	}

	return constraints, nil
}

// findBDDFile 查找任务的 BDD 文件
// 搜索路径: workspaceRoot/<identifier>/bdd.json 或 bdd.md 或 .feature
func (cm *ConstraintManager) findBDDFile(taskID string) string {
	// 获取工作流以获取 identifier
	workflow := cm.engine.GetWorkflow(taskID)
	if workflow == nil {
		return ""
	}

	// 尝试从 metadata 中获取 identifier
	identifier := ""
	if workflow.Metadata != nil {
		identifier = workflow.Metadata["identifier"]
	}

	if identifier == "" {
		return ""
	}

	return cm.findBDDFileByIdentifier(identifier)
}

// findBDDFileByIdentifier 根据标识符查找 BDD 文件
func (cm *ConstraintManager) findBDDFileByIdentifier(identifier string) string {
	// 清理标识符作为工作空间键
	workspaceKey := SanitizeTaskID(identifier)
	workspacePath := filepath.Join(cm.workspaceRoot, workspaceKey)

	// 尝试查找 BDD 文件（优先 JSON 格式）
	bddFiles := []string{"bdd.json", "bdd.md", "BDD.json", "BDD.md", "bdd.feature"}
	for _, filename := range bddFiles {
		bddPath := filepath.Join(workspacePath, filename)
		if _, err := os.Stat(bddPath); err == nil {
			return bddPath
		}
	}

	return ""
}

// loadBDDFile 加载 BDD 文件内容
func (cm *ConstraintManager) loadBDDFile(filePath, taskID string) (*BDDConstraints, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// 根据文件扩展名解析
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".json":
		return cm.parseBDDJSON(data, filePath, taskID)
	case ".md":
		return cm.parseBDDMarkdown(data, filePath, taskID)
	case ".feature":
		return cm.parseBDDFeature(data, filePath, taskID)
	default:
		// 默认尝试 JSON 解析
		return cm.parseBDDJSON(data, filePath, taskID)
	}
}

// parseBDDJSON 解析 JSON 格式的 BDD 文件
func (cm *ConstraintManager) parseBDDJSON(data []byte, filePath, taskID string) (*BDDConstraints, error) {
	var constraints BDDConstraints
	if err := json.Unmarshal(data, &constraints); err != nil {
		return nil, fmt.Errorf("failed to parse BDD JSON: %w", err)
	}

	constraints.FilePath = filePath
	if taskID != "" {
		constraints.TaskID = taskID
	}

	return &constraints, nil
}

// parseBDDMarkdown 解析 Markdown 格式的 BDD 文件
func (cm *ConstraintManager) parseBDDMarkdown(data []byte, filePath, taskID string) (*BDDConstraints, error) {
	content := string(data)
	lines := strings.Split(content, "\n")

	constraints := &BDDConstraints{
		TaskID:    taskID,
		FilePath:  filePath,
		Scenarios: []BDDScenario{},
	}

	var currentScenario *BDDScenario

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 检测场景标题
		if strings.HasPrefix(line, "### Scenario:") || strings.HasPrefix(line, "## Scenario:") || strings.HasPrefix(line, "Scenario:") {
			// 保存上一个场景
			if currentScenario != nil && currentScenario.Name != "" {
				constraints.Scenarios = append(constraints.Scenarios, *currentScenario)
			}

			// 开始新场景
			name := strings.TrimPrefix(line, "### Scenario:")
			name = strings.TrimPrefix(name, "## Scenario:")
			name = strings.TrimPrefix(name, "Scenario:")
			name = strings.TrimSpace(name)

			currentScenario = &BDDScenario{
				Name:  name,
				Given: []string{},
				When:  []string{},
				Then:  []string{},
			}
			continue
		}

		if currentScenario == nil {
			continue
		}

		// 解析 Given/When/Then
		if strings.HasPrefix(line, "- Given:") || strings.HasPrefix(line, "Given:") {
			given := strings.TrimPrefix(line, "- Given:")
			given = strings.TrimPrefix(given, "Given:")
			currentScenario.Given = append(currentScenario.Given, strings.TrimSpace(given))
		} else if strings.HasPrefix(line, "- When:") || strings.HasPrefix(line, "When:") {
			when := strings.TrimPrefix(line, "- When:")
			when = strings.TrimPrefix(when, "When:")
			currentScenario.When = append(currentScenario.When, strings.TrimSpace(when))
		} else if strings.HasPrefix(line, "- Then:") || strings.HasPrefix(line, "Then:") {
			then := strings.TrimPrefix(line, "- Then:")
			then = strings.TrimPrefix(then, "Then:")
			currentScenario.Then = append(currentScenario.Then, strings.TrimSpace(then))
		}
	}

	// 保存最后一个场景
	if currentScenario != nil && currentScenario.Name != "" {
		constraints.Scenarios = append(constraints.Scenarios, *currentScenario)
	}

	return constraints, nil
}

// parseBDDFeature 解析 .feature 格式的 BDD 文件（Gherkin 格式）
func (cm *ConstraintManager) parseBDDFeature(data []byte, filePath, taskID string) (*BDDConstraints, error) {
	content := string(data)

	// 使用现有的 Gherkin 解析器
	rules, err := ParseGherkinContent(content)
	if err != nil {
		return nil, err
	}

	constraints := &BDDConstraints{
		TaskID:     taskID,
		FilePath:   filePath,
		Feature:    rules.Feature,
		Scenarios:  rules.Scenarios,
		Summary:    rules.Summary,
	}

	return constraints, nil
}

// FormatConstraintsForPrompt 格式化 BDD 约束为 Prompt 注入格式
// 返回格式化的字符串，可直接注入到 Agent Prompt 中
func (cm *ConstraintManager) FormatConstraintsForPrompt(constraints *BDDConstraints) string {
	if constraints == nil || len(constraints.Scenarios) == 0 {
		return ""
	}

	var builder strings.Builder

	builder.WriteString("## BDD 验收标准\n\n")
	builder.WriteString("以下是必须通过的 BDD 场景：\n\n")

	for i, scenario := range constraints.Scenarios {
		builder.WriteString(fmt.Sprintf("### Scenario %d: %s\n", i+1, scenario.Name))

		if len(scenario.Given) > 0 {
			for j, given := range scenario.Given {
				if j == 0 {
					builder.WriteString(fmt.Sprintf("- Given: %s\n", given))
				} else {
					builder.WriteString(fmt.Sprintf("  And: %s\n", given))
				}
			}
		}
		if len(scenario.When) > 0 {
			for j, when := range scenario.When {
				if j == 0 {
					builder.WriteString(fmt.Sprintf("- When: %s\n", when))
				} else {
					builder.WriteString(fmt.Sprintf("  And: %s\n", when))
				}
			}
		}
		if len(scenario.Then) > 0 {
			for j, then := range scenario.Then {
				if j == 0 {
					builder.WriteString(fmt.Sprintf("- Then: %s\n", then))
				} else {
					builder.WriteString(fmt.Sprintf("  And: %s\n", then))
				}
			}
		}

		builder.WriteString("\n")
	}

	builder.WriteString("**重要**: 你的实现必须使上述所有场景的测试通过。\n")

	return builder.String()
}

// FormatConstraintsForPromptWithScenarios 直接格式化场景列表
func (cm *ConstraintManager) FormatConstraintsForPromptWithScenarios(scenarios []BDDScenario) string {
	if len(scenarios) == 0 {
		return ""
	}

	constraints := &BDDConstraints{Scenarios: scenarios}
	return cm.FormatConstraintsForPrompt(constraints)
}

// ValidateImplementation 验证实现是否符合 BDD 约束
// 检查指定路径的代码是否能通过 BDD 场景测试
// 返回验证结果和错误信息
func (cm *ConstraintManager) ValidateImplementation(taskID string, codePath string) (*ValidationResult, error) {
	// 加载 BDD 约束
	constraints, err := cm.LoadBDDConstraints(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to load BDD constraints: %w", err)
	}

	if constraints == nil {
		return &ValidationResult{
			TaskID:          taskID,
			CodePath:        codePath,
			Passed:          true,
			Message:         "No BDD constraints found, validation skipped",
			Skipped:         true,
			ScenarioResults: []ScenarioValidationResult{},
		}, nil
	}

	// 检查代码路径是否存在
	if _, err := os.Stat(codePath); os.IsNotExist(err) {
		return &ValidationResult{
			TaskID:          taskID,
			CodePath:        codePath,
			Passed:          false,
			Message:         "Code path does not exist",
			Skipped:         false,
			ScenarioResults: []ScenarioValidationResult{},
		}, nil
	}

	// 执行验证（实际实现中需要调用测试运行器）
	results := cm.validateScenarios(constraints.Scenarios, codePath)

	// 计算总体结果
	allPassed := true
	for _, r := range results {
		if !r.Passed {
			allPassed = false
			break
		}
	}

	message := "验证通过"
	if !allPassed {
		message = "部分场景验证失败"
	}

	return &ValidationResult{
		TaskID:           taskID,
		CodePath:         codePath,
		Passed:           allPassed,
		Message:          message,
		Skipped:          false,
		ScenarioResults:  results,
		ConstraintsPath:  constraints.FilePath,
	}, nil
}

// validateScenarios 验证每个场景
func (cm *ConstraintManager) validateScenarios(scenarios []BDDScenario, codePath string) []ScenarioValidationResult {
	results := []ScenarioValidationResult{}

	for _, scenario := range scenarios {
		// 实际实现中，这里需要:
		// 1. 解析场景的 Given/When/Then
		// 2. 运行对应的测试
		// 3. 收集测试结果

		// 格式化详情
		details := fmt.Sprintf("Given: %v, When: %v, Then: %v", scenario.Given, scenario.When, scenario.Then)

		// 当前为模拟实现，标记所有场景为待验证
		result := ScenarioValidationResult{
			ScenarioName: scenario.Name,
			Passed:       false, // 需要实际测试运行
			Message:      "Pending actual test execution",
			Details:      details,
		}
		results = append(results, result)
	}

	return results
}

// ValidationResult 验证结果
type ValidationResult struct {
	// TaskID 任务ID
	TaskID string `json:"task_id"`
	// CodePath 代码路径
	CodePath string `json:"code_path"`
	// Passed 是否通过验证
	Passed bool `json:"passed"`
	// Message 结果消息
	Message string `json:"message"`
	// Skipped 是否跳过验证
	Skipped bool `json:"skipped"`
	// ScenarioResults 各场景验证结果
	ScenarioResults []ScenarioValidationResult `json:"scenario_results"`
	// ConstraintsPath BDD 约束文件路径
	ConstraintsPath string `json:"constraints_path,omitempty"`
}

// ScenarioValidationResult 单个场景的验证结果
type ScenarioValidationResult struct {
	// ScenarioName 场景名称
	ScenarioName string `json:"scenario_name"`
	// Passed 是否通过
	Passed bool `json:"passed"`
	// Message 结果消息
	Message string `json:"message"`
	// Details 详细信息
	Details string `json:"details,omitempty"`
}

// GetCachedConstraints 获取缓存的约束条件
func (cm *ConstraintManager) GetCachedConstraints(taskID string) *BDDConstraints {
	return cm.constraints[taskID]
}

// ClearCache 清除缓存
func (cm *ConstraintManager) ClearCache(taskID string) {
	delete(cm.constraints, taskID)
}

// ClearAllCache 清除所有缓存
func (cm *ConstraintManager) ClearAllCache() {
	cm.constraints = make(map[string]*BDDConstraints)
}

// SaveBDDConstraints 保存 BDD 约束到文件
func (cm *ConstraintManager) SaveBDDConstraints(identifier string, constraints *BDDConstraints) error {
	// 计算保存路径
	workspaceKey := SanitizeTaskID(identifier)
	workspacePath := filepath.Join(cm.workspaceRoot, workspaceKey)

	// 确保目录存在
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}

	// 保存为 JSON 文件
	bddFilePath := filepath.Join(workspacePath, "bdd.json")
	data, err := json.MarshalIndent(constraints, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal constraints: %w", err)
	}

	if err := os.WriteFile(bddFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write BDD file: %w", err)
	}

	constraints.FilePath = bddFilePath

	// 缓存约束
	if constraints.TaskID != "" {
		cm.constraints[constraints.TaskID] = constraints
	}

	return nil
}

// HasConstraints 检查任务是否有 BDD 约束
func (cm *ConstraintManager) HasConstraints(taskID string) bool {
	constraints, err := cm.LoadBDDConstraints(taskID)
	if err != nil {
		return false
	}
	return constraints != nil && len(constraints.Scenarios) > 0
}

// GetConstraintFilePath 获取约束文件路径
func (cm *ConstraintManager) GetConstraintFilePath(taskID string) string {
	constraints, err := cm.LoadBDDConstraints(taskID)
	if err != nil || constraints == nil {
		return ""
	}
	return constraints.FilePath
}

// GetConstraintFilePathUnlocked 获取约束文件路径（不加锁，用于 Engine 内部调用）
func (cm *ConstraintManager) GetConstraintFilePathUnlocked(taskID string) string {
	// 首先检查缓存
	if cached, exists := cm.constraints[taskID]; exists {
		return cached.FilePath
	}

	// 获取工作流信息以获取 identifier
	workflow := cm.engine.GetWorkflow(taskID)
	if workflow == nil {
		return ""
	}

	// 尝试从 metadata 中获取 identifier
	identifier := ""
	if workflow.Metadata != nil {
		identifier = workflow.Metadata["identifier"]
	}

	// 如果没有 identifier，尝试从缓存中找对应的文件
	if identifier == "" {
		return ""
	}

	// 查找 BDD 文件
	return cm.findBDDFileByIdentifier(identifier)
}

// GetBDDConstraintsForPrompt 获取格式化的 BDD 约束 Prompt
// 这是一个便捷方法，直接返回可注入到 Agent Prompt 的约束文本
func (cm *ConstraintManager) GetBDDConstraintsForPrompt(taskID string) (string, error) {
	constraints, err := cm.LoadBDDConstraints(taskID)
	if err != nil {
		// 如果工作流不存在或阶段未完成，返回空字符串（无约束）
		if err == ErrWorkflowNotFound || err == ErrInvalidStage {
			return "", nil
		}
		return "", err
	}

	if constraints == nil {
		return "", nil
	}

	return cm.FormatConstraintsForPrompt(constraints), nil
}

// ConstraintsFromBDDRules 从 BDDRules 创建 BDDConstraints
func ConstraintsFromBDDRules(rules *BDDRules, taskID, identifier string) *BDDConstraints {
	if rules == nil {
		return nil
	}

	return &BDDConstraints{
		TaskID:     taskID,
		Identifier: identifier,
		Feature:    rules.Feature,
		Scenarios:  rules.Scenarios,
		Summary:    rules.Summary,
	}
}

// ToBDDRules 转换为 BDDRules
func (c *BDDConstraints) ToBDDRules() *BDDRules {
	return &BDDRules{
		Feature:   c.Feature,
		Scenarios: c.Scenarios,
		Summary:   c.Summary,
	}
}