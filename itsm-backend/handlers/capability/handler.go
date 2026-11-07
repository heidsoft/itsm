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
		pilotReason: "交付已接阿里云 ECS RunInstances 真实 Provider（租户云账号 + secret:// 或 env://ITSM_ALIYUN 凭据引用，未配置即失败关闭），审批→交付→SLA/流程/CMDB 收尾已打通；ClientToken 幂等、失败审计带对账证据、永久性云错误快速失败、人工裁决落审计谱系均已闭环。2026-11-02 真实账号端到端验收已跑到 RunInstances（云侧因账号余额不足拒绝，凭据/签名/账号/地域链路全部验证通过）。剩余阻断：① 只覆盖 ECS 一种资源；② 缺自动释放/回滚动作，中途失败靠幂等键对账与人工判定兜底；③ 目录项到 CI Type、交付模板、BPMN、SLA 的绑定口径仍分散。晋升条件：账号充值后完成一次真实账号创建并释放 ECS 的闭环验收 + 目录项绑定统一 + 自动补偿或第二资源类型。",
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
		pilotReason: "执行链与定时调度已闭环（Job 执行器 + 每分钟 cron 消费 schedule），五个厂商适配器已装配，云服务类型基线随 cmdb-core 预置，发现结果按「云服务类型 → CI 类型」映射回写配置项（表外类型显式跳过，必填属性凑不齐时不建账）；凑不齐必填项的资源已有补录入口（GET /discovery/pending-resources 列出属性缺口，POST .../register 补录建账）。剩余阻断：每轮发现都是全量分页列举，没有增量水位线（只按 missing_count 判定既有资源失联），真实账号下的发现未验收。晋升条件：增量水位线 + 真实账号下的全量与增量发现验收。",
	},
	{
		key: "cmdbReconciliation", maturity: MaturityPilot, built: true, dependencies: []string{"cmdb", "cmdbDiscovery"}, readinessEnv: "ITSM_CMDB_DISCOVERY_READY",
		pilotReason: "差异能算、能决策（POST /discovery/results/:id/apply|reject），应用后同步刷新配置项并把反查 ID 记在差异上；但回写目前只覆盖基线内的云主机/Pod 类型，云上连续缺失只把配置项标成同步滞后、不给出退役建议，且缺少真实数据验收。晋升条件：真实数据对账验收 + 覆盖更多资源类型。",
	},
	// 2026-10-31 运行时降为 Pilot，与 README 成熟度表、商业能力契约对齐：
	// 此前矩阵对客户下发 GA，但问题域创建从不落 workflow.start 命令
	// （WorkflowStartCommandHandler 的 problem 分支无生产者，生命周期仅在已有
	// 实例时桥接），ServiceTask 外部副作用与流程完成仍分属两个事务，补偿只支持
	// 结束事件驱动且无内置模板。
	{
		key: "workflow", maturity: MaturityPilot, built: true,
		pilotReason: "流程定义、绑定、实例、任务、变量与历史齐备，工单与事件已使用持久化 command/outbox 启动流程，但运行时仍有跨域缺口：问题域创建不落 workflow.start（problem 分支无生产者，生命周期仅在已有实例时桥接）、ServiceTask 外部副作用与流程完成分属两个事务、补偿只支持结束事件驱动且无内置模板、存量 pending 待办与跨域实例级 E2E 未补。晋升条件：问题域 workflow.start 生产者 + ServiceTask 与流程完成同事务 + 补偿内置模板 + 跨域实例级 E2E。",
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
		pilotReason: "管理者服务质量视图（GET /reports/service-quality）已把 KPI/SLA/事件分布/满意度/趋势/工单统计聚合成单请求并携带 metricVersion 口径版本，2026-11-07 起问题/变更域统计块接入同一响应（复用两域 GetStats 端点的同一份映射），报表页也有真实消费入口（前端「报表中心」顶部总览，含窗口切换与块缺省的显式降级展示）；报表目录也不再有死链（清单与路由共用同一份 reportCatalog，守卫测试逐条断言每个 path 都已注册）。但 MSP 跨租户汇总边界仍未定义（服务商视角拿不到下属租户的合并口径），导出只有动作级审计、缺少导出行数与数据范围摘要。晋升条件：MSP 汇总口径 + 导出产物审计。",
	},
	// 2026-10-30 「线上是哪个版本」已收敛到 published_version 指针：发布时
	// （评审通过 /publish / PUT status 三条入口）在同一事务内为即将上线的正文
	// 落版本快照并把指针指过去；已发布文章的在线更新（正文类字段变化但不翻转
	// is_published）同样落新快照并移动指针，撤回与恢复清指针。发布时刻及此后
	// 每次线上正文变更在版本表里都有对应行，指针是唯一事实源；客户端自带的
	// publishedVersion 一律忽略。
	{
		key: "knowledge", maturity: MaturityPilot, built: true,
		pilotReason: "发布版本语义已收敛：published_version 指向线上正文对应的版本快照，三条发布入口（评审通过 /publish / PUT status=published）与已发布文章的在线更新统一维护该指针，撤回与恢复清指针，客户端无法伪造；2026-10-31 分类可见性也已在列表、详情、分类、统计、推荐/最近、检索与问题页反查读面统一收口：SQL 先过滤后分页（total 真实）、作者豁免、权限状态不可得时失败关闭，跨角色回归已覆盖。但知识检索仍缺人工验收口径：典型问题集召回率与真实语料下的引用质量未做验收。晋升条件：典型问题集召回率人工验收 + 真实语料引用质量验收。",
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
		pilotReason: "分诊与工单 RCA 已强制 promptVersion/model/confidence 并落 ai_audit 底账（采纳/拒绝经 PATCH 翻转，不再由前端自报）；LLM 调用已按租户归集（ai_llm_calls 带 tenant_id/request_id，/ai/metrics 为租户口径；升级前的历史行无法回填归因，按哨兵值排除、租户无归因行时如实报不可用）。溯源已覆盖全能力面且 model 不再失真：分诊/RCA 之外，摘要（method 三态 llm/direct/fallback + 真实模型名）、预测（口径版本 + 真实置信度 + auditId）、chat（检索模式 + 真实命中分 + auditId）均已强制溯源并落 ai_audit 底账；此前 TriageService 把 model 写死成 gpt-4o-mini（生产配的并不是它，底账系统性地指向错误模型）、邮件抽取器未配模型时兜底成同一个硬编码值，现改由 provider 自报（ModelReportingProvider），报不出时如实写 unknown。深度分析（SQL 聚合）刻意不补 model 而只给口径版本与 method=aggregate——给没有模型参与的聚合伪造 model 比缺失更危险。token 配额与成本口径已落地（AIQuotaService：租户月度配额从既有 ai_llm_calls 计量读取不建新表、未配置=不限制且如实报 quotaConfigured=false、超限网关拒绝并留计量痕、评估失败 fail-open——配额是预算保护不是安全边界；单价未配置时 estimatedCost=0 且 costConfigured=false，绝不给猜的单价；/ai/metrics 附带 quota/cost，告警分 warning/exceeded 两级）。评测门禁的自我实现已修（此前候选=期望答案，三项指标恒满分且把实现改坏照样绿），且 CI 评测已接真实生产路径：RAG 期望按文章标题引用真实语料（datasets/corpus/rag_corpus.jsonl，与库的主键解耦），检索走生产 service.RAGService 关键词路径（真实 SQL，召回率 0.80 为真实测得而非自我实现）；预测分数由生产 SLABreachPredictor 产出（排序层=预计解决时长/SLA 窗口，与人工标注独立产生），标注依据逐条留存；摘要候选走 SummarizeService 真实降级路径并有退化敏感性守卫。LLM 档验收通道已就绪（cmd/ai-eval：由人执行、读 llm.* 配置构造真实 provider、配置缺失报错不悄悄降级、产出归档报告，指标与 CI 同口径）。记忆层已落地（短期层）：会话历史回读并抽出实体/实义短语，作为**独立补充检索路**参与召回——此前每轮只用当前 query 检索，「它怎么修」这类指代在整串包含语义下必然零命中，是结构性的不可能；实测指代型 query 由零命中变为命中相关文档。不新建表（数据本就在 messages 表）、不拼接 query（拼接会让当前 query 也命不中）、记忆条目带 fromMemory/memoryQuery 标注且在 prompt 中显式区分「会话记忆线索」与「知识库正文」，不得让模型把「聊过」说成「文档里写着」。情景层亦已落地：历史工单处置经验（tickets.resolution）按标题关键词召回——此前 RAG 关键词检索只覆盖知识文章，「上次这个故障怎么解决的」结构上答不了；情景召回必须经 datascope.ApplyTicketFilter 行级过滤（工单是业务数据，把用户无权查看的工单处置喂给 LLM 等于绕过权限外泄内容），且只取已解决且有处置结论的工单，条目标 object_type=ticket_experience 并在 prompt 中标注「历史工单处置经验，非知识库正文」——历史结论是先例不是标准流程。语义层（RAG+知识库）与程序性层（Skills）此前已具备。渐进式披露亦已落地：检索结果带 truncated/contentLength 信号位（模型此前无从判断片段是否完整，拿半截内容当全文硬答），新增只读工具 get_kb_article 按需读全文（此前工具清单只有检索没有读全文，需要细节的问题结构上答不了——evidence gap 无法闭合）；读全文与检索共用同一套准入（租户/未软删/已发布/时效/分类可见性），两处口径不同会出现「检索命中但读不到」或「绕过权限的读通道」；读不到时报错而非返回空内容（空内容会让模型当「没有正文」然后编造）。剩余阻断：LLM 档与真实生产语料的验收**尚未由人执行过**——指标达标与否没有任何一次真实运行与签字记录，CI 的降级档不能代替真实模型的质量结论（此句曾在记忆层批次合成时意外丢失，本次恢复）。晋升条件：跑一次 cmd/ai-eval -mode llm + 真实知识库语料的质量验收并归档报告（召回率/摘要质量/预测 AUC 达标，验收人确认）。剩余阻断：LLM 档与真实生产语料的验收**尚未由人执行过**——指标达标与否没有任何一次真实运行与签字记录，CI 的降级档不能代替真实模型的质量结论。晋升条件：跑一次 cmd/ai-eval -mode llm + 真实知识库语料的质量验收并归档报告（召回率/摘要质量/预测 AUC 达标，验收人确认）。",
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
		pilotReason: "Send/HealthCheck/VerifySignature/ParseInbound 与入站去重、回调路由、审计注入齐备，死信重放（/admin/operations）与市场安装闭环也已接通；验签按官方算法（sha256(timestamp+nonce+encrypt_key+body) 拼接）实现，Encrypt Key 加密回调支持 AES 解密（含加密 url_verification，challenge 明文回包）。但这些覆盖仍停在单测与路由测试，未与真实飞书租户做生产验收，也没有在真实渠道故障下演练过死信重放。晋升条件：真实租户收发 + 验签 + 一次真实故障的死信重放演练。",
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
		pilotReason: "工单标签（ticket_tags）CRUD、工单绑定与 /admin/tags 页已完整，种子基线也不再错写进组织标签表；但组织标签（tags 表，挂项目/应用/微服务/部门/团队）只有只读查询（GET /api/v1/system/tags 与兼容别名 GET /api/v1/tags），仓储里的 Create/Delete 没有注册路由、整套 TagService 未装配，两张表并存且组织标签侧无写入口。晋升条件：组织标签管理面（写入与挂接）或与工单标签收敛。",
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
		pilotReason: "SSO/OIDC 的 11 个端点与服务齐备，已有 httptest 假 IdP 覆盖登录→回调→换票→JWT 会话整条链路，租户未启用 IdP 时 health 如实降级；但没有与真实 IdP 的联调证据（验签算法、声明映射与时钟偏差以线上为准）。晋升条件：至少一家真实 IdP 的登录/回调/换票验收。",
	},
	// 企微/钉钉连接器（2026-10-21 Disabled→Pilot 转正）：Send/HealthCheck/VerifySignature/
	// ParseInbound 四项核心与飞书同层级实现（connector/builtin/{wecom,dingtalk}，编译期
	// 接口断言），回调路由与审计注入已在 bootstrap 装配（app.go Connector 段 → RouterConfig
	// → RegisterRoutes）。渠道诊断（GET /notifications/channels）对两渠道返回真实健康信号。
	// 与 feishu 同款 readinessEnv 门控：部署是否就绪由 ITSM_WECOM_READY / ITSM_DINGTALK_READY
	// 如实申报，代码能力不再按「未包含在商业发布」隐藏。
	{
		key: "connector.wecom", maturity: MaturityPilot, built: true, dependencies: []string{"notification"}, readinessEnv: "ITSM_WECOM_READY",
		pilotReason: "回调验签与消息收发未与真实企业微信租户联调；OAuth 授权、任务双向同步等飞书级深度集成未做。晋升条件：真实租户收发 + 验签验收。",
	},
	{
		key: "connector.dingtalk", maturity: MaturityPilot, built: true, dependencies: []string{"notification"}, readinessEnv: "ITSM_DINGTALK_READY",
		pilotReason: "回调验签与消息收发未与真实钉钉租户联调；工作通知/待办同步等深度集成未做。晋升条件：真实租户收发 + 验签验收。",
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
