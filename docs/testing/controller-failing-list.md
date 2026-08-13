# Controller 测试失败与零覆盖清单（v1.1 → v1.5）

> Companion to **PR-0.3** of the Test Improvement Plan. See plan section
> "阶段 0 / PR-0.3" for originating rationale.
>
> **Source evidence**
> - `plans/notification-tx-outbox-completion-report.md:50` — "controller/ 包 9 个既有失败已 `git stash` 对照复现，与本次改动无关"
> - `docs/生产就绪审计报告-2026-07-12.md` 第三节 — 65 个 controller 中 44 个无对应测试
> - `go test ./controller/ -count=1` (`output/coverage/backend-coverage.log` PR-0.1, re-run 2026-06-28) — 9 个失败明细

---

## A. 9 个失败用例（必须立即修复）

reproduce: `cd itsm-backend && GOTOOLCHAIN=auto go test ./controller/ -count=1`
跑出来的实际日志在 `output/coverage/backend-coverage.log`，关键摘要：

| # | 用例 | 文件 | 根因 | 修复 PR | 优先级 |
|---|------|------|------|---------|--------|
| F-1 | `TestSubmitTaskDecision_ApprovePath` | `bpmn_workflow_controller_test.go:212` | **缺租户上下文**：mock 中间件写 `tenant_id=0` → 业务路由返回 401 (code 2001) | PR-FIX-BPMN-1 | P0 |
| F-2 | `TestSubmitTaskDecision_RoutesNumericAndStringIDs` | `bpmn_workflow_controller_test.go:243` | 同上：缺租户上下文 | PR-FIX-BPMN-1 | P0 |
| F-3 | `TestSubmitTaskDecision_RejectWithCommentPasses` | `bpmn_workflow_controller_test.go:273` | 同上 | PR-FIX-BPMN-1 | P0 |
| F-4 | `TestGetApprovalHistory_TenantScoped` | `bpmn_workflow_controller_test.go:285` | 同上 | PR-FIX-BPMN-1 | P0 |
| F-5 | `TestGetApprovalHistory_PropagatesError` | `bpmn_workflow_controller_test.go:303` | 同上（return 500 被 401 拦截） | PR-FIX-BPMN-1 | P0 |
| F-6 | `TestTicketController_GetTicket` | `ticket_controller_test.go:240` | **`ticket_categories.code` 唯一约束冲突**：`createTestTenantAndUserForTicket` 用硬编码 `"incident"` 反复插入 | PR-FIX-TICKET-CAT | P0 |
| F-7 | `TestTicketController_ListTickets` | `ticket_controller_test.go:292` | 同上 | PR-FIX-TICKET-CAT | P0 |
| F-8 | `TestTicketController_UpdateTicket` | `ticket_controller_test.go:353` | 同上 | PR-FIX-TICKET-CAT | P0 |
| F-9 | `TestTicketController_DeleteTicket` | `ticket_controller_test.go:430` | 同上 | PR-FIX-TICKET-CAT | P0 |

### F-1..F-5 修复草案（PR-FIX-BPMN-1）

`bpmn_workflow_controller_test.go:329` 暴露了 `getBPMNTenantContext` 调试路径，
日志 `[DEBUG] getBPMNTenantContext: tenant_id=0, role=, user_id=0` 表明 mock 中间件
没有为这些用例设置 tenant。修复方法是在 setup helper 中：

1. 注入真实 tenant helper（与 `ticket_controller_test.go:78` 的
   `createTestTenantAndUserForTicket` 一致），而不是依赖 `X-Test-Role`。
2. 把 mock 中间件的 `Set("tenant_id", tenantID)` 中的 tenantID 改成具体非零值。
3. 在 PR 中增加 `defer cleanup` 删除产生的租户/用户/任务，保证隔离。

### F-6..F-9 修复草案（PR-FIX-TICKET-CAT）

`controller/ticket_controller_test.go:104-111` 的 `createTestTenantAndUserForTicket`
硬编码：

```go
_, err = client.TicketCategory.Create().
    SetName("incident").
    SetCode("incident").     // ← 硬编码，跨测试冲突
    SetTenantID(tenant.ID).
    Save(ctx)
```

修复方法（任选其一）：

- **方案 A（推荐）**：把 `SetCode("incident-" + uniqueID)` 让 code 含 `uniqueTestID()`；
- **方案 B**：使用 ent 的 upsert `OnConflict()`；
- **方案 C**：`TestMain` 增加 `client.TicketCategory.Delete().ExecX(ctx)` 清理。

---

## B. 41 个零测试 controller（按"业务关键 × 当前风险"排序）

> 与 `docs/生产就绪审计报告-2026-07-12.md` 第三节的 44 个无测试列表合并去重后剩余 41 个；
> 三个已在 PR-0.3 的 §A 失败用例间接拥有测试文件（`bpmn_workflow_controller_test.go` 5 个失败、`ticket_controller_test.go` 4 个失败），不算零覆盖。

| 排名 | 文件 | 模块 | 业务关键度 | 当前风险 | 建议 PR |
|------|------|------|---------|----------|---------|
| 1 | `change_approval_controller.go` | 变更 | P0 | ⚠⚠⚠ | 1.8 |
| 2 | `notification_controller.go` + `simple_notification_controller.go` + `notification_preference_controller.go` | 通知 | P0 | ⚠⚠ (告警链路) | 1.11 |
| 3 | `sla_policy_controller.go` + `sla_template_controller.go` | SLA | P0 | ⚠⚠⚠ (v1.6.7 路由修复点) | 1.7 |
| 4 | `connector_controller.go` | 连接器 | P0 (连接器市场 v1 重点) | ⚠⚠ (v1.1 P7/P8) | 1.10 |
| 5 | `service_controller.go` | 服务目录/请求 | P1 | ⚠⚠ | 1.12 |
| 6 | `release_controller.go` | 发布 | P1 | ⚠⚠ | 1.12 |
| 7 | `asset_controller.go` + `asset_license_controller.go` | 资产 | P1 | ⚠⚠ | 1.12 |
| 8 | `ticket_assignment_controller.go` + `ticket_assignment_smart_controller.go` | 工单分配 | P1 | ⚠ | 1.13 |
| 9 | `ticket_attachment_controller.go` + `ticket_comment_controller.go` + `ticket_dependency_controller.go` + `ticket_tag_controller.go` + `ticket_view_controller.go` + `ticket_workflow_controller.go` | 工单子域 (6 个) | P1 | ⚠ | 1.13 |
| 10 | `bpmn_ai_generator_controller.go` + `bpmn_dashboard_controller.go` + `bpmn_monitoring_controller.go` + `bpmn_process_trigger_controller.go` | BPMN 监控/AI | P2 | ⚠ (P3) | 阶段 2 |
| 11 | `prediction_controller.go` + `root_cause_controller.go` | AI 预测/根因 | P2 | ⚠ | 阶段 2 |
| 12 | `knowledge_integration_controller.go` | 知识整合 | P1 | ⚠ (知识发布闭环) | 1.13 |
| 13 | `msp_controller.go` | MSP 多租户 | P0 | ⚠⚠ (跨租户隔离) | 3.1 |
| 14 | `a2ui_ticket_controller.go` | a2ui 工单 | P2 | 低 | 阶段 5 |
| 15 | `application_controller.go` + `group_controller.go` + `menu_controller.go` + `system_config_controller.go` | 系统管理 | P2 | 低 | 5.x |
| 16 | `escalation_matrix_controller.go` | 升级矩阵 | P2 | ⚠ (B-6 复测点) | 1.7 |
| 17 | `feishu_controller.go` + `cloud_controller.go` + `vendor_controller.go` | 集成 | P2 | 低 | 1.10 |
| 18 | `survey_controller.go` + `provisioning_controller.go` | 收集/开通 | P2 | 低 | 阶段 5 |

> 注：表中"建议 PR"对应 Test Improvement Plan 的 PR 编号。详见计划文档"五、阶段 1"。

---

## C. 与已存在测试文件的 controller（覆盖率补强，PR-FIX-COVER）

下列 controller 已有测试文件，但覆盖率仍偏低（见 `coverage-audit.md`），
属于 PR-FIX-BPMN-1 / PR-FIX-TICKET-CAT 之外的"覆盖率回填"：

- `incident_controller_test.go` (81 行 → 偏低)
- `approval_controller_test.go` (332 行，但多场景未覆盖)
- `cmdb_topology_controller_test.go` (185 行，环/桥接场景未覆盖)
- `connector_controller_test.go` (216 行，但实例键/manifest gate 未覆盖)

→ 详见计划 PR-1.1, 1.5, 1.10

---

## D. 与 CI/验收对齐

- **CI 准入**：`.github/workflows/test-coverage-guard.yml` 已经拦住"改 controller 忘改测试"，所以 §B 的零测试 controller 任何后续 PR 都必须连带测试。
- **GA 门禁**：`.github/workflows/ga-gate.yml` 当前 `1%` floor，必须先把 §A 修复到 0 失败，再把 floor 上调到对应阶段的目标。
- **手动验收**：在 §A 修复合入前，开发不应再 `git stash` 这 9 个用例；应保持 `FAIL` 明示在 CI 上。

---

## E. 跟踪表（贴 PR 链接用）

| Task | Status | Owner | PR | Notes |
|------|--------|-------|----|-------|
| PR-FIX-BPMN-1 (F-1..F-5) | ✅ 已修（2026-08-12） | 后端 | — | `controller/bpmn_workflow_controller_test.go` 现以 `Set("tenant_id", tenantID)` 启用实际租户上下文，9 个用例全部 PASS |
| PR-FIX-TICKET-CAT (F-6..F-9) | ✅ 已修（2026-08-12） | 后端 | — | `createTestTenantAndUserForTicket` 已采用唯一化 code 并补 cleanup，4 个用例全部 PASS |
| §B 表中 PR-1.7 (sla) | 🟡 进行中（v1.5） | 后端 | — | 含 sla_policy/sla_template |
| §B 表中 PR-1.8 (change) | 🟡 部分（bridge_test 已存在） | 后端 | — | 见 PR-FIX-CMDB-BPMN 增量 |
| §B 表中 PR-1.10 (connector) | 🟡 进行中（v1.5） | 后端 | — | 含 manifest gate |
| §B 表中 PR-1.11 (notification) | ✅ 主体完成（v1.7） | 后端 | — | `notification_tx_test.go` 落地 + stage-1.11 控制器测试补强 |
| §B 表中 PR-1.12 (asset/release/service_catalog) | ✅ 主体完成（v1.7） | 后端 | — | release/asset/service_catalog controller_test 已补 |
| §B 表中 PR-1.13 (ticket 子域) | ⏳ v1.5 | 后端 | — | 含 11 个 ticket_*_service |

---

## F. v1.7 收尾补充（PR-FIX-CMDB-BPMN / PR-FIX-NOTIF-OUTBOX / PR-FIX-CMDB-TOPO）

针对 §B 剩余的高业务 × 高风险 controller，本阶段（v1.7）新增四块契约测试：

| 任务 | 测试文件 | 验收 |
|------|----------|------|
| 2.1 BPMN 桥接 | `handlers/change/bpmn_contract_test.go`（新建）| 审批/驳回/加签/撤回 状态机 + 超时升级 + 多实例会签 |
| 2.2 Approval Chain RBAC | `controller/approval_chain_controller_test.go`（扩展）| 多级 · 加签 · 撤回 · 跨租户 403/404 |
| 2.3 CMDB 拓扑 / 影响分析 递归爆栈 | `handlers/cmdb/topology_invariant_test.go`（新建）| 自环 · 50 层链 · 环早返回 · 跨租户隔离 |
| 2.4 Notification Outbox Tx | `controller/notification_tx_test.go`（扩展）| 入箱 → tx_commit 同生同死 · worker 幂等 · 跨租户不可见 |

这四块与 §A 的 9 个修复不重叠，是面向核心模块的契约防御。
| §B 表中 PR-3.1 (msp) | ⏳ v1.5 | 后端 | — | 跨租户隔离回归 |

---

## F. 复跑命令

```bash
# 复现 9 个失败
cd itsm-backend
GOTOOLCHAIN=auto go test ./controller/ -count=1 -timeout=120s \
  > /tmp/controller-failure.log 2>&1
grep '^--- FAIL' /tmp/controller-failure.log    # 应显示 9 行

# 复现 §B 零测试列表
cd itsm-backend/controller
for f in $(ls *.go | grep -v _test.go); do
  testf="${f%.go}_test.go"
  [ ! -f "$testf" ] && echo "$f"
done | wc -l    # 41 (含 api_contract_test 不计入)
```
