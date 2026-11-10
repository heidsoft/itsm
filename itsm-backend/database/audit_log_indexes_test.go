package database

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// L7 回归锁：audit_logs 的 Count + Order(created_at DESC) 必须有可用复合索引。
// 索引走 EnsureOperationalIndexes 的幂等 DDL（bootstrap 在 InitializeStorage
// 末尾调用），因此守卫直接检查该函数的语句清单，而不是只看有没有迁移文件。
func TestEnsureOperationalIndexes_IncludesAuditLogIndexes(t *testing.T) {
	src, err := os.ReadFile("database.go")
	require.NoError(t, err)
	body := operationalIndexesFuncBody(string(src))
	require.NotEmpty(t, body, "找不到 EnsureOperationalIndexes")

	for _, name := range []string{
		"audit_logs_tenant_created_idx",
		"audit_logs_tenant_action_created_idx",
		"audit_logs_tenant_resource_created_idx",
	} {
		idx := strings.Index(body, name)
		require.GreaterOrEqual(t, idx, 0, "缺少索引 %s", name)
		segment := body[idx:]
		if end := strings.Index(segment, "\n\t\t},"); end >= 0 {
			segment = segment[:end]
		}
		require.Contains(t, segment, "IF NOT EXISTS", "%s 必须幂等", name)
		require.Contains(t, segment, "ON audit_logs", "%s 必须落在 audit_logs 上", name)
		require.Contains(t, segment, "created_at DESC", "%s 必须支撑倒序扫描", name)
	}
}

func operationalIndexesFuncBody(src string) string {
	marker := "func EnsureOperationalIndexes("
	i := strings.Index(src, marker)
	if i < 0 {
		return ""
	}
	rest := src[i:]
	if j := strings.Index(rest[len(marker):], "\nfunc "); j >= 0 {
		return rest[:len(marker)+j]
	}
	return rest
}
