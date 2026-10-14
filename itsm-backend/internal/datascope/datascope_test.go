package datascope

import "testing"

// TestIsDataScopeAllRole 锁定角色→可见范围映射，防止回归导致越权收窄或放宽。
func TestIsDataScopeAllRole(t *testing.T) {
	allRoles := []string{"super_admin", "admin", "manager", "sysadmin"}
	for _, r := range allRoles {
		if !IsDataScopeAllRole(r) {
			t.Errorf("role %q 应为 DataScopeAll（全租户可见）", r)
		}
	}

	// 非管理角色（含空串）必须收窄到 OwnedOrAssigned，安全默认是“收窄而非放宽”。
	notAll := []string{"end_user", "agent", "", "guest", "unknown"}
	for _, r := range notAll {
		if IsDataScopeAllRole(r) {
			t.Errorf("role %q 不应为 DataScopeAll", r)
		}
	}
}

// TestDataScopeConstants 确保两个枚举值互不相同。
func TestDataScopeConstants(t *testing.T) {
	if DataScopeAll == DataScopeOwnedOrAssigned {
		t.Fatal("DataScopeAll 必须与 DataScopeOwnedOrAssigned 不同")
	}
}

// TestCanWriteResource_AdminLikeRolesAlwaysAllowed 管理角色全租户可写。
func TestCanWriteResource_AdminLikeRolesAlwaysAllowed(t *testing.T) {
	for _, role := range []string{"super_admin", "admin", "manager", "sysadmin"} {
		if !CanWriteResource(1, role, 999, nil) {
			t.Errorf("role %q 应全租户可写", role)
		}
		if !CanWriteResource(0, role, 999, nil) {
			t.Errorf("role %q 即使无 actorID 仍应可写（管理角色绕过行级 scope）", role)
		}
	}
}

// TestCanWriteResource_OwnerOrAssignee 普通角色仅 owner/受理人可写。
func TestCanWriteResource_OwnerOrAssignee(t *testing.T) {
	assignee := 42
	other := 7

	// owner 命中
	if !CanWriteResource(other, "agent", other, nil) {
		t.Error("owner 应可写")
	}
	// 受理人命中
	if !CanWriteResource(assignee, "agent", 999, &assignee) {
		t.Error("受理人应可写")
	}
	// 均不命中 → 拒绝
	if CanWriteResource(other, "agent", 999, &assignee) {
		t.Error("非 owner 且非受理人必须拒绝")
	}
	// 未分配时仅 owner 可写
	if CanWriteResource(assignee, "agent", 999, nil) {
		t.Error("未分配时非 owner 必须拒绝")
	}
}

// TestCanWriteResource_SafetyNarrowing 安全收窄：无效身份/未知角色按拒绝处理。
func TestCanWriteResource_SafetyNarrowing(t *testing.T) {
	// actorID 无效 → 拒绝（即使 role 是 agent）
	if CanWriteResource(0, "agent", 0, nil) {
		t.Error("actorID=0 必须拒绝（无有效身份）")
	}
	if CanWriteResource(-1, "agent", -1, nil) {
		t.Error("actorID<0 必须拒绝")
	}
	// 未知/空角色 + 非 owner → 拒绝
	if CanWriteResource(5, "unknown_role", 999, nil) {
		t.Error("未知角色非 owner 必须拒绝")
	}
	if CanWriteResource(5, "", 999, nil) {
		t.Error("空角色非 owner 必须拒绝")
	}
	// 受理人指针指向 0 不可越权
	zero := 0
	if CanWriteResource(5, "agent", 999, &zero) {
		t.Error("受理人 ID=0 必须拒绝")
	}
}
