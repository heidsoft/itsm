package cmdb

// 回归：ListDiscoveryResults 必须真分页（Count + Offset/Limit），不能 All() 整表读。
//
// 历史缺陷：repo 用 q.All(ctx) 把本租户全部差异一次性读进内存，handler 完全忽略
// page/pageSize（itsm-cli 传了这两个参数被静默忽略）。单行含 resource_snapshot +
// diff 两个 JSON 大字段，差异表随发现轮次累积，整表读是内存与延迟隐患。
//
// 本文件在仓储层用真实 ent 库锁住：total 是全量计数，items 受 page/pageSize 约束，
// 且按 (created_at, id) 稳定排序——与同域 ListDiscoveryJobs 完全同口径。

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListDiscoveryResults_IsActuallyPaginated(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:discovery_results_paging?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()
	repo := NewEntRepository(client)

	tenant, err := client.Tenant.Create().SetName("分页租户").SetCode("dr-paging").
		SetDomain("dr-paging.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	// DiscoveryResult.job_id → discovery_jobs.source_id → discovery_sources，逐层建宿主。
	_, err = client.DiscoverySource.Create().
		SetID("ds-paging").SetName("分页发现源").SetSourceType("cloud_api").
		SetProvider("alibaba").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	job, err := client.DiscoveryJob.Create().
		SetSourceID("ds-paging").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		_, err := repo.CreateDiscoveryResult(ctx, &DiscoveryResult{
			JobID: job.ID, Action: "update", ResourceType: "server",
			ResourceID:       "res-" + string(rune('a'+i)),
			ResourceIdentity: "id-" + string(rune('a'+i)), IdentityVersion: 1,
			ResourceSnapshot: map[string]interface{}{}, BeforeHash: "b", AfterHash: "a", Diff: map[string]interface{}{},
			Status: "pending", TenantID: tenant.ID,
		})
		require.NoError(t, err)
	}

	// 第 1 页 2 条：total 必须是全量 5，不是本页 2。
	page1, total, err := repo.ListDiscoveryResults(ctx, tenant.ID, 0, 1, 2)
	require.NoError(t, err)
	assert.Equal(t, 5, total, "total 必须是过滤条件下的全量计数")
	assert.Len(t, page1, 2, "pageSize=2 必须只返回 2 条，不能整表读")

	// 第 3 页只剩 1 条。
	page3, total3, err := repo.ListDiscoveryResults(ctx, tenant.ID, 0, 3, 2)
	require.NoError(t, err)
	assert.Equal(t, 5, total3)
	assert.Len(t, page3, 1, "第 3 页应只剩 1 条")

	// 页间不重叠：稳定排序 (created_at,id) 保证翻页不丢不重。
	seen := map[int]bool{}
	for _, item := range append(page1, page3...) {
		assert.False(t, seen[item.ID], "分页结果不得重复: id=%d", item.ID)
		seen[item.ID] = true
	}
	assert.Len(t, seen, 3, "两页合计 3 条不重复")

	// jobID 过滤 + 分页叠加时，total 是过滤后的计数。
	_, filteredTotal, err := repo.ListDiscoveryResults(ctx, tenant.ID, job.ID, 1, 2)
	require.NoError(t, err)
	assert.Equal(t, 5, filteredTotal, "全部 5 条都属于该 job")
	_, filteredTotal, err = repo.ListDiscoveryResults(ctx, tenant.ID, 999, 1, 2)
	require.NoError(t, err)
	assert.Zero(t, filteredTotal, "不存在的 job 过滤后 total 应为 0")
}
