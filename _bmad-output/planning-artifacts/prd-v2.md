---
stepsCompleted: ["step-01-init", "step-02-discovery", "step-02b-vision", "step-02c-executive-summary", "step-03-success", "step-04-journeys", "step-05-domain", "step-06-innovation", "step-07-project-type", "step-08-scoping", "step-09-functional", "step-10-nonfunctional", "step-11-polish"]
inputDocuments:
  - _bmad-output/planning-artifacts/sprint-change-proposal-2026-04-05.md
workflowType: 'prd'
documentCounts:
  briefCount: 0
  researchCount: 2
  brainstormingCount: 0
  projectDocsCount: 1
classification:
  projectType: CLI Tool / Daemon Service
  domain: Developer Tool / AI Agent Orchestration
  complexity: medium
  projectContext: brownfield
workflowPreferences:
  bddFormat: Gherkin
---

# Product Requirements Document - my-symphony (v2.0)

**作者:** DK
**日期:** 2026-04-05
**版本:** 2.0 (架构重构)

---

## Executive Summary

Symphony 是**基于 BMAD Multi-Agent 的自动化开发 Harness 工具**，采用 Planner-Generator-Evaluator 三层架构，实现从需求到代码的全流程自动化。

### 核心价值

| 维度 | 价值主张 |
|------|----------|
| **自动化** | BMAD 专家 Agent 驱动，而非静态 Prompt 文件 |
| **并行化** | Generator 子任务并行执行，提升效率 |
| **迭代优化** | GAN 式评估-修复循环，持续改进代码质量 |
| **最小人工干预** | 用户只参与需求澄清，其余自动流转 |

### 核心架构: P-G-E 模式

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Planner   │ ──▶ │  Generator  │ ──▶ │  Evaluator  │
│ (需求理解)  │     │ (代码生成)  │     │ (质量验证)  │
└─────────────┘     └─────────────┘     └─────────────┘
       │                   ▲                   │
       │                   │                   │
       │                   └───────────────────┘
       │                      迭代修复 (max 5)
       ▼
   产出不可变
```

### What Makes This Special

**与 v1.0 的核心差异：**

| 维度 | v1.0 | v2.0 |
|------|------|------|
| 执行引擎 | Prompt 文件 | BMAD Multi-Agent |
| 流程模式 | 顺序流转 | P-G-E 循环 |
| 人工节点 | 5 个审核点 | 1 个（需求澄清） |
| 任务执行 | 串行 | 并行 |
| 迭代机制 | 无 | 最多 5 次代码修复 |

---

## Project Classification

| 维度 | 分类 |
|------|------|
| **Project Type** | CLI Tool (init) + Daemon Service (runtime) |
| **Domain** | Developer Tool / AI Agent Harness |
| **Complexity** | Medium |
| **Project Context** | Brownfield (基于 v1.0 重构) |

---

## Success Criteria

### User Success

**Aha! 时刻：**
- 提出需求后，只参与澄清对话，其余全自动完成
- 看到多个子任务并行执行，加快交付速度
- 迭代修复自动进行，无需人工介入

**用户完成场景：**
```
提出需求 → 需求澄清(人工) → 等待自动执行 → 查看结果 → 完成
```

### Business Success

| 时间节点 | 成功定义 |
|----------|----------|
| **1 个月** | 新架构 MVP 上线，完成第一个真实开发任务 |
| **3 个月** | 验证 P-G-E 架构有效性，迭代优化效率 |
| **12 个月** | 完全依赖工具进行开发，人工干预率 < 10% |

### Technical Success

| 指标 | 目标 |
|------|------|
| 需求一次通过率 | > 60% |
| 迭代修复成功率 | > 80% (5 次迭代内) |
| 平均交付时间 | < 2 小时/需求 |
| 人工干预率 | < 10% |

### Measurable Outcomes

| 指标 | 度量方式 |
|------|----------|
| **需求澄清质量** | 用户满意度评分 |
| **代码一次通过率** | 无迭代即通过的任务比例 |
| **迭代效率** | 平均迭代次数 |
| **并行加速比** | 并行执行 vs 串行执行的时间比 |
| **使用率** | 日常开发任务中使用 Symphony 的比例 |

---

## Product Scope

### MVP (Phase 1)

**核心验证目标：** P-G-E 架构能产出确定性输出

**Must-Have Capabilities:**

| 功能 | 必需 | 原因 |
|------|------|------|
| BMAD Agent 调用框架 | ✅ | 核心执行引擎 |
| Planner 模块 | ✅ | 需求理解与规划 |
| Generator 模块 | ✅ | 测试与代码生成 |
| Evaluator 模块 | ✅ | 质量验证 |
| 迭代修复机制 | ✅ | 自动改进能力 |
| Beads 子任务结构 | ✅ | 状态持久化 |
| Web 看板 | ✅ | 进度可视化 |

**Out of Scope for MVP:**

| 功能 | 原因 |
|------|------|
| 自定义工作流模板 | 先验证固定流程 |
| 多项目并行 | 单项目验证优先 |
| 团队协作 | 单用户优先 |

---

## User Journeys

### 用户类型

| 类型 | 描述 |
|------|------|
| **开发者** | 唯一用户角色，既是使用者也是管理者 |
| **交互方式** | CLI (init/start) + 本地 Web 页面 (监控) |

### Journey 1: 完整需求流程

**主角：** DK，个人开发者

**旅程叙事：**

1. **提出需求** - DK 在 Web 页面提交："添加用户登录功能"

2. **需求澄清** - Symphony 调用 BMAD PM Agent：
   - 页面显示 PM Agent 提问
   - DK 回答问题
   - **这是唯一的人工参与节点**

3. **自动规划** - Planner 自动完成：
   - BDD 规则生成 (QA Agent)
   - 领域建模 (Architect Agent)
   - 架构设计 (Architect Agent)
   - 接口设计 (Architect Agent)
   - **产出不可变**

4. **并行生成** - Generator 并行启动：
   - BDD 测试脚本 (QA Agent)
   - 集成测试 (QA Agent)
   - 单元测试 (Dev Agent)
   - 代码实现 (Dev Agent)

5. **自动评估** - Evaluator 执行：
   - BDD 验收
   - TDD 验收
   - 代码审计
   - 风格评审

6. **迭代修复** - 如有失败：
   - 失败报告传递给 Generator
   - 代码修复 (最多 5 次)

7. **完成** - DK 收到通知，查看结果

**情感曲线：** 期待 → 参与 → 放心 → 满足

### Journey 2: 监控与进度查看

**主角：** DK，正在处理多个需求

**旅程叙事：**

1. **查看看板** - 打开 Web 页面，看到三类任务：
   - Planner 类任务 (已完成)
   - Generator 类任务 (进行中)
   - Evaluator 类任务 (待执行)

2. **查看详情** - 点击 Generator 任务，看到：
   - G1-G4 并行执行状态
   - 当前进度 (3/4 完成)
   - 执行日志

3. **迭代通知** - 收到迭代通知：
   - Evaluator 发现问题
   - Generator 正在修复
   - 迭代次数: 2/5

**情感曲线：** 掌控感 → 透明 → 信任

### Journey 3: 规划问题处理

**主角：** DK，需求理解有偏差

**旅程叙事：**

1. **需求澄清** - DK 提交需求，回答问题

2. **执行失败** - 迭代 5 次后仍有问题

3. **转人工处理** - 系统提示：
   > ⚠️ 迭代次数已达上限
   > 建议检查需求理解是否正确

4. **创建新需求** - DK 创建新需求修复规划问题

**情感曲线：** 顺畅 → 意外 → 分析 → 决断

---

## 状态机设计

### 三层状态流转

```
┌─────────────────────────────────────────────────────────────────┐
│                         任务状态                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Planner 状态                                             │   │
│  │                                                          │   │
│  │ 待开始 → 需求澄清中 → 规划中 → 完成                      │   │
│  │              ↓                                           │   │
│  │         (人工参与)                                       │   │
│  └─────────────────────────────────────────────────────────┘   │
│                           │                                      │
│                           ▼                                      │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Generator 状态                                           │   │
│  │                                                          │   │
│  │ 待开始 → 测试编码中 → 代码实现中 → 完成                  │   │
│  │              ↓              ↓                            │   │
│  │         (并行执行)    (迭代修复)                         │   │
│  └─────────────────────────────────────────────────────────┘   │
│                           │                                      │
│                           ▼                                      │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Evaluator 状态                                           │   │
│  │                                                          │   │
│  │ 待开始 → 评估中 → 完成                                   │   │
│  │              ↓                                           │   │
│  │         ┌────┴────┐                                      │   │
│  │         ↓         ↓                                      │   │
│  │      全部通过   有失败                                    │   │
│  │         │         │                                      │   │
│  │         ▼         ▼                                      │   │
│  │     任务完成   迭代修复                                   │   │
│  │                    │                                      │   │
│  │                    ▼                                      │   │
│  │               迭代 > 5 ?                                  │   │
│  │                 ↙    ↘                                   │   │
│  │               否      是                                  │   │
│  │               ↓       ↓                                   │   │
│  │          继续     转人工                                  │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 子任务状态

| 类别 | 子任务 | 状态流转 |
|------|--------|----------|
| **Planner** | P1-P5 | 待开始 → 进行中 → 完成 |
| **Generator** | G1-G4 | 待开始 → 进行中 → 完成 |
| **Generator (迭代)** | G5, G6... | 待开始 → 进行中 → 完成 |
| **Evaluator** | E1, E2... | 待开始 → 进行中 → 完成 |

---

## Domain-Specific Requirements

### BMAD Agent 集成

| 阶段 | Agent | 职责 |
|------|-------|------|
| **Planner** | `bmad-agent-pm` | 需求理解、澄清对话 |
| | `bmad-agent-qa` | BDD 规则生成 |
| | `bmad-agent-architect` | 领域建模、架构设计、接口设计 |
| **Generator** | `bmad-agent-qa` | BDD 测试脚本、集成测试 |
| | `bmad-agent-dev` | 单元测试、代码实现 |
| **Evaluator** | `bmad-agent-qa` | BDD/TDD 验收执行 |
| | `bmad-code-review` | 代码审计 |
| | `bmad-editorial-review-prose` | 代码风格评审 |

### 迭代机制

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `max_iterations` | 5 | 最大迭代次数 |
| 失败报告 | 对话上下文 | Generator 接收并修复代码 |
| 迭代范围 | 代码实现 | 不修改 Planner 产出 |

### Beads 子任务结构

```
父任务 (SYM-001)
├── Planner 子任务
│   ├── SYM-001-P1: 需求澄清
│   ├── SYM-001-P2: BDD规则
│   ├── SYM-001-P3: 领域建模
│   ├── SYM-001-P4: 架构设计
│   └── SYM-001-P5: 接口设计
├── Generator 子任务
│   ├── SYM-001-G1: BDD测试脚本 (v1)
│   ├── SYM-001-G2: 集成测试 (v1)
│   ├── SYM-001-G3: 单元测试 (v1)
│   ├── SYM-001-G4: 代码实现 (v1)
│   ├── SYM-001-G5: 代码实现 (v2) [迭代新增]
│   └── ...
└── Evaluator 子任务
    ├── SYM-001-E1: 评估验收 (v1)
    ├── SYM-001-E2: 评估验收 (v2) [迭代新增]
    └── ...
```

---

## Functional Requirements

### 初始化与配置管理

- FR1: 用户可以通过 CLI 交互式初始化项目配置
- FR2: 用户可以配置 BMAD Agent 启用/禁用
- FR3: 用户可以配置最大迭代次数
- FR4: 系统可以生成 `.sym/` 目录结构
- FR5: 系统可以在启动时验证配置有效性
- FR6: 用户可以修改配置文件（重启生效）
- FR7: 系统可以检测 BMAD Agent 可用性
- FR8: 系统可以检测 Beads CLI 可用性

### Planner 模块

- FR10: 系统可以调用 BMAD PM Agent 进行需求澄清
- FR11: 用户可以在 Web 页面查看 AI 提问并回答
- FR12: 系统可以调用 BMAD QA Agent 生成 BDD 规则
- FR13: 系统可以调用 BMAD Architect Agent 进行领域建模
- FR14: 系统可以调用 BMAD Architect Agent 进行架构设计
- FR15: 系统可以调用 BMAD Architect Agent 进行接口设计
- FR16: Planner 产出后不可修改（一次需求中）

### Generator 模块

- FR20: 系统可以并行启动多个 Generator 子任务
- FR21: 系统可以调用 BMAD QA Agent 生成 BDD 测试脚本
- FR22: 系统可以调用 BMAD QA Agent 生成集成测试
- FR23: 系统可以调用 BMAD Dev Agent 生成单元测试
- FR24: 系统可以调用 BMAD Dev Agent 实现代码
- FR25: Generator 测试编码完成后，顺序执行代码实现
- FR26: Generator 可以接收 Evaluator 失败报告并修复代码

### Evaluator 模块

- FR30: 系统可以执行 BDD 验收测试
- FR31: 系统可以执行 TDD 验收测试
- FR32: 系统可以调用 BMAD Code Review Agent 进行代码审计
- FR33: 系统可以调用 BMAD Editorial Review Agent 进行风格评审
- FR34: Evaluator 只评估代码，不判断失败类型
- FR35: Evaluator 生成失败报告传递给 Generator

### 迭代机制

- FR40: 系统可以记录迭代次数
- FR41: 迭代次数达到上限时转人工处理
- FR42: 每次迭代创建新的 Generator 和 Evaluator 子任务
- FR43: 迭代修复只针对代码实现，不修改 Planner 产出
- FR44: 用户可以在 Web 页面查看迭代进度

### 任务管理

- FR50: 用户可以在 Web 页面创建新需求
- FR51: 系统可以创建三类子任务 (Planner/Generator/Evaluator)
- FR52: 系统可以管理子任务依赖关系
- FR53: 用户可以在 Web 页面查看任务看板
- FR54: 用户可以在 Web 页面查看执行日志
- FR55: 系统可以在任务完成后进行 Git 提交

### 外部集成

- FR60: 系统可以与 Beads tracker 集成管理任务
- FR61: 系统可以从 Beads 获取任务状态
- FR62: 系统可以更新 Beads 任务状态

---

## Non-Functional Requirements

### Integration

- NFR1: BMAD Agent 执行可等待超过 24 小时（无硬性超时限制）
- NFR2: 系统可以检测 Beads CLI 可用性，不可用时返回明确错误
- NFR3: 系统可以检测 BMAD Agent 可用性

### Reliability

- NFR4: 服务崩溃后可通过 Beads 任务状态恢复现场
- NFR5: 配置修改后重启服务生效（无自动重启需求）

### Observability

- NFR6: 系统可以记录执行日志，支持故障排查
- NFR7: 系统可以记录迭代历史，支持问题追溯

### Performance

- NFR8: Generator 子任务应并行执行，提升效率
- NFR9: Web 页面应实时更新任务状态（SSE）

---

## Configuration

```yaml
# .sym/config.yaml

# 基础配置
project_name: my-project
tracker:
  type: beads
  
# Harness 配置
harness:
  max_iterations: 5        # 最大迭代次数
  
  # BMAD Agent 配置
  bmad:
    enabled: true
    agents:
      planner:
        - bmad-agent-pm
        - bmad-agent-architect
        - bmad-agent-qa
      generator:
        - bmad-agent-dev
        - bmad-agent-qa
      evaluator:
        - bmad-code-review
        - bmad-agent-qa
        - bmad-editorial-review-prose
```

---

## Risk Mitigation

| 风险 | 缓解策略 |
|------|----------|
| BMAD Agent 不可用 | 启动时检测，返回明确错误 |
| 迭代无法收敛 | 最大迭代次数限制，转人工处理 |
| 规划问题导致失败 | 创建新需求修复，不修改当次规划 |
| 并行任务冲突 | Generator Phase 1 并行，Phase 2 顺序 |

---

## Appendix

### 参考文档

1. [Anthropic: Harness design for long-running application development](https://www.anthropic.com/engineering/harness-design-long-running-apps)
2. [OpenAI: Engineering in an Agent-First World](https://openai.com/index/harness-engineering/)
3. Sprint Change Proposal: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-04-05.md`

### 术语表

| 术语 | 定义 |
|------|------|
| **P-G-E** | Planner-Generator-Evaluator 三层架构 |
| **Harness** | 编排 AI Agent 的控制框架 |
| **迭代** | Evaluator 失败后 Generator 修复代码的循环 |
| **子任务** | Beads 中归属于父任务的具体执行单元 |