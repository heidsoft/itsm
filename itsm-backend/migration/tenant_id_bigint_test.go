package migration

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// bigintMigration 是 P2-4 的类型统一迁移。
const bigintMigration = "../migrations/20261031_ai_vector_bigint.sql"

// bigintMigrationDown 是它的回滚脚本。
const bigintMigrationDown = "../migrations/20261031_ai_vector_bigint_down.sql"

// tenantIDIntPattern 匹配「把 tenant_id 声明成 int4」的 DDL 片段。
//
// 刻意只匹配 INT / INTEGER 两种写法，不匹配 BIGINT / BIGSERIAL：
// PostgreSQL 里 `BIGINT` 含子串 `INT`，朴素的正则会把它误判为违规。
var tenantIDIntPattern = regexp.MustCompile(`(?i)\btenant_id\s+(INT|INTEGER)\b`)

// TestTenantIDBigintInvariantIsEnforced 锁定 P2-4 的核心修复不被回退。
//
// 类型统一之所以不能只靠一次性 ALTER：全库 tenant_id 有两条建表链路
// （ent 生成 bigint、手写 SQL 习惯写 INT），只要有人在手写迁移里再写一次
// `tenant_id INTEGER`，新表就又是 int4，而 int4/int8 混用会让 tenant_id 上的
// 索引在 JOIN 里失效——无声、无报错，只在慢查询里体现。
//
// 本测试做两件事：
//  1. 断言类型统一迁移保留了**泛化**的 tenant_id 段（而不是被收窄成硬编码
//     表名列表）——否则未来新增的表不受保护；
//  2. 断言 20261031 之后新增的磁盘迁移不再声明 int4 tenant_id。
func TestTenantIDBigintInvariantIsEnforced(t *testing.T) {
	raw, err := os.ReadFile(bigintMigration)
	require.NoError(t, err, "P2-4 类型统一迁移必须存在：%s", bigintMigration)
	sql := string(raw)

	// 1. 泛化段必须存在：选择条件同时包含 column_name='tenant_id' 与
	//    data_type='integer'，说明它按元数据扫描而不是按表名枚举。
	assert.Contains(t, sql, "column_name = 'tenant_id'",
		"P2-4 迁移必须保留泛化的 tenant_id 扫描：按表名硬编码的列表无法覆盖未来新增表")
	assert.Contains(t, sql, "data_type = 'integer'",
		"P2-4 迁移必须按 information_schema 的 data_type 判断，而不是按表名猜测")

	// 2. 显式目标列必须保留（审计点名，不属于通用 *_id 规则）。
	for _, want := range []string{"'user_id', 'item_id'", "'object_id'"} {
		assert.Contains(t, sql, want,
			"P2-4 迁移必须覆盖审计点名的引用列：%s", want)
	}

	// 3. 自检段必须存在：转换失败要显式 RAISE EXCEPTION，而不是记一笔"已应用"。
	assert.Contains(t, sql, "RAISE EXCEPTION 'P2-4 未完成",
		"P2-4 迁移必须有结果自检：不变量未达成时失败，避免账本撒谎")
}

// TestNewMigrationsDoNotDeclareInt4TenantID 防止后续手写迁移重新引入 int4 tenant_id。
//
// 时间基准取本迁移的日期前缀：更早的迁移是历史事实（已应用、已参与 checksum），
// 由 P2-4 迁移的泛化段在运行时兜住；更晚的迁移属于新代码，必须在写入时就正确。
func TestNewMigrationsDoNotDeclareInt4TenantID(t *testing.T) {
	const cutoff = "20261031"

	entries, err := os.ReadDir("../migrations")
	require.NoError(t, err)

	var offenders []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".sql") || strings.HasSuffix(name, "_down.sql") {
			continue
		}
		// 只检查日期化迁移；无日期前缀的历史脚本不参与（它们早已应用）。
		if len(name) < 8 || name[:8] <= cutoff {
			continue
		}

		raw, err := os.ReadFile(filepath.Join("../migrations", name))
		require.NoError(t, err)
		for i, line := range strings.Split(string(raw), "\n") {
			// 注释里出现 "tenant_id INTEGER" 是说明文字，不是 DDL。
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "--") {
				continue
			}
			if tenantIDIntPattern.MatchString(line) {
				offenders = append(offenders, name+":"+itoa(i+1)+"  "+trimmed)
			}
		}
	}

	sort.Strings(offenders)
	assert.Empty(t, offenders,
		"新增迁移不得声明 int4 tenant_id（应为 BIGINT）：\n%s", strings.Join(offenders, "\n"))
}

// TestBigintMigrationRollbackIsLosslessAware 锁定回滚脚本的有损保护。
//
// int8 -> int4 越界会静默截断或报错，运维为了"回滚成功"很可能先删数据。
// 回滚脚本必须先检查值域并显式拒绝，这条断言防止它被简化成裸 ALTER。
func TestBigintMigrationRollbackIsLosslessAware(t *testing.T) {
	raw, err := os.ReadFile(bigintMigrationDown)
	require.NoError(t, err, "P2-4 回滚脚本必须存在：%s", bigintMigrationDown)
	down := string(raw)

	assert.Contains(t, down, "2147483647",
		"回滚脚本必须检查 int4 上限（2147483647），否则越界值会被拒绝或截断")
	assert.Contains(t, down, "RAISE EXCEPTION",
		"回滚脚本遇到越界值必须显式拒绝，而不是静默继续")
	assert.Contains(t, down, "tenant_id 的 int8 统一未回滚",
		"回滚脚本必须声明 tenant_id 泛化修正不回滚（回到 int4 是回归）")
}

// itoa 避免为一条测试引入 strconv 依赖噪音。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
