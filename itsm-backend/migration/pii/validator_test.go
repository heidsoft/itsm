package pii

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckNotNullDefaultsDetectsMissingDefault(t *testing.T) {
	sql := `
		ALTER TABLE users ADD COLUMN email varchar(255) NOT NULL;
		ALTER TABLE users ADD COLUMN phone varchar(32);
	`
	v, err := CheckNotNullDefaults(sql)
	require.NoError(t, err)
	require.Len(t, v, 1)
	require.Equal(t, "users", v[0].Table)
	require.Equal(t, "email", v[0].Column)
	require.False(t, v[0].HasDefault)
}

func TestCheckNotNullDefaultsRecognizesDefaultValue(t *testing.T) {
	sql := `
		ALTER TABLE users ADD COLUMN email varchar(255) NOT NULL DEFAULT '';
		ALTER TABLE users ADD COLUMN age int NOT NULL DEFAULT 0;
		ALTER TABLE users ADD COLUMN active boolean NOT NULL DEFAULT false;
		ALTER TABLE users ADD COLUMN created_at timestamptz NOT NULL DEFAULT now();
	`
	v, err := CheckNotNullDefaults(sql)
	require.NoError(t, err)
	require.Len(t, v, 4)
	for _, item := range v {
		require.True(t, item.HasDefault, "every NOT NULL 必须带 DEFAULT：%v", item)
	}
}

func TestCheckNotNullDefaultsAllowsNullableColumns(t *testing.T) {
	sql := `
		ALTER TABLE users ADD COLUMN email varchar(255);
		ALTER TABLE users ADD COLUMN bio text;
	`
	v, err := CheckNotNullDefaults(sql)
	require.NoError(t, err)
	require.Empty(t, v)
}

func TestCheckNotNullDefaultsIgnoresComments(t *testing.T) {
	sql := `
		-- 这一行不能被解析成 ADD COLUMN
		-- ALTER TABLE fake ADD COLUMN x varchar NOT NULL;
		ALTER TABLE users ADD COLUMN email varchar(255) NOT NULL;
	`
	v, err := CheckNotNullDefaults(sql)
	require.NoError(t, err)
	require.Len(t, v, 1)
	require.Equal(t, "users", v[0].Table)
}

func TestCheckNotNullDefaultsHandlesMultipleStatements(t *testing.T) {
	sql := `
		CREATE TABLE foo (id int);
		ALTER TABLE bar ADD COLUMN x varchar NOT NULL;
		ALTER TABLE bar ADD COLUMN y varchar NOT NULL DEFAULT 'a';
		INSERT INTO bar VALUES (1);
	`
	v, err := CheckNotNullDefaults(sql)
	require.NoError(t, err)
	require.Len(t, v, 2)
	require.False(t, v[0].HasDefault)
	require.True(t, v[1].HasDefault)
}

func TestCheckNotNullDefaultsHandlesBackticks(t *testing.T) {
	sql := "ALTER TABLE `users` ADD COLUMN `email` varchar(255) NOT NULL;"
	v, err := CheckNotNullDefaults(sql)
	require.NoError(t, err)
	require.Len(t, v, 1)
	require.Equal(t, "users", v[0].Table)
	require.Equal(t, "email", v[0].Column)
}

func TestCheckNotNullDefaultsHandlesQuotedDollarTags(t *testing.T) {
	sql := `
		CREATE OR REPLACE FUNCTION mask_it(t text) RETURNS text AS $func$
		BEGIN RETURN t;
		END; $func$ LANGUAGE plpgsql;
		ALTER TABLE users ADD COLUMN email varchar(255) NOT NULL;
	`
	v, err := CheckNotNullDefaults(sql)
	require.NoError(t, err)
	require.Len(t, v, 1)
	require.Equal(t, "email", v[0].Column)
}

func TestCheckNotNullDefaultsCaseInsensitive(t *testing.T) {
	sql := "alter table Users add column Email Varchar(255) not null;"
	v, err := CheckNotNullDefaults(sql)
	require.NoError(t, err)
	require.Len(t, v, 1)
	require.False(t, v[0].HasDefault)
}

func TestSplitSQLStatementsHandlesSemicolonsInStrings(t *testing.T) {
	parts := splitSQLStatements(`INSERT INTO t VALUES ('a;b'); INSERT INTO t VALUES ('c');`)
	require.Len(t, parts, 2)
	require.True(t, strings.Contains(parts[0], "('a;b')"))
}

func TestStripSQLCommentsRemovesLineComments(t *testing.T) {
	got := stripSQLComments("SELECT 1; -- inline comment\nSELECT 2;")
	require.False(t, strings.Contains(got, "inline comment"))
	require.True(t, strings.Contains(got, "SELECT 1"))
}