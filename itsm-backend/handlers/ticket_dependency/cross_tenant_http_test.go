// Package ticket_dependency 的跨租户 HTTP 回归测试。
//
// 依赖影响分析（GET /api/v1/tickets/:id/dependencies）是跨租户信息泄漏的高敏感面：
// 分析结果会回显工单号、标题和受影响集合。隔离保证必须从真实生产入口
// （router.SetupRoutes 布线 + JWT 认证 + RBAC RequirePermission + handler + service + ent）
// 发起验证，参照 router/ticket_attachment_route_test.go 的搭台方式。
//
// 已钉住的契约：
//   - 同租户 owner 分析是正控制：200，回显本租户工单号/标题；
//   - 跨租户/不存在 → 404/4004，且错误形态同构（存在性不可区分），
//     响应内不得出现任何租户 A 的工单标题；
//   - GET 的 action 走查询参数绑定（路由不绑定请求体）。
package ticket_dependency_test

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
	dependencyHandler "itsm-backend/handlers/ticket_dependency"
	"itsm-backend/middleware"
	"itsm-backend/router"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type fixture struct {
	engine   *gin.Engine
	client   *ent.Client
	secret   string
	tenantA  *ent.Tenant
	tenantB  *ent.Tenant
	userA    *ent.User // tenant A super_admin：能穿过 RBAC 正常操作本租户
	stranger *ent.User // tenant B super_admin：能穿过 RBAC，但资源全在租户 A
	ticketA  *ent.Ticket
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:dep_cross_tenant_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := client.Tenant.Create().SetName("Dep A").SetCode("dep-a").SetDomain("dep-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("Dep B").SetCode("dep-b").SetDomain("dep-b.example.com").SetStatus("active").SaveX(ctx)

	userA := client.User.Create().SetUsername("dep-a-owner").SetEmail("dep-a@example.com").SetName("Owner A").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	stranger := client.User.Create().SetUsername("dep-b-stranger").SetEmail("dep-b@example.com").SetName("Stranger B").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantB.ID).SaveX(ctx)

	ticketA := client.Ticket.Create().
		SetTicketNumber("TKT-DEP-A-001").
		SetTitle("租户A依赖分析基线工单").
		SetType("incident").
		SetPriority("high").
		SetStatus("open").
		SetRequesterID(userA.ID).
		SetTenantID(tenantA.ID).
		SaveX(ctx)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	const secret = "dependency-cross-tenant-secret"
	router.SetupRoutes(engine, &router.RouterConfig{
		JWTSecret:               secret,
		Logger:                  logger,
		Client:                  client,
		TicketDependencyHandler: dependencyHandler.NewHandler(service.NewTicketDependencyService(client, logger), logger),
	})

	return &fixture{
		engine: engine, client: client, secret: secret,
		tenantA: tenantA, tenantB: tenantB, userA: userA, stranger: stranger,
		ticketA: ticketA,
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

func TestAnalyzeDependencyImpactCrossTenant(t *testing.T) {
	fx := newFixture(t)

	t.Run("同租户owner分析是正控制且回显本租户工单", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d/dependencies?action=close", fx.ticketA.ID), fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"code":0`)

		var resp struct {
			Code int `json:"code"`
			Data struct {
				TicketID     int    `json:"ticketId"`
				TicketNumber string `json:"ticketNumber"`
				TicketTitle  string `json:"ticketTitle"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Equal(t, 0, resp.Code)
		require.Equal(t, fx.ticketA.ID, resp.Data.TicketID)
		require.Equal(t, "TKT-DEP-A-001", resp.Data.TicketNumber)
		require.Equal(t, "租户A依赖分析基线工单", resp.Data.TicketTitle)
	})

	t.Run("跨租户分析404且零内容泄漏", func(t *testing.T) {
		_ = fx.client.Ticket.Create().
			SetTicketNumber("TKT-DEP-B-001").
			SetTitle("租户B依赖工厂工单").
			SetType("incident").
			SetPriority("low").
			SetStatus("open").
			SetRequesterID(fx.stranger.ID).
			SetTenantID(fx.tenantB.ID).
			SaveX(context.Background())

		w := fx.do(t, http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d/dependencies?action=close", fx.ticketA.ID), fx.stranger)
		// not-found 稳定映射 404/4004；错误响应不回显任何租户 A 内容。
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"code":4004`)
		assert.NotContains(t, w.Body.String(), "TKT-DEP-A-001")
		assert.NotContains(t, w.Body.String(), "租户A依赖分析基线工单")
	})

	t.Run("跨租户与不存在工单的分析错误形态同构", func(t *testing.T) {
		w1 := fx.do(t, http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d/dependencies?action=delete", fx.ticketA.ID), fx.stranger)
		w2 := fx.do(t, http.MethodGet, "/api/v1/tickets/999999/dependencies?action=delete", fx.stranger)
		require.Equal(t, http.StatusNotFound, w1.Code)
		require.Equal(t, http.StatusNotFound, w2.Code)
		require.Contains(t, w1.Body.String(), `"code":4004`)
		require.Equal(t, w1.Body.String(), w2.Body.String(),
			"跨租户与不存在必须是同一种错误形态（存在性不可区分）")
	})
}
