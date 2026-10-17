package router

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// 工单规则配置面的权限码归属。
//
// 该面曾有三套口径：菜单挂 ticket:read（人人有码）或 workflow:write（租管没有），
// 接口却要求 system_config:*。隔离栈实测 it_admin 看得到菜单、进页面 GET 直接
// 403 / code 2003；end_user 有 ticket:read 菜单码、接口同样拒绝；
// 反过来 security_admin 有 system_config:read 能读接口却没有任何菜单入口。
//
// 本测试锁死收敛后的单一口径：规则 CRUD 走 automation_rule / assignment_rule，
// 针对单个工单的分配动作走 ticket:assign，任何一条都不得再挂回 system_config:*
// （那份码集是 SMTP、密码策略、2FA 等租户级安全配置）。
func TestTicketRuleConfigRoutesPermissionCodes(t *testing.T) {
	want := map[string]string{
		"GET /api/v1/tickets/automation-rules":           "automation_rule:read",
		"POST /api/v1/tickets/automation-rules":          "automation_rule:write",
		"GET /api/v1/tickets/automation-rules/:id":       "automation_rule:read",
		"PUT /api/v1/tickets/automation-rules/:id":       "automation_rule:write",
		"DELETE /api/v1/tickets/automation-rules/:id":    "automation_rule:write",
		"POST /api/v1/tickets/automation-rules/:id/test": "automation_rule:write",
		"GET /api/v1/tickets/assignment-rules":           "assignment_rule:read",
		"POST /api/v1/tickets/assignment-rules":          "assignment_rule:write",
		"POST /api/v1/tickets/assignment-rules/test":     "assignment_rule:write",
		"GET /api/v1/tickets/assignment-rules/:id":       "assignment_rule:read",
		"PUT /api/v1/tickets/assignment-rules/:id":       "assignment_rule:write",
		"DELETE /api/v1/tickets/assignment-rules/:id":    "assignment_rule:write",
		"POST /api/v1/tickets/:id/auto-assign":           "ticket:assign",
		"GET /api/v1/tickets/assign-recommendations/:id": "ticket:assign",
	}

	got := scanTicketRoutePermissions(t)
	for key, code := range want {
		actual, ok := got[key]
		if !ok {
			t.Errorf("路由 %s 未声明 RequirePermission，DBOnly 下将对除 super_admin 外的所有角色不可达", key)
			continue
		}
		if actual != code {
			t.Errorf("路由 %s 的权限码是 %q，应为 %q", key, actual, code)
		}
	}

	for key, code := range got {
		if !strings.Contains(key, "/automation-rules") && !strings.Contains(key, "/assignment-rules") {
			continue
		}
		if strings.HasPrefix(code, "system_config:") || code == "ticket:read" || code == "workflow:write" {
			t.Errorf("规则配置路由 %s 回退到旧口径 %q：菜单码、读接口、写接口必须同属 automation_rule/assignment_rule 域", key, code)
		}
	}
}

// scanTicketRoutePermissions 解析 router/ticket_routes.go，返回 "METHOD /完整路径" → "resource:action"。
func scanTicketRoutePermissions(t *testing.T) map[string]string {
	t.Helper()

	src, err := os.ReadFile("ticket_routes.go")
	if err != nil {
		t.Fatalf("读取 ticket_routes.go 失败: %v", err)
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "ticket_routes.go", src, 0)
	if err != nil {
		t.Fatalf("解析 ticket_routes.go 失败: %v", err)
	}

	out := map[string]string{}
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) < 2 {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		method := map[string]string{"GET": "GET", "POST": "POST", "PUT": "PUT", "PATCH": "PATCH", "DELETE": "DELETE"}[sel.Sel.Name]
		if method == "" {
			return true
		}
		path, ok := call.Args[0].(*ast.BasicLit)
		if !ok || path.Kind != token.STRING {
			return true
		}
		perm := ""
		for _, arg := range call.Args[1:] {
			inner, ok := arg.(*ast.CallExpr)
			if !ok {
				continue
			}
			innerSel, ok := inner.Fun.(*ast.SelectorExpr)
			if !ok || innerSel.Sel.Name != "RequirePermission" || len(inner.Args) != 2 {
				continue
			}
			resource, ok1 := inner.Args[0].(*ast.BasicLit)
			action, ok2 := inner.Args[1].(*ast.BasicLit)
			if !ok1 || !ok2 {
				continue
			}
			perm = strings.Trim(resource.Value, `"`) + ":" + strings.Trim(action.Value, `"`)
		}
		if perm == "" {
			return true
		}
		rel := strings.Trim(strings.Trim(path.Value, `"`), "/")
		out[method+" /api/v1/tickets/"+rel] = perm
		return true
	})
	return out
}
