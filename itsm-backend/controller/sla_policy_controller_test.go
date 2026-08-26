package controller

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSLAPolicyController_NoSnakeCaseQuery 锁住 Bug 6 修复。
// sla_policy_controller 必须使用 camelCase query 参数；任何 snake_case ctx.Query 都属于回归。
// 修复前：?ticket_type / ?customer_tier / ?start_date / ?end_date（snake_case）
// 修复后：?ticketType / ?customerTier / ?startDate / ?endDate（camelCase）
func TestSLAPolicyController_NoSnakeCaseQuery(t *testing.T) {
	src := readControllerSource(t, "sla_policy_controller.go")
	for _, bad := range []string{"ticket_type", "customer_tier", "start_date", "end_date"} {
		assert.False(t, strings.Contains(src, `Query("`+bad+`")`),
			"Bug 6 验收：sla_policy_controller 不应再 ctx.Query(%q)", bad)
	}
	// 当前 controller 实际使用的 query：ticketType / priority / customerTier。
	// 凡是后续新增的 SLA 相关 query，都必须继续走 camelCase，由这条断言兜底。
	for _, good := range []string{"ticketType", "customerTier"} {
		assert.True(t, strings.Contains(src, `Query("`+good+`")`),
			"Bug 6 验收：sla_policy_controller 必须使用 ctx.Query(%q)", good)
	}
}

// readControllerSource 读取同包内 controller 源文件，返回全文字符串。
// 与 grep 验收形成双重保护：CI 既跑静态检查也跑单测，任一回退都会被立刻发现。
// filename 必须是 hard-coded 控制器文件名，函数强制 base-dir 检查以避免任何路径漂移。
func readControllerSource(t *testing.T, name string) string {
	t.Helper()
	// 拒绝任何带路径分隔符或 .. 的 filename，确保不会读出 controller 目录。
	require.NotContains(t, name, "/", "filename 必须是 plain basename")
	require.NotContains(t, name, "..", "filename 拒绝 .. 段")
	require.NotContains(t, name, "\\", "filename 拒绝 Windows 分隔符")

	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	base := filepath.Dir(file)
	target := filepath.Join(base, name)
	// 规范化后必须仍在 base 目录内。
	cleanBase := filepath.Clean(base) + string(os.PathSeparator)
	cleanTarget := filepath.Clean(target)
	require.True(t, strings.HasPrefix(cleanTarget, cleanBase),
		"resolved path %q escaped base dir %q", cleanTarget, cleanBase)

	src, err := os.ReadFile(cleanTarget)
	require.NoError(t, err)
	return string(src)
}