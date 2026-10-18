// Package analytics 的跨租户 HTTP 回归测试。
//
// 工单分析概览（GET /api/v1/analytics/tickets）是聚合统计面：泄漏不体现在
// 单条内容，而是体现在「计数/分布把别家租户的工单算进来」。因此本文件的
// 探针以精确计数为核心：跨租户时 total 与各分布必须只含本租户。
// 从真实生产入口发起验证，参照 router/ticket_attachment_route_test.go。
//
// 已钉住的契约：
//   - 响应 data={total,statusGroups,priorityGroups,trend30d,generatedAt}；
//   - 同租户 owner 的 total 等于自己租户的工单数；
//   - 跨租户读取的 total / 分布必须与对端无关（不把租户 A 工单计入）。
package analytics_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	analyticsHandler "itsm-backend/handlers/analytics"
	"itsm-backend/middleware"
	"itsm-backend/router"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type fixture struct {
	engine   *gin.Engine
	client   *ent.Client
	secret   string
	tenantA  *ent.Tenant
	tenantB  *ent.Tenant
	userA    *ent.User
	stranger *ent.User
	ticketsA int
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:analytics_cross_tenant_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := client.Tenant.Create().SetName("Analytics A").SetCode("ana-a").SetDomain("ana-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("Analytics B").SetCode("ana-b").SetDomain("ana-b.example.com").SetStatus("active").SaveX(ctx)

	userA := client.User.Create().SetUsername("ana-a-owner").SetEmail("ana-a@example.com").SetName("Owner A").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	stranger := client.User.Create().SetUsername("ana-b-stranger").SetEmail("ana-b@example.com").SetName("Stranger B").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantB.ID).SaveX(ctx)

	mkTicket := func(number string, tenant *ent.Tenant, requesterID int, status string) {
		client.Ticket.Create().
			SetTicketNumber(number).
			SetTitle("统计指标工单 " + number).
			SetType("incident").
			SetPriority("high").
			SetStatus(status).
			SetRequesterID(requesterID).
			SetTenantID(tenant.ID).
			SaveX(ctx)
	}
	// 租户 A：3 条工单（open/close 分布），成为 owner 正控制的精确预期。
	mkTicket("TKT-ANA-A-001", tenantA, userA.ID, "open")
	mkTicket("TKT-ANA-A-002", tenantA, userA.ID, "open")
	mkTicket("TKT-ANA-A-003", tenantA, userA.ID, "closed")
	// 租户 B：2 条工单。跨租户读取时若被计入，分布立刻露馅。
	mkTicket("TKT-ANA-B-001", tenantB, stranger.ID, "open")
	mkTicket("TKT-ANA-B-002", tenantB, stranger.ID, "closed")

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	const secret = "analytics-cross-tenant-secret"
	router.SetupRoutes(engine, &router.RouterConfig{
		JWTSecret:        secret,
		Logger:           logger,
		Client:           client,
		AnalyticsHandler: analyticsHandler.NewHandler(service.NewAnalyticsService(client, logger)),
	})

	return &fixture{
		engine: engine, client: client, secret: secret,
		tenantA: tenantA, tenantB: tenantB, userA: userA, stranger: stranger,
		ticketsA: 3,
	}
}

func (fx *fixture) do(t *testing.T, method, path string, user *ent.User) *httptest.ResponseRecorder {
	t.Helper()
	token, err := middleware.GenerateAccessToken(user.ID, user.Username, string(user.Role), user.TenantID, fx.secret, time.Hour)
	require.NoError(t, err)
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	fx.engine.ServeHTTP(w, req)
	return w
}

func TestGetTicketAnalyticsCrossTenant(t *testing.T) {
	fx := newFixture(t)

	t.Run("同租户owner概览是正控制且total精确等于本租户工单数", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, "/api/v1/analytics/tickets", fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"code":0`)

		var resp struct {
			Code int `json:"code"`
			Data struct {
				Total        int `json:"total"`
				StatusGroups []struct {
					Status string `json:"status"`
					Count  int    `json:"count"`
				} `json:"statusGroups"`
				PriorityGroups []struct {
					Priority string `json:"priority"`
					Count    int    `json:"count"`
				} `json:"priorityGroups"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Equal(t, 0, resp.Code)
		require.Equal(t, fx.ticketsA, resp.Data.Total, "total 必须等于本租户工单数")

		sum := 0
		for _, g := range resp.Data.StatusGroups {
			sum += g.Count
		}
		require.Equal(t, fx.ticketsA, sum, "status 分布计数之和必须等于本租户工单数，不得混入他租户")
	})

	t.Run("跨租户概览的聚合不得计入对端租户工单", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, "/api/v1/analytics/tickets", fx.stranger)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		var resp struct {
			Code int `json:"code"`
			Data struct {
				Total        int `json:"total"`
				StatusGroups []struct {
					Status string `json:"status"`
					Count  int    `json:"count"`
				} `json:"statusGroups"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Equal(t, 0, resp.Code)
		require.Equal(t, 2, resp.Data.Total, "租户 B 读取时 total 只能是自己的 2 条工单")
		sum := 0
		for _, g := range resp.Data.StatusGroups {
			sum += g.Count
		}
		require.Equal(t, 2, sum, "status 分布不得把租户 A 工单算进来")
	})
}
