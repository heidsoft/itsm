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

const acceptanceVersion = "2026.08-private-preview"

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
	readOnly     bool
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
	// 服务请求维持 Pilot，阻断项不在交付状态机（那一层 2026-10-21 已收口）：
	// 真正没解决的是「交付本身是不是真的」。
	{
		key: "serviceRequest", maturity: MaturityPilot, built: true, dependencies: []string{"workflow", "notification"},
		pilotReason: "交付 Provider 仍是 StubProvider，不会真正创建云资源；多步交付中途失败后的资源回滚/补偿未实现；目录项到 CI Type、交付模板、BPMN、SLA 的绑定口径仍分散。晋升条件：接真实 provider + 补偿动作 + 目录项绑定统一。",
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
	// 发现任务会在 scope 解析处整批失败。CI 回写接通后，剩下的阻断项是真实账号验收。
	{
		key: "cmdbDiscovery", maturity: MaturityPilot, built: true, dependencies: []string{"cmdb"}, readinessEnv: "ITSM_CMDB_DISCOVERY_READY",
		pilotReason: "执行链与定时调度已闭环（Job 执行器 + 每分钟 cron 消费 schedule），五个厂商适配器已装配，云服务类型基线随 cmdb-core 预置，发现结果按「云服务类型 → CI 类型」映射回写配置项（表外类型显式跳过，必填属性凑不齐时不建账、留给人工录入）；但真实账号下的全量与增量发现未验收，这类需要人工补全的资源也还没有录入入口。晋升条件：真实账号发现验收 + 人工补全入口。",
	},
	{
		key: "cmdbReconciliation", maturity: MaturityPilot, built: true, dependencies: []string{"cmdb", "cmdbDiscovery"}, readinessEnv: "ITSM_CMDB_DISCOVERY_READY",
		pilotReason: "差异能算、能决策（POST /discovery/results/:id/apply|reject），应用后同步刷新配置项并把反查 ID 记在差异上；但回写目前只覆盖基线内的云主机/Pod 类型，云上连续缺失只把配置项标成同步滞后、不给出退役建议，且缺少真实数据验收。晋升条件：真实数据对账验收 + 覆盖更多资源类型。",
	},
	{key: "workflow", maturity: MaturityGA, built: true},
	{key: "sla", maturity: MaturityGA, built: true},
	// 报表域此前只在能力契约表里（报表/运营 = Pilot），矩阵里根本没有这一行——
	// 矩阵是前端与控制台的唯一事实源，缺行等于这个域对客户不可见。
	// 2026-10-23 服务质量视图落地后补行，理由按当前仍缺的两项写。
	{
		key: "report", maturity: MaturityPilot, built: true,
		pilotReason: "管理者服务质量视图（GET /reports/service-quality）已把 KPI/SLA/事件分布/满意度/趋势/工单统计聚合成单请求并携带 metricVersion 口径版本；但 MSP 跨租户汇总边界未定义（服务商视角拿不到下属租户的合并口径），导出只有动作级审计、缺少导出行数与数据范围摘要。晋升条件：MSP 汇总口径 + 导出产物审计。",
	},
	{
		key: "knowledge", maturity: MaturityPilot, built: true,
		pilotReason: "发布版本语义未收敛（历史版本可对比/回滚，但「线上是哪个版本」没有唯一事实源）；分类可见性的跨角色检索未做端到端验收；召回率（典型问题集）缺人工验收口径。晋升条件：版本发布语义 + 跨角色检索验收。",
	},
	// AI 2026-10-23 三契约落地（分诊 + 工单 RCA）：响应带 promptVersion/model/auditId，
	// 服务端自动落 ai_audit 底账（accepted=NULL），采纳/拒绝走 PATCH 翻转；
	// confidence 统一 0-1 并入口校验。仍不申报 GA：这套契约只覆盖分诊与 RCA，
	// chat/摘要/analytics/prediction/agent 工具没有同等约束；evaluator 仍无门禁。
	{
		key: "ai", maturity: MaturityPilot, built: true, readinessEnv: "ITSM_AI_READY",
		pilotReason: "分诊与工单 RCA 已强制 promptVersion/model/confidence 并落 ai_audit 底账（采纳/拒绝经 PATCH 翻转，不再由前端自报）；但 chat、摘要、analytics、prediction、agent 工具仍未强制同一套溯源字段；ai_llm_calls 无 tenant/request 关联，成本与延迟无法按租户归集；evaluator 仍是只读报表，没有「健康分下降即阻断发布」的门禁。晋升条件：其余 AI 能力补齐溯源三契约 + LLM 调用按租户归集 + evaluator 进门禁。",
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
		pilotReason: "回调验签只有单测，未与真实飞书租户做生产验收；死信人工重放与市场安装闭环未验收。晋升条件：真实租户收发 + 验签 + 死信重放验收。",
	},
	// 市场目录 2026-10-23 补齐管理面：平台级权限码 marketplace_catalog:read/write
	// （只授 sysadmin，租户 admin 仍只有 marketplace:write 用于安装/卸载）+ 目录项
	// CRUD + 版本发布/回滚 + 只读控制台 /admin/marketplace。此前目录只有 seeder 能写，
	// 「应用市场」在客户环境里永远只有出厂那几条商品——这才是它停在 Pilot 的原因。
	// 仍不申报 GA：skill/plugin 类商品的安装只写一条安装记录，没有分发/装载机制；
	// auto_upgrade 有字段没有执行路径；dependencies 未做安装期校验。
	{
		key: "marketplace", maturity: MaturityPilot, built: true, readOnly: true,
		pilotReason: "目录管理已闭环（平台上架/下架、版本发布与回滚、控制台），但商品交付语义不完整：skill/plugin 类安装只写安装记录，没有分发/装载机制；auto_upgrade 无升级执行路径；dependencies 未做安装期校验。晋升条件：skill/plugin 装载机制 + 版本升级执行 + 依赖校验。",
	},
	{key: "project", maturity: MaturityGA, built: true},
	{key: "application", maturity: MaturityGA, built: true},
	// 工单标签（ticket_tags）已完整，缺的是另一套模型的管理面。
	{
		key: "tag", maturity: MaturityPilot, built: true,
		pilotReason: "工单标签（ticket_tags）CRUD、工单绑定与 /admin/tags 页已完整；但组织标签（tags 表，挂项目/应用/微服务/部门/团队）只由 seeder 写入，没有任何管理面，两套标签模型并存。晋升条件：组织标签管理面或与工单标签收敛。",
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
		pilotReason: "SSO/OIDC 端点与服务齐备，但没有与真实 IdP 的端到端联调证据（现有覆盖为服务单测与路由测试）；租户未启用 IdP 时 health 会如实降级。晋升条件：至少一家真实 IdP 的登录/回调/换票验收。",
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
				if admin && !item.readOnly && deploymentReady {
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
