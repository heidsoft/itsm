package menubaseline_test

import (
	"testing"

	domainrole "itsm-backend/domain/role"
	"itsm-backend/internal/authz"
	"itsm-backend/pkg/menubaseline"
)

// 菜单过滤（service/menu_service.go filterMenusByPermission）按权限码精确/别名匹配，
// 且 seeder 会把基线码写进已有行的 permission_code。
// 因此一个不在码空间的菜单码等价于「该入口永远不可见」，必须在基线层拦住。
func TestMenuBaseline_CodesExistInCatalog(t *testing.T) {
	defined := make(map[string]bool, len(authz.Definitions()))
	for _, d := range authz.Definitions() {
		defined[d.Code] = true
	}

	for _, spec := range menubaseline.MenuDefinitions() {
		if spec.PermissionCode == "" {
			continue
		}
		if _, _, ok := authz.SplitPermissionCode(spec.PermissionCode); !ok {
			t.Errorf("菜单 %q (%s) 的权限码 %q 不是 resource:action 形态", spec.Name, spec.Path, spec.PermissionCode)
			continue
		}
		if !defined[spec.PermissionCode] {
			t.Errorf("菜单 %q (%s) 使用了码空间未定义的权限码 %q", spec.Name, spec.Path, spec.PermissionCode)
		}
	}
}

// 审批待办视图的菜单归属与去重。
//
// /approvals 页面只消费 GET /bpmn/tasks（task:read）+ 工单/变更/服务请求的 pending 列表，
// 实测无任何 approval:* 调用；approval:read 属于审批记录与流程定义配置面。
// /approvals/pending 是 redirect 到 /approvals 的兼容壳，不得再作为菜单：
// 一个目的地两个菜单会让用户在「待我审批」「审批中心」两处看到同一份待办
// （审计 3.3-③ 四入口问题的去重）。
func TestMenuBaseline_ApprovalTodoMenuIsSingleEntry(t *testing.T) {
	codes := map[string]string{}
	for _, spec := range menubaseline.MenuDefinitions() {
		if spec.Path == "/approvals" || spec.Path == "/approvals/pending" {
			codes[spec.Path] = spec.PermissionCode
		}
	}
	if code, ok := codes["/approvals/pending"]; ok {
		t.Errorf("/approvals/pending 仍是菜单（权限码 %q）：它是 redirect 壳，菜单应只保留 /approvals", code)
	}
	code, ok := codes["/approvals"]
	if !ok {
		t.Fatalf("菜单基线缺少 /approvals 条目")
	}
	if code != "task:read" {
		t.Errorf("菜单 /approvals 的权限码是 %q，应为 task:read（页面数据源 GET /bpmn/tasks 的码）", code)
	}

	// 码改了还要真的有人能看到：待办处理人角色必须持有 task:read。
	roles := authz.BuiltinRolePermissionCodes()
	for _, role := range []string{domainrole.Agent, domainrole.Manager, domainrole.EndUser, domainrole.Technician} {
		found := false
		for _, c := range roles[role] {
			if c == "task:read" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("角色 %q 不持有 task:read，审批待办菜单对其不可见", role)
		}
	}
}

// 工单模板页的菜单归属。
//
// 页面数据源是 GET /api/v1/tickets/templates，接口要求 ticket_template:read；
// 菜单挂在 ticket:write 上时，只写自己工单的最终用户拿到入口后立刻 403
// （2026-10-18 本地生产栈实测 code 2003），而模板配置面本就不是他们的职责。
func TestMenuBaseline_TemplateMenuUsesTemplateRead(t *testing.T) {
	var code string
	for _, spec := range menubaseline.MenuDefinitions() {
		if spec.Path == "/tickets/templates" {
			code = spec.PermissionCode
		}
	}
	if code == "" {
		t.Fatalf("菜单基线缺少 /tickets/templates 条目")
	}
	if code != "ticket_template:read" {
		t.Errorf("菜单 /tickets/templates 的权限码是 %q，应为页面数据源要求的 ticket_template:read", code)
	}

	roles := authz.BuiltinRolePermissionCodes()
	holds := func(role, c string) bool {
		for _, x := range roles[role] {
			if x == c {
				return true
			}
		}
		return false
	}
	// 报障人不该有这个入口，也不该因为 ticket:write 的同义派生而拿到模板码。
	if holds(domainrole.EndUser, "ticket_template:read") {
		t.Errorf("end_user 持有 ticket_template:read，模板配置入口会回到报障人侧边栏")
	}
	// 真正维护模板的角色必须持字面码，否则菜单与接口同码但仍无人可见。
	for _, role := range []string{domainrole.Admin, domainrole.ITAdmin, domainrole.ServiceCatalogAdmin} {
		if !holds(role, "ticket_template:read") {
			t.Errorf("角色 %q 不持有 ticket_template:read，工单模板页对完全没有入口", role)
		}
	}
}

// 工单规则配置页的菜单归属。
//
// 三处入口曾分挂 ticket:read（人人有码，接口却拒绝）与 workflow:write（租管没有该码），
// 而接口要求 system_config:*：菜单可见性、读接口、写接口三套口径。
// 现在菜单码取该配置面的写码——这些页面是管理页，只读角色拿到入口后
// 每个增删改按钮都会 403，属于「菜单可见但接口拒绝」的老缺陷。
func TestMenuBaseline_RuleConfigMenusUseRuleCodes(t *testing.T) {
	want := map[string]string{
		"/workflow/automation":            "automation_rule:write",
		"/admin/tickets/automation-rules": "automation_rule:write",
		"/admin/tickets/assignment-rules": "assignment_rule:write",
	}
	menuCode := map[string]string{}
	for _, spec := range menubaseline.MenuDefinitions() {
		if _, ok := want[spec.Path]; ok {
			menuCode[spec.Path] = spec.PermissionCode
		}
	}
	for path, code := range want {
		got, ok := menuCode[path]
		if !ok {
			t.Fatalf("菜单基线缺少 %s 条目", path)
		}
		if got != code {
			t.Errorf("菜单 %s 的权限码是 %q，应为 %q", path, got, code)
		}
	}

	// 拿到管理页入口的角色必须同时能加载页面（列表接口用读码）。
	readOf := map[string]string{"automation_rule:write": "automation_rule:read", "assignment_rule:write": "assignment_rule:read"}
	for role, codes := range authz.BuiltinRolePermissionCodes() {
		holds := func(code string) bool {
			for _, c := range codes {
				if c == code {
					return true
				}
			}
			return false
		}
		for writeCode, readCode := range readOf {
			if holds(writeCode) && !holds(readCode) {
				t.Errorf("角色 %q 持有 %s 却缺少 %s，规则配置菜单可见但页面加载即 403", role, writeCode, readCode)
			}
		}
	}

	// 运营职责角色必须真的看得到：租管与 it_admin。
	roles := authz.BuiltinRolePermissionCodes()
	for _, role := range []string{domainrole.Admin, domainrole.ITAdmin} {
		for _, code := range []string{"automation_rule:write", "assignment_rule:write"} {
			found := false
			for _, c := range roles[role] {
				if c == code {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("角色 %q 不持有 %s，规则配置页面对其完全没有入口", role, code)
			}
		}
	}
}

// 工单统计页的菜单归属。
//
// 页面主数据源 GET /api/v1/analytics/tickets 要求 report:read；基线曾挂在 ticket:read 上，
// 于是持有 ticket:read、不持有 report:read 的坐席看得到入口，点进去就是 code 2003（2026-10-24
// 本地生产栈实测），页面再把 403 渲染成全 0。
func TestMenuBaseline_TicketAnalyticsMenuUsesReportRead(t *testing.T) {
	var code string
	for _, spec := range menubaseline.MenuDefinitions() {
		if spec.Path == "/tickets/analytics" {
			code = spec.PermissionCode
		}
	}
	if code == "" {
		t.Fatalf("菜单基线缺少 /tickets/analytics 条目")
	}
	if code != "report:read" {
		t.Errorf("菜单 /tickets/analytics 的权限码是 %q，应为页面数据源要求的 report:read", code)
	}

	holds := func(role, c string) bool {
		for _, x := range authz.BuiltinRolePermissionCodes()[role] {
			if x == c {
				return true
			}
		}
		return false
	}
	// 报表页要有真正的读者，否则改码等于把页面藏起来。
	for _, role := range []string{domainrole.Admin, domainrole.ITAdmin, domainrole.Manager} {
		if !holds(role, "report:read") {
			t.Errorf("角色 %q 不持有 report:read，工单统计页对其完全没有入口", role)
		}
	}
	// 反向锁：不要为了让坐席看得懂这个页面而给他们补 report:read——
	// 那会连带放开整个报表中心。坐席视角的统计入口是独立工作面，不在本条断言范围内。
	if holds(domainrole.Agent, "report:read") {
		t.Errorf("agent 持有 report:read，报表中心整套页面会一起进入坐席侧边栏")
	}
}

// 系统管理子树的权限码单轨。
//
// 菜单曾声明 system:write / system:read，而这两枚码只有 sysadmin 持有；其余角色
// 能看见这棵树全靠 menu_service 的 system→system_config 资源别名兜着——声明码与
// 真实可见性来源不一致（审计 3.2-⑤）。现在统一到租户配置域 system_config:*，
// 平台级运维端点（readiness/operations/timers）才继续用 system:*。
func TestMenuBaseline_SystemMenusUseConfigCodes(t *testing.T) {
	want := map[string]string{
		"/admin":               "system_config:read",
		"/admin/overview":      "system_config:read",
		"/admin/tenants":       "tenant:read",
		"/admin/system-config": "system_config:read",
		"/admin/vector-store":  "system_config:read",
		"/admin/tags":          "system_config:read",
		"/admin/capabilities":  "system_config:read",
		"/admin/menus":         "system_config:write",
	}
	got := map[string]string{}
	for _, spec := range menubaseline.MenuDefinitions() {
		if _, ok := want[spec.Path]; ok {
			got[spec.Path] = spec.PermissionCode
		}
	}
	for path, code := range want {
		if got[path] != code {
			t.Errorf("菜单 %s 的权限码是 %q，应为 %q", path, got[path], code)
		}
	}

	// 反向锁：基线里不得再出现 system:read / system:write —— 那是平台运维码，
	// 一旦回到菜单层，租管/it_admin 又会「看得到入口、点进去 403」。
	for _, spec := range menubaseline.MenuDefinitions() {
		if spec.PermissionCode == "system:read" || spec.PermissionCode == "system:write" {
			t.Errorf("菜单 %q (%s) 使用平台运维码 %q：租户配置页应使用 system_config:*", spec.Name, spec.Path, spec.PermissionCode)
		}
	}
}

// 报表中心的权限码。
//
// dashboard:read 是全角色基线（服务台落地页需要它），拿它当报表菜单码等于
// 把整套报表入口发给最终用户（审计遗留项 6）。报表是管理/分析面，取 report:read，
// 与 /reports/* 接口同码。
func TestMenuBaseline_ReportsMenuUsesReportRead(t *testing.T) {
	var code string
	for _, spec := range menubaseline.MenuDefinitions() {
		if spec.Path == "/reports" {
			code = spec.PermissionCode
		}
	}
	if code == "" {
		t.Fatalf("菜单基线缺少 /reports 条目")
	}
	if code != "report:read" {
		t.Errorf("菜单 /reports 的权限码是 %q，应为 report:read", code)
	}

	holds := func(role, c string) bool {
		for _, x := range authz.BuiltinRolePermissionCodes()[role] {
			if x == c {
				return true
			}
		}
		return false
	}
	if holds(domainrole.EndUser, "report:read") {
		t.Errorf("end_user 持有 report:read，报表中心会回到最终用户侧边栏")
	}
	if holds(domainrole.Technician, "report:read") {
		t.Errorf("technician 持有 report:read，二线技术员侧边栏会出现整套报表入口")
	}
	for _, role := range []string{domainrole.Admin, domainrole.ITAdmin, domainrole.Manager} {
		if !holds(role, "report:read") {
			t.Errorf("角色 %q 不持有 report:read，报表中心对其完全没有入口", role)
		}
	}
}

// 「客户与合同」的父子语义。
//
// 父级曾是「邮件报障」（AI 报障入口），子项却是客户资料/支持合同/来源组织/值班排班。
// 现在父级改名「客户与合同」只承载客户主数据；值班排班归事件管理（响应排班）。
func TestMenuBaseline_CustomerContractChildren(t *testing.T) {
	children := map[string]string{} // path -> parent path
	names := map[string]string{}
	for _, spec := range menubaseline.MenuDefinitions() {
		if spec.ParentPath == "/email-intake" {
			children[spec.Path] = spec.ParentPath
			names[spec.Path] = spec.Name
		}
	}
	want := []string{"/email-intake/customers", "/email-intake/contracts", "/email-intake/sources"}
	if len(children) != len(want) {
		t.Errorf("/email-intake 下有 %d 个子项（%v），应为 %d 个客户主数据子项", len(children), names, len(want))
	}
	for _, p := range want {
		if _, ok := children[p]; !ok {
			t.Errorf("/email-intake 缺少子项 %s", p)
		}
	}
	if _, ok := children["/email-intake/on-call"]; ok {
		t.Errorf("值班排班仍挂在客户主数据下：它是事件响应排班，应归事件管理")
	}

	var onCallParent string
	for _, spec := range menubaseline.MenuDefinitions() {
		if spec.Path == "/email-intake/on-call" {
			onCallParent = spec.ParentPath
		}
	}
	if onCallParent != "/incidents" {
		t.Errorf("值班排班的父菜单是 %q，应为 /incidents", onCallParent)
	}
}
