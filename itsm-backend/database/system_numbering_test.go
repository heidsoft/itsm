package database

import (
	"context"
	"database/sql"
	"testing"

	"itsm-backend/common/tenantctx"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestSystemNumberingRequiresExplicitSystemContext(t *testing.T) {
	db, err := sql.Open("sqlite3", "file:system_numbering?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	executor := NewSystemNumberingExecutor(db, zaptest.NewLogger(t).Sugar())
	_, err = executor.WithTx(context.Background(), "incident_number", func(*sql.Tx) (string, error) { return "INC-1", nil })
	require.ErrorContains(t, err, "explicit system context")
}

func TestSystemNumberingCommitsOnlySuccessfulAllocation(t *testing.T) {
	db, err := sql.Open("sqlite3", "file:system_numbering_commit?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("CREATE TABLE allocations (number TEXT NOT NULL)")
	require.NoError(t, err)

	executor := NewSystemNumberingExecutor(db, zaptest.NewLogger(t).Sugar())
	ctx := tenantctx.SystemContext(context.Background(), "test:numbering", "verify transactional allocation")
	number, err := executor.WithTx(ctx, "incident_number", func(tx *sql.Tx) (string, error) {
		_, err := tx.ExecContext(ctx, "INSERT INTO allocations(number) VALUES (?)", "INC-202609-000001")
		return "INC-202609-000001", err
	})
	require.NoError(t, err)
	require.Equal(t, "INC-202609-000001", number)
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM allocations").Scan(&count))
	require.Equal(t, 1, count)
}
