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
