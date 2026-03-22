---
tracker:
  kind: linear
  api_key: $LINEAR_API_KEY  # 使用环境变量
  project_slug: "MY-PROJECT"  # 替换为你的Linear项目标识
  active_states:
    - "Todo"
    - "In Progress"
  terminal_states:
    - "Done"
    - "Cancelled"
    - "Closed"

polling:
  interval_ms: 30000  # 30秒轮询间隔

workspace:
  root: "/tmp/symphony_workspaces"  # 工作空间根目录

hooks:
  # 创建新工作空间后执行的脚本
  after_create: |
    echo "创建新工作空间: $SYMPHONY_WORKSPACE"
    git clone https://github.com/your-org/your-repo.git .
  # 每次运行前执行的脚本
  before_run: |
    echo "准备运行代理..."
    git pull origin main
  # 每次运行后执行的脚本
  after_run: |
    echo "代理运行完成"
  timeout_ms: 60000

agent:
  # 代理类型: codex（默认）、claude、opencode
  # - codex: 使用 Codex app-server JSON-RPC 协议（需要 codex 配置段）
  # - claude: 使用 Claude Code CLI（需要 claude 配置段）
  # - opencode: 使用 OpenCode CLI（需要 opencode 配置段）
  kind: "claude"
  # 全局自定义命令（可选，优先级低于各代理专用配置）
  # command: "claude --print --output-format=stream-json --model opus-4"
  max_concurrent_agents: 10  # 全局最大并发代理数
  max_turns: 20              # 每个代理的最大对话轮次
  max_retry_backoff_ms: 300000  # 最大重试退避时间（毫秒），5分钟
  # 按问题状态的并发限制（可选），例如 In Progress 状态最多3个
  max_concurrent_agents_by_state:
    "In Progress": 3
  # 轮次超时（毫秒），仅用于非 codex agent（claude/opencode）
  # turn_timeout_ms: 3600000  # 默认1小时

# Claude Code CLI 配置（当 agent.kind: "claude" 时使用）
# CLI 默认命令: claude --print --output-format=stream-json --dangerously-skip-permissions --no-session-persistence
claude:
  command: "claude"           # CLI 命令（默认: claude）
  skip_permissions: true      # 跳过权限检查（默认: true）
  # extra_args:               # 额外命令行参数（会追加到默认参数之后）
  #   - "--model"
  #   - "opus-4"
  #   - "--max-tokens"
  #   - "4096"
  #   - "--temperature"
  #   - "0.7"

# OpenCode CLI 配置（当 agent.kind: "opencode" 时使用）
# CLI 默认命令: opencode run "<prompt>" --output-format json
opencode:
  command: "opencode"         # CLI 命令（默认: opencode）
  # extra_args:               # 额外命令行参数（会追加到默认参数之后）
  #   - "--model"
  #   - "gpt-4"
  #   - "--provider"
  #   - "openai"

# Codex 代理专用配置（当 agent.kind: "codex" 时使用）
codex:
  command: "codex app-server"  # Codex 启动命令
  approval_policy: "suggest"  # 自动审批策略: suggest（建议）、auto（自动批准）、manual（手动）
  turn_timeout_ms: 3600000     # 轮次超时（毫秒），1小时
  read_timeout_ms: 5000       # 读取超时（毫秒）
  stall_timeout_ms: 300000    # 停滞检测超时（毫秒），5分钟无输出则认为停滞

server:
  port: 8080  # HTTP服务器端口

---

# 工作任务

你正在处理一个来自Linear的问题。请按照以下步骤执行：

## 问题信息

- **问题ID**: {{ issue.id }}
- **问题标识**: {{ issue.identifier }}
- **标题**: {{ issue.title }}
- **状态**: {{ issue.state }}
- **描述**: {{ issue.description }}

## 工作指导

1. 仔细阅读问题描述，理解需要完成的任务
2. 检查代码库，了解当前的实现状态
3. 制定解决方案，考虑最佳实践
4. 实现变更，确保代码质量
5. 编写或更新测试
6. 提交变更并创建Pull Request

## 注意事项

- 遵循项目的编码规范
- 确保所有测试通过
- 添加必要的文档注释
- 如果遇到阻塞，在Linear中添加评论说明

## 尝试次数

{% if attempt %}
这是第 {{ attempt }} 次尝试。如果之前的尝试失败，请分析原因并尝试不同的方法。
{% endif %}