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
  max_concurrent_agents: 10
  max_turns: 20
  max_retry_backoff_ms: 300000  # 5分钟最大重试退避

codex:
  command: "codex app-server"
  approval_policy: "suggest"  # 自动审批策略
  turn_timeout_ms: 3600000  # 1小时轮次超时
  read_timeout_ms: 5000
  stall_timeout_ms: 300000  # 5分钟停滞检测

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