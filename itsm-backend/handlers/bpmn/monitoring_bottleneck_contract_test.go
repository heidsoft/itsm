package bpmn

import (
	"testing"
	"time"

	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 瓶颈分析的 wire 契约：/monitoring/metrics/:processKey 下发的
// bottleneckAnalysis 四个列表必须是数组，不能是 null，也不能是与流程无关的占位样本。
// 历史实现里 slowestPaths/resourceConstraints 固定返回「标准审批路径」「审批人员 85%」，
// 空结果又被 Go 的 nil slice 序列化成 null；workflow/bottlenecks 页把这些都当成分析结论展示。

func firstLevelMap(t *testing.T, data map[string]interface{}, key string) map[string]interface{} {
	t.Helper()
	value, ok := data[key]
	require.True(t, ok, "响应缺少 %s 字段", key)
	m, ok := value.(map[string]interface{})
	require.True(t, ok, "%s 应为对象，实际 %T", key, value)
	return m
}

func assertJSONArray(t *testing.T, obj map[string]interface{}, key string, wantLen int) {
	t.Helper()
	value, ok := obj[key]
	require.True(t, ok, "响应缺少 %s 字段", key)
	require.NotNil(t, value, "%s 不得为 null：空结果必须是 []", key)
	items, ok := value.([]interface{})
	require.True(t, ok, "%s 应为数组，实际 %T", key, value)
	assert.Len(t, items, wantLen, "%s 数组长度不符", key)
}

func TestMonitoringBottleneckAnalysisWireContract(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_bottleneck_contract?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	definition := seedProcessDefinition(t, client, 1)
	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	// 至少一条实例，否则 GetProcessMetrics 在 total==0 时直接返回预置骨架，绕过真实分析分支。
	seedProcessInstance(t, client, definition, 1, "bn-1", base)

	r := newEnvelopeTestRouter(t, client)
	data := decodeEnvelope(t, r, "/api/v1/bpmn/monitoring/metrics/"+definition.Key, "1")

	analysis := firstLevelMap(t, data, "bottleneckAnalysis")
	assertJSONArray(t, analysis, "bottleneckTasks", 0)
	assertJSONArray(t, analysis, "slowestPaths", 0)
	assertJSONArray(t, analysis, "resourceConstraints", 0)
	assertJSONArray(t, analysis, "recommendations", 0)
	assert.Equal(t, "low", analysis["severity"], "无瓶颈任务时应为 low，而不是缺失")
}

func TestMonitoringBottleneckAnalysisNoPlaceholderRowsForOtherTenant(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_bottleneck_tenant?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	definition := seedProcessDefinition(t, client, 1)
	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	seedProcessInstance(t, client, definition, 1, "bn-other", base)

	r := newEnvelopeTestRouter(t, client)
	// 租户 2 没有任何流程实例，绝不能因为租户 1 的数据出现分析行。
	data := decodeEnvelope(t, r, "/api/v1/bpmn/monitoring/metrics/"+definition.Key, "2")
	require.Equal(t, float64(0), data["totalInstances"], "跨租户实例计数泄漏")

	analysis := firstLevelMap(t, data, "bottleneckAnalysis")
	assertJSONArray(t, analysis, "bottleneckTasks", 0)
	assertJSONArray(t, analysis, "slowestPaths", 0)
	assertJSONArray(t, analysis, "resourceConstraints", 0)
}
