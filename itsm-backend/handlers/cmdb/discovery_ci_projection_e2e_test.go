package cmdb

import (
	"context"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/cloudresource"
	"itsm-backend/ent/configurationitem"
	"itsm-backend/ent/enttest"
	"itsm-backend/service"
	"itsm-backend/service/cloud"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 发现回写配置项的端到端证据：真实 CI 域（发号、属性校验、历史）+ 真实执行器 +
// sqlite，覆盖 建账 → 复跑刷新 → 人工字段不被覆盖 → 缺失标 stale → 资源回来清 stale。

func ecsResource(id, name, status string, tags map[string]string) cloud.DiscoveredResource {
	return cloud.DiscoveredResource{
		BaseResource: cloud.BaseResource{
			ResourceID: id, ResourceName: name, Region: "cn-hangzhou", Status: status, Tags: tags,
		},
		CloudServiceCode: "ecs",
	}
}

func TestDiscoveryCIProjectionEndToEnd(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:discovery_ci_projection?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()
	repo := NewEntRepository(client)

	tenantID, _, _ := seedDiscoveryEnv(t, repo, "disc-ci")
	ciType, err := client.CIType.Create().SetName("cloud_vm").SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)

	ciSvc := service.NewConfigurationItemService(client, logger,
		service.NewCIHistoryService(client, logger), service.NewCITagService(client, logger))
	projector := NewDiscoveryCIProjector(repo, ciSvc, ciSvc, logger)
	svc := NewServiceWithDiscoveryRuntime(repo, nil, nopLogger(), DiscoveryRuntime{CIProjector: projector})

	adapter := &fakeDiscoveryAdapter{
		provider: "aliyun", serviceCode: "ecs", regions: []string{"cn-hangzhou"},
		resources: []cloud.DiscoveredResource{
			ecsResource("i-1", "web-1", "Running", map[string]string{"env": "prod", "owner": "sre"}),
			ecsResource("i-2", "db-1", "Stopped", nil),
		},
	}
	executor := NewDiscoveryJobExecutor(repo, &fakeRegistry{adapter: adapter}, &fakeResolver{}, projector, nopLogger())

	runJob := func() *DiscoveryJob {
		t.Helper()
		created, err := svc.CreateDiscoveryJob(ctx, tenantID, 9, &dto.DiscoveryJobRequest{SourceID: "ds-disc-ci"})
		require.NoError(t, err)
		require.NoError(t, executor.Execute(ctx, tenantID, created.ID))
		job, err := repo.GetDiscoveryJob(ctx, tenantID, created.ID)
		require.NoError(t, err)
		return job
	}

	// 第一轮：两条 create 差异各自落成配置项，并写回 discovery_results.ci_id。
	job := runJob()
	require.EqualValues(t, 2, job.Summary["ciCreated"])
	require.EqualValues(t, 0, job.Summary["ciFailed"])
	require.EqualValues(t, 0, job.Summary["ciSkipped"])

	cis, err := client.ConfigurationItem.Query().
		Where(configurationitem.TenantIDEQ(tenantID)).
		Order(configurationitem.ByID()).All(ctx)
	require.NoError(t, err)
	require.Len(t, cis, 2)
	byCloudID := map[string]*ent.ConfigurationItem{}
	for _, ci := range cis {
		byCloudID[ci.CloudResourceID] = ci
	}
	web1 := byCloudID["i-1"]
	require.NotNil(t, web1)
	assertProjectedCI(t, web1, ciType.ID, "web-1", "active", "success")
	require.Equal(t, "production", web1.Environment, "无歧义的环境标签应落成 CI 环境")
	require.Equal(t, "sre", web1.AssignedTo)
	require.Equal(t, "prod", web1.Attributes["env"])
	require.NotEmpty(t, web1.CiNumber, "CI 编号由 CI 域发出，不是回写代码自己拼的")
	require.NotEmpty(t, web1.Source)

	db1 := byCloudID["i-2"]
	require.NotNil(t, db1)
	assertProjectedCI(t, db1, ciType.ID, "db-1", "inactive", "success")

	results, err := repo.ListDiscoveryResults(ctx, tenantID, job.ID)
	require.NoError(t, err)
	require.Len(t, results, 2)
	for _, r := range results {
		require.Equal(t, "create", r.Action)
		require.Greater(t, r.CIID, 0, "差异要能反查到落成的配置项")
	}

	// 人工补录的字段（描述/动态属性）不能被下一轮发现覆盖。
	_, err = ciSvc.UpdateCI(ctx, web1.ID, tenantID, &dto.UpdateCIRequest{
		Description: "owner 手工写的用途说明",
		Attributes:  map[string]interface{}{"env": "prod", "cost_center": "platform"},
	})
	require.NoError(t, err)

	// 第二轮：i-1 改名（自动应用 → CI 刷新）、i-2 消失（missing 1，未到阈值）。
	adapter.resources = []cloud.DiscoveredResource{
		ecsResource("i-1", "web-1-renamed", "Running", map[string]string{"env": "prod", "owner": "sre"}),
	}
	job = runJob()
	require.EqualValues(t, 1, job.Summary["ciUpdated"])
	require.EqualValues(t, 0, job.Summary["ciStaleMarked"])

	renamed, err := client.ConfigurationItem.Get(ctx, web1.ID)
	require.NoError(t, err)
	require.Equal(t, "web-1-renamed", renamed.Name)
	require.Equal(t, "owner 手工写的用途说明", renamed.Description)
	require.Equal(t, "platform", renamed.Attributes["cost_center"], "发现只回写自己拥有的字段")

	// 第三轮：i-2 再次缺失达阈值（staleThreshold=2）→ 资源 stale，配置项只标同步滞后。
	job = runJob()
	require.EqualValues(t, 1, job.Summary["ciStaleMarked"])

	staleCI, err := client.ConfigurationItem.Get(ctx, db1.ID)
	require.NoError(t, err)
	require.Equal(t, "stale", staleCI.CloudSyncStatus)
	require.Equal(t, "online", staleCI.LifecycleStatus, "退役是人的决定，缺失只标同步滞后")

	// 第四轮：i-2 回来 → 资源从 stale 回 active，配置项同步状态清回 success。
	adapter.resources = []cloud.DiscoveredResource{
		ecsResource("i-1", "web-1-renamed", "Running", map[string]string{"env": "prod", "owner": "sre"}),
		ecsResource("i-2", "db-1", "Stopped", nil),
	}
	job = runJob()
	require.EqualValues(t, 1, job.Summary["ciUpdated"])
	require.EqualValues(t, 0, job.Summary["ciStaleMarked"])

	returned, err := client.ConfigurationItem.Get(ctx, db1.ID)
	require.NoError(t, err)
	require.Equal(t, "success", returned.CloudSyncStatus)

	// 全程只有 2 条配置项：复跑没有把同一台机器建成第二本账。
	count, err := client.ConfigurationItem.Query().Where(configurationitem.TenantIDEQ(tenantID)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)
}

// TestApplyDiscoveryResultProjectsCI 覆盖 manual 对账策略的人工决策：
// 自动路径不回写（差异还是 pending），只有人点「应用」后才刷新配置项并留下反查 ID。
func TestApplyDiscoveryResultProjectsCI(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:discovery_ci_manual?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()
	repo := NewEntRepository(client)

	tenantID, _, _ := seedDiscoveryEnv(t, repo, "disc-manual")
	ciType, err := client.CIType.Create().SetName("cloud_vm").SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	require.NoError(t, client.DiscoverySource.UpdateOneID("ds-disc-manual").SetReconcilePolicy("manual").Exec(ctx))

	ciSvc := service.NewConfigurationItemService(client, logger,
		service.NewCIHistoryService(client, logger), service.NewCITagService(client, logger))
	projector := NewDiscoveryCIProjector(repo, ciSvc, ciSvc, logger)
	svc := NewServiceWithDiscoveryRuntime(repo, nil, nopLogger(), DiscoveryRuntime{CIProjector: projector})

	adapter := &fakeDiscoveryAdapter{
		provider: "aliyun", serviceCode: "ecs", regions: []string{"cn-hangzhou"},
		resources: []cloud.DiscoveredResource{ecsResource("i-9", "gate-9", "Running", nil)},
	}
	executor := NewDiscoveryJobExecutor(repo, &fakeRegistry{adapter: adapter}, &fakeResolver{}, projector, nopLogger())

	created, err := svc.CreateDiscoveryJob(ctx, tenantID, 9, &dto.DiscoveryJobRequest{SourceID: "ds-disc-manual"})
	require.NoError(t, err)
	require.NoError(t, executor.Execute(ctx, tenantID, created.ID))

	first, err := client.ConfigurationItem.Query().
		Where(configurationitem.CloudResourceIDEQ("i-9")).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, ciType.ID, first.CiTypeID)

	// 第二轮改名 → manual 策略只产出 pending 差异，配置项保持原样。
	adapter.resources = []cloud.DiscoveredResource{ecsResource("i-9", "gate-9-renamed", "Running", nil)}
	second, err := svc.CreateDiscoveryJob(ctx, tenantID, 9, &dto.DiscoveryJobRequest{SourceID: "ds-disc-manual"})
	require.NoError(t, err)
	require.NoError(t, executor.Execute(ctx, tenantID, second.ID))

	results, err := repo.ListDiscoveryResults(ctx, tenantID, second.ID)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, DiscoveryResultPending, results[0].Status)
	require.Zero(t, results[0].CIID, "未应用的差异不能先占上配置项")

	before, err := client.ConfigurationItem.Get(ctx, first.ID)
	require.NoError(t, err)
	require.Equal(t, "gate-9", before.Name)

	applied, err := svc.ApplyDiscoveryResult(ctx, tenantID, 9, results[0].ID)
	require.NoError(t, err)
	require.Equal(t, DiscoveryResultApplied, applied.Status)
	require.Equal(t, first.ID, applied.CIID)

	after, err := client.ConfigurationItem.Get(ctx, first.ID)
	require.NoError(t, err)
	require.Equal(t, "gate-9-renamed", after.Name, "人工应用要同步刷新配置项")

	persisted, err := repo.GetDiscoveryResult(ctx, tenantID, results[0].ID)
	require.NoError(t, err)
	require.Equal(t, first.ID, persisted.CIID, "反查 ID 要落库，不能只在内存里")
}

// TestDiscoveryCIProjectionDefersRequiredAttributes 证明「标签凑不齐必填项」走的是
// 待补录而不是写入失败：CI 一条不建，差异留在对账里等人补。
func TestDiscoveryCIProjectionDefersRequiredAttributes(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:discovery_ci_deferred?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()
	repo := NewEntRepository(client)

	tenantID, _, _ := seedDiscoveryEnv(t, repo, "disc-defer")
	ciType, err := client.CIType.Create().SetName("cloud_vm").SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	_, err = client.CIAttributeDefinition.Create().
		SetName("responsible").SetDisplayName("负责人").SetType("string").
		SetRequired(true).SetCiTypeID(ciType.ID).SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)

	ciSvc := service.NewConfigurationItemService(client, logger,
		service.NewCIHistoryService(client, logger), service.NewCITagService(client, logger))
	projector := NewDiscoveryCIProjector(repo, ciSvc, ciSvc, logger)
	svc := NewServiceWithDiscoveryRuntime(repo, nil, nopLogger(), DiscoveryRuntime{CIProjector: projector})

	adapter := &fakeDiscoveryAdapter{
		provider: "aliyun", serviceCode: "ecs", regions: []string{"cn-hangzhou"},
		resources: []cloud.DiscoveredResource{ecsResource("i-7", "bare-7", "Running", nil)},
	}
	executor := NewDiscoveryJobExecutor(repo, &fakeRegistry{adapter: adapter}, &fakeResolver{}, projector, nopLogger())

	created, err := svc.CreateDiscoveryJob(ctx, tenantID, 9, &dto.DiscoveryJobRequest{SourceID: "ds-disc-defer"})
	require.NoError(t, err)
	require.NoError(t, executor.Execute(ctx, tenantID, created.ID))

	job, err := repo.GetDiscoveryJob(ctx, tenantID, created.ID)
	require.NoError(t, err)
	require.Equal(t, DiscoveryJobSucceeded, job.Status, "待补录不能把发现任务判成失败")
	require.EqualValues(t, 1, job.Summary["ciDeferred"])
	require.EqualValues(t, 0, job.Summary["ciFailed"])
	require.EqualValues(t, 0, job.Summary["ciCreated"])

	count, err := client.ConfigurationItem.Query().Where(configurationitem.TenantIDEQ(tenantID)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, count, "缺必填项时不能臆造属性建账")

	// 云资源仍然落库，补录后重跑即可入账；对账页按 unboundResources 暴露它。
	resourceCount, err := client.CloudResource.Query().Where(cloudresource.TenantIDEQ(tenantID)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, resourceCount)
}

func assertProjectedCI(t *testing.T, ci *ent.ConfigurationItem, ciTypeID int, name, status, syncStatus string) {
	t.Helper()
	require.Equal(t, ciTypeID, ci.CiTypeID)
	require.Equal(t, "cloud_vm", ci.CiType)
	require.Equal(t, name, ci.Name)
	require.Equal(t, status, ci.Status)
	require.Equal(t, "discovery", ci.Source)
	require.Equal(t, "ds-disc-ci", ci.DiscoverySource)
	require.Equal(t, "aliyun", ci.CloudProvider)
	require.Equal(t, "instance", ci.CloudResourceType)
	require.Equal(t, "cn-hangzhou", ci.CloudRegion)
	require.Equal(t, syncStatus, ci.CloudSyncStatus)
	require.Greater(t, ci.CloudResourceRefID, 0, "配置项要能反查回云资源")
	require.False(t, ci.LastDiscovered.IsZero(), "回写必须带上发现时间")
}
