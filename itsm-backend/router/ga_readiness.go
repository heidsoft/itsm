package router

import (
	"context"
	"time"

	"itsm-backend/connector"
	"itsm-backend/ent"
)

type gaReadinessModule struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Endpoint  string `json:"endpoint"`
	DataCount int    `json:"dataCount"`
}

type gaReadinessCheck struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Notes  string `json:"notes,omitempty"`
}

type gaReadinessResponse struct {
	Version     string              `json:"version"`
	Target      string              `json:"target"`
	Status      string              `json:"status"`
	GeneratedAt time.Time           `json:"generatedAt"`
	Modules     []gaReadinessModule `json:"modules"`
	Checks      []gaReadinessCheck  `json:"checks"`
	Summary     map[string]int      `json:"summary"`
}

func buildGAReadiness(ctx context.Context, client *ent.Client) gaReadinessResponse {
	modules := []gaReadinessModule{
		{Key: "tenant_model", Name: "租户模型", Status: "ready", Endpoint: "/api/v1/tenants", DataCount: countOrZero(ctx, client, "tenants")},
		{Key: "permission_menu", Name: "权限与菜单", Status: "ready", Endpoint: "/api/v1/auth/menus", DataCount: countOrZero(ctx, client, "permissions") + countOrZero(ctx, client, "menus")},
		{Key: "roles", Name: "角色模板", Status: "ready", Endpoint: "/api/v1/roles", DataCount: countOrZero(ctx, client, "roles")},
		{Key: "service_catalog_templates", Name: "服务目录模板", Status: "ready", Endpoint: "/api/v1/service-catalogs", DataCount: countOrZero(ctx, client, "service_catalogs")},
		{Key: "sla_templates", Name: "SLA 模板", Status: "ready", Endpoint: "/api/v1/sla/definitions", DataCount: countOrZero(ctx, client, "sla_definitions")},
		{Key: "approval_templates", Name: "审批流模板", Status: "ready", Endpoint: "/api/v1/approval-workflows", DataCount: countOrZero(ctx, client, "approval_workflows")},
		{Key: "process_bindings", Name: "流程绑定", Status: "ready", Endpoint: "/api/v1/process-bindings", DataCount: countOrZero(ctx, client, "process_bindings")},
		{Key: "cmdb_types", Name: "CMDB 类型模板", Status: "ready", Endpoint: "/api/v1/configuration-items/types", DataCount: countOrZero(ctx, client, "ci_types")},
		{Key: "standard_change_templates", Name: "标准变更模板", Status: "ready", Endpoint: "/api/v1/standard-changes", DataCount: countOrZero(ctx, client, "standard_changes")},
		{Key: "connector_market", Name: "连接器市场", Status: "ready", Endpoint: "/api/v1/connectors/lifecycle", DataCount: len(connector.Default().List())},
		{Key: "ai_native", Name: "AI Native 可追踪", Status: "ready", Endpoint: "/api/v1/ai/audit", DataCount: countOrZero(ctx, client, "prompt_templates")},
		{Key: "audit", Name: "审计日志", Status: "ready", Endpoint: "/api/v1/audit-logs", DataCount: countOrZero(ctx, client, "audit_logs")},
	}

	ready := 0
	for _, module := range modules {
		if module.Status == "ready" {
			ready++
		}
	}

	return gaReadinessResponse{
		Version:     "1.0.0",
		Target:      "open-source-v1.0-ga",
		Status:      "ga_candidate",
		GeneratedAt: time.Now(),
		Modules:     modules,
		Checks: []gaReadinessCheck{
			{Key: "api_response", Name: "统一 API 响应", Status: "ready", Notes: "业务 API 继续使用 {code,message,data}"},
			{Key: "tenant_isolation", Name: "租户隔离", Status: "ready", Notes: "AI 主入口缺失 tenant 时拒绝请求"},
			{Key: "connector_lifecycle", Name: "连接器生命周期", Status: "ready", Notes: "市场/配置/健康状态已统一为 lifecycle 视图"},
			{Key: "ai_traceability", Name: "AI 可追踪", Status: "ready", Notes: "AI audit contract includes scenario/input_ref/prompt_version/model/confidence/suggestion/accepted"},
			{Key: "product_seed", Name: "产品默认初始化", Status: "ready", Notes: "默认 seed 提供可配置功能模板，不预置虚构业务记录"},
		},
		Summary: map[string]int{
			"modules_total": len(modules),
			"modules_ready": ready,
		},
	}
}

func countOrZero(ctx context.Context, client *ent.Client, key string) int {
	if client == nil {
		return 0
	}
	var (
		count int
		err   error
	)
	switch key {
	case "tenants":
		count, err = client.Tenant.Query().Count(ctx)
	case "permissions":
		count, err = client.Permission.Query().Count(ctx)
	case "menus":
		count, err = client.Menu.Query().Count(ctx)
	case "roles":
		count, err = client.Role.Query().Count(ctx)
	case "service_catalogs":
		count, err = client.ServiceCatalog.Query().Count(ctx)
	case "sla_definitions":
		count, err = client.SLADefinition.Query().Count(ctx)
	case "approval_workflows":
		count, err = client.ApprovalWorkflow.Query().Count(ctx)
	case "process_bindings":
		count, err = client.ProcessBinding.Query().Count(ctx)
	case "ci_types":
		count, err = client.CIType.Query().Count(ctx)
	case "standard_changes":
		count, err = client.StandardChange.Query().Count(ctx)
	case "prompt_templates":
		count, err = client.PromptTemplate.Query().Count(ctx)
	case "audit_logs":
		count, err = client.AuditLog.Query().Count(ctx)
	default:
		return 0
	}
	if err != nil {
		return 0
	}
	return count
}
