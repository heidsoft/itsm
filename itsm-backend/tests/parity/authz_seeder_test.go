// Package parity 跨包字段一致性守卫（批次 6 第 2 步）。
//
// 放在独立包是为了避开 import cycle：
//   - authz → seeder 触发循环（authz 测试 import seeder）→ 反过来 seeder 依赖 authz → 死锁
//   - 此处放在叶子包 tests/parity 同时 import 两者，OK
//
// 字节级断言 authz.AllCodes() 与 seeder.AllDefinedPermissionCodes() 一致：
// 两者任意漂移 CI 红，提示"权限码单一源破坏，需立即对齐"。
package parity

import (
	"sort"
	"testing"

	"itsm-backend/internal/authz"
	"itsm-backend/pkg/seeder"
)

func TestAuthzSeederCodeSetParity(t *testing.T) {
	cat := authz.AllCodes()
	old := seeder.AllDefinedPermissionCodes()

	setCat := map[string]bool{}
	for _, c := range cat {
		setCat[c] = true
	}
	setOld := map[string]bool{}
	for _, c := range old {
		setOld[c] = true
	}

	var missing, extra []string
	for c := range setOld {
		if !setCat[c] {
			missing = append(missing, c)
		}
	}
	for c := range setCat {
		if !setOld[c] {
			extra = append(extra, c)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)

	if len(missing) > 0 || len(extra) > 0 {
		t.Errorf("权限码清单不一致（批次 6 单一源破坏，需立即对齐）：\n"+
			"  仅 authz/catalog 有 %d 条：%v\n"+
			"  仅 seeder 有 %d 条：%v",
			len(extra), extra, len(missing), missing)
	}
	if len(cat) < 100 {
		t.Fatalf("authz.AllCodes 仅 %d 个码（seeder 有 %d 个），疑似提取逻辑失效", len(cat), len(old))
	}
}

// TestRoleBindingsSubsetOfCodes 角色绑定引用的每个码必须在 catalog 中存在。
//
// 防止「派生规则/手写绑定引用了 catalog 中未定义的码」——codegen 单一源后
// catalog 是权限码权威源，角色绑定引用未定义码意味着派生链破坏或迁移遗漏。
func TestRoleBindingsSubsetOfCodes(t *testing.T) {
	codes := authz.AllCodes()
	allowed := make(map[string]bool, len(codes))
	for _, c := range codes {
		allowed[c] = true
	}

	bindings := authz.BuiltinRolePermissionCodes()

	var violations []string
	for role, roleCodes := range bindings {
		for _, c := range roleCodes {
			if !allowed[c] {
				violations = append(violations, role+": "+c)
			}
		}
	}
	sort.Strings(violations)

	if len(violations) > 0 {
		t.Errorf("角色绑定引用了 catalog 中不存在的权限码（单一源破坏，需立即对齐）：\n"+
			"  %d 处违规：\n  %s\n\n"+
			"修复：把对应码补到 internal/authz/catalog.go 的 Definitions() 中。",
			len(violations), joinLines(violations))
	}
}

func joinLines(s []string) string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += "\n  "
		}
		out += v
	}
	return out
}
