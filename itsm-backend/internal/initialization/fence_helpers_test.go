package initialization

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"itsm-backend/migration"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func validateInitializationTestDatabase(name string) error {
	const prefix = "itsm_init_test_"
	if !strings.HasPrefix(name, prefix) || len(name) == len(prefix) {
		return errors.New("initialization tests require a database named itsm_init_test_<name>")
	}
	return nil
}

// Every physical connection is checked before any write and receives the same
// search_path. A one-off SET on *sql.DB would leave other pool connections unsafe.
// Only the explicit test DSN is used; application config and .env are never read.
type initializationTestConnector struct {
	driver.Connector
	schema string
}

func (c *initializationTestConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	ready := false
	defer func() {
		if !ready {
			_ = conn.Close()
		}
	}()
	querier, ok := conn.(driver.QueryerContext)
	if !ok {
		return nil, errors.New("PostgreSQL driver does not support database identity validation")
	}
	rows, err := querier.QueryContext(ctx, "SELECT current_database()::text", nil)
	if err != nil {
		return nil, err
	}
	values := make([]driver.Value, 1)
	err = rows.Next(values)
	closeErr := rows.Close()
	if err != nil || closeErr != nil {
		return nil, errors.Join(err, closeErr)
	}
	name, ok := values[0].(string)
	if !ok {
		return nil, errors.New("unexpected PostgreSQL database identity type")
	}
	if err := validateInitializationTestDatabase(name); err != nil {
		return nil, err
	}
	execer, ok := conn.(driver.ExecerContext)
	if !ok {
		return nil, errors.New("PostgreSQL driver does not support isolated test sessions")
	}
	_, err = execer.ExecContext(ctx, `SELECT set_config('search_path', $1, false),
		set_config('statement_timeout', '5000', false), set_config('lock_timeout', '3000', false)`,
		[]driver.NamedValue{{Ordinal: 1, Value: pq.QuoteIdentifier(c.schema)}})
	if err != nil {
		return nil, err
	}
	ready = true
	return conn, nil
}

type initializationPostgresFixture struct {
	ctx           context.Context
	db, secondDB  *sql.DB
	store, second *SQLStore
	schema        string
}

func newInitializationPostgresFixture(t *testing.T) *initializationPostgresFixture {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("ITSM_INITIALIZATION_TEST_DSN"))
	if dsn == "" {
		t.Skip("set ITSM_INITIALIZATION_TEST_DSN to an isolated itsm_init_test_<name> database to run PostgreSQL regressions")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	connector, err := pq.NewConnector(dsn)
	require.NoError(t, err, "parse explicit initialization test DSN")
	admin := sql.OpenDB(&initializationTestConnector{Connector: connector, schema: "pg_catalog"})
	t.Cleanup(func() { require.NoError(t, admin.Close()) })
	// Ping invokes the read-only database-name guard before CREATE SCHEMA.
	require.NoError(t, admin.PingContext(ctx), "validate initialization test database before any writes")
	schema := "init_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = admin.ExecContext(ctx, "CREATE SCHEMA "+pq.QuoteIdentifier(schema))
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, err := admin.ExecContext(cleanupCtx, "DROP SCHEMA "+pq.QuoteIdentifier(schema)+" CASCADE")
		require.NoError(t, err, "remove only this test's isolated schema")
	})

	open := func() *sql.DB {
		db := sql.OpenDB(&initializationTestConnector{Connector: connector, schema: schema})
		db.SetMaxOpenConns(4)
		db.SetMaxIdleConns(4)
		t.Cleanup(func() { require.NoError(t, db.Close()) })
		require.NoError(t, db.PingContext(ctx))
		return db
	}
	db, secondDB := open(), open()
	// Hold several physical connections simultaneously to check pool isolation.
	for _, pool := range []*sql.DB{db, secondDB} {
		var conns []*sql.Conn
		for range 3 {
			conn, err := pool.Conn(ctx)
			require.NoError(t, err)
			defer conn.Close() // Also release checked-out connections if an assertion fails.
			conns = append(conns, conn)
			var actual string
			require.NoError(t, conn.QueryRowContext(ctx, "SELECT current_schema()").Scan(&actual))
			require.Equal(t, schema, actual)
		}
		for _, conn := range conns {
			require.NoError(t, conn.Close())
		}
	}

	// Reuse the production migration, never a copied lease/ledger implementation.
	ddl := migration.GetMigrationSQL("008_add_initialization_ledger")
	require.NotEmpty(t, ddl)
	_, err = db.ExecContext(ctx, ddl)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `CREATE TABLE initialization_test_business (
		scope_type TEXT NOT NULL, scope_id BIGINT NOT NULL, source_key TEXT NOT NULL,
		value TEXT NOT NULL, PRIMARY KEY (scope_type, scope_id, source_key)
	)`)
	require.NoError(t, err)
	store, err := NewSQLStore(db)
	require.NoError(t, err)
	second, err := NewSQLStore(secondDB)
	require.NoError(t, err)
	return &initializationPostgresFixture{ctx: ctx, db: db, secondDB: secondDB, store: store, second: second, schema: schema}
}

func (f *initializationPostgresFixture) expireLease(t *testing.T, scope Scope, component string) {
	t.Helper()
	// Set expiry after the business transaction began, but before Complete runs.
	// This also catches using transaction-stable NOW() for the final fence check.
	result, err := f.secondDB.ExecContext(f.ctx, `UPDATE initialization_installations
		SET lease_expires_at = clock_timestamp()
		WHERE scope_type = $1 AND scope_id = $2 AND component = $3`, scope.Type, scope.ID, component)
	require.NoError(t, err)
	count, err := result.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

func (f *initializationPostgresFixture) businessCount(t *testing.T) int {
	t.Helper()
	var count int
	require.NoError(t, f.secondDB.QueryRowContext(f.ctx, "SELECT COUNT(*) FROM initialization_test_business").Scan(&count))
	return count
}

func (f *initializationPostgresFixture) installation(t *testing.T, scope Scope) InstallationStatus {
	t.Helper()
	statuses, err := f.second.Status(f.ctx, scope)
	require.NoError(t, err)
	require.Len(t, statuses, 1)
	return statuses[0]
}

func (f *initializationPostgresFixture) installationSnapshot(t *testing.T, scope Scope) string {
	t.Helper()
	var snapshot string
	// Include heartbeat_at and last_run_id, which the public Status DTO omits.
	require.NoError(t, f.secondDB.QueryRowContext(f.ctx, `SELECT to_jsonb(i)::text FROM initialization_installations i
		WHERE scope_type = $1 AND scope_id = $2`, scope.Type, scope.ID).Scan(&snapshot))
	return snapshot
}

func (f *initializationPostgresFixture) assertRunAndAttempt(t *testing.T, runID int64, want string) {
	t.Helper()
	var runStatus, attemptStatus string
	var runCompleted, attemptCompleted sql.NullTime
	require.NoError(t, f.secondDB.QueryRowContext(f.ctx, `SELECT r.status, r.completed_at, a.status, a.completed_at
		FROM initialization_runs r JOIN initialization_component_attempts a ON a.run_id = r.id
		WHERE r.id = $1`, runID).Scan(&runStatus, &runCompleted, &attemptStatus, &attemptCompleted))
	require.Equal(t, want, runStatus)
	require.Equal(t, want, attemptStatus)
	require.Equal(t, want != "running", runCompleted.Valid)
	require.Equal(t, want != "running", attemptCompleted.Valid)
}

// A tiny product-data fixture, not a substitute SQLStore: all business writes
// and their verification use the actual driver supplied by Engine.Apply.
func writeAndVerifyInitializationFixture(ctx context.Context, scope Scope, plan Plan, d dialect.Driver) (Result, error) {
	// Nested Ent-style transactions must not commit the Engine-owned transaction.
	nested, err := d.Tx(ctx)
	if err != nil {
		return Result{}, err
	}
	if err := nested.Exec(ctx, `INSERT INTO initialization_test_business (scope_type, scope_id, source_key, value)
		VALUES ($1, $2, $3, $4) ON CONFLICT (scope_type, scope_id, source_key) DO NOTHING`,
		[]any{scope.Type, scope.ID, plan.Component, plan.TargetVersion}, nil); err != nil {
		return Result{}, err
	}
	if err := nested.Commit(); err != nil {
		return Result{}, err
	}
	var rows entsql.Rows
	if err := d.Query(ctx, `SELECT value FROM initialization_test_business
		WHERE scope_type = $1 AND scope_id = $2 AND source_key = $3`,
		[]any{scope.Type, scope.ID, plan.Component}, &rows); err != nil {
		return Result{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return Result{}, errors.Join(errors.New("verification cannot see transaction-local business write"), rows.Err())
	}
	var value string
	if err := rows.Scan(&value); err != nil {
		return Result{}, err
	}
	if value != plan.TargetVersion {
		return Result{}, fmt.Errorf("unexpected initialized value %q", value)
	}
	return Result{
		Summary:          map[string]any{"verified": true},
		RollbackMetadata: map[string]any{"sourceKey": plan.Component},
	}, rows.Err()
}
