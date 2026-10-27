package cmdb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestService_GetGovernanceReport 锁治理看板的三面数据同源契约：
// 质量指标消费权威 CI 全集、差异分桶消费「CI 全集 × 云资源全集」的
// DiffReconciliation 结果、顶部计数与两侧全集行数一致。
func TestService_GetGovernanceReport(t *testing.T) {
	now := time.Now()
	recent := now.Add(-1 * time.Hour)     // 远小于 7 天阈值
	stale := now.Add(-8 * 24 * time.Hour) // 超过 7 天阈值

	t.Run("三面数据同源且分桶正确", func(t *testing.T) {
		// 3 个 CI：1 个与云资源 ref=11 关联（noop），1 个 ref=99 指向已消失
		// 的资源（retire_confirm），1 个无任何来源与 owner（孤儿+不完整）。
		cis := []*ConfigurationItem{
			{ID: 1, CINumber: "CI-1", CloudResourceRefID: 11, DiscoverySource: "aliyun", OwnedBy: "ops", CloudSyncTime: &recent},
			{ID: 2, CINumber: "CI-2", CloudResourceRefID: 99, DiscoverySource: "aliyun", OwnedBy: "ops", CloudSyncTime: &stale},
			{ID: 3, CINumber: "CI-3"},
		}
		resources := []*CloudResource{
			{ID: 11, Provider: "aliyun", ResourceID: "i-a", TenantID: 1},
			{ID: 22, Provider: "aliyun", ResourceID: "i-b", TenantID: 1}, // 未关联 → add
		}

		svc := newTestService(&mockRepository{
			listCloudResourcesFn: func(ctx context.Context, tenantID int, filter CloudResourceFilter) ([]*CloudResource, error) {
				return resources, nil
			},
			listCIsForReconciliationFn: func(ctx context.Context, tenantID int) ([]*ConfigurationItem, error) {
				return cis, nil
			},
		})

		report, err := svc.GetGovernanceReport(context.Background(), 1)
		require.NoError(t, err)

		// 顶部计数与两侧全集一致
		assert.Equal(t, 2, report.CloudResources)
		assert.Equal(t, 3, report.CIs)

		// 质量指标：total=3，无 scrapped → active=3/retired=0；
		// CI-2 同步时间超 7 天 → stale=1；CI-3 无来源无 owner → orphan=1+incomplete=1。
		assert.Equal(t, 3, report.Quality.Total)
		assert.Equal(t, 3, report.Quality.Active)
		assert.Equal(t, 0, report.Quality.Retired)
		assert.Equal(t, 1, report.Quality.Stale)
		assert.Equal(t, 1, report.Quality.Orphan)
		assert.Equal(t, 1, report.Quality.Incomplete)
		assert.InDelta(t, 66.67, report.Quality.CompletenessPct, 0.01)

		// 差异分桶：ref=11 已关联 noop=1；ref=22 未关联 add=1；
		// ref=99 的 CI 指向的本批消失资源 retire_confirm=1；duplicate=0。
		assert.Equal(t, map[ReconciliationAction]int{
			ReconcileActionAdd:           1,
			ReconcileActionNoOp:          1,
			ReconcileActionRetireConfirm: 1,
			ReconcileActionDuplicate:     0,
		}, report.Diff)
	})

	t.Run("空租户返回全 0 指标而非 nil", func(t *testing.T) {
		svc := newTestService(&mockRepository{})
		report, err := svc.GetGovernanceReport(context.Background(), 42)
		require.NoError(t, err)
		assert.Equal(t, 0, report.Quality.Total)
		assert.Equal(t, 0.0, report.Quality.CompletenessPct)
		// 四桶全部存在且为 0（CountByAction 初始化全词表），不是 nil map。
		assert.Equal(t, map[ReconciliationAction]int{
			ReconcileActionAdd:           0,
			ReconcileActionNoOp:          0,
			ReconcileActionRetireConfirm: 0,
			ReconcileActionDuplicate:     0,
		}, report.Diff)
	})

	t.Run("云资源读取失败原样上抛", func(t *testing.T) {
		svc := newTestService(&mockRepository{
			listCloudResourcesFn: func(ctx context.Context, tenantID int, filter CloudResourceFilter) ([]*CloudResource, error) {
				return nil, assert.AnError
			},
		})
		_, err := svc.GetGovernanceReport(context.Background(), 1)
		assert.ErrorIs(t, err, assert.AnError)
	})
}
