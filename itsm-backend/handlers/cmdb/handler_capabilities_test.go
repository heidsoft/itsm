package cmdb

import (
	"context"
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

type capabilityAdapterInspector struct{ ready bool }

func (i capabilityAdapterInspector) HasAdapter(provider, serviceCode string) bool {
	return i.ready && provider == "aliyun" && serviceCode == "ecs"
}

// SupportedProviders 与 HasAdapter 同源：ready=false 时一个厂商都拿不到适配器，
// 但清单仍须按 KnownProviders 全量返回（未接入标 true），不能返回空切片——
// 空切片会被读成「本产品不认任何厂商」。
func (i capabilityAdapterInspector) SupportedProviders() []cloud.ProviderSupport {
	out := make([]cloud.ProviderSupport, 0, len(cloud.KnownProviders))
	for _, p := range cloud.KnownProviders {
		codes := []string{}
		if i.ready && p == "aliyun" {
			codes = []string{"ecs"}
		}
		out = append(out, cloud.ProviderSupport{
			Provider:       p,
			DisplayName:    cloud.ProviderDisplayName(p),
			ServiceCodes:   codes,
			NotImplemented: len(codes) == 0,
		})
	}
	return out
}

func TestGetCapabilitiesUsesAuthenticatedTenantAndReportsUnreadyRuntime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	capturedTenantID := 0
	repo := &mockRepository{listCloudAccountsFn: func(_ context.Context, tenantID int, provider string) ([]*CloudAccount, error) {
		capturedTenantID = tenantID
		require.Equal(t, "aliyun", provider)
		return []*CloudAccount{{TenantID: tenantID, Provider: provider, IsActive: true, CredentialRef: "secret://tenant-42/aliyun-prod"}}, nil
	}}
	svc := NewServiceWithDiscoveryRuntime(repo, nil, zap.NewNop().Sugar(), DiscoveryRuntime{
		Adapters: capabilityAdapterInspector{ready: true},
		// The worker and tenant secret resolver intentionally remain unavailable.
		CredentialResolverReady: false,
		WorkerReady:             false,
	})
	router := gin.New()
	// 与生产 TenantMiddleware 一致：handler 通过 tenant_context 解析租户，
	// 只设扁平 tenant_id 不算有认证上下文。
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", 42)
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 42})
		c.Next()
	})
	router.GET("/api/v1/cmdb/capabilities", NewHandler(svc).GetCapabilities)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/cmdb/capabilities", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, 42, capturedTenantID)
	assert.JSONEq(t, `{
		"code": 0,
		"message": "success",
		"data": {
			"items": [{
				"key": "cmdbDiscovery",
				"state": "unready",
				"buildCapability": true,
				"deploymentReadiness": false,
				"tenantReadiness": true,
				"actorPermission": true,
				"missingRequirements": ["tenantSecretResolver", "discoveryWorker"]
			}]
		}
	}`, recorder.Body.String())
}

func TestGetCapabilitiesFailsClosedWithoutTenantContext(t *testing.T) {
	router := gin.New()
	router.GET("/api/v1/cmdb/capabilities", NewHandler(NewService(&mockRepository{}, nil, zap.NewNop().Sugar())).GetCapabilities)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/cmdb/capabilities", nil))

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	// 2001 是 tenant_context 缺失的统一契约码；本端点不得自带私有错误码，
	// 否则与其他 CMDB 端点的 fail-closed 语义不一致。
	assert.Contains(t, recorder.Body.String(), `"code":2001`)
	assert.NotContains(t, recorder.Body.String(), `"code":0`)
	assert.NotContains(t, recorder.Body.String(), "tenant ID is required")
}
