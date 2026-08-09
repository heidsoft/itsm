# v1.1 通知事务性 Outbox 下沉计划 — 工单 / SLA / 变更审批

## 1. 目标与范围

下一阶段把以下三个关键通知生产者下沉到各自业务事务中，确保**业务状态变更与通知入箱**严格原子：

| 域 | 通知触发点 | 当前调用点 | 当前事务性 |
|----|-----------|------------|-----------|
| 工单 | `NotifyTicketCreated` | `service/ticket_service.go:228` | ❌ 业务 `repo.Create` 已提交后才 enqueue，进程崩溃会丢通知 |
| SLA | `NotifySLABreached` | `service/sla_monitor_service.go:253` | ❌ `SLAViolation.Create` 已提交后才 enqueue |
| SLA | `NotifySLAAlertLevelChanged` | `service/sla_alert_service.go:509` | ❌ `SLAAlertHistory.Create` 已提交后才 enqueue |
| 变更 | 审批提交/批准/驳回 | `handlers/change/service.go:149` 仅有日志；`ProcessApproval` 同理 | ❌ 当前实现根本没发通知；`SubmitForApproval` 已用 raw `db.BeginTx`，但 outbox 写入需在同一 tx |

非目标：本阶段不重写 `NotifyTicketAssigned`、`NotifyTicketStatusChanged`、`NotifyTicketCommented`（这些将由后续阶段按统一改造策略下沉）；不引入新的连接器或新渠道。

## 2. 现有底座（已就绪、可复用）

1. `internal/commandbus/commandbus.go`
   - `Enqueue(ctx, client, req)`：客户端级入箱
   - `EnqueueTx(ctx, tx, req)`：**事务级入箱**（已用于 `IncidentService.CreateIncident`、`Change.EntRepository.CreateWithWorkflowCommand`）
   - `Worker.RunOnce/Run`：基于 fencing token + lease 的可靠重试/dead-letter

2. `service/ticket_notification_service.go`
   - 已实现 `EnableOutbox()` 开关
   - `enqueueNotificationDeliveries` 调用 `commandbus.Enqueue(ctx, s.client, ...)` —— **但走 client 而非 tx，是当前漏洞所在**
   - `enqueueTicketNotificationCommand(ctx, client, ...)`：签名只接受 client

3. 旁路证据
   - `service/incident_service.go:208-218`：`tx.Incident.Create().Save(ctx)` → `commandbus.EnqueueTx(ctx, tx, ...)` → `tx.Commit()`，是 **黄金参考实现**
   - `handlers/change/repository_impl.go:134-167`：`CreateWithWorkflowCommand` 用同一模式处理 change 创建

4. 既有测试 `service/notification_delivery_command_handler_test.go` 已覆盖：
   - 幂等性（重复 enqueue 仅 1 条 operational_command）
   - 跨租户拦截
   - 重试 + dead-letter + 审计脱敏

## 3. 设计要点（领域切片式）

### 3.1 通用约定

- **三件套下沉原则**：业务主表写入 + SLA/Approval 关联写入 + `commandbus.EnqueueTx` 在同一 `*ent.Tx` 中；任一失败 → 整体回滚。
- **Tx-Aware 入箱 API**：新增 `EnqueueTx(ctx, tx, ...)` 重载给所有"通知生产者"，避免在 controller/service 之间漏传 tx 导致静默退化到 client 路径。
- **拒绝 7B 双写**：通知 enqueue 一旦下沉到 tx，禁止再在 tx 外补发"保底"通知，否则会重复投递。
- **幂等键稳定**：现有 `notification:<tenantID>|<ticketID>|<recipientID>|<type>|<channel>|<idempotencyKey>` 哈希保留，SLA 端采用 `sla:<ticketID>|<type>|<periodBucket>`，change 端采用 `change:<changeID>|<action>|<approvalID|level>`，保证 retry 不重复。

### 3.2 改动面（最小集）

| 层 | 文件 | 关键变更 |
|----|------|----------|
| 命令总线 | `service/ticket_notification_service.go` | `enqueueTicketNotificationCommand` 增加 `*ent.Tx` 重载；新增 `EnqueueCreatedTicketNotificationTx`、`EnqueueSLABreachedTx`、`EnqueueSLAAlertTx` 三个对外 Tx 方法 |
| 通知 Tx API | `internal/commandbus/commandbus.go` | 无需改动（`EnqueueTx` 已存在） |
| 工单事务 | `repository/ticket/repository_impl.go` + `service/ticket_service.go` | 新增 `CreateWithOutbox(ctx, params, tx, ...)`；`CreateTicket` 改为：开 tx → repo 写入 → SLA 写 deadline → ApprovalService.TriggerApproval（同 tx 内） → notification enqueue → commit |
| SLA 事务 | `service/sla_monitor_service.go` + `service/sla_alert_service.go` | `createViolation` 改为接收 `*ent.Tx`；`checkAndCreateAlert` 接收 `*ent.Tx`；外层 `CheckTenantSLAs` 在每次创建违规前开 tx |
| 变更事务 | `handlers/change/repository_impl.go` + `handlers/change/service.go` | `SubmitForApproval` 在已有 raw tx 中加 `tx.ExecContext("INSERT INTO operational_commands ...")` 或迁移到 Ent Tx；`ProcessApproval` 同样补 enqueue |
| 启动装配 | `internal/bootstrap/app.go` | 给 `ticketNotificationService` 增加 `EnableTxOutbox()`（区别于 `EnableOutbox()`），与现有 `incidentService.EnableWorkflowOutbox()` 平行 |
| 测试 | `service/notification_delivery_command_handler_test.go` + 新增 `service/ticket_create_outbox_test.go`、`service/sla_violation_outbox_test.go`、`handlers/change/submit_for_approval_outbox_test.go` | 验证：tx rollback 不留 operational_command；tx commit 后 worker 投递成功 |

### 3.3 关键设计取舍

- **不入坑：tx 长度控制**。SLA monitor 循环中不要把整个 pageSize 放进一个 tx；按"每张 ticket 的违规检测 = 一个 tx"粒度提交，符合现有 `createViolation` 边界。
- **不引入二套 outbox 表**。继续复用 `operational_commands`（已 unique-`(tenant_id, command_type, idempotency_key)`），不同领域用不同 `aggregate_type` 隔离（`ticket` / `sla_violation` / `change_approval`）。
- **handler 复用**。继续复用现有 `NotificationDeliveryCommandHandler.Handle`，新增 handler 类型会带来无谓复杂度；SLA breach 的特殊内容（含 `exceeded_minutes`）放 payload。
- **审计与可观测性**。命令成功投递后，`notification_delivery` 表会写入 `operational_command_id`；事务性下沉不影响审计完整性，事务回滚时连审计行都不会出现，正好满足"要么全发要么全不发"的语义。
- **兼容现有 `EnableOutbox()`**。保留旧 client-level 路径给非事务调用方（如 CC、外部触发），但生产组装只走 `EnableTxOutbox()`，并删去 `ticket_service.go` line 226-231 的非事务 fallback，与 incident/change 已对齐。

## 4. 执行计划

### 阶段 A — 通知 Tx 入箱 API（1.0 天）

**A1. 扩展 `enqueueTicketNotificationCommand` 支持 `*ent.Tx`**
- 文件：`itsm-backend/service/ticket_notification_service.go`
- 新增 `enqueueTicketNotificationCommandTx(ctx, tx, ...)`，与 client 版并列；不破坏既有 `enqueueNotificationDeliveries`
- 新增公开方法：
  - `EnqueueCreatedTicketNotificationTx(ctx, tx, ticket)` → 计算接收人 + 内容 + 调 `EnqueueTx` per recipient
  - `EnqueueSLABreachedTx(ctx, tx, ticketID, violationType, exceededMinutes, tenantID)` → 站内记录 + enqueue
  - `EnqueueSLAAlertTx(ctx, tx, ticketID, level, percentage, tenantID)` → enqueue
- 幂等键生成逻辑复用现有 `sha256` 摘要

**A2. 启动装配开关**
- 文件：`itsm-backend/internal/bootstrap/app.go`
- 增加 `EnableTxOutbox()` 标记，bootstrap 内对 ticket 域启用；保持 `EnableOutbox()` 给非事务调用方

**A3. 测试**
- 现有 `TestTicketNotificationServiceEnqueuesDurableDelivery` 保留
- 新增 `TestTicketNotificationEnqueueTxRollsBack` 验证：tx rollback 时 `operational_commands` 表无对应行

### 阶段 B — 工单创建下沉（1.5 天）

**B1. Repository 增加事务化创建**
- 文件：`itsm-backend/repository/ticket/repository.go` + `repository/ticket/repository_impl.go`
- 新增接口 `CreateWithTx(ctx, tx, params, tenantID) (*Ticket, error)`
- 实现：复用 `Create` 内部 SQL builder，但传 `tx` 而不是 client；其它查询保持不变

**B2. `TicketService.CreateTicket` 重构**
- 文件：`itsm-backend/service/ticket_service.go`
- 步骤：
  1. `tx, err := s.client.Tx(ctx)`
  2. 通过 `tx.Client()` 或 `repo.CreateWithTx` 写 ticket
  3. 同 tx 内：`s.slaSvc.UpdateSLADeadlinesTx(...)`（如果存在）或直接走 `tx.Ticket.Update...`
  4. 同 tx 内：`s.approvalSvc.TriggerApprovalTx(...)`（需要新方法）或 enqueue 一个 `CommandApprovalTrigger` 命令
  5. 同 tx 内：`s.notificationSvc.EnqueueCreatedTicketNotificationTx(ctx, tx, ticket)`
  6. `tx.Commit()`
- 现有 line 248-256 的 BPMN workflow 触发从 `go func()` 改为 enqueue（沿用 incident 模式）
- 现有 line 262-298 的飞书同步保持异步（外部系统副作用，不适合事务）

**B3. 测试**
- `service/ticket_create_outbox_test.go`
  - 验证 ticket 创建成功 + `operational_commands` 中存在 `ticket:<id>:notify:created` 行
  - 验证 ticket 创建失败（FK 错误）时无 operational_command 行
  - 验证 worker.RunOnce 后产生 `ticket_notifications` 行

### 阶段 C — SLA 违规/预警下沉（1.5 天）

**C1. SLA 违规下沉**
- 文件：`itsm-backend/service/sla_monitor_service.go`
- 修改 `createViolation(ctx, t, ...)` → `createViolationTx(ctx, tx, t, ...)`：把 `SLAViolation.Create` 改用 `tx.SLAViolation.Create` + `EnqueueSLABreachedTx`
- `CheckTenantSLAs` 中每个 ticket 独立开 tx，调用 `createViolationTx`

**C2. SLA 预警下沉**
- 文件：`itsm-backend/service/sla_alert_service.go`
- 修改 `checkAndCreateAlert(ctx, ...)` → `checkAndCreateAlertTx(ctx, tx, ...)`：把 `SLAAlertHistory.Create` + `NotifySLAAlertLevelChanged` 合并到 tx
- `CheckAndTriggerAlerts` 和 `TriggerSLAWarning` 中按 ticket 粒度开 tx

**C3. 幂等键**
- SLA breach：`sla:<tenantID>:<ticketID>:<violationType>:<violationTimeBucket>`
- SLA alert：`alert:<tenantID>:<ticketID>:<ruleID>:<level>:<bucket>`

**C4. 测试**
- `service/sla_violation_outbox_test.go`
  - 验证 SLAViolation + operational_command 同步落库
  - 验证 rollback 都不留
  - 验证 worker 处理后通知成功

### 阶段 D — 变更审批通知（2.0 天）

**D1. 通知生产者实现**
- 新文件：`itsm-backend/handlers/change/notification.go`
- `ChangeApprovalNotificationProducer` 类型：
  - `EnqueueSubmitted(ctx, tx, change, approverIDs)`：tx 内 enqueue per approver
  - `EnqueueApproved(ctx, tx, change, finalApproverID)`：tx 内 enqueue to requester + creator
  - `EnqueueRejected(ctx, tx, change, rejecterID, reason)`：tx 内 enqueue to requester

**D2. 接入 `SubmitForApproval`**
- 文件：`itsm-backend/handlers/change/repository_impl.go` + `handlers/change/service.go`
- `SubmitForApproval` 当前用 raw `db.BeginTx`：
  - 选项 A（推荐）：在 raw tx 中加 `INSERT INTO operational_commands (...)` 直写，复制 commandbus.EnqueueTx 的核心逻辑
  - 选项 B：迁移到 Ent Tx（成本更高，要重写 repository_impl 中多处 raw SQL）
  - 选项 A 风险：与 Ent schema 耦合更弱，需手工维护列名一致性，但与现有 change 域的 raw-SQL 风格一致
- 决策：**选 A**，与现有风格保持一致
- `Service.SubmitChange`：调用 `repo.SubmitForApproval`（已 tx 内） → 在 repo 内部新增 `EnqueueSubmittedTx`

**D3. 接入 `ProcessApproval`**
- 文件：`handlers/change/service.go:202-243`
- 当前 `ProcessApproval` 在 rejected 分支已用 Ent Tx（line 222），可在 commit 前插入 `EnqueueRejectedTx`
- 在 approved 分支 `checkAndTransitionChange` 已写入 change 为 approved，但**未使用 tx**，需：
  - 选项 A：把 `checkAndTransitionChange` 改为 tx 内 `Update` + enqueue
  - 选项 B：在 repo 层新增 `UpdateAndEnqueueApprovedTx`

**D4. 幂等键**
- submit：`change:<changeID>:submit:<txToken>`（每次 submit 都唯一）
- approve：`change:<changeID>:approved:<finalTimestampBucket>`
- reject：`change:<changeID>:rejected:<approvalID>`

**D5. 测试**
- `handlers/change/submit_for_approval_outbox_test.go`
  - 验证 SubmitChange 提交后 operational_command 入箱
  - 验证 raw tx rollback 不留 operational_command
  - 验证 approve → reject 路径通知正确

### 阶段 E — 回归与发布（0.5 天）

**E1. 全量回归**
- `cd itsm-backend && go test ./...` 必须通过
- `cd itsm-frontend && npm run type-check && npm test` 必须通过

**E2. 文档更新**
- `docs/architecture/notification-outbox.md`（新建）记录下沉范围、tx 边界、retry 语义
- `AGENTS.md` 在"AI-Native Engineering Rules"段补一笔：`tool_invocation + notification_enqueue` 必须经由 transactional outbox

**E3. 部署顺序**
1. 部署 schema-aware 代码（tx 重载已就绪但未启用）
2. 灰度切到 `EnableTxOutbox()`，关闭原 `EnableOutbox()` 的 fallback
3. 观察 operational_command pending 队列 24h，确认无新增 dead-letter
4. 监控 `notification_delivery` 表的 `sent_at` 时延回归

## 5. 验收清单

| 项 | 度量 |
|----|------|
| 工单创建无通知丢失 | 故障注入：调用 CreateTicket 成功后强杀进程；下次启动 worker 仍能投递通知 |
| SLA 违规无通知丢失 | 同上 |
| 变更审批通知首交付 | 实跑 SubmitChange → 1s 内收到站内通知 |
| 事务原子性 | tx 回滚测试用例 100% 通过 |
| 幂等性 | 同一业务事件重复触发，notification 数量 ≤ 1 |
| 性能 | CreateTicket P95 增量 ≤ 5ms（enqueue 单行 INSERT 开销） |
| 现有回归 | go test ./... 与 npm test 全绿 |

## 6. 风险与对策

| 风险 | 影响 | 对策 |
|------|------|------|
| tx 内调用慢下游（如 LLM/邮件）拉长事务 | 锁竞争 | 通知 enqueue 仅写一行 INSERT，毫秒级；handler 异步跑 |
| SLA 监控循环开过多短 tx | 连接池压力 | 监控里 `ent.tx.active` 报警；现有循环已分页，无需每条 tx 开/关 |
| 变更 raw tx 与 Ent schema 漂移 | operational_command 写入失败 | 复用 `commandbus.EnqueueTx` 的 SQL 构造器（新增 `EnqueueRawTx(ctx, tx, req)`，复用同一字段映射） |
| 启动期开关未切换 | 双发 | 部署手册明确"先升级代码再切换开关"；CI 加 assertion：bootstrap 内若 `EnableTxOutbox` 与 `EnableOutbox` 同时为 true 则 panic |
| handler 已下线老 client 路径被外部调用 | 漏发 | 保留 `EnableOutbox()` 路径但加 `Deprecated:` 注释，监控调用量 |

## 7. 后续阶段（不在本次范围）

- 工单状态变更、分配、评论、CC 等通知同样下沉到 tx
- 跨域事件（事件总线 Watermill）的 tx-aware 包装
- Notification Preference（用户偏好）读取下沉到 tx，避免读到不一致的偏好
- 飞书/钉钉 connector 接入审批流的 tx-aware 出站