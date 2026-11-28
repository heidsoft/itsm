# 工单领域状态与操作契约

基准 `021f08c0e`。现状以生产入口和持久模型为准。目标契约用于后续修改的验收。

| 维度 | 权威记录 | 投影边界 |
| --- | --- | --- |
| 业务状态 | `ent/schema/ticket.go` 与工单领域流转校验 | `GetTicketWorkflowState` 以 `tk.Status` 为 CurrentStatus。现有动作分支包括 new/open 的接单、in_progress 的解决、resolved 的关闭，closed 为终态。仍注册 reopen 路由，需确认其兼容语义 |
| 生命周期阶段 | 工单状态与业务流程当前位置共同决定 | 申请、审核、处理、交付可用作展示分组，具体可操作性由服务端给出 |
| BPMN 执行 | `ProcessInstance`、`ProcessTask` 与运行历史 | `state-v2` 聚合节点状态。暂停、终止、失败与工单业务关闭分别验证 |
| 审批 | BPMN 决策证据及兼容 `TicketApproval` 记录 | 工单审批状态从审批记录聚合。职责分离须使用工单申请人身份；流转接单分支可能另写流程实例变量 `requester_id` |
| SLA | 策略、计时状态、违约记录和权威时间戳 | 停止或超时对业务状态的影响须经明确领域动作 |

创建入口为 `SetupTicketRoutes → TicketHandler.CreateTicket → handlers/ticket`。流转入口为 `SetupTicketRoutes → TicketWorkflowHandler → TicketWorkflowService`。详情的接单、解决、关闭读取 `state-v2.availableActions`。审批、分配、转派、撤回、升级和删除要逐项比对动作投影与写入守卫。

目标判定顺序为认证租户与角色权限、资源可见性、工单状态、流程节点、任务归属和职责分离。部门管理员、租户管理员、系统管理员的范围以实际权限和委托范围判断。提交时间重新判定。非法迁移或竞争返回冲突。删除、终止、升级需要确认和审计。可重试写入及副作用用业务版本和既有事务 outbox 关联，验收覆盖回滚和重复执行。

状态异常调查同时保存原始数据库值、DTO、流程及页面字段。已确认前端常量会将无法识别的状态显示为 open。历史数据是否含有 `unknown` 仍需隔离库取证。

BPMN 状态另有独立回退。`normalizeBpmnStatus` 将无法识别的实例状态映射为 `not_started`，而 BPMN 监控按 `failed` 统计实例。需证明失败实例选中时仍能显示真实状态，并保留无实例时的原有语义。

现有合法迁移以 `common.IsValidTicketStatusTransition` 及 `ticketStatusTransitions` 为准。以下列出不同状态间的迁移，同值判断在该函数中另行放行。具体动作仍需满足当前执行入口的权限和业务前置条件。

| 当前工单状态 | 可迁入状态 |
| --- | --- |
| new | open、assigned、in_progress、cancelled、rejected |
| assigned、open | in_progress、pending、resolved、cancelled、rejected |
| in_progress | resolved、pending、cancelled、rejected |
| pending | in_progress、resolved、open、cancelled、rejected |
| resolved | closed、in_progress、open |
| approved | in_progress、resolved、closed |
| rejected | open、cancelled |
| closed、cancelled | 终态 |

人工任务 schema 标注 created、assigned、started、completed、cancelled。SLAState 标注 pending、active、paused、breached、met。动作投影对这两套状态分别取证，再核对流程节点、处理人和计时事件。工单状态白名单与任务、SLA 的独立状态转换分别验证。

数据库侧已有 `20261031_status_check_constraints.sql`。该迁移在发现词表外存量时保留 `NOT VALID` 约束，继续拦截新的不合法写入。排查隔离库先核对迁移记录和约束的验证状态，再按租户汇总词表外状态及其数量。页面出现回退标签本身不足以证明有存量非法值。

动作投影也要验证契约一致性。`GetTicketWorkflowState` 对 resolved 同时返回 close、reopen，对审批按聚合层级与人员生成动作。应以生产写入条件逐项验证投影，尤其是 resolved 重开、closed 终态、当前一层中已完成的审批记录，以及 BPMN 无可执行任务的情形。
