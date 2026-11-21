package database

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// 2026-11-21 第三轮新增：第三轮首装门禁把 20261119_ci_owner_fk_and_certification.sql
// 手写建的 ci_tenant_owner_idx / ci_tenant_department_idx 两对冗余索引列出，
// Ent 声明了等价索引 configurationitem_tenant_id_owner_id /
// configurationitem_tenant_id_department_id（见 ent/schema/configurationitem.go
// 的 tenant_id+owner_id / tenant_id+department_id Fields 索引）。
// 收敛表必须注册这两对「保留 Ent 声明版本 + 删除手写版本」，
// 否则 fresh-install 门禁会再次因语义重复索引 FAIL。
func TestDuplicateIndexPairs_CiOwnerDepartmentRegistered(t *testing.T) {
	for _, p := range duplicateIndexPairs {
		if p[1] != "ci_tenant_owner_idx" && p[1] != "ci_tenant_department_idx" {
			continue
		}
		require.True(t, safeIndexName.MatchString(p[0]),
			"保留方索引名 %q 不在白名单内", p[0])
		require.True(t, safeIndexName.MatchString(p[1]),
			"冗余方索引名 %q 不在白名单内", p[1])
	}
	found := map[string]bool{}
	for _, p := range duplicateIndexPairs {
		found[p[1]] = true
	}
	require.True(t, found["ci_tenant_owner_idx"], "duplicateIndexPairs 缺 ci_tenant_owner_idx")
	require.True(t, found["ci_tenant_department_idx"], "duplicateIndexPairs 缺 ci_tenant_department_idx")
	require.NotContains(t, found, "", "空索引名不应注册")
}
