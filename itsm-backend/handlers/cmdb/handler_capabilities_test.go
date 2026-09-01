package cmdb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type capabilityAdapterInspector struct{ ready bool }

func (i capabilityAdapterInspector) HasAdapter(provider, serviceCode string) bool {
	return i.ready && provider == "aliyun" && serviceCode == "ecs"
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
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", 42)
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
	assert.NotContains(t, recorder.Body.String(), `"code":0`)
	assert.NotContains(t, recorder.Body.String(), "tenant ID is required")
}
