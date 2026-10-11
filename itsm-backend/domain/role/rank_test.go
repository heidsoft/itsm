package role

import "testing"

// TestRank_PrivilegeEscalationGuard 修复回归：roleRank 词表曾缺 sysadmin/security_admin/
// audit_admin/it_admin（落 default 0），持 user:write 的 admin 可铸造 sysadmin（全权限码）
// 等价超管账号。Rank 单一事实源必须保证三员高于 admin。
func TestRank_PrivilegeEscalationGuard(t *testing.T) {
	if Rank(SysAdmin) <= Rank(Admin) {
		t.Fatalf("sysadmin(%d) 必须高于 admin(%d)——否则 admin 可铸造全权限账号", Rank(SysAdmin), Rank(Admin))
	}
	if Rank(SecurityAdmin) <= Rank(Admin) || Rank(AuditAdmin) <= Rank(Admin) {
		t.Fatalf("三员(security_admin=%d, audit_admin=%d)必须高于 admin(%d)：等保三员分立，admin 不得操纵",
			Rank(SecurityAdmin), Rank(AuditAdmin), Rank(Admin))
	}
	if Rank(SuperAdmin) <= Rank(SysAdmin) {
		t.Fatalf("super_admin(%d) 必须高于 sysadmin(%d)", Rank(SuperAdmin), Rank(SysAdmin))
	}
	if Rank(Admin) <= Rank(Manager) || Rank(Manager) <= Rank(Agent) || Rank(Agent) <= Rank(EndUser) {
		t.Fatalf("层级单调性破坏: admin=%d manager=%d agent=%d end_user=%d",
			Rank(Admin), Rank(Manager), Rank(Agent), Rank(EndUser))
	}
}

// TestRank_UnknownIsZero 未知/退役/空 code 返回 0：调用方必须以 IsBuiltinCode 白名单拒绝，
// rank 不做兜底授权。
func TestRank_UnknownIsZero(t *testing.T) {
	for _, code := range []string{"", "unknown_role", "security", "dba", "l1_support"} {
		if got := Rank(code); got != 0 {
			t.Fatalf("Rank(%q) = %d, want 0（未知/退役码不得隐含层级）", code, got)
		}
	}
}

// TestRank_UnknownRejectedByBuiltinWhitelist 退役码必须被 IsBuiltinCode 拦截——
// rank 0 + 白名单双层，缺一层都会让退役角色复活。
func TestRank_UnknownRejectedByBuiltinWhitelist(t *testing.T) {
	for _, code := range Retired {
		if IsBuiltinCode(code) {
			t.Fatalf("退役角色 %q 不应属于内置词表", code)
		}
	}
	for _, code := range All {
		if !IsBuiltinCode(code) {
			t.Fatalf("主角色 %q 必须在 IsBuiltinCode 词表内", code)
		}
	}
}
