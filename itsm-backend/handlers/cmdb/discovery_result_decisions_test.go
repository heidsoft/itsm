package cmdb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"itsm-backend/common"
	"itsm-backend/ent/enttest"
)

// 差异决策面（apply / reject）回归锁。
//
// 背景：manual 对账策略下，执行器对「有变化」的资源只落 status=pending 的
// discovery_result 而不改 CloudResource（生产属性变更要人确认），但全仓没有
// 任何写入面能让这条差异被应用或驳回——对账永远停在「算出了差异」，账合不上。
// 本文件锁四件事：应用真的写回资源、重复决策被拒、非 update 差异不可决策、
// 资源消失时不静默成功。

// seedDiscoveryJob 建一个最小可用的发现任务（discovery_results.job_id 是外键）。
func seedDiscoveryJob(t *testing.T, repo *EntRepository, tenantID int, suffix string) int {
	t.Helper()
	ctx := context.Background()
	source, err := repo.CreateDiscoverySource(ctx, &DiscoverySource{
		ID: "ds-decide-" + suffix, Name: "决策源-" + suffix, SourceType: "api", Provider: "aliyun",
		ReconcilePolicy: "manual", StaleThreshold: 2, IsActive: true, TenantID: tenantID,
	})
	require.NoError(t, err)
	job, err := repo.client.DiscoveryJob.Create().
		SetSourceID(source.ID).SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	return job.ID
}

// seedPendingUpdate 造一条「待确认的 update 差异」及其对应的 CloudResource。
func seedPendingUpdate(t *testing.T, repo *EntRepository, tenantID int, identity string) (resourceID, resultID int) {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	// 账号与服务定义都是 CloudResource 的外键目标（sqlite 开了 _fk=1）。
	account, err := repo.client.CloudAccount.Create().SetTenantID(tenantID).SetProvider("aliyun").
		SetAccountID("acc-" + identity).SetAccountName("账号").SetCredentialRef("secret://test").
		SetIsActive(true).Save(ctx)
	require.NoError(t, err)
	svcDef, err := repo.client.CloudService.Create().SetTenantID(tenantID).SetProvider("aliyun").
		SetCategory("compute").SetServiceCode("ecs").SetServiceName("弹性计算").
		SetResourceTypeCode("instance").SetResourceTypeName("实例").SetAPIVersion("2014-05-26").
		SetIsActive(true).Save(ctx)
	require.NoError(t, err)

	resource, err := repo.CreateCloudResource(ctx, &CloudResource{
		CloudAccountID: account.ID, ServiceID: svcDef.ID, ResourceID: "i-1", IdentityVersion: 1,
		Provider: "aliyun", Partition: "public", CanonicalAccountID: "acc-1",
		ResourceScope: "cn-hangzhou", ServiceCode: "ecs", ResourceType: "instance",
		IdentityHash: identity, SourceID: "ds-decide", SourceFingerprint: "old-hash",
		MissingCount: 0, ResourceName: "old-name", Region: "cn-hangzhou",
		Status: "running", Tags: map[string]string{"env": "old"},
		FirstSeenAt: &now, LastSeenAt: &now, LifecycleState: "active", TenantID: tenantID,
	})
	require.NoError(t, err)

	result, err := repo.CreateDiscoveryResult(ctx, &DiscoveryResult{
		JobID: seedDiscoveryJob(t, repo, tenantID, identity), Action: "update", ResourceType: "instance", ResourceID: "i-1",
		ResourceIdentity: identity, IdentityVersion: 1,
		ResourceSnapshot: map[string]interface{}{
			"resourceId": "i-1", "resourceName": "new-name", "region": "cn-hangzhou",
			"status": "Stopped", "zone": "cn-hangzhou-h",
			"tags": map[string]interface{}{"env": "prod", "team": "sre"},
		},
		BeforeHash: "old-hash", AfterHash: "new-hash",
		Diff:     map[string]interface{}{"resourceName": map[string]interface{}{"before": "old-name", "after": "new-name"}},
		Status:   DiscoveryResultPending,
		TenantID: tenantID,
	})
	require.NoError(t, err)
	return resource.ID, result.ID
}

func TestApplyDiscoveryResultWritesResourceFacts(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:discovery_decide_apply?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	repo := NewEntRepository(client)
	tenant, err := client.Tenant.Create().SetName("T").SetCode("decide").SetDomain("decide.example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	resourceID, resultID := seedPendingUpdate(t, repo, tenant.ID, "identity-1")
	svc := NewService(repo, nil, nopLogger())

	result, err := svc.ApplyDiscoveryResult(ctx, tenant.ID, 9, resultID)
	require.NoError(t, err)
	require.Equal(t, DiscoveryResultApplied, result.Status)

	resource, err := repo.GetCloudResourceByIdentity(ctx, tenant.ID, "identity-1")
	require.NoError(t, err)
	require.Equal(t, resourceID, resource.ID)
	require.Equal(t, "new-name", resource.ResourceName)
	require.Equal(t, "inactive", resource.Status,
		"厂商原始状态必须归一后落库（Stopped → inactive，与自动应用同口径）")
	require.Equal(t, "cn-hangzhou-h", resource.Zone)
	require.Equal(t, map[string]string{"env": "prod", "team": "sre"}, resource.Tags)
	require.Equal(t, "new-hash", resource.SourceFingerprint,
		"指纹必须换成被发现的那一版，否则下一轮发现会再判 update")
	require.Equal(t, "active", resource.LifecycleState)

	// 重复应用：409 而不是再写一次（否则「早被驳回过」会被掩盖成成功）。
	_, err = svc.ApplyDiscoveryResult(ctx, tenant.ID, 9, resultID)
	require.Error(t, err)
	var appErr *common.AppError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, common.ErrCodeConflict, appErr.Code)
	require.Contains(t, appErr.Detail, "已处理")
}

func TestRejectDiscoveryResultKeepsResourceUntouched(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:discovery_decide_reject?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	repo := NewEntRepository(client)
	tenant, err := client.Tenant.Create().SetName("T").SetCode("decide-r").SetDomain("decide-r.example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	_, resultID := seedPendingUpdate(t, repo, tenant.ID, "identity-2")
	svc := NewService(repo, nil, nopLogger())

	result, err := svc.RejectDiscoveryResult(ctx, tenant.ID, 9, resultID, "  云上改动未走变更  ")
	require.NoError(t, err)
	require.Equal(t, DiscoveryResultRejected, result.Status)
	require.Equal(t, "云上改动未走变更", result.ErrorMessage, "理由应去除首尾空白")

	resource, err := repo.GetCloudResourceByIdentity(ctx, tenant.ID, "identity-2")
	require.NoError(t, err)
	require.Equal(t, "old-name", resource.ResourceName, "驳回不得改动资源")
	require.Equal(t, "old-hash", resource.SourceFingerprint)

	// 驳回后再应用：409。
	_, err = svc.ApplyDiscoveryResult(ctx, tenant.ID, 9, resultID)
	require.Error(t, err)
	var appErr *common.AppError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, common.ErrCodeConflict, appErr.Code)
}

func TestApplyDiscoveryResultRejectsNonUpdateActions(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:discovery_decide_action?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	repo := NewEntRepository(client)
	tenant, err := client.Tenant.Create().SetName("T").SetCode("decide-a").SetDomain("decide-a.example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	// create 在发现时已落库，不需要人工确认；给它一个「应用」入口等于制造
	// 「同一条事实被写两次」的路径。
	created, err := repo.CreateDiscoveryResult(ctx, &DiscoveryResult{
		JobID: seedDiscoveryJob(t, repo, tenant.ID, "create"), Action: "create", ResourceIdentity: "identity-3", IdentityVersion: 1,
		Status: DiscoveryResultPending, TenantID: tenant.ID,
	})
	require.NoError(t, err)

	svc := NewService(repo, nil, nopLogger())
	_, err = svc.ApplyDiscoveryResult(ctx, tenant.ID, 9, created.ID)
	require.Error(t, err)
	var appErr *common.AppError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, common.ErrCodeConflict, appErr.Code)
	require.Contains(t, appErr.Detail, "只有 update")
}

func TestApplyDiscoveryResultFailsClosedWhenResourceGone(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:discovery_decide_gone?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	repo := NewEntRepository(client)
	tenant, err := client.Tenant.Create().SetName("T").SetCode("decide-g").SetDomain("decide-g.example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	// 差异指向一个已经不存在的资源（发现后被删）：不能静默成功，
	// 也不能自动重建（那是 create 的语义，需要重新发现）。
	orphan, err := repo.CreateDiscoveryResult(ctx, &DiscoveryResult{
		JobID: seedDiscoveryJob(t, repo, tenant.ID, "orphan"), Action: "update", ResourceIdentity: "identity-missing", IdentityVersion: 1,
		AfterHash: "h2", Status: DiscoveryResultPending, TenantID: tenant.ID,
	})
	require.NoError(t, err)

	svc := NewService(repo, nil, nopLogger())
	_, err = svc.ApplyDiscoveryResult(ctx, tenant.ID, 9, orphan.ID)
	require.Error(t, err)
	var appErr *common.AppError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, common.ErrCodeConflict, appErr.Code)
	require.Contains(t, appErr.Detail, "重跑发现")
}

func TestDiscoveryResultDecisionsAreTenantScoped(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:discovery_decide_tenant?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	repo := NewEntRepository(client)
	tenant, err := client.Tenant.Create().SetName("T").SetCode("decide-t").SetDomain("decide-t.example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	_, resultID := seedPendingUpdate(t, repo, tenant.ID, "identity-4")
	svc := NewService(repo, nil, nopLogger())

	// 另一个租户读不到也动不了这条差异。
	_, err = svc.ApplyDiscoveryResult(ctx, tenant.ID+99, 9, resultID)
	require.Error(t, err)
	var appErr *common.AppError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, common.ErrCodeNotFound, appErr.Code)

	result, err := repo.GetDiscoveryResult(ctx, tenant.ID, resultID)
	require.NoError(t, err)
	require.Equal(t, DiscoveryResultPending, result.Status, "跨租户尝试不得改变状态")
}

func TestTransitionDiscoveryResultUsesStatusCAS(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:discovery_result_cas?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	repo := NewEntRepository(client)
	tenant, err := client.Tenant.Create().SetName("T").SetCode("decide-cas").SetDomain("decide-cas.example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)

	_, resultID := seedPendingUpdate(t, repo, tenant.ID, "identity-cas")
	claimed, err := repo.TransitionDiscoveryResult(ctx, tenant.ID, resultID, DiscoveryResultPending, DiscoveryResultApplying, "", "")
	require.NoError(t, err)
	require.Equal(t, 1, claimed)

	changed, err := repo.TransitionDiscoveryResult(ctx, tenant.ID, resultID, DiscoveryResultPending, DiscoveryResultRejected, "", "")
	require.NoError(t, err)
	require.Zero(t, changed, "pending CAS 不得覆盖已经 claim 的 applying 状态")

	changed, err = repo.TransitionDiscoveryResult(ctx, tenant.ID, resultID, DiscoveryResultApplying, DiscoveryResultApplied, "", "")
	require.NoError(t, err)
	require.Equal(t, 1, changed)
}
