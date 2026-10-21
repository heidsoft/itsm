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
		NewHandler(nil)(c)
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
	// SSO/OIDC 已实现 11 个端点，矩阵不得再声明为 disabled。
	require.Equal(t, MaturityPilot, byKey["identity.oidc"].Maturity)
	require.Equal(t, []string{"read", "manage"}, byKey["identity.oidc"].AllowedActions)
}

func TestHandlerDoesNotGrantActionsWithoutTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/capabilities", NewHandler(nil))
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

// 没有实现证据的能力不得申报 GA：标签仍有双表未收敛，模板只有
// /api/v1/tickets/templates* 注册、通用 /api/v1/templates 市场从未注册。
func TestUnprovenCapabilitiesAreNotDeclaredGA(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("ITSM_FEISHU_READY", "false")

	router := gin.New()
	router.GET("/capabilities", func(c *gin.Context) {
		c.Set("tenant_id", 7)
		c.Set("role", "admin")
		NewHandler(nil)(c)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/capabilities", nil))
	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Data struct {
			Items []Capability `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))

	byKey := make(map[string]Capability)
	for _, item := range response.Data.Items {
		byKey[item.Key] = item
	}
	require.Equal(t, MaturityPilot, byKey["tag"].Maturity)
	require.Equal(t, MaturityPilot, byKey["template"].Maturity)
}

// TestPromotedCapabilitiesAreGA 锁定 2026-10-21 的通知域 GA 晋升：
// 投递审计读面 + 渠道诊断补齐后，「没收到通知」的三类根因都可定位。
// 问题域同批修了两处阻断，但因 /problems/:id/sla 仍显式 503 而维持 Pilot。
func TestPromotedCapabilitiesAreGA(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/capabilities", func(c *gin.Context) {
		c.Set("tenant_id", 7)
		c.Set("role", "admin")
		NewHandler(nil)(c)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/capabilities", nil))
	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Data struct {
			Items []Capability `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))

	byKey := make(map[string]Capability)
	for _, item := range response.Data.Items {
		byKey[item.Key] = item
	}
	require.Equal(t, MaturityGA, byKey["notification"].Maturity)
	require.Equal(t, []string{"read", "manage"}, byKey["notification"].AllowedActions)
	// 问题域的晋升被 SLA 未接入卡住：修了阻断不等于具备 GA 证据。
	require.Equal(t, MaturityPilot, byKey["problem"].Maturity)
}
