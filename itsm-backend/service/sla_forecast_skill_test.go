package service

import (
	"context"
	"testing"
	"time"

	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestSLAForecastSkill_Execute(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	skill := NewSLAForecastSkill(client, nil, logger)

	ctx := context.Background()
	startDate := time.Now().AddDate(0, -2, 0)
	endDate := time.Now()

	input := &ForecastInput{
		TenantID:  1,
		StartDate: startDate,
		EndDate:   endDate,
		Metrics:   []string{"volume"},
	}

	output, err := skill.Execute(ctx, input)
	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, "statistical+llm", output.Model)
	assert.NotEmpty(t, output.Predictions)
}

func TestSLAForecastSkill_CalculateConfidenceInterval(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	skill := NewSLAForecastSkill(client, nil, logger)

	tests := []struct {
		name   string
		counts []int
	}{
		{"empty", []int{}},
		{"single", []int{10}},
		{"stable", []int{10, 10, 10, 10}},
		{"growing", []int{5, 8, 12, 15}},
		{"volatile", []int{1, 20, 5, 15}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mean, lower, upper := skill.calculateConfidenceInterval(tt.counts)
			if len(tt.counts) == 0 {
				assert.Equal(t, 0.0, mean)
				assert.Equal(t, 0.0, lower)
				assert.Equal(t, 0.0, upper)
			} else {
				assert.GreaterOrEqual(t, mean, 0.0)
				assert.LessOrEqual(t, lower, mean)
				assert.GreaterOrEqual(t, upper, mean)
			}
		})
	}
}

func TestSLAForecastSkill_CalculateOverallConfidence(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	skill := NewSLAForecastSkill(client, nil, logger)

	tests := []struct {
		name    string
		counts  []int
		minConf float64
		maxConf float64
	}{
		{"empty", []int{}, 0.5, 0.5},
		{"single", []int{10}, 0.0, 0.5},
		{"few", []int{1, 2, 3, 4}, 0.0, 0.5},
		{"many", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 0.3, 0.95},
		{"stable", []int{10, 10, 10, 10, 10, 10, 10, 10, 10, 10}, 0.5, 0.95},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := skill.calculateOverallConfidence(tt.counts)
			assert.GreaterOrEqual(t, conf, tt.minConf)
			assert.LessOrEqual(t, conf, tt.maxConf)
		})
	}
}

func TestSLAForecastSkill_DetermineTrend(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	skill := NewSLAForecastSkill(client, nil, logger)

	tests := []struct {
		name     string
		counts   []int
		expected string
	}{
		{"empty", []int{}, "stable"},
		{"single", []int{10}, "stable"},
		{"stable", []int{10, 10, 10, 10}, "stable"},
		{"increasing", []int{5, 7, 9, 12, 15}, "increasing"},
		{"decreasing", []int{15, 12, 9, 7, 5}, "decreasing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trend := skill.determineTrend(tt.counts)
			assert.Equal(t, tt.expected, trend)
		})
	}
}

func TestSLAForecastSkill_DetectSeasonality(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	skill := NewSLAForecastSkill(client, nil, logger)

	tests := []struct {
		name     string
		counts   []int
		wantKeys int
	}{
		{"empty", []int{}, 0},
		{"single", []int{10}, 0},
		{"few", []int{1, 2, 3}, 0},
		{"many", []int{1, 2, 3, 4, 5, 6, 7, 8}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := skill.detectSeasonality(tt.counts)
			assert.Len(t, result, tt.wantKeys)
		})
	}
}

func TestForecastEvaluator_CalculateMAE(t *testing.T) {
	evaluator := NewForecastEvaluator()

	// Add predictions with known errors
	evaluator.AddPrediction(10, 12, 8, 14) // error = 2
	evaluator.AddPrediction(10, 8, 8, 14)  // error = 2
	evaluator.AddPrediction(10, 11, 8, 14) // error = 1

	mae := evaluator.CalculateMAE()
	assert.InDelta(t, 1.667, mae, 0.01)
}

func TestForecastEvaluator_CalculateMASE(t *testing.T) {
	evaluator := NewForecastEvaluator()

	// Simple data: predictions perfect, actuals match
	evaluator.AddPrediction(10, 10, 8, 12)
	evaluator.AddPrediction(12, 12, 10, 14)
	evaluator.AddPrediction(14, 14, 12, 16)

	mase := evaluator.CalculateMASE()
	assert.InDelta(t, 0, mase, 0.01)
}

func TestForecastEvaluator_CalculateCoverageRate(t *testing.T) {
	evaluator := NewForecastEvaluator()

	// All predictions within bounds
	evaluator.AddPrediction(10, 10, 8, 12)
	evaluator.AddPrediction(10, 9, 8, 12)
	evaluator.AddPrediction(10, 11, 8, 12)

	rate := evaluator.CalculateCoverageRate()
	assert.InDelta(t, 1.0, rate, 0.01)

	// One out of bounds
	evaluator2 := NewForecastEvaluator()
	evaluator2.AddPrediction(10, 10, 8, 12)
	evaluator2.AddPrediction(10, 15, 8, 12) // out of bounds

	rate2 := evaluator2.CalculateCoverageRate()
	assert.InDelta(t, 0.5, rate2, 0.01)
}

func TestForecastEvaluator_SortAnomaliesBySeverity(t *testing.T) {
	evaluator := NewForecastEvaluator()

	// Add predictions with different deviation levels
	evaluator.AddPrediction(10, 5, 8, 12)  // deviation = 5 (most severe)
	evaluator.AddPrediction(10, 11, 8, 12) // deviation = 1 (within bounds)
	evaluator.AddPrediction(10, 20, 8, 12) // deviation = 8 (most severe)

	anomalies := evaluator.SortAnomaliesBySeverity()
	assert.Len(t, anomalies, 2) // Only 2 are anomalies (actual outside bounds)
	assert.Greater(t, anomalies[0].Actual-anomalies[0].UpperBound,
		anomalies[1].Actual-anomalies[1].UpperBound)
}
