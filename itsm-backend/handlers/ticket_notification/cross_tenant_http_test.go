// Package ticket_notification 的跨租户 HTTP 回归测试。
//
// 工单通知（GET/POST /api/v1/tickets/:id/notifications）横跨两个高危面：
// 通知正文内容（跨租户读泄漏）与通知投递副作用（跨租户写入/引用注入）。
// 隔离保证必须从真实生产入口发起验证，参照 router/ticket_attachment_route_test.go。
//
// 已钉住的契约：
//   - 同租户 owner 列表/发送是正控制：正文可见、行与通用通知同步创建；
//   - 跨租户列表 200 但 items 恒空（内容零泄漏、零副作用）——当前生产语义；
//   - 跨租户发送 fail closed 且零副作用（当前 500/5001：service 返回裸
//     `ticket not found`，classifyError 分类不到 404。修复映射为 404/4004 后
//     需同步更新本文件该断言，与 comment 域 B3 同模式）；
//   - 【已知缺陷，如实钉住】同租户工单发送引用跨租户 userID：直发路径校验
//     用户存在时 `user.ID(userID)` 不带租户谓词，租户 A 工单可为租户 B 用户
//     创建通知行（跨租户引用注入）。补用户租户谓词后需同步断言为「不建行、
//     静默跳过或稳定拒绝」。
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
		// 已知缺陷（如实钉住）：service 直发路径返回裸 `ticket not found`，
		// classifyError 分类不到 404，跨租户探测表现为 500/5001。fail closed
		// 成立（零副作用），修复映射为 404/4004 后需同步更新本断言。
		require.Equal(t, http.StatusInternalServerError, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"code":5001`)
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

	t.Run("同租户工单引用跨租户userID是当前可复现的注入点", func(t *testing.T) {
		before := fx.countRows(t)
		w := fx.do(t, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/notifications", fx.ticketA.ID), fx.userA,
			[]byte(`{"userIds":[`+fmt.Sprint(fx.stranger.ID)+`],"type":"ticket_commented","channel":"in_app","content":"跨租户用户引用注入"}`))
		// 已知缺陷（如实钉住）：service 校验用户存在用裸 user.ID(userID)，
		// 不带租户谓词——租户 A 工单能为租户 B 用户创建通知行，形成跨租户
		// 引用注入。补用户租户谓词后本断言应改为「不建行」并同步更新。
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		after := fx.countRows(t)
		require.Equal(t, before.ticketNotifs+1, after.ticketNotifs, "当前实现确实为跨租户用户创建了通知行")

		row := fx.client.TicketNotification.Query().Order(ent.Desc("id")).FirstX(context.Background())
		require.Equal(t, fx.stranger.ID, row.UserID, "钉住注入点：通知行 user_id 引用了租户 B 用户")
		require.Equal(t, fx.tenantA.ID, row.TenantID)
	})
}
