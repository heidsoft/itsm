// authz-gen 从路由声明生成预检映射（权限单一真源体系，2026-09-17 批次 5）。
//
// 用法：cd itsm-backend && go run ./cmd/authz-gen
// 产物：middleware/rbac_precheck_gen.go（禁手改，新鲜度由 TestPrecheckMapIsFresh 守卫）
//
// 何时需要重新生成：新增/修改/删除任何 RequirePermission 家族声明之后。
// 守卫测试（TestRoutePrecheckAlignment / TestPrecheckMapIsFresh）会在 CI 拦下过期生成物。
package main

import (
	"fmt"
	"os"

	"itsm-backend/middleware"
)

func main() {
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
}
