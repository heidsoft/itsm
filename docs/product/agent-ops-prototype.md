# AgentOps 产品原型能力契约

## CAPABILITY

面向企业 SRE、运维负责人和技术故障指挥官，提供一个“AI 判断、人工授权、工具执行、全程审计”的生产故障处置工作台。首版通过支付链路故障演示，验证企业是否愿意让受控 Agent 进入生产流程，以及是否愿意为执行治理和故障提效付费。

## CONSTRAINTS

- AI 默认是决策支持，不拥有静默执行生产操作的权限。
- 每个工具动作必须绑定操作者、租户、资源范围、风险等级、授权窗口和审计记录。
- L3 及以上动作必须人工批准；授权只能使用一次并自动过期。
- 故障判断必须显示证据来源、置信度、模型或 Skill 版本。
- 执行失败必须保留原始状态、错误和回滚入口。
- 原型数据为演示数据，不代表当前后端已实现全部 AgentOps 能力。

## IMPLEMENTATION CONTRACT

### Actors

- 故障指挥官：查看证据、批准或驳回计划、结束故障。
- SRE Copilot：关联告警、日志、CMDB、变更和历史事件，生成处置计划。
- 策略引擎：评估风险、签发短期授权、阻断越权动作。
- 连接器运行时：调用监控、日志、Kubernetes、数据库和协作工具。
- 审计员：查询执行证据包和责任链。

### Surfaces

- 故障驾驶舱：影响、MTTR、证据时间线、CMDB 拓扑。
- Agent 判断：根因假设、证据权重、处置计划和预期结果。
- 执行控制：身份、授权范围、风险等级、授权窗口和工具参数。
- 审计证据：审批签名、授权签发、工具执行、状态验证和哈希。

### States and transitions

`detecting -> diagnosed -> awaiting_approval -> authorized -> executing -> verifying -> resolved`

异常分支：

- `diagnosed -> rejected -> replanning`
- `executing -> failed -> rollback_pending -> rolled_back`
- `authorized -> expired -> awaiting_approval`

### Interface and data implications

- Incident 增加 AI diagnosis、evidence references、confidence 和 response plan 引用。
- Tool invocation 记录 actor、on-behalf-of user、connector、action、masked input、policy decision、result 和 timestamps。
- Approval 与现有 BPMN 用户任务结合，不建立第二套审批引擎。
- CMDB 负责影响关系；连接器负责实际调用；Agent/Skill 不直接持有生产凭证。

## NON-GOALS

- 首版不自建指标、日志或链路追踪存储。
- 首版不承诺完全自治修复生产故障。
- 首版不覆盖所有业务流程和通用 Agent 开发平台。
- 首版不替代 Prometheus、Grafana、Kubernetes 或现有 ITSM。

## OPEN QUESTIONS

- 第一批设计合作客户优先使用 Kubernetes、虚拟机还是数据库场景？
- 客户允许自动执行的最高风险级别是什么？
- 审计证据需要满足等保、ISO 27001 还是行业监管要求？
- 首个付费套餐按节点、Agent 调用量、故障量还是年度平台授权计费？

## HANDOFF

当前原型适合客户访谈和融资演示。下一阶段先完成 5 家设计合作客户验证，再决定后端实现顺序；推荐优先落地 Prometheus、Kubernetes、Webhook/飞书连接器和一次性授权审计链路。
