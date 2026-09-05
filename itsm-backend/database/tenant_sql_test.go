package database

import (
	"context"
	"database/sql"
	"testing"

	"itsm-backend/common/tenantctx"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestWithTenantSQLRejectsContextTenantMismatch(t *testing.T) {
	db, err := sql.Open("sqlite3", "file:tenant_sql_scope?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = WithTenantSQL(tenantctx.WithTenantID(context.Background(), 7), db, 8,
		func(SQLExecutor) (struct{}, error) { return struct{}{}, nil })
	require.ErrorContains(t, err, "does not match")
}

func TestWithTenantSQLUsesScopedExecutor(t *testing.T) {
	db, err := sql.Open("sqlite3", "file:tenant_sql_execute?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("CREATE TABLE scoped_values (tenant_id INTEGER NOT NULL, value TEXT NOT NULL)")
	require.NoError(t, err)

	value, err := WithTenantSQL(tenantctx.WithTenantID(context.Background(), 7), db, 7,
		func(q SQLExecutor) (string, error) {
			_, err := q.ExecContext(context.Background(), "INSERT INTO scoped_values (tenant_id, value) VALUES (?, ?)", 7, "ok")
			if err != nil {
				return "", err
			}
			var got string
			if err := q.QueryRowContext(context.Background(), "SELECT value FROM scoped_values WHERE tenant_id = ?", 7).Scan(&got); err != nil {
				return "", err
			}
			return got, nil
		})
	require.NoError(t, err)
	require.Equal(t, "ok", value)
}
