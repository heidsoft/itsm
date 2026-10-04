package seeder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"go.uber.org/zap"

	domainrole "itsm-backend/domain/role"
	"itsm-backend/internal/authz"
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
//  3. 与 middleware.RolePermissions 硬编码兜底同名的词表角色，其 DB 码集必须覆盖
//     兜底的每一个 (resource,action) 对（防双权威表漂移——本 P0 的根因类别）。
//     覆盖范围分硬档（admin/technician，差异即红）与待拍板档（差异以子测试
//     t.Skip 登记，见 plans/product-remediation-plan-2026-10-03.md §3 N4）。
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

		// 契约 3：与硬编码兜底覆盖校验（本 P0 根因类别）。
		// R2-a（2026-10-03）：范围从手工枚举 {admin, technician} 扩大为
		// 「词表角色 ∩ middleware.RolePermissions 同名键」——不再手工维护角色清单，
		// middleware 未来新增与词表同名的角色会自动纳入守卫。
		// 分两档（plans/product-remediation-plan-2026-10-03.md §2 R2-a / §3 N4）：
		//   硬档 admin/technician：DB 码集定义为硬编码兜底的完整镜像（本 P0 补齐对象），差异即 CI 红；
		//   待拍板档（manager/agent/sysadmin/end_user）：DB 码集是 2026-09-07 有意裁剪的子集
		//   （小于硬编码兜底），扩大范围后守卫实测红出差异清单见 §3 N4，是否收敛待 N4 拍板，
		//   拍板前以子测试 t.Skip 登记真实差异，不静默留红 CI、也不静默吞掉差异。
		// 注意：middleware 词表另有 security / msp_viewer 两个键与 domain 词表不同名，
		// 不在本守卫覆盖内；同名缺口（it_admin、security_admin 无 middleware 条目）同样不受影响。
		parityHardRoles := map[string]bool{
			domainrole.Admin:      true,
			domainrole.Technician: true,
		}
		hardcoded, hasHardcoded := middleware.RolePermissions[role]
		if !hasHardcoded || len(hardcoded) == 0 {
			continue
		}
		missingPairs := []string{}
		for _, p := range hardcoded {
			pair := p.Resource + ":" + p.Action
			if !grantedPairs[pair] {
				missingPairs = append(missingPairs, pair)
			}
		}
		if len(missingPairs) == 0 {
			continue
		}
		if parityHardRoles[role] {
			for _, pair := range missingPairs {
				t.Errorf("角色 %q 的 DB 码集缺少硬编码兜底中的 %s：unconfigured 兜底与 configured 实际授权不一致", role, pair)
			}
			continue
		}
		t.Run("parity/"+role, func(t *testing.T) {
			t.Skipf("DB 码集缺少硬编码兜底 %v；属 2026-09-07 有意裁剪的子集，是否收敛待 plans/product-remediation-plan-2026-10-03.md §3 N4 拍板", missingPairs)
		})
	}
}

// TestBuiltinRolePermissionCodes_NoDeadKeys 权限码映射的键必须 ⊆ 实际播种的角色 code。
//
// 背景（2026-10-03 R7-6）：BuiltinRolePermissionCodes() 的键必须对应实际会被播种到
// roles 表的角色。播种来源有二：1) BuiltinRoles() 返回的 domainrole.All 9 个核心角色；
// 2) 默认种子配置 config/seed/default.json 中的 roles 清单（18 个扩展角色）。
// 死键（不在任一来源中的键）永远不会被 seedRolePermissions 匹配，携带的权限码会
// 与真实角色漂移，增加审计与维护负担。
func TestBuiltinRolePermissionCodes_NoDeadKeys(t *testing.T) {
	seededRoles := make(map[string]bool)
	for _, r := range domainrole.All {
		seededRoles[r] = true
	}
	for _, r := range BuiltinRoles() {
		seededRoles[r.Code] = true
	}
	cfg := loadSeedConfig(zap.NewNop().Sugar())
	for _, r := range cfg.Roles {
		seededRoles[r.Code] = true
	}

	// loadSeedConfig 在测试上下文中无法通过 resolveSeedConfigFile 找到 JSON 配置
	// （测试二进制运行在临时目录），因此需要直接读取项目中的 JSON 配置文件，
	// 把仅存在于 JSON 中的角色（如 change_manager、service_catalog_admin）也纳入允许集。
	_, testFile, _, _ := runtime.Caller(0)
	jsonPath := filepath.Join(filepath.Dir(testFile), "..", "..", "config", "seed", "default.json")
	if data, err := os.ReadFile(jsonPath); err == nil {
		var jsonCfg SeedConfig
		if err := json.Unmarshal(data, &jsonCfg); err == nil {
			for _, r := range jsonCfg.Roles {
				seededRoles[r.Code] = true
			}
		}
	}

	for role := range authz.BuiltinRolePermissionCodes() {
		if role == domainrole.SuperAdmin {
			continue // Login ["*"] 旁路，不依赖 DB 权限行
		}
		if !seededRoles[role] {
			t.Errorf("BuiltinRolePermissionCodes() 包含死键 %q：该角色不在 domainrole.All 或默认种子配置中，seedRolePermissions 永远不会匹配到它", role)
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

// TestRetiredRolePermissionCodes_Guard 显式退役清单守卫（2026-10-03 R2-d）。
//
// 「只增不减」契约下 applyRetiredRolePermissions 是播种器唯一的收缩通道，
// 清单写错就会静默删掉线上授权行，故锁三条契约：
//  1. 键 ⊆ BuiltinRolePermissionCodes() 的键——退役只对内置角色有意义，
//     拼错角色名会让 delete 静默空转（无法靠运行时发现）；
//  2. 值 ⊆ 权限清单（Definitions()）——退役清单引用不存在或不在 seedPermissions
//     创建范围内的码，等于宣告一次永不生效的退役；
//  3. 值与该角色现行内置码集不得重叠——既授予又退役 = 配置矛盾，
//     播种顺序会让最终状态依赖实现细节。
func TestRetiredRolePermissionCodes_Guard(t *testing.T) {
	roleCodes := builtinRolePermissionCodes()
	defs := permissionDefinitions()

	defByCode := make(map[string]bool, len(defs))
	for _, d := range defs {
		defByCode[d.Code] = true
	}

	for role, retired := range authz.RetiredRolePermissionCodes() {
		set, ok := roleCodes[role]
		if !ok {
			t.Errorf("退役清单的键 %q 不在 builtinRolePermissionCodes() 中：applyRetiredRolePermissions 将静默空转", role)
			continue
		}
		builtin := make(map[string]bool, len(set))
		for _, code := range set {
			builtin[code] = true
		}
		for _, code := range retired {
			if !defByCode[code] {
				t.Errorf("角色 %q 退役的权限码 %q 不在 permissionDefinitions() 清单中：seedPermissions 不会创建该码，退役永不生效", role, code)
				continue
			}
			if builtin[code] {
				t.Errorf("角色 %q 的权限码 %q 同时出现在内置码集与退役清单中：既授予又退役，播种结果依赖顺序，属配置矛盾", role, code)
			}
		}
	}
}
