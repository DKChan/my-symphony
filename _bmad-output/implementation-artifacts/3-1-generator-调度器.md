# Story 3.1: Generator 调度器

Status: review

## Story

As a 系统架构师,
I want Generator 能调度多个子任务,
so that 可以并行执行测试编码.

## Acceptance Criteria

1. **AC1: Phase 1 并行执行**
   - Given Planner 产出完成
   - When Generator 开始执行
   - Then Phase 1 (测试编码) 并行启动
   - And G1, G2, G3 同时开始

2. **AC2: Phase 2 顺序执行**
   - Given Phase 1 所有任务完成
   - When 最后一个测试脚本完成
   - Then Phase 2 (代码实现) 开始
   - And G4 顺序执行

**FRs covered:** FR20, FR25

## Tasks / Subtasks

- [x] Task 1: 定义 Generator 接口和数据结构 (AC: 1, 2)
  - [x] 1.1 在 `internal/harness/` 创建 `generator.go`
  - [x] 1.2 定义 `Generator` 接口
  - [x] 1.3 定义 `Phase1Output` 结构体
  - [x] 1.4 定义 `Phase2Output` 结构体
  - [x] 1.5 定义 `GeneratorImpl` 实现结构体

- [x] Task 2: 实现 Phase 1 并行执行 (AC: 1)
  - [x] 2.1 实现 `ExecutePhase1()` 方法
  - [x] 2.2 使用 goroutine 并行执行 G1, G2, G3
  - [x] 2.3 使用 sync.WaitGroup 等待完成
  - [x] 2.4 处理并行任务失败

- [x] Task 3: 实现 Phase 2 顺序执行 (AC: 2)
  - [x] 3.1 实现 `ExecutePhase2()` 方法
  - [x] 3.2 顺序执行 G4
  - [x] 3.3 支持失败报告输入 (迭代场景)

- [x] Task 4: 单元测试 (AC: 1, 2)
  - [x] 4.1 创建 `generator_test.go`
  - [x] 4.2 测试 Phase 1 并行执行
  - [x] 4.3 测试 Phase 2 顺序执行
  - [x] 4.4 测试失败处理

## Dev Notes

### 架构要点

1. **Generator 执行流程**
   ```
   Phase 1: 测试编码 (并行)
   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
   │     G1      │  │     G2      │  │     G3      │
   │  BDD测试    │  │  集成测试   │  │  单元测试   │
   └─────────────┘  └─────────────┘  └─────────────┘
         │                │                │
         └────────────────┼────────────────┘
                          │
                          ▼ (等待全部完成)

   Phase 2: 代码实现 (顺序)
                   ┌─────────────┐
                   │     G4      │
                   │  代码实现   │
                   └─────────────┘
   ```

2. **并行执行模式**
   ```go
   // Phase 1: 并行执行
   var wg sync.WaitGroup
   var results [3]*AgentOutput
   var errors [3]error
   
   for i, task := range []string{"G1", "G2", "G3"} {
       wg.Add(1)
       go func(idx int, taskID string) {
           defer wg.Done()
           results[idx], errors[idx] = executeTask(ctx, taskID)
       }(i, task)
   }
   wg.Wait()
   ```

3. **Agent 配置**
   - G1: bmad-agent-qa (BDD测试脚本)
   - G2: bmad-agent-qa (集成测试)
   - G3: bmad-agent-dev (单元测试)
   - G4: bmad-agent-dev (代码实现)

### Project Structure Notes

**新增文件:**
- `internal/harness/generator.go` - Generator 执行器
- `internal/harness/generator_test.go` - 单元测试

### References

- [Source: _bmad-output/planning-artifacts/epics-v2.md#L291-L312] - Story 定义
- [Source: _bmad-output/planning-artifacts/architecture-v2.md#L136-L165] - 并行执行模型
- [Source: _bmad-output/planning-artifacts/architecture-v2.md#L428-L456] - Generator Interface

### 测试标准

- 目标覆盖率: 70%
- Mock 策略: MockAgentCaller

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (claude-opus-4-6)

### Debug Log References

- 所有测试通过，无 debug 日志需要参考

### Completion Notes List

1. ✅ 实现了 Generator 接口和 GeneratorImpl 结构体
2. ✅ 实现了 Phase1Output 和 Phase2Output 数据结构
3. ✅ 实现了 ExecutePhase1() 方法，使用 goroutine 并行执行 G1, G2, G3
4. ✅ 实现了 ExecutePhase2() 方法，顺序执行 G4
5. ✅ 支持失败报告输入用于迭代修复
6. ✅ 添加了完整的单元测试，包括并行执行验证
7. ✅ 测试了迭代计数功能

**说明:** Generator 核心调度框架已完成，后续 Stories 3.2-3.5 将细化具体生成任务的实现。

### File List

**新增文件:**
- `internal/harness/generator.go`
- `internal/harness/generator_test.go`

### Change Log

- 2026-04-06: Story 创建
- 2026-04-06: 完成 Generator 实现，所有测试通过