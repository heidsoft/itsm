package authz

import (
	"sort"
	"testing"

	"itsm-backend/pkg/seeder"
)

// TestParityWithLegacyDefinitions 迁移期一致性验证：catalog.AllCodes 与
// seeder.AllDefinedPermissionCodes 字节级一致（仅当此测试通过，seeder 才能切到本包）。
//
// 失败排查：catalog 多/少码 = 搬移时漏/加，逐条对照 seeder.go 1614–1811 行的 permissionDefinitions()。
func TestParityWithLegacyDefinitions(t *testing.T) {
	cat := AllCodes()
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
		t.Errorf("catalog 与 seeder 权限码清单不一致，批次 6 第 1 步搬移有遗漏或多余：\n"+
			"  仅 catalog 多 %d 条：%v\n"+
			"  仅 seeder 有 %d 条：%v",
			len(extra), extra, len(missing), missing)
	}

	if len(cat) < 100 {
		t.Fatalf("catalog 仅 %d 个码（seeder 有 %d 个），疑似提取逻辑失效", len(cat), len(old))
	}
}
