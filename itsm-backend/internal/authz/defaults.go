package authz

import (
	"strings"

	domainrole "itsm-backend/domain/role"
)

// RolePermissionDefaults 返回「角色 → 权限码（resource:action）」的编译期默认权限集。
//
// 这里是权限兜底的唯一来源，DB 播种也派生自同一张表；再维护一份独立的硬编码兜底表
// 就会形成「播种权威 vs 兜底权威」双轨漂移。派生规则：
//   - 内置词表角色直接派生自 BuiltinRolePermissionCodes()，兜底与播种默认一致；
//   - super_admin 保留 *:* 通配（与 Login ["*"] 旁路口径一致）；
//   - sysadmin 不用 *:* 字面通配，改用 allPermissionCodes() 枚举口径
//     （与 configured 态 DB 实际授权一致）；
//   - msp_* 已并入 BuiltinRolePermissionCodes()，因此自动派生、不再单独列条目
//     （此前只在兜底表里有、DB 播种没有，DBOnly 态下 MSP 角色是空集）。
//
// 词表外的角色（含已退役的 legacy security，存量数据由迁移改值）不在本表。
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
