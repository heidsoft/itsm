package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FilesystemMigrations 包外可用的「从目录发现的迁移」读取与排序入口。
//
// 2026-09-08（外部审计修复）：迁移体系原本是「Go 硬编码注册表 + 孤儿 SQL 目录」双轨制，
// 导致 migrations/*.sql 自 5 月以来从未被自动加载执行（典型案例 add_missing_indexes.sql，
// 写好 4 个月、75 张表因此只剩 PK 索引）。本文件落地「目录即真相」单一源：
//
//   - 版本号 = 文件名 stem（如 20260828_create_alerts）；无 YYYYMMDD 前缀的脚本走兜底映射；
//   - 同一 version 的 *_down.sql 自动成对，RollbackSQL 字段填充；
//   - 与 RegisteredMigrations（Go 硬编码）合并去重，磁盘优先（覆盖同 version 的硬编码 SQL）；
//   - 返回值直接喂给 migrator.RunMigrations，完成自发现 → 注册 → 执行 → 登记闭环。
//
// 单一源治理收益：未来新增 SQL 只需放文件，无需改 Go 代码；CI 扫描即可发现未登记迁移。
//
// 注意：为了避免运行时依赖（含 prod 镜像）突然多出未知的副作用，本发现器只读真实磁盘
// 文件，不接受网络/配置注入。MIGRATIONS_DIR 环境变量仅用于测试覆盖（默认相对路径）。
func FilesystemMigrations(dir string) ([]Migration, error) {
	if dir == "" {
		dir = defaultMigrationsDir()
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir %q: %w", dir, err)
	}

	type pair struct {
		up    string
		down  string
		desc  string
		title string
	}
	ups := map[string]*pair{}
	downs := map[string]string{}
	order := []string{}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		fullPath := filepath.Join(dir, name)
		bytes, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", fullPath, err)
		}
		body := string(bytes)

		if strings.HasSuffix(name, "_down.sql") {
			base := strings.TrimSuffix(name, "_down.sql")
			downs[base] = body
			continue
		}

		version, desc := parseMigrationHeader(name, body)
		if version == "" {
			// 不是 up 文件也不是 _down，跳过；按命名规范不该有这种文件
			continue
		}
		if _, exists := ups[version]; exists {
			return nil, fmt.Errorf("duplicate migration version %q (%s)", version, name)
		}
		p := &pair{up: body, desc: desc, title: name}
		ups[version] = p
		order = append(order, version)
	}

	sort.Strings(order)

	out := make([]Migration, 0, len(order))
	for _, v := range order {
		p := ups[v]
		out = append(out, Migration{
			Version:     v,
			Description: p.desc,
			RollbackSQL: downs[v],
			SQLContent:  p.up,
		})
	}
	return out, nil
}

// parseMigrationHeader 从文件名 + 文件首两行注释中解析版本号与描述。
//
// 约定（与历史迁移风格对齐）：
//
//	文件名 = 版本号（不含 .sql 后缀）；
//	若文件名无 YYYYMMDD_ 前缀，使用文件名本体作为版本号；
//	描述取首条非空 -- 注释（去除 -- 前缀与首尾空白），缺省回退到文件名。
func parseMigrationHeader(name, body string) (string, string) {
	version := strings.TrimSuffix(name, ".sql")
	if !strings.HasPrefix(version, "20") && version != "add_missing_indexes" {
		// 非日期化且非已知别名 → 拒绝：避免误把 README.md.txt 之类当迁移
		return "", ""
	}
	desc := version
	for _, line := range strings.SplitN(body, "\n", 8) {
		l := strings.TrimSpace(line)
		if strings.HasPrefix(l, "--") {
			comment := strings.TrimSpace(strings.TrimPrefix(l, "--"))
			if comment != "" && !strings.HasPrefix(comment, "===") {
				desc = comment
				break
			}
		}
	}
	return version, desc
}

func defaultMigrationsDir() string {
	if v := os.Getenv("MIGRATIONS_DIR"); v != "" {
		return v
	}
	// 相对 cwd 解析，prod 二进制工作目录为 /app
	return "migrations"
}

// MergeWithRegistered 以磁盘为真相：磁盘版本覆盖 RegisteredMigrations 中同名条目；
// 磁盘新增条目追加；RegisteredMigrations 与 LegacyMigrations 中独有且 GetMigrationSQL
// 非空的条目保留（向后兼容 Go 内嵌 SQL 迁移，例如 002-006/007-021）。
//
// 返回值按 Version 排序，方便重复执行幂等。
func MergeWithRegistered(fs []Migration) []Migration {
	byVersion := map[string]Migration{}
	all := append([]Migration{}, RegisteredMigrations...)
	all = append(all, LegacyMigrations...)
	for _, m := range all {
		if GetMigrationSQL(m.Version) != "" {
			if _, dup := byVersion[m.Version]; !dup {
				byVersion[m.Version] = m
			}
		}
	}
	for _, m := range fs {
		if m.SQLContent != "" {
			byVersion[m.Version] = m
		}
	}
	out := make([]Migration, 0, len(byVersion))
	for _, m := range byVersion {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out
}