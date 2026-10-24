package problem

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"
	"itsm-backend/service"
)

// 回归：GetTrends/GetHotspots 曾直接序列化没有 json tag 的域实体，
// 整包 PascalCase（Period/TotalProblems/Hotspots/...），agent 与前端只能猜
// 字段名。tag 已与 dto/problem_trend_dto.go 对齐，这里锁死 wire 键名不回潮。

func newTrendTestHandler(t *testing.T) *Handler {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:problem_trend_shape?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	return NewHandler(NewService(NewEntRepository(client), client, service.NewTicketServiceForTest(client, zaptest.NewLogger(t).Sugar()), zaptest.NewLogger(t).Sugar()))
}

func serveProblemGet(t *testing.T, h *Handler, tenantID int, route string) *httptest.ResponseRecorder {
	t.Helper()
	r := gin.New()
	r.GET(route, func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: tenantID})
		if route == "/api/v1/problems/trends" {
			h.GetTrends(c)
			return
		}
		h.GetHotspots(c)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, route, nil))
	return w
}

func decodeProblemData(t *testing.T, w *httptest.ResponseRecorder) map[string]json.RawMessage {
	t.Helper()
	var envelope struct {
		Code int                        `json:"code"`
		Data map[string]json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope), w.Body.String())
	require.Zero(t, envelope.Code)
	return envelope.Data
}

func TestProblemTrend_WireKeysAreCamelCase(t *testing.T) {
	h := newTrendTestHandler(t)
	w := serveProblemGet(t, h, 1, "/api/v1/problems/trends")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	data := decodeProblemData(t, w)
	for _, key := range []string{
		"period", "totalProblems", "resolvedProblems", "openProblems", "resolutionRate",
		"avgResolutionTimeHours", "categoryBreakdown", "priorityBreakdown", "trendDirection",
		"topCategories", "monthlyTrend",
	} {
		require.Contains(t, data, key, "data 缺少 camelCase 键 %q，整包: %s", key, w.Body.String())
	}
	for _, banned := range []string{"Period", "TotalProblems", "AvgResolutionTimeHours", "TrendDirection", "MonthlyTrend"} {
		require.NotContains(t, data, banned, "PascalCase 键 %q 不得再出现", banned)
	}
}

func TestProblemHotspots_WireKeysAreCamelCase(t *testing.T) {
	h := newTrendTestHandler(t)
	w := serveProblemGet(t, h, 1, "/api/v1/problems/hotspots")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	data := decodeProblemData(t, w)
	for _, key := range []string{"periodStart", "periodEnd", "categoryBreakdown", "priorityBreakdown", "hotspots", "avgPerCategory"} {
		require.Contains(t, data, key, "data 缺少 camelCase 键 %q，整包: %s", key, w.Body.String())
	}
	for _, banned := range []string{"PeriodStart", "PeriodEnd", "Hotspots", "AvgPerCategory"} {
		require.NotContains(t, data, banned, "PascalCase 键 %q 不得再出现", banned)
	}
}
