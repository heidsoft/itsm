package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 本文件守卫 SLA 违约预测器的性质。
//
// 背景：ai/eval 的预测 AUC 门禁此前把分数直接取自数据集自带的
// breach_probability_min——分数与标签构造上一致（期望违约的给 0.7、不违约
// 的给 0.2），AUC 恒 1.00，那个门禁不测任何生产代码。本预测器让 AUC 终于
// 测到真实实现，前提是它自己的排序语义正确。

// TestPredictorRanksRiskCorrectly 锁住排序能力（AUC 只看排序）：
// 历史均值超过 SLA 窗口的工单，风险分必须高于历史均值远低于窗口的工单。
func TestPredictorRanksRiskCorrectly(t *testing.T) {
	p := NewSLABreachPredictor()

	high, err := p.Predict(SLABreachFeatures{
		HistoricalAvgResolutionHours: 8,
		HorizonMinutes:               240, // 4h 窗口，历史 8h → 必超
		Priority:                     "medium",
		SimilarTicketsLast30d:        5,
	})
	require.NoError(t, err)

	low, err := p.Predict(SLABreachFeatures{
		HistoricalAvgResolutionHours: 1,
		HorizonMinutes:               240,
		Priority:                     "medium",
		SimilarTicketsLast30d:        5,
	})
	require.NoError(t, err)

	assert.Greater(t, high.Probability, low.Probability,
		"历史均值 8h/窗口 4h 的风险必须高于 1h/4h")
	assert.Greater(t, high.RiskRatio, 1.0, "超窗口的比值应 >1")
	assert.Less(t, low.RiskRatio, 1.0, "远低于窗口的比值应 <1")
}

// TestPredictorQueueLoadIncreasesRisk 队列负载加重必须推高风险：
// 相似工单越多，预计解决时长越长。
func TestPredictorQueueLoadIncreasesRisk(t *testing.T) {
	p := NewSLABreachPredictor()
	base := SLABreachFeatures{
		HistoricalAvgResolutionHours: 2,
		HorizonMinutes:               240,
		Priority:                     "high",
		SimilarTicketsLast30d:        1,
	}
	light, err := p.Predict(base)
	require.NoError(t, err)

	heavy := base
	heavy.SimilarTicketsLast30d = 100
	loaded, err := p.Predict(heavy)
	require.NoError(t, err)

	assert.Greater(t, loaded.Probability, light.Probability,
		"队列负载 100 单的风险必须高于 1 单")
}

// TestPredictorRejectsInvalidFeatures 非法特征报错而不是猜值：
// 编造的概率会进入告警与调度，比缺一个数更糟。
func TestPredictorRejectsInvalidFeatures(t *testing.T) {
	p := NewSLABreachPredictor()
	_, err := p.Predict(SLABreachFeatures{HistoricalAvgResolutionHours: 0, HorizonMinutes: 240})
	require.Error(t, err, "历史均值为 0 必须报错")

	_, err = p.Predict(SLABreachFeatures{HistoricalAvgResolutionHours: 1, HorizonMinutes: 0})
	require.Error(t, err, "SLA 窗口为 0 必须报错")
}

// TestPredictorProbabilityRange 概率必须在 (0,1) 开区间内：两端意味着
// 「绝对确定」，而排序工具不应该输出确定性结论。
func TestPredictorProbabilityRange(t *testing.T) {
	p := NewSLABreachPredictor()
	for _, f := range []SLABreachFeatures{
		{HistoricalAvgResolutionHours: 0.01, HorizonMinutes: 100000, Priority: "low", SimilarTicketsLast30d: 0},
		{HistoricalAvgResolutionHours: 1000, HorizonMinutes: 1, Priority: "critical", SimilarTicketsLast30d: 999},
	} {
		risk, err := p.Predict(f)
		require.NoError(t, err)
		assert.Greater(t, risk.Probability, 0.0)
		assert.Less(t, risk.Probability, 1.0, "排序工具不得输出绝对确定的 0 或 1")
	}
}
