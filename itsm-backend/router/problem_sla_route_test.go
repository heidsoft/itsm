package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	problemHandler "itsm-backend/handlers/problem"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 回归（2026-10-02 边缘功能收口 E2-2）：GET /api/v1/problems/:id/sla 此前恒返回
// 200 + {slaStatus:"none", responseTimeUsed:0, ...}。schema 里本就没有问题 SLA 的
// 截止时间字段，SLA 引擎也没有任何为 problem 写 sla_state 的路径，所以这个空成功
// 让调用方把「能力没接」读成「该问题的 SLA 正常且未超时」，与「未配置」不可区分。
// 现在与问题评论同样显式 unready（503/5003），且不存在/跨租户仍先 404。

func TestProblemSLARouteIsExplicitlyUnready(t *testing.T) {
	dsn := fmt.Sprintf("file:router_problem_sla_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenantA := client.Tenant.Create().SetName("Prob A").SetCode("prob-a").SetDomain("prob-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("Prob B").SetCode("prob-b").SetDomain("prob-b.example.com").SetStatus("active").SaveX(ctx)
	userA := client.User.Create().SetUsername("prob-a-admin").SetEmail("prob-a@example.com").SetName("A").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	stranger := client.User.Create().SetUsername("prob-b-admin").SetEmail("prob-b@example.com").SetName("B").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantB.ID).SaveX(ctx)
	probA := client.Problem.Create().SetTitle("根因未定").SetDescription("验证 SLA 显式 unready").
		SetPriority("medium").SetStatus("open").SetCreatedBy(userA.ID).SetTenantID(tenantA.ID).SaveX(ctx)

	repo := problemHandler.NewEntRepository(client)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	const secret = "problem-sla-secret"
	SetupRoutes(r, &RouterConfig{
		JWTSecret:      secret,
		Logger:         logger,
		Client:         client,
		ProblemHandler: problemHandler.NewHandler(problemHandler.NewService(repo, logger)),
	})

	do := func(t *testing.T, method, path string, user *ent.User) (int, problemEnvelope) {
		t.Helper()
		token, err := middleware.GenerateAccessToken(user.ID, user.Username, string(user.Role), user.TenantID, secret, time.Hour)
		require.NoError(t, err)

		req := httptest.NewRequest(method, path, strings.NewReader(""))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		var env problemEnvelope
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
		return w.Code, env
	}

	slaPath := fmt.Sprintf("/api/v1/problems/%d/sla", probA.ID)

	t.Run("已存在的问题返回 503/5003，不再伪造 SLA 数据", func(t *testing.T) {
		w, env := do(t, http.MethodGet, slaPath, userA)
		assert.Equal(t, http.StatusServiceUnavailable, w, "unready 不得伪装成 200 空成功: body=%s", string(env.Data))
		assert.Equal(t, 5003, env.Code)
		assert.Contains(t, env.Message, "尚未接入")
		assert.NotContains(t, string(env.Data), "slaStatus", "响应不得再带伪造的 slaStatus/零值倒计时")
		assert.NotContains(t, string(env.Data), "resolutionBreached")
	})

	t.Run("不存在与跨租户仍是 404/4004，不被 unready 吞掉", func(t *testing.T) {
		w, env := do(t, http.MethodGet, "/api/v1/problems/987654/sla", userA)
		assert.Equal(t, http.StatusNotFound, w, "body=%s", string(env.Data))
		assert.Equal(t, 4004, env.Code)

		w, env = do(t, http.MethodGet, slaPath, stranger)
		assert.Equal(t, http.StatusNotFound, w, "跨租户不得探到 tenant A 的问题: body=%s", string(env.Data))
		assert.Equal(t, 4004, env.Code)
	})

	t.Run("非数字 ID 是 400，未认证是 401", func(t *testing.T) {
		w, env := do(t, http.MethodGet, "/api/v1/problems/abc/sla", userA)
		assert.Equal(t, http.StatusBadRequest, w, "body=%s", string(env.Data))
		assert.Equal(t, 1001, env.Code)

		req := httptest.NewRequest(http.MethodGet, slaPath, nil)
		wUnauth := httptest.NewRecorder()
		r.ServeHTTP(wUnauth, req)
		assert.Equal(t, http.StatusUnauthorized, wUnauth.Code, "body=%s", wUnauth.Body.String())
	})
}

// problemEnvelope 是 { code, message, data } 的解析目标。
type problemEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}
