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

// TestNoPermissionDeclarationIsDropped 守卫：任何 RequirePermission 家族声明都必须被
// 扫描器定位成 /api/v1 完整路径，一条都不许丢。
//
// 2026-10-04 P0 根因：扫描器此前只认函数内 `x := y.Group(...)` 派生的分组，解析不出
// 以 `*gin.RouterGroup` 形参传入的组根（如 SetupTicketTypeRoutes(tenant *gin.RouterGroup)），
// 于是这类声明在 `/api/v1` 前缀过滤处被静默丢弃、进不了预检映射。DBOnly 生产模式下
// RBACMiddleware 的路径预检对未映射路径一律拒绝，工单类型 CRUD/preset 安装、teams、
// tags、problem-relationships、approval-records、my-approvals 等 21 个端点对
// 非 super_admin 恒 403（ga-gate 租户隔离步骤自 2026-09-27 起持续红的根因）。
//
// 丢声明=锁死端点，所以这里必须硬失败，而不是只让生成器报错。
func TestNoPermissionDeclarationIsDropped(t *testing.T) {
	unresolved, err := ScanUnresolvedPermissionRoutes()
	if err != nil {
		t.Fatalf("扫描权限路由声明失败: %v", err)
	}
	if len(unresolved) == 0 {
		return
	}
	t.Errorf("%d 条权限声明无法定位完整路径，会被预检丢弃（非 super_admin 恒 403）：\n", len(unresolved))
	for _, r := range unresolved {
		t.Errorf("  %s:%d %s %s -> %s:%s", r.File, r.RouteLine, r.Method, r.FullPath, r.Resource, r.Action)
	}
	t.Error("修复：把该域路由注册到函数内显式派生的分组（grp := tenant.Group(\"/xxx\")）" +
		"并让 grp 继承组根，或在 route_scan.go 的注册约定里补该形态。")
}
