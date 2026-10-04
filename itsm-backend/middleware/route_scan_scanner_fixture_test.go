package middleware

import (
	"path/filepath"
	"testing"
)

// TestScannerResolvesEveryRegistrationShape 锁住 route_scan.go 认得的注册形态。
//
// 预检映射由路由声明生成，所以「扫描器认得哪些注册形态」就是授权面的真相边界：
// 一条声明定位不出完整路径，对应端点在 DBOnly 生产下对非 super_admin 恒 403
// （2026-10-04 P0，21 个端点被静默丢弃的根因）。夹具把四种形态钉死，
// 新增注册风格时必须先在 testdata 里证明扫描器认得它。
func TestScannerResolvesEveryRegistrationShape(t *testing.T) {
	// base 只用于拼 base/../router；anchor 目录无需存在。
	base := filepath.Join("testdata", "routescan", "anchor")
	declared, unresolved, err := scanDeclaredPermissionRoutesFrom(base)
	if err != nil {
		t.Fatalf("扫描夹具失败: %v", err)
	}

	want := map[string]string{ // "METHOD /full/path" -> "resource:action"
		"GET /api/v1/fixture-things":         "fixture:read",
		"POST /api/v1/fixture-things":        "fixture:write",
		"DELETE /api/v1/fixture-things/*":    "fixture:delete",
		"GET /api/v1/fixture-alias":          "fixture_alias:read",
		"GET /api/v1/fixture-absolute/items": "fixture_absolute:read",
	}
	for _, d := range declared {
		key := d.Method + " " + d.FullPath
		got := d.Resource + ":" + d.Action
		expect, ok := want[key]
		if !ok {
			t.Errorf("意外的已解析声明 %s -> %s（路径拼装漂移）", key, got)
			continue
		}
		if expect != got {
			t.Errorf("%s: 期望 %s，实际 %s", key, expect, got)
		}
		delete(want, key)
	}
	for key, perm := range want {
		t.Errorf("夹具声明 %s -> %s 未被解析（丢失即端点锁死）", key, perm)
	}

	if len(unresolved) != 1 {
		t.Fatalf("期望恰好 1 条无法定位的声明，实际 %d 条: %+v", len(unresolved), unresolved)
	}
	u := unresolved[0]
	if u.Method != "GET" || u.FullPath != "/fixture-mystery" || u.Resource != "fixture_mystery" || u.Action != "read" {
		t.Errorf("unresolved 记录不符: %+v", u)
	}
}
