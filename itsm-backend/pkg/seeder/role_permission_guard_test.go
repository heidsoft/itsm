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
)

// TestBuiltinRolePermissionCodes_Guard 内置词表角色 DB 权限码集守卫。
//
// 背景（2026-09-17 P0）：BuiltinRoles() 把 users.role 词表角色写入 roles 表
// （DBOnly 三态判定为 configured），但 builtinRolePermissionCodes() 此前缺
// admin/technician 条目 → roles 表权限行空集 = DB 显式撤销 →
// admin/technician 用户除未挂权限中间件的路由外全部 403。
//
// 2026-10-05 N4/N5 拍板（plans/product-remediation-plan-2026-10-03.md §3）：
// middleware.RolePermissions 独立硬编码表已删除，编译期兜底默认权限改由
// authz.RolePermissionDefaults() 单一真源派生（内置角色直接派生自播种码集，
// 结构性不可能漂移；R2-a 待拍板档 t.Skip 同批退役）。原契约 3 替换为：
//  1. 词表角色（除 super_admin 走 Login ["*"] 旁路）必须有非空 DB 权限码集；
//  2. 码集引用的每个码必须在 permissionDefinitions() 清单中（防码空间漂移）；
//  3. N4 扩权回归锁：manager/agent/end_user 必须持有拍板补齐的码（防静默回收）；
//  4. 兜底覆盖锁：词表角色与仅运行时的 msp_* 角色都必须有非空编译期默认权限，
//     且与 DB 码集一致（词表角色）；
//  5. 封闭词表锁：domain/role.Retired 中的退役角色码不得出现在任何权限映射中。
func TestBuiltinRolePermissionCodes_Guard(t *testing.T) {
	roleCodes := builtinRolePermissionCodes()
	defs := permissionDefinitions()

	defByCode := make(map[string]permissionDef, len(defs))
	for _, d := range defs {
		defByCode[d.Code] = d
	}

	// 契约 3（N4 扩权回归锁）：拍板补齐的码一旦消失即 CI 红。
	// manager 26 / agent 7 / end_user 16；agent 的 alerts:read 按拍板收敛为 alert:read
	// （路由预检实测零条 alerts:* 声明，见 authz/roles.go agent 条目注释）。
	n4Granted := map[string][]string{
		"manager": {
			"ticket:assign", "ticket:escalate", "ticket:export",
			"notification:read", "notification:write", "incident:write", "dashboard:read",
			"cmdb:read", "service_catalog:read", "sla:read", "bpmn:read",
			"release:read", "release:write", "asset:read", "asset:write",
			"license:read", "license:write", "group:read", "group:write",
			"org:read", "org:write", "project:read", "project:write",
			"application:read", "application:write", "ai:read",
		},
		"agent": {
			"notification:write", "dashboard:read", "service_catalog:read",
			"change:write", "alert:read", "group:read", "bpmn:read",
		},
		"end_user": {
			"notification:write", "dashboard:read", "ai:read", "ai:write",
			"sla:read", "system_config:read", "org:read", "department:read",
			"cmdb:read", "incident:read", "change:read", "problem:read",
			"bpmn:read", "release:read", "asset:read", "license:read",
		},
	}
	for role, codes := range n4Granted {
		granted := make(map[string]bool, len(roleCodes[role]))
		for _, c := range roleCodes[role] {
			granted[c] = true
		}
		for _, c := range codes {
			if !granted[c] {
				t.Errorf("N4 扩权回归：角色 %q 缺少拍板补齐的权限码 %q", role, c)
			}
		}
	}

	defaults := authz.RolePermissionDefaults()

	// 契约 4 前置：仅存在于运行时、不进 roles 表的 MSP 协作角色必须有显式默认集。
	// super_admin 走 Login ["*"] 旁路，但兜底默认仍必须非空。
	nonSeededRoles := []string{"super_admin", "msp_viewer", "msp_tech", "msp_specialist", "msp_manager", "msp_admin"}
	for _, role := range nonSeededRoles {
		if len(defaults[role]) == 0 {
			t.Errorf("角色 %q 在 authz.RolePermissionDefaults() 中缺失或为空：删除 middleware.RolePermissions 后该角色 unconfigured 兜底将全 403", role)
		}
	}

	// 契约 5（2026-10-07 角色词表重设）：退役角色码不得残留在任何权限映射里。
	// 词表是封闭的，残留条目就是双权威回潮——它永远不会被 seedRolePermissions 命中，
	// 却会让「按角色查权限」的读者以为该角色仍然有效。
	for _, retired := range domainrole.Retired {
		if codes, ok := defaults[retired]; ok {
			t.Errorf("退役角色 %q 仍在 authz.RolePermissionDefaults() 中（%d 码）：请删除条目", retired, len(codes))
		}
		if codes, ok := roleCodes[retired]; ok {
			t.Errorf("退役角色 %q 仍在 builtinRolePermissionCodes() 中（%d 码）：请删除条目", retired, len(codes))
		}
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

		// 契约 4：词表角色的编译期兜底默认必须与 DB 码集一致（单一真源锁）。
		// 派生上 defaults[role] = BuiltinRolePermissionCodes()[role]，
		// 一旦有人重新引入第二张表或旁路派生，本循环立即红。
		defaultPairs := make(map[string]bool, len(defaults[role]))
		for _, code := range defaults[role] {
			if _, _, valid := authz.SplitPermissionCode(code); valid {
				defaultPairs[code] = true
			}
		}
		if len(defaultPairs) == 0 {
			t.Errorf("角色 %q 在 authz.RolePermissionDefaults() 中缺失：unconfigured 兜底为空将全 403", role)
		}
		for pair := range defaultPairs {
			if !grantedPairs[pair] {
				// 派生规则（task 基线/同义奇偶）可能让 DB 码集多于绑定原始码，反向包含才成立：
				// DB 码集 ⊇ defaults 码集。defaults ⊄ DB 即双权威回潮。
				t.Errorf("角色 %q 兜底默认 %s 不在 DB 码集中：单一真源被破坏（双权威回潮）", role, pair)
			}
		}
	}
}

// TestBuiltinRolePermissionCodes_NoDeadKeys 权限码映射的键必须 ⊆ 实际播种的角色 code。
//
// 背景（2026-10-03 R7-6；2026-10-07 词表重设后重写）：BuiltinRolePermissionCodes()
// 的键必须对应实际会被播种到 roles 表的角色。播种来源有三：
//  1. BuiltinRoles()——users.role 主角色词表（domainrole.All）；
//  2. PracticeRoles()——ITIL 实践角色与 MSP 协作角色（只经 user_roles 边叠加）；
//  3. 种子配置 config/seed/default.json 的 roles 清单——租户级追加，内置码不再复制。
//
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
	for _, r := range PracticeRoles() {
		seededRoles[r.Code] = true
	}
	cfg := loadSeedConfig(zap.NewNop().Sugar())
	for _, r := range cfg.Roles {
		seededRoles[r.Code] = true
	}

	// loadSeedConfig 在测试上下文中无法通过 resolveSeedConfigFile 找到 JSON 配置
	// （测试二进制运行在临时目录），因此需要直接读取项目中的 JSON 配置文件。
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
			t.Errorf("BuiltinRolePermissionCodes() 包含死键 %q：该角色不在 BuiltinRoles/PracticeRoles 或默认种子配置中，seedRolePermissions 永远不会匹配到它", role)
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
