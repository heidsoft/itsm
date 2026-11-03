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

// RLS 策略姿态守卫。
//
// 背景：RLS 策略散落在权威内联流（migration/migrations.go 的 009/019）与
// 磁盘日期化流（migrations/*.sql）里，谓词与命名各写各的。已实测到两类
// 真实缺陷：
//
//  1. knowledge_article_reviews 的策略读 app.tenant_id，而驱动注入的是
//     app.current_tenant —— enforce 下该表恒返回 0 行，且没有任何报错；
//  2. 试点脚本留下无表名后缀的 tenant_isolation 策略名，同表可并存两条策略。
//
// 这些缺陷不会让任何测试变红：数据库不报错，应用只是"读不到数据"。因此
// 必须有静态守卫，让下一次手写策略时在 CI 就失败。

const (
	rlsAlignmentMigration = "../migrations/20261031_rls_policy_alignment.sql"
	rlsRolesMigration     = "../migrations/20261031_rls_roles_bootstrap.sql"
	rlsCanonicalPrefix    = "tenant_isolation_"
	// rlsCutoff 是「策略姿态统一」的日期前缀。更早的文件是历史事实
	// （已应用、已参与 checksum），由 20261031 的统一迁移在运行时修正；
	// 更晚的文件必须直接写对。
	rlsCutoff = "20261031"
)

var (
	rlsEnableRe = regexp.MustCompile(`(?i)ALTER\s+TABLE\s+(?:IF\s+EXISTS\s+)?([a-z_][a-z0-9_]*)\s+ENABLE\s+ROW\s+LEVEL\s+SECURITY`)
	rlsPolicyRe = regexp.MustCompile(`(?i)CREATE\s+POLICY\s+([a-z_][a-z0-9_]*)\s+ON\s+([a-z_][a-z0-9_]*)`)
	rlsGUCRe    = regexp.MustCompile(`(?i)current_setting\s*\(\s*'([^']+)'`)
)

// legacyTenantPolicies 是历史遗留、由 20261031 统一迁移修正的策略。
//
// 允许清单刻意最小化：新增条目必须同时说明「为什么不能直接改历史文件」
// 与「由哪个迁移修正」，否则这个清单会变成掩盖新漂移的垃圾桶。
var legacyTenantPolicies = map[string]string{
	"knowledge_article_reviews_tenant_policy": "20261015 已应用（checksum 冻结），谓词用 app.tenant_id；由 20261031_rls_policy_alignment.sql 统一为 tenant_isolation_knowledge_article_reviews",
}

// rlsSource 表示一个迁移源文件里的 SQL 文本。
type rlsSource struct {
	name string
	date string // YYYYMMDD；migrations.go 视为最早
	sql  string
}

// loadRLSSources 读取权威内联流与磁盘迁移流（排除 *_down.sql）。
func loadRLSSources(t *testing.T) []rlsSource {
	t.Helper()

	inline, err := os.ReadFile("migrations.go")
	require.NoError(t, err, "权威内联迁移流必须可读")
	sources := []rlsSource{{name: "migration/migrations.go", date: "00000000", sql: string(inline)}}

	entries, err := os.ReadDir("../migrations")
	require.NoError(t, err)

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".sql") || strings.HasSuffix(name, "_down.sql") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join("../migrations", name))
		require.NoError(t, err)
		date := "00000000"
		if len(name) >= 8 {
			date = name[:8]
		}
		sources = append(sources, rlsSource{name: name, date: date, sql: string(raw)})
	}

	// 与迁移器一致：内联流先执行，磁盘流按字典序。
	sort.SliceStable(sources, func(i, j int) bool {
		if sources[i].date != sources[j].date {
			return sources[i].date < sources[j].date
		}
		return sources[i].name < sources[j].name
	})
	return sources
}

// policyBlock 返回 CREATE POLICY 之后到下一个分号之前的文本，用于读谓词。
func policyBlock(sql string, start int) string {
	rest := sql[start:]
	if i := strings.Index(rest, ";"); i >= 0 {
		return rest[:i]
	}
	if len(rest) > 500 {
		return rest[:500]
	}
	return rest
}

// functionBlock 返回 CREATE OR REPLACE FUNCTION 的完整定义（到 $$ LANGUAGE ... ;）。
//
// 不能复用 policyBlock：helper 函数体里的第一个分号在 RETURN 语句末尾，
// 会把它后面的 EXCEPTION 段截掉。
func functionBlock(sql string, start int) string {
	rest := sql[start:]
	lower := strings.ToLower(rest)
	if i := strings.Index(lower, "$$ language"); i >= 0 {
		end := i + len("$$ language")
		if j := strings.Index(rest[end:], ";"); j >= 0 {
			return rest[:end+j+1]
		}
		return rest[:end]
	}
	if len(rest) > 1500 {
		return rest[:1500]
	}
	return rest
}

// TestEveryRLSTableHasPolicy 保证「启用 RLS」与「有策略」成对出现。
//
// 只 ENABLE 不 CREATE POLICY 的表在 enforce 下默认拒绝所有行：业务会大
// 面积返回 0 行，而日志里没有任何错误——这是最难排查的一类故障。
func TestEveryRLSTableHasPolicy(t *testing.T) {
	enabled := map[string][]string{}
	policies := map[string][]string{}

	for _, src := range loadRLSSources(t) {
		for _, m := range rlsEnableRe.FindAllStringSubmatch(src.sql, -1) {
			enabled[strings.ToLower(m[1])] = append(enabled[strings.ToLower(m[1])], src.name)
		}
		for _, m := range rlsPolicyRe.FindAllStringSubmatchIndex(src.sql, -1) {
			name := strings.ToLower(src.sql[m[2]:m[3]])
			table := strings.ToLower(src.sql[m[4]:m[5]])
			policies[table] = append(policies[table], name)
		}
	}

	require.NotEmpty(t, enabled, "守卫测试必须能解析到 RLS 表；解析失败会让本测试变成空转")

	var missing []string
	for table := range enabled {
		if len(policies[table]) == 0 {
			missing = append(missing, table)
		}
	}
	sort.Strings(missing)
	assert.Empty(t, missing,
		"以下表启用了 RLS 却没有任何策略（enforce 下恒 0 行）：%s", strings.Join(missing, ", "))
}

// TestRLSPolicyNamesAndPredicatesAreCanonical 锁定策略命名与租户来源。
//
// 规则：
//   - 策略名必须是 tenant_isolation_<table>（历史遗留见 legacyTenantPolicies）；
//   - 谓词里的 GUC 必须是 app.current_tenant —— 驱动只注入这个名字；
//   - 遗留条目必须被 20261031 的统一迁移覆盖。
func TestRLSPolicyNamesAndPredicatesAreCanonical(t *testing.T) {
	var namingOffenders, gucOffenders []string

	for _, src := range loadRLSSources(t) {
		for _, m := range rlsPolicyRe.FindAllStringSubmatchIndex(src.sql, -1) {
			name := strings.ToLower(src.sql[m[2]:m[3]])
			table := strings.ToLower(src.sql[m[4]:m[5]])
			canonical := rlsCanonicalPrefix + table

			if name != canonical {
				if reason, allowed := legacyTenantPolicies[name]; allowed {
					t.Logf("%s: legacy policy %q allowed (%s)", src.name, name, reason)
				} else {
					namingOffenders = append(namingOffenders,
						src.name+": policy "+name+" on "+table+" (want "+canonical+")")
				}
			}

			for _, g := range rlsGUCRe.FindAllStringSubmatch(policyBlock(src.sql, m[1]), -1) {
				if strings.EqualFold(g[1], "app.current_tenant") {
					continue
				}
				if _, allowed := legacyTenantPolicies[name]; allowed {
					t.Logf("%s: legacy policy %q uses GUC %q (allowed)", src.name, name, g[1])
					continue
				}
				gucOffenders = append(gucOffenders,
					src.name+": policy "+name+" on "+table+" reads "+g[1]+" (want app.current_tenant)")
			}
		}
	}

	sort.Strings(namingOffenders)
	sort.Strings(gucOffenders)
	assert.Empty(t, namingOffenders, "RLS 策略名必须为 tenant_isolation_<table>：\n%s", strings.Join(namingOffenders, "\n"))
	assert.Empty(t, gucOffenders, "RLS 策略必须读 app.current_tenant（驱动只注入该变量）：\n%s", strings.Join(gucOffenders, "\n"))

	// 遗留条目不是"永久豁免"：必须有迁移在运行时修正它。
	if len(legacyTenantPolicies) > 0 {
		raw, err := os.ReadFile(rlsAlignmentMigration)
		require.NoError(t, err, "存在遗留策略时必须有统一迁移：%s", rlsAlignmentMigration)
		alignment := string(raw)
		assert.Contains(t, alignment, "'tenant_isolation_' || target.table_name",
			"统一迁移必须按 tenant_isolation_<table> 动态重建策略，而不是硬编码表名列表")
		assert.Contains(t, alignment, "RAISE EXCEPTION",
			"统一迁移必须有结果自检：漏表时要失败，而不是记一笔已应用")
	}
}

// TestTenantHelperReadsDriverVariable 锁定 helper 的 GUC 名与驱动一致。
//
// 009 原本读 app.current_tenant_id，019 前向修正为 app.current_tenant。
// 如果有人在新迁移里"重新定义" helper 且写回旧名字，所有策略会同时失效。
func TestTenantHelperReadsDriverVariable(t *testing.T) {
	var lastDef string
	var lastSource string

	for _, src := range loadRLSSources(t) {
		idx := strings.LastIndex(strings.ToLower(src.sql), "create or replace function get_current_tenant_id")
		if idx < 0 {
			continue
		}
		lastDef = functionBlock(src.sql, idx)
		lastSource = src.name
	}

	require.NotEmpty(t, lastDef, "必须存在 get_current_tenant_id() 的定义")
	guces := rlsGUCRe.FindAllStringSubmatch(lastDef, -1)
	require.NotEmpty(t, guces, "helper 必须读取一个 GUC")
	for _, g := range guces {
		assert.Equal(t, "app.current_tenant", strings.ToLower(g[1]),
			"%s 中 get_current_tenant_id() 必须读 app.current_tenant（驱动注入的变量）", lastSource)
	}
	assert.Contains(t, strings.ToLower(lastDef), "exception",
		"helper 必须在非法值上 fail closed，而不是把错误文本当成 tenant_id")
}

// TestRLSRolesBootstrapHasNoCredentials 防止角色引导脚本再次写入硬编码口令。
//
// 历史脚本写的是 PASSWORD 'REPLACE_IN_PRODUCTION'：照抄执行就得到一个
// 「口令公开且可登录」的角色，比不建角色更危险。正确做法是 NOLOGIN 建角色，
// 口令由运维在带外设置。
func TestRLSRolesBootstrapHasNoCredentials(t *testing.T) {
	raw, err := os.ReadFile(rlsRolesMigration)
	require.NoError(t, err, "RLS 角色引导迁移必须存在：%s", rlsRolesMigration)
	sql := string(raw)

	assert.NotContains(t, sql, "REPLACE_IN_PRODUCTION",
		"角色引导脚本不得包含占位口令")
	assert.NotRegexp(t, regexp.MustCompile(`(?i)PASSWORD\s+'[^']*'`), sql,
		"角色引导脚本不得写入任何口令字面量；口令必须由运维带外设置")
	assert.Contains(t, sql, "NOLOGIN",
		"角色必须默认 NOLOGIN，直到运维显式设置 LOGIN + 口令")
	assert.Regexp(t, regexp.MustCompile(`(?i)itsm_app[^;]*\bNOBYPASSRLS`), sql,
		"itsm_app 必须 NOBYPASSRLS，否则策略对它无效")
	assert.Regexp(t, regexp.MustCompile(`(?i)itsm_admin[^;]*\bBYPASSRLS`), sql,
		"itsm_admin 必须 BYPASSRLS，供迁移/后台任务跨租户使用")
}

// TestRLSRolesBootstrapIsPairedWithRollback 保证迁移可回滚。
func TestRLSRolesBootstrapIsPairedWithRollback(t *testing.T) {
	for _, path := range []string{
		rlsRolesMigration,
		rlsAlignmentMigration,
	} {
		down := strings.TrimSuffix(path, ".sql") + "_down.sql"
		info, err := os.Stat(down)
		require.NoError(t, err, "迁移必须有 _down.sql 配对（discovery 按此规则配对）：%s", down)
		assert.Greater(t, info.Size(), int64(0), "回滚脚本不能为空：%s", down)
	}
}
