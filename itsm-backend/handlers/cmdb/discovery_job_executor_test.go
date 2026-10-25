package cmdb

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/internal/commandbus"
	"itsm-backend/service/cloud"

	_ "github.com/mattn/go-sqlite3"
)

// 云发现任务面已从 fail-closed（503）转正：创建入队 / 列表 / 详情 / 取消 / 重试
// 的真实契约。「来源校验 + 事务入队」的服务层契约见 service_test.go 的
// CreateDiscoveryJob 子测试；本文件锁 handler 信封与执行器端到端行为。

func TestDiscoveryJobsHandler_ServiceNotWiredFailsClosed(t *testing.T) {
	// svc 未装配（NewHandler(nil)）时必须显式 503，不能 nil 指针 panic。
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/jobs", NewHandler(nil).CreateDiscoveryJob)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(`{"sourceId":"ds-1"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// fakeDiscoveryAdapter 返回固定资源序列，供执行器端到端测试。
type fakeDiscoveryAdapter struct {
	provider          string
	serviceCode       string
	regions           []string
	resources         []cloud.DiscoveredResource
	resourcesByRegion map[string][]cloud.DiscoveredResource
}

func (f *fakeDiscoveryAdapter) Provider() string    { return f.provider }
func (f *fakeDiscoveryAdapter) ServiceCode() string { return f.serviceCode }
func (f *fakeDiscoveryAdapter) InitClients(_ context.Context, _ *ent.CloudAccount, regions []string) (map[string]cloud.Client, error) {
	clients := make(map[string]cloud.Client, len(regions))
	for _, region := range regions {
		clients[region] = &stubCloudClient{}
	}
	return clients, nil
}
func (f *fakeDiscoveryAdapter) ListRegions(_ context.Context, _ *ent.CloudAccount) ([]string, error) {
	return f.regions, nil
}
func (f *fakeDiscoveryAdapter) DiscoverRegion(_ context.Context, _ *ent.CloudAccount, region string, _ cloud.Client, nextToken string) (*cloud.PageResult, error) {
	if nextToken != "" {
		return &cloud.PageResult{Resources: nil}, nil
	}
	if f.resourcesByRegion != nil {
		return &cloud.PageResult{Resources: f.resourcesByRegion[region]}, nil
	}
	return &cloud.PageResult{Resources: f.resources}, nil
}
func (f *fakeDiscoveryAdapter) Close() {}
func (f *fakeDiscoveryAdapter) ValidateCredential(_ context.Context, _ *ent.CloudAccount) error {
	return nil
}

type stubCloudClient struct{}

func (c *stubCloudClient) Close() error { return nil }

type fakeRegistry struct{ adapter cloud.CloudDiscoveryAdapter }

func (f *fakeRegistry) Get(provider, serviceCode string) (cloud.CloudDiscoveryAdapter, bool) {
	if f.adapter == nil {
		return nil, false
	}
	if f.adapter.Provider() == provider && f.adapter.ServiceCode() == serviceCode {
		return f.adapter, true
	}
	return nil, false
}

type fakeResolver struct{}

func (f *fakeResolver) Resolve(_ context.Context, _ int, _ string) (string, error) {
	return `{"type":"ak","accessKeyId":"k","accessKeySecret":"s"}`, nil
}

func nopLogger() *zap.SugaredLogger { return zap.NewNop().Sugar() }

// seedDiscoveryEnv 建租户/账号/服务定义/发现源的最小可执行环境。
func seedDiscoveryEnv(t *testing.T, repo Repository, code string) (tenantID, accountID, sourceID int) {
	t.Helper()
	ctx := context.Background()
	tenant, err := repo.(*EntRepository).client.Tenant.Create().SetName("T").SetCode(code).
		SetDomain(code + ".example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	account, err := repo.(*EntRepository).client.CloudAccount.Create().SetTenantID(tenant.ID).SetProvider("aliyun").
		SetAccountID("acc-" + code).SetAccountName("账号").SetCredentialRef("secret://test").
		SetIsActive(true).Save(ctx)
	require.NoError(t, err)
	_, err = repo.(*EntRepository).client.CloudService.Create().SetTenantID(tenant.ID).SetProvider("aliyun").
		SetCategory("compute").SetServiceCode("ecs").SetServiceName("弹性计算").
		SetResourceTypeCode("instance").SetResourceTypeName("实例").SetAPIVersion("2014-05-26").
		SetIsActive(true).Save(ctx)
	require.NoError(t, err)
	source, err := repo.CreateDiscoverySource(ctx, &DiscoverySource{
		ID: "ds-" + code, Name: "测试源", SourceType: "api", Provider: "aliyun",
		CloudAccountID: account.ID, ServiceCodes: []string{"ecs"}, Regions: []string{"cn-hangzhou"},
		ReconcilePolicy: "discovered_wins", StaleThreshold: 2, IsActive: true, TenantID: tenant.ID,
	})
	require.NoError(t, err)
	return tenant.ID, account.ID, len(source.ID)
}

// 执行器端到端（sqlite）：创建入队 → 执行 → CloudResource/DiscoveryResult 落库 →
// 复跑 unchanged/update → 资源消失推进 missing_count → 超阈值置 stale → 终态幂等。
func TestDiscoveryJobExecutor_EndToEnd(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:discovery_exec?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	repo := NewEntRepository(client)

	tenantID, accountID, _ := seedDiscoveryEnv(t, repo, "disc-exec")

	// 服务层创建任务（真实事务入队路径）。
	svc := NewService(repo, nil, nopLogger())
	created, err := svc.CreateDiscoveryJob(ctx, tenantID, 9, &dto.DiscoveryJobRequest{SourceID: "ds-disc-exec"})
	require.NoError(t, err)
	require.Equal(t, DiscoveryJobQueued, created.Status)

	// 命令已入队且 payload 携带 jobID。
	cmd, err := client.OperationalCommand.Query().
		Where(operationalcommand.CommandType(commandbus.CommandRunCMDBCloudDiscovery)).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "pending", cmd.Status)
	require.EqualValues(t, created.ID, cmd.Payload["discoveryJobId"])

	adapter := &fakeDiscoveryAdapter{
		provider: "aliyun", serviceCode: "ecs", regions: []string{"cn-hangzhou"},
		resources: []cloud.DiscoveredResource{
			{BaseResource: cloud.BaseResource{ResourceID: "i-1", ResourceName: "web-1", Region: "cn-hangzhou", Status: "Running"},
				CloudServiceCode: "ecs"},
			{BaseResource: cloud.BaseResource{ResourceID: "i-2", ResourceName: "db-1", Region: "cn-hangzhou", Status: "Stopped"},
				CloudServiceCode: "ecs"},
		},
	}
	executor := NewDiscoveryJobExecutor(repo, &fakeRegistry{adapter: adapter}, &fakeResolver{}, nil, nopLogger())

	require.NoError(t, executor.Execute(ctx, tenantID, created.ID))

	job, err := repo.GetDiscoveryJob(ctx, tenantID, created.ID)
	require.NoError(t, err)
	require.Equal(t, DiscoveryJobSucceeded, job.Status)
	require.Equal(t, 100, job.Progress)
	require.EqualValues(t, 2, job.Summary["discovered"])
	require.EqualValues(t, 2, job.Summary["created"])

	// CloudResource 落库 + 溯源。
	resources, err := repo.ListCloudResourcesByAccountService(ctx, tenantID, accountID, "ecs")
	require.NoError(t, err)
	require.Len(t, resources, 2)
	for _, r := range resources {
		assert.NotEmpty(t, r.IdentityHash)
		assert.Equal(t, "ds-disc-exec", r.SourceID)
		assert.NotEmpty(t, r.SourceFingerprint)
		assert.Equal(t, 0, r.MissingCount)
		assert.Equal(t, "active", r.LifecycleState)
		assert.NotNil(t, r.LastSeenAt)
	}

	// DiscoveryResult：两条 create。
	results, err := repo.ListDiscoveryResults(ctx, tenantID, created.ID)
	require.NoError(t, err)
	require.Len(t, results, 2)
	for _, r := range results {
		assert.Equal(t, "create", r.Action)
		assert.Equal(t, "applied", r.Status)
	}

	// 第二轮：i-1 改名（update diff）、i-2 不变（unchanged）。
	adapter.resources = []cloud.DiscoveredResource{
		{BaseResource: cloud.BaseResource{ResourceID: "i-1", ResourceName: "web-1-renamed", Region: "cn-hangzhou", Status: "Running"},
			CloudServiceCode: "ecs"},
		{BaseResource: cloud.BaseResource{ResourceID: "i-2", ResourceName: "db-1", Region: "cn-hangzhou", Status: "Stopped"},
			CloudServiceCode: "ecs"},
	}
	second, err := svc.CreateDiscoveryJob(ctx, tenantID, 9, &dto.DiscoveryJobRequest{SourceID: "ds-disc-exec"})
	require.NoError(t, err)
	require.NoError(t, executor.Execute(ctx, tenantID, second.ID))

	job2, err := repo.GetDiscoveryJob(ctx, tenantID, second.ID)
	require.NoError(t, err)
	require.Equal(t, DiscoveryJobSucceeded, job2.Status)
	require.EqualValues(t, 1, job2.Summary["updated"])
	require.EqualValues(t, 1, job2.Summary["unchanged"])

	// 第三轮：资源全部消失 → missing_count 推进（阈值 2，第一轮缺失）。
	adapter.resources = nil
	third, err := svc.CreateDiscoveryJob(ctx, tenantID, 9, &dto.DiscoveryJobRequest{SourceID: "ds-disc-exec"})
	require.NoError(t, err)
	require.NoError(t, executor.Execute(ctx, tenantID, third.ID))
	afterMissing, err := repo.ListCloudResourcesByAccountService(ctx, tenantID, accountID, "ecs")
	require.NoError(t, err)
	for _, r := range afterMissing {
		assert.Equal(t, 1, r.MissingCount)
		assert.Equal(t, "active", r.LifecycleState)
	}

	// 第四轮：再次缺失 → 达到阈值置 stale。
	fourth, err := svc.CreateDiscoveryJob(ctx, tenantID, 9, &dto.DiscoveryJobRequest{SourceID: "ds-disc-exec"})
	require.NoError(t, err)
	require.NoError(t, executor.Execute(ctx, tenantID, fourth.ID))
	afterStale, err := repo.ListCloudResourcesByAccountService(ctx, tenantID, accountID, "ecs")
	require.NoError(t, err)
	for _, r := range afterStale {
		assert.Equal(t, 2, r.MissingCount)
		assert.Equal(t, "stale", r.LifecycleState)
	}

	// 终态任务重复投递幂等跳过。
	require.NoError(t, executor.Execute(ctx, tenantID, created.ID))

	// 来源的最近成功时间已更新。
	updatedSource, err := repo.GetDiscoverySource(ctx, tenantID, "ds-disc-exec")
	require.NoError(t, err)
	require.NotNil(t, updatedSource.LastSuccessAt)
}

func TestDiscoveryJobExecutor_MultiRegionMissingScanIsRegionScoped(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:discovery_multiregion?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	repo := NewEntRepository(client)

	tenantID, accountID, _ := seedDiscoveryEnv(t, repo, "disc-multiregion")
	require.NoError(t, client.DiscoverySource.UpdateOneID("ds-disc-multiregion").
		SetRegions([]string{"cn-hangzhou", "cn-beijing"}).Exec(ctx))
	svc := NewService(repo, nil, nopLogger())

	adapter := &fakeDiscoveryAdapter{
		provider: "aliyun", serviceCode: "ecs",
		regions: []string{"cn-hangzhou", "cn-beijing"},
		resourcesByRegion: map[string][]cloud.DiscoveredResource{
			"cn-hangzhou": {{BaseResource: cloud.BaseResource{ResourceID: "hz-1", Region: "cn-hangzhou", Status: "Running"}, CloudServiceCode: "ecs"}},
			"cn-beijing":  {{BaseResource: cloud.BaseResource{ResourceID: "bj-1", Region: "cn-beijing", Status: "Running"}, CloudServiceCode: "ecs"}},
		},
	}
	executor := NewDiscoveryJobExecutor(repo, &fakeRegistry{adapter: adapter}, &fakeResolver{}, nil, nopLogger())

	first, err := svc.CreateDiscoveryJob(ctx, tenantID, 9, &dto.DiscoveryJobRequest{SourceID: "ds-disc-multiregion"})
	require.NoError(t, err)
	require.NoError(t, executor.Execute(ctx, tenantID, first.ID))

	// 第二轮北京资源确实缺失，但杭州资源仍存在。缺失扫描只能推进北京资源。
	adapter.resourcesByRegion["cn-beijing"] = nil
	second, err := svc.CreateDiscoveryJob(ctx, tenantID, 9, &dto.DiscoveryJobRequest{SourceID: "ds-disc-multiregion"})
	require.NoError(t, err)
	require.NoError(t, executor.Execute(ctx, tenantID, second.ID))

	resources, err := repo.ListCloudResourcesByAccountService(ctx, tenantID, accountID, "ecs")
	require.NoError(t, err)
	require.Len(t, resources, 2)
	for _, resource := range resources {
		switch resource.ResourceID {
		case "hz-1":
			require.Zero(t, resource.MissingCount, "杭州资源不得因北京 scope 被判缺失")
			require.Equal(t, "active", resource.LifecycleState)
		case "bj-1":
			require.Equal(t, 1, resource.MissingCount)
			require.Equal(t, "active", resource.LifecycleState)
		default:
			t.Fatalf("unexpected resource %s", resource.ResourceID)
		}
	}
}

// 取消与重试语义：queued 可直接取消；终态再取消 409；failed 可派生重试任务。
func TestDiscoveryJobCancelAndRetry(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:discovery_cancel?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	repo := NewEntRepository(client)

	tenantID, _, _ := seedDiscoveryEnv(t, repo, "disc-cancel")
	svc := NewService(repo, nil, nopLogger())

	// queued → cancel 直接终态。
	job, err := svc.CreateDiscoveryJob(ctx, tenantID, 1, &dto.DiscoveryJobRequest{SourceID: "ds-disc-cancel"})
	require.NoError(t, err)
	cancelled, err := svc.CancelDiscoveryJob(ctx, tenantID, job.ID)
	require.NoError(t, err)
	require.Equal(t, DiscoveryJobCancelled, cancelled.Status)

	// 终态再取消 → 409。
	_, err = svc.CancelDiscoveryJob(ctx, tenantID, job.ID)
	require.Error(t, err)
	appErr, ok := err.(*common.AppError)
	require.True(t, ok)
	require.Equal(t, common.ErrCodeConflict, appErr.Code)

	// succeeded → retry 409。
	_, err = svc.RetryDiscoveryJob(ctx, tenantID, 1, job.ID)
	require.Error(t, err)

	// failed → retry 生成新任务（parent_job_id 溯源）。
	failed, err := svc.CreateDiscoveryJob(ctx, tenantID, 1, &dto.DiscoveryJobRequest{SourceID: "ds-disc-cancel"})
	require.NoError(t, err)
	fetched, err := repo.GetDiscoveryJob(ctx, tenantID, failed.ID)
	require.NoError(t, err)
	now := time.Now()
	_, err = ClaimDiscoveryJob(fetched, "worker-1", now, time.Minute)
	require.NoError(t, err)
	require.NoError(t, TransitionDiscoveryJob(fetched, DiscoveryJobFailed, now))
	fetched.ErrorMessage = "模拟失败"
	require.NoError(t, repo.UpdateDiscoveryJob(ctx, fetched))

	retried, err := svc.RetryDiscoveryJob(ctx, tenantID, 1, failed.ID)
	require.NoError(t, err)
	require.Equal(t, DiscoveryJobQueued, retried.Status)
	require.Equal(t, failed.ID, retried.ParentJobID)
	require.NotEqual(t, failed.ID, retried.ID)
}

// 幂等键：同键重复创建 409 并携带既有任务 ID。
func TestDiscoveryJobIdempotencyConflict(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:discovery_idem?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	repo := NewEntRepository(client)
	tenantID, _, _ := seedDiscoveryEnv(t, repo, "disc-idem")
	svc := NewService(repo, nil, nopLogger())

	first, err := svc.CreateDiscoveryJob(ctx, tenantID, 1, &dto.DiscoveryJobRequest{SourceID: "ds-disc-idem", IdempotencyKey: "key-1"})
	require.NoError(t, err)

	_, err = svc.CreateDiscoveryJob(ctx, tenantID, 1, &dto.DiscoveryJobRequest{SourceID: "ds-disc-idem", IdempotencyKey: "key-1"})
	require.Error(t, err)
	appErr, ok := err.(*common.AppError)
	require.True(t, ok)
	require.Equal(t, common.ErrCodeConflict, appErr.Code)
	require.Contains(t, appErr.Detail, "job=")
	require.NotEqual(t, 0, first.ID)
}
