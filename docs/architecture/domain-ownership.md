# 领域所有权与迁移边界

| 领域 | 当前生产入口 | 目标所有者 | 事务所有者 | 当前策略 |
|---|---|---|---|---|
| Ticket | legacy controller/service | Ticket application service | Ticket service | 首版保留，不并入通用 Ticket 抽象 |
| Incident | legacy + `handlers/incident` | `handlers/incident` | Incident application service | 保持 API，逐端点迁移后一次切路由 |
| Change | legacy + `handlers/change` | `handlers/change` | Change application service | BPMN 启动只写 command |
| Problem / Known Error | `handlers/problem`、`handlers/known_error` | 对应领域切片 | 对应 application service | 补齐知识发布闭环 |
| Service Request | legacy + `handlers/service_request` | `handlers/service_request` | Request application service | 审批、通知和 provisioning 使用 command |
| CMDB | `handlers/cmdb` + cloud service | `handlers/cmdb` | CMDB service / Job | 发现长任务使用 Job，command 只调度 |
| SLA | legacy + `handlers/sla` | `handlers/sla` | SLA service | 仅 Worker 执行计时与升级 |
| Workflow | BPMN services | Workflow application service | 发起领域事务 + command | BPMN 是唯一编排层 |
| Notification | notification services | Notification delivery handler | 生产领域事务 + command | 所有渠道统一 `notification.deliver` |
| Connector | connector framework | Connector application service | Connector service | 飞书先行，Marketplace 暂只读 Pilot |

## 迁移完成定义

一个领域只有在以下条件同时满足后才能删除 legacy 装配：稳定 HTTP 合同保持兼容；全部端点已切换；状态机和租户规则在服务层；事务内产生审计与 command；契约、越权、并发和故障恢复测试通过。迁移过程中禁止同一路由随机落到两套实现。
