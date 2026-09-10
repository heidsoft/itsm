// Command migration-lint 串联三件事：
//  1. 从 ./ent/schema 抽取 PII 注解生成 (table, column) → Strategy Policy；
//  2. 给定迁移 SQL，校验所有新增 NOT NULL 列是否带 DEFAULT（缺则拒绝合并）；
//  3. 输出每个 PII 表的「脱敏副本」SQL，供运维灌入预发 / 开发环境。
//
// 用法：
//
//	go run ./cmd/migration-lint -schema-dir ./ent/schema -migration-sql ./migrations/20260830_x.sql
//
// 输出纯文本报告，便于 CI / pre-commit hook 解析。
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"entgo.io/ent/entc/load"

	"itsm-backend/migration/pii"
)

func main() {
	schemaDir := flag.String("schema-dir", "./ent/schema", "ent schema 目录")
	migrationPath := flag.String("migration-sql", "", "要校验的迁移 SQL 文件路径（可选）")
	table := flag.String("table", "", "只生成指定表的脱敏副本 SQL（默认全量）")
	strict := flag.Bool("strict", true, "缺 DEFAULT / 缺 PII 策略时 exit 1")
	flag.Parse()

	spec, err := (&load.Config{Path: *schemaDir}).Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load schema %s: %v\n", *schemaDir, err)
		os.Exit(2)
	}
	descriptors, err := pii.ExtractFromLoadedSchema(spec.Schemas)
	if err != nil {
		fmt.Fprintf(os.Stderr, "extract descriptors: %v\n", err)
		os.Exit(2)
	}
	policy, err := pii.PolicyFromDescriptors(descriptors)
	if err != nil {
		fmt.Fprintf(os.Stderr, "policy: %v\n", err)
		os.Exit(2)
	}

	failed := false
	fmt.Printf("== PII policy ==\n")
	if policy.Empty() {
		fmt.Println("  (none)")
	} else {
		for table, cols := range policy.TableColumns {
			strats := policy.Strategies[table]
			if len(strats) == 0 {
				continue
			}
			fmt.Printf("  %s:\n", table)
			for _, col := range cols {
				s, ok := strats[col]
				if !ok {
					continue
				}
				fmt.Printf("    - %s: %s\n", col, s)
			}
		}
	}

	if *migrationPath != "" {
		fmt.Printf("\n== NOT NULL / DEFAULT check (%s) ==\n", *migrationPath)
		sql, err := os.ReadFile(*migrationPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read migration %s: %v\n", *migrationPath, err)
			os.Exit(2)
		}
		violations, err := pii.CheckNotNullDefaults(string(sql))
		if err != nil {
			fmt.Fprintf(os.Stderr, "check: %v\n", err)
			os.Exit(2)
		}
		if len(violations) == 0 {
			fmt.Println("  OK: every new NOT NULL column carries DEFAULT")
		} else {
			for _, v := range violations {
				if v.HasDefault {
					fmt.Println("  OK     " + v.String())
				} else {
					fmt.Println("  MISS   " + v.String())
					failed = true
				}
			}
		}
	}

	fmt.Printf("\n== Masked copy SQL ==\n")
	for _, t := range sortedKeys(policy.TableColumns) {
		if *table != "" && t != *table {
			continue
		}
		if _, ok := policy.Strategies[t]; !ok {
			continue
		}
		sql, err := pii.MaskedCopySQL(t, nil, policy, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "masked copy %s: %v\n", t, err)
			failed = true
			continue
		}
		fmt.Printf("\n-- mask_%s --\n%s\n", t, sql)
	}

	if failed && *strict {
		os.Exit(1)
	}
}

func sortedKeys(m map[string][]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	// simple ascii sort; reader-friendly enough for CI output
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if strings.Compare(out[i], out[j]) > 0 {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}