# 架构设计

本文档描述 Symphony 的架构设计和核心原则。

## 项目结构详解

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

### 2. 配置来源优先级

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

## 重试机制

1. **正常退出**: 安排 1 秒后的续行重试
2. **异常退出**: 指数退避重试
   - 公式: `delay = min(10000 * 2^(attempt-1), max_backoff)`
   - 默认最大退避: 5 分钟

## 钩子系统

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