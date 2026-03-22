# AGENTS.md

本文档为 AI 代理（如 Claude Code、Cursor 等）提供项目规则和指南，确保代码修改符合项目规范。

## 项目概述

Symphony 是一个编码代理编排服务，基于 [OpenAI Symphony SPEC](https://github.com/openai/symphony/blob/main/SPEC.md) 实现。它持续从问题跟踪器（如 Linear、GitHub）读取工作，为每个问题创建隔离的工作空间，并在工作空间内运行编码代理会话。

### 技术栈

- **语言**: Go 1.22
- **Web 框架**: Gin
- **配置格式**: YAML (WORKFLOW.md 前置内容) + Markdown (提示模板)
- **模板引擎**: Liquid 兼容语法

## 项目结构

```
symphony/
├── cmd/symphony/          # 主程序入口
├── internal/
│   ├── agent/             # 代理运行器接口和实现
│   │   ├── runner.go      # Runner 接口定义和工厂函数
│   │   ├── codex.go       # Codex app-server 适配器
│   │   ├── claude.go      # Claude Code CLI 适配器
│   │   ├── opencode.go    # OpenCode CLI 适配器
│   │   └── prompt.go      # 提示模板渲染
│   ├── config/            # 配置解析和管理
│   │   └── config.go      # 配置结构、默认值、验证
│   ├── domain/            # 领域模型
│   │   └── entities.go    # 核心实体定义
│   ├── orchestrator/      # 核心编排器
│   │   └── orchestrator.go # 调度、协调、重试逻辑
│   ├── router/            # 路由配置
│   ├── server/            # HTTP 服务器
│   │   ├── server.go      # Gin 服务器设置
│   │   ├── handlers/      # HTTP 处理器
│   │   ├── presenter/     # 数据展示层
│   │   ├── components/    # UI 组件
│   │   └── static/        # 静态资源
│   ├── tracker/           # 问题跟踪器客户端
│   │   ├── tracker.go     # Tracker 接口定义
│   │   ├── linear.go      # Linear GraphQL 客户端
│   │   ├── github.go      # GitHub API 客户端
│   │   └── mock.go        # 测试用 Mock 客户端
│   ├── workflow/          # 工作流加载器
│   │   └── loader.go      # WORKFLOW.md 解析
│   ├── workspace/         # 工作空间管理
│   │   └── manager.go     # 创建、清理、钩子执行
│   └── common/            # 公共类型
├── docs/
│   └── SPEC.md            # OpenAI Symphony 规范文档
├── WORKFLOW.md            # 工作流配置和提示模板
└── README.md              # 项目说明
```

## 核心设计原则

### 1. 接口驱动设计

项目使用接口来解耦组件，便于测试和扩展：

```go
// Tracker 接口 - 问题跟踪器抽象
type Tracker interface {
    FetchCandidateIssues(ctx context.Context, activeStates []string) ([]*domain.Issue, error)
    FetchIssuesByStates(ctx context.Context, states []string) ([]*domain.Issue, error)
    FetchIssueStatesByIDs(ctx context.Context, ids []string) ([]*domain.Issue, error)
}

// Runner 接口 - 代理运行器抽象
type Runner interface {
    RunAttempt(ctx context.Context, issue *domain.Issue, workspacePath string,
        attempt *int, promptTemplate string, callback EventCallback) (*RunAttemptResult, error)
}
```

**规则**: 添加新的跟踪器或代理类型时，必须实现对应的接口。

### 2. 配置来源

配置按以下优先级加载：

1. CLI 参数（如 `-workflow`、`-port`）
2. WORKFLOW.md YAML 前置内容
3. 环境变量（`$VAR_NAME` 格式）
4. 内置默认值

**规则**: 新增配置项时，必须在 `config.go` 中定义默认值。

### 3. 工作空间隔离

每个问题在独立的工作空间中执行，工作空间路径为：
```
<workspace.root>/<sanitized_issue_identifier>
```

**安全规则**:
- 工作空间键名必须经过清理（只保留 `[A-Za-z0-9._-]`）
- 工作空间路径必须在配置的根目录内
- 代理进程的工作目录必须设置为工作空间路径

## 代码规范

### Go 代码风格

1. **命名规范**
   - 导出函数/类型使用 PascalCase
   - 私有函数/变量使用 camelCase
   - 接口名使用 `-er` 后缀（如 `Tracker`、`Runner`、`Loader`）

2. **错误处理**
   - 使用 `fmt.Errorf` 包装错误，保留上下文
   - 错误消息使用小写开头，不以句号结尾
   ```go
   if err != nil {
       return nil, fmt.Errorf("failed to create workspace: %w", err)
   }
   ```

3. **注释规范**
   - 包注释放在 `package` 语句之前
   - 导出类型/函数必须有注释
   - 注释以类型/函数名开头
   ```go
   // Issue 表示标准化的问题记录
   type Issue struct {
       // ID 是稳定的跟踪器内部 ID
       ID string `json:"id"`
   }
   ```

4. **Context 使用**
   - 所有长时间运行的操作必须接受 `context.Context`
   - 使用 `context.WithTimeout` 设置超时
   ```go
   func (m *Manager) CreateForIssue(ctx context.Context, identifier string) (*domain.Workspace, error)
   ```

### 并发模式

1. **状态保护**
   - 使用 `sync.RWMutex` 保护共享状态
   - 读操作使用 `RLock/RUnlock`
   - 写操作使用 `Lock/Unlock`
   ```go
   type Orchestrator struct {
       mu    sync.RWMutex
       state *domain.OrchestratorState
   }

   func (o *Orchestrator) GetState() *domain.OrchestratorState {
       o.mu.RLock()
       defer o.mu.RUnlock()
       // ...
   }
   ```

2. **Goroutine 管理**
   - 使用 `context.Context` 控制 goroutine 生命周期
   - 启动 goroutine 时传递 context

### 日志规范

1. **结构化日志**
   - 使用 `key=value` 格式
   - 包含操作结果
   ```go
   fmt.Printf("worker for %s completed successfully (turns: %d)\n", issue.Identifier, result.TurnCount)
   ```

2. **必须记录的事件**
   - 会话启动/结束
   - 配置变更
   - 重试调度
   - 错误和超时

## 扩展指南

### 添加新的问题跟踪器

1. 在 `internal/tracker/` 创建新文件（如 `jira.go`）
2. 实现 `Tracker` 接口
3. 在 `tracker.go` 的 `NewTracker` 工厂函数中添加分支
4. 更新 `config.go` 添加必要的配置字段
5. 更新 `ValidateDispatchConfig` 添加验证逻辑

### 添加新的代理类型

1. 在 `internal/agent/` 创建新文件
2. 实现 `Runner` 接口
3. 在 `runner.go` 的 `NewRunner` 工厂函数中添加分支
4. 如有需要，在 `config.go` 添加专用配置结构

### 添加新的钩子

1. 在 `config.go` 的 `HooksConfig` 结构添加字段
2. 在 `workspace/manager.go` 添加执行方法
3. 在适当的位置调用钩子（参考现有钩子）

## 配置验证规则

配置验证在 `ValidateDispatchConfig` 中实现，关键规则：

| 字段 | 规则 |
|------|------|
| `tracker.kind` | 必需，支持 `linear`、`github`、`mock` |
| `tracker.api_key` | 非 mock 类型必需，支持 `$VAR` 格式 |
| `tracker.project_slug` | Linear 类型必需 |
| `tracker.repo` | GitHub 类型必需，格式 `owner/repo` |
| `agent.kind` | 支持 `codex`、`claude`、`opencode` |
| `codex.command` | Codex 类型必需 |

## HTTP API 端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/` | GET | 仪表板界面 |
| `/api/v1/state` | GET | 获取当前状态 |
| `/api/v1/:identifier` | GET | 获取问题详情 |
| `/api/v1/refresh` | POST | 触发刷新 |

## 测试规范

### 测试文件组织

1. **文件命名**: `*_test.go`，与被测文件同目录
2. **包命名**: 使用 `<package>_test` 形式进行外部测试，或直接使用相同包名进行内部测试
   ```go
   // 外部测试（推荐，测试公开 API）
   package config_test

   // 内部测试（需要访问私有成员时使用）
   package config
   ```

3. **测试函数命名**: `Test<FunctionName>` 或 `Test<Scenario>`

### 测试风格和模式

#### 1. 表驱动测试（必须使用）

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

#### 2. 子测试组织

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

#### 3. 临时资源清理

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

#### 4. 环境变量测试

设置测试环境变量后必须清理：

```go
func TestParseConfig(t *testing.T) {
    os.Setenv("TEST_API_KEY", "test-key-123")
    defer os.Unsetenv("TEST_API_KEY")  // 必须清理

    // 测试代码...
}
```

### 测试覆盖要求

#### 必须测试的场景

| 模块 | 必须覆盖的场景 |
|------|----------------|
| `config` | 默认值、环境变量解析、验证成功/失败 |
| `workflow` | 空内容、无前置内容、有效前置内容、无效 YAML、非映射前置内容 |
| `workspace` | 创建工作空间、复用工作空间、路径清理、删除工作空间、路径验证 |
| `tracker/mock` | 获取候选问题、按状态获取、按 ID 获取、状态更新 |
| `domain` | 所有实体字段、状态常量、边界值 |
| `orchestrator` | 调度逻辑、重试机制、协调逻辑、并发控制 |

#### 覆盖率要求

- 新增代码必须有对应测试
- Bug 修复必须添加回归测试
- 目标覆盖率: 核心逻辑 > 80%

### Mock 使用

#### Mock Tracker

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

#### 测试配置

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

### 运行测试

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

### 测试检查清单

新增代码时，请确认：

- [ ] 测试文件与被测文件同目录
- [ ] 使用表驱动测试覆盖多场景
- [ ] 使用 `t.Run()` 组织子测试
- [ ] 临时资源（文件、目录、环境变量）使用 `defer` 清理
- [ ] 测试函数名清晰描述测试场景
- [ ] 错误消息包含期望值和实际值
- [ ] 覆盖了正常路径和错误路径
- [ ] 边界值和空值情况已测试

### 集成测试

#### 使用 Mock Workflow

项目提供了 `test/test-mock-workflow.md` 用于测试：

```bash
# 使用 mock 配置运行服务
./bin/symphony -workflow test/test-mock-workflow.md -port 8080
```

#### 验证点

集成测试应验证：

1. **Mock Tracker 正常工作** - 问题能正确获取
2. **状态流转逻辑** - 状态变更被正确处理
3. **编排调度功能** - 问题按优先级调度
4. **工作空间创建/清理** - 工作空间正确管理

## 钩子脚本规则

钩子在 WORKFLOW.md 中定义，执行规则：

| 钩子 | 触发时机 | 失败行为 |
|------|----------|----------|
| `after_create` | 新工作空间创建后 | 中止创建 |
| `before_run` | 每次代理运行前 | 中止运行 |
| `after_run` | 每次代理运行后 | 忽略 |
| `before_remove` | 工作空间删除前 | 忽略 |

**环境变量**:
- `SYMPHONY_WORKSPACE`: 工作空间路径
- `SYMPHONY_HOOK`: 钩子名称

## 重试机制

1. **正常退出**: 安排 1 秒后的续行重试
2. **异常退出**: 指数退避重试
   - 公式: `delay = min(10000 * 2^(attempt-1), max_backoff)`
   - 默认最大退避: 5 分钟

## 禁止事项

1. **不要** 直接修改 `internal/domain/entities.go` 中的核心实体，除非遵循 SPEC.md
2. **不要** 在 orchestrator 之外的地方直接操作运行时状态
3. **不要** 跳过工作空间路径验证（安全约束）
4. **不要** 在日志中输出敏感信息（API 密钥等）
5. **不要** 创建循环依赖（保持包的清晰层次）

## 参考资料

- `docs/SPEC.md`: OpenAI Symphony 规范文档
- `WORKFLOW.md`: 配置示例和提示模板
- `README.md`: 项目使用说明