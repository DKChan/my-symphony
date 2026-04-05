---
tracker:
  kind: mock
  active_states:
    - "Todo"
    - "In Progress"
  terminal_states:
    - "Done"
    - "Cancelled"

polling:
  interval_ms: 30000

workspace:
  root: "/tmp/symphony_workspaces"

agent:
  kind: "claude"
  max_concurrent_agents: 10
  max_turns: 20

claude:
  command: "claude"
  skip_permissions: true

server:
  port: 8080

---

# 巋试工作流

这是一个使用 mock tracker 的测试工作流。

## 问题信息

- **问题ID**: {{ issue.id }}
- **标题**: {{ issue.title }}
- **状态**: {{ issue.state }}
- **描述**: {{ issue.description }}

请处理这个问题。