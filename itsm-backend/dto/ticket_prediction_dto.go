package dto

import "time"

// 趋势预测相关DTO
type TrendPredictionRequest struct {
	TimeRange    []string               `json:"time_range" binding:"required,len=2" example:"[\"2024-01-01\",\"2024-12-31\"]"`
	PredictionType string               `json:"prediction_type" binding:"required,oneof=volume type priority resource" example:"volume"`
	Filters      map[string]interface{} `json:"filters,omitempty"`
	Model        string                 `json:"model,omitempty" example:"arima"`
}

type TrendPredictionResponse struct {
	Predictions  []PredictionDataPoint  `json:"predictions"`
	Confidence   float64                `json:"confidence" example:"0.85"`
	Model        string                  `json:"model" example:"arima"`
	GeneratedAt  time.Time               `json:"generated_at" example:"2024-01-01T00:00:00Z"`
}

type PredictionDataPoint struct {
	Date           string   `json:"date" example:"2024-02-01"`
	PredictedValue float64  `json:"predicted_value" example:"45.0"`
	LowerBound     float64  `json:"lower_bound" example:"38.0"`
	UpperBound     float64  `json:"upper_bound" example:"52.0"`
	Confidence     float64  `json:"confidence" example:"0.85"`
	Category       string   `json:"category,omitempty" example:"database"`
	Priority       string   `json:"priority,omitempty" example:"high"`
}
