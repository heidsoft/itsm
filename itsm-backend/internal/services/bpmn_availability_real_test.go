package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// seedProcessInstanceWithStatus 造一条指定状态的流程实例，用于验证可用性由真实结果推导。
func seedProcessInstanceWithStatus(t *testing.T, client *ent.Client, tenantID int, key, status string) {
	t.Helper()
	ctx := context.Background()
	dep, err := client.ProcessDeployment.Create().
		SetDeploymentID(fmt.Sprintf("DEP-AVAIL-%d-%s", tenantID, key)).
		SetDeploymentName("availability seed").SetDeploymentTime(time.Now()).
		SetTenantID(tenantID).SetDeployedBy("test").Save(ctx)
	require.NoError(t, err)
	def, err := client.ProcessDefinition.Create().
		SetKey(key).SetName("availability seed").SetVersion("1").
		SetBpmnXML([]byte("<bpmn/>")).SetIsActive(true).SetIsLatest(true).
		SetTenantID(tenantID).SetDeploymentID(dep.ID).SetDeployedAt(time.Now()).Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessInstance.Create().
		SetProcessInstanceID(key).SetProcessDefinitionKey(def.Key).
		SetProcessDefinitionID(def.ID).SetStatus(status).
		SetTenantID(tenantID).SetStartTime(time.Now()).Save(ctx)
	require.NoError(t, err)
}

// TestBPMNMonitor_AvailabilityDerivedFromRealOutcomes 锁住「可用性」不是写死的
// 99.9 假指标：旧实现 `metrics.Availability = 99.9 // 默认值，实际应该基于监控数据
// 计算`，无论实例全部失败还是一半失败都恒报 99.9%，性能报告据此得出「服务高度可用」
// 的错误结论。可用性必须由真实失败率推导（availability = 100 - errorRate）。
func TestBPMNMonitor_AvailabilityDerivedFromRealOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name      string
		statuses  []string
		wantError float64
		wantAvail float64
	}{
		{"all_completed", []string{"completed", "completed"}, 0, 100},
		{"half_failed", []string{"completed", "failed"}, 50, 50},
		{"all_failed", []string{"failed", "failed"}, 100, 0},
		// 生命周期实际写 terminated，必须与 failed 使用相同失败口径。
		{"all_terminated", []string{"terminated", "terminated"}, 100, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dsn := fmt.Sprintf("file:bpmn_avail_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
			client := enttest.Open(t, "sqlite3", dsn)
			t.Cleanup(func() { _ = client.Close() })
			for i, status := range tc.statuses {
				seedProcessInstanceWithStatus(t, client, 1, fmt.Sprintf("PI-AVAIL-%s-%d", tc.name, i), status)
			}
			svc := NewBPMNMonitoringService(client, nil, zaptest.NewLogger(t).Sugar())
			metrics, err := svc.calculatePerformanceMetrics(context.Background(), client.ProcessInstance.Query())
			require.NoError(t, err)
			response, err := svc.GetProcessMetrics(context.Background(), &dto.ProcessMetricsRequest{TenantID: 1})
			require.NoError(t, err)
			assert.InDelta(t, tc.wantError, float64(response.FailedInstances)/float64(response.TotalInstances)*100, 0.001)
			assert.Equal(t, metrics.ErrorRate, response.PerformanceMetrics.ErrorRate)
			assert.InDelta(t, tc.wantError, metrics.ErrorRate, 0.001)
			assert.InDelta(t, tc.wantAvail, metrics.Availability, 0.001,
				"可用性必须由真实失败率推导，不得恒为写死的 99.9")
		})
	}
}

// TestBPMNMonitor_ProcessMetricsNoFakeAvailability 锁住 GetProcessMetrics 骨架：
// 无实例时旧实现预置 Availability=99.9，把「没有任何流程数据」伪装成「99.9% 可用」。
// 无数据时可用性应为 0（无测量值），不得凭空给出一个看似健康的百分数。
func TestBPMNMonitor_ProcessMetricsNoFakeAvailability(t *testing.T) {
	dsn := fmt.Sprintf("file:bpmn_avail_empty_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { _ = client.Close() })
	svc := NewBPMNMonitoringService(client, nil, zaptest.NewLogger(t).Sugar())
	metrics, err := svc.GetProcessMetrics(context.Background(), &dto.ProcessMetricsRequest{TenantID: 1})
	require.NoError(t, err)
	require.NotNil(t, metrics)
	assert.Zero(t, metrics.PerformanceMetrics.Availability,
		"无流程实例时可用性必须为 0（未测量），不得预置 99.9 假健康值")
}
