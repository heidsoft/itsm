package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	baseent "entgo.io/ent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// countFault 注入「第 nth 次 Count 查询失败」，不影响 Exist/All/FirstID。
// 这样 GetSystemHealth 的连通性 Exist 先成功，三个 Count 才能被单独打挂，
// 精确复现「DB 计数故障仍报 healthy」的假健康缺陷。
func countFault(nth int) baseent.Interceptor {
	seen := 0
	return baseent.InterceptFunc(func(next baseent.Querier) baseent.Querier {
		return baseent.QuerierFunc(func(ctx context.Context, q baseent.Query) (baseent.Value, error) {
			qc := baseent.QueryFromContext(ctx)
			if qc != nil && qc.Op == baseent.OpQueryCount {
				seen++
				if seen == nth {
					return 0, fmt.Errorf("injected count failure #%d", nth)
				}
			}
			return next.Query(ctx, q)
		})
	})
}

// allFault 注入「第 nth 次 All 查询失败」，用于 GetPerformanceAlerts 的 longRunningTasks。
func allFault(nth int) baseent.Interceptor {
	seen := 0
	return baseent.InterceptFunc(func(next baseent.Querier) baseent.Querier {
		return baseent.QuerierFunc(func(ctx context.Context, q baseent.Query) (baseent.Value, error) {
			qc := baseent.QueryFromContext(ctx)
			if qc != nil && qc.Op == baseent.OpQueryAll {
				seen++
				if seen == nth {
					return nil, errors.New("injected All failure")
				}
			}
			return next.Query(ctx, q)
		})
	})
}

func newBPMNMonitorFaultClient(t *testing.T, inters ...baseent.Interceptor) *ent.Client {
	t.Helper()
	dsn := fmt.Sprintf("file:bpmn_mon_fault_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { _ = client.Close() })
	client.Intercept(inters...)
	return client
}

func newBPMNMonitorFaultService(t *testing.T, inters ...baseent.Interceptor) *BPMNMonitoringService {
	t.Helper()
	return NewBPMNMonitoringService(newBPMNMonitorFaultClient(t, inters...), nil, zaptest.NewLogger(t).Sugar())
}

// TestBPMNMonitor_HealthFailClosedOnAnyCountError 是「撤掉守卫即失败」的回归锁：
// 旧实现三处 Count 全部 `if err == nil` 吞错，DB 计数故障时仍返回
// status=healthy/database=connected， readiness/liveness 探针据此认为系统正常，
// 掩盖真实的查询故障。逐一对三个 Count 注入故障，断言整体判 unhealthy。
func TestBPMNMonitor_HealthFailClosedOnAnyCountError(t *testing.T) {
	for nth := 1; nth <= 3; nth++ {
		t.Run(fmt.Sprintf("count_%d", nth), func(t *testing.T) {
			svc := newBPMNMonitorFaultService(t, countFault(nth))
			health, err := svc.GetSystemHealth(context.Background(), 1)
			require.NoError(t, err, "健康接口保持 200 + unhealthy 载荷契约")
			require.NotNil(t, health)
			assert.Equal(t, "unhealthy", health["status"], "Count 故障必须判 unhealthy")
			assert.Equal(t, "disconnected", health["database"], "Count 故障必须标 database=disconnected")
		})
	}
}

// TestBPMNMonitor_HealthSuccessPathKeepsCounts 锁住正常路径：三个 Count 成功时
// 仍返回真实计数与 healthy，防止 fail-closed 实现误伤成功分支。
func TestBPMNMonitor_HealthSuccessPathKeepsCounts(t *testing.T) {
	svc := newBPMNMonitorFaultService(t)
	health, err := svc.GetSystemHealth(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "healthy", health["status"])
	assert.Equal(t, "connected", health["database"])
	assert.EqualValues(t, 0, health["activeProcesses"])
}

// TestBPMNMonitor_PerformanceMetricsErrorRateFailClosed 直接调用
// calculatePerformanceMetrics：ErrorRate 的 Count 失败必须返回错误。
// 旧实现 `if err == nil && total > 0` 吞错，把故障算成 0% 错误率。
func TestBPMNMonitor_PerformanceMetricsErrorRateFailClosed(t *testing.T) {
	client := newBPMNMonitorFaultClient(t, countFault(1)) // 此函数只有一次 Count：错误率查询
	svc := NewBPMNMonitoringService(client, nil, zaptest.NewLogger(t).Sugar())
	_, err := svc.calculatePerformanceMetrics(context.Background(), client.ProcessInstance.Query())
	require.Error(t, err, "错误率计数失败必须向上返回，不得默默算成 0%")
}

// TestBPMNMonitor_StatusCountsFailClosed 是审计点名的「5 处统计吞错」回归锁：
// GetProcessMetrics 的 5 个状态计数旧实现全部 `count, _ :=`，DB 故障时返回
// 全 0 的各状态实例数与 SuccessRate，监控页据此显示「零运行/零失败」的假象。
// 逐一注入第 2~6 次 Count（第 1 次是 total，成功）失败，断言整体返回 error。
func TestBPMNMonitor_StatusCountsFailClosed(t *testing.T) {
	statuses := []string{"running", "completed", "failed", "terminated", "suspended"}
	for i, status := range statuses {
		t.Run(status, func(t *testing.T) {
			client := newBPMNMonitorFaultClient(t, countFault(i+2)) // total=1, 状态计数=2..6
			seedOneProcessInstance(t, client, 1, "PI-STATUS-"+status)
			svc := NewBPMNMonitoringService(client, nil, zaptest.NewLogger(t).Sugar())
			_, err := svc.GetProcessMetrics(context.Background(), &dto.ProcessMetricsRequest{TenantID: 1})
			require.Error(t, err, "%s 计数失败必须向上返回", status)
		})
	}
}

// seedOneProcessInstance 造一条 process_instances，让 GetProcessMetrics 越过
// `total == 0` 的快速返回，真正执行 5 个状态 Count。
func seedOneProcessInstance(t *testing.T, client *ent.Client, tenantID int, key string) {
	t.Helper()
	ctx := context.Background()
	dep, err := client.ProcessDeployment.Create().
		SetDeploymentID(fmt.Sprintf("DEP-MON-%d-%s", tenantID, key)).
		SetDeploymentName("monitor fault seed").
		SetDeploymentTime(time.Now()).
		SetTenantID(tenantID).
		SetDeployedBy("test").
		Save(ctx)
	require.NoError(t, err)
	def, err := client.ProcessDefinition.Create().
		SetKey(key).SetName("monitor seed").SetVersion("1").
		SetBpmnXML([]byte("<bpmn/>")).SetIsActive(true).SetIsLatest(true).
		SetTenantID(tenantID).SetDeploymentID(dep.ID).SetDeployedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessInstance.Create().
		SetProcessInstanceID(key).SetProcessDefinitionKey(def.Key).
		SetProcessDefinitionID(def.ID).SetStatus("running").
		SetTenantID(tenantID).SetStartTime(time.Now()).
		Save(ctx)
	require.NoError(t, err)
}

// TestBPMNMonitor_GetProcessMetricsErrorRatePropagates 锁住端到端传播：
// 即使 calculatePerformanceMetrics 内部返回错误，GetProcessMetrics 也不得用
// `if err == nil` 吞掉后返回带 0% 错误率骨架的假成功响应。
func TestBPMNMonitor_GetProcessMetricsErrorRatePropagates(t *testing.T) {
	client := newBPMNMonitorFaultClient(t, countFault(7)) // total(1) + 5状态(2..6) + perf的ErrorRate(7)
	seedOneProcessInstance(t, client, 1, "PI-PERF-PROP")
	svc := NewBPMNMonitoringService(client, nil, zaptest.NewLogger(t).Sugar())
	metrics, err := svc.GetProcessMetrics(context.Background(), &dto.ProcessMetricsRequest{TenantID: 1})
	require.Error(t, err, "性能指标计算失败必须让整个 metrics 接口报错")
	assert.Nil(t, metrics, "不得返回带预置骨架的假成功响应")
}

// TestBPMNMonitor_AlertsFailClosed 覆盖 GetPerformanceAlerts 两处吞错：
// longRunningTasks 的 All 与 terminatedInstances 的 Count 故障都曾返回空告警数组
// （假「一切正常」）。现在任一失败都必须返回 error 交给 handler 映射 500。
func TestBPMNMonitor_AlertsFailClosed(t *testing.T) {
	t.Run("long_running_all_failure", func(t *testing.T) {
		svc := newBPMNMonitorFaultService(t, allFault(1))
		alerts, err := svc.GetPerformanceAlerts(context.Background(), 1)
		require.Error(t, err)
		assert.Nil(t, alerts, "故障时必须返回 nil，不得返回空告警伪装正常")
	})
	t.Run("terminated_count_failure", func(t *testing.T) {
		svc := newBPMNMonitorFaultService(t, countFault(1))
		alerts, err := svc.GetPerformanceAlerts(context.Background(), 1)
		require.Error(t, err)
		assert.Nil(t, alerts)
	})
}
