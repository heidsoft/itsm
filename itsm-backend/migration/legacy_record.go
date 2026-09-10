package migration

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// 本文件承载「迁移账本调和」：把 unified migration 体系落地（2026-09-08 目录
// 自发现 / 更早的 Go 注册表记账）之前就已生效、但从未记账的历史迁移幂等补记
// 进 schema_migrations——只登记，不执行（adopt, don't replay）。
//
// 两个调和面（2026-09-10 prod init 日志实证）：
//  1. LegacyMigrations 001-006：Go 注册表之前的历史版本，内嵌 SQL 引用旧表名
//     （002 的 tenant 表——现库为 tenants），重放必炸；
//  2. 日期化磁盘迁移（migrations/2026*.sql）：发现机制上线前效果已由 ent 基线
//     或手工执行覆盖，但账本 0 登记——发现机制把它们当 pending 重放，撞
//     NOT NULL/已存在约束（如 20260501_enable_rbac_from_db 的 23502）。
//
// 判别「既有安装 vs 全新安装」：看账本里 unified 流（007+）的最早 applied_at。
// unified 记账在 2026-09-05 的 init 部署中产生；日期化迁移文件最晚为 2026-09-08
// 进入目录。既有安装的最早记账时间必然早于「发现机制上线」，此时所有日期化
// 迁移按定义都是历史（收养）；全新安装则尚无 007+ 记账（或记账晚于发现机制
// 上线），日期化迁移按正常 pending 执行。
//
// 幂等性：version 是主键，已存在的记录跳过；纯 INSERT，无副作用，可安全重入。
// 注意：收养条目有意不带 checksum（历史 SQL 不再执行，无从校验），执行查询
// 用「存在即跳过」而非「checksum 比对」语义。

// adoptionCutoffUTC 是目录自发现机制上线的时刻（2026-09-08 发布窗口）。
// 最早 unified 记账早于该时刻 → 既有安装；否则 → 全新安装。
var adoptionCutoffUTC = time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)

// recordLegacyMigrationsApplied 补记 legacy 001-006（无条件收养）。
func recordLegacyMigrationsApplied(ctx context.Context, db *sql.DB, logger legacyLogger) error {
	for _, m := range LegacyMigrations {
		recorded, err := recordMigrationAdopted(ctx, db, m.Version, m.Description, m.RollbackSQL, "legacy")
		if err != nil {
			return err
		}
		if recorded && logger != nil {
			logger.Infow("legacy migration recorded (adopted, not executed)",
				"version", m.Version)
		}
	}
	return nil
}

// adoptUnrecordedFilesystemMigrations 收养既有安装上未记账的日期化磁盘迁移。
//
// fs 为 FilesystemMigrations 的发现结果；函数只处理「既有安装」判定为真的库。
func adoptUnrecordedFilesystemMigrations(ctx context.Context, db *sql.DB, fs []Migration, logger legacyLogger) (int, error) {
	existing, err := isExistingInstallation(ctx, db)
	if err != nil {
		return 0, err
	}
	if !existing {
		return 0, nil
	}
	adopted := 0
	for _, m := range fs {
		if !isPreDiscoveryVersion(m.Version) {
			continue // 只收养发现机制上线前已存在的历史版本
		}
		recorded, err := recordMigrationAdopted(ctx, db, m.Version, m.Description, m.RollbackSQL, "adopted")
		if err != nil {
			return adopted, err
		}
		if recorded {
			adopted++
			if logger != nil {
				logger.Infow("pre-discovery filesystem migration adopted (recorded, not executed)",
					"version", m.Version)
			}
		}
	}
	return adopted, nil
}

// isExistingInstallation 判定当前库是否为「发现机制上线前的既有安装」：
// unified 流（007 起）的最早 applied_at 早于 adoptionCutoffUTC。
func isExistingInstallation(ctx context.Context, db *sql.DB) (bool, error) {
	const q = `
		SELECT EXISTS (
			SELECT 1 FROM schema_migrations
			WHERE version >= '007' AND applied_at < $1
		)`
	var exists bool
	if err := db.QueryRowContext(ctx, q, adoptionCutoffUTC).Scan(&exists); err != nil {
		return false, fmt.Errorf("detect existing installation: %w", err)
	}
	return exists, nil
}

// isPreDiscoveryVersion 判定版本名是否属于发现机制上线前已存在的迁移：
// 日期化版本（YYYYMMDD_ 前缀）的日期必须早于 cutoff（否则是既有安装上的
// 新迁移，必须正常执行而非收养——2026-09-11 复盘修正：原实现只看前缀不看
// 日期，会把新批次误收养成"已执行"，静默跳过 DDL）；
// 另收养已知的非日期化历史别名。
func isPreDiscoveryVersion(version string) bool {
	if version == "add_missing_indexes" {
		return true
	}
	if len(version) < 8 {
		return false
	}
	prefix := version[:8]
	if !allDigits(prefix) {
		return false
	}
	d, err := time.ParseInLocation("20060102", prefix, time.UTC)
	if err != nil {
		return false
	}
	return d.Before(adoptionCutoffUTC)
}

func allDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}

// recordMigrationAdopted 幂等插入一条「已收养」账本记录；返回是否新插入。
func recordMigrationAdopted(ctx context.Context, db *sql.DB, version, description, rollbackSQL, releaseTag string) (bool, error) {
	var exists bool
	if err := db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`,
		version,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("check migration %s: %w", version, err)
	}
	if exists {
		return false, nil
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO schema_migrations
			(version, description, applied_at, rollback_sql, checksum, execution_ms, release_version)
		VALUES ($1, $2, $3, $4, '', 0, $5)
	`, version, description, time.Now(), rollbackSQL, releaseTag)
	if err != nil {
		return false, fmt.Errorf("record migration %s: %w", version, err)
	}
	return true, nil
}

// RecordLegacyMigrationsApplied 导出给 bootstrap 使用的 legacy 补账入口。
func RecordLegacyMigrationsApplied(ctx context.Context, db *sql.DB, logger legacyLogger) error {
	return recordLegacyMigrationsApplied(ctx, db, logger)
}

// AdoptUnrecordedFilesystemMigrations 导出给 bootstrap 的日期化迁移收养入口。
func AdoptUnrecordedFilesystemMigrations(ctx context.Context, db *sql.DB, fs []Migration, logger legacyLogger) (int, error) {
	return adoptUnrecordedFilesystemMigrations(ctx, db, fs, logger)
}

// legacyLogger 最小日志接口，避免与具体日志库耦合（zap.SugaredLogger 天然满足）。
type legacyLogger interface {
	Infow(msg string, keysAndValues ...interface{})
}
