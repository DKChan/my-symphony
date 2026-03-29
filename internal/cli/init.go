// Package cli 提供命令行界面功能
package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dministrator/symphony/internal/config"
	"gopkg.in/yaml.v3"
)

// InitOptions 包含 init 命令的选项
type InitOptions struct {
	TrackerType string
	AgentType   string
	ProjectPath string
	NonInteractive bool
}

// InitCommand 实现 symphony init 命令
type InitCommand struct {
	options InitOptions
	scanner *bufio.Scanner
}

// NewInitCommand 创建新的 init 命令
func NewInitCommand(opts InitOptions) *InitCommand {
	return &InitCommand{
		options: opts,
		scanner: bufio.NewScanner(os.Stdin),
	}
}

// Run 执行 init 命令
func (c *InitCommand) Run() error {
	fmt.Println("========================================")
	fmt.Println("  Symphony 初始化向导")
	fmt.Println("========================================")
	fmt.Println()

	// 如果提供了项目路径，切换到该目录
	if c.options.ProjectPath != "" {
		if err := os.Chdir(c.options.ProjectPath); err != nil {
			return fmt.Errorf("init.dir_access: 无法访问目录 %s: %w", c.options.ProjectPath, err)
		}
	}

	// 检查 .sym 目录是否已存在
	symDir := ".sym"
	if _, err := os.Stat(symDir); err == nil {
		fmt.Printf("警告: .sym 目录已存在\n")
		if !c.promptConfirm("是否覆盖现有配置?", false) {
			fmt.Println("初始化已取消")
			return nil
		}
	}

	// 收集用户输入
	trackerType := c.options.TrackerType
	if trackerType == "" {
		trackerType = c.promptSelect(
			"请选择 Tracker 类型",
			[]string{"linear", "github", "mock", "beads"},
			"mock",
		)
	}

	agentType := c.options.AgentType
	if agentType == "" {
		agentType = c.promptSelect(
			"请选择 AI Agent CLI 类型",
			[]string{"codex", "claude", "opencode"},
			"codex",
		)
	}

	// 收集 tracker 特定配置
	trackerConfig := c.collectTrackerConfig(trackerType)

	// 生成配置
	cfg := c.generateConfig(trackerType, agentType, trackerConfig)

	// 创建目录结构
	if err := c.createDirectoryStructure(symDir, cfg); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("  初始化完成!")
	fmt.Println("========================================")
	fmt.Printf("配置文件已生成: %s/config.yaml\n", symDir)
	fmt.Printf("工作流模板已生成: %s/workflow.md\n", symDir)
	fmt.Printf("Prompt 模板已生成: %s/prompts/\n", symDir)
	fmt.Println()
	fmt.Println("下一步:")
	fmt.Println("  1. 编辑 .sym/config.yaml 配置您的 tracker 凭证")
	fmt.Println("  2. 编辑 .sym/workflow.md 自定义工作流提示")
	fmt.Println("  3. 运行 'symphony start -w .sym/workflow.md' 启动服务")

	return nil
}

// trackerConfigData 包含 tracker 特定配置数据
type trackerConfigData struct {
	apiKey      string
	projectSlug string
	repo        string
	endpoint    string
}

// collectTrackerConfig 收集 tracker 特定配置
func (c *InitCommand) collectTrackerConfig(trackerType string) *trackerConfigData {
	data := &trackerConfigData{}

	switch trackerType {
	case "linear":
		fmt.Println("\nLinear Tracker 配置:")
		data.apiKey = c.promptInput("请输入 Linear API Key (或设置 LINEAR_API_KEY 环境变量)", "")
		data.projectSlug = c.promptInput("请输入 Project Slug", "")
		data.endpoint = "https://api.linear.app/graphql"

	case "github":
		fmt.Println("\nGitHub Tracker 配置:")
		data.apiKey = c.promptInput("请输入 GitHub Token (或设置 GITHUB_TOKEN 环境变量)", "")
		data.repo = c.promptInput("请输入仓库 (格式: owner/repo)", "")
		data.endpoint = "https://api.github.com"

	case "beads":
		fmt.Println("\nBeads Tracker 配置 (本地 CLI tracker):")
		data.endpoint = c.promptInput("请输入 Beads 端点 (可选，回车跳过)", "")

	default:
		// mock 不需要额外配置
	}

	return data
}

// generateConfig 生成配置
func (c *InitCommand) generateConfig(trackerType, agentType string, trackerData *trackerConfigData) *config.Config {
	cfg := config.DefaultConfig()

	// 更新 tracker 配置
	cfg.Tracker.Kind = trackerType
	if trackerData.endpoint != "" {
		cfg.Tracker.Endpoint = trackerData.endpoint
	}
	if trackerData.apiKey != "" {
		cfg.Tracker.APIKey = trackerData.apiKey
	}
	if trackerData.projectSlug != "" {
		cfg.Tracker.ProjectSlug = trackerData.projectSlug
	}
	if trackerData.repo != "" {
		cfg.Tracker.Repo = trackerData.repo
	}

	// 更新 agent 配置
	cfg.Agent.Kind = agentType

	return cfg
}

// createDirectoryStructure 创建目录结构和文件
func (c *InitCommand) createDirectoryStructure(symDir string, cfg *config.Config) error {
	// 创建 .sym 目录
	if err := os.MkdirAll(symDir, 0755); err != nil {
		return fmt.Errorf("init.dir_create: 无法创建目录 %s: %w", symDir, err)
	}

	// 创建 prompts 目录
	promptsDir := filepath.Join(symDir, "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		return fmt.Errorf("init.dir_create: 无法创建目录 %s: %w", promptsDir, err)
	}

	// 生成 config.yaml
	if err := c.generateConfigYAML(symDir, cfg); err != nil {
		return err
	}

	// 生成 workflow.md
	if err := c.generateWorkflowMD(symDir); err != nil {
		return err
	}

	// 生成 prompt 模板文件
	if err := c.generatePromptTemplates(promptsDir); err != nil {
		return err
	}

	return nil
}

// generateConfigYAML 生成 config.yaml 文件
func (c *InitCommand) generateConfigYAML(symDir string, cfg *config.Config) error {
	configPath := filepath.Join(symDir, "config.yaml")

	// 创建可序列化的配置结构
	configData := c.buildConfigMap(cfg)

	data, err := yaml.Marshal(configData)
	if err != nil {
		return fmt.Errorf("init.config_serialize: 无法序列化配置: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("init.config_write: 无法写入配置文件: %w", err)
	}

	return nil
}

// buildConfigMap 构建配置映射（用于 YAML 序列化）
func (c *InitCommand) buildConfigMap(cfg *config.Config) map[string]interface{} {
	return map[string]interface{}{
		"tracker": map[string]interface{}{
			"kind":            cfg.Tracker.Kind,
			"endpoint":        cfg.Tracker.Endpoint,
			"api_key":         c.maskAPIKey(cfg.Tracker.APIKey),
			"project_slug":    cfg.Tracker.ProjectSlug,
			"repo":            cfg.Tracker.Repo,
			"active_states":   cfg.Tracker.ActiveStates,
			"terminal_states": cfg.Tracker.TerminalStates,
		},
		"polling": map[string]interface{}{
			"interval_ms": cfg.Polling.IntervalMs,
		},
		"workspace": map[string]interface{}{
			"root": cfg.Workspace.Root,
		},
		"hooks": map[string]interface{}{
			"timeout_ms": cfg.Hooks.TimeoutMs,
		},
		"agent": map[string]interface{}{
			"kind":                    cfg.Agent.Kind,
			"max_concurrent_agents":   cfg.Agent.MaxConcurrentAgents,
			"max_turns":               cfg.Agent.MaxTurns,
			"max_retry_backoff_ms":    cfg.Agent.MaxRetryBackoffMs,
			"max_concurrent_agents_by_state": cfg.Agent.MaxConcurrentAgentsByState,
		},
		"clarification": map[string]interface{}{
			"max_rounds": 5,
		},
		"execution": map[string]interface{}{
			"max_retries": 3,
		},
		"codex": map[string]interface{}{
			"command":         cfg.Codex.Command,
			"turn_timeout_ms": cfg.Codex.TurnTimeoutMs,
			"read_timeout_ms": cfg.Codex.ReadTimeoutMs,
			"stall_timeout_ms": cfg.Codex.StallTimeoutMs,
		},
	}
}

// maskAPIKey 遮蔽 API Key（如果存在）
func (c *InitCommand) maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

// generateWorkflowMD 生成 workflow.md 文件
func (c *InitCommand) generateWorkflowMD(symDir string) error {
	workflowPath := filepath.Join(symDir, "workflow.md")
	workflowContent := `---
tracker:
  kind: mock
polling:
  interval_ms: 30000
agent:
  kind: codex
  max_concurrent_agents: 5
  max_turns: 20
---

# 工作流提示模板

你是一个专业的软件开发助手。请根据以下任务描述进行开发：

## 任务信息

**任务ID**: {{.ID}}
**标题**: {{.Title}}
**描述**: {{.Description}}
**优先级**: {{.Priority}}

## 指导原则

1. 仔细阅读任务描述，理解需求
2. 编写清晰、可维护的代码
3. 确保代码有适当的测试覆盖
4. 遵循项目的编码规范

## 输出要求

请提供：
1. 实现方案说明
2. 关键代码片段
3. 测试建议
`

	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		return fmt.Errorf("init.workflow_write: 无法写入工作流文件: %w", err)
	}

	return nil
}

// generatePromptTemplates 生成 prompt 模板文件
func (c *InitCommand) generatePromptTemplates(promptsDir string) error {
	templates := map[string]string{
		"clarification.md": `# 需求澄清提示模板

请根据以下任务信息，提出澄清问题以更好地理解需求。

## 任务信息

**任务ID**: {{.ID}}
**标题**: {{.Title}}
**描述**: {{.Description}}

## 澄清问题

请针对以下方面提出问题：

1. **功能范围**: 任务的具体功能边界是什么？
2. **技术约束**: 是否有特定的技术栈要求？
3. **验收标准**: 如何验证任务完成？
4. **依赖关系**: 是否依赖其他任务或系统？
5. **风险评估**: 有哪些潜在风险？

## 输出格式

请以结构化的方式提出问题，便于后续跟进。
`,
		"bdd.md": `# 行为驱动开发 (BDD) 提示模板

请根据以下任务信息，编写 BDD 场景。

## 任务信息

**任务ID**: {{.ID}}
**标题**: {{.Title}}
**描述**: {{.Description}}

## BDD 场景模板

请使用 Gherkin 语法编写场景：

` + "```gherkin" + `
Feature: [功能名称]

  Background:
    Given [前置条件]

  Scenario: [场景名称]
    Given [前置条件]
    When [触发动作]
    Then [预期结果]

  Scenario: [另一个场景]
    Given [前置条件]
    When [触发动作]
    Then [预期结果]
` + "```" + `

## 场景覆盖

请确保覆盖：
1. 正常流程 (Happy Path)
2. 异常处理 (Error Handling)
3. 边界条件 (Edge Cases)
`,
		"architecture.md": `# 架构设计提示模板

请根据以下任务信息，设计系统架构。

## 任务信息

**任务ID**: {{.ID}}
**标题**: {{.Title}}
**描述**: {{.Description}}

## 架构设计要求

请提供以下架构设计内容：

### 1. 系统概览
- 整体架构图
- 主要组件及其职责

### 2. 数据流
- 数据输入/输出
- 数据处理流程

### 3. 技术选型
- 编程语言和框架
- 数据库选择
- 第三方服务集成

### 4. 接口设计
- API 接口定义
- 数据模型

### 5. 非功能性需求
- 性能考虑
- 安全措施
- 可扩展性设计
`,
		"implementation.md": `# 实现指南提示模板

请根据以下任务信息和架构设计，提供实现指南。

## 任务信息

**任务ID**: {{.ID}}
**标题**: {{.Title}}
**描述**: {{.Description}}

## 实现指南

### 1. 代码结构

请按照以下结构组织代码：

` + "```" + `
src/
├── api/          # API 层
├── service/      # 业务逻辑层
├── repository/   # 数据访问层
├── model/        # 数据模型
└── util/         # 工具函数
` + "```" + `

### 2. 编码规范

1. 使用清晰的命名
2. 编写注释和文档
3. 遵循 SOLID 原则
4. 编写单元测试

### 3. 实现步骤

请按以下顺序实现：

1. **数据模型定义**
2. **接口定义**
3. **业务逻辑实现**
4. **API 端点实现**
5. **测试用例编写**

### 4. 测试要求

- 单元测试覆盖率 >= 80%
- 集成测试覆盖主要流程
- 使用 mock 进行依赖隔离
`,
		"verification.md": `# 验证检查提示模板

请根据以下任务信息，执行验证检查。

## 任务信息

**任务ID**: {{.ID}}
**标题**: {{.Title}}
**描述**: {{.Description}}

## 验证检查清单

### 1. 功能验证

- [ ] 所有功能点已实现
- [ ] 边界条件已处理
- [ ] 错误处理完整

### 2. 代码质量

- [ ] 代码风格一致
- [ ] 注释清晰完整
- [ ] 无明显代码异味

### 3. 测试覆盖

- [ ] 单元测试通过
- [ ] 集成测试通过
- [ ] 测试覆盖率达标

### 4. 性能验证

- [ ] 响应时间满足要求
- [ ] 资源使用合理
- [ ] 无内存泄漏

### 5. 安全检查

- [ ] 输入验证完整
- [ ] 无安全漏洞
- [ ] 敏感数据处理正确

### 6. 文档完整性

- [ ] API 文档已更新
- [ ] README 已更新
- [ ] 变更日志已更新

## 验证结果

请提供详细的验证报告。
`,
	}

	for filename, content := range templates {
		filePath := filepath.Join(promptsDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("init.prompt_write: 无法写入 prompt 文件 %s: %w", filename, err)
		}
	}

	return nil
}

// promptSelect 提供选择提示
func (c *InitCommand) promptSelect(prompt string, options []string, defaultOption string) string {
	fmt.Printf("%s:\n", prompt)
	for i, opt := range options {
		fmt.Printf("  %d. %s", i+1, opt)
		if opt == defaultOption {
			fmt.Print(" (默认)")
		}
		fmt.Println()
	}

	for {
		fmt.Printf("请输入选项序号 [1-%d]: ", len(options))
		if !c.scanner.Scan() {
			return defaultOption
		}

		input := strings.TrimSpace(c.scanner.Text())
		if input == "" {
			return defaultOption
		}

		// 检查是否直接输入了选项值
		for _, opt := range options {
			if strings.EqualFold(input, opt) {
				return opt
			}
		}

		// 尝试解析为序号
		var choice int
		if _, err := fmt.Sscanf(input, "%d", &choice); err == nil {
			if choice >= 1 && choice <= len(options) {
				return options[choice-1]
			}
		}

		fmt.Printf("无效输入，请输入 1-%d 之间的数字或选项名称\n", len(options))
	}
}

// promptInput 提供输入提示
func (c *InitCommand) promptInput(prompt, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}

	if !c.scanner.Scan() {
		return defaultValue
	}

	input := strings.TrimSpace(c.scanner.Text())
	if input == "" {
		return defaultValue
	}
	return input
}

// promptConfirm 提供确认提示
func (c *InitCommand) promptConfirm(prompt string, defaultValue bool) bool {
	defaultHint := "n"
	if defaultValue {
		defaultHint = "y"
	}

	fmt.Printf("%s [%s/N]: ", prompt, defaultHint)
	if !c.scanner.Scan() {
		return defaultValue
	}

	input := strings.ToLower(strings.TrimSpace(c.scanner.Text()))
	if input == "" {
		return defaultValue
	}

	return input == "y" || input == "yes"
}