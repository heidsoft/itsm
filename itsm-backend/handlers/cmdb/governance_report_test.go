package cmdb

import (
	"context"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubRequirementReader 是 CIRequirementReader 的测试替身：按 ciTypeID 返回
// 预设的必填属性名列表，未声明的类型返回空（即「该类型没有必填定义」）。
type stubRequirementReader struct {
	byType map[int][]string
	err    error
	calls  []int
}

func (s *stubRequirementReader) ListRequiredCIAttributes(ctx context.Context, tenantID, ciTypeID int) ([]dto.CIAttributeRequirement, error) {
	s.calls = append(s.calls, ciTypeID)
	if s.err != nil {
		return nil, s.err
	}
	names := s.byType[ciTypeID]
	out := make([]dto.CIAttributeRequirement, 0, len(names))
	for _, name := range names {
		out = append(out, dto.CIAttributeRequirement{Name: name, DisplayName: name})
	}
	return out, nil
}

// TestService_GetGovernanceReport 锁治理看板的四组契约：
//  1. 三块数据面**不同源**——Diff 用带云引用的 CI，Quality/Duplicate 用全租户 CI；
//  2. 完整率按该类型的必填属性定义算，不按 owner/来源；
//  3. 重复率基于 CI 间身份聚类，不再是「同批重复」（旧实现该分桶恒 0）；
//  4. 必填定义读取失败必须失败关闭，不得降级成「都填齐了」。
func TestService_GetGovernanceReport(t *testing.T) {
	now := time.Now()
	recent := now.Add(-1 * time.Hour)     // 远小于 7 天阈值
	stale := now.Add(-8 * 24 * time.Hour) // 超过 7 天阈值

	t.Run("治理基数覆盖全租户含手工CI且完整率按必填属性", func(t *testing.T) {
		// 治理基数（全租户 CI）：
		//   - CI-1：带云引用 ref=11，类型 10，必填属性齐 → 参与对账(noop) + 完整
		//   - CI-2：带云引用 ref=99（云上已消失）→ 参与对账(retire_confirm) + 必填缺 vendor
		//   - CI-4：**无云引用的手工 CI**，类型 10，必填属性齐
		//          → 进治理基数（完整性/重复率），但不进对账基数
		everything := []*ConfigurationItem{
			{ID: 1, CINumber: "CI-1", CITypeID: 10, CloudResourceRefID: 11,
				LifecycleStatus: common.CILifecycleStatusOnline, CloudSyncTime: &recent,
				Attributes: map[string]interface{}{"serial_number": "SN-1", "vendor": "Dell"}},
			{ID: 2, CINumber: "CI-2", CITypeID: 10, CloudResourceRefID: 99,
				LifecycleStatus: common.CILifecycleStatusOnline, CloudSyncTime: &stale,
				Attributes: map[string]interface{}{"serial_number": "SN-2"}},
			{ID: 4, CINumber: "CI-4", CITypeID: 10,
				LifecycleStatus: common.CILifecycleStatusOnline,
				Attributes:      map[string]interface{}{"serial_number": "SN-4", "vendor": "HP"}},
		}
		// 对账基数：仓储只返回带云引用的 CI（模拟真实 ListCIsForReconciliation）
		reconcileOnly := []*ConfigurationItem{everything[0], everything[1]}
		resources := []*CloudResource{
			{ID: 11, Provider: "aliyun", ResourceID: "i-a", TenantID: 1},
			{ID: 22, Provider: "aliyun", ResourceID: "i-b", TenantID: 1}, // 未关联 → add
		}

		svc := newTestService(&mockRepository{
			listCloudResourcesFn: func(ctx context.Context, tenantID int, filter CloudResourceFilter) ([]*CloudResource, error) {
				return resources, nil
			},
			listCIsForReconciliationFn: func(ctx context.Context, tenantID int) ([]*ConfigurationItem, error) {
				return reconcileOnly, nil
			},
			listCIsForGovernanceFn: func(ctx context.Context, tenantID int) ([]*ConfigurationItem, error) {
				return everything, nil
			},
		}).WithCIRequirementReader(&stubRequirementReader{byType: map[int][]string{
			10: {"serial_number", "vendor"},
		}})

		report, err := svc.GetGovernanceReport(context.Background(), 1)
		require.NoError(t, err)

		// 🔴 基数差异：CIs 上报治理基数（3），不是对账基数（2）。
		// 若改回共用对账基数，手工 CI-4 消失 → 断言必红。
		assert.Equal(t, 3, report.CIs, "CIs 必须是全租户基数（含手工 CI）")
		assert.Equal(t, 2, report.CloudResources)

		// 质量指标基于全租户 3 条
		assert.Equal(t, 3, report.Quality.Total)
		assert.Equal(t, 3, report.Quality.Active)
		assert.Equal(t, 0, report.Quality.Retired)
		assert.Equal(t, 1, report.Quality.Stale, "CI-2 同步超 7 天")
		assert.Equal(t, 3, report.Quality.Assessable, "三条都属于有必填定义的类型 10")
		assert.Equal(t, 1, report.Quality.AttributeDefinedTypes)
		// CI-2 缺 vendor
		assert.Equal(t, 1, report.Quality.Incomplete, "只有 CI-2 缺必填 vendor")
		assert.InDelta(t, 66.67, report.Quality.CompletenessPct, 0.01)
		// 缺失统计应指向 vendor
		require.Len(t, report.Quality.MissingByAttribute, 1)
		assert.Equal(t, "vendor", report.Quality.MissingByAttribute[0].Attribute)
		assert.Equal(t, 1, report.Quality.MissingByAttribute[0].Missing)

		// 差异分桶基于对账基数（2 条带云引用的 CI × 2 个云资源）
		assert.Equal(t, map[ReconciliationAction]int{
			ReconcileActionAdd:           1,
			ReconcileActionNoOp:          1,
			ReconcileActionRetireConfirm: 1,
			ReconcileActionDuplicate:     0,
		}, report.Diff)
	})

	t.Run("治理看板纳入确认新鲜度且分母与完整率不同源", func(t *testing.T) {
		// 这一格是能力矩阵里登记的阻断「治理看板未纳入确认新鲜度」的回归锁。
		// 修好之前 report.Certification 全是零值，本测试会红在 Verified 上。
		freshConfirm := now.Add(-20 * 24 * time.Hour)  // 90 天阈值内
		staleConfirm := now.Add(-200 * 24 * time.Hour) // 超 90 天
		everything := []*ConfigurationItem{
			{ID: 1, CITypeID: 10, LifecycleStatus: common.CILifecycleStatusOnline,
				LastConfirmedAt: &freshConfirm}, // 新鲜
			{ID: 2, CITypeID: 10, LifecycleStatus: common.CILifecycleStatusOnline,
				LastConfirmedAt: &staleConfirm}, // 确认过但过期
			{ID: 3, CITypeID: 10, LifecycleStatus: common.CILifecycleStatusOnline},   // 从未确认
			{ID: 4, CITypeID: 10, LifecycleStatus: common.CILifecycleStatusScrapped}, // 退役不进分母
		}
		svc := newTestService(&mockRepository{
			listCIsForGovernanceFn: func(ctx context.Context, tenantID int) ([]*ConfigurationItem, error) {
				return everything, nil
			},
			listCIsForReconciliationFn: func(ctx context.Context, tenantID int) ([]*ConfigurationItem, error) {
				return nil, nil
			},
		}).WithCIRequirementReader(&stubRequirementReader{byType: map[int][]string{
			10: {"serial_number"},
		}})

		report, err := svc.GetGovernanceReport(context.Background(), 1)
		require.NoError(t, err)

		assert.Equal(t, 3, report.Certification.Verified, "退役 CI 不进认证分母")
		assert.Equal(t, 1, report.Certification.FreshConfirmed)
		assert.Equal(t, 1, report.Certification.StaleConfirmed)
		assert.Equal(t, 1, report.Certification.NeverConfirmed)
		assert.InDelta(t, 33.33, report.Certification.FreshnessPct, 0.01)
		assert.Equal(t, 90, report.Certification.FreshThresholdDays,
			"阈值天数必须随响应下发，前端不能硬编码")

		// 🔴 四个 CI 都有类型 10 的必填定义，完整率分母含退役那条（4），
		// 认证分母排除它（3）——两者刻意不同源。
		assert.Equal(t, 4, report.Quality.Assessable, "完整率分母含退役 CI")
		assert.NotEqual(t, report.Quality.Assessable, report.Certification.Verified,
			"两个分母必须有可见差异，合并即回归")
	})

	t.Run("无属性定义时不得报满分", func(t *testing.T) {
		// 全新租户：CI 建了、字段也填了，但没配任何必填属性定义。
		// 旧实现会报 100% 完整率；正确行为是「不可评估」。
		everything := []*ConfigurationItem{
			{ID: 1, CINumber: "CI-1", CITypeID: 10, LifecycleStatus: common.CILifecycleStatusOnline,
				Attributes: map[string]interface{}{"serial_number": "SN-1"}},
			{ID: 2, CINumber: "CI-2", CITypeID: 10, LifecycleStatus: common.CILifecycleStatusOnline},
		}
		svc := newTestService(&mockRepository{
			listCIsForGovernanceFn: func(ctx context.Context, tenantID int) ([]*ConfigurationItem, error) {
				return everything, nil
			},
			listCIsForReconciliationFn: func(ctx context.Context, tenantID int) ([]*ConfigurationItem, error) {
				return nil, nil
			},
		}).WithCIRequirementReader(&stubRequirementReader{byType: map[int][]string{}})

		report, err := svc.GetGovernanceReport(context.Background(), 1)
		require.NoError(t, err)

		assert.Equal(t, 2, report.Quality.Total)
		assert.Equal(t, 0, report.Quality.Assessable, "没有必填定义 → 不可评估")
		assert.Equal(t, 0, report.Quality.Incomplete)
		assert.Equal(t, 0.0, report.Quality.CompletenessPct)
		assert.Equal(t, 0, report.Quality.AttributeDefinedTypes)
	})

	t.Run("未装配读取器时不可评估而非报错", func(t *testing.T) {
		// 治理看板在投影器未装配的环境里仍应能打开；如实呈现「无法评估」。
		svc := newTestService(&mockRepository{
			listCIsForGovernanceFn: func(ctx context.Context, tenantID int) ([]*ConfigurationItem, error) {
				return []*ConfigurationItem{{ID: 1, CITypeID: 10, LifecycleStatus: common.CILifecycleStatusOnline}}, nil
			},
		})
		report, err := svc.GetGovernanceReport(context.Background(), 1)
		require.NoError(t, err, "未装配读取器不应报错")
		assert.Equal(t, 1, report.Quality.Total)
		assert.Equal(t, 0, report.Quality.Assessable)
		assert.Equal(t, 0.0, report.Quality.CompletenessPct)
	})

	t.Run("重复聚类识别同资产两笔账", func(t *testing.T) {
		// 两条 CI 指向同一条云资源：这是真实的重复建账（发现建了一条、手工又建一条）。
		everything := []*ConfigurationItem{
			{ID: 1, CINumber: "CI-1", CITypeID: 10, LifecycleStatus: common.CILifecycleStatusOnline,
				CloudProvider: "aliyun", CloudResourceID: "i-dup", AssetTag: "T-1"},
			{ID: 2, CINumber: "CI-2", CITypeID: 10, LifecycleStatus: common.CILifecycleStatusOnline,
				CloudProvider: "aliyun", CloudResourceID: "i-dup", AssetTag: "T-1"},
			{ID: 3, CINumber: "CI-3", CITypeID: 10, LifecycleStatus: common.CILifecycleStatusOnline,
				CloudProvider: "aliyun", CloudResourceID: "i-unique", AssetTag: "T-3"},
		}
		svc := newTestService(&mockRepository{
			listCIsForGovernanceFn: func(ctx context.Context, tenantID int) ([]*ConfigurationItem, error) {
				return everything, nil
			},
		}).WithCIRequirementReader(&stubRequirementReader{byType: map[int][]string{
			10: {"serial_number"},
		}})

		report, err := svc.GetGovernanceReport(context.Background(), 1)
		require.NoError(t, err)

		require.Len(t, report.DuplicateGroups, 1)
		assert.Equal(t, DuplicateReasonCloudIdentity, report.DuplicateGroups[0].Reason)
		assert.Equal(t, []int{1, 2}, report.DuplicateGroups[0].CIIDs)
		assert.Equal(t, 2, report.DuplicateCICount)
		// 分母用 Active（3），2/3 = 66.67%
		assert.InDelta(t, 66.67, report.DuplicateRatePct, 0.01)
	})

	t.Run("空租户返回全 0 指标而非 nil", func(t *testing.T) {
		svc := newTestService(&mockRepository{})
		report, err := svc.GetGovernanceReport(context.Background(), 42)
		require.NoError(t, err)
		assert.Equal(t, 0, report.Quality.Total)
		assert.Equal(t, 0.0, report.Quality.CompletenessPct)
		assert.Equal(t, 0.0, report.DuplicateRatePct)
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

	t.Run("治理基数读取失败原样上抛", func(t *testing.T) {
		svc := newTestService(&mockRepository{
			listCIsForGovernanceFn: func(ctx context.Context, tenantID int) ([]*ConfigurationItem, error) {
				return nil, assert.AnError
			},
		})
		_, err := svc.GetGovernanceReport(context.Background(), 1)
		assert.ErrorIs(t, err, assert.AnError)
	})

	// 🔴 失败关闭：必填定义读不出来时，完整率算不了。
	// 若降级成「空 specs → 不可评估」继续出报告，运维会看到 Assessable=0 的
	// 「正常」快照，而真实原因是查询坏了——这正是本批要消灭的 fail-open。
	t.Run("必填属性定义读取失败必须失败关闭", func(t *testing.T) {
		svc := newTestService(&mockRepository{
			listCIsForGovernanceFn: func(ctx context.Context, tenantID int) ([]*ConfigurationItem, error) {
				return []*ConfigurationItem{{ID: 1, CITypeID: 10, LifecycleStatus: common.CILifecycleStatusOnline}}, nil
			},
		}).WithCIRequirementReader(&stubRequirementReader{err: assert.AnError})

		report, err := svc.GetGovernanceReport(context.Background(), 1)
		require.Error(t, err, "必填定义读取失败必须返回错误，不得输出残缺快照")
		assert.Nil(t, report)
		assert.ErrorIs(t, err, assert.AnError)
	})

	// 只查出现过的类型，不扫全表（避免 N+1）。
	t.Run("只按出现过的 ciTypeID 查必填定义", func(t *testing.T) {
		reader := &stubRequirementReader{byType: map[int][]string{10: {"a"}, 20: {"b"}}}
		svc := newTestService(&mockRepository{
			listCIsForGovernanceFn: func(ctx context.Context, tenantID int) ([]*ConfigurationItem, error) {
				// ciTypeID<=0 的历史脏数据不应触发查询
				return []*ConfigurationItem{
					{ID: 1, CITypeID: 10, LifecycleStatus: common.CILifecycleStatusOnline},
					{ID: 2, CITypeID: 0, LifecycleStatus: common.CILifecycleStatusOnline},
				}, nil
			},
		}).WithCIRequirementReader(reader)

		_, err := svc.GetGovernanceReport(context.Background(), 1)
		require.NoError(t, err)
		assert.Equal(t, []int{10}, reader.calls, "只应查询出现过的有效类型，且各一次")
	})
}
