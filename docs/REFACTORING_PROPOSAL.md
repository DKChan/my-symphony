# My-Symphony 重构方案

> 状态: 草案  
> 创建时间: 2026年4月20日  
> 作者: AI Agent Analysis  

---

## 一、当前系统问题汇总

### 1.1 致命缺陷（P0 - 阻塞生产）

#### 问题 1: BlockedBy 永远为空（阻塞检测完全失效）

**位置**: `internal/tracker/linear.go:295`

**代码**:
```go
func (li LinearIssue) ToDomain() *domain.Issue {
    issue := &domain.Issue{
        // ...
        BlockedBy:   make([]domain.BlockerRef, 0),  // ⚠️ 永远为空数组！
    }
}
```

**问题描述**: Linear API 查询中没有获取 `relations` 字段，但代码中却依赖它进行阻塞检测。

**影响**:
- Todo 状态的阻塞检测逻辑完全失效
- 被阻塞的任务会被错误地调度执行
- 导致资源浪费和任务冲突

**修复方案**: 在 GraphQL 查询中添加 `relations` 字段获取阻塞关系

```graphql
query {
  issues {
    nodes {
      id
      # ... 现有字段
      relations {
        nodes {
          type
          relatedIssue { id identifier state { name } }
        }
      }
    }
  }
}
```

---

#### 问题 2: 状态流转缺乏持久化（严重）

**位置**: `internal/orchestrator/orchestrator.go:28-32`

**代码**:
```go
type Orchestrator struct {
    state        *domain.OrchestratorState  // 纯内存状态
    retryTimers  map[string]*time.Timer     // 内存定时器
    // ...
}
```

**问题描述**: 所有状态保存在内存中，服务重启后全部丢失。

**影响**:
1. 服务重启丢失所有运行中任务和重试队列
2. 无法水平扩展，多实例部署会导致重复调度
3. 无法优雅升级，重启 = 中断所有正在执行的任务

**SPEC 要求**: Section 18.2 "Retry queue MUST be durable across restarts"

**当前状态**: ❌ 未实现（代码中标记为 TODO）

---

#### 问题 3: 全局 Token 统计被覆盖（数据完整性风险）

**位置**: `internal/orchestrator/orchestrator.go:426-428`

**代码**:
```go
// 更新全局统计
o.state.CodexTotals.InputTokens = entry.Session.CodexInputTokens
o.state.CodexTotals.OutputTokens = entry.Session.CodexOutputTokens
o.state.CodexTotals.TotalTokens = entry.Session.CodexTotalTokens
```

**问题描述**: 直接赋值而非累加，多个并发 session 的统计互相覆盖。

**影响**: 最终只保留最后一个 session 的统计值，数据完全错误。

---

#### 问题 4: Codex readLine Bug（数据丢失）

**位置**: `internal/agent/codex.go:67`

**代码**:
```go
scanner := bufio.NewScanner(stdout)  // 每次调用创建新 Scanner
```

**问题描述**: Scanner 有内部缓冲区，第一次调用可能消费多行，后续调用丢失已缓冲内容。

**影响**: 高吞吐场景下必现数据丢失。

---

### 1.2 架构债务（P1 - 影响维护）

#### 问题 5: 状态定义过度

**定义了 11 个状态，实际只使用 3-4 个**:

| 定义的状态 | 实际使用 |
|-----------|---------|
| preparing_workspace | ❌ 未使用 |
| building_prompt | ❌ 未使用 |
| launching_agent_process | ❌ 未使用 |
| initializing_session | ❌ 未使用 |
| streaming_turn | ⚠️ 部分使用 |
| finishing | ❌ 未使用 |
| succeeded | ✅ 使用 |
| failed | ✅ 使用 |
| timed_out | ⚠️ 部分使用 |
| stalled | ⚠️ 部分使用 |
| canceled_by_reconcile | ⚠️ 部分使用 |

**实际流转**: `Claimed → Running → Succeeded/Failed`

---

#### 问题 6: 重试机制设计缺陷

**位置**: `internal/orchestrator/orchestrator.go:480-513`

**问题**:
1. 定时器与状态不同步 — 定时器触发时可能状态已变
2. 无持久化 — 重启后重试队列丢失
3. 无最大重试次数限制 — 可能无限重试
4. 退避算法固定 — 无抖动，可能惊群

---

#### 问题 7: 终态处理不一致

**位置**: `internal/orchestrator/orchestrator.go:170`

**代码**:
```go
if o.cfg.IsTerminalState(issue.State) {
    go o.terminateAndCleanup(id, entry)  // 异步终止
}
```

**问题**:
1. 异步终止无等待 — 启动后就返回
2. Agent 进程可能仍在运行 — 工作空间被删除但进程还在
3. 无优雅关闭 — 直接删除，无信号通知

---

#### 问题 8: Server 代码过于臃肿

**位置**: `internal/server/server.go`

**问题**: 1353 行，内嵌完整 HTML/CSS/JS，不可维护，违反关注点分离。

---

#### 问题 9: 无构建系统

**问题**: 无 Makefile/Taskfile，CI 无法标准化。

---

#### 问题 10: 二进制文件提交 Git

**位置**: `bin/symphony`

**问题**: 膨胀仓库，版本混乱。

---

### 1.3 代码质量问题（P2）

#### 问题 11: 核心模块零测试

**有测试**: config, domain, workflow, workspace（边缘模块）  
**无测试**: orchestrator, agent, tracker, server（核心模块）

**核心编排逻辑 673 行零测试** — 最大质量风险。

---

#### 问题 12: 依赖版本偏旧

**位置**: `go.mod`

**问题**: gin v1.9.1（2023 年版本），存在已知安全漏洞。

---

#### 问题 13: 无 context 超时传播

**位置**: `internal/tracker/linear.go:31-34`

**代码**:
```go
httpClient: &http.Client{
    Timeout: 30 * time.Second,  // 硬编码
},
```

**问题**: tracker 层 HTTP 调用无 context 超时保护。

---

#### 问题 14: 使用 fmt.Printf 而非结构化日志

**位置**: 全局

**问题**: 全部使用 `fmt.Printf` 输出，无法被监控系统采集。

---

## 二、重构方案：事件驱动架构

### 2.1 架构目标

1. **状态持久化** — 服务重启后可恢复
2. **可观测性** — 完整的状态流转追踪
3. **可扩展性** — 支持多实例部署
4. **可测试性** — 核心逻辑可自动化测试
5. **优雅关闭** — 支持任务优雅终止

---

### 2.2 架构设计

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        事件驱动状态流转架构                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │                      大循环：状态机（State Machine）                   │  │
│   │                                                                     │  │
│   │   ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐         │  │
│   │   │ Pending │───►│ Running │───►│Success  │    │ Failed  │         │  │
│   │   │         │    │         │    │         │    │         │         │  │
│   │   └────┬────┘    └────┬────┘    └─────────┘    └────┬────┘         │  │
│   │        │              │                             │               │  │
│   │        │              │                             │               │  │
│   │        └──────────────┴─────────────────────────────┘               │  │
│   │                       │                                             │  │
│   │                       ▼                                             │  │
│   │                  ┌─────────┐                                        │  │
│   │                  │ Retry   │                                        │  │
│   │                  │ Queue   │                                        │  │
│   │                  └────┬────┘                                        │  │
│   │                       │                                             │  │
│   │                       └────────────────────────────────────────►    │  │
│   │                                                                     │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                    │                                        │
│                                    │ Event: TaskCreated                    │
│                                    │ Event: TaskStarted                    │
│                                    │ Event: TaskCompleted                  │
│                                    │ Event: TaskFailed                     │
│                                    │ Event: TaskRetryScheduled             │
│                                    ▼                                        │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │                      小循环：任务执行（Task Executor）                 │  │
│   │                      （在 Running 状态内部）                          │  │
│   │                                                                     │  │
│   │   ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐         │  │
│   │   │ Prepare │───►│ Execute │───►│ Monitor │───►│ Finish  │         │  │
│   │   │Workspace│    │  Agent  │    │ Progress│    │ Cleanup │         │  │
│   │   └─────────┘    └────┬────┘    └─────────┘    └─────────┘         │  │
│   │                       │                                             │  │
│   │                       │ 子步骤循环（Agent 内部）                      │  │
│   │                       ▼                                             │  │
│   │              ┌─────────────────┐                                    │  │
│   │              │  Planning       │                                    │  │
│   │              │  Coding         │                                    │  │
│   │              │  Testing        │                                    │  │
│   │              │  (Agent Loop)   │                                    │  │
│   │              └─────────────────┘                                    │  │
│   │                                                                     │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

### 2.3 核心组件

#### 事件总线（Event Bus）

```go
type EventBus interface {
    Publish(ctx context.Context, event Event) error
    Subscribe(eventType string, handler EventHandler) error
    Start() error
    Stop() error
}

type Event struct {
    ID        string                 `json:"id"`
    Type      string                 `json:"type"`
    Timestamp time.Time              `json:"timestamp"`
    IssueID   string                 `json:"issue_id"`
    Payload   map[string]interface{} `json:"payload"`
}
```

**事件类型**:
- `task:created` — 任务创建
- `task:scheduled` — 任务已安排重试
- `task:started` — 任务开始执行
- `task:step:started` — 执行步骤开始
- `task:step:completed` — 执行步骤完成
- `task:step:failed` — 执行步骤失败
- `task:completed` — 任务完成
- `task:failed` — 任务失败
- `task:retry:scheduled` — 重试已安排
- `task:canceled` — 任务取消
- `task:stalled` — 任务停滞

---

#### 状态机（State Machine）

**简化后的状态**:
```go
const (
    StatePending    TaskState = "pending"    // 等待调度
    StateScheduled  TaskState = "scheduled"  // 已安排重试
    StateRunning    TaskState = "running"    // 运行中
    StateSucceeded  TaskState = "succeeded"  // 成功
    StateFailed     TaskState = "failed"     // 失败
    StateCanceled   TaskState = "canceled"   // 已取消
)
```

**状态转换规则**:
```go
var StateTransitions = map[TaskState]map[string]TaskState{
    StatePending: {
        EventTaskStarted:        StateRunning,
        EventTaskScheduled:      StateScheduled,
    },
    StateScheduled: {
        EventTaskStarted:        StateRunning,
    },
    StateRunning: {
        EventTaskCompleted:      StateSucceeded,
        EventTaskFailed:         StateFailed,
        EventTaskRetryScheduled: StateScheduled,
        EventTaskCanceled:       StateCanceled,
    },
    StateFailed: {
        EventTaskRetryScheduled: StateScheduled,
    },
}
```

---

#### 任务执行器（Task Executor）

**执行步骤**:
```go
var ExecutionFlow = []ExecutionStep{
    {
        Name:    "prepare_workspace",
        Execute: prepareWorkspace,
        Timeout: 30 * time.Second,
    },
    {
        Name:    "build_prompt",
        Execute: buildPrompt,
        Timeout: 10 * time.Second,
    },
    {
        Name:    "run_before_hook",
        Execute: runBeforeHook,
        Timeout: 60 * time.Second,
    },
    {
        Name:      "execute_agent",
        Execute:   executeAgent,
        Timeout:   30 * time.Minute,
        Retryable: true,
    },
    {
        Name:    "run_after_hook",
        Execute: runAfterHook,
        Timeout: 30 * time.Second,
    },
    {
        Name:    "cleanup",
        Execute: cleanup,
        Timeout: 10 * time.Second,
    },
}
```

---

#### 持久化层（Event Store）

```go
type EventStore interface {
    Append(ctx context.Context, event Event) error
    GetEvents(ctx context.Context, issueID string, fromVersion int) ([]Event, error)
    GetCurrentState(ctx context.Context, issueID string) (*TaskStateSnapshot, error)
    SaveSnapshot(ctx context.Context, snapshot *TaskStateSnapshot) error
}

type TaskStateSnapshot struct {
    IssueID         string                 `json:"issue_id"`
    State           TaskState              `json:"state"`
    ExecutionState  ExecutionState         `json:"execution_state"`
    Version         int                    `json:"version"`
    Events          []Event                `json:"events"`
    Metadata        map[string]interface{} `json:"metadata"`
    CreatedAt       time.Time              `json:"created_at"`
    UpdatedAt       time.Time              `json:"updated_at"`
}
```

**存储选型**:
- **PostgreSQL**: 单机/小规模，事务支持
- **Redis Streams**: 高性能，支持消费组
- **SQLite**: 开发/测试，零配置

---

### 2.4 代码结构重构

```
internal/
├── eventbus/           # 事件总线
│   ├── eventbus.go     # 接口定义
│   ├── redis/          # Redis 实现
│   └── memory/         # 内存实现（测试）
├── statemachine/       # 状态机
│   ├── statemachine.go # 核心状态机
│   ├── states.go       # 状态定义
│   └── handlers.go     # 状态处理器
├── executor/           # 任务执行器
│   ├── executor.go     # 执行器接口
│   ├── loop.go         # 小循环实现
│   ├── steps.go        # 执行步骤
│   └── agent/          # Agent 调用
├── store/              # 持久化
│   ├── store.go        # 存储接口
│   ├── postgres/       # PostgreSQL 实现
│   └── snapshot.go     # 快照管理
├── scheduler/          # 调度器
│   ├── scheduler.go    # 调度逻辑
│   └── retry.go        # 重试管理
└── reconciler/         # 协调器
    └── reconciler.go   # 状态协调
```

---

## 三、测试策略

### 3.1 测试目标

针对上述问题，构造可执行的自动化测试，确保重构后的系统：
1. 状态流转正确
2. 持久化可靠
3. 并发安全
4. 可恢复
5. 可观测

---

### 3.2 测试分类

#### 类别 A: 单元测试（Unit Tests）

**A1. BlockedBy 解析测试**
```go
// TestBlockedByParsing
// 目标: 验证 Linear API 返回的 relations 正确解析为 BlockedBy
// 输入: 包含 relations 的 GraphQL 响应
// 期望: BlockedBy 数组正确填充
// 覆盖: 问题 1

func TestBlockedByParsing(t *testing.T) {
    // Given: 包含阻塞关系的 Linear 响应
    response := `{
        "issues": {
            "nodes": [{
                "id": "issue-1",
                "relations": {
                    "nodes": [{
                        "type": "blocks",
                        "relatedIssue": {
                            "id": "issue-2",
                            "identifier": "PROJ-2",
                            "state": {"name": "In Progress"}
                        }
                    }]
                }
            }]
        }
    }`
    
    // When: 解析为领域模型
    issue := parseLinearResponse(response)
    
    // Then: BlockedBy 正确填充
    assert.Len(t, issue.BlockedBy, 1)
    assert.Equal(t, "issue-2", issue.BlockedBy[0].ID)
    assert.Equal(t, "In Progress", *issue.BlockedBy[0].State)
}
```

**A2. 阻塞检测逻辑测试**
```go
// TestShouldDispatchWithBlockers
// 目标: 验证被阻塞的任务不会被调度
// 输入: 有 BlockedBy 且阻塞项非终态的 Issue
// 期望: shouldDispatch 返回 false
// 覆盖: 问题 1

func TestShouldDispatchWithBlockers(t *testing.T) {
    // Given: 被阻塞的 Todo 状态 Issue
    issue := &domain.Issue{
        State: "Todo",
        BlockedBy: []domain.BlockerRef{
            {State: stringPtr("In Progress")},  // 非终态
        },
    }
    
    // When: 检查是否应该调度
    should := orchestrator.shouldDispatch(issue)
    
    // Then: 不应该调度
    assert.False(t, should)
}
```

**A3. Token 统计累加测试**
```go
// TestTokenStatsAccumulation
// 目标: 验证全局 Token 统计正确累加而非覆盖
// 输入: 多个并发 Session 的 Token 统计
// 期望: 总计为各 Session 之和
// 覆盖: 问题 3

func TestTokenStatsAccumulation(t *testing.T) {
    // Given: 两个 Session 的统计
    session1 := &domain.LiveSession{CodexTotalTokens: 1000}
    session2 := &domain.LiveSession{CodexTotalTokens: 2000}
    
    // When: 依次更新统计
    orchestrator.updateStats(session1)
    orchestrator.updateStats(session2)
    
    // Then: 总计为 3000，不是 2000
    assert.Equal(t, int64(3000), orchestrator.state.CodexTotals.TotalTokens)
}
```

**A4. Codex Scanner 复用测试**
```go
// TestCodexScannerReuse
// 目标: 验证 Scanner 在 Session 级别复用
// 输入: 多行输出
// 期望: 所有行被正确读取，无丢失
// 覆盖: 问题 4

func TestCodexScannerReuse(t *testing.T) {
    // Given: 多行输出
    output := "line1\nline2\nline3\n"
    
    // When: 多次读取
    lines := readAllLines(output)
    
    // Then: 所有行被读取
    assert.Equal(t, []string{"line1", "line2", "line3"}, lines)
}
```

---

#### 类别 B: 集成测试（Integration Tests）

**B1. 事件持久化测试**
```go
// TestEventPersistence
// 目标: 验证事件正确持久化到存储
// 输入: 一系列状态变更事件
// 期望: 事件可从存储恢复
// 覆盖: 问题 2

func TestEventPersistence(t *testing.T) {
    // Given: PostgreSQL 测试数据库
    store := postgres.NewTestStore(t)
    
    // When: 发布事件
    eventBus.Publish(ctx, Event{Type: EventTaskCreated, IssueID: "issue-1"})
    eventBus.Publish(ctx, Event{Type: EventTaskStarted, IssueID: "issue-1"})
    eventBus.Publish(ctx, Event{Type: EventTaskCompleted, IssueID: "issue-1"})
    
    // Then: 可从存储恢复
    events, _ := store.GetEvents(ctx, "issue-1", 0)
    assert.Len(t, events, 3)
    assert.Equal(t, EventTaskCompleted, events[2].Type)
}
```

**B2. 状态恢复测试**
```go
// TestStateRecovery
// 目标: 验证服务重启后可恢复状态
// 输入: 运行中任务的事件流
// 期望: 重启后可恢复到正确状态
// 覆盖: 问题 2

func TestStateRecovery(t *testing.T) {
    // Given: 运行中的任务
    orchestrator.Start()
    orchestrator.Dispatch(issue)
    
    // When: 模拟重启（创建新实例，从存储恢复）
    newOrchestrator := NewOrchestratorFromStore(store)
    
    // Then: 状态恢复正确
    state := newOrchestrator.GetState(issue.ID)
    assert.Equal(t, StateRunning, state)
}
```

**B3. 重试队列持久化测试**
```go
// TestRetryQueuePersistence
// 目标: 验证重试队列在重启后不丢失
// 输入: 失败任务安排重试
// 期望: 重启后重试任务仍在队列
// 覆盖: 问题 2, 6

func TestRetryQueuePersistence(t *testing.T) {
    // Given: 失败任务安排重试
    orchestrator.HandleFailure(issue, retryable: true)
    
    // When: 模拟重启
    newOrchestrator := NewOrchestratorFromStore(store)
    
    // Then: 重试任务在队列
    retryEntry := newOrchestrator.GetRetryEntry(issue.ID)
    assert.NotNil(t, retryEntry)
}
```

**B4. 并发安全测试**
```go
// TestConcurrentStateUpdates
// 目标: 验证并发状态更新安全
// 输入: 多个并发 Session 更新统计
// 期望: 无竞态条件，数据正确
// 覆盖: 问题 3

func TestConcurrentStateUpdates(t *testing.T) {
    // Given: 多个并发更新
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            orchestrator.updateStats(&domain.LiveSession{CodexTotalTokens: 100})
            wg.Done()
        }()
    }
    wg.Wait()
    
    // Then: 总计正确
    assert.Equal(t, int64(10000), orchestrator.state.CodexTotals.TotalTokens)
}
```

---

#### 类别 C: 端到端测试（E2E Tests）

**C1. 完整任务生命周期测试**
```go
// TestCompleteTaskLifecycle
// 目标: 验证完整任务生命周期
// 输入: 创建 → 调度 → 执行 → 完成
// 期望: 状态流转正确，事件完整
// 覆盖: 问题 2, 5, 6

func TestCompleteTaskLifecycle(t *testing.T) {
    // Given: 测试环境
    env := SetupE2EEnvironment(t)
    
    // When: 创建任务并等待完成
    issue := env.CreateIssue("Test Task")
    env.WaitForCompletion(issue.ID, timeout: 5*time.Minute)
    
    // Then: 验证事件流
    events := env.GetEvents(issue.ID)
    assertEventSequence(t, events, []string{
        EventTaskCreated,
        EventTaskStarted,
        EventTaskStepStarted,
        EventTaskStepCompleted,
        EventTaskCompleted,
    })
}
```

**C2. 优雅关闭测试**
```go
// TestGracefulShutdown
// 目标: 验证优雅关闭机制
// 输入: 运行中任务，发送关闭信号
// 期望: 任务优雅终止，状态正确
// 覆盖: 问题 7

func TestGracefulShutdown(t *testing.T) {
    // Given: 运行中的任务
    orchestrator.Dispatch(issue)
    
    // When: 发送关闭信号
    orchestrator.Shutdown(ctx, gracePeriod: 30*time.Second)
    
    // Then: 任务优雅终止
    assert.Eventually(t, func() bool {
        return orchestrator.IsStopped()
    }, 30*time.Second, 100*time.Millisecond)
}
```

**C3. 重试机制测试**
```go
// TestRetryMechanism
// 目标: 验证重试机制正确工作
// 输入: 失败任务
// 期望: 按退避策略重试，最终成功或失败
// 覆盖: 问题 6

func TestRetryMechanism(t *testing.T) {
    // Given: 总是失败的任务
    env.SetupFailingAgent()
    
    // When: 调度任务
    orchestrator.Dispatch(issue)
    
    // Then: 验证重试次数
    time.Sleep(5 * time.Minute)
    retryCount := env.GetRetryCount(issue.ID)
    assert.GreaterOrEqual(t, retryCount, 3)
    assert.LessOrEqual(t, retryCount, maxRetries)
}
```

---

#### 类别 D: 性能测试（Performance Tests）

**D1. 高并发调度测试**
```go
// TestHighConcurrencyDispatch
// 目标: 验证高并发下调度性能
// 输入: 1000 个并发任务
// 期望: 无竞态条件，调度延迟可接受
// 覆盖: 问题 2, 3

func TestHighConcurrencyDispatch(t *testing.T) {
    // Given: 1000 个任务
    tasks := generateTasks(1000)
    
    // When: 并发调度
    start := time.Now()
    for _, task := range tasks {
        go orchestrator.Dispatch(task)
    }
    
    // Then: 调度延迟可接受
    elapsed := time.Since(start)
    assert.Less(t, elapsed, 10*time.Second)
}
```

**D2. 事件回放性能测试**
```go
// TestEventReplayPerformance
// 目标: 验证事件回放性能
// 输入: 10000 个事件
// 期望: 回放时间可接受
// 覆盖: 问题 2

func TestEventReplayPerformance(t *testing.T) {
    // Given: 10000 个事件
    events := generateEvents(10000)
    store.SaveEvents(events)
    
    // When: 回放
    start := time.Now()
    statemachine.Replay(events)
    elapsed := time.Since(start)
    
    // Then: 回放时间可接受
    assert.Less(t, elapsed, 5*time.Second)
}
```

---

### 3.3 测试覆盖率要求

| 模块 | 覆盖率目标 | 关键测试 |
|------|-----------|---------|
| statemachine | 90%+ | 状态转换、事件处理、并发安全 |
| executor | 85%+ | 步骤执行、超时处理、错误恢复 |
| eventbus | 80%+ | 发布订阅、持久化、恢复 |
| store | 80%+ | CRUD、事务、快照 |
| scheduler | 75%+ | 调度策略、重试逻辑 |
| orchestrator | 75%+ | 集成测试为主 |

---

### 3.4 测试工具

```go
// 测试工具包
package testutil

// FakeTracker 用于测试的 Tracker 实现
type FakeTracker struct {
    Issues []*domain.Issue
}

// FakeRunner 用于测试的 Runner 实现
type FakeRunner struct {
    ShouldFail bool
    Delay      time.Duration
}

// EventRecorder 记录所有事件用于断言
type EventRecorder struct {
    Events []Event
}

// StateSnapshot 状态快照用于比较
type StateSnapshot struct {
    State          string
    ExecutionState string
    Metadata       map[string]interface{}
}
```

---

## 四、实施计划

### Phase 1: 基础设施（2 周）

1. **添加事件总线接口和内存实现**
   - 创建 `internal/eventbus/eventbus.go`
   - 实现内存版本用于测试

2. **添加存储接口和 PostgreSQL 实现**
   - 创建 `internal/store/store.go`
   - 实现 PostgreSQL 存储

3. **添加测试框架**
   - 创建 `testutil` 包
   - 添加 FakeTracker、FakeRunner

### Phase 2: 状态机重构（2 周）

1. **提取状态机逻辑**
   - 创建 `internal/statemachine/statemachine.go`
   - 实现状态转换规则

2. **简化状态定义**
   - 从 11 个状态简化为 6 个
   - 删除未使用的中间状态

3. **添加状态机测试**
   - 实现 A1-A4 测试
   - 确保 90%+ 覆盖率

### Phase 3: 执行器重构（2 周）

1. **重构任务执行器**
   - 创建 `internal/executor/executor.go`
   - 实现小循环（ExecutionFlow）

2. **修复 Codex Scanner 问题**
   - Session 级别复用 Scanner

3. **添加执行器测试**
   - 实现 B1-B4 测试

### Phase 4: 集成与优化（2 周）

1. **集成事件驱动架构**
   - 替换原有 orchestrator 逻辑

2. **修复 BlockedBy 问题**
   - 更新 Linear GraphQL 查询

3. **添加集成测试**
   - 实现 C1-C3 测试

### Phase 5: 性能与完善（1 周）

1. **性能测试**
   - 实现 D1-D2 测试

2. **添加优雅关闭**
   - 实现信号处理

3. **完善文档**
   - 更新 README、API 文档

---

## 五、风险评估

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|---------|
| 重构引入新 Bug | 高 | 高 | 充分测试，渐进式迁移 |
| 性能下降 | 中 | 中 | 性能测试，基准对比 |
| 数据迁移失败 | 低 | 高 | 备份，回滚方案 |
| 工期延误 | 中 | 中 | 分阶段交付，MVP 优先 |

---

## 六、附录

### 6.1 参考资料

- [Event Sourcing Pattern](https://martinfowler.com/eaaDev/EventSourcing.html)
- [CQRS Pattern](https://martinfowler.com/bliki/CQRS.html)
- [State Machine Pattern](https://refactoring.guru/design-patterns/state)

### 6.2 相关 Issue

- Issue #1: BlockedBy 解析失败
- Issue #2: 状态持久化缺失
- Issue #3: Token 统计错误
- Issue #4: Scanner 数据丢失

### 6.3 变更日志

| 版本 | 日期 | 变更 |
|------|------|------|
| 0.1 | 2026-04-20 | 初始草案 |

---

**注意**: 本文档为重构方案草案，具体实现时需根据实际情况调整。
