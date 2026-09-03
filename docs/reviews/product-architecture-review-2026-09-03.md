# ITSM 产品架构审查与复盘报告

> 审查日期：2026-09-03  
> 审查基线：`main@26345538`（包含审查时工作区未提交改动）  
> 审查方法：静态代码与生产 Router/Worker 接线核查、现有自动化测试与产品契约复核、竞品官方资料对照。未连接生产环境，本文不构成容量、安全或上线验收证明。

## 1. 执行摘要

项目已经不是“工单 Demo”，而是一个边界较完整的企业 ITSM 模块化单体：ITIL 四件套、BPMN、CMDB、SLA、知识/RAG、资产、租户、邮件与飞书均有真实后端入口；可靠 command/outbox、租约、fencing、死信和重放也已形成可复用底座。当前最准确的产品定位是：**工单/事件、CMDB 核心、SLA、RBAC/租户具备 GA 候选基础，其余大多仍是 Pilot；整个产品尚不应整体宣称 ServiceNow 级生产完备。**

核心判断：

- **优势**：国内私有部署、开源可控、BPMN 可编排、ITIL 与 CMDB 上下文结合、AI 走网关并保留确定性降级、异步执行可靠性设计明显强于一般开源工单系统。
- **主要短板**：跨模块业务闭环和生产验收弱于功能广度；变更、请求交付、云发现、知识发布、连接器、管理报表仍缺“最后一公里”；部分前端同时存在真实 API 与契约兼容/Mock 痕迹。
- **最大架构风险**：`handlers/<domain>` 与 160 个顶层 `service/*.go` 并存，领域所有权仍分散；BPMN 名义上采用 `lib-bpmn-engine + CustomProcessEngine`，实际持久化生产语义主要由自研引擎承担；Pilot 页面数量过多，容易造成销售承诺超前。
- **建议路线**：未来两个版本停止扩菜单，先完成变更原子性、请求交付补偿、CMDB 发现 worker、知识索引一致性、真实渠道投递、管理指标口径与跨租户 E2E 六个闭环。

## 2. 范围、口径与代码规模

### 2.1 统计结果

| 指标 | 数量 | 统计口径 |
| --- | ---: | --- |
| 后端领域目录 | 67 | `itsm-backend/handlers/` 一级目录 |
| 标准领域 Handler 文件 | 65 | `handlers/**/handler.go` |
| 顶层业务 Service 文件 | 160 | `service/*.go`，排除测试 |
| 旧 `controller/` 生产 Go 文件 | 0 | 当前目录已无生产 Go 文件，迁移方向正确 |
| 静态 HTTP 注册调用 | 1,031 | Router/handlers 中 `.GET/.POST/.PUT/.PATCH/.DELETE` 文本计数；包含测试装配和兼容注册，**不等于 1,031 个唯一 API** |
| Ent Schema | 133 | `ent/schema/*.go` |
| 前端页面入口 | 166 | App Router 的 `page.tsx` |
| 前端 API client 文件 | 79 | `src/lib/api/` 顶层 TypeScript 文件，排除测试 |
| 后端手写 Go LOC | 233,075 | 排除生成的 `ent/` 和 Swagger `docs/` |
| 前端 TS/TSX LOC | 237,773 | `itsm-frontend/src/` |
| 后端测试文件 | 261 | 排除 Ent 生成目录 |
| 前端测试文件 | 200 | `*.test.ts(x)` |

规模证据应结合结构理解：总路由装配集中在 [`router/router.go`](../../itsm-backend/router/router.go)，领域装配在 [`internal/bootstrap/app.go`](../../itsm-backend/internal/bootstrap/app.go)；前端页面位于 [`src/app`](../../itsm-frontend/src/app)。大而全的文件数说明覆盖面，不说明流程已通过生产验收。

### 2.2 成熟度定义

- **GA 候选**：生产入口、核心状态规则、租户/RBAC、主要测试均存在，可以进入客户验收，但不等于已完成生产认证。
- **Pilot**：存在真实实现和可运行入口，但跨模块闭环、故障恢复、权限矩阵、容量或 E2E 至少有一项明显缺口。
- **骨架/Disabled**：仅模型、页面、适配器或显式关闭的能力，不应对外承诺。

## 3. 模块完整性审计

### 3.1 总览

| 模块 | 成熟度 | 已实现 | 主要缺口 |
| --- | --- | --- | --- |
| 事件管理 | GA 候选 | 独立实体/仓储/服务/Handler、重大事件、CI、规则、监控与告警 | `force` 更新策略、权威时间、告警源治理、容量与恢复演练 |
| 问题管理 | Pilot | 问题生命周期、调查、根因、Known Error、关联事件 | 根因 CI 强引用、重复事件聚类、解决后知识发布闭环 |
| 变更管理 | Pilot | 风险、CI 影响、审批/CAB、PIR、标准变更、BPMN 桥接 | BPMN 与业务状态非原子、窗口冲突、影响门禁、回滚演练 |
| 服务请求 | Pilot | 独立请求、目录关联、审批链、BPMN、Provisioning 基础 | 目录版本快照、履约补偿、交付任务与 CI 变更一致性 |
| BPMN/审批 | Pilot | 定义、版本、绑定、实例、任务、变量、网关、会签、ServiceTask durable command | 引擎双轨语义、计时/边界事件覆盖、补偿、实例级恢复 E2E |
| AI | Pilot | Chat、Triage、Summarize、RAG、技能/工具、审计骨架 | evaluator/反馈非强制、prompt/model 版本覆盖不足、高风险动作治理 |
| CMDB 核心 | GA 候选 | CI 类型/属性、CI、关系、历史、拓扑、影响、导入导出、对账基础 | 图查询总预算、历史写入原子性、数据质量治理、CSDM/服务模型 |
| CMDB 云发现 | Pilot/未就绪 | 云账号、阿里云适配、资源身份、发现能力状态 | 生产 worker 与 tenant secret resolver 未就绪，多云、Diff、退役治理 |
| 服务目录/审批链 | Pilot | 目录 CRUD、审批链、服务请求桥接 | 版本化发布、动态表单契约、SLA/流程/CI 类型统一绑定 |
| SLA | GA 候选 | 策略、模板、截止时间、暂停/恢复、预警、违规、监控 | 工作日历、跨域统一、分布式权威时钟、升级动作验收 |
| NOC 工作台 | Pilot | 真实重大事件查询、筛选、分页 | KPI 由当前页计算；无告警流、时间线、协同、值班、拓扑与 Runbook 聚合 |
| 知识库/RAG | Pilot | 文章 CRUD、关键词/向量搜索、问答降级 | 向量删除/一致性、文章版本发布、可见性/RBAC E2E、反馈评估 |
| 资产/License | Pilot | 资产生命周期、分配/退役、许可证席位与统计 | 采购/合同/折旧/盘点/发现、回收席位、续费告警、审计与异步任务租户边界 |
| 多租户/MSP | GA 候选/Pilot | tenant middleware、tenant predicate、RLS、MSP allocation/context | system actor 统一、跨域权限矩阵、MSP 计费/配额/白标/客户隔离验收 |
| 飞书 | Pilot | OAuth、Webhook、工单同步、durable sync command | 回调验签/重放防护、租户密钥生命周期、双向状态映射、健康检查与投递审计 |
| 邮件 Intake | Pilot+ | 来源/客户/合同/值班、解析、入站编排、出站 durable command | 邮件线程幂等、附件安全、退信/DSN、真实邮箱 provider 运维 E2E |
| 报表/仪表盘 | Pilot | Dashboard、ITIL/SLA/CMDB 局部读模型与导出基础 | 统一指标语义、数据口径版本、计划报表、MSP 汇总、Mock 高级监控清理 |

### 3.2 ITIL 四件套

**事件管理。** 真实入口从 [`router/router.go`](../../itsm-backend/router/router.go) 的 `/incidents` 进入 [`handlers/incident/handler.go`](../../itsm-backend/handlers/incident/handler.go)，经 [`service.go`](../../itsm-backend/handlers/incident/service.go) 和 [`repository_impl.go`](../../itsm-backend/handlers/incident/repository_impl.go) 落库；生命周期契约见 [`lifecycle_contract_test.go`](../../itsm-backend/handlers/incident/lifecycle_contract_test.go)。重大事件、监控指标和告警分别由 [`incident_monitoring_service.go`](../../itsm-backend/service/incident_monitoring_service.go) 与 [`incident_alerting_service.go`](../../itsm-backend/service/incident_alerting_service.go) 支撑。缺口是生产告警源接入与去重/抑制、战情协同和完整恢复旅程，而非基础 CRUD。

**问题管理。** [`handlers/problem`](../../itsm-backend/handlers/problem) 已形成垂直切片，另有 [`problem_investigation_service.go`](../../itsm-backend/service/problem_investigation_service.go)、[`root_cause_analysis_service.go`](../../itsm-backend/service/root_cause_analysis_service.go) 和 Known Error Handler。当前受影响 CI 仍存在弱引用路径，事件聚类、根因 CI 与知识发布未形成强制事务闭环，因此保持 Pilot。

**变更管理。** 生产路由 `/changes` 接入 [`handlers/change`](../../itsm-backend/handlers/change)，具备审批历史/outbox 测试、BPMN bridge、CAB、PIR 与标准变更。关键风险是流程推进与领域状态落库不是单一原子提交；影响摘要未成为提交审批的服务端门禁，维护窗口冲突与自动回滚仍不足。证据见 [`service_bpmn_bridge_test.go`](../../itsm-backend/handlers/change/service_bpmn_bridge_test.go)、[`repository_approval_outbox_test.go`](../../itsm-backend/handlers/change/repository_approval_outbox_test.go) 和 [`pir_service.go`](../../itsm-backend/service/pir_service.go)。

**服务请求。** `/service-requests` 从 Router 进入 [`handlers/service_request`](../../itsm-backend/handlers/service_request)，已有 BPMN bridge 和审批链回归测试；目录在 [`handlers/service_catalog`](../../itsm-backend/handlers/service_catalog)。主要缺少可追溯的目录版本快照、失败补偿、履约任务/SLA/CI 更新的统一事务或 Saga 语义。

### 3.3 BPMN 工作流与审批

生产调用链为：`router.go -> handlers/bpmn.Handler.RegisterRoutes -> workflow/process trigger services -> CustomProcessEngine -> Ent + operational_commands worker`。装配证据在 [`handlers/bpmn/handler.go`](../../itsm-backend/handlers/bpmn/handler.go) 和 [`internal/bootstrap/app.go`](../../itsm-backend/internal/bootstrap/app.go)。

已支持定义/部署、实例、用户任务、变量、排他/并行/包容网关、会签、审批记录、监控、Lint 和 AI 生成。ServiceTask 不在事务内调用远端：[`bpmn_process_engine.go`](../../itsm-backend/service/bpmn_process_engine.go) 写 durable command，Worker 再由 [`bpmn_service_task_command_handler.go`](../../itsm-backend/service/bpmn_service_task_command_handler.go) 执行并用 lease/fencing 校验完成。

**技术选型复盘：** `go.mod` 引入 `github.com/nitram509/lib-bpmn-engine v0.2.4`，封装见 [`pkg/bpmn/engine_adapter.go`](../../itsm-backend/pkg/bpmn/engine_adapter.go)；该适配器明确写明上游不支持状态导出/恢复。生产持久化、任务、网关、历史和可靠 ServiceTask 语义主要来自 [`CustomProcessEngine`](../../itsm-backend/service/bpmn_process_engine.go)。因此应将选型描述为“**BPMN XML/运行时能力参考与适配 + 自研持久化流程内核**”，不能简单宣称第三方引擎天然具备企业恢复能力。

处理器方面，当前适配层注册 Ticket、Incident、Change、ServiceRequest 和 Generic Handler；通知与 Webhook 更多通过 connector/notification command 解耦，而不是每种 BPMN 元素都有独立完备的原生 handler。缺少边界事件、定时器精确恢复、补偿事务、子流程/消息关联等企业 BPMN 语义的系统验收。

### 3.4 AI、知识库与 RAG

生产 AI API 位于 `/ai`，实现见 [`handlers/ai`](../../itsm-backend/handlers/ai)。底层包括 [`llm_gateway.go`](../../itsm-backend/service/llm_gateway.go)、[`triage_service.go`](../../itsm-backend/service/triage_service.go)、[`summarize_service.go`](../../itsm-backend/service/summarize_service.go)、[`rag_service.go`](../../itsm-backend/service/rag_service.go) 与 [`vector_store.go`](../../itsm-backend/service/vector_store.go)。

- Chat/Triage/Summarize/RAG Search 均有真实路径，不是单纯聊天页面。
- 模型调用经 LLM Gateway，便于 provider 切换、超时和错误分类；无模型时保留确定性 fallback，ITIL 主链不应被 AI 阻断。
- RAG 支持向量与关键词路径，但默认关键词后备是进程内能力；向量删除、文章发布版本与索引一致性仍不足。
- [`ai_evaluator.go`](../../itsm-backend/service/ai_evaluator.go) 与 telemetry 已存在，但 confidence、模型/prompt 版本、接受/拒绝反馈没有成为所有 AI 能力的强制数据契约。
- 工具/技能框架存在，但高风险操作仍应坚持建议优先、显式授权、审计和 durable command；当前不足以承诺自治运维。

### 3.5 CMDB

生产路由单独定义于 [`router/cmdb_routes.go`](../../itsm-backend/router/cmdb_routes.go)，核心执行经 [`handlers/cmdb/production_service.go`](../../itsm-backend/handlers/cmdb/production_service.go) 及顶层 CI type/item/relationship/history 服务。CI 类型、动态属性、CI 实例、关系类型、拓扑、影响分析、历史、导入导出、保存视图均已存在，且近期测试覆盖根 CI 与关联资源租户隔离，见 [`production_topology_test.go`](../../itsm-backend/handlers/cmdb/production_topology_test.go)。

云发现仍是明确 Pilot：[`internal/bootstrap/app.go`](../../itsm-backend/internal/bootstrap/app.go) 将 `WorkerReady` 设为 `false`，并注释 tenant secret resolver 与 durable worker 后续补齐；[`handler_capabilities_test.go`](../../itsm-backend/handlers/cmdb/handler_capabilities_test.go) 也断言缺少 `tenantSecretResolver`、`discoveryWorker`。现阶段主要有阿里云适配与资源身份/任务生命周期基础，尚缺生产采集 worker、多云适配、来源优先级、Diff/对账、孤儿资源退役和规模测试。

拓扑最大深度有限制，但每层可取全部关系，缺总节点/边预算与分页/截断契约；高扇出图存在资源消耗风险。CI 保存与历史写入也需要进一步确认原子性。

### 3.6 服务目录、SLA 与 NOC

服务目录和服务请求已有独立实体与审批链，但目录项到表单 schema、BPMN definition version、SLA policy、CI type/provisioning template 的绑定还不是统一的发布快照。建议引入 `catalog_item_version`，已发布版本不可变，实例仅引用版本 ID。

SLA 有策略、模板、监控、告警、暂停/恢复与升级服务，生产入口位于 `/sla`，实现证据为 [`handlers/sla`](../../itsm-backend/handlers/sla)、[`sla_monitor_service.go`](../../itsm-backend/service/sla_monitor_service.go) 和 [`sla_alert_service.go`](../../itsm-backend/service/sla_alert_service.go)。不足是工作日历、节假日、跨地域时区、分布式权威时间以及事件/请求/变更统一适用性。

NOC 页面 [`app/(main)/noc/page.tsx`](../../itsm-frontend/src/app/(main)/noc/page.tsx) 查询真实重大事件，但“高优事件”“正在处理”由当前 20 条分页数据计算，不能代表全量；页面也没有消费已有 IncidentAlert/Metric 服务。它目前是重大事件列表视图，不是完整 NOC 作战台。

### 3.7 资产与 License

[`handlers/asset/routes.go`](../../itsm-backend/handlers/asset/routes.go) 注册资产与许可证 CRUD、分配、退役和统计；[`asset_license_service.go`](../../itsm-backend/service/asset_license_service.go) 对席位分配使用事务和条件更新，具备基础并发保护。

产品完整性仍有限：缺采购、合同、供应商履约、成本/折旧、盘点、资产发现与 CMDB reconciliation 闭环；License 缺回收/转移席位、软件安装发现与合规核算、续费工作流。`CheckLicenseStatus` 扫描全部租户记录，应明确 system context、分租户批处理、租约、审计和失败恢复后再进入生产 Worker。

### 3.8 多租户与 MSP

请求链有 tenant middleware，领域仓储广泛使用 `TenantIDEQ`，数据库侧还有 [`database/rls`](../../itsm-backend/database/rls)；MSP `/api/v1/msp` 提供 context、allocation、客户工单和报表入口。异步跨租户扫描使用显式 system context 的设计已经出现。

风险在“一致性”而非“完全没有隔离”：133 个 schema、旧顶层 service 与后台任务使手工 predicate 容易遗漏；部分管理员角色和 system bypass 范围仍需统一。MSP 目前更接近委派访问 Pilot，计费、套餐/配额、白标、客户级密钥与审计视图仍不完整。所有新增租户资源应继续要求真实 Router 跨租户拒绝测试，RLS 只作为纵深防御。

### 3.9 飞书、邮件 Intake、报表

飞书真实入口在 [`router/feishu_routes.go`](../../itsm-backend/router/feishu_routes.go)：OAuth、公开 callback、公开 Webhook、工单同步；异步同步 command 在 bootstrap 注册。最大风险是公开入口必须有强制验签、时间窗/nonce 重放防护、instance 到 tenant 的安全解析，以及失败投递审计。还缺卡片交互、审批/评论/状态的稳定双向映射和连接器健康检查闭环。

邮件 Intake 的实现深度高于一般 Pilot：[`handlers/email_intake/orchestrator.go`](../../itsm-backend/handlers/email_intake/orchestrator.go) 将入站处理和出站邮件接入 operational command，相关事务测试在 [`service_test.go`](../../itsm-backend/handlers/email_intake/service_test.go)。上线前仍需 MIME/附件大小与恶意内容治理、Message-ID/线程幂等、退信处理、邮箱限流和真实 provider 故障演练。

报表入口 `/reports` 与 dashboard service 已存在，但前端 [`components/reports/RealTimeMonitoring.tsx`](../../itsm-frontend/src/components/reports/RealTimeMonitoring.tsx) 和 [`AdvancedAnalytics.tsx`](../../itsm-frontend/src/components/reports/AdvancedAnalytics.tsx) 仍内置 Mock 数据，且 [`product-capabilities.ts`](../../itsm-frontend/src/config/product-capabilities.ts) 显式关闭 advanced reporting。现阶段应只承诺只读局部汇总；管理驾驶舱、指标版本、计划分发、钻取、导出审计和 MSP 安全聚合均待补。

## 4. 架构质量评估

### 4.1 总体结构

模块化单体适合当前阶段：单事务可以覆盖领域写入、审计和 outbox，部署成本也适合国内私有化。`handlers/<domain>` 垂直切片已经覆盖 67 个领域，旧 controller 生产文件归零，这是正确方向。

但 160 个顶层 service 文件仍承载 BPMN、CMDB、AI、SLA、通知等大量业务规则，与领域目录形成“新垂直切片 + 旧共享服务层”并存。建议按用例所有权逐步搬迁，不做一次性重构：

1. 先为每个路由维护 `route -> handler -> use case -> repository/command` 清单。
2. 新规则只进入领域 package；跨域只调用公开 service/command。
3. 当顶层 service 只有单一领域消费者时，迁回该领域并保留 facade 兼容。
4. 给 shared service 定义稳定接口、tenant/context 约束和错误分类。

### 4.2 异步可靠性

[`internal/commandbus/commandbus.go`](../../itsm-backend/internal/commandbus/commandbus.go) 已实现 durable `operational_commands`、唯一幂等键、claim/lease、heartbeat、fencing token、指数退避、最大次数和 dead-letter；运维 API 在 `/admin/operations/commands` 提供列表、详情、重放和取消。BPMN ServiceTask、通知、邮件、工单/事件规则、飞书同步、CMDB 导入导出等已逐步接入。

这是项目的重要竞争资产，但仍需收口：

- 为所有外部副作用建立 `recipient + channel` delivery 记录，并在外部调用前查成功状态。
- provider 支持幂等 token 时透传 command key，覆盖“外部成功、本地审计失败”。
- replay 保留原始业务幂等身份并单独审计，不生成新 key 绕过防重。
- 清点仍在请求提交后 enqueue、同步网络调用、裸 goroutine 的遗留路径。
- 将 Worker lag、lease lost、retry、dead-letter、replay 变为低基数指标和告警。

### 4.3 多租户安全

当前是“应用 predicate + PostgreSQL RLS +显式 system context”的正确组合，优于只依赖 Header 或 ORM hook 的方案。改进重点应放在关联 ID 注入、聚合/Raw SQL、缓存 key、导出和 Worker 重新加载权威资源上。MSP 必须始终区分 operator tenant、customer tenant 与 delegated scope，不能用角色名推导跨租户权限。

### 4.4 前端架构与契约

App Router、API client 和产品 capability 开关已经建立，但 166 个页面说明产品面大于当前 GA 面。NOC 中 `resp.incidents || resp.items || []` 是典型双契约兼容，违反单一 DTO 原则；高级报表仍含 Mock。应以 Router/DTO 为唯一事实源，逐个删除多字段 fallback 和未注册 client，并让菜单同时依赖 build、deployment、tenant readiness 与 actor permission。

## 5. 产品竞争力评估

### 5.1 对标结论

| 维度 | 本项目 | ServiceNow | Atlassian JSM | 国内/本地化产品 | 差距判断 |
| --- | --- | --- | --- | --- | --- |
| ITIL 核心 | 四件套均有真实实现，成熟度不齐 | 完整套件、重大事件/值班/流程挖掘 | 请求/事件/问题/变更成熟，DevOps 协同强 | 蓝鲸重流程；嘉为/ManageEngine 强一体化交付 | 本项目“广度可比、深度不足” |
| 工作流 | BPMN + 自研持久化内核，可审计 | Flow/Workflow 平台与海量应用 | 无代码 automation、生态联动 | 蓝鲸可视化流程与平台能力突出 | 本项目标准化潜力强，运营工具和处理器生态弱 |
| CMDB | 核心模型/关系/拓扑较完整，发现 Pilot | CSDM、Discovery、Service Mapping、IRE、Health | Assets、Discovery、30+ adapters、Data Manager | 国内产品强调 CMDB+AIOps+自动化一体化 | 最大差距是发现、reconciliation、数据健康和服务模型 |
| AI | 网关、RAG、技能、审计和降级设计合理 | AI agents、Control Tower、Agent Fabric、丰富用例 | Atlassian Intelligence/Rovo、AIOps | 国内厂商加速 AIOps/知识问答 | 架构方向先进，缺可量化 evaluator 与生产数据闭环 |
| 生态 | connector/skill/plugin/CLI 有方向 | 企业工作流、Service Graph、应用商店 | 3,000+ 集成生态 | 蓝鲸 PaaS/作业/CMDB 生态，本地交付强 | 生态数量、认证、版本治理和商业伙伴体系差距巨大 |
| 部署/本地化 | 开源、轻量私有部署、飞书优先 | 平台能力强但成本与实施复杂 | 云与 Data Center，研发工具链优势 | 私有化、信创、本地实施成熟 | 本项目可用轻量和开放切入，但需补信创与交付认证 |
| 运营与分析 | 局部指标，advanced reporting disabled | Platform Analytics、Process Mining | 报表、AIOps、资产分析 | 项目制报表与大屏成熟 | 管理者价值证明不足 |

ServiceNow 当前套餐把 Service Catalog、Incident、Asset/CMDB、Virtual Agent、Workflow Data Fabric、Service Graph connectors 作为基础，并在更高层提供 Major Incident、On-call、Change、Problem、Process Mining 和 AI agents；其 CMDB 强调 CSDM、Discovery、Service Mapping、IRE 与 Health。Atlassian JSM 的优势是研发协同、无代码 automation、Assets/Discovery、30+ adapters 与 Data Manager。国内侧，腾讯蓝鲸 ITSM 强调可视化流程和蓝鲸 PaaS/CMDB/标准运维联动；嘉为蓝鲸强调监控、告警、CMDB、自动化一体化；ManageEngine 中国站覆盖 ITIL、资产发现、软件许可、合同、报表以及微信/钉钉集成。

资料来源（访问于 2026-09-03）：

- [ServiceNow ITSM 套餐与能力](https://www.servicenow.com/products/itsm/pricing.html)
- [ServiceNow CMDB](https://www.servicenow.com/products/servicenow-platform/configuration-management-database.html)
- [ServiceNow ITSM Agentic AI 用例](https://www.servicenow.com/docs/r/it-service-management/now-assist-for-it-service-management-itsm/now-assist-itsm-ai-agents-use-cases.html)
- [Atlassian JSM 功能](https://www.atlassian.com/software/jira/service-management/features)
- [Atlassian Assets 与 CMDB](https://www.atlassian.com/software/jira/service-management/features/asset-and-configuration-management)
- [腾讯蓝鲸 bk-itsm](https://github.com/TencentBlueKing/bk-itsm)
- [嘉为蓝鲸 ITSM](https://www.canway.net/ITSM/1055.html)
- [ManageEngine ServiceDesk Plus 中国站](https://www.manageengine.cn/products/service-desk/index-integ.html)

### 5.2 差异化卖点

1. **真正开源且面向国内企业的 AI-Native ITSM**：不是仅在工单旁放 Chat，而是让 AI 输出进入审批、知识、CMDB、流程与审计边界。
2. **BPMN 作为统一编排语言**：比私有规则 DSL 更容易交付、迁移和形成模板市场；领域状态机仍保留最终裁决权。
3. **可靠副作用作为平台能力**：command/outbox + lease/fencing/dead-letter/replay 可成为连接器、技能和 Agent 执行的共同安全底座。
4. **轻量私有部署与国产协作入口**：飞书先行，并可扩展企业微信、钉钉、Webhook；比 ServiceNow 实施轻，比海外 SaaS 更适合数据驻留要求。
5. **开放扩展面**：Connector、Skill、Plugin、CLI 共同形成二次开发和伙伴交付入口，潜在壁垒是经过治理的模板/连接器资产，而不是代码量。

### 5.3 商业化路径

- **社区版**：完整核心 ITIL、基础 BPMN、CMDB 核心、单租户/基础多租户、开放 API；以 Apache-2.0 获客并建立模板社区。
- **企业版**：SSO/SCIM、审计保留、HA、备份恢复、RLS 强化、组织级权限、国产数据库/信创认证、企业连接器、生产运维工具。
- **AI/自动化增值包**：模型网关治理、私有模型、evaluator、知识索引、Agent/Skill 审批与额度；按席位 + AI 用量或节点计费。
- **MSP 版**：客户租户、委派范围、SLA 分账、白标、配额/账单、跨客户安全报表；按客户实例/受管资产/技术员组合收费。
- **市场与交付**：连接器、流程模板、行业 Skill、CMDB discovery adapter 分成；建立签名、权限 manifest、兼容矩阵、审核和撤回机制。
- **优先行业**：先选择私有化强、飞书/企微普及、流程复杂但 ServiceNow 成本敏感的 200–3000 人组织；以“6–8 周可验收闭环”而不是“功能列表”销售。

## 6. 风险清单与改进优先级

### 6.1 Pilot 完成度风险

| 优先级 | 风险 | 影响 | 建议验收门槛 |
| --- | --- | --- | --- |
| P0 | 变更审批/BPMN 与业务状态非原子 | 审批成功但变更状态不一致 | 同事务或可证明的 Saga；失败/重复/恢复 E2E |
| P0 | CMDB 云发现生产依赖显式未就绪 | 页面存在但无法可靠采集 | tenant secret、worker、lease/fencing、Diff/对账、退役全链路 |
| P0 | 公开飞书 callback/Webhook 安全闭环不足 | 伪造、重放、跨租户注入 | 验签、时间窗、nonce、instance-tenant 校验、审计回归 |
| P0 | 知识/RAG 可见性与索引一致性不足 | 跨权限泄露或过期答案 | 发布版本 + ACL 双阶段过滤 + 删除/重建/回滚 E2E |
| P1 | 服务请求履约缺补偿 | 目录请求与外部资源状态分裂 | 版本快照、步骤 delivery、补偿/replay、CI 结果绑定 |
| P1 | NOC/报表指标口径不可靠 | 管理决策失真 | 服务端聚合、指标版本、全量口径、钻取一致性 |
| P1 | License 定时扫描缺明确租户 Worker 语义 | 扫描越界、失败不可恢复 | 分租户 durable job、system reason、租约与审计 |
| P1 | AI evaluator 与反馈非强制 | 无法证明效果与安全性 | 所有建议记录 model/prompt/confidence/evidence/feedback |

### 6.2 技术债务

1. **领域所有权分散**：67 个 handler 领域与 160 个顶层 service 并存，容易形成绕过公开用例的调用。
2. **BPMN 引擎定位模糊**：第三方 adapter 与自研生产内核并存，需要 ADR 明确每层职责、支持的 BPMN 子集、恢复语义和迁移策略。
3. **超大 Router**：[`router.go`](../../itsm-backend/router/router.go) 约 1,991 行，应按领域拆注册文件并生成路由清单，但不能改变现有 URL/ACL。
4. **前端契约兼容**：多字段 fallback 会掩盖 DTO 漂移；NOC 已有实例，应逐项按契约测试删除。
5. **页面面大于交付面**：166 个页面中存在 Disabled 能力和 Mock 展示，应由 runtime capability 驱动，避免“可见即承诺”。
6. **测试数量不等于覆盖质量**：261/200 个测试文件是良好基础，但当前全量 Go 测试仍有已知编译阻塞记录；需要以真实 Router/Worker、跨租户和失败恢复为准。
7. **监控与错误分类不统一**：外部 provider、command、AI、connector 应共享稳定 error class、低基数指标与安全日志字段。

### 6.3 建议实施顺序

**0–30 天：可信度收口**

- 修复全量测试编译阻塞，冻结新页面；建立当前 commit 的 backend/frontend/GA gate 基线。
- 完成变更提交审批的原子性方案与回归；完成飞书公开入口安全测试。
- 清除生产 UI 中 Mock 和双字段 fallback；Disabled 功能默认不可见。
- 建立 Pilot 清单，每项指定 owner、退出条件和可执行验收命令。

**31–90 天：打通三条可销售闭环**

- 事件：告警进入 -> 重大事件 -> CI 影响 -> SLA -> 协同 -> PIR -> 问题/知识。
- 变更：风险/CI -> 窗口冲突 -> BPMN/CAB -> 实施/回滚 -> PIR -> CMDB 更新。
- 请求：版本化目录 -> 审批 -> durable provisioning -> 补偿 -> SLA -> 交付验收。
- 同期完成知识 ACL/索引一致性、CMDB discovery worker 和统一指标服务。

**91–180 天：商业化与生态**

- 企业版 HA/备份/审计/SSO/信创验收；MSP 配额、账单和客户隔离。
- 飞书转 GA，再按同一 connector lifecycle 扩展企微、钉钉和 Webhook。
- 发布签名 Connector/Skill manifest、兼容矩阵、市场审核与撤回机制。
- 用真实客户数据建立 AI evaluator：分诊准确率、摘要采纳率、检索引用正确率、自动化节省时长和安全拒绝率。

## 7. 目标架构决策

建议保持模块化单体，不立即拆微服务；将 `operational_commands` 作为跨域副作用边界，将 BPMN 作为编排层，将领域 Service 作为状态规则唯一所有者，将 CMDB/知识作为上下文数据层，将 AI 作为带证据和审计的建议/受控执行层。

```text
Web / 飞书 / 邮件 / API
          |
  Auth + Tenant + RBAC
          |
Domain Handler -> Domain Use Case -> Repository/Ent + Audit
          |                 |
          |          same transaction
          +------> operational_commands
                         |
              lease/fencing Worker
                         |
        BPMN / Connector / Notification / AI Job

BPMN 负责编排；领域服务裁决状态；CMDB/Knowledge 提供租户内上下文；
AI 输出必须带 model/prompt/confidence/evidence/feedback/audit。
```

## 8. 最终结论

产品已经具备国内开源企业 ITSM 的有竞争力底座，尤其是“BPMN + CMDB + AI + durable command + 私有部署”的组合。短期成功不取决于再增加模块，而取决于把 Pilot 收敛成可重复验收的闭环。建议对外使用“核心能力 GA 候选、扩展能力 Pilot”的分层表述；完成 P0 后选择 2–3 个设计伙伴跑 90 天生产试点，用可量化 SLA、MTTR、变更失败率、知识命中率与 AI 采纳率证明价值，再扩大连接器与市场投入。

