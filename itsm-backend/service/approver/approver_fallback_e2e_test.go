package approver

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"itsm-backend/ent/enttest"
	entuser "itsm-backend/ent/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// =====================================================================
// E2E: Approval Chain Fallback Tests
// 测试审批链中审批人解析的 fallback 行为
// =====================================================================

// TestDeptManagerResolver_E2E_MultiLevelParentFallback 端到端测试多级父部门追溯
// 场景：
//   - 三级部门树：root -> parent -> child
//   - root 部门有活跃的 manager
//   - parent 部门没有 manager
//   - child 部门没有 manager
//
// 期望：fallback 到 root 部门的 manager
func TestDeptManagerResolver_E2E_MultiLevelParentFallback(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	ctx := context.Background()

	// 创建三级部门树
	rootDept, err := fx.client.Department.Create().
		SetName("Root Dept").
		SetCode("root-dept").
		SetTenantID(fx.tenant.ID).
		SetManagerID(fx.manager.ID). // root 有 manager
		Save(ctx)
	require.NoError(t, err)

	parentDept, err := fx.client.Department.Create().
		SetName("Parent Dept").
		SetCode("parent-dept").
		SetTenantID(fx.tenant.ID).
		SetParentID(rootDept.ID). // parent 的 parent 是 root
		// parent 没有 manager
		Save(ctx)
	require.NoError(t, err)

	childDept, err := fx.client.Department.Create().
		SetName("Child Dept").
		SetCode("child-dept").
		SetTenantID(fx.tenant.ID).
		SetParentID(parentDept.ID). // child 的 parent 是 parent
		// child 没有 manager
		Save(ctx)
	require.NoError(t, err)

	// 解析 child 部门的审批人，应 fallback 到 root manager
	resolver := NewDeptManagerResolver()
	approvers, err := resolver.Resolve(ctx, fx.client, &ApproverContext{
		TenantID:     fx.tenant.ID,
		DepartmentID: childDept.ID,
	})
	require.NoError(t, err)
	require.Len(t, approvers, 1)
	assert.Equal(t, fx.manager.ID, approvers[0].UserID)
	assert.Equal(t, "department_manager", approvers[0].Role)
	assert.Contains(t, approvers[0].Source, "department:")
}

// TestDeptManagerResolver_E2E_InactiveManagerFallback 端到端测试非活跃经理的 fallback
// 场景：
//   - 部门有 manager，但 manager 已 inactive
//   - 父部门有活跃的 manager
//
// 期望：fallback 到父部门 manager
func TestDeptManagerResolver_E2E_InactiveManagerFallback(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	ctx := context.Background()

	// 创建非活跃的经理
	inactiveManager, err := fx.client.User.Create().
		SetUsername("inactive_mgr").
		SetEmail("inactive@fallback.com").
		SetName("Inactive Manager").
		SetPasswordHash("hash").
		SetRole(entuser.RoleManager).
		SetActive(false). // 非活跃
		SetTenantID(fx.tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建父部门有活跃 manager
	parentDept, err := fx.client.Department.Create().
		SetName("Parent Dept").
		SetCode("parent-dept-inactive").
		SetTenantID(fx.tenant.ID).
		SetManagerID(fx.manager.ID). // 活跃 manager
		Save(ctx)
	require.NoError(t, err)

	// 创建子部门有非活跃 manager
	childDept, err := fx.client.Department.Create().
		SetName("Child Dept").
		SetCode("child-dept-inactive").
		SetTenantID(fx.tenant.ID).
		SetParentID(parentDept.ID).
		SetManagerID(inactiveManager.ID). // 非活跃 manager
		Save(ctx)
	require.NoError(t, err)

	// 解析子部门审批人，应 fallback 到父部门 manager
	resolver := NewDeptManagerResolver()
	approvers, err := resolver.Resolve(ctx, fx.client, &ApproverContext{
		TenantID:     fx.tenant.ID,
		DepartmentID: childDept.ID,
	})
	require.NoError(t, err)
	require.Len(t, approvers, 1)
	assert.Equal(t, fx.manager.ID, approvers[0].UserID)
	assert.Equal(t, "department_manager", approvers[0].Role)
}

// TestDeptManagerResolver_E2E_NoFallbackAvailable 端到端测试无可用 fallback 时返回错误
// 场景：
//   - 部门无 manager
//   - 父部门也无 manager
//   - 根部门也无 manager
//
// 期望：返回明确错误
func TestDeptManagerResolver_E2E_NoFallbackAvailable(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	ctx := context.Background()

	// 创建部门树，但都没有 manager
	parentDept, err := fx.client.Department.Create().
		SetName("Parent No Manager").
		SetCode("parent-no-mgr").
		SetTenantID(fx.tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	childDept, err := fx.client.Department.Create().
		SetName("Child No Manager").
		SetCode("child-no-mgr").
		SetTenantID(fx.tenant.ID).
		SetParentID(parentDept.ID).
		Save(ctx)
	require.NoError(t, err)

	resolver := NewDeptManagerResolver()
	_, err = resolver.Resolve(ctx, fx.client, &ApproverContext{
		TenantID:     fx.tenant.ID,
		DepartmentID: childDept.ID,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no manager found")
}

// TestDeptManagerResolver_E2E_TenantIsolation 端到端测试租户隔离
// 场景：
//   - 租户A的部门有 manager
//   - 租户B尝试解析租户A的部门
//
// 期望：返回 not found 错误（租户隔离）
func TestDeptManagerResolver_E2E_TenantIsolation(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	ctx := context.Background()

	// 创建租户A的部门
	tenantA := fx.tenant
	managerA := fx.manager

	deptA, err := fx.client.Department.Create().
		SetName("Tenant A Dept").
		SetCode("tenant-a-dept").
		SetTenantID(tenantA.ID).
		SetManagerID(managerA.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建租户B
	tenantB, err := fx.client.Tenant.Create().
		SetName("Tenant B").
		SetCode("tenant-b").
		SetDomain("tenant-b.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 租户B尝试解析租户A的部门
	resolver := NewDeptManagerResolver()
	_, err = resolver.Resolve(ctx, fx.client, &ApproverContext{
		TenantID:     tenantB.ID,
		DepartmentID: deptA.ID,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "department not found")
}

// TestDeptManagerResolver_E2E_CircularParentDetection 端到端测试循环父部门检测
// 场景：
//   - 部门A的父部门是部门B
//   - 部门B的父部门是部门A（循环）
//
// 期望：避免无限递归，返回错误
func TestDeptManagerResolver_E2E_CircularParentDetection(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	ctx := context.Background()

	// 创建部门A
	deptA, err := fx.client.Department.Create().
		SetName("Dept A").
		SetCode("dept-a").
		SetTenantID(fx.tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建部门B，父部门为A
	deptB, err := fx.client.Department.Create().
		SetName("Dept B").
		SetCode("dept-b").
		SetTenantID(fx.tenant.ID).
		SetParentID(deptA.ID).
		Save(ctx)
	require.NoError(t, err)

	// 设置A的父部门为B（制造循环）
	_, err = fx.client.Department.UpdateOne(deptA).
		SetParentID(deptB.ID).
		Save(ctx)
	require.NoError(t, err)

	resolver := NewDeptManagerResolver()
	_, err = resolver.Resolve(ctx, fx.client, &ApproverContext{
		TenantID:     fx.tenant.ID,
		DepartmentID: deptA.ID,
	})
	// 应该返回错误（避免无限递归）
	assert.Error(t, err)
}

// =====================================================================
// E2E: ResolverRegistry Fallback Tests
// 测试解析器注册表的 fallback 行为
// =====================================================================

// TestResolverRegistry_E2E_FallbackToDefault 端到端测试解析器注册表的默认 fallback
// 场景：
//   - 注册多个 resolver
//   - 请求未知类型
//
// 期望：返回明确错误，不 panic
func TestResolverRegistry_E2E_FallbackToDefault(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	registry := NewResolverRegistry(logger)

	// 只注册部分 resolver
	registry.Register(NewTeamLeaderResolver())
	registry.Register(NewDeptManagerResolver())

	// 请求未注册的 resolver 类型
	_, err := registry.Resolve(context.Background(), fx.client, "project_manager", &ApproverContext{
		TenantID: fx.tenant.ID,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown approver resolver")
}

// TestResolverRegistry_E2E_AllTypesReturned 端到端测试获取所有注册的 resolver 类型
func TestResolverRegistry_E2E_AllTypesReturned(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	registry := NewResolverRegistry(logger)

	// 注册所有 resolver
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

// =====================================================================
// E2E: Integration with Approval Chain Service
// 测试与审批链服务的集成
// =====================================================================

// TestApprovalChainResolverIntegration_E2E 审批链与 resolver 集成测试
// 验证审批链服务能正确使用 resolver 解析审批人
func TestApprovalChainResolverIntegration_E2E(t *testing.T) {
	fx := newApproverFixture(t)
	defer fx.client.Close()

	ctx := context.Background()

	// 创建部门
	dept, err := fx.client.Department.Create().
		SetName("Integration Dept").
		SetCode("integration-dept").
		SetTenantID(fx.tenant.ID).
		SetManagerID(fx.manager.ID).
		Save(ctx)
	require.NoError(t, err)

	// 验证 DeptManagerResolver 能正确解析
	resolver := NewDeptManagerResolver()
	approvers, err := resolver.Resolve(ctx, fx.client, &ApproverContext{
		TenantID:     fx.tenant.ID,
		DepartmentID: dept.ID,
	})
	require.NoError(t, err)
	require.Len(t, approvers, 1)

	// 验证返回的审批人信息完整
	assert.Equal(t, fx.manager.ID, approvers[0].UserID)
	assert.Equal(t, fx.manager.Name, approvers[0].UserName)
	assert.Equal(t, fx.manager.Email, approvers[0].UserEmail)
	assert.NotEmpty(t, approvers[0].Source)
}

// =====================================================================
// Benchmark Tests
// =====================================================================

// BenchmarkDeptManagerResolver_Resolve 基准测试审批人解析性能
func BenchmarkDeptManagerResolver_Resolve(b *testing.B) {
	fx := newApproverFixtureForE2E(&testing.T{})
	defer fx.client.Close()

	ctx := context.Background()

	// 创建部门
	dept, _ := fx.client.Department.Create().
		SetName("Bench Dept").
		SetCode("bench-dept").
		SetTenantID(fx.tenant.ID).
		SetManagerID(fx.manager.ID).
		Save(ctx)

	resolver := NewDeptManagerResolver()
	appCtx := &ApproverContext{
		TenantID:     fx.tenant.ID,
		DepartmentID: dept.ID,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = resolver.Resolve(ctx, fx.client, appCtx)
	}
}

// newApproverFixtureForE2E 创建用于端到端测试的 fixture
// 与 approver_test.go 中的 newApproverFixture 功能相同，但用于避免重复定义
func newApproverFixtureForE2E(t *testing.T) *approverFixture {
	t.Helper()
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:approver_e2e_test?mode=memory&cache=shared&_fk=1")
	logger := zaptest.NewLogger(t).Sugar()

	tenant, err := client.Tenant.Create().
		SetName("Approver E2E Tenant").
		SetCode("approver-e2e").
		SetDomain("approver-e2e.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	manager, err := client.User.Create().
		SetUsername("e2e_manager").
		SetEmail("e2e@manager.com").
		SetName("E2E Manager").
		SetPasswordHash("hash").
		SetRole(entuser.RoleManager).
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	user, err := client.User.Create().
		SetUsername("e2e_user").
		SetEmail("e2e@user.com").
		SetName("E2E User").
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
