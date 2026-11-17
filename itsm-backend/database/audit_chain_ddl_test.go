package database

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// L8 回归锁：AI 决策底账事件表必须随 bootstrap 幂等落地。
//
// 该表不在 ent schema 里，也不新增迁移文件（迁移账本已断裂，见长期记忆），
// 唯一装配点就是 EnsureOperationalIndexes。守卫直接扫该函数的语句清单：
// 缺任何一条，空库首装就没有决策链，而 PATCH 会在运行期直接报错。
func TestEnsureOperationalIndexes_IncludesAIAuditLedgerChain(t *testing.T) {
	src, err := os.ReadFile("database.go")
	require.NoError(t, err)
	body := operationalIndexesFuncBody(string(src))
	require.NotEmpty(t, body, "找不到 EnsureOperationalIndexes")

	names := []string{
		"ai_audit_ledger_events.table",
		"ai_audit_ledger_events.chain_unique_idx",
		"ai_audit_ledger_events.tenant_id_idx",
		"ai_audit_ledger_events.audit_id_idx",
	}
	for _, name := range names {
		// 必须匹配整条 name 标记：只做 strings.Contains 的话，把名字改成
		// "<name>.LEGACY" 也会命中前缀，守卫会在 DDL 已被改名/删除时假绿。
		marker := `name: "` + name + `",`
		idx := strings.Index(body, marker)
		require.GreaterOrEqual(t, idx, 0, "缺少 DDL 语句 %s", name)
		segment := body[idx:]
		if end := strings.Index(segment, "\n\t\t},"); end >= 0 {
			segment = segment[:end]
		}
		require.Contains(t, segment, "IF NOT EXISTS", "%s 必须幂等", name)
		require.Contains(t, segment, "ai_audit_ledger_events", "%s 必须落在事件表上", name)
	}

	require.Contains(t, body, "CREATE TABLE IF NOT EXISTS ai_audit_ledger_events")
	require.Contains(t, body, "UNIQUE INDEX IF NOT EXISTS ai_audit_ledger_events_tenant_prev_hash_idx")
	require.Contains(t, body, "ON ai_audit_ledger_events (tenant_id, prev_hash)",
		"同租户同一链尾只能有一个后继，否则并发追加会让链分叉")
	require.Contains(t, body, "prev_hash TEXT NOT NULL")
	require.Contains(t, body, "hash TEXT NOT NULL")
	require.Contains(t, body, "actor_user_id INT NOT NULL")
}
