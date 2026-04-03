# 代码规范

本文档定义项目的代码风格和最佳实践。

## Go 代码风格

### 1. 命名规范

- 导出函数/类型使用 PascalCase
- 私有函数/变量使用 camelCase
- 接口名使用 `-er` 后缀（如 `Tracker`、`Runner`、`Loader`）

### 2. 错误处理

使用 `fmt.Errorf` 包装错误，保留上下文。错误消息使用小写开头，不以句号结尾：

```go
if err != nil {
    return nil, fmt.Errorf("failed to create workspace: %w", err)
}
```

### 3. 注释规范

- 包注释放在 `package` 语句之前
- 导出类型/函数必须有注释
- 注释以类型/函数名开头

```go
// Issue 表示标准化的问题记录
type Issue struct {
    // ID 是稳定的跟踪器内部 ID
    ID string `json:"id"`
}
```

### 4. Context 使用

所有长时间运行的操作必须接受 `context.Context`，使用 `context.WithTimeout` 设置超时：

```go
func (m *Manager) CreateForIssue(ctx context.Context, identifier string) (*domain.Workspace, error)
```

## 并发模式

### 1. 状态保护

使用 `sync.RWMutex` 保护共享状态：
- 读操作使用 `RLock/RUnlock`
- 写操作使用 `Lock/Unlock`

```go
type Orchestrator struct {
    mu    sync.RWMutex
    state *domain.OrchestratorState
}

func (o *Orchestrator) GetState() *domain.OrchestratorState {
    o.mu.RLock()
    defer o.mu.RUnlock()
    // ...
}
```

### 2. Goroutine 管理

使用 `context.Context` 控制 goroutine 生命周期，启动 goroutine 时传递 context。

## 日志规范

### 1. 结构化日志

使用 `key=value` 格式，包含操作结果：

```go
fmt.Printf("worker for %s completed successfully (turns: %d)\n", issue.Identifier, result.TurnCount)
```

### 2. 必须记录的事件

- 会话启动/结束
- 配置变更
- 重试调度
- 错误和超时