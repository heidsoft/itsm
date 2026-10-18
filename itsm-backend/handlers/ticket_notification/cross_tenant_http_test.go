// Package ticket_notification 的跨租户 HTTP 回归测试。
//
// 工单通知（GET/POST /api/v1/tickets/:id/notifications）横跨两个高危面：
// 通知正文内容（跨租户读泄漏）与通知投递副作用（跨租户写入/引用注入）。
// 隔离保证必须从真实生产入口发起验证，参照 router/ticket_attachment_route_test.go。
//
// 已钉住的契约：
//   - 同租户 owner 列表/发送是正控制：正文可见、行与通用通知同步创建；
//   - 跨租户列表 200 但 items 恒空（内容零泄漏、零副作用）——当前生产语义；
//   - 跨租户发送 404/4004 且零副作用（B5 修复：service 改用 BusinessError
//     呈现 not-found，与 comment 域 B3 同模式）；
//   - 同租户工单不能为跨租户 userID 创建通知行（B5 修复：直发路径用户
//     存在性校验带租户谓词，跨租户引用以「用户不存在」静默跳过、零行，
//     与幽灵 userID 的可观测形态不可区分）。
package ticket_notification_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	notificationHandler "itsm-backend/handlers/ticket_notification"
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
	userA    *ent.User // tenant A super_admin
	stranger *ent.User // tenant B super_admin
	ticketA  *ent.Ticket
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:notif_cross_tenant_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := client.Tenant.Create().SetName("Notif A").SetCode("notif-a").SetDomain("notif-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("Notif B").SetCode("notif-b").SetDomain("notif-b.example.com").SetStatus("active").SaveX(ctx)

	userA := client.User.Create().SetUsername("notif-a-owner").SetEmail("notif-a@example.com").SetName("Owner A").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	stranger := client.User.Create().SetUsername("notif-b-stranger").SetEmail("notif-b@example.com").SetName("Stranger B").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantB.ID).SaveX(ctx)

	ticketA := client.Ticket.Create().
		SetTicketNumber("TKT-NOTIF-A-001").
		SetTitle("租户A通知基线工单").
		SetType("incident").
		SetPriority("high").
		SetStatus("open").
		SetRequesterID(userA.ID).
		SetTenantID(tenantA.ID).
		SaveX(ctx)

	client.TicketNotification.Create().
		SetTicketID(ticketA.ID).
		SetUserID(userA.ID).
		SetType("ticket_created").
		SetChannel("in_app").
		SetContent("机密通知内容-A").
		SetTenantID(tenantA.ID).
		SetStatus("sent").
		SaveX(ctx)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	const secret = "notification-cross-tenant-secret"
	router.SetupRoutes(engine, &router.RouterConfig{
		JWTSecret:                 secret,
		Logger:                    logger,
		Client:                    client,
		TicketNotificationHandler: notificationHandler.NewHandler(service.NewTicketNotificationService(client, logger), logger),
	})

	return &fixture{
		engine: engine, client: client, secret: secret,
		tenantA: tenantA, tenantB: tenantB, userA: userA, stranger: stranger,
		ticketA: ticketA,
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
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	fx.engine.ServeHTTP(w, req)
	return w
}

type rowCounters struct {
	ticketNotifs int
	genericNotif int
}

func (fx *fixture) countRows(t *testing.T) rowCounters {
	t.Helper()
	ctx := context.Background()
	return rowCounters{
		ticketNotifs: fx.client.TicketNotification.Query().CountX(ctx),
		genericNotif: fx.client.Notification.Query().CountX(ctx),
	}
}

func TestTicketNotificationCrossTenant(t *testing.T) {
	fx := newFixture(t)

	t.Run("同租户owner列表是正控制且正文可见", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d/notifications", fx.ticketA.ID), fx.userA, nil)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"code":0`)
		require.Contains(t, w.Body.String(), `"total":1`)
		require.Contains(t, w.Body.String(), "机密通知内容-A")
	})

	t.Run("跨租户列表fail closed且零内容零副作用", func(t *testing.T) {
		before := fx.countRows(t)
		w := fx.do(t, http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d/notifications", fx.ticketA.ID), fx.stranger, nil)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		// 当前生产语义：tickets 列表走 ticketID+tenantID 复合过滤，跨租户返回
		// 空列表而非 404。内容零泄漏是必须成立的硬约束。
		require.Contains(t, w.Body.String(), `"total":0`)
		require.Contains(t, w.Body.String(), `"items":[]`)
		assert.NotContains(t, w.Body.String(), "机密通知内容-A")
		assert.Equal(t, before, fx.countRows(t), "跨租户列表不得产生任何副作用")
	})

	t.Run("跨租户发送fail closed且零副作用", func(t *testing.T) {
		before := fx.countRows(t)
		w := fx.do(t, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/notifications", fx.ticketA.ID), fx.stranger,
			[]byte(`{"userIds":[`+fmt.Sprint(fx.stranger.ID)+`],"type":"ticket_commented","channel":"in_app","content":"跨租户投放探测"}`))
		// not-found 稳定映射 404/4004（service 用 BusinessError 呈现，B5 修复后不再是
		// 500/5001 毛刺）；跨租户与不存在的工单同观测、零副作用。
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"code":4004`)
		assert.Equal(t, before, fx.countRows(t), "跨租户发送不得创建任何通知行")
	})

	t.Run("同租户in_app发送是正控制：域行与通用通知同步创建", func(t *testing.T) {
		before := fx.countRows(t)
		w := fx.do(t, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/notifications", fx.ticketA.ID), fx.userA,
			[]byte(`{"userIds":[`+fmt.Sprint(fx.userA.ID)+`],"type":"ticket_commented","channel":"in_app","content":"同租户投递正控制"}`))
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"code":0`)

		after := fx.countRows(t)
		require.Equal(t, before.ticketNotifs+1, after.ticketNotifs, "必须新建一条工单通知行")
		require.Equal(t, before.genericNotif+1, after.genericNotif, "必须同步新建一条通用通知行")

		created := fx.client.TicketNotification.Query().
			Order(ent.Desc("id")).
			FirstX(context.Background())
		require.Equal(t, "同租户投递正控制", created.Content)
		require.Equal(t, "sent", created.Status)
		require.Equal(t, fx.tenantA.ID, created.TenantID)
	})

	t.Run("同租户工单不能为跨租户userID创建通知行", func(t *testing.T) {
		before := fx.countRows(t)
		w := fx.do(t, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/notifications", fx.ticketA.ID), fx.userA,
			[]byte(`{"userIds":[`+fmt.Sprint(fx.stranger.ID)+`],"type":"ticket_commented","channel":"in_app","content":"跨租户用户引用注入"}`))
		// 修复后语义（B5）：用户存在性校验带租户谓词，租户 B 用户对租户 A
		// 请求等同「用户不存在」→ 静默跳过、200 成功但零行、不产生跨租户引用。
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"code":0`)
		after := fx.countRows(t)
		assert.Equal(t, before, after, "跨租户用户引用不得创建任何通知行")
	})

	t.Run("不存在的userID被静默跳过且与跨租户userID不可区分", func(t *testing.T) {
		before := fx.countRows(t)
		w := fx.do(t, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/notifications", fx.ticketA.ID), fx.userA,
			[]byte(`{"userIds":[999999],"type":"ticket_commented","channel":"in_app","content":"幽灵用户探测"}`))
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"code":0`)
		assert.Equal(t, before, fx.countRows(t), "不存在用户必须静默跳过、零行")
		// 与跨租户 userID 的错误/成功形态保持一致性：两者对外观感一致，
		// 差异只体现在库内是否落行。
	})
}
