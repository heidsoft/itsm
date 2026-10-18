// Package ticket_rating 的评分面 HTTP 契约回归（B10.1）。
//
// 路由与权限码（router/ticket_routes.go:283）：
//
//	POST /tickets/:id/rating → ticket:update
//	GET  /tickets/:id/rating → ticket:read
//	GET  /tickets/rating-stats → ticket:read（租户取自上下文）
//
// 锁定语义：
//   - [TR1 已修复] 业务违规不再兜底 500：待解决工单评分 → 400/1001；
//     非申请人评分 → 403/2003；重复评分 → 409/4090。
//   - [TR2 已修复] 评分落库改为条件更新（Rating=0）；并发双提交在
//     「读取时未评 → 写入时已被评」窗口内第二次命中 0 行 → 409/4090，
//     不再静默覆盖第一次评分。
//   - [TR3 既有正确面] 跨租户/不存在工单 → ent NotFound 包装 → 404/4004；
//     统计只聚合本租可见评分，响应字段 camelCase。
package ticket_rating_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/ticket"
	ratingHandler "itsm-backend/handlers/ticket_rating"
	"itsm-backend/middleware"
	"itsm-backend/router"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type fixture struct {
	engine         *gin.Engine
	client         *ent.Client
	secret         string
	tenantA        *ent.Tenant
	tenantB        *ent.Tenant
	requesterA     *ent.User // 申请人（agent，持 ticket:update）
	endUser        *ent.User // 评分角色面正控（持 ticket:update）
	auditAdminUser *ent.User // audit_admin：不持 ticket:update
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:rating_http_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := client.Tenant.Create().SetName("Rating A").SetCode("rating-a").SetDomain("rating-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("Rating B").SetCode("rating-b").SetDomain("rating-b.example.com").SetStatus("active").SaveX(ctx)

	requesterA := client.User.Create().SetUsername("rating-a-agent").SetEmail("rating-a@example.com").SetName("Agent A").
		SetPasswordHash("hash").SetRole("agent").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	endUser := client.User.Create().SetUsername("rating-a-enduser").SetEmail("rating-a-end@example.com").SetName("EndUser A").
		SetPasswordHash("hash").SetRole("end_user").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	auditAdminUser := client.User.Create().SetUsername("rating-a-audit").SetEmail("rating-a-audit@example.com").SetName("Audit A").
		SetPasswordHash("hash").SetRole("audit_admin").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	const secret = "rating-contract-secret"
	router.SetupRoutes(engine, &router.RouterConfig{
		JWTSecret: secret,
		Logger:    logger,
		Client:    client,
		TicketRatingHandler: ratingHandler.NewHandler(
			service.NewTicketRatingService(client, logger), logger),
	})

	return &fixture{
		engine: engine, client: client, secret: secret,
		tenantA: tenantA, tenantB: tenantB,
		requesterA: requesterA, endUser: endUser, auditAdminUser: auditAdminUser,
	}
}

func (fx *fixture) do(t *testing.T, method, path string, user *ent.User, payload ...string) *httptest.ResponseRecorder {
	t.Helper()
	token, err := middleware.GenerateAccessToken(user.ID, user.Username, string(user.Role), user.TenantID, fx.secret, time.Hour)
	require.NoError(t, err)
	var reader io.Reader
	if len(payload) > 0 {
		reader = strings.NewReader(payload[0])
	}
	req := httptest.NewRequest(method, path, reader)
	if len(payload) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	fx.engine.ServeHTTP(w, req)
	return w
}

type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder) envelope {
	t.Helper()
	var env envelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
	return env
}

func (fx *fixture) createTicket(t *testing.T, ctx context.Context, code string, status string, tenantID, requesterID int) *ent.Ticket {
	t.Helper()
	return fx.client.Ticket.Create().SetTicketNumber(code).SetTitle("评分工单 " + code).
		SetStatus(status).SetPriority("medium").SetRequesterID(requesterID).
		SetTenantID(tenantID).SaveX(ctx)
}

func TestTicketRatingHTTPContract(t *testing.T) {
	fx := newFixture(t)
	ctx := context.Background()
	pending := fx.createTicket(t, ctx, "RT-PENDING", "open", fx.tenantA.ID, fx.requesterA.ID)
	resolved := fx.createTicket(t, ctx, "RT-RESOLVED", "resolved", fx.tenantA.ID, fx.requesterA.ID)
	ticketB := fx.createTicket(t, ctx, "RT-B", "resolved", fx.tenantB.ID, fx.requesterA.ID)

	t.Run("待解决工单评分400不再500", func(t *testing.T) {
		w := fx.do(t, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/rating", pending.ID),
			fx.requesterA, `{"rating":5}`)
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 1001, decodeEnvelope(t, w).Code)
	})

	t.Run("非申请人评分是403", func(t *testing.T) {
		// 同租户的 end_user 持 ticket:update（放行权限面），但不是申请人 → 403/2003。
		w := fx.do(t, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/rating", resolved.ID),
			fx.endUser, `{"rating":2}`)
		require.Equal(t, http.StatusForbidden, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 2003, decodeEnvelope(t, w).Code)
	})

	t.Run("正控评分落库与camelCase", func(t *testing.T) {
		w := fx.do(t, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/rating", resolved.ID),
			fx.requesterA, `{"rating":5,"comment":"服务满意"}`)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 0, decodeEnvelope(t, w).Code)
		require.Contains(t, w.Body.String(), `"ratedAt"`, "字段必须是 camelCase")
		require.NotContains(t, w.Body.String(), `"rated_at"`, "不得出现 snake_case 字段")
	})

	t.Run("重复评分是409", func(t *testing.T) {
		w := fx.do(t, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/rating", resolved.ID),
			fx.requesterA, `{"rating":4}`)
		require.Equal(t, http.StatusConflict, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4090, decodeEnvelope(t, w).Code)
		// 已有评分不被覆盖
		after, err := fx.client.Ticket.Query().Where(ticket.IDEQ(resolved.ID)).Only(ctx)
		require.NoError(t, err)
		require.Equal(t, 5, after.Rating, "重复提交不得覆盖第一次评分")
	})

	t.Run("条件更新挡并发覆盖重放", func(t *testing.T) {
		// 直接命中条件更新写入面：Rating 已非 0 的行再走条件更新必 0 行命中。
		// 等价于竞态窗口二次请求与服务端 remove+rewrite 之间的不变量：
		// 「写入时 Rating=0」是评分原子性的唯一守卫（TR2）。
		fresh := fx.createTicket(t, ctx, "RT-RACE", "closed", fx.tenantA.ID, fx.requesterA.ID)
		fx.client.Ticket.UpdateOneID(fresh.ID).SetRating(5).ExecX(ctx)
		w := fx.do(t, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/rating", fresh.ID),
			fx.requesterA, `{"rating":1}`)
		require.Equal(t, http.StatusConflict, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4090, decodeEnvelope(t, w).Code)
		after, err := fx.client.Ticket.Query().Where(ticket.IDEQ(fresh.ID)).Only(ctx)
		require.NoError(t, err)
		require.Equal(t, 5, after.Rating)
	})

	t.Run("跨租户评分是404掩码", func(t *testing.T) {
		w := fx.do(t, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/rating", ticketB.ID),
			fx.requesterA, `{"rating":5}`)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)
		w = fx.do(t, http.MethodGet, fmt.Sprintf("/api/v1/tickets/%d/rating", ticketB.ID), fx.requesterA)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)
	})

	t.Run("stats只统计本租且camelCase", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, "/api/v1/tickets/rating-stats", fx.requesterA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		body := w.Body.String()
		require.Contains(t, body, `"totalRatings"`, "字段必须是 camelCase")
		require.Contains(t, body, `"averageRating"`)
		require.NotContains(t, body, `"total_ratings"`, "不得出现 snake_case 字段")
	})

	t.Run("ticket:update权限403禁拒", func(t *testing.T) {
		// audit_admin 不持 ticket:update（roles.go 权威词表；user.role 枚举无 problem_manager）。
		fresh := fx.createTicket(t, ctx, "RT-PM", "resolved", fx.tenantA.ID, fx.requesterA.ID)
		w := fx.do(t, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/rating", fresh.ID),
			fx.auditAdminUser, `{"rating":5}`)
		require.Equal(t, http.StatusForbidden, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 2003, decodeEnvelope(t, w).Code)
	})
}
