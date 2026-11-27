package mcp

import (
	"context"
	"errors"
	"os"
	"strings"

	"itsm-backend/dto"
	"itsm-backend/service/marketplace"
)

// 装配缺失错误。两者都是 fail-closed 语义：
// 缺执行器 → 调不动；缺授权目录 → 非 human 主体不得通行。
// ErrNoExecutor 是执行器未装配时的错误。
var ErrNoExecutor = errors.New("mcp: 执行器未装配")

// defaultVersion 是未注入 APP_VERSION 时的兜底版本。
//
// 与 router.defaultAppVersion 同口径：优先读 APP_VERSION 环境变量，
// 回落默认值。不复用 router 的 appVersion() 是因为它是未导出函数，
// 而 router 已 import 本包 —— 反向引用会形成 import 循环。
// 两处读取同一个环境变量，口径不会漂移。
const defaultVersion = "1.6.61"

// Version 返回本服务端对外声明的版本号。
func Version() string {
	if v := strings.TrimSpace(os.Getenv("APP_VERSION")); v != "" {
		return v
	}
	return defaultVersion
}

// 装配适配器：把既有领域能力接到本包的窄契约上。
//
// 这一层刻意做得薄 —— 每行都是一次「已有实现 → 窄契约」的转接，
// 不含任何业务判断。业务判断留在原实现里（ExecuteTool 的权限链、
// marketplace 的交付门禁、authz 的主体策略），避免第二套判定。

// ToolCatalogAdapter 把既有工具注册表接到 ToolCatalog 窄契约。
//
// 复用 internal/services 的 ToolRegistry.ListToolsForTenant ——
// 它已按租户裁剪 CMDB 类型枚举，无需另写租户过滤。
type ToolCatalogAdapter struct {
	// Lister 是工具列举能力，由 internal/services.ToolRegistry 实现。
	Lister interface {
		ListToolsForTenant(ctx context.Context, tenantID int) []dto.ToolDefinition
	}
}

// 编译期断言：适配器必须满足窄契约。
var _ ToolCatalog = (*ToolCatalogAdapter)(nil)

// ListToolsForTenant 转发到既有注册表。
func (a *ToolCatalogAdapter) ListToolsForTenant(ctx context.Context, tenantID int) []dto.ToolDefinition {
	if a == nil || a.Lister == nil {
		return nil
	}
	return a.Lister.ListToolsForTenant(ctx, tenantID)
}

// ToolExecutorAdapter 把 ai.Service.ExecuteTool 接到 ToolExecutor 窄契约。
//
// 复用既有执行链 —— 它内部已串好 Gate 2 角色 RBAC、Gate 2.5 主体零信任、
// 写工具审批流与审计落库。MCP 通道不新增执行路径，只新增协议入口。
type ToolExecutorAdapter struct {
	// Executor 是 AI 域的工具执行入口。
	Executor interface {
		ExecuteTool(ctx context.Context, userID, tenantID int, role, name string, args map[string]interface{}) (interface{}, int, error)
	}
}

var _ ToolExecutor = (*ToolExecutorAdapter)(nil)

// ExecuteTool 转发到既有执行链。
func (a *ToolExecutorAdapter) ExecuteTool(ctx context.Context, userID, tenantID int, role, name string, args map[string]interface{}) (interface{}, int, error) {
	if a == nil || a.Executor == nil {
		return nil, 0, ErrNoExecutor
	}
	return a.Executor.ExecuteTool(ctx, userID, tenantID, role, name, args)
}

// EntitlementAdapter 把 marketplace 的交付门禁接到 EntitlementChecker 窄契约.
//
// 复用 service/marketplace.Service.SkillInvocationEntitlement ——
// 「目录上架 + 租户安装才可用」的既有门禁。MCP 暴露的工具面受同一门禁约束，
// 否则 MCP 通道会成为绕过市场交付的旁路。
type EntitlementAdapter struct {
	// Checker 是市场侧的技能交付判定。
	Checker interface {
		SkillInvocationEntitlement(ctx context.Context, tenantID int, code string) (marketplace.SkillEntitlement, error)
	}
}

var _ EntitlementChecker = (*EntitlementAdapter)(nil)

// SkillInvocationEntitlement 转发到市场侧判定并投影成本包视图。
func (a *EntitlementAdapter) Check(ctx context.Context, tenantID int, code string) (Entitlement, error) {
	if a == nil || a.Checker == nil {
		// 未装配门禁 → 放行（由租户隔离与执行链兜底），但不提供按购买收敛工具面的能力。
		return Entitlement{Allowed: true, Reason: "entitlement checker not configured"}, nil
	}
	ent, err := a.Checker.SkillInvocationEntitlement(ctx, tenantID, code)
	if err != nil {
		return Entitlement{}, err
	}
	return Entitlement{Allowed: ent.Allowed, Reason: ent.Reason}, nil
}
