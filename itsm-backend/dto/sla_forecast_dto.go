package dto

import "time"

// SLA 预测的输入/输出值类型（原定义在共享层 service/sla_forecast_skill.go）。
// Viewer 复用同包 DashboardViewer，别名身份与 service 侧完全一致。

type ForecastInput struct {
	TenantID  int
	StartDate time.Time
	EndDate   time.Time
	Metrics   []string // ["volume", "response_time", "sla_compliance"]
	// Viewer 是读取者身份（R1）。SLAForecastSkill 是 /api/v1/ai/predictions
	// 的首选实现，此前与 legacy 预测一样只按 tenant 拉全量工单；零值 viewer
	// 按最窄档收窄为空集，不会静默放宽。后台无请求者上下文的调用须显式
	// 传 ScopeAll，而不是省略。
	Viewer DashboardViewer
}

type PredictionPoint struct {
	Date       time.Time `json:"date"`
	Predicted  float64   `json:"predicted"`
	LowerBound float64   `json:"lowerBound"`
	UpperBound float64   `json:"upperBound"`
	Confidence float64   `json:"confidence"`
	Anomaly    bool      `json:"anomaly"`
}

type ForecastOutput struct {
	Predictions  []PredictionPoint `json:"predictions"`
	Confidence   float64           `json:"confidence"`   // 真实计算
	Model        string            `json:"model"`        // "statistical" | "llm_insight"
	Insights     string            `json:"insights"`     // LLM生成的洞察
	Seasonality  map[string]bool   `json:"seasonality"`  // 检测到的季节性
	Trend        string            `json:"trend"`        // "increasing" | "decreasing" | "stable"
	AnomalyDates []string          `json:"anomalyDates"` // 异常日期
}

type ForecastData struct {
	HistoricalCounts []int           // 历史数据序列
	Dates            []time.Time     // 对应日期
	Mean             float64         // 均值
	StdDev           float64         // 标准差
	Seasonality      map[string]bool // 季节性
}
