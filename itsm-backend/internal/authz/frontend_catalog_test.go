package authz

// 前端权限码目录生成物的新鲜度守卫。
//
// catalog.gen.ts 的单一来源是本包 Definitions()。有人在后端加/删权限码
// 却忘记重新生成前端目录时，前端会继续使用过期码空间：轻则新码无法
// 在前端引用，重则已删码仍被前端引用（幽灵码复活）。本测试在 CI 拦下
// 这种漂移，与 TestPrecheckMapIsFresh 同一机制。

import (
	"os"
	"strings"
	"testing"
)

func TestFrontendCatalogGenIsFresh(t *testing.T) {
	path, err := FrontendCatalogGenFilePath()
	if err != nil {
		t.Fatalf("定位生成物路径失败: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取前端目录生成物失败（先运行 cd itsm-backend && go run ./cmd/authz-gen）: %v", err)
	}
	src := string(raw)

	if !strings.Contains(src, "DO NOT EDIT") {
		t.Errorf("生成物缺少 DO NOT EDIT 头，疑似手写文件: %s", path)
	}

	defs := Definitions()
	if len(defs) == 0 {
		t.Fatal("Definitions() 为空")
	}

	// 1) 每个码都必须出现在生成物里（PERM/catalog 双份，至少一份命中即可断言漂移）
	for _, d := range defs {
		if !strings.Contains(src, `code: "`+d.Code+`"`) {
			t.Errorf("权限码 %q 未出现在 catalog.gen.ts：后端码空间已变更，请重新生成（cd itsm-backend && go run ./cmd/authz-gen）", d.Code)
		}
	}

	// 2) 生成物不得包含码空间之外的条目（数量守帐：catalog 数组条目数 == Definitions 数）
	got := strings.Count(src, "{ code: \"")
	if got != len(defs) {
		t.Errorf("catalog.gen.ts 条目数 %d != Definitions() %d：生成物与后端码空间不一致，请重新生成", got, len(defs))
	}
}
