# Story 1.1: CLI init 交互式初始化

Status: done

## Story

As a 开发者,
I want 通过 CLI 交互式初始化项目配置,
so that 我可以快速配置 Symphony 项目.

## Acceptance Criteria

```gherkin
Given 用户在项目根目录执行 `symphony init`
When CLI 启动交互式问答
Then 用户可以配置项目名称
And 用户可以配置 Tracker 类型 (Beads)
And 用户可以配置 BMAD Agent 启用状态
And 用户可以配置最大迭代次数
And 系统生成 `.sym/config.yaml` 配置文件
```

## Tasks / Subtasks

- [x] Task 1: 更新 InitOptions 结构体 (AC: FR1-3)
  - [x] 1.1: 添加 ProjectName 字段
  - [x] 1.2: 添加 BMADEnabled 字段
  - [x] 1.3: 添加 MaxIterations 字段

- [x] Task 2: 更新交互式问答流程 (AC: #1)
  - [x] 2.1: 添加项目名称配置提示
  - [x] 2.2: 添加 BMAD Agent 启用/禁用选择
  - [x] 2.3: 添加最大迭代次数配置 (默认 5)
  - [x] 2.4: 保持现有 Tracker 类型选择 (默认 beads)

- [x] Task 3: 更新配置生成逻辑 (AC: #1)
  - [x] 3.1: 生成符合 v2.0 架构的 config.yaml
  - [x] 3.2: 包含 harness.max_iterations 配置项
  - [x] 3.3: 包含 harness.bmad.enabled 配置项
  - [x] 3.4: 包含 harness.bmad.agents 配置结构

- [x] Task 4: 更新 Config 结构体 (AC: #1)
  - [x] 4.1: 添加 HarnessConfig 类型定义
  - [x] 4.2: 添加 BMADConfig 类型定义
  - [x] 4.3: 更新 ParseConfig 解析逻辑
  - [x] 4.4: 更新 DefaultConfig 默认值

- [x] Task 5: 编写单元测试 (AC: #1)
  - [x] 5.1: 测试 InitOptions 新字段
  - [x] 5.2: 测试交互式问答流程
  - [x] 5.3: 测试配置生成逻辑
  - [x] 5.4: 测试 Config 结构体解析

## Dev Notes

### 架构约束

**P-G-E 架构要求：** 新配置格式必须包含 harness 部分，支持 Planner-Generator-Evaluator 三层架构。

[Source: architecture-v2.md#Configuration Schema]

### 配置格式参考

v2.0 新配置格式：

```yaml
# .sym/config.yaml
project_name: my-project
user_name: DK

tracker:
  type: beads

harness:
  max_iterations: 5        # 最大迭代次数，默认 5
  bmad:
    enabled: true
    agents:
      planner: [bmad-agent-pm, bmad-agent-qa, bmad-agent-architect]
      generator: [bmad-agent-qa, bmad-agent-dev]
      evaluator: [bmad-code-review, bmad-editorial-review-prose]

server:
  port: 8080
```

[Source: architecture-v2.md#Configuration Schema]

### 现有代码分析

**现有 InitCommand (internal/cli/init.go:38-105):**
- 已实现 Tracker 类型选择 (github/mock/beads)
- 已实现 Agent 类型选择 (codex/claude/opencode)
- 已实现目录结构创建
- 已实现配置文件生成

**需要修改的部分:**
- `InitOptions` 结构体: 添加 ProjectName, BMADEnabled, MaxIterations 字段
- `Run()` 方法: 添加新的交互式问答
- `generateConfig()` 方法: 生成新格式配置
- `buildConfigMap()` 方法: 构建新配置结构

### 配置结构体扩展

**需要添加到 internal/config/config.go:**

```go
// HarnessConfig Harness 配置
type HarnessConfig struct {
    MaxIterations int        `json:"max_iterations"`
    BMAD          BMADConfig `json:"bmad"`
}

// BMADConfig BMAD Agent 配置
type BMADConfig struct {
    Enabled bool     `json:"enabled"`
    Agents  []string `json:"agents,omitempty"`
}
```

[Source: architecture-v2.md#Interface Definitions]

### 错误码规范

使用统一错误码格式：
- `init.dir_access: 无法访问目录`
- `init.dir_create: 无法创建目录`
- `init.config_serialize: 无法序列化配置`
- `init.config_write: 无法写入配置文件`

[Source: architecture-v2.md#Error Handling]

### 测试策略

- 单元测试覆盖率: >= 70%
- 使用 table-driven tests 测试不同配置组合
- Mock bufio.Scanner 进行交互式测试

[Source: architecture-v2.md#Testing Strategy]

### Project Structure Notes

**修改文件:**
- `internal/cli/init.go` - CLI 初始化逻辑
- `internal/cli/init_test.go` - 单元测试
- `internal/config/config.go` - 配置结构体
- `internal/config/config_test.go` - 配置测试

**保持不变:**
- `cmd/symphony/main.go` - CLI 入口
- 其他 workflow/prompt 模板生成

[Source: architecture-v2.md#Project Structure]

### References

- [PRD FR1-3: 初始化与配置管理] prd-v2.md#Functional Requirements
- [Architecture Config Schema] architecture-v2.md#Configuration Schema
- [Architecture Error Codes] architecture-v2.md#Error Handling
- [Architecture Testing Strategy] architecture-v2.md#Testing Strategy
- [Epics Story 1.1] epics-v2.md#Story 1.1: CLI init 交互式初始化

## Dev Agent Record

### Agent Model Used

glm-5 (并行子 agent 执行)

### Debug Log References

无

### Completion Notes List

**实现完成** (2026-04-05):

1. **InitOptions 扩展**: 添加 ProjectName、BMADEnabled、MaxIterations 字段
2. **Config 结构体扩展**: 添加 HarnessConfig、BMADConfig 类型，更新 DefaultConfig 和 ParseConfig
3. **交互式问答**: 新增项目名称、BMAD启用、迭代次数配置提示
4. **配置生成**: generateConfig() 和 buildConfigMap() 支持 harness 配置
5. **测试覆盖率**: cli 84.2%, config 87.6% (达标 >=70%)

**关键技术决策**:
- Tracker 默认值改为 "beads" (符合 v2.0 架构)
- BMAD 默认 agents 列表包含 5 个核心 agent
- ProjectName 存储在 WorkspaceConfig 结构体中

### File List

- internal/cli/init.go (修改)
- internal/cli/init_test.go (修改)
- internal/config/config.go (修改)
- internal/config/config_test.go (修改)

### Change Log

- 2026-04-05: Story 1.1 实现完成，所有 AC 满足，测试通过

## Review Findings

**Code Review:** 2026-04-05

### Patch (已修复)

- [x] [Review][Patch] BMAD Agents 改为分组结构 — 已修改 config.go/init.go 按 planner/generator/evaluator 分组
- [x] [Review][Patch] MaxIterations 非交互模式默认为 0 — 已修复：默认使用 5
- [x] [Review][Patch] BMADEnabled 默认值问题 — 已修复：改为 *bool 类型，nil 时使用默认值 true
- [x] [Review][Patch] workspace.project_name 未从配置解析 — 已添加 ParseConfig 解析
- [x] [Review][Patch] MaxIterations 缺少配置验证 — 已添加到 ValidateDispatchConfig/ValidateSymphonyConfig
- [x] [Review][Patch] 缺少 bmad-editorial-review-prose — 已添加到 evaluator 默认列表
- [x] [Review][Patch] BMAD agents 条件逻辑问题 — 已修复：移除条件判断，使用 DefaultConfig 默认值

### Deferred

- [x] [Review][Defer] Linear tracker 移除 — 用户确认为 hotfix 内容
- [x] [Review][Defer] GitHub 空值验证缺失 — 预先存在的问题，不在本 Story 范围