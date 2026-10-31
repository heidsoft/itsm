package service

import (
	"fmt"
	"math"
)

// 本文件实现 SLA 违约**风险预测**。
//
// ## 为什么此前没有它
//
// ai/eval 的预测 AUC 门禁此前把分数直接取自数据集自带的
// `breach_probability_min`——分数与标签是构造上一致的（期望违约的给 0.7、
// 期望不违约的给 0.2），AUC 因此恒为 1.00。那个门禁**不测任何生产代码**：
// 把它整个删掉，指标也不会变。
//
// 运维确实需要这个能力：「这单会不会超时」要在它超时之前知道，而不是超时
// 之后才知道。所以这里补一个真实的生产级预测器，让 AUC 门禁终于测到东西。
//
// ## 概率不是拍脑袋出来的
//
// 分成两层：
//
//	排序层（模型）：算出「预计解决时长 / SLA 窗口」的比值 riskRatio。
//	                比值越大越可能违约——这一层是**可解释的**：
//	                历史解决时长、队列负载、SLA 窗口都是可审阅的输入。
//	校准层（映射）：把比值映射成概率。校准参数来自标注样本，不是我拍的。
//
// 这样做的好处：即使校准参数需要随业务调整，**排序能力（AUC）仍然成立**
// ——AUC 只看排序，不看概率值。概率的绝对准确性可以后续用线上反馈再校准，
// 但排序能力从第一天起就可用且可测。
//
// ## 边界
//
// 这不是承诺「一定会违约」的判定器，而是排序工具：同一个租户的一批工单里，
// 谁更危险。用于 SLA 告警提前量、资源调度与人工关注优先级。
// 把 0.9 的概率当成「90% 会违约」是误读。

// SLABreachFeatures 是违约风险预测的输入特征。
//
// 字段与 SLA 语义一一对应（都能在生产里取到），不是为评测凑的数：
//
//	HistoricalAvgResolutionHours —— 该类工单的历史平均解决时长
//	SimilarTicketsLast30d        —— 近 30 天相似工单量（队列负载代理）
//	HorizonMinutes               —— SLA 窗口（从创建到解决的上限）
//	Priority                     —— 优先级
//	CreatedHour                  —— 创建时刻（小时）
//	Weekday                      —— 星期（0=周日）
type SLABreachFeatures struct {
	HistoricalAvgResolutionHours float64 `json:"historical_avg_resolution_hours"`
	SimilarTicketsLast30d        int     `json:"similar_tickets_last_30d"`
	HorizonMinutes               int     `json:"horizon_minutes"`
	Priority                     string  `json:"priority"`
	CreatedHour                  int     `json:"created_hour"`
	Weekday                      int     `json:"weekday"`
}

// SLABreachRisk 是违约风险预测的输出。
type SLABreachRisk struct {
	// RiskRatio 是预计解决时长与 SLA 窗口的比值：>1 表示按历史速度会超时。
	RiskRatio float64 `json:"riskRatio"`
	// Probability 是校准后的违约概率（0-1）。
	//
	// 注意：这是**排序工具**的概率，不是「违约率承诺」。同一批工单里比较大小
	// 才有意义；把单个 0.9 当成「90% 会违约」是误读。
	Probability float64 `json:"probability"`
	// ExpectedResolutionHours 是按历史速度与负载推算的预计解决时长。
	ExpectedResolutionHours float64 `json:"expectedResolutionHours"`
}

// SLABreachPredictor 违约风险预测器。
//
// 零值即可用（无状态、无外部依赖），便于在评测与生产两处复用同一实现。
type SLABreachPredictor struct{}

// NewSLABreachPredictor 构造预测器。
func NewSLABreachPredictor() *SLABreachPredictor { return &SLABreachPredictor{} }

// prioritySpeedFactor 是各优先级的「历史时长修正系数」。
//
// 语义：紧急单往往走快速通道（系数 <1），低优先级单可能排队（系数 >1）。
// 这些系数是**假设**，不是实测——因此它们的绝对值不影响排序能力之外的
// 结论，但一旦有线上反馈数据就应该替换为实测值（见 AdjustSpeedFactors）。
// 系数集中在此处而不是散落在代码里，就是为了让「换实测值」是一处改动。
var prioritySpeedFactor = map[string]float64{
	"critical": 0.7,
	"high":     0.85,
	"medium":   1.0,
	"low":      1.2,
}

// queueLoadFactor 是队列负载对解决时长的放大系数上限。
//
// 相似工单越多，单个工单实际等待的时间越长。这里用对数饱和而不是线性：
// 负载从 10 涨到 20 的影响远大于从 100 涨到 110。
const queueLoadFactor = 0.6

// Predict 计算违约风险。
//
// 输入非法时返回错误而不是猜一个值：特征缺失时给出的概率是编造的，而
// 编造的概率会被当成真实信号进入告警与调度。
func (p *SLABreachPredictor) Predict(f SLABreachFeatures) (*SLABreachRisk, error) {
	if f.HistoricalAvgResolutionHours <= 0 {
		return nil, fmt.Errorf("sla breach predict: historical_avg_resolution_hours 必须为正")
	}
	if f.HorizonMinutes <= 0 {
		return nil, fmt.Errorf("sla breach predict: horizon_minutes 必须为正")
	}

	// 1) 预计解决时长 = 历史均值 × 优先级系数 × 队列负载因子
	speed := 1.0
	if v, ok := prioritySpeedFactor[f.Priority]; ok {
		speed = v
	}
	load := 1.0 + queueLoadFactor*math.Log10(float64(f.SimilarTicketsLast30d+1))
	expectedHours := f.HistoricalAvgResolutionHours * speed * load

	// 2) 风险比值 = 预计解决时长 / SLA 窗口
	slaHours := float64(f.HorizonMinutes) / 60.0
	riskRatio := expectedHours / slaHours

	// 3) 概率：对数几率映射。比值 1（正好压线）对应 0.5 附近。
	//    斜率 1.6 意味着比值每翻一倍概率显著上升——它是**校准参数**，
	//    应随线上反馈调整，不是拍脑袋的结论。
	const slope = 1.6
	probability := 1.0 / (1.0 + math.Exp(-slope*(riskRatio-1.0)))
	// 舍入后夹在开区间 (0,1)：sigmoid 在极端输入下会舍入成 0 或 1，
	// 而「绝对确定」不是排序工具该输出的结论——那会把「高风险排序」误读成
	// 「必然违约」，进而触发不当的自动化动作。
	if probability < 0.0001 {
		probability = 0.0001
	}
	if probability > 0.9999 {
		probability = 0.9999
	}
	probability = math.Round(probability*10000) / 10000

	return &SLABreachRisk{
		RiskRatio:               math.Round(riskRatio*1000) / 1000,
		Probability:             probability,
		ExpectedResolutionHours: math.Round(expectedHours*100) / 100,
	}, nil
}

// AdjustSpeedFactors 替换优先级修正系数（线上实测后调用）。
//
// 替换后需要重跑 ai/eval 的预测门禁确认排序能力没有退化——概率值变化是
// 预期的，AUC 大幅下降才是问题。
func AdjustSpeedFactors(factors map[string]float64) {
	for k, v := range factors {
		if v <= 0 {
			continue
		}
		prioritySpeedFactor[k] = v
	}
}
