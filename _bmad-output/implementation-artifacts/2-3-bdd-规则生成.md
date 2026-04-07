# Story 2.3: BDD 规则生成

Status: review

## Story

As a 系统用户,
I want 系统调用 BMAD QA Agent 生成 BDD 规则,
so that 我可以快速获得验收标准.

## Acceptance Criteria

1. **AC1: BDD 规则生成调用**
   - Given 需求澄清完成
   - When Planner 进入 BDD 规则生成阶段
   - Then 系统调用 BMAD QA Agent
   - And QA Agent 返回 Gherkin 格式的 BDD 规则
   - And 规则保存到任务上下文

**FRs covered:** FR12

## Tasks / Subtasks

- [x] Task 1: 验证现有实现 (AC: 1)
  - [x] 1.1 检查 `internal/harness/planner.go` 中 `GenerateBDDRules` 方法
  - [x] 1.2 确认方法调用正确的 BMAD Agent (bmad-agent-qa)
  - [x] 1.3 确认返回结果正确存储到 PlannerOutput

- [x] Task 2: 完善 BDD 规则生成测试 (AC: 1)
  - [x] 2.1 在 `planner_test.go` 添加 `GenerateBDDRules` 测试
  - [x] 2.2 测试正常生成流程
  - [x] 2.3 测试 Agent 调用失败场景
  - [x] 2.4 测试产出更新正确性

- [x] Task 3: 集成测试 (AC: 1)
  - [x] 3.1 测试 Planner.Execute 流程中 P2 阶段
  - [x] 3.2 测试 BDD 规则生成后的产出不可变性

## Dev Notes

### 实现说明

本 Story 的核心功能已在 Story 2.2 中部分实现：

**已实现部分** (`internal/harness/planner.go`):
- `GenerateBDDRules(ctx, taskID, requirements)` 方法已存在
- 调用 `bmad-agent-qa` Agent
- 使用 AgentCaller.Call() 执行
- 结果存储到 PlannerOutput.BDDRules 字段

**需要补充**:
- 单元测试覆盖
- 集成到完整 Planner 执行流程

### 架构要点

1. **BDD 规则生成阶段 (P2)**
   - Agent: bmad-agent-qa
   - 输入: 需求澄清后的需求描述
   - 输出: Gherkin 格式的 BDD 规则
   - 存储: PlannerOutput.BDDRules

2. **产出不可变**
   - PlannerOutput 创建后标记 Immutable=true
   - BDDRules 作为产出的一部分，一旦生成不可修改

3. **Agent 调用流程**
   ```go
   input := &AgentInput{
       AgentName: "bmad-agent-qa",
       Task:      "根据以下需求生成 Gherkin 格式的 BDD 规则:\n\n" + requirements,
       Context: map[string]string{
           "phase": "bdd_generation",
       },
   }
   output, err := p.agentCaller.Call(ctx, input)
   ```

### Project Structure Notes

**现有文件 (复用)**:
- `internal/harness/planner.go` - 已有 GenerateBDDRules 方法
- `internal/harness/planner_test.go` - 补充测试

**无新增文件**

### References

- [Source: _bmad-output/planning-artifacts/epics-v2.md#L202-L217] - Story 定义
- [Source: _bmad-output/planning-artifacts/architecture-v2.md#L403-L426] - Planner 接口定义
- [Source: internal/harness/planner.go#L126-L152] - GenerateBDDRules 实现

### Previous Story Intelligence

**来自 Story 2.1 (BMAD Agent 调用框架)**:
- AgentCaller 接口已实现
- AgentInput/AgentOutput 结构体已定义
- 超时控制和错误处理已完善

**来自 Story 2.2 (需求澄清)**:
- Planner 接口已定义
- PlannerOutput 结构体已定义 (含 BDDRules 字段)
- AgentCaller 集成到 Planner
- Execute 方法已实现基础框架

### 测试标准

- 目标覆盖率: 70%
- 使用表驱动测试
- Mock 策略: 手写 Mock (MVP 阶段)

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (claude-opus-4-6)

### Debug Log References

- 所有测试通过，无 debug 日志需要参考

### Completion Notes List

1. ✅ 验证了现有 GenerateBDDRules 实现
2. ✅ 添加了 MockAgentCaller 用于单元测试
3. ✅ 更新 Planner 接口接受 AgentCaller interface 类型
4. ✅ 添加了 GenerateBDDRules 测试覆盖
5. ✅ 测试 Agent 调用失败场景
6. ✅ 测试产出不可变性保持

**说明:** 核心功能已在 Story 2.2 预实现，本 Story 主要补充了测试覆盖。

### File List

**修改文件:**
- `internal/harness/planner.go` - 更新接口接受 AgentCaller interface
- `internal/harness/planner_test.go` - 添加 MockAgentCaller 和 BDD 测试

### Change Log

- 2026-04-06: Story 创建，核心功能已在 Story 2.2 中预实现
- 2026-04-06: 完成测试覆盖，所有测试通过