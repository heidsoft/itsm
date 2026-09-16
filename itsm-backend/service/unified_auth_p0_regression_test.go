package service

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"
	"itsm-backend/service/bpmn"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ---------------------------------------------------------------------------
// Step 1: CanGrantRoles — rank check for RoleIDs
// ---------------------------------------------------------------------------

func TestCanGrantRoles_RejectsHigherRank(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:can_grant_roles?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()
	svc := NewUserService(client, logger)

	tenant, err := client.Tenant.Create().
		SetName("T").SetCode("CGR").SetDomain("cgr.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	adminRole, err := client.Role.Create().SetCode("admin").SetName("管理员").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	// manager (rank 3) trying to grant admin (rank 4) should fail
	err = svc.CanGrantRoles(ctx, tenant.ID, []int{adminRole.ID}, "manager")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "无权限分配高于自身角色")
}

func TestCanGrantRoles_AllowsSameOrLowerRank(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:can_grant_roles_ok?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()
	svc := NewUserService(client, logger)

	tenant, err := client.Tenant.Create().
		SetName("T").SetCode("CGROK").SetDomain("cgrok.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	agentRole, err := client.Role.Create().SetCode("agent").SetName("坐席").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	// manager (rank 3) granting agent (rank 2) should succeed
	err = svc.CanGrantRoles(ctx, tenant.ID, []int{agentRole.ID}, "manager")
	require.NoError(t, err)
}

func TestCanGrantRoles_RejectsCrossTenantRole(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:can_grant_roles_cross?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()
	svc := NewUserService(client, logger)

	tenantA, err := client.Tenant.Create().
		SetName("A").SetCode("CGRA").SetDomain("cgra.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	tenantB, err := client.Tenant.Create().
		SetName("B").SetCode("CGRB").SetDomain("cgrb.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	roleInB, err := client.Role.Create().SetCode("agent").SetName("坐席").SetTenantID(tenantB.ID).Save(ctx)
	require.NoError(t, err)

	// Trying to grant a role from tenant B while operating in tenant A should fail
	err = svc.CanGrantRoles(ctx, tenantA.ID, []int{roleInB.ID}, "admin")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不属于当前租户")
}

// ---------------------------------------------------------------------------
// Step 4: DelegateTask — actor + target user validation
// ---------------------------------------------------------------------------

func TestDelegateTask_RejectsNonAssigneeActor(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:delegate_non_assignee?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	logger := zaptest.NewLogger(t).Sugar()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("T").SetCode("DNA").SetDomain("dna.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	assignee, err := client.User.Create().
		SetUsername("assignee").SetEmail("a@test.com").SetName("A").
		SetPasswordHash("h").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	otherUser, err := client.User.Create().
		SetUsername("other").SetEmail("o@test.com").SetName("O").
		SetPasswordHash("h").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	delegatee, err := client.User.Create().
		SetUsername("delegatee").SetEmail("d@test.com").SetName("D").
		SetPasswordHash("h").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	_ = delegatee

	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-DNA").SetDeploymentName("D").SetIsActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	def, err := client.ProcessDefinition.Create().
		SetKey("dna").SetName("D").SetVersion("1").SetIsLatest(true).SetIsActive(true).
		SetBpmnXML([]byte("<bpmn/>")).SetDeploymentID(deployment.ID).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)
	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-DNA").SetProcessDefinitionKey(def.Key).SetProcessDefinitionID(def.ID).
		SetStatus("running").SetVariables(map[string]interface{}{}).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.ProcessTask.Create().
		SetTaskID("TASK-DNA-1").SetTaskDefinitionKey("T").SetTaskName("T").
		SetProcessDefinitionKey(def.Key).SetProcessInstanceID(instance.ID).
		SetStatus("assigned").SetAssignee(assignee.Username).SetTenantID(tenant.ID).
		SetTaskVariables(map[string]interface{}{"allowDelegate": true}).
		Save(ctx)
	require.NoError(t, err)

	// Actor is otherUser, not the assignee — should be rejected
	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenant.ID)
	workflowCtx = context.WithValue(workflowCtx, bpmn.BPMNUserIDContextKey, otherUser.ID)

	err = engine.TaskService().DelegateTask(workflowCtx, "TASK-DNA-1", "delegatee")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "只有当前处理人才能委托")
}

func TestDelegateTask_RejectsForeignTargetUser(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:delegate_foreign_target?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	logger := zaptest.NewLogger(t).Sugar()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	ctx := context.Background()

	tenantA, err := client.Tenant.Create().
		SetName("A").SetCode("DFTA").SetDomain("dfta.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	tenantB, err := client.Tenant.Create().
		SetName("B").SetCode("DFTB").SetDomain("dftb.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	assignee, err := client.User.Create().
		SetUsername("assignee").SetEmail("a@test.com").SetName("A").
		SetPasswordHash("h").SetRole("agent").SetActive(true).SetTenantID(tenantA.ID).
		Save(ctx)
	require.NoError(t, err)

	// Target user belongs to tenant B, not tenant A
	foreignTarget, err := client.User.Create().
		SetUsername("foreign").SetEmail("f@test.com").SetName("F").
		SetPasswordHash("h").SetRole("agent").SetActive(true).SetTenantID(tenantB.ID).
		Save(ctx)
	require.NoError(t, err)

	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-DFT").SetDeploymentName("D").SetIsActive(true).SetTenantID(tenantA.ID).
		Save(ctx)
	require.NoError(t, err)
	def, err := client.ProcessDefinition.Create().
		SetKey("dft").SetName("D").SetVersion("1").SetIsLatest(true).SetIsActive(true).
		SetBpmnXML([]byte("<bpmn/>")).SetDeploymentID(deployment.ID).SetTenantID(tenantA.ID).
		Save(ctx)
	require.NoError(t, err)
	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-DFT").SetProcessDefinitionKey(def.Key).SetProcessDefinitionID(def.ID).
		SetStatus("running").SetVariables(map[string]interface{}{}).SetTenantID(tenantA.ID).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.ProcessTask.Create().
		SetTaskID("TASK-DFT-1").SetTaskDefinitionKey("T").SetTaskName("T").
		SetProcessDefinitionKey(def.Key).SetProcessInstanceID(instance.ID).
		SetStatus("assigned").SetAssignee(assignee.Username).SetTenantID(tenantA.ID).
		SetTaskVariables(map[string]interface{}{"allowDelegate": true}).
		Save(ctx)
	require.NoError(t, err)

	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenantA.ID)
	workflowCtx = context.WithValue(workflowCtx, bpmn.BPMNUserIDContextKey, assignee.ID)

	// Using foreign user's ID as target
	err = engine.TaskService().DelegateTask(workflowCtx, "TASK-DFT-1", foreignTarget.Username)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "委托目标用户校验失败")
}

// ---------------------------------------------------------------------------
// Step 5: AssignTask — actor + target user validation
// ---------------------------------------------------------------------------

func TestAssignTask_RejectsMissingTenantContext(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:assign_no_tenant?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	logger := zaptest.NewLogger(t).Sugar()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)

	// No tenant context — should fail
	err := engine.TaskService().AssignTask(context.Background(), "TASK-X", "someone")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "租户")
}

func TestAssignTask_RejectsCrossTenantTarget(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:assign_cross_tenant?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	logger := zaptest.NewLogger(t).Sugar()
	engine := NewCustomProcessEngine(client, logger).(*CustomProcessEngine)
	ctx := context.Background()

	tenantA, err := client.Tenant.Create().
		SetName("A").SetCode("ACTA").SetDomain("acta.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	tenantB, err := client.Tenant.Create().
		SetName("B").SetCode("ACTB").SetDomain("actb.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	actor, err := client.User.Create().
		SetUsername("actor").SetEmail("actor@test.com").SetName("Actor").
		SetPasswordHash("h").SetRole("manager").SetActive(true).SetTenantID(tenantA.ID).
		Save(ctx)
	require.NoError(t, err)

	foreignUser, err := client.User.Create().
		SetUsername("foreign").SetEmail("foreign@test.com").SetName("F").
		SetPasswordHash("h").SetRole("agent").SetActive(true).SetTenantID(tenantB.ID).
		Save(ctx)
	require.NoError(t, err)

	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("DEP-ACT").SetDeploymentName("D").SetIsActive(true).SetTenantID(tenantA.ID).
		Save(ctx)
	require.NoError(t, err)
	def, err := client.ProcessDefinition.Create().
		SetKey("act").SetName("D").SetVersion("1").SetIsLatest(true).SetIsActive(true).
		SetBpmnXML([]byte("<bpmn/>")).SetDeploymentID(deployment.ID).SetTenantID(tenantA.ID).
		Save(ctx)
	require.NoError(t, err)
	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-ACT").SetProcessDefinitionKey(def.Key).SetProcessDefinitionID(def.ID).
		SetStatus("running").SetVariables(map[string]interface{}{}).SetTenantID(tenantA.ID).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.ProcessTask.Create().
		SetTaskID("TASK-ACT-1").SetTaskDefinitionKey("T").SetTaskName("T").
		SetProcessDefinitionKey(def.Key).SetProcessInstanceID(instance.ID).
		SetStatus("created").SetTenantID(tenantA.ID).
		Save(ctx)
	require.NoError(t, err)

	workflowCtx := context.WithValue(ctx, bpmn.BPMNTenantIDContextKey, tenantA.ID)
	workflowCtx = context.WithValue(workflowCtx, bpmn.BPMNUserIDContextKey, actor.ID)

	err = engine.TaskService().AssignTask(workflowCtx, "TASK-ACT-1", foreignUser.Username)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "分配目标用户校验失败")
}
