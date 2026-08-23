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
