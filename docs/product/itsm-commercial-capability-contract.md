# ITSM 商业化整体能力契约

## CAPABILITY

面向企业 IT 管理员、服务台、二线运维、变更经理、审批人和业务员工，系统应提供一条可审计的端到端服务管理链路：用户通过服务目录或事件入口提出需求，工单按 SLA 和 BPMN 流转，处理人员以 CMDB 的 CI、关系和影响范围作为上下文，问题与已知错误沉淀复用知识，变更通过风险、审批、实施和复盘闭环执行，通知与企业连接器触达参与人，AI 只在有权限、可降级、可追踪的边界内提供建议。

商业化目标不是让每个菜单都有页面，而是让以下主链路可以由真实客户数据连续运行、失败可恢复、过程可审计、结果可验收。

## 当前能力结论

### 成熟度定义

- **GA 候选**：核心模型、服务规则、租户边界和主要接口已存在，可进入生产验收；
- **Pilot**：有真实实现，但跨模块闭环、运维、权限或测试仍不完整；
- **Disabled**：骨架、占位或不能安全形成真实业务结果，不应作为已交付能力展示。

| 能力域 | 当前成熟度 | 已有业务证据 | 商业化缺口 |
|---|---|---|---|
| 工单/事件 | GA 候选 | 事件状态机、CI 多选关联、优先级矩阵、重大事件、BPMN 触发、租户过滤；生产装配启用持久化 command/outbox | 仍需固化监控告警来源治理、积压恢复和目标容量验收 |
| 问题/已知错误 | Pilot | 问题状态、工作流、从问题创建已知错误、受影响 CI 字段 | CI 关联仍有字符串/弱引用路径；重复事件聚类、根因 CI、解决后知识发布未形成强制闭环 |
| 变更 | GA 候选 | 受影响 CI 校验、CMDB 影响摘要、关键 CI/高风险依赖/开放事件统计、风险与回滚建议、审批能力 | 影响摘要尚未成为提交审批的后端门禁；实施窗口、冲突检测、实施结果和 PIR 需统一验收 |
| 服务目录/服务请求 | Pilot | 目录、请求、审批、CI ID、云资源引用与 CI 创建/复用 | 目录项到 CI Type、交付模板、BPMN、SLA 的绑定不统一；Provisioning 生命周期与失败补偿不足 |
| CMDB | GA 核心 + Pilot 发现 | CI、类型、关系、拓扑、历史、影响分析可用；阿里云连通已验证 | 云发现 Job/Worker/Diff/对账/审计尚未落地；数据质量、来源 ownership、退役治理待补 |
| BPMN/审批 | Pilot | 流程定义、绑定、实例、任务、变量、历史；工单与事件已使用持久化 command/outbox | 剩余业务域仍需收敛可靠触发，并补齐补偿、重放和实例级 E2E |
| SLA | GA 候选 | 定义、截止时间、违规、预警、通知、指标、逐租户 watcher | 主要围绕 Ticket，事件/请求等对象的统一绑定需确认；日历/工作时间、暂停计时、升级动作需生产验收 |
| 知识/RAG | Pilot | 文章、关键词检索、向量检索降级、LLM 问答、租户过滤 | 向量删除未实现；发布版本、可见性/RBAC 的端到端检索验证不足；问题解决到知识审核发布不闭环 |
| AI | Pilot | LLM Gateway、分诊、摘要、RAG、AI 审计和工具框架 | 置信度、接受/拒绝反馈、prompt/model 版本和统一 evaluator 未在每个能力中强制；高风险动作不可自动执行 |
| 连接器/通知 | Pilot/Disabled 混合 | 连接器注册与生命周期骨架，飞书实现基础较强，企微/钉钉/Webhook 有骨架 | 真实健康检查、租户密钥、消息幂等、重试/死信、回调验签、投递审计及市场安装闭环不完整 |
| RBAC/多租户/审计 | GA 候选 | 角色权限、Endpoint ACL、租户过滤、AuditLog、多处跨租户测试 | 各域仍需能力级权限矩阵；后台任务/system actor、审计相关 ID、敏感字段脱敏必须统一 |
| 报表/运营 | Pilot | SLA、事件等局部指标和 Dashboard | 缺少面向管理者的统一服务质量指标、数据口径版本、导出审计和 MSP 跨租户汇总边界 |

## CONSTRAINTS

### 固定产品规则

1. **业务主线优先于模块数量**：首个商业版本只承诺能跑通并验收的链路，不承诺所有现有菜单。
2. **CMDB 是流程上下文，不是独立资产表**：事件、问题、变更、服务请求必须引用同一 CI 身份、关系和来源语义。
3. **BPMN 是唯一流程编排层**：不得在各业务域增加第二套审批或状态流引擎。
4. **业务落库与异步动作可靠衔接**：流程启动、通知、连接器投递、AI 调用、云发现不得依赖 HTTP 请求内 goroutine 保证交付；采用持久化 command/outbox、可重试 Worker 和幂等消费。
5. **后端状态机是权威**：事件、问题、变更、请求、知识发布、CI 生命周期均由服务层校验，前端不能独自改变状态。
6. **所有自动化都有来源和审计**：actor、tenant、request、业务对象、流程实例、连接器投递或 AI invocation 必须可关联。
7. **AI 默认是决策支持**：无模型时确定性降级；有模型时保留模型、prompt、输入来源、置信度、采纳/拒绝和反馈。
8. **密钥只存引用**：连接器和云账号使用 tenant-scoped SecretProvider，API、日志和审计不返回密钥。
9. **多租户失败关闭**：后台任务、搜索、RAG、导出、回调和拓扑查询必须显式携带 tenant；平台 bypass 要有原因和审计。
10. **能力状态由后端事实驱动**：build capability、deployment readiness、tenant readiness 和 actor permission 共同决定 UI，隐藏菜单不是授权。

### 核心业务不变量

- 一个业务动作失败不能留下虚假的成功状态，例如“工单已创建但流程永远没启动”。
- CI 被退役、合并或来源变化时，历史事件、问题、变更和请求引用仍可解释。
- 变更进入审批前必须完成受影响 CI 校验；关键 CI 命中、开放重大事件或缺少回滚计划时按策略阻断或升级审批。
- 问题关闭前必须明确根因、解决方案或无法确定原因；Known Error 需要 workaround 和受影响范围。
- 服务请求的审批完成不等于交付完成；交付失败必须进入可重试、人工接管或补偿状态。
- SLA 计时使用服务端权威时间和策略版本；暂停、恢复、节假日和策略变更均可追溯。
- 知识只有已发布、当前版本且当前 actor 可见的内容才能进入 RAG。
- 连接器投递至少一次，业务侧按幂等键去重；失败进入重试/死信并可人工重放。

## IMPLEMENTATION CONTRACT

### Actors

- 员工/申请人：提交服务请求、事件，查看进度和知识；
- 服务台：分诊、关联 CI、响应和升级；
- 二线/三线处理人：处理任务、问题、变更和知识；
- 变更经理/CAB：评估风险、审批、窗口和 PIR；
- CMDB 管理员：维护模型、来源、对账和数据质量；
- 平台/租户管理员：流程、SLA、权限、连接器、AI 和模板配置；
- System Worker：执行可恢复的流程命令、通知、同步和索引任务，使用明确 system actor。

### 商业 MVP 主链路

#### 链路 A：事件到恢复

`告警/人工报障 → 事件创建 → CI 关联 → 影响/优先级 → SLA → 分派/BPMN → 处理 → 恢复确认 → 关闭 → 审计/指标`

上线门禁：CI 关联跨租户失败；流程启动可恢复；SLA 预警/违规真实投递；重大事件升级；关闭码、时间线和审计完整。

#### 链路 B：重复事件到问题/知识

`重复事件 → 问题 → 根因 CI/关系 → workaround → Known Error → 解决 → 知识草稿 → 审核发布 → RAG 可检索`

上线门禁：事件关联问题；CI 使用强引用；Known Error 生命周期；知识版本与权限；向量索引/删除一致；无 LLM 时关键词检索可用。

#### 链路 C：受控变更

`变更创建 → 受影响 CI → CMDB 影响摘要 → 风险策略 → CAB/BPMN → 窗口 → 实施 → 验证/回滚 → PIR → 关闭`

上线门禁：影响分析进入提交门禁；审批与业务状态一致；高风险变更必须回滚计划；失败可回退；PIR 和审计不可跳过。

#### 链路 D：服务请求到交付

`目录项 → 表单校验 → SLA/流程绑定 → 审批 → 交付任务 → CI 创建/变更 → 验证 → 完成/补偿`

上线门禁：目录版本固定到请求；审批与交付分态；Provisioning 幂等；失败可人工接管；创建的 CI 保留请求和执行来源。

### 跨域接口

| 提供方 | 消费方 | 必须稳定的契约 |
|---|---|---|
| CMDB | 事件/问题/变更/请求 | CI identity、criticality、environment、lifecycle、关系、影响范围、source health |
| BPMN | 所有业务域 | definition/binding version、instance/task/status、business correlation、command result |
| SLA | 工单/事件/请求 | policy version、start/pause/resume/deadline/breach、calendar、escalation |
| 知识 | 服务台/AI/RAG | published version、visibility、tenant、source citation、index state |
| 连接器 | 通知/审批/告警 | tenant installation、secret ref、idempotency key、delivery status、callback signature |
| AI | ITIL 各域 | capability status、model/prompt version、confidence、evidence、decision feedback、audit ID |
| 审计 | 全域 | tenant、actor、request/correlation ID、object、action、before/after hash、result |

### 业务状态与可靠执行

建立统一但不混淆领域的可靠执行机制：

- 业务事务写入实体状态和 outbox command；
- Worker 按 tenant 和 command type 领取，使用 lease/fencing；
- BPMN 启动、连接器投递、AI 调用、索引、CMDB 同步各自幂等；
- command 保存 attempt、heartbeat、错误分类、next retry 和 dead-letter 状态；
- 业务页面可见“流程/通知/交付未启动或失败”，不能静默吞错；
- 重放动作受 RBAC 控制并写审计。

### 商业化能力状态接口

建议统一提供 `/api/v1/capabilities`，每项返回：

- `key`、`maturity` (`ga|pilot|disabled`)；
- `buildAvailable`、`deploymentReady`、`tenantReady`；
- `permissions` 与当前 actor 的 allowed actions；
- `dependencies`、`degradedReason`、`lastHealthCheckAt`；
- `acceptanceVersion` 与已通过的验收套件版本。

CMDB、AI、RAG、连接器和外部通知首先接入；基础 ITIL 页面不因能力接口故障而不可用。

### 运营与可观测性

商业版本至少持续度量：

- 事件量、首次响应、恢复时间、重开率、重大事件数；
- SLA 达成率、预警到响应时长、违规和升级成功率；
- 问题消除的重复事件数、Known Error/知识转化率；
- 变更成功率、紧急变更率、回滚率、变更引发事件数；
- 请求审批耗时、交付耗时、失败/人工接管率；
- CMDB 完整率、stale 比例、来源健康、同步覆盖和冲突；
- 流程命令、通知、连接器、AI、索引任务队列深度和失败率。

所有管理指标要有租户、时间范围、状态定义和版本化口径。

## 产品化优先级

### P0：先形成可售卖闭环

1. 将剩余业务域的 BPMN/通知触发继续收敛为持久化 outbox/worker，并固化积压、重放和死信运维；
2. 固化“事件—CI—SLA—流程—恢复”和“变更—CI—风险—审批—PIR”验收旅程；
3. 完成 CMDB 阿里云 ECS Job/Worker/Diff/对账/审计闭环；
4. 统一后端 capability/readiness 与前端入口；
5. 为每条主链路补租户、RBAC、审计、失败恢复和 E2E 测试。

### P1：提高客户持续使用价值

1. 问题—Known Error—知识发布/RAG 闭环；
2. 服务目录—审批—交付—CI 变更闭环；
3. 飞书生产连接器，随后再扩企微和钉钉；
4. SLA 工作日历、暂停/恢复和升级治理；
5. CMDB 数据质量和管理者运营报表。

### P2：形成差异化

1. AI 影响分析、CAB 摘要、根因候选、SLA 预测的评测与反馈闭环；
2. 可声明 Skill、连接器市场和插件市场；
3. MSP 跨租户运营、商业套餐和容量治理；
4. 更多云厂商、Kubernetes 和自动关系发现。

## NON-GOALS

- 不在当前阶段复刻 ServiceNow 的全部 CSDM、Discovery、ITOM、HRSD 或 SecOps；
- 不把每个现有页面都升级为商业承诺；
- 不用 AI 替代审批、权限、状态机和高风险操作的人类责任；
- 不同时生产化飞书、企微、钉钉和所有云厂商；
- 不在业务 Controller 中新增临时 goroutine、直连第三方 SDK或第二套流程引擎；
- 不用演示数据或前端本地状态证明功能可用。

## OPEN QUESTIONS

以下产品决策不会阻止 P0 架构收敛，但进入定价和客户试点前需要确认：

1. 首个付费客户主要是内部 IT 服务台、运维中心，还是 MSP？三者的首页、权限和报表不同。
2. 首个部署形态是私有化单租户、私有化多租户，还是 SaaS？`env://`、SecretProvider、升级和运维边界取决于此。
3. 首个企业消息渠道是否确定为飞书？建议只选择一个做生产闭环。
4. 商业 MVP 是否要求 SSO/LDAP/组织同步？如果要求，它属于 P0 而非连接器市场的普通条目。
5. 自动化交付首个真实场景是什么：阿里云 ECS、账号权限、软件安装，还是通用 Webhook？
6. 开源版与商业版是同一核心加企业插件，还是统一功能、商业支持收费？需要形成明确能力矩阵。

## HANDOFF

该能力合同已经足够支撑 P0 架构评审和实施拆分。下一步不应继续横向新增菜单，而应建立“商业验收旅程”执行计划：先实现可靠 command/outbox 基座，再并行固化事件/变更两条旅程和 CMDB 真实同步，最后以生产 E2E、故障恢复、租户隔离和审计报告作为发布门禁。
