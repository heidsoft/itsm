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
