package cmdb

import (
	"testing"
	"time"

	"itsm-backend/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 确认新鲜度（CMDB Health 的 Compliance 维度）纯函数锁。
//
// 这组测试的重点不是「算出几个数」，而是锁住三条容易被后人改坏的口径：
//  1. 三个计数互斥完备（和恒等于分母）——下游画堆叠条的信任基础；
//  2. 退役 CI 不进分母——要求已退役资产「确认仍然准确」是无意义要求；
//  3. 分母与完整率的 Assessable 不同源——合并会让两个指标互相污染。

func newCIAt(t time.Time) *ConfigurationItem {
	confirmed := t
	return &ConfigurationItem{LastConfirmedAt: &confirmed}
}

// TestComputeCertificationMetrics_PartitionIsExhaustive 锁互斥完备不变量：
// NeverConfirmed + StaleConfirmed + FreshConfirmed == Verified。
// 反向验证：把任一分支改成「也计数到另一桶」→ 本测试立即红在等值断言上。
func TestComputeCertificationMetrics_PartitionIsExhaustive(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	const fresh = 90 * 24 * time.Hour

	cis := []*ConfigurationItem{
		{ID: 1, LifecycleStatus: common.CILifecycleStatusOnline},                         // 从未确认
		{ID: 2, LifecycleStatus: common.CILifecycleStatusOnline, LastConfirmedAt: nil},   // 显式 nil
		newCIAt(now.Add(-10 * 24 * time.Hour)),                                           // 新鲜
		newCIAt(now.Add(-89 * 24 * time.Hour)),                                           // 临界内（差 1 天）
		newCIAt(now.Add(-91 * 24 * time.Hour)),                                           // 刚好过期
		newCIAt(now.Add(-400 * 24 * time.Hour)),                                          // 严重过期
		{ID: 7, LifecycleStatus: common.CILifecycleStatusScrapped, LastConfirmedAt: nil}, // 退役：不进分母
		{ID: 8, LifecycleStatus: common.CILifecycleStatusScrapped, LastConfirmedAt: func() *time.Time {
			ts := now.Add(-500 * 24 * time.Hour)
			return &ts
		}()}, // 退役即使确认过也不进分母
		nil, // 防御 nil 元素
	}

	m := ComputeCertificationMetrics(cis, fresh, now)

	require.Equal(t, 6, m.Verified, "分母应排除 2 条退役 CI 与 1 条 nil")
	assert.Equal(t, 2, m.NeverConfirmed, "ID1 与 ID2 从未确认")
	assert.Equal(t, 2, m.StaleConfirmed, "ID5(91d) 与 ID6(400d) 过期")
	assert.Equal(t, 2, m.FreshConfirmed, "ID3(10d) 与 ID4(89d) 新鲜")

	// 不变量：三桶之和恒等于分母。任何一条 CI 落入两桶或落空都会打破它。
	assert.Equal(t, m.Verified, m.NeverConfirmed+m.StaleConfirmed+m.FreshConfirmed,
		"三个计数必须互斥且完备")

	assert.Equal(t, 33.33, m.FreshnessPct, "2/6 = 33.33%")
	assert.Equal(t, 90, m.FreshThresholdDays, "阈值天数必须随响应下发")
}

// TestComputeCertificationMetrics_RetiredNotDemanded 锁「退役不进分母」。
// 若把退役也算进去，一条 1000 条资产、只有 5 条在用的租户会永远显示 0.5%
// 认证率，运维直接放弃这个指标。
func TestComputeCertificationMetrics_RetiredNotDemanded(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	cis := []*ConfigurationItem{
		newCIAt(now.Add(-1 * time.Hour)),
		newCIAt(now.Add(-1 * time.Hour)),
		{ID: 3, LifecycleStatus: common.CILifecycleStatusScrapped},
		{ID: 4, LifecycleStatus: common.CILifecycleStatusScrapped},
		{ID: 5, LifecycleStatus: common.CILifecycleStatusScrapped},
	}
	m := ComputeCertificationMetrics(cis, 90*24*time.Hour, now)
	assert.Equal(t, 2, m.Verified)
	assert.Equal(t, 100.0, m.FreshnessPct, "在用资产全部新鲜即 100%，不因退役资产被稀释")
}

// TestComputeCertificationMetrics_EmptyIsZeroNotHundred 锁空集语义。
// 🔴 Verified=0 → 0%，不可解读为「全部过期」，也不可解读为「全部新鲜」。
// 这是治理看板最容易被误读的一格：新租户打开看到 0%，需要文案说清「尚无资产」。
func TestComputeCertificationMetrics_EmptyIsZeroNotHundred(t *testing.T) {
	m := ComputeCertificationMetrics(nil, 90*24*time.Hour, time.Now())
	assert.Equal(t, 0, m.Verified)
	assert.Equal(t, 0.0, m.FreshnessPct, "无资产时必须是 0，不能是 100")
	assert.Equal(t, 0, m.NeverConfirmed+m.StaleConfirmed+m.FreshConfirmed)
}

// TestComputeCertificationMetrics_ZeroThresholdAllStale 锁阈值边界：
// threshold<=0 时任何「确认过的」都算过期（now.Sub > 0 恒成立）。
// 这个退化行为不该静默变成 100%。
func TestComputeCertificationMetrics_ZeroThresholdAllStale(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	cis := []*ConfigurationItem{newCIAt(now.Add(-time.Second))}
	m := ComputeCertificationMetrics(cis, 0, now)
	assert.Equal(t, 1, m.Verified)
	assert.Equal(t, 0, m.FreshConfirmed)
	assert.Equal(t, 1, m.StaleConfirmed)
}

// TestComputeCertificationMetrics_DenominatorDiffersFromAssessable 锁「认证分母
// 与完整率分母不是同一个数」。
//
// 用一个退役且不可评估的 CI 证明：它既不在 Assessable（无必填定义）也不在
// Verified（已退役）。两者都排除它，但依据不同——若后人把两处合并成「同一个
// 分母」，任一侧的语义就会被另一侧绑架。
func TestComputeCertificationMetrics_DenominatorDiffersFromAssessable(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	cis := []*ConfigurationItem{
		{ID: 1, CITypeID: 99, LifecycleStatus: common.CILifecycleStatusOnline}, // 无必填定义 → 不在 Assessable，但在 Verified
	}

	attr := ComputeAttributeQuality(cis, map[int][]string{}, 7*24*time.Hour, now)
	cert := ComputeCertificationMetrics(cis, 90*24*time.Hour, now)

	assert.Equal(t, 0, attr.Assessable, "无必填定义的类型不进完整率分母")
	assert.Equal(t, 1, cert.Verified, "但它是在用资产，必须进认证分母")
	assert.NotEqual(t, attr.Assessable, cert.Verified,
		"两个分母刻意不同源，相等只是本次数据的巧合而非契约")
}
