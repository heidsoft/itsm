package seeder

import (
	"testing"

	domainrole "itsm-backend/domain/role"
	"itsm-backend/middleware"
)

// TestBuiltinRolePermissionCodes_Guard 内置词表角色 DB 权限码集守卫。
//
// 背景（2026-09-17 P0）：BuiltinRoles() 把 users.role 词表角色写入 roles 表
// （DBOnly 三态判定为 configured），但 builtinRolePermissionCodes() 此前缺
// admin/technician 条目 → roles 表权限行空集 = DB 显式撤销 →
// admin/technician 用户除未挂权限中间件的路由外全部 403。
//
// 本守卫锁三条契约，缺一 CI 红：
//  1. 词表角色（除 super_admin 走 Login ["*"] 旁路）必须有非空 DB 权限码集；
//  2. 码集引用的每个码必须在 permissionDefinitions() 清单中（防码空间漂移）；
//  3. admin/technician 的码集必须覆盖 middleware.RolePermissions 硬编码
//     兜底的每一个 (resource,action) 对（防双权威表漂移——本 P0 的根因类别）。
func TestBuiltinRolePermissionCodes_Guard(t *testing.T) {
	roleCodes := builtinRolePermissionCodes()
	defs := permissionDefinitions()

	defByCode := make(map[string]permissionDef, len(defs))
	defByPair := make(map[string]bool, len(defs)) // "resource:action"
	for _, d := range defs {
		defByCode[d.Code] = d
		defByPair[d.Resource+":"+d.Action] = true
	}

	for _, role := range domainrole.All {
		if role == domainrole.SuperAdmin {
			continue // Login ["*"] 旁路，不依赖 DB 权限行
		}

		set, ok := roleCodes[role]
		if !ok || len(set) == 0 {
			t.Errorf("内置角色 %q 在 builtinRolePermissionCodes() 中缺失或为空集：DBOnly configured 态下空集=显式撤销，该角色用户将全 403", role)
			continue
		}

		// 契约 2：码集必须落在权限定义清单内（码空间单一源）
		grantedPairs := make(map[string]bool, len(set))
		for _, code := range set {
			def, exists := defByCode[code]
			if !exists {
				t.Errorf("角色 %q 引用的权限码 %q 不在 permissionDefinitions() 清单中：seedPermissions 不会创建该码，role_permissions 将静默缺失", role, code)
				continue
			}
			grantedPairs[def.Resource+":"+def.Action] = true
		}

		// 契约 3：与硬编码兜底全等覆盖（本 P0 根因类别）。
		// 范围说明：admin/technician 的 DB 码集定义为硬编码兜底的完整镜像（本 P0 补齐的对象）。
		// end_user/agent/manager 等角色的 DB 码集是 2026-09-07 有意裁剪的子集（小于硬编码
		// 兜底），属已知且接受的双表差异，不做全等校验——若未来要统一，需产品决策 + 全链路
		// 回归，不应在本守卫中静默扩大权限。
		parityRoles := map[string]bool{
			domainrole.Admin:      true,
			domainrole.Technician: true,
		}
		if !parityRoles[role] {
			continue
		}
		hardcoded, hasHardcoded := middleware.RolePermissions[role]
		if !hasHardcoded || len(hardcoded) == 0 {
			continue
		}
		for _, p := range hardcoded {
			pair := p.Resource + ":" + p.Action
			if !grantedPairs[pair] {
				t.Errorf("角色 %q 的 DB 码集缺少硬编码兜底中的 (%s, %s)：unconfigured 兜底与 configured 实际授权不一致", role, p.Resource, p.Action)
			}
		}
	}
}

// TestPermissionDefinitions_NoDuplicateCodes 权限定义清单内 code 不得重复。
func TestPermissionDefinitions_NoDuplicateCodes(t *testing.T) {
	seen := make(map[string]bool)
	for _, d := range permissionDefinitions() {
		if seen[d.Code] {
			t.Errorf("权限码 %q 在 permissionDefinitions() 中重复定义", d.Code)
			continue
		}
		seen[d.Code] = true
	}
}
