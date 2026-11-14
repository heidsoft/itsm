# Plans 目录说明

> Status: current（索引更新于 2026-11-14，基于 HEAD `0de5fd68b`）
> 权威层级遵循 [`docs/documentation-governance.md`](../docs/documentation-governance.md)：
> `plans/` 是第 4 层的**计划与设计输入**，不证明功能已经实现。
> 计划里的 checkbox、退出标准或「完成报告」不等于已发布；判断实现状态必须回到
> 当前源码、数据库迁移、运行时 API 与最新 CI 结果。

## 1. 路线以 ROADMAP 为准

**当前路线只维护根目录 [`ROADMAP.md`](../ROADMAP.md)（Last synced 2026-11-08）**，
发布账本以 [`CHANGELOG.md`](../CHANGELOG.md) 为准。
本目录的蓝图若与 ROADMAP 冲突，以 ROADMAP 为准，并在对应计划里标记 `superseded` 或写明依赖关系。

`open-source-commercialization-2026q3-blueprint.md` 是 2026 Q3 的历史主线蓝图，
不再作为「当前主线」引用。

## 2. 执行中账本（下一轮开发的直接依据）

| 文件 | 状态 |
|:--|:--|
| [`service-layer-migration-plan.md`](service-layer-migration-plan.md) | Status: 执行中。L.1/L.3/L.4 权威读数以 `scripts/docs-gate/check-product-drift.sh` 的 C.8 硬模式输出、`product-surface-baseline.txt` 的 HEAD 值及文末批次追加为准；不在正文记进度数字 |
| [`bpmn-implementation-split-design.md`](bpmn-implementation-split-design.md) | G0 / G0-b 已落地（service 层整体归位 `internal/services/`）；**G1–G3 仍为设计、未落地** |

## 3. 设计蓝本（draft，未进入实施，保留为路线输入）

| 文件 | 主题 |
|:--|:--|
| [`ai-security-operations-blueprint.md`](ai-security-operations-blueprint.md) | 安全运营域 S1–S5 总蓝图 |
| [`exposure-answer-design.md`](exposure-answer-design.md) | 前置环节 S0：外部安全事件暴露面自答 |
| [`security-operations-feature-design.md`](security-operations-feature-design.md) | F1–F10 功能规格 |
| [`security-operations-product-deep-dive.md`](security-operations-product-deep-dive.md) | 对标 ServiceNow 的纵深设计 |
| [`product-remediation-plan-2026-10-03.md`](product-remediation-plan-2026-10-03.md) | 产品完善计划，待拍板 |
| [`incident-operations-platform-blueprint.md`](incident-operations-platform-blueprint.md) | 14 Epic 的实施蓝图（上游需求未纳入本仓库） |
| [`cmdb-commercialization-blueprint.md`](cmdb-commercialization-blueprint.md) | CMDB MVP 收敛 |
| [`production-data-initialization-blueprint.md`](production-data-initialization-blueprint.md) | 可审计 / 可重入的初始化体系 |
| [`production-readiness-industry-blueprint.md`](production-readiness-industry-blueprint.md) | 生产级部署与行业拓展 |

## 4. 历史审计 / 阶段计划（historical，仅作追溯）

| 文件 | 说明 |
|:--|:--|
| [`edge-feature-stability-audit-2026-10-02.md`](edge-feature-stability-audit-2026-10-02.md) | 10-02 一次性边缘功能盘点（220KB），结论已迁入 ROADMAP / CHANGELOG |
| [`product-feature-completeness-plan-2026-10-01.md`](product-feature-completeness-plan-2026-10-01.md) | 10-01 功能完整性计划；`scripts/` 下 4 个历史 runner 的唯一引用方 |
| [`scope-convergence-plan-2026-09-28.md`](scope-convergence-plan-2026-09-28.md) | 09-28 收敛研发计划 |
| [`open-source-commercialization-2026q3-blueprint.md`](open-source-commercialization-2026q3-blueprint.md) | Q3 商业化主线蓝图（已被 ROADMAP 取代） |

这些文件**不得作为完成证明或发布依据**。治理规范明确「仓库不设归档层」针对的是
「只描述某次历史运行的一次性报告」；计划与设计蓝本仍保留，但须按本文分组标注状态。

## 5. 维护约定

1. 新增计划在标题后标注 `Status: draft / current / superseded / historical` 之一。
2. checklist 打勾后不得写「已发布」，必须链接对应测试 / 部署证据，且证据来自当前 CI 或本次重跑。
3. 路线变更只改根目录 `ROADMAP.md`，不要在 plans 里另立「当前主线」。
4. `plans/` 不引用 `output/` 下的报告作为事实源（`output/` 已 gitignore，对全新 clone 是失效链接）。
5. 完整盘点与判定依据见本地报告 `output/scripts-tests-plans-audit-2026-11-14.md`（不入库）。
