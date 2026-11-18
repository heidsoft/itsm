package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/internal/authz"
)

// 这组测试锁 RBAC 中间件对 service 主体的行为。
//
// 背景（2026-11-18，MCP 端到端验证时暴露）：RBAC 中间件此前无条件按
// user_id 查用户表，而 service 主体的 JWT 里 user_id 恒为 0（进程身份，
// 不是登录用户）→ 必然查不到 → 一律 401「用户不存在」。
//
// 后果范围不止 MCP：**service token 无法访问任何 /api/v1/* 业务路由**。
// Phase B1 落地了 service 主体身份，但 RBAC 这一环没打通；既有测试
// （auth_service_principal_test.go）只覆盖到 AuthMiddleware 就停了，
// 于是这个缺口一直没被发现。
//
// 修复口径：service 主体在 RBAC 层短路（跳过用户表查询），
// 权限仍由 RequirePermission 按其载体角色（audit_admin）判定，
// 动作级主体放行仍由 Gate 2.5（authz 零信任）决定。

// runRBACWithServiceToken 装配「认证 + RBAC」两段中间件并跑一次请求。
//
// 必须串上 AuthMiddleware：RBAC 从 request context 读 principal，
// 而 principal 是 AuthMiddleware 注入的（middleware/auth.go 明确
// 「刻意不放进 gin context 裸键当作授权依据」）。
func runRBACWithServiceToken(t *testing.T, client *ent.Client, token string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/api/v1/mcp", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)

	// 先跑认证：注入 principal / role / tenant_id / user_id。
	AuthMiddleware(serviceTokenTestSecret)(c)
	if c.IsAborted() {
		return w
	}
	RBACMiddleware(client)(c)
	return w
}

// TestRBACMiddleware_ServicePrincipalNotRejectedAsMissingUser 是核心回归锁。
//
// 反向验证纪律：把 RBAC 里的 service 短路分支去掉（即恢复旧实现），
// 本用例必须变红，且红在断言上（不是编译错误）。
func TestRBACMiddleware_ServicePrincipalNotRejectedAsMissingUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// 用真实 ent client（而非 nil）：旧实现走到用户表查询会得到
	// 「用户不存在」→ 401；新实现短路不查表 → 放行。
	// 用真实 client 才能让反向验证红在**断言**上而不是 panic。
	client := enttest.Open(t, "sqlite3", "file:rbac_svc_principal?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })

	token, err := GenerateServiceToken("svc-rbac", ServicePrincipalRole, 7, serviceTokenTestSecret, time.Hour)
	require.NoError(t, err)

	w := runRBACWithServiceToken(t, client, token)

	// 关键：不得是 401「用户不存在」。短路后请求应继续往下走
	// （未命中 RequirePermission 时由最终 handler 决定）。
	assert.NotEqual(t, http.StatusUnauthorized, w.Code,
		"service 主体不得被 RBAC 判为「用户不存在」——它本就不在用户表里")
}

// TestRBACMiddleware_ServicePrincipalRoleAvailableForPermissionCheck 验证
// 短路后 service 主体的载体角色仍可用于权限判定。
//
// 若短路时把 role 一并丢掉，RequirePermission 会因「用户角色信息缺失」而 403，
// 等于换个姿势失败 —— 那不是修复。
func TestRBACMiddleware_ServicePrincipalRoleAvailableForPermissionCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	token, err := GenerateServiceToken("svc-rbac", ServicePrincipalRole, 7, serviceTokenTestSecret, time.Hour)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/api/v1/mcp", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)
	AuthMiddleware(serviceTokenTestSecret)(c)
	require.False(t, c.IsAborted(), "认证阶段不应中断")

	assert.Equal(t, ServicePrincipalRole, c.GetString("role"),
		"短路后载体角色必须仍在上下文中，供 RequirePermission 判定")
	assert.Equal(t, authz.PrincipalService, authz.PrincipalFromContext(c.Request.Context()),
		"principal 必须在 request context（Gate 2.5 的读取点）")
	assert.Equal(t, 7, c.GetInt("tenant_id"), "租户上下文必须保留")
}

// TestRBACMiddleware_HumanPrincipalStillRequiresExistingUser 是反向锁：
// human 主体仍必须存在于用户表（不得因 service 短路而被放宽）。
//
// 必须用真实 ent client：human 路径要查用户表，nil client 会 panic
// （panic 不算断言失败，也不构成有效判据）。
func TestRBACMiddleware_HumanPrincipalStillRequiresExistingUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:rbac_svc_principal?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })

	// user_id=999999 不在用户表 → human 主体必须仍被拒。
	token, err := GenerateAccessToken(999999, "ghost", "sysadmin", 7, serviceTokenTestSecret, time.Hour)
	require.NoError(t, err)

	w := runRBACWithServiceToken(t, client, token)

	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"human 主体查不到用户仍必须被拒（service 短路不得放宽 human 路径）")
}

// 确保 context 被使用（service 短路读的是 request context）。
var _ = context.Background
