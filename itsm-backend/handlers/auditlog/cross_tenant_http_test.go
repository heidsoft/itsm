// Package auditlog 的跨租户 HTTP 回归测试。
//
// 审计日志读取（GET /api/v1/audit-logs）承载全平台的操作证据链，是内容
// 泄漏最敏感的读取面之一。隔离保证必须从真实生产入口
// （router.SetupRoutes 布线 + JWT 认证 + RBAC RequirePermission + handler + service + ent）
// 发起验证，参照 router/ticket_attachment_route_test.go 的搭台方式。
//
// 已钉住的契约：
//   - 响应是平台五键 {items,total,page,pageSize,totalPages}；
//   - 用户只能看到本租户的审计行：行数、内容、tenantId 全部按租户收敛；
//   - 跨租户的程序化标记（路径/动作）不得出现在任何响应里。
package auditlog_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	auditHandler "itsm-backend/handlers/auditlog"
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
	userA    *ent.User
	stranger *ent.User
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:audit_cross_tenant_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()
	tenantA := client.Tenant.Create().SetName("Audit A").SetCode("audit-a").SetDomain("audit-a.example.com").SetStatus("active").SaveX(ctx)
	tenantB := client.Tenant.Create().SetName("Audit B").SetCode("audit-b").SetDomain("audit-b.example.com").SetStatus("active").SaveX(ctx)

	userA := client.User.Create().SetUsername("audit-a-owner").SetEmail("audit-a@example.com").SetName("Owner A").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantA.ID).SaveX(ctx)
	stranger := client.User.Create().SetUsername("audit-b-stranger").SetEmail("audit-b@example.com").SetName("Stranger B").
		SetPasswordHash("hash").SetRole("super_admin").SetActive(true).SetTenantID(tenantB.ID).SaveX(ctx)

	// 租户 A 的审计证据行（程序化标记后续用于泄漏探针）。
	client.AuditLog.Create().
		SetPath("/api/v1/tickets/1001").
		SetMethod("POST").
		SetResource("ticket").
		SetAction("create_ticket_a_marker").
		SetStatusCode(200).
		SetRequestBody("审计内容租户A机密").
		SetTenantID(tenantA.ID).
		SetUserID(userA.ID).
		SaveX(ctx)
	// 租户 B 的审计证据行。
	client.AuditLog.Create().
		SetPath("/api/v1/tickets/9001").
		SetMethod("DELETE").
		SetResource("ticket").
		SetAction("delete_ticket_b_marker").
		SetStatusCode(200).
		SetRequestBody("审计内容租户B机密").
		SetTenantID(tenantB.ID).
		SetUserID(stranger.ID).
		SaveX(ctx)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	const secret = "audit-cross-tenant-secret"
	// 装配修复后的语义（P2）：/audit-logs 只依赖 AuditLogHandler 自身——
	// 此前注册嵌在 `CommonHandler != nil` 块内，只装配 AuditLogHandler 时
	// 路由会静默 404。本 fixture 刻意不装配 CommonHandler 来锁定该契约。
	router.SetupRoutes(engine, &router.RouterConfig{
		JWTSecret:       secret,
		Logger:          logger,
		Client:          client,
		AuditLogHandler: auditHandler.NewHandler(service.NewAuditLogService(client, logger), logger),
	})

	return &fixture{
		engine: engine, client: client, secret: secret,
		tenantA: tenantA, tenantB: tenantB, userA: userA, stranger: stranger,
	}
}

func (fx *fixture) do(t *testing.T, method, path string, user *ent.User) *httptest.ResponseRecorder {
	t.Helper()
	token, err := middleware.GenerateAccessToken(user.ID, user.Username, string(user.Role), user.TenantID, fx.secret, time.Hour)
	require.NoError(t, err)
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	fx.engine.ServeHTTP(w, req)
	return w
}

func TestListAuditLogsCrossTenant(t *testing.T) {
	fx := newFixture(t)

	t.Run("同租户owner是正控制且遵循五键分页契约", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, "/api/v1/audit-logs?page=1&pageSize=20", fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"code":0`)

		var resp struct {
			Code int `json:"code"`
			Data struct {
				Items []struct {
					TenantID int    `json:"tenantId"`
					Action   string `json:"action"`
				} `json:"items"`
				Total      int `json:"total"`
				Page       int `json:"page"`
				PageSize   int `json:"pageSize"`
				TotalPages int `json:"totalPages"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Equal(t, 0, resp.Code)
		require.Equal(t, 1, len(resp.Data.Items))
		require.Equal(t, 1, resp.Data.Total)
		require.Equal(t, fx.tenantA.ID, resp.Data.Items[0].TenantID)
		require.Equal(t, "create_ticket_a_marker", resp.Data.Items[0].Action)
		require.Contains(t, w.Body.String(), "审计内容租户A机密")
	})

	t.Run("跨租户读取fail closed且零泄漏", func(t *testing.T) {
		w := fx.do(t, http.MethodGet, "/api/v1/audit-logs?page=1&pageSize=20", fx.stranger)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		assert.NotContains(t, w.Body.String(), "create_ticket_a_marker")
		assert.NotContains(t, w.Body.String(), "审计内容租户A机密")
		assert.NotContains(t, w.Body.String(), "/api/v1/tickets/1001")

		var resp struct {
			Code int `json:"code"`
			Data struct {
				Items []struct {
					TenantID int    `json:"tenantId"`
					Action   string `json:"action"`
				} `json:"items"`
				Total int `json:"total"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Equal(t, 0, resp.Code)
		require.Equal(t, 1, resp.Data.Total)
		for _, it := range resp.Data.Items {
			require.Equal(t, fx.tenantB.ID, it.TenantID, "可见行必须全部属于租户 B")
			require.Equal(t, "delete_ticket_b_marker", it.Action)
		}
	})

	t.Run("过滤不命中的用户依然只能看到自己租户的行", func(t *testing.T) {
		// 用一个不存在的 action 过滤，确认 total=0 时也不泄漏任何对端内容。
		w := fx.do(t, http.MethodGet, "/api/v1/audit-logs?action=no_such_action&page=1&pageSize=20", fx.userA)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		assert.NotContains(t, w.Body.String(), "delete_ticket_b_marker")
		assert.NotContains(t, w.Body.String(), "/api/v1/tickets/9001")
		var resp struct {
			Data struct {
				Total int `json:"total"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Equal(t, 0, resp.Data.Total)
	})
}
