# Story 8.4: Web UI 文件读取适配

Status: done

## Story

As a 开发者,
I want Web UI 能读取 FileTracker 的文件格式,
so that 可以在页面展示任务状态.

## Acceptance Criteria

1. **AC1: 任务列表展示**
   - Given tracker.Kind 为 "file"
   - When Web UI 加载任务列表
   - Then 扫描 .sym/*/task.md 文件
   - And 解析 YAML frontmatter 获取状态
   - And 展示任务列表

2. **AC2: 三类任务看板展示**
   - Given 选择某个任务
   - When 展示任务详情
   - Then 解析 markdown 列表获取子任务状态
   - And 展示 Planner/Generator/Evaluator 三类任务
   - And 显示进度标记 (✅ ❌ ⏳ ⬜)

3. **AC3: 迭代历史展示**
   - Given 任务有迭代版本
   - When 展示迭代历史
   - Then 显示所有版本文件
   - And 显示各版本失败原因
   - And 显示当前迭代进度

4. **AC4: 对话历史展示**
   - Given 用户查看子任务详情
   - When 点击对话历史
   - Then 解析子任务详情文件
   - And 展示对话记录

**FRs covered:** FR53, FR54

## Tasks / Subtasks

- [ ] Task 1: 实现文件状态读取 API (AC: 1)
  - [ ] 1.1 Dashboard 组件支持 file tracker
  - [ ] 1.2 扫描 .sym 目录
  - [ ] 1.3 解析 task.md 文件

- [ ] Task 2: 实现三类任务看板展示 (AC: 2)
  - [ ] 2.1 解析 markdown 列表
  - [ ] 2.2 渲染三类任务状态
  - [ ] 2.3 状态标记映射

- [ ] Task 3: 实现迭代历史展示 (AC: 3)
  - [ ] 3.1 扫描版本文件
  - [ ] 3.2 展示迭代进度
  - [ ] 3.3 展示失败报告

- [ ] Task 4: 实现对话历史展示 (AC: 4)
  - [ ] 4.1 解析子任务详情文件
  - [ ] 4.2 渲染对话记录

## Dev Notes

### 文件解析逻辑

Dashboard 组件需要新增 file tracker 支持:

```go
// dashboard.go
func (d *Dashboard) loadTasksFromFileTracker() []*TaskInfo {
    files, _ := os.ReadDir(".sym")
    for _, f := range files {
        taskMd := filepath.Join(".sym", f.Name(), "task.md")
        data := parseTaskMd(taskMd)
        // ...
    }
}
```

### 状态标记映射

| Markdown 标记 | UI 显示 |
|---------------|---------|
| ✅ | 完成 (绿色) |
| ❌ | 失败 (红色) |
| ⏳ | 进行中 (黄色) |
| ⬜ | 待开始 (灰色) |

### Project Structure Notes

**修改文件**:
- `internal/server/components/dashboard.go` - 文件读取逻辑
- `internal/server/presenter/presenter.go` - 文件解析

### References

- [Source: internal/server/components/dashboard.go] - 现有 Dashboard 实现
- [Source: _bmad-output/planning-artifacts/epics-v2.md#L944-L972] - Story 定义

## Change Log

- 2026-04-07: Story 创建