package cmdb

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type readyAdapterInspector struct{}

func (readyAdapterInspector) HasAdapter(provider, serviceCode string) bool {
	return provider == "aliyun" && serviceCode == "ecs"
}

func TestDiscoveryHealthReportsInitializedWorker(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := NewServiceWithDiscoveryRuntime(nil, nil, zap.NewNop().Sugar(), DiscoveryRuntime{
		Adapters: readyAdapterInspector{}, CredentialResolverReady: true, WorkerReady: true,
	})
	r := gin.New()
	r.GET("/api/v1/cmdb/discovery/health", NewHandler(service).GetDiscoveryHealth)

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/cmdb/discovery/health", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"code":0,"message":"success","data":{"key":"cmdbDiscovery","state":"ready","buildCapability":true,"deploymentReadiness":true,"tenantReadiness":false,"actorPermission":true,"missingRequirements":[]}}`, recorder.Body.String())
}
