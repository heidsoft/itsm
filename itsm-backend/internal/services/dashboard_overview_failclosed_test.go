package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"itsm-backend/ent/enttest"

	baseent "entgo.io/ent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// dashboardAllFault 让第 nth 次 All 查询失败，其余查询正常。
func dashboardAllFault(nth int) baseent.Interceptor {
	seen := 0
	return baseent.InterceptFunc(func(next baseent.Querier) baseent.Querier {
		return baseent.QuerierFunc(func(ctx context.Context, q baseent.Query) (baseent.Value, error) {
			qc := baseent.QueryFromContext(ctx)
			if qc != nil && qc.Op == baseent.OpQueryAll {
				seen++
				if seen == nth {
					return nil, fmt.Errorf("injected dashboard All failure #%d", nth)
				}
			}
			return next.Query(ctx, q)
		})
	})
}

func newDashboardGetterFaultService(t *testing.T, inters ...baseent.Interceptor) *DashboardService {
	t.Helper()
	dsn := fmt.Sprintf("file:dash_getter_fault_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { _ = client.Close() })
	client.Intercept(inters...)
	return &DashboardService{client: client, logger: zaptest.NewLogger(t).Sugar()}
}

// TestDashboardGetters_FailClosedOnQueryError 锁住看板子分析的 fail-closed 契约：
// GetDashboardOverview 会把子分析的错误上抛（与 KPI/趋势/SLA 同口径），因此每个
// getter 自身必须在 DB 查询失败时返回 error，而不能吞掉后返回空数组——「查询故障」
// 被伪装成「最近无活动 / 无满意度 / 无响应时间分布 / 无团队负载 / 无高峰数据」。
// responseTime/teamWorkload/peakHours 旧实现已上抛；recentActivities 与 satisfaction
// 旧实现 `return []..., nil` 吞错，是本次修复点。
func TestDashboardGetters_FailClosedOnQueryError(t *testing.T) {
	for _, tc := range []struct {
		name string
		call func(*DashboardService) error
	}{
		{"recentActivities", func(s *DashboardService) error {
			_, err := s.getRecentActivitiesForDashboard(context.Background(), 1, 10)
			return err
		}},
		{"satisfaction", func(s *DashboardService) error {
			_, err := s.getSatisfactionDataForDashboard(context.Background(), 1, 4)
			return err
		}},
		{"responseTimeDistribution", func(s *DashboardService) error {
			_, err := s.getResponseTimeDistribution(context.Background(), 1)
			return err
		}},
		{"teamWorkload", func(s *DashboardService) error {
			_, err := s.getTeamWorkload(context.Background(), 1)
			return err
		}},
		{"peakHours", func(s *DashboardService) error {
			_, err := s.getPeakHours(context.Background(), 1)
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := newDashboardGetterFaultService(t, dashboardAllFault(1))
			err := tc.call(svc)
			require.Error(t, err, "%s 查询失败必须上抛，不得伪装成空数据", tc.name)
			assert.Contains(t, err.Error(), "injected dashboard All failure")
		})
	}
}
