// authz-gen 从路由声明生成预检映射 + 从权限码权威清单生成前端码目录。
//
// 用法：cd itsm-backend && go run ./cmd/authz-gen
// 产物：
//  1. middleware/rbac_precheck_gen.go（禁手改，新鲜度由 TestPrecheckMapIsFresh 守卫）
//  2. ../itsm-frontend/src/lib/permissions/catalog.gen.ts
//     （前端权限码的唯一一份；禁手改，新鲜度由 internal/authz/frontend_catalog_test.go 守卫）
//
// 何时需要重新生成：新增/修改/删除任何 RequirePermission 家族声明，
// 或修改 internal/authz/catalog.go 的 Definitions() 之后。
// 守卫测试（TestRoutePrecheckAlignment / TestPrecheckMapIsFresh /
// TestFrontendCatalogGenIsFresh）会在 CI 拦下过期生成物。
package main

import (
	"fmt"
	"os"

	"itsm-backend/internal/authz"
	"itsm-backend/middleware"
)

func main() {
	// 产物 1：RBAC 路径预检映射
	src, err := middleware.GeneratePrecheckSource()
	if err != nil {
		fmt.Fprintln(os.Stderr, "生成失败:", err)
		os.Exit(1)
	}
	// 输出位置以 middleware 包源码目录为锚（route_scan.go 同目录），与 cwd 无关
	out, err := middleware.PrecheckGenFilePath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "定位输出文件失败:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(out, src, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "写入失败:", err)
		os.Exit(1)
	}
	fmt.Printf("已生成 %s\n", out)

	// 产物 2：前端权限码目录（前端码空间唯一一份）
	ts, err := authz.GenerateFrontendCatalogTS()
	if err != nil {
		fmt.Fprintln(os.Stderr, "生成前端权限目录失败:", err)
		os.Exit(1)
	}
	tsOut, err := authz.FrontendCatalogGenFilePath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "定位前端权限目录输出文件失败:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(tsOut, ts, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "写入前端权限目录失败:", err)
		os.Exit(1)
	}
	fmt.Printf("已生成 %s\n", tsOut)
}
