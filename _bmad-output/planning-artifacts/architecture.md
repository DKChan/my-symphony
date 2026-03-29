---
stepsCompleted: [1, 2, 3, 4, 5, 6, 7, 8]
inputDocuments:
  - _bmad-output/planning-artifacts/prd.md
workflowType: 'architecture'
project_name: 'my-symphony'
user_name: 'DK'
date: '2026-03-29'
lastStep: 8
status: 'complete'
completedAt: '2026-03-29'
---

# Architecture Decision Document

_This document builds collaboratively through step-by-step discovery. Sections are appended as we work through each architectural decision together._

## Document Setup

**Input Documents:**
- PRD: `_bmad-output/planning-artifacts/prd.md`

**Historical Reference:**
- `docs/SPEC.md` (仅供参考)

## Project Context Analysis

### Requirements Overview

**Functional Requirements: 56 条，8 个能力领域**

| 能力领域 | FR 数量 | 架构影响 |
|----------|---------|----------|
| 初始化与配置管理 | 8 | CLI 入口、配置系统 |
| 任务生命周期管理 | 5 | 状态机、任务编排 |
| 需求澄清 | 8 | AI Agent 调用、Web 交互 |
| 规则生成与管理 | 9 | BDD/TDD 引擎、规则存储 |
| 执行与监控 | 10 | AI Agent CLI 集成、Web 实时更新 |
| 验收与报告 | 4 | 报告生成、测试结果聚合 |
| 异常处理与恢复 | 5 | 错误处理、状态恢复 |
| 外部集成 | 7 | Beads tracker、Git、AI Agent CLI |

**Non-Functional Requirements: 5 条**

| NFR | 架构约束 |
|-----|----------|
| NFR1: AI Agent CLI 等待 >24h | 长时间运行进程管理 |
| NFR2: Beads CLI 可用性检测 | 启动时健康检查 |
| NFR3: 崩溃后状态恢复 | 状态持久化到 Beads |
| NFR4: 配置重启生效 | 无热加载需求 |
| NFR5: 执行日志记录 | 日志系统、可观测性 |

### Scale & Complexity

**Primary domain:** CLI Tool + Daemon Service + Local Web UI

**Complexity level:** Medium

**Estimated architectural components:** 6-8 核心模块

**复杂度指标:**

| 指标 | 状态 | 影响 |
|------|------|------|
| 实时特性 | ✅ | Web UI 状态实时更新 |
| 多租户 | ❌ | 单用户本地服务 |
| 合规要求 | ❌ | 无 |
| 集成复杂度 | 中等 | Beads + AI Agent CLI + Git |
| 用户交互复杂度 | 中等 | Web 看板 + 澄清对话 |
| 数据复杂度 | 低 | 本地 CLI tracker |

### Core Architectural Challenges

1. **AI Agent CLI 集成层**
   - 长时间运行（>24h）的进程管理
   - 输出解析和错误处理
   - 状态监控和进度提取

2. **状态机引擎**
   - 任务状态流转（7 个主状态 + 4 个异常状态）
   - 阶段间依赖阻塞
   - 崩溃后恢复

3. **规则引擎**
   - BDD 规则生成和存储
   - TDD 规则生成和存储
   - 规则执行验证

4. **Web UI 实时通信**
   - 任务状态变化实时推送
   - AI Agent 执行进度更新
   - 日志流式展示

### Technical Constraints & Dependencies

| 约束 | 说明 |
|------|------|
| 本地服务 | 无需远程部署，localhost only |
| Beads tracker | 本地 CLI，必需依赖 |
| AI Agent CLI | 外部进程调用，长时间运行 |
| Git | 默认集成，提交为结束点 |
| 平台 | Linux/macOS MVP，Windows 后续 |

### Cross-Cutting Concerns

| 关注点 | 影响范围 |
|--------|----------|
| 配置管理 | CLI、Daemon、所有模块 |
| 错误处理 | AI Agent 调用、Beads 集成、Web UI |
| 日志/可观测性 | 所有模块 |
| 状态持久化 | 任务状态、规则、对话记录 |

## Starter Template Evaluation

### Brownfield Project - 现有技术栈

**项目类型:** Brownfield（现有代码库）

**无需新 Starter Template** - 项目已有成熟的技术基础。

### 现有技术栈

**Language & Runtime:**
- Go 1.22

**Web Framework:**
- Gin v1.9.1

**现有模块结构:**
```
cmd/symphony/main.go          # CLI 入口
internal/
├── agent/                    # AI Agent 集成
├── config/                   # 配置管理
├── orchestrator/             # 编排器
├── workflow/                 # 工作流加载
├── domain/                   # 领域实体
├── workspace/                # 工作区管理
├── tracker/                  # Tracker 集成
├── server/                   # Web 服务器 (handlers, SSE, static)
├── router/                   # 路由
└── common/                   # 公共类型
```

### 已实现的功能

| 模块 | 状态 | 说明 |
|------|------|------|
| AI Agent 集成 | ✅ | claude, codex, opencode |
| Tracker 接口 | ✅ | linear, github, mock |
| 工作流加载 | ✅ | YAML + Markdown 解析 |
| 配置管理 | ✅ | YAML 配置 |
| Web 服务器 | ✅ | Gin + SSE 实时推送 |
| 编排器 | ✅ | 任务编排 |

### 现有架构模式

| 模式 | 应用 |
|------|------|
| Clean Architecture | 分层结构 |
| 依赖注入 | 接口抽象 |
| Factory Pattern | Tracker, Agent 创建 |
| 接口隔离 | Tracker, Agent 接口 |

### 依赖清单

| 依赖 | 版本 | 用途 |
|------|------|------|
| gin-gonic/gin | v1.9.1 | Web 框架 |
| gopkg.in/yaml.v3 | v3.0.1 | YAML 解析 |
| radovskyb/watcher | v1.0.7 | 文件监控 |
| stretchr/testify | v1.8.3 | 测试框架 |

## Core Architectural Decisions

### Decision Priority Analysis

**Critical Decisions (已完成):**
1. 数据架构 - 职责边界划分
2. 状态机架构 - Workflow Engine 设计
3. Prompt 管理 - 阶段内嵌方案

**Important Decisions (已完成):**
1. 错误处理 - 前缀约定方案

**Deferred Decisions (Phase 2):**
1. workflow.md 配置解析 - 自定义工作流支持

### Data Architecture

#### 职责边界

| 数据类型 | 存储位置 | 管理者 |
|----------|----------|--------|
| 任务 | Beads | Symphony |
| 任务状态 | Beads | Symphony |
| 需求对话记录 | Beads | Symphony |
| BDD/TDD 规则 | 工程内 `docs/` 或用户自定义 | AI Agent |
| 架构设计文档 | 工程内 | AI Agent |
| 验收报告 | 工程内 | AI Agent |

**架构影响：** Symphony 不需要规则存储模块，只需触发 AI Agent 生成/读取规则文件。

#### 目录结构

```
.sym/
├── config.yaml           # 系统配置
├── workflow.md           # 阶段编排定义（Phase 2 解析）
└── prompts/              # Prompt 文件目录
    ├── clarification.md
    ├── bdd.md
    ├── architecture.md
    ├── implementation.md
    └── verification.md
```

### State Machine Architecture

#### MVP 实现

**方案：** 硬编码默认 workflow

**阶段流转：**
```
待开始 → 需求澄清中 → 待审核BDD → 待审核架构 → 实现中 → 待验收 → 完成
             ↓              ↓           ↓          ↓
          (超限跳过)     (驳回)      (驳回)     (失败)
             ↓              ↓           ↓          ↓
          需求不完整      待设计      待设计    待人工处理
```

**Phase 2 扩展：**
- 从 workflow.md 解析自定义阶段定义

#### 架构组件

```
┌─────────────────────────────────────────────────────────┐
│                    Workflow Engine                       │
├─────────────────────────────────────────────────────────┤
│  DefaultWorkflow (硬编码)                                │
│  ├── clarification → bdd_review → architecture_review   │
│  │   → implementation → verification                    │
│  │                                                       │
│  StageExecutor                                           │
│  ├── 判断人工/自动节点                                    │
│  └── 触发下一阶段                                        │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
                 ┌─────────────────┐
                 │  Orchestrator   │ (现有调度器)
                 │  - Agent 调度   │
                 │  - 运行管理     │
                 │  - 重试逻辑     │
                 └─────────────────┘
```

### Error Handling

**方案：** 使用现有 error + 前缀约定

**格式：** `<模块>.<错误类型>: <简短描述>`

**示例：**
```go
errors.New("config.not_found: 配置文件不存在")
errors.New("tracker.unavailable: Beads CLI 不可用")
errors.New("agent.timeout: Agent 执行超时")
errors.New("prompt.parse_error: Prompt 解析失败")
```

**错误码模块前缀：**
| 模块 | 前缀 |
|------|------|
| 配置 | `config.` |
| Tracker | `tracker.` |
| Agent | `agent.` |
| Prompt | `prompt.` |
| Workflow | `workflow.` |

### Prompt Management

**方案：** 阶段内嵌 prompt 路径

**数据结构：**
```go
type StageDefinition struct {
    Name         string // 阶段名称
    Trigger      string // "agent" 或 "human"
    PromptPath   string // prompt 文件路径
    Next         string // 下一阶段名称
    OnFailure    string // 失败时跳转阶段
}
```

**默认阶段定义：**
| 阶段 | Trigger | Prompt Path | Next |
|------|---------|-------------|------|
| clarification | agent | `.sym/prompts/clarification.md` | bdd_review |
| bdd_review | human | - | architecture_review |
| architecture_review | human | - | implementation |
| implementation | agent | `.sym/prompts/implementation.md` | verification |
| verification | agent | `.sym/prompts/verification.md` | completed |

### Decision Impact Analysis

**Implementation Sequence:**
1. 定义 StageDefinition 结构体
2. 实现 DefaultWorkflow 硬编码阶段
3. 实现 StageExecutor 阶段流转逻辑
4. 扩展 Orchestrator 集成 Workflow Engine
5. 实现错误码前缀约定

**Cross-Component Dependencies:**
- Workflow Engine → Orchestrator（调用 Agent 调度）
- Workflow Engine → Prompt 加载（读取 prompt 文件）
- StageExecutor → Tracker（更新任务状态）

## Implementation Patterns & Consistency Rules

### 命名模式

| 类别 | 规则 | 示例 |
|------|------|------|
| 包名 | 小写单词 | `tracker`, `agent`, `orchestrator` |
| 接口 | PascalCase | `Tracker`, `Runner` |
| 导出函数 | PascalCase | `NewTracker`, `FetchCandidateIssues` |
| 私有函数 | camelCase | `buildPrompt`, `removeBlock` |
| 变量 | camelCase | `trackerClient`, `workspaceMgr` |
| 常量 | PascalCase | `StatusPreparingWorkspace` |
| 错误变量 | Err前缀 | `ErrMissingWorkflowFile` |
| 阶段名 | snake_case | `clarification`, `bdd_review` |
| 状态常量 | PascalCase | `StagePending`, `StageRunning` |

### 接口模式

```go
// 接口定义 + Factory 函数
type Tracker interface { ... }
func NewTracker(cfg *config.Config) Tracker { ... }

// 测试用接口（Go 隐式实现）
type MockOrchestrator struct {}
func (m *MockOrchestrator) Dispatch(ctx context.Context, issue *domain.Issue) error { ... }
```

### 错误处理模式

```go
// internal/common/errors/errors.go - 集中管理
package errors

import "errors"

var (
    // Config 模块
    ErrConfigNotFound = errors.New("config.not_found: 配置文件不存在")
    ErrConfigInvalid  = errors.New("config.invalid: 配置格式无效")

    // Tracker 模块
    ErrTrackerUnavailable = errors.New("tracker.unavailable: Beads CLI 不可用")

    // Agent 模块
    ErrAgentTimeout = errors.New("agent.timeout: Agent 执行超时")
    ErrAgentFailed  = errors.New("agent.failed: Agent 执行失败")

    // Workflow 模块
    ErrWorkflowInvalid     = errors.New("workflow.invalid: 工作流定义无效")
    ErrStageTransitionFail = errors.New("workflow.stage_failed: 阶段流转失败")

    // Prompt 模块
    ErrPromptNotFound  = errors.New("prompt.not_found: Prompt 文件不存在")
    ErrPromptParseErr  = errors.New("prompt.parse_error: Prompt 解析失败")
)
```

### 日志模式

```go
import "log/slog"

slog.Debug("polling for issues", "interval_ms", cfg.Polling.IntervalMs)
slog.Info("stage transition", "task_id", taskID, "from", from, "to", to)
slog.Warn("clarification limit exceeded", "task_id", taskID, "rounds", rounds)
slog.Error("agent execution failed", "task_id", taskID, "error", err)

// 字段命名：snake_case
```

### 测试模式

```go
// 表驱动测试
func TestWorkflowEngine_ExecuteStage(t *testing.T) {
    engine := &workflowEngine{
        tracker:      &MockTracker{},
        orchestrator: &MockOrchestrator{},
        promptLoader: &MockPromptLoader{},
    }

    tests := []struct {
        name    string
        taskID  string
        wantErr error
    }{
        {"success case", "TASK-001", nil},
        {"not found", "TASK-999", errors.ErrPromptNotFound},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := engine.ExecuteStage(context.Background(), tt.taskID)
            if tt.wantErr != nil {
                assert.ErrorContains(t, err, tt.wantErr.Error())
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Mock 策略

| 阶段 | 策略 |
|------|------|
| MVP | 手写 Mock |
| Phase 2+ | mockgen 工具 |

### 覆盖率目标

| 阶段 | 目标 |
|------|------|
| MVP | 60% |
| Phase 2+ | 80% |

### 强制规则

**All AI Agents MUST:**

1. 错误码定义在 `internal/common/errors/errors.go`
2. 使用 `log/slog` 结构化日志
3. 阶段名 `snake_case`，状态常量 `PascalCase`
4. 测试使用表驱动模式
5. 测试文件与源文件同目录

## Project Structure & Boundaries

### Complete Project Directory Structure

```
symphony/
├── cmd/
│   └── symphony/
│       └── main.go              # CLI 入口
├── internal/
│   ├── agent/                   # AI Agent 集成
│   │   ├── runner.go            # Agent 执行器
│   │   ├── runner_test.go
│   │   ├── prompt.go            # Prompt 构建
│   │   ├── prompt_test.go
│   │   ├── claude.go            # Claude CLI 集成
│   │   ├── claude_test.go
│   │   ├── codex.go             # Codex CLI 集成
│   │   ├── codex_test.go
│   │   ├── opencode.go          # OpenCode CLI 集成
│   │   └── opencode_test.go
│   ├── common/                  # 公共类型和错误
│   │   ├── types.go
│   │   ├── types_test.go
│   │   └── errors/              # [新增]
│   │       └── errors.go        # 集中错误码
│   ├── config/                  # 配置管理
│   │   ├── config.go
│   │   └── config_test.go
│   ├── domain/                  # 领域实体
│   │   ├── entities.go
│   │   └── entities_test.go
│   ├── orchestrator/            # 编排器
│   │   ├── orchestrator.go
│   │   └── orchestrator_test.go
│   ├── router/                  # 路由
│   │   ├── router.go
│   │   └── router_test.go
│   ├── server/                  # Web 服务器
│   │   ├── server.go
│   │   ├── server_test.go
│   │   ├── handlers/
│   │   │   ├── api_handler.go
│   │   │   ├── dashboard_handler.go
│   │   │   ├── sse_handler.go
│   │   │   ├── static_handler.go
│   │   │   └── handlers_test.go
│   │   ├── presenter/
│   │   │   ├── presenter.go
│   │   │   └── presenter_test.go
│   │   ├── components/
│   │   │   ├── dashboard.go
│   │   │   └── components_test.go
│   │   └── static/
│   │       └── embed.go
│   ├── tracker/                 # Tracker 集成
│   │   ├── tracker.go           # 接口定义
│   │   ├── linear.go            # Linear 实现
│   │   ├── github.go            # GitHub 实现
│   │   ├── mock.go              # Mock 实现
│   │   ├── mock_test.go
│   │   └── beads.go             # [新增] Beads 实现
│   ├── workflow/                # 工作流
│   │   ├── loader.go            # workflow.md 加载
│   │   ├── loader_test.go
│   │   ├── engine.go            # [新增] Workflow Engine
│   │   ├── engine_test.go       # [新增]
│   │   └── stages.go            # [新增] 阶段定义
│   └── workspace/               # 工作区管理
│       ├── manager.go
│       └── manager_test.go
├── docs/
│   └── SPEC.md                  # 历史文档
├── test/                        # 测试数据
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── WORKFLOW.md
```

### New Files Summary

| 文件 | 说明 |
|------|------|
| `internal/tracker/beads.go` | Beads tracker 实现 |
| `internal/workflow/engine.go` | Workflow Engine 核心 |
| `internal/workflow/engine_test.go` | Engine 测试 |
| `internal/workflow/stages.go` | 阶段定义 |
| `internal/common/errors/errors.go` | 集中错误码 |

### Architectural Boundaries

```
┌─────────────────────────────────────────────────────────────┐
│                      CLI Entry (cmd/)                        │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   Workflow Engine (新增)                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │   stages.go │  │  engine.go  │  │ 阶段流转逻辑 │          │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
└─────────────────────────────────────────────────────────────┘
         │                    │                    │
         ▼                    ▼                    ▼
┌─────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Tracker   │    │   Orchestrator  │    │      Agent      │
│ (Beads新增) │    │   (现有调度器)   │    │ (claude/codex)  │
└─────────────┘    └─────────────────┘    └─────────────────┘
         │                    │                    │
         └────────────────────┼────────────────────┘
                              ▼
                    ┌─────────────────┐
                    │   Web Server    │
                    │ (Gin + SSE)     │
                    └─────────────────┘
```

### Data Flow

```
用户需求 ──▶ Tracker (Beads) ──▶ Workflow Engine ──▶ Orchestrator ──▶ Agent
                │                      │                  │
                ▼                      ▼                  ▼
           任务状态               阶段状态更新         Agent 执行
           对话记录               人工审核等待         日志输出
```

### Requirements to Structure Mapping

| FR 类别 | 模块 | 文件 |
|---------|------|------|
| 初始化与配置管理 (FR1-8) | config, cmd | `config/config.go`, `cmd/symphony/main.go` |
| 任务生命周期管理 (FR9-13) | workflow, tracker | `workflow/engine.go`, `tracker/beads.go` |
| 需求澄清 (FR14-21) | workflow, agent | `workflow/stages.go`, `agent/prompt.go` |
| 规则生成与管理 (FR22-30) | workflow, server | `workflow/engine.go`, `server/handlers/` |
| 执行与监控 (FR31-40) | orchestrator, agent | `orchestrator/orchestrator.go`, `agent/runner.go` |
| 验收与报告 (FR41-44) | workflow, server | `workflow/stages.go`, `server/handlers/` |
| 异常处理与恢复 (FR45-49) | workflow, orchestrator | `workflow/engine.go`, `orchestrator/orchestrator.go` |
| 外部集成 (FR50-56) | tracker, agent | `tracker/beads.go`, `agent/*.go` |

### Integration Points

**内部通信：**

| 组件间 | 通信方式 |
|--------|----------|
| Workflow Engine ↔ Orchestrator | Go 函数调用 |
| Orchestrator ↔ Agent | goroutine + channel |
| Server ↔ Orchestrator | SSE (Server-Sent Events) |
| Engine ↔ Tracker | Go 函数调用 |

**外部集成：**

| 集成点 | 方式 | 文件 |
|--------|------|------|
| Beads CLI | 命令行调用 | `tracker/beads.go` |
| AI Agent CLI | 命令行调用 | `agent/claude.go`, `agent/codex.go`, `agent/opencode.go` |
| Git | 命令行调用 | `workspace/manager.go` |
| Web UI | HTTP + SSE | `server/` |

## Interface Definitions

### Tracker Interface

```go
// internal/tracker/tracker.go

// Tracker Tracker 接口（供 mock 测试）
type Tracker interface {
    CreateTask(ctx context.Context, title, description string) (*BeadsIssue, error)
    GetTask(ctx context.Context, identifier string) (*BeadsIssue, error)
    UpdateStage(ctx context.Context, identifier string, state *StageState) error
    AppendConversation(ctx context.Context, identifier string, turn ConversationTurn) error
    ListTasksByState(ctx context.Context, states []string) ([]*BeadsIssue, error)
    GetTasksByStage(ctx context.Context, stages []string) ([]*BeadsIssue, error)
    CheckAvailability() error
}
```

### Data Structures

```go
// BeadsIssue Beads 任务结构（含自定义字段）
type BeadsIssue struct {
    ID          string         `json:"id"`
    Identifier  string         `json:"identifier"`
    Title       string         `json:"title"`
    Description string         `json:"description"`
    State       string         `json:"state"`
    Custom      map[string]any `json:"custom"` // 自定义字段
}

// StageState 阶段状态
type StageState struct {
    CurrentStage string             `json:"current_stage"`
    StageData    map[string]any     `json:"stage_data,omitempty"`
    Conversation []ConversationTurn `json:"conversation,omitempty"`
}

// ConversationTurn 对话轮次
type ConversationTurn struct {
    Role      string `json:"role"`      // "ai" or "user"
    Content   string `json:"content"`
    Timestamp int64  `json:"timestamp"`
}
```

### Beads Client Implementation

```go
// internal/tracker/beads.go

type BeadsClient struct {
    beadsPath string // beads CLI 路径
    repoPath  string // 项目路径
}

func NewBeadsClient(beadsPath, repoPath string) *BeadsClient

// CreateTask 创建任务（含超时控制）
func (b *BeadsClient) CreateTask(ctx context.Context, title, description string) (*BeadsIssue, error) {
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    // ...
}

// GetTask 获取任务（含阶段状态）
func (b *BeadsClient) GetTask(ctx context.Context, identifier string) (*BeadsIssue, error)

// UpdateStage 更新任务阶段
func (b *BeadsClient) UpdateStage(ctx context.Context, identifier string, state *StageState) error

// AppendConversation 追加对话记录
func (b *BeadsClient) AppendConversation(ctx context.Context, identifier string, turn ConversationTurn) error

// ListTasksByState 列出指定状态的任务
func (b *BeadsClient) ListTasksByState(ctx context.Context, states []string) ([]*BeadsIssue, error)

// GetTasksByStage 批量获取指定阶段的任务（用于启动恢复）
func (b *BeadsClient) GetTasksByStage(ctx context.Context, stages []string) ([]*BeadsIssue, error)

// CheckAvailability 检查 Beads CLI 可用性
func (b *BeadsClient) CheckAvailability() error
```

### Error Definitions

```go
// internal/common/errors/errors.go

var (
    // Tracker 相关错误
    ErrTrackerNotFound      = errors.New("tracker.not_found: Beads CLI 路径不存在")
    ErrTrackerUnavailable   = errors.New("tracker.unavailable: Beads CLI 执行失败")
    ErrTrackerInvalidOutput = errors.New("tracker.invalid_output: Beads CLI 输出格式无效")
    ErrTrackerTimeout       = errors.New("tracker.timeout: Beads CLI 执行超时")
)
```

### Stage State Serialization

**序列化格式（存储在 Beads Custom 字段）：**

```json
{
  "current_stage": "clarification",
  "stage_data": {
    "clarification_rounds": 3,
    "clarification_limit": 5
  },
  "conversation": [
    {
      "role": "ai",
      "content": "登录方式是邮箱还是手机号？",
      "timestamp": 1711718400
    },
    {
      "role": "user",
      "content": "邮箱",
      "timestamp": 1711718500
    }
  ]
}
```

### Crash Recovery Flow

```go
// internal/workflow/engine.go

// RestoreFromBeads 从 Beads 恢复状态
func (e *WorkflowEngine) RestoreFromBeads(ctx context.Context, taskID string) error {
    // 1. 获取任务
    issue, err := e.tracker.GetTask(ctx, taskID)
    if err != nil {
        return fmt.Errorf("%w: %v", errors.ErrTrackerUnavailable, err)
    }

    // 2. 解析阶段状态
    stageState, err := parseStageState(issue.Custom)
    if err != nil {
        return fmt.Errorf("%w: %v", errors.ErrWorkflowInvalid, err)
    }

    // 3. 恢复引擎状态
    e.taskStages[taskID] = stageState.CurrentStage
    e.conversations[taskID] = stageState.Conversation

    // 4. 继续执行
    return e.ExecuteStage(ctx, taskID, stageState.CurrentStage)
}

// RestoreAll 启动时恢复所有进行中的任务
func (e *WorkflowEngine) RestoreAll(ctx context.Context) error {
    // 获取所有非终态任务
    issues, err := e.tracker.GetTasksByStage(ctx, []string{
        "clarification", "bdd_review", "architecture_review", "implementation", "verification",
    })
    if err != nil {
        return err
    }

    for _, issue := range issues {
        if err := e.RestoreFromBeads(ctx, issue.Identifier); err != nil {
            slog.Error("failed to restore task", "task_id", issue.Identifier, "error", err)
        }
    }
    return nil
}
```

**恢复场景示例：**

```
服务崩溃前：
  Task-001 在 "需求澄清中" 阶段
  已进行 3 轮对话
  用户最后回答："使用邮箱登录"

服务重启后：
  1. GetTasksByStage 获取进行中任务
  2. RestoreFromBeads 读取 Task-001
  3. 解析：current_stage = clarification
  4. 解析：conversation = [Q1/A1, Q2/A2, Q3/A3]
  5. 恢复到澄清阶段继续
```

## Architecture 验证结果

### 一致性验证 ✅

**决策兼容性:**
- Go 1.22 + Gin v1.9.1 ✅ 兼容
- slog 结构化日志与 Go 1.22 ✅ 内置支持
- Clean Architecture 与现有模块结构 ✅ 一致
- Interface + Factory 模式与 Go 隐式实现 ✅ 兼容

**模式一致性:**
- 命名规范：包名小写、接口 PascalCase、函数 camelCase ✅ 一致
- 错误码 `<module>.<type>` 格式 ✅ 全文档统一
- 阶段名 snake_case、状态 PascalCase ✅ 一致

**结构对齐:**
- Project structure 支持所有新增模块 ✅
- 新增文件位置明确 ✅
- Integration points 定义清晰 ✅

### 需求覆盖验证 ✅

**FR 覆盖:**

| FR 类别 | 架构支持 | 状态 |
|---------|----------|------|
| 初始化与配置管理 (FR1-8) | config + cmd | ✅ |
| 任务生命周期管理 (FR9-13) | workflow + tracker | ✅ |
| 需求澄清 (FR14-21) | workflow stages + agent prompt | ✅ |
| 规则生成与管理 (FR22-30) | workflow + server handlers | ✅ |
| 执行与监控 (FR31-40) | orchestrator + agent + SSE | ✅ |
| 验收与报告 (FR41-44) | workflow verification stage | ✅ |
| 异常处理与恢复 (FR45-49) | workflow engine + crash recovery | ✅ |
| 外部集成 (FR50-56) | tracker beads + agent CLI | ✅ |

**NFR 覆盖:**

| NFR | 架构支持 | 状态 |
|-----|----------|------|
| NFR1: Agent 等待 >24h | goroutine + channel，无硬超时 | ✅ |
| NFR2: Beads CLI 检测 | CheckAvailability() | ✅ |
| NFR3: 崩溃恢复 | RestoreAll + StageState serialization | ✅ |
| NFR4: 配置重启生效 | 无热加载设计 | ✅ |
| NFR5: 日志记录 | slog 结构化日志 | ✅ |

### 实现就绪验证 ✅

**决策完整性:**
- Tracker interface ✅ 定义完整
- StageDefinition 结构 ✅ 定义完整
- Workflow Engine 组件 ✅ 模块划分清晰
- 错误码 ✅ 集中管理文件已定义

**结构完整性:**
- 目录结构 ✅ 完整定义
- 新增文件位置 ✅ 明确
- Integration points ✅ 已映射

**模式完整性:**
- 命名规范 ✅ 完整
- 错误处理模式 ✅ 完整
- 日志模式 ✅ 完整
- 测试模式 ✅ 表驱动 + Mock 策略

### 缺口分析结果

**严重缺口:** 无

**重要缺口:** 已在 Advanced Elicitation 中解决
- ✅ Beads CLI Interface 已定义
- ✅ Crash Recovery serialization 已定义

**可选改进 (Phase 2):**
- OrchestratorInterface 抽象（便于测试 mock）
- PromptLoader interface 抽象
- workflow.md 配置解析逻辑

### 完整性检查清单

**✅ 需求分析**
- [x] 项目上下文已分析
- [x] 规模和复杂度已评估
- [x] 技术约束已识别
- [x] 横切关注点已映射

**✅ 架构决策**
- [x] 数据架构已定义
- [x] 状态机架构已定义
- [x] 错误处理策略已定义
- [x] Prompt 管理已定义

**✅ 实现模式**
- [x] 命名规范已建立
- [x] 接口模式已定义
- [x] 错误处理模式已文档化
- [x] 日志模式已文档化
- [x] 测试模式已文档化

**✅ 项目结构**
- [x] 目录结构完整定义
- [x] 新增文件已映射
- [x] 集成点已文档化
- [x] 需求到结构映射已完成

**✅ 接口定义**
- [x] Tracker interface 已定义
- [x] 数据结构已定义
- [x] BeadsClient 实现已定义
- [x] 错误码已定义
- [x] 崩溃恢复流程已定义

### 就绪度评估

**整体状态:** ✅ 已就绪，可进入实现阶段

**信心等级:** 高

**核心优势:**
- 清晰的职责边界（Beads 存储状态，项目 repo 存储规则）
- 完整的崩溃恢复设计（RestoreAll + StageState serialization）
- 与现有代码库一致的模式（接口 + Factory）
- 可测试的接口抽象（Tracker interface）

**后续增强方向:**
- workflow.md 配置解析（Phase 2）
- 更多 interface 抽象
- Windows 平台支持

### 实现交接

**AI Agent 实现指南:**
1. 严格遵循架构决策
2. 使用一致的实现模式
3. 遵守项目结构和边界
4. 遇到架构问题时参考此文档

**优先实现顺序:**
1. `internal/common/errors/errors.go` - 集中错误码
2. `internal/tracker/beads.go` - Beads tracker 实现
3. `internal/workflow/stages.go` - 阶段定义
4. `internal/workflow/engine.go` - Workflow Engine 核心