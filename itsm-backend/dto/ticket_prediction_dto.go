package dto

import "time"

// 趋势预测相关DTO
type TrendPredictionRequest struct {
	TimeRange      []string               `json:"timeRange" binding:"required,len=2" example:"[\"2024-01-01\",\"2024-12-31\"]"`
	PredictionType string                 `json:"predictionType" binding:"required,oneof=volume type priority resource" example:"volume"`
	Filters        map[string]interface{} `json:"filters,omitempty"`
	Model          string                 `json:"model,omitempty" example:"arima"`
}

type TrendPredictionResponse struct {
	Predictions []PredictionDataPoint `json:"predictions"`
	Confidence  float64               `json:"confidence" example:"0.85"`
	Model       string                `json:"model" example:"arima"`
	GeneratedAt time.Time             `json:"generatedAt" example:"2024-01-01T00:00:00Z"`
}

type PredictionDataPoint struct {
	Date           string  `json:"date" example:"2024-02-01"`
	PredictedValue float64 `json:"predictedValue" example:"45.0"`
	LowerBound     float64 `json:"lowerBound" example:"38.0"`
	UpperBound     float64 `json:"upperBound" example:"52.0"`
	Confidence     float64 `json:"confidence" example:"0.85"`
	Category       string  `json:"category,omitempty" example:"database"`
	Priority       string  `json:"priority,omitempty" example:"high"`
}
