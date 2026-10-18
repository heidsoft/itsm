// Package ticket_comment 的跨租户 HTTP 回归测试。
//
// 工单评论是最典型的内容泄漏面：评论可能含内部备注（isInternal）与 @ 提及。
// 隔离保证必须从真实生产入口（router.SetupRoutes 布线 + JWT + RBAC + handler +
// service + ent）发起验证。参照 router/ticket_attachment_route_test.go 的搭台方式。
//
// 已钉住的契约：
//   - 跨租户读/写评论 fail closed，且与「资源不存在」错误形态不可区分；
//   - 跨租户的创建/更新/删除都零副作用（不建行、不改内容、不删行）；
//   - Create/List 的 not-found 已按业务码映射 404/4004（service 用
//     BusinessError/对 ent.NotFoundError 包装，classifyError 可识别）；
//   - Update/Delete 的跨租户失败同样稳定 404/4004。
package ticket_comment_test

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
	ticketCommentHandler "itsm-backend/handlers/ticket_comment"
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
	userA    *ent.User // tenant A super_admin：工单申请人 + 评论作者
	stranger *ent.User // tenant B super_admin：能穿过 RBAC，但工单在租户 A
	ticketA  *ent.Ticket
	commentA *ent.TicketComment
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:ticket_comment_cross_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := client.Tenant.Create().SetName("Comment A").SetCode("comment-a").SetDomain("comment-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("Comment B").SetCode("comment-b").SetDomain("comment-b.example.com").SetStatus("active").SaveX(ctx)

	userA := client.User.Create().SetUsername("comment-a-owner").SetEmail("comment-a@example.com").SetName("Owner A").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	stranger := client.User.Create().SetUsername("comment-b-stranger").SetEmail("comment-b@example.com").SetName("Stranger B").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantB.ID).SaveX(ctx)

	ticketA := client.Ticket.Create().
		SetTicketNumber("TKT-COMMENT-A-001").
		SetTitle("评论跨租户回归").
		SetDescription("验证评论读写的租户隔离").
		SetType("incident").
		SetPriority("high").
		SetStatus("open").
		SetRequesterID(userA.ID).
		SetTenantID(tenantA.ID).
		SaveX(ctx)

	const baselineContent = "内部基线回复内容"
	commentA := client.TicketComment.Create().
		SetTicketID(ticketA.ID).
		SetUserID(userA.ID).
		SetContent(baselineContent).
		SetTenantID(tenantA.ID).
		SaveX(ctx)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	const secret = "ticket-comment-cross-tenant-secret"
	router.SetupRoutes(engine, &router.RouterConfig{
		JWTSecret:            secret,
		Logger:               logger,
		Client:               client,
		TicketCommentHandler: ticketCommentHandler.NewHandler(service.NewTicketCommentService(client, logger), logger),
	})

	return &fixture{
		engine: engine, client: client, secret: secret,
		tenantA: tenantA, tenantB: tenantB, userA: userA, stranger: stranger,
		ticketA: ticketA, commentA: commentA,
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

func (fx *fixture) nComments(t *testing.T) int {
	t.Helper()
	rows, err := fx.client.TicketComment.Query().All(context.Background())
	require.NoError(t, err)
	return len(rows)
}

func TestTicketCommentCrossTenant_HTTP(t *testing.T) {
	fx := newFixture(t)
	ctx := context.Background()
	commentsPath := fmt.Sprintf("/api/v1/tickets/%d/comments", fx.ticketA.ID)

	t.Run("同租户作者全链路（正控制）", func(t *testing.T) {
		w := fx.do(t, http.MethodPost, commentsPath, fx.userA, []byte(`{"content":"owner 首条回复"}`))
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"code":0`)
		require.Contains(t, w.Body.String(), "owner 首条回复")

		wList := fx.do(t, http.MethodGet, commentsPath, fx.userA, nil)
		require.Equal(t, http.StatusOK, wList.Code)
		require.Contains(t, wList.Body.String(), "内部基线回复内容", "正控制：种子评论必须真实可见")
		require.Contains(t, wList.Body.String(), "owner 首条回复")
	})

	t.Run("跨租户列表 fail closed 且零内容泄漏", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, commentsPath, fx.stranger, nil)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"code":4004`)
		assert.NotContains(t, w.Body.String(), "内部基线回复内容")
		assert.NotContains(t, w.Body.String(), "owner 首条回复")
	})

	t.Run("跨租户创建 fail closed 且零副作用", func(t *testing.T) {
		before := fx.nComments(t)
		w := fx.do(t, http.MethodPost, commentsPath, fx.stranger, []byte(`{"content":"跨租户注入评论"}`))
		// not-found 稳定映射 404/4004（不再是 5001/500）。
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"code":4004`)

		assert.Equal(t, before, fx.nComments(t), "跨租户创建不得产生任何评论行")
		rows, err := fx.client.TicketComment.Query().All(ctx)
		require.NoError(t, err)
		for _, r := range rows {
			assert.NotContains(t, r.Content, "跨租户注入", "注入内容不得落库")
		}
	})

	t.Run("跨租户更新 404 且内容不被篡改", func(t *testing.T) {
		w := fx.do(t, http.MethodPut,
			fmt.Sprintf("%s/%d", commentsPath, fx.commentA.ID), fx.stranger, []byte(`{"content":"被篡改的内容"}`))
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"code":4004`)

		comment, err := fx.client.TicketComment.Get(ctx, fx.commentA.ID)
		require.NoError(t, err)
		assert.Equal(t, "内部基线回复内容", comment.Content, "跨租户更新不得落库任何改动")
	})

	t.Run("跨租户删除 404 且评论原样留存", func(t *testing.T) {
		before := fx.nComments(t)
		w := fx.do(t, http.MethodDelete,
			fmt.Sprintf("%s/%d", commentsPath, fx.commentA.ID), fx.stranger, nil)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"code":4004`)

		assert.Equal(t, before, fx.nComments(t), "跨租户删除不得删掉租户内评论")
		_, err := fx.client.TicketComment.Get(ctx, fx.commentA.ID)
		require.NoError(t, err, "被探针命中的评论必须原样留存")
	})

	t.Run("不存在的工单与跨租户工单错误形态不可区分", func(t *testing.T) {
		wCross := fx.do(t, http.MethodGet, commentsPath, fx.stranger, nil)
		wMissing := fx.do(t, http.MethodGet, "/api/v1/tickets/999999/comments", fx.stranger, nil)
		assert.Equal(t, wCross.Code, wMissing.Code)
		assert.Equal(t, wCross.Body.String(), wMissing.Body.String(),
			"租外墙外探测不得靠响应差异枚举工单存在性")
	})

	t.Run("不存在的评论与跨租户评论更新同形态", func(t *testing.T) {
		wCross := fx.do(t, http.MethodPut,
			fmt.Sprintf("%s/%d", commentsPath, fx.commentA.ID), fx.stranger, []byte(`{"content":"篡改"}`))
		wMissing := fx.do(t, http.MethodPut,
			fmt.Sprintf("%s/999999", commentsPath), fx.stranger, []byte(`{"content":"篡改"}`))
		assert.Equal(t, wCross.Code, wMissing.Code)
		assert.Equal(t, wCross.Body.String(), wMissing.Body.String())
	})
}
