# Review 目录说明

> Status: historical collection。

本目录中的评审报告是特定日期的代码审查快照，不是当前架构或发布事实源。报告里的问题可能已经修复，标记“已完成”的能力也可能在后续改动中回归。

进行新开发前，请先阅读 [`docs/documentation-governance.md`](../documentation-governance.md)，再以当前源码、测试和运行时证据验证报告结论。仍有效的架构规则应迁移到 `docs/architecture/` 或 `docs/product/`，仍有效的工作项应进入根目录 roadmap 或 issue。

## 迁移状态（v1.5）

已迁移的评审结论：

- BPMN 表达式拒绝策略、变量原子化、ServiceTask Handler、CMDB 跨租户隔离 / 拓扑深度 → [`docs/architecture/workflow-cmdb-invariants.md`](../architecture/workflow-cmdb-invariants.md)
- 主流程闭环 / 契约治理 / AI 默认门禁 / 测试夹具唯一性 → [`docs/architecture/domain-ownership.md`](../architecture/domain-ownership.md) §跨领域不变量
- 测试优先级 / 夹具不变量 / Jest 退出异常 / API smoke 17 项 → [`docs/testing/test-invariants.md`](../testing/test-invariants.md)
- 前端 UX 不变量（error.tsx / loading.tsx / 默认凭据 / ErrorBoundary 跳转） → [`docs/testing/static-analysis-gates.md`](../testing/static-analysis-gates.md) §5.6–5.9

## 已迁移文件

- `servicenow-benchmark-2026-06-18.md` → [`docs/archive/reviews/servicenow-benchmark-2026-06-18.md`](../archive/reviews/servicenow-benchmark-2026-06-18.md)
  - 原因：报告中的 v1.1 / v1.2 / v1.4 / v2.0 / v3.0 路线已完全被 [`ROADMAP.md`](../../ROADMAP.md) 取代；ServiceNow GAP 列表已迁移到 [`docs/product/`](../product/) 阶段改进计划。

## 仍保留的报告

| 报告 | 状态 | 保留理由 |
|---|---|---|
| `architecture-review-2026-06-14.md` | historical | BPMN / CMDB 不变量主要迁移来源；保留作历史 |
| `frontend-ux-review-2026-06-19.md` | historical | §5.6–5.9 门禁主要迁移来源 |
| `module-function-retrospective-2026-07-10.md` | historical | 主流程闭环 / 契约治理迁移来源 |
| `system-function-review-result-2026-07-01.md` | historical | F-1..F-9 测试夹具修复、GA readiness 12 modules 基线 |
| `system-function-review-checklist-2026-07-01.md` | historical | checklist 原件 |
| `commercial-readiness-acceptance-report-2026-06-18.md` | historical | 商用验收快照 |
| `browser-e2e-test-report-2026-06-18.md` | historical | 浏览器 E2E 烟测基线 |
| `browser-functional-test-report-2026-06-20.md` | historical | 浏览器功能测试 |
| `deep-business-test-report-2026-06-18.md` | historical | 深度业务流测试（默认账号过时，但仍含 API 验证记录） |

