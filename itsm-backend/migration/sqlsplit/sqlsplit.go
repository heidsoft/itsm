package sqlsplit

import (
	"strings"
)

// SplitSQLStatements 按语句边界切分 SQL 脚本。
//
// 与朴素 strings.Split(sql, ";") 的区别：分号只有出现在「语句之外」时才是分隔符。
// 以下位置的分号不切：
//   - 单引号字符串字面量（含 ” 转义）：'it”s; fine'
//   - 双引号标识符（含 "" 转义）："weird;name"
//   - dollar-quoted 体（$$...$$ 与 $tag$...$tag$）：DO $$ BEGIN ...; ... END $$;
//   - 行注释（-- 到行尾）与块注释（/* ... */，PostgreSQL 允许嵌套）。
//
// 背景（2026-10-30 fresh-install-gate run 40985843148 实证）：
//
//	迁移执行器对含 CONCURRENTLY 的脚本走「事务外逐语句执行」路径，而该路径原先用
//	strings.Split(sql, ";") 切分。20261028_rls_core_tables.sql 里的
//	DO $$ ... $$; 块因此被切碎，PostgreSQL 报
//	"unterminated dollar-quoted string at or near $$"，整条初始化链中断。
//	本函数复用 tx_control.go 的词法原语（scanSingleQuoted / scanDollarQuoted /
//	dollarTagLength / skipBlockComment），与事务控制剥离共用同一套扫描语义——
//	两处若各写一套，切分器与剥离器会各自漂移，最终对「什么是一条语句」的理解不一致。
func SplitStatements(sql string) []string {
	var out []string
	var b strings.Builder
	b.Grow(len(sql))

	n := len(sql)
	i := 0
	for i < n {
		switch c := sql[i]; {
		case c == '\'':
			j := ScanSingleQuoted(sql, i)
			b.WriteString(sql[i:j])
			i = j
		case c == '"':
			j := scanDoubleQuoted(sql, i)
			b.WriteString(sql[i:j])
			i = j
		case c == '$' && DollarTagLength(sql, i) > 0:
			j := ScanDollarQuoted(sql, i)
			b.WriteString(sql[i:j])
			i = j
		case c == '-' && i+1 < n && sql[i+1] == '-':
			j := i
			for j < n && sql[j] != '\n' {
				j++
			}
			b.WriteString(sql[i:j])
			i = j
		case c == '/' && i+1 < n && sql[i+1] == '*':
			j := SkipBlockComment(sql, i)
			b.WriteString(sql[i:j])
			i = j
		case c == ';':
			out = append(out, b.String())
			b.Reset()
			i++
		default:
			b.WriteByte(c)
			i++
		}
	}
	if rest := strings.TrimSpace(b.String()); rest != "" {
		out = append(out, rest)
	}
	return out
}

// StripSQLComments 去掉 SQL 里的注释（行注释与块注释），字符串字面量与
// dollar-quoted 体的内容原样保留。
//
// 用途：判定一段 SQL 是否「真的」含有某个关键字时，必须先去掉注释——
// 否则注释里一句「本文件无 CONCURRENTLY」就会把迁移误判为需要非事务路径
// （2026-10-30 fresh-install-gate 实证：20261028_rls_core_tables.sql 的
// 第 26 行注释正是这么触发了错误路径）。
func StripComments(sql string) string {
	var b strings.Builder
	b.Grow(len(sql))

	n := len(sql)
	i := 0
	for i < n {
		switch c := sql[i]; {
		case c == '\'':
			j := ScanSingleQuoted(sql, i)
			b.WriteString(sql[i:j])
			i = j
		case c == '"':
			j := scanDoubleQuoted(sql, i)
			b.WriteString(sql[i:j])
			i = j
		case c == '$' && DollarTagLength(sql, i) > 0:
			j := ScanDollarQuoted(sql, i)
			b.WriteString(sql[i:j])
			i = j
		case c == '-' && i+1 < n && sql[i+1] == '-':
			for i < n && sql[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < n && sql[i+1] == '*':
			i = SkipBlockComment(sql, i)
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String()
}

// scanDoubleQuoted 返回 i 处双引号标识符（含成对 "" 转义）的结束偏移。
func scanDoubleQuoted(sql string, i int) int {
	n := len(sql)
	j := i + 1
	for j < n {
		if sql[j] == '"' {
			if j+1 < n && sql[j+1] == '"' {
				j += 2
				continue
			}
			return j + 1
		}
		j++
	}
	return n
}

// ScanSingleQuoted 返回 i 处单引号字符串字面量（含成对 ” 转义）的结束偏移。
func ScanSingleQuoted(sql string, i int) int {
	n := len(sql)
	j := i + 1
	for j < n {
		if sql[j] == '\'' {
			if j+1 < n && sql[j+1] == '\'' {
				j += 2
				continue
			}
			return j + 1
		}
		j++
	}
	return n
}

// DollarTagLength 返回 i 处 dollar-quote 开启标签（$$ 或 $tag$）的长度；不是标签则返回 0。
func DollarTagLength(sql string, i int) int {
	n := len(sql)
	j := i + 1
	for j < n && (isASCIILetter(sql[j]) || isASCIIDigit(sql[j]) || sql[j] == '_') {
		j++
	}
	if j < n && sql[j] == '$' {
		return j - i + 1
	}
	return 0
}

// ScanDollarQuoted 返回 i 处 dollar-quoted 体（含首尾标签）的结束偏移；
// 未闭合时原样保留到文件尾，避免误截断。
func ScanDollarQuoted(sql string, i int) int {
	tagLen := DollarTagLength(sql, i)
	tag := sql[i : i+tagLen]
	if k := strings.Index(sql[i+tagLen:], tag); k >= 0 {
		return i + tagLen + k + tagLen
	}
	return len(sql)
}

// SkipBlockComment 返回 i 处块注释（/* ... */）的结束偏移。
func SkipBlockComment(sql string, i int) int {
	n := len(sql)
	j := i + 2
	for j+1 < n && !(sql[j] == '*' && sql[j+1] == '/') {
		j++
	}
	if j+1 < n {
		return j + 2
	}
	return n
}

func isASCIILetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isASCIIDigit(c byte) bool {
	return c >= '0' && c <= '9'
}
