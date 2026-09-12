package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAIMetrics_JSONShape_CamelCase P1-4（2026-09-06 UAT 修复）：
// AIMetrics 结构体的 json tag 全部 camelCase，前端 TS 类型 AIMetrics 字段对齐。
// 本测试无需 sqlite/PG 即可保证序列化契约——任何人误改 json tag 即失败。
func TestAIMetrics_JSONShape_CamelCase(t *testing.T) {
	m := &AIMetrics{
		TotalRequests:          5,
		TotalFeedback:          3,
		UsefulFeedback:         2,
		UsefulRate:             0.666,
		ByKind:                 map[string]interface{}{"rag": 3, "triage": 2},
		AvgResponseTimeSeconds: 16.5,
		LLMCallCount:           33,
		ResponseTimeAvailable:  true,
	}
	raw, err := json.Marshal(m)
	require.NoError(t, err)

	// 必备 camelCase key（与前端 AIMetrics 接口一一对齐）
	keys := []string{
		"totalRequests",
		"totalFeedback",
		"usefulFeedback",
		"usefulRate",
		"byKind",
		"avgResponseTimeSeconds",
		"llmCallCount",
		"responseTimeAvailable",
	}
	for _, k := range keys {
		require.Contains(t, string(raw), `"`+k+`"`, "AIMetrics JSON must contain %s", k)
	}

	// 显式禁止 snake_case 泄漏
	forbidden := []string{
		"total_requests",
		"total_feedback",
		"useful_feedback",
		"useful_rate",
		"by_kind",
		"avg_response_time_seconds",
		"llm_call_count",
		"response_time_available",
	}
	for _, k := range forbidden {
		require.NotContains(t, string(raw), `"`+k+`"`, "snake_case key %q must not leak", k)
	}
}
