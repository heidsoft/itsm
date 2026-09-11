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
	p, err := service.Update(ctx, tenant.ID, p.ID, &Problem{Status: "investigating"}, user.ID, "agent")
	require.NoError(t, err)
	assert.Nil(t, p.ResolvedAt)
	p, err = service.Update(ctx, tenant.ID, p.ID, &Problem{Status: "resolved"}, user.ID, "agent")
	require.NoError(t, err)
	require.NotNil(t, p.ResolvedAt)
	p, err = service.Update(ctx, tenant.ID, p.ID, &Problem{Status: "investigating"}, user.ID, "agent")
	require.NoError(t, err)
	assert.Nil(t, p.ResolvedAt)

	_, err = service.Update(ctx, tenant.ID, p.ID, &Problem{Status: "unknown"}, user.ID, "agent")
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

	_, err := service.CloseProblem(ctx, tenant.ID, p.ID, user.ID, "agent", "不得跳过分析")
	require.Error(t, err)
	var bizErr *common.BusinessError
	require.ErrorAs(t, err, &bizErr, "open 直接 closed 必须被状态机以 BusinessError 拒绝")
	assert.Equal(t, common.ConflictCode, bizErr.Code)
	p, err = service.InvestigateProblem(ctx, tenant.ID, p.ID, user.ID, "agent")
	require.NoError(t, err)
	p, err = service.UpdateRootCause(ctx, tenant.ID, p.ID, user.ID, "agent", "连接池耗尽")
	require.NoError(t, err)
	p, err = service.UpdateSolution(ctx, tenant.ID, p.ID, user.ID, "agent", "临时扩容", "修复连接泄漏")
	require.NoError(t, err)
	p, err = service.Update(ctx, tenant.ID, p.ID, &Problem{Status: "resolved"}, user.ID, "agent")
	require.NoError(t, err)
	p, err = service.CloseProblem(ctx, tenant.ID, p.ID, user.ID, "agent", "修复连接泄漏并观察稳定")
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

	require.NoError(t, service.Delete(ctx, p.ID, tenant.ID, user.ID, "agent"))
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
	_, err := service.InvestigateProblem(ctx, tenant.ID, p.ID, user.ID, "agent")
	require.NoError(t, err)

	// investigating -> closed 非法：必须先经过 resolved
	_, err = service.CloseProblem(ctx, tenant.ID, p.ID, user.ID, "agent", "resolution")
	require.Error(t, err)
	var bizErr *common.BusinessError
	require.ErrorAs(t, err, &bizErr, "状态机违规必须是 BusinessError")
	assert.Equal(t, common.ConflictCode, bizErr.Code)

	// 合法路径 investigating -> resolved -> closed 应成功
	_, err = service.Update(ctx, tenant.ID, p.ID, &Problem{Status: "resolved"}, user.ID, "agent")
	require.NoError(t, err)
	closed, err := service.CloseProblem(ctx, tenant.ID, p.ID, user.ID, "agent", "resolution")
	require.NoError(t, err)
	assert.Equal(t, "closed", closed.Status)
}

// TestProblemWritePathRowLevelGuard 锁定 P1-DataScope 写路径行级校验：
// 写权限 ⊆ 读权限——非 owner 且非受理人的普通角色改/删他人问题单必须 403。
func TestProblemWritePathRowLevelGuard(t *testing.T) {
	client, service, ctx := setupProblemHandlerTest(t)
	defer client.Close()
	tenant := createProblemHandlerTenant(t, ctx, client, "rowguard")
	owner := createProblemHandlerUser(t, ctx, client, tenant.ID, "rowguard-owner")
	p := createProblemHandlerProblem(t, ctx, service, tenant.ID, owner.ID)

	stranger := createProblemHandlerUser(t, ctx, client, tenant.ID, "rowguard-stranger")
	assignee := createProblemHandlerUser(t, ctx, client, tenant.ID, "rowguard-assignee")

	// 管理角色先把 assignee 设为受理人（owner 委派场景）
	_, err := service.Update(ctx, tenant.ID, p.ID, &Problem{AssigneeID: &assignee.ID}, 0, "manager")
	require.NoError(t, err, "管理角色应可委派受理人")

	// 受理人可写
	_, err = service.Update(ctx, tenant.ID, p.ID, &Problem{Priority: "low"}, assignee.ID, "agent")
	require.NoError(t, err, "受理人应可修改")

	// 非 owner 且非受理人 → 403 Forbidden
	_, err = service.Update(ctx, tenant.ID, p.ID, &Problem{Title: "hijack"}, stranger.ID, "agent")
	require.Error(t, err, "无关普通角色修改他人问题单必须被拒绝")
	var appErr *common.AppError
	require.ErrorAs(t, err, &appErr, "必须是 AppError（403）")
	assert.Equal(t, common.ErrCodeForbidden, appErr.Code)

	err = service.Delete(ctx, p.ID, tenant.ID, stranger.ID, "agent")
	require.Error(t, err, "无关普通角色删除他人问题单必须被拒绝")
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, common.ErrCodeForbidden, appErr.Code)

	// 管理角色可写/可删
	_, err = service.Update(ctx, tenant.ID, p.ID, &Problem{Title: "admin touch"}, 0, "manager")
	require.NoError(t, err, "管理角色应全租户可写")

	// owner 可删
	require.NoError(t, service.Delete(ctx, p.ID, tenant.ID, owner.ID, "agent"), "owner 应可删除自己的问题单")

	// owner 仍可见数据未被越权篡改
	fetched, err := service.Get(ctx, p.ID, tenant.ID)
	if err == nil {
		assert.NotEqual(t, "hijack", fetched.Title, "被拒绝的修改不应落库")
	}
}
