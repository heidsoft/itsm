package router

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"itsm-backend/pkg/seeder"
)

// catalogDeadCodeBaseline 反向死码棘轮基线：catalog 定义但 router RequirePermission 声明与
// precheck（rbac_precheck_gen.go 生成条目 + precheck_fallback.go 族级回退）都未引用的权限码数。
// 正向守卫 TestRoutePermissionCodesAreDefined 已锁「路由声明 ⊆ catalog」；本守卫锁反向：
// 存量死码（alert/alerts 拼写漂移、ticket_type 细分、msp 写操作等 21 个，2026-10-11 审计）
// 逐条裁决「退役或补路由引用」后应只降不升；新增死码立即红。
const catalogDeadCodeBaseline = 21

// TestCatalogDeadCodesAdvisory 反向死码守卫（棘轮 advisory）：死码数超过基线即失败，
// 等于/低于基线时仅输出清单供持续审计。口径不含 action 别名（view→read 等只影响
// 菜单侧展示匹配，不影响路由可达性）。
func TestCatalogDeadCodesAdvisory(t *testing.T) {
	codes := seeder.AllDefinedPermissionCodes()
	if len(codes) < 100 {
		t.Fatalf("权限码权威源仅 %d 个码，疑似提取逻辑失效", len(codes))
	}
	catalog := make(map[string]bool, len(codes))
	for _, c := range codes {
		catalog[c] = true
	}

	referenced := map[string]bool{}
	// 1) router + handlers/common 的 RequirePermission/RequirePermissionAny 直接声明
	for _, pattern := range []string{"*.go", "../handlers/common/*.go"} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatalf("glob %s: %v", pattern, err)
		}
		for _, m := range matches {
			if strings.HasSuffix(m, "_test.go") {
				continue
			}
			src, err := os.ReadFile(m)
			if err != nil {
				t.Fatalf("read %s: %v", m, err)
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, m, src, 0)
			if err != nil {
				t.Fatalf("parse %s: %v", m, err)
			}
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
				if len(call.Args) >= 2 {
					r := basicLitString(call.Args[0])
					a := basicLitString(call.Args[1])
					if r != "" && a != "" {
						referenced[r+":"+a] = true
					}
				}
				return true
			})
		}
	}

	// 2) precheck 两文件（generated + 手写族级回退，格式稳定且有 TestPrecheckMapIsFresh
	//    / TestRoutePrecheckAlignment 守卫锁住，正则提取安全）
	precheckRe := regexp.MustCompile(`Resource:\s*"([a-z_]+)",\s*Action:\s*"([a-z_]+)"`)
	for _, f := range []string{"../middleware/rbac_precheck_gen.go", "../middleware/precheck_fallback.go"} {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		for _, match := range precheckRe.FindAllStringSubmatch(string(src), -1) {
			referenced[match[1]+":"+match[2]] = true
		}
	}

	dead := make([]string, 0)
	for code := range catalog {
		if !referenced[code] {
			dead = append(dead, code)
		}
	}
	sort.Strings(dead)

	if len(dead) > catalogDeadCodeBaseline {
		t.Errorf("catalog 死码数 %d > 基线 %d，新增死码必须处理（退役或补路由引用）。清单：\n%s",
			len(dead), catalogDeadCodeBaseline, strings.Join(dead, "\n"))
		return
	}
	t.Logf("catalog 死码 %d 个（基线 %d，只降不升）：\n%s", len(dead), catalogDeadCodeBaseline, strings.Join(dead, "\n"))
}

func basicLitString(e ast.Expr) string {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	s, err := strconv.Unquote(lit.Value)
	if err != nil {
		return ""
	}
	return s
}
