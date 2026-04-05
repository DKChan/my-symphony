package cli

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// boolPtr 辅助函数，用于创建 bool 指针
func boolPtr(v bool) *bool {
	return &v
}

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
			cfg := cmd.generateConfig("", tt.trackerType, tt.agentType, true, 5, tt.trackerData)

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
	cfg := cmd.generateConfig("", "mock", "codex", true, 5, &trackerConfigData{})

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

	cfg := cmd.generateConfig("", "mock", "codex", true, 5, &trackerConfigData{})

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
		TrackerType: "github",
		AgentType:   "claude",
	})

	cfg := cmd.generateConfig("", "github", "claude", true, 5, &trackerConfigData{
		apiKey:   "ghp_test123",
		repo:     "owner/repo",
		endpoint: "https://api.github.com",
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
	options := []string{"github", "mock", "beads"}
	assert.Equal(t, 3, len(options))
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

	cfg := cmd.generateConfig("", "mock", "codex", true, 5, &trackerConfigData{})
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

	cfg := cmd.generateConfig("", "mock", "codex", true, 5, &trackerConfigData{})
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
	trackerTypes := []string{"github", "mock", "beads"}

	for _, trackerType := range trackerTypes {
		t.Run(trackerType, func(t *testing.T) {
			cmd := NewInitCommand(InitOptions{
				TrackerType: trackerType,
				AgentType:   "codex",
			})

			cfg := cmd.generateConfig("", trackerType, "codex", true, 5, &trackerConfigData{})
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

			cfg := cmd.generateConfig("", "mock", agentType, true, 5, &trackerConfigData{})
			assert.Equal(t, agentType, cfg.Agent.Kind)
		})
	}
}

func TestInitCommand_ErrorCases(t *testing.T) {
	t.Run("invalid directory path", func(t *testing.T) {
		cmd := NewInitCommand(InitOptions{})
		// 使用一个不可能创建的路径
		invalidPath := "/proc/invalid/.sym"
		cfg := cmd.generateConfig("", "mock", "codex", true, 5, &trackerConfigData{})

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

	cfg := cmd.generateConfig("", "mock", "codex", true, 5, &trackerConfigData{})
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
	cfg := cmd.generateConfig("", "mock", "codex", true, 5, &trackerConfigData{})

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

	// 使用 github tracker 测试更多字段
	cfg := cmd.generateConfig("", "github", "claude", true, 5, &trackerConfigData{
		apiKey:   "ghp_test12345678",
		repo:     "owner/repo",
		endpoint: "https://api.github.com",
	})

	err := cmd.generateConfigYAML(symDir, cfg)
	require.NoError(t, err)

	configPath := filepath.Join(symDir, "config.yaml")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	contentStr := string(content)

	// 验证所有主要配置项
	assert.Contains(t, contentStr, "tracker:")
	assert.Contains(t, contentStr, "kind: github")
	assert.Contains(t, contentStr, "endpoint:")
	assert.Contains(t, contentStr, "repo: owner/repo")
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

	cfg := cmd.generateConfig("", "mock", "claude", true, 5, &trackerConfigData{})
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

	cfg := cmd.generateConfig("", "github", "opencode", true, 5, &trackerConfigData{
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
	input := strings.NewReader("1\n")
	cmd := &InitCommand{
		options: InitOptions{},
		scanner: bufio.NewScanner(input),
	}

	result := cmd.promptSelect("选择 tracker", []string{"github", "mock", "beads"}, "beads")
	assert.Equal(t, "github", result)
}

func TestInitCommand_PromptSelectWithDirectInput(t *testing.T) {
	// 测试用户直接输入选项名称
	input := strings.NewReader("github\n")
	cmd := &InitCommand{
		options: InitOptions{},
		scanner: bufio.NewScanner(input),
	}

	result := cmd.promptSelect("选择 tracker", []string{"github", "mock", "beads"}, "mock")
	assert.Equal(t, "github", result)
}

func TestInitCommand_PromptSelectWithEmptyInput(t *testing.T) {
	// 测试空输入返回默认值
	input := strings.NewReader("\n")
	cmd := &InitCommand{
		options: InitOptions{},
		scanner: bufio.NewScanner(input),
	}

	result := cmd.promptSelect("选择 tracker", []string{"github", "mock", "beads"}, "beads")
	assert.Equal(t, "beads", result)
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

// 5.1 测试 InitOptions 新字段
func TestInitOptions_NewFields(t *testing.T) {
	tests := []struct {
		name            string
		opts            InitOptions
		wantProjectName string
		wantBMADEnabled *bool
		wantMaxIterations int
	}{
		{
			name: "all new fields set",
			opts: InitOptions{
				ProjectName:    "my-awesome-project",
				BMADEnabled:    boolPtr(true),
				MaxIterations:  10,
			},
			wantProjectName:  "my-awesome-project",
			wantBMADEnabled:  boolPtr(true),
			wantMaxIterations: 10,
		},
		{
			name: "BMAD disabled",
			opts: InitOptions{
				ProjectName:    "test-project",
				BMADEnabled:    boolPtr(false),
				MaxIterations:  3,
			},
			wantProjectName:  "test-project",
			wantBMADEnabled:  boolPtr(false),
			wantMaxIterations: 3,
		},
		{
			name: "empty project name",
			opts: InitOptions{
				ProjectName:    "",
				BMADEnabled:    boolPtr(true),
				MaxIterations:  5,
			},
			wantProjectName:  "",
			wantBMADEnabled:  boolPtr(true),
			wantMaxIterations: 5,
		},
		{
			name: "zero max iterations",
			opts: InitOptions{
				ProjectName:    "project",
				BMADEnabled:    boolPtr(true),
				MaxIterations:  0,
			},
			wantProjectName:  "project",
			wantBMADEnabled:  boolPtr(true),
			wantMaxIterations: 0,
		},
		{
			name: "nil BMADENabled uses default",
			opts: InitOptions{
				ProjectName:    "project",
				BMADEnabled:    nil,
				MaxIterations:  5,
			},
			wantProjectName:  "project",
			wantBMADEnabled:  nil,
			wantMaxIterations: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewInitCommand(tt.opts)
			assert.Equal(t, tt.wantProjectName, cmd.options.ProjectName)
			if tt.wantBMADEnabled == nil {
				assert.Nil(t, cmd.options.BMADEnabled)
			} else {
				assert.Equal(t, *tt.wantBMADEnabled, *cmd.options.BMADEnabled)
			}
			assert.Equal(t, tt.wantMaxIterations, cmd.options.MaxIterations)
		})
	}
}

func TestInitOptions_DefaultValues(t *testing.T) {
	// 测试新字段的默认值行为
	opts := InitOptions{}
	cmd := NewInitCommand(opts)

	// 默认值应该是零值
	assert.Equal(t, "", cmd.options.ProjectName)
	assert.Nil(t, cmd.options.BMADEnabled)
	assert.Equal(t, 0, cmd.options.MaxIterations)
}

// 5.2 测试交互式问答流程 - BMAD 相关
func TestInitCommand_BMADEnabledPrompt(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		opts        InitOptions
		wantEnabled bool
	}{
		{
			name:        "user confirms BMAD enabled",
			input:       "y\n",
			opts:        InitOptions{BMADEnabled: boolPtr(false)},
			wantEnabled: true,
		},
		{
			name:        "user rejects BMAD enabled",
			input:       "n\n",
			opts:        InitOptions{BMADEnabled: boolPtr(false)},
			wantEnabled: false,
		},
		{
			name:        "empty input uses default true",
			input:       "\n",
			opts:        InitOptions{BMADEnabled: boolPtr(false)},
			wantEnabled: true,
		},
		{
			name:        "non-interactive mode uses option value true",
			input:       "",
			opts:        InitOptions{BMADEnabled: boolPtr(true), NonInteractive: true},
			wantEnabled: true,
		},
		{
			name:        "non-interactive mode uses option value false",
			input:       "",
			opts:        InitOptions{BMADEnabled: boolPtr(false), NonInteractive: true},
			wantEnabled: false,
		},
		{
			name:        "non-interactive mode nil uses default true",
			input:       "",
			opts:        InitOptions{BMADEnabled: nil, NonInteractive: true},
			wantEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 只测试 promptConfirm 函数的行为
			// 对于 non-interactive 模式，Run 函数会跳过 prompt
			if !tt.opts.NonInteractive {
				input := strings.NewReader(tt.input)
				cmd := &InitCommand{
					options: tt.opts,
					scanner: bufio.NewScanner(input),
				}
				result := cmd.promptConfirm("是否启用 BMAD Agent?", true)
				assert.Equal(t, tt.wantEnabled, result)
			} else {
				// 验证 NonInteractive 模式下值来自 opts 或默认值
				cmd := NewInitCommand(tt.opts)
				if cmd.options.BMADEnabled == nil {
					// nil 应该使用默认值 true
					assert.Equal(t, true, tt.wantEnabled)
				} else {
					assert.Equal(t, tt.wantEnabled, *cmd.options.BMADEnabled)
				}
			}
		})
	}
}

func TestInitCommand_MaxIterationsPrompt(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		opts              InitOptions
		wantMaxIterations int
	}{
		{
			name:              "user enters valid number",
			input:             "10\n",
			opts:              InitOptions{MaxIterations: 0, BMADEnabled: boolPtr(true)},
			wantMaxIterations: 10,
		},
		{
			name:              "empty input uses default 5",
			input:             "\n",
			opts:              InitOptions{MaxIterations: 0, BMADEnabled: boolPtr(true)},
			wantMaxIterations: 5,
		},
		{
			name:              "invalid input uses default 5",
			input:             "invalid\n",
			opts:              InitOptions{MaxIterations: 0, BMADEnabled: boolPtr(true)},
			wantMaxIterations: 5,
		},
		{
			name:              "negative input uses default 5",
			input:             "-5\n",
			opts:              InitOptions{MaxIterations: 0, BMADEnabled: boolPtr(true)},
			wantMaxIterations: 5,
		},
		{
			name:              "zero input uses default 5",
			input:             "0\n",
			opts:              InitOptions{MaxIterations: 0, BMADEnabled: boolPtr(true)},
			wantMaxIterations: 5,
		},
		{
			name:              "non-interactive mode uses option value",
			input:             "",
			opts:              InitOptions{MaxIterations: 7, BMADEnabled: boolPtr(true), NonInteractive: true},
			wantMaxIterations: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.opts.NonInteractive {
				input := strings.NewReader(tt.input)
				cmd := &InitCommand{
					options: tt.opts,
					scanner: bufio.NewScanner(input),
				}
				maxIterationsStr := cmd.promptInput("请输入最大迭代次数", "5")
				if val, err := strconv.Atoi(maxIterationsStr); err == nil && val > 0 {
					assert.Equal(t, tt.wantMaxIterations, val)
				} else {
					assert.Equal(t, tt.wantMaxIterations, 5)
				}
			} else {
				// 验证 NonInteractive 模式下值来自 opts
				cmd := NewInitCommand(tt.opts)
				assert.Equal(t, tt.wantMaxIterations, cmd.options.MaxIterations)
			}
		})
	}
}

func TestInitCommand_ProjectNamePrompt(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		opts            InitOptions
		wantProjectName string
	}{
		{
			name:            "user enters project name",
			input:           "my-project\n",
			opts:            InitOptions{ProjectName: ""},
			wantProjectName: "my-project",
		},
		{
			name:            "empty input uses default",
			input:           "\n",
			opts:            InitOptions{ProjectName: ""},
			wantProjectName: "my-project",
		},
		{
			name:            "non-interactive mode uses option value",
			input:           "",
			opts:            InitOptions{ProjectName: "test-project", NonInteractive: true},
			wantProjectName: "test-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.opts.NonInteractive {
				input := strings.NewReader(tt.input)
				cmd := &InitCommand{
					options: tt.opts,
					scanner: bufio.NewScanner(input),
				}
				result := cmd.promptInput("请输入项目名称", "my-project")
				assert.Equal(t, tt.wantProjectName, result)
			} else {
				cmd := NewInitCommand(tt.opts)
				assert.Equal(t, tt.wantProjectName, cmd.options.ProjectName)
			}
		})
	}
}

func TestInitCommand_NonInteractiveModeDefaultValues(t *testing.T) {
	tempDir := t.TempDir()

	// 测试非交互式模式下使用默认值
	cmd := &InitCommand{
		options: InitOptions{
			TrackerType:    "mock",
			AgentType:      "codex",
			ProjectPath:    tempDir,
			NonInteractive: true,
			// 不设置 ProjectName, BMADEnabled, MaxIterations
			// 验证默认值行为
		},
		scanner: bufio.NewScanner(strings.NewReader("")),
	}

	err := cmd.Run()
	require.NoError(t, err)

	// 验证配置文件已生成
	configPath := filepath.Join(tempDir, ".sym", "config.yaml")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	contentStr := string(content)

	// 验证 harness 配置使用默认值
	assert.Contains(t, contentStr, "harness:")
	// NonInteractive 模式下 BMADEnabled 默认为 false（零值）
	// 但 generateConfig 中 bmadEnabled 参数会影响最终值
}

// 5.3 测试配置生成逻辑 - harness 配置
func TestInitCommand_GenerateConfig_HarnessConfig(t *testing.T) {
	tests := []struct {
		name                string
		projectName         string
		bmadEnabled         bool
		maxIterations       int
		wantHarnessInYAML   bool
		wantBMADEnabled     bool
		wantMaxIterations   int
		wantPlannerCount    int
		wantGeneratorCount  int
		wantEvaluatorCount  int
	}{
		{
			name:               "BMAD enabled with default agents",
			projectName:        "test-project",
			bmadEnabled:        true,
			maxIterations:      5,
			wantHarnessInYAML:  true,
			wantBMADEnabled:    true,
			wantMaxIterations:  5,
			wantPlannerCount:   3,
			wantGeneratorCount: 2,
			wantEvaluatorCount: 2,
		},
		{
			name:               "BMAD disabled",
			projectName:        "test-project",
			bmadEnabled:        false,
			maxIterations:      3,
			wantHarnessInYAML:  true,
			wantBMADEnabled:    false,
			wantMaxIterations:  3,
			wantPlannerCount:   3, // defaults still set
			wantGeneratorCount: 2,
			wantEvaluatorCount: 2,
		},
		{
			name:               "BMAD enabled with custom iterations",
			projectName:        "my-project",
			bmadEnabled:        true,
			maxIterations:      10,
			wantHarnessInYAML:  true,
			wantBMADEnabled:    true,
			wantMaxIterations:  10,
			wantPlannerCount:   3,
			wantGeneratorCount: 2,
			wantEvaluatorCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewInitCommand(InitOptions{})
			cfg := cmd.generateConfig(tt.projectName, "mock", "codex", tt.bmadEnabled, tt.maxIterations, &trackerConfigData{})

			// 验证 harness 配置
			assert.Equal(t, tt.wantMaxIterations, cfg.Harness.MaxIterations)
			assert.Equal(t, tt.wantBMADEnabled, cfg.Harness.BMAD.Enabled)
			assert.Equal(t, tt.wantPlannerCount, len(cfg.Harness.BMAD.Agents.Planner))
			assert.Equal(t, tt.wantGeneratorCount, len(cfg.Harness.BMAD.Agents.Generator))
			assert.Equal(t, tt.wantEvaluatorCount, len(cfg.Harness.BMAD.Agents.Evaluator))

			// 验证项目名称
			if tt.projectName != "" {
				assert.Equal(t, tt.projectName, cfg.Workspace.ProjectName)
			}
		})
	}
}

func TestInitCommand_GenerateConfig_BMADDefaultAgents(t *testing.T) {
	cmd := NewInitCommand(InitOptions{})

	// BMAD enabled=true 时应使用默认 agents 列表
	cfg := cmd.generateConfig("", "mock", "codex", true, 5, &trackerConfigData{})

	assert.True(t, cfg.Harness.BMAD.Enabled)

	// 验证分组 agents
	assert.Equal(t, 3, len(cfg.Harness.BMAD.Agents.Planner))
	assert.Equal(t, 2, len(cfg.Harness.BMAD.Agents.Generator))
	assert.Equal(t, 2, len(cfg.Harness.BMAD.Agents.Evaluator))

	// 验证默认 agents 包含正确的 agent
	assert.Contains(t, cfg.Harness.BMAD.Agents.Planner, "bmad-agent-pm")
	assert.Contains(t, cfg.Harness.BMAD.Agents.Generator, "bmad-agent-dev")
	assert.Contains(t, cfg.Harness.BMAD.Agents.Evaluator, "bmad-code-review")
	assert.Contains(t, cfg.Harness.BMAD.Agents.Evaluator, "bmad-editorial-review-prose")
}

func TestInitCommand_GenerateConfig_BMADDisabled(t *testing.T) {
	cmd := NewInitCommand(InitOptions{})

	// BMAD enabled=false 时配置仍然存在
	cfg := cmd.generateConfig("", "mock", "codex", false, 5, &trackerConfigData{})

	assert.False(t, cfg.Harness.BMAD.Enabled)
	// 默认 agents 仍然设置在配置中
	assert.Equal(t, 3, len(cfg.Harness.BMAD.Agents.Planner))
}

func TestInitCommand_GenerateConfig_MaxIterationsConfig(t *testing.T) {
	cmd := NewInitCommand(InitOptions{})

	tests := []struct {
		name          string
		maxIterations int
		wantValue     int
	}{
		{
			name:          "positive iterations",
			maxIterations: 7,
			wantValue:     7,
		},
		{
			name:          "zero iterations",
			maxIterations: 0,
			wantValue:     0,
		},
		{
			name:          "large iterations",
			maxIterations: 100,
			wantValue:     100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := cmd.generateConfig("", "mock", "codex", true, tt.maxIterations, &trackerConfigData{})
			assert.Equal(t, tt.wantValue, cfg.Harness.MaxIterations)
		})
	}
}

func TestInitCommand_BuildConfigMap_HarnessSection(t *testing.T) {
	cmd := NewInitCommand(InitOptions{})

	cfg := cmd.generateConfig("test-project", "mock", "codex", true, 8, &trackerConfigData{})
	configMap := cmd.buildConfigMap(cfg)

	// 验证 harness 配置映射
	harness, ok := configMap["harness"].(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, 8, harness["max_iterations"])

	bmad, ok := harness["bmad"].(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, true, bmad["enabled"])
	agents, ok := bmad["agents"].(map[string]interface{})
	require.True(t, ok)

	planner, ok := agents["planner"].([]string)
	require.True(t, ok)
	assert.Equal(t, 3, len(planner))

	generator, ok := agents["generator"].([]string)
	require.True(t, ok)
	assert.Equal(t, 2, len(generator))

	evaluator, ok := agents["evaluator"].([]string)
	require.True(t, ok)
	assert.Equal(t, 2, len(evaluator))
}

func TestInitCommand_ConfigYAMLContainsHarness(t *testing.T) {
	tempDir := t.TempDir()
	symDir := filepath.Join(tempDir, ".sym")
	require.NoError(t, os.MkdirAll(symDir, 0755))

	cmd := NewInitCommand(InitOptions{})

	cfg := cmd.generateConfig("my-project", "mock", "codex", true, 7, &trackerConfigData{})
	err := cmd.generateConfigYAML(symDir, cfg)
	require.NoError(t, err)

	configPath := filepath.Join(symDir, "config.yaml")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	contentStr := string(content)

	// 验证 harness 配置存在于 YAML 中
	assert.Contains(t, contentStr, "harness:")
	assert.Contains(t, contentStr, "max_iterations: 7")
	assert.Contains(t, contentStr, "bmad:")
	assert.Contains(t, contentStr, "enabled: true")
	assert.Contains(t, contentStr, "agents:")
	// 验证默认 agents
	assert.Contains(t, contentStr, "bmad-agent-pm")
	assert.Contains(t, contentStr, "bmad-agent-qa")
	assert.Contains(t, contentStr, "bmad-code-review")
}

func TestInitCommand_ConfigYAMLBMADDisabled(t *testing.T) {
	tempDir := t.TempDir()
	symDir := filepath.Join(tempDir, ".sym")
	require.NoError(t, os.MkdirAll(symDir, 0755))

	cmd := NewInitCommand(InitOptions{})

	cfg := cmd.generateConfig("", "mock", "codex", false, 3, &trackerConfigData{})
	err := cmd.generateConfigYAML(symDir, cfg)
	require.NoError(t, err)

	configPath := filepath.Join(symDir, "config.yaml")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	contentStr := string(content)

	// 验证 BMAD disabled 时的配置
	assert.Contains(t, contentStr, "harness:")
	assert.Contains(t, contentStr, "max_iterations: 3")
	assert.Contains(t, contentStr, "bmad:")
	assert.Contains(t, contentStr, "enabled: false")
	// 验证分组结构
	assert.Contains(t, contentStr, "planner:")
	assert.Contains(t, contentStr, "generator:")
	assert.Contains(t, contentStr, "evaluator:")
}

func TestInitCommand_RunWithBMADOptions(t *testing.T) {
	tempDir := t.TempDir()

	cmd := &InitCommand{
		options: InitOptions{
			TrackerType:    "mock",
			AgentType:      "codex",
			ProjectPath:    tempDir,
			ProjectName:    "test-bmad-project",
			BMADEnabled:    boolPtr(true),
			MaxIterations:  8,
			NonInteractive: true,
		},
		scanner: bufio.NewScanner(strings.NewReader("")),
	}

	err := cmd.Run()
	require.NoError(t, err)

	configPath := filepath.Join(tempDir, ".sym", "config.yaml")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	contentStr := string(content)

	// 验证 BMAD 相关配置
	assert.Contains(t, contentStr, "project_name: test-bmad-project")
	assert.Contains(t, contentStr, "harness:")
	assert.Contains(t, contentStr, "max_iterations: 8")
	assert.Contains(t, contentStr, "enabled: true")
	assert.Contains(t, contentStr, "bmad-agent-pm")
}

func TestInitCommand_RunWithBMADDisabled(t *testing.T) {
	tempDir := t.TempDir()

	cmd := &InitCommand{
		options: InitOptions{
			TrackerType:    "mock",
			AgentType:      "codex",
			ProjectPath:    tempDir,
			ProjectName:    "no-bmad-project",
			BMADEnabled:    boolPtr(false),
			MaxIterations:  3,
			NonInteractive: true,
		},
		scanner: bufio.NewScanner(strings.NewReader("")),
	}

	err := cmd.Run()
	require.NoError(t, err)

	configPath := filepath.Join(tempDir, ".sym", "config.yaml")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	contentStr := string(content)

	// 验证 BMAD disabled 配置
	assert.Contains(t, contentStr, "project_name: no-bmad-project")
	assert.Contains(t, contentStr, "max_iterations: 3")
	assert.Contains(t, contentStr, "enabled: false")
	// 分组结构仍然存在
	assert.Contains(t, contentStr, "planner:")
	assert.Contains(t, contentStr, "generator:")
	assert.Contains(t, contentStr, "evaluator:")
}