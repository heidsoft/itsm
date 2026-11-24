package service

import (
	"context"
	"testing"
	"time"

	"itsm-backend/ent"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// createStuckInstanceFixture 建一条真实的最小实例记录（deployment -> definition -> instance）。
// startTime 显式传入，便于把「老实例」与「新实例」区分开。
func createStuckInstanceFixture(t *testing.T, client *ent.Client, suffix string, tenantID int, status string, startTime time.Time) *ent.ProcessInstance {
	t.Helper()
	ctx := context.Background()

	dep, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-STUCK-" + suffix).
		SetDeploymentName("Stuck Deployment " + suffix).
		SetDeploymentTime(startTime).
		SetDeployedBy("test").
		SetIsActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	def, err := client.ProcessDefinition.Create().
		SetKey("stuck_test_" + suffix).
		SetName("Stuck Test " + suffix).
		SetVersion("1").
		SetIsLatest(true).
		SetBpmnXML([]byte(`<bpmn:definitions/>`)).
		SetDeploymentID(dep.ID).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	inst, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-STUCK-" + suffix).
		SetProcessDefinitionKey("stuck_test_" + suffix).
		SetProcessDefinitionID(def.ID).
		SetStatus(status).
		SetStartTime(startTime).
		SetCurrentActivityID("Activity_" + suffix).
		SetCurrentActivityName("卡住的活动 " + suffix).
		SetTenantID(tenantID).
		SetVariables(map[string]interface{}{"stage": suffix}).
		Save(ctx)
	require.NoError(t, err)
	return inst
}

func newStuckScanner(t *testing.T, name string) (*ent.Client, *TimeoutScanner) {
	t.Helper()
	client := newTimeoutScannerClient(t, name)
	return client, NewTimeoutScanner(client, zaptest.NewLogger(t).Sugar())
}

// TestScanStuckInstances_DetectsOnlyOldRunningInstances 验证三个维度同时生效：
// 状态 running、启动时间早于阈值、租户隔离。任一维度缺失都会多算。
func TestScanStuckInstances_DetectsOnlyOldRunningInstances(t *testing.T) {
	client, scanner := newStuckScanner(t, "stuck-detect")
	ctx := context.Background()
	tenantA, _ := setupTimeoutScannerTenant(t, client, "stuck-a")
	tenantB, _ := setupTimeoutScannerTenant(t, client, "stuck-b")
	threshold := 2 * time.Hour

	// 唯一应被检出的：本租户 + running + 早于阈值
	hit := createStuckInstanceFixture(t, client, "old-running", tenantA, "running", time.Now().Add(-3*time.Hour))
	// 新实例不应检出
	createStuckInstanceFixture(t, client, "fresh-running", tenantA, "running", time.Now().Add(-10*time.Minute))
	// 已结束实例不应检出
	createStuckInstanceFixture(t, client, "old-completed", tenantA, "completed", time.Now().Add(-3*time.Hour))
	// 挂起实例不应检出
	createStuckInstanceFixture(t, client, "old-suspended", tenantA, "suspended", time.Now().Add(-3*time.Hour))
	// 其他租户的老 running 不应检出
	createStuckInstanceFixture(t, client, "other-tenant", tenantB, "running", time.Now().Add(-3*time.Hour))

	count, err := scanner.ScanStuckInstances(ctx, tenantA, threshold)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "只应检出本租户早于阈值且仍 running 的那一个实例")

	// 同时确认「只观测不处置」：状态、结束时间、变量、当前活动都不被改写。
	got, err := client.ProcessInstance.Get(ctx, hit.ID)
	require.NoError(t, err)
	assert.Equal(t, "running", got.Status, "扫描不得自动终止或改状态")
	assert.True(t, got.EndTime.IsZero(), "扫描不得写 end_time")
	assert.Equal(t, "Activity_old-running", got.CurrentActivityID, "扫描不得推进当前活动")
	assert.Equal(t, "old-running", got.Variables["stage"], "扫描不得改写业务变量")
}

// TestScanStuckInstances_NoStuckInstances 无卡死实例时返回 0 且不报错。
func TestScanStuckInstances_NoStuckInstances(t *testing.T) {
	client, scanner := newStuckScanner(t, "stuck-empty")
	ctx := context.Background()
	tenantID, _ := setupTimeoutScannerTenant(t, client, "stuck-empty")
	createStuckInstanceFixture(t, client, "only-fresh", tenantID, "running", time.Now())

	count, err := scanner.ScanStuckInstances(ctx, tenantID, time.Hour)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

// TestScanStuckInstances_InvalidParams 无效租户与无效阈值必须显式报错，
// 而不是退化成「全租户扫描」或「所有实例都算卡死」。
func TestScanStuckInstances_InvalidParams(t *testing.T) {
	client, scanner := newStuckScanner(t, "stuck-invalid")
	ctx := context.Background()
	tenantID, _ := setupTimeoutScannerTenant(t, client, "stuck-invalid")
	createStuckInstanceFixture(t, client, "old", tenantID, "running", time.Now().Add(-3*time.Hour))

	_, err := scanner.ScanStuckInstances(ctx, 0, time.Hour)
	require.Error(t, err, "tenantID <= 0 必须报错")
	_, err = scanner.ScanStuckInstances(ctx, tenantID, 0)
	require.Error(t, err, "olderThan <= 0 必须报错")
	_, err = scanner.ScanStuckInstances(ctx, tenantID, -time.Hour)
	require.Error(t, err, "负阈值必须报错")
}

// TestScanStuckInstances_ThresholdBoundary 阈值边界：恰好早于阈值算卡死，
// 恰好晚于阈值不算。防止用 <= 与 < 混淆导致误报。
func TestScanStuckInstances_ThresholdBoundary(t *testing.T) {
	client, scanner := newStuckScanner(t, "stuck-boundary")
	ctx := context.Background()
	tenantID, _ := setupTimeoutScannerTenant(t, client, "stuck-boundary")
	threshold := time.Hour

	createStuckInstanceFixture(t, client, "older", tenantID, "running", time.Now().Add(-threshold-time.Minute))
	createStuckInstanceFixture(t, client, "newer", tenantID, "running", time.Now().Add(-threshold+time.Minute))

	count, err := scanner.ScanStuckInstances(ctx, tenantID, threshold)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "只有早于阈值一侧应被检出")
}
