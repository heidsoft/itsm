package middleware

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// 本文件是「路由声明 = 权限单一真源」体系的扫描层（2026-09-17 批次 5，P0-E 治本）。
//
// 授权判定是两道闸串联：RBACMiddleware 路径预检(ResourceActionMap) → 路由级 RequirePermission。
// 历史上预检映射与路由声明各自手写，靠守卫测试事后对账——漂移能被发现，但每次都要人肉补表。
// 批次 5 把因果倒过来：ResourceActionMap 的路由条目**从路由声明自动生成**
// （cmd/authz-gen → middleware/rbac_precheck_gen.go），人只维护少量显式回退策略。
//
// 本扫描器原先内嵌在 route_precheck_alignment_test.go 中，现提升为正式代码，
// 供 ① codegen（cmd/authz-gen）② 守卫测试（middleware/router_test 侧）共同消费。

// DeclaredRoute 一条「挂了权限声明」的路由。
type DeclaredRoute struct {
	File      string
	Method    string
	FullPath  string // 归一化后的完整路径（:id/*param → *），带 /api/v1 前缀
	Resource  string
	Action    string
	RouteLine int
}

// ScanDeclaredPermissionRoutes 解析路由注册源码，返回所有挂了 RequirePermission 家族声明的路由。
//
// 支持两种声明位置：
//  1. 路由调用的参数里内联 RequirePermission（含 RequireMSPPermission）；
//  2. 路由所属分组用 .Use(...) 或 .Group(path, ...) 挂载的组级权限。
//
// 路径前缀按函数粒度追踪 `x := y.Group("/p")` 的累积关系；
// 组根统一视为 /api/v1（router/ 与 handlers/ 的 RegisterRoutes 均挂在该前缀下）。
func ScanDeclaredPermissionRoutes() ([]DeclaredRoute, error) {
	// 以本文件位置锚定仓库内 itsm-backend 目录，调用方 cwd 无关
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("无法定位 route_scan.go 所在目录")
	}
	base := filepath.Dir(thisFile)
	return scanDeclaredPermissionRoutesFrom(base)
}

func scanDeclaredPermissionRoutesFrom(base string) ([]DeclaredRoute, error) {
	var files []string
	for _, pattern := range []string{filepath.Join(base, "..", "router", "*.go"), filepath.Join(base, "..", "handlers", "*", "*.go")} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("glob %s: %w", pattern, err)
		}
		for _, m := range matches {
			if strings.HasSuffix(m, "_test.go") {
				continue
			}
			files = append(files, m)
		}
	}
	sort.Strings(files)

	routeMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true,
	}

	var out []DeclaredRoute
	for _, file := range files {
		src, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", file, err)
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, src, 0)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", file, err)
		}
		rel := filepath.ToSlash(file)

		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}

			// 每个函数独立追踪分组（避免跨函数同名变量串味）
			groupPath := map[string]string{}
			// var -> 组级声明的 (resource, action)
			groupDecl := map[string][][2]string{}

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				switch node := n.(type) {
				case *ast.AssignStmt:
					if len(node.Lhs) != 1 || len(node.Rhs) != 1 {
						return true
					}
					ident, ok := node.Lhs[0].(*ast.Ident)
					if !ok {
						return true
					}
					call, ok := node.Rhs[0].(*ast.CallExpr)
					if !ok {
						return true
					}
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok || sel.Sel.Name != "Group" || len(call.Args) == 0 {
						return true
					}
					parent := "/api/v1"
					if p, ok := sel.X.(*ast.Ident); ok {
						if known, exists := groupPath[p.Name]; exists {
							parent = known
						}
					}
					if lit, ok := call.Args[0].(*ast.BasicLit); ok {
						if seg, err := strconv.Unquote(lit.Value); err == nil {
							// gin 惯例：挂在根 Engine（r）上的组用绝对路径段 "/api/v1/..."，
							// 此时父前缀为空；挂在 tenant 组下的用相对段。两种风格共存。
							if strings.HasPrefix(seg, "/api/v1") {
								parent = ""
							}
							groupPath[ident.Name] = parent + seg
						}
					}
					if perms := permissionPairs(call.Args[1:]); len(perms) > 0 {
						groupDecl[ident.Name] = append(groupDecl[ident.Name], perms...)
					}
				case *ast.ExprStmt:
					call, ok := node.X.(*ast.CallExpr)
					if !ok {
						return true
					}
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok || sel.Sel.Name != "Use" {
						return true
					}
					recv, ok := sel.X.(*ast.Ident)
					if !ok {
						return true
					}
					if perms := permissionPairs(call.Args); len(perms) > 0 {
						groupDecl[recv.Name] = append(groupDecl[recv.Name], perms...)
					}
				}
				return true
			})

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || !routeMethods[sel.Sel.Name] || len(call.Args) == 0 {
					return true
				}
				recv, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}

				// 路由级声明优先；无则继承组级声明
				perms := permissionPairs(call.Args[1:])
				if len(perms) == 0 {
					perms = groupDecl[recv.Name]
				}
				if len(perms) == 0 {
					return true
				}

				lit, ok := call.Args[0].(*ast.BasicLit)
				if !ok {
					return true
				}
				seg, err := strconv.Unquote(lit.Value)
				if err != nil {
					return true
				}
				full := normalizeRoutePath(groupPath[recv.Name] + seg)
				if !strings.HasPrefix(full, "/api/v1/") && full != "/api/v1" {
					return true
				}

				for _, p := range perms {
					out = append(out, DeclaredRoute{
						File:      rel,
						Method:    sel.Sel.Name,
						FullPath:  full,
						Resource:  p[0],
						Action:    p[1],
						RouteLine: fset.Position(call.Pos()).Line,
					})
				}
				return true
			})
		}
	}
	return out, nil
}

// permissionPairs 从参数列表中提取 RequirePermission 家族的 (resource, action) 对。
func permissionPairs(args []ast.Expr) [][2]string {
	var out [][2]string
	for _, a := range args {
		call, ok := a.(*ast.CallExpr)
		if !ok {
			continue
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		switch sel.Sel.Name {
		case "RequirePermission", "RequirePermissionAny", "RequireMSPPermission":
		default:
			continue
		}
		if len(call.Args) < 2 {
			continue
		}
		res, ok := stringLit(call.Args[0])
		if !ok {
			continue
		}
		for _, actArg := range call.Args[1:] {
			act, ok := stringLit(actArg)
			if !ok {
				continue
			}
			out = append(out, [2]string{res, act})
		}
	}
	return out
}

func stringLit(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return s, true
}

// normalizeRoutePath 把 gin 路径参数（:id/:key）归一为 *，
// 以便与 ResourceActionMap 的通配符模式比对。
func normalizeRoutePath(p string) string {
	if p == "" {
		return "/api/v1"
	}
	segs := strings.Split(p, "/")
	for i, s := range segs {
		if strings.HasPrefix(s, ":") || strings.HasPrefix(s, "*") {
			segs[i] = "*"
		}
	}
	return strings.Join(segs, "/")
}
