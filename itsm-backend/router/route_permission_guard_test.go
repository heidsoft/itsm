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
// 写路由不得只靠 ResourceActionMap 的粗粒度路径预检兜底：预检只确认请求进了受保护分组，
// 具体动作权限要由路由级中间件声明，否则同分组内的低权限角色也能触发写操作。
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
	"../handlers/dingtalk/handler.go POST /dingtalk/webhook/:instance_id": "钉钉事件回调，走钉钉签名校验",
	"../handlers/wecom/handler.go POST /wecom/webhook/:instance_id":       "企微事件回调，走企微签名校验",
	"bootstrap_routes.go POST /create-admin":                              "初始化引导：仅在系统未初始化时可用，凭一次性 bootstrap token",

	// —— 2. 身份自省面：只需已认证身份 + 归属校验 ——
	"common_system_routes.go POST /logout":        "登出，凭会话自身即可失效",
	"common_system_routes.go POST /switch-tenant": "切换至该用户已归属的租户，授权依据是租户成员关系而非资源动作",
	"router.go POST /ws/ticket":                   "颁发一次性 WebSocket 票据，凭已认证身份且仅限本人会话",

	// —— 3. SSO 登录发起与账号自助面（与 /logout、/switch-tenant 同模式）——
	// initiate/callback 是 OIDC 登录链路的前半段，会话尚未建立，
	// 授权依据是 issuer + state + PKCE + nonce 而非角色；自助面只操作调用者本人绑定。
	// 这四条豁免与 sso_routes.go 的注册一一对应：若守卫报「白名单存在失效条目」，
	// 说明对应路由被改名或未注册，应同步修订本清单，而不是删掉守卫。
	"sso_routes.go POST /auth/sso/initiate": "SSO 登录发起，会话未建立，凭 issuer + state + PKCE",
	"sso_routes.go POST /auth/sso/callback": "SSO 回调，凭 state + nonce 校验 IdP 断言",
	"sso_routes.go POST /sso/bind":          "绑定外部身份到本人账号，仅影响调用者自己",
	"sso_routes.go POST /sso/unbind":        "解绑外部身份，仅影响调用者自己",

	// —— 4. 自助面：改的是调用者自己的凭据与资料 ——
	// 用 user:write 会让端用户无法维护自己账号（该码是管理用户目录的语义）；
	// handler 内部 user_id 只取鉴权上下文，不接受请求体指定目标用户，
	// 且资料 DTO 白名单排除 role/username/active，不具备提权能力。
	"account_routes.go PUT /change-password": "自助改密，仅作用于调用者本人并校验旧密码",
	"account_routes.go PUT /profile":         "自助改资料，仅作用于调用者本人且字段白名单不含角色/状态/用户名",
}

// readRouteExemptions 白名单：明确接受「无行内权限中间件」的读路由。
//
// 键格式：`<相对路径> <METHOD> <路径>`。只允许四类，且必须写明理由：
//  1. 公开面（健康检查、bootstrap 状态、CSRF token、Swagger 文档）——
//     登录前或纯技术端点，不返回业务数据；
//  2. 身份自省面（/auth/me、/auth/session、/auth/profile、/auth/tenants）——
//     仅需已认证身份，授权依据是「调用者与自身/自身租户的归属关系」；
//  3. 外部 OAuth 回调（飞书）——凭 provider 侧的 state/code 校验；
//  4. 分组级已挂权限中间件但不在本扫描器可识别形式内的路由。
//
// 为什么读路由也要守：此前只有写路由受守卫覆盖，读路由无任何声明时
// 只能靠 RBAC 中间件的路径预检兜底，而预检推出的资源名可能不在权限词表
// （global-search 即是如此：推出的 global-search:read 无人持有，DBOnly 下
// 除 super_admin 外全员 403，功能实际不可用）。读路由同样可能返回跨域
// 敏感数据（全局搜索返回标题与描述正文），无声明即无授权依据。
var readRouteExemptions = map[string]string{
	// —— 1. 公开面：纯技术端点，不返回业务数据 ——
	"router.go GET /health":                                              "健康检查，不含业务数据",
	"router.go GET /healthz":                                             "健康检查别名，K8s/常见探针路径",
	"router.go GET /readyz":                                              "就绪探针，只回初始化布尔状态，供编排层摘流",
	"router.go GET /version":                                             "版本号与构建渠道，前端版本提示用",
	"router.go GET /metrics":                                             "Prometheus 抓取面，仅 AuthMiddleware 不含业务数据；prod 未映射宿主端口",
	"router.go GET /csrf-token":                                          "CSRF token 下发，本身就是鉴权凭据的载体",
	"router.go GET /swagger/*any":                                        "Swagger 文档面，仅描述 API 形状",
	"bootstrap_routes.go GET /status":                                    "初始化状态探测，只回布尔状态",
	"../handlers/bpmn/dashboard.go GET /metrics":                         "BPMN 仪表盘读面，路由由 SetupRoutes 统一挂 bpmn:read",
	"../handlers/bpmn/dashboard.go GET /process/:key/metrics":            "同上",
	"../handlers/bpmn/dashboard.go GET /sla/violations":                  "同上",
	"../handlers/bpmn/dashboard.go GET /sla/compliance":                  "同上",
	"../handlers/bpmn/dashboard.go GET /tenant/stats":                    "同上",
	"../handlers/bpmn/dashboard.go GET /bottlenecks":                     "同上",
	"../handlers/bpmn/monitoring.go GET /metrics":                        "BPMN 监控读面，路由由 SetupRoutes 统一挂 bpmn:read",
	"../handlers/bpmn/monitoring.go GET /metrics/:processKey":            "同上",
	"../handlers/bpmn/monitoring.go GET /instances/:instanceId/status":   "同上",
	"../handlers/bpmn/monitoring.go GET /instances/status":               "同上",
	"../handlers/bpmn/monitoring.go GET /instances/:instanceId/timeline": "同上",
	"../handlers/bpmn/monitoring.go GET /performance":                    "同上",
	"../handlers/bpmn/monitoring.go GET /performance/alerts":             "同上",
	"../handlers/bpmn/monitoring.go GET /health":                         "同上",

	// —— 2. 身份自省面：只需已认证身份 + 归属校验 ——
	"common_system_routes.go GET /me":      "当前登录用户自身信息",
	"common_system_routes.go GET /session": "当前会话自身信息",
	"common_system_routes.go GET /tenants": "当前用户已归属的租户列表",
	"common_system_routes.go GET /menus":   "当前用户可见菜单（按其权限码过滤）",
	"common_system_routes.go GET /profile": "当前用户自身资料",
	"sso_routes.go GET /sso/identities":    "列出调用者本人已绑定的外部身份，仅影响自己",

	// —— 3. 外部 OAuth 回调 / 登录前发现面 ——
	"feishu_routes.go GET /feishu/oauth/callback/:instance_id": "飞书 OAuth 回调，凭 provider 侧 state/code 校验",
	"sso_routes.go GET /auth/sso/providers":                    "登录页拉取已启用的 SSO 提供方，会话尚未建立",

	// —— 4. 其他 ——
	// 能力发现面：hasPermission 在 RBAC 主链路里显式放行（middleware/rbac.go
	// 的 `/api/v1/capabilities` 分支），理由是 UI 需要先知道「哪些能力可用」
	// 才能决定渲染什么——给它挂权限码会形成循环依赖。返回的是成熟度与就绪
	// 元数据，allowedActions 已按角色裁剪。
	"router.go GET /capabilities":                       "能力发现面，RBAC 主链路显式放行，仅回成熟度/就绪元数据",
	"router.go GET /ws/notifications":                   "WebSocket 连接端点，凭一次性短期票据认证（/ws/ticket 颁发）",
	"msp_routes.go GET /status":                         "MSP 状态探测，模块未装配时返回能力声明",
	"../handlers/marketplace/handler.go GET /items":     "市场目录读面，分组级已挂 marketplace:read",
	"../handlers/marketplace/handler.go GET /items/:id": "同上",
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

// TestReadRoutesRequirePermission 守卫：所有读路由必须显式挂权限中间件。
//
// 判定口径与 TestWriteRoutesRequirePermission 完全一致（同一套扫描器），
// 区别只在方法集与白名单。读路由此前完全不在守卫覆盖内，导致两类问题：
//   - 无声明 → RBAC 路径预检推出的资源名可能不在权限词表（global-search
//     推出 global-search:read，无人持有，DBOnly 下除 super_admin 外全员 403，
//     功能实际不可用）；
//   - 有声明但声明得宽 → 跨域聚合读面（全局搜索、工作台）用一枚读码
//     换来全域内容，越权读不可见。
func TestReadRoutesRequirePermission(t *testing.T) {
	routes := scanRoutesWithMethods(t, map[string]bool{"GET": true, "HEAD": true})

	var unprotected []scannedRoute
	for _, r := range routes {
		if r.hasPerm {
			continue
		}
		if _, ok := readRouteExemptions[r.file+" "+r.method+" "+r.path]; ok {
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
		sb.WriteString("以下读路由未挂权限中间件（仅有路径预检兜底 → 低权角色可能读到跨域数据）：\n")
		for _, r := range unprotected {
			sb.WriteString("  " + r.file + ":" + strconv.Itoa(r.line) + "  " +
				r.method + " " + r.path + "   (接收者: " + r.recv + ")\n")
		}
		sb.WriteString("\n修复方式：路由参数里加 middleware.RequirePermission(\"<resource>\", \"read\")。\n")
		sb.WriteString("如确属公开面或仅需已认证身份，请加入 readRouteExemptions 并写明理由。")
		t.Error(sb.String())
	}

	// 白名单防腐化：每条都必须命中真实路由，且写明理由。
	matched := make(map[string]bool, len(readRouteExemptions))
	for _, r := range routes {
		key := r.file + " " + r.method + " " + r.path
		if _, ok := readRouteExemptions[key]; ok {
			matched[key] = true
		}
	}
	var stale []string
	for key, reason := range readRouteExemptions {
		if strings.TrimSpace(reason) == "" {
			stale = append(stale, key+" —— 缺少豁免理由")
		}
		if !matched[key] {
			stale = append(stale, key+" —— 未命中任何已注册路由（路由改名/删除后请同步白名单）")
		}
	}
	if len(stale) > 0 {
		sort.Strings(stale)
		t.Errorf("readRouteExemptions 存在失效条目：\n  %s", strings.Join(stale, "\n  "))
	}
}

// scanRouteRegistrations 解析路由注册源码，返回所有写方法路由调用。
func scanRouteRegistrations(t *testing.T) []scannedRoute {
	t.Helper()
	return scanRoutesWithMethods(t, map[string]bool{"POST": true, "PUT": true, "PATCH": true, "DELETE": true})
}

// scanRoutesWithMethods 解析路由注册源码，返回指定方法集的路由调用。
//
// 方法集参数化：写路由与读路由各用一个守卫，但共用同一套 AST 判定逻辑
// （行内 RequirePermission / 分组级权限中间件）。两套判定必须同源——
// 分叉的扫描器会各自漂移，最终两边的「已保护」定义对不上。
func scanRoutesWithMethods(t *testing.T, methods map[string]bool) []scannedRoute {
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
			if !ok || !methods[sel.Sel.Name] {
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
