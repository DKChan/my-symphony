# Story 2.1: BMAD Agent 调用框架

Status: review

## Story

As a 系统架构师,
I want 系统能调用 BMAD Agent,
so that 可以利用专家 Agent 完成特定任务.

## Acceptance Criteria

1. **AC1: Agent 调用流程**
   - Given 系统需要调用 BMAD Agent
   - When 调用 `AgentCaller.Call()`
   - Then 系统构建 Agent 输入参数
   - And 系统执行 BMAD Agent CLI
   - And 系统解析 Agent 输出

2. **AC2: 超时处理**
   - Given Agent 执行超时
   - When 超过配置的超时时间
   - Then 返回 `agent.timeout` 错误

3. **AC3: 可用性检查**
   - Given 系统启动时
   - When 检查 Agent 可用性
   - Then 验证 Claude CLI 是否可用
   - And 返回明确的错误信息

**FRs covered:** NFR1, NFR3

## Tasks / Subtasks

- [x] Task 1: 定义 AgentCaller 接口和数据结构 (AC: 1, 2, 3)
  - [x] 1.1 在 `internal/harness/` 目录创建 `agent_caller.go`
  - [x] 1.2 定义 `AgentCaller` 接口 (Call, CheckAvailability)
  - [x] 1.3 定义 `AgentInput` 结构体
  - [x] 1.4 定义 `AgentOutput` 结构体
  - [x] 1.5 定义 `AgentCallerImpl` 实现结构体

- [x] Task 2: 实现 AgentCaller 核心逻辑 (AC: 1, 2)
  - [x] 2.1 实现 `NewAgentCaller()` 构造函数
  - [x] 2.2 实现 `Call()` 方法 - 构建 prompt 并调用 Claude CLI
  - [x] 2.3 实现 `CheckAvailability()` 方法
  - [x] 2.4 实现超时控制 (context + timeout)
  - [x] 2.5 解析 Claude CLI 的 stream-json 输出

- [x] Task 3: 错误处理和日志记录 (AC: 2, 3)
  - [x] 3.1 定义 Agent 相关错误码
  - [x] 3.2 实现 `agent.timeout` 错误返回
  - [x] 3.3 实现 `agent.unavailable` 错误返回
  - [x] 3.4 添加 slog 结构化日志

- [x] Task 4: 单元测试 (AC: 1, 2, 3)
  - [x] 4.1 创建 `agent_caller_test.go`
  - [x] 4.2 测试正常调用流程
  - [x] 4.3 测试超时处理
  - [x] 4.4 测试可用性检查
  - [x] 4.5 测试错误处理

## Dev Notes

### 架构要点

1. **BMAD Agent 调用方式**
   - BMAD agents 通过 Claude Code CLI 调用，使用 `/agent-name` 语法
   - 例如: `claude --print /bmad-agent-pm "任务描述"`
   - Claude CLI 的 stream-json 格式输出需要解析

2. **复用现有代码**
   - 已有 `internal/agent/runner.go` 定义了 Runner 接口
   - 已有 `internal/agent/claude.go` 实现了 Claude CLI 调用
   - 新的 AgentCaller 应该复用这些现有实现

3. **与现有架构的关系**
   - `internal/harness/agent_caller.go` 是 P-G-E 架构的 Agent 调用层
   - 它封装了对 BMAD agents 的调用逻辑
   - 下层使用 `internal/agent/claude.go` 执行实际的 CLI 调用

### 技术约束

1. **超时控制**
   - 使用 `context.WithTimeout` 实现超时
   - 超时时间从配置读取 (可配置无超时)
   - NFR1 要求支持 >24h 的长时间执行

2. **错误码格式**
   - 格式: `<module>.<type>: <description>`
   - 定义在 `internal/common/errors/errors.go`
   - 已有: `ErrAgentTimeout`, `ErrAgentUnavailable`

3. **日志规范**
   - 使用 `log/slog` 结构化日志
   - 字段命名使用 `snake_case`

### 参考代码

**现有 Runner 接口** (`internal/agent/runner.go`):
```go
type Runner interface {
    RunAttempt(ctx context.Context, issue *domain.Issue, workspacePath string,
        attempt *int, promptTemplate string, callback EventCallback) (*RunAttemptResult, error)
}
```

**Claude CLI 调用** (`internal/agent/claude.go`):
- 使用 `claude --print --output-format=stream-json`
- 解析 JSON 事件流
- 处理 session 和 turn

### Project Structure Notes

**新增文件:**
- `internal/harness/agent_caller.go` - BMAD Agent 调用抽象
- `internal/harness/agent_caller_test.go` - 单元测试

**修改文件:**
- `internal/common/types.go` - 添加 TaskStageToKanbanColumn 映射更新
- `internal/common/types_test.go` - 更新相关测试
- `internal/server/presenter/presenter.go` - 修复看板列映射 bug
- `internal/server/presenter/presenter_test.go` - 更新相关测试
- `internal/server/components/dashboard.go` - 修复阶段显示名称获取

### References

- [Source: _bmad-output/planning-artifacts/epics-v2.md#L153-L175] - Story 定义
- [Source: _bmad-output/planning-artifacts/architecture-v2.md#L495-L524] - AgentCaller 接口定义
- [Source: _bmad-output/planning-artifacts/architecture-v2.md#L617-L636] - 错误码定义
- [Source: internal/agent/runner.go] - 现有 Runner 接口
- [Source: internal/agent/claude.go] - Claude CLI 实现

### 测试标准

- 目标覆盖率: 60%
- 使用表驱动测试
- Mock 策略: 手写 Mock（MVP 阶段）

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (claude-opus-4-6)

### Debug Log References

- 所有测试通过，无 debug 日志需要参考

### Completion Notes List

1. ✅ 实现了 `AgentCaller` 接口和 `AgentCallerImpl` 结构体
2. ✅ 实现了 `AgentInput` 和 `AgentOutput` 数据结构
3. ✅ 实现了 `Call()` 方法，支持 BMAD Agent 调用
4. ✅ 实现了 `CheckAvailability()` 方法，检查 CLI 可用性
5. ✅ 实现了超时控制，使用 `context.WithTimeout`
6. ✅ 使用现有的 `internal/common/errors` 错误定义
7. ✅ 添加了 slog 结构化日志
8. ✅ 创建了完整的单元测试

**附带修复:**
- 修复了 `internal/server/presenter/presenter.go` 中的看板列映射 bug
- 修复了 `internal/server/components/dashboard.go` 中的阶段显示名称获取问题
- 更新了 `TaskStageToKanbanColumn` 映射，适配 P-G-E 架构的 5 列看板

### File List

**新增文件:**
- `internal/harness/agent_caller.go`
- `internal/harness/agent_caller_test.go`

**修改文件:**
- `internal/common/types.go`
- `internal/common/types_test.go`
- `internal/server/presenter/presenter.go`
- `internal/server/presenter/presenter_test.go`
- `internal/server/components/dashboard.go`

### Change Log

- 2026-04-06: 完成 Story 2.1 实现，所有 AC 满足，所有测试通过