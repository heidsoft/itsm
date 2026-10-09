package authz

import (
	"strings"

	domainrole "itsm-backend/domain/role"
)

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
//   - msp_* 为 GetMSPRBACRole 映射的运行时角色，在此显式列出默认集。
//
// 2026-10-07 角色模型重设：legacy users.role="security" 条目已随该值退役删除
// （存量数据由迁移改值到 security_admin），权限集不再包含词表外的角色。
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
	defaults[domainrole.SuperAdmin] = []string{"*:*"}

	// MSP 场景（middleware.GetMSPRBACRole 的映射目标角色）。
	defaults[domainrole.MspViewer] = []string{
		"msp:read", "msp_customer:read", "msp_ticket:read",
		"msp_allocation:read", "msp_report:read",
	}
	defaults[domainrole.MspTech] = []string{
		"msp:read", "msp_customer:read", "msp_ticket:read", "msp_ticket:write",
		"msp_allocation:read", "msp_report:read",
	}
	defaults[domainrole.MspSpecialist] = []string{
		"msp:read", "msp_customer:read", "msp_customer:write",
		"msp_ticket:read", "msp_ticket:write",
		"msp_allocation:read", "msp_report:read",
	}
	defaults[domainrole.MspManager] = []string{
		"msp:read", "msp:write", "msp_customer:read", "msp_customer:write",
		"msp_ticket:read", "msp_ticket:write",
		"msp_allocation:read", "msp_allocation:write",
		"msp_report:read", "msp_report:write",
	}
	defaults[domainrole.MspAdmin] = []string{
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
