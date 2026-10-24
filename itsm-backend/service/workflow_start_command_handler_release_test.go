package service

import (
	"context"
	"fmt"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/ent/processinstance"
	"itsm-backend/internal/commandbus"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ==================== release 域起流契约测试 ====================
// 锁死 WorkflowStartCommandHandler release 分支的语义：
// 发布创建落 CommandStartBPMN 命令后，处理器必须按默认绑定
// release → release_approval_flow 起流（与 bpmn_process_binding_service.go
// 默认绑定表一致），变量携带发布上下文，TriggeredBy 为创建人；
// 幂等：已存在 businessKey=release:<id> 实例时不得重复起流；
// 租户隔离：跨租户 release 视为未找到（fail-closed，不泄漏存在性）。

func setupReleaseWorkflowTest(t *testing.T) (*ent.Client, context.Context) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:rel_start_%d?mode=memory&cache=shared&_fk=1", nextRouteTestSeq()))
	ctx := context.Background()
	t.Cleanup(func() { _ = client.Close() })
	return client, ctx
}

func createReleaseWorkflowTenant(ctx context.Context, client *ent.Client, suffix string) (*ent.Tenant, error) {
	return client.Tenant.Create().
		SetName("Release Workflow Tenant " + suffix).
		SetCode("relwf" + suffix).
		SetDomain("relwf-" + suffix + ".example.com").
		SetStatus("active").
		Save(ctx)
}

func createReleaseWorkflowRelease(ctx context.Context, client *ent.Client, tenantID, createdBy int, suffix string) (*ent.Release, error) {
	routeTestSeqCounter++
	return client.Release.Create().
		SetReleaseNumber(fmt.Sprintf("REL-START-%d", routeTestSeqCounter)).
		SetTitle("起流契约测试发布 " + suffix).
		SetType("minor").
		SetSeverity("medium").
		SetEnvironment("staging").
		SetRequiresApproval(true).
		SetCreatedBy(createdBy).
		SetTenantID(tenantID).
		Save(ctx)
}

func createReleaseWorkflowCommand(ctx context.Context, client *ent.Client, tenantID, releaseID int) (*ent.OperationalCommand, error) {
	routeTestSeqCounter++
	return client.OperationalCommand.Create().
		SetTenantID(tenantID).
		SetCommandType(commandbus.CommandStartBPMN).
		SetAggregateType("release").
		SetAggregateID(releaseID).
		SetIdempotencyKey(fmt.Sprintf("release:%d:workflow:start", releaseID)).
		SetPayload(map[string]interface{}{"businessType": "release", "businessId": releaseID}).
		SetStatus(commandbus.StatusPending).
		Save(ctx)
}

// TestWorkflowStart_ReleaseUsesDefaultBindingFlow 发布命令必须起
// release_approval_flow，且变量/TriggeredBy 契约与 ApplyReleaseApproval 桥接匹配。
func TestWorkflowStart_ReleaseUsesDefaultBindingFlow(t *testing.T) {
	client, ctx := setupReleaseWorkflowTest(t)
	tenant, err := createReleaseWorkflowTenant(ctx, client, "default")
	require.NoError(t, err)

	creator, err := client.User.Create().
		SetUsername("relwf-creator-default").
		SetEmail("relwf-creator-default@example.com").
		SetName("Release Creator").
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	rel, err := createReleaseWorkflowRelease(ctx, client, tenant.ID, creator.ID, "default")
	require.NoError(t, err)
	cmd, err := createReleaseWorkflowCommand(ctx, client, tenant.ID, rel.ID)
	require.NoError(t, err)

	trigger := &fakeWorkflowTrigger{}
	handler := NewWorkflowStartCommandHandler(client, trigger, nil, zaptest.NewLogger(t).Sugar())

	require.NoError(t, handler.Handle(ctx, cmd))
	require.Len(t, trigger.reqs, 1)
	req := trigger.reqs[0]
	assert.Equal(t, "release_approval_flow", req.ProcessDefinitionKey)
	assert.Equal(t, dto.BusinessTypeRelease, req.BusinessType)
	assert.Equal(t, rel.ID, req.BusinessID)
	assert.Equal(t, tenant.ID, req.TenantID)
	// TriggeredBy 为创建人：与 release_service.triggerWorkflowForRelease 一致。
	assert.Equal(t, fmt.Sprintf("%d", creator.ID), req.TriggeredBy)
	// 变量契约：桥接与流程表达式依赖的键必须齐全。
	assert.Equal(t, rel.ID, req.Variables["release_id"])
	assert.Equal(t, rel.ReleaseNumber, req.Variables["release_number"])
	assert.Equal(t, rel.Title, req.Variables["title"])
	assert.Equal(t, "draft", req.Variables["status"])
	assert.Equal(t, creator.ID, req.Variables["created_by"])
	assert.Equal(t, true, req.Variables["requires_approval"])
}

// TestWorkflowStart_ReleaseMissingAggregateFails 命令指向不存在的发布时必须
// 报错（命令总线据此重试/进死信），不得静默跳过留下「发布已建、无流程」。
func TestWorkflowStart_ReleaseMissingAggregateFails(t *testing.T) {
	client, ctx := setupReleaseWorkflowTest(t)
	tenant, err := createReleaseWorkflowTenant(ctx, client, "missing")
	require.NoError(t, err)

	cmd, err := createReleaseWorkflowCommand(ctx, client, tenant.ID, 999999)
	require.NoError(t, err)

	trigger := &fakeWorkflowTrigger{}
	handler := NewWorkflowStartCommandHandler(client, trigger, nil, zaptest.NewLogger(t).Sugar())

	err = handler.Handle(ctx, cmd)
	require.Error(t, err)
	assert.Empty(t, trigger.reqs, "发布不存在时不得触发流程")
}

// TestWorkflowStart_ReleaseTenantIsolation 跨租户命令必须 fail-closed：
// 发布属于租户 B、命令租户 A 时视为未找到，不得跨租户起流。
func TestWorkflowStart_ReleaseTenantIsolation(t *testing.T) {
	client, ctx := setupReleaseWorkflowTest(t)
	tenantA, err := createReleaseWorkflowTenant(ctx, client, "isoA")
	require.NoError(t, err)
	tenantB, err := createReleaseWorkflowTenant(ctx, client, "isoB")
	require.NoError(t, err)

	creator, err := client.User.Create().
		SetUsername("relwf-creator-iso").
		SetEmail("relwf-creator-iso@example.com").
		SetName("ISO Creator").
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenantB.ID).
		Save(ctx)
	require.NoError(t, err)

	rel, err := createReleaseWorkflowRelease(ctx, client, tenantB.ID, creator.ID, "iso")
	require.NoError(t, err)
	// 命令以租户 A 的身份处理租户 B 的发布。
	cmd, err := createReleaseWorkflowCommand(ctx, client, tenantA.ID, rel.ID)
	require.NoError(t, err)

	trigger := &fakeWorkflowTrigger{}
	handler := NewWorkflowStartCommandHandler(client, trigger, nil, zaptest.NewLogger(t).Sugar())

	err = handler.Handle(ctx, cmd)
	require.Error(t, err)
	assert.Empty(t, trigger.reqs, "跨租户发布不得触发流程")
}

// TestWorkflowStart_ReleaseIdempotentWhenInstanceExists 已存在
// businessKey=release:<id> 的运行中实例时，重复命令必须 no-op（幂等重放）。
func TestWorkflowStart_ReleaseIdempotentWhenInstanceExists(t *testing.T) {
	client, ctx := setupReleaseWorkflowTest(t)
	tenant, err := createReleaseWorkflowTenant(ctx, client, "idem")
	require.NoError(t, err)

	creator, err := client.User.Create().
		SetUsername("relwf-creator-idem").
		SetEmail("relwf-creator-idem@example.com").
		SetName("Idem Creator").
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	rel, err := createReleaseWorkflowRelease(ctx, client, tenant.ID, creator.ID, "idem")
	require.NoError(t, err)

	// 预置一个 running 实例（businessKey 与处理器推导口径一致）。
	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID(fmt.Sprintf("DEP-relwf-idem-%d", rel.ID)).
		SetDeploymentName("Release Idem Deployment").
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	def, err := client.ProcessDefinition.Create().
		SetKey("release_approval_flow").
		SetName("发布审批流程").
		SetBpmnXML([]byte("<definitions/>")).
		SetDeploymentID(deployment.ID).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessInstance.Create().
		SetProcessInstanceID(fmt.Sprintf("PI-relwf-idem-%d", rel.ID)).
		SetProcessDefinitionKey(def.Key).
		SetProcessDefinitionID(def.ID).
		SetBusinessKey(fmt.Sprintf("release:%d", rel.ID)).
		SetStatus("running").
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	cmd, err := createReleaseWorkflowCommand(ctx, client, tenant.ID, rel.ID)
	require.NoError(t, err)

	trigger := &fakeWorkflowTrigger{}
	handler := NewWorkflowStartCommandHandler(client, trigger, nil, zaptest.NewLogger(t).Sugar())

	require.NoError(t, handler.Handle(ctx, cmd))
	assert.Empty(t, trigger.reqs, "已存在运行中实例时不得重复起流")
}

// TestReleaseService_CreateReleasePersistsWorkflowCommand 锁死 outbox 路径：
// 启用 outbox 后 CreateRelease 必须与业务写入同事务落 CommandStartBPMN 命令，
// 幂等键/Payload/AggregateType 与命令处理器 release 分支匹配。
func TestReleaseService_CreateReleasePersistsWorkflowCommand(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:rel_outbox_cmd?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenant, err := client.Tenant.Create().
		SetName("Release Outbox Tenant").
		SetCode("reloutbox").
		SetDomain("reloutbox.example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	creator, err := client.User.Create().
		SetUsername("rel-outbox-creator").
		SetEmail("rel-outbox-creator@example.com").
		SetName("Outbox Creator").
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	svc := NewReleaseService(client, logger)
	svc.EnableWorkflowOutbox()

	resp, err := svc.CreateRelease(ctx, &dto.CreateReleaseRequest{
		Title:       "外发箱起流测试发布",
		Description: "验证 outbox 命令落库",
		Type:        "minor",
		Environment: "staging",
		Severity:    "medium",
	}, creator.ID, tenant.ID)
	require.NoError(t, err)
	require.NotNil(t, resp)

	cmd, err := client.OperationalCommand.Query().Where(
		operationalcommand.TenantIDEQ(tenant.ID),
		operationalcommand.CommandTypeEQ(commandbus.CommandStartBPMN),
		operationalcommand.AggregateTypeEQ("release"),
	).Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, commandbus.CommandStartBPMN, cmd.CommandType)
	assert.Equal(t, "release", cmd.AggregateType)
	assert.Equal(t, resp.ID, cmd.AggregateID)
	assert.Equal(t, fmt.Sprintf("release:%d:workflow:start", resp.ID), cmd.IdempotencyKey)
	assert.Equal(t, tenant.ID, cmd.TenantID)
	assert.Equal(t, "pending", cmd.Status)
	assert.Equal(t, "release", cmd.Payload["businessType"])
	// Payload 经 JSON 列往返，数字形态为 float64（实例变量双形态陷阱），用 EqualValues 对齐。
	assert.EqualValues(t, resp.ID, cmd.Payload["businessId"])
}

// TestReleaseService_CreateReleaseWithoutOutboxDoesNotEnqueue 未启用 outbox 时
// 不得落命令（兼容路径只允许 goroutine 触发，不碰外发箱）。
func TestReleaseService_CreateReleaseWithoutOutboxDoesNotEnqueue(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:rel_nooutbox?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenant, err := client.Tenant.Create().
		SetName("Release NoOutbox Tenant").
		SetCode("relnooutbox").
		SetDomain("relnooutbox.example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	creator, err := client.User.Create().
		SetUsername("rel-nooutbox-creator").
		SetEmail("rel-nooutbox-creator@example.com").
		SetName("NoOutbox Creator").
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	svc := NewReleaseService(client, logger)
	// 未调用 EnableWorkflowOutbox，且未注入 trigger service：
	// 两条起流路径都不启用，只验证不落命令。

	resp, err := svc.CreateRelease(ctx, &dto.CreateReleaseRequest{
		Title: "无外发箱发布",
		Type:  "minor",
	}, creator.ID, tenant.ID)
	require.NoError(t, err)
	require.NotNil(t, resp)

	count, err := client.OperationalCommand.Query().Where(
		operationalcommand.TenantIDEQ(tenant.ID),
	).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "未启用 outbox 时不得落任何命令")
}

// TestReleaseService_OutboxCommandStartsApprovalFlow 端到端：outbox 命令经
// WorkflowStartCommandHandler + 真实 ProcessTriggerService 起流，产生
// businessKey=release:<id> 的 running 实例——把「发布已建」与「审批流在跑」
// 两件事在测试里合拢。
func TestReleaseService_OutboxCommandStartsApprovalFlow(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:rel_outbox_e2e?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenant, err := client.Tenant.Create().
		SetName("Release E2E Tenant").
		SetCode("rele2e").
		SetDomain("rele2e.example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	creator, err := client.User.Create().
		SetUsername("rel-e2e-creator").
		SetEmail("rel-e2e-creator@example.com").
		SetName("E2E Creator").
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 部署 release_approval_flow 定义（最小可执行子集：开始→技术评审→结束），
	// key 与默认绑定表一致。
	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-relwf-e2e").
		SetDeploymentName("Release E2E Deployment").
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.ProcessDefinition.Create().
		SetKey("release_approval_flow").
		SetName("发布审批流程").
		SetBpmnXML([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="release_approval_flow" name="发布审批流程" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="发布申请"/>
    <bpmn:userTask id="Activity_TechReview" name="技术评审"/>
    <bpmn:endEvent id="EndEvent_Complete" name="发布完成"/>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="Activity_TechReview"/>
    <bpmn:sequenceFlow id="Flow_Complete" sourceRef="Activity_TechReview" targetRef="EndEvent_Complete"/>
  </bpmn:process>
</bpmn:definitions>`)).
		SetDeploymentID(deployment.ID).
		SetTenantID(tenant.ID).
		SetIsActive(true).
		SetIsLatest(true).
		Save(ctx)
	require.NoError(t, err)

	svc := NewReleaseService(client, logger)
	svc.EnableWorkflowOutbox()

	resp, err := svc.CreateRelease(ctx, &dto.CreateReleaseRequest{
		Title:       "端到端起流发布",
		Description: "outbox 命令驱动审批流",
		Type:        "major",
		Environment: "production",
		Severity:    "high",
	}, creator.ID, tenant.ID)
	require.NoError(t, err)

	// 取出 outbox 落的命令并交给命令处理器执行（生产由命令总线轮询完成）。
	cmd, err := client.OperationalCommand.Query().Where(
		operationalcommand.CommandTypeEQ(commandbus.CommandStartBPMN),
		operationalcommand.AggregateTypeEQ("release"),
		operationalcommand.AggregateIDEQ(resp.ID),
	).Only(ctx)
	require.NoError(t, err)

	engine := NewCustomProcessEngine(client, logger)
	trigger := NewProcessTriggerService(client, engine)
	handler := NewWorkflowStartCommandHandler(client, trigger, nil, logger)
	require.NoError(t, handler.Handle(ctx, cmd))

	instance, err := client.ProcessInstance.Query().Where(
		processinstance.TenantIDEQ(tenant.ID),
		processinstance.BusinessKeyEQ(fmt.Sprintf("release:%d", resp.ID)),
	).Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, "release_approval_flow", instance.ProcessDefinitionKey)
	assert.Equal(t, "running", instance.Status)
	// 变量里的业务上下文（流程表达式/桥接依赖）必须随起流落库。
	// 实例变量同样经 JSON 列往返，数字形态为 float64，用 EqualValues 对齐。
	assert.EqualValues(t, resp.ID, instance.Variables["release_id"])
	assert.Equal(t, "draft", instance.Variables["status"])

	// 重复执行同一命令必须幂等（命令总线重试/重放不得产生第二个实例）。
	require.NoError(t, handler.Handle(ctx, cmd))
	count, err := client.ProcessInstance.Query().Where(
		processinstance.TenantIDEQ(tenant.ID),
		processinstance.BusinessKeyEQ(fmt.Sprintf("release:%d", resp.ID)),
	).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "重复命令不得产生第二个流程实例")
}
