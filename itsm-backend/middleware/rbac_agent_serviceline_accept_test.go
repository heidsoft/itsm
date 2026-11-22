package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"itsm-backend/domain/role"
)

// 回归：服务台坐席（agent）在 DBOnly（企业级默认模式）下必须能通过
// RequirePermission("ticket","assign") 命中——它是服务台受理
// POST /tickets/workflow/accept 与转派 POST /tickets/:id/assign 的防线。
// 参考：permission_cache_poison_test.go 的 enttest 真实 DB 验证风格。
func TestRequirePermission_AgentAcceptTransfer_DBOnlySeeded(t *testing.T) {
	restoreMode := PermissionConfig.Mode
	PermissionConfig.Mode = PermissionConfigModeDBOnly
	t.Cleanup(func() {
		PermissionConfig.Mode = restoreMode
		InvalidateAllPermissionCaches()
	})
	InvalidateAllPermissionCaches()

	ctx := context.Background()
	client, _ := openSharedCacheSQLite(t, "agent_accept_rbac")

	tnt := client.Tenant.Create().
		SetName("ServicelineTenant").SetCode("stl").SetDomain("stl.test").SetStatus("active").
		SaveX(ctx)
	ag := client.Role.Create().
		SetCode(role.Agent).SetName("服务台坐席").SetTenantID(tnt.ID).SaveX(ctx)

	// 模拟 seedRolePermissions 播种：agent 拥有 ticket:read/write/assign 等内置码。
	seeded := []struct{ resource, action string }{
		{"ticket", "read"}, {"ticket", "write"}, {"ticket", "assign"},
	}
	for _, c := range seeded {
		p := client.Permission.Create().
			SetCode(c.resource + ":" + c.action).
			SetName("perm-" + c.resource + "-" + c.action).
			SetResource(c.resource).SetAction(c.action).
			SetTenantID(tnt.ID).SaveX(ctx)
		client.RolePermission.Create().
			SetRoleID(ag.ID).SetPermissionID(p.ID).SetTenantID(tnt.ID).SaveX(ctx)
	}

	// fail-closed 基线：空角色集拒绝。
	require.False(t, AuthorizeResource(ctx, client, nil, "ticket", "assign", tnt.ID))

	// 真实路由链路：认证中间件注入 agent + RequirePermission 防线 → 200。
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("role", role.Agent)
		c.Set("roles", []string{role.Agent})
		c.Set("tenant_id", tnt.ID)
		c.Set("client", client)
		c.Next()
	})
	router.POST("/api/v1/tickets/workflow/accept",
		RequirePermission("ticket", "assign"),
		func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "ok"}) })

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tickets/workflow/accept", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "agent 必须通过 /tickets/workflow/accept 防线，got: %s", w.Body.String())

	// 转派（/tickets/:id/assign）复用同一 RequirePermission，同一判定路径。
	router2 := gin.New()
	router2.Use(func(c *gin.Context) {
		c.Set("role", role.Agent)
		c.Set("roles", []string{role.Agent})
		c.Set("tenant_id", tnt.ID)
		c.Set("client", client)
		c.Next()
	})
	router2.POST("/api/v1/tickets/7/assign",
		RequirePermission("ticket", "assign"),
		func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "ok"}) })
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/tickets/7/assign", nil)
	w2 := httptest.NewRecorder()
	router2.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code, "agent 必须通过 /tickets/:id/assign 防线，got: %s", w2.Body.String())
}
