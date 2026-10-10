package authz

import domainrole "itsm-backend/domain/role"

// 本文件是「角色 → 权限码」绑定的权威源（2026-09-17 批次 6 第 3 步）。
//
// 词表边界（2026-10-07 角色模型重设）：本 map 的键必须恰好等于
// domain/role.All ∪ domain/role.Practice ∪ domain/role.MSP（由
// domain/role/contract_test.go 锁死）；岗位/团队型条目（旧 it_director、
// ops_*、l1/l2/l3_*、dba、network_eng、sd_manager、rd_manager、developer、
// qa_engineer、dept_manager、team_lead、guest）与 legacy security 已退役，
// 组织架构差异由部门/团队/租户自建角色承载。
//
// 派生规则说明：
//   - 同义奇偶补齐（write ⇒ {create, update} 等）：修 admin/technician 等持有 write 但路由声明
//     细分动作（create/update）的角色在路由级 RequirePermission 精确匹配下 403 的问题
//   - task:read/update 全角色基线（审计员只读除外）：任何角色都可能被流程指派任务
//     （变更评审、服务请求审批、工单流转），粒度控制不在角色层（ListUserTasks 只返回本人/候选任务）
//   - task:admin 管理/监督角色加权：跨用户全量任务视图
//
// 修改本文件后必须跑 go test ./tests/parity/ 与 ./internal/authz/ 验证
// 派生输出与历史一致（seeder 仅做落库，行为零变化）。

// BuiltinRolePermissionCodes 返回内置角色 → 权限码映射（已应用派生规则：task:read/update
// 全角色基线、task:admin 管理角色加权、同义奇偶补齐、admin 显式补齐）。
//
// 这是角色权限绑定的权威源（2026-09-17 批次 6 第 3 步）；seeder 仅做落库。
// 修改本文件后必须跑 go test ./tests/parity/ ./internal/authz/ ./pkg/seeder/ 验证派生输出
// 与历史一致（任何漂移 CI 红并给出修复指引）。
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
		// 此前 seedRolePermissions 漏定义该键，导致 role_permissions 表中权限为 0，
		// 所有 agent 用户访问任意 API 均 403（2026-09-16 P0 修复）。
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
			// 2026-10-05 N4 拍板扩权：补 R2-a 实测差集中坐席工作流所需 7 码。
			// alert 拼写收敛口径：路由声明仅使用 alert 资源（rbac_precheck_gen 实测零条
			// alerts:* 声明），原 middleware 的 alerts:read 属兜底表内部漂移，不再授予。
			"notification:write", "dashboard:read", "service_catalog:read",
			"change:write", "alert:read", "group:read", "bpmn:read",
		},
		// 安全管理员（等保三员②）
		domainrole.SecurityAdmin: {
			"ticket:read", "ticket_type:read", "incident:read", "problem:read",
			"system:read", "user:read", "role:read",
			"knowledge:read", "report:read",
			// 服务请求审批（2026-09-07 P1-B：审批角色权限补齐）
			"service_request:read", "service_request:write", "service_request:approve",
			// 2026-10-07 角色重设：等保三员②需要看到权限模型与安全配置本身
			// （permission 清单、system_config、审计轨迹、流程与告警），仍无 system:write。
			"permission:read", "audit:read", "system_config:read",
			"bpmn:read", "workflow:read", "alert:read", "notification:read", "dashboard:read",
			"change:read", "release:read", "approval:read",
		},
		// 安全审计员（等保三员③，users.role=audit_admin 对齐）：
		// 全租户只读视野 + audit:read，无任何写权限码；行级写仍由权限码二次拦截。
		domainrole.AuditAdmin: {
			"ticket:read", "ticket_type:read", "incident:read", "problem:read", "change:read",
			"system:read", "user:read", "role:read", "report:read",
			"audit:read", "permission:read", "system_config:read",
			"approval:read", "service_request:read", "service_catalog:read",
			"knowledge:read", "cmdb:read", "asset:read", "license:read", "vendor:read",
			"sla:read", "release:read", "workflow:read", "bpmn:read", "task:read",
			"notification:read", "dashboard:read", "alert:read", "ai:read",
			"team:read", "department:read", "group:read", "org:read",
			"project:read", "application:read", "on_call:read", "msp:read",
		},
		// 部门经理（users.role=manager 对齐；服务请求 L1 审批角色）
		// 2026-10-05 N4 拍板扩权：补齐 plans/product-remediation-plan-2026-10-03.md §3
		// R2-a 实测差集 26 对（原 middleware.RolePermissions["manager"] 独有码）。
		domainrole.Manager: {
			"ticket:read", "ticket_type:read", "ticket:write", "incident:read",
			"problem:read", "change:read", "report:read",
			"user:read", "department:read", "team:read",
			"knowledge:read",
			"service_request:read", "service_request:write", "service_request:approve",
			// N4 扩权（26）：工单操作与全域读、经理视角聚合
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
			"ticket:read", "ticket_type:read", "ticket:write", "incident:read", "incident:write",
			"problem:read", "change:read", "asset:read", "cmdb:read",
			"user:read", "team:read", "knowledge:read", "report:read",
			"service_request:read", "service_request:write", "service_request:approve",
			// 2026-10-07 角色重设：流程运营职责需要配置面（工单类型/分类/模板、SLA、
			// 流程与发布读、服务目录），不含 user/role/system 写（归 admin 与三员）。
			"ticket_type:manage", "ticket_category:read", "ticket_tag:read", "ticket_template:read",
			"sla:read", "sla:write", "workflow:read", "workflow:write", "bpmn:read",
			"service_catalog:read", "release:read", "notification:read", "notification:write",
			"dashboard:read", "group:read", "approval:read",
		},
		// 最终用户：可提交和维护自己的工单/服务请求
		// 2026-10-05 N4 拍板扩权：补 R2-a 实测差集 16 对，以只读可见性为主
		// （ITIL 读面 + 仪表盘/通知/AI 基线能力）。
		domainrole.EndUser: {
			"ticket:read", "ticket:write", "ticket:create", "ticket:update",
			"knowledge:read", "service_catalog:read",
			"service_request:read", "service_request:write",
			"notification:read", "user:read",
			"notification:write", "dashboard:read", "ai:read", "ai:write",
			"sla:read", "system_config:read", "org:read", "department:read",
			"cmdb:read", "incident:read", "change:read", "problem:read",
			"bpmn:read", "release:read", "asset:read", "license:read",
		},
		// 租户管理员（users.role=admin 对齐；2026-09-17 P0 曾与 middleware admin 91 对全等，
		// 2026-10-05 N5 后 middleware 表已删除，本条目即 admin 唯一权威源）
		// 2026-09-17 P0：此前缺条目导致 roles 表 admin 行权限空集=DB 显式撤销，admin 用户全 403）
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
			"org:read", "org:write",
			"department:read", "department:create", "department:update", "department:delete",
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
		},
		// 二线技术员（users.role=technician 对齐；2026-10-05 N5 后本条目即 technician 唯一权威源，
		// 2026-09-17 P0 曾按 middleware 兜底 16 对补齐）
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
			// 2026-09-17 P0：剥离 bpmn:write。二线技术员的语义是「处理指派给自己的任务」，
			// 不是「设计/发布/触发流程」；流程定义与实例写权限收归 admin 及以上。
			"bpmn:read",
			"task:read", "task:update",
		},
	}

	// 任务面基线（2026-09-17 P0「越权写收口」）。
	// 任何角色都可能被流程指派任务（变更评审、服务请求审批、工单流转），
	// 因此 task:read / task:update 是全角色基线，而不是个别角色的特权。
	// 粒度控制不在角色层：ListUserTasks 只返回本人/候选任务；task:update 的每一步
	// （认领态区分、自审批防护、委托/加签目标校验）由 handler 层 authorizeTaskActor 收口。
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

	// 同义动作奇偶补齐（2026-09-17 批次 2/3）。
	// 路由声明使用细分动作（create/update/...），历史种子只给了 write——
	// 持有 write 的角色在路由级 RequirePermission 精确匹配下 403
	// （实测 admin 缺 ticket:update/create，连更新工单都不可达）。
	// 规则：write ⇒ 同义细分动作。这是**让已有 write 真正可用**的补齐，
	// 不放大能力边界；delete/assign/approve 等非同义动作不在此自动授予。
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

	// admin 显式补齐（2026-09-17 批次 2/3）：租内最高管理角色，
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

func allPermissionCodes() []string {
	return []string{
		"ticket:read", "ticket:write", "ticket:create", "ticket:update", "ticket:assign",
		"ticket:escalate", "ticket:resolve", "ticket:close", "ticket:export", "ticket:import",
		"ticket:admin", "ticket:delete",
		"ticket_type:read", "ticket_type:manage", "ticket_type:install_preset", "ticket_type:archive",
		"ticket_category:read", "ticket_category:create", "ticket_category:update", "ticket_category:delete",
		"ticket_tag:read", "ticket_tag:create", "ticket_tag:update", "ticket_tag:delete",
		"ticket_template:read", "ticket_template:create", "ticket_template:update", "ticket_template:delete",
		"incident:read", "incident:write", "incident:delete",
		"problem:read", "problem:write", "problem:delete",
		"change:read", "change:write", "change:delete", "change:approve", "change:rollback",
		"release:read", "release:write", "release:delete", "release:approve", "release:rollback",
		"asset:read", "asset:write", "asset:delete",
		"cmdb:read", "cmdb:write", "cmdb:delete",
		"report:read", "report:write",
		"license:read", "license:write", "license:delete",
		"service:read", "service:write",
		"service_catalog:read", "service_catalog:write", "service_catalog:delete",
		"service_request:read", "service_request:write", "service_request:delete",
		"sla:read", "sla:write", "sla:delete",
		"user:read", "user:write", "user:delete",
		"group:read", "group:write",
		"role:read", "role:write",
		"department:read", "department:write",
		"team:read", "team:write",
		"approval:read", "approval:write",
		"workflow:read", "workflow:write",
		// BPMN 流程引擎（2026-09-17 P0：sysadmin/总监此前缺 bpmn 码 → 流程模块不可用）
		"bpmn:read", "bpmn:write", "bpmn:delete",
		// 任务面（2026-09-17 P0）：与 bpmn 分权，见 permissionDefinitions 注释
		"task:read", "task:update", "task:admin",
		"knowledge:read", "knowledge:write", "knowledge:delete",
		"system:read", "system:write",
		"org:read", "org:write",
		"project:read", "project:write",
		"application:read", "application:write",
		"audit:read",
		"ai:read", "ai:write",
		"connector:read", "connector:write",
		"email_intake:read", "email_intake:review", "email_intake:retry", "email_intake:override",
		"customer_master:read", "customer_master:write", "support_contract:read", "support_contract:write",
		"on_call:read", "on_call:write",
		"vendor:read", "vendor:write", "vendor:delete",
		"survey:write",
		"msp:read", "msp:write",
		"msp_customer:read", "msp_customer:write",
		"msp_ticket:read", "msp_ticket:write",
		"msp_allocation:read", "msp_allocation:write",
		"msp_report:read", "msp_report:write",
		"marketplace:read", "marketplace:write",
	}
}

// RetiredRolePermissionCodes 返回「角色 → 显式退役权限码」清单（2026-10-03 R2-d）。
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
// 当前为空清单：尚无需要退役的授权。
func RetiredRolePermissionCodes() map[string][]string {
	return map[string][]string{}
}
