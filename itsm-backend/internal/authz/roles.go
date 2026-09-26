package authz

// 本文件是「角色 → 权限码」绑定的权威源（2026-09-17 批次 6 第 3 步）。
//
// 派生规则说明：
//   - 同义奇偶补齐（write ⇒ {create, update} 等）：修 admin/technician 等持有 write 但路由声明
//     细分动作（create/update）的角色在路由级 RequirePermission 精确匹配下 403 的问题
//   - task:read/update 全角色基线（除 guest）：任何角色都可能被流程指派任务（变更评审、
//     服务请求审批、工单流转），粒度控制不在角色层（ListUserTasks 只返回本人/候选任务）
//   - task:admin 13 个管理/监督角色加权：跨用户全量任务视图
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
		// 系统管理员：所有权限
		"sysadmin": allPermissionCodes(),
		// IT总监：全局读写（不含系统管理）
		"it_director": allExcept([]string{"system:write", "msp:write", "msp_allocation:write"}),
		// 运维总监：运维相关读写
		"ops_director": allExcept([]string{"system:write", "msp:write", "msp_allocation:write", "msp_report:write"}),
		// 运维经理：运维相关读写
		"ops_manager": {
			"ticket:read", "ticket:write", "ticket:create", "ticket:update",
			"ticket:assign", "ticket:escalate", "ticket:export", "ticket:delete",
			"incident:read", "incident:write",
			"ticket_type:read", "ticket_type:manage", "ticket_type:install_preset", "ticket_type:archive",
			"problem:read", "problem:write", "change:read", "change:write",
			"asset:read", "asset:write", "cmdb:read", "cmdb:write",
			"sla:read", "workflow:read", "report:read",
			"team:read", "department:read", "user:read", "ai:read",
		},
		// 运维工程师：运维操作
		"ops_engineer": {
			"ticket:read", "ticket:write", "incident:read", "incident:write",
			"problem:read", "change:read", "asset:read", "asset:write",
			"cmdb:read", "cmdb:write", "sla:read", "knowledge:read", "knowledge:write", "ai:read",
		},
		// DBA工程师
		"dba": {
			"ticket:read", "incident:read", "problem:read", "problem:write",
			"change:read", "change:write", "asset:read", "cmdb:read", "cmdb:write",
			"knowledge:read", "knowledge:write", "ai:read",
		},
		// 网络安全工程师
		"network_eng": {
			"ticket:read", "incident:read", "incident:write", "problem:read",
			"change:read", "asset:read", "cmdb:read", "sla:read",
			"knowledge:read", "knowledge:write", "ai:read",
		},
		// 服务台主管
		"sd_manager": {
			"ticket:read", "ticket:write", "ticket:create", "ticket:update",
			"ticket:assign", "ticket:escalate", "ticket:export",
			"incident:read", "incident:write",
			"ticket_type:read", "ticket_type:manage", "ticket_type:install_preset", "ticket_type:archive",
			"problem:read", "change:read", "sla:read", "sla:write",
			"knowledge:read", "knowledge:write", "report:read",
			"user:read", "team:read", "ai:read",
		},
		// 变更经理：负责变更生命周期、审批协同和发布联动
		"change_manager": {
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
		"service_catalog_admin": {
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
		// 一线支持工程师：具备工单全生命周期操作权限
		"l1_support": {
			"ticket:read", "ticket:write", "ticket:create", "ticket:update",
			"ticket:assign", "ticket:escalate", "ticket:export",
			"ticket_type:read",
			"incident:read", "incident:write",
			"knowledge:read", "user:read", "sla:read", "notification:read", "ai:read",
		},
		// 服务台坐席（agent，domain/role 内置）：一线接单与处理，与 l1_support 等权，
		// 额外覆盖服务请求 L1 审批（approvers-l1 组兜底）。
		// 此前 seedRolePermissions 漏定义该键，导致 role_permissions 表中权限为 0，
		// 所有 agent 用户访问任意 API 均 403（2026-09-16 P0 修复）。
		"agent": {
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
		},
		// 二线支持工程师
		"l2_support": {
			"ticket:read", "ticket:write", "ticket:create", "ticket:update",
			"ticket:assign", "ticket:escalate", "ticket:export",
			"ticket_type:read",
			"incident:read", "incident:write",
			"problem:read", "change:read", "asset:read",
			"knowledge:read", "knowledge:write", "user:read", "sla:read", "notification:read", "ai:read",
		},
		// 三线专家
		"l3_expert": {
			"ticket:read", "ticket:write", "ticket:create", "ticket:update",
			"ticket:assign", "ticket:escalate", "ticket:export",
			"ticket_type:read",
			"incident:read", "incident:write",
			"problem:read", "problem:write", "change:read", "change:write",
			"asset:read", "cmdb:read", "knowledge:read", "knowledge:write",
			"sla:read", "workflow:read", "notification:read", "ai:read",
		},
		// 研发经理
		"rd_manager": {
			"ticket:read", "ticket_type:read", "problem:read", "change:read", "change:write",
			"release:read", "release:write", "workflow:read", "workflow:write",
			"knowledge:read", "knowledge:write", "report:read",
		},
		// 开发工程师
		"developer": {
			"ticket:read", "ticket_type:read", "problem:read", "change:read",
			"release:read", "knowledge:read", "knowledge:write",
		},
		// 测试工程师
		"qa_engineer": {
			"ticket:read", "ticket_type:read", "problem:read", "change:read",
			"release:read", "knowledge:read", "knowledge:write", "report:read",
		},
		// 安全管理员
		"security_admin": {
			"ticket:read", "ticket_type:read", "incident:read", "problem:read",
			"system:read", "user:read", "role:read",
			"knowledge:read", "report:read",
			// 服务请求审批（2026-09-07 P1-B：审批角色权限补齐）
			"service_request:read", "service_request:write", "service_request:approve",
		},
		// 审计管理员
		"audit_admin": {
			"ticket:read", "ticket_type:read", "incident:read", "problem:read", "change:read",
			"system:read", "user:read", "role:read", "report:read",
		},
		// 部门经理（users.role=manager 对齐；服务请求 L1 审批角色）
		"manager": {
			"ticket:read", "ticket_type:read", "ticket:write", "incident:read",
			"problem:read", "change:read", "report:read",
			"user:read", "department:read", "team:read",
			"knowledge:read",
			"service_request:read", "service_request:write", "service_request:approve",
		},
		// IT管理员（users.role=it_admin 对齐；服务请求 L2 审批角色）
		"it_admin": {
			"ticket:read", "ticket_type:read", "ticket:write", "incident:read", "incident:write",
			"problem:read", "change:read", "asset:read", "cmdb:read",
			"user:read", "team:read", "knowledge:read", "report:read",
			"service_request:read", "service_request:write", "service_request:approve",
		},
		// 部门经理（业务条线经理，rd_manager 等场景之外的通用审批角色）
		"dept_manager": {
			"ticket:read", "ticket_type:read", "ticket:write", "incident:read",
			"problem:read", "change:read", "report:read",
			"user:read", "department:read", "team:read",
			"knowledge:read",
			"service_request:read", "service_request:approve",
		},
		// 团队主管
		"team_lead": {
			"ticket:read", "ticket_type:read", "ticket:write", "incident:read",
			"problem:read", "change:read", "team:read",
			"user:read", "knowledge:read",
		},
		// 安全审批人：可读工单/事件/问题/变更/知识库/通知，做安全审批
		"security": {
			"ticket:read", "ticket:write",
			"incident:read", "incident:write",
			"problem:read",
			"change:read", "change:write",
			"release:read",
			"knowledge:read",
			"notification:read",
			"asset:read",
			"team:read", "user:read",
		},
		// 普通用户：可提交和维护自己的工单/服务请求
		"end_user": {
			"ticket:read", "ticket:write", "ticket:create", "ticket:update",
			"knowledge:read", "service_catalog:read",
			"service_request:read", "service_request:write",
			"notification:read", "user:read",
		},
		// 访客
		"guest": {
			"knowledge:read",
		},
		// 租户管理员（users.role=admin 对齐；与 middleware.RolePermissions["admin"] 91 对全等，
		// 2026-09-17 P0：此前缺条目导致 roles 表 admin 行权限空集=DB 显式撤销，admin 用户全 403）
		"admin": {
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
		// 二线技术员（users.role=technician 对齐；与 middleware.RolePermissions["technician"] 16 对全等，2026-09-17 P0 补齐）
		"technician": {
			"ticket:read", "ticket:write",
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
	// guest 除外——外部访客不参与任何内部流程。
	for role, codes := range m {
		if role == "guest" {
			continue
		}
		m[role] = appendMissingCodes(codes, "task:read", "task:update")
	}
	// 跨用户全量任务视图（GET /bpmn/tasks/all、GET /workflow/tasks/all）只给管理与监督角色。
	for _, role := range []string{
		"sysadmin", "it_director", "ops_director", "admin",
		"manager", "dept_manager", "team_lead", "sd_manager", "ops_manager",
		"change_manager", "it_admin", "security_admin", "audit_admin",
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
		if role == "guest" {
			continue
		}
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
	if codes, ok := m["admin"]; ok {
		m["admin"] = appendMissingCodes(codes,
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

func allExcept(exclude []string) []string {
	excludeSet := make(map[string]bool, len(exclude))
	for _, code := range exclude {
		excludeSet[code] = true
	}
	result := make([]string, 0)
	for _, code := range allPermissionCodes() {
		if !excludeSet[code] {
			result = append(result, code)
		}
	}
	return result
}
