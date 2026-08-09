# Release Note — Notification Tx Outbox 下沉

> v1.1 · 阶段 A→E 全部完成 · 2026-08-09

## 范围

把以下三个关键通知生产者下沉到各自业务事务，确保业务状态变更与通知入箱严格原子：

| 域 | 触发方法 | 新 Tx 入箱方法 |
| --- | --- | --- |
| 工单创建 | `NotifyTicketCreated` | `NotifyTicketCreatedTx` |
| SLA 违规 | `NotifySLABreached` | `NotifySLABreachedTx` |
| SLA 预警 | `NotifySLAAlertLevelChanged` | `NotifySLAAlertLevelChangedTx` |
| 变更审批待办 | (新 producer) | `NotifyChangeApprovalRequiredTx` |
| 变更审批结论 | (新 producer) | `NotifyChangeApprovalDecidedTx` |

## 启动开关

`bootstrap/app.go` 注册 `EnableTxOutbox()`，与 incident/change 的 `EnableWorkflowOutbox()` 命名对齐。Tx Outbox 默认开启；旧 `EnableOutbox()` 保留为兼容路径（deprecated）。

## 同生同死契约

1. `txOutboxEnabled` 为 true 才入箱；未启用直接 fail-closed。
2. 入箱 SQL 与业务 SQL 共享 `*ent.Tx`：commit 同生，rollback 同死。
3. `occurrenceKey = notification:<tid>|<rcp>|<type>|<channel>|<key>`，sha256 截断。
4. Tx 内仅入箱 `in_app`；Email/SMS 留交 connector 异步处理。
5. TenantID 强校验：与所属实体一致，不匹配立即拒绝。

## 改动文件

```
itsm-backend/service/ticket_notification_service.go            +Notify*Tx (5)，EnableTxOutbox
itsm-backend/service/ticket_service.go                         CreateTicket 改单事务
itsm-backend/service/sla_monitor_service.go                    CreateViolation 入 tx
itsm-backend/service/sla_alert_service.go                      CreateAlertHistory 入 tx
itsm-backend/repository/ticket/repository.go + repository_impl.go   CreateWithTx 移植
itsm-backend/internal/bootstrap/app.go                         EnableTxOutbox 启动开关
itsm-backend/service/ticket_notification_tx_test.go            +5 用例（A/B）
itsm-backend/service/sla_violation_alert_tx_test.go            +7 用例（C）
itsm-backend/service/change_approval_notification_tx_test.go   +7 用例（D）
itsm-backend/service/notification_delivery_command_handler_test.go   +helper fixture
plans/notification-tx-outbox-v1.md                             规划+完成记录
```

## 测试

- `go test ./service/ -count=1`：35.148s PASS（19 个新增用例全过）
- `go vet ./service ./repository ./handlers`：clean
- `go test ./tests/contract ./tests/rbac`：PASS
- `controller/` 包 9 个既有失败已 `git stash` 对照复现，与本次改动无关

## 风险与对策

| 风险 | 等级 | 对策 |
| --- | --- | --- |
| Tx 内调外部下游 | 低 | 入箱仅一行 INSERT；Email/SMS 异步 |
| 双开关并存重复投递 | 中 | `EnableTxOutbox` 模式下 `EnableOutbox` 自动 fail-closed |
| 变更 rawDB 与 Ent drift | 中 | 已发 producer；UpdateChangeApproval 迁回 ent.Tx 时再闭环 |
| 重复投递 | 低 | occurrenceKey + 唯一约束 |

## 兼容性

- 旧 `EnableOutbox()` 兼容路径保留，加 Deprecated 注释。
- `NotifyTicketCreated` / `NotifySLABreached` / `NotifySLAAlertLevelChanged` 老方法保留，委托给 Tx API；行为对外不变。
- 业务调用方无需改动；通知侧幂等键格式不变。

## 后续工作

- 把 `changeApprovalServ.UpdateChangeApproval` 由 rawDB 迁回 `*ent.Tx`，并在 transaction 内调用 `NotifyChangeApprovalDecidedTx`，关闭变更审批侧通知缺口。
- `controller/` 包 9 个既有测试失败（缺少租户上下文）由 controller 重构阶段处理。
