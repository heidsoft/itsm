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
	"itsm-backend/ent/notification"
	"itsm-backend/ent/user"
	domainCommon "itsm-backend/handlers/common"
	notificationHandler "itsm-backend/handlers/notification"
	ticketNotificationHandler "itsm-backend/handlers/ticket_notification"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 回归（2026-10-03 边缘功能收口 E4-7）：通知域的两个列表信封都不在标准契约上。
//
//   - GET /api/v1/notifications 返回 {notifications,total,page,size}：集合键用了领域名
//     （AGENTS.md 明令 items 单一契约），页长键用了被禁止的别名 size，且没有 totalPages，
//     前端因此只能写「后端分页字段是 size（不是 pageSize）」这种注释并把字段名硬编到调用点。
//   - GET /api/v1/tickets/:id/notifications 返回 {notifications,total}：service 侧是
//     无 Offset/Limit 的 All(ctx)，属文档承认的「不分页的列表」合法形状，但集合键
//     同样必须收敛为 items。
//
// 另外实测到 page/pageSize 在 handler 与 service 各夹一次（同一规则两套真相），而 pageSize=0
// 在 Ent 的 sqlgraph 里是 `if q.Limit != 0` —— 0 表示不加 LIMIT，整表返回。现在 HTTP 入口只
// 走 common.GetPaginationFromQuery（缺省 1/20，越界值回落默认而不是夹到 100），service 侧保留
// 一道 common.ValidatePagination 给非 HTTP 调用方；响应五键由 NewPaginationResponse 算出。
// 两个 helper 的默认页长不同（10 vs 20）是已登记的存量债务，见审计计划 E4-9，因此本文件只在
// 真实路由上锁 20 这条对外契约。
//
// 这些断言全部打在**生产注册**上：SetupRoutes → SetupCommonSystemRoutes /
// SetupTicketRoutes → middleware.RequirePermission。

const notifEnvelopeSecret = "notification-envelope-secret"

// notifFixedCreatedAt 让同一批通知的 created_at 完全相同：排序是
// desc(created_at), asc(id)，因此期望顺序就是插入时的升序 ID。
var notifFixedCreatedAt = time.Date(2026, 9, 10, 8, 0, 0, 0, time.UTC)

func TestNotificationListRouteEnvelopeContract(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:router_notif_envelope_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	logger := zaptest.NewLogger(t).Sugar()
	notifService := service.NewNotificationService(client)
	notifPrefService := service.NewNotificationPreferenceService(client, logger)
	ticketNotifService := service.NewTicketNotificationService(client, logger)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetupRoutes(r, &RouterConfig{
		JWTSecret: notifEnvelopeSecret,
		Logger:    logger,
		Client:    client,
		// 通知路由整块挂在 SetupCommonSystemRoutes 里，而后者只在 CommonHandler 非空时
		// 调用；这里必须一起装配，否则测的是「路由没注册」而不是信封契约。
		CommonHandler:             domainCommon.NewHandler(domainCommon.NewService(domainCommon.NewEntRepository(client), notifEnvelopeSecret, logger, client)),
		NotificationHandler:       notificationHandler.NewHandler(notifService, notifPrefService, logger),
		TicketNotificationHandler: ticketNotificationHandler.NewHandler(ticketNotifService, logger),
	})

	// 租户 A：userA 有 25 条通知（其中 10 条已读），userB 有 4 条 —— 用于用户维度收敛。
	tenantA := seedNotifTenant(ctx, t, client, "notif-a")
	userA := notifUserID(ctx, t, client, tenantA, "notif-a-user-a")
	userB := notifUserID(ctx, t, client, tenantA, "notif-a-user-b")
	// 租户 B：2 条通知，用于租户维度收敛。
	tenantB := seedNotifTenant(ctx, t, client, "notif-b")
	userC := notifUserID(ctx, t, client, tenantB, "notif-b-user-a")

	allA := notifIDs(ctx, t, client, userA)
	require.Len(t, allA, 25)

	do := func(t *testing.T, userID, tenantID int, query string) (int, string, notifEnvelope) {
		t.Helper()
		token, err := middleware.GenerateAccessToken(userID, fmt.Sprintf("user-%d", userID), "super_admin", tenantID, notifEnvelopeSecret, time.Hour)
		require.NoError(t, err)

		path := "/api/v1/notifications"
		if query != "" {
			path += "?" + query
		}
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		var env notifEnvelope
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
		return w.Code, w.Body.String(), env
	}

	t.Run("只返回标准五键，不再出现 notifications 或 size", func(t *testing.T) {
		status, body, env := do(t, userA, tenantA, "")
		require.Equal(t, http.StatusOK, status, body)
		require.Equal(t, 0, env.Code, body)

		assert.Equal(t, []string{"items", "page", "pageSize", "total", "totalPages"},
			envelopeKeys(t, env.Data),
			"通知列表必须只有标准信封五键: body=%s", body)

		var res notifListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 1, res.Page)
		assert.Equal(t, 20, res.PageSize)
		assert.Equal(t, 25, res.Total, "total 必须是全量，不是当前页长度")
		assert.Equal(t, 2, res.TotalPages)
		assert.Equal(t, allA[:20], notifItemIDs(res.Items))
	})

	t.Run("三页拼接不重不漏", func(t *testing.T) {
		var paged []int
		for page := 1; page <= 3; page++ {
			_, body, env := do(t, userA, tenantA, fmt.Sprintf("page=%d&pageSize=10", page))
			require.Equal(t, 0, env.Code, body)
			var res notifListWire
			require.NoError(t, json.Unmarshal(env.Data, &res), body)
			assert.Equal(t, page, res.Page)
			assert.Equal(t, 25, res.Total)
			assert.Equal(t, 3, res.TotalPages)
			want := 10
			if page == 3 {
				want = 5
			}
			require.Len(t, res.Items, want)
			paged = append(paged, notifItemIDs(res.Items)...)
		}
		assert.Equal(t, allA, paged, "并列 created_at 必须由 ID 兜底出确定归属")
	})

	t.Run("pageSize=0 夹紧为默认页长而不是整表返回", func(t *testing.T) {
		_, body, env := do(t, userA, tenantA, "page=1&pageSize=0")
		require.Equal(t, 0, env.Code, body)
		var res notifListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 20, res.PageSize)
		assert.Len(t, res.Items, 20, "pageSize=0 在 Ent 里等于不加 LIMIT，修复前返回全部 25 条")
		assert.Equal(t, 2, res.TotalPages)
	})

	t.Run("page 为负数回到第一页", func(t *testing.T) {
		_, body, env := do(t, userA, tenantA, "page=-1&pageSize=10")
		require.Equal(t, 0, env.Code, body)
		var res notifListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 1, res.Page, "负 OFFSET 不得进入查询")
		assert.Equal(t, allA[:10], notifItemIDs(res.Items))
	})

	t.Run("pageSize 超上限回落到默认页长", func(t *testing.T) {
		_, body, env := do(t, userA, tenantA, "page=1&pageSize=5000")
		require.Equal(t, 0, env.Code, body)
		var res notifListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		// common.GetPaginationFromQuery 的语义是「越界回落到默认」而不是截到上限，
		// 与 /changes/pirs 等所有走该通道的列表一致。
		assert.Equal(t, 20, res.PageSize)
		assert.Len(t, res.Items, 20)
	})

	t.Run("read 过滤同步收敛 total 与 totalPages", func(t *testing.T) {
		_, body, env := do(t, userA, tenantA, "read=true")
		require.Equal(t, 0, env.Code, body)
		var res notifListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 10, res.Total, "total 必须是过滤后的数量")
		assert.Len(t, res.Items, 10)
		assert.Equal(t, 1, res.TotalPages)
		for _, item := range res.Items {
			assert.True(t, item.Read)
		}
	})

	t.Run("无通知时 items 序列化为 []", func(t *testing.T) {
		_, body, env := do(t, userC, tenantB, "read=false")
		require.Equal(t, 0, env.Code, body)
		assert.JSONEq(t, "[]", jsonOf(t, env.Data, "items"), body)
		var res notifListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 0, res.Total)
		assert.Equal(t, 0, res.TotalPages)
	})

	t.Run("同租户另一用户看不到别人的通知", func(t *testing.T) {
		_, body, env := do(t, userB, tenantA, "pageSize=100")
		require.Equal(t, 0, env.Code, body)
		var res notifListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 4, res.Total)
		require.Len(t, res.Items, 4)
		for _, item := range res.Items {
			assert.Equal(t, userB, item.UserID)
			assert.NotContains(t, allA, item.ID, "别人的通知 ID 不得出现在响应里")
		}
	})

	t.Run("跨租户不可见", func(t *testing.T) {
		_, body, env := do(t, userC, tenantB, "pageSize=100")
		require.Equal(t, 0, env.Code, body)
		var res notifListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 2, res.Total)
		assert.NotEmpty(t, notifIDs(ctx, t, client, userC))
		for _, item := range res.Items {
			assert.Equal(t, tenantB, item.TenantID)
			assert.NotContains(t, allA, item.ID)
		}
	})

	t.Run("查询参数自报 userId/tenantId 不参与查询", func(t *testing.T) {
		_, body, env := do(t, userB, tenantA, fmt.Sprintf("pageSize=100&userId=%d&tenantId=%d", userA, tenantB))
		require.Equal(t, 0, env.Code, body)
		var res notifListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), body)
		assert.Equal(t, 4, res.Total, "身份只能来自认证上下文，自报参数不得扩大可见范围")
		for _, item := range res.Items {
			assert.Equal(t, userB, item.UserID)
			assert.Equal(t, tenantA, item.TenantID)
		}
	})

	t.Run("未认证返回 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
	})

	// 工单通知：不分页的合法形状，但集合键必须收敛为 items。
	t.Run("工单通知列表返回 {items,total} 且不带领域名键", func(t *testing.T) {
		ticketA, tenantAUser := seedNotifTicket(ctx, t, client, tenantA, "TKT-NOTIF-A")
		seedTicketNotifs(ctx, t, client, ticketA, tenantAUser, tenantA, 3)

		token, err := middleware.GenerateAccessToken(tenantAUser, "ticket-actor", "super_admin", tenantA, notifEnvelopeSecret, time.Hour)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d/notifications", ticketA), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())

		var env notifEnvelope
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), w.Body.String())
		require.Equal(t, 0, env.Code, w.Body.String())
		assert.Equal(t, []string{"items", "total"}, envelopeKeys(t, env.Data),
			"不分页的列表只允许 {items,total}：出现分页键就是伪造契约")

		var res ticketNotifListWire
		require.NoError(t, json.Unmarshal(env.Data, &res), w.Body.String())
		assert.Equal(t, 3, res.Total)
		require.Len(t, res.Items, 3)
	})

	t.Run("工单通知按租户收敛", func(t *testing.T) {
		ticketA, actorA := seedNotifTicket(ctx, t, client, tenantA, "TKT-NOTIF-CROSS-A")
		seedTicketNotifs(ctx, t, client, ticketA, actorA, tenantA, 2)
		ticketB, _ := seedNotifTicket(ctx, t, client, tenantB, "TKT-NOTIF-CROSS-B")
		seedTicketNotifs(ctx, t, client, ticketB, userC, tenantB, 5)

		// 用租户 A 的身份去读租户 B 的工单通知：主查询按 tenant_id 收敛，必须为空。
		token, err := middleware.GenerateAccessToken(actorA, "ticket-actor-a", "super_admin", tenantA, notifEnvelopeSecret, time.Hour)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d/notifications", ticketB), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())

		var env notifEnvelope
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), w.Body.String())
		assert.JSONEq(t, "[]", jsonOf(t, env.Data, "items"), w.Body.String())
		assert.Equal(t, "0", jsonOf(t, env.Data, "total"))
	})
}

// notifListWire 按线上契约声明解析目标，不引用 dto：锁的是响应形状本身。
type notifListWire struct {
	Items      []notifItemWire `json:"items"`
	Total      int             `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"pageSize"`
	TotalPages int             `json:"totalPages"`
}

type notifItemWire struct {
	ID       int  `json:"id"`
	UserID   int  `json:"userId"`
	TenantID int  `json:"tenantId"`
	Read     bool `json:"read"`
}

type ticketNotifListWire struct {
	Items []struct {
		ID       int `json:"id"`
		TicketID int `json:"ticketId"`
	} `json:"items"`
	Total int `json:"total"`
}

type notifEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func notifItemIDs(items []notifItemWire) []int {
	ids := make([]int, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

// seedNotifTenant 建一个租户及其两个用户：user-a 25 条（10 条已读）、user-b 4 条。
func seedNotifTenant(ctx context.Context, t *testing.T, client *ent.Client, code string) int {
	t.Helper()

	tenant, err := client.Tenant.Create().
		SetName("Notif " + code).
		SetCode(code).
		SetDomain(code + ".example.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	plan := map[string]struct{ total, read int }{
		"user-a": {25, 10},
		"user-b": {4, 0},
	}
	if code == "notif-b" {
		plan = map[string]struct{ total, read int }{
			"user-a": {2, 2},
		}
	}

	for suffix, counts := range plan {
		user, err := client.User.Create().
			SetUsername(code + "-" + suffix).
			SetEmail(code + "-" + suffix + "@example.com").
			SetName("Notif " + suffix).
			SetPasswordHash("hash").
			SetRole("end_user").
			SetActive(true).
			SetTenantID(tenant.ID).
			Save(ctx)
		require.NoError(t, err)
		seedNotifications(ctx, t, client, user.ID, tenant.ID, counts.total, counts.read)
	}
	return tenant.ID
}

// seedNotifications 建 total 条通知，前 read 条已读；created_at 全部相同以逼出 ID 兜底排序。
func seedNotifications(ctx context.Context, t *testing.T, client *ent.Client, userID, tenantID, total, read int) {
	t.Helper()

	for i := 0; i < total; i++ {
		_, err := client.Notification.Create().
			SetTitle("通知").
			SetMessage(fmt.Sprintf("第 %d 条", i)).
			SetType("info").
			SetRead(i < read).
			SetUserID(userID).
			SetTenantID(tenantID).
			SetCreatedAt(notifFixedCreatedAt).
			Save(ctx)
		require.NoError(t, err)
	}
}

func notifUserID(ctx context.Context, t *testing.T, client *ent.Client, tenantID int, username string) int {
	t.Helper()
	found, err := client.User.Query().Where(user.Username(username)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, tenantID, found.TenantID)
	return found.ID
}

// notifIDs 返回某用户通知的完整升序 ID 序列。
func notifIDs(ctx context.Context, t *testing.T, client *ent.Client, userID int) []int {
	t.Helper()
	ids, err := client.Notification.Query().
		Where(notification.UserID(userID)).
		IDs(ctx)
	require.NoError(t, err)
	sort.Ints(ids)
	return ids
}

// seedNotifTicket 在指定租户建一条工单及其请求人，返回工单 ID 与用户 ID。
func seedNotifTicket(ctx context.Context, t *testing.T, client *ent.Client, tenantID int, number string) (int, int) {
	t.Helper()

	user, err := client.User.Create().
		SetUsername(number).
		SetEmail(number + "@example.com").
		SetName("Ticket Actor").
		SetPasswordHash("hash").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)

	ticket, err := client.Ticket.Create().
		SetTicketNumber(number).
		SetTitle("通知信封回归").
		SetDescription("验证工单通知列表的 items 键").
		SetType("incident").
		SetPriority("medium").
		SetStatus("open").
		SetRequesterID(user.ID).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return ticket.ID, user.ID
}

func seedTicketNotifs(ctx context.Context, t *testing.T, client *ent.Client, ticketID, userID, tenantID, count int) {
	t.Helper()

	for i := 0; i < count; i++ {
		_, err := client.TicketNotification.Create().
			SetTicketID(ticketID).
			SetUserID(userID).
			SetType("commented").
			SetChannel("in_app").
			SetContent("通知内容").
			SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err)
	}
}
