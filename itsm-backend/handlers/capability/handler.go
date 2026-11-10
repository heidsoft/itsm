// Package capability exposes the product capability control plane.
package capability

import (
	"os"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/domain/role"

	"github.com/gin-gonic/gin"
)

// acceptanceVersion 是能力矩阵最近一次对账的日期（YYYY.MM.DD）：核对 registry
// 成熟度、README/商业契约口径、pilotReason 与对应回归证据。它不是发布版本，
// 也不代表全部能力已验收；但不得早于 CHANGELOG 最新发布版本，发版未复核矩阵
// 会让 TestAcceptanceVersionIsNotBehindLatestRelease 失败。
const acceptanceVersion = "2026.10.31"

type Maturity string

const (
	MaturityGA       Maturity = "ga"
	MaturityPilot    Maturity = "pilot"
	MaturityDisabled Maturity = "disabled"
)

// Capability is the backend-owned contract used by every product surface.
type Capability struct {
	Key               string     `json:"key"`
	Maturity          Maturity   `json:"maturity"`
	BuildAvailable    bool       `json:"buildAvailable"`
	DeploymentReady   bool       `json:"deploymentReady"`
	TenantReady       bool       `json:"tenantReady"`
	AllowedActions    []string   `json:"allowedActions"`
	Dependencies      []string   `json:"dependencies"`
	DegradedReason    string     `json:"degradedReason,omitempty"`
	LastHealthCheckAt *time.Time `json:"lastHealthCheckAt,omitempty"`
	// PilotReason 回答「为什么它还不是 GA」，以及晋升需要补哪块可验证证据。
	// 只有 maturity=pilot 时非空。引入原因：此前的理由只写在 Go 注释里，
	// 修完阻断项后注释没人回头改，矩阵对客户呈现的仍是几个月前的结论
	// （2026-10-22 盘点发现 problem/template 的注释所引用的缺陷早已不存在）。
	PilotReason string `json:"pilotReason,omitempty"`
	// Health 是探测到的客观健康信号；无探测维度时为 nil（JSON 省略）。
	// 与 LastHealthCheckAt 的区别：后者回答「何时探测」，前者回答「探测结果」。
	Health            *Health `json:"health,omitempty"`
	AcceptanceVersion string  `json:"acceptanceVersion"`
}

type definition struct {
	key          string
	maturity     Maturity
	built        bool
	dependencies []string
	readinessEnv string
	// pilotReason 仅 pilot 条目必填，且必须是「当前仍然成立」的阻断项 +
	// 可验证的晋升条件；ga 条目必须留空（留着就是过期理由）。
	// 由 TestPilotReasonsAreCurrentAndActionable 强制。
	pilotReason string
}

var registry = []definition{
	{key: "ticket", maturity: MaturityGA, built: true},
	{key: "incident", maturity: MaturityGA, built: true, dependencies: []string{"workflow", "sla", "notification"}},
	// 问题域 2026-10-22 晋升 GA：此前两个申报 GA 的阻断项都已闭环并有装配证据——
	//   1) 关闭证据校验（无解决方案或无根因结论一律 409）；
	//   2) KEDB 关联落库（problemId 进 DTO、同租户校验、写 problem 边）；
	//   3) SLA 真实契约（创建即按 service_type="problem" 绑定 sla_states，
	//      resolved/closed 打解决时间戳，GET /problems/:id/sla 返回 bound/status，
	//      在 internal/bootstrap/app.go 里 SetSLAService 已注入，不是测试里才有的路径）；
	//   4) 评论独立表 + 分页读面。
	// 2026-10-22 盘点时发现矩阵注释仍写着「SLA 端点仍 503」——那是 SLA 落地之前的
	// 旧结论，代码里 503 桩早已移除（handlers/ 全目录只剩通知投递一处 503，
	// 语义是「outbox 未装配」）。注释过期导致的假 Pilot 会隐藏一个已可投产的能力。
	{key: "problem", maturity: MaturityGA, built: true, dependencies: []string{"sla", "workflow"}},
	{key: "change", maturity: MaturityGA, built: true, dependencies: []string{"workflow"}},
	// 服务请求维持 Pilot：2026-10-31 已接阿里云 ECS RunInstances 真实交付并打通
	// 审批→交付→SLA/流程/CMDB 收尾；2026-11-02 真实账号端到端验收跑到
	// RunInstances（云侧因账号余额不足拒绝，凭据/签名/账号/地域链路均已验证通过），
	// 并补齐 ClientToken 幂等、失败对账证据、永久错误快速失败、人工裁决审计与
	// 创建阶段审批计划 fail-closed。剩余阻断是：只覆盖 ECS、缺自动释放/回滚动作、
	// 目录项绑定口径分散，以及一次成功创建并释放 ECS 的闭环验收（账号侧阻塞）。
	// 理由必须与代码同步，由 TestServiceRequestPilotReasonReflectsDeliveryReality 双向锁定。
	{
		key: "serviceRequest", maturity: MaturityPilot, built: true, dependencies: []string{"workflow", "notification"},
		pilotReason: "Delivery uses the real Alibaba Cloud ECS RunInstances provider with tenant cloud accounts and secret:// or env://ITSM_ALIYUN credential references; missing configuration fails closed. Approval, provisioning and SLA/workflow/CMDB completion are connected. ClientToken idempotency, failure audits with reconciliation evidence, fast failure for permanent cloud errors and audited manual adjudication are implemented. Real-account end-to-end acceptance on 2026-11-02 reached RunInstances: insufficient account balance prevented creation, while credentials, signatures, account and region were verified. Remaining blockers: only ECS is supported; automatic release/rollback is missing, so partial failures require idempotency reconciliation and manual decisions; catalog bindings to CI types, delivery templates, BPMN and SLA remain scattered. Promotion criteria: fund the account and complete real-account ECS creation and release acceptance, unify catalog bindings, and add automatic compensation or a second resource type.",
	},
	{key: "cmdb", maturity: MaturityGA, built: true},
	{key: "cmdb.configurationItems", maturity: MaturityGA, built: true, dependencies: []string{"cmdb"}},
	{key: "cmdb.ciTypes", maturity: MaturityGA, built: true, dependencies: []string{"cmdb"}},
	{key: "cmdb.relationships", maturity: MaturityGA, built: true, dependencies: []string{"cmdb"}},
	{key: "cmdb.topology", maturity: MaturityGA, built: true, dependencies: []string{"cmdb"}},
	// 云发现 2026-10-23 两条阻断项闭环：① 执行者——Job 级执行器消费
	// `cmdb.cloud_discovery.run`（租约认领/diff/证据落库/终态），旧 Runner.RunAll
	// 删除；② 调度——`discovery_sources.schedule` 此前全仓无消费者，现已由
	// DiscoveryScheduler 每分钟按租户时区消费（到期判定、在途守卫、跨副本幂等键）。
	// 2026-10-25 复核适配器数量：「只有阿里云 ECS 一种」已过期，注册表装了
	// aliyun/ecs、tencent/cvm、aws/ec2、volcengine/ecs、k8s/pod 五个。
	// 同批补上 cloud_services 产品基线（cmdb-core 组件按注册表装配结果预置，
	// 与适配器由守卫测试双向锁定）——此前全仓无 INSERT 与 seed，新租户的
	// 发现任务会在 scope 解析处整批失败。2026-10-31 复核：发现没有增量通道，
	// 每轮全量分页列举、只按 missing_count 判失联；剩下的阻断是增量水位线与真实账号验收。
	{
		key: "cmdbDiscovery", maturity: MaturityPilot, built: true, dependencies: []string{"cmdb"}, readinessEnv: "ITSM_CMDB_DISCOVERY_READY",
		pilotReason: "The execution chain and scheduler are connected: the Job executor runs discovery and minute-level cron consumes schedules. Five provider adapters and the cmdb-core cloud service baseline are installed. Discovery maps cloud service types to CI types; unknown types are skipped and resources missing required attributes are not registered. GET /discovery/pending-resources lists missing attributes, and POST .../register completes registration. Remaining blockers: scans perform full pagination without incremental watermarks; missing_count only detects missing existing resources; discovery has not been accepted against real accounts. Promotion criteria: incremental watermarks and real-account full and incremental discovery acceptance.",
	},
	{
		key: "cmdbReconciliation", maturity: MaturityPilot, built: true, dependencies: []string{"cmdb", "cmdbDiscovery"}, readinessEnv: "ITSM_CMDB_DISCOVERY_READY",
		pilotReason: "Differences can be calculated and decided through POST /discovery/results/:id/apply|reject. Applying a difference refreshes its CI and records the reverse CI ID. Write-back currently covers only baseline cloud VM/Pod types. Repeatedly missing cloud resources are marked as lagging synchronization without retirement recommendations, and real-data acceptance is missing. Promotion criteria: real-data reconciliation acceptance and more resource types.",
	},
	// 2026-10-31 运行时降为 Pilot，与 README 成熟度表、商业能力契约对齐：
	// 此前矩阵对客户下发 GA，但问题域创建从不落 workflow.start 命令
	// （WorkflowStartCommandHandler 的 problem 分支无生产者，生命周期仅在已有
	// 实例时桥接），ServiceTask 外部副作用与流程完成仍分属两个事务，补偿只支持
	// 结束事件驱动且无内置模板。
	{
		key: "workflow", maturity: MaturityPilot, built: true,
		pilotReason: "Definitions, bindings, instances, tasks, variables and history are available. Tickets and incidents start workflows through durable command/outbox records. Runtime gaps remain: the problem domain does not produce workflow.start on creation and bridges lifecycle only when an instance already exists; ServiceTask external side effects and workflow completion use separate transactions; compensation supports only end-event triggers and has no built-in templates; existing pending tasks and cross-domain instance-level E2E coverage remain incomplete. Promotion criteria: a problem domain workflow.start producer, atomic ServiceTask and workflow completion, built-in compensation templates and cross-domain instance-level E2E coverage.",
	},
	{key: "sla", maturity: MaturityGA, built: true},
	// 报表域此前只在能力契约表里（报表/运营 = Pilot），矩阵里根本没有这一行——
	// 矩阵是前端与控制台的唯一事实源，缺行等于这个域对客户不可见。
	// 2026-10-23 服务质量视图落地后补行，理由按当前仍缺的两项写。
	// 2026-10-30 清单死链已修：报表目录原本自报 7 张报表，其中 cmdb-quality 与
	// catalog-usage 只有清单项、没有注册路由（点了 404）。清单提取为 reportCatalog
	// 由接口与守卫测试共用，两条报表接上既有数据源（CMDB 治理看板 / 服务目录
	// 统计），并挂 report:read + 域 read 双闸门——报表页不是权限的旁路。
	{
		key: "report", maturity: MaturityPilot, built: true,
		pilotReason: "GET /reports/service-quality aggregates manager-facing KPI, SLA, incident distribution, satisfaction, trends and ticket statistics in one request with a metricVersion definition version. Report paths share reportCatalog with route registration, and guard tests verify every path is registered. MSP cross-tenant aggregation boundaries remain undefined, so providers cannot obtain consolidated customer metrics. Export audits record the action but lack exported row counts and a data-scope summary. Promotion criteria: MSP aggregation definitions and export artifact auditing.",
	},
	// 2026-10-30 「线上是哪个版本」已收敛到 published_version 指针：发布时
	// （评审通过 /publish / PUT status 三条入口）在同一事务内为即将上线的正文
	// 落版本快照并把指针指过去；已发布文章的在线更新（正文类字段变化但不翻转
	// is_published）同样落新快照并移动指针，撤回与恢复清指针。发布时刻及此后
	// 每次线上正文变更在版本表里都有对应行，指针是唯一事实源；客户端自带的
	// publishedVersion 一律忽略。
	{
		key: "knowledge", maturity: MaturityPilot, built: true,
		pilotReason: "published_version identifies the snapshot of the online article. Review approval, /publish, PUT status=published and online edits consistently maintain the pointer; withdrawal and restoration clear it, and clients cannot forge it. Since 2026-10-31, category visibility is enforced in lists, details, categories, statistics, recommendations/recent items, retrieval and reverse lookup from problems: SQL filters before pagination, totals are accurate, authors retain access, permission lookup failures fail closed and cross-role regressions are covered. Retrieval still lacks human acceptance against representative question sets and real-corpus citation quality. Promotion criteria: human recall acceptance on representative questions and citation quality acceptance with real content.",
	},
	// AI 2026-10-23 三契约落地（分诊 + 工单 RCA）：响应带 promptVersion/model/auditId，
	// 服务端自动落 ai_audit 底账（accepted=NULL），采纳/拒绝走 PATCH 翻转；
	// confidence 统一 0-1 并入口校验。2026-10-26 LLM 调用按租户归集：
	// ai_llm_calls 增 tenant_id/request_id（迁移 20261026），LLMObserver 从请求 ctx
	// 回填归属，/ai/metrics 改租户口径（无归因行时如实 responseTimeAvailable=false）。
	// 仍不申报 GA：代码侧三项验收缺口（虚构 ID、预测分值自给、假验收开关）
	// 已于 2026-10-31 闭环——期望改按文章标题引用 + 真实语料/真实检索器（召回率
	// 0.80 为真实测得）、预测分数由生产 SLABreachPredictor 产出（与人工标注独立）、
	// cmd/ai-eval 提供可执行验收命令。剩下的阻断是**人工执行**的 LLM 档与真实
	// 语料验收尚未跑过，指标达标与否没有人签过字。
	{
		key: "ai", maturity: MaturityPilot, built: true, readinessEnv: "ITSM_AI_READY",
		pilotReason: "Triage and ticket RCA require promptVersion/model/confidence and persist ai_audit records; acceptance/rejection is changed through PATCH rather than client assertions. ai_llm_calls attributes calls by tenant_id/request_id and /ai/metrics is tenant-scoped. Historical calls without attributable ownership are excluded using sentinel values; tenants without attributed rows are reported unavailable. Provenance also covers summaries (llm/direct/fallback method and actual model), predictions (definition version, actual confidence and auditId), and chat (retrieval mode, actual hit scores and auditId). ModelReportingProvider supplies the actual model; unavailable identity is recorded as unknown rather than the former hard-coded gpt-4o-mini in triage and email extraction. SQL aggregation uses method=aggregate and a definition version without inventing a model. AIQuotaService reads monthly tenant usage from ai_llm_calls without a new table. Unconfigured quotas are unlimited with quotaConfigured=false; exceeding quota is rejected and metered by the gateway. Quota evaluation failures fail open because quotas protect budgets rather than security boundaries. Missing unit prices yield estimatedCost=0 and costConfigured=false; /ai/metrics includes quota/cost with warning/exceeded levels. Evaluation no longer compares expected answers to themselves. CI exercises production RAGService keyword retrieval using real SQL and article titles in datasets/corpus/rag_corpus.jsonl rather than database IDs; recall 0.80 is measured. Production SLABreachPredictor scores use estimated resolution duration divided by the SLA window, independently of documented human labels. Summary candidates use the real SummarizeService fallback with degradation sensitivity guards. Human-run cmd/ai-eval reads llm.* configuration, constructs real providers, fails on missing configuration and archives reports using the CI metric definitions. Short-term memory reads conversation history and extracts entities/meaningful phrases as an independent retrieval path, enabling follow-up references. It reuses messages without a new table or concatenating the current query. Entries carry fromMemory/memoryQuery and prompts distinguish conversation memory from knowledge text. Episodic memory retrieves resolved ticket resolutions using title keywords. datascope.ApplyTicketFilter enforces row-level access; only resolved tickets with conclusions are included, with object_type=ticket_experience and prompts identifying historical experience rather than authoritative knowledge. Semantic RAG/knowledge and procedural Skills memory already exist. Progressive disclosure adds truncated/contentLength signals and read-only get_kb_article for full text. Retrieval and full-text reads share tenant, deletion, publication, validity and category visibility checks; inaccessible full text returns an error rather than empty content. Remaining blocker: no human has executed and signed off LLM acceptance with real production content; CI fallback evaluation cannot establish real-model quality. Promotion criteria: run cmd/ai-eval -mode llm, validate real knowledge corpus recall, summary quality and prediction AUC, and archive a reviewer-approved report.",
	},
	// 通知域 2026-10-20/21 闭环三件事：投递审计读面（GET /notification-deliveries）、
	// 失败重试（保留旧记录作失败证据）、渠道诊断（GET /notifications/channels：
	// 渠道是否装配 / 是否健康 / 用户目标是否就绪）。加上既有的偏好过滤、
	// 幂等键与 outbox 重试，「没收到通知」的三类根因都能在一次请求里定位。
	{key: "notification", maturity: MaturityGA, built: true},
	// 三家连接器代码同层级（Send/HealthCheck/VerifySignature/ParseInbound + 回调路由 +
	// 审计注入），维持 Pilot 的原因是「没有真实租户联调证据」，不是实现缺失。
	{
		key: "connector.feishu", maturity: MaturityPilot, built: true, dependencies: []string{"notification"}, readinessEnv: "ITSM_FEISHU_READY",
		pilotReason: "Send/HealthCheck/VerifySignature/ParseInbound, inbound deduplication, callback routes and audit injection are implemented. Dead-letter replay through /admin/operations and marketplace installation are connected. Coverage remains limited to unit and route tests; production acceptance with a real Feishu tenant and dead-letter replay during a real channel failure have not been performed. Promotion criteria: real-tenant messaging and signature verification, plus a dead-letter replay drill after a real failure.",
	},
	// 市场目录 2026-10-23 补齐管理面：平台级权限码 marketplace_catalog:read/write
	// （只授 sysadmin，租户 admin 仍只有 marketplace:write 用于安装/卸载）+ 目录项
	// CRUD + 版本发布/回滚 + 只读控制台 /admin/marketplace。此前目录只有 seeder 能写，
	// 「应用市场」在客户环境里永远只有出厂那几条商品——这才是它停在 Pilot 的原因。
	// 2026-10-27 依赖闸门双边落地：安装前校验待装版本声明的依赖已在同租户
	// 安装且版本满足（缺口逐项 409）；卸载前反向检查是否仍被 active/disabled
	// 的已安装组件依赖（列出依赖方 409），并在 revoke 连接器之前判定。
	// skill/plugin 缺少分发/装载机制，安装与配置入口返回 501，不再创建有效安装记录。
	// auto_upgrade 有字段没有执行路径；存量安装未做一次性对齐（闸门之前落库的
	// 记录可能已悬空），升级时的重新校验也无处挂载（升级本身尚未实现）。
	//
	// 2026-11-03 上述四条阻断全部闭环，晋升 GA（按实测证据，不按注释）：
	//   1) 技能装载：版本行 manifest_path=`skill://<技能码>` 必须命中本进程
	//      SkillRegistry（internal/bootstrap/app.go 的 SetSkillRuntime 注入，
	//      与 handlers/skill 的调用准入共用同一实例）；命中即置 active 并交付，
	//      命不中则安装记录如实置 failed + error_message，不谎报已安装。
	//      种子在建库前校验技能定位符，装不出来的内置商品进不了目录。
	//   2) 升级执行：POST /items/:id/upgrade 先按目标版本复核依赖再换运行态，
	//      目标版本比当前旧即回滚。点名了不可用的版本返回 400，未点名而目录里
	//      没有可用已发布版本返回 409；运行态没能起来同样是 409 且记录保持原版本。
	//   3) auto_upgrade：PATCH /items/:id/auto-upgrade 为 *bool 显式开关并落审计，
	//      后台每轮 RunAutoUpgradePass 由 app.go 装配执行（有装配守卫锁执行者来源），
	//      单商品失败不冻结其它商品，扫描范围只含可交付且已上架的条目。
	//   4) 存量依赖对齐：GET /marketplace/dependency-status 逐条安装复核前置，
	//      区分 unsatisfied 与 declaration_missing，只读不修数据，由管理员决定
	//      补装前置还是补目录版本行。
	// 类型边界：平台只交付连接器与技能。插件需要宿主运行时，平台不托管第三方代码，
	// 因此目录写入/上架拒绝 plugin 类型（400），出厂的插件条目改为停用（deprecated）
	// 而非删除——存量安装记录与审计仍指向它，历史仍可查看与卸载。
	// 制品仓库、签名校验与沙箱执行属 roadmap（Plugin marketplace v2），不在本承诺内。
	{key: "marketplace", maturity: MaturityGA, built: true},
	{key: "project", maturity: MaturityGA, built: true},
	{key: "application", maturity: MaturityGA, built: true},
	// 工单标签（ticket_tags）已完整，缺的是另一套模型的管理面。
	// 2026-10-30 修掉一处真实污染：initialization 种子把**工单标签基线写进了
	// 组织标签表**（tags），verify 也断言 tags——而工单标签域读写的是 ticket_tags。
	// 后果是每个新环境的工单标签种子都落进死表（生产实测 tags 15 行全是错放的
	// 工单标签、ticket_tags 0 行）。现已改为写 ticket_tags，并加守卫测试锁表。
	{
		key: "tag", maturity: MaturityPilot, built: true,
		pilotReason: "Ticket tag CRUD in ticket_tags, ticket binding and /admin/tags are complete, and the seed baseline no longer writes to the organization tag table. Organization tags in tags, attached to projects/applications/microservices/departments/teams, have read-only GET /api/v1/system/tags and the compatibility alias GET /api/v1/tags. Repository Create/Delete methods have no registered routes, TagService is not wired and the two tables coexist without organization tag write operations. Promotion criteria: organization tag management for writing and association, or convergence with ticket tags.",
	},
	// 2026-10-22 晋升 GA：模板的真实表面是 /api/v1/tickets/templates 的 8 个端点
	// （列表/分类/创建/详情/更新/状态/复制/删除，权限码 ticket_template:*），
	// 前端 /tickets/templates 页（903 行）+ handler 测试齐备，菜单基线也已同码。
	// 旧注释引用的「template-api.ts 调用通用 /api/v1/templates 从未注册」是幽灵缺陷：
	// 前端仓库里不存在 template-api.ts，也不存在任何 /api/v1/templates 调用。
	{key: "template", maturity: MaturityGA, built: true},
	// SSO/OIDC 已完整实现（router/sso_routes.go 的 11 个端点 + handlers/auth/sso_*.go），
	// 此前误标为 disabled/built:false，会让按矩阵门控的界面隐藏一个可用能力。
	// 是否真正可用取决于租户有没有启用身份提供方，由 HealthFor 如实报告。
	{
		key: "identity.oidc", maturity: MaturityPilot, built: true,
		pilotReason: "All 11 SSO/OIDC endpoints and services are available. An httptest fake IdP covers login, callback, token exchange and JWT sessions; health degrades when the tenant has no enabled IdP. Integration acceptance with a real IdP is missing; production signature algorithms, claim mapping and clock skew still require validation. Promotion criteria: login, callback and token exchange acceptance with at least one real IdP.",
	},
	// 企微/钉钉连接器（2026-10-21 Disabled→Pilot 转正）：Send/HealthCheck/VerifySignature/
	// ParseInbound 四项核心与飞书同层级实现（connector/builtin/{wecom,dingtalk}，编译期
	// 接口断言），回调路由与审计注入已在 bootstrap 装配（app.go Connector 段 → RouterConfig
	// → RegisterRoutes）。渠道诊断（GET /notifications/channels）对两渠道返回真实健康信号。
	// 与 feishu 同款 readinessEnv 门控：部署是否就绪由 ITSM_WECOM_READY / ITSM_DINGTALK_READY
	// 如实申报，代码能力不再按「未包含在商业发布」隐藏。
	{
		key: "connector.wecom", maturity: MaturityPilot, built: true, dependencies: []string{"notification"}, readinessEnv: "ITSM_WECOM_READY",
		pilotReason: "Callback signature verification and messaging have not been integrated with a real WeCom tenant. Deeper integration comparable to Feishu, including OAuth and bidirectional task synchronization, is not implemented. Promotion criteria: real-tenant messaging and signature verification acceptance.",
	},
	{
		key: "connector.dingtalk", maturity: MaturityPilot, built: true, dependencies: []string{"notification"}, readinessEnv: "ITSM_DINGTALK_READY",
		pilotReason: "Callback signature verification and messaging have not been integrated with a real DingTalk tenant. Deeper work notification and to-do synchronization integration is not implemented. Promotion criteria: real-tenant messaging and signature verification acceptance.",
	},
}

// NewHandler 返回能力矩阵端点。
//
// prober 为 nil 时仍返回矩阵，但不附带健康信号——宁可没有信号，也不返回伪造的
// 「刚刚检查过」。这也让单元测试能在不搭数据库的情况下校验矩阵形状。
func NewHandler(prober *Prober) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleStr := c.GetString("role")
		tenantID := c.GetInt("tenant_id")
		tenantReady := tenantID > 0
		admin := role.IsAdminLike(roleStr)

		items := make([]Capability, 0, len(registry))
		for _, item := range registry {
			dependencies := item.dependencies
			if dependencies == nil {
				dependencies = []string{}
			}
			deploymentReady := item.built
			degradedReason := ""
			if item.maturity == MaturityDisabled {
				deploymentReady = false
				degradedReason = "capability is not included in the current commercial release"
			} else if item.readinessEnv != "" && !envEnabled(item.readinessEnv) {
				deploymentReady = false
				degradedReason = "deployment dependency is not ready"
			}

			var health *Health
			if prober != nil && item.built && item.maturity != MaturityDisabled {
				health = prober.HealthFor(c.Request.Context(), item.key, tenantID)
				if health != nil && !health.OK {
					// 探测到不可用时不得继续声称部署就绪：矩阵的价值在于让调用方
					// 据此隐藏入口，谎报就绪比不报更糟。
					deploymentReady = false
					if health.Reason != "" {
						degradedReason = health.Reason
					}
				}
			}

			actions := []string{}
			if item.built && item.maturity != MaturityDisabled && tenantReady {
				actions = append(actions, "read")
				if admin && deploymentReady {
					actions = append(actions, "manage")
				}
			}

			capability := Capability{
				Key: item.key, Maturity: item.maturity, BuildAvailable: item.built,
				DeploymentReady: deploymentReady, TenantReady: tenantReady,
				AllowedActions: actions, Dependencies: dependencies,
				DegradedReason: degradedReason, Health: health,
				AcceptanceVersion: acceptanceVersion,
			}
			// 只有 pilot 才下发理由：GA 条目带一条历史理由，等于让调用方
			// 同时看到「ga」和「它为什么还不是 GA」两个互相矛盾的事实。
			if item.maturity == MaturityPilot {
				capability.PilotReason = item.pilotReason
			}
			if health != nil {
				checkedAt := health.CheckedAt
				capability.LastHealthCheckAt = &checkedAt
			}
			items = append(items, capability)
		}

		common.Success(c, gin.H{"items": items, "acceptanceVersion": acceptanceVersion})
	}
}

func envEnabled(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
