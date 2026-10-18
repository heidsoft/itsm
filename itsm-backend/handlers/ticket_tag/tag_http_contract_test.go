// Package ticket_tag 的 CRUD 与工单绑定面 HTTP 契约回归（B10.1）。
//
// 路由与权限码：
//
//	/ticket-tags 组（router/ticket_tag_routes.go）→ ticket_tag:{read,create,update,delete}
//	POST/DELETE /tickets/:id/tags（router/ticket_routes.go:290）→ ticket_tag:{create,delete}
//
// 锁定语义：
//   - [TT1 已修复] DELETE /ticket-tags/{id} 条件删除验 affected rows：
//     重放/跨租户删除 → 404/4004（此前 0 行静默 nil → 200 假成功）。
//   - [TT2 既有正确面] 同租户重名 → 409/4090；空名 → 400/1001；被工单引用的
//     标签删除 → 409/4090；assign 跨租户标签/工单 → 404/4004 掩码；
//     assign 幂等（重复绑定同标签不报错不重复落库）。响应 camelCase。
package ticket_tag_test

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
	"itsm-backend/ent/tickettag"
	tagHandler "itsm-backend/handlers/ticket_tag"
	"itsm-backend/middleware"
	"itsm-backend/router"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type fixture struct {
	engine  *gin.Engine
	client  *ent.Client
	secret  string
	tenantA *ent.Tenant
	tenantB *ent.Tenant
	userA   *ent.User
	userB   *ent.User
	ticketA *ent.Ticket
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:tag_http_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := client.Tenant.Create().SetName("Tag A").SetCode("tag-a").SetDomain("tag-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("Tag B").SetCode("tag-b").SetDomain("tag-b.example.com").SetStatus("active").SaveX(ctx)

	userA := client.User.Create().SetUsername("tag-a-owner").SetEmail("tag-a@example.com").SetName("Owner A").
		SetPasswordHash("hash").SetRole("admin").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	userB := client.User.Create().SetUsername("tag-b-owner").SetEmail("tag-b@example.com").SetName("Owner B").
		SetPasswordHash("hash").SetRole("admin").SetActive(true).SetTenantID(tenantB.ID).SaveX(ctx)
	ticketA := client.Ticket.Create().SetTicketNumber("TT-A-001").SetTitle("标签工单").
		SetStatus("open").SetPriority("medium").SetRequesterID(userA.ID).
		SetTenantID(tenantA.ID).SaveX(ctx)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	const secret = "tag-contract-secret"
	router.SetupRoutes(engine, &router.RouterConfig{
		JWTSecret: secret,
		Logger:    logger,
		Client:    client,
		TicketTagHandler: tagHandler.NewHandler(
			service.NewTicketTagService(client), logger),
	})

	return &fixture{
		engine: engine, client: client, secret: secret,
		tenantA: tenantA, tenantB: tenantB, userA: userA, userB: userB, ticketA: ticketA,
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

func TestTicketTagHTTPContract(t *testing.T) {
	fx := newFixture(t)
	base := "/api/v1/ticket-tags"
	ctx := context.Background()

	var createdID int
	t.Run("create与list正控制及camelCase", func(t *testing.T) {
		w := fx.do(t, http.MethodPost, base, fx.userA, `{"name":"紧急","description":"高优标签"}`)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 0, decodeEnvelope(t, w).Code)
		body := w.Body.String()
		require.Contains(t, body, `"isActive"`, "字段必须是 camelCase")
		require.Contains(t, body, `"color":"#1890ff"`, "未传 color 应补默认色")
		require.NotContains(t, body, `"is_active"`, "不得出现 snake_case 字段")

		// 记录创建 id（列表取回）
		w = fx.do(t, http.MethodGet, base, fx.userA)
		require.Equal(t, http.StatusOK, w.Code)
		var env envelope = decodeEnvelope(t, w)
		var list struct {
			Items []struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"items"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &list))
		require.NotEmpty(t, list.Items)
		require.Equal(t, "紧急", list.Items[0].Name)
		createdID = list.Items[0].ID
	})

	t.Run("同名标签409与空名400", func(t *testing.T) {
		w := fx.do(t, http.MethodPost, base, fx.userA, `{"name":"紧急"}`)
		require.Equal(t, http.StatusConflict, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4090, decodeEnvelope(t, w).Code)

		w = fx.do(t, http.MethodPost, base, fx.userA, `{"name":"   "}`)
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 1001, decodeEnvelope(t, w).Code)
	})

	t.Run("get跨租户404掩码", func(t *testing.T) {
		tagB := fx.client.TicketTag.Create().SetName("他租标签").SetTenantID(fx.tenantB.ID).SaveX(ctx)
		w := fx.do(t, http.MethodGet, fmt.Sprintf("%s/%d", base, tagB.ID), fx.userA)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)
	})

	t.Run("update正控制与他租404", func(t *testing.T) {
		w := fx.do(t, http.MethodPut, fmt.Sprintf("%s/%d", base, createdID), fx.userA,
			`{"color":"#ff0000","isActive":false}`)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"color":"#ff0000"`)

		tagB := fx.client.TicketTag.Create().SetName("他租更新目标").SetTenantID(fx.tenantB.ID).SaveX(ctx)
		w = fx.do(t, http.MethodPut, fmt.Sprintf("%s/%d", base, tagB.ID), fx.userA, `{"color":"#000000"}`)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)
	})

	t.Run("delete重放跨租户是404不再假成功", func(t *testing.T) {
		oneOff := fx.client.TicketTag.Create().SetName("待删标签").SetTenantID(fx.tenantA.ID).SaveX(ctx)
		w := fx.do(t, http.MethodDelete, fmt.Sprintf("%s/%d", base, oneOff.ID), fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		// 重放删除 → 404（TT1：此前 0 行静默 200 假成功）
		w = fx.do(t, http.MethodDelete, fmt.Sprintf("%s/%d", base, oneOff.ID), fx.userA)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)

		// 跨租户删除 → 同掩码且对端保留
		tagB := fx.client.TicketTag.Create().SetName("他租待删标签").SetTenantID(fx.tenantB.ID).SaveX(ctx)
		w = fx.do(t, http.MethodDelete, fmt.Sprintf("%s/%d", base, tagB.ID), fx.userA)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)
		_, err := fx.client.TicketTag.Query().Where(tickettag.IDEQ(tagB.ID)).First(ctx)
		require.NoError(t, err, "跨租户删除不得注销对端行")
	})

	t.Run("assign正控幂等与跨租户404", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/tickets/%d/tags", fx.ticketA.ID)
		w := fx.do(t, http.MethodPost, path, fx.userA, fmt.Sprintf(`{"tagIds":[%d]}`, createdID))
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		// 幂等：重复 assign 同标签 → 200，不产生重复绑定
		w = fx.do(t, http.MethodPost, path, fx.userA, fmt.Sprintf(`{"tagIds":[%d]}`, createdID))
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		// 绑定走 ticket 侧的 tags edge（tickettag.tickets 是另一张独立 join 表）
		edges, err := fx.client.Ticket.Query().Where(ticket.IDEQ(fx.ticketA.ID)).
			QueryTags().Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, edges, "重复绑定不得产生第二条关联")

		// 跨租户工单 → 404 掩码
		ticketB := fx.client.Ticket.Create().SetTicketNumber("TT-B-001").SetTitle("他租工单").
			SetStatus("open").SetPriority("medium").SetRequesterID(fx.userB.ID).
			SetTenantID(fx.tenantB.ID).SaveX(ctx)
		w = fx.do(t, http.MethodPost, fmt.Sprintf("/api/v1/tickets/%d/tags", ticketB.ID),
			fx.userA, fmt.Sprintf(`{"tagIds":[%d]}`, createdID))
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)

		// 跨租户标签 → 404 掩码（sentinel ErrTicketTagNotFound）
		tagB := fx.client.TicketTag.Create().SetName("他租绑定标签").SetTenantID(fx.tenantB.ID).SaveX(ctx)
		w = fx.do(t, http.MethodPost, path, fx.userA, fmt.Sprintf(`{"tagIds":[%d]}`, tagB.ID))
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)
	})

	t.Run("在用标签删除409与移除后可删", func(t *testing.T) {
		// createdID 已绑定 fx.ticketA → 删除应 409
		w := fx.do(t, http.MethodDelete, fmt.Sprintf("%s/%d", base, createdID), fx.userA)
		require.Equal(t, http.StatusConflict, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4090, decodeEnvelope(t, w).Code)

		// 解绑后可删
		w = fx.do(t, http.MethodDelete,
			fmt.Sprintf("/api/v1/tickets/%d/tags", fx.ticketA.ID), fx.userA,
			fmt.Sprintf(`{"tagIds":[%d]}`, createdID))
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		w = fx.do(t, http.MethodDelete, fmt.Sprintf("%s/%d", base, createdID), fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	})
}
