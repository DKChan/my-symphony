# Story 8.2: FileClient 实现

Status: done

## Story

As a 系统架构师,
I want 实现 FileClient 满足 Tracker 接口,
so that 系统可以用文件系统作为 Tracker 后端.

## Acceptance Criteria

1. **AC1: Tracker 接口实现**
   - Given tracker.Kind 配置为 "file"
   - When NewTracker 创建 Tracker
   - Then 返回 FileClient 实例
   - And FileClient 实现所有 Tracker 接口方法

2. **AC2: 任务创建**
   - Given CreateTask 被调用
   - When 创建新任务
   - Then 创建 `.sym/{id}/task.md` 状态索引
   - And 创建 Planner/Generator/Evaluator 目录
   - And 创建初始子任务详情文件

3. **AC3: 任务查询**
   - Given GetTask/ListTasksByState 被调用
   - When 查询任务
   - Then 解析状态索引文件的 YAML frontmatter
   - And 解析 markdown 列表获取子任务状态
   - And 返回 domain.Issue 结构

4. **AC4: 状态更新**
   - Given UpdateStage 被调用
   - When 更新任务状态
   - Then 更新状态索引文件的 frontmatter
   - And 更新对应子任务状态标记
   - And 更新 updated 时间戳

5. **AC5: 对话追加**
   - Given AppendConversation 被调用
   - When 追加对话记录
   - Then 写入对应子任务详情文件
   - And 添加时间戳和角色标记

**FRs covered:** FR61

## Tasks / Subtasks

- [ ] Task 1: 实现 FileClient 结构 (AC: 1)
  - [ ] 1.1 创建 internal/tracker/file.go
  - [ ] 1.2 实现 NewFileClient 工厂函数
  - [ ] 1.3 更 NewTracker 支持 "file" kind

- [ ] Task 2: 实现任务创建方法 (AC: 2)
  - [ ] 2.1 实现 CreateTask
  - [ ] 2.2 实现 CreateSubTask

- [ ] Task 3: 实现任务查询方法 (AC: 3)
  - [ ] 3.1 实现 GetTask
  - [ ] 3.2 实现 ListTasksByState
  - [ ] 3.3 实现 FetchCandidateIssues

- [ ] Task 4: 实现状态更新方法 (AC: 4)
  - [ ] 4.1 实现 UpdateStage
  - [ ] 4.2 实现 GetStageState

- [ ] Task 5: 实现对话记录方法 (AC: 5)
  - [ ] 5.1 实现 AppendConversation
  - [ ] 5.2 实现 GetConversationHistory

## Dev Notes

### Tracker 接口映射

| Tracker 方法 | FileClient 实现 |
|--------------|-----------------|
| CreateTask | 创建目录 + task.md + 子任务文件 |
| CreateSubTask | 创建子任务详情文件 |
| GetTask | 解析 task.md frontmatter |
| ListTasksByState | 扫描 .sym/*/task.md |
| UpdateStage | 更新 task.md frontmatter |
| GetStageState | 解析 task.md frontmatter |
| AppendConversation | 追加到子任务详情文件 |
| GetConversationHistory | 解析子任务详情文件 |

### 文件解析策略

使用 `gopkg.in/yaml.v3` 解析 frontmatter:
- 分割 `---` 边界
- 解析 YAML 元数据
- 解析 markdown 内容

### Project Structure Notes

**新增文件**:
- `internal/tracker/file.go` - FileClient 实现
- `internal/tracker/file_test.go` - 单元测试

**修改文件**:
- `internal/tracker/tracker.go` - NewTracker 支持 "file"
- `internal/config/config.go` - Tracker.Kind 支持 "file"

### References

- [Source: internal/tracker/tracker.go] - Tracker 接口定义
- [Source: internal/tracker/beads.go] - BeadsClient 实现参考
- [Source: _bmad-output/planning-artifacts/epics-v2.md#L883-L912] - Story 定义

## Change Log

- 2026-04-07: Story 创建