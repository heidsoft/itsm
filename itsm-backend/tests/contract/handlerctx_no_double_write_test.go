package contract

// 同源响应双写收口棘轮（ratchet）。
//
// handlerctx.ResolveTenantID / ResolveActorID / ResolveResourceIDAndTenant 等 helper
// 在缺租户上下文、缺用户上下文或资源归属校验失败时，已经通过
// middleware.AbortIfTenantError 写入一份「单一权威」401/403/500 响应并 c.Abort()。
// 调用方在 !ok 分支**只**应该 return，再写一份 common.Fail(...) 就是把第二个 JSON
// 文档塞进同一响应体：客户端解析失败、Go runtime 记 superfluous WriteHeader 警告。
//
// AGENTS.md「能力状态与失败语义」要求 Controller 必须稳定映射 401/403/404/409/500，
// 不允许把同一失败写两遍，也不允许「同一请求下写多份 envelope」导致响应体拼接。
//
// 本文件锁住这一契约：
//   - 禁止在 handlerctx.Resolve* 调用之后的 `if !ok { ... }` 分支内再调用
//     common.Fail* / common.RespondError / common.NotFound* / common.InternalError*
//     / common.FailWithErr / c.JSON / c.AbortWithStatusJSON 等写响应 helper；
//   - 禁止在 handlerctx.Resolve* 调用之后的 !ok 分支内缺失 return，
//     否则 ok=false 会继续走到后续 service 调用、写第二份响应或在 nil 上 panic；
//   - 双向棘轮：基线 24 处固定，新增双写点或「漏 return」都会红。
//
// 跑测命令：cd itsm-backend && go test ./tests/contract/... -run TestHandlerCtxNoDoubleWrite
// 重新采集基线：HANDLERCTX_NO_DOUBLE_WRITE_DUMP=1 go test ./tests/contract/... -run TestHandlerCtxNoDoubleWrite -v

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"
)

// handlerCtxScanFiles 是 E4-46 收敛涉及的 6 个 handler 文件清单（路径相对 itsm-backend）。
// 任何新增 handler 文件调用 handlerctx.Resolve* 后写出双写响应，本测试都不会覆盖；
// 那是新增 handler 的代码评审责任，本文件不替代评审。
var handlerCtxScanFiles = []string{
	"handlers/ticket_type/handler.go",
	"handlers/ai/handler.go",
	"handlers/operations/handler.go",
	"handlers/ticket_rating/handler.go",
	"handlers/timer/handler.go",
	"handlers/skill/handler.go",
}

// handlerCtxDoubleWriteBaseline 是 2026-10-04 由本文件扫描器实测的「双写响应点位」
// 清单（file|func|line|name）。所有 24 处都已收口：调用方只保留 `if !ok { return }`，
// 删掉委托给 middleware 的 common.Fail / 二次 envelope。
//
// 2026-10-04 E4-46 收口以下文件全集：
//   - handlers/ticket_type/handler.go 9 处：CreateTicketType/UpdateTicketType/
//     GetTicketType/ListTicketTypes/DeleteTicketType/setStatus(EnableTicketType/
//     DisableTicketType 共用)/CloneTicketType/RestoreTicketType/InstallPreset
//   - handlers/ai/handler.go 4 处：GetDeepAnalytics/GetTrendPrediction/SaveFeedback/
//     GetEvaluation
//   - handlers/operations/handler.go 4 处：List/Get/mutate(Replay/Cancel 共用)/
//     bulk(BulkReplay/BulkCancel 共用)
//   - handlers/ticket_rating/handler.go 3 处：SubmitRating/GetRating/GetRatingStats
//   - handlers/timer/handler.go 3 处：List/Get/Stats
//   - handlers/skill/handler.go 1 处：tenantID 注入 input map 的前置解析
//
// 因此本基线现在是空集——任何再回到 `if !ok { common.Fail(...) }` 形态的 handler 都会
// 在本测试里失败。
var handlerCtxDoubleWriteBaseline = map[string]int{}

// handlerCtxNoReturnBaseline 是 2026-10-04 由本文件扫描器实测的「handlerctx.Resolve*
// 之后缺少 return 提前退出的早期分支」点位清单（file|func|line）。所有 24 处都已
// 在 `if !ok` 分支里 return；这是 E4-46 与此基线同步收口的另一面。
//
// 因此本基线现在是空集——handlerctx.Resolve* 之后 `if !ok` 分支不 return（继续执行
// 后续 service 调用）属于把 ok=false 当成成功路径继续走，会写第二份响应或在 nil
// 上 panic。
var handlerCtxNoReturnBaseline = map[string]int{}

// forbiddenResponseCallNames 是「handlerctx.Resolve* 之后的 !ok 分支内禁止调用」的
// 写响应 helper 集合（function 名）。它们与 AST 选择器 `SelectorExpr.Sel.Name`
// 严格匹配，避免误伤其他包同名 helper。
//
// 列入黑名单的依据是 AGENTS.md「能力状态与失败语义」要求的「Controller 必须稳定
// 映射 validation / unauthorized / forbidden / not-found / conflict / unavailable
// / internal error；禁止把所有错误返回 500 或泄漏原始 SQL/provider 错误」，以及
// common.RespondError / common.FailWithErr 已经把 err 净化后再 envelope 的事实。
var forbiddenResponseCallNames = []string{
	"Fail",
	"FailWithErr",
	"FailWithData",
	"ParamError",
	"ParamErrorWithErr",
	"ValidationErrorResponse",
	"AuthFailed",
	"Forbidden",
	"NotFound",
	"NotFoundWithErr",
	"InternalError",
	"InternalErrorWithErr",
	"RespondError",
	"RespondBusinessError",
}

// forbiddenGinResponseCallNames 是「handlerctx.Resolve* 之后的 !ok 分支内禁止调用」
// 的 gin 自身写响应 helper 集合。它们绕过 common.* 的 envelope 契约，直接拼 JSON。
var forbiddenGinResponseCallNames = []string{
	"JSON",
	"AbortWithStatusJSON",
	"AbortWithStatus",
	"Render",
}

// handlerCtxViolation 描述扫描器命中的一处违规点位。
type handlerCtxViolation struct {
	file string
	fn   string
	line int
	name string // 双写时是 helper 名；no-return 时是「分支体描述」
	kind string // "double-write" / "no-return"
}

// scanHandlerCtxViolations 扫描 handlerCtxScanFiles，定位两类违规：
//
//  1. 双写响应：handlerctx.Resolve* 调用之后的 `if !ok` 分支体内出现写响应 helper，
//     或者分支体多于一条语句；
//  2. 漏 return：handlerctx.Resolve* 调用之后的 `if !ok` 分支体首语句不是 return。
//
// 6 个文件 24 处都是同一形态（`<name>, ok := handlerctx.Resolve*(c)` 紧接
// `if !ok { return }`），所以扫描器只看 AssignmentStmt 直接前导的 IfStmt，不进入
// 嵌套闭包；其它场景（如 helper 函数返回 ok 之后再判断）属于另一类契约，留给
// 评审与日后可能的扩展。
func scanHandlerCtxViolations(t *testing.T) []handlerCtxViolation {
	t.Helper()

	var out []handlerCtxViolation
	fset := token.NewFileSet()
	for _, rel := range handlerCtxScanFiles {
		path := "../.." + "/" + rel
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", rel, err)
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			stmts := fn.Body.List
			for i := 0; i < len(stmts)-1; i++ {
				assign, ok := stmts[i].(*ast.AssignStmt)
				if !ok || len(assign.Rhs) != 1 {
					continue
				}
				if !isHandlerCtxResolveCall(assign.Rhs[0]) {
					continue
				}
				ifStmt, ok := stmts[i+1].(*ast.IfStmt)
				if !ok {
					continue
				}
				if !isNotOkCond(ifStmt.Cond) {
					continue
				}
				body := ifStmt.Body
				if body == nil {
					out = append(out, handlerCtxViolation{
						file: rel, fn: fn.Name.Name, line: ifLine(fset, ifStmt),
						name: "if !ok 分支体缺失",
						kind: "no-return",
					})
					continue
				}
				if len(body.List) == 0 {
					out = append(out, handlerCtxViolation{
						file: rel, fn: fn.Name.Name, line: ifLine(fset, ifStmt),
						name: "if !ok 分支体为空",
						kind: "no-return",
					})
					continue
				}

				// 分支体首语句必须 return，否则判为 no-return。
				first := body.List[0]
				if _, ok := first.(*ast.ReturnStmt); !ok {
					out = append(out, handlerCtxViolation{
						file: rel, fn: fn.Name.Name, line: ifLine(fset, ifStmt),
						name: "if !ok 分支首语句不是 return: " + stmtLabel(first),
						kind: "no-return",
					})
				}

				// 整个分支体（包含嵌套调用）都不能调写响应 helper。
				for _, sub := range body.List {
					ast.Inspect(sub, func(n ast.Node) bool {
						call, ok := n.(*ast.CallExpr)
						if !ok {
							return true
						}
						name := callName(call)
						if name == "" {
							return true
						}
						if isForbiddenResponseCall(name) {
							out = append(out, handlerCtxViolation{
								file: rel, fn: fn.Name.Name, line: ifLine(fset, ifStmt),
								name: name,
								kind: "double-write",
							})
						}
						return true
					})
				}

				// 分支体多于一条语句属于可疑：返回后面追加了别的语句。
				if len(body.List) > 1 {
					out = append(out, handlerCtxViolation{
						file: rel, fn: fn.Name.Name, line: ifLine(fset, ifStmt),
						name: "分支体多于一条语句（疑似漏写 else 或并列）",
						kind: "double-write",
					})
				}
			}
		}
	}
	return out
}

// ifLine 把 IfStmt 的起始行（去掉 package 行的偏移后）作为定位锚点。
func ifLine(fset *token.FileSet, stmt ast.Node) int {
	return fset.Position(stmt.Pos()).Line
}

// stmtLabel 把 AST 节点压缩成一段人类可读的字符串，便于错误信息定位。
func stmtLabel(stmt ast.Stmt) string {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		if call, ok := s.X.(*ast.CallExpr); ok {
			if name := callName(call); name != "" {
				return "ExprStmt(" + name + ")"
			}
			return "ExprStmt(<call>)"
		}
		return "ExprStmt"
	case *ast.ReturnStmt:
		return "ReturnStmt"
	case *ast.IfStmt:
		return "IfStmt"
	case *ast.BlockStmt:
		return fmt.Sprintf("BlockStmt(%d)", len(s.List))
	case *ast.AssignStmt:
		return "AssignStmt"
	case *ast.DeclStmt:
		return "DeclStmt"
	default:
		return "Stmt"
	}
}

// callName 从 CallExpr 还原「common.Fail」或「c.JSON」这种选择器形式的名字。
// 若调用不是 selector call（例如直接函数名），返回空字符串让调用方跳过。
func callName(call *ast.CallExpr) string {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	return sel.Sel.Name
}

// isForbiddenResponseCall 判定 helper 名是否属于「写响应 / 二次 envelope」黑名单。
// 列表以项目当前 common helper 命名收敛为准；common 包内若新增 Fail* 同形 helper
// 而未列入本表，不会被本测试覆盖，那是新增 helper 的评审责任。
func isForbiddenResponseCall(name string) bool {
	for _, c := range forbiddenResponseCallNames {
		if name == c {
			return true
		}
	}
	for _, c := range forbiddenGinResponseCallNames {
		if name == c {
			return true
		}
	}
	return false
}

// isHandlerCtxResolveCall 判定表达式是否是 handlerctx.Resolve*(c) 形式。
// 不区分具体的 ResolveXxx 名称，因为 helper 家族会随业务增长。
func isHandlerCtxResolveCall(expr ast.Expr) bool {
	sel, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	pkgSel, ok := sel.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkgIdent, ok := pkgSel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return pkgIdent.Name == "handlerctx" && strings.HasPrefix(pkgSel.Sel.Name, "Resolve")
}

// isNotOkCond 判定表达式是否是 `!ok` 形式。
func isNotOkCond(expr ast.Expr) bool {
	un, ok := expr.(*ast.UnaryExpr)
	if !ok || un.Op != token.NOT {
		return false
	}
	id, ok := un.X.(*ast.Ident)
	if !ok {
		return false
	}
	return id.Name == "ok"
}

// TestHandlerCtxNoDoubleWrite 禁止 handlerctx.Resolve* 调用之后的 !ok 分支内
// 再出现写响应 helper / 多于一条语句。
func TestHandlerCtxNoDoubleWrite(t *testing.T) {
	all := scanHandlerCtxViolations(t)
	if t.Failed() {
		return
	}

	measured := map[string]int{}
	for _, site := range all {
		if site.kind != "double-write" {
			continue
		}
		key := site.file + "|" + site.fn + "|L" + itoa(site.line) + "|" + site.name
		measured[key]++
	}

	if os.Getenv("HANDLERCTX_NO_DOUBLE_WRITE_DUMP") == "1" {
		keys := sortedKeys(measured)
		t.Logf("扫描器实测 %d 处双写响应点位：", len(measured))
		for _, k := range keys {
			t.Logf("  %s", k)
		}
		return
	}

	var added, stale []string
	allowed := map[string]struct{}{}
	for k := range handlerCtxDoubleWriteBaseline {
		allowed[k] = struct{}{}
	}
	for k := range measured {
		if _, ok := allowed[k]; !ok {
			added = append(added, k)
		}
	}
	for k := range allowed {
		if _, ok := measured[k]; !ok {
			stale = append(stale, k)
		}
	}

	sort.Strings(added)
	sort.Strings(stale)
	if len(added) > 0 {
		t.Errorf(
			"handlerctx.Resolve* 之后的 !ok 分支又出现了写响应调用或多于一条语句，会产生同源响应双写。\n"+
				"  唯一合法的形态是：if !ok { return }（helper 已通过 middleware.AbortIfTenantError 写过 401/403/500）\n"+
				"  命中清单（%d）：\n    %s",
			len(added), strings.Join(added, "\n    "),
		)
	}
	if len(stale) > 0 {
		t.Errorf(
			"基线条目已被收口，请同步从 handlerCtxDoubleWriteBaseline 删除（这是收敛信号）：\n    %s",
			strings.Join(stale, "\n    "),
		)
	}
}

// TestHandlerCtxResolveMustHaveReturn 校验 handlerctx.Resolve* 调用之后的 !ok 分支
// 必须包含 return，禁止把 ok=false 当成功继续走后续 service 调用。
func TestHandlerCtxResolveMustHaveReturn(t *testing.T) {
	all := scanHandlerCtxViolations(t)
	if t.Failed() {
		return
	}

	measured := map[string]int{}
	for _, site := range all {
		if site.kind != "no-return" {
			continue
		}
		key := site.file + "|" + site.fn + "|L" + itoa(site.line)
		measured[key]++
	}

	var added, stale []string
	allowed := map[string]struct{}{}
	for k := range handlerCtxNoReturnBaseline {
		allowed[k] = struct{}{}
	}
	for k := range measured {
		if _, ok := allowed[k]; !ok {
			added = append(added, k)
		}
	}
	for k := range allowed {
		if _, ok := measured[k]; !ok {
			stale = append(stale, k)
		}
	}

	sort.Strings(added)
	sort.Strings(stale)
	if len(added) > 0 {
		t.Errorf(
			"handlerctx.Resolve* 之后的 !ok 分支必须 return，否则 ok=false 会继续走到后续 service 调用。\n"+
				"  命中清单（%d）：\n    %s",
			len(added), strings.Join(added, "\n    "),
		)
	}
	if len(stale) > 0 {
		t.Errorf(
			"基线条目已被收口，请同步从 handlerCtxNoReturnBaseline 删除：\n    %s",
			strings.Join(stale, "\n    "),
		)
	}
}

// sortedKeys 把 map[string]int 的 key 排序后返回，便于稳定日志输出。
func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// itoa 是 strconv.Itoa 的局部别名，避免在测试断言里重复 import 一大段。
func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}
