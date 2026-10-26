package cmdb

// 云发现「人工补全入口」的行为锁定：
//   - 待补全队列的排除口径必须与发现回写一致（未映射 / 缺 CI 类型 / 已建账 / 必填可被标签或默认值满足），
//     否则队列会把不该人补的东西也推给人；
//   - 补录建账仍只有 ConfigurationItemService.CreateCI 一个规则所有者（必填、发号、历史）；
//   - 跨租户读取与写入 fail closed，且经过真实 handler 入口证明整条链接通。

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/cloudaccount"
	"itsm-backend/ent/cloudservice"
	"itsm-backend/ent/configurationitem"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type completionFixture struct {
	client    *ent.Client
	repo      *EntRepository
	projector *DiscoveryCIProjector
	svc       *Service
	tenantA   int
	tenantB   int
	ciTypeA   *ent.CIType
}

func newCompletionFixture(t *testing.T) *completionFixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:discovery_manual_completion?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()
	repo := NewEntRepository(client)

	tenantA, _, _ := seedDiscoveryEnv(t, repo, "mc-a")
	tenantB, _, _ := seedDiscoveryEnv(t, repo, "mc-b")

	// 两个租户各有一套 cloud_vm + 必填 responsible；A 额外给「带默认值」的 cost_center，
	// 用它钉住「默认值可自动满足，不进补全队列」这条与 NormalizeAttributes 对齐的口径。
	ciTypeA := seedCompletionCIType(ctx, t, client, tenantA, "responsible", "cost_center")
	seedCompletionCIType(ctx, t, client, tenantB, "responsible", "")

	ciSvc := service.NewConfigurationItemService(client, logger,
		service.NewCIHistoryService(client, logger), service.NewCITagService(client, logger))
	projector := NewDiscoveryCIProjector(repo, ciSvc, ciSvc, logger)
	return &completionFixture{
		client:    client,
		repo:      repo,
		projector: projector,
		svc:       NewServiceWithDiscoveryRuntime(repo, nil, nopLogger(), DiscoveryRuntime{CIProjector: projector}),
		tenantA:   tenantA,
		tenantB:   tenantB,
		ciTypeA:   ciTypeA,
	}
}

func seedCompletionCIType(ctx context.Context, t *testing.T, client *ent.Client, tenantID int, requiredName, defaultedName string) *ent.CIType {
	t.Helper()
	ciType, err := client.CIType.Create().SetName("cloud_vm").SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	_, err = client.CIAttributeDefinition.Create().
		SetName(requiredName).SetDisplayName("负责人").SetType("string").
		SetRequired(true).SetCiTypeID(ciType.ID).SetTenantID(tenantID).Save(ctx)
	require.NoError(t, err)
	if defaultedName != "" {
		_, err = client.CIAttributeDefinition.Create().
			SetName(defaultedName).SetDisplayName("成本中心").SetType("string").
			SetRequired(true).SetDefaultValue("platform").
			SetCiTypeID(ciType.ID).SetTenantID(tenantID).Save(ctx)
		require.NoError(t, err)
	}
	return ciType
}

// seedResource 按 seedPendingUpdate 的外键口径直接造一条已落库的云资源。
func (f *completionFixture) seedResource(t *testing.T, tenantID int, serviceCode, resourceType, resourceID string, tags map[string]string) *CloudResource {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	serviceID := 0
	svcDef, err := f.client.CloudService.Query().
		Where(cloudservice.TenantID(tenantID)).
		First(ctx)
	require.NoError(t, err)
	serviceID = svcDef.ID
	account, err := f.client.CloudAccount.Query().
		Where(cloudaccount.TenantID(tenantID)).
		First(ctx)
	require.NoError(t, err)

	resource, err := f.repo.CreateCloudResource(ctx, &CloudResource{
		CloudAccountID: account.ID, ServiceID: serviceID, ResourceID: resourceID, IdentityVersion: 1,
		Provider: "aliyun", Partition: "public", CanonicalAccountID: "acc-mc",
		ResourceScope: "cn-hangzhou", ServiceCode: serviceCode, ResourceType: resourceType,
		IdentityHash: "mc-" + resourceID, SourceID: "ds-mc", SourceFingerprint: "fp",
		ResourceName: "res-" + resourceID, Region: "cn-hangzhou",
		Status: "running", Tags: tags,
		FirstSeenAt: &now, LastSeenAt: &now, LifecycleState: "active", TenantID: tenantID,
	})
	require.NoError(t, err)
	return resource
}

func (f *completionFixture) ciCount(t *testing.T, tenantID int) int {
	t.Helper()
	count, err := f.client.ConfigurationItem.Query().
		Where(configurationitem.TenantIDEQ(tenantID)).Count(context.Background())
	require.NoError(t, err)
	return count
}

// TestListPendingExcludesPerProjectionRules 钉住队列口径：只有「映射了类型、类型存在、
// 未建账、且必填确实凑不齐」的资源才是待补全。
func TestListPendingExcludesPerProjectionRules(t *testing.T) {
	f := newCompletionFixture(t)
	ctx := context.Background()

	need := f.seedResource(t, f.tenantA, "ecs", "instance", "i-need", nil)
	tagged := f.seedResource(t, f.tenantA, "ecs", "instance", "i-tagged", map[string]string{"responsible": "alice"})
	unmapped := f.seedResource(t, f.tenantA, "rds", "instance", "i-rds", nil)
	typeMissing := f.seedResource(t, f.tenantA, "pod", "pod", "i-pod", nil)
	linked := f.seedResource(t, f.tenantA, "ecs", "instance", "i-linked", nil)
	f.seedResource(t, f.tenantB, "ecs", "instance", "i-b1", nil)

	_, err := f.svc.RegisterPendingDiscoveryResource(ctx, f.tenantA, linked.ID, 9, ManualCompletionRequest{
		Attributes: map[string]interface{}{"responsible": "bob"},
	})
	require.NoError(t, err)

	pending, err := f.svc.ListPendingDiscoveryResources(ctx, f.tenantA)
	require.NoError(t, err)
	require.Len(t, pending, 1, "只有必填凑不齐的 i-need 该进队列")
	item := pending[0]
	require.Equal(t, need.ID, item.Resource.ID)
	require.Equal(t, "cloud_vm", item.TargetCITypeName)
	require.Equal(t, []string{"responsible"}, item.MissingAttributeNames(), "带默认值的 cost_center 不算缺口")

	var satisfiedFlagged bool
	for _, attr := range item.Attributes {
		if attr.Name == "cost_center" {
			require.True(t, attr.Satisfied, "默认值必须被判为满足")
			satisfiedFlagged = true
		}
	}
	require.True(t, satisfiedFlagged)

	// 已建账 / 已满足 / 未映射 / 缺类型都不该在队列里
	for _, e := range []*CloudResource{tagged, unmapped, typeMissing, linked} {
		for _, got := range pending {
			require.NotEqual(t, e.ID, got.Resource.ID, "%s 不该进待补全队列", e.ResourceID)
		}
	}

	// 租户 B 的队列与 A 完全隔离
	pendingB, err := f.svc.ListPendingDiscoveryResources(ctx, f.tenantB)
	require.NoError(t, err)
	require.Len(t, pendingB, 1)
	require.Equal(t, "i-b1", pendingB[0].Resource.ResourceID)
}

func TestCompleteManuallyBuildsLinkedCI(t *testing.T) {
	f := newCompletionFixture(t)
	ctx := context.Background()
	res := f.seedResource(t, f.tenantA, "ecs", "instance", "i-need", map[string]string{"env": "prod"})

	created, err := f.svc.RegisterPendingDiscoveryResource(ctx, f.tenantA, res.ID, 9, ManualCompletionRequest{
		Name:        "web-manual",
		Environment: "production",
		AssignedTo:  "zhangsan",
		Attributes:  map[string]interface{}{"responsible": "张三", "env": "manual-wins"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.CINumber, "编号必须由 CI 域发出")

	ci, err := f.client.ConfigurationItem.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, f.ciTypeA.ID, ci.CiTypeID)
	require.Equal(t, "web-manual", ci.Name, "人工命名覆盖资源名推断")
	require.Equal(t, "production", ci.Environment)
	require.Equal(t, "zhangsan", ci.AssignedTo)
	require.Equal(t, "张三", ci.Attributes["responsible"])
	require.Equal(t, "manual-wins", ci.Attributes["env"], "同名键人工提交覆盖标签推断")
	require.Equal(t, "platform", ci.Attributes["cost_center"], "默认值由 CI 域补齐")
	require.Equal(t, res.ID, ci.CloudResourceRefID, "建账即与云资源建立反查链接")
	require.Equal(t, "discovery", ci.Source)

	// 建账后从队列消失，复跑不产生第二本账
	pending, err := f.svc.ListPendingDiscoveryResources(ctx, f.tenantA)
	require.NoError(t, err)
	require.Empty(t, pending)
	require.Equal(t, 1, f.ciCount(t, f.tenantA))
}

func TestCompleteManuallyFailureSemantics(t *testing.T) {
	f := newCompletionFixture(t)
	ctx := context.Background()

	missing := f.seedResource(t, f.tenantA, "ecs", "instance", "i-missing", nil)
	unmapped := f.seedResource(t, f.tenantA, "rds", "instance", "i-rds", nil)
	typeMissing := f.seedResource(t, f.tenantA, "pod", "pod", "i-pod", nil)
	linked := f.seedResource(t, f.tenantA, "ecs", "instance", "i-linked", map[string]string{"responsible": "bob"})
	_, err := f.svc.RegisterPendingDiscoveryResource(ctx, f.tenantA, linked.ID, 9, ManualCompletionRequest{
		Attributes: map[string]interface{}{"responsible": "bob"},
	})
	require.NoError(t, err)

	// 必填仍然不齐 → 400 校验语义，且不能留下半成品配置项
	_, err = f.svc.RegisterPendingDiscoveryResource(ctx, f.tenantA, missing.ID, 9, ManualCompletionRequest{})
	requireAppErrorStatus(t, err, http.StatusBadRequest, "必填")
	require.Equal(t, 1, f.ciCount(t, f.tenantA))

	// 未映射 → 400 并指向手工创建路径
	_, err = f.svc.RegisterPendingDiscoveryResource(ctx, f.tenantA, unmapped.ID, 9, ManualCompletionRequest{})
	requireAppErrorStatus(t, err, http.StatusBadRequest, "未登记 CI 类型映射")

	// 类型不存在 → 400，指引先建类型
	_, err = f.svc.RegisterPendingDiscoveryResource(ctx, f.tenantA, typeMissing.ID, 9, ManualCompletionRequest{})
	requireAppErrorStatus(t, err, http.StatusBadRequest, "尚未创建 CI 类型")

	// 已建账 → 409，且提示已有 CI
	_, err = f.svc.RegisterPendingDiscoveryResource(ctx, f.tenantA, linked.ID, 9, ManualCompletionRequest{})
	requireAppErrorStatus(t, err, http.StatusConflict, "已关联配置项")

	// 不存在的资源 / 跨租户资源 → 404（tenant-scoped 读取，不泄漏存在性差异）
	_, err = f.svc.RegisterPendingDiscoveryResource(ctx, f.tenantA, 999999, 9, ManualCompletionRequest{})
	requireAppErrorStatus(t, err, http.StatusNotFound, "")
	_, err = f.svc.RegisterPendingDiscoveryResource(ctx, f.tenantB, linked.ID, 9, ManualCompletionRequest{})
	requireAppErrorStatus(t, err, http.StatusNotFound, "")
	require.Equal(t, 1, f.ciCount(t, f.tenantA), "跨租户补全不得动 A 的账")
}

func TestManualCompletionProjectorNotWired(t *testing.T) {
	f := newCompletionFixture(t)
	ctx := context.Background()

	// requirements 未注入：ListPending 必须报「未装配」，不能静默返回空队列
	noReader := NewDiscoveryCIProjector(f.repo, f.projector.writer, nil, nopLogger())
	_, err := noReader.ListPending(ctx, f.tenantA)
	require.ErrorIs(t, err, ErrDiscoveryProjectorNotWired)

	// Service 未挂 projector：读写都 fail closed
	plain := NewService(f.repo, nil, nopLogger())
	_, err = plain.ListPendingDiscoveryResources(ctx, f.tenantA)
	require.ErrorIs(t, err, ErrDiscoveryProjectorNotWired)
	_, err = plain.RegisterPendingDiscoveryResource(ctx, f.tenantA, 1, 9, ManualCompletionRequest{})
	require.ErrorIs(t, err, ErrDiscoveryProjectorNotWired)
}

// ---- 真实 handler 入口：HTTP 状态、业务 code、camelCase DTO 与跨租户拒绝 ----

func newCompletionRouter(svc *Service, tenantID int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if tenantID > 0 {
			c.Set("tenant_id", tenantID)
			c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
		}
		c.Next()
	})
	h := NewHandler(svc)
	router.GET("/api/v1/cmdb/discovery/pending-resources", h.ListPendingDiscoveryResources)
	router.POST("/api/v1/cmdb/discovery/pending-resources/:id/register", h.RegisterPendingDiscoveryResource)
	return router
}

type pendingEnvelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Items []struct {
			ID                 int `json:"id"`
			TargetCiTypeID     int `json:"targetCiTypeId"`
			RequiredAttributes []struct {
				Name         string `json:"name"`
				Satisfied    bool   `json:"satisfied"`
				CurrentValue string `json:"currentValue"`
			} `json:"requiredAttributes"`
			MissingAttributeNames []string `json:"missingAttributeNames"`
		} `json:"items"`
		Total      int `json:"total"`
		Page       int `json:"page"`
		PageSize   int `json:"pageSize"`
		TotalPages int `json:"totalPages"`
	} `json:"data"`
}

func TestPendingResourcesHTTPEndpoints(t *testing.T) {
	f := newCompletionFixture(t)
	need := f.seedResource(t, f.tenantA, "ecs", "instance", "i-need", nil)
	f.seedResource(t, f.tenantB, "ecs", "instance", "i-b1", nil)

	// 列表：标准信封 + camelCase 契约字段
	recorder := httptest.NewRecorder()
	newCompletionRouter(f.svc, f.tenantA).ServeHTTP(recorder,
		httptest.NewRequest(http.MethodGet, "/api/v1/cmdb/discovery/pending-resources?page=1&pageSize=20", nil))
	require.Equal(t, http.StatusOK, recorder.Code, "body=%s", recorder.Body.String())
	var env pendingEnvelope
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &env))
	require.Equal(t, 0, env.Code)
	require.Equal(t, 1, env.Data.Total)
	require.Equal(t, 1, env.Data.TotalPages)
	require.Len(t, env.Data.Items, 1)
	require.Equal(t, need.ID, env.Data.Items[0].ID)
	require.Equal(t, f.ciTypeA.ID, env.Data.Items[0].TargetCiTypeID)
	require.Equal(t, []string{"responsible"}, env.Data.Items[0].MissingAttributeNames)

	// 越界页：total 不变，items 为空——分页不能伪造总数
	recorder = httptest.NewRecorder()
	newCompletionRouter(f.svc, f.tenantA).ServeHTTP(recorder,
		httptest.NewRequest(http.MethodGet, "/api/v1/cmdb/discovery/pending-resources?page=99", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &env))
	require.Equal(t, 1, env.Data.Total)
	require.Empty(t, env.Data.Items)

	// 跨租户读取：B 看不到 A 的队列
	recorder = httptest.NewRecorder()
	newCompletionRouter(f.svc, f.tenantB).ServeHTTP(recorder,
		httptest.NewRequest(http.MethodGet, "/api/v1/cmdb/discovery/pending-resources", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &env))
	require.Equal(t, 1, env.Data.Total)
	require.NotEqual(t, need.ID, env.Data.Items[0].ID)

	// 跨租户写入：租户 B 尝试补全 A 的资源 → 404 / 4004，且 A 的配置项数量不动
	require.Equal(t, 0, f.ciCount(t, f.tenantA))
	body := strings.NewReader(`{"attributes":{"responsible":"eve"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cmdb/discovery/pending-resources/"+itoa(need.ID)+"/register", body)
	req.Header.Set("Content-Type", "application/json")
	recorder = httptest.NewRecorder()
	newCompletionRouter(f.svc, f.tenantB).ServeHTTP(recorder, req)
	require.Equal(t, http.StatusNotFound, recorder.Code, "body=%s", recorder.Body.String)
	require.Contains(t, recorder.Body.String(), `"code":4004`)
	require.Equal(t, 0, f.ciCount(t, f.tenantA))

	// 本租户补全：200 + 建出的配置项
	body = strings.NewReader(`{"name":"web-http","attributes":{"responsible":"张三"}}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/cmdb/discovery/pending-resources/"+itoa(need.ID)+"/register", body)
	req.Header.Set("Content-Type", "application/json")
	recorder = httptest.NewRecorder()
	newCompletionRouter(f.svc, f.tenantA).ServeHTTP(recorder, req)
	require.Equal(t, http.StatusOK, recorder.Code, "body=%s", recorder.Body.String)
	require.Contains(t, recorder.Body.String(), `"ciNumber"`)
	require.Equal(t, 1, f.ciCount(t, f.tenantA))

	// 必填仍不齐：400 / 4000 + 中文文案，不是 500
	need2 := f.seedResource(t, f.tenantA, "ecs", "instance", "i-need2", nil)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/cmdb/discovery/pending-resources/"+itoa(need2.ID)+"/register",
		strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	recorder = httptest.NewRecorder()
	newCompletionRouter(f.svc, f.tenantA).ServeHTTP(recorder, req)
	require.Equal(t, http.StatusBadRequest, recorder.Code, "body=%s", recorder.Body.String)
	require.Contains(t, recorder.Body.String(), `"code":4000`)
	require.Contains(t, recorder.Body.String(), "必填")

	// 回写未装配：503 / 5003，不能伪装成空队列
	recorder = httptest.NewRecorder()
	newCompletionRouter(NewService(f.repo, nil, nopLogger()), f.tenantA).ServeHTTP(recorder,
		httptest.NewRequest(http.MethodGet, "/api/v1/cmdb/discovery/pending-resources", nil))
	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"code":5003`)

	// 缺租户上下文：fail closed（ResolveTenantID 自行写出错误响应）
	recorder = httptest.NewRecorder()
	newCompletionRouter(f.svc, 0).ServeHTTP(recorder,
		httptest.NewRequest(http.MethodGet, "/api/v1/cmdb/discovery/pending-resources", nil))
	require.NotEqual(t, http.StatusOK, recorder.Code)
}

func requireAppErrorStatus(t *testing.T, err error, status int, wantMessage string) {
	t.Helper()
	require.Error(t, err)
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr), "expect *common.AppError, got %T: %v", err, err)
	require.Equal(t, status, appErr.HTTPStatus, "err=%v", err)
	if wantMessage != "" {
		require.Contains(t, appErr.Error()+appErr.Detail, wantMessage)
	}
}
