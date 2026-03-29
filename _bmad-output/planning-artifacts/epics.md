---
stepsCompleted: ["step-01-validate-prerequisites", "step-02-design-epics", "step-03-create-stories", "step-04-final-validation", "step-05-sprint-planning"]
inputDocuments:
  - _bmad-output/planning-artifacts/prd.md
  - _bmad-output/planning-artifacts/architecture.md
workflowType: 'epics'
project_name: 'my-symphony'
user_name: 'DK'
date: '2026-03-29'
status: 'complete'
completedAt: '2026-03-30'
---

# my-symphony - Epic Breakdown

## Overview

This document provides the complete epic and story breakdown for my-symphony, decomposing the requirements from the PRD, UX Design if it exists, and Architecture requirements into implementable stories.

## Requirements Inventory

### Functional Requirements

**初始化与配置管理 (FR1-8):**
- FR1: 用户可以通过 CLI 交互式初始化项目配置
- FR2: 用户可以配置 tracker 类型
- FR3: 用户可以配置 AI Agent CLI 类型
- FR4: 系统可以生成 `.sym/` 目录结构
- FR5: 系统可以在启动时验证配置有效性
- FR6: 用户可以修改配置文件（重启生效）
- FR7: 用户可以配置澄清轮次上限
- FR8: 用户可以配置执行重试上限

**任务生命周期管理 (FR9-13):**
- FR9: 用户可以在 Web 页面创建新需求
- FR10: 系统可以自动管理任务状态流转
- FR11: 系统可以创建任务层级结构（父任务 + 子阶段任务）
- FR12: 系统可以管理任务间依赖关系（阶段阻塞）
- FR13: 用户可以取消进行中的需求

**需求澄清 (FR14-21):**
- FR14: 系统可以调用 AI Agent 进行需求理解
- FR15: 用户可以在 Web 页面查看 AI 提问
- FR16: 用户可以在 Web 页面提交回答
- FR17: 系统可以显示澄清进度步骤
- FR18: 系统可以限制澄清轮次上限
- FR19: 用户可以跳过澄清直接流转
- FR20: 系统可以标记需求为"不完整"
- FR21: 用户可以在提交回答后看到确认反馈

**规则生成与管理 (FR22-30):**
- FR22: 系统可以在需求明确后自动生成 BDD 规则
- FR23: 系统可以在架构设计后自动生成 TDD 规则
- FR24: 用户可以在 Web 页面查看 BDD 规则内容
- FR25: 用户可以在 Web 页面通过 BDD 规则
- FR26: 用户可以在 Web 页面驳回 BDD 规则
- FR27: 用户可以在 Web 页面查看架构设计（含 TDD 规则）
- FR28: 用户可以在 Web 页面通过架构设计
- FR29: 用户可以在 Web 页面驳回架构设计
- FR30: 系统可以将审核通过的规则作为约束条件

**执行与监控 (FR31-40):**
- FR31: 用户可以启动后台服务
- FR32: 系统可以调用 AI Agent CLI 执行开发任务
- FR33: 系统可以监控任务执行状态变化
- FR34: 用户可以在 Web 页面查看任务看板
- FR35: 用户可以在 Web 页面按状态筛选任务
- FR36: 用户可以在 Web 页面查看任务进度摘要
- FR37: 用户可以在 Web 页面查看执行日志
- FR38: 系统可以自动检测执行错误
- FR39: 系统可以在执行失败时自动重试
- FR40: 用户可以在状态变化后看到视觉反馈

**验收与报告 (FR41-44):**
- FR41: 系统可以生成验收报告（含测试结果、BDD 通过情况）
- FR42: 用户可以在 Web 页面查看验收报告
- FR43: 用户可以通过验收并流转到"完成"
- FR44: 用户可以驳回验收并流转回"实现中"

**异常处理与恢复 (FR45-49):**
- FR45: 系统可以将执行失败的任务流转到"待人工处理"状态
- FR46: 用户可以在 Web 页面查看失败详情和建议
- FR47: 用户可以在手动修复后继续执行
- FR48: 用户可以重新澄清需求
- FR49: 用户可以放弃需求并流转到"已取消"

**外部集成 (FR50-56):**
- FR50: 系统可以从 Beads tracker 获取任务内容
- FR51: 系统可以获取需求澄清阶段的用户对话记录
- FR52: 系统可以将任务内容和对话记录作为 AI Agent CLI 输入参数
- FR53: 系统可以解析 AI Agent CLI 输出
- FR54: 系统可以处理 AI Agent CLI 错误
- FR55: 系统可以与 Beads tracker 集成更新任务状态
- FR56: 系统可以在任务完成后进行 Git 提交

**Total FRs: 56**

### NonFunctional Requirements

**Integration (NFR1-2):**
- NFR1: AI Agent CLI 执行可等待超过 24 小时（无硬性超时限制）
- NFR2: 系统可以检测 Beads CLI 可用性，不可用时返回明确错误

**Reliability (NFR3-4):**
- NFR3: 服务崩溃后可通过 Beads 任务状态恢复现场
- NFR4: 配置修改后重启服务生效（无自动重启需求）

**Observability (NFR5):**
- NFR5: 系统可以记录执行日志，支持故障排查

**Total NFRs: 5**

### Additional Requirements

**从 Architecture 提取的技术需求:**

**新增文件实现:**
- ARCH-1: 实现 `internal/common/errors/errors.go` - 集中错误码管理
- ARCH-2: 实现 `internal/tracker/beads.go` - Beads tracker 实现
- ARCH-3: 实现 `internal/workflow/stages.go` - 阶段定义
- ARCH-4: 实现 `internal/workflow/engine.go` - Workflow Engine 核心
- ARCH-5: 实现 `internal/workflow/engine_test.go` - Engine 测试

**接口定义:**
- ARCH-6: Tracker interface 定义
- ARCH-7: 数据结构定义
- ARCH-8: BeadsClient 实现含 30s 超时控制

**崩溃恢复:**
- ARCH-9: 实现 RestoreFromBeads 单任务恢复
- ARCH-10: 实现 RestoreAll 启动时批量恢复
- ARCH-11: StageState 序列化到 Beads Custom 字段

**命名规范:**
- ARCH-12: 包名小写单词
- ARCH-13: 接口 PascalCase
- ARCH-14: 导出函数 PascalCase
- ARCH-15: 私有函数 camelCase
- ARCH-16: 阶段名 snake_case
- ARCH-17: 状态常量 PascalCase
- ARCH-18: 错误变量 Err 前缀

**错误处理模式:**
- ARCH-19: 错误码格式 `<module>.<type>: <description>`
- ARCH-20: 错误码定义在 `internal/common/errors/errors.go`

**日志模式:**
- ARCH-21: 使用 slog 结构化日志
- ARCH-22: 字段命名 snake_case

**测试模式:**
- ARCH-23: 表驱动测试
- ARCH-24: Mock 策略 MVP 手写，Phase 2 mockgen
- ARCH-25: 覆盖率目标 MVP 60%，Phase 2 80%
- ARCH-26: 测试文件与源文件同目录

**现有模块扩展:**
- ARCH-27: 扩展 Orchestrator 集成 Workflow Engine
- ARCH-28: 扩展 Tracker 添加 Beads 实现
- ARCH-29: 扩展 Server handlers 添加阶段审核 API
- ARCH-30: 扩展 Agent runner 支持长时间运行

**内部通信:**
- ARCH-31: Workflow Engine ↔ Orchestrator: Go 函数调用
- ARCH-32: Orchestrator ↔ Agent: goroutine + channel
- ARCH-33: Server ↔ Orchestrator: SSE
- ARCH-34: Engine ↔ Tracker: Go 函数调用

**外部集成:**
- ARCH-35: Beads CLI 命令行调用
- ARCH-36: AI Agent CLI 命令行调用
- ARCH-37: Git 命令行调用
- ARCH-38: Web UI HTTP + SSE

**Total Additional Requirements: 38**

### UX Design Requirements

无 UX Design 文档（已有 Web UI，可跳过）

### FR Coverage Map

FR1 → Epic 1 - CLI 交互式初始化
FR2 → Epic 1 - 配置 tracker 类型
FR3 → Epic 1 - 配置 AI Agent CLI 类型
FR4 → Epic 1 - 生成 .sym/ 目录结构
FR5 → Epic 1 - 启动时验证配置有效性
FR6 → Epic 1 - 修改配置文件（重启生效）
FR7 → Epic 1 - 配置澄清轮次上限
FR8 → Epic 1 - 配置执行重试上限
FR9 → Epic 2 - Web 页面创建新需求
FR10 → Epic 2 - 自动管理任务状态流转
FR11 → Epic 2 - 创建任务层级结构
FR12 → Epic 2 - 管理任务间依赖关系
FR13 → Epic 2 - 取消进行中的需求
FR14 → Epic 3 - 调用 AI Agent 进行需求理解
FR15 → Epic 3 - Web 页面查看 AI 提问
FR16 → Epic 3 - Web 页面提交回答
FR17 → Epic 3 - 显示澄清进度步骤
FR18 → Epic 3 - 限制澄清轮次上限
FR19 → Epic 3 - 跳过澄清直接流转
FR20 → Epic 3 - 标记需求为"不完整"
FR21 → Epic 3 - 提交回答后确认反馈
FR22 → Epic 4 - 需求明确后自动生成 BDD 规则
FR23 → Epic 5 - 架构设计后自动生成 TDD 规则
FR24 → Epic 4 - Web 页面查看 BDD 规则内容
FR25 → Epic 4 - Web 页面通过 BDD 规则
FR26 → Epic 4 - Web 页面驳回 BDD 规则
FR27 → Epic 5 - Web 页面查看架构设计
FR28 → Epic 5 - Web 页面通过架构设计
FR29 → Epic 5 - Web 页面驳回架构设计
FR30 → Epic 4 - 将审核通过的规则作为约束条件
FR31 → Epic 1 - 启动后台服务
FR32 → Epic 6 - 调用 AI Agent CLI 执行开发任务
FR33 → Epic 6 - 监控任务执行状态变化
FR34 → Epic 2 - Web 页面查看任务看板
FR35 → Epic 2 - Web 页面按状态筛选任务
FR36 → Epic 6 - Web 页面查看任务进度摘要
FR37 → Epic 6 - Web 页面查看执行日志
FR38 → Epic 6 - 自动检测执行错误
FR39 → Epic 6 - 执行失败时自动重试
FR40 → Epic 6 - 状态变化后视觉反馈
FR41 → Epic 7 - 生成验收报告
FR42 → Epic 7 - Web 页面查看验收报告
FR43 → Epic 7 - 通过验收流转到"完成"
FR44 → Epic 7 - 驳回验收流转回"实现中"
FR45 → Epic 8 - 流转到"待人工处理"状态
FR46 → Epic 8 - Web 页面查看失败详情和建议
FR47 → Epic 8 - 手动修复后继续执行
FR48 → Epic 8 - 重新澄清需求
FR49 → Epic 8 - 放弃需求流转到"已取消"
FR50 → Epic 1 - 从 Beads tracker 获取任务内容
FR51 → Epic 9 - 获取需求澄清阶段的用户对话记录
FR52 → Epic 6 - 任务内容和对话记录作为 Agent 输入参数
FR53 → Epic 6 - 解析 AI Agent CLI 输出
FR54 → Epic 6 - 处理 AI Agent CLI 错误
FR55 → Epic 1 - 与 Beads tracker 集成更新任务状态
FR56 → Epic 7 - 任务完成后进行 Git 提交

NFR1 → Epic 9 - AI Agent CLI 执行等待 >24h
NFR2 → Epic 1 - Beads CLI 可用性检测
NFR3 → Epic 9 - 崩溃后状态恢复
NFR4 → Epic 1 - 配置修改后重启生效
NFR5 → Epic 9 - 执行日志记录

## Epic List

### Epic 1: 项目初始化与服务启动
用户可以初始化项目配置并启动后台服务，验证 Beads tracker 连接正常。
**FRs covered:** FR1-8, FR31, FR50, FR55, NFR2, NFR4 (13 FRs)

### Epic 2: 任务创建与看板管理
用户可以在 Web 页面创建任务，查看任务看板，按状态筛选任务。
**FRs covered:** FR9-13, FR34-35 (6 FRs)

### Epic 3: 需求澄清交互流程
用户可以与 AI Agent 完成需求澄清对话，查看进度，跳过或标记需求不完整。
**FRs covered:** FR14-21 (8 FRs)

### Epic 4: BDD 规则生成与审核
用户可以在需求明确后查看自动生成的 BDD 规则，并通过或驳回。
**FRs covered:** FR22, FR24-26, FR30 (6 FRs)

### Epic 5: 架构设计与 TDD 规则审核
用户可以查看架构设计和 TDD 规则，并通过或驳回。
**FRs covered:** FR23, FR27-29 (4 FRs)

### Epic 6: AI Agent 执行与实时监控
用户可以实时监控 AI Agent 执行进度，查看日志，接收错误和状态变化的视觉反馈。
**FRs covered:** FR32-33, FR36-40, FR52-54 (10 FRs)

### Epic 7: 验收报告与任务完成
用户可以查看验收报告，通过验收完成任务，或驳回回到实现中，完成后 Git 提交。
**FRs covered:** FR41-44, FR56 (5 FRs)

### Epic 8: 异常处理与人工干预
用户可以在任务失败时查看详情，手动修复后继续执行，或重新澄清/放弃需求。
**FRs covered:** FR45-49 (5 FRs)

### Epic 9: 服务可靠性与崩溃恢复
用户信任服务崩溃后能恢复进度，长时间执行不会超时，所有操作有日志记录。
**FRs covered:** FR51, NFR1, NFR3, NFR5 (1 FR + 4 NFRs)

---

## Epic 1: 项目初始化与服务启动

用户可以初始化项目配置并启动后台服务，验证 Beads tracker 连接正常。

### Story 1.1: CLI init 交互式初始化

As a **开发者**,
I want **通过 CLI 交互式初始化项目配置**,
So that **我可以快速配置 Symphony 项目**.

**Acceptance Criteria:**

**Given** 用户在项目根目录执行 `symphony init`
**When** CLI 启动交互式问答
**Then** 用户可以选择 tracker 类型
**And** 用户可以选择 AI Agent CLI 类型
**And** 系统生成 `.sym/` 目录结构
**And** 系统生成 `.sym/config.yaml` 配置文件

**Given** 用户完成交互式问答
**When** 系统生成配置
**Then** `.sym/prompts/` 目录包含默认 prompt 文件
**And** 配置文件包含 tracker、agent、polling 等基础配置

**FRs covered:** FR1, FR2, FR3, FR4

---

### Story 1.2: 配置验证与管理

As a **开发者**,
I want **系统在启动时验证配置有效性**,
So that **我可以确保配置正确后再启动服务**.

**Acceptance Criteria:**

**Given** 用户执行 `symphony start`
**When** 系统读取 `.sym/config.yaml`
**Then** 系统验证 tracker 配置是否有效
**And** 系统验证 AI Agent CLI 路径是否存在
**And** 系统验证 prompt 文件是否存在

**Given** 配置验证失败
**When** 系统检测到无效配置
**Then** 返回错误码 `config.invalid: <具体原因>`
**And** 提示用户修改配置

**Given** 用户修改配置文件
**When** 用户重启服务
**Then** 新配置生效

**FRs covered:** FR5, FR6, FR7, FR8, NFR4

---

### Story 1.3: Beads Tracker 集成

As a **开发者**,
I want **系统与 Beads tracker 集成以获取和更新任务**,
So that **我可以使用本地 CLI tracker 管理任务**.

**Acceptance Criteria:**

**Given** 服务启动时
**When** 系统检查 Beads CLI 可用性
**Then** 调用 `CheckAvailability()` 验证
**And** 不可用时返回错误 `tracker.unavailable: Beads CLI 不可用`

**Given** Beads CLI 可用
**When** 系统需要获取任务
**Then** 调用 `GetTask()` 获取任务详情
**And** 调用 `ListTasksByState()` 获取任务列表

**Given** 任务状态变化
**When** 系统更新任务
**Then** 调用 `UpdateStage()` 更新阶段状态
**And** 超时控制为 30 秒

**FRs covered:** FR50, FR55, NFR2

---

### Story 1.4: CLI start 启动后台服务

As a **开发者**,
I want **通过 CLI start 启动后台服务**,
So that **我可以让 Symphony 在后台监控和执行任务**.

**Acceptance Criteria:**

**Given** 配置验证通过
**When** 用户执行 `symphony start`
**Then** 后台服务启动
**And** Web 服务器在配置端口启动
**And** 任务轮询开始

**Given** 服务已启动
**When** 用户访问 `http://localhost:<port>`
**Then** 显示 Web 看板页面

**Given** 服务运行中
**When** 用户按 Ctrl+C
**Then** 服务优雅关闭
**And** 正在执行的任务状态保存到 Beads

**FRs covered:** FR31

---

## Epic 2: 任务创建与看板管理

用户可以在 Web 页面创建任务，查看任务看板，按状态筛选任务。

### Story 2.1: Web 页面创建新需求

As a **开发者**,
I want **在 Web 页面创建新需求**,
So that **我可以提交开发任务给 Symphony 处理**.

**Acceptance Criteria:**

**Given** 用户访问 Web 看板
**When** 用户点击"创建需求"按钮
**Then** 显示需求创建表单
**And** 表单包含标题和描述字段

**Given** 用户填写需求表单
**When** 用户点击"提交"
**Then** 系统在 Beads 创建父任务
**And** 系统创建 5 个子阶段任务（澄清、BDD审核、架构审核、实现、验收）
**And** 子任务使用 dependency 阻塞关系
**And** 页面跳转到任务详情

**FRs covered:** FR9, FR11, FR12

---

### Story 2.2: 任务状态自动流转

As a **开发者**,
I want **系统自动管理任务状态流转**,
So that **我可以看到任务在不同阶段间正确推进**.

**Acceptance Criteria:**

**Given** 父任务创建完成
**When** 子任务按依赖顺序就绪
**Then** 第一个子任务（需求澄清）状态变为"进行中"
**And** 后续子任务保持"待开始"状态

**Given** 当前阶段任务完成
**When** 阶段成功结束
**Then** 当前任务状态变为"完成"
**And** 下一阶段任务状态变为"进行中"
**And** 父任务状态更新为当前阶段名称

**Given** 阶段执行失败
**When** 系统检测到失败
**Then** 任务流转到对应的异常状态
**And** 保留失败原因

**FRs covered:** FR10

---

### Story 2.3: 任务看板展示

As a **开发者**,
I want **在 Web 页面查看任务看板**,
So that **我可以一目了然地看到所有任务状态**.

**Acceptance Criteria:**

**Given** 用户访问 Web 首页
**When** 页面加载完成
**Then** 显示任务看板
**And** 任务按状态分列展示（待开始、进行中、待审核、完成等）
**And** 每个任务卡片显示标题、状态、当前阶段

**Given** 看板加载完成
**When** 任务状态变化
**Then** 通过 SSE 实时更新看板
**And** 任务卡片移动到对应列
**And** 显示视觉反馈动画

**FRs covered:** FR34

---

### Story 2.4: 任务状态筛选

As a **开发者**,
I want **在 Web 页面按状态筛选任务**,
So that **我可以快速找到特定状态的任务**.

**Acceptance Criteria:**

**Given** 看板显示多个任务
**When** 用户点击状态筛选器
**Then** 显示所有可用状态选项
**And** 包含：全部、待开始、进行中、待审核、待人工处理、完成、已取消

**Given** 用户选择筛选状态
**When** 筛选应用
**Then** 只显示匹配状态的任务
**And** 显示筛选结果数量

**FRs covered:** FR35

---

### Story 2.5: 取消进行中的需求

As a **开发者**,
I want **取消进行中的需求**,
So that **我可以停止不需要的任务继续执行**.

**Acceptance Criteria:**

**Given** 任务处于进行中状态
**When** 用户点击"取消"按钮
**Then** 显示确认对话框
**And** 提示取消操作不可逆

**Given** 用户确认取消
**When** 取消操作执行
**Then** 任务状态变为"已取消"
**And** 正在执行的 Agent 进程被终止
**And** 子任务状态同步更新

**FRs covered:** FR13

---

## Epic 3: 需求澄清交互流程

用户可以与 AI Agent 完成需求澄清对话，查看进度，跳过或标记需求不完整。

### Story 3.1: AI Agent 需求理解调用

As a **开发者**,
I want **系统能调用 AI Agent 对需求进行理解和分析**,
So that **AI 能主动识别需求中不明确的地方并提出问题**.

**Acceptance Criteria:**

**Given** 一个新任务进入"需求澄清"阶段
**When** 阶段开始执行
**Then** 系统读取任务标题和描述
**And** 系统调用 AI Agent CLI，传入 clarification prompt
**And** AI Agent 返回澄清问题或确认需求已明确

**Given** AI Agent 返回澄清问题
**When** 系统解析响应
**Then** 问题保存到任务的澄清记录中
**And** 任务状态变为"等待用户回答"

**FRs covered:** FR14

---

### Story 3.2: Web 页面查看 AI 提问

As a **开发者**,
I want **在 Web 页面查看 AI 提出的澄清问题**,
So that **我可以理解 AI 需要哪些信息来明确需求**.

**Acceptance Criteria:**

**Given** 任务处于"等待用户回答"状态
**When** 用户打开任务详情页
**Then** 页面显示 AI 提出的当前问题
**And** 显示澄清进度（当前轮次/总上限）
**And** 显示历史问答记录

**FRs covered:** FR15, FR17

---

### Story 3.3: Web 页面提交回答

As a **开发者**,
I want **在 Web 页面提交对 AI 问题的回答**,
So that **我可以提供信息帮助 AI 理解需求**.

**Acceptance Criteria:**

**Given** 用户查看 AI 提问
**When** 用户输入回答并点击"提交"
**Then** 回答保存到澄清记录
**And** 系统调用 AI Agent 继续下一轮澄清
**And** 页面显示提交成功反馈
**And** 如果 AI 还有问题，显示新问题；否则进入下一阶段

**FRs covered:** FR16, FR21

---

### Story 3.4: 澄清轮次限制与跳过

As a **开发者**,
I want **系统能限制澄清轮次并允许我跳过澄清**,
So that **澄清不会无限循环，我可以强制推进流程**.

**Acceptance Criteria:**

**Given** 配置了澄清轮次上限（默认 5 轮）
**When** 澄清达到上限
**Then** 系统自动标记需求为"不完整"
**And** 任务流转到"待人工处理"状态

**Given** 用户想要跳过澄清
**When** 用户点击"跳过澄清"按钮
**Then** 系统标记需求为"不完整"
**And** 任务直接流转到下一阶段

**FRs covered:** FR18, FR19, FR20

---

## Epic 4: BDD 规则生成与审核

用户可以在需求明确后查看自动生成的 BDD 规则，并通过或驳回。

### Story 4.1: BDD 规则自动生成

As a **Symphony 用户**,
I want **系统在需求澄清完成后自动生成 BDD 规则**,
So that **我可以快速获得验收标准，无需手动编写测试规则**.

**Acceptance Criteria:**

**Given** 一个任务已完成需求澄清阶段
**When** 阶段流转触发 BDD 规则生成
**Then** 系统调用 AI Agent 使用 `.sym/prompts/bdd.md` 模板生成 BDD 规则
**And** 生成的 BDD 规则存储到项目的 `docs/bdd/` 目录
**And** 任务状态流转到"待审核 BDD"（`bdd_review`）
**And** 生成的规则包含 Gherkin 格式的场景描述

**FRs covered:** FR22

---

### Story 4.2: Web 页面查看 BDD 规则

As a **Symphony 用户**,
I want **在 Web 页面查看自动生成的 BDD 规则内容**,
So that **我可以审核规则是否符合需求预期**.

**Acceptance Criteria:**

**Given** 一个任务处于"待审核 BDD"状态
**When** 用户在 Web 看板点击该任务
**Then** 页面显示 BDD 规则内容详情
**And** 规则内容以 Gherkin 格式展示（Scenario、Given、When、Then 结构）
**And** 提供"通过"和"驳回"两个操作按钮
**And** 页面显示任务基本信息（标题、描述、当前阶段）

**FRs covered:** FR24

---

### Story 4.3: Web 页面审核 BDD 规则

As a **Symphony 用户**,
I want **在 Web 页面通过或驳回 BDD 规则**,
So that **我可以控制需求验收标准，确保规则符合预期**.

**Acceptance Criteria:**

**Given** 一个任务处于"待审核 BDD"状态且用户已查看 BDD 规则
**When** 用户点击"通过"按钮
**Then** 任务状态流转到"待审核架构"（`architecture_review`）
**And** BDD 规则被标记为"已通过"
**And** 页面显示确认反馈"BDD 规则审核通过"

**Given** 一个任务处于"待审核 BDD"状态且用户已查看 BDD 规则
**When** 用户点击"驳回"按钮并可选填写驳回原因
**Then** 任务状态流转回"待设计"（`pending_design`）
**And** 页面显示确认反馈"BDD 规则已驳回，需重新生成"

**FRs covered:** FR25, FR26

---

### Story 4.4: 审核通过规则作为约束条件

As a **Symphony 用户**,
I want **审核通过的 BDD 规则作为后续执行的约束条件**,
So that **AI Agent 在实现阶段会遵守这些规则**.

**Acceptance Criteria:**

**Given** 一个任务的 BDD 规则已审核通过
**When** 任务流转到实现阶段
**Then** BDD 规则文件路径作为约束条件参数传递给 AI Agent
**And** Agent Prompt 包含明确的指令："实现必须满足以下 BDD 规则"
**And** 验收阶段会验证实现是否符合 BDD 规则

**FRs covered:** FR30

---

## Epic 5: 架构设计与 TDD 规则审核

用户可以查看架构设计和 TDD 规则，并通过或驳回。

### Story 5.1: 架构设计与 TDD 规则自动生成

As a **开发者**,
I want **系统在 BDD 审核通过后自动生成架构设计和 TDD 规则**,
So that **AI 实现有明确的技术指导和测试约束**.

**Acceptance Criteria:**

**Given** BDD 规则审核通过
**When** 任务进入架构设计阶段
**Then** 系统调用 AI Agent 生成架构设计文档
**And** 系统生成 TDD 规则（测试驱动开发约束）
**And** 架构设计存储到项目 `docs/architecture/` 目录
**And** TDD 规则存储到 `docs/tdd/` 目录
**And** 任务状态流转到"待审核架构"

**FRs covered:** FR23

---

### Story 5.2: Web 页面查看架构设计

As a **开发者**,
I want **在 Web 页面查看架构设计和 TDD 规则**,
So that **我可以审核技术方案是否符合预期**.

**Acceptance Criteria:**

**Given** 任务处于"待审核架构"状态
**When** 用户打开任务详情
**Then** 页面显示架构设计文档内容
**And** 页面显示 TDD 规则列表
**And** 提供"通过"和"驳回"操作按钮

**FRs covered:** FR27

---

### Story 5.3: Web 页面审核架构设计

As a **开发者**,
I want **在 Web 页面通过或驳回架构设计**,
So that **我可以控制技术方案，确保方向正确**.

**Acceptance Criteria:**

**Given** 用户查看架构设计后点击"通过"
**When** 提交通过操作
**Then** 任务流转到"实现中"阶段
**And** 架构设计作为实现阶段的输入约束

**Given** 用户点击"驳回"
**When** 填写驳回原因并提交
**Then** 任务流转回"待设计"状态
**And** 驳回原因记录到任务历史

**FRs covered:** FR28, FR29

---

## Epic 6: AI Agent 执行与实时监控

用户可以实时监控 AI Agent 执行进度，查看日志，接收错误和状态变化的视觉反馈。

### Story 6.1: AI Agent CLI 执行调用

As a **开发者**,
I want **系统能调用 AI Agent CLI 执行开发任务**,
So that **AI 能自动完成代码实现**.

**Acceptance Criteria:**

**Given** 任务进入"实现中"阶段
**When** 阶段开始执行
**Then** 系统构建包含需求、BDD、架构、TDD 的完整 prompt
**And** 系统调用配置的 AI Agent CLI（claude/codex/opencode）
**And** Agent 在工作空间中执行开发任务

**Given** Agent 执行过程中
**When** Agent 输出日志或状态
**Then** 系统实时捕获并记录
**And** 通过 SSE 推送到 Web 页面

**FRs covered:** FR32, FR52

---

### Story 6.2: 执行状态监控

As a **开发者**,
I want **系统能监控任务执行状态变化**,
So that **我能知道任务当前处于什么状态**.

**Acceptance Criteria:**

**Given** 任务正在执行
**When** Agent 进程状态变化
**Then** 系统检测到状态变化
**And** 更新任务状态到 Beads
**And** 通过 SSE 通知 Web 页面

**Given** Agent 执行完成
**When** 进程正常退出
**Then** 系统解析执行结果
**And** 根据结果流转到下一阶段或异常状态

**FRs covered:** FR33, FR53

---

### Story 6.3: Web 页面查看进度摘要

As a **开发者**,
I want **在 Web 页面查看任务进度摘要**,
So that **我能快速了解任务执行情况**.

**Acceptance Criteria:**

**Given** 任务正在执行
**When** 用户打开任务详情
**Then** 页面显示当前阶段
**And** 显示已用时间
**And** 显示进度摘要（如：正在编写测试用例 3/10）

**FRs covered:** FR36

---

### Story 6.4: Web 页面查看执行日志

As a **开发者**,
I want **在 Web 页面查看执行日志**,
So that **我能详细了解 AI 的执行过程**.

**Acceptance Criteria:**

**Given** 任务正在执行或有执行历史
**When** 用户点击"查看日志"
**Then** 页面显示执行日志列表
**And** 支持分页（每页 100 条）
**And** 支持实时更新（通过 SSE）

**FRs covered:** FR37

---

### Story 6.5: 执行错误检测与重试

As a **开发者**,
I want **系统能自动检测执行错误并重试**,
So that **临时错误不会阻塞任务完成**.

**Acceptance Criteria:**

**Given** Agent 执行过程中发生错误
**When** 系统检测到错误
**Then** 记录错误详情
**And** 如果重试次数未达上限，自动重新执行
**And** 如果达到上限，流转到"待人工处理"状态

**Given** 配置了重试上限（默认 3 次）
**When** 错误发生
**Then** 系统按配置进行重试

**FRs covered:** FR38, FR39, FR54

---

### Story 6.6: 状态变化视觉反馈

As a **开发者**,
I want **在状态变化后看到视觉反馈**,
So that **我能立即注意到任务进展**.

**Acceptance Criteria:**

**Given** 用户在看板页面
**When** 任务状态变化
**Then** 任务卡片有动画效果
**And** 状态列显示更新提示
**And** 可选的浏览器通知

**FRs covered:** FR40

---

## Epic 7: 验收报告与任务完成

用户可以查看验收报告，通过验收完成任务，或驳回回到实现中，完成后 Git 提交。

### Story 7.1: 验收报告生成

As a **系统用户**,
I want **系统在实现阶段完成后自动生成验收报告**,
So that **我可以基于客观证据决定是否验收通过**.

**Acceptance Criteria:**

**Given** 任务已完成实现阶段，状态流转到"待验收"
**When** Workflow Engine 触发 verification 阶段执行
**Then** 系统调用 AI Agent 执行 `.sym/prompts/verification.md` prompt
**And** AI Agent 运行测试套件并收集测试结果
**And** AI Agent 执行 BDD 规则验证并收集通过情况
**And** 系统将验收报告存储到工程内 `docs/verification_report.md`
**And** 任务状态保持"待验收"，等待人工审核

**FRs covered:** FR41

---

### Story 7.2: 验收报告 Web 页面展示

As a **开发者**,
I want **在 Web 页面查看验收报告内容**,
So that **我可以快速了解任务完成质量并做出验收决策**.

**Acceptance Criteria:**

**Given** 任务处于"待验收"状态，验收报告已生成
**When** 用户在 Web 看板点击该任务卡片
**Then** 页面展示验收报告详情区域
**And** 显示测试结果摘要：总测试数、通过数、失败数
**And** 显示 BDD 规则通过情况：规则名称、状态（通过/失败）
**And** 显示验收报告完整内容（支持展开/折叠）
**And** 提供"通过验收"和"驳回验收"两个操作按钮

**FRs covered:** FR42

---

### Story 7.3: 验收通过/驳回状态流转

As a **开发者**,
I want **在 Web 页面通过或驳回验收**,
So that **我可以控制任务最终状态，确保交付质量符合预期**.

**Acceptance Criteria:**

**Given** 任务处于"待验收"状态，验收报告已展示
**When** 用户点击"通过验收"按钮
**Then** 系统调用 Tracker 更新任务状态为"完成"
**And** Web 页面显示成功反馈，任务卡片移到"完成"区域
**And** 触发 Git 提交流程

**Given** 任务处于"待验收"状态，验收报告已展示
**When** 用户点击"驳回验收"按钮
**Then** 系统调用 Tracker 更新任务状态回"实现中"
**And** Web 页面显示驳回反馈，任务卡片移回"实现中"区域
**And** 系统保留验收失败记录供后续修复参考

**FRs covered:** FR43, FR44

---

### Story 7.4: 任务完成后 Git 提交

As a **开发者**,
I want **任务验收通过后系统自动执行 Git 提交**,
So that **所有变更可以保存到版本控制系统，形成可追溯的交付记录**.

**Acceptance Criteria:**

**Given** 任务验收通过，状态流转到"完成"
**When** Workflow Engine 检测到任务进入 completed 阶段
**Then** 系统在工作空间目录执行 Git commit
**And** commit message 包含任务标识符和标题
**And** 提交成功后记录 commit hash 到日志
**And** 提交失败时返回明确错误，不影响任务完成状态

**FRs covered:** FR56

---

## Epic 8: 异常处理与人工干预

用户可以在任务失败时查看详情，手动修复后继续执行，或重新澄清/放弃需求。

### Story 8.1: 失败任务流转到待人工处理状态

As a **系统用户**,
I want **当任务执行失败时，系统能自动将其流转到"待人工处理"状态**,
So that **我能及时发现问题并采取干预措施**.

**Acceptance Criteria:**

**Given** 一个任务正在执行中，且已达到配置的重试上限
**When** 执行再次失败
**Then** 系统将任务状态从"执行中"流转到"待人工处理"
**And** 系统记录失败原因、失败时间、重试次数到任务详情
**And** 系统通过 SSE 向 Web UI 推送状态变更通知

**FRs covered:** FR45

---

### Story 8.2: 失败详情展示与建议查看

As a **产品负责人**,
I want **在 Web 页面查看失败任务的详细信息和系统给出的修复建议**,
So that **我能快速理解失败原因，判断最佳干预方式**.

**Acceptance Criteria:**

**Given** 存在状态为"待人工处理"的任务
**When** 用户在 Web 页面点击该任务
**Then** 页面展示任务详情面板，包含：
  - 任务标识符和标题
  - 失败阶段
  - 失败时间
  - 重试次数
  - 错误类型和错误消息
  - AI Agent 最后输出的日志片段
  - 系统生成的修复建议

**FRs covered:** FR46

---

### Story 8.3: 手动修复后继续执行

As a **开发人员**,
I want **在手动修复问题后，能让任务从失败点继续执行**,
So that **已完成的工作不会丢失，任务能顺利完成**.

**Acceptance Criteria:**

**Given** 一个任务处于"待人工处理"状态
**When** 用户在 Web 页面点击"继续执行"按钮
**Then** 系统将任务状态流转回"执行中"
**And** 系统保留已完成的阶段进度
**And** 系统从失败阶段重新开始执行
**And** 重置重试计数器为 0

**FRs covered:** FR47

---

### Story 8.4: 需求重新澄清或放弃

As a **产品负责人**,
I want **能选择重新澄清需求或放弃整个任务**,
So that **当需求本身有问题时能重新定义，或当需求不再需要时能清理资源**.

**Acceptance Criteria:**

**Given** 一个任务处于"待人工处理"状态
**When** 用户在 Web 页面点击"重新澄清需求"按钮
**Then** 系统将任务状态流转到"需求澄清"阶段
**And** 清除之前的 BDD 规则和架构设计
**And** 保留原始需求描述

**Given** 一个任务处于"待人工处理"状态
**When** 用户在 Web 页面点击"放弃需求"按钮
**Then** 系统弹出确认对话框
**And** 用户确认后任务状态流转到"已取消"
**And** 清理关联的工作空间目录

**FRs covered:** FR48, FR49

---

## Epic 9: 服务可靠性与崩溃恢复

用户信任服务崩溃后能恢复进度，长时间执行不会超时，所有操作有日志记录。

### Story 9.1: 结构化执行日志系统

As a **运维人员**,
I want **系统能记录结构化的执行日志**,
So that **我可以在故障发生时快速定位问题原因并排查**.

**Acceptance Criteria:**

**Given** 系统正在执行一个任务
**When** 任务状态发生变化或发生错误
**Then** 系统使用 `log/slog` 记录一条结构化日志
**And** 日志字段命名遵循 `snake_case` 规范

**Given** 任务执行过程中发生错误
**When** 错误被捕获
**Then** 日志记录包含 `error_code`、`error_message`、`stack_trace` 字段
**And** 日志级别为 `ERROR` 或更高

**FRs covered:** NFR5

---

### Story 9.2: 无超时限制的长时间执行支持

As a **系统用户**,
I want **AI Agent CLI 执行能够等待超过 24 小时，没有硬性超时限制**,
So that **复杂或大规模任务不会因为超时而被中断**.

**Acceptance Criteria:**

**Given** 用户配置了 `turn_timeout_ms` 参数为 0 或负数
**When** Agent CLI 执行
**Then** 系统将任务视为无超时限制，允许无限等待
**And** 通过 goroutine + channel 实现阻塞等待

**Given** 任务执行超过 24 小时
**When** 任务仍在正常活动
**Then** 系统继续等待执行完成

**FRs covered:** NFR1

---

### Story 9.3: 服务崩溃后的任务状态恢复

As a **系统用户**,
I want **服务崩溃重启后能够自动恢复所有进行中的任务现场**,
So that **我不需要手动重新启动任务，进度不会丢失**.

**Acceptance Criteria:**

**Given** 服务启动时
**When** 存在进行中的 Beads 任务
**Then** 系统调用 `RestoreAll()` 方法扫描所有活跃状态任务
**And** 对每个任务根据 `StageState` 序列化数据恢复执行阶段

**Given** 一个任务在执行阶段崩溃
**When** 服务重启并恢复该任务
**Then** 系统从 Beads 任务的 `Custom` 字段读取 `StageState`
**And** 根据阶段状态恢复到：继续执行、等待审核、或重试

**FRs covered:** NFR3

---

### Story 9.4: 需求澄清对话记录获取与传递

As a **系统架构师**,
I want **系统能够从 Beads tracker 获取需求澄清阶段的完整用户对话记录**,
So that **AI Agent CLI 能够理解历史对话上下文**.

**Acceptance Criteria:**

**Given** 一个任务处于"需求澄清已完成"阶段
**When** 系统准备调用 AI Agent CLI 执行下一阶段
**Then** 系统调用 Beads tracker API 获取该任务的澄清对话记录
**And** 对话记录包含：用户提问、AI 回答、澄清轮次、时间戳

**Given** 系统获取了澄清对话记录
**When** 构建 AI Agent CLI 输入参数
**Then** 对话记录作为 prompt 上下文的一部分传递给 Agent

**FRs covered:** FR51

---

## Story Summary

| Epic | Stories | FRs Covered |
|------|---------|-------------|
| Epic 1 | 4 | FR1-8, FR31, FR50, FR55, NFR2, NFR4 |
| Epic 2 | 5 | FR9-13, FR34-35 |
| Epic 3 | 4 | FR14-21 |
| Epic 4 | 4 | FR22, FR24-26, FR30 |
| Epic 5 | 3 | FR23, FR27-29 |
| Epic 6 | 6 | FR32-33, FR36-40, FR52-54 |
| Epic 7 | 4 | FR41-44, FR56 |
| Epic 8 | 4 | FR45-49 |
| Epic 9 | 4 | FR51, NFR1, NFR3, NFR5 |
| **Total** | **38** | **56 FRs + 5 NFRs** |