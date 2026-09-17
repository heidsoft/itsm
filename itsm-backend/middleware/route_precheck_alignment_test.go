package middleware

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
)

// precheckMismatchAllowlist 已知「路由声明 与 预检解析」口径错配的台账。
//
// 键格式：`<相对路径> <METHOD> <完整路径>`。每条必须写明理由与去向。
//
// 2026-09-17 批次 3 已清零：18 条 POST-read 错配（预检同步放宽为 read）与
// POST /bpmn/ai/preview 一并修复，台账清空但机制保留——新错配必须先登记
// （写明理由与去向）才能合入，修复后失效条目会被本测试强制清理。
var precheckMismatchAllowlist = map[string]string{}

// TestRoutePrecheckAlignment 守卫：路由声明的 (resource, action) 必须与路径预检解析结果一致。
//
// 背景（2026-09-17 P0 实测教训）
// ----------------------------
// 授权判定是**两道闸串联**，且顺序固定：
//
//	RBACMiddleware 路径预检(ResourceActionMap)  →  路由级 RequirePermission
//
// 预检先跑且失败即拒。因此只要预检口径与路由声明不一致，就会出现
// 「角色明明有声明的权限、却拿不到路由」——最隐蔽的形态是：
//
//	POST /bpmn/lint  路由声明 bpmn:read，预检却因 /api/v1/bpmn/* 解析成 bpmn:write
//	→ 只有 bpmn:read 的角色被 403（prod 探针实测 technician 403 / admin 400）
//
// 根因是 ResourceActionMap 按方法分段存放，条目放错段（放 GET 段对 POST 请求无效）
// 或干脆没登记。本测试把「预检解析结果必须命中该路由的声明集合」固化为断言，
// 覆盖全部动作（不只 read）；多动作声明（RequirePermissionAny）命中任一即通过。
//
// 作用轴说明：预检无匹配（nil）→ 预检空操作，判定完全由路由级决定，不算冲突；
// 但这意味着该路径落入了 L3 URL 推断，推断失配风险由探针回归兜底（ Known gap）。
func TestRoutePrecheckAlignment(t *testing.T) {
	routes := scanDeclaredPermissionRoutes(t)

	// 按 (file, method, fullPath) 聚合声明集合，支持 RequirePermissionAny 多动作
	type key struct {
		file, method, path string
	}
	declared := map[key]map[[2]string]bool{}
	order := []key{}
	for _, r := range routes {
		k := key{r.file, r.method, r.fullPath}
		if declared[k] == nil {
			declared[k] = map[[2]string]bool{}
			order = append(order, k)
		}
		declared[k][[2]string{r.resource, r.action}] = true
	}

	passed := map[string]bool{}
	failed := map[string]bool{}
	var violations []string
	for _, k := range order {
		resolved := getPermissionFromPath(k.method, k.path)
		if resolved == nil {
			continue
		}
		pair := [2]string{resolved.Resource, resolved.Action}
		id := k.file + " " + k.method + " " + k.path
		if declared[k][pair] {
			passed[id] = true
			continue
		}
		failed[id] = true
		if _, ok := precheckMismatchAllowlist[id]; ok {
			continue
		}
		want := make([]string, 0, len(declared[k]))
		for p := range declared[k] {
			want = append(want, p[0]+":"+p[1])
		}
		sort.Strings(want)
		violations = append(violations, fmt.Sprintf(
			"%s  %s %s\n      路由声明 %s，预检解析为 %s:%s",
			k.file, k.method, k.path, strings.Join(want, " | "), resolved.Resource, resolved.Action))
	}

	if len(violations) > 0 {
		sort.Strings(violations)
		t.Errorf("以下路由的「声明权限」与「路径预检解析」不一致，"+
			"低权角色会因预检先拒而拿不到已授权的路由：\n  %s\n\n"+
			"修复：在 middleware.ResourceActionMap 的**对应方法段**补最具体条目"+
			"（条目放错方法段等于没放）。\n"+
			"确属待处理欠账的，加入 precheckMismatchAllowlist 并写明去向。",
			strings.Join(violations, "\n  "))
	}

	// 台账防腐化：条目要么仍有理由（仍错配），要么应被删除
	var stale []string
	for id, reason := range precheckMismatchAllowlist {
		if strings.TrimSpace(reason) == "" {
			stale = append(stale, id+" —— 缺少理由")
		}
		if !failed[id] {
			state := "已不再错配"
			if passed[id] {
				state = "预检已与声明一致"
			} else {
				state = "未命中任何已注册路由（改名/删除后请同步台账）"
			}
			stale = append(stale, id+" —— "+state)
		}
	}
	if len(stale) > 0 {
		sort.Strings(stale)
		t.Errorf("precheckMismatchAllowlist 存在失效条目：\n  %s", strings.Join(stale, "\n  "))
	}
}

type declaredRoute struct {
	file      string
	method    string
	fullPath  string
	resource  string
	action    string
	routeLine int
}

// scanDeclaredPermissionRoutes 解析路由注册源码，返回所有「挂了 RequirePermission 家族声明」的路由。
//
// 支持两种声明位置：
//  1. 路由调用的参数里内联 RequirePermission；
//  2. 路由所属分组用 .Use(...) 或 .Group(path, ...) 挂载的组级权限。
//
// 路径前缀按函数粒度追踪 `x := y.Group("/p")` 的累积关系；
// 组根统一视为 /api/v1（router/ 与 handlers/ 的 RegisterRoutes 均挂在该前缀下）。
func scanDeclaredPermissionRoutes(t *testing.T) []declaredRoute {
	t.Helper()

	var files []string
	for _, pattern := range []string{"../router/*.go", "../handlers/*/*.go"} {
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

	routeMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true,
	}

	var out []declaredRoute
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
					out = append(out, declaredRoute{
						file:      rel,
						method:    sel.Sel.Name,
						fullPath:  full,
						resource:  p[0],
						action:    p[1],
						routeLine: fset.Position(call.Pos()).Line,
					})
				}
				return true
			})
		}
	}
	return out
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
