package router

import (
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

// writeRouteExemptions 白名单：明确接受「无行内权限中间件」的写路由。
//
// 键格式：`<相对路径> <METHOD> <路径>`。只允许三类，且必须写明理由：
//  1. 公开面（登录/注册/找回密码/Webhook 回调）——登录前不存在角色可言，
//     鉴权由独立机制承担（Webhook 签名、重置令牌、bootstrap token）；
//  2. 身份自省面（logout / switch-tenant / ws-ticket）——仅需已认证身份，
//     授权依据是「该用户与目标租户/自身会话的归属关系」，不是资源动作权限；
//  3. 分组级已挂权限中间件但不在本扫描器可识别形式内的路由（目前为空）。
//
// 新增条目前必须写明理由；否则请直接在路由上挂 middleware.RequirePermission。
// 背景：2026-09-17 P0「越权写收口」——technician（rank=2）曾可创建/发布 BPMN 流程定义、
// 触发流程、改写流程绑定，因为这些写路由只靠 ResourceActionMap 的粗粒度路径预检兜底。
var writeRouteExemptions = map[string]string{
	// —— 1. 公开面：登录前 / 外部回调 ——
	"router.go POST /auth/login":                                          "登录入口，登录前无角色",
	"router.go POST /auth/refresh":                                        "刷新令牌，凭 refresh token 而非会话角色",
	"router.go POST /refresh-token":                                       "刷新令牌（兼容路径）",
	"router.go POST /auth/register":                                       "注册入口，登录前无角色",
	"router.go POST /auth/forgot-password":                                "找回密码入口，登录前无角色",
	"router.go POST /auth/reset-password":                                 "重置密码，凭一次性重置令牌",
	"router.go POST /auth/validate-reset-token":                           "校验一次性重置令牌",
	"feishu_routes.go POST /feishu/webhook/:instance_id":                  "飞书事件回调，走飞书签名校验",
	"../handlers/feishu/handler.go POST /feishu/webhook/:instance_id":     "飞书事件回调，走飞书签名校验",
	"../handlers/dingtalk/handler.go POST /dingtalk/webhook/:instance_id": "钉钉事件回调，走钉钉签名校验",
	"../handlers/wecom/handler.go POST /wecom/webhook/:instance_id":       "企微事件回调，走企微签名校验",
	"bootstrap_routes.go POST /create-admin":                              "初始化引导：仅在系统未初始化时可用，凭一次性 bootstrap token",

	// —— 2. 身份自省面：只需已认证身份 + 归属校验 ——
	"common_system_routes.go POST /logout":        "登出，凭会话自身即可失效",
	"common_system_routes.go POST /switch-tenant": "切换至该用户已归属的租户，授权依据是租户成员关系而非资源动作",
	"router.go POST /ws/ticket":                   "颁发一次性 WebSocket 票据，凭已认证身份且仅限本人会话",
}

type scannedRoute struct {
	file    string
	line    int
	method  string
	path    string
	recv    string
	hasPerm bool
}

// TestWriteRoutesRequirePermission 守卫：所有写方法路由必须显式挂权限中间件。
//
// 扫描范围：router/ 与 handlers/<domain>/ 下的非测试 Go 源文件（路由注册点）。
// 判定为「已保护」的条件（任一满足）：
//  1. 该路由调用自身的参数里出现 middleware.RequirePermission(...)；
//  2. 接收者分组由 .Group(path, 权限中间件...) 创建，或后续对该分组调用过 .Use(权限中间件)；
//  3. 命中 writeRouteExemptions 白名单（需写明理由）。
func TestWriteRoutesRequirePermission(t *testing.T) {
	routes := scanRouteRegistrations(t)

	var unprotected []scannedRoute
	for _, r := range routes {
		if r.hasPerm {
			continue
		}
		if _, ok := writeRouteExemptions[r.file+" "+r.method+" "+r.path]; ok {
			continue
		}
		unprotected = append(unprotected, r)
	}

	if len(unprotected) > 0 {
		sort.Slice(unprotected, func(i, j int) bool {
			if unprotected[i].file != unprotected[j].file {
				return unprotected[i].file < unprotected[j].file
			}
			return unprotected[i].line < unprotected[j].line
		})
		var sb strings.Builder
		sb.WriteString("以下写路由未挂权限中间件（仅有路径预检兜底 → 低权角色可能获得细粒度写能力）：\n")
		for _, r := range unprotected {
			sb.WriteString("  " + r.file + ":" + strconv.Itoa(r.line) + "  " +
				r.method + " " + r.path + "   (接收者: " + r.recv + ")\n")
		}
		sb.WriteString("\n修复方式：路由参数里加 middleware.RequirePermission(\"<resource>\", \"<action>\")。\n")
		sb.WriteString("如确属公开面或已在别处鉴权，请加入 writeRouteExemptions 并写明理由。")
		t.Error(sb.String())
	}

	// 白名单防腐化：每条都必须命中真实路由，且写明理由。
	matched := make(map[string]bool, len(writeRouteExemptions))
	for _, r := range routes {
		key := r.file + " " + r.method + " " + r.path
		if _, ok := writeRouteExemptions[key]; ok {
			matched[key] = true
		}
	}
	var stale []string
	for key, reason := range writeRouteExemptions {
		if strings.TrimSpace(reason) == "" {
			stale = append(stale, key+" —— 缺少豁免理由")
		}
		if !matched[key] {
			stale = append(stale, key+" —— 未命中任何已注册路由（路由改名/删除后请同步白名单）")
		}
	}
	if len(stale) > 0 {
		sort.Strings(stale)
		t.Errorf("writeRouteExemptions 存在失效条目：\n  %s", strings.Join(stale, "\n  "))
	}
}

// scanRouteRegistrations 解析路由注册源码，返回所有写方法路由调用。
func scanRouteRegistrations(t *testing.T) []scannedRoute {
	t.Helper()

	var files []string
	for _, pattern := range []string{"*.go", "../handlers/*/*.go"} {
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

	writeMethods := map[string]bool{"POST": true, "PUT": true, "PATCH": true, "DELETE": true}

	var out []scannedRoute
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

		// 第一遍：收集「已挂权限中间件的分组变量」
		protectedGroups := map[string]bool{}
		ast.Inspect(f, func(n ast.Node) bool {
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
				if !ok || !isSelectorCall(call, "Group") {
					return true
				}
				if hasPermissionArg(call.Args) {
					protectedGroups[ident.Name] = true
				}
			case *ast.ExprStmt:
				call, ok := node.X.(*ast.CallExpr)
				if !ok || !isSelectorCall(call, "Use") {
					return true
				}
				sel, _ := call.Fun.(*ast.SelectorExpr)
				recv, ok := sel.X.(*ast.Ident)
				if ok && hasPermissionArg(call.Args) {
					protectedGroups[recv.Name] = true
				}
			}
			return true
		})

		// 第二遍：收集写路由调用
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || !writeMethods[sel.Sel.Name] {
				return true
			}
			recvIdent, ok := sel.X.(*ast.Ident)
			recv := "?"
			if ok {
				recv = recvIdent.Name
			}
			path := ""
			if len(call.Args) > 0 {
				var err error
				if path, err = strconv.Unquote(exprString(call.Args[0])); err != nil {
					path = exprString(call.Args[0])
				}
			}
			hasPerm := hasPermissionArg(call.Args) || protectedGroups[recv]
			out = append(out, scannedRoute{
				file:    rel,
				line:    fset.Position(call.Pos()).Line,
				method:  sel.Sel.Name,
				path:    path,
				recv:    recv,
				hasPerm: hasPerm,
			})
			return true
		})
	}
	return out
}

// hasPermissionArg 判断参数列表中是否出现 RequirePermission 调用。
func hasPermissionArg(args []ast.Expr) bool {
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
			return true
		}
	}
	return false
}

// isSelectorCall 判断调用是否为 x.<name>(...) 形式。
func isSelectorCall(call *ast.CallExpr, name string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == name
}

// exprString 返回表达式的源码近似文本（用于无法 Unquote 的常量拼接场景）。
func exprString(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.BasicLit:
		return v.Value
	case *ast.Ident:
		return v.Name
	case *ast.BinaryExpr:
		return exprString(v.X) + exprString(v.Y)
	case *ast.SelectorExpr:
		return exprString(v.X) + "." + v.Sel.Name
	}
	return "<expr>"
}
