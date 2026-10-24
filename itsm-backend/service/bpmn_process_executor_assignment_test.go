package service

import (
	"context"
	"strconv"
	"testing"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/processtask"
)

func newAssignmentTestEnv(t *testing.T, name string) (*CustomProcessEngine, context.Context, *ent.Client, int, *ent.ProcessInstance) {
	t.Helper()
	client := enttest.Open(t, dialect.SQLite, "file:assignment_"+name+"?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	tenantID := createTimerTenant(t, client, "assign-"+name)
	defID := createTestProcessDefinition(t, client, tenantID, "assign-"+name, bizBPMNSimpleApproval)
	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID("PI-ASSIGN-" + name).
		SetProcessDefinitionKey("assign-" + name).
		SetProcessDefinitionID(defID).
		SetStatus("running").
		SetTenantID(tenantID).
		SetVariables(map[string]interface{}{}).
		Save(context.Background())
	require.NoError(t, err)

	engine := NewCustomProcessEngine(client, zaptest.NewLogger(t).Sugar()).(*CustomProcessEngine)
	return engine, context.Background(), client, tenantID, instance
}

func newAssignmentUser(t *testing.T, client *ent.Client, tenantID int, username string) *ent.User {
	t.Helper()
	u, err := client.User.Create().
		SetUsername(username).
		SetEmail(username + "@example.com").
		SetName(username).
		SetPasswordHash("test-hash").
		SetTenantID(tenantID).
		Save(context.Background())
	require.NoError(t, err)
	return u
}

func setAssignmentRequester(instance *ent.ProcessInstance, user *ent.User) {
	instance.Variables["requester_id"] = user.ID
}

func TestResolveTaskAssignmentDeclaredPoolBeatsRequesterFallback(t *testing.T) {
	engine, ctx, client, tenantID, instance := newAssignmentTestEnv(t, "pool-beats-requester")
	requester := newAssignmentUser(t, client, tenantID, "requester")
	agent := newAssignmentUser(t, client, tenantID, "agent")
	setAssignmentRequester(instance, requester)

	assignment := engine.resolveTaskAssignment(ctx, client, instance, &BPMNUserTask{
		ID: "Accept", Name: "受理", TaskPurpose: "work", CandidateUsers: agent.Username,
	})

	assert.Equal(t, taskAssignmentPool, assignment.state)
	assert.Empty(t, assignment.assignee, "声明候选池的任务不得被发起人兜底抢占")
	assert.Equal(t, agent.Username, assignment.candidateUsers)
	assert.Equal(t, "bpmn_candidates", assignment.source)
}

func TestResolveTaskAssignmentStaticAssigneeWins(t *testing.T) {
	engine, ctx, client, tenantID, instance := newAssignmentTestEnv(t, "static-wins")
	newAssignmentUser(t, client, tenantID, "requester")
	assigned := newAssignmentUser(t, client, tenantID, "assigned")
	other := newAssignmentUser(t, client, tenantID, "other")

	assignment := engine.resolveTaskAssignment(ctx, client, instance, &BPMNUserTask{
		ID: "Review", Name: "复核", TaskPurpose: "work",
		Assignee: assigned.Username, CandidateUsers: other.Username,
	})

	assert.Equal(t, taskAssignmentAssigned, assignment.state)
	assert.Equal(t, assigned.Username, assignment.assignee)
	assert.Equal(t, "bpmn_assignee", assignment.source)
}

func TestResolveTaskAssignmentApprovalNeverFallsBackToRequester(t *testing.T) {
	engine, ctx, client, tenantID, instance := newAssignmentTestEnv(t, "approval-no-fallback")
	requester := newAssignmentUser(t, client, tenantID, "requester")
	setAssignmentRequester(instance, requester)

	assignment := engine.resolveTaskAssignment(ctx, client, instance, &BPMNUserTask{
		ID: "Approval_L1", Name: "经理审批", TaskPurpose: "approval",
	})

	assert.Equal(t, taskAssignmentUnassigned, assignment.state, "审批任务没有审批人时必须待指派")
	assert.Empty(t, assignment.assignee)
	assert.Equal(t, "no_approver_available", assignment.reason)
}

func TestResolveTaskAssignmentApprovalFiltersRequesterFromPool(t *testing.T) {
	engine, ctx, client, tenantID, instance := newAssignmentTestEnv(t, "approval-filter-pool")
	requester := newAssignmentUser(t, client, tenantID, "requester")
	manager := newAssignmentUser(t, client, tenantID, "manager")
	setAssignmentRequester(instance, requester)

	assignment := engine.resolveTaskAssignment(ctx, client, instance, &BPMNUserTask{
		ID: "Approval_L1", Name: "经理审批", TaskPurpose: "approval",
		CandidateUsers: requester.Username + "," + manager.Username,
	})

	assert.Equal(t, taskAssignmentPool, assignment.state)
	assert.Empty(t, assignment.assignee)
	assert.Equal(t, manager.Username, assignment.candidateUsers)
	assert.Equal(t, "requester_filtered_from_pool", assignment.reason)
}

func TestResolveTaskAssignmentApprovalFiltersRequesterStaticAssignee(t *testing.T) {
	engine, ctx, client, tenantID, instance := newAssignmentTestEnv(t, "approval-filter-static")
	requester := newAssignmentUser(t, client, tenantID, "requester")
	setAssignmentRequester(instance, requester)

	assignment := engine.resolveTaskAssignment(ctx, client, instance, &BPMNUserTask{
		ID: "Approval_L1", Name: "经理审批", TaskPurpose: "approval",
		Assignee: requester.Username,
	})

	assert.Equal(t, taskAssignmentUnassigned, assignment.state)
	assert.Empty(t, assignment.assignee)
	assert.Equal(t, "self_approval_filtered", assignment.reason)
}

func TestResolveTaskAssignmentApprovalFiltersRequesterFromGroup(t *testing.T) {
	engine, ctx, client, tenantID, instance := newAssignmentTestEnv(t, "approval-filter-group")
	requester := newAssignmentUser(t, client, tenantID, "requester")
	manager := newAssignmentUser(t, client, tenantID, "manager")
	setAssignmentRequester(instance, requester)

	_, err := client.Group.Create().
		SetName("manager").
		SetTenantID(tenantID).
		AddMemberIDs(requester.ID, manager.ID).
		Save(ctx)
	require.NoError(t, err)

	assignment := engine.resolveTaskAssignment(ctx, client, instance, &BPMNUserTask{
		ID: "Approval_L1", Name: "经理审批", TaskPurpose: "approval",
		CandidateGroups: "manager",
	})

	assert.Equal(t, taskAssignmentPool, assignment.state)
	assert.Empty(t, assignment.assignee)
	assert.Equal(t, manager.Username, assignment.candidateUsers)
	assert.Equal(t, "requester_filtered_from_pool", assignment.reason)
}

func TestResolveTaskAssignmentWorkFallsBackToRequester(t *testing.T) {
	engine, ctx, client, tenantID, instance := newAssignmentTestEnv(t, "work-requester")
	requester := newAssignmentUser(t, client, tenantID, "requester")
	setAssignmentRequester(instance, requester)

	assignment := engine.resolveTaskAssignment(ctx, client, instance, &BPMNUserTask{
		ID: "Confirm", Name: "申请人确认", TaskPurpose: "work",
	})

	assert.Equal(t, taskAssignmentAssigned, assignment.state)
	assert.Equal(t, strconv.Itoa(requester.ID), assignment.assignee)
	assert.Equal(t, "requester_id", assignment.source)
}

func TestResolveTaskAssignmentReworkFallsBackToRequester(t *testing.T) {
	engine, ctx, client, tenantID, instance := newAssignmentTestEnv(t, "rework-requester")
	requester := newAssignmentUser(t, client, tenantID, "requester")
	setAssignmentRequester(instance, requester)

	assignment := engine.resolveTaskAssignment(ctx, client, instance, &BPMNUserTask{
		ID: "Rework", Name: "返工", TaskPurpose: "rework",
	})

	assert.Equal(t, taskAssignmentAssigned, assignment.state)
	assert.Equal(t, strconv.Itoa(requester.ID), assignment.assignee)
}

func TestCreateUserTaskPersistsUnassignedMarker(t *testing.T) {
	engine, ctx, _, _, instance := newAssignmentTestEnv(t, "persist-unassigned")
	requesterUser := newAssignmentUser(t, engine.client, instance.TenantID, "requester")
	setAssignmentRequester(instance, requesterUser)

	require.NoError(t, engine.createUserTask(ctx, engine.client, instance, &BPMNProcess{}, &BPMNUserTask{
		ID: "Approval_L1", Name: "经理审批", TaskPurpose: "approval",
	}))

	stored, err := engine.client.ProcessTask.Query().
		Where(processtask.TaskDefinitionKey("Approval_L1")).
		Only(ctx)
	require.NoError(t, err)
	assert.Empty(t, stored.Assignee)
	assert.Equal(t, taskAssignmentUnassigned, stored.TaskVariables["assignmentState"])
	assert.Equal(t, "none", stored.TaskVariables["assignmentSource"])
	assert.Equal(t, "no_approver_available", stored.TaskVariables["assignmentReason"])

	nilAssignee, err := engine.client.ProcessTask.Query().
		Where(processtask.AssigneeIsNil()).
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, nilAssignee, "未指派任务的 assignee 必须是 NULL，认领路径依赖 AssigneeIsNil")
}
