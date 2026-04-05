---
stepsCompleted: [1, 2, 3, 4, 5, 6, 7, 8]
inputDocuments:
  - _bmad-output/planning-artifacts/prd-v2.md
workflowType: 'architecture'
project_name: 'my-symphony'
user_name: 'DK'
date: '2026-04-05'
status: 'complete'
---

# Architecture Decision Document (v2.0)

_基于 P-G-E 架构的 Symphony Harness 设计_

## Document Setup

**Input Documents:**
- PRD v2.0: `_bmad-output/planning-artifacts/prd-v2.md`
- Sprint Change Proposal: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-04-05.md`

**参考实践:**
- Anthropic: Harness design for long-running apps
- OpenAI: Engineering in an Agent-First World

---

## Project Context Analysis

### Requirements Overview

**Functional Requirements: 62 条**

| 能力领域 | FR 数量 | 架构影响 |
|----------|---------|----------|
| 初始化与配置管理 | 8 | CLI 入口、配置系统 |
| Planner 模块 | 7 | BMAD Agent 调用、需求理解 |
| Generator 模块 | 7 | 并行执行、代码生成 |
| Evaluator 模块 | 6 | 测试执行、代码审计 |
| 迭代机制 | 5 | 失败处理、迭代计数 |
| 任务管理 | 6 | Beads 子任务结构 |
| 外部集成 | 3 | Beads tracker |

**Non-Functional Requirements: 9 条**

| NFR | 架构约束 |
|-----|----------|
| NFR1: BMAD Agent 等待 >24h | 长时间运行进程管理 |
| NFR2-3: 可用性检测 | 启动时健康检查 |
| NFR4: 崩溃后状态恢复 | 状态持久化到 Beads |
| NFR6-7: 日志记录 | slog 结构化日志 |
| NFR8: 并行执行 | goroutine 并发控制 |

### Scale & Complexity

**Primary domain:** CLI Tool + Daemon Service + Local Web UI

**Complexity level:** Medium

**核心架构组件:** 6 个

### Core Architectural Challenges

1. **BMAD Agent 集成层**
   - Agent 调用抽象
   - 输入/输出处理
   - 长时间运行管理

2. **P-G-E 编排引擎**
   - Planner → Generator → Evaluator 流程控制
   - 并行任务调度
   - 迭代循环控制

3. **子任务状态管理**
   - 三类子任务 (Planner/Generator/Evaluator)
   - 迭代时新建子任务
   - Beads 集成

---

## Core Architectural Decisions

### Decision 1: 三层架构 (P-G-E)

**决策:** 采用 Planner-Generator-Evaluator 三层架构

**理由:**
- Anthropic/OpenAI 实践验证有效
- 职责分离清晰
- 支持 GAN 式迭代优化

**架构图:**

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Harness Orchestrator                         │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                         Planner                                │  │
│  │                                                                │  │
│  │  职责: 需求理解与规划                                          │  │
│  │  Agent: PM, QA, Architect                                      │  │
│  │  产出: BDD规则, 领域模型, 架构设计, 接口定义                   │  │
│  │  约束: 产出不可变                                              │  │
│  │                                                                │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                │                                     │
│                                ▼                                     │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                        Generator                               │  │
│  │                                                                │  │
│  │  职责: 测试编码与代码实现                                      │  │
│  │  Agent: QA, Dev                                                │  │
│  │  产出: BDD测试脚本, 集成测试, 单元测试, 代码实现               │  │
│  │  模式: Phase 1 并行 → Phase 2 顺序                             │  │
│  │                                                                │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                │                                     │
│                                ▼                                     │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                        Evaluator                               │  │
│  │                                                                │  │
│  │  职责: 质量验证                                                │  │
│  │  Agent: QA, Code Review, Editorial                             │  │
│  │  产出: 测试结果, 审计报告, 风格报告                            │  │
│  │  约束: 只评估代码，不判断失败类型                              │  │
│  │                                                                │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

### Decision 2: 并行执行模型

**决策:** Generator Phase 1 并行，Phase 2 顺序

**理由:**
- 测试脚本之间无依赖，可并行
- 代码实现依赖测试定义，需顺序

**执行模型:**

```
Generator 执行流程:

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

**实现方式:**
```go
// Phase 1: 并行执行
var wg sync.WaitGroup
for _, task := range []string{"G1", "G2", "G3"} {
    wg.Add(1)
    go func(taskID string) {
        defer wg.Done()
        executeTask(ctx, taskID)
    }(task)
}
wg.Wait()

// Phase 2: 顺序执行
executeTask(ctx, "G4")
```

---

### Decision 3: 迭代机制

**决策:** 最大迭代 5 次，只修复代码

**理由:**
- Planner 产出作为契约，保持稳定
- 避免复杂的失败类型判断
- 规划问题通过新需求修复

**迭代流程:**

```
┌─────────────┐
│  Evaluator  │
└─────────────┘
       │
       ▼
  有失败项？
       │
    ┌──┴──┐
   否     是
    │      │
    ▼      ▼
  完成   迭代次数+1
           │
           ▼
      迭代 > 5 ?
         │
      ┌──┴──┐
     否     是
      │      │
      ▼      ▼
   继续   转人工
   修复   处理
```

**数据结构:**

```go
type IterationState struct {
    Count       int       `json:"count"`
    MaxCount    int       `json:"max_count"`
    FailedItems []string  `json:"failed_items"`
    Report      string    `json:"report"`  // 失败报告 (对话上下文)
}
```

---

### Decision 4: Beads 子任务结构

**决策:** 三类子任务，迭代时新建

**结构设计:**

```
父任务: SYM-001
│
├── [Planner 类]
│   ├── SYM-001-P1: 需求澄清
│   ├── SYM-001-P2: BDD规则
│   ├── SYM-001-P3: 领域建模
│   ├── SYM-001-P4: 架构设计
│   └── SYM-001-P5: 接口设计
│
├── [Generator 类]
│   ├── SYM-001-G1: BDD测试脚本
│   ├── SYM-001-G2: 集成测试
│   ├── SYM-001-G3: 单元测试
│   ├── SYM-001-G4: 代码实现 (v1)
│   ├── SYM-001-G5: 代码实现 (v2)  [迭代2]
│   └── ...
│
└── [Evaluator 类]
    ├── SYM-001-E1: 评估 (v1)
    ├── SYM-001-E2: 评估 (v2)  [迭代2]
    └── ...
```

**依赖关系:**

```
P1 → P2 → P3 → P4 → P5
                   │
                   ▼
              G1 ─┼─ G2 ─┼─ G3
                   │      │
                   └──────┘
                       │
                       ▼
                      G4
                       │
                       ▼
                      E1
                       │
                  有失败？ → 创建 G5, E2
```

---

## Project Structure & Boundaries

### Complete Project Directory Structure

```
symphony/
├── cmd/
│   └── symphony/
│       └── main.go              # CLI 入口
├── internal/
│   ├── harness/                 # [新增] P-G-E 编排引擎
│   │   ├── orchestrator.go      # 主编排器
│   │   ├── orchestrator_test.go
│   │   ├── planner.go           # Planner 执行器
│   │   ├── planner_test.go
│   │   ├── generator.go         # Generator 执行器
│   │   ├── generator_test.go
│   │   ├── evaluator.go         # Evaluator 执行器
│   │   ├── evaluator_test.go
│   │   ├── iteration.go         # 迭代管理
│   │   └── iteration_test.go
│   ├── agent/                   # BMAD Agent 调用层
│   │   ├── caller.go            # Agent 调用抽象
│   │   ├── caller_test.go
│   │   ├── pm.go                # PM Agent 调用
│   │   ├── architect.go         # Architect Agent 调用
│   │   ├── qa.go                # QA Agent 调用
│   │   ├── dev.go               # Dev Agent 调用
│   │   └── review.go            # Review Agent 调用
│   ├── common/                  # 公共类型和错误
│   │   ├── types.go
│   │   └── errors/
│   │       └── errors.go
│   ├── config/                  # 配置管理
│   │   ├── config.go
│   │   └── config_test.go
│   ├── domain/                  # 领域实体
│   │   ├── entities.go
│   │   └── entities_test.go
│   ├── router/                  # 路由
│   │   └── router.go
│   ├── server/                  # Web 服务器
│   │   ├── server.go
│   │   ├── handlers/
│   │   │   ├── api_handler.go
│   │   │   ├── dashboard_handler.go
│   │   │   └── sse_handler.go
│   │   └── static/
│   ├── tracker/                 # Tracker 集成
│   │   ├── tracker.go
│   │   └── beads.go
│   └── workspace/               # 工作区管理
│       └── manager.go
├── docs/
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

### 新增文件汇总

| 文件 | 说明 |
|------|------|
| `internal/harness/orchestrator.go` | P-G-E 主编排器 |
| `internal/harness/planner.go` | Planner 执行器 |
| `internal/harness/generator.go` | Generator 执行器 (含并行控制) |
| `internal/harness/evaluator.go` | Evaluator 执行器 |
| `internal/harness/iteration.go` | 迭代计数与管理 |
| `internal/agent/caller.go` | BMAD Agent 调用抽象 |

### 废弃文件

| 文件 | 原因 |
|------|------|
| `internal/workflow/engine.go` | 替换为 harness/orchestrator.go |
| `internal/workflow/stages.go` | 替换为 harness/planner.go 等 |

---

## Interface Definitions

### Harness Orchestrator Interface

```go
// internal/harness/orchestrator.go

// Orchestrator P-G-E 编排器接口
type Orchestrator interface {
    // Execute 执行完整的 P-G-E 流程
    Execute(ctx context.Context, taskID string) error
    
    // GetStatus 获取当前执行状态
    GetStatus(taskID string) *ExecutionStatus
    
    // Cancel 取消执行
    Cancel(taskID string) error
}

// ExecutionStatus 执行状态
type ExecutionStatus struct {
    TaskID      string        `json:"task_id"`
    Phase       string        `json:"phase"`       // "planner", "generator", "evaluator"
    Status      string        `json:"status"`      // "running", "completed", "failed"
    Iteration   int           `json:"iteration"`
    SubTasks    []SubTaskInfo `json:"sub_tasks"`
    StartTime   time.Time     `json:"start_time"`
    EndTime     *time.Time    `json:"end_time,omitempty"`
}

// SubTaskInfo 子任务信息
type SubTaskInfo struct {
    ID       string `json:"id"`
    Category string `json:"category"` // "planner", "generator", "evaluator"
    Name     string `json:"name"`
    Status   string `json:"status"`
}
```

### Planner Interface

```go
// internal/harness/planner.go

// Planner 规划器接口
type Planner interface {
    // Execute 执行规划流程
    Execute(ctx context.Context, taskID string) (*PlannerOutput, error)
}

// PlannerOutput 规划器产出
type PlannerOutput struct {
    TaskID         string          `json:"task_id"`
    BDDRules       string          `json:"bdd_rules"`       // Gherkin 格式
    DomainModel    string          `json:"domain_model"`    // 领域模型描述
    Architecture   string          `json:"architecture"`    // 架构设计文档
    APIInterfaces  string          `json:"api_interfaces"`  // 接口定义
    CreatedAt      time.Time       `json:"created_at"`
    // 产出不可变标记
    Immutable      bool            `json:"immutable"`       // 始终为 true
}
```

### Generator Interface

```go
// internal/harness/generator.go

// Generator 生成器接口
type Generator interface {
    // ExecutePhase1 执行 Phase 1 (测试编码，并行)
    ExecutePhase1(ctx context.Context, taskID string, plannerOutput *PlannerOutput) (*Phase1Output, error)
    
    // ExecutePhase2 执行 Phase 2 (代码实现，顺序)
    ExecutePhase2(ctx context.Context, taskID string, phase1Output *Phase1Output, failureReport string) (*Phase2Output, error)
}

// Phase1Output Phase 1 产出
type Phase1Output struct {
    TaskID         string `json:"task_id"`
    BDDTestScript  string `json:"bdd_test_script"`  // 可执行测试代码
    IntegrationTest string `json:"integration_test"` // 集成测试代码
    UnitTest       string `json:"unit_test"`        // 单元测试代码
}

// Phase2Output Phase 2 产出
type Phase2Output struct {
    TaskID       string `json:"task_id"`
    CodePath     string `json:"code_path"`     // 实现代码路径
    Summary      string `json:"summary"`       // 实现摘要
}
```

### Evaluator Interface

```go
// internal/harness/evaluator.go

// Evaluator 评估器接口
type Evaluator interface {
    // Execute 执行评估
    Execute(ctx context.Context, taskID string, generatorOutput *Phase2Output) (*EvaluatorOutput, error)
}

// EvaluatorOutput 评估器产出
type EvaluatorOutput struct {
    TaskID       string        `json:"task_id"`
    Passed       bool          `json:"passed"`
    BDDResult    TestResult    `json:"bdd_result"`
    TDDResult    TestResult    `json:"tdd_result"`
    CodeReview   ReviewResult  `json:"code_review"`
    StyleReview  ReviewResult  `json:"style_review"`
    FailureReport string       `json:"failure_report,omitempty"` // 失败报告 (对话上下文)
}

// TestResult 测试结果
type TestResult struct {
    Passed      bool     `json:"passed"`
    Total       int      `json:"total"`
    PassedCount int      `json:"passed_count"`
    FailedCases []string `json:"failed_cases,omitempty"`
}

// ReviewResult 审查结果
type ReviewResult struct {
    Passed  bool     `json:"passed"`
    Issues  []string `json:"issues,omitempty"`
}
```

### BMAD Agent Caller Interface

```go
// internal/agent/caller.go

// AgentCaller BMAD Agent 调用接口
type AgentCaller interface {
    // Call 调用 Agent
    Call(ctx context.Context, input *AgentInput) (*AgentOutput, error)
    
    // CheckAvailability 检查 Agent 可用性
    CheckAvailability() error
}

// AgentInput Agent 输入
type AgentInput struct {
    AgentName   string            `json:"agent_name"`   // 如 "bmad-agent-pm"
    Task        string            `json:"task"`         // 任务描述
    Context     map[string]string `json:"context"`      // 上下文信息
    WorkingDir  string            `json:"working_dir"`  // 工作目录
}

// AgentOutput Agent 输出
type AgentOutput struct {
    Success     bool              `json:"success"`
    Content     string            `json:"content"`      // 输出内容
    Duration    time.Duration     `json:"duration"`
    Error       string            `json:"error,omitempty"`
}
```

---

## Data Flow

### 完整执行流程

```
用户创建需求
     │
     ▼
┌─────────────────┐
│  Web Handler    │
└─────────────────┘
     │
     ▼
┌─────────────────┐     ┌─────────────────┐
│     Tracker     │────▶│ 创建 Beads 任务 │
└─────────────────┘     └─────────────────┘
     │
     ▼
┌─────────────────────────────────────────────────────────────┐
│                     Harness Orchestrator                     │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Planner                                              │   │
│  │                                                      │   │
│  │  P1: PM Agent (需求澄清) ─── 人工参与                │   │
│  │  P2: QA Agent (BDD规则)                              │   │
│  │  P3: Architect Agent (领域建模)                      │   │
│  │  P4: Architect Agent (架构设计)                      │   │
│  │  P5: Architect Agent (接口设计)                      │   │
│  │                                                      │   │
│  │  产出: PlannerOutput (不可变)                        │   │
│  └─────────────────────────────────────────────────────┘   │
│                           │                                  │
│                           ▼                                  │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Generator                                            │   │
│  │                                                      │   │
│  │  Phase 1 (并行):                                     │   │
│  │  G1: QA Agent (BDD测试脚本)                          │   │
│  │  G2: QA Agent (集成测试)                             │   │
│  │  G3: Dev Agent (单元测试)                            │   │
│  │                                                      │   │
│  │  Phase 2 (顺序):                                     │   │
│  │  G4: Dev Agent (代码实现)                            │   │
│  │                                                      │   │
│  └─────────────────────────────────────────────────────┘   │
│                           │                                  │
│                           ▼                                  │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Evaluator                                            │   │
│  │                                                      │   │
│  │  E1: QA Agent (BDD验收)                              │   │
│  │  E2: QA Agent (TDD验收)                              │   │
│  │  E3: Code Review Agent (代码审计)                    │   │
│  │  E4: Editorial Agent (风格评审)                      │   │
│  │                                                      │   │
│  │  产出: EvaluatorOutput                               │   │
│  └─────────────────────────────────────────────────────┘   │
│                           │                                  │
│                           ▼                                  │
│                    ┌──────┴──────┐                          │
│                   通过           失败                        │
│                    │              │                          │
│                    ▼              ▼                          │
│                  完成      迭代次数+1                        │
│                                │                             │
│                                ▼                             │
│                          迭代 > 5 ?                          │
│                            ↙    ↘                           │
│                          否      是                          │
│                          │       │                          │
│                          ▼       ▼                          │
│                     回到 G4   转人工                         │
│                    (带失败报告)                              │
│                                                              │
└─────────────────────────────────────────────────────────────┘
     │
     ▼
┌─────────────────┐
│   Git Commit    │
└─────────────────┘
```

---

## Error Handling

### 错误码定义

```go
// internal/common/errors/errors.go

var (
    // Harness 相关错误
    ErrIterationExceeded = errors.New("harness.iteration_exceeded: 迭代次数已达上限")
    ErrPlannerFailed     = errors.New("harness.planner_failed: Planner 执行失败")
    ErrGeneratorFailed   = errors.New("harness.generator_failed: Generator 执行失败")
    ErrEvaluatorFailed   = errors.New("harness.evaluator_failed: Evaluator 执行失败")
    
    // Agent 相关错误
    ErrAgentUnavailable  = errors.New("agent.unavailable: BMAD Agent 不可用")
    ErrAgentTimeout      = errors.New("agent.timeout: Agent 执行超时")
    ErrAgentFailed       = errors.New("agent.failed: Agent 执行失败")
    
    // Tracker 相关错误
    ErrTrackerUnavailable = errors.New("tracker.unavailable: Beads CLI 不可用")
)
```

---

## Configuration Schema

```yaml
# .sym/config.yaml

# 基础配置
project_name: my-project
user_name: DK

# Tracker 配置
tracker:
  type: beads
  path: beads  # Beads CLI 路径

# Harness 配置
harness:
  max_iterations: 5        # 最大迭代次数，默认 5
  
  # BMAD Agent 配置
  bmad:
    enabled: true
    base_path: ~/.bmad     # BMAD 安装路径
    
    agents:
      planner:
        - name: bmad-agent-pm
          role: 需求理解
        - name: bmad-agent-qa
          role: BDD规则生成
        - name: bmad-agent-architect
          role: 架构设计
          
      generator:
        - name: bmad-agent-qa
          role: 测试脚本
        - name: bmad-agent-dev
          role: 代码实现
          
      evaluator:
        - name: bmad-agent-qa
          role: 测试执行
        - name: bmad-code-review
          role: 代码审计
        - name: bmad-editorial-review-prose
          role: 风格评审

# Web 服务配置
server:
  port: 8080
```

---

## Testing Strategy

### 测试覆盖目标

| 模块 | 目标覆盖率 |
|------|-----------|
| harness/ | 70% |
| agent/ | 60% |
| tracker/ | 80% |

### 测试策略

| 类型 | 范围 |
|------|------|
| **单元测试** | 各组件独立功能 |
| **集成测试** | P-G-E 完整流程 |
| **E2E 测试** | 用户旅程覆盖 |

### Mock 策略

```go
// 测试用 Mock
type MockAgentCaller struct{}

func (m *MockAgentCaller) Call(ctx context.Context, input *AgentInput) (*AgentOutput, error) {
    return &AgentOutput{
        Success: true,
        Content: "mock output",
    }, nil
}

func (m *MockAgentCaller) CheckAvailability() error {
    return nil
}
```

---

## Implementation Priority

### Phase 1: 基础设施

1. `internal/harness/orchestrator.go` - 主编排器
2. `internal/agent/caller.go` - Agent 调用抽象
3. 配置更新

### Phase 2: Planner

4. `internal/harness/planner.go` - Planner 执行器
5. Agent 调用实现 (PM, QA, Architect)

### Phase 3: Generator

6. `internal/harness/generator.go` - Generator 执行器 (含并行)
7. Agent 调用实现 (Dev, QA)

### Phase 4: Evaluator

8. `internal/harness/evaluator.go` - Evaluator 执行器
9. Agent 调用实现 (Code Review, Editorial)

### Phase 5: 迭代机制

10. `internal/harness/iteration.go` - 迭代管理
11. 失败报告处理

### Phase 6: 集成

12. Web Handler 适配
13. Beads 子任务结构
14. 完整流程测试

---

## Verification Checklist

- [x] 需求分析完成
- [x] 架构决策完成
- [x] 接口定义完成
- [x] 数据流设计完成
- [x] 错误处理设计完成
- [x] 配置结构设计完成
- [x] 测试策略定义完成
- [x] 实现优先级确定