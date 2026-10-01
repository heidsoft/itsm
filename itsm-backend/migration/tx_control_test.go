package migration

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestStripEmbeddedTxControl_Basic(t *testing.T) {
	in := "BEGIN;\nCREATE TABLE a (id INT);\nCOMMIT;\n"
	got := stripEmbeddedTxControl(in)
	assert.NotContains(t, got, "BEGIN;")
	assert.NotContains(t, got, "COMMIT;")
	assert.Contains(t, got, "CREATE TABLE a (id INT);")
}

func TestStripEmbeddedTxControl_PreservesPlPgSQLBody(t *testing.T) {
	body := `DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM t) THEN
        RAISE NOTICE 'has; rows';
    END IF;
END $$;`
	in := "BEGIN;\n" + body + "\nCOMMIT;\n"
	got := stripEmbeddedTxControl(in)
	assert.Contains(t, got, body, "dollar-quoted PL/pgSQL body must be copied verbatim")
	assert.NotContains(t, got, "BEGIN;")
	assert.NotContains(t, got, "COMMIT;")
}

func TestStripEmbeddedTxControl_PreservesStringLiteral(t *testing.T) {
	in := `INSERT INTO t (v) VALUES ('a;b');`
	got := stripEmbeddedTxControl(in)
	assert.Contains(t, got, `'a;b'`)
}

func TestStripEmbeddedTxControl_PreservesFunctionEndSemicolon(t *testing.T) {
	// 20260501_rbac_endpoint_acls.sql:1513 的实证场景：COMMIT; 之后紧跟
	// LANGUAGE plpgsql; 函数体，体内有 END;。剥离器不得触碰函数体内容。
	in := `BEGIN;
CREATE OR REPLACE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN NEW.updated_at = NOW();
RETURN NEW;
END;
$$ LANGUAGE plpgsql;
COMMIT;
`
	got := stripEmbeddedTxControl(in)
	assert.Contains(t, got, "END;\n$$ LANGUAGE plpgsql;", "function body END; must survive")
	assert.Contains(t, got, "NEW.updated_at = NOW();")
	assert.NotContains(t, got, "BEGIN;")
	assert.NotContains(t, got, "COMMIT;")
}

func TestStripEmbeddedTxControl_CaseInsensitive(t *testing.T) {
	for _, in := range []string{"begin;\nSELECT 1;\ncommit;\n", "Begin;\nSELECT 1;\nCommit;\n"} {
		got := stripEmbeddedTxControl(in)
		assert.NotContains(t, got, "BEGIN;")
		assert.NotContains(t, got, "COMMIT;")
		assert.Contains(t, got, "SELECT 1;")
	}
}

func TestStripEmbeddedTxControl_StartTransaction(t *testing.T) {
	got := stripEmbeddedTxControl("START TRANSACTION;\nSELECT 1;\n")
	assert.NotContains(t, got, "START TRANSACTION;")
	assert.Contains(t, got, "SELECT 1;")
}

func TestStripEmbeddedTxControl_Idempotent(t *testing.T) {
	in := "BEGIN;\nCREATE TABLE b (id INT);\nCOMMIT;\n"
	once := stripEmbeddedTxControl(in)
	twice := stripEmbeddedTxControl(once)
	assert.Equal(t, once, twice)
}

func TestStripEmbeddedTxControl_RepoMigrationsCleanAfterStrip(t *testing.T) {
	// 守卫扫描：仓库内所有 *.sql 迁移经剥离后不得残留顶层事务控制语句，
	// 也不得有剥离器误伤（剥离前后非空白内容应保持一致性）。
	dir := filepath.Join("..", "migrations")
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	count := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".sql") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, err)
		stripped := stripEmbeddedTxControl(string(raw))
		// 剥离后不应再有顶层 BEGIN;/COMMIT;/START TRANSACTION;
		for _, line := range strings.Split(stripped, "\n") {
			trimmed := strings.TrimSpace(line)
			upper := strings.ToUpper(trimmed)
			if upper == "BEGIN;" || upper == "COMMIT;" || upper == "START TRANSACTION;" || upper == "BEGIN TRANSACTION;" {
				t.Errorf("%s: leftover tx-control statement after strip: %q", name, trimmed)
			}
		}
		count++
	}
	assert.Greater(t, count, 0, "expected to scan at least one migration file")

	// 实证危险样本：顶层 COMMIT; 之后紧跟 dollar-quoted 函数体，体内含 END;。剥离只能去掉
	// 顶层事务控制，函数体必须逐字节保留。原样本取自 20260501_rbac_endpoint_acls.sql
	// （仓库内唯一带 $$ LANGUAGE plpgsql; 的脚本），2026-10-01 该迁移退役后改用内联夹具，
	// 并改为扫描仓库内所有仍含 dollar-quoted 体的真实迁移。
	dangerSample := "BEGIN;\nCREATE TABLE t (id INT);\nCOMMIT;\n" +
		"CREATE OR REPLACE FUNCTION trg() RETURNS TRIGGER AS $$ BEGIN\nRETURN NEW;\nEND;\n$$ LANGUAGE plpgsql;\n" +
		"DO $$ BEGIN\n  IF EXISTS (SELECT 1 FROM t) THEN COMMIT; END IF;\nEND $$;\n"
	dangerStripped := stripEmbeddedTxControl(dangerSample)
	assert.Contains(t, dangerStripped, "END;\n$$ LANGUAGE plpgsql;", "dollar-quoted function body must survive in fixture")
	assert.NotContains(t, dangerStripped, "\nCOMMIT;\n", "top-level COMMIT; must be stripped")
	assert.Contains(t, dangerStripped, "IF EXISTS (SELECT 1 FROM t) THEN COMMIT; END IF;",
		"COMMIT; inside a dollar-quoted body must not be touched")

	dollarBodies := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".sql") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, err)
		body := stripSQLCommentLines(string(raw))
		for _, marker := range []string{"$$", "$body$", "$func$"} {
			if strings.Contains(body, marker) {
				dollarBodies++
				// 剥离后 dollar-quoted 体必须原样存在（体首字符不被破坏）
				assert.Contains(t, stripEmbeddedTxControl(string(raw)), marker,
					"%s: dollar-quoted body must survive strip", name)
				break
			}
		}
	}
	assert.Greater(t, dollarBodies, 0, "expected at least one repo migration with a dollar-quoted body")
}

// stripSQLCommentLines 去掉 -- 行注释，避免把注释里对 dollar 标签的说明当成函数体。
func stripSQLCommentLines(sql string) string {
	var out []string
	for _, line := range strings.Split(sql, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// TestApplyMigration_EmbeddedTxControl 是修复的端到端回归：
// 修复前，含 BEGIN;/COMMIT; 的脚本会让内层 COMMIT 提前结束托管事务，
// 账本 INSERT 落入 autocommit，随后 tx.Commit 命中 idle 连接报
// "pq: unexpected transaction status idle"，且后续 pending 迁移被整体中止。
// 修复后，整个脚本应在单个托管事务内原子执行并正常落账。
func TestApplyMigration_EmbeddedTxControl(t *testing.T) {
	db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=itsm dbname=itsm sslmode=disable")
	if err != nil {
		t.Skip("Database not available, skipping integration test")
	}
	defer db.Close()
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Skip("Cannot ping database, skipping integration test: " + err.Error())
	}

	version := "99990001_tx_control_regression"
	table := "test_tx_control_regression"

	m := NewMigrator(db, zap.NewNop().Sugar())
	require.NoError(t, m.EnsureMigrationsTable(ctx))

	// 幂等预清理（脚本自身 DROP IF EXISTS，这里额外清账本与残留表）
	_, _ = db.ExecContext(ctx, `DELETE FROM schema_migrations WHERE version = $1`, version)
	_, _ = db.ExecContext(ctx, "DROP TABLE IF EXISTS "+table)

	mig := Migration{
		Version:     version,
		Description: "tx control regression",
		SQLContent: "BEGIN;\n" +
			"CREATE TABLE " + table + " (id BIGSERIAL PRIMARY KEY, note TEXT);\n" +
			"INSERT INTO " + table + " (note) VALUES ('inside-managed-tx');\n" +
			"COMMIT;\n",
	}

	require.NoError(t, m.ApplyMigration(ctx, mig))

	// 表存在且数据写入
	var note string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT note FROM "+table).Scan(&note))
	assert.Equal(t, "inside-managed-tx", note)

	// 账本 checksum 必须等于原始 SQL 的 checksum（剥离不得影响 checksum 口径）
	var checksum string
	require.NoError(t, db.QueryRowContext(ctx,
		"SELECT checksum FROM schema_migrations WHERE version = $1", version).Scan(&checksum))
	assert.Equal(t, checksumSQL(mig.SQLContent), checksum)

	// 清理
	_, err = db.ExecContext(ctx, "DROP TABLE IF EXISTS "+table)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `DELETE FROM schema_migrations WHERE version = $1`, version)
	require.NoError(t, err)
}
