package service

import (
	"context"
	"testing"

	domainSR "itsm-backend/domain/servicerequest"
	"itsm-backend/ent"
	"itsm-backend/ent/auditlog"
	"itsm-backend/ent/enttest"
	"itsm-backend/infrastructure/cloud"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type recordingDeliveryListener struct {
	started  int
	finished []bool
	details  []string
}

func (r *recordingDeliveryListener) OnProvisioningStarted(_ context.Context, tenantID, serviceRequestID, actorUserID int) {
	r.started++
}

func (r *recordingDeliveryListener) OnProvisioningFinished(_ context.Context, tenantID, serviceRequestID, actorUserID int, delivered bool, detail string) {
	r.finished = append(r.finished, delivered)
	r.details = append(r.details, detail)
}

type recordingCIProjector struct {
	requests []ProvisioningCIProjectionRequest
	result   ProvisioningCIProjectionResult
}

func (r *recordingCIProjector) ProjectProvisionedResource(_ context.Context, req ProvisioningCIProjectionRequest) ProvisioningCIProjectionResult {
	r.requests = append(r.requests, req)
	return r.result
}

func TestCreateTaskFromServiceRequestRejectsUnsupportedResourceType(t *testing.T) {
	client := enttest.Open(t, dialect.SQLite, "file:prov-unsupported-resource?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	tenantID, requesterID := seedProvisioningPrerequisites(t, ctx, client)

	catalog, err := client.ServiceCatalog.Create().
		SetTenantID(tenantID).
		SetName("provisioning").
		SetStatus("enabled").
		Save(ctx)
	require.NoError(t, err)

	req, err := client.ServiceRequest.Create().
		SetTenantID(tenantID).
		SetCatalogID(catalog.ID).
		SetRequesterID(requesterID).
		SetTitle("Need OSS").
		SetStatus(domainSR.StatusSecurityApproved).
		SetFormData(map[string]any{"resource_type": "oss"}).
		Save(ctx)
	require.NoError(t, err)

	svc := NewProvisioningService(client, zaptest.NewLogger(t).Sugar())
	_, err = svc.CreateTaskFromServiceRequest(ctx, req.ID, tenantID, requesterID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "暂不支持交付资源类型 oss")

	taskCount, err := client.ProvisioningTask.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, taskCount, "不支持的类型必须在建任务前拒绝")
	reloaded, err := client.ServiceRequest.Get(ctx, req.ID)
	require.NoError(t, err)
	require.Equal(t, domainSR.StatusSecurityApproved, reloaded.Status)
}

func TestCreateTaskFromServiceRequestNotifiesDeliveryStarted(t *testing.T) {
	client := enttest.Open(t, dialect.SQLite, "file:prov-start-listener?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	tenantID, requesterID := seedProvisioningPrerequisites(t, ctx, client)

	catalog, err := client.ServiceCatalog.Create().
		SetTenantID(tenantID).
		SetName("provisioning").
		SetStatus("enabled").
		Save(ctx)
	require.NoError(t, err)
	req, err := client.ServiceRequest.Create().
		SetTenantID(tenantID).
		SetCatalogID(catalog.ID).
		SetRequesterID(requesterID).
		SetTitle("Need ECS").
		SetStatus(domainSR.StatusSecurityApproved).
		Save(ctx)
	require.NoError(t, err)

	listener := &recordingDeliveryListener{}
	svc := NewProvisioningService(client, zaptest.NewLogger(t).Sugar())
	svc.SetDeliveryListener(listener)
	_, err = svc.CreateTaskFromServiceRequest(ctx, req.ID, tenantID, requesterID)
	require.NoError(t, err)

	require.Equal(t, 1, listener.started)
	require.Empty(t, listener.finished, "建任务只是开始，不能提前通知完成")
}

func TestExecuteTaskRejectsUnsupportedResourceType(t *testing.T) {
	fixture := newCloudDeliveryFixture(t, []string{"cn-hangzhou"}, false, func(ctx context.Context, client *ent.Client, tenantID int, ref string) {
		seedSingleAliyunAccount(t, client, ctx, tenantID, ref)
	})
	ctx := context.Background()
	_, err := fixture.client.ProvisioningTask.UpdateOneID(fixture.task.ID).SetResourceType("oss").Save(ctx)
	require.NoError(t, err)

	provider := &fakeCloudProvider{requiresAccount: true}
	listener := &recordingDeliveryListener{}
	svc := NewProvisioningService(fixture.client, zaptest.NewLogger(t).Sugar())
	svc.SetProvider(provider)
	svc.SetCloudRuntime(&fakeSecretResolver{values: map[string]string{}})
	svc.SetDeliveryListener(listener)

	_, err = svc.ExecuteTask(ctx, fixture.task.ID, fixture.tenantID, fixture.requester)
	require.ErrorIs(t, err, ErrProvisioningNotExecutable)
	require.Empty(t, provider.calls, "不支持的类型不得发起云 API 调用")
	require.Equal(t, 1, listener.started)
	require.Equal(t, []bool{false}, listener.finished)

	failed, err := fixture.client.ProvisioningTask.Get(ctx, fixture.task.ID)
	require.NoError(t, err)
	require.Equal(t, "failed", failed.Status)
	require.Contains(t, failed.ErrorMessage, "暂不支持交付资源类型 oss")
}

func TestExecuteTaskProjectsCIAndNotifiesDeliveryFinished(t *testing.T) {
	fixture := newCloudDeliveryFixture(t, []string{"cn-hangzhou"}, false, func(ctx context.Context, client *ent.Client, tenantID int, ref string) {
		seedSingleAliyunAccount(t, client, ctx, tenantID, ref)
	})
	ctx := context.Background()

	provider := &fakeCloudProvider{
		requiresAccount: true,
		result: &cloud.ExecuteResult{Resources: map[string]string{
			"instance_id": "i-abc123",
			"account_id":  "1234567890123456",
			"region_id":   "cn-hangzhou",
		}},
	}
	resolver := &fakeSecretResolver{values: map[string]string{
		fixture.accountRef: `{"type":"ak","access_key_id":"ak-test","access_key_secret":"sk-test"}`,
	}}
	listener := &recordingDeliveryListener{}
	projector := &recordingCIProjector{result: ProvisioningCIProjectionResult{
		Status: ProvisioningCIProjectionCreated,
		CIID:   501,
	}}
	svc := NewProvisioningService(fixture.client, zaptest.NewLogger(t).Sugar())
	svc.SetProvider(provider)
	svc.SetCloudRuntime(resolver)
	svc.SetDeliveryListener(listener)
	svc.SetCIProjector(projector)

	executed, err := svc.ExecuteTask(ctx, fixture.task.ID, fixture.tenantID, fixture.requester)
	require.NoError(t, err)
	require.Equal(t, 1, listener.started)
	require.Equal(t, []bool{true}, listener.finished)
	require.Len(t, projector.requests, 1)
	require.Equal(t, fixture.tenantID, projector.requests[0].TenantID)
	require.Equal(t, fixture.request.ID, projector.requests[0].ServiceRequestID)
	require.Equal(t, fixture.task.ID, projector.requests[0].ProvisioningTaskID)
	require.Equal(t, "i-abc123", projector.requests[0].Resources["instance_id"])

	projection, ok := executed.Result["cmdbProjection"].(map[string]any)
	require.True(t, ok, "投影结果必须写回任务结果供运维排查")
	require.Equal(t, ProvisioningCIProjectionCreated, projection["status"])

	reloaded, err := fixture.client.ServiceRequest.Get(ctx, fixture.request.ID)
	require.NoError(t, err)
	require.Equal(t, 501, reloaded.CiID, "首次交付出的 CI 必须挂到服务请求上")

	audits, err := fixture.client.AuditLog.Query().
		Where(auditlog.TenantID(fixture.tenantID), auditlog.ActionEQ("provision.ci.projected")).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, audits, 1)
}
