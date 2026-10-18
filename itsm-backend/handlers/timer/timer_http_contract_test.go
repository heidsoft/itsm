// Package timer 的读面 HTTP 契约回归（B10.1）。
//
// 路由与权限码（router/router.go:415）：auth 组 /timers，仅读面
// （List/Get/Stats）→ system:read。
//
// 锁定语义：
//   - [TM1 已修复] 不存在的 timerID：ent NotFound 在 service.Get 换算成
//     notFoundError 掩码 → 404/4004（此前直接上抛 → handler InternalError
//     → 500/5001，把「不存在」伪装成服务端故障）。跨租户 timerID 保持
//     读后掩码同形 404（timerID 为 BPMN 生成的全局唯一字符串，存在性
//     侧信道可忽略；不改变 list/stats 的谓词过滤面）。
//   - [TM2 既有正确面] List 按 tenantID 过滤、只含本租 timer 行；
//     信封字段 camelCase（timerId/processDefinitionKey/...）。
//   - [TM3 既有正确面] Stats 按租户聚合。
//   - [TM4 既有正确面] end_user 不持 system:read → 403/2003。
package timer_test

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
	"itsm-backend/middleware"
	"itsm-backend/router"

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
	endUser *ent.User // end_user 不持 system:read → 403 探针
	timerA  string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:timer_http_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := client.Tenant.Create().SetName("Timer A").SetCode("timer-a").SetDomain("timer-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("Timer B").SetCode("timer-b").SetDomain("timer-b.example.com").SetStatus("active").SaveX(ctx)

	userA := client.User.Create().SetUsername("timer-a-owner").SetEmail("timer-a@example.com").SetName("Owner A").
		SetPasswordHash("hash").SetRole("security_admin").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	endUser := client.User.Create().SetUsername("timer-a-enduser").SetEmail("timer-a-end@example.com").SetName("EndUser A").
		SetPasswordHash("hash").SetRole("end_user").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)

	timerA := client.ProcessTimer.Create().SetTimerID("timer-a-001").SetTimerType("start").
		SetProcessDefinitionKey("process-a").SetTimerExpression("PT1H").
		SetExpressionType("duration").SetFireAt(time.Now().Add(time.Hour)).
		SetIdempotencyKey("timer-a-001-key").SetTenantID(tenantA.ID).SaveX(ctx)
	client.ProcessTimer.Create().SetTimerID("timer-b-001").SetTimerType("start").
		SetProcessDefinitionKey("process-b").SetTimerExpression("PT2H").
		SetExpressionType("duration").SetFireAt(time.Now().Add(2 * time.Hour)).
		SetIdempotencyKey("timer-b-001-key").SetTenantID(tenantB.ID).SaveX(ctx)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	const secret = "timer-contract-secret"
	router.SetupRoutes(engine, &router.RouterConfig{
		JWTSecret: secret,
		Logger:    logger,
		Client:    client,
		// timer 接线（router.go:412）不进 RouterConfig：config.Client != nil
		// 即自建 DBTimerStore + Handler，fixture 与生产同型。
	})

	return &fixture{
		engine: engine, client: client, secret: secret,
		tenantA: tenantA, tenantB: tenantB,
		userA: userA, endUser: endUser, timerA: timerA.TimerID,
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

func TestTimerHTTPContract(t *testing.T) {
	fx := newFixture(t)
	base := "/api/v1/timers"

	t.Run("list只含本租且camelCase", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, base, fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 0, decodeEnvelope(t, w).Code)
		body := w.Body.String()
		require.Contains(t, body, `"timerId"`, "字段必须是 camelCase")
		require.Contains(t, body, `"processDefinitionKey"`)
		require.NotContains(t, body, `"timer_id"`, "不得出现 snake_case 字段")
		require.Contains(t, body, "timer-a-001", "本租 timer 应可见")
		require.NotContains(t, body, "timer-b-001", "列表不得混入他租 timer")
	})

	t.Run("get跨租户与不存在同形404", func(t *testing.T) {
		// 本租可见
		w := fx.do(t, http.MethodGet, base+"/"+fx.timerA, fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		// 他租 timerID → 与不存在同形 404（TM1 掩码契约）
		w = fx.do(t, http.MethodGet, base+"/timer-b-001", fx.userA)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)

		// 不存在 timerID → 同掩码
		w = fx.do(t, http.MethodGet, base+"/timer-ghost-999", fx.userA)
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 4004, decodeEnvelope(t, w).Code)
	})

	t.Run("stats本租聚合", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, base+"/stats", fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 0, decodeEnvelope(t, w).Code)
	})

	t.Run("非法路径与权限码", func(t *testing.T) {
		// GET /timers/（缺失 id，但 gin 路由下空 id 走列表）——改用 system:read 授权探针：
		// end_user 不持 system:read（roles.go 权威词表）。
		w := fx.do(t, http.MethodGet, base, fx.endUser)
		require.Equal(t, http.StatusForbidden, w.Code, "body=%s", w.Body.String())
		require.Equal(t, 2003, decodeEnvelope(t, w).Code)
	})
}
