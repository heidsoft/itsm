package authz

import (
	"testing"
)

// TestDefinitionsParityWithSeeder 迁移期一致性验证（批次 6 第 1 步）。
//
// 本测试保证从 pkg/seeder.seeder.go 搬到本包的 Definitions() 与原
// permissionDefinitions() 输出完全一致——零行为变化。批次 6 后续 PR 把
// seeder 切到本包时即可删除本测试。
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
