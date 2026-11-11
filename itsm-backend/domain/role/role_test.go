package role

import "testing"

// 校验 users.role 枚举与 roles.code 词表没有漂移。
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

// 实践角色与主角色词表必须互斥：IsPracticeCode 是「用户把实践角色写进
// users.role」这条错误路径的判据，词表重叠会让提示错判成主角色。
func TestPracticeCodesAreDisjointFromMainRoles(t *testing.T) {
	if len(Practice) == 0 {
		t.Fatal("Practice 词表为空")
	}
	for _, code := range Practice {
		if !IsPracticeCode(code) {
			t.Errorf("IsPracticeCode(%q) = false, want true", code)
		}
		if IsBuiltinCode(code) {
			t.Errorf("%q 同时出现在 Practice 与 All，users.role 判据会被误判", code)
		}
	}
	for _, code := range All {
		if IsPracticeCode(code) {
			t.Errorf("主角色 %q 被 IsPracticeCode 命中，词表重叠", code)
		}
	}
	if IsPracticeCode("") || IsPracticeCode("not_a_role") {
		t.Error("未知/空 code 不得被判为实践角色")
	}
}
