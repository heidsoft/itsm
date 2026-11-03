package service_request

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent"
	"itsm-backend/ent/auditlog"
	"itsm-backend/ent/enttest"
)

type recordedNotification struct {
	title    string
	message  string
	kind     string
	userIDs  []int
	tenantID int
}

type recordingNotifier struct {
	calls []recordedNotification
}

func (n *recordingNotifier) CreateSystemNotification(_ context.Context, title, message, notificationType string, userIDs []int, tenantID int) error {
	n.calls = append(n.calls, recordedNotification{
		title: title, message: message, kind: notificationType, userIDs: userIDs, tenantID: tenantID,
	})
	return nil
}

type staleFixture struct {
	client      *ent.Client
	svc         *Service
	notifier    *recordingNotifier
	tenantID    int
	requesterID int
	oldDef      *ent.ProcessDefinition
	newDef      *ent.ProcessDefinition
}

func newStaleFixture(t *testing.T) *staleFixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:"+t.TempDir()+"/stale_sr.db?_fk=1")
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenant, err := client.Tenant.Create().
		SetName("StaleTenant").
		SetCode("stale-" + strconv.FormatInt(time.Now().UnixNano(), 10)).
		SetDomain("stale.test").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	requester, err := client.User.Create().
		SetUsername("stale-requester-" + strconv.FormatInt(time.Now().UnixNano(), 10)).
		SetEmail("stale-requester@example.com").
		SetName("Stale Requester").
		SetPasswordHash("hash").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	fixture := &staleFixture{
		client:      client,
		tenantID:    tenant.ID,
		requesterID: requester.ID,
		notifier:    &recordingNotifier{},
	}
	fixture.oldDef = fixture.createDefinition(t, "2.0.0", false, "old")
	fixture.newDef = fixture.createDefinition(t, "2.1.0", true, "new")

	fixture.svc = NewService(NewEntRepository(client), nil, nil, client, logger, nil)
	fixture.svc.SetSystemNotifier(fixture.notifier)
	return fixture
}

func (f *staleFixture) createDefinition(t *testing.T, version string, latest bool, suffix string) *ent.ProcessDefinition {
	t.Helper()
	ctx := context.Background()
	deployment, err := f.client.ProcessDeployment.Create().
		SetDeploymentID("stale-dep-" + suffix + "-" + version).
		SetDeploymentName("Stale Deployment " + suffix).
		SetDeploymentTime(time.Now()).
		SetTenantID(f.tenantID).
		Save(ctx)
	require.NoError(t, err)
	def, err := f.client.ProcessDefinition.Create().
		SetKey(staleSRProcessKey).
		SetName("服务请求流程").
		SetVersion(version).
		SetIsLatest(latest).
		SetBpmnXML([]byte("<definitions/>")).
		SetDeploymentID(deployment.ID).
		SetDeployedAt(time.Now()).
		SetTenantID(f.tenantID).
		Save(ctx)
	require.NoError(t, err)
	return def
}

func (f *staleFixture) createRequest(t *testing.T, status string) int {
	t.Helper()
	req, err := f.client.ServiceRequest.Create().
		SetTenantID(f.tenantID).
		SetCatalogID(1).
		SetRequesterID(f.requesterID).
		SetStatus(status).
		SetTitle("旧版流程请求 " + status).
		Save(context.Background())
	require.NoError(t, err)
	return req.ID
}

// createInstance 播种运行中实例与待办任务；返回实例与任务 ID。
func (f *staleFixture) createInstance(t *testing.T, def *ent.ProcessDefinition, requestID int) (*ent.ProcessInstance, *ent.ProcessTask) {
	t.Helper()
	ctx := context.Background()
	instance, err := f.client.ProcessInstance.Create().
		SetProcessInstanceID(fmt.Sprintf("stale-pi-%d-%s", requestID, def.Version)).
		SetProcessDefinitionKey(def.Key).
		SetProcessDefinitionID(def.ID).
		SetBusinessKey(staleSRBusinessKeyPrefix + strconv.Itoa(requestID)).
		SetStatus("running").
		SetTenantID(f.tenantID).
		Save(ctx)
	require.NoError(t, err)
	task, err := f.client.ProcessTask.Create().
		SetTaskID(fmt.Sprintf("stale-task-%d-%s", requestID, def.Version)).
		SetTaskDefinitionKey("Activity_Accept").
		SetTaskName("请求受理").
		SetTaskType("user_task").
		SetProcessDefinitionKey(def.Key).
		SetProcessInstanceID(instance.ID).
		SetStatus("assigned").
		SetTenantID(f.tenantID).
		Save(ctx)
	require.NoError(t, err)
	return instance, task
}

func (f *staleFixture) createPendingApproval(t *testing.T, requestID int) *ent.ServiceRequestApproval {
	t.Helper()
	approval, err := f.client.ServiceRequestApproval.Create().
		SetTenantID(f.tenantID).
		SetServiceRequestID(requestID).
		SetLevel(1).
		SetStep("manager").
		SetStatus("pending").
		Save(context.Background())
	require.NoError(t, err)
	return approval
}

func TestTerminateStaleServiceRequestInstances_DryRunDoesNotWrite(t *testing.T) {
	f := newStaleFixture(t)
	ctx := context.Background()
	requestID := f.createRequest(t, SRStatusSubmitted)
	instance, _ := f.createInstance(t, f.oldDef, requestID)
	approval := f.createPendingApproval(t, requestID)

	items, err := f.svc.TerminateStaleServiceRequestInstances(ctx, f.tenantID, f.requesterID, "", true)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, staleActionCancel, items[0].Action)
	assert.False(t, items[0].Applied)

	req, err := f.client.ServiceRequest.Get(ctx, requestID)
	require.NoError(t, err)
	assert.Equal(t, SRStatusSubmitted, req.Status)
	persistedInstance, err := f.client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "running", persistedInstance.Status)
	persistedApproval, err := f.client.ServiceRequestApproval.Get(ctx, approval.ID)
	require.NoError(t, err)
	assert.Equal(t, ApprovalStatusPending, persistedApproval.Status)
	assert.Empty(t, f.notifier.calls)
	assert.Zero(t, f.auditCount(t))
}

func TestTerminateStaleServiceRequestInstances_CancelsAndPromptsRebuild(t *testing.T) {
	f := newStaleFixture(t)
	ctx := context.Background()
	requestID := f.createRequest(t, SRStatusManagerApproved)
	instance, task := f.createInstance(t, f.oldDef, requestID)
	approval := f.createPendingApproval(t, requestID)

	items, err := f.svc.TerminateStaleServiceRequestInstances(ctx, f.tenantID, f.requesterID, "升级到 v2.1.0", false)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, staleActionCancel, items[0].Action)
	assert.True(t, items[0].Applied)
	assert.Equal(t, "2.0.0", items[0].DefinitionVersion)
	assert.Equal(t, "2.1.0", items[0].LatestVersion)

	req, err := f.client.ServiceRequest.Get(ctx, requestID)
	require.NoError(t, err)
	assert.Equal(t, SRStatusCancelled, req.Status)

	persistedInstance, err := f.client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "terminated", persistedInstance.Status)
	persistedTask, err := f.client.ProcessTask.Get(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", persistedTask.Status)
	persistedApproval, err := f.client.ServiceRequestApproval.Get(ctx, approval.ID)
	require.NoError(t, err)
	assert.Equal(t, ApprovalStatusCancelled, persistedApproval.Status)

	require.Len(t, f.notifier.calls, 1)
	assert.Equal(t, []int{f.requesterID}, f.notifier.calls[0].userIDs)
	assert.Equal(t, f.tenantID, f.notifier.calls[0].tenantID)
	assert.Contains(t, f.notifier.calls[0].message, "重新提交")
	assert.Equal(t, 1, f.auditCount(t))

	// 再跑一次不再产生处置对象（实例已终止，状态已取消）。
	again, err := f.svc.TerminateStaleServiceRequestInstances(ctx, f.tenantID, f.requesterID, "", false)
	require.NoError(t, err)
	assert.Empty(t, again)
}

func TestTerminateStaleServiceRequestInstances_IgnoresLatestDefinition(t *testing.T) {
	f := newStaleFixture(t)
	ctx := context.Background()
	requestID := f.createRequest(t, SRStatusSubmitted)
	instance, _ := f.createInstance(t, f.newDef, requestID)

	items, err := f.svc.TerminateStaleServiceRequestInstances(ctx, f.tenantID, f.requesterID, "", false)
	require.NoError(t, err)
	assert.Empty(t, items)
	req, err := f.client.ServiceRequest.Get(ctx, requestID)
	require.NoError(t, err)
	assert.Equal(t, SRStatusSubmitted, req.Status)
	persistedInstance, err := f.client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "running", persistedInstance.Status)
}

func TestTerminateStaleServiceRequestInstances_SkipsProvisioning(t *testing.T) {
	f := newStaleFixture(t)
	ctx := context.Background()
	requestID := f.createRequest(t, SRStatusProvisioning)
	instance, _ := f.createInstance(t, f.oldDef, requestID)

	items, err := f.svc.TerminateStaleServiceRequestInstances(ctx, f.tenantID, f.requesterID, "", false)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, staleActionSkip, items[0].Action)
	assert.False(t, items[0].Applied)
	req, err := f.client.ServiceRequest.Get(ctx, requestID)
	require.NoError(t, err)
	assert.Equal(t, SRStatusProvisioning, req.Status)
	persistedInstance, err := f.client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "running", persistedInstance.Status)
	assert.Empty(t, f.notifier.calls)
}

func TestTerminateStaleServiceRequestInstances_TerminatesResidualWorkflowForCancelledRequest(t *testing.T) {
	f := newStaleFixture(t)
	ctx := context.Background()
	requestID := f.createRequest(t, SRStatusCancelled)
	instance, _ := f.createInstance(t, f.oldDef, requestID)

	items, err := f.svc.TerminateStaleServiceRequestInstances(ctx, f.tenantID, f.requesterID, "", false)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, staleActionTerminateWorkflow, items[0].Action)
	assert.True(t, items[0].Applied)
	persistedInstance, err := f.client.ProcessInstance.Get(ctx, instance.ID)
	require.NoError(t, err)
	assert.Equal(t, "terminated", persistedInstance.Status)
	assert.Empty(t, f.notifier.calls, "已取消的请求不重复提示重建")
}

func TestTerminateStaleServiceRequestInstances_IgnoresOtherBusinessKeys(t *testing.T) {
	f := newStaleFixture(t)
	ctx := context.Background()
	_, err := f.client.ProcessInstance.Create().
		SetProcessInstanceID("stale-pi-ticket").
		SetProcessDefinitionKey(staleSRProcessKey).
		SetProcessDefinitionID(f.oldDef.ID).
		SetBusinessKey("ticket:42").
		SetStatus("running").
		SetTenantID(f.tenantID).
		Save(ctx)
	require.NoError(t, err)

	items, err := f.svc.TerminateStaleServiceRequestInstances(ctx, f.tenantID, f.requesterID, "", false)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func (f *staleFixture) auditCount(t *testing.T) int {
	t.Helper()
	count, err := f.client.AuditLog.Query().
		Where(auditlog.TenantID(f.tenantID), auditlog.ActionEQ("cancel_stale_process_instance")).
		Count(context.Background())
	require.NoError(t, err)
	return count
}
