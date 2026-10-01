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
		"退役迁移只能是 no-op；恢复动态端点 ACL 需按 endpoint_ac_ls 重写并补授权回归测试")
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
