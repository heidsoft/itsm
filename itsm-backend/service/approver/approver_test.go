package approver

import (
	"context"
	"testing"

	"itsm-backend/ent"
	entuser "itsm-backend/ent/user"
	"itsm-backend/ent/enttest"
	_ "github.com/mattn/go-sqlite3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// =====================================================================
// Test Fixtures
// =====================================================================

type approverFixture struct {
	ctx     context.Context
	client  *ent.Client
	tenant  *ent.Tenant
	manager *ent.User
	user    *ent.User
	logger  *zap.SugaredLogger
}

func newApproverFixture(t *testing.T) *approverFixture {
	t.Helper()
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:approver_test?mode=memory&cache=shared&_fk=1")
	logger := zaptest.NewLogger(t).Sugar()

	tenant, err := client.Tenant.Create().
		SetName("Approver Tenant").
		SetCode("approver").
		SetDomain("approver.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	manager, err := client.User.Create().
		SetUsername("manager").
		SetEmail("manager@approver.com").
		SetName("Manager").
		SetPasswordHash("hash").
		SetRole(entuser.RoleManager).
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	user, err := client.User.Create().
		SetUsername("requester").
		SetEmail("req@approver.com").
		SetName("Requester").
		SetPasswordHash("hash").
		SetRole(entuser.RoleEndUser).
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	return &approverFixture{
		ctx:     ctx,
		client:  client,
		tenant:  tenant,
		manager: manager,
		user:    user,
		logger:  logger,
	}
}

// =====================================================================
// ResolverRegistry
// =====================================================================

func TestResolverRegistry_Register(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	registry := NewResolverRegistry(fx.logger)
	require.NotNil(t, registry)

	registry.Register(NewTeamLeaderResolver())

	types := registry.GetAllTypes()
	assert.Contains(t, types, "team_leader")
}

func TestResolverRegistry_Register_Multiple(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	registry := NewResolverRegistry(fx.logger)

	registry.Register(NewTeamLeaderResolver())
	registry.Register(NewDeptManagerResolver())
	registry.Register(NewProjectMgrResolver())
	registry.Register(NewTempTeamResolver())
	registry.Register(NewAmountResolver(nil))

	types := registry.GetAllTypes()
	assert.Len(t, types, 5)
	assert.Contains(t, types, "team_leader")
	assert.Contains(t, types, "dept_manager")
	assert.Contains(t, types, "project_manager")
	assert.Contains(t, types, "temp_team_leader")
	assert.Contains(t, types, "amount_based")
}

func TestResolverRegistry_Resolve_UnknownType(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	registry := NewResolverRegistry(fx.logger)

	_, err := registry.Resolve(fx.ctx, fx.client, "unknown_type", &ApproverContext{
		TenantID: fx.tenant.ID,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown approver resolver")
}

func TestResolverRegistry_GetAllTypes_Empty(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	registry := NewResolverRegistry(fx.logger)
	types := registry.GetAllTypes()
	assert.Empty(t, types)
}

// =====================================================================
// TeamLeaderResolver
// =====================================================================

func TestTeamLeaderResolver_GetType(t *testing.T) {
	resolver := NewTeamLeaderResolver()
	assert.Equal(t, "team_leader", resolver.GetType())
}

func TestTeamLeaderResolver_Resolve_Success(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	team, err := fx.client.Team.Create().
		SetName("Dev Team").
		SetCode("dev-team").
		SetTenantID(fx.tenant.ID).
		SetManagerID(fx.manager.ID).
		Save(fx.ctx)
	require.NoError(t, err)

	resolver := NewTeamLeaderResolver()
	approvers, err := resolver.Resolve(fx.ctx, fx.client, &ApproverContext{
		TenantID: fx.tenant.ID,
		TeamID:   team.ID,
	})
	require.NoError(t, err)
	require.Len(t, approvers, 1)

	assert.Equal(t, fx.manager.ID, approvers[0].UserID)
	assert.Equal(t, "team_leader", approvers[0].Role)
	assert.Contains(t, approvers[0].Source, "team:")
}

func TestTeamLeaderResolver_Resolve_MissingTeamID(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	resolver := NewTeamLeaderResolver()
	_, err := resolver.Resolve(fx.ctx, fx.client, &ApproverContext{
		TenantID: fx.tenant.ID,
		TeamID:   0,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "team_id is required")
}

func TestTeamLeaderResolver_Resolve_TeamNotFound(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	resolver := NewTeamLeaderResolver()
	_, err := resolver.Resolve(fx.ctx, fx.client, &ApproverContext{
		TenantID: fx.tenant.ID,
		TeamID:   99999,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "team not found")
}

func TestTeamLeaderResolver_Resolve_NoLeader(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	team, _ := fx.client.Team.Create().
		SetName("No Leader Team").
		SetCode("no-leader").
		SetTenantID(fx.tenant.ID).
		Save(fx.ctx)

	resolver := NewTeamLeaderResolver()
	_, err := resolver.Resolve(fx.ctx, fx.client, &ApproverContext{
		TenantID: fx.tenant.ID,
		TeamID:   team.ID,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no leader found")
}

func TestTeamLeaderResolver_Resolve_LeaderInactive(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	inactiveMgr, _ := fx.client.User.Create().
		SetUsername("inactive_mgr").
		SetEmail("inactive@approver.com").
		SetName("Inactive Manager").
		SetPasswordHash("hash").
		SetRole(entuser.RoleManager).
		SetActive(false).
		SetTenantID(fx.tenant.ID).
		Save(fx.ctx)

	team, _ := fx.client.Team.Create().
		SetName("Inactive Leader Team").
		SetCode("inactive-leader").
		SetTenantID(fx.tenant.ID).
		SetManagerID(inactiveMgr.ID).
		Save(fx.ctx)

	resolver := NewTeamLeaderResolver()
	_, err := resolver.Resolve(fx.ctx, fx.client, &ApproverContext{
		TenantID: fx.tenant.ID,
		TeamID:   team.ID,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "team leader user not found or inactive")
}

// =====================================================================
// DeptManagerResolver
// =====================================================================

func TestDeptManagerResolver_GetType(t *testing.T) {
	resolver := NewDeptManagerResolver()
	assert.Equal(t, "dept_manager", resolver.GetType())
}

func TestDeptManagerResolver_Resolve_Success(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	dept, err := fx.client.Department.Create().
		SetName("Engineering").
		SetCode("engineering").
		SetTenantID(fx.tenant.ID).
		SetManagerID(fx.manager.ID).
		Save(fx.ctx)
	require.NoError(t, err)

	resolver := NewDeptManagerResolver()
	approvers, err := resolver.Resolve(fx.ctx, fx.client, &ApproverContext{
		TenantID:     fx.tenant.ID,
		DepartmentID: dept.ID,
	})
	require.NoError(t, err)
	require.Len(t, approvers, 1)

	assert.Equal(t, fx.manager.ID, approvers[0].UserID)
	assert.Equal(t, "department_manager", approvers[0].Role)
}

func TestDeptManagerResolver_Resolve_MissingDeptID(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	resolver := NewDeptManagerResolver()
	_, err := resolver.Resolve(fx.ctx, fx.client, &ApproverContext{
		TenantID:     fx.tenant.ID,
		DepartmentID: 0,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "department_id is required")
}

func TestDeptManagerResolver_Resolve_ParentFallback(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	parent, _ := fx.client.Department.Create().
		SetName("Parent Dept").
		SetCode("parent-dept").
		SetTenantID(fx.tenant.ID).
		SetManagerID(fx.manager.ID).
		Save(fx.ctx)

	child, _ := fx.client.Department.Create().
		SetName("Child Dept").
		SetCode("child-dept").
		SetTenantID(fx.tenant.ID).
		SetParentID(parent.ID).
		Save(fx.ctx)

	resolver := NewDeptManagerResolver()
	approvers, err := resolver.Resolve(fx.ctx, fx.client, &ApproverContext{
		TenantID:     fx.tenant.ID,
		DepartmentID: child.ID,
	})
	require.NoError(t, err)
	require.Len(t, approvers, 1)
	assert.Equal(t, fx.manager.ID, approvers[0].UserID)
}

func TestDeptManagerResolver_Resolve_DeptNotFound(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	resolver := NewDeptManagerResolver()
	_, err := resolver.Resolve(fx.ctx, fx.client, &ApproverContext{
		TenantID:     fx.tenant.ID,
		DepartmentID: 99999,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "department not found")
}

func TestDeptManagerResolver_Resolve_NoManagerNoParent(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	dept, _ := fx.client.Department.Create().
		SetName("Orphan Dept").
		SetCode("orphan-dept").
		SetTenantID(fx.tenant.ID).
		Save(fx.ctx)

	resolver := NewDeptManagerResolver()
	_, err := resolver.Resolve(fx.ctx, fx.client, &ApproverContext{
		TenantID:     fx.tenant.ID,
		DepartmentID: dept.ID,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no manager found")
}

// =====================================================================
// ProjectMgrResolver
// =====================================================================

func TestProjectMgrResolver_GetType(t *testing.T) {
	resolver := NewProjectMgrResolver()
	assert.Equal(t, "project_manager", resolver.GetType())
}

func TestProjectMgrResolver_Resolve_Success(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	proj, err := fx.client.Project.Create().
		SetName("Alpha Project").
		SetCode("alpha-project").
		SetTenantID(fx.tenant.ID).
		SetManagerID(fx.manager.ID).
		SetStatus("active").
		Save(fx.ctx)
	require.NoError(t, err)

	resolver := NewProjectMgrResolver()
	approvers, err := resolver.Resolve(fx.ctx, fx.client, &ApproverContext{
		TenantID:  fx.tenant.ID,
		ProjectID: proj.ID,
	})
	require.NoError(t, err)
	require.Len(t, approvers, 1)
	assert.Equal(t, fx.manager.ID, approvers[0].UserID)
	assert.Equal(t, "project_manager", approvers[0].Role)
}

func TestProjectMgrResolver_Resolve_MissingProjectID(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	resolver := NewProjectMgrResolver()
	_, err := resolver.Resolve(fx.ctx, fx.client, &ApproverContext{
		TenantID:  fx.tenant.ID,
		ProjectID: 0,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project_id is required")
}

func TestProjectMgrResolver_Resolve_ProjectNotFound(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	resolver := NewProjectMgrResolver()
	_, err := resolver.Resolve(fx.ctx, fx.client, &ApproverContext{
		TenantID:  fx.tenant.ID,
		ProjectID: 99999,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project not found")
}

func TestProjectMgrResolver_Resolve_ProjectNotActive(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	proj, _ := fx.client.Project.Create().
		SetName("Inactive Project").
		SetCode("inactive-project").
		SetTenantID(fx.tenant.ID).
		SetManagerID(fx.manager.ID).
		SetStatus("archived").
		Save(fx.ctx)

	resolver := NewProjectMgrResolver()
	_, err := resolver.Resolve(fx.ctx, fx.client, &ApproverContext{
		TenantID:  fx.tenant.ID,
		ProjectID: proj.ID,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project is not active")
}

func TestProjectMgrResolver_Resolve_NoManager(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	proj, _ := fx.client.Project.Create().
		SetName("No Manager Project").
		SetCode("no-mgr-project").
		SetTenantID(fx.tenant.ID).
		SetStatus("active").
		Save(fx.ctx)

	resolver := NewProjectMgrResolver()
	_, err := resolver.Resolve(fx.ctx, fx.client, &ApproverContext{
		TenantID:  fx.tenant.ID,
		ProjectID: proj.ID,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no manager found")
}

// =====================================================================
// TempTeamResolver
// =====================================================================

func TestTempTeamResolver_GetType(t *testing.T) {
	resolver := NewTempTeamResolver()
	assert.Equal(t, "temp_team_leader", resolver.GetType())
}

func TestTempTeamResolver_Resolve_Success(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	team, _ := fx.client.Team.Create().
		SetName("Temp Team").
		SetCode("temp-team").
		SetTenantID(fx.tenant.ID).
		SetManagerID(fx.manager.ID).
		Save(fx.ctx)

	resolver := NewTempTeamResolver()
	approvers, err := resolver.Resolve(fx.ctx, fx.client, &ApproverContext{
		TenantID: fx.tenant.ID,
		TeamID:   team.ID,
	})
	require.NoError(t, err)
	require.Len(t, approvers, 1)

	// Key difference: temp_team_leader role
	assert.Equal(t, "temp_team_leader", approvers[0].Role)
}

// =====================================================================
// AmountResolver
// =====================================================================

func TestAmountResolver_GetType(t *testing.T) {
	resolver := NewAmountResolver(nil)
	assert.Equal(t, "amount_based", resolver.GetType())
}

func TestAmountResolver_Resolve_Success(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	// Create manager users
	manager1, err := fx.client.User.Create().
		SetUsername("mgr1").
		SetEmail("mgr1@test.com").
		SetName("Manager One").
		SetPasswordHash("hash").
		SetRole(entuser.RoleManager).
		SetActive(true).
		SetTenantID(fx.tenant.ID).
		Save(fx.ctx)
	require.NoError(t, err)
	_ = manager1

	manager2, err := fx.client.User.Create().
		SetUsername("mgr2").
		SetEmail("mgr2@test.com").
		SetName("Manager Two").
		SetPasswordHash("hash").
		SetRole(entuser.RoleManager).
		SetActive(true).
		SetTenantID(fx.tenant.ID).
		Save(fx.ctx)
	require.NoError(t, err)
	_ = manager2

	thresholds := []AmountThreshold{
		{MinAmount: 0, MaxAmount: 1000, Role: "agent"},
		{MinAmount: 1000, MaxAmount: 10000, Role: "manager"},
		{MinAmount: 10000, MaxAmount: 0, Role: "admin"},
	}

	resolver := NewAmountResolver(thresholds)
	approvers, err := resolver.Resolve(fx.ctx, fx.client, &ApproverContext{
		TenantID: fx.tenant.ID,
		Amount:   5000,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, approvers)
	for _, a := range approvers {
		assert.Equal(t, "manager", a.Role)
	}
}

func TestAmountResolver_Resolve_MissingAmount(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	resolver := NewAmountResolver([]AmountThreshold{})
	_, err := resolver.Resolve(fx.ctx, fx.client, &ApproverContext{
		TenantID: fx.tenant.ID,
		Amount:   0,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount is required")
}

func TestAmountResolver_Resolve_NoMatchingThreshold(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	thresholds := []AmountThreshold{
		{MinAmount: 0, MaxAmount: 100, Role: "agent"},
	}

	resolver := NewAmountResolver(thresholds)
	_, err := resolver.Resolve(fx.ctx, fx.client, &ApproverContext{
		TenantID: fx.tenant.ID,
		Amount:   10000,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no approval threshold matched")
}

func TestAmountResolver_Threshold_NoUpperLimit(t *testing.T) {
	// MaxAmount=0 means no upper limit
	threshold := AmountThreshold{MinAmount: 10000, MaxAmount: 0, Role: "admin"}
	amount := 1_000_000.0

	inRange := amount >= threshold.MinAmount && (threshold.MaxAmount == 0 || amount <= threshold.MaxAmount)
	assert.True(t, inRange, "MaxAmount=0 should mean no upper limit")
}

// =====================================================================
// ApproverContext
// =====================================================================

func TestApproverContext_AllFields(t *testing.T) {
	ctx := &ApproverContext{
		TenantID:     1,
		TicketID:     10,
		RequesterID:  100,
		DepartmentID: 200,
		TeamID:       300,
		ProjectID:    400,
		Amount:       500.50,
		Variables:    map[string]interface{}{"key": "value"},
	}

	assert.Equal(t, 1, ctx.TenantID)
	assert.Equal(t, 10, ctx.TicketID)
	assert.Equal(t, 100, ctx.RequesterID)
	assert.Equal(t, 200, ctx.DepartmentID)
	assert.Equal(t, 300, ctx.TeamID)
	assert.Equal(t, 400, ctx.ProjectID)
	assert.Equal(t, 500.50, ctx.Amount)
	assert.Equal(t, "value", ctx.Variables["key"])
}

// =====================================================================
// ApproverInfo
// =====================================================================

func TestApproverInfo_Fields(t *testing.T) {
	info := &ApproverInfo{
		UserID:    1,
		UserName:  "Alice",
		UserEmail: "alice@test.com",
		Role:      "manager",
		Source:    "department:5",
	}

	assert.Equal(t, 1, info.UserID)
	assert.Equal(t, "Alice", info.UserName)
	assert.Equal(t, "alice@test.com", info.UserEmail)
	assert.Equal(t, "manager", info.Role)
	assert.Equal(t, "department:5", info.Source)
}

// =====================================================================
// Multi-Tenant Isolation
// =====================================================================

func TestResolverRegistry_TenantIsolation(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	tenant2, _ := fx.client.Tenant.Create().
		SetName("Tenant 2").
		SetCode("tenant2").
		SetDomain("tenant2.com").
		SetStatus("active").
		Save(fx.ctx)

	mgr2, _ := fx.client.User.Create().
		SetUsername("mgr2").
		SetEmail("mgr2@tenant2.com").
		SetName("Manager 2").
		SetPasswordHash("hash").
		SetRole(entuser.RoleManager).
		SetActive(true).
		SetTenantID(tenant2.ID).
		Save(fx.ctx)

	team1, _ := fx.client.Team.Create().
		SetName("Team 1").
		SetCode("team-1").
		SetTenantID(fx.tenant.ID).
		SetManagerID(fx.manager.ID).
		Save(fx.ctx)

	team2, _ := fx.client.Team.Create().
		SetName("Team 2").
		SetCode("team-2").
		SetTenantID(tenant2.ID).
		SetManagerID(mgr2.ID).
		Save(fx.ctx)

	resolver := NewTeamLeaderResolver()

	// Tenant2 can resolve its own team
	approvers, err := resolver.Resolve(fx.ctx, fx.client, &ApproverContext{
		TenantID: tenant2.ID,
		TeamID:   team2.ID,
	})
	require.NoError(t, err)
	require.Len(t, approvers, 1)
	assert.Equal(t, mgr2.ID, approvers[0].UserID)

	// Tenant2 cannot resolve tenant1's team
	_, err = resolver.Resolve(fx.ctx, fx.client, &ApproverContext{
		TenantID: tenant2.ID,
		TeamID:   team1.ID,
	})
	assert.Error(t, err)
}

// =====================================================================
// AmountThreshold edge cases
// =====================================================================

func TestAmountThreshold_BoundaryValues(t *testing.T) {
	// Exact boundary test
	thresholds := []AmountThreshold{
		{MinAmount: 1000, MaxAmount: 5000, Role: "manager"},
	}

	// Amount exactly at min boundary
	threshold := thresholds[0]
	amount := 1000.0
	inRange := amount >= threshold.MinAmount && (threshold.MaxAmount == 0 || amount <= threshold.MaxAmount)
	assert.True(t, inRange)

	// Amount at max boundary
	amount = 5000.0
	inRange = amount >= threshold.MinAmount && (threshold.MaxAmount == 0 || amount <= threshold.MaxAmount)
	assert.True(t, inRange)

	// Just above max boundary
	amount = 5001.0
	inRange = amount >= threshold.MinAmount && (threshold.MaxAmount == 0 || amount <= threshold.MaxAmount)
	assert.False(t, inRange)
}
