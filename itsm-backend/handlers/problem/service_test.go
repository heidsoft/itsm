package problem

import (
	"context"
	"fmt"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func setupProblemHandlerTest(t *testing.T) (*ent.Client, *Service, context.Context) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:problem-handler-%s?mode=memory&cache=shared&_fk=1", t.Name()))
	repo := NewEntRepository(client)
	return client, NewService(repo, zaptest.NewLogger(t).Sugar()), context.Background()
}

func createProblemHandlerTenant(t *testing.T, ctx context.Context, client *ent.Client, suffix string) *ent.Tenant {
	t.Helper()
	tenant, err := client.Tenant.Create().
		SetName("Problem Tenant " + suffix).
		SetCode("problem-" + suffix).
		SetDomain("problem-" + suffix + ".example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	return tenant
}

func createProblemHandlerUser(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, suffix string) *ent.User {
	t.Helper()
	user, err := client.User.Create().
		SetUsername("problem-" + suffix).
		SetEmail("problem-" + suffix + "@example.com").
		SetName("Problem User").
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return user
}

func createProblemHandlerProblem(t *testing.T, ctx context.Context, service *Service, tenantID, userID int) *Problem {
	t.Helper()
	p, err := service.Create(ctx, tenantID, &Problem{
		Title: "Repeated outage", Description: "Repeated production outage", Priority: "high", CreatedBy: userID,
	})
	require.NoError(t, err)
	return p
}

func TestProblemServiceLifecycleAndTimestamps(t *testing.T) {
	client, service, ctx := setupProblemHandlerTest(t)
	defer client.Close()
	tenant := createProblemHandlerTenant(t, ctx, client, "lifecycle")
	user := createProblemHandlerUser(t, ctx, client, tenant.ID, "lifecycle")
	p := createProblemHandlerProblem(t, ctx, service, tenant.ID, user.ID)

	assert.Equal(t, "open", p.Status)
	p, err := service.Update(ctx, tenant.ID, p.ID, &Problem{Status: "investigating"})
	require.NoError(t, err)
	assert.Nil(t, p.ResolvedAt)
	p, err = service.Update(ctx, tenant.ID, p.ID, &Problem{Status: "resolved"})
	require.NoError(t, err)
	require.NotNil(t, p.ResolvedAt)
	p, err = service.Update(ctx, tenant.ID, p.ID, &Problem{Status: "investigating"})
	require.NoError(t, err)
	assert.Nil(t, p.ResolvedAt)

	_, err = service.Update(ctx, tenant.ID, p.ID, &Problem{Status: "unknown"})
	require.Error(t, err)
	var badStatusErr *common.BusinessError
	require.ErrorAs(t, err, &badStatusErr, "非法状态必须是 BusinessError")
	assert.Equal(t, common.ConflictCode, badStatusErr.Code)
}

func TestGoldenJourney_ProblemRCAResolvedAndClosed(t *testing.T) {
	client, service, ctx := setupProblemHandlerTest(t)
	defer client.Close()
	tenant := createProblemHandlerTenant(t, ctx, client, "golden")
	user := createProblemHandlerUser(t, ctx, client, tenant.ID, "golden")
	p := createProblemHandlerProblem(t, ctx, service, tenant.ID, user.ID)

	_, err := service.CloseProblem(ctx, tenant.ID, p.ID, "不得跳过分析")
	require.Error(t, err)
	var bizErr *common.BusinessError
	require.ErrorAs(t, err, &bizErr, "open 直接 closed 必须被状态机以 BusinessError 拒绝")
	assert.Equal(t, common.ConflictCode, bizErr.Code)
	p, err = service.InvestigateProblem(ctx, tenant.ID, p.ID)
	require.NoError(t, err)
	p, err = service.UpdateRootCause(ctx, tenant.ID, p.ID, "连接池耗尽")
	require.NoError(t, err)
	p, err = service.UpdateSolution(ctx, tenant.ID, p.ID, "临时扩容", "修复连接泄漏")
	require.NoError(t, err)
	p, err = service.Update(ctx, tenant.ID, p.ID, &Problem{Status: "resolved"})
	require.NoError(t, err)
	p, err = service.CloseProblem(ctx, tenant.ID, p.ID, "修复连接泄漏并观察稳定")
	require.NoError(t, err)
	assert.Equal(t, "closed", p.Status)
	assert.Equal(t, "连接池耗尽", p.RootCause)
	assert.Equal(t, "临时扩容", p.Workaround)
	require.NotNil(t, p.ResolvedAt)
	require.NotNil(t, p.ClosedAt)

	_, err = service.Get(ctx, p.ID, tenant.ID+1)
	require.Error(t, err, "cross-tenant direct ID must fail closed")
}

func TestProblemRepositorySoftDeleteExcludedEverywhere(t *testing.T) {
	client, service, ctx := setupProblemHandlerTest(t)
	defer client.Close()
	tenant := createProblemHandlerTenant(t, ctx, client, "delete")
	user := createProblemHandlerUser(t, ctx, client, tenant.ID, "delete")
	p := createProblemHandlerProblem(t, ctx, service, tenant.ID, user.ID)

	require.NoError(t, service.Delete(ctx, p.ID, tenant.ID))
	_, err := service.Get(ctx, p.ID, tenant.ID)
	require.True(t, ent.IsNotFound(err))
	list, total, err := service.List(ctx, tenant.ID, 1, 10, nil, user.ID, "super_admin")
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, list)
	stats, err := service.GetStats(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Zero(t, stats.Total)

	stored, err := client.Problem.Get(ctx, p.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.DeletedAt)
}

func TestProblemAssociationsEnforceTenantBoundary(t *testing.T) {
	client, service, ctx := setupProblemHandlerTest(t)
	defer client.Close()
	tenantA := createProblemHandlerTenant(t, ctx, client, "association-a")
	tenantB := createProblemHandlerTenant(t, ctx, client, "association-b")
	userA := createProblemHandlerUser(t, ctx, client, tenantA.ID, "association-a")
	userB := createProblemHandlerUser(t, ctx, client, tenantB.ID, "association-b")
	p := createProblemHandlerProblem(t, ctx, service, tenantA.ID, userA.ID)

	localTicket, err := client.Ticket.Create().
		SetTitle("Local ticket").SetTicketNumber("PRB-LOCAL").SetRequesterID(userA.ID).SetTenantID(tenantA.ID).Save(ctx)
	require.NoError(t, err)
	foreignTicket, err := client.Ticket.Create().
		SetTitle("Foreign ticket").SetTicketNumber("PRB-FOREIGN").SetRequesterID(userB.ID).SetTenantID(tenantB.ID).Save(ctx)
	require.NoError(t, err)

	require.NoError(t, service.AddAssociations(ctx, tenantA.ID, p.ID, "ticket", []int{localTicket.ID, localTicket.ID}))
	err = service.AddAssociations(ctx, tenantA.ID, p.ID, "ticket", []int{foreignTicket.ID})
	require.ErrorContains(t, err, "current tenant")

	withAssociations, err := service.GetWithAssociations(ctx, p.ID, tenantA.ID)
	require.NoError(t, err)
	require.Len(t, withAssociations.Tickets, 1)
	assert.Equal(t, localTicket.ID, withAssociations.Tickets[0].ID)
}

// TestProblemInvalidTransitionReturnsBusinessError 确保状态机违规返回
// BusinessError(4090) 而非裸 error —— 否则 handler 会误判为内部错误返回 500。
func TestProblemInvalidTransitionReturnsBusinessError(t *testing.T) {
	client, service, ctx := setupProblemHandlerTest(t)
	tenant := createProblemHandlerTenant(t, ctx, client, "bizerr")
	user := createProblemHandlerUser(t, ctx, client, tenant.ID, "bizerr")
	p := createProblemHandlerProblem(t, ctx, service, tenant.ID, user.ID)

	// open -> investigating 合法
	_, err := service.InvestigateProblem(ctx, tenant.ID, p.ID)
	require.NoError(t, err)

	// investigating -> closed 非法：必须先经过 resolved
	_, err = service.CloseProblem(ctx, tenant.ID, p.ID, "resolution")
	require.Error(t, err)
	var bizErr *common.BusinessError
	require.ErrorAs(t, err, &bizErr, "状态机违规必须是 BusinessError")
	assert.Equal(t, common.ConflictCode, bizErr.Code)

	// 合法路径 investigating -> resolved -> closed 应成功
	_, err = service.Update(ctx, tenant.ID, p.ID, &Problem{Status: "resolved"})
	require.NoError(t, err)
	closed, err := service.CloseProblem(ctx, tenant.ID, p.ID, "resolution")
	require.NoError(t, err)
	assert.Equal(t, "closed", closed.Status)
}
