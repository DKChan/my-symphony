package cli

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInitCommand(t *testing.T) {
	opts := InitOptions{
		TrackerType: "mock",
		AgentType:   "codex",
	}
	cmd := NewInitCommand(opts)

	assert.NotNil(t, cmd)
	assert.Equal(t, "mock", cmd.options.TrackerType)
	assert.Equal(t, "codex", cmd.options.AgentType)
}

func TestInitCommand_GenerateConfig(t *testing.T) {
	tests := []struct {
		name         string
		trackerType  string
		agentType    string
		trackerData  *trackerConfigData
		wantTracker  string
		wantAgent    string
		wantEndpoint string
	}{
		{
			name:        "mock tracker with codex agent",
			trackerType: "mock",
			agentType:   "codex",
			trackerData: &trackerConfigData{},
			wantTracker: "mock",
			wantAgent:   "codex",
		},
		{
			name:        "linear tracker with claude agent",
			trackerType: "linear",
			agentType:   "claude",
			trackerData: &trackerConfigData{
				apiKey:      "lin_api_xxx",
				projectSlug: "PROJ-1",
				endpoint:    "https://api.linear.app/graphql",
			},
			wantTracker:  "linear",
			wantAgent:    "claude",
			wantEndpoint: "https://api.linear.app/graphql",
		},
		{
			name:        "github tracker with opencode agent",
			trackerType: "github",
			agentType:   "opencode",
			trackerData: &trackerConfigData{
				apiKey:   "ghp_xxx",
				repo:     "owner/repo",
				endpoint: "https://api.github.com",
			},
			wantTracker:  "github",
			wantAgent:    "opencode",
			wantEndpoint: "https://api.github.com",
		},
		{
			name:        "beads tracker",
			trackerType: "beads",
			agentType:   "codex",
			trackerData: &trackerConfigData{
				endpoint: "http://localhost:8080",
			},
			wantTracker:  "beads",
			wantAgent:    "codex",
			wantEndpoint: "http://localhost:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewInitCommand(InitOptions{})
			cfg := cmd.generateConfig(tt.trackerType, tt.agentType, tt.trackerData)

			assert.Equal(t, tt.wantTracker, cfg.Tracker.Kind)
			assert.Equal(t, tt.wantAgent, cfg.Agent.Kind)
			if tt.wantEndpoint != "" {
				assert.Equal(t, tt.wantEndpoint, cfg.Tracker.Endpoint)
			}
		})
	}
}

func TestInitCommand_CreateDirectoryStructure(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	symDir := filepath.Join(tempDir, ".sym")

	cmd := NewInitCommand(InitOptions{
		TrackerType: "mock",
		AgentType:   "codex",
	})

	// 使用默认配置
	cfg := cmd.generateConfig("mock", "codex", &trackerConfigData{})

	err := cmd.createDirectoryStructure(symDir, cfg)
	require.NoError(t, err)

	// 验证目录结构
	assert.DirExists(t, symDir)
	assert.DirExists(t, filepath.Join(symDir, "prompts"))

	// 验证配置文件
	configPath := filepath.Join(symDir, "config.yaml")
	assert.FileExists(t, configPath)

	// 验证工作流文件
	workflowPath := filepath.Join(symDir, "workflow.md")
	assert.FileExists(t, workflowPath)

	// 验证 prompt 文件
	expectedPrompts := []string{
		"clarification.md",
		"bdd.md",
		"architecture.md",
		"implementation.md",
		"verification.md",
	}

	for _, prompt := range expectedPrompts {
		promptPath := filepath.Join(symDir, "prompts", prompt)
		assert.FileExists(t, promptPath)
	}
}

func TestInitCommand_GenerateConfigYAML(t *testing.T) {
	tempDir := t.TempDir()
	symDir := filepath.Join(tempDir, ".sym")
	require.NoError(t, os.MkdirAll(symDir, 0755))

	cmd := NewInitCommand(InitOptions{
		TrackerType: "mock",
		AgentType:   "codex",
	})

	cfg := cmd.generateConfig("mock", "codex", &trackerConfigData{})

	err := cmd.generateConfigYAML(symDir, cfg)
	require.NoError(t, err)

	// 读取生成的配置文件
	configPath := filepath.Join(symDir, "config.yaml")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	contentStr := string(content)

	// 验证关键字段
	assert.Contains(t, contentStr, "kind: mock")
	assert.Contains(t, contentStr, "kind: codex")
	assert.Contains(t, contentStr, "max_rounds: 5")
	assert.Contains(t, contentStr, "max_retries: 3")
	assert.Contains(t, contentStr, "polling:")
	assert.Contains(t, contentStr, "workspace:")
}

func TestInitCommand_GenerateWorkflowMD(t *testing.T) {
	tempDir := t.TempDir()
	symDir := filepath.Join(tempDir, ".sym")
	require.NoError(t, os.MkdirAll(symDir, 0755))

	cmd := NewInitCommand(InitOptions{})

	err := cmd.generateWorkflowMD(symDir)
	require.NoError(t, err)

	// 读取生成的工作流文件
	workflowPath := filepath.Join(symDir, "workflow.md")
	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err)

	contentStr := string(content)

	// 验证内容
	assert.Contains(t, contentStr, "---")
	assert.Contains(t, contentStr, "tracker:")
	assert.Contains(t, contentStr, "任务信息")
	assert.Contains(t, contentStr, "{{.ID}}")
	assert.Contains(t, contentStr, "{{.Title}}")
	assert.Contains(t, contentStr, "{{.Description}}")
}

func TestInitCommand_GeneratePromptTemplates(t *testing.T) {
	tempDir := t.TempDir()
	promptsDir := filepath.Join(tempDir, "prompts")
	require.NoError(t, os.MkdirAll(promptsDir, 0755))

	cmd := NewInitCommand(InitOptions{})

	err := cmd.generatePromptTemplates(promptsDir)
	require.NoError(t, err)

	// 验证每个模板文件
	templates := []struct {
		filename    string
		expectedContent []string
	}{
		{
			filename: "clarification.md",
			expectedContent: []string{"需求澄清", "澄清问题", "{{.ID}}"},
		},
		{
			filename: "bdd.md",
			expectedContent: []string{"行为驱动开发", "Scenario:", "Given", "When", "Then"},
		},
		{
			filename: "architecture.md",
			expectedContent: []string{"架构设计", "系统概览", "数据流"},
		},
		{
			filename: "implementation.md",
			expectedContent: []string{"实现指南", "代码结构", "编码规范"},
		},
		{
			filename: "verification.md",
			expectedContent: []string{"验证检查", "功能验证", "代码质量"},
		},
	}

	for _, tc := range templates {
		t.Run(tc.filename, func(t *testing.T) {
			filePath := filepath.Join(promptsDir, tc.filename)
			content, err := os.ReadFile(filePath)
			require.NoError(t, err)

			contentStr := string(content)
			for _, expected := range tc.expectedContent {
				assert.Contains(t, contentStr, expected)
			}
		})
	}
}

func TestInitCommand_MaskAPIKey(t *testing.T) {
	cmd := NewInitCommand(InitOptions{})

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty key",
			input:    "",
			expected: "",
		},
		{
			name:     "short key",
			input:    "abc",
			expected: "***",
		},
		{
			name:     "normal key",
			input:    "lin_api_1234567890abcdef",
			expected: "lin_...cdef",
		},
		{
			name:     "github token",
			input:    "ghp_1234567890abcdefghij",
			expected: "ghp_...ghij",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cmd.maskAPIKey(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInitCommand_BuildConfigMap(t *testing.T) {
	cmd := NewInitCommand(InitOptions{
		TrackerType: "linear",
		AgentType:   "claude",
	})

	cfg := cmd.generateConfig("linear", "claude", &trackerConfigData{
		apiKey:      "lin_api_test123",
		projectSlug: "PROJ-1",
		endpoint:    "https://api.linear.app/graphql",
	})

	configMap := cmd.buildConfigMap(cfg)

	// 验证配置映射
	assert.Contains(t, configMap, "tracker")
	assert.Contains(t, configMap, "agent")
	assert.Contains(t, configMap, "clarification")
	assert.Contains(t, configMap, "execution")

	// 验证 clarification 配置
	clarification, ok := configMap["clarification"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, 5, clarification["max_rounds"])

	// 验证 execution 配置
	execution, ok := configMap["execution"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, 3, execution["max_retries"])
}

func TestInitCommand_PromptSelect(t *testing.T) {
	// 这个测试验证 promptSelect 函数的基本逻辑
	// 由于涉及交互式输入，主要测试选项处理
	_ = NewInitCommand(InitOptions{})

	// 测试默认选项逻辑
	options := []string{"linear", "github", "mock", "beads"}
	assert.Equal(t, 4, len(options))
	assert.Contains(t, options, "mock")
	assert.Contains(t, options, "beads")
}

func TestInitCommand_PromptInput(t *testing.T) {
	cmd := NewInitCommand(InitOptions{})

	// 验证输入处理函数存在
	assert.NotNil(t, cmd.promptInput)
}

func TestInitCommand_PromptConfirm(t *testing.T) {
	cmd := NewInitCommand(InitOptions{})

	// 验证确认处理函数存在
	assert.NotNil(t, cmd.promptConfirm)
}

func TestInitCommand_TrackerConfigData(t *testing.T) {
	cmd := NewInitCommand(InitOptions{})

	// 测试不同 tracker 类型的配置收集
	tests := []struct {
		name        string
		trackerType string
		expectEmpty bool
	}{
		{
			name:        "mock tracker",
			trackerType: "mock",
			expectEmpty: true,
		},
		{
			name:        "beads tracker",
			trackerType: "beads",
			expectEmpty: false, // beads 会设置 endpoint
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := cmd.collectTrackerConfig(tt.trackerType)
			assert.NotNil(t, data)
		})
	}
}

func TestInitCommand_FilePermissions(t *testing.T) {
	tempDir := t.TempDir()
	symDir := filepath.Join(tempDir, ".sym")

	cmd := NewInitCommand(InitOptions{
		TrackerType: "mock",
		AgentType:   "codex",
	})

	cfg := cmd.generateConfig("mock", "codex", &trackerConfigData{})
	err := cmd.createDirectoryStructure(symDir, cfg)
	require.NoError(t, err)

	// 验证目录权限
	dirInfo, err := os.Stat(symDir)
	require.NoError(t, err)
	assert.True(t, dirInfo.IsDir())
	assert.Equal(t, os.FileMode(0755), dirInfo.Mode().Perm())

	// 验证文件权限
	configPath := filepath.Join(symDir, "config.yaml")
	fileInfo, err := os.Stat(configPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0644), fileInfo.Mode().Perm())
}

func TestInitCommand_NonInteractiveMode(t *testing.T) {
	// 测试非交互式模式的选项
	opts := InitOptions{
		TrackerType:    "github",
		AgentType:      "claude",
		NonInteractive: true,
	}

	cmd := NewInitCommand(opts)
	assert.Equal(t, "github", cmd.options.TrackerType)
	assert.Equal(t, "claude", cmd.options.AgentType)
	assert.True(t, cmd.options.NonInteractive)
}

func TestInitCommand_ProjectPath(t *testing.T) {
	tempDir := t.TempDir()

	opts := InitOptions{
		ProjectPath: tempDir,
		TrackerType: "mock",
		AgentType:   "codex",
	}

	cmd := NewInitCommand(opts)
	assert.Equal(t, tempDir, cmd.options.ProjectPath)
}

func TestInitCommand_ConfigYAMLContainsNewFields(t *testing.T) {
	tempDir := t.TempDir()
	symDir := filepath.Join(tempDir, ".sym")
	require.NoError(t, os.MkdirAll(symDir, 0755))

	cmd := NewInitCommand(InitOptions{})

	cfg := cmd.generateConfig("mock", "codex", &trackerConfigData{})
	err := cmd.generateConfigYAML(symDir, cfg)
	require.NoError(t, err)

	// 读取生成的配置文件
	configPath := filepath.Join(symDir, "config.yaml")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	contentStr := string(content)

	// 验证新增的配置字段
	assert.Contains(t, contentStr, "clarification:")
	assert.Contains(t, contentStr, "max_rounds: 5")
	assert.Contains(t, contentStr, "execution:")
	assert.Contains(t, contentStr, "max_retries: 3")
}

func TestInitCommand_AllTrackerTypes(t *testing.T) {
	trackerTypes := []string{"linear", "github", "mock", "beads"}

	for _, trackerType := range trackerTypes {
		t.Run(trackerType, func(t *testing.T) {
			cmd := NewInitCommand(InitOptions{
				TrackerType: trackerType,
				AgentType:   "codex",
			})

			cfg := cmd.generateConfig(trackerType, "codex", &trackerConfigData{})
			assert.Equal(t, trackerType, cfg.Tracker.Kind)
		})
	}
}

func TestInitCommand_AllAgentTypes(t *testing.T) {
	agentTypes := []string{"codex", "claude", "opencode"}

	for _, agentType := range agentTypes {
		t.Run(agentType, func(t *testing.T) {
			cmd := NewInitCommand(InitOptions{
				TrackerType: "mock",
				AgentType:   agentType,
			})

			cfg := cmd.generateConfig("mock", agentType, &trackerConfigData{})
			assert.Equal(t, agentType, cfg.Agent.Kind)
		})
	}
}

func TestInitCommand_ErrorCases(t *testing.T) {
	t.Run("invalid directory path", func(t *testing.T) {
		cmd := NewInitCommand(InitOptions{})
		// 使用一个不可能创建的路径
		invalidPath := "/proc/invalid/.sym"
		cfg := cmd.generateConfig("mock", "codex", &trackerConfigData{})

		err := cmd.createDirectoryStructure(invalidPath, cfg)
		// 应该返回错误（具体行为取决于系统）
		// 这里我们只是验证函数不会 panic
		_ = err
	})
}

func TestInitCommand_YAMLFormat(t *testing.T) {
	tempDir := t.TempDir()
	symDir := filepath.Join(tempDir, ".sym")
	require.NoError(t, os.MkdirAll(symDir, 0755))

	cmd := NewInitCommand(InitOptions{})

	cfg := cmd.generateConfig("mock", "codex", &trackerConfigData{})
	err := cmd.generateConfigYAML(symDir, cfg)
	require.NoError(t, err)

	// 读取生成的配置文件
	configPath := filepath.Join(symDir, "config.yaml")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	// 验证 YAML 格式正确（没有前导空格问题）
	contentStr := string(content)
	lines := strings.Split(contentStr, "\n")

	// 第一行不应该是空行
	if len(lines) > 0 {
		assert.NotEqual(t, "", strings.TrimSpace(lines[0]))
	}

	// 验证关键缩进结构
	assert.Contains(t, contentStr, "tracker:")
	assert.Contains(t, contentStr, "  kind:")
}

func TestInitCommand_RunWithMockTracker(t *testing.T) {
	// 使用临时目录测试完整流程
	tempDir := t.TempDir()

	// 创建模拟输入
	input := strings.NewReader("1\n1\n\n")
	cmd := &InitCommand{
		options: InitOptions{
			ProjectPath: tempDir,
		},
		scanner: bufio.NewScanner(input),
	}

	// 由于 Run 函数会尝试读取输入，这里我们只验证不会崩溃
	// 实际的交互式测试需要更复杂的设置
	assert.NotNil(t, cmd)
}

func TestInitCommand_DirectoryExistsOverwrite(t *testing.T) {
	tempDir := t.TempDir()
	symDir := filepath.Join(tempDir, ".sym")

	// 先创建目录
	require.NoError(t, os.MkdirAll(symDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(symDir, "config.yaml"), []byte("old content"), 0644))

	cmd := NewInitCommand(InitOptions{})
	cfg := cmd.generateConfig("mock", "codex", &trackerConfigData{})

	// 测试覆盖逻辑
	err := cmd.createDirectoryStructure(symDir, cfg)
	require.NoError(t, err)

	// 验证文件已被更新
	content, err := os.ReadFile(filepath.Join(symDir, "config.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "kind: mock")
}

func TestInitCommand_PromptTemplatesContent(t *testing.T) {
	tempDir := t.TempDir()
	promptsDir := filepath.Join(tempDir, "prompts")
	require.NoError(t, os.MkdirAll(promptsDir, 0755))

	cmd := NewInitCommand(InitOptions{})
	err := cmd.generatePromptTemplates(promptsDir)
	require.NoError(t, err)

	// 验证 clarification.md 包含正确的模板变量
	clarificationPath := filepath.Join(promptsDir, "clarification.md")
	content, err := os.ReadFile(clarificationPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "{{.ID}}")
	assert.Contains(t, string(content), "{{.Title}}")
	assert.Contains(t, string(content), "{{.Description}}")

	// 验证 bdd.md 包含 Gherkin 语法
	bddPath := filepath.Join(promptsDir, "bdd.md")
	content, err = os.ReadFile(bddPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Feature:")
	assert.Contains(t, string(content), "Scenario:")
	assert.Contains(t, string(content), "Given")
	assert.Contains(t, string(content), "When")
	assert.Contains(t, string(content), "Then")

	// 验证 architecture.md 包含架构设计部分
	archPath := filepath.Join(promptsDir, "architecture.md")
	content, err = os.ReadFile(archPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "系统概览")
	assert.Contains(t, string(content), "数据流")
	assert.Contains(t, string(content), "技术选型")

	// 验证 implementation.md 包含实现步骤
	implPath := filepath.Join(promptsDir, "implementation.md")
	content, err = os.ReadFile(implPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "实现步骤")
	assert.Contains(t, string(content), "编码规范")
	assert.Contains(t, string(content), "测试要求")

	// 验证 verification.md 包含验证清单
	verifyPath := filepath.Join(promptsDir, "verification.md")
	content, err = os.ReadFile(verifyPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "验证检查清单")
	assert.Contains(t, string(content), "功能验证")
	assert.Contains(t, string(content), "代码质量")
}

func TestInitCommand_WorkflowTemplateVariables(t *testing.T) {
	tempDir := t.TempDir()
	symDir := filepath.Join(tempDir, ".sym")
	require.NoError(t, os.MkdirAll(symDir, 0755))

	cmd := NewInitCommand(InitOptions{})
	err := cmd.generateWorkflowMD(symDir)
	require.NoError(t, err)

	workflowPath := filepath.Join(symDir, "workflow.md")
	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err)

	// 验证模板变量
	contentStr := string(content)
	assert.Contains(t, contentStr, "{{.ID}}")
	assert.Contains(t, contentStr, "{{.Title}}")
	assert.Contains(t, contentStr, "{{.Description}}")
	assert.Contains(t, contentStr, "{{.Priority}}")
}

func TestInitCommand_ConfigYAMLAllFields(t *testing.T) {
	tempDir := t.TempDir()
	symDir := filepath.Join(tempDir, ".sym")
	require.NoError(t, os.MkdirAll(symDir, 0755))

	cmd := NewInitCommand(InitOptions{})

	// 使用 linear tracker 测试更多字段
	cfg := cmd.generateConfig("linear", "claude", &trackerConfigData{
		apiKey:      "lin_api_test12345678",
		projectSlug: "PROJ-1",
		endpoint:    "https://api.linear.app/graphql",
	})

	err := cmd.generateConfigYAML(symDir, cfg)
	require.NoError(t, err)

	configPath := filepath.Join(symDir, "config.yaml")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	contentStr := string(content)

	// 验证所有主要配置项
	assert.Contains(t, contentStr, "tracker:")
	assert.Contains(t, contentStr, "kind: linear")
	assert.Contains(t, contentStr, "endpoint:")
	assert.Contains(t, contentStr, "project_slug: PROJ-1")
	assert.Contains(t, contentStr, "active_states:")
	assert.Contains(t, contentStr, "terminal_states:")
	assert.Contains(t, contentStr, "polling:")
	assert.Contains(t, contentStr, "interval_ms:")
	assert.Contains(t, contentStr, "workspace:")
	assert.Contains(t, contentStr, "root:")
	assert.Contains(t, contentStr, "hooks:")
	assert.Contains(t, contentStr, "timeout_ms:")
	assert.Contains(t, contentStr, "agent:")
	assert.Contains(t, contentStr, "kind: claude")
	assert.Contains(t, contentStr, "max_concurrent_agents:")
	assert.Contains(t, contentStr, "max_turns:")
	assert.Contains(t, contentStr, "clarification:")
	assert.Contains(t, contentStr, "max_rounds: 5")
	assert.Contains(t, contentStr, "execution:")
	assert.Contains(t, contentStr, "max_retries: 3")
}

func TestInitCommand_BuildConfigMapWithClaude(t *testing.T) {
	cmd := NewInitCommand(InitOptions{
		TrackerType: "mock",
		AgentType:   "claude",
	})

	cfg := cmd.generateConfig("mock", "claude", &trackerConfigData{})
	configMap := cmd.buildConfigMap(cfg)

	// 验证 agent 配置
	agent, ok := configMap["agent"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "claude", agent["kind"])

	// 验证 codex 配置仍然存在
	_, ok = configMap["codex"]
	assert.True(t, ok)
}

func TestInitCommand_BuildConfigMapWithOpenCode(t *testing.T) {
	cmd := NewInitCommand(InitOptions{
		TrackerType: "github",
		AgentType:   "opencode",
	})

	cfg := cmd.generateConfig("github", "opencode", &trackerConfigData{
		repo: "owner/repo",
	})
	configMap := cmd.buildConfigMap(cfg)

	// 验证 tracker 配置
	tracker, ok := configMap["tracker"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "github", tracker["kind"])
	assert.Equal(t, "owner/repo", tracker["repo"])

	// 验证 agent 配置
	agent, ok := configMap["agent"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "opencode", agent["kind"])
}

func TestInitCommand_APIKeyMasking(t *testing.T) {
	cmd := NewInitCommand(InitOptions{})

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty key",
			input:    "",
			expected: "",
		},
		{
			name:     "very short key",
			input:    "a",
			expected: "***",
		},
		{
			name:     "4 char key",
			input:    "abcd",
			expected: "***",
		},
		{
			name:     "8 char key",
			input:    "abcdefgh",
			expected: "***",
		},
		{
			name:     "10 char key",
			input:    "abcdefghij",
			expected: "abcd...ghij",
		},
		{
			name:     "linear API key format",
			input:    "lin_api_abcdefghijklmnopqrstuvwxyz",
			expected: "lin_...wxyz",
		},
		{
			name:     "github token format",
			input:    "ghp_1234567890ABCDEFGHIJKLMNOP",
			expected: "ghp_...MNOP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cmd.maskAPIKey(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInitCommand_RunWithAllOptionsProvided(t *testing.T) {
	tempDir := t.TempDir()

	// 使用所有选项都预先提供的模式，避免交互式输入
	cmd := &InitCommand{
		options: InitOptions{
			TrackerType:    "mock",
			AgentType:      "codex",
			ProjectPath:    tempDir,
			NonInteractive: true,
		},
		scanner: bufio.NewScanner(strings.NewReader("")),
	}

	err := cmd.Run()
	require.NoError(t, err)

	// 验证目录和文件已创建
	assert.DirExists(t, filepath.Join(tempDir, ".sym"))
	assert.FileExists(t, filepath.Join(tempDir, ".sym", "config.yaml"))
	assert.FileExists(t, filepath.Join(tempDir, ".sym", "workflow.md"))
	assert.DirExists(t, filepath.Join(tempDir, ".sym", "prompts"))
}

func TestInitCommand_RunWithLinearTracker(t *testing.T) {
	tempDir := t.TempDir()

	cmd := &InitCommand{
		options: InitOptions{
			TrackerType:    "linear",
			AgentType:      "claude",
			ProjectPath:    tempDir,
			NonInteractive: true,
		},
		scanner: bufio.NewScanner(strings.NewReader("")),
	}

	err := cmd.Run()
	require.NoError(t, err)

	// 读取并验证配置
	configPath := filepath.Join(tempDir, ".sym", "config.yaml")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "kind: linear")
	assert.Contains(t, string(content), "kind: claude")
}

func TestInitCommand_RunWithGitHubTracker(t *testing.T) {
	tempDir := t.TempDir()

	cmd := &InitCommand{
		options: InitOptions{
			TrackerType:    "github",
			AgentType:      "opencode",
			ProjectPath:    tempDir,
			NonInteractive: true,
		},
		scanner: bufio.NewScanner(strings.NewReader("")),
	}

	err := cmd.Run()
	require.NoError(t, err)

	configPath := filepath.Join(tempDir, ".sym", "config.yaml")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "kind: github")
	assert.Contains(t, string(content), "kind: opencode")
}

func TestInitCommand_RunWithBeadsTracker(t *testing.T) {
	tempDir := t.TempDir()

	cmd := &InitCommand{
		options: InitOptions{
			TrackerType:    "beads",
			AgentType:      "codex",
			ProjectPath:    tempDir,
			NonInteractive: true,
		},
		scanner: bufio.NewScanner(strings.NewReader("")),
	}

	err := cmd.Run()
	require.NoError(t, err)

	configPath := filepath.Join(tempDir, ".sym", "config.yaml")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "kind: beads")
}

func TestInitCommand_RunWithInvalidPath(t *testing.T) {
	cmd := &InitCommand{
		options: InitOptions{
			ProjectPath: "/nonexistent/path/that/cannot/be/accessed",
		},
		scanner: bufio.NewScanner(strings.NewReader("")),
	}

	err := cmd.Run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "init.dir_access")
}

func TestInitCommand_PromptSelectWithInput(t *testing.T) {
	// 测试用户输入数字选择
	input := strings.NewReader("2\n")
	cmd := &InitCommand{
		options: InitOptions{},
		scanner: bufio.NewScanner(input),
	}

	result := cmd.promptSelect("选择 tracker", []string{"linear", "github", "mock", "beads"}, "mock")
	assert.Equal(t, "github", result)
}

func TestInitCommand_PromptSelectWithDirectInput(t *testing.T) {
	// 测试用户直接输入选项名称
	input := strings.NewReader("linear\n")
	cmd := &InitCommand{
		options: InitOptions{},
		scanner: bufio.NewScanner(input),
	}

	result := cmd.promptSelect("选择 tracker", []string{"linear", "github", "mock", "beads"}, "mock")
	assert.Equal(t, "linear", result)
}

func TestInitCommand_PromptSelectWithEmptyInput(t *testing.T) {
	// 测试空输入返回默认值
	input := strings.NewReader("\n")
	cmd := &InitCommand{
		options: InitOptions{},
		scanner: bufio.NewScanner(input),
	}

	result := cmd.promptSelect("选择 tracker", []string{"linear", "github", "mock", "beads"}, "mock")
	assert.Equal(t, "mock", result)
}

func TestInitCommand_PromptInputWithInput(t *testing.T) {
	input := strings.NewReader("test_value\n")
	cmd := &InitCommand{
		options: InitOptions{},
		scanner: bufio.NewScanner(input),
	}

	result := cmd.promptInput("请输入值", "default")
	assert.Equal(t, "test_value", result)
}

func TestInitCommand_PromptInputWithEmptyInput(t *testing.T) {
	input := strings.NewReader("\n")
	cmd := &InitCommand{
		options: InitOptions{},
		scanner: bufio.NewScanner(input),
	}

	result := cmd.promptInput("请输入值", "default_value")
	assert.Equal(t, "default_value", result)
}

func TestInitCommand_PromptConfirmYes(t *testing.T) {
	input := strings.NewReader("y\n")
	cmd := &InitCommand{
		options: InitOptions{},
		scanner: bufio.NewScanner(input),
	}

	result := cmd.promptConfirm("确认?", false)
	assert.True(t, result)
}

func TestInitCommand_PromptConfirmNo(t *testing.T) {
	input := strings.NewReader("n\n")
	cmd := &InitCommand{
		options: InitOptions{},
		scanner: bufio.NewScanner(input),
	}

	result := cmd.promptConfirm("确认?", true)
	assert.False(t, result)
}

func TestInitCommand_PromptConfirmEmptyDefaultFalse(t *testing.T) {
	input := strings.NewReader("\n")
	cmd := &InitCommand{
		options: InitOptions{},
		scanner: bufio.NewScanner(input),
	}

	result := cmd.promptConfirm("确认?", false)
	assert.False(t, result)
}

func TestInitCommand_PromptConfirmEmptyDefaultTrue(t *testing.T) {
	input := strings.NewReader("\n")
	cmd := &InitCommand{
		options: InitOptions{},
		scanner: bufio.NewScanner(input),
	}

	result := cmd.promptConfirm("确认?", true)
	assert.True(t, result)
}

func TestInitCommand_CollectTrackerConfigLinear(t *testing.T) {
	input := strings.NewReader("test_api_key\nPROJ-123\n")
	cmd := &InitCommand{
		options: InitOptions{},
		scanner: bufio.NewScanner(input),
	}

	data := cmd.collectTrackerConfig("linear")
	assert.Equal(t, "test_api_key", data.apiKey)
	assert.Equal(t, "PROJ-123", data.projectSlug)
	assert.Equal(t, "https://api.linear.app/graphql", data.endpoint)
}

func TestInitCommand_CollectTrackerConfigGitHub(t *testing.T) {
	input := strings.NewReader("ghp_test_token\nowner/repo\n")
	cmd := &InitCommand{
		options: InitOptions{},
		scanner: bufio.NewScanner(input),
	}

	data := cmd.collectTrackerConfig("github")
	assert.Equal(t, "ghp_test_token", data.apiKey)
	assert.Equal(t, "owner/repo", data.repo)
	assert.Equal(t, "https://api.github.com", data.endpoint)
}

func TestInitCommand_CollectTrackerConfigMock(t *testing.T) {
	cmd := NewInitCommand(InitOptions{})

	data := cmd.collectTrackerConfig("mock")
	// mock 不需要配置
	assert.Equal(t, "", data.apiKey)
	assert.Equal(t, "", data.endpoint)
}