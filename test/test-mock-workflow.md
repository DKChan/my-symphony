---
tracker:
  kind: mock
  active_states: [Todo, In Progress]
  terminal_states: [Done, Cancelled]
  mock_issues:
  - id: "1"
    identifier: "TEST-1"
    title: "测试任务1 - 创建用户API"
    state: Todo
    priority: 1
    labels: [feature, backend]
  - id: "2"
    identifier: "TEST-2"
    title: "测试任务2 - 修复登录Bug"
    state: In Progress
    priority: 2
    labels: [bug, urgent]
  - id: "3"
    identifier: "TEST-3"
    title: "测试任务3 - 完成文档"
    state: Todo
    labels: [docs]
  - id: "4"
    identifier: "TEST-4"
    title: "测试任务4 - 性能优化"
    state: Done
    labels: [optimization]
  - id: "5"
    identifier: "TEST-5"
    title: "测试任务5 - 优化用户体验"
    description: "让系统更快。"  # ← 模糊需求：缺少具体指标
    state: Todo
    priority: 3
    labels: [ux]

polling:
  interval_ms: 3000

workspace:
  root: "/tmp/test_symphony_workspaces"

agent:
  kind: claude
  max_concurrent_agents: 2
  max_turns: 3

claude:
  skip_permissions: true

hooks:
  before_run: 'echo "开始执行: {{.Identifier}} - {{.Title}}"'
  after_run: 'echo "完成执行: {{.Identifier}}"'

server:
  port: 8080
---

# 测试工作流

请完成这个测试任务，输出 "Hello from Mock Tracker Test"

## 测试目标

1. 验证 Mock Tracker 正常工作
2. 验证状态流转逻辑
3. 验证编排调度功能
4. 验证需求澄清能力

## 验证点 (TDD - 红色阶段)

### 验证clarification反问

**前置条件**：TEST-5（模糊需求）处于 `Todo` 状态

**期望行为**：
- Agent 接收到 TEST-5 后，应检测到需求描述模糊（"让系统更快"无具体指标）
- Agent 不应直接执行，而应先反问澄清
- 期望的澄清问题应包含以下关键词之一：
  - "什么指标" / "which metric"
  - "具体定义" / "specifically"
  - "如何衡量" / "how to measure"
  - "多少" / "how much"
  - "从 X 到 Y" / "from X to Y"

**验证方法**：
1. 检查 agent 工作空间中的对话记录
2. 或检查 agent 输出的首条消息是否包含澄清问题

**当前状态**：❌ 未实现（测试会失败）

---

### 验证清晰需求的正常执行

**前置条件**：TEST-1（清晰需求）处于 `Todo` 状态

**期望行为**：
- Agent 识别需求清晰（标题+描述完整）
- Agent 直接开始执行，不发送澄清问题

**验证方法**：
1. 检查 agent 是否输出了预期的执行结果
2. 验证工作空间中是否产生了预期的文件/改动

**当前状态**：✅ 应正常工作
