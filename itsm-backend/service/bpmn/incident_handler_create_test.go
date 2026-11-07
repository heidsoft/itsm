package bpmn

import (
	"context"
	"fmt"
	"testing"
	"time"

	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 本文件锁住 create_incident 的发号约束。
//
// 背景（2026-11-07 全模块缺陷排查 P0）：createIncident 从不设置
// incident_number，而 schema 要求 NotEmpty + Unique，Save必 ValidationError，
// BPMN 创建事件的生产链路（CallbackRegistry → 本 handler）必然失败进死信。
// 修复后编号与 HTTP 创建共用 incident_number_seq（database.GenerateIncidentNumber）。

func newIncidentCreateFixture(t *testing.T) (context.Context, *IncidentServiceTaskHandler, int, int) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:incident_create?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })

	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("create-test").
		SetCode("create-test").
		Save(ctx)
	require.NoError(t, err)

	user, err := client.User.Create().
		SetUsername("reporter").
		SetEmail("reporter@example.com").
		SetName("报告人").
		SetPasswordHash("x").
		SetRole("agent").
		SetTenantID(tenant.ID).
		SetActive(true).
		Save(ctx)
	require.NoError(t, err)

	handler := NewIncidentServiceTaskHandler(client, zaptest.NewLogger(t).Sugar())
	return ctx, handler, tenant.ID, user.ID
}

// TestIncidentCreate_GeneratesNumber 创建成功必须落租户内唯一编号，
// 且编号来自与 HTTP 主链共享的发号器（本地年月口径）。
func TestIncidentCreate_GeneratesNumber(t *testing.T) {
	ctx, handler, tenantID, userID := newIncidentCreateFixture(t)

	var gotYear, gotMonth int
	handler.generateNumber = func(_ context.Context, year, month int) (string, error) {
		gotYear, gotMonth = year, month
		return "INC-202611-000001", nil
	}

	result, err := handler.Execute(ctx, nil, map[string]interface{}{
		"action":      "create_incident",
		"title":       "流程创建事件",
		"description": "BPMN 服务任务",
		"type":        "incident",
		"priority":    "high",
		"severity":    "high",
		"tenant_id":   tenantID,
		"reporter_id": userID,
	})

	require.NoError(t, err, "缺少 incident_number 时这里必 ValidationError（修复前红）")
	require.NotNil(t, result)
	assert.True(t, result.Success)

	year, month, _ := time.Now().Date()
	assert.Equal(t, year, gotYear, "发号年月必须用本地时区，与 HTTP 主链一致")
	assert.Equal(t, int(month), gotMonth)

	incidents, err := handler.client.Incident.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, incidents, 1)
	inc := incidents[0]
	assert.Equal(t, "INC-202611-000001", inc.IncidentNumber)
	assert.Equal(t, tenantID, inc.TenantID)
	assert.Equal(t, userID, inc.ReporterID)
	assert.Equal(t, "new", inc.Status)
	assert.Equal(t, "high", inc.Priority, "显式传入的枚举值必须透传")
	assert.Equal(t, "high", inc.Severity)
	assert.Equal(t, "INC-202611-000001", result.OutputVars["incident_number"],
		"OutputVars 必须回真实编号供下游节点引用")
}

// TestIncidentCreate_EachCallGetsFreshNumber 每次创建都必须重新发号，不得复用常量。
func TestIncidentCreate_EachCallGetsFreshNumber(t *testing.T) {
	ctx, handler, tenantID, userID := newIncidentCreateFixture(t)

	seq := 0
	handler.generateNumber = func(_ context.Context, _, _ int) (string, error) {
		seq++
		return fmt.Sprintf("INC-202611-%06d", seq), nil
	}

	vars := map[string]interface{}{
		"action":      "create_incident",
		"title":       "连续创建",
		"tenant_id":   tenantID,
		"reporter_id": userID,
	}
	_, err := handler.Execute(ctx, nil, vars)
	require.NoError(t, err)
	_, err = handler.Execute(ctx, nil, vars)
	require.NoError(t, err)

	incidents, err := handler.client.Incident.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, incidents, 2)
	assert.NotEqual(t, incidents[0].IncidentNumber, incidents[1].IncidentNumber)
	assert.Equal(t, 2, seq)
	for _, inc := range incidents {
		assert.Equal(t, "medium", inc.Priority, "未传 priority 必须用 schema 默认值而非空串")
		assert.Equal(t, "medium", inc.Severity)
		assert.Equal(t, "incident", inc.Type)
	}
}

// TestIncidentCreate_NumberFailureCreatesNothing 发号失败必须整体失败，不留半行数据。
func TestIncidentCreate_NumberFailureCreatesNothing(t *testing.T) {
	ctx, handler, tenantID, userID := newIncidentCreateFixture(t)

	handler.generateNumber = func(_ context.Context, _, _ int) (string, error) {
		return "", fmt.Errorf("sequence unavailable")
	}

	result, err := handler.Execute(ctx, nil, map[string]interface{}{
		"action":      "create_incident",
		"title":       "发号失败",
		"tenant_id":   tenantID,
		"reporter_id": userID,
	})

	require.Error(t, err)
	assert.Nil(t, result)
	count, err := handler.client.Incident.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, count, "发号失败不得落库")
}

// TestIncidentCreate_DefaultGeneratorFailsClosed 未注入发号器且进程无数据库时
// 必须 fail closed，而不是写空编号或 panic。
func TestIncidentCreate_DefaultGeneratorFailsClosed(t *testing.T) {
	ctx, handler, tenantID, userID := newIncidentCreateFixture(t)

	result, err := handler.Execute(ctx, nil, map[string]interface{}{
		"action":      "create_incident",
		"title":       "无发号器",
		"tenant_id":   tenantID,
		"reporter_id": userID,
	})

	require.Error(t, err)
	assert.Nil(t, result)
	count, err := handler.client.Incident.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, count)
}