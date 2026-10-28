package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 报表 SLA 合规率口径回归锁。
//
// 缺陷：calculateSummary 把「解决率」(resolved/total) 当成「SLA 合规率」导出，
// 二者根本不是一回事——工单可以已解决但超时（违规），也可以未解决但仍在 SLA 内。
// 用解决率冒充合规率会让管理层月报与 CSV 导出系统性失真。
// 合规率的权威事实是 SLAViolation 记录（由 SLA 监控域写入）：
// 有违规记录的工单即不合规，合规率 = (总数 - 有违规的工单数) / 总数。

func seedAnalyticsTicket(t *testing.T, ctx context.Context, client *ent.Client, tenantID, userID int, num, status string) *ent.Ticket {
	t.Helper()
	createdAt := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	builder := client.Ticket.Create().
		SetTitle("Ticket " + num).
		SetDescription("desc").
		SetPriority("medium").
		SetType("incident").
		SetStatus(status).
		SetTicketNumber(num).
		SetTenantID(tenantID).
		SetRequesterID(userID).
		SetCreatedAt(createdAt)
	if status == "resolved" || status == "closed" {
		resolvedAt := createdAt.Add(2 * time.Hour)
		builder = builder.SetResolvedAt(resolvedAt)
	}
	tk, err := builder.Save(ctx)
	require.NoError(t, err)
	return tk
}

func seedSLAViolation(t *testing.T, ctx context.Context, client *ent.Client, tenantID, ticketID int) {
	t.Helper()
	def, err := client.SLADefinition.Create().
		SetName("SLA").
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.SLAViolation.Create().
		SetTenantID(tenantID).
		SetTicketID(ticketID).
		SetSLADefinitionID(def.ID).
		SetViolationType("resolution").
		Save(ctx)
	require.NoError(t, err)
}

func analyticsSLARequest() *dto.DeepAnalyticsRequest {
	return &dto.DeepAnalyticsRequest{
		TimeRange:  []string{"2024-01-01", "2024-01-31"},
		Dimensions: []string{"status"},
		Metrics:    []string{"count"},
	}
}

// TestAnalyticsSLACompliance_ResolvedButBreached 全部已解决（解决率 100%），
// 但其中一单有 SLA 违规：合规率必须是 75%，而不是把解决率 100% 当合规率。
func TestAnalyticsSLACompliance_ResolvedButBreached(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:analytics_sla_breached?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	svc := NewAnalyticsService(client, zaptest.NewLogger(t).Sugar())

	tenant, err := createAnalyticsTestTenant(ctx, client, "slabreach")
	require.NoError(t, err)
	user, err := createAnalyticsTestUser(ctx, client, tenant.ID, "slabreach")
	require.NoError(t, err)

	var breached *ent.Ticket
	for i := 0; i < 4; i++ {
		tk := seedAnalyticsTicket(t, ctx, client, tenant.ID, user.ID, fmt.Sprintf("TKT-SLAB-%03d", i), "resolved")
		if i == 0 {
			breached = tk
		}
	}
	seedSLAViolation(t, ctx, client, tenant.ID, breached.ID)

	resp, err := svc.GetDeepAnalytics(ctx, analyticsSLARequest(), tenant.ID)
	require.NoError(t, err)
	require.Equal(t, 4, resp.Summary.Total)
	require.Equal(t, 4, resp.Summary.Resolved)
	require.InDelta(t, 75.0, resp.Summary.SLACompliance, 0.001,
		"合规率应按违规事实算（3/4），不是解决率（4/4）")
}

// TestAnalyticsSLACompliance_OpenWithoutViolation 全部未解决（解决率 0%）但无违规：
// 合规率必须是 100%，证明它已与解决率解耦。
func TestAnalyticsSLACompliance_OpenWithoutViolation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:analytics_sla_open?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	svc := NewAnalyticsService(client, zaptest.NewLogger(t).Sugar())

	tenant, err := createAnalyticsTestTenant(ctx, client, "slaopen")
	require.NoError(t, err)
	user, err := createAnalyticsTestUser(ctx, client, tenant.ID, "slaopen")
	require.NoError(t, err)

	for i := 0; i < 4; i++ {
		seedAnalyticsTicket(t, ctx, client, tenant.ID, user.ID, fmt.Sprintf("TKT-SLAO-%03d", i), "open")
	}

	resp, err := svc.GetDeepAnalytics(ctx, analyticsSLARequest(), tenant.ID)
	require.NoError(t, err)
	require.Equal(t, 4, resp.Summary.Total)
	require.Equal(t, 0, resp.Summary.Resolved)
	require.InDelta(t, 100.0, resp.Summary.SLACompliance, 0.001,
		"无违规即合规，未解决不等于违反 SLA")
}

// TestAnalyticsSLACompliance_TenantScoped 其他租户的违规记录不得影响本租户合规率。
func TestAnalyticsSLACompliance_TenantScoped(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:analytics_sla_tenant?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	svc := NewAnalyticsService(client, zaptest.NewLogger(t).Sugar())

	tenantA, err := createAnalyticsTestTenant(ctx, client, "slaTa")
	require.NoError(t, err)
	tenantB, err := createAnalyticsTestTenant(ctx, client, "slaTb")
	require.NoError(t, err)
	userA, err := createAnalyticsTestUser(ctx, client, tenantA.ID, "slaTa")
	require.NoError(t, err)
	userB, err := createAnalyticsTestUser(ctx, client, tenantB.ID, "slaTb")
	require.NoError(t, err)

	tkA := seedAnalyticsTicket(t, ctx, client, tenantA.ID, userA.ID, "TKT-SLATA-001", "resolved")
	tkB := seedAnalyticsTicket(t, ctx, client, tenantB.ID, userB.ID, "TKT-SLATB-001", "resolved")
	// 只有 B 租户的工单违规。
	seedSLAViolation(t, ctx, client, tenantB.ID, tkB.ID)
	_ = tkA

	respA, err := svc.GetDeepAnalytics(ctx, analyticsSLARequest(), tenantA.ID)
	require.NoError(t, err)
	require.InDelta(t, 100.0, respA.Summary.SLACompliance, 0.001, "A 租户无违规，不受 B 租户违规影响")

	respB, err := svc.GetDeepAnalytics(ctx, analyticsSLARequest(), tenantB.ID)
	require.NoError(t, err)
	require.InDelta(t, 0.0, respB.Summary.SLACompliance, 0.001, "B 租户唯一工单违规，合规率 0")
}
