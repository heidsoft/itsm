# 核心域业务实现问题台账(2026-08-29)

> 范围:工单 / 事件故障 / 变更 / 工作流 / CMDB 五域业务实现审查
> 方法:5 个并行审查代理通读核心文件,按 AGENTS.md 强制规则(状态机/租户隔离/事务边界/并发控制/错误处理)评审
> 状态图例:🔴待修 🟡本批已修 ⚪待核实

## 域级评级汇总

| 域 | 评级 | 核心问题 |
|---|---|---|
| 工单 | 基本可用有暗坑 | 升级状态机缺失、生命周期审计/通知未入事务 |
| 事件/故障 | **风险较高** | 5 端点租户解析不安全、升级路径事务拆分 |
| 变更 | 基本可用有暗坑 | 并发审批竞态、标准变更绕过工作流 |
| 工作流 | 基本可用有暗坑 | 两处关键查询绕过事务客户端(并发重复任务/网关死锁) |
| CMDB | 基本可用有暗坑 | 批量操作租户缺失、删除非原子 |

## P0 缺陷(10)

### 工单
| # | 位置 | 问题 | 状态 |
|---|---|---|---|
| T1 | `service/ticket_service.go:1812-1835` | `EscalateTicket` 未校验状态转换合法性,终态工单被强制改回 in_progress | ✅已修 |
| T2 | `service/ticket_lifecycle_service.go:88-141` | 审计仅写日志不落库;通知在事务外同步发送 | 🔴 |

### 事件
| # | 位置 | 问题 | 状态 |
|---|---|---|---|
| I1 | `controller/incident_controller.go:821,859,1082,1684,1718` | 5 个端点用 `ctx.GetInt("tenant_id")` 而非安全解析,中间件未设值时以 0 值读写 | ✅已修 |
| I2 | `service/incident_service.go:1743-1793` | `EscalateToMajorIncident` 状态变更与审计事件不在同一事务 | ✅已修 |
| I3 | `service/incident_service.go:1108-1203` | `EscalateIncident` 三次独立写入无事务;未排除 resolved 态 | ✅已修 |

### 变更
| # | 位置 | 问题 | 状态 |
|---|---|---|---|
| C1 | `handlers/standard_change/handler.go:463-466` | 实例化标准变更直接返回 Ent 模型,且不创建 BPMN workflow command,变更永不启动流程 | 🔴 |
| C2 | `handlers/change/service.go:524+543` | `checkAndTransitionChange` 审批汇总与状态写入非原子,并发审批竞态 | 🔴 |

### 工作流
| # | 位置 | 问题 | 状态 |
|---|---|---|---|
| W1 | `service/bpmn_process_engine.go:673` | `createUserTask` 幂等检查用事务外客户端,并发下重复创建任务 | ✅已修 |
| W2 | `service/bpmn_process_engine.go:1224` | `allIncomingBranchesCompleted` 汇聚判断用事务外客户端,并行网关提前汇聚或死锁 | ✅已修 |

### CMDB
| # | 位置 | 问题 | 状态 |
|---|---|---|---|
| M1 | `service/configuration_item_service.go:947` | `BatchUpdateCI` update 缺租户谓词与乐观锁 | ✅已修 |
| M2 | `service/cloud/runner.go:113-114` | reconcile 查询 CloudService 缺租户过滤 | ✅已修 |
| M3 | `service/configuration_item_service.go:412-478` | `DeleteCI` 关系清理与 CI 删除非原子,失败致拓扑损坏 | 🔴(需先设计可审计软删除/历史保留模型) |

## P1 风险(20)

| 域 | 位置 | 问题 | 状态 |
|---|---|---|---|
| 工单 | `ticket_notification_service.go:1083-1090` | `resolveTenantID` 跨租户读工单反查租户 | 🔴 |
| 工单 | `ticket_service.go:1658` | `MarkFirstResponse` 吞错,SLA 首响可能永不标记 | 🔴 |
| 工单 | `ticket_service.go:2033-2042` | 升级处理人硬编码用户 ID 1/2/3,跨租户错派风险 | 🔴 |
| 工单 | `repository/ticket/repository_impl.go:746-754` | 编号 DB 降级路径 TOCTOU 窗口 | 🔴 |
| 工单 | `ticket_service.go:404-463` | legacy 路径 fire-and-forget goroutine | 🔴 |
| 事件 | `incident_service.go:1131-1136` | 允许从 resolved 升级(状态机绕过) | 🔴(并入 I3) |
| 事件 | `incident_service.go:745-773` | `AssignIncident` 有 AddVersion 无 VersionEQ 条件 | ✅已修 |
| 事件 | `root_cause_analysis_service.go:222-301` | 事件转问题后无下游联动 | ⚪ |
| 事件 | `incident_service.go:386-409` | 规则/工作流触发存在 fire-and-forget 路径 | 🔴 |
| 变更 | `handlers/change/service.go:247-254` | 提交后 BPMN 推进失败仅记日志,流程悬挂 | 🔴 |
| 变更 | `handlers/change/handler.go:492-519` | `AssignChange` 未校验 assignee 属当前租户 | 🔴 |
| 变更 | `handlers/change` 治理字段 | 已提审变更可经 PUT 静默换处理人 | 🔴 |
| 变更 | `handlers/change/repository_impl.go:726-760` | `ListByDateRange` 全表加载内存过滤 | 🔴 |
| 变更 | `handlers/standard_change/handler.go:339-341` | `DeleteStandardChange` Update 缺租户条件 | ✅已修 |
| 工作流 | `bpmn_process_engine.go:226-298` | `StartProcess` 无事务,失败留僵尸实例 | 🔴(ServiceTask 外部副作用需先改 durable command) |
| 工作流 | `bpmn_process_engine.go:2635-2658` | `DelegateTask` 无事务、受托人校验缺失、原审批人仍可完成 | 🔴 |
| 工作流 | `bpmn_process_engine.go:1973-1984` | `SetProcessInstanceVariables` 覆盖式写入,可丢引擎保留变量 | 🔴 |
| 工作流 | `bpmn_process_engine.go:536-546` | 排他网关不支持 default flow | 🔴 |
| CMDB | `cloud_discovery_service.go:422-428` | `upsertCloudResource` 查重缺租户条件 | 🔴 |
| CMDB | `service/cloud/runner.go:48-64` | `RunAll` 并发发现无互斥,绕过 DiscoveryJob lease | 🔴 |

## P2 改进(摘要)

工单:审计含完整工单正文应截断;错误分类用字符串匹配应改 `errors.Is`;更新后重复 Get。
事件:Acknowledge/Resolve 缺资源级鉴权;列表接口同时接受 `pageSize`/`size` 别名;SLA pause 与 on_hold 联动待核实。
变更:列表响应 key `changes` 应改 `items`;`ProcessApproval` 吞错;legacy `change_service.go` 与新层双实现待清理。
工作流:每次 CompleteTask 重建引擎;`validateVariableValue` 对 JSON 数字/时间的断言必然失败;审计 actor 取值 key 不一致。
CMDB:CI upsert OR 匹配可能误命中;`wouldCreateCycle` 全量加载边。

## 修复批次规划

- **本批(小切口外科修复,2026-08-29 已完成并验证)**:W1、W2、I1、M1、M2、T1、事件 AssignIncident 乐观锁、standard_change 删除租户条件 — 编译通过,`service`/`controller`/`handlers/standard_change`/`service/cloud` 测试全绿
- **事务包裹批次(2026-08-29 部分完成并验证)**:I2、I3 已修。M3 需先解决 CI 历史强外键与硬删除冲突；StartProcess 需先将 ServiceTask 外部副作用改为 durable command，禁止在数据库事务内直接执行远程回调。
- **第三批(语义类,需设计评审)**:C1 标准变更走 workflow、C2 审批竞态条件更新、DelegateTask 语义、审计落库、编号降级路径
