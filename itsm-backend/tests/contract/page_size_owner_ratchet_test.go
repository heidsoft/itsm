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
// 跑测命令：cd itsm-backend && go test ./tests/contract/... -run TestPageSizeOwner

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
// 条目格式 <相对 itsm-backend 的文件路径>|<自建缺省页长>；同一文件同一页长
// 有多处就重复多行，计数一并比对。
var selfParsedPageSizeBaseline = []string{
	"handlers/bpmn/monitoring.go|20",
	"handlers/bpmn/monitoring.go|20",
	"handlers/bpmn/workflow_template.go|20",
	"handlers/known_error/handler.go|20",
	"handlers/known_error/handler.go|20",
	"handlers/knowledge/handler.go|10",
	"handlers/operations/handler.go|20",
	"handlers/release/handler.go|10",
	"handlers/sla/handler.go|20",
	"handlers/standard_change/handler.go|20",
	"handlers/timer/handler.go|20",
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
			b.WriteString("新增 handler 自建分页缺省值，分页解析必须走 common.GetPaginationFromQuery 单一所有者：\n  ")
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

// scanSelfParsedPageSize 用 AST 找出 DefaultQuery("pageSize", <字面量>) 调用。
// 只看真正的调用表达式，因此注释与被注释掉的历史代码不会被误计；
// 参数不是字符串字面量（动态缺省）时同样报违规，因为缺省值必须来自 common。
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
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "DefaultQuery" || len(call.Args) < 2 {
				return true
			}
			if lit, ok := call.Args[0].(*ast.BasicLit); !ok || strings.Trim(lit.Value, `"`) != "pageSize" {
				return true
			}

			key := rel + "|?"
			if def, ok := call.Args[1].(*ast.BasicLit); ok && def.Kind == token.STRING {
				key = rel + "|" + strings.Trim(def.Value, `"`)
			}
			pos := fset.Position(call.Pos())
			sites = append(sites, selfParsedSite{
				key:   key,
				line:  pos.Line,
				debug: rel + ":" + strconv.Itoa(pos.Line),
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
