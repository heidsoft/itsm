package contract

// 分页缺省值收敛棘轮（ratchet）。
//
// 平台只允许一个分页解析所有者：common.GetPaginationFromQuery（缺省
// common.DefaultPageSize=20，只采纳 [1, MaxPageSize=100]，越界与非数字回落缺省，
// 页码下界 1）。handler 里自写 c.DefaultQuery("pageSize", "10") 会同时制造三件事：
// 自己一份缺省值、自己一份上界（或根本没有上界）、以及「信封回显的值」与
// 「真正下传给 Offset/Limit 的值」分家——非数字或 0 会让 Ent 的 Limit 被整个跳过，
// 于是一次写错分页参数的请求把该租户读穿。2026-10-04 在 GET /api/v1/incidents
// 与 GET /api/v1/incidents/alerts/active 实测命中并已收敛，见台账 E4-9b。
//
// 本棘轮不要求一次性清零，但禁止新增，也不允许基线悄悄增长。
// 收敛一条的做法：把该 handler 改成 pg := common.GetPaginationFromQuery(c)，
// 查询与信封都用 pg.Page / pg.PageSize，配套路由测试锁住缺省、越界、非数字、
// 非正页码与上界采纳，然后从基线删掉这一条。
//
// 判定形态（2026-10-04 扩展，台账 E4-47a）：handler 层只要把 "pageSize" 或 "size"
// 作为字面量实参传给任何被调函数，就算一处自建页长读取点，因此同时覆盖
//   - ctx.Query("pageSize") 与 ctx.DefaultQuery("pageSize", "20")
//   - 自定义 helper：queryIntParam(c, "pageSize", "size", 10）、parseSkillListPagination
//     内部的 c.Query("pageSize")、inputInt(in, "pageSize", 20)
//
// 此前只匹配 DefaultQuery("pageSize", 字面量)，于是「没有上界」「camelCase 与 size 双别名」
// 「手抄 (0,100]」这些形态既不在 9 处基线里、新增也不会转红（实测漏掉 8 处）。
// 结构体 json tag 与 gin.H 的键不是调用的直接实参，所以 `json:"pageSize"` 和
// gin.H{"pageSize": size} 不会被误计。
//
// 跑测命令：cd itsm-backend && go test ./tests/contract/ -run TestPageSizeSingleOwnerRatchet

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

// selfParsedPageSizeBaseline 是 2026-10-04 由扫描器实测的存量清单，
// 条目格式 <相对 itsm-backend 的文件路径>|<被调函数名>|<该处写死的缺省页长，
// 无法从实参看出时为 ?>；同一键有多处就重复多行，计数一并比对。
//
// 扩展判定形态后新增的 8 处（原先 DefaultQuery-only 扫描器看不见）。
// 台账 E4-47 已全量收敛：sla/skill/workbench（①-④）+ release/standard_change/
// knowledge/known_error/operations/timer/bpmn/workflow_template/approval（第二批）
// 共 14 处，全部改走 common.GetPaginationFromQuery(c)。
// 回归锁：handlers/sla/pagination_contract_test.go、handlers/skill/handler_test.go、
// handlers/workbench/handler_test.go、handlers/approval/routes.go 兜底路径。
// sla/skill/workbench 的 6 处（E4-47①–④）+ release/standard_change/knowledge/known_error/
// operations/timer/bpmn/workflow_template/approval 的 8 处（E4-47 第二批）已全部收敛并删除。
var selfParsedPageSizeBaseline = []string{
	"handlers/ai/skills.go|inputInt|20",
}

// pageSizeOwnerDirs 是被扫描的 HTTP 入口层。业务 service 层的兜底夹紧走
// common.ValidatePagination，不在本棘轮范围内（它读的就是同一组常量）。
var pageSizeOwnerDirs = []string{"handlers", "controller"}

type selfParsedSite struct {
	key   string
	line  int
	debug string
}

// backendRoot 是 itsm-backend 目录。本包的其它棘轮同样用 filepath.Glob("..","..",...)
// 定位仓库根，这里保持一致：跑测必须在 go test 的包目录下，不需要环境变量。
func backendRoot(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("定位 itsm-backend 根目录失败: %v", err)
	}
	return root
}

// TestPageSizeSingleOwnerRatchet 禁止 handler 再自建分页缺省值/上界解析。
func TestPageSizeSingleOwnerRatchet(t *testing.T) {
	root := backendRoot(t)

	var got []selfParsedSite
	for _, dir := range pageSizeOwnerDirs {
		got = append(got, scanSelfParsedPageSize(t, filepath.Join(root, dir))...)
	}

	count := func(entries []string) map[string]int {
		out := make(map[string]int, len(entries))
		for _, e := range entries {
			out[e]++
		}
		return out
	}

	gotKeys := make([]string, 0, len(got))
	for _, site := range got {
		gotKeys = append(gotKeys, site.key)
	}

	gotCounts, wantCounts := count(gotKeys), count(selfParsedPageSizeBaseline)

	var added []string
	for key, n := range gotCounts {
		if n > wantCounts[key] {
			added = append(added, key+"（现 "+strconv.Itoa(n)+" 处，基线 "+strconv.Itoa(wantCounts[key])+" 处）")
		}
	}
	var stale []string
	for key, n := range wantCounts {
		if gotCounts[key] < n {
			stale = append(stale, key+"（现 "+strconv.Itoa(gotCounts[key])+" 处，基线仍写 "+strconv.Itoa(n)+" 处，请收紧基线）")
		}
	}
	sort.Strings(added)
	sort.Strings(stale)

	if len(added) > 0 || len(stale) > 0 {
		var b strings.Builder
		if len(added) > 0 {
			b.WriteString("新增 handler 自建分页页长读取点，HTTP 入口必须走 common.GetPaginationFromQuery 单一所有者：\n  ")
			b.WriteString(strings.Join(added, "\n  "))
			b.WriteString("\n")
		}
		if len(stale) > 0 {
			b.WriteString("以下条目已收敛，请从 selfParsedPageSizeBaseline 删除：\n  ")
			b.WriteString(strings.Join(stale, "\n  "))
			b.WriteString("\n")
		}
		var sites []string
		for _, site := range got {
			sites = append(sites, site.debug)
		}
		sort.Strings(sites)
		b.WriteString("当前实测站点：\n  ")
		b.WriteString(strings.Join(sites, "\n  "))
		t.Errorf("%s", b.String())
	}
}

// scanSelfParsedPageSize 用 AST 找出 handler 层把 "pageSize"/"size" 当字面量实参
// 传下去的调用点。只看真正的调用表达式及其**直接**实参，因此：
//   - 注释掉的历史代码不会被误计；
//   - `json:"pageSize"` 结构体标签、gin.H{"pageSize": ...} 的键都不是直接实参，不会误计；
//   - 嵌套调用会被 ast.Inspect 单独访问，所以 strconv.Atoi(ctx.DefaultQuery("pageSize","20"))
//     只算一处，defaultString(c.Query("pageSize"), "20") 也只算一处。
func scanSelfParsedPageSize(t *testing.T, dir string) []selfParsedSite {
	t.Helper()

	var sites []selfParsedSite
	fset := token.NewFileSet()

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// controller/ 在某些分支上不存在，缺失目录不算违规。
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			t.Fatalf("解析 %s 失败: %v", path, parseErr)
		}

		rel, relErr := filepath.Rel(backendRoot(t), path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || !readsPageSizeLiteral(call) {
				return true
			}

			key := rel + "|" + calleeName(call.Fun) + "|" + literalDefault(call)
			line := fset.Position(call.Pos()).Line
			sites = append(sites, selfParsedSite{
				key:   key,
				line:  line,
				debug: rel + ":" + strconv.Itoa(line) + " (" + key + ")",
			})
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("扫描 %s 失败: %v", dir, err)
	}
	return sites
}

// pageSizeLiteralNames 是分页长度在读取点里出现过的字面量名。
// size 是 sla 域仍在接受的别名（AGENTS「禁止同时发送 pageSize 与 size」），
// 因此按读取点登记而不是按合法别名放行。
var pageSizeLiteralNames = map[string]bool{"pageSize": true, "size": true}

// readsPageSizeLiteral 判断该调用的直接实参里是否出现分页长度字面量。
func readsPageSizeLiteral(call *ast.CallExpr) bool {
	for _, arg := range call.Args {
		lit, ok := arg.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			continue
		}
		if pageSizeLiteralNames[strings.Trim(lit.Value, `"`)] {
			return true
		}
	}
	return false
}

// calleeName 取被调函数名，用于在基线里区分同一文件里的不同形态。
func calleeName(fun ast.Expr) string {
	switch f := fun.(type) {
	case *ast.SelectorExpr:
		return f.Sel.Name
	case *ast.Ident:
		return f.Name
	default:
		return "expr"
	}
}

// literalDefault 取该调用最后一个字面量实参作为写死的缺省页长。
// DefaultQuery("pageSize","20") 取到 20，queryIntParam(c,"pageSize","size",10) 取到 10，
// inputInt(in,"pageSize",20) 取到 20；c.Query("pageSize") 的缺省不在这次调用里，记 ?。
func literalDefault(call *ast.CallExpr) string {
	if len(call.Args) == 0 {
		return "?"
	}
	lit, ok := call.Args[len(call.Args)-1].(*ast.BasicLit)
	if !ok {
		return "?"
	}
	value := strings.Trim(lit.Value, `"`)
	if pageSizeLiteralNames[value] {
		return "?"
	}
	return value
}
