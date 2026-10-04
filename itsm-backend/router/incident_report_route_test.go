package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	incidentHandler "itsm-backend/handlers/incident"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 回归（2026-10-04 边缘功能收口 E4-7）：GET /api/v1/incidents/stats/report。
//
// /reports/incident-trends 修复前读的是工单分析接口，事件域没有任何窗口报表端点，
// 页面只能在浏览器里对分页结果二次累加，并把后端分钟当小时渲染。本测试打在
// router/incident_routes.go 的真实注册上（含 RequirePermission("incident","read")），
// 证明生产入口可达、租户隔离与参数错误语义在中间件链之后依然成立，
// 并锁定 /stats/report 没有被 /incidents/:id 抢走。

type incidentReportEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func TestIncidentReportRouteIsRegisteredAndTenantScoped(t *testing.T) {
	dsn := fmt.Sprintf("file:router_incident_report_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := client.Tenant.Create().SetName("Inc A").SetCode("inc-a").SetDomain("inc-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("Inc B").SetCode("inc-b").SetDomain("inc-b.example.com").SetStatus("active").SaveX(ctx)
	userA := client.User.Create().SetUsername("inc-a-admin").SetEmail("inc-a@example.com").SetName("A").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	userB := client.User.Create().SetUsername("inc-b-admin").SetEmail("inc-b@example.com").SetName("B").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantB.ID).SaveX(ctx)

	// 窗口用绝对日期，避免用例跨午夜后读数漂移。
	from := time.Now().AddDate(0, 0, -3)
	to := time.Now()
	newIncident := func(tenantID, reporterID int, number, status string, createdAt time.Time) {
		client.Incident.Create().
			SetTitle("报表 " + number).
			SetDescription("验证事件趋势报表端点").
			SetStatus(status).
			SetPriority("medium").
			SetSeverity("medium").
			SetIncidentNumber(number).
			SetReporterID(reporterID).
			SetTenantID(tenantID).
			SetCreatedAt(createdAt).
			SaveX(ctx)
	}
	newIncident(tenantA.ID, userA.ID, "INC-RPT-A-1", "new", incidentReportNoon(to, -2))
	newIncident(tenantA.ID, userA.ID, "INC-RPT-A-2", "resolved", incidentReportNoon(to, -1))
	// 窗口外创建：不得进入 A 的 cohort。
	newIncident(tenantA.ID, userA.ID, "INC-RPT-A-3", "new", incidentReportNoon(to, -20))
	newIncident(tenantB.ID, userB.ID, "INC-RPT-B-1", "new", incidentReportNoon(to, -1))

	gin.SetMode(gin.TestMode)
	r := gin.New()
	const secret = "incident-report-secret"
	SetupRoutes(r, &RouterConfig{
		JWTSecret:       secret,
		Logger:          logger,
		Client:          client,
		IncidentHandler: incidentHandler.NewHandler(incidentHandler.NewService(incidentHandler.NewEntRepository(client), client, nil, nil, nil, nil, nil, logger)),
	})

	do := func(t *testing.T, path string, user *ent.User) (int, incidentReportEnvelope) {
		t.Helper()
		token, err := middleware.GenerateAccessToken(user.ID, user.Username, string(user.Role), user.TenantID, secret, time.Hour)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		var env incidentReportEnvelope
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
		return w.Code, env
	}

	query := "dateFrom=" + from.Format("2006-01-02") + "&dateTo=" + to.Format("2006-01-02")
	path := "/api/v1/incidents/stats/report?" + query

	t.Run("生产路由返回报表读模型", func(t *testing.T) {
		w, env := do(t, path, userA)
		assert.Equal(t, http.StatusOK, w, "body=%s", string(env.Data))
		assert.Zero(t, env.Code, env.Message)

		var data map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(env.Data, &data))
		keys := make([]string, 0, len(data))
		for key := range data {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		assert.Equal(t,
			[]string{"avgResolutionMinutes", "byPriority", "byStatus", "createdInWindow", "dailyTrend", "resolvedInWindow", "window"},
			keys, "生产入口的契约字段必须与 handler 层一致")

		var report struct {
			Window struct {
				DateFrom string `json:"dateFrom"`
				DateTo   string `json:"dateTo"`
				Days     int    `json:"days"`
			} `json:"window"`
			CreatedInWindow int `json:"createdInWindow"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &report))
		assert.Equal(t, from.Format("2006-01-02"), report.Window.DateFrom, "窗口必须按请求回显")
		assert.Equal(t, to.Format("2006-01-02"), report.Window.DateTo)
		assert.Equal(t, 2, report.CreatedInWindow, "窗口外的 INC-RPT-A-3 不得计入")
	})

	t.Run("租户 B 只看见自己的事件", func(t *testing.T) {
		w, env := do(t, path, userB)
		require.Equal(t, http.StatusOK, w, "body=%s", string(env.Data))
		require.Zero(t, env.Code, env.Message)
		assert.NotContains(t, string(env.Data), "INC-RPT-A", "响应不得泄漏租户 A 的标记")

		var report struct {
			CreatedInWindow int `json:"createdInWindow"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &report))
		assert.Equal(t, 1, report.CreatedInWindow)
	})

	t.Run("未认证 401，非法窗口 400", func(t *testing.T) {
		unauth := httptest.NewRecorder()
		r.ServeHTTP(unauth, httptest.NewRequest(http.MethodGet, path, nil))
		assert.Equal(t, http.StatusUnauthorized, unauth.Code, "body=%s", unauth.Body.String())

		// 只传一端是参数错误，不得静默补另一端后返回缺省窗口。
		partial := "/api/v1/incidents/stats/report?dateFrom=" + from.Format("2006-01-02")
		w, env := do(t, partial, userA)
		assert.Equal(t, http.StatusBadRequest, w, "body=%s", string(env.Data))
		assert.Equal(t, 1001, env.Code, env.Message)
	})

	t.Run("静态段不被 /incidents/:id 吃掉", func(t *testing.T) {
		// /incidents/stats 是遗留 PostgreSQL 标量聚合：测试装配没有原始 DB，
		// stats 仓储为 nil，所以必然 500/5001。这里只锁路由归属——请求必须落到
		// stats handler，而不是被 :id 当成事件编号（400/1001）或查不到（404/4004）。
		w, env := do(t, "/api/v1/incidents/stats", userA)
		assert.NotEqual(t, http.StatusBadRequest, w, "stats 被当成事件 ID 解析: body=%s", string(env.Data))
		assert.NotEqual(t, http.StatusNotFound, w, "stats 被 /incidents/:id 抢走: body=%s", string(env.Data))
		assert.Equal(t, http.StatusInternalServerError, w, "body=%s", string(env.Data))
		assert.Equal(t, 5001, env.Code, env.Message)

		w, env = do(t, "/api/v1/incidents/987654", userA)
		assert.Equal(t, http.StatusNotFound, w, "body=%s", string(env.Data))
		assert.Equal(t, 4004, env.Code, env.Message)
	})
}

// incidentReportNoon 返回基准日当天偏移 dayOffset 天的正午时刻。
func incidentReportNoon(base time.Time, dayOffset int) time.Time {
	day := base.AddDate(0, 0, dayOffset)
	return time.Date(day.Year(), day.Month(), day.Day(), 12, 0, 0, 0, time.Local)
}
