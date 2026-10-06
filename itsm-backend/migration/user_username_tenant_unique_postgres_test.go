package migration

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"

	"itsm-backend/ent"
	"itsm-backend/ent/migrate"
)

// This file is the measured proof for the approved fix (方案 C):
// users.username moves from a field-level global unique key to a
// (tenant_id, username) composite unique key, while users.email stays global.
//
// It runs the real bytes of migrations/20261006_tenant_scope_user_username_unique.sql
// against a throwaway schema, so a typo in that file cannot pass review.

const userUsernameUniqueVersion = "20261006_tenant_scope_user_username_unique"

// PostgreSQL always reports uniqueness breaches as SQLSTATE 23505.
const uniqueViolationSQLState = "23505"

func validateUserUniqueTestDatabase(name string) error {
	const prefix = "itsm_init_test_"
	if !strings.HasPrefix(name, prefix) || len(name) == len(prefix) {
		return errors.New("this regression requires a database named itsm_init_test_<name>")
	}
	return nil
}

// Every physical connection re-validates the database identity and pins
// search_path, so a pooled connection can never write into a real schema.
type userUniqueTestConnector struct {
	driver.Connector
	schema string
}

func (c *userUniqueTestConnector) Connect(ctx context.Context) (driver.Conn, error) {
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
		return nil, errors.New("driver cannot verify database identity")
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
		return nil, errors.New("unexpected database identity type")
	}
	if err := validateUserUniqueTestDatabase(name); err != nil {
		return nil, err
	}
	execer, ok := conn.(driver.ExecerContext)
	if !ok {
		return nil, errors.New("driver cannot pin an isolated search_path")
	}
	if _, err := execer.ExecContext(ctx,
		`SELECT set_config('search_path', $1, false), set_config('statement_timeout', '10000', false)`,
		[]driver.NamedValue{{Ordinal: 1, Value: pq.QuoteIdentifier(c.schema)}}); err != nil {
		return nil, err
	}
	ready = true
	return conn, nil
}

// newUserUniqueTestSchema opens the explicit regression DSN and creates one
// throwaway schema per test. Without the DSN the whole file skips: it never
// falls back to application config or a local database.
func newUserUniqueTestSchema(t *testing.T) (context.Context, *sql.DB) {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("ITSM_INITIALIZATION_TEST_DSN"))
	if dsn == "" {
		t.Skip("set ITSM_INITIALIZATION_TEST_DSN to an isolated itsm_init_test_<name> database to run the users unique-key regression")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	t.Cleanup(cancel)

	connector, err := pq.NewConnector(dsn)
	require.NoError(t, err, "parse explicit regression DSN")
	admin := sql.OpenDB(&userUniqueTestConnector{Connector: connector, schema: "pg_catalog"})
	t.Cleanup(func() { require.NoError(t, admin.Close()) })
	// Ping runs the read-only identity guard before any DDL.
	require.NoError(t, admin.PingContext(ctx), "validate regression database identity before any writes")

	schema := "useruniq_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = admin.ExecContext(ctx, "CREATE SCHEMA "+pq.QuoteIdentifier(schema))
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		_, err := admin.ExecContext(cleanupCtx, "DROP SCHEMA "+pq.QuoteIdentifier(schema)+" CASCADE")
		require.NoError(t, err, "remove only this test's isolated schema")
	})

	db := sql.OpenDB(&userUniqueTestConnector{Connector: connector, schema: schema})
	db.SetMaxOpenConns(2)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, db.PingContext(ctx))
	return ctx, db
}

// migrationSQL returns the on-disk script under test, exactly as the bootstrap
// job would execute it.
func migrationSQL(t *testing.T, version string) string {
	t.Helper()
	migs, err := FilesystemMigrations("../migrations")
	require.NoError(t, err)
	for _, m := range migs {
		if m.Version == version {
			require.NotEmpty(t, m.SQLContent, "%s must carry SQL", version)
			return m.SQLContent
		}
	}
	t.Fatalf("migration %s not discovered on disk", version)
	return ""
}

// legacyUsersDDL reproduces the pre-fix shape measured on the production
// database: username and email each carry a field-level global unique key, and
// PostgreSQL names them users_username_key / users_email_key.
const legacyUsersDDL = `
CREATE TABLE users (
	id serial PRIMARY KEY,
	username text NOT NULL UNIQUE,
	email text NOT NULL UNIQUE,
	name text NOT NULL,
	password_hash text NOT NULL,
	active boolean NOT NULL DEFAULT true,
	tenant_id integer NOT NULL
)`

func uniqueIndexesOnUsername(t *testing.T, ctx context.Context, db *sql.DB, schema string) []string {
	t.Helper()
	rows, err := db.QueryContext(ctx, `
		SELECT i.indexname
		FROM pg_indexes i
		JOIN pg_class c ON c.relname = i.indexname
		JOIN pg_index x ON x.indexrelid = c.oid
		WHERE i.schemaname = $1 AND i.tablename = 'users' AND x.indisunique
		  AND EXISTS (
			SELECT 1 FROM unnest(x.indkey) AS k(attnum)
			JOIN pg_attribute a ON a.attrelid = x.indrelid AND a.attnum = k.attnum
			WHERE a.attname = 'username')`, schema)
	require.NoError(t, err)
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		names = append(names, name)
	}
	require.NoError(t, rows.Err())
	return names
}

func createAdmin(t *testing.T, ctx context.Context, db *sql.DB, tenantID int, username, email string) error {
	t.Helper()
	_, err := db.ExecContext(ctx,
		`INSERT INTO users (username, email, name, password_hash, tenant_id) VALUES ($1, $2, $3, 'x', $4)`,
		username, email, username, tenantID)
	return err
}

func requireUniqueViolation(t *testing.T, err error, want string) {
	t.Helper()
	require.Error(t, err, want)
	var pqErr *pq.Error
	require.True(t, errors.As(err, &pqErr), "expected a PostgreSQL error, got %T: %v", err, err)
	require.Equal(t, uniqueViolationSQLState, string(pqErr.Code),
		"expected a uniqueness breach, got %s: %s", pqErr.Code, pqErr.Message)
}

// TestUserUsernameUniqueMigration_ReplacesGlobalKeyWithTenantScope 是方案 C 的核心
// 证明：迁移前第二个租户无法创建 admin（这就是线上 tenant 2 没有管理员的根因），
// 迁移后同租户重名仍被拒绝、跨租户同名放行、email 继续全局唯一。
func TestUserUsernameUniqueMigration_ReplacesGlobalKeyWithTenantScope(t *testing.T) {
	ctx, db := newUserUniqueTestSchema(t)
	schema := schemaOfTestDB(t, ctx, db)

	_, err := db.ExecContext(ctx, legacyUsersDDL)
	require.NoError(t, err)

	require.NoError(t, createAdmin(t, ctx, db, 1, "admin", "admin-1@example.com"))

	// Pre-fix behaviour, asserted rather than assumed.
	requireUniqueViolation(t, createAdmin(t, ctx, db, 2, "admin", "admin-2@example.com"),
		"legacy global username key must still block a second tenant's admin")
	require.Equal(t, []string{"users_username_key"}, uniqueIndexesOnUsername(t, ctx, db, schema))

	_, err = db.ExecContext(ctx, migrationSQL(t, userUsernameUniqueVersion))
	require.NoError(t, err, "migration must apply cleanly to the legacy shape")

	// Post-fix: each tenant owns its own administrator.
	require.NoError(t, createAdmin(t, ctx, db, 2, "admin", "admin-2@example.com"))
	// ... but uniqueness inside one tenant survives the relaxation.
	requireUniqueViolation(t, createAdmin(t, ctx, db, 2, "admin", "another-2@example.com"),
		"per-tenant username uniqueness must still be enforced")
	// ... and email stays globally unique, which is what keeps password recovery
	// unambiguous (handlers/auth/service.go locates the account by email alone).
	requireUniqueViolation(t, createAdmin(t, ctx, db, 3, "ops", "admin-1@example.com"),
		"email must remain globally unique")

	require.Equal(t, []string{"user_tenant_id_username"}, uniqueIndexesOnUsername(t, ctx, db, schema))
}

// TestUserUsernameUniqueMigration_RefusesExistingDuplicates 锁定「先检测再建索引」：
// 存量同租户重名必须让迁移报错，而不是静默跳过或建出半个索引。
func TestUserUsernameUniqueMigration_RefusesExistingDuplicates(t *testing.T) {
	ctx, db := newUserUniqueTestSchema(t)
	schema := schemaOfTestDB(t, ctx, db)

	_, err := db.ExecContext(ctx, legacyUsersDDL)
	require.NoError(t, err)
	// A duplicate inside one tenant can only exist once the global key is gone,
	// so drop it by hand to reach the state the guard must catch.
	_, err = db.ExecContext(ctx, `ALTER TABLE users DROP CONSTRAINT users_username_key`)
	require.NoError(t, err)
	require.NoError(t, createAdmin(t, ctx, db, 1, "admin", "a@example.com"))
	require.NoError(t, createAdmin(t, ctx, db, 1, "admin", "b@example.com"))

	_, err = db.ExecContext(ctx, migrationSQL(t, userUsernameUniqueVersion))
	require.Error(t, err, "duplicate (tenant_id, username) rows must abort the migration")
	require.Contains(t, err.Error(), "deduplicate")
	// The guard runs before any DDL, so a failing migration must not have
	// created the replacement index either.
	require.NotContains(t, uniqueIndexesOnUsername(t, ctx, db, schema), "user_tenant_id_username",
		"the failing migration must leave no half-applied index set")
}

// TestUserUsernameUniqueMigration_MatchesGeneratedEntSchema 防止两套真相：
// ent 自动建表和版本化迁移必须收敛到同一个索引名，否则每次升级都会多建/漏建索引。
func TestUserUsernameUniqueMigration_MatchesGeneratedEntSchema(t *testing.T) {
	ctx, db := newUserUniqueTestSchema(t)
	schema := schemaOfTestDB(t, ctx, db)

	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	// Default options keep auto-migrate additive, mirroring the production
	// bootstrap job which never enables WithDropColumn/WithDropIndex.
	require.NoError(t, client.Schema.Create(ctx))

	// The generated schema is the authority for what auto-migrate installs.
	require.Equal(t, "user_tenant_id_username", userUsernameIndexNameFromGeneratedSchema(t))

	script := migrationSQL(t, userUsernameUniqueVersion)
	for round := 1; round <= 2; round++ {
		_, err := db.ExecContext(ctx, script)
		require.NoErrorf(t, err, "migration must be idempotent on an ent-managed database (round %d)", round)
	}
	require.Equal(t, []string{"user_tenant_id_username"}, uniqueIndexesOnUsername(t, ctx, db, schema))
}

func userUsernameIndexNameFromGeneratedSchema(t *testing.T) string {
	t.Helper()
	table := migrate.UsersTable
	for _, idx := range table.Indexes {
		if !idx.Unique || len(idx.Columns) != 2 {
			continue
		}
		if idx.Columns[0].Name == "tenant_id" && idx.Columns[1].Name == "username" {
			return idx.Name
		}
	}
	t.Fatalf("generated ent schema has no unique (tenant_id, username) index on users")
	return ""
}

// schemaOfTestDB reads back the pinned search_path instead of trusting the
// helper, so assertions target the schema the connections actually use.
func schemaOfTestDB(t *testing.T, ctx context.Context, db *sql.DB) string {
	t.Helper()
	var schema string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT current_schema()").Scan(&schema))
	require.True(t, strings.HasPrefix(schema, "useruniq_"),
		"test must run inside a throwaway schema, got %q", schema)
	return schema
}
