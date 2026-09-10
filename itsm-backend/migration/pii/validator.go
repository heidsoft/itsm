package pii

import (
	"fmt"
	"regexp"
	"strings"
)

// NotNullViolation 描述一个 NOT NULL 但缺少 DEFAULT 的新列，
// 即不显式填值则 INSERT 老数据会失败。
type NotNullViolation struct {
	Table      string
	Column     string
	Type       string
	HasDefault bool
}

// ViolationsText 用于 CLI 输出；行间换行符兼容普通终端与 CI 日志。
func (v NotNullViolation) String() string {
	marker := "MISSING_DEFAULT"
	if v.HasDefault {
		marker = "OK_WITH_DEFAULT"
	}
	return fmt.Sprintf("%s table=%s column=%s type=%s", marker, v.Table, v.Column, v.Type)
}

// CheckNotNullDefaults 解析一段迁移 SQL，检测所有新增 NOT NULL 列是否带 DEFAULT。
// 当前仅支持 PostgreSQL 与 SQLite 风格的简单 ALTER TABLE ADD COLUMN。
// 形如：
//
//	ALTER TABLE foo ADD COLUMN bar varchar(255) NOT NULL;
//	ALTER TABLE foo ADD COLUMN bar varchar(255) NOT NULL DEFAULT '';
//	ALTER TABLE foo ADD COLUMN bar int NOT NULL DEFAULT 0;
//	ALTER TABLE foo ADD COLUMN bar boolean NOT NULL DEFAULT false;
//	ALTER TABLE foo ADD COLUMN bar timestamp NOT NULL DEFAULT now();
//	ALTER TABLE foo ADD COLUMN bar bigint;  -- 可空 → OK
//
// 多语句支持：分号分隔；注释行（-- 开头）跳过。
func CheckNotNullDefaults(sql string) ([]NotNullViolation, error) {
	var violations []NotNullViolation
	stmts := splitSQLStatements(sql)
	addRe := regexp.MustCompile(`(?is)\bALTER\s+TABLE\s+(?:` + "`" + `|"+)?([a-zA-Z0-9_]+)(?:` + "`" + `|"+)?\s+ADD\s+COLUMN\s+(?:` + "`" + `|"+)?([a-zA-Z0-9_]+)(?:` + "`" + `|"+)?\s+([^\n;]+)`)
	for _, stmt := range stmts {
		stripped := stripSQLComments(stmt)
		if stripped == "" {
			continue
		}
		matches := addRe.FindAllStringSubmatch(stripped, -1)
		for _, m := range matches {
			table := strings.Trim(m[1], `"`+"`")
			col := strings.Trim(m[2], `"`+"`")
			rest := strings.ToLower(m[3])
			hasNotNull := strings.Contains(rest, "not null")
			if !hasNotNull {
				continue
			}
			hasDefault := strings.Contains(rest, "default ")
			violations = append(violations, NotNullViolation{
				Table:      table,
				Column:     col,
				Type:       firstWord(rest),
				HasDefault: hasDefault,
			})
		}
	}
	return violations, nil
}

// splitSQLStatements 按分号切 SQL；保留 $tag$ 与引号内的分号不被误切。
func splitSQLStatements(sql string) []string {
	out := []string{}
	var b strings.Builder
	inSingle, inDouble, inDollar := false, false, false
	for i := 0; i < len(sql); i++ {
		c := sql[i]
		switch {
		case inDollar:
			b.WriteByte(c)
			if c == '$' && i+1 < len(sql) && (sql[i+1] == '$' || isAlphaNumeric(sql[i+1])) {
				// 简化：dollar 标签结束条件忽略；正常 SQL 中 ADD COLUMN 不会落在 dollar 体内
				if i+1 < len(sql) && sql[i+1] == '$' {
					b.WriteByte(sql[i+1])
					i++
					inDollar = false
				}
			}
		case inSingle:
			b.WriteByte(c)
			if c == '\'' {
				inSingle = false
			}
		case inDouble:
			b.WriteByte(c)
			if c == '"' {
				inDouble = false
			}
		case c == '\'':
			b.WriteByte(c)
			inSingle = true
		case c == '"':
			b.WriteByte(c)
			inDouble = true
		case c == '$':
			b.WriteByte(c)
			if i+1 < len(sql) && sql[i+1] == '$' {
				b.WriteByte(sql[i+1])
				i++
				inDollar = true
			}
		case c == ';':
			out = append(out, b.String())
			b.Reset()
		default:
			b.WriteByte(c)
		}
	}
	if rest := strings.TrimSpace(b.String()); rest != "" {
		out = append(out, rest)
	}
	return out
}

// stripSQLComments 去掉行注释；-- 之后整行（不含引号内）的字符丢弃。
func stripSQLComments(sql string) string {
	var b strings.Builder
	inSingle, inDouble := false, false
	for i := 0; i < len(sql); i++ {
		c := sql[i]
		switch {
		case inSingle:
			b.WriteByte(c)
			if c == '\'' {
				inSingle = false
			}
		case inDouble:
			b.WriteByte(c)
			if c == '"' {
				inDouble = false
			}
		case c == '\'':
			b.WriteByte(c)
			inSingle = true
		case c == '"':
			b.WriteByte(c)
			inDouble = true
		case c == '-' && i+1 < len(sql) && sql[i+1] == '-':
			// 跳过到换行
			for i < len(sql) && sql[i] != '\n' {
				i++
			}
			if i < len(sql) {
				b.WriteByte('\n')
			}
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

func firstWord(s string) string {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func isAlphaNumeric(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}
