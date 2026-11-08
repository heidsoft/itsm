package bpmn

import (
	"context"
	"fmt"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type ticketStatusGateEnv struct {
	client   *ent.Client
	handler  *TicketServiceTaskHandler
	ctx      context.Context
	tenantID int
	userID   int
}

func setupTicketStatusGateTest(t *testing.T, name string) *ticketStatusGateEnv {
	t.Helper()
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:ticket_status_gate_%s_%d?mode=memory&cache=shared&_fk=1", name, time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	logger := zaptest.NewLogger(t).Sugar()
	handler := NewTicketServiceTaskHandler(client, logger)

	ctx := context.Background()
	tenant, err := client.Tenant.Create().
		SetName("Gate Tenant " + name).SetCode("gate" + name).SetDomain("gate" + name + ".test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	user, err := client.User.Create().
		SetUsername("gateuser" + name).SetEmail("gate" + name + "@test.com").SetName("Gate User").
		SetPasswordHash("hash").SetRole("end_user").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	workflowCtx := context.WithValue(ctx, BPMNTenantIDContextKey, tenant.ID)
	return &ticketStatusGateEnv{client: client, handler: handler, ctx: workflowCtx, tenantID: tenant.ID, userID: user.ID}
}

func createGateTicket(t *testing.T, env *ticketStatusGateEnv, status, resolution string) *ent.Ticket {
	t.Helper()
	create := env.client.Ticket.Create().
		SetTitle("闸门用例工单 " + status).
		SetTicketNumber(fmt.Sprintf("TKT-GATE-%s-%d", status, time.Now().UnixNano())).
		SetStatus(status).
		SetPriority("medium").
		SetType("incident").
		SetRequesterID(env.userID).
		SetTenantID(env.tenantID)
	if resolution != "" {
		create = create.SetResolution(resolution)
	}
	tk, err := create.Save(env.ctx)
	require.NoError(t, err)
	return tk
}

// TestTicketUpdateStatusStateMachineGate BPMN 服务任务改状态必须走权威状态机：
// 合法迁移 CAS 推进版本；非法迁移（终态复活/缺解决方案）拒绝且版本不变。
func TestTicketUpdateStatusStateMachineGate(t *testing.T) {
	env := setupTicketStatusGateTest(t, "matrix")

	cases := []struct {
		name        string
		from        string
		to          string
		resolution  string
		wantVersion int // 期望迁移后的 version；0 表示迁移必须被拒绝
	}{
		{"open→in_progress 合法", "open", "in_progress", "", 2},
		{"closed→in_progress 拒绝", "closed", "in_progress", "", 0},
		{"cancelled→open 拒绝", "cancelled", "open", "", 0},
		{"open→resolved 缺解决方案 拒绝", "open", "resolved", "", 0},
		{"resolved 有解决方案 →closed 合法", "resolved", "closed", "已解决", 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tk := createGateTicket(t, env, tc.from, tc.resolution)

			result, err := env.handler.Execute(env.ctx, nil, map[string]interface{}{
				"business_id": tk.ID,
				"action":      "update_status",
				"new_status":  tc.to,
			})

			if tc.wantVersion == 0 {
				require.Error(t, err, "非法迁移必须被拒绝")
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.True(t, result.Success)
			}

			after, err := env.client.Ticket.Get(env.ctx, tk.ID)
			require.NoError(t, err)
			if tc.wantVersion == 0 {
				assert.Equal(t, tc.from, after.Status, "拒绝后状态必须保持原样")
				assert.Equal(t, 1, after.Version, "拒绝后版本不得推进")
			} else {
				assert.Equal(t, tc.to, after.Status)
				assert.Equal(t, tc.wantVersion, after.Version, "合法迁移必须推进版本")
			}
		})
	}
}

// TestTicketUpdateStatusClosedKeepsResolvedAt resolved→closed 只写 closed_at，
// 不得改写已发生的 resolved_at（生命周期耗时口径锚定数据库权威时间戳）。
func TestTicketUpdateStatusClosedKeepsResolvedAt(t *testing.T) {
	env := setupTicketStatusGateTest(t, "closedts")

	resolvedAt := time.Now().Add(-time.Hour)
	tk, err := env.client.Ticket.Create().
		SetTitle("closed 时间戳口径工单").
		SetTicketNumber(fmt.Sprintf("TKT-GATE-TS-%d", time.Now().UnixNano())).
		SetStatus("resolved").
		SetPriority("medium").
		SetType("incident").
		SetRequesterID(env.userID).
		SetResolution("已解决").
		SetResolvedAt(resolvedAt).
		SetTenantID(env.tenantID).
		Save(env.ctx)
	require.NoError(t, err)

	_, err = env.handler.Execute(env.ctx, nil, map[string]interface{}{
		"business_id": tk.ID,
		"action":      "update_status",
		"new_status":  "closed",
	})
	require.NoError(t, err)

	after, err := env.client.Ticket.Get(env.ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, "closed", after.Status)
	assert.False(t, after.ClosedAt.IsZero(), "closed 必须写 closed_at")
	assert.True(t, after.ResolvedAt.Equal(resolvedAt), "resolved_at 不得被 closed 覆盖")
	assert.Equal(t, 2, after.Version)
}

// TestTicketUpdateStatusCrossTenantFailClosed 异租户上下文必须 fail closed：
// 工单状态不得被无权租户的调用改写。
func TestTicketUpdateStatusCrossTenantFailClosed(t *testing.T) {
	env := setupTicketStatusGateTest(t, "cross")
	tk := createGateTicket(t, env, "open", "")

	foreignCtx := context.WithValue(context.Background(), BPMNTenantIDContextKey, env.tenantID+999)
	_, err := env.handler.Execute(foreignCtx, nil, map[string]interface{}{
		"business_id": tk.ID,
		"action":      "update_status",
		"new_status":  "in_progress",
	})
	require.Error(t, err, "异租户上下文必须被拒绝")

	after, err := env.client.Ticket.Get(env.ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, "open", after.Status, "异租户调用不得改写工单状态")
}
