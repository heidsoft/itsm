package authz

import domainrole "itsm-backend/domain/role"

// 本文件是「角色 → 权限码」绑定的权威源。
//
// 词表边界：本 map 的键必须恰好等于
// domain/role.All ∪ domain/role.Practice ∪ domain/role.MSP（由
// domain/role/contract_test.go 锁死）；岗位/团队型条目（旧 it_director、
// ops_*、l1/l2/l3_*、dba、network_eng、sd_manager、rd_manager、developer、
// qa_engineer、dept_manager、team_lead、guest）与 legacy security 已退役，
// 组织架构差异由部门/团队/租户自建角色承载。
//
// 派生规则说明（见 BuiltinRolePermissionCodes 尾部实现）：
//   - 同义动作补齐（write ⇒ {create, update} 等）：路由按细分动作声明
//     RequirePermission，只授予 write 的角色会 403
//   - task:read/update 全角色基线（审计员只读除外）：任何角色都可能被流程指派任务
//     （变更评审、服务请求审批、工单流转），粒度控制不在角色层（ListUserTasks 只返回本人/候选任务）
//   - task:admin 管理/监督角色加权：跨用户全量任务视图
//   - admin 显式补齐：审批/派单/删除逐条授予，不用通配
//
// 修改本文件后必须跑 go test ./tests/parity/ ./internal/authz/ ./pkg/seeder/ 验证派生输出
// 与历史一致，否则 CI 红。

// BuiltinRolePermissionCodes 返回内置角色 → 权限码映射（已应用上面的派生规则）。
//
// 这是角色权限绑定的权威源；seeder 仅做落库。
func BuiltinRolePermissionCodes() map[string][]string {
	m := map[string][]string{
		// 系统管理员（等保三员①）：所有权限
		domainrole.SysAdmin: allPermissionCodes(),
		// 变更经理：负责变更生命周期、审批协同和发布联动
		domainrole.ChangeManager: {
			"ticket:read",
			"change:read", "change:write", "change:delete", "change:approve", "change:rollback",
			"approval:read", "approval:write",
			"release:read", "release:write", "release:approve", "release:rollback",
			"cmdb:read",
			"workflow:read", "workflow:write",
			"sla:read",
			"report:read",
			"knowledge:read", "knowledge:write",
		},
		// 服务目录管理员：负责服务目录、服务请求模板和工单模板配置
		domainrole.ServiceCatalogAdmin: {
			"service:read", "service:write",
			"service_catalog:read", "service_catalog:write", "service_catalog:delete",
			"service_request:read", "service_request:write", "service_request:delete",
			"ticket_template:read", "ticket_template:create", "ticket_template:update", "ticket_template:delete",
			"ticket_category:read", "ticket_category:create", "ticket_category:update",
			"ticket_type:read", "ticket_type:manage", "ticket_type:install_preset", "ticket_type:archive",
			"workflow:read",
			"approval:read",
			"sla:read",
			"knowledge:read",
		},
		// 问题经理：问题生命周期（根因、已知错误、变通方案）与关联事件分析
		domainrole.ProblemManager: {
			"ticket:read",
			"incident:read", "incident:write",
			"problem:read", "problem:write", "problem:delete",
			"change:read", "change:write",
			"knowledge:read", "knowledge:write",
			"cmdb:read", "sla:read", "report:read", "workflow:read", "ai:read",
		},
		// 知识管理员：文章生命周期、评审与发布，含 AI 检索面治理
		domainrole.KnowledgeManager: {
			"knowledge:read", "knowledge:write", "knowledge:delete", "knowledge:admin",
			"ticket:read", "incident:read", "problem:read",
			"ai:read", "ai:write", "report:read", "ticket_category:read",
		},
		// 配置管理员（CMDB Steward）：CI 类型/实例/关系与对账，拓扑与影响分析
		domainrole.CmdbAdmin: {
			"cmdb:read", "cmdb:write", "cmdb:delete",
			"asset:read", "asset:write", "asset:delete",
			"license:read", "license:write", "vendor:read",
			"change:read", "incident:read", "problem:read",
			"report:read", "sla:read", "ai:read",
		},
		// 服务台坐席（agent，domain/role 内置）：一线接单与处理，
		// 额外覆盖服务请求 L1 审批（approvers-l1 组兜底）。
		// 该键缺失时 role_permissions 里 agent 是空集，DBOnly 下所有 agent 用户访问任意
		// API 都 403。
		domainrole.Agent: {
			// 工单核心（11）：坐席需要工单全生命周期
			"ticket:read", "ticket:write", "ticket:create", "ticket:update",
			"ticket:assign", "ticket:escalate", "ticket:resolve", "ticket:close",
			"ticket:export", "ticket:import", "ticket:delete",
			// 工单元数据（4）：坐席起单与维护需要
			"ticket_type:read",
			"ticket_category:read", "ticket_tag:read", "ticket_template:read",
			// 事件（3）：服务台典型处理对象
			"incident:read", "incident:write", "incident:delete",
			// 问题（2）：坐席发现/登记问题
			"problem:read", "problem:write",
			// 变更（1）：只读，不能 write/approve/rollback（变更由评审组走流程）
			"change:read",
			// 知识库（2）：坐席维护 KKB
			"knowledge:read", "knowledge:write",
			// SLA（1）：仅查看自己 SLA 进度
			"sla:read",
			// 服务请求（3）：坐席通常作为 L1 审批人
			"service_request:read", "service_request:write", "service_request:approve",
			// 上下文读权限（7）：坐席处理工单时的最小视角
			"user:read", "team:read", "department:read",
			"asset:read", "cmdb:read",
			"notification:read", "ai:read",
			// 坐席工作流（7）：alert 资源名是单数，路由只声明 alert:*；
			// 旧兜底表的 alerts:read 从未匹配过任何路由，不再授予。
			"notification:write", "dashboard:read", "service_catalog:read",
			"change:write", "alert:read", "group:read", "bpmn:read",
		},
		// 安全管理员（等保三员②）
		domainrole.SecurityAdmin: {
			"ticket:read", "ticket_type:read", "incident:read", "problem:read",
			"system:read", "user:read", "role:read",
			"knowledge:read", "report:read",
			// 服务请求审批
			"service_request:read", "service_request:write", "service_request:approve",
			// 等保三员②需要看到权限模型与安全配置本身
			// （permission 清单、system_config、审计轨迹、流程与告警），仍无 system:write。
			"permission:read", "audit:read", "system_config:read",
			// 规则配置只读：治理视野需要看到自动化/分配规则本身，但不授予写码。
			"automation_rule:read", "assignment_rule:read",
			"bpmn:read", "workflow:read", "alert:read", "notification:read", "dashboard:read",
			"change:read", "release:read", "approval:read",
		},
		// 安全审计员（等保三员③，users.role=audit_admin 对齐）：
		// 全租户只读视野 + audit:read，无任何写权限码；行级写仍由权限码二次拦截。
		domainrole.AuditAdmin: {
			"ticket:read", "ticket_type:read", "incident:read", "problem:read", "change:read",
			"system:read", "user:read", "role:read", "report:read",
			"audit:read", "permission:read", "system_config:read",
			"automation_rule:read", "assignment_rule:read",
			"approval:read", "service_request:read", "service_catalog:read",
			"knowledge:read", "cmdb:read", "asset:read", "license:read", "vendor:read",
			"sla:read", "release:read", "workflow:read", "bpmn:read", "task:read",
			"notification:read", "dashboard:read", "alert:read", "ai:read",
			"team:read", "department:read", "group:read", "org:read",
			"project:read", "application:read", "on_call:read", "msp:read",
		},
		// 部门经理（users.role=manager 对齐；服务请求 L1 审批角色）
		domainrole.Manager: {
			"ticket:read", "ticket_type:read", "ticket:write", "incident:read",
			// 建单页分类下拉：manager 持 ticket:write（派生 create/update）与 ticket:assign，
			// 能建单却读不到分类，实测 GET /api/v1/ticket-categories 返回 403。
			"ticket_category:read",
			"problem:read", "change:read", "report:read",
			"user:read", "department:read", "team:read",
			"knowledge:read",
			"service_request:read", "service_request:write", "service_request:approve",
			"ticket:assign", "ticket:escalate", "ticket:export",
			"notification:read", "notification:write",
			"incident:write", "dashboard:read", "cmdb:read",
			"service_catalog:read", "sla:read", "bpmn:read",
			"release:read", "release:write",
			"asset:read", "asset:write", "license:read", "license:write",
			"group:read", "group:write", "org:read", "org:write",
			"project:read", "project:write",
			"application:read", "application:write", "ai:read",
		},
		// IT管理员（users.role=it_admin 对齐；服务请求 L2 审批角色）：
		// ITIL 流程运营所有者——工单类型、SLA、流程绑定与目录配置的落地者。
		domainrole.ITAdmin: {
			"ticket:read", "ticket_type:read", "ticket:write", "ticket:assign", "incident:read", "incident:write",
			"problem:read", "change:read", "asset:read", "cmdb:read", "cmdb:write",
			"user:read", "team:read", "knowledge:read", "report:read",
			"service_request:read", "service_request:write", "service_request:approve",
			// 流程运营职责需要配置面（工单类型/分类/模板、SLA、流程与发布读、
			// 服务目录、CI 类型建模），不含 user/role/system 写（归 admin 与三员）。
			"ticket_type:manage", "ticket_category:read", "ticket_tag:read", "ticket_template:read",
			"sla:read", "sla:write", "workflow:read", "workflow:write", "bpmn:read",
			// 规则配置面属流程运营职责；此前接口挂在 system_config:* 上，
			// it_admin 看得到菜单却恒 403（实测 code 2003）。
			"automation_rule:read", "automation_rule:write",
			"assignment_rule:read", "assignment_rule:write",
			"service_catalog:read", "release:read", "notification:read", "notification:write",
			"dashboard:read", "group:read", "approval:read",
			// AI 分派建议/摘要是 IT 运营一线工具，其余运营角色（agent/manager/end_user）均持 ai:read。
			"ai:read",
		},
		// 最终用户：写面只限自己的工单/服务请求，其余资源一律只读。
		domainrole.EndUser: {
			"ticket:read", "ticket:write", "ticket:create", "ticket:update",
			// 建单页分类下拉（/ticket-categories）与工单详情流转进度（state-v2）
			// 属最终用户旅程，按「其余资源一律只读」口径授予只读码。
			"ticket_category:read", "workflow:read",
			"knowledge:read", "service_catalog:read",
			"service_request:read", "service_request:write",
			"notification:read", "user:read",
			"notification:write", "dashboard:read", "ai:read", "ai:write",
			// system_config:read 已撤：前端只有 /admin/system-config 会读它，最终用户没有该入口，
			// 持码却等于能整表读取 SMTP/密码策略/2FA 等租户级配置。
			"sla:read", "org:read", "department:read",
			"cmdb:read", "incident:read", "change:read", "problem:read",
			"bpmn:read", "release:read", "asset:read", "license:read",
		},
		// 租户管理员（users.role=admin 对齐）。middleware 的 admin 兜底表已删除，
		// 本条目即 admin 权限的唯一来源；缺条目会让 roles 表 admin 行成为空集，
		// DBOnly 模式下 admin 用户全量 403。
		domainrole.Admin: {
			"ticket:read", "ticket:write", "ticket:delete", "ticket:admin",
			"notification:read", "notification:write",
			"ticket_category:read", "ticket_category:write", "ticket_category:delete",
			"ticket_tag:read", "ticket_tag:write", "ticket_tag:delete",
			"ticket_template:read", "ticket_template:write", "ticket_template:delete",
			"ticket_type:read", "ticket_type:write", "ticket_type:create", "ticket_type:update", "ticket_type:delete", "ticket_type:manage", "ticket_type:archive",
			"user:read", "user:write", "user:delete",
			"dashboard:read", "dashboard:admin",
			"knowledge:read", "knowledge:write", "knowledge:admin",
			"cmdb:read", "cmdb:write", "cmdb:delete",
			"tenant:read", "tenant:write",
			"incident:read", "incident:write", "incident:force-update", "incident:admin",
			"service_catalog:read", "service_catalog:write", "service_catalog:delete",
			"service_request:read", "service_request:write",
			"change:read", "change:write", "change:delete",
			"problem:read", "problem:write", "problem:delete",
			"release:read", "release:write", "release:delete",
			"sla:read", "sla:write", "sla:delete",
			"alert:read", "alert:write",
			"alerts:read", "alerts:write",
			"audit:read",
			"ai:read", "ai:write",
			"role:read", "role:write", "role:delete",
			"permission:read",
			"system_config:read", "system_config:write",
			// 工单规则配置面：租管需要看到并维护自动化/分配规则。
			"automation_rule:read", "automation_rule:write",
			"assignment_rule:read", "assignment_rule:write",
			"org:read", "org:write",
			"department:read", "department:create", "department:update", "department:delete",
			// 团队面：admin 条目此前完全没有 team:*，DBOnly 下租管打开团队页与
			// 任何成员写操作都是 403（菜单可见但接口拒绝）。
			"team:read", "team:write",
			"project:read", "project:write", "project:delete",
			"application:read", "application:write",
			"group:read", "group:write",
			"bpmn:read", "bpmn:write", "bpmn:delete",
			"task:read", "task:update", "task:admin",
			"asset:read", "asset:write", "asset:delete",
			"license:read", "license:write", "license:delete",
			"report:read",
			"msp:read",
			"marketplace:read", "marketplace:write",
			"sso:read", "sso:write",
		},
		// 二线技术员（users.role=technician 对齐）；本条目即 technician 权限的唯一来源。
		domainrole.Technician: {
			"ticket:read", "ticket:write",
			// 二线处理语义：接手升级单据并在团队内转派/继续升级。
			// 与退役的 l2_support 权限面等价（ticket:write 由同义补齐展开为 create/update）。
			"ticket:assign", "ticket:escalate",
			"notification:read",
			"knowledge:read",
			"cmdb:read",
			"incident:read", "incident:write",
			"service_catalog:read",
			"service_request:read", "service_request:write",
			"alert:read",
			"alerts:read",
			"ai:read",
			"group:read",
			// 剥离 bpmn:write。二线技术员的语义是「处理指派给自己的任务」，
			// 不是「设计/发布/触发流程」；流程定义与实例写权限收归 admin 及以上。
			"bpmn:read",
			"task:read", "task:update",
		},
	}

	// 任务面基线。
	// 任何角色都可能被流程指派任务（变更评审、服务请求审批、工单流转），
	// 因此 task:read / task:update 是全角色基线，而不是个别角色的特权。
	// 粒度控制不在角色层：ListUserTasks 只返回本人/候选任务；task:update 的每一步
	// （认领态区分、自审批防护、委托/加签目标校验）由 BPMN 引擎 authorizeTaskActor 收口。
	// 例外：等保三员③（audit_admin）按分立原则只读，不授予 task:update。
	for role, codes := range m {
		if role == domainrole.AuditAdmin {
			m[role] = appendMissingCodes(codes, "task:read")
			continue
		}
		m[role] = appendMissingCodes(codes, "task:read", "task:update")
	}
	// 跨用户全量任务视图（GET /bpmn/tasks/all、GET /workflow/tasks/all）只给管理与监督角色。
	for _, role := range []string{
		domainrole.SysAdmin, domainrole.Admin, domainrole.Manager, domainrole.ITAdmin,
		domainrole.SecurityAdmin, domainrole.AuditAdmin,
		domainrole.ChangeManager, domainrole.ServiceCatalogAdmin,
	} {
		if codes, ok := m[role]; ok {
			m[role] = appendMissingCodes(codes, "task:admin")
		}
	}

	// 同义动作补齐：路由按细分动作（create/update/…）声明 RequirePermission，
	// 而种子只授予 write，精确匹配下会 403——admin 连更新工单都不可达。
	// 规则 write ⇒ 同义细分动作，只让已有 write 真正可用，不放大能力边界；
	// delete/assign/approve 等非同类动作不在此自动授予。
	synonymParity := map[string][]string{
		"ticket":          {"create", "update"},
		"ticket_category": {"create", "update"},
		"ticket_tag":      {"create", "update"},
		"ticket_template": {"create", "update"},
		"department":      {"create", "update", "delete"},
		"notification":    {"create"},
		"system":          {"read"},
	}
	for role, codes := range m {
		has := func(code string) bool {
			for _, c := range codes {
				if c == code {
					return true
				}
			}
			return false
		}
		var extras []string
		for fam, acts := range synonymParity {
			if !has(fam + ":write") {
				continue
			}
			for _, a := range acts {
				extras = append(extras, fam+":"+a)
			}
		}
		if len(extras) > 0 {
			m[role] = appendMissingCodes(codes, extras...)
		}
	}

	// admin 显式补齐：租内最高管理角色，
	// 审批/派单/删除类动作由种子明确授予而非通配。
	if codes, ok := m[domainrole.Admin]; ok {
		m[domainrole.Admin] = appendMissingCodes(codes,
			"ticket:assign", "ticket:escalate", "ticket:export",
			"change:approve", "change:rollback",
			"release:approve", "release:rollback",
			"service_request:approve",
			"incident:delete", "knowledge:delete",
		)
	}

	return m
}

func appendMissingCodes(codes []string, extra ...string) []string {
	seen := make(map[string]bool, len(codes)+len(extra))
	out := make([]string, 0, len(codes)+len(extra))
	for _, c := range codes {
		if !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	for _, c := range extra {
		if !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	return out
}

// allPermissionCodes 返回 sysadmin 的全量权限码。
//
// 直接委托 catalog.AllCodes()（Definitions() 的权威投影），不再手写枚举。
// 历史上这里是一份手写清单，与 catalog.go 存在差集：新增权限码时只改 catalog
// 不改这里，sysadmin 就悄悄少了一批码——而 sysadmin 恰恰被假定为"全量"，
// 结果是最高权限角色反而拿不到新能力，且没有任何告警。
// 实测 2026-10-13 修复前：手写清单 129 个，catalog 160 个，缺 33 个
// （tenant:read/write、notification:*、system_config:*、dashboard:*、alert:*、
// permission:read、role:delete、service_request:approve、incident:admin 等）。
func allPermissionCodes() []string {
	return AllCodes()
}

// RetiredRolePermissionCodes 返回「角色 → 显式退役权限码」清单。
//
// 播种收敛契约自此为「只增不减」：seedRolePermissions 不再按内置码集反向删除
// role_permissions 中多出的授权行——否则运维在数据库里手工补的授权（如为
// sysadmin 补授 catalog 中内置清单缺失的码）会在下一次播种时被静默撤销。
// 产品若在某版本中故意从 BuiltinRolePermissionCodes 移除某角色的权限码，
// 必须在同一提交里把该 (角色, 码) 登记到本清单，播种器才会从既有安装中移除。
//
// 守卫契约（pkg/seeder/role_permission_guard_test.go 锁定）：
//   - 键 ⊆ BuiltinRolePermissionCodes() 的键；
//   - 值 ⊆ 权限清单（Definitions()）；
//   - 值与该角色现行内置码集不得重叠（既授予又退役 = 配置矛盾）。
//
// 退役清单必须与内置码集的收缩同步：只改 BuiltinRolePermissionCodes 不会撤销
// 既有安装的授权行，最终用户仍拿着旧权限码。
func RetiredRolePermissionCodes() map[string][]string {
	return map[string][]string{
		// GET /api/v1/system-configs 整表返回 SMTP、密码策略、2FA 等租户级配置，
		// 前端只有 /admin/system-config 读它，最终用户没有该入口。
		domainrole.EndUser: {"system_config:read"},
	}
}
