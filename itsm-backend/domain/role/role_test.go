package role

import "testing"

// 角色词表单一源回归：防再次漂移（2026-09-07 服务请求深测 P1-B）。
func TestRoleVocabularyContract(t *testing.T) {
	// users.role 与 roles.code 必须共用的核心审批角色
	required := []string{Manager, ITAdmin, SecurityAdmin}
	inAll := func(code string) bool {
		for _, r := range All {
			if r == code {
				return true
			}
		}
		return false
	}
	for _, code := range required {
		if !inAll(code) {
			t.Errorf("All 缺少审批角色 %q —— service_request 审批链依赖它", code)
		}
	}

	// IsAdminLike 口径必须与 datascope 全量可见一致
	for _, r := range []string{SuperAdmin, Admin, Manager, SysAdmin} {
		if !IsAdminLike(r) {
			t.Errorf("IsAdminLike(%q) = false, want true", r)
		}
	}
	for _, r := range []string{EndUser, Agent, Technician, ITAdmin, SecurityAdmin} {
		if IsAdminLike(r) {
			t.Errorf("IsAdminLike(%q) = true, want false", r)
		}
	}

	// 审批角色判定：L1/L2/L3 三层角色必须可审批
	for _, r := range required {
		if !IsServiceRequestApprover(r) {
			t.Errorf("IsServiceRequestApprover(%q) = false, want true", r)
		}
	}
	if IsServiceRequestApprover("guest") {
		t.Error("IsServiceRequestApprover(guest) = true, want false")
	}
}

// All 清单不得有重复项（防复制粘贴漂移）。
func TestAllNoDuplicates(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range All {
		if seen[r] {
			t.Errorf("All 中角色 %q 重复", r)
		}
		seen[r] = true
	}
}
