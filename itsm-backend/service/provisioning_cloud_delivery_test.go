package service

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"itsm-backend/domain/provisioning"
	domainSR "itsm-backend/domain/servicerequest"
	"itsm-backend/ent"
	"itsm-backend/ent/auditlog"
	"itsm-backend/ent/cloudaccount"
	"itsm-backend/ent/enttest"
	"itsm-backend/infrastructure/cloud"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type fakeCloudProvider struct {
	requiresAccount bool
	result          *cloud.ExecuteResult
	err             error
	calls           []cloud.ExecuteRequest
}

func (f *fakeCloudProvider) Name() string { return "fake-aliyun" }

func (f *fakeCloudProvider) RequiresCloudAccount() bool { return f.requiresAccount }

func (f *fakeCloudProvider) Execute(_ context.Context, req cloud.ExecuteRequest) (*cloud.ExecuteResult, error) {
	f.calls = append(f.calls, req)
	if f.err != nil {
		return nil, f.err
	}
	if f.result != nil {
		return f.result, nil
	}
	return &cloud.ExecuteResult{Resources: map[string]string{"instance_id": "i-test"}}, nil
}

type fakeSecretResolver struct {
	values map[string]string
	err    error
	calls  int
}

func (f *fakeSecretResolver) Resolve(_ context.Context, _ int, ref string) (string, error) {
	f.calls++
	if f.err != nil {
		return "", f.err
	}
	value, ok := f.values[ref]
	if !ok {
		return "", fmt.Errorf("secret not configured: %s", ref)
	}
	return value, nil
}

type cloudDeliveryFixture struct {
	client     *ent.Client
	tenantID   int
	requester  int
	catalog    *ent.ServiceCatalog
	request    *ent.ServiceRequest
	task       *ent.ProvisioningTask
	accountRef string
}

func newCloudDeliveryFixture(t *testing.T, availableRegions []string, needsPublicIP bool, setupAccounts func(ctx context.Context, client *ent.Client, tenantID int, ref string)) *cloudDeliveryFixture {
	t.Helper()
	dsn := "file:cloud_delivery_" + strings.NewReplacer("/", "_", " ", "_").Replace(t.Name()) + "?mode=memory&cache=shared&_fk=1"
	client := enttest.Open(t, dialect.SQLite, dsn)
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()

	tenantID, requesterID := seedProvisioningPrerequisites(t, ctx, client)
	ref := fmt.Sprintf("secret://tenant-%d/aliyun-prod", tenantID)
	setupAccounts(ctx, client, tenantID, ref)

	catalog, err := client.ServiceCatalog.Create().
		SetTenantID(tenantID).
		SetName("ecs-delivery").
		SetStatus("enabled").
		SetAvailableRegions(availableRegions).
		Save(ctx)
	require.NoError(t, err)

	form := map[string]any{
		"regionId":        "cn-hangzhou",
		"imageId":         "aliyun_3_x64_20G_alibase_20240819.vhd",
		"instanceType":    "ecs.e-c1m1.large",
		"securityGroupId": "sg-test",
		"vSwitchId":       "vsw-test",
	}
	req, err := client.ServiceRequest.Create().
		SetTenantID(tenantID).
		SetCatalogID(catalog.ID).
		SetRequesterID(requesterID).
		SetTitle("Need ECS").
		SetStatus(domainSR.StatusProvisioning).
		SetNeedsPublicIP(needsPublicIP).
		SetFormData(form).
		Save(ctx)
	require.NoError(t, err)

	task, err := client.ProvisioningTask.Create().
		SetTenantID(tenantID).
		SetServiceRequestID(req.ID).
		SetProvider(string(provisioning.ProviderAlicloud)).
		SetResourceType(string(provisioning.ResourceECS)).
		SetStatus(string(provisioning.TaskPending)).
		SetPayload(form).
		Save(ctx)
	require.NoError(t, err)

	return &cloudDeliveryFixture{
		client:     client,
		tenantID:   tenantID,
		requester:  requesterID,
		catalog:    catalog,
		request:    req,
		task:       task,
		accountRef: ref,
	}
}

func seedSingleAliyunAccount(t *testing.T, client *ent.Client, ctx context.Context, tenantID int, ref string) {
	t.Helper()
	_, err := client.CloudAccount.Create().
		SetTenantID(tenantID).
		SetProvider("aliyun").
		SetAccountID("1234567890123456").
		SetAccountName("prod-hangzhou").
		SetCredentialRef(ref).
		SetRegionWhitelist([]string{"cn-hangzhou"}).
		SetIsActive(true).
		Save(ctx)
	require.NoError(t, err)
}

func TestExecuteTaskResolvesTenantCloudAccountAndCredential(t *testing.T) {
	fixture := newCloudDeliveryFixture(t, []string{"cn-hangzhou"}, false, func(ctx context.Context, client *ent.Client, tenantID int, ref string) {
		seedSingleAliyunAccount(t, client, ctx, tenantID, ref)
	})
	ctx := context.Background()

	provider := &fakeCloudProvider{
		requiresAccount: true,
		result:          &cloud.ExecuteResult{Resources: map[string]string{"instance_id": "i-abc123", "region_id": "cn-hangzhou"}},
	}
	resolver := &fakeSecretResolver{values: map[string]string{
		fixture.accountRef: `{"type":"ak","access_key_id":"ak-test","access_key_secret":"sk-test"}`,
	}}
	svc := NewProvisioningService(fixture.client, zaptest.NewLogger(t).Sugar())
	svc.SetProvider(provider)
	svc.SetCloudRuntime(resolver)

	executed, err := svc.ExecuteTask(ctx, fixture.task.ID, fixture.tenantID, fixture.requester)
	require.NoError(t, err)
	require.Equal(t, string(provisioning.TaskSucceeded), executed.Status)
	require.Equal(t, 1, resolver.calls)
	require.Len(t, provider.calls, 1)

	call := provider.calls[0]
	require.Equal(t, fixture.tenantID, call.TenantID)
	require.Equal(t, "cn-hangzhou", call.Region)
	require.Equal(t, "1234567890123456", call.AccountID)
	require.Equal(t, "ak-test", call.Credential.AccessKeyID)
	require.Equal(t, "sk-test", call.Credential.AccessKeySecret)
	require.Equal(t, fmt.Sprintf("itsm-provisioning-task-%d", fixture.task.ID), call.ClientToken)
	require.Equal(t, false, call.Payload["needsPublicIp"])

	reloaded, err := fixture.client.ServiceRequest.Get(ctx, fixture.request.ID)
	require.NoError(t, err)
	require.Equal(t, domainSR.StatusDelivered, reloaded.Status)

	audits, err := fixture.client.AuditLog.Query().
		Where(auditlog.TenantID(fixture.tenantID), auditlog.ActionEQ("provision.execute.succeeded")).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, audits, 1)
	require.NotNil(t, audits[0].RequestBody)
	auditBody := *audits[0].RequestBody
	require.NotContains(t, auditBody, "sk-test", "审计不得落凭据")
	require.NotContains(t, auditBody, "ak-test", "审计不得落凭据")
}

func TestExecuteTaskFailsClosedWithoutCredentialResolver(t *testing.T) {
	fixture := newCloudDeliveryFixture(t, []string{"cn-hangzhou"}, false, func(ctx context.Context, client *ent.Client, tenantID int, ref string) {
		seedSingleAliyunAccount(t, client, ctx, tenantID, ref)
	})
	ctx := context.Background()

	provider := &fakeCloudProvider{requiresAccount: true}
	svc := NewProvisioningService(fixture.client, zaptest.NewLogger(t).Sugar())
	svc.SetProvider(provider)

	_, err := svc.ExecuteTask(ctx, fixture.task.ID, fixture.tenantID, fixture.requester)
	require.ErrorIs(t, err, ErrProvisioningNotExecutable)
	require.Empty(t, provider.calls, "配置缺失时不得发起云 API 调用")

	reloaded, err := fixture.client.ProvisioningTask.Get(ctx, fixture.task.ID)
	require.NoError(t, err)
	require.Equal(t, string(provisioning.TaskFailed), reloaded.Status)
	require.Contains(t, reloaded.ErrorMessage, "凭据解析器未装配")

	sr, err := fixture.client.ServiceRequest.Get(ctx, fixture.request.ID)
	require.NoError(t, err)
	require.Equal(t, domainSR.StatusFailed, sr.Status)
}

func TestExecuteTaskRejectsRegionOutsideCatalog(t *testing.T) {
	fixture := newCloudDeliveryFixture(t, []string{"cn-beijing"}, false, func(ctx context.Context, client *ent.Client, tenantID int, ref string) {
		seedSingleAliyunAccount(t, client, ctx, tenantID, ref)
	})
	ctx := context.Background()

	provider := &fakeCloudProvider{requiresAccount: true}
	resolver := &fakeSecretResolver{values: map[string]string{
		fixture.accountRef: `{"type":"ak","access_key_id":"ak-test","access_key_secret":"sk-test"}`,
	}}
	svc := NewProvisioningService(fixture.client, zaptest.NewLogger(t).Sugar())
	svc.SetProvider(provider)
	svc.SetCloudRuntime(resolver)

	_, err := svc.ExecuteTask(ctx, fixture.task.ID, fixture.tenantID, fixture.requester)
	require.ErrorIs(t, err, ErrProvisioningNotExecutable)
	require.Empty(t, provider.calls)
	require.Zero(t, resolver.calls)
}

func TestExecuteTaskRejectsAmbiguousCloudAccounts(t *testing.T) {
	fixture := newCloudDeliveryFixture(t, []string{"cn-hangzhou"}, false, func(ctx context.Context, client *ent.Client, tenantID int, ref string) {
		for i := 0; i < 2; i++ {
			_, err := client.CloudAccount.Create().
				SetTenantID(tenantID).
				SetProvider("aliyun").
				SetAccountID(fmt.Sprintf("account-%d", i)).
				SetAccountName(fmt.Sprintf("prod-%d", i)).
				SetCredentialRef(ref).
				SetRegionWhitelist([]string{"cn-hangzhou"}).
				SetIsActive(true).
				Save(ctx)
			require.NoError(t, err)
		}
	})
	ctx := context.Background()

	provider := &fakeCloudProvider{requiresAccount: true}
	svc := NewProvisioningService(fixture.client, zaptest.NewLogger(t).Sugar())
	svc.SetProvider(provider)
	svc.SetCloudRuntime(&fakeSecretResolver{values: map[string]string{}})

	_, err := svc.ExecuteTask(ctx, fixture.task.ID, fixture.tenantID, fixture.requester)
	require.ErrorIs(t, err, ErrProvisioningNotExecutable)
	require.Contains(t, err.Error(), "cloudAccountId")
	require.Empty(t, provider.calls)
}

func TestExecuteTaskSelectsExplicitCloudAccount(t *testing.T) {
	fixture := newCloudDeliveryFixture(t, []string{"cn-hangzhou"}, false, func(ctx context.Context, client *ent.Client, tenantID int, ref string) {
		for i := 0; i < 2; i++ {
			_, err := client.CloudAccount.Create().
				SetTenantID(tenantID).
				SetProvider("aliyun").
				SetAccountID(fmt.Sprintf("account-%d", i)).
				SetAccountName(fmt.Sprintf("prod-%d", i)).
				SetCredentialRef(ref).
				SetRegionWhitelist([]string{"cn-hangzhou"}).
				SetIsActive(true).
				Save(ctx)
			require.NoError(t, err)
		}
	})
	ctx := context.Background()

	accounts, err := fixture.client.CloudAccount.Query().Order(ent.Asc(cloudaccount.FieldID)).All(ctx)
	require.NoError(t, err)
	require.Len(t, accounts, 2)
	payload := map[string]any{
		"regionId":        "cn-hangzhou",
		"imageId":         "img",
		"instanceType":    "ecs.e-c1m1.large",
		"securityGroupId": "sg",
		"vSwitchId":       "vsw",
		"cloudAccountId":  float64(accounts[1].ID),
	}
	_, err = fixture.client.ProvisioningTask.UpdateOneID(fixture.task.ID).SetPayload(payload).Save(ctx)
	require.NoError(t, err)

	provider := &fakeCloudProvider{requiresAccount: true}
	resolver := &fakeSecretResolver{values: map[string]string{
		fixture.accountRef: `{"type":"ak","access_key_id":"ak-test","access_key_secret":"sk-test"}`,
	}}
	svc := NewProvisioningService(fixture.client, zaptest.NewLogger(t).Sugar())
	svc.SetProvider(provider)
	svc.SetCloudRuntime(resolver)

	_, err = svc.ExecuteTask(ctx, fixture.task.ID, fixture.tenantID, fixture.requester)
	require.NoError(t, err)
	require.Len(t, provider.calls, 1)
	require.Equal(t, accounts[1].AccountID, provider.calls[0].AccountID)
}

func TestExecuteTaskKeepsClientTokenStableAcrossRetries(t *testing.T) {
	fixture := newCloudDeliveryFixture(t, []string{"cn-hangzhou"}, false, func(ctx context.Context, client *ent.Client, tenantID int, ref string) {
		seedSingleAliyunAccount(t, client, ctx, tenantID, ref)
	})
	ctx := context.Background()

	provider := &fakeCloudProvider{requiresAccount: true, err: fmt.Errorf("temporary network error")}
	resolver := &fakeSecretResolver{values: map[string]string{
		fixture.accountRef: `{"type":"ak","access_key_id":"ak-test","access_key_secret":"sk-test"}`,
	}}
	svc := NewProvisioningService(fixture.client, zaptest.NewLogger(t).Sugar())
	svc.SetProvider(provider)
	svc.SetCloudRuntime(resolver)

	_, err := svc.ExecuteTask(ctx, fixture.task.ID, fixture.tenantID, fixture.requester)
	require.Error(t, err)
	require.Len(t, provider.calls, 1)

	_, err = svc.RetryTask(ctx, fixture.task.ID, fixture.tenantID, fixture.requester, "重试")
	require.NoError(t, err)

	provider.err = nil
	_, err = svc.ExecuteTask(ctx, fixture.task.ID, fixture.tenantID, fixture.requester)
	require.NoError(t, err)
	require.Len(t, provider.calls, 2)
	require.Equal(t, provider.calls[0].ClientToken, provider.calls[1].ClientToken,
		"重试必须复用同一 ClientToken，否则阿里云会再创建一台实例")
}
