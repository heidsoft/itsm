package authz

import "strings"

// RolePermissionDefaults 返回「角色 → 权限码（resource:action）」的编译期默认权限集。
//
// 2026-10-05 N5 拍板（plans/product-remediation-plan-2026-10-03.md §3）：
// 取代此前 middleware.RolePermissions 独立硬编码表，消除「DB 播种权威 vs 兜底权威」
// 双权威漂移。派生规则：
//   - 内置词表角色直接派生自 BuiltinRolePermissionCodes()（单一真源），
//     兜底与播种默认一致，漂移不可能再发生；
//   - super_admin 保留 *:* 通配（与 Login ["*"] 旁路口径一致）；
//   - sysadmin 不再用 *:* 字面通配，统一为 allPermissionCodes() 枚举口径
//     （与 configured 态 DB 实际授权一致）；
//   - security 为 users.role 存量 legacy 值（ent/schema/user.go 标注"新代码禁用"），
//     msp_* 为 GetMSPRBACRole 映射的运行时角色，均不入库播种，在此显式列出默认集。
//
// 返回码集为只读视图，调用方不得修改。
func RolePermissionDefaults() map[string][]string {
	return rolePermissionDefaults
}

var rolePermissionDefaults = func() map[string][]string {
	defaults := make(map[string][]string, len(builtinRoleBindingKeys)+8)
	for role, codes := range BuiltinRolePermissionCodes() {
		defaults[role] = codes
	}

	// 平台超管：通配旁路（DB 播种侧不依赖此条目）。
	defaults["super_admin"] = []string{"*:*"}

	// legacy users.role="security" 存量值（原 middleware.RolePermissions["security"]）。
	defaults["security"] = []string{
		"user:read", "knowledge:read", "knowledge:list",
		"notification:read", "notification:list", "notification:write",
		"ticket:read", "ticket:list", "incident:read", "incident:list",
		"problem:read", "problem:list", "change:read", "change:list",
		"approval:read", "approval:write",
		"service_catalog:read", "service_request:read", "service_request:write",
		"bpmn:read", "task:read", "task:update",
		"release:read", "asset:read", "license:read", "dashboard:read",
		"cmdb:read", "sla:read",
	}

	// MSP 场景（middleware.GetMSPRBACRole 的映射目标角色）。
	defaults["msp_viewer"] = []string{
		"msp:read", "msp_customer:read", "msp_ticket:read",
		"msp_allocation:read", "msp_report:read",
	}
	defaults["msp_tech"] = []string{
		"msp:read", "msp_customer:read", "msp_ticket:read", "msp_ticket:write",
		"msp_allocation:read", "msp_report:read",
	}
	defaults["msp_specialist"] = []string{
		"msp:read", "msp_customer:read", "msp_customer:write",
		"msp_ticket:read", "msp_ticket:write",
		"msp_allocation:read", "msp_report:read",
	}
	defaults["msp_manager"] = []string{
		"msp:read", "msp:write", "msp_customer:read", "msp_customer:write",
		"msp_ticket:read", "msp_ticket:write",
		"msp_allocation:read", "msp_allocation:write",
		"msp_report:read", "msp_report:write",
	}
	defaults["msp_admin"] = []string{
		"msp:*", "msp_customer:*", "msp_ticket:*", "msp_allocation:*", "msp_report:*",
	}

	return defaults
}()

// builtinRoleBindingKeys 仅用于预分配 map 容量。
var builtinRoleBindingKeys = func() map[string]struct{} {
	m := make(map[string]struct{})
	for role := range BuiltinRolePermissionCodes() {
		m[role] = struct{}{}
	}
	return m
}()

// SplitPermissionCode 将 "resource:action" 拆为 resource 与 action。
func SplitPermissionCode(code string) (resource, action string, ok bool) {
	resource, action, ok = strings.Cut(code, ":")
	if !ok || resource == "" || action == "" {
		return "", "", false
	}
	return resource, action, true
}
