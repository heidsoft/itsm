package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAIMetrics_JSONShape_CamelCase 锁定 AIMetrics 的序列化契约：
// json tag 全部 camelCase，与前端 TS 类型 AIMetrics 字段一致。
// 不需要 sqlite/PG，误改 json tag 就会失败。
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
