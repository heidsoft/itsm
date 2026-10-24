package cmdb

// 本文件锁 GET /api/v1/cmdb/discovery/providers 的三条契约：
//
//  1. 清单必须**全量**返回 cloud.KnownProviders，而不是只回「有适配器」的厂商。
//     只回后者会让新接入的厂商从建号下拉里消失（这正是 2026-10-24 实测到的缺陷：
//     火山引擎与 k8s 适配器已接入，但 dto 枚举与前端下拉都停留在六个旧值，
//     产品面根本选不到）。
//  2. 注册表未装配时不得返回空列表——那会被读成「本产品不认任何厂商」。
//  3. 缺租户上下文必须 fail closed，未认证请求不该探测出本实例支持哪些云厂商。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"itsm-backend/middleware"
	"itsm-backend/service/cloud"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// providersEnvelope 只解出关心的字段，避免断言被响应里其它键的变化打断。
type providersEnvelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Items []struct {
			Provider       string   `json:"provider"`
			DisplayName    string   `json:"displayName"`
			ServiceCodes   []string `json:"serviceCodes"`
			NotImplemented bool     `json:"notImplemented"`
		} `json:"items"`
		Total int `json:"total"`
	} `json:"data"`
}

// newProvidersRouter 复刻生产 TenantMiddleware 的上下文形状。
// withTenant=false 用于验证 fail closed。
func newProvidersRouter(t *testing.T, svc *Service, withTenant bool) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if withTenant {
			c.Set("tenant_id", 42)
			c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 42})
		}
		c.Next()
	})
	router.GET("/api/v1/cmdb/discovery/providers", NewHandler(svc).ListCloudProviders)
	return router
}

func callProviders(t *testing.T, router *gin.Engine) (int, providersEnvelope) {
	t.Helper()
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/cmdb/discovery/providers", nil))
	var env providersEnvelope
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &env), "body=%s", recorder.Body.String())
	return recorder.Code, env
}

func TestListCloudProvidersReturnsFullKnownProviderSet(t *testing.T) {
	svc := NewServiceWithDiscoveryRuntime(&mockRepository{}, nil, zap.NewNop().Sugar(), DiscoveryRuntime{
		Adapters: capabilityAdapterInspector{ready: true},
	})
	status, env := callProviders(t, newProvidersRouter(t, svc, true))

	require.Equal(t, http.StatusOK, status)
	require.Equal(t, 0, env.Code)

	// 全量：KnownProviders 有几个就回几个，不能按「有没有适配器」过滤。
	require.Len(t, env.Data.Items, len(cloud.KnownProviders),
		"清单必须是厂商全集，否则新接入的厂商会从建号下拉里消失")
	assert.Equal(t, len(cloud.KnownProviders), env.Data.Total)

	byProvider := make(map[string]struct {
		DisplayName    string
		ServiceCodes   []string
		NotImplemented bool
	}, len(env.Data.Items))
	for _, it := range env.Data.Items {
		byProvider[it.Provider] = struct {
			DisplayName    string
			ServiceCodes   []string
			NotImplemented bool
		}{it.DisplayName, it.ServiceCodes, it.NotImplemented}
	}

	for _, p := range cloud.KnownProviders {
		got, ok := byProvider[p]
		require.True(t, ok, "厂商 %q 未出现在清单里", p)
		assert.Equal(t, cloud.ProviderDisplayName(p), got.DisplayName)
	}

	// 替身只装了 aliyun/ecs：它必须有服务码且未标未接入。
	assert.Equal(t, []string{"ecs"}, byProvider["aliyun"].ServiceCodes)
	assert.False(t, byProvider["aliyun"].NotImplemented, "已装配适配器不得标为未接入")

	// 其余厂商识别得到但没适配器：要标 true，且 serviceCodes 给空数组而不是 null
	// （null 会让前端渲染时判空炸掉）。
	for _, p := range cloud.KnownProviders {
		if p == "aliyun" {
			continue
		}
		assert.True(t, byProvider[p].NotImplemented, "%q 未装配适配器，必须标为未接入", p)
		assert.NotNil(t, byProvider[p].ServiceCodes, "%q 的 serviceCodes 应为 [] 而非 null", p)
		assert.Empty(t, byProvider[p].ServiceCodes)
	}
}

func TestListCloudProvidersWithoutRegistryStillListsAllProviders(t *testing.T) {
	// 注册表未装配（Adapters 为 nil）：清单仍要全量返回，只是全部标未接入。
	svc := NewService(&mockRepository{}, nil, zap.NewNop().Sugar())
	status, env := callProviders(t, newProvidersRouter(t, svc, true))

	require.Equal(t, http.StatusOK, status)
	require.Len(t, env.Data.Items, len(cloud.KnownProviders),
		"注册表未装配时返回空清单会被读成「本产品不认任何厂商」")
	for _, it := range env.Data.Items {
		assert.True(t, it.NotImplemented)
		assert.NotNil(t, it.ServiceCodes)
	}
}

func TestListCloudProvidersFailsClosedWithoutTenantContext(t *testing.T) {
	svc := NewServiceWithDiscoveryRuntime(&mockRepository{}, nil, zap.NewNop().Sugar(), DiscoveryRuntime{
		Adapters: capabilityAdapterInspector{ready: true},
	})
	status, env := callProviders(t, newProvidersRouter(t, svc, false))

	// 清单不含租户数据，但仍要求租户上下文：未认证请求不应探测出本实例支持哪些厂商。
	require.Equal(t, http.StatusUnauthorized, status)
	assert.NotEqual(t, 0, env.Code)
	assert.Empty(t, env.Data.Items)
}
