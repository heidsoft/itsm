package bootstrap

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// functionSource 返回 app.go 中指定函数的源码文本，用于装配守卫。
func functionSource(t *testing.T, source []byte, funcName string) string {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "app.go", string(source), 0)
	require.NoError(t, err)

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != funcName || fn.Body == nil {
			continue
		}
		start := fset.Position(fn.Body.Lbrace).Offset
		end := fset.Position(fn.Body.Rbrace).Offset + 1
		return string(source[start:end])
	}
	t.Fatalf("function %q not found in app.go", funcName)
	return ""
}

// TestBackgroundTasksReuseInjectedSLAServices 锁定后台定时任务的依赖来源：
// startBackgroundTasks 只能使用 Application 上由 NewApplication 注入完成的实例。
// 私有副本会让 CheckSLAViolations 每轮命中 requireSLAStore 的 503 fail closed
// （见 service/sla_monitor_service_test.go 的 C4 回归），预警与升级通知则静默为 0。
func TestBackgroundTasksReuseInjectedSLAServices(t *testing.T) {
	source, err := os.ReadFile("app.go")
	require.NoError(t, err)
	worker := functionSource(t, source, "startBackgroundTasks")

	for _, forbidden := range []string{
		"service.NewSLAMonitorService(",
		"service.NewEscalationService(",
	} {
		require.NotContainsf(t, worker, forbidden,
			"后台任务不得自建未注入依赖的 %s", forbidden)
	}

	require.Contains(t, worker, "app.SLAMonitorService.CheckAllTenantsSLA(")
	require.Contains(t, worker, "app.EscalationService.ProcessEscalations(")
}

// TestSLAServicesAreFullyInjected 校验 NewApplication 对这两个实例的注入清单，
// 并把它们挂到 Application 上；缺任何一项都会让某条 SLA 能力变成假成功。
func TestSLAServicesAreFullyInjected(t *testing.T) {
	source, err := os.ReadFile("app.go")
	require.NoError(t, err)
	assembly := functionSource(t, source, "NewApplication")

	// slaStore: 违规检查读写 sla_states；notificationSvc: 违规通知；
	// alertService: 预警与告警触发计数。
	for _, required := range []string{
		"slaMonitorService.SetSLAStore(",
		"slaMonitorService.SetNotificationService(",
		"slaMonitorService.SetAlertService(",
		"escalationService.SetNotificationService(",
		"SLAMonitorService: slaMonitorService,",
		"EscalationService: escalationService,",
	} {
		require.Truef(t, strings.Contains(assembly, required),
			"NewApplication 缺少装配项 %q", required)
	}
}
