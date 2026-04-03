# AGENTS.md

本文档是 AI 代理的导航入口。详细信息请查阅 `docs/` 目录下的专门文档。

## 项目概述

Symphony 是一个编码代理编排服务，基于 [OpenAI Symphony SPEC](https://github.com/openai/symphony/blob/main/SPEC.md) 实现。它持续从问题跟踪器读取工作，为每个问题创建隔离的工作空间，并运行编码代理会话。

**技术栈**: Go 1.22 / Gin / YAML + Markdown 模板

## 快速参考

| 主题 | 文档 | 用途 |
|------|------|------|
| 架构设计 | [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | 理解项目结构和核心设计原则 |
| 代码规范 | [docs/CODING_STANDARDS.md](docs/CODING_STANDARDS.md) | 编码风格、错误处理、并发模式 |
| 测试规范 | [docs/TESTING.md](docs/TESTING.md) | 测试风格、覆盖率要求、Mock 使用 |
| 扩展指南 | [docs/EXTENSION_GUIDE.md](docs/EXTENSION_GUIDE.md) | 添加新跟踪器、代理类型、钩子 |
| API 端点 | [docs/API.md](docs/API.md) | HTTP API 规范 |
| 配置验证 | [docs/CONFIGURATION.md](docs/CONFIGURATION.md) | 配置字段规则和优先级 |

## 项目结构

```
symphony/
├── cmd/symphony/          # 主程序入口
├── internal/
│   ├── agent/             # 代理运行器 (Runner 接口)
│   ├── config/            # 配置解析和验证
│   ├── domain/            # 核心实体定义
│   ├── orchestrator/      # 编排器 (调度、重试)
│   ├── server/            # HTTP 服务器和处理器
│   ├── tracker/           # 问题跟踪器 (Tracker 接口)
│   ├── workflow/          # WORKFLOW.md 解析
│   └── workspace/         # 工作空间管理
├── docs/                   # 知识库 (本目录)
├── WORKFLOW.md            # 工作流配置和提示模板
└── README.md              # 项目说明
```

## 核心接口

```go
// Tracker - 问题跟踪器抽象
type Tracker interface {
    FetchCandidateIssues(ctx context.Context, activeStates []string) ([]*domain.Issue, error)
    FetchIssuesByStates(ctx context.Context, states []string) ([]*domain.Issue, error)
    FetchIssueStatesByIDs(ctx context.Context, ids []string) ([]*domain.Issue, error)
}

// Runner - 代理运行器抽象
type Runner interface {
    RunAttempt(ctx context.Context, issue *domain.Issue, workspacePath string,
        attempt *int, promptTemplate string, callback EventCallback) (*RunAttemptResult, error)
}
```

## 关键约束

| 约束 | 说明 |
|------|------|
| 工作空间隔离 | 每个问题在独立工作空间执行，路径必须经过清理和验证 |
| 接口驱动 | 新增跟踪器/代理必须实现对应接口 |
| 配置优先级 | CLI 参数 > WORKFLOW.md > 环境变量 > 默认值 |

## 禁止事项

1. 不要直接修改 `internal/domain/entities.go` 核心实体（除非遵循 SPEC.md）
2. 不要在 orchestrator 之外操作运行时状态
3. 不要跳过工作空间路径验证
4. 不要在日志中输出敏感信息
5. 不要创建循环依赖

## 参考资料

- `docs/SPEC.md` - OpenAI Symphony 规范文档
- `WORKFLOW.md` - 配置示例和提示模板
- `README.md` - 项目使用说明