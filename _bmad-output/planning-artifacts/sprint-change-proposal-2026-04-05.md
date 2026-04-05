# Sprint Change Proposal

**项目**: my-symphony
**作者**: DK
**日期**: 2026-04-05
**状态**: 待审批
**变更类型**: 架构重构 (Major)

---

## 1. Issue Summary

### 问题陈述

当前 Symphony 架构基于**静态 Prompt 文件驱动**的顺序阶段流转，存在以下核心限制：

| 限制 | 影响 |
|------|------|
| **Prompt 专业性不足** | 无法利用 BMAD 的专家角色能力（PM、Architect、QA 等） |
| **顺序流转效率低** | 无法并行处理独立的子任务 |
| **人工干预过多** | 5 个人工审核节点，自动化程度受限 |
| **缺乏迭代优化** | 无 GAN 式的评估-修复循环机制 |

### 触发原因

基于 Anthropic 和 OpenAI 的 Harness Engineering 实践，提出架构演进：

- **Anthropic**: Planner-Generator-Evaluator 三层架构
- **OpenAI**: 智能体优先开发、仓库即知识库

### 核心变更

```
旧架构: Prompt 文件 → 顺序阶段流转 → 人工审核 → 验收

新架构: Planner (BMAD Agents) → Generator (并行子任务) → Evaluator (多角度验证) → 迭代修复
```

### 核心设计原则

| 原则 | 说明 |
|------|------|
| **Planner 产出不可变** | 规划完成后，一次需求中不再修改 BDD 规则、架构设计、接口定义 |
| **Evaluator 只评估代码** | 不判断失败类型，只报告测试结果和 CR 意见 |
| **迭代只针对代码** | Generator 根据失败报告修复代码实现 |
| **规划问题下次修复** | 需求理解/规则/架构问题 → 创建新需求修复 |

**设计理由**：
- 简化架构复杂度，避免复杂的失败类型判断
- Planner 产出作为"契约"，一次需求中保持稳定
- 规划问题通过新需求修复，保持职责清晰

---

## 2. Impact Analysis

### 2.1 Epic 影响评估

| 原 Epic | 状态 | 处理方式 |
|---------|------|----------|
| Epic 1: 项目初始化与服务启动 | 🟡 保留 | 简化，移除 Prompt 文件相关 |
| Epic 2-9 | 🔴 废弃 | 完全重构为新的 P-G-E 结构 |

### 2.2 新 Epic 结构

```
Epic 1: 项目初始化与配置 (简化)
   ├── Story 1.1: CLI init 交互式初始化 (保留)
   ├── Story 1.2: 配置验证与管理 (保留)
   ├── Story 1.3: Beads Tracker 集成 (保留)
   └── Story 1.4: CLI start 启动后台服务 (保留)

Epic 2: Planner 模块实现 (新增)
   ├── Story 2.1: BMAD Agent 调用框架
   ├── Story 2.2: 需求澄清 (人工参与)
   ├── Story 2.3: BDD 规则生成
   ├── Story 2.4: 领域建模
   ├── Story 2.5: 架构设计
   └── Story 2.6: 接口设计

Epic 3: Generator 模块实现 (新增)
   ├── Story 3.1: Generator 调度器
   ├── Story 3.2: BDD 测试脚本生成 (并行)
   ├── Story 3.3: 集成测试生成 (并行)
   ├── Story 3.4: 单元测试生成 (并行)
   └── Story 3.5: 代码实现

Epic 4: Evaluator 模块实现 (新增)
   ├── Story 4.1: Evaluator 调度器
   ├── Story 4.2: BDD 验收执行
   ├── Story 4.3: TDD 验收执行
   ├── Story 4.4: 代码审计 (CR)
   └── Story 4.5: 代码风格评审

Epic 5: 迭代回流机制 (新增)
   ├── Story 5.1: 失败报告生成与传递
   ├── Story 5.2: 迭代计数与限制
   ├── Story 5.3: 转人工处理流程
   └── Story 5.4: 最大迭代次数配置

Epic 6: Beads 任务结构适配 (修改)
   ├── Story 6.1: 三类子任务结构
   ├── Story 6.2: 迭代任务创建与依赖
   └── Story 6.3: 任务状态流转适配

Epic 7: Web UI 适配 (修改)
   ├── Story 7.1: 三类任务看板展示
   ├── Story 7.2: 迭代进度展示
   └── Story 7.3: 失败报告展示
```

### 2.3 Artifact 冲突分析

#### PRD 修改需求

| 部分 | 修改程度 | 说明 |
|------|----------|------|
| Executive Summary | 🔴 重写 | 核心价值：从 Prompt 驱动 → BMAD Agent 驱动 |
| User Journeys | 🔴 重写 | 用户只参与需求澄清阶段 |
| 状态机预览 | 🔴 重写 | P-G-E 循环架构 |
| FR1-8, FR50-56 | 🟢 保留 | 初始化和外部集成相关 |
| FR9-49 | 🔴 重写 | 核心 FR 需要重新定义 |
| NFR1-5 | 🟢 保留 | 可复用 |

#### Architecture 修改需求

| 组件 | 修改程度 | 说明 |
|------|----------|------|
| Workflow Engine | 🔴 替换 | 替换为 P-G-E Orchestrator |
| Agent 调用层 | 🔴 重构 | BMAD Agent 调用框架 |
| Tracker 集成 | 🟡 扩展 | 子任务结构变更 |
| Server/Handlers | 🟢 保留 | 小幅修改适配新结构 |

#### 代码影响

| 目录 | 影响 | 处理 |
|------|------|------|
| `internal/workflow/` | 🔴 废弃 | 替换为 `internal/harness/` |
| `internal/agent/` | 🟡 重构 | 改为 BMAD 调用层 |
| `internal/tracker/` | 🟡 扩展 | 新增子任务结构 |
| `internal/orchestrator/` | 🟡 重构 | 集成 P-G-E |
| `internal/server/` | 🟢 保留 | 适配新 UI 需求 |

---

## 3. Recommended Approach

### 选择路径: Option 1 - 直接调整 (架构演进)

### 理由

1. **技术可行性高**
   - BMAD Agent 框架已就绪
   - Anthropic/OpenAI 实践已验证架构有效性
   - 核心模块（Tracker、Server）可复用

2. **业务价值明确**
   - 减少人工干预 → 提升自动化效率
   - 并行处理 → 加快交付速度
   - 迭代优化 → 提高代码质量

3. **风险可控**
   - 原 MVP 已验证基础能力
   - 可增量实现新 Epic
   - 不影响已完成的 Tracker 集成

### 实现策略

```
Phase 1: 基础设施 (Epic 1 + Epic 6)
   - 保留现有初始化流程
   - 适配 Beads 子任务结构

Phase 2: Planner 实现 (Epic 2)
   - BMAD Agent 调用框架
   - 需求澄清、BDD、架构、接口生成

Phase 3: Generator 实现 (Epic 3)
   - 测试脚本并行生成
   - 代码实现

Phase 4: Evaluator 实现 (Epic 4)
   - 测试验收
   - 代码审计

Phase 5: 迭代机制 (Epic 5)
   - 失败回流
   - 迭代限制

Phase 6: UI 适配 (Epic 7)
   - 新看板展示
```

### 工作量估算

| Epic | 预估 Story 数 | 复杂度 |
|------|---------------|--------|
| Epic 1 (简化) | 4 | 低 |
| Epic 2 (Planner) | 6 | 高 |
| Epic 3 (Generator) | 5 | 高 |
| Epic 4 (Evaluator) | 5 | 高 |
| Epic 5 (迭代) | 4 | 中 |
| Epic 6 (Beads) | 3 | 中 |
| Epic 7 (UI) | 3 | 低 |
| **总计** | **30** | - |

---

## 4. Detailed Change Proposals

### 4.1 新架构设计

```
┌─────────────────────────────────────────────────────────────────────┐
│                           Symphony Harness                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                         Planner                               │   │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐        │   │
│  │  │需求澄清 │→ │BDD规则  │→ │领域建模 │→ │架构+接口│        │   │
│  │  │(人工)   │  │(QA)     │  │(Arch)   │  │(Arch)   │        │   │
│  │  └─────────┘  └─────────┘  └─────────┘  └─────────┘        │   │
│  │                                                              │   │
│  │  ✅ 产出不可变：一次需求中不再修改                           │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                │                                     │
│                                ▼                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                        Generator                              │   │
│  │                                                                │   │
│  │   Phase 1: 测试编码 (并行)                                    │   │
│  │   ┌───────────┐  ┌───────────┐  ┌───────────┐               │   │
│  │   │G1: BDD    │  │G2: 集成   │  │G3: 单元   │               │   │
│  │   │测试脚本   │  │测试       │  │测试(TDD)  │               │   │
│  │   └───────────┘  └───────────┘  └───────────┘               │   │
│  │                                │                               │   │
│  │                                ▼                               │   │
│  │   Phase 2: 代码实现                                            │   │
│  │   ┌─────────────────────────────────────────────┐            │   │
│  │   │G4: 代码实现 (基于 G1-G3 的测试定义)          │            │   │
│  │   └─────────────────────────────────────────────┘            │   │
│  │                                ↑                               │   │
│  │                                │ 失败报告 (对话上下文)         │   │
│  │                                │ ⚠️ 只修复代码，不修改规则     │   │
│  └────────────────────────────────┼─────────────────────────────┘   │
│                                   │                                  │
│  ┌────────────────────────────────┼─────────────────────────────┐   │
│  │                        Evaluator                              │   │
│  │                                                               │   │
│  │   🎯 只评估代码实现，不判断失败类型                           │   │
│  │                                                               │   │
│  │  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────┐ │   │
│  │  │E1: BDD    │  │E2: TDD    │  │E3: 代码   │  │E4: 风格   │ │   │
│  │  │验收       │  │验收       │  │审计(CR)   │  │评审       │ │   │
│  │  └───────────┘  └───────────┘  └───────────┘  └───────────┘ │   │
│  │                                │                               │   │
│  │         ┌──────────────────────┼────────────────┐            │   │
│  │         ▼                      ▼                ▼            │   │
│  │     全部通过              有失败项          迭代超限(>5)      │   │
│  │         │                      │                │            │   │
│  │         ▼                      ▼                ▼            │   │
│  │     任务完成          Generator 修复代码    转人工处理        │   │
│  │                                │                               │   │
│  │                                ▼                               │   │
│  │                         迭代次数 + 1                           │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  📋 规划问题处理：创建新需求修复（不在当次需求中修改）              │
└─────────────────────────────────────────────────────────────────────┘
                                │
                                ▼
                    ┌─────────────────────┐
                    │   Beads Tracker     │
                    │                     │
                    │  父任务             │
                    │  ├── Planner 子任务 │
                    │  ├── Generator 子任务│
                    │  └── Evaluator 子任务│
                    └─────────────────────┘
```

### 4.2 Beads 子任务结构

```
父任务: SYM-001 用户登录功能
│
├── [Planner 类] - 只执行一次
│   ├── SYM-001-P1: 需求澄清 (人工参与)
│   ├── SYM-001-P2: BDD规则设计
│   ├── SYM-001-P3: 领域建模
│   ├── SYM-001-P4: 架构设计
│   └── SYM-001-P5: 接口设计
│
├── [Generator 类] - 迭代时新增
│   ├── SYM-001-G1: BDD测试脚本 (v1)
│   ├── SYM-001-G2: 集成测试 (v1)
│   ├── SYM-001-G3: 单元测试 (v1)
│   ├── SYM-001-G4: 代码实现 (v1)
│   │
│   ├── (迭代 2 新增)
│   ├── SYM-001-G5: 代码实现 (v2)
│   │
│   ├── (迭代 3 新增)
│   ├── SYM-001-G6: 代码实现 (v3)
│
├── [Evaluator 类] - 每次迭代新建
│   ├── SYM-001-E1: 评估验收 (v1) → 失败
│   │
│   ├── (迭代 2)
│   ├── SYM-001-E2: 评估验收 (v2) → 通过
```

### 4.3 关键参数配置

```yaml
# .sym/config.yaml 新增配置项

harness:
  # 迭代限制
  max_iterations: 5        # 最大迭代次数，默认 5
  
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
```

### 4.4 BMAD Agent 映射

| 阶段 | 子任务 | BMAD Agent | 职责 |
|------|--------|------------|------|
| **Planner** | 需求澄清 | `bmad-agent-pm` | 需求理解、澄清对话 |
| | BDD 规则 | `bmad-agent-qa` | 生成 Gherkin 格式规则 |
| | 领域建模 | `bmad-agent-architect` | 领域模型设计 |
| | 架构设计 | `bmad-agent-architect` | 技术架构设计 |
| | 接口设计 | `bmad-agent-architect` | API 接口定义 |
| **Generator** | BDD 测试脚本 | `bmad-agent-qa` | Gherkin → 可执行测试 |
| | 集成测试 | `bmad-agent-qa` | 集成测试代码 |
| | 单元测试 | `bmad-agent-dev` | TDD 单元测试 |
| | 代码实现 | `bmad-agent-dev` | 功能实现 |
| **Evaluator** | BDD 验收 | `bmad-agent-qa` | 执行 BDD 测试 |
| | TDD 验收 | `bmad-agent-qa` | 执行单元测试 |
| | 代码审计 | `bmad-code-review` | 多角度代码审查 |
| | 风格评审 | `bmad-editorial-review-prose` | 代码风格检查 |

---

## 5. Implementation Handoff

### 变更范围分类: Major

**原因**: 架构根本性重构，涉及核心模块重写。

### 交接计划

| 角色 | 职责 | 交付物 |
|------|------|--------|
| **PM/Architect** | 审批变更提案、确认 Epic 规划 | 本文档 |
| **Dev Team** | 实现新 Epic | 代码、测试 |
| **QA** | 验证新流程 | 测试报告 |

### 实现顺序建议

```
Week 1: Epic 1 (简化) + Epic 6 (Beads 适配)
Week 2-3: Epic 2 (Planner)
Week 4-5: Epic 3 (Generator)
Week 6: Epic 4 (Evaluator)
Week 7: Epic 5 (迭代机制)
Week 8: Epic 7 (UI 适配) + 集成测试
```

### 成功标准

| 标准 | 验证方式 |
|------|----------|
| Planner 能完成需求澄清 | 执行一个需求，验证输出 |
| Planner 产出不可变 | 验证规划完成后不允许修改 |
| Generator 能并行生成测试和代码 | 验证 G1-G4 子任务并行执行 |
| Evaluator 只评估代码 | 验证不进行失败类型判断 |
| 迭代只修复代码 | 验证失败报告只传递给 Generator |
| 最大迭代次数生效 | 验证第 6 次迭代转人工 |
| Beads 子任务正确创建和依赖 | 检查 Beads 任务结构 |
| 规划问题创建新需求 | 验证超限后人工处理流程 |

---

## 6. Approval

**请确认是否批准此 Sprint Change Proposal:**

- [x] 批准，开始实施
- [ ] 需要修改（请说明）
- [ ] 不批准

**审批人**: DK
**日期**: 2026-04-05
**备注**: 已批准，开始实施新架构

---

## Appendix: 参考文档

1. [Anthropic: Harness design for long-running application development](https://www.anthropic.com/engineering/harness-design-long-running-apps)
2. [OpenAI: Engineering in an Agent-First World](https://openai.com/index/harness-engineering/)
3. 现有 PRD: `_bmad-output/planning-artifacts/prd.md`
4. 现有 Architecture: `_bmad-output/planning-artifacts/architecture.md`