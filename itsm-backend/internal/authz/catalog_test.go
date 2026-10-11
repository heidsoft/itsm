package authz

import (
	"testing"
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
