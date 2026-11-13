package bootstrap

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNewApplicationWiresGlobalZapLogger 锁定组合根把配置好的 logger 设为 zap 全局。
// 缺这一行时，auth/rbac/tenant/csrf/acl 等直接用 zap.S() 的安全拒绝路径会打到
// zap 默认的 no-op 全局日志器，生产一条都不落盘（注入式 sugar 与 HTTP 访问日志不受影响）。
func TestNewApplicationWiresGlobalZapLogger(t *testing.T) {
	source, err := os.ReadFile("app.go")
	require.NoError(t, err)
	assembly := functionSource(t, source, "NewApplication")
	require.Contains(t, assembly, "zap.ReplaceGlobals(logger)",
		"NewApplication 必须把 initLogger 的结果设为 zap 全局，否则 zap.S() 安全日志静默")
}
