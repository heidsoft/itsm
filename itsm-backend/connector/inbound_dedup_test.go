package connector

import (
	"context"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func newDedupClient(t *testing.T) *ent.Client {
	t.Helper()
	name := strings.ReplaceAll(t.Name(), "/", "_")
	return enttest.Open(t, "sqlite3", "file:dedup-"+name+"?mode=memory&cache=shared&_fk=1")
}

func TestInboundDedupFirstSeenAndRejects(t *testing.T) {
	c := newDedupClient(t)
	defer c.Close()
	dedup := NewInboundDeduper(c, 5*time.Minute)
	ctx := context.Background()

	seen, err := dedup.HandleBeforeProcess(ctx, 1, "feishu", "evt-1", []byte("body-1"))
	require.NoError(t, err)
	require.False(t, seen, "first time should not be seen")

	seen, err = dedup.HandleBeforeProcess(ctx, 1, "feishu", "evt-1", []byte("body-1"))
	require.NoError(t, err)
	require.True(t, seen, "second time should be marked seen")
}

func TestInboundDedupIsolatesByTenantAndConnector(t *testing.T) {
	c := newDedupClient(t)
	defer c.Close()
	dedup := NewInboundDeduper(c, 5*time.Minute)
	ctx := context.Background()

	_, err := dedup.HandleBeforeProcess(ctx, 1, "feishu", "evt-shared", []byte("a"))
	require.NoError(t, err)
	for _, c := range []struct {
		tenantID int
		conn     string
	}{
		{2, "feishu"},
		{1, "dingtalk"},
	} {
		seen, err := dedup.HandleBeforeProcess(ctx, c.tenantID, c.conn, "evt-shared", []byte("b"))
		require.NoError(t, err)
		require.False(t, seen, "(%d,%s) should not collide", c.tenantID, c.conn)
	}
}

func TestInboundDedupRejectsMissingFields(t *testing.T) {
	c := newDedupClient(t)
	defer c.Close()
	dedup := NewInboundDeduper(c, 5*time.Minute)
	ctx := context.Background()

	_, err := dedup.HandleBeforeProcess(ctx, 0, "feishu", "x", nil)
	require.Error(t, err)

	_, err = dedup.HandleBeforeProcess(ctx, 1, "", "x", nil)
	require.Error(t, err)

	_, err = dedup.HandleBeforeProcess(ctx, 1, "feishu", "", nil)
	require.Error(t, err)
}

func TestInboundDedupCleanupExpired(t *testing.T) {
	c := newDedupClient(t)
	defer c.Close()
	dedup := NewInboundDeduper(c, 1*time.Hour)
	dedup.now = func() time.Time { return time.Now().Add(-2 * time.Hour) }
	ctx := context.Background()
	_, err := dedup.HandleBeforeProcess(ctx, 1, "feishu", "expired-1", []byte("x"))
	require.NoError(t, err)
	dedup.now = time.Now
	n, err := dedup.CleanupExpired(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, n)
}

func TestInboundDedupMarkProcessedAndLookup(t *testing.T) {
	c := newDedupClient(t)
	defer c.Close()
	dedup := NewInboundDeduper(c, 5*time.Minute)
	ctx := context.Background()

	_, err := dedup.HandleBeforeProcess(ctx, 1, "feishu", "evt-2", []byte("x"))
	require.NoError(t, err)

	require.NoError(t, dedup.MarkProcessed(ctx, 1, "feishu", "evt-2", "ok"))

	status, ok, err := dedup.LookupStatus(ctx, 1, "feishu", "evt-2")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "ok", status)

	_, ok, err = dedup.LookupStatus(ctx, 1, "feishu", "missing")
	require.NoError(t, err)
	require.False(t, ok)
}

// TestInboundDedupOverridesExpiredRowWithoutCleanup 锁住 schema 注释承诺的语义：
// 「expires_at 后允许覆盖」。唯一约束不会随时间自己消失，旧实现完全依赖
// CleanupExpired —— 而它在生产里零调用点（只有测试调），于是同一个 event_id
// 会被永久判重复。现在命中约束时会顺手删掉已过期的同一行再重试。
func TestInboundDedupOverridesExpiredRowWithoutCleanup(t *testing.T) {
	c := newDedupClient(t)
	defer c.Close()
	dedup := NewInboundDeduper(c, 5*time.Minute)
	ctx := context.Background()

	t0 := time.Now()
	dedup.now = func() time.Time { return t0 }

	seen, err := dedup.HandleBeforeProcess(ctx, 1, "feishu", "evt-exp", []byte("body"))
	require.NoError(t, err)
	require.False(t, seen)

	// TTL 内重复 → 真重复
	seen, err = dedup.HandleBeforeProcess(ctx, 1, "feishu", "evt-exp", []byte("body"))
	require.NoError(t, err)
	require.True(t, seen)

	// 时间推进到 TTL 之后，且**没有**任何清理任务跑过
	dedup.now = func() time.Time { return t0.Add(10 * time.Minute) }
	seen, err = dedup.HandleBeforeProcess(ctx, 1, "feishu", "evt-exp", []byte("body"))
	require.NoError(t, err)
	require.False(t, seen, "expired row must be overridable even when cleanup never ran")

	// 覆盖后仍是普通的一行，且未过期时再次重复
	rows, err := c.ConnectorInboundDedup.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	seen, err = dedup.HandleBeforeProcess(ctx, 1, "feishu", "evt-exp", []byte("body"))
	require.NoError(t, err)
	require.True(t, seen)
}
