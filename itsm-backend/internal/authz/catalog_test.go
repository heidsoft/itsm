package authz

import (
	"testing"

	domainrole "itsm-backend/domain/role"
)

// TestDefinitionsParityWithSeeder 校验权限码目录的内部一致性：Code 必须等于
// Resource:Action，AllCodes 与 Definitions 同序同量，且码全局唯一。
// 目录本身已从 pkg/seeder 迁入本包，seeder 直接消费 Definitions()。
func TestDefinitionsParityWithSeeder(t *testing.T) {
	cat := AllCodes()
	if len(cat) < 100 {
		t.Fatalf("catalog 仅 %d 个码，疑似提取逻辑失效", len(cat))
	}

	// 字段切分一致性：Code 的 "{resource}:{action}" 必须与字段匹配
	for _, d := range Definitions() {
		want := d.Resource + ":" + d.Action
		if d.Code != want {
			t.Errorf("权限码 %q 的 Resource:Action 拆分与 Code 不一致：want=%q", d.Code, want)
		}
	}

	// 内部一致性：AllCodes 与 Definitions 同序（保持 Definitions 的源序，便于 codegen diff）
	defs := Definitions()
	if len(cat) != len(defs) {
		t.Fatalf("AllCodes(%d) 与 Definitions 长度(%d) 不一致", len(cat), len(defs))
	}
	for i, c := range cat {
		if c != defs[i].Code {
			t.Errorf("AllCodes[%d]=%q != defs[%d].Code=%q", i, c, i, defs[i].Code)
		}
	}

	// 全局唯一性（无重复码）
	seen := map[string]int{}
	for _, c := range cat {
		seen[c]++
	}
	for c, n := range seen {
		if n > 1 {
			t.Errorf("权限码 %q 在 Definitions 中重复 %d 次", c, n)
		}
	}

	_ = len(cat) // 排序由 codegen 阶段处理（保持源序便于 review）
}

// TestBuiltinRolesUseOnlyCatalogCodes 校验内置角色的权限码全部来自 catalog。
//
// 这是"权限码漂移"的兜底：若 BuiltinRolePermissionCodes() 引用了 catalog 里
// 不存在的码（拼写错误、码被重命名但角色清单没跟着改），
// checkPermissionMatch 是精确匹配，该路由会对除 super_admin 外的所有角色永久 403，
// 而没有任何机制会报错。
func TestBuiltinRolesUseOnlyCatalogCodes(t *testing.T) {
	cat := AllCodes()
	valid := make(map[string]bool, len(cat))
	for _, c := range cat {
		valid[c] = true
	}

	for role, codes := range BuiltinRolePermissionCodes() {
		for _, c := range codes {
			if !valid[c] {
				t.Errorf("角色 %s 引用了 catalog 中不存在的权限码 %q", role, c)
			}
		}
	}
}

// TestSysAdminGetsFullCatalog 校验 sysadmin 的播种码集等于 catalog 全量。
//
// 该测试锁定 2026-10-13 的修复：此前 allPermissionCodes() 是一份手写清单，
// 与 catalog 存在差集（实测 129 vs 160，缺 33 个含 tenant:read/write、
// notification:*、system_config:* 等），导致最高权限角色拿不到新能力且无告警。
// 现在必须直接取权威投影，任何人改回手写枚举都会在这里失败。
func TestSysAdminGetsFullCatalog(t *testing.T) {
	roles := BuiltinRolePermissionCodes()
	sysadmin, ok := roles[string(domainrole.SysAdmin)]
	if !ok {
		t.Fatal("BuiltinRolePermissionCodes 缺少 sysadmin 条目")
	}

	cat := AllCodes()
	got := make(map[string]bool, len(sysadmin))
	for _, c := range sysadmin {
		got[c] = true
	}
	if len(got) != len(sysadmin) {
		t.Errorf("sysadmin 码集内重复码：%d 个码去重后 %d 个", len(sysadmin), len(got))
	}

	for _, c := range cat {
		if !got[c] {
			t.Errorf("sysadmin 缺少 catalog 权限码 %q", c)
		}
	}
	if len(got) < len(cat) {
		t.Errorf("sysadmin 码集(%d) 少于 catalog(%d)，存在缺码", len(got), len(cat))
	}
	if len(got) > len(cat) {
		t.Errorf("sysadmin 码集(%d) 多于 catalog(%d)，出现 catalog 之外的越权码", len(got), len(cat))
	}
}
