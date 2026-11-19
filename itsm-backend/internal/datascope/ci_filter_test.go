package datascope

// 回归：CI 行级权限谓词必须与工单侧同口径——管理角色全量、普通角色按 owner/department 收窄，
// 且「算不出范围」（userID<=0，含 service 主体）一律空集，绝不放宽成全租户。
//
// 额外锁一条 CI 独有的语义（2026-11-19 上线阻塞项）：
// owner_id/department_id 是新增列，存量 CI 全为 NULL。这类「未归属资产」必须是
// 未归属公共资产可见（租户内），否则上线当天非管理角色的 CMDB 列表整体变空。
//
// 治理看板刻意不接本谓词（用户拍板：治理是全局质量视图）；本测试只锁业务读面的过滤语义。

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyCIFilter_ScopeSemantics(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ci_datascope?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()

	tenant, err := client.Tenant.Create().SetName("CI 权限租户").SetCode("ci-ds").
		SetDomain("ci-ds.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	ciType, err := client.CIType.Create().SetName("Server").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	// 四条 CI：alice(1) 持有、bob(2) 持有、一条 owner_id+department_id 全空（未归属）、
	// 一条 owner=alice 但 department 空（回填只成功一半的中间态）。
	_, err = client.ConfigurationItem.Create().SetName("alice-ci").SetCiTypeID(ciType.ID).
		SetTenantID(tenant.ID).SetOwnerID(1).SetDepartmentID(10).Save(ctx)
	require.NoError(t, err)
	_, err = client.ConfigurationItem.Create().SetName("bob-ci").SetCiTypeID(ciType.ID).
		SetTenantID(tenant.ID).SetOwnerID(2).SetDepartmentID(20).Save(ctx)
	require.NoError(t, err)
	_, err = client.ConfigurationItem.Create().SetName("orphan-ci").SetCiTypeID(ciType.ID).
		SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	_, err = client.ConfigurationItem.Create().SetName("partial-ci").SetCiTypeID(ciType.ID).
		SetTenantID(tenant.ID).SetOwnerID(1).Save(ctx)
	require.NoError(t, err)

	ids := func(ds DataScope, userID, deptID int) []string {
		q := ApplyCIFilter(client.ConfigurationItem.Query(), userID, deptID, ds)
		cis, err := q.All(ctx)
		require.NoError(t, err)
		out := make([]string, 0, len(cis))
		for _, ci := range cis {
			out = append(out, ci.Name)
		}
		return out
	}

	// All：不过滤，四条全见。
	assert.Len(t, ids(DataScopeAll, 1, 10), 4)

	// OwnedOrAssigned：本人持有的两条 + 未归属公共资产一条（bob 的不可见）。
	assert.ElementsMatch(t, []string{"alice-ci", "partial-ci", "orphan-ci"},
		ids(DataScopeOwnedOrAssigned, 1, 10))

	// Department：本人 owner（alice-ci/partial-ci）+ 本部门 owner（dept 20 的 bob-ci）
	// + 未归属公共资产（orphan-ci）。
	got := ids(DataScopeDepartment, 1, 20)
	assert.ElementsMatch(t, []string{"alice-ci", "partial-ci", "bob-ci", "orphan-ci"}, got,
		"部门档应含本人 owner + 本部门 owner + 未归属公共资产")

	// departmentID<=0 时不得加部门谓词：DepartmentIDEQ(0) 会命中 department_id=0
	// 的脏数据行，等于凭空放行。此时只剩「本人 owner + 未归属」。
	assert.ElementsMatch(t, []string{"alice-ci", "partial-ci", "orphan-ci"},
		ids(DataScopeDepartment, 1, 0),
		"部门ID缺失时不得退化成「任意部门」")

	// 安全收窄：userID=0（service 主体）在两档下都是空集，绝不返回全量。
	assert.Empty(t, ids(DataScopeOwnedOrAssigned, 0, 10), "userID=0 不得放宽为不过滤")
	assert.Empty(t, ids(DataScopeDepartment, 0, 10), "userID=0 不得放宽为不过滤")
	assert.Empty(t, ids(DataScope(99), 1, 10), "未知 scope 必须安全收窄")
}
