package pii

import (
	"database/sql"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

// TestMigrationUpDownRoundTrip 用 SQLite 内存库演练 up → 兼容读 → down → 数据保留
// 四个阶段,确保同一迁移的 rollback 真能恢复旧 schema 且旧行不丢。
func TestMigrationUpDownRoundTrip(t *testing.T) {
	db, err := sql.Open("sqlite3", "file:compat?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	defer db.Close()

	mustExec(t, db, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			email VARCHAR(255) NOT NULL
		);
		INSERT INTO users (id, email) VALUES (1, 'alice@example.com'), (2, 'bob@example.com');
	`)

	upSQL := "ALTER TABLE users ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT 0;"
	downSQL := "ALTER TABLE users DROP COLUMN email_verified;"

	// Phase 1: apply up
	mustExec(t, db, upSQL)

	// Phase 2: 兼容读 — 老 SELECT * 仍能跑,且新列有默认值
	rows := queryAll(t, db, "SELECT id, email, email_verified FROM users ORDER BY id")
	require.Len(t, rows, 2)
	require.Equal(t, "alice@example.com", rows[0]["email"])
	require.Equal(t, "0", rows[0]["email_verified"])

	// Phase 3: NOT NULL / DEFAULT 校核通过 — 上面 ALTER 有 DEFAULT 0
	violations, err := CheckNotNullDefaults(upSQL)
	require.NoError(t, err)
	require.Len(t, violations, 1)
	require.True(t, violations[0].HasDefault)

	// Phase 4: apply down, schema 回到 up 之前
	mustExec(t, db, downSQL)
	rowsAfterDown := queryAll(t, db, "SELECT id, email FROM users ORDER BY id")
	require.Len(t, rowsAfterDown, 2, "down 后数据必须保留")
	require.Equal(t, "alice@example.com", rowsAfterDown[0]["email"])

	// 校验 email_verified 列已不存在
	_, err = db.Exec("SELECT email_verified FROM users LIMIT 1")
	require.Error(t, err, "down 后不应再能 SELECT email_verified")
	require.True(t, strings.Contains(err.Error(), "no such column"))
}

func TestMigrationUpMissingDefaultDetected(t *testing.T) {
	// NOT NULL + 缺 DEFAULT → 校核必须报 MISS
	upSQL := "ALTER TABLE users ADD COLUMN legacy_password VARCHAR(255) NOT NULL;"
	violations, err := CheckNotNullDefaults(upSQL)
	require.NoError(t, err)
	require.Len(t, violations, 1)
	require.False(t, violations[0].HasDefault, "缺 DEFAULT 必须被识别")
}

func mustExec(t *testing.T, db *sql.DB, sql string) {
	t.Helper()
	if _, err := db.Exec(sql); err != nil {
		t.Fatalf("exec failed: %v\nSQL:\n%s", err, sql)
	}
}

func queryAll(t *testing.T, db *sql.DB, sql string) []map[string]string {
	t.Helper()
	rows, err := db.Query(sql)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	out := []map[string]string{}
	for rows.Next() {
		raw := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range raw {
			ptrs[i] = &raw[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			t.Fatalf("scan failed: %v", err)
		}
		row := make(map[string]string, len(cols))
		for i, c := range cols {
			row[c] = stringFromAny(raw[i])
		}
		out = append(out, row)
	}
	return out
}

func stringFromAny(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	case int64:
		return formatInt(x)
	case int:
		return formatInt(int64(x))
	case int32:
		return formatInt(int64(x))
	case float64:
		if x == float64(int64(x)) {
			return formatInt(int64(x))
		}
		return formatFloat(x)
	case bool:
		if x {
			return "1"
		}
		return "0"
	}
	return ""
}

func formatFloat(v float64) string {
	// 简单实现:委托给 fmt,但本测试期望 string 比较,避免引入额外依赖
	return formatInt(int64(v)) + ".nonint"
}

func formatInt(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	buf := []byte{}
	for v > 0 {
		buf = append([]byte{byte('0' + v%10)}, buf...)
		v /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}