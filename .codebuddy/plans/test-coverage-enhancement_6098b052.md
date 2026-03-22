---
name: test-coverage-enhancement
overview: 为 Symphony 项目补充单元测试，提高测试覆盖率至核心模块 80% 以上
todos:
  - id: orchestrator-core-tests
    content: 为 orchestrator 添加核心调度和重试逻辑的单元测试
    status: completed
  - id: agent-prompt-tests
    content: 为 agent/prompt 添加提示词构建和模板处理的单元测试
    status: completed
  - id: agent-runner-tests
    content: 为 agent/runner 添加工厂函数的单元测试
    status: completed
  - id: common-types-tests
    content: 为 common 添加 SSE 广播器和辅助函数的单元测试
    status: completed
  - id: config-supplement-tests
    content: 补充 config 配置验证和解析的测试场景
    status: completed
  - id: workspace-hook-tests
    content: 补充 workspace 钩子执行和错误处理的测试
    status: completed
---

## 产品概述

为 Symphony 编码代理编排服务补充单元测试，提升测试覆盖率到目标水平（核心逻辑 > 80%），确保代码质量和可靠性。

## 核心需求

1. 检查当前测试覆盖情况
2. 识别无测试或测试不足的模块
3. 为关键模块补充单元测试，优先保证核心业务逻辑的测试覆盖
4. 遵循项目测试规范（表驱动测试、t.Run()组织、资源清理等）

## 技术栈

- 测试框架: Go testing 包
- 断言工具: 项目无 testify 依赖，使用标准库
- Mock 工具: 基于接口的 mock 实现

## 实现方案

### 测试优先级分级

#### 优先级 1 - 核心业务逻辑（必须实现）

这些模块是系统核心，缺乏测试会导致重大风险：

1. **internal/orchestrator/orchestrator_test.go** (新建)

- 测试目标: 覆盖调度、重试、协调、并发控制核心逻辑
- 关键测试场景:
    - shouldDispatch: 正常调度、必填字段缺失、状态检查、阻塞依赖、并发限制
    - sortForDispatch: 优先级排序、创建时间排序、标识符排序
    - scheduleRetry: 正常退出重试、异常退出重试、退避时间计算
    - calculateBackoff: 指数退避、最大退避限制
    - reconcile: 终态处理、非活跃状态处理、状态更新
    - checkStalled: 超时检测、活动时间更新
    - hasAvailableSlots: 全局限制、按状态限制
    - dispatch: 双重检查、状态管理
    - onAgentEvent: token统计、事件处理
- 测试工具: 使用 Mock Tracker，模拟各种状态流转

2. **internal/agent/prompt_test.go** (新建)

- 测试目标: 覆盖提示词构建和模板处理逻辑
- 关键测试场景:
    - buildPrompt: 基本字段替换、nil字段处理、attempt参数
    - removeBlock: 保留块内容、移除块内容、多个块处理、嵌套块处理
- 测试工具: 创建 Issue 测试数据，使用表驱动测试

3. **internal/agent/runner_test.go** (新建)

- 测试目标: 覆盖 Runner 工厂函数
- 关键测试场景:
    - NewRunner: codex类型、claude类型、opencode类型、默认类型
- 测试工具: 表驱动测试，验证返回类型

#### 优先级 2 - 基础设施和配置（重要）

这些模块支撑系统运行，需要足够测试覆盖：

1. **internal/common/types_test.go** (新建)

- 测试目标: 覆盖 SSE 广播器和辅助函数
- 关键测试场景:
    - SSEBroadcaster.Subscribe/Unsubscribe: 订阅/取消订阅、并发安全
    - SSEBroadcaster.Broadcast: 广播事件、最后一个载荷保存
    - SSEBroadcaster.GetLastPayload: 获取最后载荷
    - TotalRuntimeSeconds: 计算运行时间
    - FormatRuntimeSeconds: 格式化输出
    - FormatRuntimeAndTurns: 格式化带轮次的运行时间
    - StateBadgeClass: 状态映射到CSS类
    - FormatInt: 数字格式化（K/M后缀）
    - PrettyValue: JSON格式化
    - EscapeHTML: HTML转义
- 测试工具: 并发测试、表驱动测试

2. **internal/config/config_test.go** (补充)

- 测试目标: 提升覆盖率到 80%+
- 补充测试场景:
    - ValidateDispatchConfig: GitHub类型配置验证、GitHub缺少repo、Claude配置、OpenCode配置
    - ParseConfig: 嵌套配置、类型转换、环境变量未设置
    - DefaultConfig: 所有默认值验证
- 测试工具: 表驱动测试，覆盖所有验证分支

3. **internal/workspace/manager_test.go** (补充)

- 测试目标: 提升覆盖率到 80%+，补充钩子测试
- 补充测试场景:
    - RunBeforeRunHook/RunAfterRunHook/RunBeforeRemoveHook: 钩子执行、钩子不存在、钩子失败
    - runHook: 超时处理、环境变量设置、输出截断
    - RemoveWorkspace: 路径不存在、路径验证、删除失败
    - GetWorkspacePath: 路径清理验证
    - CleanupTerminalWorkspaces: 批量清理、部分失败处理
- 测试工具: 临时脚本文件、超时测试

#### 优先级 3 - HTTP层和客户端（重要但可后续补充）

这些模块需要测试但相对独立：

1. **internal/server/handlers/api_handler_test.go** (新建)

- 测试目标: 覆盖 API 处理器
- 关键测试场景:
    - HandleGetState: 正常返回、空状态
    - HandleGetIssue: 找到运行中的问题、找到重试中的问题、问题不存在
    - HandleRefresh: 正常返回
- 测试工具: Gin测试路由、Mock Orchestrator

2. **internal/server/handlers/sse_handler_test.go** (新建)

- 测试目标: 覆盖 SSE 处理器
- 关键测试场景:
    - Handle: 设置SSE头、订阅广播器、发送初始状态、流式传输、客户端断开
- 测试工具: Gin测试路由、Mock Broadcaster

3. **internal/server/presenter/presenter_test.go** (新建)

- 测试目标: 覆盖数据展示层
- 关键测试场景:
    - BuildStatePayload: 空状态、有运行任务、有重试任务、完整状态
    - BuildIssuePayload: 运行中的问题、重试中的问题、问题不存在
    - BuildRefreshPayload: 正常返回
- 测试工具: 创建 OrchestratorState 测试数据

4. **internal/router/router_test.go** (新建)

- 测试目标: 覆盖路由配置
- 关键测试场景:
    - SetupRouter: 路由注册正确、所有端点可访问
    - BuildRouter: 测试模式路由器创建
- 测试工具: Gin路由测试、Mock组件

5. **internal/tracker/linear_test.go** (新建)

- 测试目标: 覆盖 Linear GraphQL 客户端
- 关键测试场景:
    - NewLinearClient: 客户端创建
    - FetchCandidateIssues: 正常返回、分页处理、错误处理
    - FetchIssuesByStates: 正常返回、空状态、错误处理
    - FetchIssueStatesByIDs: 批量查询、部分失败
    - LinearIssue.ToDomain: 字段转换、标签标准化、时间解析
    - doRequest: HTTP请求、GraphQL错误处理
- 测试工具: HTTP Server Mock、表驱动测试

6. **internal/tracker/github_test.go** (新建)

- 测试目标: 覆盖 GitHub REST API 客户端
- 关键测试场景:
    - NewGitHubClient: 客户端创建、repo格式解析
    - FetchCandidateIssues: 状态过滤、PR过滤、分页处理
    - FetchIssuesByStates: 多状态查询、PR过滤
    - FetchIssueStatesByIDs: 单个查询、批量查询、ID格式解析
    - githubIssue.toDomain: 状态label提取、默认状态映射
    - extractIssueNumber: 多种ID格式
    - isPullRequest: PR判断
- 测试工具: HTTP Server Mock、表驱动测试

## 实现细节

### Mock 设计

#### Mock Orchestrator

```
type MockOrchestrator struct {
    mu           sync.RWMutex
    state        *domain.OrchestratorState
    stateChanges chan struct{}
}

func NewMockOrchestrator() *MockOrchestrator {
    return &MockOrchestrator{
        state: &domain.OrchestratorState{
            Running:       make(map[string]*domain.RunningEntry),
            RetryAttempts: make(map[string]*domain.RetryEntry),
            Completed:     make(map[string]struct{}),
            CodexTotals:   &domain.CodexTotals{},
        },
        stateChanges: make(chan struct{}, 10),
    }
}

func (m *MockOrchestrator) GetState() *domain.OrchestratorState {
    m.mu.RLock()
    defer m.mu.RUnlock()
    // 返回状态副本
    return m.state
}
```

#### Mock Broadcaster

```
type MockBroadcaster struct {
    mu        sync.RWMutex
    clients   map[chan *common.SSEEvent]struct{}
    lastEvent *common.SSEEvent
}

func NewMockBroadcaster() *MockBroadcaster {
    return &MockBroadcaster{
        clients: make(map[chan *common.SSEEvent]struct{}),
    }
}

func (m *MockBroadcaster) Subscribe() chan *common.SSEEvent {
    m.mu.Lock()
    defer m.mu.Unlock()
    ch := make(chan *common.SSEEvent, 10)
    m.clients[ch] = struct{}{}
    return ch
}
```

### 测试工具函数

#### 创建测试 Issue

```
func createTestIssue(id, identifier, title, state string, priority int) *domain.Issue {
    now := time.Now()
    return &domain.Issue{
        ID:          id,
        Identifier:  identifier,
        Title:       title,
        State:       state,
        Priority:    &priority,
        CreatedAt:   &now,
        UpdatedAt:   &now,
        Labels:      []string{},
        BlockedBy:   []domain.BlockerRef{},
    }
}
```

#### 创建测试配置

```
func createTestConfig() *config.Config {
    cfg := config.DefaultConfig()
    cfg.Tracker.Kind = "mock"
    cfg.Workspace.Root = "/tmp/test-workspaces"
    cfg.Agent.MaxConcurrentAgents = 2
    return cfg
}
```

### 覆盖率目标

- orchestrator: 目标 80%+
- agent/prompt: 目标 90%+
- agent/runner: 目标 100%
- common: 目标 85%+
- config: 目标 80%+
- workspace: 目标 80%+
- server/handlers: 目标 75%+
- server/presenter: 目标 90%+
- router: 目标 70%+
- tracker/linear: 目标 70%+
- tracker/github: 目标 70%

## 注意事项

1. **资源清理**: 所有临时文件、目录、环境变量必须使用 defer 清理
2. **并发测试**: SSEBroadcaster 等并发组件需要专门的并发测试
3. **超时处理**: 钩子、网络请求等需要测试超时场景
4. **错误路径**: 每个函数的错误分支都需要测试覆盖
5. **边界条件**: 空值、nil、边界值等特殊场景需要测试
6. **Mock隔离**: 每个测试用例使用独立的 Mock 实例，避免相互影响
7. **表驱动测试**: 多场景测试必须使用表驱动模式
8. **t.Run组织**: 使用 t.Run() 组织相关测试用例，提高可读性