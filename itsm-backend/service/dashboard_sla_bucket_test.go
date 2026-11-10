package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/ent"
)

// 回归背景：GetSLAComplianceData 曾把整体合规率直接抄给
// ResponseTimeCompliance 与 ResolutionTimeCompliance，三个指标恒等。
// 面板上并排展示三个数字、其中两个是另一个的副本——看起来有分析，实际只有
// 一个数。现在按 violation_type（response_* / resolution_*）分桶，两侧各自成立。

// createSLAFixtureTicket 造一张近 30 天内、归属指定 SLA 定义的工单。
func createSLAFixtureTicket(t *testing.T, ctx context.Context, client *ent.Client, tenantID, requesterID, slaDefID int, number string) *ent.Ticket {
	t.Helper()
	tk, err := client.Ticket.Create().
		SetTitle("SLA bucket ticket " + number).
		SetStatus("open").
		SetTicketNumber(number).
		SetRequesterID(requesterID).
		SetTenantID(tenantID).
		SetSLADefinitionID(slaDefID).
		Save(ctx)
	require.NoError(t, err)
	return tk
}

func TestDashboardSLACompliance_SeparatesResponseAndResolutionBreaches(t *testing.T) {
	client, svc, ctx := setupDashboardTest(t)
	defer client.Close()

	tenant, err := createDashboardTestTenant(ctx, client, "sla_bucket")
	require.NoError(t, err)
	user, err := createDashboardTestUser(ctx, client, tenant.ID, "sla_bucket")
	require.NoError(t, err)

	slaDef, err := client.SLADefinition.Create().
		SetName("Bucket SLA").
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 4 张工单：#1 仅响应超时，#2 仅解决超时，#3 两者都超时，#4 无违规。
	var tickets []*ent.Ticket
	for i := 1; i <= 4; i++ {
		tickets = append(tickets, createSLAFixtureTicket(t, ctx, client, tenant.ID, user.ID, slaDef.ID, fmt.Sprintf("BUCKET-%03d", i)))
	}

	mk := func(ticketID int, violationType string) {
		_, vErr := client.SLAViolation.Create().
			SetSLADefinitionID(slaDef.ID).
			SetTicketID(ticketID).
			SetViolationType(violationType).
			SetTenantID(tenant.ID).
			Save(ctx)
		require.NoError(t, vErr)
	}
	mk(tickets[0].ID, "response_time")
	mk(tickets[1].ID, "resolution_time")
	mk(tickets[2].ID, "response_time")
	mk(tickets[2].ID, "resolution_time")

	data, err := svc.GetSLAComplianceData(ctx, tenant.ID)
	require.NoError(t, err)

	assert.Equal(t, 4, data.TotalTickets)
	assert.Equal(t, 3, data.BreachedTickets, "违约工单按 ticket_id 去重")
	assert.Equal(t, 1, data.CompliantTickets)

	// 响应时限：违约 2 张（#1、#3）→ 50%
	assert.InDelta(t, 50.0, data.ResponseTimeCompliance, 0.05)
	// 解决时限：违约 2 张（#2、#3）→ 50%
	assert.InDelta(t, 50.0, data.ResolutionTimeCompliance, 0.05)
	// 整体：(4-3)/4 → 25%
	assert.InDelta(t, 25.0, data.ComplianceRate, 0.05)
}

// 只有响应类违规时，解决时限达成率必须保持 100%，不再被整体合规率带偏。
func TestDashboardSLACompliance_ResponseOnlyViolationKeepsResolutionFull(t *testing.T) {
	client, svc, ctx := setupDashboardTest(t)
	defer client.Close()

	tenant, err := createDashboardTestTenant(ctx, client, "sla_resp_only")
	require.NoError(t, err)
	user, err := createDashboardTestUser(ctx, client, tenant.ID, "sla_resp_only")
	require.NoError(t, err)

	slaDef, err := client.SLADefinition.Create().
		SetName("Response Only SLA").
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	first := createSLAFixtureTicket(t, ctx, client, tenant.ID, user.ID, slaDef.ID, "RESP-001")
	createSLAFixtureTicket(t, ctx, client, tenant.ID, user.ID, slaDef.ID, "RESP-002")

	_, err = client.SLAViolation.Create().
		SetSLADefinitionID(slaDef.ID).
		SetTicketID(first.ID).
		SetViolationType("response_time").
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	data, err := svc.GetSLAComplianceData(ctx, tenant.ID)
	require.NoError(t, err)

	assert.InDelta(t, 50.0, data.ResponseTimeCompliance, 0.05, "2 张里 1 张响应超时")
	assert.InDelta(t, 100.0, data.ResolutionTimeCompliance, 0.05, "没有任何解决时限违规")
	assert.InDelta(t, 50.0, data.ComplianceRate, 0.05)
}
