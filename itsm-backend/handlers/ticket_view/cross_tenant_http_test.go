// Package ticket_view 的跨租户 HTTP 回归测试。
//
// 工单视图（保存的过滤器/列配置）是查询面：跨租户读 GET 是内容泄漏面，
// PUT/DELETE 是跨租户篡改/删除面。隔离保证必须从真实生产入口
// （router.SetupRoutes 布线 + JWT + RBAC + handler + service + ent）发起验证。
// 参照 router/ticket_attachment_route_test.go 的搭台方式。
//
// 已钉住的契约（按当前生产行为如实断言）：
//   - Update/Get/Delete 的跨租户探测以 404/4004 返回（service 把
//     ent.NotFoundError 以 %w 包装进 `ticket view not found`，分类正确）；
//   - 跨租户读/更/删零泄漏零副作用：不回显视图内容、不改名、不删行；
//   - 跨租户与不存在资源的错误形态不可区分；
//   - owner 对自己视图的读/改/删是正控制。
package ticket_view_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	ticketViewHandler "itsm-backend/handlers/ticket_view"
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
	engine  *gin.Engine
	client  *ent.Client
	secret  string
	tenantA *ent.Tenant
	tenantB *ent.Tenant
	userA   *ent.User // tenant A super_admin：视图创建人
	stranger *ent.User // tenant B super_admin：能穿过 RBAC，但视图在租户 A
	viewA   *ent.TicketView
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:ticket_view_cross_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := client.Tenant.Create().SetName("View A").SetCode("view-a").SetDomain("view-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("View B").SetCode("view-b").SetDomain("view-b.example.com").SetStatus("active").SaveX(ctx)

	userA := client.User.Create().SetUsername("view-a-owner").SetEmail("view-a@example.com").SetName("Owner A").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	stranger := client.User.Create().SetUsername("view-b-stranger").SetEmail("view-b@example.com").SetName("Stranger B").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantB.ID).SaveX(ctx)

	viewA := client.TicketView.Create().
		SetName("Tenant A 私有视图").
		SetIsShared(false).
		SetCreatedBy(userA.ID).
		SetTenantID(tenantA.ID).
		SaveX(ctx)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	const secret = "ticket-view-cross-tenant-secret"
	router.SetupRoutes(engine, &router.RouterConfig{
		JWTSecret:         secret,
		Logger:            logger,
		Client:            client,
		TicketViewHandler: ticketViewHandler.NewHandler(service.NewTicketViewService(client, logger), logger),
	})

	return &fixture{
		engine: engine, client: client, secret: secret,
		tenantA: tenantA, tenantB: tenantB, userA: userA, stranger: stranger, viewA: viewA,
	}
}

func (fx *fixture) do(t *testing.T, method, path string, user *ent.User, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	token, err := middleware.GenerateAccessToken(user.ID, user.Username, string(user.Role), user.TenantID, fx.secret, time.Hour)
	require.NoError(t, err)
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	fx.engine.ServeHTTP(w, req)
	return w
}

func envelope(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out), "body=%s", w.Body.String())
	return out
}

func envelopeItems(t *testing.T, w *httptest.ResponseRecorder) []any {
	t.Helper()
	// 列表响应的 data 是 {items,total} 对象形状，不是裸数组。
	data, _ := envelope(t, w)["data"].(map[string]any)
	items, _ := data["items"].([]any)
	return items
}

func TestTicketViewCrossTenant_HTTP(t *testing.T) {
	fx := newFixture(t)
	ctx := context.Background()
	idPath := fmt.Sprintf("/api/v1/tickets/views/%d", fx.viewA.ID)

	t.Run("同租户创建人读视图（正控制）", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, idPath, fx.userA, nil)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"code":0`)
		require.Contains(t, w.Body.String(), "Tenant A 私有视图")
	})

	t.Run("跨租户读视图 404 且零内容泄漏", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, idPath, fx.stranger, nil)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"code":4004`)
		assert.NotContains(t, w.Body.String(), "Tenant A 私有视图")
		assert.NotContains(t, w.Body.String(), "filters")
	})

	t.Run("跨租户更新 404 且名称不被篡改", func(t *testing.T) {
		w := fx.do(t, http.MethodPut, idPath, fx.stranger, []byte(`{"name":"被劫持的视图"}`))
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"code":4004`)

		view, err := fx.client.TicketView.Get(ctx, fx.viewA.ID)
		require.NoError(t, err)
		assert.Equal(t, "Tenant A 私有视图", view.Name, "跨租户更新不得落库")
	})

	t.Run("跨租户删除 404 且视图原样留存", func(t *testing.T) {
		w := fx.do(t, http.MethodDelete, idPath, fx.stranger, nil)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"code":4004`)

		_, err := fx.client.TicketView.Get(ctx, fx.viewA.ID)
		require.NoError(t, err, "被探针命中的视图必须原样留存")
	})

	t.Run("跨租户列表零泄漏", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, "/api/v1/tickets/views", fx.stranger, nil)
		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Body.String(), `"code":0`)
		assert.Empty(t, envelopeItems(t, w), "租户外的保存视图不应出现在列表里")
	})

	t.Run("跨租户与不存在资源错误形态不可区分", func(t *testing.T) {
		wCross := fx.do(t, http.MethodGet, idPath, fx.stranger, nil)
		wMissing := fx.do(t, http.MethodGet, "/api/v1/tickets/views/999999", fx.stranger, nil)
		assert.Equal(t, wCross.Code, wMissing.Code)
		assert.Equal(t, wCross.Body.String(), wMissing.Body.String(),
			"观测同构才能防止靠响应差异枚举视图存在性")

		putCross := fx.do(t, http.MethodPut, idPath, fx.stranger, []byte(`{"name":"x"}`))
		putMissing := fx.do(t, http.MethodPut, "/api/v1/tickets/views/999999", fx.stranger, []byte(`{"name":"x"}`))
		assert.Equal(t, putCross.Code, putMissing.Code)
		assert.Equal(t, putCross.Body.String(), putMissing.Body.String())
	})

	t.Run("owner 读改删自己的视图是正控制", func(t *testing.T) {
		newName := "Tenant A 改名后视图"
		wPut := fx.do(t, http.MethodPut, idPath, fx.userA, []byte(fmt.Sprintf(`{"name":%q}`, newName)))
		require.Equal(t, http.StatusOK, wPut.Code, "body=%s", wPut.Body.String())
		require.Contains(t, wPut.Body.String(), `"code":0`)

		view, err := fx.client.TicketView.Get(ctx, fx.viewA.ID)
		require.NoError(t, err)
		assert.Equal(t, newName, view.Name, "正控制：owner 更新必须真实落库")

		wList := fx.do(t, http.MethodGet, "/api/v1/tickets/views", fx.userA, nil)
		require.Equal(t, http.StatusOK, wList.Code)
		assert.Len(t, envelopeItems(t, wList), 1, "正控制：owner 列表必须能看到自己的视图")

		wDel := fx.do(t, http.MethodDelete, idPath, fx.userA, nil)
		require.Equal(t, http.StatusOK, wDel.Code, "body=%s", wDel.Body.String())
		_, err = fx.client.TicketView.Get(ctx, fx.viewA.ID)
		require.Error(t, err, "正控制：owner 删除必须真实删行")
	})
}
