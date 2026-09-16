package middleware

import (
	"context"
	"testing"
)

// 2026-09-16 P0 修复：DBOnly 模式下三态语义区分：
//   - unavailable (client==nil)   → fail-closed（不允许任何授权），保留 baseline fail-closed 测试
//   - unconfigured (DB 无角色行)   → 走硬编码 RolePermissions 兜底（避免新装/小租户端点全 403）
//   - configured (DB 角色行存在)   → 以 DB 为准（含空集=显式撤销，仍 fail-closed）
//
// 本测试只覆盖走 unconfigured 分支的语义（需有可用 ent client 且 DB 中无角色行）；
// unavailable 分支由既有的 TestSmartCheckPermission_DBOnlyFailClosed /
// TestDBOnlyPermissionModeDoesNotUseHardcodedFallback 等 fail-closed 测试守护。
// configured 显式撤销分支需要 ent mock（entt 或 sqlmock），此处不展开。
func TestAuthorizeResource_DBOnlyClientNilFailClosed(t *testing.T) {
	origMode := PermissionConfig.Mode
	defer func() { PermissionConfig.Mode = origMode }()
	PermissionConfig.Mode = PermissionConfigModeDBOnly

	ctx := context.Background()
	// client=nil → unavailable → fail-closed，与 baseline 一致
	if AuthorizeResource(ctx, nil, []string{"end_user"}, "dashboard", "read", 1) {
		t.Fatal("DBOnly + client=nil 应 fail-closed（与既有 TestSmartCheckPermission_DBOnlyFailClosed 一致）")
	}
	// 即便硬编码里有这条权限，也仍 fail-closed
	if AuthorizeResource(ctx, nil, []string{"admin"}, "ticket", "delete", 1) {
		t.Fatal("DBOnly + client=nil 下硬编码不可生效")
	}
	// super_admin 直通不受 DBOnly / unavailable 影响
	if !AuthorizeResource(ctx, nil, []string{"super_admin"}, "anything", "anything", 1) {
		t.Fatal("super_admin 必须直通")
	}
}

// Fallback 模式下走 loadPermissionsFromDB → 硬编码兜底保持既有契约。
// 该测试锁定 DBOnly 之外的模式行为不变。
func TestAuthorizeResource_FallbackModeHardcodeFallback(t *testing.T) {
	origMode := PermissionConfig.Mode
	defer func() { PermissionConfig.Mode = origMode }()
	PermissionConfig.Mode = PermissionConfigModeFallback

	ctx := context.Background()
	// Fallback + client=nil → loadPermissionsFromDB 返回 nil → 走硬编码兜底（既有契约）
	if !AuthorizeResource(ctx, nil, []string{"end_user"}, "dashboard", "read", 1) {
		t.Fatal("Fallback + client=nil 应回退硬编码（dashboard:read）")
	}
	if AuthorizeResource(ctx, nil, []string{"end_user"}, "ticket", "delete", 1) {
		t.Fatal("Fallback 模式下 end_user 仍不应有 ticket:delete")
	}
}

// Merge 模式（DB + 硬编码并集）：client=nil 时硬编码生效（与既有契约一致）。
func TestAuthorizeResource_MergeModeClientNilUsesHardcode(t *testing.T) {
	origMode := PermissionConfig.Mode
	defer func() { PermissionConfig.Mode = origMode }()
	PermissionConfig.Mode = PermissionConfigModeMerge

	ctx := context.Background()
	if !AuthorizeResource(ctx, nil, []string{"end_user"}, "dashboard", "read", 1) {
		t.Fatal("Merge + client=nil 应回退硬编码（dashboard:read 命中）")
	}
}

// loadPermissionsByMode 的三态分流逻辑：在 nil client 场景下 DBOnly 必须返回 nil，
// 否则会破坏既有 fail-closed 测试。已通过 TestAuthorizeResource_DBOnlyClientNilFailClosed
// 端到端验证；此处再单测 loadPermissionsByMode 自身的 DBOnly 分支。
func TestLoadPermissionsByMode_DBOnlyClientNil(t *testing.T) {
	origMode := PermissionConfig.Mode
	defer func() { PermissionConfig.Mode = origMode }()
	PermissionConfig.Mode = PermissionConfigModeDBOnly

	perms := loadPermissionsByMode(context.Background(), nil, "end_user", 1)
	if perms != nil {
		t.Fatalf("DBOnly + client=nil 应返回 nil（unavailable → fail-closed），got %v", perms)
	}
	perms = loadPermissionsByMode(context.Background(), nil, "admin", 1)
	if perms != nil {
		t.Fatalf("DBOnly + client=nil 应返回 nil（即使是 admin 也 fail-closed），got %v", perms)
	}
}

// 三态枚举 + loadPermissionsFromDBDBOnlyState 的 nil-client 单测：unavailable 分支。
// 该测试保证 fail-closed 守门——后续如果有人误把 nil client 改成「走兜底」会立刻失败。
func TestLoadPermissionsFromDBDBOnlyState_NilClient(t *testing.T) {
	state, perms := loadPermissionsFromDBDBOnlyState(context.Background(), nil, "admin", 1)
	if state != permissionDBOnlyUnavailable {
		t.Fatalf("client=nil 应返回 unavailable, got %v", state)
	}
	if perms != nil {
		t.Fatalf("unavailable 分支应返回 nil perms, got %v", perms)
	}
}

// HardcodeOnly 模式：仍走硬编码（既有契约不变）。
func TestLoadPermissionsByMode_HardcodeOnlyMode(t *testing.T) {
	origMode := PermissionConfig.Mode
	defer func() { PermissionConfig.Mode = origMode }()
	PermissionConfig.Mode = PermissionConfigModeHardcodeOnly

	perms := loadPermissionsByMode(context.Background(), nil, "end_user", 1)
	if len(perms) == 0 {
		t.Fatal("HardcodeOnly 模式应走硬编码（end_user 不应有空权限集）")
	}
	if !hasPermissionInSlice(perms, "ticket", "read") {
		t.Error("HardcodeOnly 模式下 end_user 应有 ticket:read")
	}
}

// hasPermissionInSlice 简易断言辅助
func hasPermissionInSlice(perms []Permission, resource, action string) bool {
	for _, p := range perms {
		if p.Resource == resource && (p.Action == action || p.Action == "*" || p.Action == "admin") {
			return true
		}
	}
	return false
}
