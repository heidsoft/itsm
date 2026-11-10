package router

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/ent/enttest"
	accountHandler "itsm-backend/handlers/account"
	commonHandler "itsm-backend/handlers/common"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 身份面写操作的审计留痕回归锁。
//
// 缺陷（2026-11-05 白盒审查）：审计中间件只挂在 tenant 分组，而
// /api/v1/account/*（自助改密）与 /api/v1/auth/{logout,switch-tenant}
// 挂在 engine 根层，绕过了审计。结果是「改密码」这类最该留痕的操作
// 在 audit_logs 里一条都没有，事后无法回答「谁在何时从哪里改了密码」。
//
// 这条用例从真实路由入口证明：改密请求会落审计，且明文密码不会进审计库。

func TestAccountChangePassword_IsAuditedAndMasked(t *testing.T) {
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:router_account_audit_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()
	tenant := client.Tenant.Create().SetName("审计租户").SetCode("audit-t").SetDomain("audit-t.example.com").SetStatus("active").SaveX(ctx)
	user := client.User.Create().
		SetUsername("audit-user").SetEmail("audit-user@example.com").SetName("Audit").
		SetPasswordHash("hash").SetRole("end_user").SetActive(true).SetTenantID(tenant.ID).SaveX(ctx)

	const secret = "account-audit-secret"
	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetupRoutes(r, &RouterConfig{
		JWTSecret: secret,
		Logger:    logger,
		Client:    client,
		// userService 传 nil：本用例只验「路由→审计」这条链，
		// handler 会在鉴权与参数校验之后返回 503，不影响留痕判定。
		AccountHandler: accountHandler.NewHandler(nil),
	})

	token, err := middleware.GenerateAccessToken(user.ID, user.Username, string(user.Role), tenant.ID, secret, time.Hour)
	require.NoError(t, err)

	body := `{"oldPassword":"OldPass2026!","newPassword":"NewPass2026!"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/account/change-password", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.NotEqual(t, http.StatusNotFound, w.Code, "路由必须存在：body=%s", w.Body.String())
	require.NotEqual(t, http.StatusUnauthorized, w.Code, "鉴权应通过：body=%s", w.Body.String())

	logs, err := client.AuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, logs, 1, "改密必须落一条审计，实际 0 条说明中间件没挂上")

	entry := logs[0]
	require.Equal(t, "/api/v1/account/change-password", entry.Path)
	require.Equal(t, http.MethodPut, entry.Method)
	require.Equal(t, tenant.ID, entry.TenantID, "审计必须带租户上下文，否则事后无法按租户追溯")
	require.Equal(t, user.ID, entry.UserID, "审计必须带操作者，否则留痕无意义")

	storedBody := ""
	if entry.RequestBody != nil {
		storedBody = *entry.RequestBody
	}
	require.NotContains(t, storedBody, "NewPass2026!", "明文新密码绝不能进审计库")
	require.NotContains(t, storedBody, "OldPass2026!", "明文旧密码绝不能进审计库")
	require.Contains(t, storedBody, "***")
}

// TestAuthLogout_IsAudited 锁住 /api/v1/auth 分组的审计挂载。
//
// logout 与 switch-tenant 挂在 engine 根层的 /api/v1/auth 分组，不在 tenant
// 分组内，tenant 层审计覆盖不到。这里用真实请求证明登出会落审计
// （登出会吊销 refresh token，是会话劫持排查的关键线索）。
func TestAuthLogout_IsAudited(t *testing.T) {
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:router_logout_audit_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))
	t.Cleanup(func() { client.Close() })

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()
	tenant := client.Tenant.Create().SetName("登出租户").SetCode("logout-t").SetDomain("logout-t.example.com").SetStatus("active").SaveX(ctx)
	user := client.User.Create().
		SetUsername("logout-user").SetEmail("logout-user@example.com").SetName("Logout").
		SetPasswordHash("hash").SetRole("end_user").SetActive(true).SetTenantID(tenant.ID).SaveX(ctx)

	const secret = "logout-audit-secret"
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Logout 只需要 jwtSecret 用于校验并吊销 token；repo 在登出路径上不参与。
	SetupCommonSystemRoutes(r, r.Group("/api/v1"), &RouterConfig{
		JWTSecret:     secret,
		Logger:        logger,
		Client:        client,
		CommonHandler: commonHandler.NewHandler(commonHandler.NewService(nil, secret, logger, client)),
	})

	token, err := middleware.GenerateAccessToken(user.ID, user.Username, string(user.Role), tenant.ID, secret, time.Hour)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(httptest.NewRecorder(), req)

	logs, err := client.AuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, logs, 1, "登出必须落审计，0 条说明 /api/v1/auth 分组没挂审计中间件")
	require.Equal(t, "/api/v1/auth/logout", logs[0].Path)
	require.Equal(t, http.MethodPost, logs[0].Method)
}
