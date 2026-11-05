package sla

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/common"
	"itsm-backend/ent/slaviolation"
)

// 回归：check-compliance 曾只是纯计算，compliant=false 时既不写 sla_violations，
// 也不触发升级，导致 /sla/violations 恒为 0、stats/monitoring 恒报 100% 合规。
// 判定结果必须落库，且重复调用不得灌水。
func TestSLA_CheckCompliance_PersistsViolations(t *testing.T) {
	_, client, tenantID := setupSLAHandler(t)
	defer client.Close()
	ctx := context.Background()
	uid := slaUniqueID()
	now := time.Now().UTC().Truncate(time.Second)

	user, err := client.User.Create().
		SetUsername("sla-viol-user-" + uid).
		SetEmail("sla-viol-" + uid + "@example.com").
		SetName("SLA Violation User").
		SetPasswordHash("hashedpassword").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	def, err := client.SLADefinition.Create().
		SetName("合规检查SLA-" + uid).
		SetServiceType("incident").
		SetPriority(common.PriorityHigh).
		SetResponseTime(30).
		SetResolutionTime(240).
		SetIsActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	ticket, err := client.Ticket.Create().
		SetTitle("已超时工单-" + uid).
		SetTicketNumber("TKT-VIOL-" + uid).
		SetStatus(common.TicketStatusOpen).
		SetPriority(common.PriorityHigh).
		SetRequesterID(user.ID).
		SetTenantID(tenantID).
		SetSLADefinitionID(def.ID).
		SetCreatedAt(now.Add(-6 * time.Hour)).
		SetSLAResponseDeadline(now.Add(-5 * time.Hour)).
		SetSLAResolutionDeadline(now.Add(-1 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	pastResponse := now.Add(-2 * time.Hour)
	pastResolution := now.Add(-30 * time.Minute)
	_, err = client.SLAState.Create().
		SetTenantID(tenantID).
		SetAggregateType("ticket").
		SetAggregateID(ticket.ID).
		SetStatus("active").
		SetSLADefinitionID(def.ID).
		SetResponseDeadline(pastResponse).
		SetResolutionDeadline(pastResolution).
		Save(ctx)
	require.NoError(t, err)

	repo := NewEntRepository(client)
	svc := NewService(repo, zaptest.NewLogger(t).Sugar())

	res, err := svc.CheckSLACompliance(ctx, ticket.ID, tenantID)
	require.NoError(t, err)
	assert.False(t, res.Compliant, "两个 deadline 都已过期，必须判定违规")
	assert.Contains(t, res.Message, "SLA 违规")

	open := false
	count, err := repo.CountViolations(ctx, tenantID, &open)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "响应超时与解决超时应各登记一条未解决违规")

	rows, err := client.SLAViolation.Query().
		Where(slaviolation.TenantID(tenantID), slaviolation.TicketID(ticket.ID)).
		All(ctx)
	require.NoError(t, err)
	types := map[string]bool{}
	for _, row := range rows {
		types[row.ViolationType] = true
		assert.Equal(t, def.ID, row.SLADefinitionID)
		assert.False(t, row.IsResolved)
	}
	assert.True(t, types["response_time"])
	assert.True(t, types["resolution_time"])

	// 幂等：第二次检查不能重复插入。
	res2, err := svc.CheckSLACompliance(ctx, ticket.ID, tenantID)
	require.NoError(t, err)
	assert.False(t, res2.Compliant)
	count2, err := repo.CountViolations(ctx, tenantID, &open)
	require.NoError(t, err)
	assert.Equal(t, 2, count2, "重复 check-compliance 不得灌水")
}

// 反向用例：未超时时绝不产生违规记录。
func TestSLA_CheckCompliance_CompliantDoesNotPersist(t *testing.T) {
	_, client, tenantID := setupSLAHandler(t)
	defer client.Close()
	ctx := context.Background()
	uid := slaUniqueID()
	now := time.Now().UTC().Truncate(time.Second)

	user, err := client.User.Create().
		SetUsername("sla-ok-user-" + uid).
		SetEmail("sla-ok-" + uid + "@example.com").
		SetName("SLA OK User").
		SetPasswordHash("hashedpassword").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	def, err := client.SLADefinition.Create().
		SetName("合规检查SLA-OK-" + uid).
		SetServiceType("incident").
		SetPriority(common.PriorityMedium).
		SetResponseTime(30).
		SetResolutionTime(240).
		SetIsActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	ticket, err := client.Ticket.Create().
		SetTitle("未超时工单-" + uid).
		SetTicketNumber("TKT-OK-" + uid).
		SetStatus(common.TicketStatusOpen).
		SetPriority(common.PriorityMedium).
		SetRequesterID(user.ID).
		SetTenantID(tenantID).
		SetSLADefinitionID(def.ID).
		SetCreatedAt(now).
		SetSLAResponseDeadline(now.Add(30 * time.Minute)).
		SetSLAResolutionDeadline(now.Add(4 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.SLAState.Create().
		SetTenantID(tenantID).
		SetAggregateType("ticket").
		SetAggregateID(ticket.ID).
		SetStatus("active").
		SetSLADefinitionID(def.ID).
		SetResponseDeadline(now.Add(30 * time.Minute)).
		SetResolutionDeadline(now.Add(4 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	repo := NewEntRepository(client)
	svc := NewService(repo, zaptest.NewLogger(t).Sugar())
	res, err := svc.CheckSLACompliance(ctx, ticket.ID, tenantID)
	require.NoError(t, err)
	assert.True(t, res.Compliant)

	open := false
	count, err := repo.CountViolations(ctx, tenantID, &open)
	require.NoError(t, err)
	assert.Zero(t, count, "未超时的工单不得落违规记录")
}
