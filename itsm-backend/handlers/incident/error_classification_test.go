package incident

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"

	baseent "entgo.io/ent"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// queryFault 对所有查询注入 DB 故障，用于区分「事件不存在」与「查询失败」。
// 故障在夹具建好之后才挂载，因此只影响被测请求。
func queryFault() baseent.Interceptor {
	return baseent.InterceptFunc(func(next baseent.Querier) baseent.Querier {
		return baseent.QuerierFunc(func(context.Context, baseent.Query) (baseent.Value, error) {
			return nil, errors.New("injected db query failure")
		})
	})
}

func TestIncidentSubresources_ClassifyDBFaultSeparatelyFromNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:incident_subresource_err_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })

	tenant := client.Tenant.Create().SetName("subresource").SetCode("subresource").SetDomain("subresource.test").SetStatus("active").SaveX(ctx)
	user := client.User.Create().SetUsername("subresource").SetName("Subresource").SetEmail("subresource@example.com").SetPasswordHash("hash").SetTenantID(tenant.ID).SetActive(true).SaveX(ctx)
	inc := client.Incident.Create().SetTitle("subresource incident").SetIncidentNumber("INC-SUBRESOURCE-1").SetStatus("in_progress").SetReporterID(user.ID).SetTenantID(tenant.ID).SaveX(ctx)

	logger := zap.NewNop().Sugar()
	h := NewHandler(NewService(NewEntRepository(client), client, nil, nil, nil, nil, nil, logger))
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenant.ID)
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenant.ID, Tenant: tenant})
		c.Set("user_id", user.ID)
		c.Set("role", "admin")
	})
	r.GET("/api/v1/incidents/:id/root-cause", h.GetRootCause)
	r.GET("/api/v1/incidents/:id/impact-assessment", h.GetImpactAssessment)
	r.GET("/api/v1/incidents/:id/classification", h.GetClassification)
	r.GET("/api/v1/incidents/:id/events", h.GetIncidentEvents)
	r.GET("/api/v1/incidents/:id/alerts", h.GetIncidentAlerts)
	r.GET("/api/v1/incidents/:id/metrics", h.GetIncidentMetrics)

	paths := []string{"root-cause", "impact-assessment", "classification", "events", "alerts", "metrics"}
	call := func(id int, path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/incidents/%d/%s", id, path), nil))
		return w
	}

	t.Run("真不存在仍是 404", func(t *testing.T) {
		for _, path := range paths {
			w := call(999999, path)
			require.Equal(t, http.StatusNotFound, w.Code, path+" body="+w.Body.String())
			var env struct {
				Code int `json:"code"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
			require.Equal(t, common.NotFoundErrorCode, env.Code, path)
		}
	})

	// 故障在正控制之后挂载，避免污染不存在用例。
	client.Intercept(queryFault())
	t.Run("DB 故障返回 500 且不能伪报事件不存在", func(t *testing.T) {
		for _, path := range paths {
			w := call(inc.ID, path)
			require.Equal(t, http.StatusInternalServerError, w.Code, path+" body="+w.Body.String())
			var env struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
			require.Equal(t, common.InternalErrorCode, env.Code, path)
			require.NotContains(t, env.Message, "not found", path)
			require.NotContains(t, w.Body.String(), "injected db query failure", path)
		}
	})
}
