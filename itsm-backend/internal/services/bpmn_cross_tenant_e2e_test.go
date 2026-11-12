package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/processauditlog"
	"itsm-backend/service/bpmn"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ==================== 跨租户 / 跨业务域实例级 E2E ====================
//
// 真实租户联调签字前的自动化证据：两个租户在同一个数据库里各自真实起流，
// 业务键覆盖工单域与问题域，验证本批四项收口在租户边界上互不串写：
//   1. ServiceTask 完成凭据 / receipt 按实例隔离；
//   2. 显式补偿触发跨租户 fail-closed，且不泄漏补偿范围与审计；
//   3. 存量待办迁移计划与 apply 按租户收窄，正常在途实例不被误伤；
//   4. 管理面操作全程带 tenantID 谓词，不做全局 ID 直查。
//
// 这里证明的是「代码层租户隔离」，不能替代真实生产租户的联调签字：
// 后者由 scripts/acceptance/workflow-cross-domain-acceptance.sh 产出证据、
// 由 docs/acceptance/workflow-cross-domain-signoff.md 的签字栏人工确认。

const crossTenantServiceTaskBPMN = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="Process_cross_tenant_service" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1"/>
    <bpmn:serviceTask id="ServiceTask_1" implementation="cross_tenant_handler"/>
    <bpmn:endEvent id="EndEvent_1"/>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="ServiceTask_1"/>
    <bpmn:sequenceFlow id="Flow_2" sourceRef="ServiceTask_1" targetRef="EndEvent_1"/>
  </bpmn:process>
</bpmn:definitions>`

type crossTenantFixture struct {
	client   *ent.Client
	engine   *CustomProcessEngine
	tenantA  *ent.Tenant
	tenantB  *ent.Tenant
	actorA   *ent.User
	actorB   *ent.User
	ctxA     context.Context
	ctxB     context.Context
	keyA     string
	keyB     string
	instance map[int]int // tenantID -> 真实起流的实例 ID
}

func setupCrossTenantFixture(t *testing.T, dbName string) *crossTenantFixture {
	t.Helper()
	client := newInventoryTestClient(t, dbName)
	ctx := context.Background()
	tenantA, actorA := setupInventoryTenant(ctx, t, client, dbName+"_a")
	tenantB, actorB := setupInventoryTenant(ctx, t, client, dbName+"_b")

	engine := NewCustomProcessEngine(client, zaptest.NewLogger(t).Sugar()).(*CustomProcessEngine)
	engine.callbackRegistry.RegisterHandler(&compensationHandler{id: "cross_tenant_handler"})

	f := &crossTenantFixture{
		client: client, engine: engine,
		tenantA: tenantA, tenantB: tenantB, actorA: actorA, actorB: actorB,
		ctxA:     context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenantA.ID),
		ctxB:     context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenantB.ID),
		instance: map[int]int{},
	}
	for _, tenant := range []*ent.Tenant{tenantA, tenantB} {
		key := fmt.Sprintf("crossTenantFlow_%d", tenant.ID)
		dep, err := client.ProcessDeployment.Create().
			SetDeploymentID("DEP-CROSS-" + key).SetDeploymentName(key).SetTenantID(tenant.ID).Save(ctx)
		require.NoError(t, err)
		_, err = client.ProcessDefinition.Create().
			SetKey(key).SetName(key).SetBpmnXML([]byte(crossTenantServiceTaskBPMN)).
			SetDeploymentID(dep.ID).SetTenantID(tenant.ID).
			SetIsActive(true).SetIsLatest(true).Save(ctx)
		require.NoError(t, err)
		if tenant.ID == tenantA.ID {
			f.keyA = key
		} else {
			f.keyB = key
		}
	}
	return f
}

// startForTenant 以指定租户身份真实起流（业务键覆盖工单/问题两个域）。
func (f *crossTenantFixture) startForTenant(t *testing.T, tenant *ent.Tenant, key, businessKey string) int {
	t.Helper()
	workflowCtx := context.WithValue(context.Background(), bpmn.BPMNTenantIDContextKey, tenant.ID)
	instance, err := f.engine.StartProcess(workflowCtx, key, businessKey, map[string]interface{}{})
	require.NoError(t, err)
	f.instance[tenant.ID] = instance.ID
	return instance.ID
}

func (f *crossTenantFixture) drainServiceTaskCommands(t *testing.T) {
	t.Helper()
	runServiceTaskCommands(t, context.Background(), f.client, f.engine, f.tenantA.ID)
}

// TestCrossTenantServiceTaskReceiptsAreIsolated 真实起流后，两个租户、
// 两个业务域的 ServiceTask 完成凭据与 receipt 各自落在自己的实例上；
// 用 A 的身份触发 B 的补偿必须 fail-closed，且 B 的补偿范围与审计零变化。
func TestCrossTenantServiceTaskReceiptsAreIsolated(t *testing.T) {
	f := setupCrossTenantFixture(t, "cross_receipt")
	ctx := context.Background()

	// A 走工单域、B 走问题域：跨业务域 + 跨租户同时覆盖。
	f.startForTenant(t, f.tenantA, f.keyA, "ticket:9001")
	f.startForTenant(t, f.tenantB, f.keyB, "problem:9002")
	f.drainServiceTaskCommands(t)

	for _, tenant := range []*ent.Tenant{f.tenantA, f.tenantB} {
		inst, err := f.client.ProcessInstance.Get(ctx, f.instance[tenant.ID])
		require.NoError(t, err)
		assert.Equal(t, tenant.ID, inst.TenantID)
		assert.Equal(t, "completed", inst.Status, "ServiceTask 完成后流程必须正常结束")
		receipts, ok := inst.Variables["_serviceTaskReceipts"].(map[string]interface{})
		require.True(t, ok, "同事务落库的副作用 receipt 必须存在")
		assert.Contains(t, receipts, "ServiceTask_1")
		assert.Equal(t, 1, intMapVariable(inst.Variables, "_serviceTaskExecutedOccurrences")["ServiceTask_1"])
		assert.NotEmpty(t, inst.Variables["_compensationRegistered"],
			"完成同事务必须登记补偿范围")
	}

	// 跨租户触发：A 的身份拿 B 的实例，必须当「不存在」处理。
	// B 自己起流会写 started / activity_completed 审计，因此这里比较的是
	// 「跨租户触发前后 B 名下审计条数不变」，而不是期望从零开始。
	auditsBefore, err := f.client.ProcessAuditLog.Query().Where(processauditlog.TenantIDEQ(f.tenantB.ID)).Count(ctx)
	require.NoError(t, err)
	_, err = f.engine.TriggerCompensation(f.ctxA, CompensationTriggerRequest{
		TenantID: f.tenantA.ID, ActorUserID: f.actorA.ID,
		InstanceID: f.instance[f.tenantB.ID], Reason: "跨租户边界探测",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不存在或不属于租户")

	instB, err := f.client.ProcessInstance.Get(ctx, f.instance[f.tenantB.ID])
	require.NoError(t, err)
	assert.Empty(t, intMapVariable(instB.Variables, "_compensationExecuted"),
		"跨租户触发不得消费 B 的补偿范围")
	auditsAfter, err := f.client.ProcessAuditLog.Query().Where(processauditlog.TenantIDEQ(f.tenantB.ID)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, auditsBefore, auditsAfter, "跨租户触发不得在 B 名下写审计")

	// 正确租户触发只影响自己；B 的变量保持原样。
	result, err := f.engine.TriggerCompensation(f.ctxA, CompensationTriggerRequest{
		TenantID: f.tenantA.ID, ActorUserID: f.actorA.ID,
		InstanceID: f.instance[f.tenantA.ID], Reason: "本租户复核发现需回滚",
	})
	require.NoError(t, err)
	assert.True(t, result.Applied)
	assert.Equal(t, []string{"ServiceTask_1"}, result.Targets)
	instBAgain, err := f.client.ProcessInstance.Get(ctx, f.instance[f.tenantB.ID])
	require.NoError(t, err)
	assert.Equal(t, instB.Variables["_compensationRegistered"], instBAgain.Variables["_compensationRegistered"])
	assert.Empty(t, intMapVariable(instBAgain.Variables, "_compensationExecuted"))
}

// TestCrossTenantPendingMigrationIsScoped 存量待办迁移按租户收窄：
// A 的孤儿实例被终结、B 的同构数据与正常在途实例一律不动，审计也不串租户。
func TestCrossTenantPendingMigrationIsScoped(t *testing.T) {
	f := setupCrossTenantFixture(t, "cross_migration")
	ctx := context.Background()

	type orphan struct {
		instanceID int
		taskID     int
	}
	closedPerTenant := map[int]orphan{}
	for i, tenant := range []*ent.Tenant{f.tenantA, f.tenantB} {
		ticket, err := f.client.Ticket.Create().
			SetTicketNumber(fmt.Sprintf("CROSS-CLOSED-%d", i)).SetRequesterID(1).
			SetTitle("已关闭工单").SetStatus("closed").SetTenantID(tenant.ID).Save(ctx)
		require.NoError(t, err)
		inst := createInventoryInstance(ctx, t, f.client, tenant.ID,
			fmt.Sprintf("cross-orphan-%d", i), fmt.Sprintf("ticket:%d", ticket.ID), "running")
		task := createInventoryTask(ctx, t, f.client, tenant.ID, inst.ID,
			fmt.Sprintf("cross-orphan-t-%d", i), common.ProcessTaskStatusCreated, "", 2*time.Hour)
		closedPerTenant[tenant.ID] = orphan{instanceID: inst.ID, taskID: task.ID}
	}

	opts := PendingMigrationOptions{
		TenantID: f.tenantA.ID, ActorUserID: f.actorA.ID, Reason: "跨租户迁移范围验收",
	}
	plan, err := BuildPendingMigrationPlan(ctx, f.client, opts)
	require.NoError(t, err)
	assert.Equal(t, 1, plan.Summary.OrphanInstances, "计划不得把 B 的孤儿实例算进来")
	assert.Equal(t, f.tenantA.ID, plan.TenantID)
	for _, item := range plan.Items {
		assert.NotEqual(t, closedPerTenant[f.tenantB.ID].instanceID, item.InstanceID)
	}

	_, err = f.engine.ApplyPendingMigration(ctx, plan, opts)
	require.NoError(t, err)

	instA, err := f.client.ProcessInstance.Get(ctx, closedPerTenant[f.tenantA.ID].instanceID)
	require.NoError(t, err)
	assert.Equal(t, "terminated", instA.Status)
	instB, err := f.client.ProcessInstance.Get(ctx, closedPerTenant[f.tenantB.ID].instanceID)
	require.NoError(t, err)
	assert.Equal(t, "running", instB.Status, "B 的孤儿实例不得被 A 的迁移顺手清理")
	taskB, err := f.client.ProcessTask.Get(ctx, closedPerTenant[f.tenantB.ID].taskID)
	require.NoError(t, err)
	assert.Equal(t, common.ProcessTaskStatusCreated, taskB.Status)
	auditsB, err := f.client.ProcessAuditLog.Query().Where(processauditlog.TenantIDEQ(f.tenantB.ID)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, auditsB, "迁移审计不得落到 B 名下")
}
