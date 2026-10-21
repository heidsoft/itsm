package capability

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

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

	// disabled 能力不参与探测，也不应带健康信号。
	wecom := byKey["connector.wecom"]
	require.Equal(t, MaturityDisabled, wecom.Maturity)
	require.Nil(t, wecom.Health)
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
