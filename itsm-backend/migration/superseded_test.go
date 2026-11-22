package migration

import (
	"context"
	"database/sql"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 本文件锁住 checksum 门禁的唯一例外通道（superseded.go）。
//
// 不变量：
//  1. 已发布迁移内容被评审修正后，既有安装不得因 checksum 变化被拦在升级门外；
//  2. 例外只对「评审过的那一份内容」生效，任何再次改动重新变回硬失败；
//  3. 没有登记过的版本，checksum 不匹配仍然硬失败（不得因为引入例外而放宽全局）。

const appliedMigrationsQuery = `SELECT version, description, applied_at, rollback_sql, checksum, execution_ms, release_version FROM schema_migrations ORDER BY version`

func newAppliedLedger(t *testing.T, rows ...appliedRow) *sql.DB {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	result := sqlmock.NewRows([]string{
		"version", "description", "applied_at", "rollback_sql",
		"checksum", "execution_ms", "release_version",
	})
	for _, r := range rows {
		result.AddRow(r.version, "desc", time.Now(), nil, r.checksum, int64(0), "v-test")
	}
	mock.ExpectQuery(regexp.QuoteMeta(appliedMigrationsQuery)).WillReturnRows(result)
	mock.ExpectClose()
	return db
}

type appliedRow struct {
	version  string
	checksum string
}

func pendingWith(t *testing.T, db *sql.DB, available []Migration) error {
	t.Helper()
	m := NewMigrator(db, nil)
	_, err := m.GetPendingMigrations(context.Background(), available)
	return err
}

// repositoryMigration 读取磁盘上的真实迁移，避免测试内联一份可能与文件不一致的 SQL。
func repositoryMigration(t *testing.T, version string) Migration {
	t.Helper()
	dir := os.Getenv("MIGRATIONS_DIR")
	if dir == "" {
		dir = "../migrations"
	}
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("skip: migrations dir not available at %q (%v)", dir, err)
	}
	migs, err := FilesystemMigrations(dir)
	require.NoError(t, err)
	for _, m := range migs {
		if m.Version == version {
			return m
		}
	}
	t.Fatalf("migration %s not discovered on disk", version)
	return Migration{}
}

// 例外通道自身必须与仓库里的真实文件对齐：登记的白名单 checksum 一旦与文件不符，
// 说明有人又改了已发布迁移却没有同步评审记录——在 CI 就失败，而不是等到客户升级。
func TestSupersededMigrations_AllowedChecksumMatchesRepositoryFile(t *testing.T) {
	require.NotEmpty(t, supersededMigrations, "白名单为空时本测试失去意义")

	for version, sup := range supersededMigrations {
		mig := repositoryMigration(t, version)
		require.Equal(t, sup.AllowedChecksum, checksumSQL(mig.SQLContent),
			"迁移 %s 的已发布内容又被改动，但 superseded.go 未同步；"+
				"已应用该版本的实例会在下次升级时报 checksum mismatch", version)
		require.NotEmpty(t, sup.Reason, "迁移 %s 的例外必须写明原因", version)
	}
}

func TestPendingMigrations_AllowsReviewedSupersededContent(t *testing.T) {
	const version = "20261118_mcp_read_permission"
	mig := repositoryMigration(t, version)

	// 账本里是修正前的旧 checksum：v1.6.59+ 已应用该版本的实例就是这样。
	db := newAppliedLedger(t, appliedRow{version: version, checksum: "3055378b27e7ef2cd420ea84c17ff0326ec4d90a742e46666fe73778e1f02edb"})

	require.NoError(t, pendingWith(t, db, []Migration{mig}),
		"已评审修正的迁移不得阻断既有安装升级")
}

func TestPendingMigrations_RejectsUnreviewedDriftOfSupersededVersion(t *testing.T) {
	const version = "20261118_mcp_read_permission"
	mig := repositoryMigration(t, version)
	// 在白名单版本上再改一次内容：不在评审范围内，必须继续硬失败。
	mig.SQLContent += "\n-- unreviewed change\n"

	db := newAppliedLedger(t, appliedRow{version: version, checksum: "3055378b27e7ef2cd420ea84c17ff0326ec4d90a742e46666fe73778e1f02edb"})

	err := pendingWith(t, db, []Migration{mig})
	require.Error(t, err, "白名单只放行评审过的那一份内容，不得变成通用跳过开关")
	require.Contains(t, err.Error(), "checksum mismatch")
}

func TestPendingMigrations_RejectsDriftOfOrdinaryMigration(t *testing.T) {
	mig := Migration{Version: "20260101_ordinary", SQLContent: "SELECT 1;"}
	db := newAppliedLedger(t, appliedRow{version: mig.Version, checksum: "deadbeef"})

	err := pendingWith(t, db, []Migration{mig})
	require.Error(t, err, "未登记版本必须保持原有硬失败语义")
	assert.Contains(t, err.Error(), "checksum mismatch")
}

// 内容未变时不应触发任何告警路径，也不得把版本重新排进 pending。
func TestPendingMigrations_UnchangedContentStaysPending(t *testing.T) {
	const version = "20261118_mcp_read_permission"
	mig := repositoryMigration(t, version)

	db := newAppliedLedger(t, appliedRow{version: version, checksum: checksumSQL(mig.SQLContent)})
	m := NewMigrator(db, nil)
	pending, err := m.GetPendingMigrations(context.Background(), []Migration{mig})
	require.NoError(t, err)
	assert.Empty(t, pending)
}
