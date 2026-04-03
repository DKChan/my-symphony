# 扩展指南

本文档说明如何扩展 Symphony 的功能。

## 添加新的问题跟踪器

1. 在 `internal/tracker/` 创建新文件（如 `jira.go`）
2. 实现 `Tracker` 接口
3. 在 `tracker.go` 的 `NewTracker` 工厂函数中添加分支
4. 更新 `config.go` 添加必要的配置字段
5. 更新 `ValidateDispatchConfig` 添加验证逻辑

### Tracker 接口

```go
type Tracker interface {
    FetchCandidateIssues(ctx context.Context, activeStates []string) ([]*domain.Issue, error)
    FetchIssuesByStates(ctx context.Context, states []string) ([]*domain.Issue, error)
    FetchIssueStatesByIDs(ctx context.Context, ids []string) ([]*domain.Issue, error)
}
```

### 示例：添加 Jira 支持

```go
// internal/tracker/jira.go
package tracker

type JiraClient struct {
    client   *http.Client
    baseURL   string
    apiToken  string
    project   string
}

func NewJiraClient(baseURL, apiToken, project string) *JiraClient {
    return &JiraClient{
        client:   &http.Client{Timeout: 30 * time.Second},
        baseURL:  baseURL,
        apiToken: apiToken,
        project:  project,
    }
}

func (j *JiraClient) FetchCandidateIssues(ctx context.Context, activeStates []string) ([]*domain.Issue, error) {
    // 实现 Jira API 调用
}

// ... 实现其他接口方法
```

然后在 `tracker.go` 的 `NewTracker` 中添加：

```go
func NewTracker(cfg *config.TrackerConfig) (Tracker, error) {
    switch cfg.Kind {
    case "jira":
        return NewJiraClient(cfg.BaseURL, cfg.APIKey, cfg.Project), nil
    // ...
    }
}
```

## 添加新的代理类型

1. 在 `internal/agent/` 创建新文件
2. 实现 `Runner` 接口
3. 在 `runner.go` 的 `NewRunner` 工厂函数中添加分支
4. 如有需要，在 `config.go` 添加专用配置结构

### Runner 接口

```go
type Runner interface {
    RunAttempt(ctx context.Context, issue *domain.Issue, workspacePath string,
        attempt *int, promptTemplate string, callback EventCallback) (*RunAttemptResult, error)
}
```

## 添加新的钩子

1. 在 `config.go` 的 `HooksConfig` 结构添加字段
2. 在 `workspace/manager.go` 添加执行方法
3. 在适当的位置调用钩子（参考现有钩子）

### 钩子配置结构

```go
type HooksConfig struct {
    AfterCreate  string `yaml:"after_create"`
    BeforeRun    string `yaml:"before_run"`
    AfterRun     string `yaml:"after_run"`
    BeforeRemove string `yaml:"before_remove"`
}
```

### 钩子执行示例

```go
func (m *Manager) executeHook(ctx context.Context, name, script, workspacePath string) error {
    if script == "" {
        return nil
    }

    cmd := exec.CommandContext(ctx, "sh", "-c", script)
    cmd.Dir = workspacePath
    cmd.Env = append(os.Environ(),
        "SYMPHONY_WORKSPACE="+workspacePath,
        "SYMPHONY_HOOK="+name,
    )

    return cmd.Run()
}
```