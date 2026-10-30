package sqlsplit

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSplitSQLStatements_RespectsDollarQuotedAndQuotedSemicolons 锁定切分器的
// 词法边界：只有「语句之外」的分号才是分隔符。
//
// 背景（2026-10-30 fresh-install-gate run 40985843148 实证）：非事务路径原先用
// strings.Split(sql, ";") 切分，DO $$ ... $$; 块被切碎，PostgreSQL 报
// "unterminated dollar-quoted string at or near $$"，全新安装中断。
func TestSplitSQLStatements_RespectsDollarQuotedAndQuotedSemicolons(t *testing.T) {
	cases := []struct {
		name  string
		sql   string
		want  int
		first string
	}{
		{
			name:  "dollar-quoted DO 块整体成一条",
			sql:   "DO $$ BEGIN RAISE NOTICE 'x'; END $$; SELECT 1;",
			want:  2,
			first: "DO $$ BEGIN RAISE NOTICE 'x'; END $$",
		},
		{
			name:  "带标签的 dollar-quoted 同样整体成一条",
			sql:   "CREATE FUNCTION f() RETURNS int AS $function$ BEGIN RETURN 1; END $function$ LANGUAGE plpgsql;",
			want:  1,
			first: "CREATE FUNCTION f() RETURNS int AS $function$ BEGIN RETURN 1; END $function$ LANGUAGE plpgsql",
		},
		{
			name:  "单引号内的分号不切",
			sql:   "INSERT INTO t VALUES ('a;b'); SELECT 1;",
			want:  2,
			first: "INSERT INTO t VALUES ('a;b')",
		},
		{
			name:  "单引号的成对转义不提前结束",
			sql:   "INSERT INTO t VALUES ('it''s;ok'); SELECT 1;",
			want:  2,
			first: "INSERT INTO t VALUES ('it''s;ok')",
		},
		{
			name:  "双引号标识符内的分号不切",
			sql:   `CREATE TABLE "odd;name" (id int); SELECT 1;`,
			want:  2,
			first: `CREATE TABLE "odd;name" (id int)`,
		},
		{
			name:  "行注释里的分号不切",
			sql:   "SELECT 1 -- trailing ; comment\n; SELECT 2;",
			want:  2,
			first: "SELECT 1 -- trailing ; comment",
		},
		{
			name:  "块注释里的分号不切",
			sql:   "/* a; b */ SELECT 1;",
			want:  1,
			first: "/* a; b */ SELECT 1",
		},
		{
			name:  "普通多语句按分号切",
			sql:   "SELECT 1; SELECT 2; SELECT 3;",
			want:  3,
			first: "SELECT 1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SplitStatements(tc.sql)
			require.Len(t, got, tc.want, "切分条数不符：%q", tc.sql)
			assert.Equal(t, tc.first, strings.TrimSpace(got[0]))
		})
	}
}

// TestStripSQLComments_KeepsStringContent 锁定去注释不伤字符串内容。
func TestStripSQLComments_KeepsStringContent(t *testing.T) {
	cases := []struct {
		name string
		sql  string
		want string
	}{
		{"行注释被去掉", "SELECT 1 -- comment\nFROM t", "SELECT 1 \nFROM t"},
		{"字符串里的双横线保留", "SELECT '--not-a-comment'", "SELECT '--not-a-comment'"},
		{"字符串里的斜杠星号保留", "SELECT '/* not a block */'", "SELECT '/* not a block */'"},
		{"块注释被去掉", "/* c */ SELECT 1", " SELECT 1"},
		{"dollar-quoted 内的注释保留", "DO $$ -- keep\n BEGIN END $$", "DO $$ -- keep\n BEGIN END $$"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, StripComments(tc.sql))
		})
	}
}

// TestConcurrentlyDetectionIgnoresComments 锁定「是否需要非事务路径」的判定
// 不受注释影响。
//
// 背景（2026-10-30 fresh-install-gate run 40985843148 实证）：
// 20261028_rls_core_tables.sql 第 26 行注释写着「无 CONCURRENTLY / 数据回填」，
// 全文匹配把这条本可在事务内执行的迁移误判到非事务路径，而该路径的逐语句执行
// 会切碎 DO $$ ... $$ 块——一句注释换来一次全新安装中断。
func TestConcurrentlyDetectionIgnoresComments(t *testing.T) {
	commentOnly := "-- 本文件不含 CONCURRENTLY / 数据回填\nCREATE POLICY p ON tickets USING (tenant_id = 1);"
	assert.NotContains(t, strings.ToUpper(StripComments(commentOnly)), "CONCURRENTLY",
		"注释里的 CONCURRENTLY 不应被当成真实语句")

	realUsage := "CREATE INDEX CONCURRENTLY idx_tickets_ticket_number ON tickets (ticket_number);"
	assert.Contains(t, strings.ToUpper(StripComments(realUsage)), "CONCURRENTLY",
		"真实的 CREATE INDEX CONCURRENTLY 必须被检出")
}
