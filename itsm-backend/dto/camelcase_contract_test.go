package dto

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// snakeCaseKey 匹配 form/query tag 中下划线分隔的 key。
// 合法 camelCase key 不含下划线。
var snakeCaseKey = regexp.MustCompile(`[a-z]+_[a-z_]+`)

// TestAllRequestDTOsUseCamelCaseFormTags 扫描本包（dto/）下所有 DTO 文件中的
// struct field tag（form / query），禁止出现 snake_case key。
//
// 契约依据：AGENTS.md "API 契约单一事实来源" 与 "Snake_case 零新增规则"：
// 所有 HTTP/JSON 交互字段必须使用 camelCase，DTO form/query tag 不得新增下划线。
//
// 实现说明：
//  1. AST 解析避免在编译期破坏性变化（DTO 体量小且字段多）；
//  2. 直接检查原始 tag 字符串（包括可能的 `omitempty` 等修饰），不依赖反射 Bind；
//  3. 对外只暴露一个测试入口，任何字段违规都会让测试失败，并指出文件名 + 字段名。
func TestAllRequestDTOsUseCamelCaseFormTags(t *testing.T) {
	matches, err := filepath.Glob("*.go")
	require.NoError(t, err)

	var violations []string
	for _, file := range matches {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		if strings.HasSuffix(file, ".go.disabled") {
			continue
		}
		// mappers 只承载 Entity -> DTO 转换，但为安全起见仍扫描其 tag。
		// types / ontology 多为类型别名或领域结构，无 HTTP 入参；但同样扫描以防误用。

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		require.NoErrorf(t, err, "parse %s", file)

		ast.Inspect(f, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok || st.Fields == nil {
				return true
			}
			typeName := ts.Name.Name
			for _, field := range st.Fields.List {
				if field.Tag == nil {
					continue
				}
				tagLiteral := field.Tag.Value
				if !strings.Contains(tagLiteral, "form:") && !strings.Contains(tagLiteral, "query:") {
					continue
				}
				if len(field.Names) == 0 {
					continue
				}
				for _, name := range field.Names {
					if checkTagSnakeCase(tagLiteral) {
						violations = append(violations, file+":"+typeName+"."+name.Name+" => "+tagLiteral)
					}
				}
			}
			return true
		})
	}

	if len(violations) > 0 {
		t.Fatalf("发现 %d 个 DTO form/query tag 仍使用 snake_case：\n  %s\n\n" +
			"AGENTS.md 要求：HTTP/JSON 字段统一 camelCase，禁止 `form:\"x_y\"` / `query:\"x_y\"`。",
			len(violations), strings.Join(violations, "\n  "))
	}
}

// checkTagSnakeCase 检查单个字段的 tag 字符串是否含有 snake_case key。
// 返回 true 表示发现违规。
func checkTagSnakeCase(tagLiteral string) bool {
	for _, prefix := range []string{`form:"`, `query:"`} {
		idx := strings.Index(tagLiteral, prefix)
		if idx < 0 {
			continue
		}
		start := idx + len(prefix)
		end := strings.Index(tagLiteral[start:], `"`)
		if end < 0 {
			continue
		}
		key := tagLiteral[start : start+end]
		keyPart := strings.SplitN(key, ",", 2)[0]
		if keyPart == "" || keyPart == "-" {
			continue
		}
		if snakeCaseKey.MatchString(keyPart) {
			return true
		}
	}
	return false
}
