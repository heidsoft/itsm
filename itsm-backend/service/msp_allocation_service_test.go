package service

import (
	"context"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/auditlog"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/mspallocation"
	"itsm-backend/ent/tenant"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 回归：重新分配不得改写已归档行的 deassigned_at。
//
// 修复前 Create 的第 4 步执行了
//
//	Update().Where(MspUserID, CustomerTenant, DeassignedAtNotNil()).SetDeassignedAt(now)
//
// 它匹配的是已经解除的归档记录，于是每一次「解除后再分配」都会把这些行的历史
// 结束时间改写为当前时间——GET /api/v1/msp/allocations/history 读到的
// 「什么时候解除的」全是假的。活跃重复分配本来已由前面的存在性检查挡下，
// 这一步没有任何要关闭的对象。
func TestMSPAllocationCreate_ReAllocationPreservesArchivedHistory(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:msp_alloc_realloc?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	mspTenant, err := client.Tenant.Create().SetName("MSP-H").SetCode("msp-h-a").SetType("msp_provider").Save(ctx)
	require.NoError(t, err)
	customer, err := client.Tenant.Create().SetName("Cust H").SetCode("cust-h-a").SetType("customer").Save(ctx)
	require.NoError(t, err)
	agent, err := client.User.Create().
		SetUsername("alloc_agent").SetEmail("alloc@example.com").SetName("Alloc Agent").
		SetPasswordHash("hash").SetTenantID(mspTenant.ID).Save(ctx)
	require.NoError(t, err)

	archivedAt := time.Date(2026, 2, 1, 8, 30, 0, 0, time.UTC)
	archived, err := client.MSPAllocation.Create().
		SetMspUserID(agent.ID).
		SetCustomerTenantID(customer.ID).
		SetRole("primary").
		SetAssignedAt(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)).
		SetDeassignedAt(archivedAt).
		Save(ctx)
	require.NoError(t, err)

	svc := NewMSPAllocationService(client, logger)
	// 管理员旁路只跳过租户类型校验，不影响下面的历史行为。
	created, err := svc.Create(ctx, agent.ID, agent.ID, customer.ID, "backup", "super_admin")
	require.NoError(t, err)
	require.NotNil(t, created)

	reloaded, err := client.MSPAllocation.Get(ctx, archived.ID)
	require.NoError(t, err)
	assert.True(t, reloaded.DeassignedAt.Equal(archivedAt),
		"归档行的解除时间必须保持 %v，实际被改写为 %v", archivedAt, reloaded.DeassignedAt)

	// 新记录必须是活跃行，历史与当前状态各自独立。
	activeCount, err := client.MSPAllocation.Query().
		Where(
			mspallocation.MspUserIDEQ(agent.ID),
			mspallocation.HasCustomerTenantWith(tenant.IDEQ(customer.ID)),
			mspallocation.DeassignedAtIsNil(),
		).
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, activeCount)
}

// 回归：非平台管理员的操作者必须与目标 MSP 用户同属一个 MSP 租户，
// 防止 MSP 租户 A 的持有者给租户 B 代授权（跨租户授权注入）。
func TestMSPAllocationCreate_RejectsCrossTenantOperator(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:msp_alloc_cross?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA, err := client.Tenant.Create().SetName("MSP-A").SetCode("msp-a").SetType("msp_provider").Save(ctx)
	require.NoError(t, err)
	tenantB, err := client.Tenant.Create().SetName("MSP-B").SetCode("msp-b").SetType("msp_provider").Save(ctx)
	require.NoError(t, err)
	customer, err := client.Tenant.Create().SetName("Cust X").SetCode("cust-x").SetType("customer").Save(ctx)
	require.NoError(t, err)
	operatorA, err := client.User.Create().
		SetUsername("op_a").SetEmail("op-a@example.com").SetName("Operator A").
		SetPasswordHash("hash").SetTenantID(tenantA.ID).Save(ctx)
	require.NoError(t, err)
	targetB, err := client.User.Create().
		SetUsername("agent_b").SetEmail("agent-b@example.com").SetName("Agent B").
		SetPasswordHash("hash").SetTenantID(tenantB.ID).Save(ctx)
	require.NoError(t, err)

	svc := NewMSPAllocationService(client, logger)
	_, err = svc.Create(ctx, operatorA.ID, targetB.ID, customer.ID, "primary", "agent")
	require.ErrorIs(t, err, ErrMSPAllocationForbidden)

	err = svc.Deactivate(ctx, operatorA.ID, targetB.ID, customer.ID, "agent")
	require.ErrorIs(t, err, ErrMSPAllocationForbidden)

	count, err := client.MSPAllocation.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, count, "被拒绝的操作不得遗留任何分配行")
}

// 同租户 MSP 管理员创建/解除分配成功，且写审计日志。
func TestMSPAllocationCreate_SameTenantOperatorSucceedsWithAudit(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:msp_alloc_audit?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	mspTenant, err := client.Tenant.Create().SetName("MSP-M").SetCode("msp-m").SetType("msp_provider").Save(ctx)
	require.NoError(t, err)
	customer, err := client.Tenant.Create().SetName("Cust M").SetCode("cust-m").SetType("customer").Save(ctx)
	require.NoError(t, err)
	admin, err := client.User.Create().
		SetUsername("msp_admin").SetEmail("msp-admin@example.com").SetName("MSP Admin").
		SetPasswordHash("hash").SetTenantID(mspTenant.ID).Save(ctx)
	require.NoError(t, err)
	agent, err := client.User.Create().
		SetUsername("msp_agent").SetEmail("msp-agent@example.com").SetName("MSP Agent").
		SetPasswordHash("hash").SetTenantID(mspTenant.ID).Save(ctx)
	require.NoError(t, err)

	svc := NewMSPAllocationService(client, logger)
	created, err := svc.Create(ctx, admin.ID, agent.ID, customer.ID, "primary", "tenant_admin")
	require.NoError(t, err)
	require.NotNil(t, created)

	require.NoError(t, svc.Deactivate(ctx, admin.ID, agent.ID, customer.ID, "tenant_admin"))

	audits, err := client.AuditLog.Query().Order(ent.Desc(auditlog.FieldCreatedAt)).All(ctx)
	require.NoError(t, err)
	actions := make([]string, 0, len(audits))
	for _, a := range audits {
		actions = append(actions, a.Action)
	}
	assert.Contains(t, actions, "msp_allocation_create")
	assert.Contains(t, actions, "msp_allocation_deactivate")
}
