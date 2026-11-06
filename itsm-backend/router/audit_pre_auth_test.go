package router

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"
)

// L3：审计必须排在鉴权链之前。
//
// 旧链序 Auth → RBAC → CSRF → Tenant → Audit，前三道与 TenantMiddleware
// 都会 Abort：令牌失效/爆破 401、RBAC 403、跨租户试探在到达 Audit 之前
// 就结束了，audit_logs 里一条都没有。这里从真实 Router 入口证明：
// 无凭据请求（连 shouldAuditRequest 都判 false 的非敏感 GET）仍会留痕。

func newAuditOrderRouter(t *testing.T) (*gin.Engine, *ent.Client) {
	t.Helper()
	client := enttest.Open(t, "sqlite3",
		fmt.Sprintf("file:router_audit_order_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })
	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetupRoutes(r, &RouterConfig{
		JWTSecret: "audit-order-secret",
		Logger:    zaptest.NewLogger(t).Sugar(),
		Client:    client,
	})
	return r, client
}

func TestAuditMiddleware_RecordsRejectedRequestsBeforeAuth(t *testing.T) {
	r, client := newAuditOrderRouter(t)
	ctx := context.Background()

	// /api/v1/capabilities 挂在 auth 组内、且路径不含任何敏感资源关键词，
	// 因此 shouldAuditRequest(GET) == false：这条用例同时锁住「4xx 补记」。
	const attempts = 3
	for i := 0; i < attempts; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/capabilities", nil)
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusUnauthorized, w.Code,
			"必须是鉴权拒绝，否则测的不是 Abort 路径：body=%s", w.Body.String())
	}

	logs, err := client.AuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, logs, attempts,
		"每一笔 401 都要独立留痕；0 条说明审计仍挂在鉴权链之后")
	for _, entry := range logs {
		assert.Equal(t, http.StatusUnauthorized, entry.StatusCode)
		assert.Equal(t, "/api/v1/capabilities", entry.Path)
		assert.Equal(t, http.MethodGet, entry.Method)
		// 未认证请求没有可靠的操作者/租户上下文，留 0 是可预期的，
		// 但 IP/method/path/status 必须齐全，才能用于爆破溯源。
		assert.NotEmpty(t, entry.IP)
	}
}

// 正控制：审计上移不能变成「所有请求都落一行」。已认证的成功请求若是非敏感
// GET，仍应零审计；单笔请求最多一条（防止 auth 组与 tenant 子组重复挂载）。
func TestAuditMiddleware_NoDuplicateOnAuthenticatedRequest(t *testing.T) {
	r, client := newAuditOrderRouter(t)
	ctx := context.Background()

	tenant := client.Tenant.Create().SetName("审计顺序租户").SetCode("audit-order").
		SetDomain("audit-order.example.com").SetStatus("active").SaveX(ctx)
	user := client.User.Create().SetUsername("audit-order-user").
		SetEmail("audit-order@example.com").SetName("Audit").SetPasswordHash("hash").
		SetRole("end_user").SetActive(true).SetTenantID(tenant.ID).SaveX(ctx)
	token, err := middleware.GenerateAccessToken(user.ID, user.Username, string(user.Role), tenant.ID,
		"audit-order-secret", time.Hour)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/capabilities", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	require.NotEqual(t, http.StatusUnauthorized, w.Code, "token 必须被接受，否则测不到完整鉴权链")

	logs, err := client.AuditLog.Query().All(ctx)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(logs), 1,
		"单笔请求最多一条审计；>1 说明 auth 组与子组重复挂载")
	if w.Code < 400 {
		assert.Empty(t, logs, "成功的非敏感 GET 不应因 4xx 补记逻辑被写入审计")
	}
}
