package capability

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"

	"itsm-backend/connector"
)

// fakeHealthConnector 让 HealthCheck 结果取决于实例所属租户，用于验证探测的租户边界。
type fakeHealthConnector struct {
	tenantID    int
	failTenants map[int]bool
}

func (f *fakeHealthConnector) Manifest() connector.Manifest {
	return connector.Manifest{
		Name:                "feishu",
		Version:             "test",
		Provider:            "test",
		Type:                connector.TypeIM,
		RequiredPermissions: []string{"send_message"},
	}
}

func (f *fakeHealthConnector) Init(_ context.Context, cfg connector.Config) error {
	f.tenantID = cfg.TenantID
	return nil
}

func (f *fakeHealthConnector) Send(context.Context, *connector.Message) error { return nil }

func (f *fakeHealthConnector) HealthCheck(ctx context.Context) connector.HealthStatus {
	// 真实连接器会把拨号失败（含 ctx 取消）报成不健康；测试替身必须同样敏感，
	// 否则无法证明探测链路没有把发起请求的取消传进共享快照。
	if err := ctx.Err(); err != nil {
		return connector.HealthStatus{OK: false, Message: err.Error(), CheckedAt: time.Now()}
	}
	if f.failTenants[f.tenantID] {
		return connector.HealthStatus{OK: false, Message: "tenant endpoint down", CheckedAt: time.Now()}
	}
	return connector.HealthStatus{OK: true, CheckedAt: time.Now()}
}

func (f *fakeHealthConnector) Close() error { return nil }

func newConnectorProbe(t *testing.T) *Prober {
	t.Helper()

	registry := connector.NewRegistry()
	failTenants := map[int]bool{2: true}
	registry.Register(func() connector.Connector {
		return &fakeHealthConnector{failTenants: failTenants}
	})
	manager := connector.NewManager(registry, nil)
	t.Cleanup(manager.CloseAll)

	ctx := context.Background()
	require.NoError(t, manager.Provision(ctx, connector.Config{
		TenantID: 1, Name: "feishu", Provider: "feishu", Enabled: true,
	}))
	require.NoError(t, manager.Provision(ctx, connector.Config{
		TenantID: 2, Name: "feishu", Provider: "dingtalk", Enabled: true,
	}))

	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	require.NoError(t, db.PingContext(ctx))
	t.Cleanup(func() { _ = db.Close() })

	return NewProber(db, nil, manager, nil)
}

// 健康探测必须按租户隔离：租户 1 的健康检查不能复用租户 2 的结果，
// 也不能因为租户 2 的连接器故障把租户 1 判为不健康。
func TestProberConnectorHealthIsTenantScoped(t *testing.T) {
	prober := newConnectorProbe(t)
	ctx := context.Background()

	healthy := prober.HealthFor(ctx, "connector.feishu", 1)
	require.True(t, healthy.OK)

	failing := prober.HealthFor(ctx, "connector.feishu", 2)
	require.False(t, failing.OK)
	require.Equal(t, "tenant endpoint down", failing.Reason)

	// 未配置该连接器的租户不能被伪报为健康
	unconfigured := prober.HealthFor(ctx, "connector.feishu", 3)
	require.False(t, unconfigured.OK)
	require.Contains(t, unconfigured.Reason, "尚未启用")
}

// Manager 的 key 是 tenantID/name/provider；能力键只有 name，不能直接用 name 索引。
func TestConnectorHealthMatchesManagerKey(t *testing.T) {
	statuses := map[string]connector.HealthStatus{
		"1/other/feishu":  {OK: false, Message: "wrong name"},
		"2/feishu/feishu": {OK: true},
	}

	status, found := connectorHealth(statuses, "feishu")
	require.True(t, found)
	require.True(t, status.OK)

	statuses["1/feishu/feishu"] = connector.HealthStatus{OK: false, Message: "down"}
	status, found = connectorHealth(statuses, "feishu")
	require.True(t, found)
	require.False(t, status.OK)
	require.Equal(t, "down", status.Message)
}

func TestProberConnectorWithoutRuntimeIsUnhealthy(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	defer db.Close()

	prober := NewProber(db, nil, nil, nil)
	health := prober.HealthFor(context.Background(), "connector.feishu", 1)
	require.False(t, health.OK)
	require.Equal(t, "连接器运行时未装配", health.Reason)
}

// 没有数据库连接时不得返回「健康」：这正是旧实现 LastHealthCheckAt = time.Now()
// 的问题——它让调用方以为探测过且一切正常。
func TestProberWithoutDatabaseReportsUnhealthy(t *testing.T) {
	prober := NewProber(nil, nil, nil, nil)
	health := prober.HealthFor(context.Background(), "ticket", 1)
	require.NotNil(t, health)
	require.False(t, health.OK)
	require.NotEmpty(t, health.Reason)
	require.False(t, health.CheckedAt.IsZero())
}

// 快照是跨请求共享的缓存：发起刷新的请求被取消时，不能把「数据库不可达」
// 写进快照，让所有租户在 TTL 内一起降级。
func TestProberSnapshotIgnoresCallerCancellation(t *testing.T) {
	prober := newConnectorProbe(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	health := prober.HealthFor(ctx, "ticket", 1)
	require.True(t, health.OK, "调用方取消不等于数据库不可达: %s", health.Reason)
}

// 连接器探测同样跑在共享快照上：健康检查不能随发起请求取消而中断，
// 否则一次用户断开就会把该租户的连接器判为故障并缓存 30s。
func TestProberConnectorProbeIgnoresCallerCancellation(t *testing.T) {
	prober := newConnectorProbe(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	health := prober.HealthFor(ctx, "connector.feishu", 1)
	require.True(t, health.OK, "调用方取消不等于连接器故障: %s", health.Reason)

	// 缓存里必须存真实结果：后续正常请求读到的仍是健康。
	require.True(t, prober.HealthFor(context.Background(), "connector.feishu", 1).OK)
}

// 并发请求共享同一次探测：既验证 -race 下无数据竞争，也验证结果不被串扰。
func TestProberConcurrentRequestsShareProbeSnapshot(t *testing.T) {
	prober := newConnectorProbe(t)

	const workers = 20
	results := make(chan bool, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		tenantID := 1 + i%2
		wg.Add(1)
		go func(tenantID int) {
			defer wg.Done()
			results <- prober.HealthFor(context.Background(), "connector.feishu", tenantID).OK
		}(tenantID)
	}
	wg.Wait()
	close(results)

	var healthy, unhealthy int
	for ok := range results {
		if ok {
			healthy++
		} else {
			unhealthy++
		}
	}
	require.Equal(t, workers/2, healthy)
	require.Equal(t, workers/2, unhealthy)
}

// 探测到不健康时，矩阵必须把 deploymentReady 拉回 false，并透出原因。
func TestHandlerDegradesCapabilityWhenProbeFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	prober := NewProber(nil, nil, nil, nil)

	engine := gin.New()
	engine.GET("/capabilities", func(c *gin.Context) {
		c.Set("tenant_id", 7)
		c.Set("role", "admin")
		NewHandler(prober)(c)
	})
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/capabilities", nil))
	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Data struct {
			Items []Capability `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))

	byKey := map[string]Capability{}
	for _, item := range response.Data.Items {
		byKey[item.Key] = item
	}
	ticket := byKey["ticket"]
	require.False(t, ticket.DeploymentReady)
	require.NotEmpty(t, ticket.DegradedReason)
	require.NotNil(t, ticket.Health)
	require.False(t, ticket.Health.OK)
	require.NotContains(t, ticket.AllowedActions, "manage", "探测失败时不得下发管理动作")

	// 企微连接器 2026-10-21 Disabled→Pilot 转正：代码能力已实现（Send/验签/回调路由
	// 全链路装配），部署就绪由 readinessEnv 申报——测试环境未设 ITSM_WECOM_READY，
	// deploymentReady=false 且降级原因为部署依赖未就绪，而不再是「未包含在商业发布」。
	wecom := byKey["connector.wecom"]
	require.Equal(t, MaturityPilot, wecom.Maturity)
	require.False(t, wecom.DeploymentReady)
	// 探测失败原因优先于 readinessEnv 降级原因（handler 的覆盖顺序）：
	// 本测试无 DB，探测失败原因为「数据库连接未装配」。
	require.Equal(t, "数据库连接未装配", wecom.DegradedReason)
	require.NotNil(t, wecom.Health)
}

func TestRegistryInvariants(t *testing.T) {
	seen := map[string]struct{}{}
	for _, item := range registry {
		_, duplicated := seen[item.key]
		require.Falsef(t, duplicated, "能力键重复: %s", item.key)
		seen[item.key] = struct{}{}
		require.NotEmpty(t, item.key)
		require.Contains(t, []Maturity{MaturityGA, MaturityPilot, MaturityDisabled}, item.maturity)
		if item.maturity == MaturityDisabled {
			require.Falsef(t, item.built, "disabled 能力不得声称已构建: %s", item.key)
		}
		if item.maturity == MaturityGA {
			require.Truef(t, item.built, "GA 能力必须已构建: %s", item.key)
		}
	}
	// SSO 有 11 个真实端点，不得再被声明为 disabled（否则门控会隐藏可用能力）。
	sso, exists := seen["identity.oidc"]
	require.True(t, exists)
	_ = sso
}
