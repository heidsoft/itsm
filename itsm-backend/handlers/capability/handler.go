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
}

var registry = []definition{
	{key: "ticket", maturity: MaturityGA, built: true},
	{key: "incident", maturity: MaturityGA, built: true, dependencies: []string{"workflow", "sla", "notification"}},
	// 问题域 2026-10-21 修掉两处生产阻断：
	//   1) 关闭证据校验——无解决方案或无根因结论一律 409（此前 investigating→resolved→closed
	//      可以全程不填分析结论，KEDB 沉淀的是不可复用的空壳记录）；
	//   2) KEDB 关联落库——前端「转为已知错误」传的 problemId 此前未在 DTO 声明，
	//      被 json 解码静默丢弃，已知错误与来源问题彻底脱钩。
	// 但仍不申报 GA：GET /problems/:id/sla 与 /comments 仍显式返回 503（问题没有 SLA
	// 截止时间字段，也没有写入 sla_states 的路径）。SLA 接入后才具备晋升条件。
	{key: "problem", maturity: MaturityPilot, built: true},
	{key: "change", maturity: MaturityGA, built: true, dependencies: []string{"workflow"}},
	{key: "serviceRequest", maturity: MaturityPilot, built: true, dependencies: []string{"workflow", "notification"}},
	{key: "cmdb", maturity: MaturityGA, built: true},
	{key: "cmdb.configurationItems", maturity: MaturityGA, built: true, dependencies: []string{"cmdb"}},
	{key: "cmdb.ciTypes", maturity: MaturityGA, built: true, dependencies: []string{"cmdb"}},
	{key: "cmdb.relationships", maturity: MaturityGA, built: true, dependencies: []string{"cmdb"}},
	{key: "cmdb.topology", maturity: MaturityGA, built: true, dependencies: []string{"cmdb"}},
	{key: "cmdbDiscovery", maturity: MaturityPilot, built: true, dependencies: []string{"cmdb"}, readinessEnv: "ITSM_CMDB_DISCOVERY_READY"},
	{key: "cmdbReconciliation", maturity: MaturityPilot, built: true, dependencies: []string{"cmdb", "cmdbDiscovery"}, readinessEnv: "ITSM_CMDB_DISCOVERY_READY"},
	{key: "workflow", maturity: MaturityGA, built: true},
	{key: "sla", maturity: MaturityGA, built: true},
	{key: "knowledge", maturity: MaturityPilot, built: true},
	{key: "ai", maturity: MaturityPilot, built: true, readinessEnv: "ITSM_AI_READY"},
	// 通知域 2026-10-20/21 闭环三件事：投递审计读面（GET /notification-deliveries）、
	// 失败重试（保留旧记录作失败证据）、渠道诊断（GET /notifications/channels：
	// 渠道是否装配 / 是否健康 / 用户目标是否就绪）。加上既有的偏好过滤、
	// 幂等键与 outbox 重试，「没收到通知」的三类根因都能在一次请求里定位。
	{key: "notification", maturity: MaturityGA, built: true},
	{key: "connector.feishu", maturity: MaturityPilot, built: true, dependencies: []string{"notification"}, readinessEnv: "ITSM_FEISHU_READY"},
	{key: "marketplace", maturity: MaturityPilot, built: true, readOnly: true},
	{key: "project", maturity: MaturityGA, built: true},
	{key: "application", maturity: MaturityGA, built: true},
	// 标签双表（ticket_tags 与 tags.code）尚未收敛到单一事实源，不申报 GA。
	{key: "tag", maturity: MaturityPilot, built: true},
	// 只有 /api/v1/tickets/templates* 有真实路由；template-api.ts 调用的通用
	// /api/v1/templates 市场后端从未注册。
	{key: "template", maturity: MaturityPilot, built: true},
	// SSO/OIDC 已完整实现（router/sso_routes.go 的 11 个端点 + handlers/auth/sso_*.go），
	// 此前误标为 disabled/built:false，会让按矩阵门控的界面隐藏一个可用能力。
	// 是否真正可用取决于租户有没有启用身份提供方，由 HealthFor 如实报告。
	{key: "identity.oidc", maturity: MaturityPilot, built: true},
	// 企微/钉钉连接器（2026-10-21 Disabled→Pilot 转正）：Send/HealthCheck/VerifySignature/
	// ParseInbound 四项核心与飞书同层级实现（connector/builtin/{wecom,dingtalk}，编译期
	// 接口断言），回调路由与审计注入已在 bootstrap 装配（app.go Connector 段 → RouterConfig
	// → RegisterRoutes）。渠道诊断（GET /notifications/channels）对两渠道返回真实健康信号。
	// 与 feishu 同款 readinessEnv 门控：部署是否就绪由 ITSM_WECOM_READY / ITSM_DINGTALK_READY
	// 如实申报，代码能力不再按「未包含在商业发布」隐藏。
	{key: "connector.wecom", maturity: MaturityPilot, built: true, dependencies: []string{"notification"}, readinessEnv: "ITSM_WECOM_READY"},
	{key: "connector.dingtalk", maturity: MaturityPilot, built: true, dependencies: []string{"notification"}, readinessEnv: "ITSM_DINGTALK_READY"},
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
