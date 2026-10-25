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

// 审批待办视图的菜单归属。
//
// /approvals 页面只消费 GET /bpmn/tasks（task:read）+ 工单/变更/服务请求的 pending 列表，
// 实测无任何 approval:* 调用；approval:read 属于审批记录与流程定义配置面。
// 基线曾把它当菜单码：agent/manager/end_user/technician 持有 task:read 却不持有 approval:read，
// 而 /approvals 挂在 /workflow 下（buildMenuTree 归入 admin 桶，前端只给管理角色渲染），
// 结果待办处理人在 UI 上没有任何审批入口。
func TestMenuBaseline_ApprovalTodoMenusUseTaskRead(t *testing.T) {
	codes := map[string]string{}
	for _, spec := range menubaseline.MenuDefinitions() {
		switch spec.Path {
		case "/approvals", "/approvals/pending":
			codes[spec.Path] = spec.PermissionCode
		}
	}
	for _, path := range []string{"/approvals", "/approvals/pending"} {
		code, ok := codes[path]
		if !ok {
			t.Fatalf("菜单基线缺少 %s 条目", path)
		}
		if code != "task:read" {
			t.Errorf("菜单 %s 的权限码是 %q，应为 task:read（页面数据源 GET /bpmn/tasks 的码）", path, code)
		}
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
