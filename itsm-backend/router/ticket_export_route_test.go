package router

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/ent/enttest"
	ticketHandler "itsm-backend/handlers/ticket"
	"itsm-backend/middleware"
	ticketrepo "itsm-backend/repository/ticket"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 回归（2026-10-02 边缘功能收口 E1-1）：POST /api/v1/tickets/export 此前恒返回
// HTTP 500 / code 5001，因为 handlers/ticket 只留了占位实现，而带租户谓词的
// 可用实现一直存在于 service 层。必须从真实路由入口证明：
//   - 租户内导出返回文件流，且只含本租户工单；
//   - csv/excel 的 Content-Disposition 与 Content-Type 和真实字节格式一致；
//   - 跨租户不得泄漏对方工单编号；
//   - 未实现的 pdf 是参数错误，不是 500。

func setupExportRouteTest(t *testing.T) (*gin.Engine, exportRouteFixture) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:router_ticket_export_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := client.Tenant.Create().SetName("Export A").SetCode("export-a").SetDomain("export-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("Export B").SetCode("export-b").SetDomain("export-b.example.com").SetStatus("active").SaveX(ctx)

	userA := client.User.Create().SetUsername("export-a-admin").SetEmail("export-a@example.com").SetName("A").SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	userB := client.User.Create().SetUsername("export-b-admin").SetEmail("export-b@example.com").SetName("B").SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantB.ID).SaveX(ctx)

	newTicket := func(tenantID int, number, title string) {
		client.Ticket.Create().
			SetTicketNumber(number).
			SetTitle(title).
			SetDescription("=cmd|' /C calc'!A0").
			SetType("incident").
			SetPriority("high").
			SetStatus("open").
			SetRequesterID(userA.ID).
			SetTenantID(tenantID).
			SaveX(ctx)
	}
	newTicket(tenantA.ID, "TKT-EXPORT-A-001", "A 租户工单")
	newTicket(tenantB.ID, "TKT-EXPORT-B-001", "B 租户工单")

	productionSvc := service.NewTicketServiceForTest(client, logger)
	svc := ticketHandler.NewService(ticketHandler.NewEntRepository(ticketrepo.NewEntRepository(client, logger)), productionSvc, nil, logger)

	const secret = "ticket-export-secret"
	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetupRoutes(r, &RouterConfig{
		JWTSecret:     secret,
		Logger:        logger,
		Client:        client,
		TicketHandler: ticketHandler.NewHandler(svc),
	})
	return r, exportRouteFixture{userA: userA.ID, userB: userB.ID, tenantA: tenantA.ID, tenantB: tenantB.ID, secret: secret}
}

type exportRouteFixture struct {
	userA   int
	userB   int
	tenantA int
	tenantB int
	secret  string
}

func doExport(t *testing.T, r *gin.Engine, userID, tenantID int, secret, body string) *httptest.ResponseRecorder {
	t.Helper()
	token, err := middleware.GenerateAccessToken(userID, "export-admin", "super_admin", tenantID, secret, time.Hour)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tickets/export", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestTicketExportRoute_ReturnsTenantScopedFile(t *testing.T) {
	r, fx := setupExportRouteTest(t)

	t.Run("csv 返回真实 CSV 且只含本租户工单", func(t *testing.T) {
		w := doExport(t, r, fx.userA, fx.tenantA, fx.secret, `{"format":"csv"}`)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		assert.Equal(t, `attachment; filename="tickets.csv"`, w.Header().Get("Content-Disposition"))
		assert.Contains(t, w.Header().Get("Content-Type"), "text/csv")

		body := w.Body.String()
		assert.True(t, bytes.HasPrefix(w.Body.Bytes(), []byte("\ufeff")), "CSV 必须带 UTF-8 BOM，否则 Excel 打开中文乱码")
		assert.Contains(t, body, "工单编号,标题,描述,状态,优先级,创建时间,更新时间", "表头顺序必须稳定")
		assert.Contains(t, body, "TKT-EXPORT-A-001")
		assert.NotContains(t, body, "TKT-EXPORT-B-001", "跨租户工单不得出现在导出文件里")
		// 描述里的公式必须在导出前中和，防止电子表格公式注入。
		assert.Contains(t, body, "'=cmd")
	})

	t.Run("excel 返回真实 xlsx 字节", func(t *testing.T) {
		w := doExport(t, r, fx.userA, fx.tenantA, fx.secret, `{"format":"excel"}`)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		assert.Equal(t, `attachment; filename="tickets.xlsx"`, w.Header().Get("Content-Disposition"))
		assert.Contains(t, w.Header().Get("Content-Type"), "spreadsheetml")
		assert.True(t, bytes.HasPrefix(w.Body.Bytes(), []byte("PK")), "xlsx 是 zip 容器，必须以 PK 开头")
		assert.Greater(t, w.Body.Len(), 100)
	})

	t.Run("同一请求两次导出字节完全一致", func(t *testing.T) {
		first := doExport(t, r, fx.userA, fx.tenantA, fx.secret, `{"format":"csv"}`).Body.Bytes()
		second := doExport(t, r, fx.userA, fx.tenantA, fx.secret, `{"format":"csv"}`).Body.Bytes()
		assert.True(t, bytes.Equal(first, second), "导出的列序/行序必须确定，否则自动化比对无法信任导出文件")
	})

	t.Run("租户 B 导出看不到租户 A 的工单", func(t *testing.T) {
		w := doExport(t, r, fx.userB, fx.tenantB, fx.secret, `{"format":"csv"}`)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		body := w.Body.String()
		assert.Contains(t, body, "TKT-EXPORT-B-001")
		assert.NotContains(t, body, "TKT-EXPORT-A-001")
	})

	t.Run("status 过滤在导出内生效", func(t *testing.T) {
		w := doExport(t, r, fx.userA, fx.tenantA, fx.secret, `{"format":"csv","filters":{"status":"resolved"}}`)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		body := w.Body.String()
		assert.NotContains(t, body, "TKT-EXPORT-A-001", "open 工单不应出现在 resolved 导出里")
		// 只剩表头行，说明过滤条件真实下到了查询。
		assert.Equal(t, 1, bytes.Count(w.Body.Bytes(), []byte("\n")))
	})

	t.Run("未实现的 pdf 是参数错误而不是 500", func(t *testing.T) {
		w := doExport(t, r, fx.userA, fx.tenantA, fx.secret, `{"format":"pdf"}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), `"code":1001`)
	})
}
