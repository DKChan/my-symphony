# Story 4.2: BDD 验收执行

Status: review

## Story

As a 系统用户,
I want 系统执行 BDD 验收测试,
so that 可以验证需求是否满足.

## Acceptance Criteria

1. **AC1: BDD 验收执行**
   - Given BDD 测试脚本可用
   - When Evaluator E1 开始执行
   - Then 系统运行 BDD 测试
   - And 记录测试结果
   - And 报告失败用例

**FRs covered:** FR30

## Tasks / Subtasks

- [x] Task 1: 验证现有实现 (AC: 1)
  - [x] 1.1 检查 `internal/harness/evaluator.go` 中 `executeBDDTest` 方法
  - [x] 1.2 确认调用 bmad-agent-qa
  - [x] 1.3 确认结果解析正确

- [x] Task 2: 单元测试 (AC: 1)
  - [x] 2.1 测试正常执行
  - [x] 2.2 测试失败场景

## Dev Notes

### 实现说明

本 Story 的核心功能已在 Story 4.1 中实现：

**已实现部分** (`internal/harness/evaluator.go`):
- `executeBDDTest(ctx, taskID, generatorOutput)` 方法
- 调用 `bmad-agent-qa` Agent
- 使用 `parseTestResult` 解析结果

### Project Structure Notes

**现有文件 (复用)**:
- `internal/harness/evaluator.go` - 已有 executeBDDTest 方法
- `internal/harness/evaluator_test.go` - 已有测试覆盖

**无新增文件**

### References

- [Source: _bmad-output/planning-artifacts/epics-v2.md#L448-L464] - Story 定义

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (claude-opus-4-6)

### Completion Notes List

1. ✅ 验证了 executeBDDTest 实现
2. ✅ 确认调用正确的 Agent
3. ✅ 测试覆盖已包含在 TestEvaluatorExecute 中

### Change Log

- 2026-04-06: Story 创建，功能已在 Story 4.1 实现