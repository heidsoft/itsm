# 开源产品能力说明

> Status: current。最后复盘：2026-11-07。本文描述当前源码可交付的业务边界，不以页面、规划或历史测试报告代替运行时事实。成熟度与运行时能力矩阵（`GET /api/v1/capabilities`，源 `handlers/capability/handler.go`）保持同口径。

## 产品承诺

开源用户可以从干净检出开始，通过 Docker Compose 部署一个可登录、可验证、可扩展的 ITSM 实例。产品围绕“请求进入—类型解析—分派与时限—流程执行—解决与审计”构建业务主线，并以 CMDB、知识、AI 和连接器提供上下文与扩展能力。

## 使用角色与核心闭环

| 角色 | 可以完成的核心工作 |
|:---|:---|
| 员工/请求人 | 从服务门户或工单入口提交请求，填写类型驱动的动态表单，跟踪处理状态 |
| 服务台工程师 | 分类、分派、关联 CI、处理 SLA、记录沟通并解决工单或事件 |
| 流程参与人 | 接收 BPMN 用户任务，完成审批或处理，并保留实例与执行历史 |
| ITIL 负责人 | 管理事件、问题、变更、服务目录、知识和 SLA 策略 |
| CMDB 管理员 | 管理 CI 类型、实例、关系、拓扑、历史与影响分析 |
| 平台管理员 | 管理租户、用户、角色权限、TicketType、流程绑定、审计与部署状态 |

工单入口的当前执行链路为：

`TicketType/Preset → 动态表单校验 → Ticket 与类型快照持久化 → Assignment/SLA/Workflow 统一绑定解析 → command/outbox → Worker 执行 → 历史与审计`

TicketType 是运行时配置，不是前端静态枚举。创建工单时会校验租户、启用状态、字段类型、必填项和引用权限，并保存 `ticketTypeId`、类型快照与 `formFields`；后续类型配置变化不会抹去已创建工单的业务语义。

## 当前能力矩阵

状态含义：**GA 候选**表示可进入企业生产验收；**Pilot**表示存在真实实现，但仍需按部署场景补齐闭环或非功能验证；**Disabled/规划中**不得作为交付承诺。

| 业务域 | 状态 | 当前可验证能力 | 主要边界 |
|:---|:---:|:---|:---|
| 工单、事件 | GA 候选 | 生命周期、分类、分派、CI、SLA、评论、附件、可靠工作流触发 | 仍需按目标容量验证恢复与并发 |
| TicketType 与动态表单 | GA 候选 | 字段设计、完整 Renderer、类型快照、Preset 安装、归档恢复、Workflow/SLA/Assignment 绑定、独立 RBAC 和审计 | 复杂表单升级与大规模配置需专项验收 |
| 变更管理 | GA 候选 | 风险评估、受影响 CI、提交门禁（关键 CI/开放事件强制 high 风险等级 + 窗口冲突拦截）、审批链原子收口（终态流转与审批链同事务）、会签/阈值/多级链在审批端点真实生效、审批资格后端判定随响应下发、实施结果如实记录（失败/回滚原因必填）、回滚清单逐项确认、PIR | 大规模并发审批与复杂会签场景建议生产前按场景验收（与工单同容量口径） |
| 问题/Known Error | GA 候选 | 根因分析、临时方案、关联事件、Known Error、关闭证据校验（无解决方案或根因结论一律 409）、SLA 真实契约（创建即按 problem 绑定 sla_states）、评论分页、知识沉淀（草稿进评审流） | 无已知阻断；按目标容量验收恢复与并发（与工单同口径） |
| 服务目录/服务请求 | Pilot | 目录、请求、BPMN 唯一编排审批（三级审批网关 + 审批计划变量，免审批目录项直接履约）、审批资格门控；阿里云 ECS RunInstances 真实交付（租户云账号 `secret://` 凭据引用、ClientToken 幂等、失败审计带对账证据、永久性云错误快速失败、人工裁决落审计谱系）；2026-11-02 真实账号端到端验证已跑到 RunInstances（凭据/签名/账号/地域链路验证通过，云侧因余额不足拒绝） | ① 仅覆盖 ECS 一种资源；② 缺自动释放/回滚动作，中途失败靠幂等键对账与人工判定兜底；③ 目录项到 CI Type、交付模板、BPMN、SLA 的绑定口径分散；④ 一次真实账号创建并释放 ECS 的闭环验收被账号余额阻塞 |
| CMDB 核心 | GA 候选 | CI 类型、实例、关系、历史、拓扑、影响分析、导入导出基础 | 数据质量、规模、恢复与调和需场景验收 |
| 云发现 | Pilot | Job 执行器（租约认领/diff/证据落库/终态）+ 每分钟调度消费 `discovery_sources.schedule`、五厂商适配器（阿里云/腾讯云/AWS/火山引擎/K8s）、云服务类型基线随 cmdb-core 预置、按「云服务类型 → CI 类型」映射回写配置项、必填属性缺口转人工补录 | 每轮发现是全量分页列举，没有增量水位线（只按 missing_count 判定既有资源失联）；真实账号下的发现尚未验收 |
| CMDB 调和 | Pilot | 差异计算与决策（`POST /discovery/results/:id/apply|reject`），应用后同步刷新配置项并把反查 ID 记在差异上 | 回写只覆盖基线内云主机/Pod 类型；云上连续缺失只把配置项标成同步滞后、不给退役建议；缺真实数据验收 |
| BPMN/审批 | Pilot | 定义、版本、绑定、实例、任务、变量和历史；审批资格由后端判定并随响应下发；工单与事件使用持久化 command/outbox 启动流程 | 问题域创建不落 workflow.start（problem 分支无生产者，生命周期仅在已有实例时桥接）；ServiceTask 外部副作用与流程完成分属两个事务；补偿只支持结束事件驱动且无内置模板；存量 pending 待办与跨域实例级 E2E 未补 |
| SLA | GA 候选 | 策略、业务日历、截止时间、预警、违规与指标 | 暂停恢复和跨领域一致性需持续验证 |
| 知识/RAG | Pilot | 文章、可见性过滤（SQL 先过滤后分页、作者豁免、权限不可得时失败关闭）、关键词/向量检索、推荐与降级、发布门禁、`published_version` 线上版本指针（三条发布入口与在线更新同事务维护）、索引删除一致性（删除/驳回/撤编/版本恢复走同事务 outbox 同步清理向量库） | 典型问题集召回率与真实语料引用质量的人工验收未做 |
| AI | Pilot | LLM Gateway、分诊、摘要、RAG、预测、chat 全能力面强制 promptVersion/model/confidence 并落 ai_audit 底账（采纳/拒绝经 PATCH 翻转）、LLM 调用按租户计量与配额/成本口径、`cmd/ai-eval` 四类 golden 集评测门禁（接真实生产路径，RAG/预测不自我实现）、记忆层（会话实体回读 + 历史工单处置经验召回，行级数据范围过滤） | LLM 档与真实生产语料的验收尚未由人执行——指标达标与否没有真实运行与签字记录，CI 降级档不能替代；离线评测结果只落 JSON 文件，无落库与前端报告面 |
| 通知/连接器 | 通知 GA 候选；连接器 Pilot | 通知：outbox 投递、幂等键、失败重试、投递审计读面、渠道诊断。连接器：飞书/企微/钉钉 Send/HealthCheck/验签/入站解析 + 回调路由 + 审计注入 + 死信重放 + 市场安装闭环；飞书验签按官方算法（`sha256(timestamp+nonce+encrypt_key+body)` 拼接，常量时间比较）并带 nonce 防重放，Encrypt Key 加密回调已支持 AES 解密（含加密的 url_verification 请求，challenge 按官方 SDK 语义明文回包） | 连接器覆盖仍停在单测与路由测试：无真实租户收发/验签联调、无真实故障下的死信重放演练；渠道部署就绪由 READY 环境变量申报 |
| 报表与运营 | Pilot | 管理者服务质量视图 `GET /reports/service-quality`（KPI/SLA/事件分布/满意度/趋势/工单统计单请求聚合，2026-11-07 起问题/变更域统计块接入同一响应；带 metricVersion 口径版本，行级 scope=tenant/own），前端「报表中心」顶部总览已真实消费（窗口切换、块缺省显式降级展示）；reportCatalog 七张报表与路由共用同一清单（无死链，report:read + 域 read 双闸门） | MSP 跨租户汇总口径未定义；导出只有动作级审计、缺行数与数据范围摘要 |
| 应用市场/扩展交付 | GA | 目录与版本发布、连接器与技能安装、升级与回滚、`auto_upgrade` 由 worker 真正执行、存量依赖对齐报告、安装记录商品字段与配置脱敏 | 没有制品分发层：技能装载的是随二进制发布的内置代码，第三方技能上传/签名/沙箱执行未实现；插件不交付（平台无宿主运行时，安装返回 `501`、上架返回 `400`）；真实租户「上架→安装→升级→回滚」尚未由客户语料验收 |
| RBAC/多租户 | GA 候选 | 角色权限、Endpoint ACL、租户过滤、TicketType 独立权限、审计 | 新域仍必须补 ACL 与跨租户回归 |

## 约束

- 默认部署路径必须以 `docker compose` 和 `.env.example` / `.env.prod.example` 为入口，不依赖本地私有配置。
- 前端默认后端地址统一为 `http://localhost:8090`；Docker 生产构建使用 `http://itsm-backend:8090`。
- API 健康检查以 `/api/v1/health` 为准。
- 所有用户可见的写入操作必须真实持久化，或显式提示未接入后端；禁止用本地模拟数据伪装成功。
- 默认账号只能作为开发/首次安装引导出现，生产部署文档必须要求更改密码和密钥。
- 多租户、RBAC、审批、TicketType 引用校验和服务请求状态流转属于发布门禁，不能作为演示功能处理。

## 实现契约

- Actors: 系统管理员、服务台工程师、审批人、普通员工、开源部署者。
- Surfaces: `README.md`、`.env.example`、`.env.prod.example`、`docker-compose.dev.yml`、`docker-compose.prod.yml`、`docker-compose.e2e.yml`、`docker-compose.fresh-gate.yml`、前端自助门户、后端 `/api/v1/*`（仓库根没有 `docker-compose.yml`）。
- States and transitions: TicketType 支持 draft/active/inactive 与归档恢复；禁用或跨租户类型不得用于新建工单。ITIL 状态流转由后端服务校验。
- Interfaces: HTTP JSON 使用 camelCase，Handler 返回 DTO；工单类型以 `/api/v1/ticket-types` 和 `/api/v1/ticket-type-presets` 为真实来源，服务目录以 `/api/v1/service-catalogs` 和 `/api/v1/service-requests` 为真实来源。
- Data implications: 收藏、评分、门户配置、服务分析、目录导出在后端模型和接口补齐前，不应显示为可持久化能力。
- Operator requirements: 发布前需要通过前端 `type-check`、前端生产构建、后端 `go test ./...`、Docker Compose 健康检查和登录烟测。

## 非目标

- 不在本能力中新增商业化计费、多组织 SaaS 订阅或云市场分发。
- 不承诺所有规划中的 AI 能力在无 LLM Key 时完整可用；必须有清晰降级或关闭入口。
- 不用前端假数据替代未完成的后端服务。

## 开源用户的验收路径

1. 按根目录 README 启动开发环境，确认前后端健康检查通过。
2. 使用管理员创建或安装一个 TicketType Preset，配置动态字段及 Workflow/SLA/Assignment 绑定。
3. 使用普通用户读取启用类型并创建工单，确认非法字段、禁用类型和跨租户引用被拒绝。
4. 确认工单保存类型/表单快照，并产生预期的 SLA、流程实例、任务和审计记录。
5. 生产前执行后端全量测试、前端类型检查与构建、API 契约检查、数据库迁移演练、备份恢复和目标环境 E2E。

具体命令与运维要求见[部署指南](../deployment-optimization.md)、[生产初始化运行手册](../delivery/production-initialization.md)和[测试指南](../testing/README.md)。运行时是否向当前用户开放某能力，以认证后的 `GET /api/v1/capabilities` 为准。

## 明确不承诺

- Pilot 能力不等于未经验收即可用于生产；开源发行版也不承诺某个云厂商、消息渠道或 LLM 在所有环境开箱即用。
- 页面、路由、Schema、seed 数据或测试文件存在，不单独证明业务闭环已完成。
- AI 默认提供建议与降级路径，不绕过租户、RBAC、审批或人工责任。
- Scaffold、演示数据和未来 Roadmap 不计入当前可交付能力。

## 贡献与后续方向

贡献者应优先沿现有领域边界扩展：业务规则放后端服务，流程使用 BPMN/command，企业集成进入 connector lifecycle，AI 调用进入 LLM Gateway，所有新增表和接口同时补 tenant、ACL、DTO、审计与回归测试。当前优先级见根目录 [ROADMAP](../../ROADMAP.md)。
