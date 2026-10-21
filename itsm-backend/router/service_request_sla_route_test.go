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
	cmdbHandler "itsm-backend/handlers/cmdb"
	scHandler "itsm-backend/handlers/service_catalog"
	srHandler "itsm-backend/handlers/service_request"
	"itsm-backend/middleware"
	"itsm-backend/service"
	"itsm-backend/service/sla"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// GET /api/v1/service-requests/:id/sla 的真实契约（2026-10-21 统一绑定落地）：
// 服务请求创建即按 service_type="service_request" 绑定 sla_states
// （与工单/事件/问题同一引擎、同一多态表），端点返回 200 真实契约。
//
// 契约红线：bound=false 是「没配 SLA」的如实回答，不返回伪数据；
// 服务请求无 priority 字段，定义按 service_type 匹配（不带优先级）。
// 跨租户 404，未认证 401。

func TestServiceRequestSLARouteContract(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:router_sr_sla_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { client.Close() })

	logger := zaptest.NewLogger(t).Sugar()
	scRepo := scHandler.NewEntRepository(client)
	slaSvc := service.NewTicketSLAService(client, logger)
	slaSvc.SetSLAStore(sla.NewStore(client, sla.NewEngine(nil)))

	svc := srHandler.NewService(
		srHandler.NewEntRepository(client),
		scRepo,
		cmdbHandler.NewEntRepository(client),
		client,
		logger,
		nil,
	)
	svc.SetSLAService(slaSvc)
	handler := srHandler.NewHandler(svc)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	const secret = "sr-sla-secret"
	SetupRoutes(r, &RouterConfig{
		JWTSecret:             secret,
		Logger:                logger,
		Client:                client,
		ServiceRequestHandler: handler,
	})

	// tenant A：不配 SLA 定义 → 请求应如实 bound=false。
	tenantA := client.Tenant.Create().SetName("SR SLA A").SetCode("sr-sla-a").SetDomain("sr-sla-a.example.com").SetStatus("active").SaveX(ctx)
	userA := client.User.Create().SetUsername("sr-sla-a-admin").SetEmail("sr-sla-a@example.com").SetName("A").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	catA := client.ServiceCatalog.Create().SetName("VPN 开通").SetDescription("d").SetStatus("enabled").SetTenantID(tenantA.ID).SaveX(ctx)

	// tenant B：配了 service_request 类型定义（不带优先级）→ 创建即绑定。
	tenantB := client.Tenant.Create().SetName("SR SLA B").SetCode("sr-sla-b").SetDomain("sr-sla-b.example.com").SetStatus("active").SaveX(ctx)
	userB := client.User.Create().SetUsername("sr-sla-b-admin").SetEmail("sr-sla-b@example.com").SetName("B").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantB.ID).SaveX(ctx)
	catB := client.ServiceCatalog.Create().SetName("云主机开通").SetDescription("d").SetStatus("enabled").SetTenantID(tenantB.ID).SaveX(ctx)
	client.SLADefinition.Create().SetName("服务请求 SLA").
		SetServiceType("service_request").
		SetResponseTime(60).SetResolutionTime(480).
		SetIsActive(true).SetTenantID(tenantB.ID).SaveX(ctx)

	do := func(t *testing.T, method, path, body string, user *ent.User) (int, srSLAEnvelope) {
		t.Helper()
		token, err := middleware.GenerateAccessToken(user.ID, user.Username, string(user.Role), user.TenantID, secret, time.Hour)
		require.NoError(t, err)
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		var env srSLAEnvelope
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "body=%s", w.Body.String())
		return w.Code, env
	}

	createRequest := func(t *testing.T, user *ent.User, catalogID int) int {
		t.Helper()
		body := fmt.Sprintf(`{"catalogId":%d,"title":"开通验证","reason":"SLA 绑定契约测试","dataClassification":"internal","complianceAck":true,"expireAt":"%s"}`,
			catalogID, time.Now().AddDate(0, 1, 0).Format(time.RFC3339))
		code, env := do(t, http.MethodPost, "/api/v1/service-requests", body, user)
		require.Equal(t, http.StatusOK, code, "create body=%s", string(env.Data))
		var created struct {
			ID int `json:"id"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &created), "data=%s", string(env.Data))
		return created.ID
	}

	t.Run("未配 service_request 定义时 bound=false/unbound，不伪造数据", func(t *testing.T) {
		id := createRequest(t, userA, catA.ID)
		w, env := do(t, http.MethodGet, fmt.Sprintf("/api/v1/service-requests/%d/sla", id), "", userA)
		assert.Equal(t, http.StatusOK, w, "body=%s", string(env.Data))
		var slaResp struct {
			Bound     bool   `json:"bound"`
			SLAStatus string `json:"slaStatus"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &slaResp))
		assert.False(t, slaResp.Bound, "未配定义必须如实 bound=false")
		assert.Equal(t, "unbound", slaResp.SLAStatus)
	})

	t.Run("配了定义时创建即绑定，有截止时间", func(t *testing.T) {
		id := createRequest(t, userB, catB.ID)
		w, env := do(t, http.MethodGet, fmt.Sprintf("/api/v1/service-requests/%d/sla", id), "", userB)
		assert.Equal(t, http.StatusOK, w, "body=%s", string(env.Data))
		var slaResp struct {
			Bound              bool   `json:"bound"`
			SLAStatus          string `json:"slaStatus"`
			SLADefinitionID    int    `json:"slaDefinitionId"`
			ResponseDeadline   string `json:"responseDeadline"`
			ResolutionDeadline string `json:"resolutionDeadline"`
		}
		require.NoError(t, json.Unmarshal(env.Data, &slaResp))
		assert.True(t, slaResp.Bound, "有匹配定义时创建即绑定: body=%s", string(env.Data))
		assert.NotZero(t, slaResp.SLADefinitionID)
		assert.Contains(t, []string{"active", "warning"}, slaResp.SLAStatus,
			"响应窗口在 30 分钟预警语义内两者皆合法: body=%s", string(env.Data))
		assert.NotEmpty(t, slaResp.ResponseDeadline)
		assert.NotEmpty(t, slaResp.ResolutionDeadline)
	})

	t.Run("跨租户 404，未认证 401，非数字 ID 400", func(t *testing.T) {
		id := createRequest(t, userA, catA.ID)

		// service_request 域的 not-found 契约（与 Get 端点一致）：HTTP 200 + 业务码 404。
		// 跨租户探不到他租请求，响应无 data（不暴露存在性）。
		w, env := do(t, http.MethodGet, fmt.Sprintf("/api/v1/service-requests/%d/sla", id), "", userB)
		assert.Equal(t, 404, env.Code, "跨租户不得探到他租请求: body=%s", string(env.Data))
		assert.Empty(t, string(env.Data), "not-found 响应不得携带 data")

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/service-requests/%d/sla", id), nil)
		wUnauth := httptest.NewRecorder()
		r.ServeHTTP(wUnauth, req)
		assert.Equal(t, http.StatusUnauthorized, wUnauth.Code)

		w, env = do(t, http.MethodGet, "/api/v1/service-requests/abc/sla", "", userA)
		assert.Equal(t, http.StatusBadRequest, w, "body=%s", string(env.Data))
	})
}

// srSLAEnvelope 是 { code, message, data } 的解析目标。
type srSLAEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}
