package cmdb

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"itsm-backend/middleware"
	"itsm-backend/service/cloud"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// withTestTenant 注入租户上下文，模拟已通过认证中间件的请求。
func withTestTenant(tenantID int) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
		c.Next()
	}
}

type readyAdapterInspector struct{}

func (readyAdapterInspector) HasAdapter(provider, serviceCode string) bool {
	return provider == "aliyun" && serviceCode == "ecs"
}

// SupportedProviders 声明本替身「装了什么」：只有阿里云 ECS。
// 必须与 HasAdapter 说同一件事——两处口径不一致正是能力清单会撒谎的根源。
func (readyAdapterInspector) SupportedProviders() []cloud.ProviderSupport {
	return []cloud.ProviderSupport{{
		Provider:     "aliyun",
		DisplayName:  cloud.ProviderDisplayName("aliyun"),
		ServiceCodes: []string{"ecs"},
	}}
}

func TestDiscoveryHealthReportsInitializedWorker(t *testing.T) {
	// 部署级开关是 fail-closed 的：装配齐全但未显式申报 ITSM_CMDB_DISCOVERY_READY
	// 时仍应报未就绪，否则域内面会承诺一条运维刻意关闭的链路。
	t.Setenv(discoveryReadinessEnv, "true")

	gin.SetMode(gin.TestMode)
	service := NewServiceWithDiscoveryRuntime(nil, nil, zap.NewNop().Sugar(), DiscoveryRuntime{
		Adapters: readyAdapterInspector{}, CredentialResolverReady: true, WorkerReady: true,
	})
	// 健康面现在与域内其余端点同口径，要求租户上下文。
	// 无认证请求不应能探测本实例的发现链路装配情况。
	r := gin.New()
	r.Use(withTestTenant(1))
	r.GET("/api/v1/cmdb/discovery/health", NewHandler(service).GetDiscoveryHealth)

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/cmdb/discovery/health", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	// 断言里刻意没有 tenantReadiness：健康端点不查租户账号，算不出这个值。
	// 此前它把该字段留成 Go 零值下发，于是同一响应同时说
	// state=ready / missingRequirements=[] / tenantReadiness=false，三句话互相打脸。
	require.JSONEq(t, `{"code":0,"message":"success","data":{"key":"cmdbDiscovery","state":"ready","buildCapability":true,"deploymentReadiness":true,"actorPermission":true,"missingRequirements":[]}}`, recorder.Body.String())
}

// 健康面与能力面必须对「部署装齐了没有」给同一结论。此前两个端点各写一份同义判断，
// 结果是同一部署在 /discovery/health 与 /cmdb/capabilities 上得出不同答案；调用方
// 无从判断该按哪一条去修。本用例把两处并排比对，任一侧漂移都会红。
func TestDiscoveryHealthAgreesWithCapabilityDeploymentReadiness(t *testing.T) {
	type stage struct {
		name     string
		env      string
		adapters DiscoveryAdapterInspector
		resolver bool
		worker   bool
	}

	stages := []stage{
		{"全齐且已申报", "true", readyAdapterInspector{}, true, true},
		{"未申报", "", readyAdapterInspector{}, true, true},
		{"缺 Worker", "true", readyAdapterInspector{}, true, false},
		{"缺解析器", "true", readyAdapterInspector{}, false, true},
		{"未装适配器", "true", nil, true, true},
	}

	gin.SetMode(gin.TestMode)
	for _, tc := range stages {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(discoveryReadinessEnv, tc.env)

			svc := NewServiceWithDiscoveryRuntime(&mockRepository{}, nil, zap.NewNop().Sugar(), DiscoveryRuntime{
				Adapters: tc.adapters, CredentialResolverReady: tc.resolver, WorkerReady: tc.worker,
			})
			health := svc.GetDiscoveryHealth()

			// 租户侧无账号是刻意的：本用例只比对部署维度，租户维度由
			// /cmdb/capabilities 报，不应污染部署结论。
			capability, err := svc.GetDiscoveryCapability(t.Context(), 42)
			require.NoError(t, err)

			require.Equal(t, capability.DeploymentReadiness, health.DeploymentReadiness,
				"部署就绪结论必须一致，否则同一条能力给出两种部署事实")
			require.Equal(t, capability.State == "ready" || capability.State == "unconfigured", health.State == "ready",
				"state 语义需自洽：部署未齐时两侧都不得声称 ready")
			// 缺失项里部署级那几条必须一致（租户项只在能力面出现）
			for _, want := range health.MissingRequirements {
				require.Contains(t, capability.MissingRequirements, want,
					"健康面报的缺失项在能力面也必须能看到")
			}
		})
	}
}

// 健康响应不得再下发 tenantReadiness。这个字段一旦回来，就必然带着一个猜出来的值。
func TestDiscoveryHealthOmitsTenantDimension(t *testing.T) {
	t.Setenv(discoveryReadinessEnv, "true")

	gin.SetMode(gin.TestMode)
	service := NewServiceWithDiscoveryRuntime(nil, nil, zap.NewNop().Sugar(), DiscoveryRuntime{
		Adapters: readyAdapterInspector{}, CredentialResolverReady: true, WorkerReady: true,
	})
	// 健康面现在与域内其余端点同口径，要求租户上下文。
	// 无认证请求不应能探测本实例的发现链路装配情况。
	r := gin.New()
	r.Use(withTestTenant(1))
	r.GET("/api/v1/cmdb/discovery/health", NewHandler(service).GetDiscoveryHealth)

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/cmdb/discovery/health", nil))
	require.Equal(t, http.StatusOK, recorder.Code)

	var body struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	_, present := body.Data["tenantReadiness"]
	require.False(t, present,
		"健康端点算不出租户维度，不能下发该字段（留零值会说谎，填值则是猜的）")
}

// 域内其余 31 个 handler 都要求租户上下文，健康面曾漏掉这一步：
// 未认证请求因此能直接读到本实例的发现链路装配情况（缺哪个 adapter、
// 凭据解析器是否就绪、worker 是否起来了）——那是一张攻击面地图。
// 本用例锁住「无租户上下文 = 401，且一个字节的装配信息都不下发」。
func TestDiscoveryHealthRequiresTenantContext(t *testing.T) {
	t.Setenv(discoveryReadinessEnv, "true")

	gin.SetMode(gin.TestMode)
	service := NewServiceWithDiscoveryRuntime(nil, nil, zap.NewNop().Sugar(), DiscoveryRuntime{
		Adapters: readyAdapterInspector{}, CredentialResolverReady: true, WorkerReady: true,
	})
	r := gin.New()
	// 刻意不挂 withTestTenant：模拟未认证/租户上下文丢失的请求。
	r.GET("/api/v1/cmdb/discovery/health", NewHandler(service).GetDiscoveryHealth)

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/cmdb/discovery/health", nil))

	require.Equal(t, http.StatusUnauthorized, recorder.Code,
		"无租户上下文的健康探测必须 401，body=%s", recorder.Body.String())

	// 响应体不得泄露任何装配细节（missingRequirements 会点名缺哪个 adapter）
	var body struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err == nil {
		for _, leaked := range []string{"missingRequirements", "deploymentReadiness", "buildCapability", "actorPermission", "state"} {
			_, present := body.Data[leaked]
			require.False(t, present, "401 响应不得下发 %q（会泄露装配地图）", leaked)
		}
	}
}
