package service

import (
	"context"
	"testing"
	"time"

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
	created, err := svc.Create(ctx, agent.ID, customer.ID, "backup", "super_admin")
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
