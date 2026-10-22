# ITSM 商业化整体能力契约

> Status: current

## CAPABILITY

面向企业 IT 管理员、服务台、二线运维、变更经理、审批人和业务员工，系统应提供一条可审计的端到端服务管理链路：用户通过服务目录或事件入口提出需求，工单按 SLA 和 BPMN 流转，处理人员以 CMDB 的 CI、关系和影响范围作为上下文，问题与已知错误沉淀复用知识，变更通过风险、审批、实施和复盘闭环执行，通知与企业连接器触达参与人，AI 只在有权限、可降级、可追踪的边界内提供建议。

商业化目标不是让每个菜单都有页面，而是让以下主链路可以由真实客户数据连续运行、失败可恢复、过程可审计、结果可验收。

## 当前能力结论

### 成熟度定义

- **GA 候选**：核心模型、服务规则、租户边界和主要接口已存在，可进入生产验收；
- **Pilot**：有真实实现，但跨模块闭环、运维、权限或测试仍不完整；
- **Disabled**：骨架、占位或不能安全形成真实业务结果，不应作为已交付能力展示。

成熟度不是文档口径，而是接口契约：`GET /api/v1/capabilities` 对每个条目下发 `maturity`，
并对 `pilot` 条目下发 `pilotReason`（**为什么还不是 GA + 晋升需要补哪块可验证证据**）。
2026-10-22 起该字段为强制项——`pilot` 缺理由或理由里没有「晋升条件」会让后端测试失败，
`ga` 条目带理由同样失败（过期理由与成熟度自相矛盾）。在此之前理由只写在 Go 注释里，
修完阻断项后没人回头改注释，矩阵会长期对外宣称一个已经不存在的问题。
运营侧可在 **管理 → 能力矩阵**（`/admin/capabilities`）核对同一份数据。

| 能力域 | 当前成熟度 | 已有业务证据 | 商业化缺口 |
|---|---|---|---|
| 工单/事件 | GA 候选 | 事件状态机、CI 多选关联、重大事件（`POST /api/v1/incidents/:id/major-incident`）、BPMN 触发、租户过滤；生产装配启用持久化 command/outbox | 仍需固化监控告警来源治理、积压恢复和目标容量验收；**优先级矩阵不是业务证据**（2026-10-02 实测：后端零 `/api/v1/priority*` 路由，`service/priority_matrix_service.go` 的 `SetPriorityMatrixService` 无任何调用点，因此 `incident_service.go:242,627` 的 impact×urgency 派生分支在生产里永不执行；前端 `priority-matrix-api.ts` 及其不可达组件链已在 E3-2 删除（2026-10-02），该能力现在既无 UI 也无客户端） |
| 问题/已知错误 | **GA** | 问题状态、工作流、从问题创建已知错误并建立 problem 关联、受影响 CI 字段；2026-10-21 起关闭必须有解决方案与根因结论（`4090`），「无法确定」也须写入根因字段；**2026-10-21 SLA 与评论 GA 阻断项落地**：问题创建即按 `service_type="problem"` 绑定 `sla_states`（与工单/事件同一 SLA 引擎与多态表），resolved/closed 打解决时间戳，`GET /problems/:id/sla` 从 503 桩重开 `200` 真实契约（`bound=false` 如实表示未绑定），SLA monitor 周期扫描与违规创建覆盖 problem；评论独立表 `problem_comments`，`GET/POST /problems/:id/comments` 从 503 桩转真实契约（跨租户 problem_id 统一 404 不暴露可枚举信号），前端详情页补 SLA 卡片与评论 Tab；**2026-10-22 矩阵晋升 GA**：四项阻断（关闭证据、KEDB 关联、SLA 读写、评论读面）均已在生产装配路径生效（`internal/bootstrap/app.go` 的 `SetSLAService` / `SetKnowledgeRepository` 有调用点），`GET /problems/:id/sla` 不再返回 503 | 重复事件聚类、根因 CI 仍缺；解决后知识**沉淀已通**（2026-10-22 `POST /problems/:id/knowledge`，见知识域行），但评审发布仍非强制闭环 |
| 变更 | **GA 候选** | 受影响 CI 校验、CMDB 影响摘要、关键 CI/高风险依赖/开放事件统计、风险与回滚建议、审批能力；**2026-10-21 门禁收口**：影响摘要被提交审批消费（关键 CI/开放事件 ⇒ 强制 high 风险等级 422 阻断），scheduled 转换复查窗口冲突；审批主链路（提交/审批通过/拒绝）已事务化并有专项测试背书，README「原子一致性待收敛」对主链路已过时；**2026-10-22 审批完整性与事务边收口**：会签/阈值/多级链求值收敛为唯一实现 `evaluateApprovalQuorum` 并接入生产路径 `/changes/:id/approve`（此前一票即批准，单票可推进会签单），未达阈值返回 `pending` 且该票入审计表、链上有驳回票返回 409；终态流转的 `change_approval_chains` 收口移入同一事务（`WorkflowTx`，收口失败整体回滚，含故障注入回归）；无路由的 legacy `ProcessApproval`/`UpdateApprovalRecord` 已删除；未绑定流程实例的回退路径改为**失败关闭**（会签/阈值/多级链未集齐直接 `409`，不再单票放行）；**2026-10-22 实施结果留痕**：新增 `POST /changes/:id/fail`（状态机一直允许 `failed` 却无入口，失败只能报成 completed 或直接回滚），`failed`/`rolled_back` 原因必填并写入新列 `outcome_note`（与状态同一条 UPDATE），PIR 创建收紧为仅 `completed`/`rolled_back`/`failed` | **2026-10-22 失败可回退落地（链路 C 最后一条上线门禁）**：回滚从「自由文本 + 一次状态翻转」升级为逐项确认的回滚清单（`change_rollback_steps` 一行一步，`GET/PUT /changes/:id/rollback-steps` + `POST .../rollback-steps/:stepId`），每步留 status/note/确认人/时间；清单存在且有 `pending` 步骤时`rolled_back` 被 `422` 拦下，清单读取失败按失败关闭；实施开始后清单锁定（否则「清空清单→点回滚」即可绕过门禁），skipped/failed 必须写依据。**刻意不做脚本自动执行**：回滚动作多在生产高危路径（数据库恢复、流量切换），由 ITSM 代跑脚本的风险远大于收益，本域承诺的是「可编排、可逐项确认、可审计」，执行仍由人或既定 runbook 完成。三处残留事务边已收敛（`ProcessApproval` 删除、降级路径随其消失、`CloseChangeApprovalChains` 收口进同一事务），**未绑定 BPMN 流程时的纯业务回退路径仍不落审批决策**（`ProcessApprovalDecision` 的实例/任务/节点字段均为流程事实，伪造等于往不可变审计表写假账）——该路径已改为 fail-closed：当前这一票不够阈值即 `409` 并提示配置流程模板，不再静默单票批准；发布前需重新评估 GA 候选资格 |
| 服务目录/服务请求 | Pilot（交付失败恢复已收口） | 目录、请求、审批、CI ID、云资源引用与 CI 创建/复用；**2026-10-21 交付链路生产化**：执行权改为原子领取（`pending|failed|过期 running → running`），已完成任务重复执行返回 `4090` 而非二次交付；`attempt_count` 封顶 5 次；`POST /provisioning-tasks/:id/retry` 重新排队（每次尝试独立幂等键，避免入箱层判重导致「接口 200 但什么都没发生」）；`POST /provisioning-tasks/:id/resolve` 支持人工接管（必填依据说明 + 判定人留痕）；worker 崩溃留下的 `running` 由 10 分钟租约回收自愈 | 目录项到 CI Type、交付模板、BPMN、SLA 的绑定不统一；交付 Provider 仍为 Stub（未接真实云 API），补偿动作（回滚已创建资源）尚未实现 |
| CMDB | GA 核心 + Pilot 发现 | CI、类型、关系、拓扑、历史、影响分析可用；阿里云连通已验证 | 云发现 Job/Worker/Diff/对账/审计尚未落地；数据质量、来源 ownership、退役治理待补 |
| BPMN/审批 | Pilot | 流程定义、绑定、实例、任务、变量、历史；工单与事件已使用持久化 command/outbox；旧审批提交停写，无 BPMN 待办时拒绝业务直批 | 配置与历史查询继续保留；跨域审批决策和业务状态仍需原子收敛，存量待办迁移需单独核对；补齐可靠触发、补偿与实例级 E2E |
| SLA | GA 候选 | 定义、截止时间、违规、预警、通知、指标、逐租户 watcher；**2026-10-21 统一绑定补齐**：ticket/incident/problem/service_request 四域全部写同一 `sla_states` 多态表（服务请求 2026-10-21 接入，`GET /service-requests/:id/sla` 真实契约），monitor 全表扫描自动覆盖新类型 | 日历/工作时间、暂停计时、升级动作需生产验收；服务请求无 priority 维度（定义仅按 service_type 匹配） |
| 知识/RAG | Pilot | 文章、关键词检索、向量检索降级、LLM 问答、租户过滤；向量删除已实现并经 outbox 接线（`service/vector_store.go:88`、`handlers/knowledge/outbox.go:75 deleteWithVectorOutbox`）；**2026-10-22 问题→知识沉淀闭环落地**：`POST/GET /problems/:id/knowledge`（现象/根因/方案模板化沉淀为草稿 + 反查，`source_problem_id` 来源追溯，无根因无方案 422 拒绝空壳沉淀，沉淀走既有评审流），调查摘要 `KnowledgeArticles` 字段接通真实数据；**2026-10-22 发布门禁收口**：`/publish` 强制 `review_status=approved`（未过评审 409/4090，堵住绕过防自审的旁路），评审通过即同事务入索引、驳回/撤回即出索引，重复发布幂等；「沉淀→评审→发布→RAG 可检索」端到端验收已自动化（`publish_gate_test.go`：草稿检索 0 条 → 审批通过 1 条 → 撤回 0 条） | 版本化发布（历史版本对比/回滚已有端点，发布版本语义未收敛）与分类可见性/RBAC 的跨角色端到端检索验证不足；知识检索的人工验收口径（典型问题集召回率）仍缺 |
| AI | Pilot | LLM Gateway、分诊、摘要、RAG、AI 审计和工具框架 | 置信度、接受/拒绝反馈、prompt/model 版本和统一 evaluator 未在每个能力中强制；高风险动作不可自动执行 |
| 连接器/通知 | **通知 GA** / 连接器 **Pilot（飞书/企微/钉钉三家）** | 通知：偏好过滤、幂等键、outbox 重试、投递审计读面（`GET /notification-deliveries` + 失败重试）、渠道诊断（`GET /notifications/channels`：装配/健康/收件目标三合一）；连接器：**2026-10-21 企微/钉钉 Disabled→Pilot 转正**——三家 Send/HealthCheck/VerifySignature/ParseInbound 同层级实现（编译期接口断言），回调路由与审计注入全链路装配，部署就绪由 `ITSM_{FEISHU,WECOM,DINGTALK}_READY` 如实申报；**2026-10-22 健康面租户收口**：`GET /connectors/health` 与 `/connectors`、`/connectors/configs`、`/connectors/lifecycle`、Provision 响应统一改用 `connector.Manager.HealthCheckTenant`（此前调 `HealthCheckAll`：任一租户既可读到别家连接器的名字与状态，也可借一次请求探测别家端点可达性），健康响应改为 `{tenantId, items[{name, provider, ok, ...}]}`，前端健康列不再硬编码租户号（此前恒取不到值，状态列永远显示"已启用"）；**2026-10-22 密钥轮换接线**：`POST /connectors/configs/:name/rotate-secret` 从零路由引用的死代码接成可用端点（`connector:write` + `audit_log action=rotate_secret` 只记键名），语义为**立即生效**（旧凭据即刻失效，无并行宽限窗口——三家 IM 收发验签都只读一份凭据，历史 `gracePeriodHours` 从未被执行已移除），Init 失败保留原凭据、落库失败回滚运行态；**2026-10-22 市场安装闭环**：安装/配置/卸载三处不再把「记录成功」当成「运行态成功」——连接器起不来时安装记录置 `failed` 并写 `error_message`（此前 `provisionConnector` 的失败被 `Warn` 吞掉后照样置 `active`，市场页显示已安装而实际一条消息都发不出去），配置更新改为**先注册成功再落库**（失败 `409` 且库内保留上一份可用配置，运行态仍是旧实例），卸载校验实例确实注销（`Revoke` 键含 provider，历史数据 provider 不一致会残留，残留即 `409` 不标 `uninstalled`），卸载后重装与 `failed` 状态下重复安装都重新走注册，不再因「记录已存在」直接标 `active`；**2026-10-22 入站回调去重收口**：飞书 webhook 的持久化去重器 `SetInboundDedup` 注入后**从未被读过**（仍在用进程内 map，进程重启即清零、多副本各存一份，而 task_event 是唯一会真正改写飞书任务的入站路径），现改为 `connector_inbound_dedup` 表登记（`timestamp:nonce`，租户隔离在列里）；业务处理失败时释放该行，让平台重投能重新接管（否则一次瞬时失败就被判「重复」永久丢事件），去重存储不可用则 `500` fail-closed；`CleanupExpired` 补上每 10 分钟的后台清理（此前唯一调用点是单测，表只增不减），并让过期行在清理任务没跑过时也能被覆盖——schema 注释一直承诺「expires_at 后允许覆盖」，但唯一约束不会随时间自己消失 | 真实飞书/企微/钉钉租户联调验收（验签算法与事件字段以线上为准）与死信人工重放仍需逐项验收；企微/钉钉的 OAuth/任务双向同步等飞书级深度集成未做 |
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

`GET /api/v1/capabilities` 已提供（`router/router.go:376` 注册，`handlers/capability/handler.go:27-36` 定义 DTO），每项返回：

- `key`、`maturity` (`ga|pilot|disabled`)；
- `buildAvailable`、`deploymentReady`、`tenantReady`；
- `allowedActions`（当前 actor 可执行动作；契约里原写的 `permissions` 字段尚未返回，属待补项）；
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

### 升级与发布安全

升级能力本身属于产品能力，而非运维附赠：

- 每个候选版本产出不可变证据包：`VERSION`（生产禁用 `latest`）、镜像 digest、Git SHA、迁移结果、E2E 结论、恢复演练记录；
- 升级前提供 Pre-upgrade Check：schema 漂移、未执行迁移与索引缺口、租户覆盖、兼容性与预估耗时，输出可评审报告；
- 库侧结构漂移必须在 ent 自动迁移之前由前置钩子自愈（`Schema.Create` 先于 ledger 迁移执行），迁移顺序契约不可依赖约定；
- 迁移必须可演练（脱敏副本 dry-run）且可回滚，回滚 RTO 需可度量；
- 部署自诊断（Doctor）与 `/api/v1/capabilities` 共用同一事实源，不得产生第二套健康真相；
- 多租户隔离提供客户可读的证据物（租户覆盖矩阵、越权用例结果、豁免理由），供企业安全评审使用。

详细设计见 [deployment-upgrade-safety-diagnosis-and-design.md](./deployment-upgrade-safety-diagnosis-and-design.md)。

## 产品化优先级

### P0：先形成可售卖闭环

1. 将剩余业务域的 BPMN/通知触发继续收敛为持久化 outbox/worker，并固化积压、重放和死信运维；
2. 固化“事件—CI—SLA—流程—恢复”和“变更—CI—风险—审批—PIR”验收旅程；
3. 完成 CMDB 阿里云 ECS Job/Worker/Diff/对账/审计闭环；
4. 统一后端 capability/readiness 与前端入口；
5. 为每条主链路补租户、RBAC、审计、失败恢复和 E2E 测试；
6. 建立升级与发布安全基线：预检脚本（环境/漂移/索引/legacy 重放）、发布证据包，并将「脚本已写未执行」纳入事故定义。

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
7. 开源版与企业版的分界是否按「自检/诊断（开源引流）」与「治理/编排/承诺（商业收费）」切分？该决策直接决定 Upgrade Guard / Doctor / Isolation Proof 的代码边界。

## HANDOFF

该能力合同已经足够支撑 P0 架构评审和实施拆分。下一步不应继续横向新增菜单，而应建立“商业验收旅程”执行计划：先实现可靠 command/outbox 基座，再并行固化事件/变更两条旅程和 CMDB 真实同步，最后以生产 E2E、故障恢复、租户隔离和审计报告作为发布门禁。
