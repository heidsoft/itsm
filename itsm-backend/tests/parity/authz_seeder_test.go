// Package parity 跨包字段一致性守卫。
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

	domainrole "itsm-backend/domain/role"
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
		t.Errorf("权限码清单不一致（单一源被破坏，需立即对齐）：\n"+
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

// TestAdminCoversDepartmentWrite 防止「路由声明了一个码、catalog 也定义了，
// 却没有任何内置角色持有」的死码复活。
//
// 回归背景：POST /api/v1/departments/:id/init-processes 声明
// RequirePermission("department", "write")，而 admin 曾被手工写成只持
// department:create/update/delete 三个细分码；DBOnly + checkPermissionMatch
// 精确匹配下，部门流程页的「初始化默认流程」对所有租户角色恒 403。
// 细分码本就由 synonymParity 从 write 派生，绑定里只应写 write 本体。
func TestAdminCoversDepartmentWrite(t *testing.T) {
	admin := authz.BuiltinRolePermissionCodes()[domainrole.Admin]
	have := make(map[string]bool, len(admin))
	for _, c := range admin {
		have[c] = true
	}

	for _, want := range []string{
		"department:write",
		"department:read",
		"department:create",
		"department:update",
		"department:delete",
	} {
		if !have[want] {
			t.Errorf("admin 缺少 %q：对应路由将 403（细分码应由 department:write 派生，不要在绑定里手抄）", want)
		}
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
