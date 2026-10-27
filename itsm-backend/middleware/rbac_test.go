package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRBACMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("No User ID in Context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/tickets", nil)

		RBACMiddleware(nil)(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "用户未认证")
	})

	t.Run("No Tenant ID in Context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/tickets", nil)
		c.Set("user_id", 1)

		RBACMiddleware(nil)(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "租户信息缺失")
	})

	// Note: "No Role in Context" test requires a valid client to query the database
	// The RBACMiddleware will attempt to fetch user from DB which panics with nil client
	// This is expected behavior - in production, client should never be nil
}

func TestRequirePermissionForRBAC(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Missing Role Returns Error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/tickets", nil)
		// No role set

		RequirePermission("ticket", "read")(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "用户角色信息缺失")
	})

	t.Run("Missing Tenant ID Returns Error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/tickets", nil)
		c.Set("role", "admin")
		// No tenant_id

		RequirePermission("ticket", "read")(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "租户信息缺失")
	})
}

func TestRequireRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Missing Role Returns Error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/tickets", nil)

		RequireRole("admin", "manager")(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "缺少角色信息")
	})

	t.Run("Role Not Allowed Returns Error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/tickets", nil)
		c.Set("role", "end_user")

		RequireRole("admin", "manager")(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "无权限执行该操作")
	})

	t.Run("Role Allowed Succeeds", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/tickets", nil)
		c.Set("role", "admin")

		RequireRole("admin", "manager")(c)

		// Should not abort
		assert.False(t, c.IsAborted())
	})

	t.Run("Role Matching Is Case Insensitive", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/tickets", nil)
		c.Set("role", "ADMIN")

		RequireRole("admin", "manager")(c)

		// Should not abort
		assert.False(t, c.IsAborted())
	})
}

func TestHasResourcePermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	original := PermissionConfig.Mode
	PermissionConfig.Mode = PermissionConfigModeHardcodeOnly
	t.Cleanup(func() { PermissionConfig.Mode = original })

	t.Run("Super Admin Has All Permissions", func(t *testing.T) {
		result := hasResourcePermission(context.Background(), nil, "super_admin", "any_resource", "any_action", 1)
		assert.True(t, result)
	})

	t.Run("Admin Has Ticket Write Permission", func(t *testing.T) {
		result := hasResourcePermission(context.Background(), nil, "admin", "ticket", "write", 1)
		assert.True(t, result)
	})

	t.Run("End User Has Ticket Read Permission", func(t *testing.T) {
		result := hasResourcePermission(context.Background(), nil, "end_user", "ticket", "read", 1)
		assert.True(t, result)
	})

	t.Run("End User Does Not Have Ticket Delete Permission", func(t *testing.T) {
		result := hasResourcePermission(context.Background(), nil, "end_user", "ticket", "delete", 1)
		assert.False(t, result)
	})

	t.Run("Agent Has Dashboard Read Permission", func(t *testing.T) {
		result := hasResourcePermission(context.Background(), nil, "agent", "dashboard", "read", 1)
		assert.True(t, result)
	})

	t.Run("Technician Has CMDB Read Permission", func(t *testing.T) {
		result := hasResourcePermission(context.Background(), nil, "technician", "cmdb", "read", 1)
		assert.True(t, result)
	})

	t.Run("Manager Has User Read Permission", func(t *testing.T) {
		result := hasResourcePermission(context.Background(), nil, "manager", "user", "read", 1)
		assert.True(t, result)
	})

	t.Run("Unknown Role Has No Permissions", func(t *testing.T) {
		result := hasResourcePermission(context.Background(), nil, "unknown_role", "ticket", "read", 1)
		assert.False(t, result)
	})

	t.Run("Wildcard Permission", func(t *testing.T) {
		// Super admin has wildcard "*" permission
		result := hasResourcePermission(context.Background(), nil, "super_admin", "ticket", "delete", 1)
		assert.True(t, result)

		result = hasResourcePermission(context.Background(), nil, "super_admin", "anything", "anything", 1)
		assert.True(t, result)
	})
}

func TestMatchPath(t *testing.T) {
	t.Run("Exact Match", func(t *testing.T) {
		assert.True(t, matchPath("/api/v1/tickets", "/api/v1/tickets"))
		assert.False(t, matchPath("/api/v1/tickets", "/api/v1/users"))
	})

	t.Run("Wildcard Match", func(t *testing.T) {
		assert.True(t, matchPath("/api/v1/tickets/*", "/api/v1/tickets/123"))
		assert.True(t, matchPath("/api/v1/tickets/*", "/api/v1/tickets/abc/edit"))
		assert.True(t, matchPath("/api/v1/tickets/*/assign", "/api/v1/tickets/123/assign"))
		assert.False(t, matchPath("/api/v1/tickets/*/assign", "/api/v1/tickets/123/close"))
		assert.False(t, matchPath("/api/v1/tickets/*", "/api/v1/users/123"))
	})

	t.Run("No Match", func(t *testing.T) {
		assert.False(t, matchPath("/api/v1/tickets", "/api/v1/users"))
		assert.False(t, matchPath("/api/v1/tickets/*", "/api/v1/tickets"))
	})
}

func TestGetPermissionFromPath(t *testing.T) {
	t.Run("GET Tickets Returns Read Permission", func(t *testing.T) {
		perm := getPermissionFromPath("GET", "/api/v1/tickets")
		assert.NotNil(t, perm)
		assert.Equal(t, "ticket", perm.Resource)
		assert.Equal(t, "read", perm.Action)
	})

	t.Run("POST Tickets Returns Write Permission", func(t *testing.T) {
		perm := getPermissionFromPath("POST", "/api/v1/tickets")
		assert.NotNil(t, perm)
		assert.Equal(t, "ticket", perm.Resource)
		// 预检以路由声明为权威：POST /tickets 声明 ticket:create，
		// 种子集里 ticket:write 蕴含 ticket:create，能力不缩水
		assert.Equal(t, "create", perm.Action)
	})

	t.Run("Assign Ticket Returns Assign Permission", func(t *testing.T) {
		perm := getPermissionFromPath("POST", "/api/v1/tickets/123/assign")
		assert.NotNil(t, perm)
		assert.Equal(t, "ticket", perm.Resource)
		assert.Equal(t, "assign", perm.Action)
	})

	t.Run("DELETE Tickets Returns Delete Permission", func(t *testing.T) {
		perm := getPermissionFromPath("DELETE", "/api/v1/tickets/123")
		assert.NotNil(t, perm)
		assert.Equal(t, "ticket", perm.Resource)
		assert.Equal(t, "delete", perm.Action)
	})

	t.Run("Unknown Path Returns Nil", func(t *testing.T) {
		perm := getPermissionFromPath("GET", "/api/v1/unknown/path")
		assert.Nil(t, perm)
	})
}

// TestPatchMethodUsesUpdateContract 要求每条**已注册**的 PATCH 路由在 ResourceActionMap
// 中有显式条目：漏登记时 SmartCheckPermission L3 会把复数路径段 "tickets" 推断为资源名，
// 导致 ticket:write/update 权限真实存在的非通配角色（l1/l2/manager）被 RBACMiddleware 误拒 403。
//
// 用例从注册路由派生而不是手写假设路径。此前这里硬编码了 PATCH /incidents|/problems|/changes
// 三条——这三条路由在本仓库根本不存在（更新一律走 PUT），却靠 precheckFallbackPolicies 里
// 一整块从未命中的 PATCH 族维持绿灯；那 22 条死条目随后又成了「未来路由的埋雷」。
// 现在改成扫描真实路由：有 PATCH 路由就必须有映射，没有路由就不需要映射。
func TestPatchMethodUsesUpdateContract(t *testing.T) {
	// 基线：唯一一条带动作语义的 PATCH 路由必须解析为 update（而非 write）。
	perm := getPermissionFromPath("PATCH", "/api/v1/tickets/123")
	require.NotNil(t, perm, "PATCH /api/v1/tickets/123 必须有显式 RBAC 映射，不得回退到 URL 推断")
	assert.Equal(t, "ticket", perm.Resource)
	assert.Equal(t, "update", perm.Action)

	routes, err := ScanAllRoutes()
	require.NoError(t, err)
	patched := 0
	for _, r := range routes {
		if r.Method != "PATCH" {
			continue
		}
		patched++
		assert.NotNil(t, getPermissionFromPath(r.Method, r.FullPath),
			"PATCH %s 已注册但预检无映射——L3 URL 推断会把复数段当资源名，非通配角色会被误拒", r.FullPath)
	}
	require.NotZero(t, patched, "扫描结果里没有 PATCH 路由——扫描器失效时本用例会假绿")
}

// TestReleaseEndpointsHaveExplicitRBACMapping 要求 ResourceActionMap 收录 releases：
// 显式映射落空时 SmartCheckPermission L3 会把
// URL 推导取复数路径段 "releases" 当资源名，而词表是单数 "release"，永不匹配；
// L4 硬编码在 prod DBOnly（fail-closed）下禁用——三者叠加导致 release 域全部端点对
// super_admin/sysadmin（通配符）以外的一切角色不可达，即便 DB 已授予 release:write。
func TestReleaseEndpointsHaveExplicitRBACMapping(t *testing.T) {
	cases := []struct {
		method   string
		path     string
		resource string
		action   string
	}{
		{"GET", "/api/v1/releases", "release", "read"},
		{"GET", "/api/v1/releases/1", "release", "read"},
		{"GET", "/api/v1/releases/stats", "release", "read"},
		{"POST", "/api/v1/releases", "release", "write"},
		{"PUT", "/api/v1/releases/1", "release", "write"},
		{"PUT", "/api/v1/releases/1/status", "release", "write"},
		{"POST", "/api/v1/releases/1/approve", "release", "approve"},
		{"POST", "/api/v1/releases/1/reject", "release", "approve"},
		{"POST", "/api/v1/releases/1/rollback", "release", "rollback"},
		{"DELETE", "/api/v1/releases/1", "release", "delete"},
	}
	for _, tc := range cases {
		perm := getPermissionFromPath(tc.method, tc.path)
		require.NotNil(t, perm, "%s %s 必须有显式 RBAC 映射", tc.method, tc.path)
		assert.Equal(t, tc.resource, perm.Resource, "%s %s", tc.method, tc.path)
		assert.Equal(t, tc.action, perm.Action, "%s %s", tc.method, tc.path)
	}
}

func TestCheckPermissionMatch_ResourceAdminIncludesActions(t *testing.T) {
	permissions := []Permission{{Resource: "ticket", Action: "admin"}}

	assert.True(t, checkPermissionMatch(permissions, "ticket", "assign"))
	assert.False(t, checkPermissionMatch(permissions, "incident", "assign"))
}

func TestDBOnlyPermissionModeDoesNotUseHardcodedFallback(t *testing.T) {
	original := PermissionConfig.Mode
	PermissionConfig.Mode = PermissionConfigModeDBOnly
	t.Cleanup(func() { PermissionConfig.Mode = original })

	permissions := loadPermissionsByMode(context.Background(), nil, "admin", 1)
	assert.Empty(t, permissions)
	assert.False(t, checkPermissionMatch(permissions, "ticket", "read"))
}

// TestHasPermission_WSTicketIdentityOnly 锁定 WS 票据颁发的身份自省面豁免：
// DBOnly fail-closed 下若漏掉该豁免，POST /ws/ticket 会对全部非 super_admin 角色
// 返回 403，前端实时通知整体失效（v1.6.18 角色场景测试发现的回归面）。
func TestHasPermission_WSTicketIdentityOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	original := PermissionConfig.Mode
	PermissionConfig.Mode = PermissionConfigModeDBOnly
	t.Cleanup(func() { PermissionConfig.Mode = original })

	t.Run("Authenticated Role Can Request WS Ticket", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("POST", "/api/v1/ws/ticket", nil)

		assert.True(t, hasPermission(nil, "end_user", "POST", "/api/v1/ws/ticket", 3, 1, c))
		assert.True(t, hasPermission(nil, "it_admin", "POST", "/api/v1/ws/ticket", 13, 1, c))
	})

	t.Run("Method And Path Variants Stay Fail-Closed", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/ws/ticket", nil)

		assert.False(t, hasPermission(nil, "end_user", "GET", "/api/v1/ws/ticket", 3, 1, c))

		c.Request, _ = http.NewRequest("POST", "/api/v1/ws/tickets", nil)
		assert.False(t, hasPermission(nil, "end_user", "POST", "/api/v1/ws/tickets", 3, 1, c))
	})
}

// TestAuthorizeResourceRejectsEmptyResourceAction 锁定空值拒绝语义。
//
// 背景：checkPermissionMatch 的通配分支在空 resource/action 下会产生意外放行
// （漏配 ToolDefinition.Resource 的工具会让持有 "*:<action>" 码的角色通过），
// 因此 AuthorizeResource 入口显式拒绝空值。这里用 super_admin 也一起锁定——
// 空值一律拒绝，不因角色高低而分叉。
func TestAuthorizeResourceRejectsEmptyResourceAction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	t.Run("empty resource is denied even for super_admin", func(t *testing.T) {
		assert.False(t, AuthorizeResource(ctx, nil, []string{"super_admin"}, "", "read", 1))
		assert.False(t, AuthorizeResourceForRole(ctx, nil, "super_admin", "", "read", 1))
	})

	t.Run("empty action is denied even for super_admin", func(t *testing.T) {
		assert.False(t, AuthorizeResource(ctx, nil, []string{"super_admin"}, "ticket", "", 1))
		assert.False(t, AuthorizeResourceForRole(ctx, nil, "super_admin", "ticket", "", 1))
	})

	t.Run("wildcard permission cannot bypass empty value check", func(t *testing.T) {
		// 持有 *:* 的超级角色也不能因空值放行：空值永远是调用方缺陷
		assert.False(t, AuthorizeResource(ctx, nil, []string{"tenant_owner"}, "", "", 1))
		assert.False(t, AuthorizeResourceForRole(ctx, nil, "tenant_owner", "", "", 1))
	})

	t.Run("non-empty values keep existing behavior", func(t *testing.T) {
		assert.True(t, AuthorizeResource(ctx, nil, []string{"super_admin"}, "ticket", "read", 1))
		assert.True(t, AuthorizeResourceForRole(ctx, nil, "super_admin", "ticket", "read", 1))
		assert.False(t, AuthorizeResource(ctx, nil, []string{"end_user"}, "ticket", "delete", 1))
	})
}
