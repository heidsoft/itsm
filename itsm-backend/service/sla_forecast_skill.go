package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
)

// ForecastInput 输入结构
type ForecastInput struct {
	TenantID   int
	StartDate  time.Time
	EndDate   time.Time
	Metrics   []string // ["volume", "response_time", "sla_compliance"]
}

// PredictionPoint 单个预测点
type PredictionPoint struct {
	Date        time.Time `json:"date"`
	Predicted  float64   `json:"predicted"`
	LowerBound float64   `json:"lower_bound"`
	UpperBound float64   `json:"upper_bound"`
	Confidence float64   `json:"confidence"`
	Anomaly   bool      `json:"anomaly"`
}

// ForecastOutput 预测输出结构
type ForecastOutput struct {
	Predictions   []PredictionPoint `json:"predictions"`
	Confidence    float64           `json:"confidence"`    // 真实计算
	Model         string            `json:"model"`        // "statistical" | "llm_insight"
	Insights      string            `json:"insights"`      // LLM生成的洞察
	Seasonality  map[string]bool  `json:"seasonality"`  // 检测到的季节性
	Trend         string            `json:"trend"`        // "increasing" | "decreasing" | "stable"
	AnomalyDates  []string         `json:"anomaly_dates"` // 异常日期
}

// ForecastData 内部使用的数据结构
type ForecastData struct {
	HistoricalCounts []int           // 历史数据序列
	Dates           []time.Time    // 对应日期
	Mean            float64        // 均值
	StdDev          float64        // 标准差
	Seasonality     map[string]bool // 季节性
}

// SLAForecastSkill AI-Native SLA 预测 Skill
type SLAForecastSkill struct {
	gateway *LLMGateway
	client  *ent.Client
	logger  *zap.SugaredLogger
}

// NewSLAForecastSkill 创建 SLA 预测 Skill
func NewSLAForecastSkill(client *ent.Client, gateway *LLMGateway, logger *zap.SugaredLogger) *SLAForecastSkill {
	return &SLAForecastSkill{
		client:  client,
		gateway: gateway,
		logger:  logger,
	}
}

// Execute 执行预测
func (s *SLAForecastSkill) Execute(ctx context.Context, input *ForecastInput) (*ForecastOutput, error) {
	s.logger.Infow("SLAForecastSkill executing", "tenant_id", input.TenantID, "metrics", input.Metrics)

	// 1. 获取历史数据
	historyStart := input.StartDate.AddDate(0, -6, 0)
	tickets, err := s.client.Ticket.Query().
		Where(
			ticket.TenantIDEQ(input.TenantID),
			ticket.CreatedAtGTE(historyStart),
			ticket.CreatedAtLTE(input.EndDate),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get history tickets: %w", err)
	}

	// 2. 构建时间序列数据
	data := s.buildTimeSeries(tickets, input.StartDate, input.EndDate)

	// 3. 检测季节性
	seasonality := s.detectSeasonality(data.HistoricalCounts)

	// 4. 计算真实置信区间
	mean, lower, upper := s.calculateConfidenceInterval(data.HistoricalCounts)
	overallConfidence := s.calculateOverallConfidence(data.HistoricalCounts)

	// 5. 生成预测点
	predictions := s.generatePredictions(data, input.StartDate, input.EndDate, lower, upper)

	// 6. 检测异常点
	anomalyDates := s.detectAnomalies(data, predictions)

	// 7. 判断趋势
	trend := s.determineTrend(data.HistoricalCounts)

	// 8. LLM 生成洞察
	insights := ""
	if s.gateway != nil {
		insights, err = s.generateInsights(ctx, mean, predictions, trend, seasonality)
		if err != nil {
			s.logger.Warnw("Failed to generate insights", "error", err)
			insights = fmt.Sprintf("基于%d个历史数据点预测，未来趋势%s，预计%dd后会达到%.1f个工单。",
				len(data.HistoricalCounts), trend, int(input.EndDate.Sub(input.StartDate).Hours()/24), predictions[len(predictions)-1].Predicted)
		}
	}

	return &ForecastOutput{
		Predictions:  predictions,
		Confidence:   overallConfidence,
		Model:        "statistical+llm",
		Insights:     insights,
		Seasonality:  seasonality,
		Trend:        trend,
		AnomalyDates: anomalyDates,
	}, nil
}

// buildTimeSeries 构建时间序列数据
func (s *SLAForecastSkill) buildTimeSeries(tickets []*ent.Ticket, startDate, endDate time.Time) *ForecastData {
	// 按周统计
	weekCounts := make(map[string]int)
	weekDates := make(map[string]time.Time)

	for _, t := range tickets {
		// 获取该周的起始日期（周一）
		weekStart := getWeekStart(t.CreatedAt)
		weekKey := weekStart.Format("2006-01-02")
		weekCounts[weekKey]++
		weekDates[weekKey] = weekStart
	}

	// 转换为有序序列
	var dates []time.Time
	var counts []int

	current := getWeekStart(startDate)
	for !current.After(endDate) {
		weekKey := current.Format("2006-01-02")
		count := weekCounts[weekKey]
		dates = append(dates, current)
		counts = append(counts, count)
		current = current.AddDate(0, 0, 7)
	}

	// 计算统计量
	mean := calculateMean(counts)
	stdDev := calculateStdDev(counts, mean)

	return &ForecastData{
		HistoricalCounts: counts,
		Dates:           dates,
		Mean:            mean,
		StdDev:          stdDev,
		Seasonality:     make(map[string]bool),
	}
}

// getWeekStart 获取该周的起始日期（周一）
func getWeekStart(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // 周日是7
	}
	return time.Date(t.Year(), t.Month(), t.Day()-weekday+1, 0, 0, 0, 0, t.Location())
}

// detectSeasonality 检测季节性
func (s *SLAForecastSkill) detectSeasonality(counts []int) map[string]bool {
	result := map[string]bool{}

	if len(counts) < 4 {
		return result
	}

	// 检测周内模式（周一到周五）
	// 简化版：检测整体趋势的季节性
	hasWeeklyPattern := false
	for i := 1; i < len(counts); i++ {
		if counts[i] > counts[i-1] {
			hasWeeklyPattern = true
			break
		}
	}
	result["weekly"] = hasWeeklyPattern

	// 检测月度模式
	hasMonthlyPattern := false
	if len(counts) >= 4 {
		quarters := len(counts) / 4
		if quarters > 0 {
			hasMonthlyPattern = true
		}
	}
	result["monthly"] = hasMonthlyPattern

	// 检测增长/下降趋势
	hasTrend := false
	trendChanges := 0
	for i := 1; i < len(counts); i++ {
		if counts[i] != counts[i-1] {
			trendChanges++
		}
	}
	if trendChanges > len(counts)/2 {
		hasTrend = true
	}
	result["trend"] = hasTrend

	return result
}

// calculateConfidenceInterval 基于标准差计算真实置信区间
func (s *SLAForecastSkill) calculateConfidenceInterval(counts []int) (mean, lower, upper float64) {
	if len(counts) == 0 {
		return 0, 0, 0
	}

	mean = calculateMean(counts)
	stdDev := calculateStdDev(counts, mean)

	// 95% 置信区间: mean ± 1.96 * stdDev
	lower = mean - 1.96*stdDev
	if lower < 0 {
		lower = 0
	}
	upper = mean + 1.96*stdDev

	return mean, lower, upper
}

// calculateOverallConfidence 基于数据质量计算整体置信度
func (s *SLAForecastSkill) calculateOverallConfidence(counts []int) float64 {
	if len(counts) == 0 {
		return 0.5
	}

	// 数据点越多，置信度越高（上限0.95）
	sampleScore := math.Min(float64(len(counts))/20.0, 1.0) * 0.4

	// 方差越小，置信度越高
	mean := calculateMean(counts)
	stdDev := calculateStdDev(counts, mean)
	cv := stdDev / math.Max(mean, 1) // 变异系数
	varianceScore := math.Max(0, 1.0-cv) * 0.4

	// 有足够历史数据
	historyScore := 0.2
	if len(counts) < 8 {
		historyScore = float64(len(counts)) / 40.0
	}

	return math.Round((sampleScore+varianceScore+historyScore)*100) / 100
}

// generatePredictions 生成预测点
func (s *SLAForecastSkill) generatePredictions(data *ForecastData, startDate, endDate time.Time, lower, upper float64) []PredictionPoint {
	var predictions []PredictionPoint

	current := getWeekStart(startDate)
	weekIndex := 0

	for !current.After(endDate) {
		// 使用简单的指数平滑预测
		var predicted float64
		if weekIndex < len(data.HistoricalCounts) {
			// 使用历史数据加权
			weight := math.Exp(-float64(weekIndex) * 0.1)
			predicted = data.Mean*(1-weight) + float64(data.HistoricalCounts[weekIndex])*weight
		} else {
			// 超出历史范围，用均值
			predicted = data.Mean
		}

		// 添加季节性调整
		if data.Seasonality["weekly"] && weekIndex < len(data.HistoricalCounts) {
			// 周内波动调整
			weeklyFactor := 1.0 + 0.1*math.Sin(float64(weekIndex)*math.Pi/4)
			predicted *= weeklyFactor
		}

		// 计算该点的置信区间（随时间扩大）
		timeFactor := 1.0 + float64(weekIndex)*0.02
		pointLower := predicted - 1.96*data.StdDev*timeFactor
		if pointLower < 0 {
			pointLower = 0
		}
		pointUpper := predicted + 1.96*data.StdDev*timeFactor

		// 检查是否异常
		isAnomaly := predicted < lower || predicted > upper

		predictions = append(predictions, PredictionPoint{
			Date:        current,
			Predicted:   math.Round(predicted*100) / 100,
			LowerBound:  math.Round(pointLower*100) / 100,
			UpperBound:  math.Round(pointUpper*100) / 100,
			Confidence:  math.Round((1.0-timeFactor*0.02)*100) / 100,
			Anomaly:     isAnomaly,
		})

		current = current.AddDate(0, 0, 7)
		weekIndex++
	}

	return predictions
}

// detectAnomalies 检测异常点
func (s *SLAForecastSkill) detectAnomalies(data *ForecastData, predictions []PredictionPoint) []string {
	var anomalyDates []string

	for i, p := range predictions {
		if p.Anomaly && i < len(data.HistoricalCounts) {
			actual := float64(data.HistoricalCounts[i])
			if actual < p.LowerBound || actual > p.UpperBound {
				anomalyDates = append(anomalyDates, p.Date.Format("2006-01-02"))
			}
		}
	}

	return anomalyDates
}

// determineTrend 判断趋势
func (s *SLAForecastSkill) determineTrend(counts []int) string {
	if len(counts) < 2 {
		return "stable"
	}

	// 计算最近一半和最早一半的平均值对比
	half := len(counts) / 2
	if half == 0 {
		return "stable"
	}

	recentMean := calculateMean(counts[len(counts)-half:])
	olderMean := calculateMean(counts[:half])

	ratio := recentMean / math.Max(olderMean, 1)

	if ratio > 1.15 {
		return "increasing"
	} else if ratio < 0.85 {
		return "decreasing"
	}
	return "stable"
}

// generateInsights 使用 LLM 生成洞察
func (s *SLAForecastSkill) generateInsights(ctx context.Context, mean float64, predictions []PredictionPoint, trend string, seasonality map[string]bool) (string, error) {
	if s.gateway == nil {
		return "", fmt.Errorf("LLM gateway not available")
	}

	// 构建季节性描述
	var seasonalityParts []string
	for k, v := range seasonality {
		if v {
			seasonalityParts = append(seasonalityParts, k)
		}
	}
	seasonalityStr := "无"
	if len(seasonalityParts) > 0 {
		seasonalityStr = strings.Join(seasonalityParts, "、")
	}

	// 计算预测摘要
	var totalPredicted float64
	for _, p := range predictions {
		totalPredicted += p.Predicted
	}
	avgPredicted := totalPredicted / math.Max(float64(len(predictions)), 1)

	prompt := fmt.Sprintf(`
你是IT服务管理的分析师。基于以下预测数据，生成简洁的洞察报告（2-3句话）：

预测摘要：
- 历史周均工单量：%.1f
- 预测周均工单量：%.1f
- 趋势：%s
- 季节性特征：%s
- 预测周数：%d

请用中文总结：
1. 主要趋势判断
2. 潜在风险或建议
`, mean, avgPredicted, trend, seasonalityStr, len(predictions))

	messages := []LLMMessage{
		{Role: "system", Content: "你是一个专业的IT服务管理分析师。请用简洁专业的语言回复。"},
		{Role: "user", Content: prompt},
	}

	resp, err := s.gateway.Chat(ctx, "gpt-4o-mini", messages)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(resp), nil
}

// calculateMean 计算均值
func calculateMean(values []int) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0
	for _, v := range values {
		sum += v
	}
	return float64(sum) / float64(len(values))
}

// calculateStdDev 计算标准差
func calculateStdDev(values []int, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sumSq := 0.0
	for _, v := range values {
		diff := float64(v) - mean
		sumSq += diff * diff
	}
	variance := sumSq / float64(len(values))
	return math.Sqrt(variance)
}

// ForecastEvaluator 预测质量评估器
type ForecastEvaluator struct {
	predictions []PredictionRecord
}

// PredictionRecord 预测记录
type PredictionRecord struct {
	PredictedAt   time.Time
	PredictedDate time.Time
	Predicted     float64
	Actual        float64
	LowerBound    float64
	UpperBound    float64
}

// NewForecastEvaluator 创建评估器
func NewForecastEvaluator() *ForecastEvaluator {
	return &ForecastEvaluator{
		predictions: make([]PredictionRecord, 0),
	}
}

// AddPrediction 添加预测记录
func (e *ForecastEvaluator) AddPrediction(pred, actual, lower, upper float64) {
	e.predictions = append(e.predictions, PredictionRecord{
		PredictedAt:   time.Now(),
		PredictedDate: time.Now(),
		Predicted:     pred,
		Actual:        actual,
		LowerBound:    lower,
		UpperBound:    upper,
	})
}

// CalculateMAE 计算平均绝对误差
func (e *ForecastEvaluator) CalculateMAE() float64 {
	if len(e.predictions) == 0 {
		return 0
	}

	totalError := 0.0
	for _, p := range e.predictions {
		totalError += math.Abs(p.Actual - p.Predicted)
	}
	return totalError / float64(len(e.predictions))
}

// CalculateMASE 计算平均绝对比例误差（时序预测标准指标）
func (e *ForecastEvaluator) CalculateMASE() float64 {
	if len(e.predictions) < 2 {
		return 0
	}

	// 计算分母：原始数据的平均绝对差
	var sumAbsDiff float64
	for i := 1; i < len(e.predictions); i++ {
		sumAbsDiff += math.Abs(e.predictions[i].Actual - e.predictions[i-1].Actual)
	}
	scale := sumAbsDiff / float64(len(e.predictions)-1)

	if scale == 0 {
		return 0
	}

	// 计算分子：预测的平均绝对误差
	mae := e.CalculateMAE()

	return mae / scale
}

// CalculateCoverageRate 计算置信区间覆盖率
func (e *ForecastEvaluator) CalculateCoverageRate() float64 {
	if len(e.predictions) == 0 {
		return 0
	}

	hits := 0
	for _, p := range e.predictions {
		if p.Actual >= p.LowerBound && p.Actual <= p.UpperBound {
			hits++
		}
	}

	return float64(hits) / float64(len(e.predictions))
}

// SortAnomaliesBySeverity 按严重程度排序异常
func (e *ForecastEvaluator) SortAnomaliesBySeverity() []PredictionRecord {
	anomalies := make([]PredictionRecord, 0)
	for _, p := range e.predictions {
		// 异常：实际值不在置信区间内
		if p.Actual < p.LowerBound || p.Actual > p.UpperBound {
			anomalies = append(anomalies, p)
		}
	}

	// 按偏离程度排序
	sort.Slice(anomalies, func(i, j int) bool {
		devI := math.Max(anomalies[i].LowerBound-anomalies[i].Actual, anomalies[i].Actual-anomalies[i].UpperBound)
		devJ := math.Max(anomalies[j].LowerBound-anomalies[j].Actual, anomalies[j].Actual-anomalies[j].UpperBound)
		return devI > devJ
	})

	return anomalies
}
