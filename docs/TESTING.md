# 测试规范

本文档定义项目的测试风格和覆盖率要求。

## 测试文件组织

### 1. 文件命名

`*_test.go`，与被测文件同目录

### 2. 包命名

```go
// 外部测试（推荐，测试公开 API）
package config_test

// 内部测试（需要访问私有成员时使用）
package config
```

### 3. 测试函数命名

`Test<FunctionName>` 或 `Test<Scenario>`

## 测试风格和模式

### 1. 表驱动测试（必须使用）

所有涉及多场景的测试必须使用表驱动测试：

```go
func TestValidateDispatchConfig(t *testing.T) {
    tests := []struct {
        name       string
        config     *config.Config
        wantValid  bool
    }{
        {
            name: "valid config",
            config: &config.Config{
                Tracker: config.TrackerConfig{
                    Kind:        "linear",
                    APIKey:      "test-key",
                    ProjectSlug: "TEST",
                },
                Codex: config.CodexConfig{
                    Command: "codex app-server",
                },
            },
            wantValid: true,
        },
        {
            name: "missing tracker kind",
            config: &config.Config{
                Tracker: config.TrackerConfig{
                    APIKey:      "test-key",
                    ProjectSlug: "TEST",
                },
            },
            wantValid: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            validation := tt.config.ValidateDispatchConfig()
            if validation.Valid != tt.wantValid {
                t.Errorf("expected valid=%v, got valid=%v, errors=%v",
                    tt.wantValid, validation.Valid, validation.Errors)
            }
        })
    }
}
```

### 2. 子测试组织

使用 `t.Run()` 组织相关测试用例：

```go
func TestMockClient(t *testing.T) {
    client := NewMockClient(mockIssues)

    t.Run("FetchCandidateIssues", func(t *testing.T) {
        // ...
    })

    t.Run("FetchIssuesByStates", func(t *testing.T) {
        // ...
    })
}
```

### 3. 临时资源清理

使用 `defer` 确保测试资源被清理：

```go
func TestCreateForIssue(t *testing.T) {
    tmpDir, err := os.MkdirTemp("", "symphony-test-*")
    if err != nil {
        t.Fatalf("failed to create temp dir: %v", err)
    }
    defer os.RemoveAll(tmpDir)  // 必须清理

    // 测试代码...
}
```

### 4. 环境变量测试

设置测试环境变量后必须清理：

```go
func TestParseConfig(t *testing.T) {
    os.Setenv("TEST_API_KEY", "test-key-123")
    defer os.Unsetenv("TEST_API_KEY")  // 必须清理

    // 测试代码...
}
```

## 测试覆盖要求

### 必须测试的场景

| 模块 | 必须覆盖的场景 |
|------|----------------|
| `config` | 默认值、环境变量解析、验证成功/失败 |
| `workflow` | 空内容、无前置内容、有效前置内容、无效 YAML、非映射前置内容 |
| `workspace` | 创建工作空间、复用工作空间、路径清理、删除工作空间、路径验证 |
| `tracker/mock` | 获取候选问题、按状态获取、按 ID 获取、状态更新 |
| `domain` | 所有实体字段、状态常量、边界值 |
| `orchestrator` | 调度逻辑、重试机制、协调逻辑、并发控制 |

### 覆盖率要求

- 新增代码必须有对应测试
- Bug 修复必须添加回归测试
- 目标覆盖率: 核心逻辑 > 80%

## Mock 使用

### Mock Tracker

使用 `internal/tracker/mock.go` 进行测试：

```go
mockIssues := []config.MockIssueConfig{
    {ID: "1", Identifier: "TEST-1", Title: "测试任务", State: "Todo"},
    {ID: "2", Identifier: "TEST-2", Title: "进行中任务", State: "In Progress"},
}
client := NewMockClient(mockIssues)

// 获取候选问题
issues, _ := client.FetchCandidateIssues(ctx, []string{"Todo"})

// 更新状态（用于测试状态流转）
client.UpdateIssueState("1", "In Progress")

// 获取状态历史
history := client.GetStateHistory("1")  // ["Todo"]
```

### 测试配置

使用 Mock Tracker 配置进行集成测试：

```yaml
# test/test-mock-workflow.md
tracker:
  kind: mock
  active_states: [Todo, In Progress]
  terminal_states: [Done, Cancelled]
  mock_issues:
    - id: "1"
      identifier: "TEST-1"
      title: "测试任务"
      state: Todo
```

## 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/config/...

# 运行特定测试
go test -run TestValidateDispatchConfig ./internal/config/

# 查看覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 测试检查清单

新增代码时，请确认：

- [ ] 测试文件与被测文件同目录
- [ ] 使用表驱动测试覆盖多场景
- [ ] 使用 `t.Run()` 组织子测试
- [ ] 临时资源（文件、目录、环境变量）使用 `defer` 清理
- [ ] 测试函数名清晰描述测试场景
- [ ] 错误消息包含期望值和实际值
- [ ] 覆盖了正常路径和错误路径
- [ ] 边界值和空值情况已测试

## 集成测试

### 使用 Mock Workflow

项目提供了 `test/test-mock-workflow.md` 用于测试：

```bash
# 使用 mock 配置运行服务
./bin/symphony -workflow test/test-mock-workflow.md -port 8080
```

### 验证点

集成测试应验证：

1. **Mock Tracker 正常工作** - 问题能正确获取
2. **状态流转逻辑** - 状态变更被正确处理
3. **编排调度功能** - 问题按优先级调度
4. **工作空间创建/清理** - 工作空间正确管理