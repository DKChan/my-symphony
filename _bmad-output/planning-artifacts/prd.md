---
stepsCompleted: ["step-01-init", "step-02-discovery", "step-02b-vision", "step-02c-executive-summary", "step-03-success", "step-04-journeys", "step-05-domain", "step-06-innovation", "step-07-project-type", "step-08-scoping", "step-09-functional", "step-10-nonfunctional", "step-11-polish"]
inputDocuments:
  - docs/SPEC.md
workflowType: 'prd'
documentCounts:
  briefCount: 0
  researchCount: 0
  brainstormingCount: 0
  projectDocsCount: 1
classification:
  projectType: CLI Tool / Developer Tool
  domain: Developer Tool
  complexity: medium
  projectContext: brownfield
workflowPreferences:
  bddFormat: Gherkin
---

# Product Requirements Document - my-symphony

**作者:** DK
**日期:** 2026-03-27

## Executive Summary

Symphony 是**自动化开发的 harness 工具**，让 AI 自主完成开发任务，输出锚定用户意图。它解决的核心问题不是 AI 能力不足，而是软件工程的**根本困难**——不可见性、复杂性、一致性、易变性导致 AI 输出发散。

**目标用户：** 个人开发者，追求 AI 辅助开发的可控性和正确性。

**人类参与节点：** 需求提出、需求澄清、BDD 规则审核、架构设计审核、验收。其余状态流转和结果产出由 harness 自动完成。

### What Makes This Special

**核心差异化：** 主动驱动 + 约束验证的开发自动化 harness，而非被动响应的 AI 编程助手或仅负责执行的 CI/CD 流水线。

**三大核心能力：**
| 能力 | 描述 |
|------|------|
| **自动化** | 声明式控制面，监控任务后自主流转 |
| **约束** | BDD + TDD 多级正确性规则，锚定用户意图 |
| **验证** | 单测全绿 → BDD 全通过 → 报告证据，层层把关 |

**核心洞察：** 在需求澄清阶段确定 BDD 规则、在架构设计阶段确定 TDD 规则，实现多级正确性约束。

## Project Classification

| 维度 | 分类 |
|------|------|
| **Project Type** | CLI Tool（init 阶段）+ Daemon Service（运行阶段） |
| **Domain** | Developer Tool / AI Agent Orchestration |
| **Complexity** | Medium |
| **Project Context** | Brownfield（在现有 Symphony 基础上添加 init 功能） |

## Success Criteria

### User Success

**Aha! 时刻：**
- 需求澄清完成后，BDD 规则自动生成
- AI 实现完成后，单测全绿、BDD 全通过的验收报告
- 整个过程无需手动编写测试代码

**用户完成场景：**
提出需求 → 需求澄清 → BDD 规则审核 → 架构审核 → 等待自动化执行 → 查看验收报告 → 交付完成

### Business Success

**时间线：**
| 时间节点 | 成功定义 |
|----------|----------|
| **1 个月** | MVP 上线，完成第一个真实开发任务 |
| **3 个月** | 完成多次迭代开发，交付可用的软件 |
| **12 个月** | 完全依赖工具进行开发，融入日常开发和交付环节 |

### Technical Success

**分层异常恢复能力：**
| 级别 | 异常类型 | 恢复策略 |
|------|----------|----------|
| **L1（轻量）** | AI 输出解析失败 | 自动重试 |
| **L2（中等）** | 进程崩溃 | 状态恢复后继续 |
| **L3（严重）** | 环境损坏 | 人工介入 |

**规则执行准确性（分离）：**
- BDD 规则执行准确率
- TDD 规则执行准确率

### Measurable Outcomes

| 指标 | 度量方式 |
|------|----------|
| **需求澄清质量** | 用户在澄清阶段的参与满意度 |
| **BDD 审核通过率** | 用户审核通过 BDD 规则数 / 生成的 BDD 规则总数 |
| **任务完成准确率** | BDD 全通过的任务数 / 总任务数 |
| **返工率** | 需要人类介入修正的任务数 / 总任务数 |
| **Cycle Time** | 需求提出到验收通过的时间 |
| **测试稳定性** | 测试连续通过无波动的比例 |
| **使用率** | 日常开发任务中使用 Symphony 的比例 |

## Product Scope

Symphony 采用分期演进策略，详见 [Project Scoping & Phased Development](#project-scoping--phased-development)。

- **MVP (Phase 1):** 验证核心价值——确定性输出
- **Growth (Phase 2):** 扩展功能与体验优化
- **Vision (Phase 3):** 平台化与生态建设

## User Journeys

### 用户类型

| 类型 | 描述 |
|------|------|
| **开发者** | 唯一用户角色，既是使用者也是管理者 |
| **交互方式** | CLI（init/start）+ 本地 Web 页面（日常操作） |

### Journey 1: 首次使用 - 从初始化到第一个需求

**主角：** DK，个人开发者，想用 Symphony 自动化开发流程

**情境：** 新项目刚搭建，想要让 Symphony 帮忙管理后续的开发任务

**旅程叙事：**

1. **初始化** - DK 在项目根目录执行 `symphony init`，CLI 交互式引导选择 Beads 作为 tracker，配置生成到 `.sym/` 目录

2. **启动服务** - 执行 `symphony start`，后台服务启动，监控任务变化

3. **打开 Web 页面** - 浏览器访问本地服务，看到空白看板，准备创建第一个需求

4. **创建需求** - 在 Web 页面提交一个简单想法："添加用户登录功能"

5. **需求澄清** - Symphony 监测到新需求，调用 AI Agent 进行需求理解：
   - 页面显示"AI 正在思考..."动画
   - 显示进度步骤：理解需求 → 检查一致性 → 生成问题
   - AI 发现不明确的地方，在评论区提问："登录方式是邮箱还是手机号？"
   - DK 选择"邮箱"，答案提交后即时反馈

6. **生成 BDD 规则** - 需求明确后，AI 生成 BDD 规则，任务流转到"待审核 BDD"状态

7. **审核 BDD** - DK 在 Web 页面查看 BDD 规则，确认无误后点击"通过"

8. **自动执行** - 任务流转到"实现中"，Symphony 调用 AI Agent 执行开发

9. **验收** - 任务流转到"待验收"，DK 查看验收报告，确认通过后流转到"完成"

**情感曲线：** 期待 → 好奇 → 参与感 → 信任 → 满足

### Journey 2: 日常使用 - 监控与干预

**主角：** DK，正在处理多个并行需求

**情境：** Symphony 服务常驻运行，DK 随时查看进度

**旅程叙事：**

1. **查看看板** - 打开 Web 页面，看到多个需求在不同状态：2 个"实现中"，1 个"待审核架构"，1 个"待验收"

2. **查看进度** - 点击"实现中"的需求，看到**人类可读的进度摘要**：
   > 📊 **当前进度**
   > - 阶段：编写测试用例
   > - 进度：3/10 测试用例
   > - 已用时间：12 分钟
   - 可展开查看详细日志（支持分页，每页 100 条）

3. **处理审核** - 点击"待审核架构"的需求，查看架构设计文档，提出修改意见，流转回"待设计"状态

4. **处理验收** - 收到通知，一个需求完成实现，流转到"待验收"，查看报告后通过

**情感曲线：** 掌控感 → 放心 → 参与感 → 满意

### Journey 3: 异常处理 - 需求澄清卡住

**主角：** DK，遇到 AI 无法理解的需求

**情境：** 提交了一个模糊的需求，AI 反复提问但始终无法明确

**旅程叙事：**

1. **创建需求** - DK 提交："优化性能"

2. **需求澄清循环** - AI 问："优化哪个模块？" DK 答："数据库查询" AI 问："优化目标是什么？" DK 答："更快" AI 问："具体指标？" DK 不确定...

3. **系统提示** - Web 页面显示提示卡片：
   > 💡 **需求质量提示**
   > 好的需求应包含：
   > - **目标**：想要达成什么效果
   > - **范围**：涉及哪些模块/功能
   > - **验收标准**：如何判断完成
   >
   > 示例："优化用户列表页面的数据库查询，目标是将响应时间从 2s 降到 500ms 以内"

4. **DK 行为分支**：
   - **分支 A**：根据提示补充需求，澄清继续
   - **分支 B**：点击"跳过澄清，直接流转"，任务标记为"需求不完整"

5. **硬性限制** - 超过 5 轮澄清后，系统强制流转：
   - 任务标记为"需求不完整"
   - 显示警告："需求澄清轮次已达上限，建议补充需求详情后重新提交"

**情感曲线：** 困惑 → 获得指引 → 明确方向 / 或 → 决断 → 接受后果

### Journey 4: 执行失败与恢复

**主角：** DK，AI 执行过程中遇到无法解决的错误

**情境：** 一个需求在"实现中"状态，AI 遇到测试失败无法修复

**旅程叙事：**

1. **正常执行** - 任务在"实现中"，DK 在看板上看到进度正常推进

2. **遇到错误** - AI 遇到测试失败，尝试自动修复 3 次后仍然失败

3. **自动流转** - 任务流转到"待人工处理"状态，Web 页面显示：
   > ⚠️ **需要人工介入**
   > - 阶段：测试用例编写
   > - 失败原因：测试 `TestUserLogin` 无法通过
   > - 已尝试修复：3 次
   > - 建议：检查测试数据或手动修复

4. **用户介入** - DK 查看失败详情，决定：
   - **选项 A**：手动修复代码，点击"继续执行"
   - **选项 B**：补充说明，点击"重新澄清需求"
   - **选项 C**：放弃该需求，流转到"已取消"

5. **恢复执行** - DK 手动修复后点击"继续执行"，任务恢复到"实现中"状态继续

**情感曲线：** 顺畅 → 惊讶 → 分析 → 决断 → 恢复

### Journey Requirements Summary

| 旅程 | 揭示的能力需求 |
|------|----------------|
| **首次使用** | CLI init/start、Web 看板、需求创建、需求澄清交互、BDD 审核、验收流程 |
| **日常使用** | 多任务并行、人类可读进度摘要、日志分页、审核干预、状态流转 |
| **异常处理（澄清）** | 需求质量提示、澄清轮次硬性限制、跳过澄清、需求标记 |
| **异常处理（执行）** | 自动错误检测、失败重试机制、待人工处理状态、恢复执行 |

### 状态机预览

```
待开始 → 需求澄清中 → 待审核BDD → 待审核架构 → 实现中 → 待验收 → 完成
             ↓              ↓           ↓          ↓
          (超限跳过)     (驳回)      (驳回)     (失败)
             ↓              ↓           ↓          ↓
          需求不完整      待设计      待设计    待人工处理
                                                       ↓
                                                    (修复后继续)
```

## Domain-Specific Requirements

### AI Agent 操作策略

| 维度 | 策略 | MVP | 后续演进 |
|------|------|-----|----------|
| **操作限制** | YOLO 模式 | 无限制 | 沙箱屏蔽 |
| **代码安全** | 信任 + VCS 回退 | 用户通过 git 回退 | - |
| **外部网络** | 允许 | ✅ | - |

### 工具集成

| 集成点 | MVP 状态 | 说明 |
|--------|----------|------|
| **Git** | ✅ 默认集成 | 提交为结束点，可配置推送 |
| **AI Agent CLI** | ✅ 核心调用方式 | claude code / opencode / codex |
| **IDE** | ❌ 不集成 | - |
| **CI/CD** | ❌ 不集成 | - |

### 上下文管理

| 维度 | 策略 |
|------|------|
| **上下文来源** | 只提供需求内容 + 阶段信息 |
| **AI Agent 自行获取** | 项目代码、文件结构等由 AI Agent CLI 自行处理 |
| **跨任务保持** | ❌ 不保持，状态通过 Beads 任务保留 |
| **调用传参** | 只传当前子任务内容 |

### 任务层级结构

```
总需求 Task (Beads Issue)
├── 状态流转: 待开始 → 进行中 → 完成/失败
├── 子阶段 Task (按工作流顺序)
│   ├── 需求澄清
│   ├── BDD 规则生成 ← blocked by 前置
│   ├── 架构设计 ← blocked by 前置
│   ├── 实现 ← blocked by 前置
│   └── 验收 ← blocked by 前置
└── 完成条件: 所有子阶段 Task 完成
```

**依赖管理：** 使用 Beads 原生 dependency 功能实现阶段间阻塞。

## Innovation & Novel Patterns

### Detected Innovation Areas

**1. 范式创新：Harness 驱动 AI**
- 从"人用 AI"转变为"harness 驱动 AI"
- Symphony 不是 AI 助手，而是声明式控制面，AI Agent 是执行单元
- 核心价值：**确定性**——规则锚定，结果不发散

**2. 方法创新：规则即代码**
- BDD 规则作为需求验收的必要条件
- TDD 规则作为代码提交的必要条件
- 规则不是文档，而是可执行的验证条件

**3. 流程创新：验证前置**
- 需求澄清阶段即确定验收标准（BDD）
- 架构设计阶段即确定实现标准（TDD）
- 实现阶段只验证不创建，红灯强制阻断

### Market Context & Competitive Landscape

**定位：Symphony 是 AI Agent 工具的编排框架**

| 层级 | 工具类型 | 代表产品 | 与 Symphony 的关系 |
|------|----------|----------|-------------------|
| **上游** | Issue Tracker | Beads, Linear | 需求来源，状态持久化 |
| **Symphony** | Harness 控制层 | - | 协调、约束、验证 |
| **下游** | AI Agent CLI | claude code, codex, opencode | 执行思考、实现 |

**差异化价值主张：**
- 直接用 AI Agent：输出不可控，可能偏离需求
- 用 Symphony + AI Agent：规则锚定，确定性输出

### Target Early Adopters

| 用户类型 | 特征 | 为何选择 Symphony |
|----------|------|-------------------|
| **有经验的开发者** | 理解 BDD/TDD 价值 | 愿意前期投入定义规则，换取后期确定性 |
| **追求确定性的场景** | 外包交付、客户项目、合规要求 | 需要可追溯的验证证据 |

### Validation Approach

| 创新假设 | 验证方式 |
|----------|----------|
| Harness 编排比直接用 AI Agent 更有效 | 对比实验：相同需求，两种方式完成 |
| 规则即代码能减少返工 | 度量返工率 vs 无约束场景 |
| 验证前置提高成功率 | 度量一次性验收通过率 |

### Risk Mitigation

| 风险 | 缓解策略 |
|------|----------|
| AI Agent 不遵守规则 | TDD 红灯强制阻断，无法进入下一阶段 |
| 规则本身错误 | 用户审核节点，规则可修改 |
| 流程过于刚性 | 支持跳过澄清、人工干预等弹性机制 |

### Product Positioning

**MVP 阶段：工具定位**
- 聚焦核心价值验证：确定性输出
- 单一工作流：需求澄清 → BDD → TDD → 实现 → 验收
- 不追求平台化，验证价值后再扩展

## Project Type Requirements

### CLI Commands

| 命令 | 用途 | 交互方式 |
|------|------|----------|
| `symphony init` | 初始化配置 | 交互式问答 |
| `symphony start` | 启动后台服务 | 脚本化 |
| `symphony --help` | 查看帮助 | 脚本化 |

### Configuration Management

**目录结构：**
```
.sym/
├── config.yaml           # 系统配置（tracker、agent、polling 等）
├── workflow.md           # 阶段编排定义（MVP 硬编码，未来可自定义）
└── prompts/
    ├── clarification.md  # 需求澄清 prompt
    ├── bdd.md           # BDD 生成 prompt
    ├── architecture.md  # 架构设计 prompt
    ├── implementation.md # 实现 prompt
    └── verification.md  # 验收 prompt
```

**配置格式：** YAML

**环境变量支持：** 支持变量替换（如 `$HOME`）

### Output Formats

| 输出类型 | CLI | Web UI |
|----------|-----|--------|
| 日志 | 结构化日志（slog） | 分页展示（每页 100 条） |
| 状态 | 状态码 + 简短消息 | 看板可视化 |
| 错误 | 结构化错误码 | 错误页面 |

### Error Code Design

采用结构化错误码格式：
```
<模块>.<错误类型>: <简短描述>

示例：
config.not_found: 配置文件不存在
tracker.auth_failed: Tracker 认证失败
agent.timeout: Agent 执行超时
prompt.parse_error: Prompt 解析失败
```

### Shell Completion

| 维度 | 状态 |
|------|------|
| **支持 shell** | bash, zsh, fish |
| **MVP 优先级** | P2（后续迭代） |

### Documentation & Examples

| 内容 | 优先级 |
|------|--------|
| 快速上手指南 | P2 |
| 命令参考文档 | P2 |
| 示例工作流模板 | P2 |

### Platform Support

| 平台 | MVP | 后续 |
|------|-----|------|
| Linux | ✅ | - |
| macOS | ✅ | - |
| Windows | ❌ | 后续迭代 |

### Configuration Behavior

| 行为 | MVP | 后续迭代 |
|------|-----|----------|
| **配置修改** | 重启服务生效 | 热加载 |
| **配置验证** | `start` 时自动验证 | 独立 `validate` 命令 |

### Testing Strategy

| 测试类型 | 范围 |
|----------|------|
| **单元测试** | 配置解析、参数验证、错误码生成 |
| **集成测试** | 完整命令执行流程、阶段流转 |
| **E2E 测试** | 用户旅程覆盖 |

## Project Scoping & Phased Development

### MVP Strategy & Philosophy

**MVP Approach:** Problem-solving MVP — 验证 harness + 约束能产出确定性结果

**核心验证目标：** 完成 Journey 1 全流程，证明「声明式约束 + AI Agent」能产出确定性输出。

**资源假设：** 完全 AI 开发，资源不受限，时间成本富足。

### MVP Feature Set (Phase 1)

**Core User Journeys Supported:**

Journey 1（首次使用）完整流程：
初始化 → 启动服务 → 创建需求 → 需求澄清 → BDD 规则审核 → TDD 规则审核 → 自动执行 → 验收

**Must-Have Capabilities:**

| 功能 | 必需 | 原因 |
|------|------|------|
| `symphony init` | ✅ | 入口必需 |
| `symphony start` | ✅ | daemon 必需 |
| Web 看板 | ✅ | 人审交互必需 |
| 需求澄清 | ✅ | Journey 1 核心步骤 |
| BDD 规则生成 + 审核 | ✅ | 核心约束锚点 |
| TDD 规则生成 + 审核 | ✅ | 核心约束锚点 |
| AI Agent 执行 | ✅ | 核心执行单元 |
| 验收报告 | ✅ | 结果验证必需 |
| Beads tracker 集成 | ✅ | MVP 唯一 tracker |

**Out of Scope for MVP:**

| 功能 | 原因 |
|------|------|
| 多任务并行监控（Journey 2） | 单任务验证优先 |
| 日志分页、进度详情展开 | 基础日志足够 |
| 异常处理的失败重试机制 | 先验证主流程 |
| 需求质量提示卡片（Journey 3） | 先验证主流程 |
| Linear tracker 支持 | Beads 优先 |
| 热加载配置 | 重启足够 |
| Shell completion | P2 |
| Windows 平台 | Linux/macOS 优先 |

### Post-MVP Features

**Phase 2 (Growth):**

| 功能 | 价值 |
|------|------|
| Linear tracker 支持 | 扩展 tracker 选项 |
| 多任务并行监控（Journey 2） | 提升日常使用体验 |
| 日志分页、进度详情展开 | 信息呈现优化 |
| 失败重试机制（Journey 4） | 异常恢复能力 |
| 需求质量提示卡片（Journey 3） | 澄清体验优化 |
| 热加载配置 | 配置修改更便捷 |
| Shell completion | CLI 体验提升 |
| 快速上手指南 + 命令参考 | 用户文档 |

**Phase 3 (Expansion):**

| 功能 | 价值 |
|------|------|
| 多 tracker 并存 | 灵活选择 |
| 团队协作支持 | 多人使用 |
| 自定义工作流模板 | 场景适配 |
| Windows 平台支持 | 平台覆盖 |
| 独立 `validate` 命令 | 配置管理优化 |
| 示例工作流模板库 | 快速上手 |

### Risk Mitigation Strategy

**Technical Risks:**

| 风险 | 缓解策略 |
|------|----------|
| AI Agent 不遵守规则 | TDD 红灯强制阻断，无法进入下一阶段 |
| 规则本身错误 | 用户审核节点，规则可修改 |
| AI 输出解析失败 | L1 自动重试机制 |

**Market Risks:**

| 风险 | 验证方式 |
|------|----------|
| Harness 编排比直接用 AI Agent 更有效 | MVP 完成后对比实验 |
| 用户不愿意前期投入定义规则 | 度量 BDD 审核通过率、澄清满意度 |

**Resource Risks:**

| 风险 | 缓解策略 |
|------|----------|
| 完全 AI 开发导致理解偏差 | 人工审核节点兜底 |
| 执行失败无法恢复 | Journey 4 待人工处理状态 + Git 回退

## Functional Requirements

### 初始化与配置管理

- FR1: 用户可以通过 CLI 交互式初始化项目配置
- FR2: 用户可以配置 tracker 类型
- FR3: 用户可以配置 AI Agent CLI 类型
- FR4: 系统可以生成 `.sym/` 目录结构
- FR5: 系统可以在启动时验证配置有效性
- FR6: 用户可以修改配置文件（重启生效）
- FR7: 用户可以配置澄清轮次上限
- FR8: 用户可以配置执行重试上限

### 任务生命周期管理

- FR9: 用户可以在 Web 页面创建新需求
- FR10: 系统可以自动管理任务状态流转
- FR11: 系统可以创建任务层级结构（父任务 + 子阶段任务）
- FR12: 系统可以管理任务间依赖关系（阶段阻塞）
- FR13: 用户可以取消进行中的需求

### 需求澄清

- FR14: 系统可以调用 AI Agent 进行需求理解
- FR15: 用户可以在 Web 页面查看 AI 提问
- FR16: 用户可以在 Web 页面提交回答
- FR17: 系统可以显示澄清进度步骤
- FR18: 系统可以限制澄清轮次上限
- FR19: 用户可以跳过澄清直接流转
- FR20: 系统可以标记需求为"不完整"
- FR21: 用户可以在提交回答后看到确认反馈

### 规则生成与管理

- FR22: 系统可以在需求明确后自动生成 BDD 规则
- FR23: 系统可以在架构设计后自动生成 TDD 规则
- FR24: 用户可以在 Web 页面查看 BDD 规则内容
- FR25: 用户可以在 Web 页面通过 BDD 规则
- FR26: 用户可以在 Web 页面驳回 BDD 规则
- FR27: 用户可以在 Web 页面查看架构设计（含 TDD 规则）
- FR28: 用户可以在 Web 页面通过架构设计
- FR29: 用户可以在 Web 页面驳回架构设计
- FR30: 系统可以将审核通过的规则作为约束条件

### 执行与监控

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

### 验收与报告

- FR41: 系统可以生成验收报告（含测试结果、BDD 通过情况）
- FR42: 用户可以在 Web 页面查看验收报告
- FR43: 用户可以通过验收并流转到"完成"
- FR44: 用户可以驳回验收并流转回"实现中"

### 异常处理与恢复

- FR45: 系统可以将执行失败的任务流转到"待人工处理"状态
- FR46: 用户可以在 Web 页面查看失败详情和建议
- FR47: 用户可以在手动修复后继续执行
- FR48: 用户可以重新澄清需求
- FR49: 用户可以放弃需求并流转到"已取消"

### 外部集成

- FR50: 系统可以从 Beads tracker 获取任务内容
- FR51: 系统可以获取需求澄清阶段的用户对话记录
- FR52: 系统可以将任务内容和对话记录作为 AI Agent CLI 输入参数
- FR53: 系统可以解析 AI Agent CLI 输出
- FR54: 系统可以处理 AI Agent CLI 错误
- FR55: 系统可以与 Beads tracker 集成更新任务状态
- FR56: 系统可以在任务完成后进行 Git 提交

## Non-Functional Requirements

### Integration

- NFR1: AI Agent CLI 执行可等待超过 24 小时（无硬性超时限制）
- NFR2: 系统可以检测 Beads CLI 可用性，不可用时返回明确错误

### Reliability

- NFR3: 服务崩溃后可通过 Beads 任务状态恢复现场
- NFR4: 配置修改后重启服务生效（无自动重启需求）

### Observability

- NFR5: 系统可以记录执行日志，支持故障排查

### Skipped Categories

| 类别 | 原因 |
|------|------|
| Performance | 暂无要求，后续优化 |
| Security | 本地服务，无敏感数据 |
| Scalability | 单用户本地服务 |
| Accessibility | 开发者工具 |