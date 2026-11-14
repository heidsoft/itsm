# E2E 跳过用例台账

> 状态：current ｜ 盘点日期：2026-11-14 ｜ 数据源：`itsm-frontend/tests/e2e/**/*.spec.ts`
> 本文由 `scripts/__tests__/e2e-skip-inventory.test.js` 自动对齐：**改了 spec 里的 skip 数量或新增/删除文件，不改本表会直接让门禁变红。**

## 1. 为什么需要台账

`itsm-frontend/tests/e2e/` 有 68 个 spec，但 CI 只在 `ga-gate.yml` 里跑
`ticket-type-full-chain.spec.ts` 一个（其余靠本地 `make test-e2e`）。15 个 spec 内共 51 处
`test.skip` / `describe.skip` / `test.fixme`。其中 27 处是**无原因、无条件的运行时裸
`test.skip()`**：当前置数据或页面元素不存在时静默跳过，测试报告里不会显示跳过理由。
这类跳过会持续制造"用例存在 = 覆盖到位"的错觉，因此先建台账、再分批治理，不一次性全接 CI
（一次性接入必然因缺数据/缺服务而假红）。

## 2. 分类定义

| 类别 | 含义 | 判定 | 治理方向 |
|:--|:--|:--|:--|
| `P` 计划占位 | 空壳 `test('...', () => {})` 或 `describe.skip` 包裹的待实现旅程 | 有明确 "will fill in" / PR 编号标注 | 保留但标注计划编号与负责人；实现一条删一条 |
| `G` 环境开关 | `test.skip(!process.env.X, 'reason')` 或能力探测后有 reason | 有条件表达式**且**有原因字符串 | 合规写法，登记到 CI 能力矩阵即可 |
| `D` 数据依赖裸跳过 | 测试体内 `if (!token) { test.skip(); return; }` | 无原因字符串 | 改为 `test.skip(cond, '原因')`，并补 fixture 使条件可满足 |

## 3. 逐文件清单（15 文件 / 51 处）

| 文件（相对 `itsm-frontend/tests/e2e/`） | 数量 | 类别 | 现状 / 触发条件 | 建议处置 |
|:--|--:|:--:|:--|:--|
| `golden/dashboard.spec.ts` | 11 | `P` | 1 个 `test.describe.skip('golden catalog stubs (PR-3.3 will fill in)')` + 10 个空体 `test.skip()` | 保留占位；PR-3.3 逐条实现，每实现一条同步更新本表 |
| `business-flows/approval-workflow.spec.ts` | 9 | `D` | 工单表格不可见 / 取不到 ticketId / 审批链节点缺失时裸跳过 | 补审批链 fixture；改为带原因的条件跳过 |
| `business-flows/sla-monitoring.spec.ts` | 7 | `D` | 登录拿不到 token、SLA 列表为空、策略项缺失时裸跳过 | 补 SLA 种子数据；至少先补 reason |
| `tickets.spec.ts` | 4 | `D` | 表格无数据行 / 详情页元素缺失时裸跳过 | 补工单 fixture |
| `problems.spec.ts` | 3 | `D` | `viewBtn` 不可见时裸跳过 | 补问题 fixture；补 reason |
| `flows/flow-ai-chat-stream.spec.ts` | 3 | `G` | `test.skip(Boolean(stream.error), 'AI/RAG 服务不可用: ...')` 等，含一条跨行调用 | 合规；登记 AI 服务为可选能力 |
| `flows/flow-ai-investigation.spec.ts` | 3 | `G` | AI 依赖探测（HTTP 503/业务 5003）与 `ai:read` 权限探测失败时带 reason 跳过 | 合规；2026-11-14 批次 A 新增的调查编排链路（API 契约 + UI 入口冒烟） |
| `performance-budgets.spec.ts` | 2 | `G` | `PERF_TESTS` / `PERF_EXPORT_PAYLOAD_JSON` 环境变量开关，均带 reason | 合规；在性能专项流水线显式打开 |
| `flows/flow-ai-tool-rbac.spec.ts` | 2 | `G` | `test.skip(true, '服务在当前环境不可用...')` 与 `test.skip(true, 'end_user ... 无工具权限...')` | ⚠️ **条件硬编码 `true`，等价于永久禁用**；应改为真实能力探测条件 |
| `business-flows/ticket-lifecycle.spec.ts` | 2 | `D` | 前置工单/状态不满足时裸跳过 | 补 fixture；补 reason |
| `soak.spec.ts` | 1 | `G` | `SOAK_TESTS` 环境变量开关，带 reason | 合规；SOAK 专项流水线显式打开 |
| `knowledge.spec.ts` | 1 | `D` | 知识库列表首个链接不可见时裸跳过 | 补知识文章 fixture |
| `flows/flow-ai-triage.spec.ts` | 1 | `G` | AI 依赖 HTTP 探测失败时带 reason 跳过 | 合规 |
| `business-flows/ticket-lifecycle-boundary.spec.ts` | 1 | `D` | 边界数据不存在时裸跳过 | 补边界 fixture |
| `business-flows/change-review-management.spec.ts` | 1 | `G` | `test.skip(!ADMIN_PASSWORD, 'E2E_ADMIN_PASSWORD not set ...')` | 合规；凭据走环境变量 |

小计：`P` = 11、`G` = 13、`D` = 27，合计 **51**。

## 4. 分批治理路线（本轮未执行）

1. **P1｜低风险**：27 处 `D` 统一补 reason（不改判定逻辑），并在 CI 报告开启 skip reason 展示。
2. **P1｜需实测**：`flow-ai-tool-rbac.spec.ts` 两处 `test.skip(true, ...)` 改为真实能力探测，
   否则「AI Tool RBAC 未启用」这条业务能力永远没有断言在跑。
3. **P1｜需整栈**：从 67 个 spec 中挑 5–10 个稳定 smoke 接进已起整栈的 `ga-gate.yml`；
   不要全量接入。
4. **P2**：`golden/dashboard.spec.ts` 的 11 个占位随 PR-3.3 逐条兑现，兑现一条即更新本表。

## 5. 口径边界

- 本表只统计 `itsm-frontend/tests/e2e/**/*.spec.ts`；根 `tests/` 目录的 40 个 tracked 文件
  与后端 Go 测试的 `t.Skip` 不在统计范围。
- 表格是**事实台账**，不是"已修复"声明：本轮未修改任何 E2E spec，未把新增 spec 接入 CI。
- 数量以源码为准；自测会在 CI `docs-gate.yml` 的 `Script Guard Tests` 中校验本表与源码一致。
