package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// 本文件锁住 AUDIT-002：20261119 迁移与 Ent Schema 在 configuration_items 上
// 建了两对同列索引，fresh-install 门禁因此持续红。修复方式是把两对登记进
// duplicateIndexPairs（保留 Ent 名字），而不是改写已入账的迁移 SQL。
//
// 既有安装上这些索引已经建好，EnsureIndexConvergence 在下一次 bootstrap 时
// 幂等删除冗余方；Ent 索引一旦缺失，收敛会拒绝删除并报错，不让访问路径裸奔。

const (
	cfgOwnerKeep   = "configurationitem_tenant_id_owner_id"
	cfgOwnerDrop   = "ci_tenant_owner_idx"
	cfgDeptKeep    = "configurationitem_tenant_id_department_id"
	cfgDeptDrop    = "ci_tenant_department_idx"
	convergenceDSN = "ITSM_INITIALIZATION_TEST_DSN"
)

func TestDuplicateIndexPairs_CoverConfigurationItemOwnership(t *testing.T) {
	// 静态锁：防止有人把两对删掉，或把保留/删除方向写反。
	found := map[string]string{}
	for _, pair := range duplicateIndexPairs {
		found[pair[0]] = pair[1]
	}
	require.Equal(t, cfgOwnerDrop, found[cfgOwnerKeep], "owner 索引必须以 Ent 名为保留方")
	require.Equal(t, cfgDeptDrop, found[cfgDeptKeep], "department 索引必须以 Ent 名为保留方")
}

// 每条连接都校验库名白名单并钉住独立 search_path，测试不可能写进真实 schema。
type indexTestConnector struct {
	driver.Connector
	schema string
}

func (c *indexTestConnector) Connect(ctx context.Context) (driver.Conn, error) {
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
	if !ok || !strings.HasPrefix(name, "itsm_init_test_") || name == "itsm_init_test_" {
		return nil, errors.New("index convergence regression requires an itsm_init_test_<name> database")
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

func newIndexConvergenceTestDB(t *testing.T) (context.Context, *sql.DB) {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv(convergenceDSN))
	if dsn == "" {
		t.Skip("set ITSM_INITIALIZATION_TEST_DSN to an isolated itsm_init_test_<name> database to run index convergence regression")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	t.Cleanup(cancel)

	connector, err := pq.NewConnector(dsn)
	require.NoError(t, err, "parse explicit regression DSN")
	admin := sql.OpenDB(&indexTestConnector{Connector: connector, schema: "pg_catalog"})
	t.Cleanup(func() { require.NoError(t, admin.Close()) })
	require.NoError(t, admin.PingContext(ctx), "validate regression database identity before any writes")

	schema := "idxconv_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = admin.ExecContext(ctx, "CREATE SCHEMA "+pq.QuoteIdentifier(schema))
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		_, err := admin.ExecContext(cleanupCtx, "DROP SCHEMA "+pq.QuoteIdentifier(schema)+" CASCADE")
		require.NoError(t, err, "remove only this test's isolated schema")
	})

	db := sql.OpenDB(&indexTestConnector{Connector: connector, schema: schema})
	db.SetMaxOpenConns(2)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, db.PingContext(ctx))

	// 复刻 fresh-install 上两处 DDL 的结果：Ent 建两份，迁移另建两份同列索引。
	_, err = db.ExecContext(ctx, `
		CREATE TABLE configuration_items (
			id serial PRIMARY KEY,
			tenant_id integer NOT NULL,
			owner_id integer,
			department_id integer
		);
		CREATE INDEX `+cfgOwnerKeep+` ON configuration_items (tenant_id, owner_id);
		CREATE INDEX `+cfgDeptKeep+` ON configuration_items (tenant_id, department_id);
		CREATE INDEX `+cfgOwnerDrop+` ON configuration_items (tenant_id, owner_id);
		CREATE INDEX `+cfgDeptDrop+` ON configuration_items (tenant_id, department_id);`)
	require.NoError(t, err)
	return ctx, db
}

func indexExists(t *testing.T, ctx context.Context, db *sql.DB, name string) bool {
	t.Helper()
	var ok bool
	require.NoError(t, db.QueryRowContext(ctx, `SELECT to_regclass($1) IS NOT NULL`, name).Scan(&ok))
	return ok
}

func TestEnsureIndexConvergence_DropsDuplicateConfigurationItemIndexes(t *testing.T) {
	ctx, db := newIndexConvergenceTestDB(t)
	pairs := [][2]string{
		{cfgOwnerKeep, cfgOwnerDrop},
		{cfgDeptKeep, cfgDeptDrop},
	}

	require.True(t, indexExists(t, ctx, db, cfgOwnerDrop), "夹具必须先造出重复索引")
	require.True(t, indexExists(t, ctx, db, cfgDeptDrop), "夹具必须先造出重复索引")

	dropped, err := ensureIndexConvergenceWithPairs(ctx, db, pairs)
	require.NoError(t, err)
	require.Equal(t, 2, dropped)

	require.True(t, indexExists(t, ctx, db, cfgOwnerKeep), "必须保留 Ent 声明的 owner 索引")
	require.True(t, indexExists(t, ctx, db, cfgDeptKeep), "必须保留 Ent 声明的 department 索引")
	require.False(t, indexExists(t, ctx, db, cfgOwnerDrop), "必须删除迁移写死的重复索引")
	require.False(t, indexExists(t, ctx, db, cfgDeptDrop), "必须删除迁移写死的重复索引")

	// 重复执行（每次 bootstrap 都会跑）必须幂等。
	dropped, err = ensureIndexConvergenceWithPairs(ctx, db, pairs)
	require.NoError(t, err)
	require.Equal(t, 0, dropped)
	require.True(t, indexExists(t, ctx, db, cfgOwnerKeep))
	require.True(t, indexExists(t, ctx, db, cfgDeptKeep))
}

func TestEnsureIndexConvergence_RefusesDropWhenKeeperMissing(t *testing.T) {
	ctx, db := newIndexConvergenceTestDB(t)
	pairs := [][2]string{
		{"configurationitem_tenant_id_missing", cfgOwnerDrop},
	}

	dropped, err := ensureIndexConvergenceWithPairs(ctx, db, pairs)
	require.Error(t, err, "保留方缺失时必须报错，不能让访问路径失去索引")
	require.Contains(t, err.Error(), "拒绝删除")
	require.Equal(t, 0, dropped)
	require.True(t, indexExists(t, ctx, db, cfgOwnerDrop), "报错时冗余方也必须保留")
}
