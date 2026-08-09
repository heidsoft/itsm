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
2. `notification.deliver`：站内、邮件和消息通知；
3. `connector.deliver`：飞书优先，具备投递幂等和回调审计；
4. `ai.invoke`：只保存 prompt/template/model 引用和业务上下文引用；
5. `cmdb.discovery` / `cmdb.reconcile`：Command 只负责调度，领域 Job/Result 仍是 CMDB 权威状态。

每种命令必须有独立 Handler、超时、最大尝试次数、错误分类、指标和死信重放权限。不得在通用 Worker 中加入领域 switch；`workflow.start` 的聚合类型路由属于该 Handler 自身。

## 运维要求

- 发布前先通过 bootstrap/migration 创建 `operational_commands`；应用进程不自动迁移；
- 监控 pending 数量、最老等待时间、processing 租约过期数、重试率和 dead-letter 数；
- 死信重放必须创建审计记录，保留原 attempt 和错误；
- Worker 可与 API 同进程启动，但生产规模化时可拆为独立进程，数据库状态机和 Handler 契约保持不变。
