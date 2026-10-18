// Package change_review 的评审组成员面 HTTP 回归（B9.1，含钉住缺陷）。
//
// 权限码：/change-review/members* 用 change:{read,write}（模型域成员管理，
// 复用 change 域权限，与 §1.41 口径一致）；fixture 用户 super_admin 直通。
//
// 钉住契约（有意保留的现状 + 已知缺陷，修复时必须同步改断言）：
//   - [C1 已修复] POST /change-review/members 引入他租/不存在 user →
//     400/1001（引用型资源缺失，与工单评论「提及用户无效」同口径）。
//   - [C2 已修复] POST /change-review/members 重复成员 → 409/4090。
//   - [C3 已修复] DELETE /change-review/members/{他租id} → 404/4004。
//   - [C4 已修复] PUT /change-review/members/{他租id} → 404/4004。
//   - [C5 已修复] GET /change-review/members 空租户列表序列化为 []（不再 null）。
//   - AddMember 的用户租户谓词已带 tenant_id（跨租户引用不会写入行），
//     但错误形态被压成 500，存在性可探测。
package change_review_test

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
	"itsm-backend/ent/changereviewmember"
	"itsm-backend/ent/enttest"
	changeReviewHandler "itsm-backend/handlers/change_review"
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
	endUser  *ent.User
	stranger *ent.User
	memberA1 *ent.ChangeReviewMember
	memberB1 *ent.ChangeReviewMember
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:change_review_http_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := client.Tenant.Create().SetName("CR A").SetCode("cr-a").SetDomain("cr-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("CR B").SetCode("cr-b").SetDomain("cr-b.example.com").SetStatus("active").SaveX(ctx)

	userA := client.User.Create().SetUsername("cr-a-owner").SetEmail("cr-a@example.com").SetName("Owner A").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	endUser := client.User.Create().SetUsername("cr-a-enduser").SetEmail("cr-a-end@example.com").SetName("EndUser A").
		SetPasswordHash("hash").SetRole("end_user").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	stranger := client.User.Create().SetUsername("cr-b-stranger").SetEmail("cr-b@example.com").SetName("Stranger B").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantB.ID).SaveX(ctx)

	mkMember := func(name string, tenant *ent.Tenant, user *ent.User) *ent.ChangeReviewMember {
		return client.ChangeReviewMember.Create().
			SetUserID(user.ID).
			SetType("REVIEW").
			SetRole("member").
			SetIsActive(true).
			SetTenantID(tenant.ID).
			SaveX(ctx)
	}
	memberA1 := mkMember("成员A1", tenantA, userA)
	memberB1 := mkMember("成员B1", tenantB, stranger)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	const secret = "change-review-secret"
	router.SetupRoutes(engine, &router.RouterConfig{
		JWTSecret:           secret,
		Logger:              logger,
		Client:              client,
		ChangeReviewHandler: changeReviewHandler.NewHandler(service.NewChangeReviewService(client, logger), logger),
	})

	return &fixture{
		engine: engine, client: client, secret: secret,
		tenantA: tenantA, tenantB: tenantB,
		userA: userA, endUser: endUser, stranger: stranger,
		memberA1: memberA1, memberB1: memberB1,
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

func TestChangeReviewMemberTenantIsolation(t *testing.T) {
	fx := newFixture(t)
	base := "/api/v1/change-review/members"
	ctx := context.Background()

	t.Run("同租列表正控制含camelCase", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, base+"?type=REVIEW", fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		env := decodeEnvelope(t, w)
		require.Equal(t, 0, env.Code)
		var members []struct {
			ID       int    `json:"id"`
			UserID   int    `json:"userId"`
			UserName string `json:"userName"`
			IsActive bool   `json:"isActive"`
			TenantID int    `json:"tenantId"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &members), "data=%s", string(env.Data))
		require.NotEmpty(t, members)
		for _, m := range members {
			require.Equal(t, fx.tenantA.ID, m.TenantID, "列表只含本租户成员")
			require.False(t, strings.Contains(m.UserName, "Stranger B"), "不得混入他租用户")
		}
	})

	t.Run("空列表序列化为数组而非null", func(t *testing.T) {
		// [C5] 空租户列表契约：应为 []（null 会迫使前端逐处判空）。
		tenantC := fx.client.Tenant.Create().SetName("CR C").SetCode("cr-c").SetDomain("cr-c.example.com").SetStatus("active").SaveX(ctx)
		userC := fx.client.User.Create().SetUsername("cr-c-owner").SetEmail("cr-c@example.com").SetName("Owner C").
			SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantC.ID).SaveX(ctx)
		w := fx.do(t, http.MethodGet, base, userC)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), "[]", "空列表必须序列化为 []，不得是 null")
	})

	t.Run("无效type回落REVIEW不动他租数据", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, base+"?type=BOGUS", fx.userA)
		require.Equal(t, http.StatusOK, w.Code)
		b, _ := fx.client.ChangeReviewMember.Query().Where(changereviewmember.TenantIDEQ(fx.tenantA.ID)).Count(ctx)
		require.Equal(t, 1, b)
	})

	t.Run("add正控制且跨租户user是400掩码", func(t *testing.T) {
		w := fx.do(t, http.MethodPost, base, fx.userA,
			`{"userId":`+fmt.Sprint(fx.userA.ID)+`,"type":"EREVIEW","role":"member"}`)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"userId":`+fmt.Sprint(fx.userA.ID))

		w = fx.do(t, http.MethodPost, base, fx.userA,
			`{"userId":`+fmt.Sprint(fx.stranger.ID)+`,"type":"REVIEW"}`)
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 1001, decodeEnvelope(t, w).Code)
		// 零副作用守卫：对端用户未被写入本租成员行
		n, err := fx.client.ChangeReviewMember.Query().Where(changereviewmember.TenantIDEQ(fx.tenantA.ID)).Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 2, n, "跨租户引用不得落库（原有 REVIEW 成员 + 新 EREVIEW 成员）")
	})

	t.Run("重复成员是409业务冲突", func(t *testing.T) {
		w := fx.do(t, http.MethodPost, base, fx.userA,
			`{"userId":`+fmt.Sprint(fx.userA.ID)+`,"type":"REVIEW"}`)
		require.Equal(t, http.StatusConflict, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4090, decodeEnvelope(t, w).Code)
	})

	t.Run("跨租户Remove与Update是404掩码", func(t *testing.T) {
		w := fx.do(t, http.MethodDelete, fmt.Sprintf("%s/%d", base, fx.memberB1.ID), fx.userA)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)
		n, err := fx.client.ChangeReviewMember.Query().Where(changereviewmember.TenantIDEQ(fx.tenantB.ID)).Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, n, "跨租户删除不得注销对端行")

		w = fx.do(t, http.MethodPut, fmt.Sprintf("%s/%d", base, fx.memberB1.ID), fx.userA, `{"role":"chair","isActive":false}`)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)
	})

	t.Run("本租Update与Remove正控制", func(t *testing.T) {
		w := fx.do(t, http.MethodPut, fmt.Sprintf("%s/%d", base, fx.memberA1.ID), fx.userA, `{"role":"chair","isActive":true}`)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"role":"chair"`)

		w = fx.do(t, http.MethodDelete, fmt.Sprintf("%s/%d", base, fx.memberA1.ID), fx.userA)
		require.Equal(t, http.StatusOK, w.Code)
		n, err := fx.client.ChangeReviewMember.Query().Where(changereviewmember.TenantIDEQ(fx.tenantA.ID)).Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, n, "仅剩 add 正控制新增的 EREVIEW 成员")
	})

	t.Run("成员面权限码403禁拒", func(t *testing.T) {
		// change:read/write；end_user 不持有写权限。
		w := fx.do(t, http.MethodPost, base, fx.endUser, `{"userId":1,"type":"REVIEW"}`)
		require.Equal(t, http.StatusForbidden, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 2003, decodeEnvelope(t, w).Code)
		w = fx.do(t, http.MethodDelete, fmt.Sprintf("%s/%d", base, 1), fx.endUser)
		require.Equal(t, http.StatusForbidden, w.Code, "body=%s", w.Body.String())
	})
}
