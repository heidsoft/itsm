package sla

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// statsRepoStub 只实现统计路径使用的方法。嵌入 nil Repository 让其余方法
// 被误调用时立即 panic，而不是悄悄返回零值掩盖接线错误。
type statsRepoStub struct {
	Repository
	definitions int
	active      int
	violations  int
	open        int
	ticketTotal int
	metSLA      int
	ticketErr   error
}

func (s statsRepoStub) CountDefinitions(_ context.Context, _ int, isActive *bool) (int, error) {
	if isActive != nil {
		return s.active, nil
	}
	return s.definitions, nil
}

func (s statsRepoStub) CountViolations(_ context.Context, _ int, isResolved *bool) (int, error) {
	if isResolved != nil {
		return s.open, nil
	}
	return s.violations, nil
}

func (s statsRepoStub) GetTicketStats(_ context.Context, _ int) (int, int, error) {
	return s.ticketTotal, s.metSLA, s.ticketErr
}

// 统计值来自 Count 查询，不受列表分页上限约束：
// 旧实现对 ListDefinitions/ListViolations 取 len，超过 1000 条时恒等于 1000。
func TestGetSLAStats_CountsBeyondListPageLimit(t *testing.T) {
	svc := NewService(statsRepoStub{
		definitions: 1500, active: 1200,
		violations: 1100, open: 50,
		ticketTotal: 2000, metSLA: 1900,
	}, zaptest.NewLogger(t).Sugar())

	stats, err := svc.GetSLAStats(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, 1500, stats.TotalDefinitions)
	assert.Equal(t, 1200, stats.ActiveDefinitions)
	assert.Equal(t, 1100, stats.TotalViolations)
	assert.Equal(t, 50, stats.OpenViolations)
	assert.Equal(t, 95.0, stats.OverallComplianceRate)
}

// 工单统计查询失败不能被当成「没有样本」降级为 0：
// 那会把一次数据库故障伪装成「全员不合规」。
func TestGetSLAStats_TicketStatsErrorIsPropagated(t *testing.T) {
	svc := NewService(statsRepoStub{
		definitions: 1, active: 1,
		ticketErr: errors.New("db down"),
	}, zaptest.NewLogger(t).Sugar())

	_, err := svc.GetSLAStats(context.Background(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "统计工单达标情况失败")
}

// 完全无样本时诚实返回 0；有历史违规但无工单时按违规处理率给出可解释读数。
func TestGetSLAStats_ZeroSamplesReportZero(t *testing.T) {
	svc := NewService(statsRepoStub{}, zaptest.NewLogger(t).Sugar())
	stats, err := svc.GetSLAStats(context.Background(), 1)
	require.NoError(t, err)
	assert.Zero(t, stats.TotalDefinitions)
	assert.Zero(t, stats.OverallComplianceRate)
}
