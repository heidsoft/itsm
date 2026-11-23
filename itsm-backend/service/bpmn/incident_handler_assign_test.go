package bpmn

import (
	"context"
	"testing"
	"time"

	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 本文件锁住 assign_incident 的「不写坏数据」约束。
//
// 背景（2026-10-24 生产栈实测）：action 与 business_id 两处断点修好后，
// assign_incident 第一次真正被执行，暴露出它直接 SetAssigneeID(0) 的问题——
// incident.assignee_id 在 schema 里是 Optional 的普通 int 列，写 0 不会报错，
// 库里就会出现「assignee_id = 0」这种指向不存在用户的记录，比抛错更难发现。
// 对照 ticket_handler.assignTicket：没有处理人时直接返回「未指定处理人ID」。

func newIncidentAssignFixture(t *testing.T) (context.Context, *IncidentServiceTaskHandler, int, int, int) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:incident_assign?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })

	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("assign-test").
		SetCode("assign-test").
		Save(ctx)
	require.NoError(t, err)

	user, err := client.User.Create().
		SetUsername("assignee").
		SetEmail("assignee@example.com").
		SetName("处理人").
		SetPasswordHash("x").
		SetRole("agent").
		SetTenantID(tenant.ID).
		SetActive(true).
		Save(ctx)
	require.NoError(t, err)

	incident, err := client.Incident.Create().
		SetTitle("自动分配复测").
		SetDescription("验证没有处理人时不得写入 0").
		SetStatus("new").
		SetPriority("high").
		SetSeverity("high").
		SetIncidentNumber("INC-202610-900001").
		SetReporterID(user.ID).
		SetTenantID(tenant.ID).
		SetDetectedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	// ResolveTenantID 只认认证 context（P1-01）：fixture 必须注入租户上下文，
	// 变量里的 tenant_id 只作为同值副本做冲突校验。
	ctx = context.WithValue(ctx, BPMNTenantIDContextKey, tenant.ID)

	handler := NewIncidentServiceTaskHandler(client, zaptest.NewLogger(t).Sugar())
	return ctx, handler, tenant.ID, incident.ID, user.ID
}

// TestIncidentAssign_NoAssigneeDoesNotWriteZero 没有处理人时必须失败，且库里不被改写。
func TestIncidentAssign_NoAssigneeDoesNotWriteZero(t *testing.T) {
	ctx, handler, tenantID, incidentID, _ := newIncidentAssignFixture(t)

	result, err := handler.Execute(ctx, nil, map[string]interface{}{
		"action":      "assign_incident",
		"incident_id": incidentID,
		"tenant_id":   tenantID,
		// 故意不传 assignee_id
	})

	require.Error(t, err, "没有处理人时必须报错，而不是把 assignee_id 写成 0")
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "assignee_id", "错误里要指出缺哪个变量")

	// 库里必须保持原样
	inc, err := handler.client.Incident.Get(ctx, incidentID)
	require.NoError(t, err)
	assert.Zero(t, inc.AssigneeID, "未指定处理人时不应写入任何 assignee_id")
	assert.Equal(t, "new", inc.Status, "失败时不应改状态")
}

// TestIncidentAssign_ExplicitAssigneeIsApplied 传入处理人时正常落库。
func TestIncidentAssign_ExplicitAssigneeIsApplied(t *testing.T) {
	ctx, handler, tenantID, incidentID, userID := newIncidentAssignFixture(t)

	result, err := handler.Execute(ctx, nil, map[string]interface{}{
		"action":      "assign_incident",
		"incident_id": incidentID,
		"assignee_id": userID,
		"tenant_id":   tenantID,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	inc, err := handler.client.Incident.Get(ctx, incidentID)
	require.NoError(t, err)
	assert.Equal(t, userID, inc.AssigneeID)
	assert.Equal(t, "assigned", inc.Status)
}

// TestIncidentAssign_KeepsExistingAssignee 事件已有处理人时沿用，不清空。
func TestIncidentAssign_KeepsExistingAssignee(t *testing.T) {
	ctx, handler, tenantID, incidentID, userID := newIncidentAssignFixture(t)

	// 先指派给某个人（模拟受理时已指派）
	_, err := handler.client.Incident.UpdateOneID(incidentID).SetAssigneeID(userID).Save(ctx)
	require.NoError(t, err)

	result, err := handler.Execute(ctx, nil, map[string]interface{}{
		"action":      "assign_incident",
		"incident_id": incidentID,
		"tenant_id":   tenantID,
		// 不传 assignee_id：应沿用已有处理人
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	inc, err := handler.client.Incident.Get(ctx, incidentID)
	require.NoError(t, err)
	assert.Equal(t, userID, inc.AssigneeID, "已有处理人不应被清空")
}

// TestIncidentAssign_CrossTenantIdentityRejected P1-01 回归（codex 审计 2026-11-23）：
// 身份键与认证上下文冲突必须拒绝——(1) 变量 tenant_id 与 context 租户不同值
// （旧实现变量优先，会把查询/更新指向变量里的租户）；(2) incident_id 指向
// 异租户事件（ctx 租户内查不到即 fail closed）。两种都不得产生任何写入。
func TestIncidentAssign_CrossTenantIdentityRejected(t *testing.T) {
	ctx, handler, tenantID, incidentID, userID := newIncidentAssignFixture(t)

	t.Run("变量 tenant_id 与认证上下文冲突", func(t *testing.T) {
		result, err := handler.Execute(ctx, nil, map[string]interface{}{
			"action":      "assign_incident",
			"incident_id": incidentID,
			"assignee_id": userID,
			"tenant_id":   tenantID + 999,
		})
		require.Error(t, err, "冲突租户必须拒绝")
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "冲突")

		inc, gerr := handler.client.Incident.Get(ctx, incidentID)
		require.NoError(t, gerr)
		assert.Zero(t, inc.AssigneeID, "拒绝后不得写入")
		assert.Equal(t, "new", inc.Status)
	})

	t.Run("incident_id 指向异租户事件", func(t *testing.T) {
		foreignTenant, err := handler.client.Tenant.Create().
			SetName("异租户").SetCode("foreign-assign").Save(ctx)
		require.NoError(t, err)
		foreign, err := handler.client.Incident.Create().
			SetTitle("异租户事件").
			SetDescription("不得被本租户流程改写").
			SetStatus("new").
			SetPriority("high").
			SetSeverity("high").
			SetIncidentNumber("INC-202610-900002").
			SetReporterID(userID).
			SetTenantID(foreignTenant.ID).
			SetDetectedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)

		_, err = handler.Execute(ctx, nil, map[string]interface{}{
			"action":      "assign_incident",
			"incident_id": foreign.ID,
			"assignee_id": userID,
			"tenant_id":   tenantID, // 与 context 同值：放行租户解析，卡在业务 ID 归属
		})
		require.Error(t, err, "异租户事件必须拒绝")
		assert.Contains(t, err.Error(), "不属于当前租户")

		after, gerr := handler.client.Incident.Get(ctx, foreign.ID)
		require.NoError(t, gerr)
		assert.Zero(t, after.AssigneeID, "异租户事件不得被改写")
		assert.Equal(t, "new", after.Status)
	})
}
