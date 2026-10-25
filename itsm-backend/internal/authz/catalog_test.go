package authz

import (
	"sort"
	"testing"

	domainrole "itsm-backend/domain/role"
)

// TestDefinitionsParityWithSeeder 校验权限码目录的内部一致性：Code 必须等于
// Resource:Action，AllCodes 与 Definitions 同序同量，且码全局唯一。
// 目录本身已从 pkg/seeder 迁入本包，seeder 直接消费 Definitions()。
func TestDefinitionsParityWithSeeder(t *testing.T) {
	cat := AllCodes()
	if len(cat) < 100 {
		t.Fatalf("catalog 仅 %d 个码，疑似提取逻辑失效", len(cat))
	}

	// 字段切分一致性：Code 的 "{resource}:{action}" 必须与字段匹配
	for _, d := range Definitions() {
		want := d.Resource + ":" + d.Action
		if d.Code != want {
			t.Errorf("权限码 %q 的 Resource:Action 拆分与 Code 不一致：want=%q", d.Code, want)
		}
	}

	// 内部一致性：AllCodes 与 Definitions 同序（保持 Definitions 的源序，便于 codegen diff）
	defs := Definitions()
	if len(cat) != len(defs) {
		t.Fatalf("AllCodes(%d) 与 Definitions 长度(%d) 不一致", len(cat), len(defs))
	}
	for i, c := range cat {
		if c != defs[i].Code {
			t.Errorf("AllCodes[%d]=%q != defs[%d].Code=%q", i, c, i, defs[i].Code)
		}
	}

	// 全局唯一性（无重复码）
	seen := map[string]int{}
	for _, c := range cat {
		seen[c]++
	}
	for c, n := range seen {
		if n > 1 {
			t.Errorf("权限码 %q 在 Definitions 中重复 %d 次", c, n)
		}
	}

	_ = len(cat) // 排序由 codegen 阶段处理（保持源序便于 review）
}

// TestBuiltinRolesUseOnlyCatalogCodes 校验内置角色的权限码全部来自 catalog。
//
// 这是"权限码漂移"的兜底：若 BuiltinRolePermissionCodes() 引用了 catalog 里
// 不存在的码（拼写错误、码被重命名但角色清单没跟着改），
// checkPermissionMatch 是精确匹配，该路由会对除 super_admin 外的所有角色永久 403，
// 而没有任何机制会报错。
func TestBuiltinRolesUseOnlyCatalogCodes(t *testing.T) {
	cat := AllCodes()
	valid := make(map[string]bool, len(cat))
	for _, c := range cat {
		valid[c] = true
	}

	for role, codes := range BuiltinRolePermissionCodes() {
		for _, c := range codes {
			if !valid[c] {
				t.Errorf("角色 %s 引用了 catalog 中不存在的权限码 %q", role, c)
			}
		}
	}
}

// TestSysAdminGetsFullCatalog 校验 sysadmin 的播种码集等于 catalog 全量。
//
// 该测试锁定 2026-10-13 的修复：此前 allPermissionCodes() 是一份手写清单，
// 与 catalog 存在差集（实测 129 vs 160，缺 33 个含 tenant:read/write、
// notification:*、system_config:* 等），导致最高权限角色拿不到新能力且无告警。
// 现在必须直接取权威投影，任何人改回手写枚举都会在这里失败。
func TestSysAdminGetsFullCatalog(t *testing.T) {
	roles := BuiltinRolePermissionCodes()
	sysadmin, ok := roles[string(domainrole.SysAdmin)]
	if !ok {
		t.Fatal("BuiltinRolePermissionCodes 缺少 sysadmin 条目")
	}

	cat := AllCodes()
	got := make(map[string]bool, len(sysadmin))
	for _, c := range sysadmin {
		got[c] = true
	}
	if len(got) != len(sysadmin) {
		t.Errorf("sysadmin 码集内重复码：%d 个码去重后 %d 个", len(sysadmin), len(got))
	}

	for _, c := range cat {
		if !got[c] {
			t.Errorf("sysadmin 缺少 catalog 权限码 %q", c)
		}
	}
	if len(got) < len(cat) {
		t.Errorf("sysadmin 码集(%d) 少于 catalog(%d)，存在缺码", len(got), len(cat))
	}
	if len(got) > len(cat) {
		t.Errorf("sysadmin 码集(%d) 多于 catalog(%d)，出现 catalog 之外的越权码", len(got), len(cat))
	}
}

// TestEndUserDoesNotHoldAdminScopedReads 锁定 end_user 的最小权限口径。
//
// 2026-10-16 隔离栈实测：end_user 持 system_config:read，可直接 GET /api/v1/system-configs
// 整表读到 SMTP/密码策略/2FA 等租户配置，并且当时 GET /api/v1/system-configs/init
// 按同一读权限守卫却整表写入默认策略（passwordMinLength=8、enable2FA=false），
// 一次 GET 即完成。前端只有 /admin/system-config 会读这些接口，最终用户没有该入口。
func TestEndUserDoesNotHoldAdminScopedReads(t *testing.T) {
	roles := BuiltinRolePermissionCodes()
	endUser, ok := roles[string(domainrole.EndUser)]
	if !ok {
		t.Fatal("BuiltinRolePermissionCodes 缺少 end_user 条目")
	}

	granted := make(map[string]bool, len(endUser))
	for _, c := range endUser {
		granted[c] = true
	}

	for _, c := range []string{
		"system_config:read", "system_config:write", "user:write", "user:delete", "tenant:read", "audit:read",
		"automation_rule:read", "automation_rule:write", "assignment_rule:read", "assignment_rule:write",
	} {
		if granted[c] {
			t.Errorf("end_user 不应持有权限码 %q：对应接口是管理面，最终用户无入口", c)
		}
	}
}

// TestRuleConfigCodesAreNotWorkflowRead 锁定规则配置面为什么不复用 workflow:read 作读守卫。
//
// 工单规则列表（automation/assignment）返回租户全部规则的条件、动作与创建人
// （dto.AutomationRuleResponse 内嵌 creator 的 username/email/role），属运营配置面；
// 而 workflow:read 同时挂着最终用户自己的旅程接口
// （/tickets/cc/my、/tickets/:id/workflow/state-v2 等流转进度读），
// end_user 必须持有它。若规则读也挂 workflow:read，等于给报障人开了整表枚举
// 租户自动化规则与人员目录的口子——与本文件上一条测试的口径直接冲突。
func TestRuleConfigCodesAreNotWorkflowRead(t *testing.T) {
	roles := BuiltinRolePermissionCodes()
	endUser := roles[string(domainrole.EndUser)]

	granted := make(map[string]bool, len(endUser))
	for _, c := range endUser {
		granted[c] = true
	}
	if !granted["workflow:read"] {
		t.Fatal("end_user 必须持有 workflow:read，否则工单详情流转进度与抄送列表不可读")
	}

	defined := make(map[string]bool, len(Definitions()))
	for _, d := range Definitions() {
		defined[d.Code] = true
	}
	for _, c := range []string{"automation_rule:read", "automation_rule:write", "assignment_rule:read", "assignment_rule:write"} {
		if !defined[c] {
			t.Errorf("权限码 %q 未登记：路由引用未定义码时，DBOnly 下除 super_admin 外永久 403", c)
		}
	}
}

// TestMarketplaceCatalogWriteIsPlatformOnly 锁定「上架」与「安装」的权限边界。
//
// marketplace_items 是全局表（无 tenant_id），所有租户共享同一份目录。
// marketplace:write 已授予租户 admin，用于安装/卸载/改配置——那是租户自己的
// 运行态。若目录写复用同一个码，任一租户管理员就能改写全平台可见的商品：
// 把 name 指向未注册 provider、把 latest_version 指向不存在的版本，
// 或直接下架别人正在用的条目。因此目录写必须是平台级码，只给 sysadmin。
//
// 这条测试的作用是防止「顺手把新码加进 admin 授权列表」这类改动——那不会
// 让任何现有测试变红，却会把跨租户污染面重新打开。
func TestMarketplaceCatalogWriteIsPlatformOnly(t *testing.T) {
	roles := BuiltinRolePermissionCodes()

	hasCode := func(role, code string) bool {
		for _, c := range roles[role] {
			if c == code {
				return true
			}
		}
		return false
	}

	if !hasCode(string(domainrole.SysAdmin), "marketplace_catalog:write") {
		t.Error("sysadmin 必须持有 marketplace_catalog:write，否则目录无管理面（Pilot 阻断项）")
	}
	for _, role := range []string{
		string(domainrole.Admin), string(domainrole.Manager), string(domainrole.EndUser), string(domainrole.Agent),
	} {
		if hasCode(role, "marketplace_catalog:write") {
			t.Errorf("%s 不应持有 marketplace_catalog:write：目录是全局表，租户角色写入即跨租户污染", role)
		}
	}
	// 安装能力不能被这次拆分误伤：租户 admin 仍要能装。
	if !hasCode(string(domainrole.Admin), "marketplace:write") {
		t.Error("admin 必须保留 marketplace:write，否则无法安装/卸载扩展")
	}
}

// TestTicketTypeManageImpliesInstallPreset 锁定「能建类型就必须能装预设」的口径。
//
// InstallPreset 的写入就是 CreateTicketType + 一条审计（service/ticket_type_service.go），
// 而 POST /api/v1/ticket-types 只按 ticket_type:manage 授权。install_preset 因此严格窄于
// manage，不构成安全边界；只授 manage 的角色会在前端「安装」按钮上稳定 403
// （2026-10-24 实测：it_admin 与租户 admin 有 manage 无 install_preset，预设库能打开、
// 每个安装都 code 2003）。反向仍需守住：只有 read 的坐席不得拿到这两个写码。
func TestTicketTypeManageImpliesInstallPreset(t *testing.T) {
	roles := BuiltinRolePermissionCodes()

	hasCode := func(codes []string, code string) bool {
		for _, c := range codes {
			if c == code {
				return true
			}
		}
		return false
	}

	names := make([]string, 0, len(roles))
	for role := range roles {
		names = append(names, role)
	}
	sort.Strings(names)
	for _, role := range names {
		codes := roles[role]
		if hasCode(codes, "ticket_type:manage") && !hasCode(codes, "ticket_type:install_preset") {
			t.Errorf("%s 持 ticket_type:manage 却无 install_preset：可以自建类型却不能装预设，安装按钮恒 403", role)
		}
	}

	agent := roles[string(domainrole.Agent)]
	if !hasCode(agent, "ticket_type:read") {
		t.Error("agent 必须持 ticket_type:read，否则建单页类型下拉与预设库都不可读")
	}
	for _, code := range []string{"ticket_type:manage", "ticket_type:install_preset", "ticket_type:archive"} {
		if hasCode(agent, code) {
			t.Errorf("agent 不应持有 %s：坐席按口径只能读工单类型配置", code)
		}
	}
}
