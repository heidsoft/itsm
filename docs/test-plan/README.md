# Test Plan 目录说明

> Status: historical collection。早期测试计划已迁移至 [`docs/archive/testing-reports/`](../archive/testing-reports/)。

本目录中保留的早期测试计划与测试结果中，默认账号、接口、状态机和"全部通过/立即上线"结论可能已过时。测试报告只证明其记录日期和 revision；缺少 Git SHA、运行镜像、数据库类型及未验证项时，不得用于当前发布签字。

## 已迁移文件

下列文件已迁移至 `docs/archive/testing-reports/`：

- `itst-test-plan-v1.md` → [`archive/testing-reports/itst-test-plan-v1.md`](../archive/testing-reports/itst-test-plan-v1.md)
- `itst-similar-bugs-test-plan-v1.md` → [`archive/testing-reports/itst-similar-bugs-test-plan-v1.md`](../archive/testing-reports/itst-similar-bugs-test-plan-v1.md)
- `itst-test-report-p0-v1.md` / `-p1-v1.md` / `-p2-v1.md` / `-summary-v1.md`

迁移原因：

- 默认账号（`admin/admin123`）已不再代表当前开发流程；
- 状态机、BPMN 字段命名、CMDB 拓扑 BFS 等关键假设已变更；
- 报告结论未附 Git SHA / 镜像 digest / 数据库类型，无法复现。

## 当前测试规范

- 可重复执行的测试方案：[`docs/testing/test-invariants.md`](../testing/test-invariants.md)
- 静态门禁（5.1-5.9）：[`docs/testing/static-analysis-gates.md`](../testing/static-analysis-gates.md)
- 覆盖率审计：[`docs/testing/coverage-audit.md`](../testing/coverage-audit.md)
- 文档权威层级：[`docs/documentation-governance.md`](../documentation-governance.md)

## 仍保留在 `docs/review/` 的报告

`docs/review/` 仍保留多份 2026-06 → 2026-07 评审报告（含规则迁移来源）。详见 [`docs/review/README.md`](../review/README.md)。
