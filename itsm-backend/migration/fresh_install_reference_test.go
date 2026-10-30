package migration

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 全新安装会执行每一个日期化磁盘迁移，而既有安装一律「收养不执行」
// （见 legacy_record.go adoptionCutoffUTC）。所以引用不存在对象的脚本只在全新部署
// 里暴露：20260501_rbac_endpoint_acls 正是这样打断整条初始化链，backend/worker/
// frontend/nginx 全部起不来（2026-10-01 清卷全新部署实证
// pq: relation "permission_definition" does not exist at 1367:13）。
//
// 本门禁固定禁止磁盘迁移流再引用 Ent 从未生成的旧式单数 RBAC 对象名。实际对象名：
// roles / role_permissions / permission_definitions / endpoint_ac_ls
// （ent/endpointacl/endpointacl.go:39）。
var legacySingularObjectRefs = []struct {
	object  string
	pattern *regexp.Regexp
}{
	{"permission_definition", regexp.MustCompile(`(?i)\b(?:from|into|update|table|join|references)\s+permission_definition\b`)},
	{"role_permission", regexp.MustCompile(`(?i)\b(?:from|into|update|table|join|references)\s+role_permission\b`)},
	{"role", regexp.MustCompile(`(?i)\b(?:from|into|update|table|join|references)\s+role\b`)},
	{"endpoint_acls", regexp.MustCompile(`(?i)\bendpoint_acls\b`)},
}

// 已被退役迁移 DROP 的死表（20260920 / 20260921）。add_missing_indexes* 的版本名按字典序
// 排在 2026* 之后，全新安装里先 DROP 再建索引必报 42P01 并中断初始化（CONCURRENTLY 走
// 逐语句路径，首错即停；2026-10-01 清卷全新部署实证）。只禁「按这些表建索引」的写法，
// DROP 它们的退役迁移自身仍可正常引用。
var retiredTableIndexRefs = regexp.MustCompile(`(?i)\bon\s+(?:public\.)?(workflow_instances|workflow_tasks|workflow_versions|workflows|cab_members)\b`)

func stripSQLComments(sql string) string {
	return regexp.MustCompile(`(?m)--.*$`).ReplaceAllString(sql, "")
}

// ---------------------------------------------------------------------------
// RLS policy 谓词依赖的函数必须在仓库内有定义点，且定义早于引用。
//
// 故障背景（2026-10-30 fresh-install-gate 实证）：
//   全部 RLS policy 的谓词都是 tenant_id = get_current_tenant_id()，
//   该函数的定义原本由历史迁移 009_enable_rls_tenant_isolation 声明、
//   019_align_rls_tenant_variable 对齐到 app.current_tenant。这两个历史迁移的
//   源文件已从仓库删除，于是：
//     · 既有安装（prod）—— 函数在建 policy 时已落地，账本有 009/019 记录，不受影响；
//     · 全新安装 —— 磁盘流与 Go 注册表都没有函数定义，20261028_rls_core_tables.sql
//       的守卫按设计抛错（unterminated dollar-quoted string 之外的第一层失败），
//       整条初始化链中断，backend/worker/frontend 全部起不来。
//   修复是新增 20261027_ensure_rls_tenant_helper.sql 幂等补回函数定义。
//
// 本门禁固定住这个不变量：policy 谓词里出现的自定义函数，要么在仓库中有
// CREATE [OR REPLACE] FUNCTION 定义点，要么它属于 PostgreSQL 内置函数。
// 只扫 policy 谓词（而不是全文件函数调用）是为了零误报——表名在
// CREATE TABLE x (...) / CHECK (...) / UNIQUE (...) 里同样带括号。
//
// 断言「扫描器至少命中一个谓词引用」是防假绿的关键：扫描器一旦随 SQL 写法
// 演化而失效，零命中会让整个门禁静默放行（2026-10-27 路由覆盖率棘轮的教训：
// 断言方向写反比没断言更危险）。
// ---------------------------------------------------------------------------

// postgresBuiltinPolicyFuncs 是 policy 谓词里允许出现的 PostgreSQL 内置函数。
// 这些函数由数据库自身提供，不依赖仓库里的定义点。
var postgresBuiltinPolicyFuncs = map[string]bool{
	"coalesce": true, "nullif": true, "current_setting": true, "set_config": true,
	"count": true, "sum": true, "min": true, "max": true, "avg": true,
	"lower": true, "upper": true, "trim": true, "concat": true, "concat_ws": true,
	"substring": true, "length": true, "now": true, "exists": true, "any": true,
	"array": true, "greatest": true, "least": true,
}

var (
	// policyStatementRe 抓出单条 CREATE POLICY 语句（含尾部分号）。
	policyStatementRe = regexp.MustCompile(`(?is)create\s+policy\b.*?;`)
	// policyFuncCallRe 抓谓词里的函数调用。
	policyFuncCallRe = regexp.MustCompile(`\b([a-z_][a-z0-9_]*)\s*\(`)
	// createFunctionRe 抓仓库里的函数定义点。
	createFunctionRe = regexp.MustCompile(`(?i)create\s+(?:or\s+replace\s+)?function\s+(?:public\.)?([a-z_][a-z0-9_]*)\s*\(`)
)

func TestDiskMigrations_PolicyPredicatesReferenceOnlyDefinedFunctions(t *testing.T) {
	dir := os.Getenv("MIGRATIONS_DIR")
	if dir == "" {
		dir = "../migrations"
	}
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("skip: migrations dir not available at %q (%v)", dir, err)
	}

	migs, err := FilesystemMigrations(dir)
	require.NoError(t, err)
	require.NotEmpty(t, migs)

	// 收集定义点：函数名 → 定义它的迁移版本号（字典序最小的那次定义）。
	definedIn := map[string]string{}
	for _, mig := range migs {
		for _, m := range createFunctionRe.FindAllStringSubmatch(stripSQLComments(mig.SQLContent), -1) {
			name := strings.ToLower(m[1])
			if prev, ok := definedIn[name]; !ok || mig.Version < prev {
				definedIn[name] = mig.Version
			}
		}
	}

	// 收集 policy 谓词里的函数调用：函数名 → 引用它的迁移版本号集合。
	referencedIn := map[string]map[string]bool{}
	totalRefs := 0
	for _, mig := range migs {
		body := stripSQLComments(mig.SQLContent)
		for _, stmt := range policyStatementRe.FindAllString(body, -1) {
			// 谓词从 USING 之后开始（USING (...) WITH CHECK (...)）。
			idx := strings.Index(strings.ToUpper(stmt), "USING")
			if idx < 0 {
				continue
			}
			for _, m := range policyFuncCallRe.FindAllStringSubmatch(stmt[idx:], -1) {
				name := strings.ToLower(m[1])
				if postgresBuiltinPolicyFuncs[name] {
					continue
				}
				if referencedIn[name] == nil {
					referencedIn[name] = map[string]bool{}
				}
				referencedIn[name][mig.Version] = true
				totalRefs++
			}
		}
	}

	// 防假绿：扫描器必须真的命中过谓词引用。零命中说明扫描器失效了
	// （SQL 写法演化、谓词形式变化），此时门禁不能静默放行。
	require.NotZero(t, totalRefs,
		"扫描器未命中任何 policy 谓词函数调用——门禁已失效，请更新正则而不是让它静默通过")

	for name, refs := range referencedIn {
		defVersion, ok := definedIn[name]
		require.True(t, ok,
			"policy 谓词引用了仓库中没有定义点的函数 %s()；全新安装会以 42883 中断初始化。"+
				"请新增幂等的 CREATE OR REPLACE FUNCTION 迁移，且其版本号必须早于引用它的迁移", name)
		for refVersion := range refs {
			require.Less(t, defVersion, refVersion,
				"函数 %s() 的定义迁移 %s 晚于引用它的迁移 %s；迁移流按版本号字典序执行，"+
					"引用时函数还不存在", name, defVersion, refVersion)
		}
	}
}

func TestDiskMigrations_DoNotReferenceLegacySingularObjects(t *testing.T) {
	dir := os.Getenv("MIGRATIONS_DIR")
	if dir == "" {
		dir = "../migrations"
	}
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("skip: migrations dir not available at %q (%v)", dir, err)
	}

	migs, err := FilesystemMigrations(dir)
	require.NoError(t, err)
	require.NotEmpty(t, migs)

	for _, mig := range migs {
		body := stripSQLComments(mig.SQLContent)
		for _, bad := range legacySingularObjectRefs {
			assert.NotRegexp(t, bad.pattern, body,
				"迁移 %s 引用了 Ent 从未生成的对象 %q；全新安装会以 42P01 中断初始化", mig.Version, bad.object)
		}
		assert.NotRegexp(t, retiredTableIndexRefs, body,
			"迁移 %s 给已退役的死表建索引；全新安装会以 42P01 中断初始化", mig.Version)
	}
}

// 退役文件必须保留版本名（既有安装账本里的 20260501_rbac_endpoint_acls 要能解析到
// 磁盘条目），且必须保持 no-op：否则它会在某次编辑中重新变成写权限数据的活跃迁移，
// 而动态端点 ACL 的 L2「命中但无权限即拒绝」语义从未被验证过。
func TestRetiredEndpointACLsMigrationStaysNoOp(t *testing.T) {
	dir := os.Getenv("MIGRATIONS_DIR")
	if dir == "" {
		dir = "../migrations"
	}
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("skip: migrations dir not available at %q (%v)", dir, err)
	}

	migs, err := FilesystemMigrations(dir)
	require.NoError(t, err)

	var retired *Migration
	for i := range migs {
		if migs[i].Version == "20260501_rbac_endpoint_acls" {
			retired = &migs[i]
			break
		}
	}
	require.NotNil(t, retired, "退役迁移必须仍按原版本名保留在迁移流中")
	assert.Equal(t, "SELECT 1;", strings.TrimSpace(stripSQLComments(retired.SQLContent)),
		"退役迁移只能是 no-op；恢复动态端点 ACL 需要按 endpoint_acls 重写并补授权测试")
}

// 磁盘迁移不得写入 process_bindings：迁移流在 seed 之前执行，此时 departments 还没有
// 数据（子查询恒为 NULL），而绑定指向的 process_definition_key 只有在 BPMN 模板部署后
// 才存在。pkg/seeder/initialization_adapter.go:571 verifyWorkflowTemplates 会遍历每条
// active 绑定并要求其 definition 存在，因此一条引用未部署 key 的绑定就能让 workflow-core
// 组件回滚、全新安装起不来（2026-10-01 清卷全新部署实证：
// 20260620_process_routing_enhancement 的 incident_general_flow /
// change_emergency_flow / release_test_flow / expense_approval_flow 在 service/bpmn
// 里没有任何载体）。
// 归属：基线绑定属于 pkg/seeder/seeder.go:678 的内置清单；部门级绑定属于
// service/bpmn_process_binding_service.go 的运行期能力。
func TestDiskMigrations_DoNotSeedProcessBindings(t *testing.T) {
	dir := os.Getenv("MIGRATIONS_DIR")
	if dir == "" {
		dir = "../migrations"
	}
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("skip: migrations dir not available at %q (%v)", dir, err)
	}

	migs, err := FilesystemMigrations(dir)
	require.NoError(t, err)

	insertProcessBindings := regexp.MustCompile(`(?i)\binsert\s+into\s+(?:public\.)?process_bindings\b`)
	for _, mig := range migs {
		assert.NotRegexp(t, insertProcessBindings, stripSQLComments(mig.SQLContent),
			"迁移 %s 直接写入 process_bindings；未部署的 key 会让 workflow-core 校验失败并中断全新安装", mig.Version)
	}
}
