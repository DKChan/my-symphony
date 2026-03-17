# Symphony - 编码代理编排服务

基于 [OpenAI Symphony SPEC](https://github.com/openai/symphony/blob/main/SPEC.md) 实现的编码代理编排服务。

## 概述

Symphony 是一个长期运行的自动化服务，它持续从问题跟踪器（如 Linear）读取工作，为每个问题创建隔离的工作空间，并在工作空间内运行编码代理会话。

### 核心功能

- **自动化工作流**: 将问题执行转换为可重复的守护进程工作流
- **隔离执行**: 在每个问题的工作空间中隔离代理执行
- **版本控制策略**: 工作流策略保存在仓库的 `WORKFLOW.md` 中
- **可观测性**: 提供运行状态监控和调试能力

## 快速开始

### 前置要求

- Go 1.22+
- Linear API 密钥
- Codex CLI（或兼容的编码代理）

### 安装

```bash
# 克隆仓库
git clone https://github.com/your-org/symphony.git
cd symphony

# 编译
go build -o bin/symphony ./cmd/symphony/
```

### 配置

1. 复制并编辑 `WORKFLOW.md`:

```bash
cp WORKFLOW.md.example WORKFLOW.md
```

2. 设置环境变量:

```bash
export LINEAR_API_KEY="your-linear-api-key"
```

3. 编辑 `WORKFLOW.md` 中的 `project_slug` 为你的 Linear 项目标识。

### 运行

```bash
# 使用默认配置运行
./bin/symphony

# 指定工作流文件
./bin/symphony -workflow /path/to/WORKFLOW.md

# 启用 HTTP 服务器
./bin/symphony -port 8080
```

## 工作流配置

`WORKFLOW.md` 文件包含 YAML 前置内容和 Markdown 提示模板。

### 配置项

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `tracker.kind` | 跟踪器类型 | `linear` |
| `tracker.api_key` | API 密钥（支持 `$VAR` 格式） | - |
| `tracker.project_slug` | Linear 项目标识 | - |
| `tracker.active_states` | 活跃状态列表 | `["Todo", "In Progress"]` |
| `tracker.terminal_states` | 终态列表 | `["Done", "Cancelled", "Closed"]` |
| `polling.interval_ms` | 轮询间隔（毫秒） | `30000` |
| `workspace.root` | 工作空间根目录 | `/tmp/symphony_workspaces` |
| `agent.max_concurrent_agents` | 最大并发代理数 | `10` |
| `agent.max_turns` | 最大轮次 | `20` |
| `codex.command` | Codex 命令 | `codex app-server` |

### 钩子

| 钩子 | 触发时机 | 失败行为 |
|------|----------|----------|
| `after_create` | 新工作空间创建后 | 中止创建 |
| `before_run` | 每次代理运行前 | 中止运行 |
| `after_run` | 每次代理运行后 | 忽略 |
| `before_remove` | 工作空间删除前 | 忽略 |

## HTTP API

当启用 HTTP 服务器时，提供以下端点：

- `GET /` - 仪表板界面
- `GET /api/v1/state` - 获取当前状态
- `GET /api/v1/:identifier` - 获取问题详情
- `POST /api/v1/refresh` - 触发刷新

## 项目结构

```
symphony/
├── cmd/symphony/        # 主程序入口
├── internal/
│   ├── agent/           # 代理运行器
│   ├── config/          # 配置解析
│   ├── domain/          # 领域模型
│   ├── orchestrator/    # 核心编排器
│   ├── server/          # HTTP 服务器
│   ├── tracker/         # 问题跟踪器客户端
│   └── workspace/       # 工作空间管理
├── WORKFLOW.md          # 工作流配置
└── README.md            # 本文档
```

## 架构

```
┌─────────────────────────────────────────────────────────────┐
│                     Symphony Service                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │   Workflow   │    │  Orchestrator │    │    Server    │  │
│  │    Loader    │───>│   (核心调度)   │<───│   (HTTP)     │  │
│  └──────────────┘    └───────┬──────┘    └──────────────┘  │
│                              │                              │
│         ┌────────────────────┼────────────────────┐        │
│         │                    │                    │        │
│         ▼                    ▼                    ▼        │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │   Tracker    │    │  Workspace   │    │    Agent     │  │
│  │   (Linear)   │    │   Manager    │    │   Runner     │  │
│  └──────────────┘    └──────────────┘    └──────────────┘  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## 开发

### 运行测试

```bash
go test ./...
```

### 代码检查

```bash
golangci-lint run
```

## 许可证

MIT License