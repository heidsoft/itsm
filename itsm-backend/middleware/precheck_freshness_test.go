package middleware

import (
	"os"
	"testing"
)

// TestPrecheckMapIsFresh 守卫：rbac_precheck_gen.go 必须与当前路由声明一致。
//
// 路由条目的单一真源是路由声明（RequirePermission 家族）。新增/修改/删除
// 路由权限后忘记 `go run ./cmd/authz-gen` 重新生成 → 本测试红，并给出修复指引。
//
// 与 TestRoutePrecheckAlignment 的分工：
//   - 本测试管**生成物新鲜度**（声明 → 生成物 是否同步）；
//   - 对齐测试管**合并结果口径**（生成物 + precheckFallbackPolicies → 预检解析
//     是否与声明一致），兜住显式回退策略引入的漂移。
func TestPrecheckMapIsFresh(t *testing.T) {
	want, err := GeneratePrecheckSource()
	if err != nil {
		t.Fatalf("从路由声明生成预检映射失败: %v", err)
	}
	path, err := PrecheckGenFilePath()
	if err != nil {
		t.Fatalf("定位生成物失败: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取生成物失败（应提交 rbac_precheck_gen.go）: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("rbac_precheck_gen.go 与路由声明不同步（路由权限已变但未重新生成）。\n" +
			"修复：cd itsm-backend && go run ./cmd/authz-gen，然后提交更新后的生成物。")
	}
}
