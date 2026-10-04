package incident

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"itsm-backend/ent/incident"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// GET /api/v1/incidents/alerts/active 的分页此前有两套自建规则：handler 里
// page<1→1 / size<1→10 / size>100→100，service 里再抄一份同样的字面量，
// 缺省页长 10 与平台契约 20 不一致。收敛后 handler 走
// common.GetPaginationFromQuery 采纳参数，service 只再用 common.ValidatePagination
// 兜非 HTTP 调用方，两层都读同一组 common 常量，不再有 10/100 字面量。

func newAlertsRouter(h *IncidentHandler, tenantID, actorID int) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
		c.Set("tenant_id", tenantID)
		c.Set("user_id", actorID)
		c.Set("role", "admin")
	})
	r.GET("/api/v1/incidents/alerts/active", h.GetActiveAlerts)
	return r
}

func doAlerts(t *testing.T, h *IncidentHandler, tenantID, actorID int, query string) map[string]interface{} {
	t.Helper()

	w := httptest.NewRecorder()
	newAlertsRouter(h, tenantID, actorID).ServeHTTP(w,
		httptest.NewRequest(http.MethodGet, "/api/v1/incidents/alerts/active?"+query, nil))

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), w.Body.String())
	require.Equal(t, float64(0), body["code"], body["message"])

	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok, "data 必须是列表信封，实际 %v", body)
	return data
}

func TestActiveAlerts_PaginationUsesPlatformOwner(t *testing.T) {
	f := newListContractFixture(t)
	f.baseSeed(t)

	// 夹具的 handler 未注入告警 service（列表契约用例不需要），这里补真实装配。
	f.handler.service.alertingSvc = service.NewIncidentAlertingService(f.client, zap.NewNop().Sugar())

	ctx := context.Background()

	inc, err := f.client.Incident.Query().Where(incident.TenantIDEQ(f.tenantA)).First(ctx)
	require.NoError(t, err, "需要一个真实事件作为告警的外键")

	for i := 0; i < 25; i++ {
		f.client.IncidentAlert.Create().
			SetIncidentID(inc.ID).
			SetAlertType("sla_breach").
			SetAlertName("分页契约告警").
			SetMessage("m").
			SetStatus("active").
			SetTenantID(f.tenantA).
			SaveX(ctx)
	}

	cases := []struct {
		name           string
		query          string
		wantPage       float64
		wantPageSize   float64
		wantTotalPages float64
		wantItems      int
	}{
		{"缺省页长取平台值 20", "", 1, 20, 2, 20},
		{"越界页长回落缺省而不是整表读取", "page=1&pageSize=5000", 1, 20, 2, 20},
		{"非数字页长回落缺省", "pageSize=abc", 1, 20, 2, 20},
		{"非正页长回落缺省", "pageSize=0", 1, 20, 2, 20},
		{"非正页码下界为 1", "page=0&pageSize=10", 1, 10, 3, 10},
		{"上界 100 原样采纳", "page=1&pageSize=100", 1, 100, 1, 25},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := doAlerts(t, f.handler, f.tenantA, f.agentA, tc.query)

			items, ok := data["items"].([]interface{})
			require.True(t, ok, "data.items 缺失: %v", data)
			assert.Equal(t, tc.wantPage, data["page"], "query=%s", tc.query)
			assert.Equal(t, tc.wantPageSize, data["pageSize"], "query=%s", tc.query)
			assert.Equal(t, tc.wantTotalPages, data["totalPages"], "query=%s", tc.query)
			assert.Len(t, items, tc.wantItems, "query=%s", tc.query)
		})
	}
}
