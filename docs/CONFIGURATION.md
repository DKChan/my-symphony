# 配置验证规则

本文档描述配置验证规则和配置优先级。

## 配置来源优先级

配置按以下优先级加载（从高到低）：

1. **CLI 参数**（如 `-workflow`、`-port`）
2. **WORKFLOW.md YAML 前置内容**
3. **环境变量**（`$VAR_NAME` 格式）
4. **内置默认值**

**规则**: 新增配置项时，必须在 `config.go` 中定义默认值。

## 字段验证规则

配置验证在 `ValidateDispatchConfig` 中实现：

| 字段 | 规则 |
|------|------|
| `tracker.kind` | 必需，支持 `linear`、`github`、`mock` |
| `tracker.api_key` | 非 mock 类型必需，支持 `$VAR` 格式 |
| `tracker.project_slug` | Linear 类型必需 |
| `tracker.repo` | GitHub 类型必需，格式 `owner/repo` |
| `agent.kind` | 支持 `codex`、`claude`、`opencode` |
| `codex.command` | Codex 类型必需 |

## 环境变量引用

配置值支持环境变量引用，格式为 `$VAR_NAME`：

```yaml
tracker:
  api_key: $LINEAR_API_KEY  # 从环境变量 LINEAR_API_KEY 读取
```

解析时会自动替换为对应的环境变量值。

## WORKFLOW.md 配置示例

```yaml
---
tracker:
  kind: linear
  api_key: $LINEAR_API_KEY
  project_slug: MYPROJECT
  active_states:
    - Todo
    - In Progress
  terminal_states:
    - Done
    - Cancelled

agent:
  kind: codex

codex:
  command: codex app-server --port 8080

workspace:
  root: /workspaces
  cleanup_after_days: 7

hooks:
  after_create: ./scripts/setup.sh
  before_run: ./scripts/prepare.sh
  after_run: ./scripts/cleanup.sh

retry:
  max_attempts: 3
  max_backoff: 5m
---

# 提示模板

这里是 Markdown 格式的提示模板...
```