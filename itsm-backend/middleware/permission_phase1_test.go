package middleware

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// Phase 1 统一判定：AuthorizeResource 多角色并集语义。
func TestAuthorizeResource_MultiRoleUnion(t *testing.T) {
	// nil client + DBOnly（默认模式）：无任何角色权限数据时应拒绝非超管角色。
	// 用 Fallback 模式验证多角色并集：硬编码里 end_user 无 ticket:delete，
	// manager 有。双角色并集应命中 manager 的权限。
	origMode := PermissionConfig.Mode
	PermissionConfig.Mode = PermissionConfigModeFallback
	defer func() { PermissionConfig.Mode = origMode }()

	ctx := context.Background()

	// 单角色：end_user 不应拥有 ticket:assign（硬编码无此授权）
	if AuthorizeResource(ctx, nil, []string{"end_user"}, "ticket", "assign", 1) {
		t.Fatal("end_user 不应拥有 ticket:assign")
	}
	// 多角色并集：end_user + manager -> manager 的 ticket:assign 生效
	if !AuthorizeResource(ctx, nil, []string{"end_user", "manager"}, "ticket", "assign", 1) {
		t.Fatal("end_user+manager 并集应拥有 ticket:assign（manager 授权）")
	}
	// super_admin 出现在任一角色即直通
	if !AuthorizeResource(ctx, nil, []string{"end_user", "super_admin"}, "anything", "anything", 1) {
		t.Fatal("含 super_admin 的角色集应直通")
	}
	// 空角色集拒绝
	if AuthorizeResource(ctx, nil, nil, "ticket", "read", 1) {
		t.Fatal("空角色集应拒绝")
	}
}

// Phase 1：GetContextRoles 主角色+附加角色去重合并。
func TestGetContextRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := gin.CreateTestContextOnly(httptest.NewRecorder(), gin.New())

	// 未注入任何角色 -> 空
	if roles := GetContextRoles(c); len(roles) != 0 {
		t.Fatalf("无角色时应返回空, got %v", roles)
	}

	// 仅主角色
	c.Set("role", "agent")
	roles := GetContextRoles(c)
	if len(roles) != 1 || roles[0] != "agent" {
		t.Fatalf("应仅含主角色, got %v", roles)
	}

	// 主角色 + 附加角色（含重复与空串）
	c.Set("roles", []string{"technician", "agent", "", "security"})
	roles = GetContextRoles(c)
	if len(roles) != 3 {
		t.Fatalf("去重后应 3 个角色, got %v", roles)
	}
	want := map[string]bool{"agent": true, "technician": true, "security": true}
	for _, r := range roles {
		if !want[r] {
			t.Fatalf("意外角色 %v, got %v", r, roles)
		}
	}
}

// Phase 1：SmartCheckPermission 的 L4 兜底在 DBOnly 下必须 fail-closed。
func TestSmartCheckPermission_DBOnlyFailClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := gin.CreateTestContextOnly(httptest.NewRecorder(), gin.New())
	c.Request = httptest.NewRequest("GET", "/api/v1/tickets/1", nil)

	origMode := PermissionConfig.Mode
	defer func() { PermissionConfig.Mode = origMode }()

	// Fallback（dev）模式：硬编码里 agent 有 ticket:read -> 允许
	PermissionConfig.Mode = PermissionConfigModeFallback
	if !SmartCheckPermission(c, nil, "agent", "GET", "/api/v1/tickets/1", 1) {
		t.Fatal("Fallback 模式下 agent 应有 ticket:read（硬编码兜底）")
	}

	// DBOnly（生产）模式：无 DB 权限数据 -> L4 不得回退硬编码，必须拒绝
	PermissionConfig.Mode = PermissionConfigModeDBOnly
	if SmartCheckPermission(c, nil, "agent", "GET", "/api/v1/tickets/1", 1) {
		t.Fatal("DBOnly 模式下 L4 不得回退硬编码（fail-closed）")
	}

	// super_admin 在 DBOnly 下仍直通
	if !SmartCheckPermission(c, nil, "super_admin", "GET", "/api/v1/tickets/1", 1) {
		t.Fatal("super_admin 应直通")
	}
}

// Phase 1：广播消息解析（"role|tenant" 载荷）。
func TestParseInvalidateMessage(t *testing.T) {
	if role, tenant, ok := parseInvalidateMessage("ops_manager|7"); !ok || role != "ops_manager" || tenant != 7 {
		t.Fatalf("合法载荷解析失败: %v %v %v", role, tenant, ok)
	}
	if _, _, ok := parseInvalidateMessage("bad-payload"); ok {
		t.Fatal("非法载荷应解析失败")
	}
	if _, _, ok := parseInvalidateMessage("role|abc"); ok {
		t.Fatal("租户ID非数字应解析失败")
	}
	if _, _, ok := parseInvalidateMessage("|7"); ok {
		t.Fatal("空角色应解析失败")
	}
	if _, _, ok := parseInvalidateMessage("ops_manager|"); ok {
		t.Fatal("空租户段应解析失败")
	}
	// 角色名本身含分隔符时按最后一个分隔符切分，租户段仍可正确落位
	if role, tenant, ok := parseInvalidateMessage("a|b|7"); !ok || role != "a|b" || tenant != 7 {
		t.Fatalf("多分隔符载荷解析失败: %v %v %v", role, tenant, ok)
	}
}
