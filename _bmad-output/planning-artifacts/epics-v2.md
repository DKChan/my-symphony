---
stepsCompleted: ["step-01-validate-prerequisites", "step-02-design-epics", "step-03-create-stories"]
inputDocuments:
  - _bmad-output/planning-artifacts/prd-v2.md
  - _bmad-output/planning-artifacts/architecture-v2.md
workflowType: 'epics'
project_name: 'my-symphony'
user_name: 'DK'
date: '2026-04-05'
status: 'complete'
---

# my-symphony - Epic Breakdown (v2.0)

## Overview

本文档提供 my-symphony v2.0 的完整 Epic 和 Story 分解，基于 P-G-E (Planner-Generator-Evaluator) 架构。

## Requirements Inventory

### Functional Requirements Summary

| 领域 | FR 数量 |
|------|---------|
| 初始化与配置管理 | 8 |
| Planner 模块 | 7 |
| Generator 模块 | 7 |
| Evaluator 模块 | 6 |
| 迭代机制 | 5 |
| 任务管理 | 6 |
| 外部集成 | 3 |
| **Total** | **42** |

---

## Epic List

| Epic | 名称 | Stories | FRs Covered |
|------|------|---------|-------------|
| Epic 1 | 项目初始化与配置 | 4 | FR1-8 |
| Epic 2 | Planner 模块实现 | 6 | FR10-16 |
| Epic 3 | Generator 模块实现 | 6 | FR20-26 |
| Epic 4 | Evaluator 模块实现 | 6 | FR30-35 |
| Epic 5 | 迭代回流机制 | 5 | FR40-44 |
| Epic 6 | Beads 任务结构适配 | 4 | FR50-55, FR60-62 |
| Epic 7 | Web UI 适配 | 4 | FR50-55 |
| **Total** | | **35** | |

---

## Epic 1: 项目初始化与配置

简化现有初始化流程，适配新的 Harness 架构。

### Story 1.1: CLI init 交互式初始化

**As a** 开发者
**I want** 通过 CLI 交互式初始化项目配置
**So that** 我可以快速配置 Symphony 项目

**Acceptance Criteria:**

```gherkin
Given 用户在项目根目录执行 `symphony init`
When CLI 启动交互式问答
Then 用户可以配置项目名称
And 用户可以配置 Tracker 类型 (Beads)
And 用户可以配置 BMAD Agent 启用状态
And 用户可以配置最大迭代次数
And 系统生成 `.sym/config.yaml` 配置文件
```

**FRs covered:** FR1, FR2, FR3

---

### Story 1.2: 配置验证与管理

**As a** 开发者
**I want** 系统在启动时验证配置有效性
**So that** 我可以确保配置正确后再启动服务

**Acceptance Criteria:**

```gherkin
Given 用户执行 `symphony start`
When 系统读取 `.sym/config.yaml`
Then 系统验证 BMAD Agent 可用性
And 系统验证 Beads CLI 可用性
And 系统验证配置格式正确

Given 配置验证失败
When 系统检测到无效配置
Then 返回错误码 `config.invalid: <具体原因>`
And 提示用户修改配置
```

**FRs covered:** FR4, FR5, FR6, FR7, FR8

---

### Story 1.3: Beads Tracker 集成

**As a** 开发者
**I want** 系统与 Beads tracker 集成以获取和更新任务
**So that** 我可以使用本地 CLI tracker 管理任务

**Acceptance Criteria:**

```gherkin
Given 服务启动时
When 系统检查 Beads CLI 可用性
Then 调用 `CheckAvailability()` 验证
And 不可用时返回错误 `tracker.unavailable`

Given Beads CLI 可用
When 系统需要获取任务
Then 调用 `GetTask()` 获取任务详情
```

**FRs covered:** FR8, FR61

---

### Story 1.4: CLI start 启动后台服务

**As a** 开发者
**I want** 通过 CLI start 启动后台服务
**So that** 我可以让 Symphony 在后台监控和执行任务

**Acceptance Criteria:**

```gherkin
Given 配置验证通过
When 用户执行 `symphony start`
Then Harness Orchestrator 初始化
And Web 服务器在配置端口启动
And 任务监控开始

Given 服务已启动
When 用户访问 `http://localhost:<port>`
Then 显示 Web 看板页面
```

**FRs covered:** NFR4, NFR5

---

## Epic 2: Planner 模块实现

实现需求理解与规划功能，产出不可变的规划文档。

### Story 2.1: BMAD Agent 调用框架

**As a** 系统架构师
**I want** 系统能调用 BMAD Agent
**So that** 可以利用专家 Agent 完成特定任务

**Acceptance Criteria:**

```gherkin
Given 系统需要调用 BMAD Agent
When 调用 `AgentCaller.Call()`
Then 系统构建 Agent 输入参数
And 系统 BMAD Agent CLI 执行
And 系统解析 Agent 输出

Given Agent 执行超时
When 超过配置的超时时间
Then 返回 `agent.timeout` 错误
```

**FRs covered:** NFR1, NFR3

---

### Story 2.2: 需求澄清 (人工参与)

**As a** 开发者
**I want** 系统调用 BMAD PM Agent 进行需求澄清
**So that** AI 能理解我的需求意图

**Acceptance Criteria:**

```gherkin
Given 一个新任务进入 Planner 阶段
When 阶段开始执行
Then 系统调用 BMAD PM Agent
And PM Agent 返回澄清问题
And Web 页面显示问题供用户回答

Given 用户提交回答
When 系统接收回答
Then 继续澄清对话或确认需求明确
And 这是唯一的人工参与节点
```

**FRs covered:** FR10, FR11

---

### Story 2.3: BDD 规则生成

**As a** 系统用户
**I want** 系统调用 BMAD QA Agent 生成 BDD 规则
**So that** 我可以快速获得验收标准

**Acceptance Criteria:**

```gherkin
Given 需求澄清完成
When Planner 进入 BDD 规则生成阶段
Then 系统调用 BMAD QA Agent
And QA Agent 返回 Gherkin 格式的 BDD 规则
And 规则保存到任务上下文
```

**FRs covered:** FR12

---

### Story 2.4: 领域建模

**As a** 系统用户
**I want** 系统调用 BMAD Architect Agent 进行领域建模
**So that** 可以建立清晰的领域模型

**Acceptance Criteria:**

```gherkin
Given BDD 规则已生成
When Planner 进入领域建模阶段
Then 系统调用 BMAD Architect Agent
And Architect Agent 返回领域模型描述
And 模型保存到任务上下文
```

**FRs covered:** FR13

---

### Story 2.5: 架构设计

**As a** 系统用户
**I want** 系统调用 BMAD Architect Agent 进行架构设计
**So that** 可以获得技术架构方案

**Acceptance Criteria:**

```gherkin
Given 领域模型已建立
When Planner 进入架构设计阶段
Then 系统调用 BMAD Architect Agent
And Architect Agent 返回架构设计文档
And 文档保存到任务上下文
```

**FRs covered:** FR14

---

### Story 2.6: 接口设计

**As a** 系统用户
**I want** 系统调用 BMAD Architect Agent 进行接口设计
**So that** 可以获得 API 接口定义

**Acceptance Criteria:**

```gherkin
Given 架构设计完成
When Planner 进入接口设计阶段
Then 系统调用 BMAD Architect Agent
And Architect Agent 返回 API 接口定义
And 定义保存到任务上下文

Given Planner 所有阶段完成
When 规划产出完成
Then 产出标记为不可变
And 后续不可修改
```

**FRs covered:** FR15, FR16

---

## Epic 3: Generator 模块实现

实现测试编码与代码生成功能，支持并行执行和迭代修复。

### Story 3.1: Generator 调度器

**As a** 系统架构师
**I want** Generator 能调度多个子任务
**So that** 可以并行执行测试编码

**Acceptance Criteria:**

```gherkin
Given Planner 产出完成
When Generator 开始执行
Then Phase 1 (测试编码) 并行启动
And G1, G2, G3 同时开始

Given Phase 1 所有任务完成
When 最后一个测试脚本完成
Then Phase 2 (代码实现) 开始
And G4 顺序执行
```

**FRs covered:** FR20, FR25

---

### Story 3.2: BDD 测试脚本生成

**As a** 系统用户
**I want** 系统调用 BMAD QA Agent 生成 BDD 测试脚本
**So that** BDD 规则可以转化为可执行测试

**Acceptance Criteria:**

```gherkin
Given Planner BDD 规则可用
When Generator G1 开始执行
Then 系统调用 BMAD QA Agent
And QA Agent 将 Gherkin 规则转为可执行测试代码
And 测试脚本保存到工作目录
```

**FRs covered:** FR21

---

### Story 3.3: 集成测试生成

**As a** 系统用户
**I want** 系统调用 BMAD QA Agent 生成集成测试
**So that** 可以验证组件间交互

**Acceptance Criteria:**

```gherkin
Given Planner 接口定义可用
When Generator G2 开始执行
Then 系统调用 BMAD QA Agent
And QA Agent 生成集成测试代码
And 测试代码保存到工作目录
```

**FRs covered:** FR22

---

### Story 3.4: 单元测试生成

**As a** 系统用户
**I want** 系统调用 BMAD Dev Agent 生成单元测试
**So that** 可以实现 TDD 流程

**Acceptance Criteria:**

```gherkin
Given Planner 架构设计可用
When Generator G3 开始执行
Then 系统调用 BMAD Dev Agent
And Dev Agent 生成单元测试代码
And 测试代码保存到工作目录
```

**FRs covered:** FR23

---

### Story 3.5: 代码实现

**As a** 系统用户
**I want** 系统调用 BMAD Dev Agent 实现代码
**So that** 需求可以转化为可运行的代码

**Acceptance Criteria:**

```gherkin
Given G1, G2, G3 全部完成
When Generator G4 开始执行
Then 系统调用 BMAD Dev Agent
And Dev Agent 实现功能代码
And 代码通过所有测试

Given 迭代修复场景
When Generator 收到失败报告
Then Dev Agent 根据报告修复代码
And 创建新的 G 子任务
```

**FRs covered:** FR24, FR26

---

### Story 3.6: 并行执行控制

**As a** 系统架构师
**I want** Generator 能正确控制并行执行
**So that** 子任务可以安全地并行运行

**Acceptance Criteria:**

```gherkin
Given Generator Phase 1 开始
When 启动 G1, G2, G3
Then 使用 goroutine 并行执行
And 使用 sync.WaitGroup 等待完成

Given 任一并行任务失败
When 检测到失败
Then 记录失败信息
And 等待其他任务完成
```

**FRs covered:** NFR8

---

## Epic 4: Evaluator 模块实现

实现质量验证功能，评估代码实现质量。

### Story 4.1: Evaluator 调度器

**As a** 系统架构师
**I want** Evaluator 能调度多个评估任务
**So that** 可以全面验证代码质量

**Acceptance Criteria:**

```gherkin
Given Generator 代码实现完成
When Evaluator 开始执行
Then 依次执行 E1, E2, E3, E4
And 收集所有评估结果
And 生成综合报告
```

**FRs covered:** FR30

---

### Story 4.2: BDD 验收执行

**As a** 系统用户
**I want** 系统执行 BDD 验收测试
**So that** 可以验证需求是否满足

**Acceptance Criteria:**

```gherkin
Given BDD 测试脚本可用
When Evaluator E1 开始执行
Then 系统运行 BDD 测试
And 记录测试结果
And 报告失败用例
```

**FRs covered:** FR30

---

### Story 4.3: TDD 验收执行

**As a** 系统用户
**I want** 系统执行 TDD 验收测试
**So that** 可以验证代码单元是否正确

**Acceptance Criteria:**

```gherkin
Given 单元测试代码可用
When Evaluator E2 开始执行
Then 系统运行单元测试
And 记录测试结果
And 报告失败用例
```

**FRs covered:** FR31

---

### Story 4.4: 代码审计

**As a** 系统用户
**I want** 系统调用 BMAD Code Review Agent 进行代码审计
**So that** 可以获得代码质量反馈

**Acceptance Criteria:**

```gherkin
Given 代码实现完成
When Evaluator E3 开始执行
Then 系统调用 BMAD Code Review Agent
And Agent 审查代码质量
And 返回审计报告

Given 审计发现问题
When Agent 返回报告
Then 记录问题列表
And 作为失败报告的一部分
```

**FRs covered:** FR32

---

### Story 4.5: 代码风格评审

**As a** 系统用户
**I want** 系统调用 BMAD Editorial Review Agent 进行风格评审
**So that** 可以保持代码风格一致

**Acceptance Criteria:**

```gherkin
Given 代码实现完成
When Evaluator E4 开始执行
Then 系统调用 BMAD Editorial Review Agent
And Agent 检查代码风格
And 返回风格报告
```

**FRs covered:** FR33

---

### Story 4.6: 失败报告生成

**As a** 系统架构师
**I want** Evaluator 生成失败报告
**So that** Generator 可以根据报告修复代码

**Acceptance Criteria:**

```gherkin
Given Evaluator 执行完成
When 存在失败项
Then 系统生成失败报告
And 报告包含所有失败项详情
And 报告通过对话上下文传递给 Generator

Given Evaluator 执行完成
When 所有项通过
Then 标记任务为完成状态
```

**FRs covered:** FR34, FR35

---

## Epic 5: 迭代回流机制

实现自动迭代修复机制，处理评估失败场景。

### Story 5.1: 迭代计数与管理

**As a** 系统架构师
**I want** 系统能正确管理迭代次数
**So that** 可以限制无限迭代

**Acceptance Criteria:**

```gherkin
Given 任务开始执行
When 初始化迭代状态
Then 迭代次数设为 1
And 最大迭代次数从配置读取 (默认 5)

Given 迭代发生
When Generator 开始修复代码
Then 迭代次数 +1
And 记录迭代历史
```

**FRs covered:** FR40

---

### Story 5.2: 迭代上限处理

**As a** 系统架构师
**I want** 迭代达到上限时转人工处理
**So that** 可以避免无限循环

**Acceptance Criteria:**

```gherkin
Given 迭代次数 = max_iterations
When Evaluator 再次返回失败
Then 任务状态流转到 "待人工处理"
And 系统提示用户迭代已达上限
And 建议用户检查需求理解是否正确
```

**FRs covered:** FR41

---

### Story 5.3: 迭代子任务创建

**As a** 系统架构师
**I want** 每次迭代创建新的子任务
**So that** 可以追踪迭代历史

**Acceptance Criteria:**

```gherkin
Given 迭代发生
When Generator 开始修复
Then 创建新的 Generator 子任务 (如 G5)
And 创建新的 Evaluator 子任务 (如 E2)
And 新子任务依赖前一次迭代的失败报告
```

**FRs covered:** FR42

---

### Story 5.4: 代码修复范围限制

**As a** 系统架构师
**I want** 迭代只修复代码不修改规划
**So that** Planner 产出保持稳定

**Acceptance Criteria:**

```gherkin
Given 迭代修复场景
When Generator 收到失败报告
Then Generator 只修改代码实现
And 不调用任何 Planner Agent
And 不修改 BDD 规则、架构设计、接口定义
```

**FRs covered:** FR43

---

### Story 5.5: 迭代进度展示

**As a** 开发者
**I want** 在 Web 页面查看迭代进度
**So that** 我可以了解当前迭代状态

**Acceptance Criteria:**

```gherkin
Given 任务正在迭代
When 用户查看任务详情
Then 显示当前迭代次数 (如 2/5)
And 显示历次迭代的失败原因
And 显示当前迭代进度
```

**FRs covered:** FR44

---

## Epic 6: Beads 任务结构适配

适配 Beads 子任务结构，支持三类子任务和迭代任务。

### Story 6.1: 三类子任务结构

**As a** 系统架构师
**I want** Beads 支持三类子任务
**So that** 可以区分 Planner/Generator/Evaluator 任务

**Acceptance Criteria:**

```gherkin
Given 创建新任务
When 系统初始化任务结构
Then 创建 Planner 类子任务 (P1-P5)
And 创建 Generator 类子任务 (G1-G4)
And 创建 Evaluator 类子任务 (E1)
And 每个子任务标记所属类别
```

**FRs covered:** FR51

---

### Story 6.2: 迭代任务创建与依赖

**As a** 系统架构师
**I want** 迭代时正确创建新子任务
**So that** 可以追踪迭代历史

**Acceptance Criteria:**

```gherkin
Given 迭代发生
When 创建新子任务
Then 创建新的 Generator 子任务 (Gn+1)
And 创建新的 Evaluator 子任务 (En+1)
And 设置子任务依赖关系
And 子任务编号递增
```

**FRs covered:** FR42, FR52

---

### Story 6.3: 任务状态流转适配

**As a** 系统架构师
**I want** 任务状态正确流转
**So that** 可以追踪整体进度

**Acceptance Criteria:**

```gherkin
Given Planner 完成
When 所有 P1-P5 完成
Then 父任务状态更新为 "Planner 完成"
And Generator 开始执行

Given Generator 完成
When G4 完成
Then 父任务状态更新为 "Generator 完成"
And Evaluator 开始执行

Given Evaluator 完成且通过
When 所有 E 项通过
Then 父任务状态更新为 "完成"
And 触发 Git 提交

Given Evaluator 完成且有失败
When 存在失败项
Then 触发迭代修复
或 达到上限转人工处理
```

**FRs covered:** FR53

---

### Story 6.4: 任务状态持久化

**As a** 系统架构师
**I want** 任务状态持久化到 Beads
**So that** 服务崩溃后可以恢复

**Acceptance Criteria:**

```gherkin
Given 子任务状态变化
When 状态更新
Then 调用 Beads API 更新任务状态
And 保存迭代次数到 Custom 字段
And 保存当前阶段到 Custom 字段

Given 服务重启
When 恢复进行中任务
Then 从 Beads 读取任务状态
And 恢复迭代次数
And 继续执行
```

**FRs covered:** NFR4, FR62

---

## Epic 7: Web UI 适配

适配 Web 界面，展示三类任务和迭代进度。

### Story 7.1: 三类任务看板展示

**As a** 开发者
**I want** 在 Web 页面看到三类任务看板
**So that** 我可以清晰了解任务结构

**Acceptance Criteria:**

```gherkin
Given 用户访问 Web 看板
When 页面加载完成
Then 显示三类任务区域
And Planner 区域显示 P1-P5 状态
And Generator 区域显示 G1-Gn 状态
And Evaluator 区域显示 E1-En 状态
```

**FRs covered:** FR53

---

### Story 7.2: 迭代进度展示

**As a** 开发者
**I want** 在 Web 页面看到迭代进度
**So that** 我可以了解当前迭代状态

**Acceptance Criteria:**

```gherkin
Given 任务正在迭代
When 用户查看任务详情
Then 显示迭代进度条 (如 2/5)
And 显示历次迭代的失败摘要
And 显示当前迭代状态
```

**FRs covered:** FR44

---

### Story 7.3: 失败报告展示

**As a** 开发者
**I want** 在 Web 页面查看失败报告
**So that** 我可以了解失败原因

**Acceptance Criteria:**

```gherkin
Given 任务迭代失败
When 用户点击失败详情
Then 显示完整失败报告
And 显示失败的测试用例
And 显示代码审计问题
And 显示风格评审问题
```

**FRs covered:** FR54

---

### Story 7.4: 实时状态更新

**As a** 开发者
**I want** Web 页面实时更新任务状态
**So that** 我可以看到最新进展

**Acceptance Criteria:**

```gherkin
Given 用户在看板页面
When 任务状态变化
Then 通过 SSE 推送更新
And 页面实时刷新任务状态
And 显示状态变化动画
```

**FRs covered:** NFR9

---

## Epic 8: FileTracker 实现

提供零依赖的文件系统 Tracker 实现，减少第三方工具依赖。

### Story 8.1: FileTracker 文件结构设计

**As a** 系统架构师
**I want** 设计 FileTracker 的文件存储结构
**So that** 可以用文件系统管理任务状态和详情

**Acceptance Criteria:**

```gherkin
Given 一个新需求任务 SYM-001
When 创建任务文件结构
Then 生成 .sym/SYM-001/task.md 状态索引文件
And 生成 Planner/Generator/Evaluator 子任务目录
And 状态索引使用 YAML frontmatter 格式
And 子任务详情文件按类别组织
```

**FRs covered:** NFR4, FR60

---

### Story 8.2: FileClient 实现

**As a** 系统架构师
**I want** 实现 FileClient 满足 Tracker 接口
**So that** 系统可以用文件系统作为 Tracker 后端

**Acceptance Criteria:**

```gherkin
Given tracker.Kind 配置为 "file"
When NewTracker 创建 Tracker
Then 返回 FileClient 实例
And FileClient 实现所有 Tracker 接口方法
And CreateTask 创建文件结构
And GetTask 从文件读取状态
And UpdateStage 更新状态索引文件
And AppendConversation 写入子任务详情文件
```

**FRs covered:** FR61

---

### Story 8.3: 并发写入安全

**As a** 系统架构师
**I want** FileTracker 支持并发写入安全
**So that** 多个子 agent 同时完成时不会产生文件冲突

**Acceptance Criteria:**

```gherkin
Given 调度器作为唯一写入者
When 子 agent 完成任务
Then 子 agent 回调调度器更新内存状态
And 调度器串行化写入文件
And 无并发写文件问题

Given Web UI 读取状态
When 多个并发读取请求
Then 文件读取不阻塞
And 返回一致的状态视图
```

**FRs covered:** NFR4

---

### Story 8.4: Web UI 文件读取适配

**As a** 开发者
**I want** Web UI 能读取 FileTracker 的文件格式
**So that** 可以在页面展示任务状态

**Acceptance Criteria:**

```gherkin
Given tracker.Kind 为 "file"
When Web UI 加载任务列表
Then 解析 .sym/*/task.md 状态索引
And 解析 YAML frontmatter 获取状态
And 展示三类任务进度
And 展示迭代历史
```

**FRs covered:** FR53, FR54

---

## Story Summary

| Epic | Stories | Priority |
|------|---------|----------|
| Epic 1: 项目初始化与配置 | 4 | P0 |
| Epic 2: Planner 模块实现 | 6 | P0 |
| Epic 3: Generator 模块实现 | 6 | P0 |
| Epic 4: Evaluator 模块实现 | 6 | P0 |
| Epic 5: 迭代回流机制 | 5 | P0 |
| Epic 6: Beads 任务结构适配 | 4 | P0 |
| Epic 7: Web UI 适配 | 4 | P1 |
| Epic 8: FileTracker 实现 | 4 | P1 |
| **Total** | **39** | |

---

## Implementation Order

```
Phase 1: 基础设施
├── Epic 1 (Story 1.1-1.4)
└── Epic 6 (Story 6.1-6.4)

Phase 2: Planner
└── Epic 2 (Story 2.1-2.6)

Phase 3: Generator
└── Epic 3 (Story 3.1-3.6)

Phase 4: Evaluator
└── Epic 4 (Story 4.1-4.6)

Phase 5: 迭代机制
└── Epic 5 (Story 5.1-5.5)

Phase 6: UI 适配
└── Epic 7 (Story 7.1-7.4)

Phase 7: FileTracker (可选)
└── Epic 8 (Story 8.1-8.4)
```