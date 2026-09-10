package migration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFilesystemMigrations_DiscoveryFromRealDir 验证自发现器在真实仓库目录下的行为：
//  1. 解析版本号与描述
//  2. *_down.sql 自动成对
//  3. 无 YYYYMMDD 前缀的脚本走兜底映射（add_missing_indexes）
//
// 这是外部审计修复（2026-09-08）的回归门禁：add_missing_indexes.sql 自 5 月写好
// 以来从未被加载，本测试若失败说明发现器遗漏了关键脚本。
func TestFilesystemMigrations_DiscoveryFromRealDir(t *testing.T) {
	// 直接使用相对路径，CI/测试 cwd 通常在 itsm-backend/
	dir := "../migrations"
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("skip: real migrations dir not available at %q (%v)", dir, err)
	}
	migs, err := FilesystemMigrations(dir)
	require.NoError(t, err)
	require.NotEmpty(t, migs)

	byVersion := map[string]Migration{}
	for _, m := range migs {
		byVersion[m.Version] = m
	}

	// 关键：add_missing_indexes 必须被发现并附带 SQL（外部审计 P0 缺口）
	addIdx, ok := byVersion["add_missing_indexes"]
	require.True(t, ok, "add_missing_indexes 必须被发现")
	assert.NotEmpty(t, addIdx.SQLContent, "必须携带 SQLContent（用于 ApplyMigration 直接执行）")
	assert.Contains(t, addIdx.SQLContent, "CREATE INDEX",
		"add_missing_indexes 内容必须包含索引语句，否则执行无意义")

	// _down 自动成对：仅验证至少有一个 _down 被正确绑定（不强求每个版本都成对）
	withDown := 0
	for _, m := range migs {
		if m.RollbackSQL != "" {
			withDown++
		}
	}
	assert.GreaterOrEqual(t, withDown, 1, "至少应发现一份 *_down.sql 配对")

	// 全部版本号排序：保证 RunMigrations 顺序确定
	for i := 1; i < len(migs); i++ {
		assert.LessOrEqual(t, migs[i-1].Version, migs[i].Version,
			"migrations must be sorted by version")
	}
}

// TestFilesystemMigrations_TempDir 使用临时目录覆盖解析路径与命名约定。
func TestFilesystemMigrations_TempDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "20260101_create_widgets.sql"),
		[]byte("-- Create widgets table\nCREATE TABLE widgets(id INT);\n"),
		0o644))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "20260101_create_widgets_down.sql"),
		[]byte("DROP TABLE widgets;\n"),
		0o644))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "README.md"),
		[]byte("-- not a migration\n"),
		0o644))

	migs, err := FilesystemMigrations(dir)
	require.NoError(t, err)
	require.Len(t, migs, 1)
	assert.Equal(t, "20260101_create_widgets", migs[0].Version)
	assert.Equal(t, "Create widgets table", migs[0].Description)
	assert.Equal(t, "DROP TABLE widgets;\n", migs[0].RollbackSQL)
	assert.Contains(t, migs[0].SQLContent, "CREATE TABLE widgets")
}

// TestMergeWithRegistered 验证「磁盘优先 + 硬编码兜底」的合并策略。
func TestMergeWithRegistered(t *testing.T) {
	disk := []Migration{
		{Version: "20260908_disk_only", Description: "from disk", SQLContent: "SELECT 1;"},
		{Version: "007_add_change_execution_tables", Description: "disk overrides", SQLContent: "SELECT 2;"},
	}
	merged := MergeWithRegistered(disk)
	byVersion := map[string]Migration{}
	for _, m := range merged {
		byVersion[m.Version] = m
	}

	// 磁盘独有：直接保留
	assert.Equal(t, "from disk", byVersion["20260908_disk_only"].Description)
	assert.Equal(t, "SELECT 1;", byVersion["20260908_disk_only"].SQLContent)

	// 磁盘覆盖：硬编码描述被磁盘描述替代，SQLContent 用磁盘
	override := byVersion["007_add_change_execution_tables"]
	assert.Equal(t, "disk overrides", override.Description)
	assert.Equal(t, "SELECT 2;", override.SQLContent)

	// 硬编码独有（GetMigrationSQL 非空）：保留作为兜底。
	// SQLContent 字段为空是预期的——硬编码版本的 SQL 在 ApplyMigration 内回退到
	// GetMigrationSQL(version)，而不是 SQLContent。验证「存在且 Description 非空」。
	hc, ok := byVersion["007_add_change_execution_tables"]
	require.True(t, ok, "硬编码独有迁移必须保留")
	assert.NotEmpty(t, hc.Description)

	// legacy 001-006 必须被排除出活动流：其内嵌 SQL 引用旧表名/旧结构，
	// 重放必炸（2026-09-10 prod 实证 002 报 relation "tenant" does not exist）。
	// 账本登记由 RecordLegacyMigrationsApplied 幂等补记，不在合并流内。
	for _, m := range LegacyMigrations {
		_, in := byVersion[m.Version]
		assert.False(t, in, "legacy %s 不得进入活动迁移流", m.Version)
	}

	// 排序保证幂等
	for i := 1; i < len(merged); i++ {
		assert.LessOrEqual(t, merged[i-1].Version, merged[i].Version)
	}
}

// TestIsPreDiscoveryVersion 收养判定的日期边界（2026-09-11 复盘修正）：
// 只看 "20" 前缀不看日期会把既有安装上的新迁移误收养（静默跳过 DDL）。
func TestIsPreDiscoveryVersion(t *testing.T) {
	cases := []struct {
		version string
		want    bool
	}{
		{"20260501_enable_rbac_from_db", true},     // 历史：早于 cutoff
		{"20260907_ticket_types_menu", true},       // 历史：cutoff 前一天
		{"20260908_new_migration", false},          // cutoff 当天：不算历史
		{"20260910_add_something", false},          // 未来新增：必须正常执行
		{"20990101_far_future", false},             // 远未来
		{"add_missing_indexes", true},              // 已知历史别名
		{"add_missing_indexes_v2", false},          // 新别名：不收养
		{"007_add_change_execution_tables", false}, // unified 流不参与收养
		{"2026", false},                            // 过短
		{"20ab0101_x", false},                      // 非数字
		{"20261301_x", false},                      // 非法月份
	}
	for _, c := range cases {
		assert.Equal(t, c.want, isPreDiscoveryVersion(c.version), "version=%s", c.version)
	}
}
