package service

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"itsm-backend/common"
	"itsm-backend/ent/processauditlog"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 存量待办迁移回归锁：只读计划 / dry-run 零写入 / apply 原子且幂等 /
// 租户隔离 / 未指派改派 / 正常在途实例不被误终结。

// TestBuildPendingMigrationPlanIsReadOnlyAndTenantScoped 计划阶段只读，
// 且他租户的孤儿实例不得进入动作清单。
func TestBuildPendingMigrationPlanIsReadOnlyAndTenantScoped(t *testing.T) {
	client := newInventoryTestClient(t, "mig_plan")
	ctx := context.Background()
	tenantA, actorA := setupInventoryTenant(ctx, t, client, "mig_a")
	tenantB, _ := setupInventoryTenant(ctx, t, client, "mig_b")

	closedA, err := client.Ticket.Create().SetTicketNumber("MIG-A-1").SetRequesterID(1).
		SetTitle("A 已关闭").SetStatus("closed").SetTenantID(tenantA.ID).Save(ctx)
	require.NoError(t, err)
	instA := createInventoryInstance(ctx, t, client, tenantA.ID, "mig-a1", fmt.Sprintf("ticket:%d", closedA.ID), "running")
	createInventoryTask(ctx, t, client, tenantA.ID, instA.ID, "mig-a1-t", common.ProcessTaskStatusCreated, "", -1)

	closedB, err := client.Ticket.Create().SetTicketNumber("MIG-B-1").SetRequesterID(1).
		SetTitle("B 已关闭").SetStatus("closed").SetTenantID(tenantB.ID).Save(ctx)
	require.NoError(t, err)
	instB := createInventoryInstance(ctx, t, client, tenantB.ID, "mig-b1", fmt.Sprintf("ticket:%d", closedB.ID), "running")
	createInventoryTask(ctx, t, client, tenantB.ID, instB.ID, "mig-b1-t", common.ProcessTaskStatusCreated, "", -1)

	plan, err := BuildPendingMigrationPlan(ctx, client, PendingMigrationOptions{
		TenantID: tenantA.ID, ActorUserID: actorA.ID, Reason: "盘点", DryRun: true,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, plan.Summary.OrphanInstances)
	assert.Equal(t, 1, plan.Summary.CancelTasks)
	assert.Equal(t, 1, plan.Summary.TerminateInstances)
	for _, item := range plan.Items {
		assert.Equal(t, instA.ID, item.InstanceID, "不得出现他租户实例")
		assert.NotEqual(t, instB.ID, item.InstanceID)
		assert.Contains(t, item.IdempotencyKey, fmt.Sprintf("pending-migration:%d:", tenantA.ID))
	}

	// 计划阶段零写入。
	count, err := client.ProcessAuditLog.Query().Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

// TestApplyPendingMigrationDryRunDoesNotWrite dry-run 不产生任何写操作与审计。
func TestApplyPendingMigrationDryRunDoesNotWrite(t *testing.T) {
	client := newInventoryTestClient(t, "mig_dry")
	ctx := context.Background()
	tenant, actor := setupInventoryTenant(ctx, t, client, "mig_dry")
	closed, err := client.Ticket.Create().SetTicketNumber("MIG-DRY-1").SetRequesterID(1).
		SetTitle("已关闭").SetStatus("closed").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	inst := createInventoryInstance(ctx, t, client, tenant.ID, "mig-dry-1", fmt.Sprintf("ticket:%d", closed.ID), "running")
	task := createInventoryTask(ctx, t, client, tenant.ID, inst.ID, "mig-dry-t", common.ProcessTaskStatusCreated, "", -1)

	opts := PendingMigrationOptions{TenantID: tenant.ID, ActorUserID: actor.ID, Reason: "dry-run 验证", DryRun: true}
	plan, err := BuildPendingMigrationPlan(ctx, client, opts)
	require.NoError(t, err)
	engine := NewCustomProcessEngine(client, zaptest.NewLogger(t).Sugar()).(*CustomProcessEngine)
	result, err := engine.ApplyPendingMigration(ctx, plan, opts)
	require.NoError(t, err)
	assert.True(t, result.DryRun)
	assert.Equal(t, 0, result.Applied)

	freshTask, err := client.ProcessTask.Get(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, common.ProcessTaskStatusCreated, freshTask.Status)
	freshInst, err := client.ProcessInstance.Get(ctx, inst.ID)
	require.NoError(t, err)
	assert.Equal(t, "running", freshInst.Status)
	count, err := client.ProcessAuditLog.Query().Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

// TestApplyPendingMigrationIsAtomicAndIdempotent apply 后状态与审计同事务落库，
// 重复执行全部 skipped，且迁移计划不得终结正常在途实例。
func TestApplyPendingMigrationIsAtomicAndIdempotent(t *testing.T) {
	client := newInventoryTestClient(t, "mig_apply")
	ctx := context.Background()
	tenant, actor := setupInventoryTenant(ctx, t, client, "mig_apply")

	closed, err := client.Ticket.Create().SetTicketNumber("MIG-APPLY-1").SetRequesterID(1).
		SetTitle("已关闭").SetStatus("closed").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	orphan := createInventoryInstance(ctx, t, client, tenant.ID, "mig-apply-orphan", fmt.Sprintf("ticket:%d", closed.ID), "running")
	orphanTask := createInventoryTask(ctx, t, client, tenant.ID, orphan.ID, "mig-apply-t", common.ProcessTaskStatusCreated, "", -1)

	openTicket, err := client.Ticket.Create().SetTicketNumber("MIG-APPLY-2").SetRequesterID(1).
		SetTitle("处理中").SetStatus("in_progress").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	live := createInventoryInstance(ctx, t, client, tenant.ID, "mig-apply-live", fmt.Sprintf("ticket:%d", openTicket.ID), "running")
	createInventoryTask(ctx, t, client, tenant.ID, live.ID, "mig-apply-live-t", common.ProcessTaskStatusCreated, "", -1)

	opts := PendingMigrationOptions{
		TenantID: tenant.ID, ActorUserID: actor.ID, Reason: "孤儿实例清理验收",
	}
	plan, err := BuildPendingMigrationPlan(ctx, client, opts)
	require.NoError(t, err)
	assert.Equal(t, 1, plan.Summary.OrphanInstances)
	assert.Equal(t, 1, plan.Summary.TerminateInstances)
	assert.Equal(t, 1, plan.Summary.CancelTasks)
	assert.Equal(t, 1, plan.Summary.ReviewOnly, "正常在途实例的未指派待办只能 review")

	engine := NewCustomProcessEngine(client, zaptest.NewLogger(t).Sugar()).(*CustomProcessEngine)
	result, err := engine.ApplyPendingMigration(ctx, plan, opts)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Skipped-1, "除 review 项外应全部 applied") // review 项计入 skipped
	assert.Equal(t, 2, result.Applied)

	freshTask, err := client.ProcessTask.Get(ctx, orphanTask.ID)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", freshTask.Status)
	freshOrphan, err := client.ProcessInstance.Get(ctx, orphan.ID)
	require.NoError(t, err)
	assert.Equal(t, "terminated", freshOrphan.Status)
	freshLive, err := client.ProcessInstance.Get(ctx, live.ID)
	require.NoError(t, err)
	assert.Equal(t, "running", freshLive.Status, "正常在途实例不得被终结")

	audits, err := client.ProcessAuditLog.Query().Where(processauditlog.TenantIDEQ(tenant.ID)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, audits, "每个 applied 动作必须留审计")

	// 重复执行：状态幂等，全部 skipped，审计条数不变。
	plan2, err := BuildPendingMigrationPlan(ctx, client, opts)
	require.NoError(t, err)
	result2, err := engine.ApplyPendingMigration(ctx, plan2, opts)
	require.NoError(t, err)
	assert.Equal(t, 0, result2.Applied)
	audits2, err := client.ProcessAuditLog.Query().Where(processauditlog.TenantIDEQ(tenant.ID)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, audits2)
}

// TestApplyPendingMigrationAssignsUnassignedTask 显式指定目标用户时改派未指派
// 待办，且不触碰实例状态；计划与请求租户不一致时 fail-closed。
func TestApplyPendingMigrationAssignsUnassignedTask(t *testing.T) {
	client := newInventoryTestClient(t, "mig_assign")
	ctx := context.Background()
	tenant, actor := setupInventoryTenant(ctx, t, client, "mig_assign")
	openTicket, err := client.Ticket.Create().SetTicketNumber("MIG-ASSIGN-1").SetRequesterID(1).
		SetTitle("处理中").SetStatus("in_progress").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	inst := createInventoryInstance(ctx, t, client, tenant.ID, "mig-assign-1", fmt.Sprintf("ticket:%d", openTicket.ID), "running")
	task := createInventoryTask(ctx, t, client, tenant.ID, inst.ID, "mig-assign-t", common.ProcessTaskStatusCreated, "", -1)

	opts := PendingMigrationOptions{
		TenantID: tenant.ID, ActorUserID: actor.ID, AssigneeID: actor.ID,
		Reason: "未指派待办改派验收",
	}
	plan, err := BuildPendingMigrationPlan(ctx, client, opts)
	require.NoError(t, err)
	assert.Equal(t, 1, plan.Summary.AssignTasks)
	assert.Equal(t, 0, plan.Summary.TerminateInstances)

	engine := NewCustomProcessEngine(client, zaptest.NewLogger(t).Sugar()).(*CustomProcessEngine)
	_, err = engine.ApplyPendingMigration(ctx, plan, PendingMigrationOptions{
		TenantID: tenant.ID + 999, ActorUserID: actor.ID, Reason: "租户不一致",
	})
	require.Error(t, err, "计划与请求租户不一致必须拒绝")

	result, err := engine.ApplyPendingMigration(ctx, plan, opts)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Applied)
	freshTask, err := client.ProcessTask.Get(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, strconv.Itoa(actor.ID), freshTask.Assignee)
	freshInst, err := client.ProcessInstance.Get(ctx, inst.ID)
	require.NoError(t, err)
	assert.Equal(t, "running", freshInst.Status)
}
