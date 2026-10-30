package migration

import (
	"strings"
	"testing"

	"itsm-backend/migration/sqlsplit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDiskMigrations_NonTransactionalPathKeepsDollarQuotedBlocks 锁定：走非事务
// 路径的磁盘迁移，其 dollar-quoted 块必须能完整切出。
//
// 背景（2026-10-30 fresh-install-gate run 40985843148 实证）：非事务路径原先用
// strings.Split(sql, ";") 切分，DO $$ ... $$; 块被切碎，PostgreSQL 报
// "unterminated dollar-quoted string at or near $$"，全新安装中断。
func TestDiskMigrations_NonTransactionalPathKeepsDollarQuotedBlocks(t *testing.T) {
	dir := "../migrations"
	migs, err := FilesystemMigrations(dir)
	require.NoError(t, err)
	require.NotEmpty(t, migs)

	checked := 0
	for _, mig := range migs {
		body := sqlsplit.StripComments(mig.SQLContent)
		if !strings.Contains(strings.ToUpper(body), "CONCURRENTLY") {
			continue
		}
		checked++
		// 逐语句切分后，每一条都不能出现「半个 dollar-quoted 体」的残片。
		for _, stmt := range sqlsplit.SplitStatements(mig.SQLContent) {
			trimmed := strings.TrimSpace(stmt)
			if trimmed == "" {
				continue
			}
			opens := strings.Count(trimmed, "$$")
			assert.Zero(t, opens%2,
				"迁移 %s 的语句切分后出现未闭合的 dollar-quoted 体：%q", mig.Version, trimmed)
		}
	}
	assert.NotZero(t, checked, "扫描器未命中任何含 CONCURRENTLY 的迁移——判定逻辑可能已失效")
}

// TestDiskMigrations_NoFalseConcurrentDetection 锁定：注释里提到 CONCURRENTLY
// 不会把迁移误判到非事务路径。
//
// 背景（2026-10-30 fresh-install-gate run 40985843148 实证）：
// 20261028_rls_core_tables.sql 第 26 行注释写着「无 CONCURRENTLY / 数据回填」，
// 全文匹配把它误判到非事务路径——一句注释换来一次全新安装中断。
//
// 断言方向：验证「判定结果」而不是「数据里不存在这种注释」。后者会因为
// 注释本身是历史事实而恒红，等于把断言写反；正确做法是断言判定函数对这些
// 文件给出与去注释后内容一致的结果。
func TestDiskMigrations_NoFalseConcurrentDetection(t *testing.T) {
	dir := "../migrations"
	migs, err := FilesystemMigrations(dir)
	require.NoError(t, err)
	require.NotEmpty(t, migs)

	// 至少存在一个「注释提到但语句不需要」的真实样本，否则这条守卫空转。
	commentOnlySamples := 0
	for _, mig := range migs {
		raw := strings.Contains(strings.ToUpper(mig.SQLContent), "CONCURRENTLY")
		stripped := strings.Contains(strings.ToUpper(sqlsplit.StripComments(mig.SQLContent)), "CONCURRENTLY")
		if !raw || stripped {
			continue
		}
		commentOnlySamples++
		// 判定必须与「去注释后」一致：这类文件应走事务路径。
		assert.False(t, needsNonTransactional(mig.SQLContent),
			"迁移 %s 的 CONCURRENTLY 只出现在注释里，不应走非事务路径", mig.Version)
	}
	assert.NotZero(t, commentOnlySamples,
		"未找到「仅注释提到 CONCURRENTLY」的迁移样本——守卫已空转，请更新样本或判定")
}
