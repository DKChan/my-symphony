# FileTracker 文件结构规范

## 1. 目录结构

```
.sym/
├── config.yaml                 # 项目配置（由 symphony init 生成）
└── {task-identifier}/          # 每个任务一个目录
    ├── task.md                 # 状态索引文件
    ├── Planner/                # Planner 类子任务
    │   ├── P1-{name}.md
    │   ├── P2-{name}.md
    │   ├── P3-{name}.md
    │   ├── P4-{name}.md
    │   └── P5-{name}.md
    ├── Generator/              # Generator 类子任务
    │   ├── G1-{name}-v{n}.md   # 版本号用于迭代
    │   ├── G2-{name}-v{n}.md
    │   ├── G3-{name}-v{n}.md
    │   └── G4-{name}-v{n}.md
    │   └── ...
    └── Evaluator/              # Evaluator 类子任务
        ├── E1-{name}-v{n}.md
        └── ...
```

## 2. 状态索引文件 (task.md)

### 2.1 YAML Frontmatter 格式

```yaml
---
# 必填字段
id: string                      # 任务唯一ID（如 "SYM-001"）
title: string                   # 任务标题
status: string                  # 状态: backlog | in-progress | completed | needs-attention
phase: string                   # 当前阶段: planner | generator | evaluator | completed
iteration: integer              # 当前迭代次数（从1开始）
created: datetime               # 创建时间（RFC3339格式）
updated: datetime               # 最后更新时间（RFC3339格式）

# 可选字段
description: string             # 任务描述
max_iterations: integer         # 最大迭代次数（默认5）
labels: [string]                # 标签列表
---
```

### 2.2 Markdown 内容格式

```markdown
# Planner

- P1: {子任务名称} {状态标记}
- P2: {子任务名称} {状态标记}
- P3: {子任务名称} {状态标记}
- P4: {子任务名称} {状态标记}
- P5: {子任务名称} {状态标记}

# Generator

- G1: {子任务名称}-v{n} {状态标记}
- G2: {子任务名称}-v{n} {状态标记}
- G3: {子任务名称}-v{n} {状态标记}
- G4: {子任务名称}-v{n} {状态标记}

# Evaluator

- E1: {子任务名称}-v{n} {状态标记}
```

### 2.3 状态标记

| 标记 | 含义 | UI 映射 |
|------|------|---------|
| `✅` | 完成 | 完成（绿色）|
| `❌` | 失败 | 失败（红色）|
| `⏳` | 进行中 | 进行中（黄色）|
| `⬜` | 待开始 | 待开始（灰色）|

## 3. 子任务详情文件

### 3.1 YAML Frontmatter 格式

```yaml
---
# 必填字段
id: string                      # 子任务ID（如 "SYM-001-P1"）
parent: string                  # 父任务ID（如 "SYM-001"）
type: string                    # 类型: planner | generator | evaluator
name: string                    # 子任务名称（如 "需求澄清"）
version: integer                # 版本号（从1开始，用于迭代）
status: string                  # 状态: pending | in-progress | completed | failed
created: datetime               # 创建时间
updated: datetime               # 最后更新时间

# 可选字段
blocked_by: [string]            # 依赖的子任务ID列表
input_refs: [string]            # 输入引用（如架构设计文件）
output_ref: string              # 输出引用（如生成的代码路径）
error_message: string           # 失败时的错误信息
---
```

### 3.2 Markdown 内容结构

```markdown
## 任务描述

{子任务的详细描述}

## 输入

{输入依赖说明}

## 对话记录

### Turn {n} - {timestamp}

**{role}:**

{content}

---

## 输出

{产出结果}

## 反馈

{如果有审核反馈}
```

## 4. 子任务名称约定

### 4.1 Planner 子任务 (P1-P5)

| ID | 名称 | 说明 |
|----|------|------|
| P1 | 需求澄清 | PM Agent 与用户对话 |
| P2 | BDD规则 | QA Agent 生成 Gherkin |
| P3 | 领域建模 | Architect Agent 建立模型 |
| P4 | 架构设计 | Architect Agent 设计架构 |
| P5 | 接口设计 | Architect Agent 定义 API |

### 4.2 Generator 子任务 (G1-G4+)

| ID | 名称 | 说明 |
|----|------|------|
| G1 | BDD测试脚本 | QA Agent 转换 Gherkin |
| G2 | 集成测试 | QA Agent 生成集成测试 |
| G3 | 单元测试 | Dev Agent 生成单元测试 |
| G4 | 代码实现 | Dev Agent 实现代码 |
| G5+ | 代码实现（迭代）| 迭代修复版本 |

### 4.3 Evaluator 子任务 (E1+)

| ID | 名称 | 说明 |
|----|------|------|
| E1 | 评估验收 | 执行测试+代码审计 |
| E2+ | 评估验收（迭代）| 迭代评估版本 |

## 5. 状态流转规则

### 5.1 阶段流转

```
backlog → planner → generator → evaluator → completed
                              ↘ needs-attention (迭代上限)
```

### 5.2 子任务状态

```
pending → in-progress → completed
                      ↘ failed (触发迭代)
```

### 5.3 迭代创建规则

- 当 Evaluator 返回 `failed` 时：
  - 创建新的 G4-v{n+1} 子任务
  - 创建新的 E1-v{n+1} 子任务
  - iteration 递增
- 当 iteration 达到 max_iterations 时：
  - 父任务状态改为 `needs-attention`
  - 停止自动迭代

## 6. 文件命名规则

### 6.1 目录命名

- 使用任务 identifier（如 `SYM-001`）
- 只允许字母、数字、连字符
- 最大长度 64 字符

### 6.2 文件命名

- Planner 子任务：`P{n}-{name}.md`（无版本号）
- Generator/Evaluator 子任务：`G{n}-{name}-v{version}.md`
- name 使用中文或英文，空格用连字符替代
- version 从 1 开始

## 7. 示例

### 7.1 状态索引示例

```yaml
---
id: SYM-001
title: 用户认证功能
status: in-progress
phase: generator
iteration: 2
created: 2026-04-07T10:00:00Z
updated: 2026-04-07T14:30:00Z
max_iterations: 5
labels: [auth, security]
---
```

```markdown
# Planner

- P1: 需求澄清 ✅
- P2: BDD规则 ✅
- P3: 领域建模 ✅
- P4: 架构设计 ✅
- P5: 接口设计 ✅

# Generator

- G1: BDD测试脚本-v1 ✅
- G2: 集成测试-v1 ✅
- G3: 单元测试-v1 ✅
- G4: 代码实现-v1 ❌
- G4: 代码实现-v2 ⏳

# Evaluator

- E1: 评估验收-v1 ❌
- E1: 评估验收-v2 ⬜
```

### 7.2 子任务详情示例

```yaml
---
id: SYM-001-G4
parent: SYM-001
type: generator
name: 代码实现
version: 2
status: in-progress
created: 2026-04-07T14:30:00Z
updated: 2026-04-07T14:30:00Z
blocked_by: [SYM-001-G3]
input_refs: [SYM-001/P4-架构设计.md, SYM-001/P5-接口设计.md]
---
```

```markdown
## 任务描述

根据架构设计和接口定义实现用户认证代码。

## 输入

- 架构设计: [P4-架构设计]
- 接口定义: [P5-接口设计]
- 失败报告: [E1-评估验收-v1]

## 对话记录

### Turn 1 - 2026-04-07T14:30:00Z

**user:**

根据上一次评估的失败报告，修复以下问题：
1. 密码验证逻辑错误
2. Session 管理未实现

**assistant:**

好的，我将修复这些问题。首先分析失败报告...

---

## 输出

（代码实现结果）
```