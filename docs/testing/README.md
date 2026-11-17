# 测试文档

> Status: current

这里放当前仍有参考价值的测试方案和测试报告。

## 当前入口

- [角色视角产品测试方案](./role-based-product-test-plan.md)
- [E2E 约定](./e2e-conventions.md)、[测试不变量](./test-invariants.md)、[覆盖率口径](./coverage-baseline.json)
- [E2E 跳过用例台账](./e2e-skip-inventory.md) — 14 文件 / 48 处 skip 的分类与治理路线，由门禁自测自动对齐

## 测试架构（权威分层，2026-11-14 复核）

六层并存，但**只有前五层部分进入 CI**，E2E 只有 1 个 spec 进入 CI：

| 层 | Runner | 触发入口 | 是否发布门禁证据 |
|:--|:--|:--|:--|
| Go 单元 / 集成（802 文件） | `go test` | `backend-ci.yml` | 是 |
| RLS 专项 | `go test -tags integration_rls` | `rls-golden-path.yml` | 是（独立 workflow） |
| Jest 单元（251 文件，`src/lib` 80% 阈值） | `jest` | `frontend-ci.yml` | 是 |
| 门禁脚本自测（12 文件 / 90 用例） | `node --test` | `docs-gate.yml` → `Script Guard Tests` | 是 |
| 远端 CI 一致性（本地诊断） | `make ci-parity` / `--strict` | 无（本地工具，不属发布门禁） | 否 |
| 整栈 smoke + TicketType E2E | curl / `playwright test` | `ga-gate.yml` | 是（仅该 1 个 spec） |
| 其余 66 个 E2E spec、根 `tests/` 全目录 | 本地手动 | `make test-e2e` / `python3 run_tests.py` | **否** |

`make test` 覆盖 Go + Jest + 门禁脚本自测三层，不含 E2E / RLS / 整栈。
判定「测试通过」时必须写明是哪一层、由哪个 workflow 触发；
把本地手动用例或未被 CI 执行的 spec 计入通过数属于口径失真。

分层归属、断言强度、skip、凭据与反向验证的编写规范见
[`tests/README.md` §4.1](../../tests/README.md)；脚本自测的接线状态见
[`scripts/README.md` §2.6](../../scripts/README.md)。

## 维护规则

- 可重复执行的测试方案放在本目录。
- 一次性测试报告保留日期，结论迁移到规范文档后物理删除，仓库不设归档目录。
- 自动化脚本放在 `docs/scripts/` 或 `tests/`，不要混在测试报告里。
