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

// precheckMismatchBatch23 批次 2/3 待逐端点判定方向的错配清单。
//
// 共同形态：**POST 方法承载只读语义**（search / chat / analyze / monitor / export），
// 路由声明 read，而 ResourceActionMap 的 POST 段只有粗粒度的 `/api/v1/<域>/* = write`
// → 只持有 read 的角色在预检层就被拒。已实测确认该机制（POST /bpmn/lint 同因）。
//
// 为什么不在本批次直接补 read 条目：修复方向**需要逐端点产品判定**，不能一刀切放宽。
//   - /ai/chat、/ai/triage、/ai/rag/search、/sla/monitor：会消耗 AI 配额或触发巡检任务，
//     声明为 write 可能才是正确语义（此时应改路由声明，而非放宽预检）；
//   - /cmdb/cis/search、/tickets/prediction/trend、/ai/analytics：纯查询/统计，
//     预检应放宽为 read。
//
// 因此归入批次 2/3「码空间补齐 + 预检映射统一」，与 B 类（95 个未定义权限码）一起
// 做一次逐端点定方向的对齐。
var precheckMismatchBatch23 = []string{
	"../router/ai_routes.go POST /api/v1/agent/tools/execute",
	"../router/ai_routes.go POST /api/v1/ai/analytics",
	"../router/ai_routes.go POST /api/v1/ai/chat",
	"../router/ai_routes.go POST /api/v1/ai/chat/stream",
	"../router/ai_routes.go POST /api/v1/ai/incidents/*/analyze",
	"../router/ai_routes.go POST /api/v1/ai/predictions",
	"../router/ai_routes.go POST /api/v1/ai/rag/search",
	"../router/ai_routes.go POST /api/v1/ai/tickets/*/analyze",
	"../router/ai_routes.go POST /api/v1/ai/triage",
	"../router/cmdb_routes.go POST /api/v1/cmdb/cis/search",
	"../router/cmdb_routes.go POST /api/v1/configuration-items/search",
	"../router/incident_routes.go POST /api/v1/incidents/monitoring",
	"../router/sla_routes.go POST /api/v1/sla/check-compliance/*",
	"../router/sla_routes.go POST /api/v1/sla/monitor",
	"../router/sla_routes.go POST /api/v1/sla/monitoring",
	"../router/ticket_routes.go POST /api/v1/tickets/analytics/deep",
	"../router/ticket_routes.go POST /api/v1/tickets/prediction/export",
	"../router/ticket_routes.go POST /api/v1/tickets/prediction/trend",
}

const precheckMismatchBatch23Reason = "既有口径漂移：POST 承载只读语义但预检按 POST 段 write 映射。" +
	"修复方向需逐端点产品判定（放宽预检 vs 收紧声明），归批次 2/3 与 B 类码空间补齐一并处理。"

// readRoutePrecheckAllowlist 已知「路由声明 read / 预检解析 write」口径错配的台账。
//
// 键格式：`<相对路径> <METHOD> <完整路径>`。每条必须写明理由与去向。
//
// 该台账是**欠账清单而非豁免**：条目一旦不再错配（预检也解析为 read），
// 测试会以「条目已失效」报错，强制清理，避免台账腐烂。
var readRoutePrecheckAllowlist = func() map[string]string {
	m := map[string]string{
		"../handlers/bpmn/ai_generator.go POST /api/v1/bpmn/ai/preview": "预检落到 /api/v1/bpmn/* = bpmn:write。本批次不动 ai_generator 授权口径——同组 " +
			"POST /bpmn/ai/generate 声明的 workflow:create 码在码空间根本不存在（B 类），" +
			"整组需与「码空间补齐 + 预检映射统一」一起改，属批次 2/3。",
	}
	for _, k := range precheckMismatchBatch23 {
		m[k] = precheckMismatchBatch23Reason
	}
	return m
}()

// TestReadRoutesPrecheckActionAlignment 守卫：路由声明 read 时，路径预检不得解析成 write。
//
// 背景（2026-09-17 P0 实测教训）
// ----------------------------
// 授权判定是**两道闸串联**，且顺序固定：
//
//	RBACMiddleware 路径预检(ResourceActionMap)  →  路由级 RequirePermission
//
// 预检先跑且失败即拒。因此只要写的动作口径与路由声明不一致，就会出现
// 「角色明明有声明的权限、却拿不到路由」——最隐蔽的形态是：
//
//	POST /bpmn/lint  路由声明 bpmn:read，预检却因 /api/v1/bpmn/* 解析成 bpmn:write
//	→ 只有 bpmn:read 的角色被 403（prod 探针实测 technician 403 / admin 400）
//
// 根因是 ResourceActionMap 按方法分段存放，条目放错段（放 GET 段对 POST 请求无效）
// 或干脆没登记。本测试把「声明 action 与预检 action 必须一致」固化为断言。
//
// 作用轴说明：只比对 **action**（read 是否被解析成写动作）。资源名差异（例如
// cmdb_ci:read 声明 vs cmdb:read 预检）另有 B 类欠账，不在此测试职责内。
func TestReadRoutesPrecheckActionAlignment(t *testing.T) {
	routes := scanDeclaredReadRoutes(t)

	passed := map[string]bool{}
	failed := map[string]bool{}
	var violations []string
	for _, r := range routes {
		resolved := getPermissionFromPath(r.method, r.fullPath)
		if resolved == nil {
			// 预检无匹配 → 预检是空操作，判定完全由路由级决定，不存在口径冲突
			continue
		}
		key := r.file + " " + r.method + " " + r.fullPath
		if resolved.Action == r.action {
			passed[key] = true
			continue
		}
		failed[key] = true
		if _, ok := readRoutePrecheckAllowlist[key]; ok {
			continue
		}
		violations = append(violations, fmt.Sprintf(
			"%s  %s %s\n      路由声明 %s:%s，预检解析为 %s:%s",
			r.file, r.method, r.fullPath, r.resource, r.action, resolved.Resource, resolved.Action))
	}

	if len(violations) > 0 {
		sort.Strings(violations)
		t.Errorf("以下路由的「声明 action」与「路径预检 action」不一致，"+
			"低权角色会因预检先拒而拿不到已授权的路由：\n  %s\n\n"+
			"修复：在 middleware.ResourceActionMap 的**对应方法段**补最具体条目"+
			"（条目放错方法段等于没放）。\n"+
			"确属待处理欠账的，加入 readRoutePrecheckAllowlist 并写明去向。",
			strings.Join(violations, "\n  "))
	}

	// 台账防腐化：条目要么仍有理由（仍错配），要么应被删除
	var stale []string
	for key, reason := range readRoutePrecheckAllowlist {
		if strings.TrimSpace(reason) == "" {
			stale = append(stale, key+" —— 缺少理由")
		}
		if !failed[key] {
			state := "已不再错配"
			if passed[key] {
				state = "预检已与声明一致"
			} else {
				state = "未命中任何已注册路由（改名/删除后请同步台账）"
			}
			stale = append(stale, key+" —— "+state)
		}
	}
	if len(stale) > 0 {
		sort.Strings(stale)
		t.Errorf("readRoutePrecheckAllowlist 存在失效条目：\n  %s", strings.Join(stale, "\n  "))
	}
}

type declaredReadRoute struct {
	file      string
	method    string
	fullPath  string
	resource  string
	action    string
	allowKey  string
	routeLine int
}

// scanDeclaredReadRoutes 解析路由注册源码，返回所有「声明 action=read」的路由。
//
// 支持两种声明位置：
//  1. 路由调用的参数里内联 RequirePermission；
//  2. 路由所属分组用 .Use(...) 或 .Group(path, ...) 挂载的组级权限。
//
// 路径前缀按函数粒度追踪 `x := y.Group("/p")` 的累积关系；
// 组根统一视为 /api/v1（router/ 与 handlers/ 的 RegisterRoutes 均挂在该前缀下）。
func scanDeclaredReadRoutes(t *testing.T) []declaredReadRoute {
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

	var out []declaredReadRoute
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
					if p[1] != "read" {
						continue
					}
					out = append(out, declaredReadRoute{
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
