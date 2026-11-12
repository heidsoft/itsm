package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== 存量待办盘点测试 ====================
// 锁死 BuildPendingWorkflowInventory 的四类语义：
//  1. 待办聚合（按流程分组、停留时长、陈旧标记、未分配计数）；
//  2. 孤儿实例（业务终态 + 流程未结束）且不猜不存在业务对象的状态；
//  3. 租户隔离；
//  4. 只读边界：结果只含建议文本，不含任何写动作。

func newInventoryTestClient(t *testing.T, dbName string) *ent.Client {
	t.Helper()
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", dbName))
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func setupInventoryTenant(ctx context.Context, t *testing.T, client *ent.Client, suffix string) (*ent.Tenant, *ent.User) {
	t.Helper()
	tenant, err := client.Tenant.Create().
		SetName("Inventory Tenant " + suffix).
		SetCode("inv" + suffix).
		SetDomain("inv-" + suffix + ".example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	user, err := client.User.Create().
		SetUsername("inv-user-" + suffix).
		SetEmail("inv-user-" + suffix + "@example.com").
		SetName("Inventory User " + suffix).
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	return tenant, user
}

func createInventoryInstance(ctx context.Context, t *testing.T, client *ent.Client, tenantID int, suffix, businessKey, status string) *ent.ProcessInstance {
	t.Helper()
	// ProcessInstance 必填 process_definition_id：先部署一个最小定义。
	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-inv-" + suffix).
		SetDeploymentName("Inventory Deployment " + suffix).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	def, err := client.ProcessDefinition.Create().
		SetKey("inv_flow_" + suffix).
		SetName("Inventory Flow " + suffix).
		SetBpmnXML([]byte("<definitions/>")).
		SetDeploymentID(deployment.ID).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-inv-" + suffix).
		SetProcessDefinitionKey(def.Key).
		SetProcessDefinitionID(def.ID).
		SetBusinessKey(businessKey).
		SetStatus(status).
		SetStartTime(time.Now().Add(-48 * time.Hour)).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return instance
}

func createInventoryTask(ctx context.Context, t *testing.T, client *ent.Client, tenantID, instanceID int, suffix, status, assignee string, createdAgo time.Duration) *ent.ProcessTask {
	t.Helper()
	task, err := client.ProcessTask.Create().
		SetTaskID("TASK-inv-" + suffix).
		SetTaskDefinitionKey("Activity_" + suffix).
		SetTaskName("待办 " + suffix).
		SetTaskType("user_task").
		SetProcessDefinitionKey("release_approval_flow").
		SetProcessInstanceID(instanceID).
		SetAssignee(assignee).
		SetStatus(status).
		SetCreatedTime(time.Now().Add(-createdAgo)).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return task
}

// TestBuildPendingWorkflowInventory_EmptyTenant 空租户：全零 + 明确建议，不得伪造积压。
func TestBuildPendingWorkflowInventory_EmptyTenant(t *testing.T) {
	client := newInventoryTestClient(t, "inv_empty")
	ctx := context.Background()
	tenant, _ := setupInventoryTenant(ctx, t, client, "empty")

	inv, err := BuildPendingWorkflowInventory(ctx, client, tenant.ID, 0)
	require.NoError(t, err)
	require.NotNil(t, inv)

	assert.Equal(t, tenant.ID, inv.TenantID)
	assert.Equal(t, DefaultStaleThresholdHours, inv.StaleThresholdHours, "阈值缺省必须取默认 24h")
	assert.Equal(t, 0, inv.Summary.PendingTasks)
	assert.Equal(t, 0, inv.Summary.RunningInstances)
	assert.Equal(t, 0, inv.Summary.OrphanInstances)
	assert.Empty(t, inv.PendingTasks)
	assert.Empty(t, inv.OrphanInstances)
	require.NotEmpty(t, inv.Recommendations)
	assert.Contains(t, inv.Recommendations[0], "无需迁移")
	assert.Contains(t, inv.Recommendations[len(inv.Recommendations)-1], "只读盘点")
}

// TestBuildPendingWorkflowInventory_PendingAggregation 待办聚合：
// 按流程分组、停留时长、陈旧标记、未分配计数、截断不丢总数。
func TestBuildPendingWorkflowInventory_PendingAggregation(t *testing.T) {
	client := newInventoryTestClient(t, "inv_pending")
	ctx := context.Background()
	tenant, user := setupInventoryTenant(ctx, t, client, "pending")

	instance := createInventoryInstance(ctx, t, client, tenant.ID, "p1", "ticket:1", "running")

	// 陈旧已分配（30h > 24h 阈值）。
	createInventoryTask(ctx, t, client, tenant.ID, instance.ID, "stale", common.ProcessTaskStatusAssigned, fmt.Sprint(user.ID), 30*time.Hour)
	// 新鲜未分配（2h < 24h）。
	createInventoryTask(ctx, t, client, tenant.ID, instance.ID, "fresh", common.ProcessTaskStatusCreated, "", 2*time.Hour)
	// 已完成任务不得计入待办。
	createInventoryTask(ctx, t, client, tenant.ID, instance.ID, "done", common.ProcessTaskStatusCompleted, fmt.Sprint(user.ID), 30*time.Hour)

	inv, err := BuildPendingWorkflowInventory(ctx, client, tenant.ID, 24)
	require.NoError(t, err)

	assert.Equal(t, 2, inv.Summary.PendingTasks, "已完成任务不是待办")
	assert.Equal(t, 1, inv.Summary.StaleTasks)
	assert.Equal(t, 1, inv.Summary.UnassignedTasks)
	assert.Equal(t, 1, inv.Summary.RunningInstances)

	require.Len(t, inv.PendingTasks, 2)
	// 最旧排前（积压优先）。
	assert.Equal(t, "TASK-inv-stale", inv.PendingTasks[0].TaskID)
	assert.True(t, inv.PendingTasks[0].Stale)
	assert.GreaterOrEqual(t, inv.PendingTasks[0].DwellHours, 29.0)
	assert.False(t, inv.PendingTasks[1].Stale)
	assert.Empty(t, inv.PendingTasks[1].Assignee)

	require.Len(t, inv.ProcessGroups, 1)
	assert.Equal(t, "release_approval_flow", inv.ProcessGroups[0].ProcessKey)
	assert.Equal(t, 2, inv.ProcessGroups[0].PendingTasks)
	assert.Equal(t, 1, inv.ProcessGroups[0].StaleTasks)

	// 陈旧/未分配建议必须出现。
	joined := fmt.Sprint(inv.Recommendations)
	assert.Contains(t, joined, "停留超过")
	assert.Contains(t, joined, "没有处理人")
}

// TestBuildPendingWorkflowInventory_OrphanDetection 孤儿识别：
// 业务已终态 + 流程 running → 孤儿；业务未终态 → 不是孤儿；
// 业务对象不存在 → 不猜状态、不判孤儿；未知前缀 → 跳过。
func TestBuildPendingWorkflowInventory_OrphanDetection(t *testing.T) {
	client := newInventoryTestClient(t, "inv_orphan")
	ctx := context.Background()
	tenant, _ := setupInventoryTenant(ctx, t, client, "orphan")

	// 孤儿：ticket 已 closed，实例仍 running。
	closedTicket, err := client.Ticket.Create().
		SetTicketNumber("INV-CLOSED-1").
		SetRequesterID(1).
		SetTitle("已关闭工单").
		SetStatus("closed").
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	createInventoryInstance(ctx, t, client, tenant.ID, "orphan1", fmt.Sprintf("ticket:%d", closedTicket.ID), "running")

	// 非孤儿：ticket 仍 in_progress。
	openTicket, err := client.Ticket.Create().
		SetTicketNumber("INV-OPEN-1").
		SetRequesterID(1).
		SetTitle("处理中工单").
		SetStatus("in_progress").
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	createInventoryInstance(ctx, t, client, tenant.ID, "open1", fmt.Sprintf("ticket:%d", openTicket.ID), "running")

	// 非孤儿：业务对象已物理删（不存在），不得猜状态。
	createInventoryInstance(ctx, t, client, tenant.ID, "ghost", "ticket:999999", "running")

	// 非孤儿：release 终态 completed + suspended 实例仍是孤儿（未结束即算）。
	rel, err := client.Release.Create().
		SetReleaseNumber("REL-INV-1").
		SetTitle("已完成发布").
		SetStatus("completed").
		SetCreatedBy(1).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	createInventoryInstance(ctx, t, client, tenant.ID, "orphan2", fmt.Sprintf("release:%d", rel.ID), "suspended")

	// 未知前缀（存量 approval:<id> 键）：跳过，不计孤儿也不报错。
	createInventoryInstance(ctx, t, client, tenant.ID, "legacy", "approval:1", "running")

	inv, err := BuildPendingWorkflowInventory(ctx, client, tenant.ID, 24)
	require.NoError(t, err)

	assert.Equal(t, 5, inv.Summary.RunningInstances, "running + suspended 都计入未结束实例")
	require.Len(t, inv.OrphanInstances, 2, "仅 closed ticket 与 completed release 两个实例是孤儿")

	byKey := map[string]OrphanInstanceItem{}
	for _, orphan := range inv.OrphanInstances {
		byKey[orphan.BusinessKey] = orphan
	}
	ticketOrphan, ok := byKey[fmt.Sprintf("ticket:%d", closedTicket.ID)]
	require.True(t, ok)
	assert.Equal(t, "closed", ticketOrphan.BusinessStatus)
	assert.Equal(t, "running", ticketOrphan.InstanceStatus)
	assert.Contains(t, ticketOrphan.Reason, "终态")
	releaseOrphan, ok := byKey[fmt.Sprintf("release:%d", rel.ID)]
	require.True(t, ok)
	assert.Equal(t, "completed", releaseOrphan.BusinessStatus)
	assert.Equal(t, "suspended", releaseOrphan.InstanceStatus)

	// 建议必须指向人工终结端点，且不包含任何自动迁移语义。
	joined := fmt.Sprint(inv.Recommendations)
	assert.Contains(t, joined, "孤儿实例")
	assert.Contains(t, joined, "terminate")
	assert.Contains(t, joined, "只读盘点")
}

// TestBuildPendingWorkflowInventory_TenantIsolation 租户隔离：
// 他租户的待办/实例/业务对象都不得进入本租户盘点。
func TestBuildPendingWorkflowInventory_TenantIsolation(t *testing.T) {
	client := newInventoryTestClient(t, "inv_tenant")
	ctx := context.Background()
	tenantA, userA := setupInventoryTenant(ctx, t, client, "ta")
	tenantB, _ := setupInventoryTenant(ctx, t, client, "tb")

	instanceB := createInventoryInstance(ctx, t, client, tenantB.ID, "tb1", "ticket:1", "running")
	createInventoryTask(ctx, t, client, tenantB.ID, instanceB.ID, "tb1", common.ProcessTaskStatusAssigned, fmt.Sprint(userA.ID), 30*time.Hour)
	// 租户 B 的已关闭工单 + running 实例（A 的盘点不得看到他）。
	closedTicketB, err := client.Ticket.Create().
		SetTicketNumber("INV-B-CLOSED").
		SetRequesterID(1).
		SetTitle("B 租户已关闭工单").
		SetStatus("closed").
		SetTenantID(tenantB.ID).
		Save(ctx)
	require.NoError(t, err)
	createInventoryInstance(ctx, t, client, tenantB.ID, "tb2", fmt.Sprintf("ticket:%d", closedTicketB.ID), "running")

	inv, err := BuildPendingWorkflowInventory(ctx, client, tenantA.ID, 24)
	require.NoError(t, err)
	assert.Equal(t, 0, inv.Summary.PendingTasks)
	assert.Equal(t, 0, inv.Summary.RunningInstances)
	assert.Equal(t, 0, inv.Summary.OrphanInstances)
	assert.Empty(t, inv.PendingTasks)
	assert.Empty(t, inv.OrphanInstances)
}

// TestBuildPendingWorkflowInventory_InvalidInputs 输入校验：
// 无 client / 非法租户必须报错（fail-closed），不得返回空盘点冒充健康。
func TestBuildPendingWorkflowInventory_InvalidInputs(t *testing.T) {
	ctx := context.Background()
	_, err := BuildPendingWorkflowInventory(ctx, nil, 1, 24)
	require.Error(t, err)

	client := newInventoryTestClient(t, "inv_invalid")
	_, err = BuildPendingWorkflowInventory(ctx, client, 0, 24)
	require.Error(t, err)
}

// TestBuildPendingWorkflowInventory_ThresholdClamp 阈值钳制：
// 超大阈值按 720h 封顶，避免误传把一切标记为新鲜。
func TestBuildPendingWorkflowInventory_ThresholdClamp(t *testing.T) {
	client := newInventoryTestClient(t, "inv_clamp")
	ctx := context.Background()
	tenant, _ := setupInventoryTenant(ctx, t, client, "clamp")

	inv, err := BuildPendingWorkflowInventory(ctx, client, tenant.ID, 99999)
	require.NoError(t, err)
	assert.Equal(t, 720, inv.StaleThresholdHours)
}

// TestParseBusinessKey 契约：businessKey 解析只认 `<type>:<正整数>`，
// 其余形态一律 ok=false（调用方跳过而非误判）。
func TestParseBusinessKey(t *testing.T) {
	cases := []struct {
		key      string
		wantType string
		wantID   int
		wantOK   bool
	}{
		{"ticket:12", "ticket", 12, true},
		{"release:7", "release", 7, true},
		{"Ticket:12", "ticket", 12, true},
		{" ticket : 12 ", "ticket", 12, true},
		{"approval:1", "approval", 1, true},
		{"ticket:0", "", 0, false},
		{"ticket:-1", "", 0, false},
		{"ticket:abc", "", 0, false},
		{"ticket", "", 0, false},
		{"", "", 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			bizType, bizID, ok := parseBusinessKey(tc.key)
			assert.Equal(t, tc.wantOK, ok)
			if tc.wantOK {
				assert.Equal(t, tc.wantType, bizType)
				assert.Equal(t, tc.wantID, bizID)
			}
		})
	}
}

// TestIsTerminalBusinessStatus 契约：终态判定按域隔离，
// 未登记域一律非终态（fail-open 的反面：不误判孤儿）。
func TestIsTerminalBusinessStatus(t *testing.T) {
	assert.True(t, isTerminalBusinessStatus("ticket", "closed"))
	assert.True(t, isTerminalBusinessStatus("ticket", "cancelled"))
	assert.False(t, isTerminalBusinessStatus("ticket", "resolved"), "resolved 仍可关闭，不是终态")
	assert.True(t, isTerminalBusinessStatus("incident", "cancelled"))
	assert.True(t, isTerminalBusinessStatus("problem", "closed"))
	assert.False(t, isTerminalBusinessStatus("problem", "resolved"))
	assert.True(t, isTerminalBusinessStatus("change", "closed"))
	assert.True(t, isTerminalBusinessStatus("change", "rejected"))
	assert.False(t, isTerminalBusinessStatus("change", "completed"), "completed 仍可归档为 closed")
	assert.True(t, isTerminalBusinessStatus("release", "rolled_back"))
	assert.False(t, isTerminalBusinessStatus("release", "failed"), "failed 可重新排期，不是终态")
	assert.True(t, isTerminalBusinessStatus("service_request", "delivered"))
	assert.False(t, isTerminalBusinessStatus("approval", "closed"), "未登记域不得判终态")
}

// TestRoundHours 停留时长取整到一位小数，消除浮点噪声。
func TestRoundHours(t *testing.T) {
	assert.InDelta(t, 23.9, roundHours(23.94), 0.001)
	assert.InDelta(t, 24.0, roundHours(24.0), 0.001)
	assert.InDelta(t, 0.0, roundHours(-1), 0.001)
}

// TestLoadBusinessStatus_NotFound 业务对象不存在时 found=false，
// 且不返回错误（盘点不得因存量脏数据整体失败）。
func TestLoadBusinessStatus_NotFound(t *testing.T) {
	client := newInventoryTestClient(t, "inv_notfound")
	ctx := context.Background()
	tenant, _ := setupInventoryTenant(ctx, t, client, "nf")

	status, found, err := loadBusinessStatus(ctx, client, "ticket", 424242, tenant.ID)
	require.NoError(t, err)
	assert.False(t, found)
	assert.Empty(t, status)

	// 未登记业务域：同样 (空, false, nil)。
	status, found, err = loadBusinessStatus(ctx, client, "approval", 1, tenant.ID)
	require.NoError(t, err)
	assert.False(t, found)
	assert.Empty(t, status)
}

// TestLoadBusinessStatus_CrossTenant 跨租户业务对象不得可见（found=false）。
func TestLoadBusinessStatus_CrossTenant(t *testing.T) {
	client := newInventoryTestClient(t, "inv_cross")
	ctx := context.Background()
	tenantA, _ := setupInventoryTenant(ctx, t, client, "xa")
	tenantB, _ := setupInventoryTenant(ctx, t, client, "xb")

	ticketB, err := client.Ticket.Create().
		SetTicketNumber("INV-XB-1").
		SetRequesterID(1).
		SetTitle("B 租户工单").
		SetStatus("closed").
		SetTenantID(tenantB.ID).
		Save(ctx)
	require.NoError(t, err)

	status, found, err := loadBusinessStatus(ctx, client, "ticket", ticketB.ID, tenantA.ID)
	require.NoError(t, err)
	assert.False(t, found, "跨租户业务对象不得进入他租户盘点")
	assert.Empty(t, status)
}

// TestProcessInstanceOpenStatuses 常量契约：未结束状态集合只含 running/suspended，
// completed/terminated 均已结束（孤儿判定的分母）。
func TestProcessInstanceOpenStatuses(t *testing.T) {
	assert.ElementsMatch(t, []string{"running", "suspended"}, openInstanceStatuses)
	assert.NotContains(t, openInstanceStatuses, "completed")
	assert.NotContains(t, openInstanceStatuses, "terminated")
}
