package migration

import (
	"regexp"
	"strings"

	"itsm-backend/migration/sqlsplit"
)

// txControlStmtRe 匹配语句边界上的事务控制语句（BEGIN;/COMMIT;/START TRANSACTION;/
// BEGIN TRANSACTION;，大小写不敏感）。只锚定语句起始位置，不会命中函数体内的
// BEGIN/END 块标记或字符串字面量——那些分别被 dollar-quote 扫描和字符串扫描整段原样复制。
var txControlStmtRe = regexp.MustCompile(`^(?i:BEGIN(?:[ \t]+TRANSACTION)?|COMMIT|START[ \t]+TRANSACTION)[ \t]*;`)

// stripEmbeddedTxControl 剥离迁移脚本自管理的事务控制语句。
//
// 背景：ApplyMigration/RollbackMigration 用 BeginTx 托管事务边界。脚本内嵌的 COMMIT;
// 会提前提交托管事务，随后账本 INSERT 落入自动提交，最终 tx.Commit 命中空闲连接报错
// （lib/pq "unexpected transaction status idle"），后续尚未执行的迁移会被整体中断。
// 剥离后脚本在托管事务内原子执行，与不带事务控制的迁移行为一致。
//
// checksum 必须基于剥离前的原始 SQL 计算（schema_migrations 账本校验依赖原始内容），
// 因此本函数只用于执行路径，不影响 checksumSQL 的输入。
func stripEmbeddedTxControl(sql string) string {
	var out strings.Builder
	out.Grow(len(sql))

	n := len(sql)
	i := 0
	atStmtStart := true

	for i < n {
		if atStmtStart {
			j := skipWhitespaceAndComments(sql, i)
			out.WriteString(sql[i:j])
			i = j
			if i >= n {
				break
			}
			if loc := txControlStmtRe.FindStringIndex(sql[i:]); loc != nil && loc[0] == 0 {
				i += loc[1]
				continue // 保持 atStmtStart，允许连续剥离
			}
			atStmtStart = false
			continue
		}

		switch c := sql[i]; {
		case c == '\'':
			j := sqlsplit.ScanSingleQuoted(sql, i)
			out.WriteString(sql[i:j])
			i = j
		case c == '$' && sqlsplit.DollarTagLength(sql, i) > 0:
			j := sqlsplit.ScanDollarQuoted(sql, i)
			out.WriteString(sql[i:j])
			i = j
		case c == '-' && i+1 < n && sql[i+1] == '-':
			j := i
			for j < n && sql[j] != '\n' {
				j++
			}
			out.WriteString(sql[i:j])
			i = j
		case c == '/' && i+1 < n && sql[i+1] == '*':
			j := sqlsplit.SkipBlockComment(sql, i)
			out.WriteString(sql[i:j])
			i = j
		case c == ';':
			out.WriteByte(c)
			i++
			atStmtStart = true
		default:
			out.WriteByte(c)
			i++
		}
	}
	return out.String()
}

func skipWhitespaceAndComments(sql string, i int) int {
	n := len(sql)
	for i < n {
		switch c := sql[i]; {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '-' && i+1 < n && sql[i+1] == '-':
			for i < n && sql[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < n && sql[i+1] == '*':
			i = sqlsplit.SkipBlockComment(sql, i)
		default:
			return i
		}
	}
	return i
}
