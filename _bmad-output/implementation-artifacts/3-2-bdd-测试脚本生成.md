# Story 3.2: BDD 测试脚本生成

Status: review

## Story

As a 系统用户,
I want 系统调用 BMAD QA Agent 生成 BDD 测试脚本,
so that BDD 规则可以转化为可执行测试.

## Acceptance Criteria

1. **AC1: BDD 测试脚本生成**
   - Given Planner BDD 规则可用
   - When Generator G1 开始执行
   - Then 系统调用 BMAD QA Agent
   - And QA Agent 将 Gherkin 规则转为可执行测试代码
   - And 测试脚本保存到工作目录

**FRs covered:** FR21

## Tasks / Subtasks

- [x] Task 1: 验证现有实现 (AC: 1)
  - [x] 1.1 检查 `internal/harness/generator.go` 中 `generateBDDTestScript` 方法
  - [x] 1.2 确认方法调用 bmad-agent-qa
  - [x] 1.3 确认结果存储到 Phase1Output.BDDTestScript

- [x] Task 2: 单元测试 (AC: 1)
  - [x] 2.1 测试正常生成流程
  - [x] 2.2 测试错误处理

## Dev Notes

### 实现说明

本 Story 的核心功能已在 Story 3.1 中实现：

**已实现部分** (`internal/harness/generator.go`):
- `generateBDDTestScript(ctx, taskID, plannerOutput)` 方法
- 调用 `bmad-agent-qa` Agent
- 使用 PlannerOutput.BDDRules 作为输入
- 结果存储到 Phase1Output.BDDTestScript

### Project Structure Notes

**现有文件 (复用)**:
- `internal/harness/generator.go` - 已有 generateBDDTestScript 方法
- `internal/harness/generator_test.go` - 已有测试覆盖

**无新增文件**

### References

- [Source: _bmad-output/planning-artifacts/epics-v2.md#L314-L332] - Story 定义

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (claude-opus-4-6)

### Completion Notes List

1. ✅ 验证了 generateBDDTestScript 实现
2. ✅ 确认调用正确的 Agent (bmad-agent-qa)
3. ✅ 测试覆盖已包含在 TestExecutePhase1 中

### Change Log

- 2026-04-06: Story 创建，功能已在 Story 3.1 实现