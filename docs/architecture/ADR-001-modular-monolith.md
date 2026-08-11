# ADR-001：模块化单体与 PostgreSQL 可靠命令基座

## 状态

Accepted（2026-08-11）

## 背景

首个商业版本面向企业私有化部署，首条生产验收旅程是“事件到恢复”。当前系统同时存在 `controller/ + service/` 横向分层和 `handlers/<domain>/` 领域切片；工作流、通知、SLA、连接器和 CMDB 同步又需要可靠异步执行。如果此时拆微服务或引入 Kafka，会扩大部署、事务一致性和运维范围，却不能直接提高首发旅程的正确性。

## 决策

1. 保持 Go 模块化单体，不在首个商业版本拆微服务。
2. `handlers/<domain>/` 是目标领域结构；迁移期间允许 legacy 层存在，但一个 HTTP 路径只能有一个生产装配。
3. PostgreSQL 同时承载业务事务、审计和 `operational_commands`。事务内写 command，由独立 Worker 使用租约、fencing、重试和死信可靠执行。
4. 不引入 Kafka。只有在跨部署单元吞吐或隔离需求经过生产数据证明后，才另立 ADR 评估消息基础设施。
5. BPMN 是唯一流程编排层；Incident、Change、Request 等领域服务仍拥有状态机和业务不变量。
6. 生产进程边界为同镜像的 `itsm-api`、`itsm-worker`、`itsm-init`；开发环境可使用 all-in-one 模式。
7. 后端 `/api/v1/capabilities` 是产品能力状态的单一事实来源。入口同时受 build、deployment、tenant readiness 和用户 action 约束。

## 目标运行结构

```text
Web / Open API / OIDC / Feishu
              |
           itsm-api
              |
   Application Command / Query
              |
 Incident · Change · Problem · Request · CMDB
              |
 PostgreSQL: domain + audit + operational command
              |
          itsm-worker
              |
 BPMN · notification · connector · SLA · RAG · CMDB Job
```

## 结果

- 业务写入和异步意图可在同一数据库事务提交，避免“业务成功但流程/通知丢失”。
- API 可独立扩容，不会重复启动 SLA、索引或升级调度器。
- Worker 失败后命令可重新认领；长任务由领域 Job 记录进度，outbox 只负责调度。
- 当前代价是共享数据库和单体发布节奏；通过领域所有权、repository 边界和独立进程模式控制耦合。

## 禁止项

- 请求结束后以 goroutine 启动关键 BPMN、通知或连接器动作。
- Controller 直接访问 Ent 或 raw SQL。
- 新建第二套审批/流程引擎。
- 前端常量自行宣布能力 GA，或展示后端未装配的正式操作。

## 相关文档

- [领域所有权](./domain-ownership.md)
- [Operational Command Outbox](./operational-command-outbox.md)
- [商业化目标架构](./commercial-ready-architecture.md)
