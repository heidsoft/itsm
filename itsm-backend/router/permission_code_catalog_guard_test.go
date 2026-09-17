package router

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"itsm-backend/pkg/seeder"
)

// TestRoutePermissionCodesAreDefined 守卫：路由声明的每个 (resource, action) 必须落在权限码权威源内。
//
// 背景（2026-09-17 P0-B，已清零）
// ----------------------------
// checkPermissionMatch（middleware/rbac.go）只做**精确匹配**或 `*` 通配：
// 角色持有 `cmdb:read` 永远匹配不上 `cmdb_ci:read`。因此一旦路由引用了
// permissionDefinitions 中不存在的码，该路由对除 super_admin 外的所有角色
// **永久 403**——DBOnly configured 态下任何授权都无法满足（audit 实测 95 种码 /
// 240 处引用，CMDB 核心整域不可用，批次 2 收敛清零）。
//
// 本测试固化该不变量：新增路由引用未定义码 → CI 红，并提示两条修复路径：
//  1. 优先收敛到既有码（零数据迁移，改路由声明）；
//  2. 确属新能力面才补码（permissionDefinitions 登记 + 角色授权 + 预检映射三处同步）。
func TestRoutePermissionCodesAreDefined(t *testing.T) {
	allowed := map[string]bool{}
	for _, c := range seeder.AllDefinedPermissionCodes() {
		allowed[c] = true
	}
	if len(allowed) < 100 {
		t.Fatalf("权限码权威源仅 %d 个码，疑似提取逻辑失效", len(allowed))
	}

	var files []string
	for _, pattern := range []string{"*.go", "../handlers/*/*.go", "../handlers/common/*.go"} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatalf("glob %s: %v", pattern, err)
		}
		for _, m := range matches {
			if strings.HasSuffix(m, "_test.go") {
				continue
			}
			files = append(files, m)
		}
	}
	sort.Strings(files)

	var violations []string
	for _, file := range files {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, src, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		rel := filepath.ToSlash(file)

		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			switch sel.Sel.Name {
			case "RequirePermission", "RequirePermissionAny":
			default:
				return true
			}
			if len(call.Args) < 2 {
				return true
			}
			resLit, ok := call.Args[0].(*ast.BasicLit)
			if !ok {
				return true
			}
			res, err := strconv.Unquote(resLit.Value)
			if err != nil {
				return true
			}
			for _, actArg := range call.Args[1:] {
				lit, ok := actArg.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				act, err := strconv.Unquote(lit.Value)
				if err != nil {
					continue
				}
				code := res + ":" + act
				if !allowed[code] {
					violations = append(violations, fmt.Sprintf(
						"%s:%d  RequirePermission(%q, %q) —— 码 %q 不在权限码权威源（pkg/seeder permissionDefinitions）",
						rel, fset.Position(call.Pos()).Line, res, act, code))
				}
			}
			return true
		})
	}

	if len(violations) > 0 {
		sort.Strings(violations)
		t.Errorf("路由声明了码空间中不存在的权限码（B 类复发），这些路由对非 super_admin 角色永久 403：\n  %s\n\n"+
			"修复优先级：① 收敛到既有码（改路由声明，零数据迁移）；② 确属新能力面才补码：\n"+
			"   permissionDefinitions 登记码 + builtinRolePermissionCodes 授权 + ResourceActionMap 预检条目 三处同步。",
			strings.Join(violations, "\n  "))
	}
}
