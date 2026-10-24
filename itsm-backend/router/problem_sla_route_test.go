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
	"itsm-backend/service"
	"itsm-backend/service/sla"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// GET /api/v1/problems/:id/sla 的真实契约（2026-10-21 GA 阻断项落地）：
// 问题创建即按 service_type="problem" 绑定 sla_states（与工单/事件同一引擎），
// 端点返回 200 + dto.ProblemSLAResponse。
//
// 核心红线（503 时代契约的延续）：bound=false 是「没配 SLA」的如实回答，
// 绝不能返回 200 + {slaStatus:"none", responseTimeUsed:0} 之类的伪数据让
// 调用方把「能力没接/没配置」读成「SLA 正常且未超时」。
// 问题不存在或跨租户时 404/4004，非数字 ID 400/1001，未认证 401。

func TestProblemSLARouteContract(t *testing.T) {
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

	// tenant B 配了 problem 类型的 SLA 定义（medium 优先级）：问题创建即绑定。
	client.SLADefinition.Create().SetName("问题 SLA").
		SetServiceType("problem").SetPriority("medium").
		SetResponseTime(30).SetResolutionTime(240).
		SetIsActive(true).SetTenantID(tenantB.ID).SaveX(ctx)

	repo := problemHandler.NewEntRepository(client)
	slaSvc := service.NewTicketSLAService(client, logger)
	slaSvc.SetSLAStore(sla.NewStore(client, sla.NewEngine(nil)))
	probSvc := problemHandler.NewService(repo, client, service.NewTicketServiceForTest(client, logger), logger)
	probSvc.SetSLAService(slaSvc)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	const secret = "problem-sla-secret"
	SetupRoutes(r, &RouterConfig{
		JWTSecret:      secret,
		Logger:         logger,
		Client:         client,
		ProblemHandler: problemHandler.NewHandler(probSvc),
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

	t.Run("未配 problem 定义时 bound=false/unbound，不伪造 SLA 数据", func(t *testing.T) {
		// tenant A 的问题（无 SLA 定义）：经 POST 创建（走绑定链路），读回应如实报未绑定。
		reqBody := `{"title":"根因未定","description":"验证 bound=false 契约","priority":"medium"}`
		token, err := middleware.GenerateAccessToken(userA.ID, userA.Username, string(userA.Role), userA.TenantID, secret, time.Hour)
		require.NoError(t, err)
		wPost := httptest.NewRecorder()
		reqPost := httptest.NewRequest(http.MethodPost, "/api/v1/problems", strings.NewReader(reqBody))
		reqPost.Header.Set("Authorization", "Bearer "+token)
		reqPost.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(wPost, reqPost)
		require.Equal(t, http.StatusOK, wPost.Code, "body=%s", wPost.Body.String())

		var created struct {
			Data struct {
				ID int `json:"id"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(wPost.Body.Bytes(), &created))
		slaPath := fmt.Sprintf("/api/v1/problems/%d/sla", created.Data.ID)

		w, env := do(t, http.MethodGet, slaPath, userA)
		assert.Equal(t, http.StatusOK, w, "body=%s", string(env.Data))
		var sla struct {
			Bound  bool   `json:"bound"`
			Status string `json:"status"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &sla))
		assert.False(t, sla.Bound, "未配定义必须如实 bound=false")
		assert.Equal(t, "unbound", sla.Status)
		assert.NotContains(t, string(env.Data), "slaStatus", "响应不得带旧伪造字段 slaStatus")
	})

	t.Run("配了 problem 定义时创建即绑定，status=active 且有截止时间", func(t *testing.T) {
		reqBody := `{"title":"绑定链路验证","description":"problem 类型 SLA 定义已配置","priority":"medium"}`
		token, err := middleware.GenerateAccessToken(stranger.ID, stranger.Username, string(stranger.Role), stranger.TenantID, secret, time.Hour)
		require.NoError(t, err)
		wPost := httptest.NewRecorder()
		reqPost := httptest.NewRequest(http.MethodPost, "/api/v1/problems", strings.NewReader(reqBody))
		reqPost.Header.Set("Authorization", "Bearer "+token)
		reqPost.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(wPost, reqPost)
		require.Equal(t, http.StatusOK, wPost.Code, "body=%s", wPost.Body.String())

		var created struct {
			Data struct {
				ID int `json:"id"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(wPost.Body.Bytes(), &created))
		slaPath := fmt.Sprintf("/api/v1/problems/%d/sla", created.Data.ID)

		w, env := do(t, http.MethodGet, slaPath, stranger)
		assert.Equal(t, http.StatusOK, w, "body=%s", string(env.Data))
		var sla struct {
			Bound              bool   `json:"bound"`
			Status             string `json:"status"`
			ResponseDeadline   string `json:"responseDeadline"`
			ResolutionDeadline string `json:"resolutionDeadline"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &sla))
		assert.True(t, sla.Bound, "有匹配定义时创建即绑定: body=%s", string(env.Data))
		// status 由 SLAStatus 实时推导：响应窗口=30 分钟预警窗，创建后立即读取
		// 必然落在预警窗内（warning），未响应且未到期也可能是 active——两者皆合法。
		assert.Contains(t, []string{"active", "warning"}, sla.Status, "body=%s", string(env.Data))
		assert.NotEmpty(t, sla.ResponseDeadline)
		assert.NotEmpty(t, sla.ResolutionDeadline)
	})

	t.Run("不存在与跨租户仍是 404/4004", func(t *testing.T) {
		w, env := do(t, http.MethodGet, "/api/v1/problems/987654/sla", userA)
		assert.Equal(t, http.StatusNotFound, w, "body=%s", string(env.Data))
		assert.Equal(t, 4004, env.Code)

		// 跨租户：stranger(B) 探 tenant A 侧 ID=1 的问题（tenant A 创建的第一个问题行），
		// 定义是否存在与隔离无关——非本租户 ID 统一 404。
		w, env = do(t, http.MethodGet, "/api/v1/problems/1/sla", stranger)
		assert.Equal(t, http.StatusNotFound, w, "跨租户不得探到他租问题: body=%s", string(env.Data))
		assert.Equal(t, 4004, env.Code)
	})

	t.Run("非数字 ID 是 400，未认证是 401", func(t *testing.T) {
		w, env := do(t, http.MethodGet, "/api/v1/problems/abc/sla", userA)
		assert.Equal(t, http.StatusBadRequest, w, "body=%s", string(env.Data))
		assert.Equal(t, 1001, env.Code)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/problems/1/sla", nil)
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
