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
	{key: "serviceRequest", maturity: MaturityPilot, built: true, dependencies: []string{"workflow", "notification"},
		pilotReason: "交付 Provider 仍是 StubProvider，不会真正创建云资源；多步交付中途失败后的资源回滚/补偿未实现；目录项到 CI Type、交付模板、BPMN、SLA 的绑定口径仍分散。晋升条件：接真实 provider + 补偿动作 + 目录项绑定统一。"},
	{key: "cmdb", maturity: MaturityGA, built: true},
	{key: "cmdb.configurationItems", maturity: MaturityGA, built: true, dependencies: []string{"cmdb"}},
	{key: "cmdb.ciTypes", maturity: MaturityGA, built: true, dependencies: []string{"cmdb"}},
	{key: "cmdb.relationships", maturity: MaturityGA, built: true, dependencies: []string{"cmdb"}},
	{key: "cmdb.topology", maturity: MaturityGA, built: true, dependencies: []string{"cmdb"}},
	// 云发现缺的不是接口而是执行者：ClaimDiscoveryJob 只在测试里被调用，队列没有消费者，
	// 任务建完不会有人领取。
	{key: "cmdbDiscovery", maturity: MaturityPilot, built: true, dependencies: []string{"cmdb"}, readinessEnv: "ITSM_CMDB_DISCOVERY_READY",
		pilotReason: "发现任务没有执行者：ClaimDiscoveryJob 仅被单测调用，队列无消费者，任务创建后不会被领取执行；真实云凭据下的全量/增量发现未联调。晋升条件：worker 消费 + 真实账号发现验收。"},
	{key: "cmdbReconciliation", maturity: MaturityPilot, built: true, dependencies: []string{"cmdb", "cmdbDiscovery"}, readinessEnv: "ITSM_CMDB_DISCOVERY_READY",
		pilotReason: "对账依赖发现任务产出的结果集，而发现链路没有消费者，对账只能对空集；差异合并策略缺少真实数据验收。晋升条件：先打通 cmdbDiscovery 执行链。"},
	{key: "workflow", maturity: MaturityGA, built: true},
	{key: "sla", maturity: MaturityGA, built: true},
	{key: "knowledge", maturity: MaturityPilot, built: true,
		pilotReason: "发布版本语义未收敛（历史版本可对比/回滚，但「线上是哪个版本」没有唯一事实源）；分类可见性的跨角色检索未做端到端验收；召回率（典型问题集）缺人工验收口径。晋升条件：版本发布语义 + 跨角色检索验收。"},
	{key: "ai", maturity: MaturityPilot, built: true, readinessEnv: "ITSM_AI_READY",
		pilotReason: "confidence、接受/拒绝反馈、prompt 与 model 版本未在每个 AI 能力上强制写入审计；evaluator 未接入调用链；高风险动作仍只能人工确认。晋升条件：审计字段强制化 + evaluator 接线。"},
	// 通知域 2026-10-20/21 闭环三件事：投递审计读面（GET /notification-deliveries）、
	// 失败重试（保留旧记录作失败证据）、渠道诊断（GET /notifications/channels：
	// 渠道是否装配 / 是否健康 / 用户目标是否就绪）。加上既有的偏好过滤、
	// 幂等键与 outbox 重试，「没收到通知」的三类根因都能在一次请求里定位。
	{key: "notification", maturity: MaturityGA, built: true},
	// 三家连接器代码同层级（Send/HealthCheck/VerifySignature/ParseInbound + 回调路由 +
	// 审计注入），维持 Pilot 的原因是「没有真实租户联调证据」，不是实现缺失。
	{key: "connector.feishu", maturity: MaturityPilot, built: true, dependencies: []string{"notification"}, readinessEnv: "ITSM_FEISHU_READY",
		pilotReason: "回调验签只有单测，未与真实飞书租户做生产验收；死信人工重放与市场安装闭环未验收。晋升条件：真实租户收发 + 验签 + 死信重放验收。"},
	{key: "marketplace", maturity: MaturityPilot, built: true, readOnly: true,
		pilotReason: "目录项没有管理面：只有 seeder 能写入，管理员无法上架/下架/改版本；且 marketplace:write 已授予租户 Admin，直接开放全局目录写会造成跨租户污染，需先引入平台级权限码。晋升条件：平台级写权限 + 目录项 CRUD。"},
	{key: "project", maturity: MaturityGA, built: true},
	{key: "application", maturity: MaturityGA, built: true},
	// 工单标签（ticket_tags）已完整，缺的是另一套模型的管理面。
	{key: "tag", maturity: MaturityPilot, built: true,
		pilotReason: "工单标签（ticket_tags）CRUD、工单绑定与 /admin/tags 页已完整；但组织标签（tags 表，挂项目/应用/微服务/部门/团队）只由 seeder 写入，没有任何管理面，两套标签模型并存。晋升条件：组织标签管理面或与工单标签收敛。"},
	// 2026-10-22 晋升 GA：模板的真实表面是 /api/v1/tickets/templates 的 8 个端点
	// （列表/分类/创建/详情/更新/状态/复制/删除，权限码 ticket_template:*），
	// 前端 /tickets/templates 页（903 行）+ handler 测试齐备，菜单基线也已同码。
	// 旧注释引用的「template-api.ts 调用通用 /api/v1/templates 从未注册」是幽灵缺陷：
	// 前端仓库里不存在 template-api.ts，也不存在任何 /api/v1/templates 调用。
	{key: "template", maturity: MaturityGA, built: true},
	// SSO/OIDC 已完整实现（router/sso_routes.go 的 11 个端点 + handlers/auth/sso_*.go），
	// 此前误标为 disabled/built:false，会让按矩阵门控的界面隐藏一个可用能力。
	// 是否真正可用取决于租户有没有启用身份提供方，由 HealthFor 如实报告。
	{key: "identity.oidc", maturity: MaturityPilot, built: true,
		pilotReason: "SSO/OIDC 端点与服务齐备，但没有与真实 IdP 的端到端联调证据（现有覆盖为服务单测与路由测试）；租户未启用 IdP 时 health 会如实降级。晋升条件：至少一家真实 IdP 的登录/回调/换票验收。"},
	// 企微/钉钉连接器（2026-10-21 Disabled→Pilot 转正）：Send/HealthCheck/VerifySignature/
	// ParseInbound 四项核心与飞书同层级实现（connector/builtin/{wecom,dingtalk}，编译期
	// 接口断言），回调路由与审计注入已在 bootstrap 装配（app.go Connector 段 → RouterConfig
	// → RegisterRoutes）。渠道诊断（GET /notifications/channels）对两渠道返回真实健康信号。
	// 与 feishu 同款 readinessEnv 门控：部署是否就绪由 ITSM_WECOM_READY / ITSM_DINGTALK_READY
	// 如实申报，代码能力不再按「未包含在商业发布」隐藏。
	{key: "connector.wecom", maturity: MaturityPilot, built: true, dependencies: []string{"notification"}, readinessEnv: "ITSM_WECOM_READY",
		pilotReason: "回调验签与消息收发未与真实企业微信租户联调；OAuth 授权、任务双向同步等飞书级深度集成未做。晋升条件：真实租户收发 + 验签验收。"},
	{key: "connector.dingtalk", maturity: MaturityPilot, built: true, dependencies: []string{"notification"}, readinessEnv: "ITSM_DINGTALK_READY",
		pilotReason: "回调验签与消息收发未与真实钉钉租户联调；工作通知/待办同步等深度集成未做。晋升条件：真实租户收发 + 验签验收。"},
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
