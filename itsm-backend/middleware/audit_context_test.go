package middleware

import (
	"context"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestAuditWriters_PreserveRequestIDAfterCancellation(t *testing.T) {
	for _, name := range []string{"login", "self-service"} {
		t.Run(name, func(t *testing.T) {
			client := enttest.Open(t, "sqlite3", "file:audit_context_"+name+"?mode=memory&_fk=1")
			defer client.Close()
			calls := 0
			// Ent 写入钩子观察实际 Exec 收到的上下文，并继续真实落库，避免只测 context 包。
			client.AuditLog.Use(func(next ent.Mutator) ent.Mutator {
				return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
					calls++
					rid, ok := RequestIDFromContext(ctx)
					require.True(t, ok, "实际审计写入上下文必须保留 request ID")
					require.Equal(t, "audit-chain-123", rid)
					require.NoError(t, ctx.Err(), "客户端取消不得取消审计")
					deadline, ok := ctx.Deadline()
					require.True(t, ok)
					require.WithinDuration(t, time.Now().Add(2*time.Second), deadline, 200*time.Millisecond)
					return next.Mutate(ctx, m)
				})
			})
			ctx, cancel := context.WithCancel(WithRequestID(context.Background(), "audit-chain-123"))
			ctx = WithLoginAuditRequest(ctx, "127.0.0.1", "audit-test")
			cancel()
			if name == "login" {
				RecordLoginAudit(ctx, client, 2, 3, "alice", "LOGIN_SUCCESS", "")
			} else {
				RecordSelfServiceAudit(ctx, client, SelfServiceAuditEvent{Path: "/api/v1/auth/register", Action: "REGISTER", StatusCode: 200, TenantID: 3, UserID: 2})
			}
			require.Equal(t, 1, calls, "append-only 无幂等键，不得重试")
			row, err := client.AuditLog.Query().Only(context.Background())
			require.NoError(t, err)
			require.Equal(t, "audit-chain-123", row.RequestID)
			require.Equal(t, 3, row.TenantID)
			require.Equal(t, 2, row.UserID)
			require.Equal(t, "127.0.0.1", row.IP)
		})
	}
}

func TestAuditWriters_NilClient(t *testing.T) {
	// 无后端时必须在读取 ctx 前早退。
	require.NotPanics(t, func() { RecordLoginAudit(context.Background(), nil, 0, 0, "", "", "") })
	require.NotPanics(t, func() { RecordSelfServiceAudit(context.Background(), nil, SelfServiceAuditEvent{}) })
}
