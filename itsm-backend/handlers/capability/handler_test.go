package capability

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestHandlerReturnsTenantAwareCapabilityContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("ITSM_FEISHU_READY", "false")

	router := gin.New()
	router.GET("/capabilities", func(c *gin.Context) {
		c.Set("tenant_id", 7)
		c.Set("role", "admin")
		Handler(c)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/capabilities", nil))
	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Code int `json:"code"`
		Data struct {
			Items []Capability `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Zero(t, response.Code)
	require.NotEmpty(t, response.Data.Items)

	byKey := make(map[string]Capability)
	for _, item := range response.Data.Items {
		byKey[item.Key] = item
	}
	require.Equal(t, MaturityPilot, byKey["marketplace"].Maturity)
	require.Equal(t, []string{"read"}, byKey["marketplace"].AllowedActions)
	require.False(t, byKey["connector.feishu"].DeploymentReady)
	require.NotEmpty(t, byKey["connector.feishu"].DegradedReason)
	require.Equal(t, MaturityDisabled, byKey["identity.oidc"].Maturity)
	require.Empty(t, byKey["identity.oidc"].AllowedActions)
}

func TestHandlerDoesNotGrantActionsWithoutTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/capabilities", Handler)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/capabilities", nil))

	var response struct {
		Data struct {
			Items []Capability `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	for _, item := range response.Data.Items {
		require.False(t, item.TenantReady)
		require.Empty(t, item.AllowedActions)
	}
}
