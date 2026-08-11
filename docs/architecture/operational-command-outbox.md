# Operational Command / Outbox 基座

## 目标

`operational_commands` 是跨领域副作用的统一可靠执行基座。业务事务只负责持久化业务实体和命令；Worker 负责可恢复执行。首批接入 `incident`、`change` 的 BPMN 启动，后续通知、连接器、AI、知识索引和 CMDB 同步必须复用同一状态机，不再在 HTTP 请求中启动无记录 goroutine。

## 状态机

```text
pending → processing → succeeded
   ↑          |
   └── retry ─┘
              └→ dead_letter
```

- `available_at` 控制首次调度和指数退避；
- `attempt/max_attempts` 控制死信门限；
- `lease_owner/lease_expires_at/fencing_token` 防止并发和旧 Worker 回写；
- Worker 在处理期间按租约的三分之一周期心跳；
- 执行语义是 at-least-once，Handler 必须以业务键保证幂等；
- `(tenant_id, command_type, idempotency_key)` 唯一，重复业务提交不能产生第二个命令。

## 事务规则

- 事件创建、事件时间线和 `workflow.start` 命令在同一 Ent 事务提交；
- 当前线上变更领域通过 `CreateWithWorkflowCommand` 在同一 Ent 事务提交；
- 事务失败时业务实体和命令必须同时回滚；
- 不允许“先提交业务，再尽力 enqueue”。

## Handler 契约

新增 Handler 时必须满足：

1. 使用命令的 `tenant_id`，不得信任外部响应返回的 tenant；
2. 以业务稳定键检查是否已经应用；
3. 尊重 `context` 取消与超时；
4. 错误可重试或可安全进入死信；
5. 错误文本不得包含密码、Token、AccessKey、连接器密钥或私密正文；
6. 高风险应用动作写领域审计，Command 记录不能替代业务审计；
7. payload 保存引用和不可变执行参数，不保存凭据；
8. 处理时间超过默认租约时依赖 Worker 心跳，不自行续租。

## 扩展顺序

1. `workflow.start`：事件、变更；
2. `notification.deliver`：已接入工单站内通知与飞书/钉钉/企微/Webhook 等企业渠道；每个收件人/渠道一个命令，使用调用方 occurrence key 派生幂等键；
3. `connector.deliver`：飞书优先，具备投递幂等和回调审计；
4. `ai.invoke`：只保存 prompt/template/model 引用和业务上下文引用；
5. `cmdb.import.process` / `cmdb.export.process`：已接入；任务与 Command 同事务创建，Command 只负责调度，领域 Job 仍是进度和结果的权威状态；导入行必须提供资产标签、序列号或云资源稳定标识以保证重试对账；
6. `cmdb.discovery` / `cmdb.reconcile`：沿用相同 Job + Command 模式，补齐分页发现、Diff、对账和待退役治理。

每种命令必须有独立 Handler、超时、最大尝试次数、错误分类、指标和死信重放权限。不得在通用 Worker 中加入领域 switch；`workflow.start` 的聚合类型路由属于该 Handler 自身。

### notification.deliver

- `TicketNotificationService` 在生产组装中只入队，不同步调用外部系统；
- payload 只保存 ticket、recipient、channel、type、content，不保存邮箱/手机号/open ID 等投递目标；目标由 Handler 在租户范围内实时解析；
- `notification_deliveries` 保存命令、收件人、渠道、掩码目标、attempt、状态、错误分类和发送时间；
- 站内通知、统一通知和投递审计在同一事务提交；
- 企业渠道通过 `connector.Manager` 投递，`connector.Message.ID` 使用 command idempotency key；连接器应把该 ID 透传给支持幂等的 Provider；
- 调用方可传 `idempotencyKey` 表示一次业务事件。未传时生成随机 occurrence key，避免内容相同的两个合法事件被永久去重；
- 外部 Provider 原始错误不写 Outbox、审计或业务日志，只记录安全错误分类。

### CMDB import/export

- `CMDBImportTask` / `CMDBExportTask` 与调度 Command 在同一 Ent 事务提交，不允许请求结束后启动 goroutine；
- Handler 只从命令租户内重载 Job，payload 仅保存 `taskId`，不携带文件内容或凭据；
- Worker 使用父 context 和任务超时，完成任务再次执行是 no-op；
- 导入按租户内资产标签、序列号或“云厂商 + 云资源 ID”对账，缺少稳定身份的行明确失败，防止 Worker 恢复时重复建 CI；
- Job 对外错误只保存安全分类，不持久化下载 URL、数据库错误或云端原始响应。

## 运维要求

- 发布前先通过 bootstrap/migration 创建 `operational_commands`；应用进程不自动迁移；
- 监控 pending 数量、最老等待时间、processing 租约过期数、重试率和 dead-letter 数；
- 死信重放必须创建审计记录，保留原 attempt 和错误；
- Worker 可与 API 同进程启动，但生产规模化时可拆为独立进程，数据库状态机和 Handler 契约保持不变。
# 运维控制面

平台运维人员通过租户隔离且受 `system:write` 权限保护的接口查看和处置可靠命令：

- `GET /api/v1/admin/operations/commands`：分页列表，支持 `status` 过滤，并返回 pending、processing、dead-letter、cancelled 和最老等待时间。
- `GET /api/v1/admin/operations/commands/:id`：查看命令详情；潜在密钥字段在 DTO 映射时脱敏。
- `POST /api/v1/admin/operations/commands/:id/replay`：只允许 dead-letter/cancelled，保留原 idempotency key、attempt 历史并增加 fencing token。
- `POST /api/v1/admin/operations/commands/:id/cancel`：只允许 pending/processing，通过 fencing 使旧 Worker 失去提交资格。

重放与取消和 `audit_logs` 写入位于同一事务，跨租户 ID 返回不存在，避免资源枚举。
